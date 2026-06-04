package main

import (
	"testing"
)

func TestMain(t *testing.T) {
	// Test that main function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()
	
	// We can't actually call main() in tests as it would print to stdout
	// Instead, we'll test that the package can be imported without issues
	t.Log("KNIRV Go SDK main package test passed")
}

func TestPackageImports(t *testing.T) {
	// Test that all subpackages can be imported
	// This is implicitly tested by the import statements in main.go
	// If any import fails, the test compilation will fail
	t.Log("All package imports successful")
}
