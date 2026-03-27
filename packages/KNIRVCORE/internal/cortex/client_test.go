package cortex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{Endpoint: srv.URL, Timeout: 5 * time.Second})
	assert.NoError(t, client.HealthCheck(context.Background()))
}

func TestChatCompletion(t *testing.T) {
	responseBody := `{"choices":[{"message":{"role":"assistant","content":"Test insight"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "Llama-3-8B-Instruct-q4f16_1", payload["model"])
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	client := NewClient(Config{Endpoint: srv.URL, Timeout: 5 * time.Second})
	text, err := client.ChatCompletion(context.Background(), "Hello", "System")
	assert.NoError(t, err)
	assert.Equal(t, "Test insight", text)
}
