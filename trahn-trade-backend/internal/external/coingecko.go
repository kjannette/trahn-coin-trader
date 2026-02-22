package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kjannette/trahn-backend/internal/httputil"
)

const coingeckoURL = "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"

type CoinGeckoClient struct {
	httpClient *http.Client
	retry      httputil.RetryConfig
}

func NewCoinGeckoClient() *CoinGeckoClient {
	return &CoinGeckoClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		retry: httputil.RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   2 * time.Second,
			MaxDelay:    10 * time.Second,
		},
	}
}

func (c *CoinGeckoClient) GetETHPrice(ctx context.Context) (float64, error) {
	resp, err := httputil.Do(ctx, c.httpClient, c.retry, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, coingeckoURL, nil)
	})
	if err != nil {
		return 0, fmt.Errorf("coingecko fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coingecko returned status %d", resp.StatusCode)
	}

	var data struct {
		Ethereum struct {
			USD float64 `json:"usd"`
		} `json:"ethereum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}

	if data.Ethereum.USD <= 0 {
		return 0, fmt.Errorf("invalid price: %f", data.Ethereum.USD)
	}

	return data.Ethereum.USD, nil
}
