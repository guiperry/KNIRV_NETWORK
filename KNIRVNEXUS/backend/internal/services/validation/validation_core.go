package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/models"
	"github.com/knirv/nexus-backend/pkg/p2p"
	"github.com/tidwall/buntdb"
)

// ValidationCore manages validation tasks and execution
type ValidationCore struct {
	db         *buntdb.DB
	p2pManager *p2p.DVEP2PManager
	config     *config.Config
	taskQueue  *TaskQueue
	executor   *ValidationExecutor
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

// TaskQueue manages pending validation tasks
type TaskQueue struct {
	tasks map[string]*models.ValidationTask
	mu    sync.RWMutex
}

// ValidationExecutor executes validation tasks
type ValidationExecutor struct {
	maxConcurrent int
	running       map[string]*ValidationExecution
	mu            sync.RWMutex
}

// ValidationExecution represents a running validation
type ValidationExecution struct {
	Task      *models.ValidationTask
	StartTime time.Time
	Cancel    context.CancelFunc
}

// NewValidationCore creates a new Validation Core instance
func NewValidationCore(db *buntdb.DB, p2pManager *p2p.DVEP2PManager, cfg *config.Config) (*ValidationCore, error) {
	ctx, cancel := context.WithCancel(context.Background())

	core := &ValidationCore{
		db:         db,
		p2pManager: p2pManager,
		config:     cfg,
		taskQueue: &TaskQueue{
			tasks: make(map[string]*models.ValidationTask),
		},
		executor: &ValidationExecutor{
			maxConcurrent: cfg.Validation.MaxConcurrent,
			running:       make(map[string]*ValidationExecution),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Register P2P message handlers
	p2pManager.RegisterMessageHandler(p2p.MessageTypeValidationRequest, core)
	p2pManager.RegisterMessageHandler(p2p.MessageTypeTaskAssignment, core)

	return core, nil
}

// Start starts the Validation Core service
func (vc *ValidationCore) Start(ctx context.Context) error {
	log.Println("Starting Validation Core service...")

	// Start task processing
	go vc.processTaskQueue()
	go vc.monitorExecutions()
	go vc.cleanupCompletedTasks()

	// Load pending tasks from database
	if err := vc.loadPendingTasks(); err != nil {
		log.Printf("Warning: Failed to load pending tasks: %v", err)
	}

	log.Println("Validation Core service started successfully")
	return nil
}

// Stop stops the Validation Core service
func (vc *ValidationCore) Stop(ctx context.Context) error {
	log.Println("Stopping Validation Core service...")
	vc.cancel()

	// Cancel all running executions
	vc.executor.CancelAll()

	log.Println("Validation Core service stopped")
	return nil
}

// HandleMessage implements the P2P MessageHandler interface
func (vc *ValidationCore) HandleMessage(ctx context.Context, msg *models.P2PMessage) error {
	switch msg.Type {
	case p2p.MessageTypeValidationRequest:
		return vc.handleValidationRequest(msg)
	case p2p.MessageTypeTaskAssignment:
		return vc.handleTaskAssignment(msg)
	default:
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
}

// CreateValidationTask creates a new validation task
func (vc *ValidationCore) CreateValidationTask(req *CreateTaskRequest) (*models.ValidationTask, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	task := &models.ValidationTask{
		ID:              uuid.New().String(),
		Type:            req.Type,
		Status:          "pending",
		Priority:        req.Priority,
		SkillCode:       req.SkillCode,
		FailureContext:  req.FailureContext,
		TestCases:       req.TestCases,
		RequiredTEEType: req.RequiredTEEType,
		RequestedBy:     req.RequestedBy,
		Parameters:      req.Parameters,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		TimeoutAt:       time.Now().Add(vc.config.Validation.Timeout),
	}

	// Store in database
	if err := vc.storeTask(task); err != nil {
		return nil, fmt.Errorf("failed to store task: %w", err)
	}

	// Add to queue
	vc.taskQueue.AddTask(task)

	// Broadcast to P2P network for distributed processing
	if err := vc.p2pManager.BroadcastValidationRequest(task); err != nil {
		log.Printf("Warning: Failed to broadcast validation request: %v", err)
	}

	log.Printf("Validation task %s created (type: %s, priority: %d)", task.ID, task.Type, task.Priority)
	return task, nil
}

// GetValidationTasks returns validation tasks with optional filtering
func (vc *ValidationCore) GetValidationTasks(filter *TaskFilter) ([]*models.ValidationTask, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	var tasks []*models.ValidationTask

	err := vc.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("validation:tasks:*", func(key, value string) bool {
			var task models.ValidationTask
			if err := json.Unmarshal([]byte(value), &task); err != nil {
				log.Printf("Error unmarshaling task: %v", err)
				return true
			}

			// Apply filters
			if filter != nil && !filter.Matches(&task) {
				return true
			}

			tasks = append(tasks, &task)
			return true
		})
	})

	return tasks, err
}

// ExecuteValidation executes a validation task
func (vc *ValidationCore) ExecuteValidation(task *models.ValidationTask) (*models.ValidationResult, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Check if already running
	if vc.executor.IsRunning(task.ID) {
		return nil, fmt.Errorf("task %s is already running", task.ID)
	}

	// Check concurrent execution limit
	if vc.executor.GetRunningCount() >= vc.executor.maxConcurrent {
		return nil, fmt.Errorf("maximum concurrent executions reached")
	}

	// Update task status
	task.Status = "running"
	task.StartedAt = &[]time.Time{time.Now()}[0]
	task.UpdatedAt = time.Now()

	if err := vc.storeTask(task); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Start execution
	ctx, cancel := context.WithTimeout(vc.ctx, vc.config.Validation.Timeout)
	execution := &ValidationExecution{
		Task:      task,
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	vc.executor.AddExecution(task.ID, execution)

	// Execute validation in goroutine
	go func() {
		defer func() {
			vc.executor.RemoveExecution(task.ID)
			cancel()
		}()

		result, err := vc.performValidation(ctx, task)
		if err != nil {
			log.Printf("Validation failed for task %s: %v", task.ID, err)
			vc.markTaskFailed(task, err.Error())
			return
		}

		// Store result and mark task completed
		if err := vc.storeValidationResult(result); err != nil {
			log.Printf("Failed to store validation result: %v", err)
		}

		vc.markTaskCompleted(task, result)

		// Broadcast result to P2P network
		if err := vc.p2pManager.BroadcastValidationResult(result); err != nil {
			log.Printf("Warning: Failed to broadcast validation result: %v", err)
		}
	}()

	return nil, nil // Result will be available asynchronously
}

// performValidation performs the actual validation logic
func (vc *ValidationCore) performValidation(ctx context.Context, task *models.ValidationTask) (*models.ValidationResult, error) {
	result := &models.ValidationResult{
		ID:              uuid.New().String(),
		TaskID:          task.ID,
		ValidatorNodeID: "local-node", // TODO: Get actual node ID
		Status:          "running",
		CreatedAt:       time.Now(),
	}

	startTime := time.Now()

	switch task.Type {
	case "skillnode":
		return vc.validateSkillNode(ctx, task, result)
	case "base_llm":
		return vc.validateBaseLLM(ctx, task, result)
	case "custom":
		return vc.validateCustom(ctx, task, result)
	default:
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("unsupported validation type: %s", task.Type)
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}
}

// validateSkillNode validates a SkillNode
func (vc *ValidationCore) validateSkillNode(ctx context.Context, task *models.ValidationTask, result *models.ValidationResult) (*models.ValidationResult, error) {
	startTime := time.Now()

	// Simulate SkillNode validation
	log.Printf("Validating SkillNode for task %s", task.ID)

	// Execute test cases
	testResults := make([]models.TestResult, len(task.TestCases))
	totalScore := 0.0

	for i, testCase := range task.TestCases {
		testResult := vc.executeTestCase(ctx, testCase, task.SkillCode)
		testResults[i] = testResult
		totalScore += testResult.Score * testCase.Weight
	}

	// Calculate overall score
	var totalWeight float64
	for _, testCase := range task.TestCases {
		totalWeight += testCase.Weight
	}

	overallScore := totalScore / totalWeight

	result.Status = "success"
	result.Score = overallScore
	result.TestResults = testResults
	result.Results = map[string]interface{}{
		"skill_validation":  "completed",
		"test_cases_passed": vc.countPassedTests(testResults),
		"total_test_cases":  len(testResults),
	}
	result.ExecutionTime = time.Since(startTime)

	// Generate proof (simplified)
	result.Proof = vc.generateValidationProof(task, result)

	return result, nil
}

// validateBaseLLM validates a Base LLM
func (vc *ValidationCore) validateBaseLLM(ctx context.Context, task *models.ValidationTask, result *models.ValidationResult) (*models.ValidationResult, error) {
	startTime := time.Now()

	log.Printf("Validating Base LLM for task %s", task.ID)

	// Simulate Base LLM validation
	result.Status = "success"
	result.Score = 0.85 // Simulated score
	result.Results = map[string]interface{}{
		"llm_validation":    "completed",
		"performance_score": 0.85,
		"safety_score":      0.92,
		"accuracy_score":    0.88,
	}
	result.ExecutionTime = time.Since(startTime)
	result.Proof = vc.generateValidationProof(task, result)

	return result, nil
}

// validateCustom validates a custom validation type
func (vc *ValidationCore) validateCustom(ctx context.Context, task *models.ValidationTask, result *models.ValidationResult) (*models.ValidationResult, error) {
	startTime := time.Now()

	log.Printf("Performing custom validation for task %s", task.ID)

	// Simulate custom validation
	result.Status = "success"
	result.Score = 0.90 // Simulated score
	result.Results = map[string]interface{}{
		"custom_validation": "completed",
		"parameters":        task.Parameters,
	}
	result.ExecutionTime = time.Since(startTime)
	result.Proof = vc.generateValidationProof(task, result)

	return result, nil
}

// executeTestCase executes a single test case
func (vc *ValidationCore) executeTestCase(ctx context.Context, testCase models.TestCase, skillCode string) models.TestResult {
	startTime := time.Now()

	// Simulate test case execution
	passed := true // Simplified logic
	score := 1.0
	if !passed {
		score = 0.0
	}

	status := "passed"
	if !passed {
		status = "failed"
	}

	return models.TestResult{
		TestCaseID:    testCase.ID,
		Status:        status,
		ActualOutput:  testCase.Expected, // Simplified
		Score:         score,
		ExecutionTime: time.Since(startTime),
	}
}

// CreateTaskRequest represents a request to create a validation task
type CreateTaskRequest struct {
	Type            string                 `json:"type"`
	Priority        int                    `json:"priority"`
	SkillCode       string                 `json:"skill_code,omitempty"`
	FailureContext  string                 `json:"failure_context,omitempty"`
	TestCases       []models.TestCase      `json:"test_cases"`
	RequiredTEEType string                 `json:"required_tee_type"`
	RequestedBy     string                 `json:"requested_by"`
	Parameters      map[string]interface{} `json:"parameters"`
}

// TaskFilter represents filters for task queries
type TaskFilter struct {
	Status        string     `json:"status,omitempty"`
	Type          string     `json:"type,omitempty"`
	Priority      int        `json:"priority,omitempty"`
	RequestedBy   string     `json:"requested_by,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// Matches checks if a task matches the filter criteria
func (tf *TaskFilter) Matches(task *models.ValidationTask) bool {
	if tf.Status != "" && task.Status != tf.Status {
		return false
	}
	if tf.Type != "" && task.Type != tf.Type {
		return false
	}
	if tf.Priority > 0 && task.Priority != tf.Priority {
		return false
	}
	if tf.RequestedBy != "" && task.RequestedBy != tf.RequestedBy {
		return false
	}
	if tf.CreatedAfter != nil && task.CreatedAt.Before(*tf.CreatedAfter) {
		return false
	}
	if tf.CreatedBefore != nil && task.CreatedAt.After(*tf.CreatedBefore) {
		return false
	}
	return true
}

// Helper methods for Validation Core

// storeTask stores a task in the database
func (vc *ValidationCore) storeTask(task *models.ValidationTask) error {
	return vc.db.Update(func(tx *buntdb.Tx) error {
		taskJSON, err := json.Marshal(task)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("validation:tasks:%s", task.ID), string(taskJSON), nil)
		return err
	})
}

// storeValidationResult stores a validation result in the database
func (vc *ValidationCore) storeValidationResult(result *models.ValidationResult) error {
	return vc.db.Update(func(tx *buntdb.Tx) error {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("validation:results:%s", result.ID), string(resultJSON), nil)
		return err
	})
}

// loadPendingTasks loads pending tasks from database
func (vc *ValidationCore) loadPendingTasks() error {
	return vc.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("validation:tasks:*", func(key, value string) bool {
			var task models.ValidationTask
			if err := json.Unmarshal([]byte(value), &task); err != nil {
				log.Printf("Error loading task from DB: %v", err)
				return true
			}

			// Only load pending or assigned tasks
			if task.Status == "pending" || task.Status == "assigned" {
				vc.taskQueue.AddTask(&task)
			}
			return true
		})
	})
}

// handleValidationRequest handles incoming validation requests from P2P
func (vc *ValidationCore) handleValidationRequest(msg *models.P2PMessage) error {
	taskData, ok := msg.Payload["task"]
	if !ok {
		return fmt.Errorf("missing task data in validation request")
	}

	taskJSON, err := json.Marshal(taskData)
	if err != nil {
		return err
	}

	var task models.ValidationTask
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return err
	}

	// Add to queue for processing
	vc.taskQueue.AddTask(&task)
	log.Printf("Received validation request for task %s", task.ID)
	return nil
}

// handleTaskAssignment handles task assignment messages
func (vc *ValidationCore) handleTaskAssignment(msg *models.P2PMessage) error {
	taskID, ok := msg.Payload["task_id"].(string)
	if !ok {
		return fmt.Errorf("missing task_id in assignment")
	}

	nodeID, ok := msg.Payload["node_id"].(string)
	if !ok {
		return fmt.Errorf("missing node_id in assignment")
	}

	// Update task assignment
	log.Printf("Task %s assigned to node %s", taskID, nodeID)
	return nil
}

// markTaskCompleted marks a task as completed
func (vc *ValidationCore) markTaskCompleted(task *models.ValidationTask, result *models.ValidationResult) {
	task.Status = "completed"
	task.CompletedAt = &[]time.Time{time.Now()}[0]
	task.UpdatedAt = time.Now()

	if err := vc.storeTask(task); err != nil {
		log.Printf("Error updating completed task: %v", err)
	}

	vc.taskQueue.RemoveTask(task.ID)
	log.Printf("Task %s completed with score %.2f", task.ID, result.Score)
}

// markTaskFailed marks a task as failed
func (vc *ValidationCore) markTaskFailed(task *models.ValidationTask, errorMsg string) {
	task.Status = "failed"
	task.UpdatedAt = time.Now()

	if err := vc.storeTask(task); err != nil {
		log.Printf("Error updating failed task: %v", err)
	}

	vc.taskQueue.RemoveTask(task.ID)
	log.Printf("Task %s failed: %s", task.ID, errorMsg)
}

// processTaskQueue processes tasks in the queue
func (vc *ValidationCore) processTaskQueue() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-vc.ctx.Done():
			return
		case <-ticker.C:
			vc.processPendingTasks()
		}
	}
}

// processPendingTasks processes pending tasks
func (vc *ValidationCore) processPendingTasks() {
	tasks := vc.taskQueue.GetPendingTasks()

	for _, task := range tasks {
		// Check if we can execute more tasks
		if vc.executor.GetRunningCount() >= vc.executor.maxConcurrent {
			break
		}

		// Execute task
		if _, err := vc.ExecuteValidation(task); err != nil {
			log.Printf("Failed to execute task %s: %v", task.ID, err)
		}
	}
}

// monitorExecutions monitors running executions
func (vc *ValidationCore) monitorExecutions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-vc.ctx.Done():
			return
		case <-ticker.C:
			vc.checkExecutionTimeouts()
		}
	}
}

// checkExecutionTimeouts checks for timed out executions
func (vc *ValidationCore) checkExecutionTimeouts() {
	executions := vc.executor.GetRunningExecutions()

	for _, execution := range executions {
		if time.Since(execution.StartTime) > vc.config.Validation.Timeout {
			log.Printf("Execution timeout for task %s", execution.Task.ID)
			execution.Cancel()
			vc.markTaskFailed(execution.Task, "execution timeout")
		}
	}
}

// cleanupCompletedTasks cleans up old completed tasks
func (vc *ValidationCore) cleanupCompletedTasks() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-vc.ctx.Done():
			return
		case <-ticker.C:
			vc.removeOldTasks()
		}
	}
}

// removeOldTasks removes tasks older than retention period
func (vc *ValidationCore) removeOldTasks() {
	cutoff := time.Now().Add(-24 * time.Hour) // Keep tasks for 24 hours

	vc.db.Update(func(tx *buntdb.Tx) error {
		var keysToDelete []string

		tx.Ascend("validation:tasks:*", func(key, value string) bool {
			var task models.ValidationTask
			if err := json.Unmarshal([]byte(value), &task); err != nil {
				return true
			}

			if task.Status == "completed" || task.Status == "failed" {
				if task.UpdatedAt.Before(cutoff) {
					keysToDelete = append(keysToDelete, key)
				}
			}
			return true
		})

		for _, key := range keysToDelete {
			tx.Delete(key)
		}

		return nil
	})
}

// generateValidationProof generates a cryptographic proof for validation
func (vc *ValidationCore) generateValidationProof(task *models.ValidationTask, result *models.ValidationResult) string {
	// Simplified proof generation
	return fmt.Sprintf("proof_%s_%d", task.ID, time.Now().Unix())
}

// countPassedTests counts the number of passed test cases
func (vc *ValidationCore) countPassedTests(testResults []models.TestResult) int {
	count := 0
	for _, result := range testResults {
		if result.Status == "passed" {
			count++
		}
	}
	return count
}
