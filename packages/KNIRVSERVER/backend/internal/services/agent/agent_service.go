// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// AgentTask represents a task submitted to the agent runtime
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

// AgentTaskRequest is the request payload for submitting a task
type AgentTaskRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Priority    int               `json:"priority"`
	Input       map[string]string `json:"input"`
}

// AgentService manages agent lifecycle for DVEs.
// NOTE: The oh-my-pi container agent and external (OpenAI/custom) agent
// implementations have been removed in favor of the KNIRVAGENT DVE Supervisor.
// This service now acts as a lightweight status provider and response broadcaster.
type AgentService struct {
	db          *database.BuntDBManager
	agents      map[string]*AgentStatus // dveID -> AgentStatus
	tasks       map[string]*AgentTask   // taskID -> AgentTask
	mu          sync.RWMutex
	running     bool

	// Broadcast channels for agent responses (dveID -> list of channels)
	subscribers map[string][]chan string
	subsMu      sync.RWMutex
}

// NewAgentService creates a new agent service
func NewAgentService(db *database.BuntDBManager) *AgentService {
	return &AgentService{
		db:          db,
		agents:      make(map[string]*AgentStatus),
		tasks:       make(map[string]*AgentTask),
		subscribers: make(map[string][]chan string),
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
	log.Printf("[AgentService] Started (oh-my-pi/external agent paths deprecated — using KNIRVAGENT supervisor)")
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

// GetAgentStatus returns the status of an agent for a DVE.
// Returns a stub status indicating the agent is managed by KNIRVAGENT.
func (s *AgentService) GetAgentStatus(dveID string) (*AgentStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, ok := s.agents[dveID]
	if !ok {
		// No local agent tracked — return a stub indicating KNIRVAGENT-managed
		return &AgentStatus{
			DVEID:   dveID,
			Running: false,
		}, nil
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

// SubmitTask stores a task record for audit purposes.
// Actual task execution is handled by the KNIRVAGENT supervisor.
func (s *AgentService) SubmitTask(dveID string, req *AgentTaskRequest) (*AgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	task := &AgentTask{
		ID:          uuid.New().String(),
		DVEID:       dveID,
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

	if err := s.saveTaskToDB(task); err != nil {
		log.Printf("[AgentService] Warning: failed to persist task %s: %v", task.ID, err)
	}

	// NOTE: Actual task execution is delegated to the KNIRVAGENT DVE Supervisor.
	// The task record is stored here for audit/logging purposes only.
	log.Printf("[AgentService] Task %s recorded for DVE %s (execution via KNIRVAGENT supervisor)", task.ID, dveID)
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

// saveTaskToDB persists a task to the database
func (s *AgentService) saveTaskToDB(task *AgentTask) error {
	if s.db == nil {
		return nil
	}
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("agent:task:"+task.ID, string(data), nil)
		return err
	})
}

// loadTasksFromDB loads tasks from the database
func (s *AgentService) loadTasksFromDB() error {
	if s.db == nil {
		return nil
	}
	return s.db.GetObjectsByPrefix("agent:task:", func(key string, value []byte) bool {
		var task AgentTask
		if err := json.Unmarshal(value, &task); err != nil {
			log.Printf("[AgentService] Error loading task from DB: %v", err)
			return true
		}
		s.tasks[task.ID] = &task
		return true
	})
}
