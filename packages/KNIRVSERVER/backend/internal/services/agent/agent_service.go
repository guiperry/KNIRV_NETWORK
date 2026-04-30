// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/runtime"
	"backend_server/internal/services/active_memory"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// AgentTask represents a task submitted to the oh-my-pi agent runtime
type AgentTask struct {
	ID          string            `json:"id"`
	DVEID       string            `json:"dve_id"`
	ContainerID string            `json:"container_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`   // research, coding, validation, analysis
	Status      string            `json:"status"` // pending, running, completed, failed
	Priority    int               `json:"priority"`
	Input       map[string]string `json:"input"`
	Output      string            `json:"output"`
	MarkdownLog string            `json:"markdown_log"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Error       string            `json:"error,omitempty"`
}

// AgentStatus represents the overall status of an agent in a DVE
type AgentStatus struct {
	DVEID          string     `json:"dve_id"`
	ContainerID    string     `json:"container_id"`
	Running        bool       `json:"running"`
	ViewportURL    string     `json:"viewport_url,omitempty"`
	ActiveTasks    int        `json:"active_tasks"`
	CompletedTasks int        `json:"completed_tasks"`
	LastActivity   *time.Time `json:"last_activity,omitempty"`
}

// AgentService manages knirvagent assistant containers within DVEs
type AgentService struct {
	db              *database.BuntDBManager
	ucm             *runtime.UnifiedContainerManager
	activeMemory    *active_memory.ActiveMemoryService
	agents          map[string]*AgentStatus     // dveID -> AgentStatus
	tasks           map[string]*AgentTask       // taskID -> AgentTask
	external_agents map[string]*AgentConnection // agentID -> AgentConnection
	mu              sync.RWMutex
	running         bool
	// apiBaseURL overrides the fallback localhost:8080 base URL used when
	// no container port mapping is available. Set in tests to reach a mock server.
	apiBaseURL string

	// Broadcast channels for agent responses (dveID -> list of channels)
	subscribers map[string][]chan string
	subsMu      sync.RWMutex
}

// NewAgentService creates a new agent service
func NewAgentService(
	db *database.BuntDBManager,
	ucm *runtime.UnifiedContainerManager,
	activeMemory *active_memory.ActiveMemoryService,
) *AgentService {
	return &AgentService{
		db:              db,
		ucm:             ucm,
		activeMemory:    activeMemory,
		agents:          make(map[string]*AgentStatus),
		tasks:           make(map[string]*AgentTask),
		external_agents: make(map[string]*AgentConnection),
		subscribers:     make(map[string][]chan string),
	}
}

// SubscribeToResponses subscribes to real-time agent responses for a DVE
func (s *AgentService) SubscribeToResponses(dveID string) chan string {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	ch := make(chan string, 100)
	s.subscribers[dveID] = append(s.subscribers[dveID], ch)
	return ch
}

// UnsubscribeFromResponses removes a subscription
func (s *AgentService) UnsubscribeFromResponses(dveID string, ch chan string) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	subs := s.subscribers[dveID]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[dveID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// BroadcastAgentMessage sends a message to all subscribers for a DVE
func (s *AgentService) BroadcastAgentMessage(dveID string, message string) {
	s.subsMu.RLock()
	defer s.subsMu.RUnlock()

	for _, ch := range s.subscribers[dveID] {
		select {
		case ch <- message:
		default:
			// Buffer full, skip
		}
	}
}

// Start starts the agent service
func (s *AgentService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("agent service already running")
	}

	if err := s.loadTasksFromDB(); err != nil {
		log.Printf("[AgentService] Warning: failed to load tasks from DB: %v", err)
	}

	s.running = true
	log.Printf("[AgentService] Started")
	return nil
}

// Stop stops the agent service
func (s *AgentService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	log.Printf("[AgentService] Stopped")
	return nil
}

// LaunchAgent provisions an oh-my-pi agent container for a DVE
func (s *AgentService) LaunchAgent(ctx context.Context, dveID string) (*AgentStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.agents[dveID]; ok && existing.Running {
		return existing, nil
	}

	if s.ucm == nil {
		return nil, fmt.Errorf("container manager not available")
	}

	config := &runtime.NestedObjectConfig{
		ObjectType:        runtime.ObjectTypeAgent,
		EnableViewport:    true,
		ViewportRenderers: []string{"http", "websocket"},
		ServicePorts:      map[string]int{"viewport": 8080, "jupyter": 8888, "lsp": 9090},
		Metadata: map[string]interface{}{
			"dve_id": dveID,
			"engine": "knirvagent",
		},
	}

	container, err := s.ucm.CreateNestedObject(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent container: %w", err)
	}

	viewportURL := ""
	if container.ViewportProxy != nil {
		viewportURL = fmt.Sprintf("/api/dve/%s/agent/viewport", dveID)
	}

	status := &AgentStatus{
		DVEID:       dveID,
		ContainerID: container.ID,
		Running:     true,
		ViewportURL: viewportURL,
	}

	s.agents[dveID] = status
	log.Printf("[AgentService] Launched knirvagent for DVE %s (container: %s)", dveID, container.ID)
	return status, nil
}

// GetAgentStatus returns the status of an agent for a DVE
func (s *AgentService) GetAgentStatus(dveID string) (*AgentStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, ok := s.agents[dveID]
	if !ok {
		return &AgentStatus{DVEID: dveID, Running: false}, nil
	}

	// Count tasks
	active, completed := 0, 0
	for _, t := range s.tasks {
		if t.DVEID == dveID {
			switch t.Status {
			case "running", "pending":
				active++
			case "completed":
				completed++
			}
		}
	}
	status.ActiveTasks = active
	status.CompletedTasks = completed

	return status, nil
}

// StopAgent stops the agent container for a DVE
func (s *AgentService) StopAgent(ctx context.Context, dveID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, ok := s.agents[dveID]
	if !ok || !status.Running {
		return nil
	}

	if s.ucm != nil && status.ContainerID != "" {
		if err := s.ucm.DestroyContainer(ctx, status.ContainerID); err != nil {
			log.Printf("[AgentService] Warning: failed to destroy container %s: %v", status.ContainerID, err)
		}
	}

	status.Running = false
	status.ContainerID = ""
	delete(s.agents, dveID)

	log.Printf("[AgentService] Stopped agent for DVE %s", dveID)
	return nil
}

// ConnectAgent registers an external agent connection to the network
func (s *AgentService) ConnectAgent(ctx context.Context, conn *AgentConnection) (*AgentConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	now := time.Now()
	conn.ConnectedAt = now
	conn.LastHeartbeat = &now

	s.external_agents[conn.AgentID] = conn
	log.Printf("[AgentService] External agent connected: %s (%s) with endpoint %s", conn.AgentID, conn.AgentName, conn.Endpoint)
	return conn, nil
}

// GetConnectedAgent returns a connected external agent by ID
func (s *AgentService) GetConnectedAgent(agentID string) (*AgentConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.external_agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not connected: %s", agentID)
	}

	return conn, nil
}

// ListConnectedAgents returns all connected external agents
func (s *AgentService) ListConnectedAgents() []*AgentConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var agents []*AgentConnection
	for _, conn := range s.external_agents {
		agents = append(agents, conn)
	}
	return agents
}

// SubmitTask submits a task to the agent for a given DVE
func (s *AgentService) SubmitTask(dveID string, req *AgentTaskRequest) (*AgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, ok := s.agents[dveID]
	if !ok || !status.Running {
		return nil, fmt.Errorf("no running agent for DVE %s", dveID)
	}

	now := time.Now()
	task := &AgentTask{
		ID:          uuid.New().String(),
		DVEID:       dveID,
		ContainerID: status.ContainerID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		Input:       req.Input,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if task.Type == "" {
		task.Type = "research"
	}
	if task.Priority == 0 {
		task.Priority = 1
	}

	s.tasks[task.ID] = task
	status.LastActivity = &now

	if err := s.saveTaskToDB(task); err != nil {
		log.Printf("[AgentService] Warning: failed to persist task %s: %v", task.ID, err)
	}

	// Asynchronously execute the task (writes markdown output)
	go s.executeTask(task)

	log.Printf("[AgentService] Submitted task %s to DVE %s agent", task.ID, dveID)
	return task, nil
}

// GetTasks returns all tasks for a DVE
func (s *AgentService) GetTasks(dveID string) ([]*AgentTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*AgentTask
	for _, t := range s.tasks {
		if t.DVEID == dveID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

// GetTask returns a single task by ID
func (s *AgentService) GetTask(taskID string) (*AgentTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ohMyPiTaskRequest is the JSON body sent to the oh-my-pi container's /api/task endpoint.
type ohMyPiTaskRequest struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Input       map[string]string `json:"input"`
	DVEID       string            `json:"dve_id"`
}

// ohMyPiTaskResponse is the JSON response returned by the oh-my-pi /api/task endpoint.
type ohMyPiTaskResponse struct {
	Success     bool   `json:"success"`
	TaskID      string `json:"task_id"`
	Output      string `json:"output"`
	MarkdownLog string `json:"markdown_log"`
	Error       string `json:"error,omitempty"`
}

// executeTask POSTs the task to the oh-my-pi container's REST API, waits for the
// response, writes the markdown output to Active Memory, and updates task state.
func (s *AgentService) executeTask(task *AgentTask) {
	s.mu.Lock()
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	task.UpdatedAt = now
	s.mu.Unlock()

	log.Printf("[AgentService] Executing task %s (%s): %s", task.ID, task.Type, task.Title)

	apiURL := s.getAgentAPIURL(task.ContainerID)
	reqBody := ohMyPiTaskRequest{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Type:        task.Type,
		Input:       task.Input,
		DVEID:       task.DVEID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		s.markTaskFailed(task, fmt.Sprintf("failed to marshal task request: %v", err))
		return
	}

	httpClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := httpClient.Post(apiURL+"/api/task", "application/json", bytes.NewReader(body))
	if err != nil {
		s.markTaskFailed(task, fmt.Sprintf("oh-my-pi container unreachable at %s: %v", apiURL, err))
		return
	}
	defer resp.Body.Close()

	var taskResp ohMyPiTaskResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&taskResp); decodeErr != nil {
		s.markTaskFailed(task, fmt.Sprintf("failed to decode container response: %v", decodeErr))
		return
	}
	if !taskResp.Success {
		s.markTaskFailed(task, taskResp.Error)
		return
	}

	// Write markdown output to Active Memory (Markdown Vault).
	if s.activeMemory != nil {
		ctx := context.Background()
		if err := s.activeMemory.RecordInteraction(ctx, task.DVEID, task.Description, taskResp.MarkdownLog); err != nil {
			log.Printf("[AgentService] Warning: failed to write task output to active memory: %v", err)
		}
	}

	s.mu.Lock()
	completedAt := time.Now()
	task.Status = "completed"
	task.CompletedAt = &completedAt
	task.Output = taskResp.Output
	task.MarkdownLog = taskResp.MarkdownLog
	task.UpdatedAt = completedAt
	s.mu.Unlock()

	if err := s.saveTaskToDB(task); err != nil {
		log.Printf("[AgentService] Warning: failed to persist completed task %s: %v", task.ID, err)
	}
	log.Printf("[AgentService] Task %s completed", task.ID)
}

// getAgentAPIURL resolves the base URL for the oh-my-pi container's task API.
// It inspects the container spec's port mappings for the auto-assigned host port
// on the agent viewport (container port 8080) and falls back to localhost:8080.
func (s *AgentService) getAgentAPIURL(containerID string) string {
	if s.ucm == nil {
		if s.apiBaseURL != "" {
			return s.apiBaseURL
		}
		return "http://localhost:8080"
	}
	container, err := s.ucm.GetContainer(containerID)
	if err != nil {
		return "http://localhost:8080"
	}
	if container.Spec != nil {
		for _, pm := range container.Spec.Ports {
			if pm.ContainerPort == 8080 && pm.HostPort > 0 {
				return fmt.Sprintf("http://localhost:%d", pm.HostPort)
			}
		}
	}
	return "http://localhost:8080"
}

// markTaskFailed records a terminal failure on the task.
func (s *AgentService) markTaskFailed(task *AgentTask, errMsg string) {
	log.Printf("[AgentService] Task %s failed: %s", task.ID, errMsg)
	s.mu.Lock()
	failedAt := time.Now()
	task.Status = "failed"
	task.CompletedAt = &failedAt
	task.Error = errMsg
	task.UpdatedAt = failedAt
	s.mu.Unlock()
	if err := s.saveTaskToDB(task); err != nil {
		log.Printf("[AgentService] Warning: failed to persist failed task %s: %v", task.ID, err)
	}
}

// AgentTaskRequest is the request payload for submitting a task
type AgentTaskRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Priority    int               `json:"priority"`
	Input       map[string]string `json:"input"`
}

// AgentConnection represents an external agent connecting to a DVE
type AgentConnection struct {
	AgentID       string                 `json:"agent_id"`
	AgentName     string                 `json:"agent_name"`
	Endpoint      string                 `json:"endpoint"`     // Agent's callback endpoint
	Capabilities  []string               `json:"capabilities"` // What resources/capabilities this agent declares
	Metadata      map[string]interface{} `json:"metadata"`
	ConnectedAt   time.Time              `json:"connected_at"`
	LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
}

func (s *AgentService) saveTaskToDB(task *AgentTask) error {
	if s.db == nil {
		return nil
	}
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("agent_task:"+task.ID, string(data), nil)
		return err
	})
}

func (s *AgentService) loadTasksFromDB() error {
	if s.db == nil {
		return nil
	}
	return s.db.GetObjectsByPrefix("agent_task:", func(key string, value []byte) bool {
		var task AgentTask
		if err := json.Unmarshal(value, &task); err == nil {
			s.tasks[task.ID] = &task
		}
		return true
	})
}
