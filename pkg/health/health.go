package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Checker is a function that checks the health of a dependency.
type Checker func(ctx context.Context) error

// Status represents the health status of a component.
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// Response is the JSON response returned by the health endpoint.
type Response struct {
	Status    Status                   `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Checks    map[string]CheckResult   `json:"checks,omitempty"`
}

// CheckResult is the result of a single health check.
type CheckResult struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Handler provides HTTP health check endpoints.
type Handler struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

// NewHandler creates a new health check handler.
func NewHandler() *Handler {
	return &Handler{
		checkers: make(map[string]Checker),
	}
}

// Register adds a named health checker.
func (h *Handler) Register(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// LivenessHandler returns a simple liveness check (always 200 if the process is running).
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Status:    StatusUp,
			Timestamp: time.Now().UTC(),
		})
	}
}

// ReadinessHandler checks all registered dependencies and returns 200/503.
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		h.mu.RLock()
		checkers := make(map[string]Checker, len(h.checkers))
		for k, v := range h.checkers {
			checkers[k] = v
		}
		h.mu.RUnlock()

		checks := make(map[string]CheckResult, len(checkers))
		overallStatus := StatusUp

		for name, checker := range checkers {
			if err := checker(ctx); err != nil {
				checks[name] = CheckResult{Status: StatusDown, Error: err.Error()}
				overallStatus = StatusDown
			} else {
				checks[name] = CheckResult{Status: StatusUp}
			}
		}

		resp := Response{
			Status:    overallStatus,
			Timestamp: time.Now().UTC(),
			Checks:    checks,
		}

		w.Header().Set("Content-Type", "application/json")
		if overallStatus == StatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(resp)
	}
}
