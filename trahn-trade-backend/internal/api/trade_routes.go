package api

import (
	"fmt"
	"net/http"

	"github.com/kjannette/trahn-backend/internal/repository"
)

type tradeJSON struct {
	T            int64   `json:"t"`
	Side         string  `json:"side"`
	Price        float64 `json:"price"`
	Qty          float64 `json:"qty"`
	GridLevel    *int    `json:"gridLevel,omitempty"`
	USDValue     float64 `json:"usdValue"`
	IsPaperTrade bool    `json:"isPaperTrade"`
}

// parseTradeMode extracts the ?mode= query parameter.
// Returns a *bool: nil = all, true = paper, false = live.
func parseTradeMode(r *http.Request) (*bool, error) {
	v := r.URL.Query().Get("mode")
	switch v {
	case "", "all":
		return nil, nil
	case "paper":
		b := true
		return &b, nil
	case "live":
		b := false
		return &b, nil
	default:
		return nil, fmt.Errorf("invalid mode %q, expected paper|live|all", v)
	}
}

func (s *Server) handleTradesToday(w http.ResponseWriter, r *http.Request) {
	mode, err := parseTradeMode(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	today := repository.TradingDayNow()

	trades, err := s.tradeRepo.GetByDay(ctx, today, mode)
	if err != nil {
		fmt.Printf("Error fetching today's trades: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	out := make([]tradeJSON, len(trades))
	for i, t := range trades {
		out[i] = tradeJSON{
			T: t.Timestamp.UnixMilli(), Side: t.Side,
			Price: t.Price, Qty: t.Quantity,
			GridLevel: t.GridLevel, USDValue: t.USDValue,
			IsPaperTrade: t.IsPaperTrade,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleTradesByDay(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if !validateDate(date) {
		writeError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	mode, err := parseTradeMode(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	trades, err := s.tradeRepo.GetByDay(ctx, date, mode)
	if err != nil {
		fmt.Printf("Error fetching trades for %s: %v\n", date, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}

	out := make([]tradeJSON, len(trades))
	for i, t := range trades {
		out[i] = tradeJSON{
			T: t.Timestamp.UnixMilli(), Side: t.Side,
			Price: t.Price, Qty: t.Quantity,
			GridLevel: t.GridLevel, USDValue: t.USDValue,
			IsPaperTrade: t.IsPaperTrade,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAllTrades(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 100)

	mode, err := parseTradeMode(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	trades, err := s.tradeRepo.GetAll(r.Context(), limit, mode)
	if err != nil {
		fmt.Printf("Error fetching all trades: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch trades")
		return
	}
	writeJSON(w, http.StatusOK, trades)
}

func (s *Server) handleTradeStats(w http.ResponseWriter, r *http.Request) {
	mode, err := parseTradeMode(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := s.tradeRepo.GetStats(r.Context(), mode)
	if err != nil {
		fmt.Printf("Error fetching trade stats: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch trade stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
