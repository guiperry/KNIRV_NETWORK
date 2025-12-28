package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/blockchain"
)

func TestNewKNIRVGraphBridge(t *testing.T) {
	bridge := NewKNIRVGraphBridge("https://api.example.com", "test-key")
	if bridge == nil {
		t.Fatal("Expected KNIRVGraphBridge, got nil")
	}
	if bridge.apiURL != "https://api.example.com" {
		t.Errorf("Expected apiURL 'https://api.example.com', got '%s'", bridge.apiURL)
	}
	if bridge.apiKey != "test-key" {
		t.Errorf("Expected apiKey 'test-key', got '%s'", bridge.apiKey)
	}
	if bridge.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if bridge.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", bridge.httpClient.Timeout)
	}
}

func TestExtractSummary(t *testing.T) {
	bridge := NewKNIRVGraphBridge("https://api.example.com", "test-key")
	blockID := uuid.New()
	block := &blockchain.Block{
		BlockID: blockID,
	}

	summary := bridge.extractSummary(block)
	expected := "Memory block " + blockID.String()
	if summary != expected {
		t.Errorf("Expected summary '%s', got '%s'", expected, summary)
	}
}

func TestExtractRelationships(t *testing.T) {
	bridge := NewKNIRVGraphBridge("https://api.example.com", "test-key")

	// Test with PrevHash
	block := &blockchain.Block{
		PrevHash: "prev-hash-123",
	}

	relationships := bridge.extractRelationships(block)
	if len(relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(relationships))
	}
	if relationships[0].Type != "FOLLOWS" {
		t.Errorf("Expected type 'FOLLOWS', got '%s'", relationships[0].Type)
	}
	if relationships[0].Target != "prev-hash-123" {
		t.Errorf("Expected target 'prev-hash-123', got '%s'", relationships[0].Target)
	}

	// Test without PrevHash
	block2 := &blockchain.Block{
		PrevHash: "",
	}

	relationships2 := bridge.extractRelationships(block2)
	if len(relationships2) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(relationships2))
	}
}

func TestSendTransaction(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/transaction" {
			t.Errorf("Expected path '/api/v1/transaction', got '%s'", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
		}

		var transaction GraphTransaction
		if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if transaction.Source != "KNIRVCHAIN" {
			t.Errorf("Expected source 'KNIRVCHAIN', got '%s'", transaction.Source)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bridge := NewKNIRVGraphBridge(server.URL, "test-key")

	blockID := uuid.New()
	block := &blockchain.Block{
		BlockID:        blockID,
		PayloadHash:    "payload-hash",
		SemanticVector: []float32{0.1, 0.2, 0.3},
		Timestamp:      time.Now().Unix(),
		Category:        blockchain.CategoryGeneral,
		PrevHash:       "prev-hash",
	}

	err := bridge.SendTransaction(context.Background(), block)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSendTransactionServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	bridge := NewKNIRVGraphBridge(server.URL, "test-key")

	block := &blockchain.Block{
		BlockID: uuid.New(),
	}

	err := bridge.SendTransaction(context.Background(), block)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestSendTransactionNetworkError(t *testing.T) {
	bridge := NewKNIRVGraphBridge("http://invalid-url", "test-key")

	block := &blockchain.Block{
		BlockID: uuid.New(),
	}

	err := bridge.SendTransaction(context.Background(), block)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}
