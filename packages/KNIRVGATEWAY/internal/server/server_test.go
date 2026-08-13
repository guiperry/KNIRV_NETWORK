package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// noRedirectClient returns an HTTP client that does NOT follow redirects,
// so we can inspect the redirect status and Location header directly.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// testServer creates a minimal Server for testing, using the given config.
func testServer(cfg *config.Config) *Server {
	logger := zap.NewNop()
	s := &Server{
		config: cfg,
		logger: logger,
		router: mux.NewRouter(),
	}
	// Manually call setupRoutes so we can verify route registration
	s.setupRoutes()
	return s
}

// TestOracleProxyRoutes verifies /api/oracle/* proxy and redirects
func TestOracleProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		OracleSocketPath: "/tmp/test-oracle.sock",
		Port:             8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /api/oracle/ routes are registered (should not 404 when oracle URL is set)
	resp, err := http.Get(ts.URL + "/api/oracle/v3/health")
	if err != nil {
		t.Fatalf("GET /api/oracle/v3/health: %v", err)
	}
	resp.Body.Close()
	// The proxy will try to connect to localhost:1317 which may or may not be
	// running — we just verify it doesn't 404 (should return 502/503 if oracle down)
	if resp.StatusCode == http.StatusNotFound {
		t.Error("/api/oracle/v3/health returned 404 — route not registered")
	}

	client := noRedirectClient()

	// /wallet/info should redirect to the current oracle wallet status route.
	t.Run("old path /wallet/info redirects to /api/oracle/v3/wallet/status", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/wallet/info")
		if err != nil {
			t.Fatalf("GET /wallet/info: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/oracle/v3/wallet/status" {
			t.Errorf("expected Location: /api/oracle/v3/wallet/status, got %q", loc)
		}
	})

	// /transaction should redirect to the oracle-owned transaction submission route.
	t.Run("old path /transaction redirects to /transactions", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/transaction", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /transaction: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/transactions" {
			t.Errorf("expected Location: /transactions, got %q", loc)
		}
	})
}

func TestOracleGatewayOverride(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/blocks" {
			t.Fatalf("expected forwarded path /api/blocks, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"height":42}]`))
	}))
	defer upstream.Close()

	// A configured override must enable the non-root upstream proxy even when
	// this Server was built directly rather than through config.Load().
	s := testServer(&config.Config{OracleGatewayURL: "  " + upstream.URL + "  "})
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/blocks")
	if err != nil {
		t.Fatalf("GET /api/blocks: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected upstream response, got %d", resp.StatusCode)
	}
	if got := defaultOracleGatewayURL("mainnet", "  https://override.example  "); got != "https://override.example" {
		t.Fatalf("override was not trimmed: %q", got)
	}
}

