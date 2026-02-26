package integration

import (
	"net/http"
	"testing"
	"time"
)

// TestAllServicesHealthy checks the /health/live endpoint for all 12 services.
// Each service is tested as a subtest so failures are reported individually.
// If a service is unreachable, the subtest is skipped (not failed), allowing
// the suite to run in environments where only some services are up.
func TestAllServicesHealthy(t *testing.T) {
	services := map[string]int{
		"product":      8001,
		"cart":         8002,
		"order":        8003,
		"checkout":     8004,
		"payment":      8005,
		"user":         8006,
		"inventory":    8007,
		"campaign":     8008,
		"notification": 8009,
		"search":       8010,
		"media":        8011,
		"gateway":      8080,
	}

	client := &http.Client{Timeout: 3 * time.Second}

	for name, port := range services {
		t.Run(name, func(t *testing.T) {
			url := baseURL(port) + "/health/live"
			resp, err := client.Get(url)
			if err != nil {
				t.Skipf("service %s on port %d not reachable: %v", name, port, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("service %s health check returned %d, want 200", name, resp.StatusCode)
			}
		})
	}
}

// TestAllServicesReady checks the /health/ready endpoint for all 12 services.
func TestAllServicesReady(t *testing.T) {
	services := map[string]int{
		"product":      8001,
		"cart":         8002,
		"order":        8003,
		"checkout":     8004,
		"payment":      8005,
		"user":         8006,
		"inventory":    8007,
		"campaign":     8008,
		"notification": 8009,
		"search":       8010,
		"media":        8011,
		"gateway":      8080,
	}

	client := &http.Client{Timeout: 3 * time.Second}

	for name, port := range services {
		t.Run(name, func(t *testing.T) {
			url := baseURL(port) + "/health/ready"
			resp, err := client.Get(url)
			if err != nil {
				t.Skipf("service %s on port %d not reachable: %v", name, port, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("service %s readiness check returned %d, want 200", name, resp.StatusCode)
			}
		})
	}
}
