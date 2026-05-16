package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_server/internal/web"

	"github.com/gorilla/mux"
)

// ── Mock ─────────────────────────────────────────────────────────────────────

type mockInnerAgentMgr struct {
	client     *http.Client
	socketPath string
	err        error
}

func (m *mockInnerAgentMgr) InnerAgentClient(dveID string) (*http.Client, string, error) {
	return m.client, m.socketPath, m.err
}

// ── Constructor ──────────────────────────────────────────────────────────────

func TestNewDVEInnerAgentWSHandler(t *testing.T) {
	h := web.NewDVEInnerAgentWSHandler(nil)
	if h == nil {
		t.Fatal("NewDVEInnerAgentWSHandler returned nil")
	}
}

// ── HandleWebSocket – missing dveId → 400 ────────────────────────────────────

func TestDVEInnerAgentWSHandler_HandleWS_NoDVEID(t *testing.T) {
	h := web.NewDVEInnerAgentWSHandler(nil)

	// Request without mux vars — dveId will be empty
	req := httptest.NewRequest("GET", "/ws/dve//inner", nil)
	w := httptest.NewRecorder()

	h.HandleWebSocket(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 BadRequest, got %d", resp.StatusCode)
	}
}

// ── HandleWebSocket – nil agent manager → 503 ────────────────────────────────

func TestDVEInnerAgentWSHandler_HandleWS_NoAgent(t *testing.T) {
	h := web.NewDVEInnerAgentWSHandler(nil)

	// Set up a request with a dveId but handler has nil agentManager
	req := httptest.NewRequest("GET", "/ws/dve/dve-test/inner", nil)
	req = mux.SetURLVars(req, map[string]string{"dveId": "dve-test"})
	w := httptest.NewRecorder()

	h.HandleWebSocket(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 ServiceUnavailable, got %d", resp.StatusCode)
	}
}
