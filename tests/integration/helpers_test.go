package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
)

// baseURL returns the base URL for a service running on the given port.
func baseURL(port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}

// uniqueEmail generates a unique email address to avoid test collisions.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d-%d@test.example.com", prefix, time.Now().UnixNano(), rand.Intn(100000))
}

// uniqueSlug generates a unique slug to avoid test collisions.
func uniqueSlug(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), rand.Intn(100000))
}

// uniqueUUID generates a deterministic-looking UUID v4 for test data.
// This uses a simple random approach; not cryptographically secure but fine for tests.
func uniqueUUID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// skipIfNotRunning performs a quick health check against a service.
// If the service is unreachable, the test is skipped (not failed).
func skipIfNotRunning(t *testing.T, port int) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL(port) + "/health/live")
	if err != nil {
		t.Skipf("service on port %d not reachable (Docker not running?): %v", port, err)
	}
	resp.Body.Close()
}

// httpGet performs an HTTP GET request and returns the status code and decoded JSON body.
func httpGet(t *testing.T, url string) (int, map[string]interface{}) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeBody(t, resp.Body)
}

// httpGetWithAuth performs an HTTP GET request with a Bearer token.
func httpGetWithAuth(t *testing.T, url string, token string) (int, map[string]interface{}) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("creating GET request for %s failed: %v", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s with auth failed: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeBody(t, resp.Body)
}

// httpPost performs an HTTP POST request with a JSON body.
func httpPost(t *testing.T, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	return doJSONRequest(t, http.MethodPost, url, body, "", nil)
}

// httpPostWithAuth performs an HTTP POST request with a JSON body and Bearer token.
func httpPostWithAuth(t *testing.T, url string, body interface{}, token string) (int, map[string]interface{}) {
	t.Helper()
	return doJSONRequest(t, http.MethodPost, url, body, token, nil)
}

// httpPut performs an HTTP PUT request with a JSON body.
func httpPut(t *testing.T, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	return doJSONRequest(t, http.MethodPut, url, body, "", nil)
}

// httpPutWithAuth performs an HTTP PUT request with a JSON body and Bearer token.
func httpPutWithAuth(t *testing.T, url string, body interface{}, token string) (int, map[string]interface{}) {
	t.Helper()
	return doJSONRequest(t, http.MethodPut, url, body, token, nil)
}

// httpPostWithHeaders performs an HTTP POST request with a JSON body and custom headers.
func httpPostWithHeaders(t *testing.T, url string, body interface{}, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	return doJSONRequest(t, http.MethodPost, url, body, "", headers)
}

// httpGetWithHeaders performs an HTTP GET request with custom headers.
func httpGetWithHeaders(t *testing.T, url string, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("creating GET request for %s failed: %v", url, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s with headers failed: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeBody(t, resp.Body)
}

// httpDeleteWithHeaders performs an HTTP DELETE request with custom headers.
func httpDeleteWithHeaders(t *testing.T, url string, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("creating DELETE request for %s failed: %v", url, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s with headers failed: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeBody(t, resp.Body)
}

// doJSONRequest is the internal helper for JSON HTTP requests.
func doJSONRequest(t *testing.T, method, url string, body interface{}, token string, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshalling request body failed: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("creating %s request for %s failed: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeBody(t, resp.Body)
}

// decodeBody reads the response body and attempts to decode it as JSON.
// If the body is empty or not JSON, it returns an empty map.
func decodeBody(t *testing.T, body io.Reader) map[string]interface{} {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		// Not JSON; return the raw string in a "raw" key for debugging.
		return map[string]interface{}{"raw": string(raw)}
	}
	return result
}

// requireStatus asserts that the HTTP status code matches the expected value.
func requireStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("expected status %d, got %d", want, got)
	}
}

// extractField extracts a value from a nested map using a dot-separated path.
// For example, extractField(data, "data.user.id") navigates data["data"]["user"]["id"].
func extractField(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// extractString is a convenience wrapper around extractField that returns a string.
func extractString(t *testing.T, data map[string]interface{}, path string) string {
	t.Helper()
	val := extractField(data, path)
	if val == nil {
		t.Fatalf("expected string at path %q, got nil", path)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string at path %q, got %T: %v", path, val, val)
	}
	return s
}

// extractFloat is a convenience wrapper that returns a float64.
func extractFloat(t *testing.T, data map[string]interface{}, path string) float64 {
	t.Helper()
	val := extractField(data, path)
	if val == nil {
		t.Fatalf("expected number at path %q, got nil", path)
	}
	f, ok := val.(float64)
	if !ok {
		t.Fatalf("expected float64 at path %q, got %T: %v", path, val, val)
	}
	return f
}
