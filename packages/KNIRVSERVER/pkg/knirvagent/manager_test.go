package knirvagent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// newTestLogger returns a no-op logger suitable for tests.
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// ──────────────────────────────────────────────
// 1. NewAgentManager creates with default max concurrency (32)
// ──────────────────────────────────────────────

func TestNewAgentManager_DefaultMaxConcurrency(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())
	assert.Equal(t, 32, am.MaxConcurrent(),
		"default max concurrency should be 32")
}

func TestNewAgentManager_CustomMaxConcurrency(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{MaxConcurrent: 8}, newTestLogger())
	assert.Equal(t, 8, am.MaxConcurrent())
}

// ──────────────────────────────────────────────
// 2. StartAgent creates an agent process entry
// ──────────────────────────────────────────────

func TestStartAgent_Success(t *testing.T) {
	// Check that a Python interpreter is available
	_, err := exec.LookPath("python3")
	if err != nil {
		_, err = exec.LookPath("python")
		if err != nil {
			t.Skip("python interpreter not found, cannot run mock agent")
		}
	}

	tmpDir := t.TempDir()
	socketDir := filepath.Join(tmpDir, "sockets")
	err = os.MkdirAll(socketDir, 0755)
	require.NoError(t, err)

	// Write a minimal mock-agent Python script that creates a Unix socket
	mockScript := filepath.Join(tmpDir, "mock_agent.py")
	scriptContent := `#!/usr/bin/env python3
import socket, os, sys, time

socket_path = None
i = 1
while i < len(sys.argv):
    arg = sys.argv[i]
    if arg == "--socket-path" and i + 1 < len(sys.argv):
        socket_path = sys.argv[i + 1]
        i += 2
    elif arg.startswith("--socket-path="):
        socket_path = arg.split("=", 1)[1]
        i += 1
    else:
        i += 1

if not socket_path:
    sys.exit(1)

try:
    os.unlink(socket_path)
except OSError:
    pass

sock = socket.socket(socket.AF_UNIX)
sock.bind(socket_path)
sock.listen(5)
os.chmod(socket_path, 0o777)
print("ready", flush=True)
time.sleep(30)
sock.close()
try:
    os.unlink(socket_path)
except OSError:
    pass
`
	err = os.WriteFile(mockScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	am := NewAgentManager(&AgentManagerConfig{
		BinaryPath:    mockScript,
		SocketDir:     socketDir,
		MaxConcurrent: 10,
	}, newTestLogger())

	ctx := context.Background()
	ap, err := am.StartAgent(ctx, "dve-test-001", 5*time.Second)
	require.NoError(t, err, "StartAgent should succeed with mock binary")
	require.NotNil(t, ap, "returned AgentProcess should not be nil")

	assert.Equal(t, "dve-test-001", ap.DVEID)
	assert.NotZero(t, ap.PID, "PID should be set")
	assert.False(t, ap.StartedAt.IsZero(), "StartedAt should be set")
	assert.NotEmpty(t, ap.SocketPath, "SocketPath should be set")
	assert.True(t, ap.isHealthy(), "agent should be healthy after successful start")

	// Verify the agent is in the manager
	got, err := am.GetAgent("dve-test-001")
	assert.NoError(t, err)
	assert.Equal(t, ap.DVEID, got.DVEID)
	assert.Equal(t, 1, am.RunningCount())

	// Cleanup
	err = am.StopAgent("dve-test-001", 2*time.Second)
	assert.NoError(t, err)
}

func TestStartAgent_BinaryNotFound(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{
		BinaryPath:    "/nonexistent/path/to/knirvagent",
		SocketDir:     t.TempDir(),
		MaxConcurrent: 10,
	}, newTestLogger())

	ctx := context.Background()
	ap, err := am.StartAgent(ctx, "dve-test-002", 1*time.Second)
	assert.Error(t, err, "should return error when binary does not exist")
	assert.Nil(t, ap, "returned AgentProcess should be nil")
	assert.Contains(t, err.Error(), "failed to start KNIRVAGENT for DVE",
		"error should indicate start failure")
	assert.Contains(t, err.Error(), "no such file or directory",
		"error should indicate binary not found")

	// Verify no entry was created in the map
	_, err = am.GetAgent("dve-test-002")
	assert.Error(t, err, "GetAgent should error for DVE that was never started")
}

