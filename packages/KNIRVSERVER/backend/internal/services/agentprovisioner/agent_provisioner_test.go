package agentprovisioner

import (
	"testing"
	"time"
)

func TestNewAgentProvisioner(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")
	if ap == nil {
		t.Fatal("expected non-nil provisioner")
	}
	if ap.workspaceRoot != "/tmp/knirv/agents" {
		t.Errorf("expected workspaceRoot /tmp/knirv/agents, got %s", ap.workspaceRoot)
	}
	if len(ap.agents) != 0 {
		t.Errorf("expected empty agents map, got %d entries", len(ap.agents))
	}
}

func TestSpawnSupervisorAgent(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	instance, err := ap.SpawnSupervisorAgent("dve-alpha-001", "DVE-Alpha", "sgx", "knirv1abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.AgentID != "agent-dve-alpha-001" {
		t.Errorf("expected agent ID 'agent-dve-alpha-001', got '%s'", instance.AgentID)
	}
	if instance.DVEID != "dve-alpha-001" {
		t.Errorf("expected DVE ID 'dve-alpha-001', got '%s'", instance.DVEID)
	}
	if instance.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", instance.Status)
	}
	if instance.WorkspacePath != "/tmp/knirv/agents/dve-alpha-001/agent" {
		t.Errorf("unexpected workspace path: %s", instance.WorkspacePath)
	}
	if instance.StartedAt.After(time.Now()) {
		t.Error("started time should be in the past")
	}
}

func TestSpawnMultipleAgents(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	agents := []struct {
		dveID, dveName, teeType, walletAddress string
	}{
		{"dve-alpha", "DVE-Alpha", "sgx", "addr1"},
		{"dve-beta", "DVE-Beta", "browser-extension", "addr2"},
		{"dve-gamma", "DVE-Gamma", "tdx", "addr3"},
	}

	for _, a := range agents {
		instance, err := ap.SpawnSupervisorAgent(a.dveID, a.dveName, a.teeType, a.walletAddress)
		if err != nil {
			t.Fatalf("failed to spawn agent for %s: %v", a.dveID, err)
		}
		if instance.AgentID != "agent-"+a.dveID {
			t.Errorf("unexpected agent ID: %s", instance.AgentID)
		}
	}

	list := ap.ListAgents()
	if len(list) != 3 {
		t.Errorf("expected 3 agents, got %d", len(list))
	}
}

func TestTerminateAgent(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	instance, err := ap.SpawnSupervisorAgent("dve-test-001", "DVE-Test", "sgx", "addr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = ap.TerminateAgent(instance.AgentID)
	if err != nil {
		t.Fatalf("unexpected error terminating agent: %v", err)
	}

	// Verify agent is removed
	_, err = ap.GetAgentStatus(instance.AgentID)
	if err == nil {
		t.Error("expected error when getting status of terminated agent")
	}
}

func TestTerminateNonexistentAgent(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	err := ap.TerminateAgent("agent-nonexistent")
	if err == nil {
		t.Error("expected error when terminating nonexistent agent")
	}
}

func TestGetAgentStatus(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	instance, err := ap.SpawnSupervisorAgent("dve-status-001", "DVE-Status", "sgx", "addr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, err := ap.GetAgentStatus(instance.AgentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.AgentID != instance.AgentID {
		t.Errorf("expected agent ID %s, got %s", instance.AgentID, status.AgentID)
	}
	if status.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", status.Status)
	}
	if status.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestGetNonexistentAgentStatus(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	_, err := ap.GetAgentStatus("agent-nonexistent")
	if err == nil {
		t.Error("expected error when getting status of nonexistent agent")
	}
}

func TestGetAgentByDVEID(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	ap.SpawnSupervisorAgent("dve-lookup-001", "DVE-Lookup", "sgx", "addr1")
	ap.SpawnSupervisorAgent("dve-lookup-002", "DVE-Lookup2", "browser-extension", "addr2")

	instance, err := ap.GetAgentByDVEID("dve-lookup-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.DVEID != "dve-lookup-001" {
		t.Errorf("expected DVE ID 'dve-lookup-001', got '%s'", instance.DVEID)
	}
}

func TestGetAgentByDVEIDNotFound(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	_, err := ap.GetAgentByDVEID("dve-nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent DVE")
	}
}

func TestListAgentsEmpty(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	agents := ap.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected empty list, got %d agents", len(agents))
	}
}

func TestTerminateCleansUpList(t *testing.T) {
	ap := NewAgentProvisioner("/tmp/knirv/agents")

	ap.SpawnSupervisorAgent("dve-cleanup-001", "DVE-Cleanup", "sgx", "addr")
	instance, _ := ap.SpawnSupervisorAgent("dve-cleanup-002", "DVE-Cleanup2", "tdx", "addr2")
	ap.SpawnSupervisorAgent("dve-cleanup-003", "DVE-Cleanup3", "browser-extension", "addr3")

	ap.TerminateAgent(instance.AgentID)

	agents := ap.ListAgents()
	if len(agents) != 2 {
		t.Errorf("expected 2 agents after termination, got %d", len(agents))
	}

	for _, a := range agents {
		if a.DVEID == "dve-cleanup-002" {
			t.Error("terminated agent should not appear in list")
		}
	}
}
