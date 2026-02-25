package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// healthResponse is the JSON structure for health check responses.
type healthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// LivenessHandler returns a handler that responds with 200 OK if the gateway process is alive.
// No dependency checks are performed; if the process can handle the request, it is alive.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:    "up",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ReadinessHandler returns a handler that responds with 200 OK.
// The gateway has no database or stateful dependencies, so readiness
// is equivalent to liveness.
func ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:    "up",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}
