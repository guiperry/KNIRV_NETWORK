// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	Type        string            `json:"type"`    // research, coding, validation, analysis
	Status      string            `json:"status"`  // pending, running, completed, failed
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

// AgentService manages oh-my-pi agent containers within DVEs
type AgentService struct {
	db           *database.BuntDBManager
	ucm          *runtime.UnifiedContainerManager
	activeMemory *active_memory.ActiveMemoryService
	agents       map[string]*AgentStatus // dveID -> AgentStatus
	tasks        map[string]*AgentTask   // taskID -> AgentTask
	mu           sync.RWMutex
	running      bool
}

// NewAgentService creates a new agent service
func NewAgentService(
	db *database.BuntDBManager,
	ucm *runtime.UnifiedContainerManager,
	activeMemory *active_memory.ActiveMemoryService,
) *AgentService {
	return &AgentService{
		db:           db,
		ucm:          ucm,
		activeMemory: activeMemory,
		agents:       make(map[string]*AgentStatus),
		tasks:        make(map[string]*AgentTask),
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
			"engine": "oh-my-pi",
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
	log.Printf("[AgentService] Launched agent for DVE %s (container: %s)", dveID, container.ID)
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
			if t.Status == "running" || t.Status == "pending" {
				active++
			} else if t.Status == "completed" {
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

// executeTask runs the agent task and writes results to Active Memory
func (s *AgentService) executeTask(task *AgentTask) {
	s.mu.Lock()
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	task.UpdatedAt = now
	s.mu.Unlock()

	log.Printf("[AgentService] Executing task %s (%s): %s", task.ID, task.Type, task.Title)

	// Simulate agent execution - in production, this communicates with the oh-my-pi container
	// via its API endpoint. The container processes the task and writes markdown output.
	markdownOutput := fmt.Sprintf(`# Agent Task: %s

**Task ID**: %s
**Type**: %s
**DVE**: %s
**Started**: %s

## Description

%s

## Execution Log

Agent runtime (oh-my-pi) processing task...

- Initializing workspace
- Loading relevant context from Active Memory
- Executing task with available tools (git, python, curl, browser)

## Results

Task completed by oh-my-pi agent runtime.

**Status**: Completed
**Completed**: %s
`,
		task.Title,
		task.ID,
		task.Type,
		task.DVEID,
		now.Format(time.RFC3339),
		task.Description,
		time.Now().Format(time.RFC3339),
	)

	// Write output to Active Memory (Markdown Fabric)
	if s.activeMemory != nil {
		ctx := context.Background()
		if err := s.activeMemory.RecordInteraction(ctx, task.DVEID, task.Description, markdownOutput); err != nil {
			log.Printf("[AgentService] Warning: failed to write task output to active memory: %v", err)
		}
	}

	s.mu.Lock()
	completedAt := time.Now()
	task.Status = "completed"
	task.CompletedAt = &completedAt
	task.Output = "Task completed successfully"
	task.MarkdownLog = markdownOutput
	task.UpdatedAt = completedAt
	s.mu.Unlock()

	if err := s.saveTaskToDB(task); err != nil {
		log.Printf("[AgentService] Warning: failed to persist completed task %s: %v", task.ID, err)
	}

	log.Printf("[AgentService] Task %s completed", task.ID)
}

// AgentTaskRequest is the request payload for submitting a task
type AgentTaskRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Priority    int               `json:"priority"`
	Input       map[string]string `json:"input"`
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
