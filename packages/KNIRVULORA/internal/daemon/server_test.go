package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"ulora/internal/api"
	"ulora/internal/config"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")
	return &config.Config{
		SocketPath:     socketPath,
		AuthToken:      "test-token-123",
		DataDir:        tmpDir,
		PythonBin:      "python3",
		VenvDir:        "",
		RequestTimeout: 30,
	}
}

func unixClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

func callSocket(t *testing.T, cfg *config.Config, method, route string, body []byte) (*http.Response, error) {
	t.Helper()
	client := unixClient(cfg.SocketPath)
	url := "http://unix" + route
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(context.Background(), method, url, nil)
	}
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	return client.Do(req)
}

func startTestServer(t *testing.T) (*Server, *config.Config, func()) {
	t.Helper()
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir datadir: %v", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	return srv, cfg, func() {
		srv.Shutdown()
		_ = <-errCh
	}
}

func TestHealthEndpoint(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := callSocket(t, cfg, "GET", "/ulora/v1/health", nil)
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var health api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if health.Status != "ok" {
		t.Fatalf("expected status 'ok', got %s", health.Status)
	}
	if health.Timestamp == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown()
	go srv.Serve()

	client := unixClient(cfg.SocketPath)

	resp, err := client.Get("http://unix/ulora/v1/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health should not require auth, got %d", resp.StatusCode)
	}

	authReq, _ := http.NewRequest("POST", "http://unix/ulora/v1/validate-manifest", bytes.NewReader([]byte("{}")))
	resp2, err := client.Do(authReq)
	if err != nil {
		t.Fatalf("auth request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", resp2.StatusCode)
	}
}

func TestAuthMiddlewareRejectsWrongToken(t *testing.T) {
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown()
	go srv.Serve()

	client := unixClient(cfg.SocketPath)

	req, _ := http.NewRequest("POST", "http://unix/ulora/v1/validate-manifest", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong token, got %d", resp.StatusCode)
	}
}

func TestNewServerRemovesStaleSocket(t *testing.T) {
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(cfg.SocketPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale socket: %v", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown()
	go srv.Serve()
}

func TestServerStructCreation(t *testing.T) {
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown()

	if srv.cfg != cfg {
		t.Fatal("expected cfg to match")
	}
	if srv.compiler == nil {
		t.Fatal("expected compiler to be initialized")
	}
	if srv.server == nil {
		t.Fatal("expected http server to be initialized")
	}
	if srv.ln == nil {
		t.Fatal("expected listener to be initialized")
	}
}

func TestSetEnginesDir(t *testing.T) {
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown()

	srv.SetEnginesDir("/custom/engines/path")
}

func TestValidateManifestEndpointInvalid(t *testing.T) {
	srv, cfg, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := callSocket(t, cfg, "POST", "/ulora/v1/validate-manifest", []byte(`{"$schema":"wrong"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid manifest, got %d", resp.StatusCode)
	}
	_ = srv
}
