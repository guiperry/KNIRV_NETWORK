package models

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"knirvmodels/pkg/vlm"
)

// TestHeadDefaultBackendMakesNoOutboundCall: with KNIRV_VLM_BACKEND unset, HEAD must
// select the "none" backend and exercise every HEAD route without a single dial. The
// sidecar endpoint variables are set to a loopback address; the default backend must
// ignore them. Every outbound dial goes through a transport that counts and refuses
// it, so any outbound attempt fails the test.
func TestHeadDefaultBackendMakesNoOutboundCall(t *testing.T) {
	t.Setenv("KNIRV_VLM_BACKEND", "")
	t.Setenv("KNIRV_VLM_ONNX_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("KNIRV_VLM_LLAMA_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("KNIRV_VLM_OPENAI_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("KNIRV_VLM_MINICPM_ENDPOINT", "http://127.0.0.1:1")

	var dials atomic.Int32
	refusing := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			dials.Add(1)
			return nil, errors.New("test: outbound dial refused")
		},
	}
	prev := http.DefaultTransport
	http.DefaultTransport = refusing
	t.Cleanup(func() {
		http.DefaultTransport = prev
		refusing.CloseIdleConnections()
	})

	backend := vlm.FromEnv()
	if backend.Name() != "none" {
		t.Fatalf("default backend = %q, want none", backend.Name())
	}
	mux := http.NewServeMux()
	NewHead(backend).Mount(mux, "/models/head")

	calls := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/models/head/status", "", http.StatusOK},
		{http.MethodGet, "/models/head/capabilities", "", http.StatusOK},
		{http.MethodPost, "/models/head/embed", `{"text":"x"}`, http.StatusServiceUnavailable},
		{http.MethodPost, "/models/head/describe", `{"text":"x"}`, http.StatusServiceUnavailable},
		{http.MethodPost, "/models/head/ground", `{"text":"x"}`, http.StatusServiceUnavailable},
		{http.MethodPost, "/models/head/visual-reason", `{"text":"x"}`, http.StatusServiceUnavailable},
		{http.MethodPost, "/models/head/generate", `{"prompt":"x"}`, http.StatusServiceUnavailable},
	}
	for _, c := range calls {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Fatalf("%s %s: status %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
	if n := dials.Load(); n != 0 {
		t.Fatalf("default backend attempted %d outbound dial(s); want 0", n)
	}
}
