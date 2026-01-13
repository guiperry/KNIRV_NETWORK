package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClientRetryLogic(t *testing.T) {
	// Create a test server that fails the first two requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
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
	logger.SetLevel(logrus.DebugLevel)

	// Create API client with retry logic
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(1*time.Second),
		core.WithRetries(3), // Allow 3 retries
		core.WithLogger(logger),
	)

	// Test health check with retries
	ctx := context.Background()
	err := client.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, requestCount) // Should have made 3 requests (2 failures + 1 success)
}

func TestAPIClientCircuitBreaker(t *testing.T) {
	// Create a test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client with circuit breaker
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(1*time.Second),
		core.WithRetries(2),
		core.WithLogger(logger),
	)

	// Make 5 requests to trigger circuit breaker
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		client.HealthCheck(ctx)
	}

	// Verify circuit breaker is open
	err := client.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestAPIClientTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client with short timeout
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(500*time.Millisecond), // 500ms timeout
		core.WithRetries(0),                    // No retries
		core.WithLogger(logger),
	)

	// Test health check with timeout
	ctx := context.Background()
	err := client.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}