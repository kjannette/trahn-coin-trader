package api

import (
	"fmt"
	"net/http"
	"time"
)

type srLatestResponse struct {
	Support      float64   `json:"support"`
	Resistance   float64   `json:"resistance"`
	Midpoint     float64   `json:"midpoint"`
	AvgPrice     *float64  `json:"avgPrice"`
	Method       string    `json:"method"`
	LookbackDays int       `json:"lookbackDays"`
	Timestamp    time.Time `json:"timestamp"`
}

func (s *Server) handleSRLatest(w http.ResponseWriter, r *http.Request) {
	sr, err := s.srRepo.GetLatest(r.Context())
	if err != nil {
		fmt.Printf("Error fetching latest S/R: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch S/R data")
		return
	}
	if sr == nil {
		writeError(w, http.StatusNotFound, "no S/R data available")
		return
	}

	writeJSON(w, http.StatusOK, srLatestResponse{
		Support:      sr.Support,
		Resistance:   sr.Resistance,
		Midpoint:     sr.Midpoint,
		AvgPrice:     sr.AvgPrice,
		Method:       sr.Method,
		LookbackDays: sr.LookbackDays,
		Timestamp:    sr.Timestamp,
	})
}

func (s *Server) handleSRHistory(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 100)

	history, err := s.srRepo.GetHistory(r.Context(), limit)
	if err != nil {
		fmt.Printf("Error fetching S/R history: %v\n", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch S/R history")
		return
	}
	writeJSON(w, http.StatusOK, history)
}
