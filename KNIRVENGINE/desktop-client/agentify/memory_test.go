package agentify

import (
	"testing"
)

func TestInMemoryManager(t *testing.T) {
	manager := NewInMemoryManager()

	// Test basic set and get
	err := manager.Set("test_key", "test_value")
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	value, err := manager.Get("test_key")
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %v", value)
	}

	// Test TTL functionality
	err = manager.SetWithTTL("ttl_key", "ttl_value", 1) // 1 second TTL
	if err != nil {
		t.Errorf("Failed to set value with TTL: %v", err)
	}

	// Should be able to get immediately
	value, err = manager.Get("ttl_key")
	if err != nil {
		t.Errorf("Failed to get TTL value: %v", err)
	}

	if value != "ttl_value" {
		t.Errorf("Expected 'ttl_value', got %v", value)
	}

	// Note: InMemoryManager doesn't enforce TTL, so the key should still be there
	// This is expected behavior for the simplified implementation

	// Test list functionality
	err = manager.Set("key1", "value1")
	if err != nil {
		t.Errorf("Failed to set key1: %v", err)
	}

	err = manager.Set("key2", "value2")
	if err != nil {
		t.Errorf("Failed to set key2: %v", err)
	}

	keys, err := manager.List()
	if err != nil {
		t.Errorf("Failed to list keys: %v", err)
	}

	expectedKeys := map[string]bool{"test_key": true, "key1": true, "key2": true, "ttl_key": true}
	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for _, key := range keys {
		if !expectedKeys[key] {
			t.Errorf("Unexpected key in list: %s", key)
		}
	}

	// Test delete functionality
	err = manager.Delete("key1")
	if err != nil {
		t.Errorf("Failed to delete key1: %v", err)
	}

	_, err = manager.Get("key1")
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}

	// Test clear functionality
	err = manager.Clear()
	if err != nil {
		t.Errorf("Failed to clear memory: %v", err)
	}

	keys, err = manager.List()
	if err != nil {
		t.Errorf("Failed to list keys after clear: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", len(keys))
	}
}