// ──────────────────────────────────────────────
// 3. GetAgent returns error for non-existent DVE
// ──────────────────────────────────────────────

func TestGetAgent_NonExistent(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{
		MaxConcurrent: 10,
	}, newTestLogger())

	ap, err := am.GetAgent("non-existent-dve")
	assert.Error(t, err, "should return error for non-existent DVE")
	assert.Nil(t, ap, "returned AgentProcess should be nil")
	assert.Contains(t, err.Error(), "no agent running for DVE")
}

// ──────────────────────────────────────────────
// 4. ListAgents returns all running agents
// ──────────────────────────────────────────────

func TestListAgents(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{
		MaxConcurrent: 10,
	}, newTestLogger())

	// Directly populate the agents map (same-package access)
	dveIDs := []string{"dve-alpha", "dve-beta", "dve-gamma"}
	for _, id := range dveIDs {
		am.agents[id] = &AgentProcess{
			DVEID:      id,
			SocketPath: filepath.Join(t.TempDir(), fmt.Sprintf("agent-%s.sock", id)),
			PID:        1000 + len(am.agents),
			StartedAt:  time.Now(),
			Healthy:    true,
		}
	}

	agents := am.ListAgents()
	assert.Len(t, agents, 3, "should list 3 agents")

	// Collect DVEIDs to verify all are present
	returnedIDs := make(map[string]bool)
	for _, ap := range agents {
		returnedIDs[ap.DVEID] = true
	}
	for _, id := range dveIDs {
		assert.True(t, returnedIDs[id], "expected DVE %s to be in the list", id)
	}
}

func TestListAgents_Empty(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())
	agents := am.ListAgents()
	assert.Empty(t, agents, "should return empty slice when no agents running")
}

// ──────────────────────────────────────────────
// 5. RunningCount returns correct count
// ──────────────────────────────────────────────

func TestRunningCount(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{
		MaxConcurrent: 10,
	}, newTestLogger())

	assert.Equal(t, 0, am.RunningCount(), "initially zero")

	// Add agents
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("dve-count-%d", i)
		am.agents[id] = &AgentProcess{
			DVEID:      id,
			SocketPath: filepath.Join(t.TempDir(), fmt.Sprintf("agent-%s.sock", id)),
			PID:        2000 + i,
			StartedAt:  time.Now(),
			Healthy:    true,
		}
	}

	assert.Equal(t, 5, am.RunningCount(), "should return 5 after adding 5 agents")

	// Remove one
	delete(am.agents, "dve-count-2")
	assert.Equal(t, 4, am.RunningCount(), "should return 4 after removing one")
}

// ──────────────────────────────────────────────
// 6. StopAgent is idempotent
// ──────────────────────────────────────────────

func TestStopAgent_Idempotent(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())

	// Stop a non-existent agent — should return nil (idempotent)
	err := am.StopAgent("non-existent-dve", 1*time.Second)
	assert.NoError(t, err, "stopping non-existent agent should be idempotent")

	// Now add an agent, stop it, then stop again
	am.agents["dve-remove"] = &AgentProcess{
		DVEID:      "dve-remove",
		SocketPath: filepath.Join(t.TempDir(), "agent-dve-remove.sock"),
		PID:        3000,
		StartedAt:  time.Now(),
		Healthy:    true,
	}

	err = am.StopAgent("dve-remove", 1*time.Second)
	assert.NoError(t, err, "first stop should succeed")

	// The agent should be removed from the map
	_, err = am.GetAgent("dve-remove")
	assert.Error(t, err, "agent should no longer exist after stop")

	// Stop again — idempotent
	err = am.StopAgent("dve-remove", 1*time.Second)
	assert.NoError(t, err, "second stop should also succeed (idempotent)")
}

