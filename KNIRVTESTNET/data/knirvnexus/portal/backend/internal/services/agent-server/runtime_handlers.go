package agentserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleStartAgent handles the /runtime/start endpoint
func (as *AgentServer) HandleStartAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Name   string                 `json:"name"`
		Binary string                 `json:"binary"`
		Config map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	if request.Name == "" || request.Binary == "" {
		http.Error(w, "Name and binary are required", http.StatusBadRequest)
		return
	}

	agent, err := as.runtimeManager.StartAgent(request.Name, request.Binary, request.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)

	log.Printf("Started agent %s (%s) from %s", request.Name, request.Binary, r.RemoteAddr)
}

// HandleStopAgent handles the /runtime/stop/{id} endpoint
func (as *AgentServer) HandleStopAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	agentID := strings.TrimPrefix(r.URL.Path, "/runtime/stop/")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	if err := as.runtimeManager.StopAgent(agentID); err != nil {
		http.Error(w, fmt.Sprintf("Error stopping agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Agent stopped successfully"}`)

	log.Printf("Stopped agent %s from %s", agentID, r.RemoteAddr)
}

// HandleListRunningAgents handles the /runtime/agents endpoint
func (as *AgentServer) HandleListRunningAgents(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	agents := as.runtimeManager.GetAgentList()

	response := struct {
		Agents []interface{} `json:"agents"`
		Count  int           `json:"count"`
	}{
		Agents: make([]interface{}, len(agents)),
		Count:  len(agents),
	}

	for i, agent := range agents {
		response.Agents[i] = agent
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAgent handles the /runtime/agent/{id} endpoint
func (as *AgentServer) HandleGetAgent(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	agentID := strings.TrimPrefix(r.URL.Path, "/runtime/agent/")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	agent, err := as.runtimeManager.GetAgent(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// HandleRuntimeStatus handles the /runtime/status endpoint
func (as *AgentServer) HandleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	agents := as.runtimeManager.GetAgentList()

	status := map[string]interface{}{
		"running":     true,
		"agent_count": len(agents),
		"max_agents":  as.maxAgents,
		"uptime":      time.Since(as.startTime).String(),
	}

	// Add resource pool status if available
	if as.runtimeManager.resourcePool != nil {
		status["resources"] = as.runtimeManager.resourcePool.GetResourceUsage()
	}

	// Add scheduler status if available
	if as.runtimeManager.scheduler != nil {
		status["scheduler"] = as.runtimeManager.scheduler.GetQueueStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
