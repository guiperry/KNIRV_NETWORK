package knirvagent

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

func TestAgentManager_applyWorkspaceResolver_LogsFailures(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	mgr := NewAgentManager(&AgentManagerConfig{}, logger)
	mgr.SetWorkspaceResolver(func(string) (string, error) {
		return "", errors.New("no active workspace for DVE test-dve")
	})

	cmd := exec.Command("true")
	mgr.applyWorkspaceResolver(cmd, "test-dve")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Message != "KNIRVAGENT workspace resolver failed" {
		t.Fatalf("unexpected log message: %q", entries[0].Message)
	}
	if got := entries[0].ContextMap()["dveID"]; got != "test-dve" {
		t.Fatalf("expected dveID context, got %#v", got)
	}
}
