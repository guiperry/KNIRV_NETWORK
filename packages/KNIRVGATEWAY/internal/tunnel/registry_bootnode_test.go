package tunnel

import (
	"testing"

	"go.uber.org/zap"
)

func TestRegistryManager_RegisterBootnode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	rm := NewRegistryManager(logger)

	err := rm.RegisterBootnode("bootnode-1", "chain-123", "192.168.1.1", 30303)
	if err != nil {
		t.Fatalf("Failed to register bootnode: %v", err)
	}

	node, err := rm.GetBootnodeByChainID("chain-123")
	if err != nil {
		t.Fatalf("Failed to get bootnode: %v", err)
	}

	if node.DevID != "bootnode-1" {
		t.Errorf("Expected devID 'bootnode-1', got '%s'", node.DevID)
	}

	if node.PublicIP != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", node.PublicIP)
	}

	if node.PublicP2PPort != 30303 {
		t.Errorf("Expected port 30303, got %d", node.PublicP2PPort)
	}

	if !node.IsBootnode {
		t.Error("Expected IsBootnode to be true")
	}
}

func TestRegistryManager_GetBootnodes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	rm := NewRegistryManager(logger)

	err := rm.RegisterBootnode("bootnode-1", "chain-1", "192.168.1.1", 30303)
	if err != nil {
		t.Fatalf("Failed to register bootnode: %v", err)
	}

	err = rm.RegisterBootnode("bootnode-2", "chain-2", "192.168.1.2", 30304)
	if err != nil {
		t.Fatalf("Failed to register bootnode: %v", err)
	}

	bootnodes, err := rm.GetBootnodes()
	if err != nil {
		t.Fatalf("Failed to get bootnodes: %v", err)
	}

	if len(bootnodes) != 2 {
		t.Errorf("Expected 2 bootnodes, got %d", len(bootnodes))
	}
}

func TestRegistryManager_GetBootnodeByChainID_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	rm := NewRegistryManager(logger)

	_, err := rm.GetBootnodeByChainID("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent chain ID")
	}
}