package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kjannette/trahn-backend/internal/repository"
)

const maxQueryLimit = 1000

var dateRegexp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Server struct {
	pool       *pgxpool.Pool
	priceRepo  *repository.PriceRepo
	tradeRepo  *repository.TradeRepo
	srRepo     *repository.SRRepo
	gridRepo   *repository.GridStateRepo
	httpServer *http.Server
	apiKey     string
}

func NewServer(pool *pgxpool.Pool, port int, apiKey, corsOrigin string) *Server {
	s := &Server{
		pool:      pool,
		priceRepo: repository.NewPriceRepo(pool),
		tradeRepo: repository.NewTradeRepo(pool),
		srRepo:    repository.NewSRRepo(pool),
		gridRepo:  repository.NewGridStateRepo(pool),
		apiKey:    apiKey,
	}

	mux := http.NewServeMux()

	// Price routes
	mux.HandleFunc("GET /v1/prices/today", s.handlePricesToday)
	mux.HandleFunc("GET /v1/prices/day/{date}", s.handlePricesByDay)
	mux.HandleFunc("GET /v1/prices/days", s.handleAvailableDays)
	mux.HandleFunc("GET /v1/prices/latest", s.handleLatestPrice)

	// Trade routes
	mux.HandleFunc("GET /v1/trades/today", s.handleTradesToday)
	mux.HandleFunc("GET /v1/trades/day/{date}", s.handleTradesByDay)
	mux.HandleFunc("GET /v1/trades/all", s.handleAllTrades)
	mux.HandleFunc("GET /v1/trades/stats", s.handleTradeStats)

	// Grid routes
	mux.HandleFunc("GET /v1/grid/current", s.handleGridCurrent)

	// S/R routes
	mux.HandleFunc("GET /v1/support-resistance/latest", s.handleSRLatest)
	mux.HandleFunc("GET /v1/support-resistance/history", s.handleSRHistory)

	// Health check (no auth required)
	mux.HandleFunc("GET /health", s.handleHealth)

	handler := s.authMiddleware(corsMiddleware(mux, corsOrigin))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	fmt.Printf("[API] REST API server started on http://localhost%s\n", s.httpServer.Addr)
	fmt.Printf("[API] Health check: http://localhost%s/health\n", s.httpServer.Addr)
	if s.apiKey != "" {
		fmt.Println("[API] Authentication: enabled (Bearer token)")
	} else {
		fmt.Println("[API] Authentication: disabled (no API_KEY configured)")
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// --- middleware ---

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth || token != s.apiKey {
			writeError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler, allowOrigin string) http.Handler {
	if allowOrigin == "" {
		allowOrigin = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- validation helpers ---

func validateDate(date string) bool {
	if !dateRegexp.MatchString(date) {
		return false
	}
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func parseLimit(r *http.Request, defaultLimit int) int {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxQueryLimit {
		return maxQueryLimit
	}
	return n
}

// --- response helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
