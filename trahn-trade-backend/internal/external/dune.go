package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/kjannette/trahn-backend/internal/httputil"
)

type DuneClient struct {
	apiKey       string
	baseURL      string
	method       string // "simple" or "percentile"
	lookbackDays int
	httpClient   *http.Client
	retry        httputil.RetryConfig

	mu           sync.Mutex
	cachedResult *SRResult
	lastFetch    time.Time
	cacheTTL     time.Duration
}

type SRResult struct {
	Support      float64   `json:"support"`
	Resistance   float64   `json:"resistance"`
	Midpoint     float64   `json:"midpoint"`
	AvgPrice     float64   `json:"avgPrice"`
	Method       string    `json:"method"`
	LookbackDays int       `json:"lookbackDays"`
	FetchedAt    time.Time `json:"fetchedAt"`
}

type DuneOptions struct {
	Method       string
	LookbackDays int
	RefreshHours int
}

func NewDuneClient(apiKey string, opts DuneOptions) *DuneClient {
	method := opts.Method
	if method == "" {
		method = "simple"
	}
	lookback := opts.LookbackDays
	if lookback <= 0 {
		lookback = 14
	}
	refreshHours := opts.RefreshHours
	if refreshHours <= 0 {
		refreshHours = 48
	}

	return &DuneClient{
		apiKey:       apiKey,
		baseURL:      "https://api.dune.com/api/v1",
		method:       method,
		lookbackDays: lookback,
		httpClient:   &http.Client{Timeout: 90 * time.Second},
		cacheTTL:     time.Duration(refreshHours) * time.Hour,
		retry: httputil.RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   3 * time.Second,
			MaxDelay:    15 * time.Second,
		},
	}
}

func (d *DuneClient) FetchSupportResistance(ctx context.Context, forceRefresh bool) (*SRResult, error) {
	d.mu.Lock()
	if !forceRefresh && d.cachedResult != nil && time.Since(d.lastFetch) < d.cacheTTL {
		cached := *d.cachedResult
		d.mu.Unlock()
		age := time.Since(d.lastFetch)
		fmt.Printf("[DUNE] Using cached S/R data (age: %.1f min)\n", age.Minutes())
		return &cached, nil
	}
	d.mu.Unlock()

	sql := d.buildSRQuery()
	rows, err := d.executeQuery(ctx, sql)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("dune returned no data for S/R query")
	}

	row := rows[0]
	result := &SRResult{
		Support:      jsonFloat(row, "support"),
		Resistance:   jsonFloat(row, "resistance"),
		Midpoint:     jsonFloat(row, "midpoint"),
		AvgPrice:     jsonFloat(row, "avg_price"),
		Method:       d.method,
		LookbackDays: d.lookbackDays,
		FetchedAt:    time.Now(),
	}

	if math.IsNaN(result.Support) || math.IsNaN(result.Resistance) || math.IsNaN(result.Midpoint) {
		return nil, fmt.Errorf("invalid S/R data from Dune")
	}
	if result.Support >= result.Resistance {
		return nil, fmt.Errorf("invalid S/R range: support %.2f >= resistance %.2f", result.Support, result.Resistance)
	}

	d.mu.Lock()
	d.cachedResult = result
	d.lastFetch = time.Now()
	d.mu.Unlock()

	fmt.Printf("[DUNE] S/R fetched successfully:\n")
	fmt.Printf("   Support: $%.2f\n", result.Support)
	fmt.Printf("   Resistance: $%.2f\n", result.Resistance)
	fmt.Printf("   Midpoint: $%.2f\n", result.Midpoint)
	fmt.Printf("   Method: %s, Lookback: %d days\n", result.Method, result.LookbackDays)

	return result, nil
}

// SeedCache pre-populates the in-memory cache from a previously persisted
// S/R result (e.g. loaded from the database on startup). The cached entry
// is only used if it falls within the configured TTL.
func (d *DuneClient) SeedCache(sr *SRResult) {
	if sr == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	age := time.Since(sr.FetchedAt)
	if age >= d.cacheTTL {
		fmt.Printf("[DUNE] DB S/R data too old (%.1f hours), not seeding cache\n", age.Hours())
		return
	}

	d.cachedResult = sr
	d.lastFetch = sr.FetchedAt
	fmt.Printf("[DUNE] Cache seeded from DB (age: %.1f min): midpoint $%.2f\n",
		age.Minutes(), sr.Midpoint)
}

