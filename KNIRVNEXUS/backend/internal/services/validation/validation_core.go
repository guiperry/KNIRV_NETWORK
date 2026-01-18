package validation

import (
	"context"
	"sync"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/objects"
	"backend_server/internal/services/p2p"

	"github.com/tidwall/buntdb"
)

// ValidationCore is the main validation service
type ValidationCore struct {
	db             *buntdb.DB
	p2pManager     *p2p.DVEP2PManager
	config         *config.Config
	inference      InferenceClient
	executor       *ValidationExecutor
	taskQueue      *TaskQueue
	mu             sync.RWMutex
	runningTasks   map[string]*ValidationTask
	completedTasks map[string]*TaskResult
}

// ValidationTask represents a validation task
type ValidationTask struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"` // pending, running, completed, failed
	Priority       int                    `json:"priority"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Data           map[string]interface{} `json:"data"`
	Result         *TaskResult            `json:"result,omitempty"`
	AssignedNodeID string                 `json:"assigned_node_id,omitempty"`
	TimeoutAt      *time.Time             `json:"timeout_at,omitempty"`
	RequestedBy    string                 `json:"requested_by,omitempty"`
}

// TaskResult represents the result of a validation task
type TaskResult struct {
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"` // valid, invalid, inconclusive
	Confidence  float64                `json:"confidence"`
	CompletedAt time.Time              `json:"completed_at"`
	Details     map[string]interface{} `json:"details"`
	Proof       string                 `json:"proof,omitempty"`
}

// ValidationExecutor manages concurrent validation executions
type ValidationExecutor struct {
	mu            sync.RWMutex
	running       map[string]*ValidationExecution
	maxConcurrent int
}

// ValidationExecution represents a running validation execution
type ValidationExecution struct {
	Task       *ValidationTask
	StartTime  time.Time
	CancelFunc context.CancelFunc
}

// Cancel cancels the execution
func (e *ValidationExecution) Cancel() {
	if e.CancelFunc != nil {
		e.CancelFunc()
	}
}

// TaskFilter filters validation tasks
type TaskFilter struct {
	Status        string     `json:"status,omitempty"`
	Type          string     `json:"type,omitempty"`
	Priority      int        `json:"priority,omitempty"`
	RequestedBy   string     `json:"requested_by,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// Matches checks if a task matches the filter criteria
func (f *TaskFilter) Matches(task *objects.ValidationTask) bool {
	if f.Status != "" && task.Status != f.Status {
		return false
	}
	if f.Type != "" && task.Type != f.Type {
		return false
	}
	if f.Priority != 0 && task.Priority != f.Priority {
		return false
	}
	if f.RequestedBy != "" && task.RequestedBy != f.RequestedBy {
		return false
	}
	if f.CreatedAfter != nil && task.CreatedAt.Before(*f.CreatedAfter) {
		return false
	}
	if f.CreatedBefore != nil && task.CreatedAt.After(*f.CreatedBefore) {
		return false
	}
	return true
}

// CreateTaskRequest represents a request to create a validation task
type CreateTaskRequest struct {
	Type           string                 `json:"type"`
	Priority       int                    `json:"priority,omitempty"`
	Data           map[string]interface{} `json:"data"`
	SkillCode      string                 `json:"skill_code,omitempty"`
	TestCases      []objects.TestCase     `json:"test_cases,omitempty"`
	RequiredTEEType string                `json:"required_tee_type,omitempty"`
	RequestedBy    string                 `json:"requested_by,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// TaskQueue manages validation task queue
type TaskQueue struct {
	mu    sync.RWMutex
	tasks map[string]*ValidationTask
}

// NewValidationCore creates a new ValidationCore instance
func NewValidationCore(db *buntdb.DB, p2pManager *p2p.DVEP2PManager, cfg *config.Config, inferenceService InferenceClient) (*ValidationCore, error) {
	executor := &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 10, // Default max concurrent executions
	}

	queue := &TaskQueue{
		tasks: make(map[string]*ValidationTask),
	}

	return &ValidationCore{
		db:             db,
		p2pManager:     p2pManager,
		config:         cfg,
		inference:      inferenceService,
		executor:       executor,
		taskQueue:      queue,
		runningTasks:   make(map[string]*ValidationTask),
		completedTasks: make(map[string]*TaskResult),
	}, nil
}

