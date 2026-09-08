package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"knirv-server/internal/monitor/aggregator"
)

func TestHealthz(t *testing.T) {
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	server.handleHealthz(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp HealthResponse
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestReadyz(t *testing.T) {
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	server.handleReadyz(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp ReadyResponse
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Ready)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestStatus(t *testing.T) {
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	server.handleStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp StatusResponse
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "KNIRV Network Monitor", resp.Data["name"])
	assert.NotEmpty(t, resp.Timestamp)
}

func TestStatusResponseStructure(t *testing.T) {
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	server.handleStatus(w, req)

	var resp StatusResponse
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	data := resp.Data
	assert.NotNil(t, data)

	requiredKeys := []string{"name", "uptime", "go_version", "goroutines", "memory", "process"}
	for _, key := range requiredKeys {
		_, exists := data[key]
		assert.True(t, exists, "Data should contain key: %s", key)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := promhttp.HandlerFor(server.registry.Registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestProcessMetricsCollect(t *testing.T) {
	metrics := aggregator.NewProcessMetrics()
	result := metrics.Collect()

	assert.NotNil(t, result)
	if cpu, ok := result["cpu_usage_percent"].(float64); ok {
		assert.GreaterOrEqual(t, cpu, 0.0)
		assert.LessOrEqual(t, cpu, 100.0)
	}
	if mem, ok := result["memory_usage_percent"].(float64); ok {
		assert.GreaterOrEqual(t, mem, 0.0)
		assert.LessOrEqual(t, mem, 100.0)
	}
	assert.Greater(t, result["uptime_seconds"].(float64), 0.0)
}

func TestRegistryCreation(t *testing.T) {
	reg := aggregator.NewRegistry()
	assert.NotNil(t, reg)
	assert.NotNil(t, reg.Registry)

	metricFamilies, err := reg.Registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	metricNames := make(map[string]bool)
	for _, family := range metricFamilies {
		metricNames[*family.Name] = true
	}

	expectedMetrics := []string{
		"network_monitor_process_cpu_seconds_total",
		"network_monitor_process_memory_bytes",
		"network_monitor_process_disk_total_bytes",
		"network_monitor_process_disk_used_bytes",
		"network_monitor_process_goroutines",
		"network_monitor_process_uptime_seconds",
		"network_monitor_scrape_errors_total",
	}

	for _, name := range expectedMetrics {
		assert.True(t, metricNames[name], "Metric %s should be registered", name)
	}
}

func TestServerConfigDefaults(t *testing.T) {
	cfg := &ServerConfig{
		Port:           "9091",
		PrometheusURL:  "http://localhost:9090",
		GrafanaURL:     "http://localhost:3333",
		ScrapeInterval: 15 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	server := NewServer(cfg)
	assert.NotNil(t, server)
	assert.Equal(t, "9091", server.config.Port)
	assert.Equal(t, "http://localhost:9090", server.config.PrometheusURL)
	assert.NotNil(t, server.registry)
}

func TestNewServerStartTime(t *testing.T) {
	before := time.Now()
	cfg := &ServerConfig{Port: "9091"}
	server := NewServer(cfg)
	after := time.Now()

	assert.True(t, server.startTime.After(before) || server.startTime.Equal(before))
	assert.True(t, server.startTime.Before(after) || server.startTime.Equal(after))
}

func TestStartAndShutdownOwnsSocketLifecycle(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "monitor.sock")
	server := NewServer(&ServerConfig{SocketPath: socketPath})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("Unix-domain sockets are unavailable in this sandbox")
		}
		assert.NoError(t, err)
	}
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}}
	resp, err := client.Get("http://knirvmonitor/health")
	assert.NoError(t, err)
	if resp != nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	assert.NoError(t, server.Shutdown(context.Background()))
	_, err = net.Dial("unix", socketPath)
	assert.Error(t, err)
}

// TestDreamFindingsEndpointAcceptsValidFinding verifies the Phase G write
// route accepts a properly formed finding and exposes it via the list route.
func TestDreamFindingsEndpointAcceptsValidFinding(t *testing.T) {
	server := NewServer(&ServerConfig{Port: "9091"})

	body := `{"finding":{"id":"f1","policyName":"dream_task_proposed_action","nodeId":"learning","action":"scale_up","confidence":0.9,"threshold":0.7,"summary":"node-1 saturated","taskKind":"learning","evidence":["node-1 success_rate=0.4"]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dream-findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleDreamFindings(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/dream-findings/list", nil)
	listW := httptest.NewRecorder()
	server.handleDreamFindingsList(listW, listReq)
	assert.Equal(t, http.StatusOK, listW.Code)

	var envelope MetricsResponse
	assert.NoError(t, json.NewDecoder(listW.Body).Decode(&envelope))
	findings, ok := envelope.Data["findings"].([]DreamFinding)
	if !ok {
		// json decoder may decode as []interface{} — handle both.
		raw, _ := envelope.Data["findings"].([]interface{})
		findings = make([]DreamFinding, 0, len(raw))
		for _, r := range raw {
			b, _ := json.Marshal(r)
			var f DreamFinding
			assert.NoError(t, json.Unmarshal(b, &f))
			findings = append(findings, f)
		}
	}
	if len(findings) != 1 {
		t.Fatalf("findings count = %d, want 1", len(findings))
	}
	if findings[0].Action != "scale_up" || findings[0].ID != "f1" {
		t.Fatalf("finding mismatch: %+v", findings[0])
	}
}

// TestDreamFindingsEndpointRejectsBelowThreshold verifies the Phase G gate:
// a finding whose confidence is below threshold must be rejected.
func TestDreamFindingsEndpointRejectsBelowThreshold(t *testing.T) {
	server := NewServer(&ServerConfig{Port: "9091"})

	body := `{"finding":{"id":"f1","policyName":"dream_task_proposed_action","nodeId":"learning","action":"scale_up","confidence":0.3,"threshold":0.7,"summary":"x","taskKind":"learning"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dream-findings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleDreamFindings(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestDreamFindingsEndpointRejectsMissingFields verifies the validator rejects
// incomplete findings rather than silently inserting garbage.
func TestDreamFindingsEndpointRejectsMissingFields(t *testing.T) {
	server := NewServer(&ServerConfig{Port: "9091"})

	cases := []string{
		`{"finding":{"id":"f1","policyName":"dream_task_proposed_action","confidence":0.9,"threshold":0.7}}`,
		`{"finding":{"id":"f1","action":"scale_up","confidence":0.9,"threshold":0.7}}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dream-findings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.handleDreamFindings(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, body)
	}
}

// TestDreamFindingsBufferIsBounded confirms the buffer evicts old entries.
func TestDreamFindingsBufferIsBounded(t *testing.T) {
	server := NewServer(&ServerConfig{Port: "9091"})
	for i := 0; i < maxDreamFindings+10; i++ {
		server.appendDreamFinding(DreamFinding{
			ID:         "f",
			PolicyName: "p",
			Action:     "a",
			Confidence: 0.9,
			Threshold:  0.7,
		})
	}
	if got := len(server.dreamFindings); got != maxDreamFindings {
		t.Fatalf("buffer size = %d, want %d", got, maxDreamFindings)
	}
}