func (d *DuneClient) NeedsRefresh() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cachedResult == nil {
		return true
	}
	return time.Since(d.lastFetch) >= d.cacheTTL
}

func (d *DuneClient) buildSRQuery() string {
	if d.method == "percentile" {
		return fmt.Sprintf(`
			SELECT
				approx_percentile(price, 0.05) as support,
				approx_percentile(price, 0.95) as resistance,
				approx_percentile(price, 0.50) as midpoint,
				AVG(price) as avg_price,
				MIN(price) as absolute_low,
				MAX(price) as absolute_high
			FROM prices.usd
			WHERE symbol = 'WETH'
				AND blockchain = 'ethereum'
				AND minute > now() - interval '%d' day
		`, d.lookbackDays)
	}

	return fmt.Sprintf(`
		SELECT
			MIN(price) as support,
			MAX(price) as resistance,
			(MIN(price) + MAX(price)) / 2 as midpoint,
			AVG(price) as avg_price
		FROM prices.usd
		WHERE symbol = 'WETH'
			AND blockchain = 'ethereum'
			AND minute > now() - interval '%d' day
	`, d.lookbackDays)
}

func (d *DuneClient) executeQuery(ctx context.Context, sql string) ([]map[string]any, error) {
	if d.apiKey == "" {
		return nil, fmt.Errorf("dune API key not configured")
	}

	fmt.Println("[DUNE] Executing S/R query...")

	body, _ := json.Marshal(map[string]string{
		"sql":         sql,
		"performance": "medium",
	})

	resp, err := httputil.Do(ctx, d.httpClient, d.retry, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/sql/execute", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Dune-API-Key", d.apiKey)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("submit query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dune query execution failed: status %d", resp.StatusCode)
	}

	var execResult struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&execResult); err != nil {
		return nil, fmt.Errorf("decode execution response: %w", err)
	}
	if execResult.ExecutionID == "" {
		return nil, fmt.Errorf("dune did not return an execution ID")
	}

	fmt.Printf("[DUNE] Query submitted, execution ID: %s\n", execResult.ExecutionID)

	const maxAttempts = 30
	const pollInterval = 2 * time.Second

	for attempt := range maxAttempts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		statusReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("%s/execution/%s/status", d.baseURL, execResult.ExecutionID), nil)
		statusReq.Header.Set("X-Dune-API-Key", d.apiKey)

		statusResp, err := d.httpClient.Do(statusReq)
		if err != nil {
			fmt.Printf("[DUNE] Status check failed (attempt %d), retrying...\n", attempt+1)
			continue
		}

		var statusData struct {
			State string `json:"state"`
			Error string `json:"error"`
		}
		json.NewDecoder(statusResp.Body).Decode(&statusData)
		statusResp.Body.Close()

		switch statusData.State {
		case "QUERY_STATE_COMPLETED", "completed":
			return d.fetchResults(ctx, execResult.ExecutionID)
		case "QUERY_STATE_FAILED", "failed":
			errMsg := statusData.Error
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return nil, fmt.Errorf("dune query failed: %s", errMsg)
		default:
			fmt.Printf("[DUNE] Query state: %s, waiting...\n", statusData.State)
		}
	}

	return nil, fmt.Errorf("dune query timed out after %d seconds", maxAttempts*int(pollInterval.Seconds()))
}

func (d *DuneClient) fetchResults(ctx context.Context, executionID string) ([]map[string]any, error) {
	resp, err := httputil.Do(ctx, d.httpClient, d.retry, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("%s/execution/%s/results", d.baseURL, executionID), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Dune-API-Key", d.apiKey)
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetch results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch dune results: status %d", resp.StatusCode)
	}

	var data struct {
		Result struct {
			Rows []map[string]any `json:"rows"`
		} `json:"result"`
		Rows []map[string]any `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode results: %w", err)
	}

	rows := data.Result.Rows
	if rows == nil {
		rows = data.Rows
	}
	return rows, nil
}

// jsonFloat extracts a float64 from a map[string]any, handling both float64 and json.Number.
func jsonFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return math.NaN()
	}
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return math.NaN()
	}
}
