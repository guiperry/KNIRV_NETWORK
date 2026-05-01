package agentprovisioner

import (
	"fmt"
	"sync"
	"time"
)

// AgentInstance represents a running supervisor KNIRVAGENT instance
type AgentInstance struct {
	AgentID       string    `json:"agent_id"`
	DVEID         string    `json:"dve_id"`
	SessionKey    string    `json:"session_key"`
	WSEndpoint    string    `json:"ws_endpoint"`
	Status        string    `json:"status"` // "running", "stopped", "error"
	StartedAt     time.Time `json:"started_at"`
	WorkspacePath string    `json:"workspace_path"`
}

// AgentStatus represents the current status of an agent
type AgentStatus struct {
	AgentID       string    `json:"agent_id"`
	Status        string    `json:"status"`
	Uptime        string    `json:"uptime"`
	TaskCount     int       `json:"task_count"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	MemoryUsage   float64   `json:"memory_usage_mb"`
}

// AgentProvisioner manages supervisor KNIRVAGENT lifecycle
type AgentProvisioner struct {
	mu            sync.RWMutex
	agents        map[string]*AgentInstance
	workspaceRoot string
}

// NewAgentProvisioner creates a new provisioner
func NewAgentProvisioner(workspaceRoot string) *AgentProvisioner {
	return &AgentProvisioner{
		agents:        make(map[string]*AgentInstance),
		workspaceRoot: workspaceRoot,
	}
}

// SpawnSupervisorAgent creates and starts a supervisor KNIRVAGENT for a DVE
// dveID - the DVE creation ID
// dveName - human readable DVE name
// teeType - the TEE type (sgx, browser-extension, etc.)
// walletAddress - the derived DVE wallet address
func (ap *AgentProvisioner) SpawnSupervisorAgent(dveID, dveName, teeType, walletAddress string) (*AgentInstance, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	agentID := fmt.Sprintf("agent-%s", dveID)
	workspacePath := fmt.Sprintf("%s/%s/agent", ap.workspaceRoot, dveID)
	sessionKey := fmt.Sprintf("sk-%s-%d", dveID[:8], time.Now().UnixNano())

	instance := &AgentInstance{
		AgentID:       agentID,
		DVEID:         dveID,
		SessionKey:    sessionKey,
		WSEndpoint:    fmt.Sprintf("ws://dve/%s/agent", dveID),
		Status:        "running",
		StartedAt:     time.Now(),
		WorkspacePath: workspacePath,
	}

	// Generate identity workspace files
	ap.generateIdentityFile(instance, dveName, teeType, walletAddress)

	ap.agents[agentID] = instance
	return instance, nil
}

// TerminateAgent stops and removes a supervisor agent
func (ap *AgentProvisioner) TerminateAgent(agentID string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	instance, ok := ap.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	instance.Status = "stopped"
	delete(ap.agents, agentID)
	return nil
}

// GetAgentStatus returns the current status of an agent
func (ap *AgentProvisioner) GetAgentStatus(agentID string) (*AgentStatus, error) {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	instance, ok := ap.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	uptime := time.Since(instance.StartedAt).Round(time.Second).String()

	return &AgentStatus{
		AgentID:       instance.AgentID,
		Status:        instance.Status,
		Uptime:        uptime,
		TaskCount:     0,
		LastHeartbeat: time.Now(),
		MemoryUsage:   0,
	}, nil
}

// ListAgents returns all active supervisor agents
func (ap *AgentProvisioner) ListAgents() []*AgentInstance {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	result := make([]*AgentInstance, 0, len(ap.agents))
	for _, a := range ap.agents {
		result = append(result, a)
	}
	return result
}

// GetAgentByDVEID looks up an agent by its DVE ID
func (ap *AgentProvisioner) GetAgentByDVEID(dveID string) (*AgentInstance, error) {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	for _, a := range ap.agents {
		if a.DVEID == dveID {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no agent found for DVE %s", dveID)
}

func (ap *AgentProvisioner) generateIdentityFile(instance *AgentInstance, dveName, teeType, walletAddress string) {
	identity := fmt.Sprintf(`# DVE Supervisor Agent

DVE ID: %s
DVE Name: %s
Owner Address: %s
TEE Type: %s
DVE URI: knirv://dve/%s/%s
Capabilities: validation, attestation, execution
Attached Policies: 
Initialized At: %s
`,
		instance.DVEID,
		dveName,
		walletAddress,
		teeType,
		instance.DVEID,
		teeType,
		instance.StartedAt.Format(time.RFC3339),
	)

	_ = identity // In production, write to instance.WorkspacePath + "/IDENTITY.md"
	_ = instance
}
