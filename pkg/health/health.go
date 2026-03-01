package health

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// Build-time variables set via ldflags:
//
//	-X github.com/utafrali/EcommerceGo/pkg/health.gitCommit=<sha>
//	-X github.com/utafrali/EcommerceGo/pkg/health.buildTime=<time>
var (
	gitCommit string
	buildTime string
)

// BuildInfo holds version and build metadata embedded in the binary.
type BuildInfo struct {
	GitCommit string `json:"git_commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
}

// resolveBuildInfo populates build metadata from ldflags and runtime/debug,
// preferring ldflags values when available.
func resolveBuildInfo() BuildInfo {
	info := BuildInfo{
		GoVersion: runtime.Version(),
		GitCommit: gitCommit,
		BuildTime: buildTime,
	}

	// Fallback to VCS info embedded by go build.
	if info.GitCommit == "" || info.BuildTime == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if info.GitCommit == "" {
						v := s.Value
						if len(v) > 8 {
							v = v[:8]
						}
						info.GitCommit = v
					}
				case "vcs.time":
					if info.BuildTime == "" {
						info.BuildTime = s.Value
					}
				}
			}
		}
	}

	return info
}

// Checker is a function that checks the health of a dependency.
type Checker func(ctx context.Context) error

// Status represents the health status of a component.
type Status string

const (
	StatusUp       Status = "up"
	StatusDown     Status = "down"
	StatusDegraded Status = "degraded"
)

// Response is the JSON response returned by the health endpoint.
type Response struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Build     *BuildInfo             `json:"build,omitempty"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult is the result of a single health check.
type CheckResult struct {
	Status   Status `json:"status"`
	Error    string `json:"error,omitempty"`
	Critical bool   `json:"critical"`
}

// checkerEntry stores a checker along with its criticality classification.
type checkerEntry struct {
	checker  Checker
	critical bool
}

// Handler provides HTTP health check endpoints.
type Handler struct {
	mu        sync.RWMutex
	checkers  map[string]checkerEntry
	buildInfo BuildInfo
}

// NewHandler creates a new health check handler. Build metadata is
// automatically resolved from ldflags and runtime/debug.
func NewHandler() *Handler {
	return &Handler{
		checkers:  make(map[string]checkerEntry),
		buildInfo: resolveBuildInfo(),
	}
}

// Register adds a named health checker. For backwards compatibility, checks
// registered with Register are treated as critical.
func (h *Handler) Register(name string, checker Checker) {
	h.RegisterCritical(name, checker)
}

// RegisterCritical adds a named health checker classified as critical.
// If a critical check fails, the readiness endpoint returns 503.
func (h *Handler) RegisterCritical(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checkerEntry{checker: checker, critical: true}
}

// RegisterNonCritical adds a named health checker classified as non-critical.
// If only non-critical checks fail, the readiness endpoint returns 200 with
// status "degraded" and a list of degraded components.
func (h *Handler) RegisterNonCritical(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checkerEntry{checker: checker, critical: false}
}

// LivenessHandler returns a simple liveness check (always 200 if the process is running).
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Response{
			Status:    StatusUp,
			Timestamp: time.Now().UTC(),
			Build:     &h.buildInfo,
		})
	}
}

// ReadinessHandler checks all registered dependencies and returns:
//   - 200 with status "up" if all checks pass
//   - 200 with status "degraded" if only non-critical checks fail
//   - 503 with status "down" if any critical check fails
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		h.mu.RLock()
		entries := make(map[string]checkerEntry, len(h.checkers))
		for k, v := range h.checkers {
			entries[k] = v
		}
		h.mu.RUnlock()

		checks := make(map[string]CheckResult, len(entries))
		hasCriticalFailure := false
		hasNonCriticalFailure := false

		for name, entry := range entries {
			if err := entry.checker(ctx); err != nil {
				checks[name] = CheckResult{Status: StatusDown, Error: err.Error(), Critical: entry.critical}
				if entry.critical {
					hasCriticalFailure = true
				} else {
					hasNonCriticalFailure = true
				}
			} else {
				checks[name] = CheckResult{Status: StatusUp, Critical: entry.critical}
			}
		}

		overallStatus := StatusUp
		httpStatus := http.StatusOK

		if hasCriticalFailure {
			overallStatus = StatusDown
			httpStatus = http.StatusServiceUnavailable
		} else if hasNonCriticalFailure {
			overallStatus = StatusDegraded
			// Degraded returns 200 so load balancers keep routing traffic.
		}

		resp := Response{
			Status:    overallStatus,
			Timestamp: time.Now().UTC(),
			Build:     &h.buildInfo,
			Checks:    checks,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
