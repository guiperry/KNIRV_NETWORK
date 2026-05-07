package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		config:            cfg,
		logger:            logger,
		router:            mux.NewRouter(),
	}
	// Manually call setupRoutes so we can verify route registration
	s.setupRoutes()
	return s
}

// TestOracleProxyRoutes verifies /api/oracle/* proxy and redirects
func TestOracleProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		KnirvOracleURL: "http://localhost:1317",
		Port:           8888,
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

	// /wallet/info should redirect to /api/oracle/wallet/info (301)
	t.Run("old path /wallet/info redirects to /api/oracle/wallet/info", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/wallet/info")
		if err != nil {
			t.Fatalf("GET /wallet/info: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/oracle/wallet/info" {
			t.Errorf("expected Location: /api/oracle/wallet/info, got %q", loc)
		}
	})

	// /transaction should redirect to /api/oracle/transaction (301)
	t.Run("old path /transaction redirects to /api/oracle/transaction", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/transaction", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /transaction: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/oracle/transaction" {
			t.Errorf("expected Location: /api/oracle/transaction, got %q", loc)
		}
	})
}

// TestChainProxyRoutes verifies /api/chain/* proxy and redirects
func TestChainProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		ChainSocketPath: "/var/lib/knirvserver/sockets/chain.sock",
		Port:            8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /api/chain/ routes are registered
	resp, err := http.Get(ts.URL + "/api/chain/health")
	if err != nil {
		t.Fatalf("GET /api/chain/health: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Error("/api/chain/health returned 404 — route not registered")
	}

	client := noRedirectClient()

	// /chain → 301 redirect to /api/chain/
	t.Run("GET /chain redirects to /api/chain/", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/chain")
		if err != nil {
			t.Fatalf("GET /chain: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/api/chain/" {
			t.Errorf("expected Location: /api/chain/, got %q", loc)
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

// TestGraphProxyRoutes verifies /api/graph/* proxy and redirects
func TestGraphProxyRoutes(t *testing.T) {
	cfg := &config.Config{
		GraphSocketPath: "/var/lib/knirvserver/sockets/graph.sock",
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

// TestMockEndpointsReplaced verifies old mock endpoints are gone
func TestMockEndpointsReplaced(t *testing.T) {
	// Test with minimal config — no sockets configured
	cfg := &config.Config{Port: 8888}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// /devs should 404 (removed, no redirect needed per spec)
	t.Run("/devs returns 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/devs")
		if err != nil {
			t.Fatalf("GET /devs: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for removed /devs, got %d", resp.StatusCode)
		}
	})

	// /txn_pool should 404 (removed)
	t.Run("/txn_pool returns 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/txn_pool")
		if err != nil {
			t.Fatalf("GET /txn_pool: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for removed /txn_pool, got %d", resp.StatusCode)
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

// TestAPIInfoNotLocal verifies /api/v1/info is NOT handled locally
// (handled by backend proxy instead)
func TestAPIInfoNotLocal(t *testing.T) {
	cfg := &config.Config{
		Port: 8888,
	}
	s := testServer(cfg)

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	// With no BackendSocketPath, /api/v1/* won't have a backend proxy,
	// but /api/v1/info should NOT have a local handler — this route
	// now falls through to the backend proxy.
	resp, err := http.Get(ts.URL + "/api/v1/info")
	if err != nil {
		t.Fatalf("GET /api/v1/info: %v", err)
	}
	defer resp.Body.Close()
	// When no backend is configured, /api/v1/* routes have no handler
	// (they're only registered inside the BackendSocketPath conditional)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 (no local /api/v1/info handler), got %d", resp.StatusCode)
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
		if body["error"] != "missing dveId" {
			t.Errorf("expected 'missing dveId' error, got %v", body["error"])
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
