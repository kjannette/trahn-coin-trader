package api

import (
	"fmt"
	"net/http"

	"github.com/kjannette/trahn-backend/internal/repository"
)

type priceJSON struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

func (s *Server) handlePricesToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := repository.TradingDayNow()
	prices, err := s.priceRepo.GetByDay(ctx, today)
	if err != nil {
		fmt.Printf("Error fetching today's prices: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch prices")
		return
	}

	out := make([]priceJSON, len(prices))
	for i, p := range prices {
		out[i] = priceJSON{T: p.Timestamp.UnixMilli(), P: p.Price}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePricesByDay(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if !validateDate(date) {
		writeError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	ctx := r.Context()
	prices, err := s.priceRepo.GetByDay(ctx, date)
	if err != nil {
		fmt.Printf("Error fetching prices for %s: %v\n", date, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch prices")
		return
	}

	out := make([]priceJSON, len(prices))
	for i, p := range prices {
		out[i] = priceJSON{T: p.Timestamp.UnixMilli(), P: p.Price}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAvailableDays(w http.ResponseWriter, r *http.Request) {
	days, err := s.priceRepo.GetAvailableDays(r.Context())
	if err != nil {
		fmt.Printf("Error fetching available days: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch available days")
		return
	}
	if days == nil {
		days = []string{}
	}
	writeJSON(w, http.StatusOK, days)
}

func (s *Server) handleLatestPrice(w http.ResponseWriter, r *http.Request) {
	price, err := s.priceRepo.GetLatest(r.Context())
	if err != nil {
		fmt.Printf("Error fetching latest price: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch latest price")
		return
	}
	if price == nil {
		writeError(w, http.StatusNotFound, "no price data available")
		return
	}
	writeJSON(w, http.StatusOK, priceJSON{T: price.Timestamp.UnixMilli(), P: price.Price})
}