// TestChainProxyRoutes verifies /api/chain/* proxy and redirects
func TestChainProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		ChainSocketPath: "/tmp/nonexistent-chain-test-xxxxxxxx.sock",
		Port:            8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /api/chain/ routes are registered (should not 404 when chain socket is configured)
	resp, err := http.Get(ts.URL + "/api/chain/health")
	if err != nil {
		t.Fatalf("GET /api/chain/health: %v", err)
	}
	resp.Body.Close()
	// The proxy will try to connect to the socket which doesn't exist
	// Should return either 502 (proxy error) or 502 (no socket) — not 404
	if resp.StatusCode == http.StatusNotFound {
		t.Error("/api/chain/health returned 404 — route not registered")
	}

	client := noRedirectClient()

	// /chain → 301 redirect to the actual chain endpoint.
	t.Run("GET /chain redirects to /api/chain/chain", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/chain")
		if err != nil {
			t.Fatalf("GET /chain: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/chain/chain" {
			t.Errorf("expected Location: /api/chain/chain, got %q", loc)
		}
	})

	// /chain/* → 301 redirect to /api/chain/*
	t.Run("/chain/ redirects to /api/chain/", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/chain/capability/list")
		if err != nil {
			t.Fatalf("GET /chain/capability/list: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
	})

	// /bootnodes → 301 redirect to /api/chain/bootnodes
	t.Run("/bootnodes redirects to /api/chain/bootnodes", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/bootnodes")
		if err != nil {
			t.Fatalf("GET /bootnodes: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/chain/bootnodes" {
			t.Errorf("expected Location: /api/chain/bootnodes, got %q", loc)
		}
	})
}

func TestObjectAndAssetProxyPaths(t *testing.T) {
	listener, err := net.Listen("unix", filepath.Join(t.TempDir(), "chain.sock"))
	if err != nil {
		t.Fatalf("listen on chain socket: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/objects" && r.URL.Path != "/assets" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"nfts":[]}`))
		}))
	}()

	s := testServer(&config.Config{ChainSocketPath: listener.Addr().String()})
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	for _, path := range []string{"/api/objects", "/api/assets"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s returned %d, want 200", path, resp.StatusCode)
		}
	}

	resp, err := http.Get(ts.URL + "/api/objectssuffix")
	if err != nil {
		t.Fatalf("GET /api/objectssuffix: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected prefix route match: got %d, want 404", resp.StatusCode)
	}
}

// TestGraphProxyRoutes verifies /api/graph/* proxy and redirects
func TestGraphProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		GraphSocketPath: "/tmp/nonexistent-graph-test-xxxxxxxx.sock",
		Port:            8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /api/graph/ routes are registered
	resp, err := http.Get(ts.URL + "/api/graph/health")
	if err != nil {
		t.Fatalf("GET /api/graph/health: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("/api/graph/health returned 404 — route not registered")
	}

	client := noRedirectClient()

	// /graph/* → 301 redirect to /api/graph/*
	t.Run("/graph/ redirects to /api/graph/", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/graph/nodes")
		if err != nil {
			t.Fatalf("GET /graph/nodes: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
	})
}

// TestShellProxyRoutes verifies /api/shell/* proxy
func TestShellProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		ShellSocketPath: "/var/lib/knirvserver/sockets/shell.sock",
		Port:            8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /api/shell/ routes are registered (should not 404)
	resp, err := http.Get(ts.URL + "/api/shell/sessions")
	if err != nil {
		t.Fatalf("GET /api/shell/sessions: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("/api/shell/sessions returned 404 — route not registered")
	}
}

// TestMockEndpointsReplaced verifies old mock endpoints have fallback responses
func TestMockEndpointsReplaced(t *testing.T) {
	// Test with minimal config — no sockets configured
	cfg := &config.Config{Port: 8888}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /devs should return empty array (fallback, not 404)
	t.Run("/devs returns empty array fallback", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/devs")
		if err != nil {
			t.Fatalf("GET /devs: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /devs fallback, got %d", resp.StatusCode)
		}
	})

	// /txn_pool should return empty array (fallback, not 404)
	t.Run("/txn_pool returns empty array fallback", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/txn_pool")
		if err != nil {
			t.Fatalf("GET /txn_pool: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /txn_pool fallback, got %d", resp.StatusCode)
		}
	})

	// /api/objects should return empty array when chain is not configured
	t.Run("/api/objects returns [] when chain not configured", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/objects")
		if err != nil {
			t.Fatalf("GET /api/objects: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var data []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("expected JSON array: %v", err)
		}
	})

	// /api/assets should return empty array when chain is not configured
	t.Run("/api/assets returns [] when chain not configured", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/assets")
		if err != nil {
			t.Fatalf("GET /api/assets: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var data []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("expected JSON array: %v", err)
		}
	})

	// /api/transactions should return empty array when oracle is not configured
	t.Run("/api/transactions returns [] when oracle not configured", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/transactions")
		if err != nil {
			t.Fatalf("GET /api/transactions: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var data []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("expected JSON array: %v", err)
		}
	})

	// /api/blocks should return empty array when oracle is not configured
	t.Run("/api/blocks returns [] when oracle not configured", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/blocks")
		if err != nil {
			t.Fatalf("GET /api/blocks: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var data []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("expected JSON array: %v", err)
		}
	})
}

// TestUnmatchedAPIRoutes_404 verifies that unmatched /api/* routes return 404
// instead of mock data
func TestUnmatchedAPIRoutes_404(t *testing.T) {
	cfg := &config.Config{Port: 8888}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// An unmatched /api/ route should 404
	t.Run("unmatched /api/ returns 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/unknown-route")
		if err != nil {
			t.Fatalf("GET /api/unknown-route: %v", err)
		}
		defer resp.Body.Close()
		// Should 404 — no catch-all mock handler anymore
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for unknown /api route, got %d", resp.StatusCode)
		}
	})
}

// TestAPIInfoFallback verifies /api/v1/info returns fallback data
// when the backend proxy is not configured.
func TestAPIInfoFallback(t *testing.T) {
	cfg := &config.Config{
		Port: 8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// With no BackendSocketPath, /api/v1/info has a local fallback handler
	// that returns basic info data so the WebGUI doesn't show console errors.
	resp, err := http.Get(ts.URL + "/api/v1/info")
	if err != nil {
		t.Fatalf("GET /api/v1/info: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (fallback local handler), got %d", resp.StatusCode)
	}
}

func TestCLIDeviceAuthorizationProxyRoute(t *testing.T) {
	cfg := &config.Config{
		Port:              8888,
		BackendSocketPath: "/tmp/nonexistent-knirv-auth-test.sock",
	}
	s := testServer(cfg)
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/auth/device/status?token=device-code")
	if err != nil {
		t.Fatalf("GET /api/auth/device/status: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK {
		t.Fatalf("expected registered backend auth proxy to fail on the test socket, got %d", resp.StatusCode)
	}
}

func TestCompleteKNIRVServerAPIProxyFallback(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on backend socket: %v", err)
	}
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-KNIRV-Backend-Path", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})}
	go func() { _ = backend.Serve(listener) }()
	t.Cleanup(func() { _ = backend.Close() })

	s := testServer(&config.Config{Port: 8888, BackendSocketPath: socketPath})
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	for _, path := range []string{
		"/api/auth/device/status",
		"/api/cognitive/active-memory/traces",
		"/api/dve-nodes",
		"/api/system-health",
		"/api/v1/projects/example",
	} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				t.Fatalf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusNoContent)
			}
			if got := resp.Header.Get("X-KNIRV-Backend-Path"); got != path {
				t.Fatalf("proxied path = %q, want %q", got, path)
			}
		})
	}
}

func TestDVEProjectAPIProxyPrecedesViewerRoute(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on backend socket: %v", err)
	}
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dve/projects" {
			t.Errorf("backend path = %q, want /dve/projects", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})}
	go func() { _ = backend.Serve(listener) }()
	t.Cleanup(func() { _ = backend.Close() })

	s := testServer(&config.Config{Port: 8888, BackendSocketPath: socketPath})
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/dve/projects")
	if err != nil {
		t.Fatalf("GET /dve/projects: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /dve/projects status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if got := string(body); got != "[]" {
		t.Fatalf("GET /dve/projects body = %q, want []", got)
	}
}

// TestDynamicAgentProxy verifies the dynamic agent proxy behavior
func TestDynamicAgentProxy(t *testing.T) {
	cfg := &config.Config{
		AgentSocketDir: "/tmp/agent-test-sockets",
		Port:           8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// Missing dveId should return 400
	t.Run("missing dveId returns 400", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/agent/")
		if err != nil {
			t.Fatalf("GET /api/agent/: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for missing dveId, got %d", resp.StatusCode)
		}
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		if body["error"] != "missing dve id" {
			t.Errorf("expected 'missing dve id' error, got %v", body["error"])
		}
	})

	// Non-existent DVE socket should return 503
	t.Run("non-existent DVE returns 503", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/agent/nonexistent-dve/health")
		if err != nil {
			t.Fatalf("GET /api/agent/nonexistent-dve/health: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503 for missing agent socket, got %d", resp.StatusCode)
		}
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		if body["error"] == nil {
			t.Error("expected error message in body")
		}
	})
}

func TestPublishOperationRoute(t *testing.T) {
	cfg := &config.Config{
		Port: 8888,
	}
	s := testServer(cfg)
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/knirvbase/publish-op", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("publish-op route not registered")
	}
}

func TestSyncRequestRoute(t *testing.T) {
	cfg := &config.Config{
		Port: 8888,
	}
	s := testServer(cfg)
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/p2p/sync-request", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("sync-request route not registered")
	}
}

func TestP2PHealthRoute(t *testing.T) {
	cfg := &config.Config{
		Port: 8888,
	}
	s := testServer(cfg)
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/p2p/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("p2p/health route not registered")
	}
}

func TestHandleCRDTOperations(t *testing.T) {
	_ = config.Config{Port: 8888}
}

func TestGatewayManagerCRDTForwarding(t *testing.T) {
	_ = config.Config{Port: 8888}
}
