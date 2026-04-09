package benchmark

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/sirupsen/logrus"
)

func BenchmarkAPIClientHealthCheck(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Minimize logging overhead

	// Create API client
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(5*time.Second),
		core.WithRetries(0), // No retries for benchmark
		core.WithLogger(logger),
	)

	// Create context
	ctx := context.Background()

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		err := client.HealthCheck(ctx)
		if err != nil {
			b.Fatalf("Health check failed: %v", err)
		}
	}
}

func BenchmarkAPIClientWithRetries(b *testing.B) {
	// Create a test server that fails the first request
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount%2 == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Minimize logging overhead

	// Create API client with retry logic
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(1*time.Second),
		core.WithRetries(3), // Allow 3 retries
		core.WithLogger(logger),
	)

	// Create context
	ctx := context.Background()

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		requestCount = 0 // Reset for each iteration
		err := client.HealthCheck(ctx)
		if err != nil {
			b.Fatalf("Health check failed: %v", err)
		}
	}
}

func BenchmarkAPIClientGetNodeInfo(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"version": "1.0.0",
			"node_id": "test-node",
			"start_time": "2023-01-01T00:00:00Z",
			"uptime": "1h",
			"network_id": "test-network",
			"peer_count": 10,
			"sync_status": "synced",
			"block_height": 100
		}`))
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Minimize logging overhead

	// Create API client
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(5*time.Second),
		core.WithRetries(0), // No retries for benchmark
		core.WithLogger(logger),
	)

	// Create context
	ctx := context.Background()

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, err := client.GetNodeInfo(ctx)
		if err != nil {
			b.Fatalf("GetNodeInfo failed: %v", err)
		}
	}
}
