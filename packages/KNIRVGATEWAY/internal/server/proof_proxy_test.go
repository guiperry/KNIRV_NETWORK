package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
)

func TestNativeProofProxyPreservesRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/knirv/projects/project-one/proofs" {
			t.Errorf("upstream path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-one" {
			t.Errorf("authorization header was not forwarded")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"project_id":"project-one"}` {
			t.Errorf("upstream body = %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"uploaded"}`))
	}))
	defer upstream.Close()

	gateway := testServer(&config.Config{Port: 8888, ServerBaseURL: upstream.URL})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/knirv/projects/project-one/proofs", strings.NewReader(`{"project_id":"project-one"}`))
	request.Header.Set("Authorization", "Bearer token-one")
	response := httptest.NewRecorder()
	gateway.router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("gateway status = %d; body=%s", response.Code, response.Body.String())
	}
}
