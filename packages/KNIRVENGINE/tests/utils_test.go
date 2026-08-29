package test

import (
	"testing"

	"KNIRVENGINE/desktop-client/internal/utils"
)

func TestUtilityFunctions(t *testing.T) {
	// Test data
	testMap := map[string]interface{}{
		"string_val": "hello",
		"int_val":    42,
		"float_val":  3.14,
		"int64_val":  int64(123456789),
	}

	// Test GetString
	if result := utils.GetString(testMap, "string_val"); result != "hello" {
		t.Errorf("GetString failed: expected 'hello', got '%s'", result)
	}

	// Test GetInt
	if result := utils.GetInt(testMap, "int_val"); result != 42 {
		t.Errorf("GetInt failed: expected 42, got %d", result)
	}

	// Test GetFloatPtr
	if result := utils.GetFloatPtr(testMap, "float_val"); result == nil || *result != 3.14 {
		t.Errorf("GetFloatPtr failed: expected 3.14, got %v", result)
	}

	// Test GetInt64Ptr
	if result := utils.GetInt64Ptr(testMap, "int64_val"); result == nil || *result != 123456789 {
		t.Errorf("GetInt64Ptr failed: expected 123456789, got %v", result)
	}

	t.Log("All utility functions are working correctly")
}
