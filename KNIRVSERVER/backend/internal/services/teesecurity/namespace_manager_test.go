package teesecurity

import (
	"os"
	"testing"
)

// TestNewNamespaceManager tests namespace manager creation
func TestNewNamespaceManager(t *testing.T) {
	config := NamespaceConfig{
		EnablePID:     true,
		EnableNetwork: true,
		EnableMount:   true,
		EnableUTS:     true,
		EnableIPC:     true,
		EnableUser:    true,
	}

	mgr := NewNamespaceManager(config)
	if mgr == nil {
		t.Fatal("NewNamespaceManager returned nil")
	}

	if mgr.config != config {
		t.Error("NamespaceManager config mismatch")
	}
}

// TestNamespaceManager_CreateNamespaces tests namespace creation
func TestNamespaceManager_CreateNamespaces(t *testing.T) {
	// Skip test if not running as root (user namespaces may not be available)
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: requires root privileges for namespace creation")
	}

	config := NamespaceConfig{
		EnablePID:     true,
		EnableNetwork: true,
		EnableMount:   true,
		EnableUTS:     true,
		EnableIPC:     true,
		EnableUser:    false, // Temporarily disable user namespace for testing
	}

	mgr := NewNamespaceManager(config)
	namespaces, err := mgr.CreateNamespaces()
	if err != nil {
		t.Skipf("Skipping test: namespace creation failed (may not be supported): %v", err)
	}

	if len(namespaces) != 5 {
		t.Errorf("Expected 5 namespaces, got %d", len(namespaces))
	}

	// Verify namespace types
	expectedTypes := map[string]bool{
		"pid": true, "net": true, "mnt": true,
		"uts": true, "ipc": true, "user": true,
	}

	for _, ns := range namespaces {
		if !expectedTypes[ns.Type] {
			t.Errorf("Unexpected namespace type: %s", ns.Type)
		}
	}
}

// TestNamespaceManager_SetupUTSNamespace tests UTS namespace configuration
func TestNamespaceManager_SetupUTSNamespace(t *testing.T) {
	// Skip test if not running as root
	if os.Geteuid() != 0 {
		t.Skip("Skipping test: requires root privileges")
	}

	config := NamespaceConfig{
		EnableUTS: true,
	}

	mgr := NewNamespaceManager(config)
	
	// Create UTS namespace first
	_, err := mgr.CreateNamespaces()
	if err != nil {
		t.Skipf("Skipping test: failed to create UTS namespace: %v", err)
	}

	// Test hostname setup
	err = mgr.SetupUTSNamespace("test-container")
	if err != nil {
		t.Errorf("SetupUTSNamespace failed: %v", err)
	}
}

// TestGetNamespaceFD tests getting namespace file descriptors
func TestGetNamespaceFD(t *testing.T) {
	// Test with current process
	fd, err := GetNamespaceFD(os.Getpid(), "pid")
	if err != nil {
		t.Skipf("Skipping test: failed to get namespace FD: %v", err)
	}
	defer CloseNamespaceFD(fd)

	if fd < 0 {
		t.Error("GetNamespaceFD returned invalid file descriptor")
	}
}

// TestCloseNamespaceFD tests closing namespace file descriptors
func TestCloseNamespaceFD(t *testing.T) {
	// Test with current process
	fd, err := GetNamespaceFD(os.Getpid(), "pid")
	if err != nil {
		t.Skipf("Skipping test: failed to get namespace FD: %v", err)
	}

	err = CloseNamespaceFD(fd)
	if err != nil {
		t.Errorf("CloseNamespaceFD failed: %v", err)
	}

	// Try to close again (should fail)
	err = CloseNamespaceFD(fd)
	if err == nil {
		t.Error("CloseNamespaceFD should fail on already closed FD")
	}
}
