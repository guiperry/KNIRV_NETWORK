package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/utils/host"
	"github.com/gorilla/mux"
)

func TestSystemHandler_GetSystemInfo(t *testing.T) {
	collector, err := host.NewSystemInfoCollector(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("Failed to create system info collector: %v", err)
	}

	handler := NewSystemHandler(collector)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/system/info", handler.GetSystemInfo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response SystemMetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OS == "" {
		t.Error("Expected OS to be set")
	}
	if response.Arch == "" {
		t.Error("Expected Arch to be set")
	}
}

func TestSystemHandler_GetDetailedSystemInfo(t *testing.T) {
	collector, err := host.NewSystemInfoCollector(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("Failed to create system info collector: %v", err)
	}

	handler := NewSystemHandler(collector)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/system/detail", handler.GetDetailedSystemInfo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/detail", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestParseUptimeToSeconds(t *testing.T) {
	tests := []struct {
		name   string
		uptime string
		want   int64
	}{
		{
			name:   "empty string",
			uptime: "",
			want:   0,
		},
		{
			name:   "1 hour",
			uptime: "1 hour",
			want:   3600,
		},
		{
			name:   "2 hours",
			uptime: "2 hours",
			want:   7200,
		},
		{
			name:   "1 day",
			uptime: "1 day",
			want:   86400,
		},
		{
			name:   "1 week",
			uptime: "1 week",
			want:   604800,
		},
		{
			name:   "2 weeks 3 days",
			uptime: "2 weeks 3 days",
			want:   1468800,
		},
		{
			name:   "30 minutes",
			uptime: "30 minutes",
			want:   1800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUptimeToSeconds(tt.uptime)
			if got != tt.want {
				t.Errorf("parseUptimeToSeconds(%q) = %d, want %d", tt.uptime, got, tt.want)
			}
		})
	}
}
