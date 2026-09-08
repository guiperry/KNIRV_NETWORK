package api

import (
	"context"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedGraphMetricsOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "graph.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	upstream := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("knirvgraph_rag_requests_total{operation=\"query\"} 7\nknirvgraph_rag_vectors 12\n"))
	})}
	go upstream.Serve(listener)
	defer upstream.Close()
	server := NewServer(&ServerConfig{ProbeSockets: map[string]string{"knirvgraph": socket}})
	server.Probes().ScrapeAll()
	for _, handler := range []http.HandlerFunc{server.handleKNIRVGraphScalability, server.handleKNIRVGraphEmbeddings} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"value":7`) || !strings.Contains(w.Body.String(), `"value":12`) {
			t.Fatalf("metrics = %d %s", w.Code, w.Body.String())
		}
	}
}

func TestGatewayRoutesUsesConfiguredSecret(t *testing.T) {
	t.Setenv("KNIRV_JWT_SECRET", "")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := jwt.Parse(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), func(*jwt.Token) (interface{}, error) { return []byte("configured-secret"), nil }, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			t.Error("invalid service credential")
			w.WriteHeader(403)
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer upstream.Close()
	server := NewServer(&ServerConfig{GatewayURL: upstream.URL, GatewayJWTSecret: "configured-secret"})
	if _, err := server.fetchGatewayRoutes(); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalMonitorSourcesUnavailable(t *testing.T) {
	server := NewServer(&ServerConfig{})
	for _, handler := range []http.HandlerFunc{server.handleKNIRVBaseMetrics, server.handleKNIRVOracleEconomics, server.handleRootFailover} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("GET", "/", nil))
		var result MetricsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || result.Data["available"] != false {
			t.Fatalf("response = %d %s", w.Code, w.Body.String())
		}
	}
}

func TestDisabledActuarialService(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	upstream := &http.Server{Handler: http.NotFoundHandler()}
	go upstream.Serve(listener)
	defer upstream.Shutdown(context.Background())
	server := NewServer(&ServerConfig{BackendSocketPath: socket})
	w := httptest.NewRecorder()
	server.handleActuarialMetrics(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatalf("response = %d %s", w.Code, w.Body.String())
	}
}

func TestGrafanaStatusReportsRuntimeConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		want      string
	}{
		{"not configured", "", `"configured":false`},
		{"configured", "https://grafana.example.test/", `"url":"https://grafana.example.test"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(&ServerConfig{GrafanaURL: tc.url})
			response := httptest.NewRecorder()
			server.handleGrafanaStatus(response, httptest.NewRequest(http.MethodGet, "/api/v1/monitor/grafana", nil))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), tc.want) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
