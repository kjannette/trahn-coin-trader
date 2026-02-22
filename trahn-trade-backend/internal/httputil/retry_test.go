package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_SuccessFirstAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second}

	resp, err := Do(context.Background(), client, cfg, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_RetriesOnServerError(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: 200 * time.Millisecond}

	resp, err := Do(context.Background(), client, cfg, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on third attempt, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestDo_AllAttemptsFail(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("upstream error"))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: 200 * time.Millisecond}

	_, err := Do(context.Background(), client, cfg, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err == nil {
		t.Fatal("expected error after all attempts failed")
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
	t.Logf("Error after retries: %v", err)
}

func TestDo_NoRetryOnClientError(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: 200 * time.Millisecond}

	resp, err := Do(context.Background(), client, cfg, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if attempts.Load() != 1 {
		t.Fatalf("should not retry on 4xx, expected 1 attempt, got %d", attempts.Load())
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	cfg := RetryConfig{MaxAttempts: 10, BaseDelay: 500 * time.Millisecond, MaxDelay: 2 * time.Second}

	_, err := Do(ctx, client, cfg, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
	t.Logf("Cancelled: %v", err)
}
