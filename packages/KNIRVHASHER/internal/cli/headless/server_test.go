package headless

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testSocketPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "test.sock")
}

func TestNewServer(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.controller == nil {
		t.Error("controller is nil")
	}
	s.Stop()
	_ = os.Remove(socketPath)
}

func TestHealthEndpoint(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}

	_ = os.Remove(socketPath)
}

func TestGetStatus(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestMethodNotAllowed(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestShutdownEndpoint(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shutdown", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestServerMux(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	routes := []string{
		"/api/v1/health",
		"/api/v1/status",
		"/api/v1/driver/start",
		"/api/v1/driver/stop",
		"/api/v1/driver/status",
		"/api/v1/pipeline/run",
		"/api/v1/pipeline/status",
		// "/api/v1/pipeline/logs", // SSE endpoint - skip in test (blocks)
		"/api/v1/verify",
		"/api/v1/asic/discover",
		"/api/v1/asic/probe",
		"/api/v1/asic/protocol",
		"/api/v1/asic/provision",
		"/api/v1/asic/troubleshoot",
		"/api/v1/asic/test",
		"/api/v1/asic/configure",
		"/api/v1/rules",
		"/api/v1/chat",
		"/api/v1/shutdown",
	}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		if w.Code == 404 && w.Body.String() == "404 page not found\n" {
			t.Errorf("route %s not registered", route)
		}
	}

	_ = os.Remove(socketPath)
}

func TestSocketPermissions(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0600)
	defer s.Stop()

	// Start server to create socket file
	if err := s.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("socket file not created: %v", err)
	}

	expected := os.FileMode(0600)
	if info.Mode().Perm() != expected {
		t.Errorf("expected permissions %v, got %v", expected, info.Mode().Perm())
	}

	_ = os.Remove(socketPath)
}

func TestVerifyEndpoint(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestVerifyEndpointWithBody(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	body := `{"mode":"semantic"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestVerifyEndpointMathematicalMode(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	body := `{"mode":"mathematical"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	// Mathematical mode may fail if hasher-host binary is not available
	// Accept both 200 (started) and 500 (binary missing) as valid responses
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestVerifyEndpointInvalidMode(t *testing.T) {
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	body := `{"mode":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	_ = os.Remove(socketPath)
}

func TestIsSocketReady(t *testing.T) {
	// Test with non-existent socket
	if IsSocketReady("/nonexistent/path.sock") {
		t.Error("expected false for nonexistent socket")
	}

	// Create a real server to test IsSocketReady
	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)

	// Start server to create socket file
	if err := s.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	if !IsSocketReady(socketPath) {
		t.Error("expected true for created socket")
	}

	_ = os.Remove(socketPath)
}

func TestParseSocketPerm(t *testing.T) {
	tests := []struct {
		input    string
		expected os.FileMode
		hasError bool
	}{
		{"0660", 0660, false},
		{"0777", 0777, false},
		{"0600", 0600, false},
		{"", 0660, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := ParseSocketPerm(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("expected error for input %s", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v for input %s", tt.expected, result, tt.input)
			}
		}
	}
}