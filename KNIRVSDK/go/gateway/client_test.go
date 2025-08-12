package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway/option"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    []option.RequestOption
		wantErr bool
	}{
		{
			name:    "Default client creation",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "Client with custom options",
			opts: []option.RequestOption{
				option.WithAPIKey("test-key"),
				option.WithBaseURL("https://test.example.com"),
			},
			wantErr: false,
		},
		{
			name: "Client with timeout option",
			opts: []option.RequestOption{
				option.WithHTTPClient(&http.Client{
					Timeout: 30 * time.Second,
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client but got nil")
				return
			}

			// Verify services are initialized (they are value types, so just check they exist)
			// Economics service should have sub-services
			if client.Economics.Skills.client == nil {
				t.Error("Economics Skills service not properly initialized")
			}

			if client.Economics.LLM.client == nil {
				t.Error("Economics LLM service not properly initialized")
			}

			// Gateway service should have sub-services
			if client.Gateway.Routes.client == nil {
				t.Error("Gateway Routes service not properly initialized")
			}

			// Health service should be initialized
			if client.Health.client == nil {
				t.Error("Health service not properly initialized")
			}

			// Integration service should be initialized
			if client.Integration.client == nil {
				t.Error("Integration service not properly initialized")
			}

			// PoAuD service should be initialized
			if client.PoAuD.client == nil {
				t.Error("PoAuD service not properly initialized")
			}
		})
	}
}

func TestClientEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{
			name:   "API Key from environment",
			envVar: "KNIRV_API_KEY",
			value:  "env-test-key",
		},
		{
			name:   "Base URL from environment",
			envVar: "KNIRV_BASE_URL",
			value:  "https://env.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv(tt.envVar, tt.value)
			defer os.Unsetenv(tt.envVar)

			client, err := NewClient()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client but got nil")
			}
		})
	}
}

func TestClientHTTPMethods(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"created": true}`))
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"updated": true}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("GET request", func(t *testing.T) {
		req, err := client.NewRequest(ctx, http.MethodGet, "/test", nil)
		if err != nil {
			t.Errorf("Failed to create GET request: %v", err)
		}

		if req.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", req.Method)
		}
	})

	t.Run("POST request", func(t *testing.T) {
		req, err := client.NewRequest(ctx, http.MethodPost, "/test", map[string]interface{}{
			"data": "test",
		})
		if err != nil {
			t.Errorf("Failed to create POST request: %v", err)
		}

		if req.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
	})

	t.Run("PUT request", func(t *testing.T) {
		req, err := client.NewRequest(ctx, http.MethodPut, "/test", map[string]interface{}{
			"data": "updated",
		})
		if err != nil {
			t.Errorf("Failed to create PUT request: %v", err)
		}

		if req.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", req.Method)
		}
	})

	t.Run("DELETE request", func(t *testing.T) {
		req, err := client.NewRequest(ctx, http.MethodDelete, "/test", nil)
		if err != nil {
			t.Errorf("Failed to create DELETE request: %v", err)
		}

		if req.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", req.Method)
		}
	})
}

func TestClientRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for expected headers
		if r.Header.Get("X-API-Key") == "" {
			t.Error("Expected X-API-Key header")
		}

		// Content-Type is only set for requests with body, so we'll check Accept instead
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Expected Accept: application/json")
		}

		if r.Header.Get("User-Agent") == "" {
			t.Error("Expected User-Agent header")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		option.WithBaseURL(server.URL),
		option.WithAPIKey("test-api-key"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req, err := client.NewRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}

	// Execute the request to trigger header validation
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
}

func TestClientTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout
	httpClient := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	client, err := NewClient(
		option.WithBaseURL(server.URL),
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req, err := client.NewRequest(ctx, http.MethodGet, "/slow", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

func TestClientRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		option.WithBaseURL(server.URL),
		option.WithRetryPolicy(3, 100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req, err := client.NewRequest(ctx, http.MethodGet, "/retry", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to execute request with retry: %v", err)
	}
	defer resp.Body.Close()

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := client.NewRequest(ctx, http.MethodGet, "/slow", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected context cancellation error but got none")
	}
}

func TestClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Bad request", "code": 400}`))
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req, err := client.NewRequest(ctx, http.MethodGet, "/error", nil)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