// ──────────────────────────────────────────────
// 7. MaxConcurrent limit prevents starting more agents
// ──────────────────────────────────────────────

func TestMaxConcurrent_Limit(t *testing.T) {
	maxC := 3
	am := NewAgentManager(&AgentManagerConfig{
		MaxConcurrent: maxC,
		BinaryPath:    "/nonexistent/binary", // will not actually be invoked
		SocketDir:     t.TempDir(),
	}, newTestLogger())

	// Fill the agents map up to the limit
	for i := 0; i < maxC; i++ {
		id := fmt.Sprintf("dve-full-%d", i)
		am.agents[id] = &AgentProcess{
			DVEID:      id,
			SocketPath: filepath.Join(t.TempDir(), fmt.Sprintf("agent-%s.sock", id)),
			PID:        4000 + i,
			StartedAt:  time.Now(),
			Healthy:    true,
		}
	}

	assert.Equal(t, maxC, am.RunningCount())

	// Attempt to start another — should hit the limit
	ctx := context.Background()
	ap, err := am.StartAgent(ctx, "dve-overflow", 1*time.Second)
	assert.Error(t, err, "should return error when at max concurrent")
	assert.Nil(t, ap, "returned AgentProcess should be nil")
	assert.Contains(t, err.Error(), "max concurrent agents reached",
		"error should mention max concurrent limit")
	assert.Contains(t, err.Error(), "3/3",
		"error should show the limit counts")
}

// ──────────────────────────────────────────────
// 8. HealthCheck returns false for non-existent DVE
// ──────────────────────────────────────────────

func TestHealthCheck_NonExistentDVE(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())

	healthy := am.HealthCheck("non-existent-dve")
	assert.False(t, healthy, "HealthCheck should return false for non-existent DVE")
}

// ──────────────────────────────────────────────
// Additional edge-case tests
// ──────────────────────────────────────────────

func TestNewAgentManager_ZeroHealthIntervalDefaults(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())
	assert.Equal(t, 30*time.Second, am.healthInterval,
		"default health interval should be 30s")
}

func TestStopAgent_NoProcess(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{}, newTestLogger())

	// Add an agent with nil Cmd — should clean up gracefully
	socketDir := t.TempDir()
	socketPath := filepath.Join(socketDir, "agent-nocmd.sock")
	am.agents["dve-nocmd"] = &AgentProcess{
		DVEID:      "dve-nocmd",
		Cmd:        nil,
		SocketPath: socketPath,
		PID:        0,
		StartedAt:  time.Now(),
		Healthy:    false,
	}

	// Create a stale socket file to verify cleanup
	err := os.WriteFile(socketPath, []byte("stale"), 0644)
	require.NoError(t, err)

	err = am.StopAgent("dve-nocmd", 1*time.Second)
	assert.NoError(t, err, "should clean up agent with nil Cmd")

	// Socket file should be removed
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err), "socket file should be cleaned up")
}

func TestStartAgent_AlreadyRunning(t *testing.T) {
	am := NewAgentManager(&AgentManagerConfig{
		MaxConcurrent: 10,
	}, newTestLogger())

	// Pre-populate a healthy agent
	am.agents["dve-already"] = &AgentProcess{
		DVEID:      "dve-already",
		SocketPath: filepath.Join(t.TempDir(), "agent-dve-already.sock"),
		PID:        5000,
		StartedAt:  time.Now(),
		Healthy:    true,
	}

	ctx := context.Background()
	ap, err := am.StartAgent(ctx, "dve-already", 1*time.Second)
	assert.NoError(t, err, "should return existing agent without error")
	assert.NotNil(t, ap, "should return existing AgentProcess")
	assert.Equal(t, 5000, ap.PID, "should return the existing agent's PID")
	assert.Equal(t, 1, am.RunningCount(), "count should not increase")
}
