package web_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/web"

	"github.com/gorilla/mux"
)

// ── Mock ─────────────────────────────────────────────────────────────────────

type mockInnerAgentProxy struct {
	resp *http.Response
	err  error
	path string
}

func (m *mockInnerAgentProxy) ForwardToInnerAgent(dveID, method, path string, body io.Reader) (*http.Response, error) {
	return m.resp, m.err
}

func (m *mockInnerAgentProxy) GetSocketPath(dveID string) (string, error) {
	return m.path, m.err
}

func (m *mockInnerAgentProxy) StartAgent(context.Context, string, time.Duration) error {
	return nil
}

// ── Constructor ──────────────────────────────────────────────────────────────

func TestNewInnerAgentHandlers(t *testing.T) {
	proxy := &mockInnerAgentProxy{}
	h := web.NewInnerAgentHandlers(proxy)
	if h == nil {
		t.Fatal("NewInnerAgentHandlers returned nil")
	}
}

// ── RegisterRoutes ───────────────────────────────────────────────────────────

func TestRegisterRoutes(t *testing.T) {
	proxy := &mockInnerAgentProxy{}
	h := web.NewInnerAgentHandlers(proxy)

	router := mux.NewRouter()

	// Must not panic when registering routes
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RegisterRoutes panicked: %v", r)
		}
	}()

	h.RegisterRoutes(router)

	// Walk the router to verify routes were actually registered
	var found int
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		if path != "" {
			found++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("router.Walk failed: %v", err)
	}
	if found == 0 {
		t.Error("expected at least one route to be registered, got 0")
	}
}

// ── Forward error → 502 BadGateway ───────────────────────────────────────────

func TestInnerAgentHandlers_Forward_Error(t *testing.T) {
	proxy := &mockInnerAgentProxy{
		err: errors.New("connection refused"),
	}
	h := web.NewInnerAgentHandlers(proxy)

	// The /spawn handler calls forward() which calls proxy.ForwardToInnerAgent
	req := httptest.NewRequest("POST", "/api/v1/dve/dve-err/inner/spawn", nil)
	req = mux.SetURLVars(req, map[string]string{"dveId": "dve-err"})
	w := httptest.NewRecorder()

	h.HandleSpawn(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502 BadGateway, got %d", resp.StatusCode)
	}
}
