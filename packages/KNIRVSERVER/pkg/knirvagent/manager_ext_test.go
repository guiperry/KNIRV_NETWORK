package knirvagent

import (
	"testing"
	"time"
)

// TestManager_GetSocketPath_EmptyDVE verifies that GetSocketPath returns an
// error when called with an empty DVE ID on a freshly created Manager that
// has no agents running.
func TestManager_GetSocketPath_EmptyDVE(t *testing.T) {
	mgr := NewManager(&ManagerConfig{
		StartTimeout: 1 * time.Second,
		StopTimeout:  1 * time.Second,
	}, newTestLogger())

	path, err := mgr.GetSocketPath("")
	if err == nil {
		t.Error("expected error for empty DVE ID, got nil")
	}
	if path != "" {
		t.Errorf("expected empty socket path, got %q", path)
	}
}

// TestManager_InnerAgentClient_NotRunning verifies that InnerAgentClient
// returns an error when the Manager has no running agents.
func TestManager_InnerAgentClient_NotRunning(t *testing.T) {
	mgr := NewManager(&ManagerConfig{
		StartTimeout: 1 * time.Second,
		StopTimeout:  1 * time.Second,
	}, newTestLogger())

	client, socketPath, err := mgr.InnerAgentClient("dve-nonexistent")
	if err == nil {
		t.Error("expected error for non-running agent, got nil")
	}
	if client != nil {
		t.Error("expected nil http.Client when agent not running")
	}
	if socketPath != "" {
		t.Errorf("expected empty socket path, got %q", socketPath)
	}
}
