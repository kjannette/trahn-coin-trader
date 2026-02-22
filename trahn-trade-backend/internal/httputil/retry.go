package httputil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var DefaultRetry = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   1 * time.Second,
	MaxDelay:    10 * time.Second,
}

// Do executes an HTTP request with exponential backoff retry.
// The buildReq function is called on each attempt to produce a fresh request
// (required because request bodies are consumed on each attempt).
func Do(ctx context.Context, client *http.Client, cfg RetryConfig, buildReq func() (*http.Request, error)) (*http.Response, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultRetry.MaxAttempts
	}

	var lastErr error
	delay := cfg.BaseDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		req, err := buildReq()
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		fmt.Printf("[RETRY] Attempt %d/%d failed: %v — retrying in %s\n",
			attempt, cfg.MaxAttempts, lastErr, delay)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return nil, fmt.Errorf("all %d attempts failed, last error: %w", cfg.MaxAttempts, lastErr)
}