// CreateValidationTask creates a new validation task
func (vc *ValidationCore) CreateValidationTask(req *CreateTaskRequest) (*ValidationTask, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Set default timeout (e.g., 1 hour from now)
	timeoutAt := time.Now().Add(1 * time.Hour)

	task := &ValidationTask{
		ID:             generateTaskID(),
		Type:           req.Type,
		Status:         "pending",
		Priority:       req.Priority,
		CreatedAt:      time.Now(),
		Data:           req.Data,
		RequestedBy:    req.RequestedBy,
		TimeoutAt:      &timeoutAt,
		AssignedNodeID: "", // Will be assigned when executed
	}

	// Store task in database (simplified)
	// In real implementation, would store in buntdb
	vc.runningTasks[task.ID] = task

	return task, nil
}

// GetValidationTasksLocal retrieves validation tasks with optional filtering (returns local types)
func (vc *ValidationCore) GetValidationTasksLocal(filter *TaskFilter) ([]*ValidationTask, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	var tasks []*ValidationTask
	for _, task := range vc.runningTasks {
		if filter != nil {
			if filter.Status != "" && task.Status != filter.Status {
				continue
			}
			if filter.Type != "" && task.Type != filter.Type {
				continue
			}
			if filter.Priority != 0 && task.Priority != filter.Priority {
				continue
			}
			if filter.RequestedBy != "" && task.RequestedBy != filter.RequestedBy {
				continue
			}
		}
		tasks = append(tasks, task)
	}

	// Apply limit if specified
	if filter != nil && filter.Limit > 0 && filter.Limit < len(tasks) {
		tasks = tasks[:filter.Limit]
	}

	return tasks, nil
}

// GetValidationTask retrieves a specific validation task
func (vc *ValidationCore) GetValidationTask(taskID string) (*ValidationTask, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	task, exists := vc.runningTasks[taskID]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return task, nil
}

// GetValidationResults returns validation results (implements ValidationClient interface)
func (vc *ValidationCore) GetValidationResults(limit int) ([]*objects.ValidationResult, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// For now, return empty slice
	return []*objects.ValidationResult{}, nil
}

// Start starts the validation core service
func (vc *ValidationCore) Start(ctx context.Context) error {
	// Implementation would start background tasks, etc.
	// For now, just return nil
	return nil
}

// Stop stops the validation core service
func (vc *ValidationCore) Stop(ctx context.Context) error {
	// Implementation would stop background tasks, etc.
	// For now, just return nil
	return nil
}

// GetValidationTasks returns validation tasks (implements ValidationClient interface)
func (vc *ValidationCore) GetValidationTasks(filter *TaskFilter) ([]*objects.ValidationTask, error) {
	// Call the local method and convert results
	localTasks, err := vc.GetValidationTasksLocal(filter)
	if err != nil {
		return nil, err
	}

	// Convert to objects.ValidationTask
	var result []*objects.ValidationTask
	for _, task := range localTasks {
		// Set default timeout if not set
		timeoutAt := time.Time{}
		if task.TimeoutAt != nil {
			timeoutAt = *task.TimeoutAt
		}
		
		objTask := &objects.ValidationTask{
			ID:             task.ID,
			Type:           task.Type,
			Status:         task.Status,
			Priority:       task.Priority,
			CreatedAt:      task.CreatedAt,
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			AssignedNodeID: task.AssignedNodeID,
			RequestedBy:    task.RequestedBy,
			Parameters:     task.Data, // Map Data to Parameters
			TimeoutAt:      timeoutAt,
		}
		result = append(result, objTask)
	}
	return result, nil
}

// ExecuteValidation executes a validation task
func (vc *ValidationCore) ExecuteValidation(task *ValidationTask) (*TaskResult, error) {
	// Implementation would execute validation logic
	// For now, return a mock result
	result := &TaskResult{
		TaskID:      task.ID,
		Status:      "valid",
		Confidence:  0.95,
		CompletedAt: time.Now(),
		Details: map[string]interface{}{
			"execution_time": "150ms",
			"validator":      "default",
		},
	}

	vc.mu.Lock()
	task.Status = "completed"
	task.CompletedAt = &result.CompletedAt
	task.Result = result
	vc.completedTasks[task.ID] = result
	vc.mu.Unlock()

	return result, nil
}

// Helper functions
func generateTaskID() string {
	return "task-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// getNodeID returns a mock node ID for validation
func getNodeID() string {
	return "validator-node-001"
}

// Error definitions
var (
	ErrTaskNotFound = &ValidationError{Message: "task not found", Code: "TASK_NOT_FOUND"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
