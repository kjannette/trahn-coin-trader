package api

import (
	"net/http"
	"time"
)

type healthResponse struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Services  healthServices `json:"services"`
}

type healthServices struct {
	Database string `json:"database"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := s.pool.Ping(r.Context()); err != nil {
		dbStatus = "disconnected"
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  healthServices{Database: dbStatus},
	})
}
