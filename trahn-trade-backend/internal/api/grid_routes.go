package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type gridCurrentResponse struct {
	BasePrice      *float64        `json:"basePrice"`
	Grid           json.RawMessage `json:"grid"`
	TradesExecuted int             `json:"tradesExecuted"`
	TotalProfit    float64         `json:"totalProfit"`
	LastUpdate     *string         `json:"lastUpdate,omitempty"`
}

func (s *Server) handleGridCurrent(w http.ResponseWriter, r *http.Request) {
	state, err := s.gridRepo.GetActive(r.Context())
	if err != nil {
		fmt.Printf("Error fetching grid state: %v\n", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if state == nil {
		writeJSON(w, http.StatusOK, gridCurrentResponse{
			Grid: json.RawMessage("[]"),
		})
		return
	}

	grid := state.GridLevelsJSON
	if grid == nil {
		grid = json.RawMessage("[]")
	}

	var lastUpdate *string
	ts := state.UpdatedAt.Format("2006-01-02T15:04:05.000Z")
	lastUpdate = &ts

	writeJSON(w, http.StatusOK, gridCurrentResponse{
		BasePrice:      state.BasePrice,
		Grid:           grid,
		TradesExecuted: state.TradesExecuted,
		TotalProfit:    state.TotalProfit,
		LastUpdate:     lastUpdate,
	})
}
