package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_NoKeyConfigured(t *testing.T) {
	s := &Server{apiKey: ""}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/trades/stats", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when no API key configured, got %d", rr.Code)
	}
}

func TestAuthMiddleware_HealthBypass(t *testing.T) {
	s := &Server{apiKey: "secret123"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health without auth, got %d", rr.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	s := &Server{apiKey: "secret123"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/prices/latest", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_WrongKey(t *testing.T) {
	s := &Server{apiKey: "secret123"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/prices/latest", nil)
	req.Header.Set("Authorization", "Bearer wrong_key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_CorrectKey(t *testing.T) {
	s := &Server{apiKey: "secret123"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/prices/latest", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAuthMiddleware_MalformedBearer(t *testing.T) {
	s := &Server{apiKey: "secret123"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.authMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/prices/latest", nil)
	req.Header.Set("Authorization", "Basic secret123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-Bearer auth, got %d", rr.Code)
	}
}

func TestValidateDate(t *testing.T) {
	valid := []string{"2024-01-15", "2025-12-31", "2020-02-29"}
	for _, d := range valid {
		if !validateDate(d) {
			t.Fatalf("expected %q to be valid", d)
		}
	}

	invalid := []string{
		"", "2024", "01-15-2024", "2024/01/15",
		"abcd-ef-gh", "2024-13-01", "2024-01-32",
		"2024-1-5", "20240115",
	}
	for _, d := range invalid {
		if validateDate(d) {
			t.Fatalf("expected %q to be invalid", d)
		}
	}
}

func TestParseLimit(t *testing.T) {
	cases := []struct {
		query    string
		deflt    int
		expected int
	}{
		{"", 100, 100},
		{"?limit=50", 100, 50},
		{"?limit=0", 100, 100},
		{"?limit=-5", 100, 100},
		{"?limit=abc", 100, 100},
		{"?limit=2000", 100, maxQueryLimit},
		{"?limit=1000", 100, 1000},
		{"?limit=1", 50, 1},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/test"+tc.query, nil)
		got := parseLimit(req, tc.deflt)
		if got != tc.expected {
			t.Fatalf("parseLimit(%q, %d) = %d, want %d", tc.query, tc.deflt, got, tc.expected)
		}
	}
}

func TestCorsMiddleware_Headers(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner, "https://myapp.example.com")

	req := httptest.NewRequest(http.MethodGet, "/v1/prices/latest", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://myapp.example.com" {
		t.Fatalf("expected custom origin, got %q", origin)
	}

	allow := rr.Header().Get("Access-Control-Allow-Headers")
	if allow == "" {
		t.Fatal("expected Allow-Headers to include Authorization")
	}
}

func TestCorsMiddleware_Preflight(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not be called for OPTIONS")
	})
	handler := corsMiddleware(inner, "*")

	req := httptest.NewRequest(http.MethodOptions, "/v1/prices/latest", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for preflight, got %d", rr.Code)
	}
}
