package memory_test

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/services/memory"
	"go.uber.org/zap"
)

func TestNewUnifiedMemorySystem(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"markdown"},
		PQCEncryption:    true,
		ArrowStreaming:   false,
		GraphRAGModel:    "test-model",
		SyncInterval:     1 * time.Hour,
		EnableAutoSync:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}
	defer system.Close()

	if system == nil {
		t.Fatal("expected non-nil UnifiedMemorySystem")
	}
}

func TestStoreInteraction(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"markdown"},
		PQCEncryption:    false,
		ArrowStreaming:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}
	defer system.Close()

	ctx := context.Background()
	err = system.StoreInteraction(ctx, "agent-1", "test error", "print('solution')")
	if err != nil {
		t.Fatalf("failed to store interaction: %v", err)
	}
}

func TestQuery(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"graphrag"},
		PQCEncryption:    false,
		ArrowStreaming:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}
	defer system.Close()

	ctx := context.Background()
	req := &memory.QueryRequest{
		Query: "test query",
		Mode:  "graphrag",
		Limit: 10,
	}

	result, err := system.Query(ctx, req)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Query != "test query" {
		t.Errorf("expected query 'test query', got '%s'", result.Query)
	}
}

func TestStreamToArrow(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"markdown"},
		PQCEncryption:    false,
		ArrowStreaming:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}
	defer system.Close()

	// Store some test data
	ctx := context.Background()
	system.StoreInteraction(ctx, "agent-1", "error 1", "solution 1")
	system.StoreInteraction(ctx, "agent-2", "error 2", "solution 2")

	// Test streaming (placeholder test)
	// In real implementation, would create a mock Flight server
}

func TestGetServices(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"markdown", "graphrag", "ontology"},
		PQCEncryption:    false,
		ArrowStreaming:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}
	defer system.Close()

	// Test getting services
	vault := system.GetVaultService()
	if vault == nil {
		t.Error("expected non-nil VaultService")
	}

	ontology := system.GetOntologyManager()
	if ontology == nil {
		t.Error("expected non-nil OntologyManager")
	}

	reasoning := system.GetReasoningEngine()
	if reasoning == nil {
		t.Error("expected non-nil ReasoningEngine")
	}

	graphrag := system.GetGraphRAGClient()
	if graphrag == nil {
		t.Error("expected non-nil GraphRAGClient")
	}
}

func TestClose(t *testing.T) {
	logger := zap.NewNop()
	cfg := &memory.MemoryConfig{
		EnabledBackends: []string{"markdown", "graphrag"},
		PQCEncryption:    false,
		ArrowStreaming:   false,
	}

	system, err := memory.NewUnifiedMemorySystem(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create UnifiedMemorySystem: %v", err)
	}

	// Close should not error
	err = system.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
