package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"go.uber.org/zap"
)

func newModelProxyTestHandler(t *testing.T, backend http.Handler, upstream http.Handler) http.Handler {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on backend socket: %v", err)
	}
	server := &http.Server{Handler: backend}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
	})

	proxyAPI := httptest.NewServer(upstream)
	t.Cleanup(proxyAPI.Close)
	handler, err := newModelCallProxy(&config.Config{
		BackendSocketPath:  socketPath,
		CLIProxyAPIBaseURL: proxyAPI.URL,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("new model-call proxy: %v", err)
	}
	return handler
}

func modelProxyBackend(ownerID string, evaluate func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-session" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/auth/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user_id":"owner-1"}`))
		case strings.HasPrefix(r.URL.Path, "/api/dve-creation/nodes/creation-1/policy/evaluate"):
			evaluate(w, r)
		case r.URL.Path == "/api/dve-creation/nodes/creation-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"owner_id":"` + ownerID + `","policy":{"fail_open":false}}}`))
		default:
			http.NotFound(w, r)
		}
	})
}

func TestModelCallProxyAllowsCreationOwnerAndForwardsRequest(t *testing.T) {
	upstreamCalls := 0
	handler := newModelProxyTestHandler(t, modelProxyBackend("owner-1", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		if r.URL.Path != "/v1/messages" {
			t.Errorf("upstream path = %q, want /v1/messages", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer valid-session" {
			t.Errorf("authorization was not forwarded")
		}
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, "/gateway/model-proxy/creation-1/anthropic/v1/messages", strings.NewReader(`{"model":"claude-test"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || upstreamCalls != 1 {
		t.Fatalf("status = %d, upstream calls = %d", response.Code, upstreamCalls)
	}
}

func TestModelCallProxyRejectsNonOwnerBeforePolicyOrUpstream(t *testing.T) {
	policyCalls, upstreamCalls := 0, 0
	handler := newModelProxyTestHandler(t, modelProxyBackend("another-owner", func(w http.ResponseWriter, _ *http.Request) {
		policyCalls++
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { upstreamCalls++ }))

	request := httptest.NewRequest(http.MethodPost, "/gateway/model-proxy/creation-1/openai/v1/chat/completions", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || policyCalls != 0 || upstreamCalls != 0 {
		t.Fatalf("status = %d, policy calls = %d, upstream calls = %d", response.Code, policyCalls, upstreamCalls)
	}
}

func TestModelCallProxyBlocksDeniedPolicy(t *testing.T) {
	upstreamCalls := 0
	handler := newModelProxyTestHandler(t, modelProxyBackend("owner-1", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"allowed":false,"reason":"model denied"}`))
	}), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { upstreamCalls++ }))

	request := httptest.NewRequest(http.MethodPost, "/gateway/model-proxy/creation-1/openai/v1/chat/completions", strings.NewReader(`{"model":"blocked"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || upstreamCalls != 0 {
		t.Fatalf("status = %d, upstream calls = %d", response.Code, upstreamCalls)
	}
}
