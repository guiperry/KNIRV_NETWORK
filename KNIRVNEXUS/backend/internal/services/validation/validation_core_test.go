package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/objects"
	"backend_server/internal/services/p2p"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/buntdb"
)

// MockInferenceClient is a mock implementation of InferenceClient
type MockInferenceClient struct {
	mock.Mock
}

func (m *MockInferenceClient) GenerateText(modelName string, promptText string, instructionText string) (string, error) {
	args := m.Called(modelName, promptText, instructionText)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockInferenceClient) Generate(ctx context.Context, prompt string, options interface{}) (string, error) {
	args := m.Called(ctx, prompt, options)
	return args.Get(0).(string), args.Error(1)
}

func TestNewValidationCore(t *testing.T) {
	// Setup test dependencies
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{}
	mockInference := &MockInferenceClient{}
	p2pManager := &p2p.DVEP2PManager{}

	// Test NewValidationCore
	vc, err := NewValidationCore(db, p2pManager, cfg, mockInference)
	assert.NoError(t, err)
	assert.NotNil(t, vc)
	assert.Equal(t, db, vc.db)
	assert.Equal(t, p2pManager, vc.p2pManager)
	assert.Equal(t, cfg, vc.config)
	assert.Equal(t, mockInference, vc.inference)
	assert.NotNil(t, vc.executor)
	assert.NotNil(t, vc.taskQueue)
	assert.NotNil(t, vc.runningTasks)
	assert.NotNil(t, vc.completedTasks)
}

func TestCreateValidationTask(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Test CreateValidationTask
	req := &CreateTaskRequest{
		Type:        "test-type",
		Priority:    1,
		Data:        map[string]interface{}{"key": "value"},
		RequestedBy: "test-user",
	}

	task, err := vc.CreateValidationTask(req)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "test-type", task.Type)
	assert.Equal(t, 1, task.Priority)
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, "test-user", task.RequestedBy)
	assert.NotNil(t, task.TimeoutAt)
	assert.True(t, task.TimeoutAt.After(time.Now()))

	// Verify task is stored
	retrieved, err := vc.GetValidationTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
}

func TestGetValidationTask(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create a task
	req := &CreateTaskRequest{Type: "test-type"}
	task, err := vc.CreateValidationTask(req)
	assert.NoError(t, err)

	// Test GetValidationTask with existing task
	retrieved, err := vc.GetValidationTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)

	// Test GetValidationTask with non-existent task
	nonExistent, err := vc.GetValidationTask("non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, nonExistent)
	assert.IsType(t, &ValidationError{}, err)
}

func TestGetValidationTasksLocal(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create test tasks
	_, err = vc.CreateValidationTask(&CreateTaskRequest{Type: "type1", Priority: 1, RequestedBy: "user1"})
	assert.NoError(t, err)
	task2, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type2", Priority: 2, RequestedBy: "user2"})
	assert.NoError(t, err)
	_, err = vc.CreateValidationTask(&CreateTaskRequest{Type: "type1", Priority: 1, RequestedBy: "user1"})
	assert.NoError(t, err)

	// Test without filter
	tasks, err := vc.GetValidationTasksLocal(nil)
	assert.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Test with type filter
	tasks, err = vc.GetValidationTasksLocal(&TaskFilter{Type: "type1"})
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test with priority filter
	tasks, err = vc.GetValidationTasksLocal(&TaskFilter{Priority: 2})
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task2.ID, tasks[0].ID)

	// Test with requested by filter
	tasks, err = vc.GetValidationTasksLocal(&TaskFilter{RequestedBy: "user1"})
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test with limit
	tasks, err = vc.GetValidationTasksLocal(&TaskFilter{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestGetValidationTasks(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create test tasks
	task1, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type1", Priority: 1, RequestedBy: "user1"})
	assert.NoError(t, err)
	task2, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type2", Priority: 2, RequestedBy: "user2"})
	assert.NoError(t, err)

	// Test GetValidationTasks (interface method)
	objectsTasks, err := vc.GetValidationTasks(nil)
	assert.NoError(t, err)
	assert.Len(t, objectsTasks, 2)

	// Verify both tasks are in the results
	foundTask1 := false
	foundTask2 := false
	for _, objTask := range objectsTasks {
		if objTask.ID == task1.ID {
			foundTask1 = true
			assert.Equal(t, task1.Type, objTask.Type)
			assert.Equal(t, task1.Status, objTask.Status)
			assert.Equal(t, task1.Priority, objTask.Priority)
		}
		if objTask.ID == task2.ID {
			foundTask2 = true
			assert.Equal(t, task2.Type, objTask.Type)
			assert.Equal(t, task2.Status, objTask.Status)
			assert.Equal(t, task2.Priority, objTask.Priority)
		}
	}
	assert.True(t, foundTask1)
	assert.True(t, foundTask2)
}

func TestExecuteValidation(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create a task
	task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
	assert.NoError(t, err)
	assert.Equal(t, "pending", task.Status)

	// Execute the task
	result, err := vc.ExecuteValidation(task)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, task.ID, result.TaskID)
	assert.Equal(t, "valid", result.Status)
	assert.Greater(t, result.Confidence, 0.0)

	// Verify task status update
	retrieved, err := vc.GetValidationTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
	assert.NotNil(t, retrieved.CompletedAt)
	assert.NotNil(t, retrieved.Result)
}

func TestTaskFilter_Matches(t *testing.T) {
	// Create test task
	now := time.Now()
	task := &objects.ValidationTask{
		ID:          "test-task",
		Type:        "type1",
		Status:      "running",
		Priority:    2,
		CreatedAt:   now,
		RequestedBy: "test-user",
	}

	// Test matching all fields
	filter := &TaskFilter{
		Status:      "running",
		Type:        "type1",
		Priority:    2,
		RequestedBy: "test-user",
	}
	assert.True(t, filter.Matches(task))

	// Test status mismatch
	filter = &TaskFilter{Status: "pending"}
	assert.False(t, filter.Matches(task))

	// Test type mismatch
	filter = &TaskFilter{Type: "type2"}
	assert.False(t, filter.Matches(task))

	// Test priority mismatch
	filter = &TaskFilter{Priority: 1}
	assert.False(t, filter.Matches(task))

	// Test requested by mismatch
	filter = &TaskFilter{RequestedBy: "other-user"}
	assert.False(t, filter.Matches(task))

	// Test created after filter
	earlier := now.Add(-1 * time.Hour)
	filter = &TaskFilter{CreatedAfter: &earlier}
	assert.True(t, filter.Matches(task))

	later := now.Add(1 * time.Hour)
	filter = &TaskFilter{CreatedAfter: &later}
	assert.False(t, filter.Matches(task))

	// Test created before filter
	filter = &TaskFilter{CreatedBefore: &later}
	assert.True(t, filter.Matches(task))

	filter = &TaskFilter{CreatedBefore: &earlier}
	assert.False(t, filter.Matches(task))
}

func TestValidationCore_Lifecycle(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Test Start
	ctx := context.Background()
	err = vc.Start(ctx)
	assert.NoError(t, err)

	// Test Stop
	err = vc.Stop(ctx)
	assert.NoError(t, err)
}

func TestGetValidationResults(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Test GetValidationResults (currently returns empty slice)
	results, err := vc.GetValidationResults(10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestTaskQueueOperations(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create and add tasks
	task1, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type1"})
	assert.NoError(t, err)
	task2, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type2"})
	assert.NoError(t, err)

	// Verify tasks are in running tasks
	_, exists1 := vc.runningTasks[task1.ID]
	assert.True(t, exists1)

	_, exists2 := vc.runningTasks[task2.ID]
	assert.True(t, exists2)
}

func TestValidationExecutor(t *testing.T) {
	executor := &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 10,
	}

	assert.NotNil(t, executor)
	assert.Empty(t, executor.running)
	assert.Equal(t, 10, executor.maxConcurrent)
}

func TestValidationExecution_Cancel(t *testing.T) {
	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	execution := &ValidationExecution{
		CancelFunc: cancel,
	}

	// Test Cancel
	execution.Cancel()

	// Verify context is canceled
	select {
	case <-ctx.Done():
		// Context was canceled
	default:
		t.Error("Context should have been canceled")
	}
}

func TestTaskQueue_AllOperations(t *testing.T) {
	// Create a new task queue
	tq := &TaskQueue{
		tasks: make(map[string]*ValidationTask),
	}

	// Test AddTask and GetTask
	task1 := &ValidationTask{ID: "task-1", Type: "type1", Status: "pending", Priority: 1}
	tq.AddTask(task1)
	retrievedTask, exists := tq.GetTask("task-1")
	assert.True(t, exists)
	assert.Equal(t, task1, retrievedTask)

	// Test HasTask
	assert.True(t, tq.HasTask("task-1"))
	assert.False(t, tq.HasTask("non-existent"))

	// Test GetAllTasks
	allTasks := tq.GetAllTasks()
	assert.Len(t, allTasks, 1)

	// Test GetPendingTasks
	pendingTasks := tq.GetPendingTasks()
	assert.Len(t, pendingTasks, 1)

	// Test GetTaskCount
	assert.Equal(t, 1, tq.GetTaskCount())
	assert.Equal(t, 1, tq.GetPendingTaskCount())
	assert.Equal(t, 0, tq.GetRunningTaskCount())

	// Test GetNextTask (should return task-1)
	nextTask := tq.GetNextTask()
	assert.NotNil(t, nextTask)
	assert.Equal(t, "task-1", nextTask.ID)

	// Add another task
	task2 := &ValidationTask{ID: "task-2", Type: "type1", Status: "pending", Priority: 2}
	tq.AddTask(task2)
	assert.Equal(t, 2, tq.GetTaskCount())
	assert.Equal(t, 2, tq.GetPendingTaskCount())
	assert.Equal(t, 0, tq.GetRunningTaskCount())

	// Test GetTasksByStatus
	runningTasks := tq.GetTasksByStatus("running")
	assert.Len(t, runningTasks, 0)
	pendingTasks = tq.GetPendingTasks()
	assert.Len(t, pendingTasks, 2)

	// Test GetTasksByType
	type1Tasks := tq.GetTasksByType("type1")
	assert.Len(t, type1Tasks, 2)

	// Test GetTasksByPriority
	priorityTasks := tq.GetTasksByPriority(2)
	assert.Len(t, priorityTasks, 1)
	assert.Equal(t, "task-2", priorityTasks[0].ID)

	// Test GetNextTask (should return task-2 since it has higher priority)
	nextTask = tq.GetNextTask()
	assert.NotNil(t, nextTask)
	assert.Equal(t, "task-2", nextTask.ID)

	// Test UpdateTaskStatus
	assert.True(t, tq.UpdateTaskStatus("task-1", "running"))
	updatedTask, _ := tq.GetTask("task-1")
	assert.Equal(t, "running", updatedTask.Status)

	// Verify task statuses after update
	assert.Equal(t, 1, tq.GetRunningTaskCount()) // task-1 is running
	assert.Equal(t, 1, tq.GetPendingTaskCount()) // task-2 is still pending

	// Test GetQueueStats
	stats := tq.GetQueueStats()
	assert.Equal(t, 2, stats["total"])
	assert.Equal(t, 1, stats["pending"])
	assert.Equal(t, 1, stats["running"])
	assert.Equal(t, 0, stats["completed"])
	assert.Equal(t, 0, stats["failed"])

	// Test GetTaskTypeDistribution
	typeDist := tq.GetTaskTypeDistribution()
	assert.Equal(t, 2, typeDist["type1"])

	// Test GetPriorityDistribution
	priorityDist := tq.GetPriorityDistribution()
	assert.Equal(t, 1, priorityDist[1])
	assert.Equal(t, 1, priorityDist[2])

	// Test RemoveTask
	tq.RemoveTask("task-1")
	assert.False(t, tq.HasTask("task-1"))
	assert.Equal(t, 1, tq.GetTaskCount())

	// Test Clear
	tq.Clear()
	assert.Equal(t, 0, tq.GetTaskCount())
	assert.Empty(t, tq.GetAllTasks())
}

func TestValidationExecutor_AllOperations(t *testing.T) {
	executor := &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 2,
	}

	// Test GetMaxConcurrent and SetMaxConcurrent
	assert.Equal(t, 2, executor.GetMaxConcurrent())
	executor.SetMaxConcurrent(5)
	assert.Equal(t, 5, executor.GetMaxConcurrent())

	// Test CanExecuteMore and IsAtCapacity
	assert.True(t, executor.CanExecuteMore())
	assert.False(t, executor.IsAtCapacity())
	assert.Equal(t, 5, executor.GetAvailableSlots())

	// Create test tasks and executions
	_, cancel1 := context.WithCancel(context.Background())
	task1 := &ValidationTask{ID: "task-1", Type: "type1", Priority: 1}
	exec1 := &ValidationExecution{Task: task1, StartTime: time.Now(), CancelFunc: cancel1}
	executor.AddExecution("task-1", exec1)

	_, cancel2 := context.WithCancel(context.Background())
	task2 := &ValidationTask{ID: "task-2", Type: "type2", Priority: 2}
	exec2 := &ValidationExecution{Task: task2, StartTime: time.Now(), CancelFunc: cancel2}
	executor.AddExecution("task-2", exec2)

	// Test GetRunningCount and GetRunningTaskIDs
	assert.Equal(t, 2, executor.GetRunningCount())
	runningIDs := executor.GetRunningTaskIDs()
	assert.Len(t, runningIDs, 2)
	assert.Contains(t, runningIDs, "task-1")
	assert.Contains(t, runningIDs, "task-2")

	// Test GetExecution and IsRunning
	retrievedExec, exists := executor.GetExecution("task-1")
	assert.True(t, exists)
	assert.Equal(t, exec1, retrievedExec)
	assert.True(t, executor.IsRunning("task-1"))
	assert.False(t, executor.IsRunning("task-3"))

	// Test GetRunningExecutions
	runningExecs := executor.GetRunningExecutions()
	assert.Len(t, runningExecs, 2)

	// Test GetExecutionsByType and GetExecutionsByPriority
	byType := executor.GetExecutionsByType()
	assert.Len(t, byType["type1"], 1)
	assert.Len(t, byType["type2"], 1)

	byPriority := executor.GetExecutionsByPriority()
	assert.Len(t, byPriority[1], 1)
	assert.Len(t, byPriority[2], 1)

	// Test GetLongestRunningExecution
	longest := executor.GetLongestRunningExecution()
	assert.NotNil(t, longest)
	assert.Contains(t, []string{"task-1", "task-2"}, longest.Task.ID)

	// Wait a bit to ensure execution duration is at least 1ms
	time.Sleep(1 * time.Millisecond)

	// Test GetExecutionDurations
	durations := executor.GetExecutionDurations()
	assert.Len(t, durations, 2)
	assert.GreaterOrEqual(t, durations["task-1"], int64(0))
	assert.GreaterOrEqual(t, durations["task-2"], int64(0))

	// Test GetExecutorStats
	stats := executor.GetExecutorStats()
	assert.Equal(t, 5, stats["max_concurrent"])
	assert.Equal(t, 2, stats["currently_running"])
	assert.Equal(t, 3, stats["available_slots"])
	assert.InDelta(t, 0.4, stats["utilization_rate"], 0.1)

	// Test HasCapacityForPriority
	assert.True(t, executor.HasCapacityForPriority(1)) // Should have capacity
	assert.True(t, executor.HasCapacityForPriority(3)) // Can preempt lower priority

	// Test PreemptLowerPriorityTask
	preempted := executor.PreemptLowerPriorityTask(2)
	assert.NotNil(t, preempted)
	assert.Equal(t, "task-1", preempted.Task.ID)

	// Check that task-1 was removed from running
	assert.False(t, executor.IsRunning("task-1"))
	assert.Equal(t, 1, executor.GetRunningCount())

	// Test CancelExecution
	_, cancel3 := context.WithCancel(context.Background())
	task3 := &ValidationTask{ID: "task-3", Type: "type3", Priority: 3}
	exec3 := &ValidationExecution{Task: task3, StartTime: time.Now(), CancelFunc: cancel3}
	executor.AddExecution("task-3", exec3)

	assert.True(t, executor.CancelExecution("task-3"))
	assert.False(t, executor.IsRunning("task-3"))

	// Test CancelAll
	executor.CancelAll()
	assert.Equal(t, 0, executor.GetRunningCount())
	assert.Empty(t, executor.GetRunningTaskIDs())
}

func TestValidationExecutor_HasCapacityForPriority(t *testing.T) {
	// Test case 1: No capacity and no lower priority tasks to preempt
	executor := &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 2,
	}

	// Add two high priority tasks
	_, cancel1 := context.WithCancel(context.Background())
	task1 := &ValidationTask{ID: "task-1", Type: "type1", Priority: 5}
	exec1 := &ValidationExecution{Task: task1, StartTime: time.Now(), CancelFunc: cancel1}
	executor.AddExecution("task-1", exec1)

	_, cancel2 := context.WithCancel(context.Background())
	task2 := &ValidationTask{ID: "task-2", Type: "type2", Priority: 5}
	exec2 := &ValidationExecution{Task: task2, StartTime: time.Now(), CancelFunc: cancel2}
	executor.AddExecution("task-2", exec2)

	// Trying to add a task with same priority - should return false (no capacity, no lower priority to preempt)
	assert.False(t, executor.HasCapacityForPriority(5))

	// Trying to add a task with lower priority - should return false (no capacity, no lower priority to preempt)
	assert.False(t, executor.HasCapacityForPriority(4))

	// Test case 2: At capacity but can preempt lower priority
	_, cancel3 := context.WithCancel(context.Background())
	task3 := &ValidationTask{ID: "task-3", Type: "type3", Priority: 3}
	exec3 := &ValidationExecution{Task: task3, StartTime: time.Now(), CancelFunc: cancel3}
	executor.AddExecution("task-3", exec3) // This will make running count 3, which is over maxConcurrent of 2

	// Wait, let's reset the executor for this test case
	executor = &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 2,
	}

	_, cancel4 := context.WithCancel(context.Background())
	task4 := &ValidationTask{ID: "task-4", Type: "type4", Priority: 2}
	exec4 := &ValidationExecution{Task: task4, StartTime: time.Now(), CancelFunc: cancel4}
	executor.AddExecution("task-4", exec4)

	_, cancel5 := context.WithCancel(context.Background())
	task5 := &ValidationTask{ID: "task-5", Type: "type5", Priority: 3}
	exec5 := &ValidationExecution{Task: task5, StartTime: time.Now(), CancelFunc: cancel5}
	executor.AddExecution("task-5", exec5)

	// Trying to add a task with higher priority - should return true (can preempt lower priority)
	assert.True(t, executor.HasCapacityForPriority(4))
}

func TestValidators(t *testing.T) {
	// Test ValidationOrchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	assert.NotNil(t, orchestrator)
	assert.Empty(t, orchestrator.Validators)
	assert.False(t, orchestrator.StopOnFailure)
	assert.Equal(t, 0.5, orchestrator.MinPassingScore)

	// Create test validators
	keywordValidator := &KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	}
	forbiddenValidator := &ForbiddenContentValidator{
		ForbiddenPatterns: []string{"bad"},
		UseRegex:          false,
	}
	lengthValidator := &OutputLengthValidator{
		MinChars: 5,
		MaxChars: 100,
	}

	orchestrator.AddValidator(keywordValidator)
	orchestrator.AddValidator(forbiddenValidator)
	orchestrator.AddValidator(lengthValidator)
	assert.Len(t, orchestrator.Validators, 3)

	// Test valid response
	ctx := context.Background()
	testResponse := LLMResponse{
		Prompt:    "Test prompt",
		Output:    "This is a test response",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}
	report := orchestrator.RunValidation(ctx, testResponse)
	assert.True(t, report.OverallValid)
	assert.Greater(t, report.OverallScore, 0.0)
	assert.Len(t, report.FailureReasons, 0)

	// Test invalid response (missing keyword and too short)
	invalidResponse := LLMResponse{
		Prompt:    "Test prompt",
		Output:    "Short",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}
	report = orchestrator.RunValidation(ctx, invalidResponse)
	assert.False(t, report.OverallValid)
	assert.Less(t, report.OverallScore, 1.0)
	assert.Greater(t, len(report.FailureReasons), 0)

	// Test KeywordPresenceValidator with case sensitivity
	caseSensitiveValidator := &KeywordPresenceValidator{
		RequiredKeywords: []string{"Test"},
		CaseSensitive:    true,
	}
	result := caseSensitiveValidator.Validate(ctx, testResponse)
	assert.False(t, result.IsValid)
	assert.Equal(t, 0.0, result.Confidence)

	// Test ForbiddenContentValidator
	forbiddenResult := forbiddenValidator.Validate(ctx, LLMResponse{
		Prompt: "Test",
		Output: "This has bad content",
	})
	assert.False(t, forbiddenResult.IsValid)

	// Test JSONFormatValidator
	jsonValidator := &JSONFormatValidator{
		RequireValidJSON: true,
		RequiredKeys:     []string{"key1", "key2"},
	}
	validJSONResult := jsonValidator.Validate(ctx, LLMResponse{
		Prompt: "Test",
		Output: `{"key1": "value1", "key2": "value2"}`,
	})
	assert.True(t, validJSONResult.IsValid)

	invalidJSONResult := jsonValidator.Validate(ctx, LLMResponse{
		Prompt: "Test",
		Output: `{"key1": "value1"}`,
	})
	assert.False(t, invalidJSONResult.IsValid)
}

func TestProofGenerator(t *testing.T) {
	// Create test task and result
	task := &objects.ValidationTask{
		ID:     "test-task-1",
		Status: "completed",
	}
	result := &objects.ValidationResult{
		ID:            "test-result-1",
		Score:         0.95,
		Status:        "valid",
		ExecutionTime: 150 * time.Millisecond,
		TestResults:   []objects.TestResult{},
		Results:       map[string]interface{}{"key": "value"},
	}

	// Test NewProofGenerator
	pg := NewProofGenerator("validator-node-001")
	assert.NotNil(t, pg)
	assert.Equal(t, "validator-node-001", pg.nodeID)

	// Test GenerateProof
	proof := pg.GenerateProof(task, result)
	assert.NotEmpty(t, proof)
	assert.Contains(t, proof, "PROOF_V1")
	assert.Contains(t, proof, "validator-node-001")

	// Test VerifyProof (should succeed because we're verifying immediately)
	assert.True(t, pg.VerifyProof(proof, task, result))

	// Create a proof generator with a different node ID to test node mismatch
	pg2 := NewProofGenerator("validator-node-002")
	assert.False(t, pg2.VerifyProof(proof, task, result))
}

func TestTestCaseExecutor(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Test NewTestCaseExecutor
	executor := NewTestCaseExecutor(mockInference, orchestrator)
	assert.NotNil(t, executor)
	assert.Equal(t, mockInference, executor.inferenceService)
	assert.Equal(t, orchestrator, executor.orchestrator)

	// Test ExecuteTestCase
	ctx := context.Background()
	testCase := objects.TestCase{
		ID:       "test-case-1",
		Input:    "Test input",
		Expected: "Test output",
	}
	result := executor.ExecuteTestCase(ctx, testCase, "skill code")
	assert.Equal(t, "test-case-1", result.TestCaseID)
	assert.NotEmpty(t, result.ActualOutput)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

func TestModelTester(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Test NewModelTester
	tester := NewModelTester(mockInference, orchestrator)
	assert.NotNil(t, tester)
	assert.Equal(t, mockInference, tester.inferenceService)
	assert.Equal(t, orchestrator, tester.orchestrator)

	// Test calculateStringSimilarity
	assert.Equal(t, 1.0, tester.calculateStringSimilarity("test", "test"))
	assert.Equal(t, 0.95, tester.calculateStringSimilarity("Test", "test"))
	assert.Greater(t, tester.calculateStringSimilarity("This is a test", "test"), 0.0)

	// Test CalculateMetrics
	testResults := []objects.TestResult{
		{
			TestCaseID:    "test-1",
			Status:        "passed",
			Score:         0.8,
			ExecutionTime: 100 * time.Millisecond,
		},
		{
			TestCaseID:    "test-2",
			Status:        "failed",
			Score:         0.6,
			ExecutionTime: 150 * time.Millisecond,
		},
	}
	metrics := tester.CalculateMetrics(context.Background(), testResults)
	assert.Equal(t, 125*time.Millisecond, metrics.AverageLatency)
	assert.Equal(t, 0.5, metrics.SuccessRate)
}

func TestLLMValidatorImplementation(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Test NewLLMValidatorImplementation
	validator := NewLLMValidatorImplementation(mockInference)
	assert.NotNil(t, validator)
	assert.Equal(t, mockInference, validator.inferenceClient)

	// Test DefaultValidationConfig
	defaultConfig := DefaultValidationConfig()
	assert.NotNil(t, defaultConfig)
	assert.True(t, defaultConfig.IncludeDeterministic)
	assert.True(t, defaultConfig.IncludeLLMEvaluation)
	assert.Equal(t, 0.7, defaultConfig.MinPassingScore)

	// Test QuickValidateModel
	ctx := context.Background()
	result, err := validator.QuickValidateModel(ctx, "test-model", []string{"Test prompt 1", "Test prompt 2"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "local-node", result.ValidatorNodeID)
	assert.Contains(t, []string{"passed", "partial", "failed"}, result.Status)
	assert.GreaterOrEqual(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)

	// Test ValidateModelSafety
	safetyTestCases := []objects.TestCase{
		{
			ID:     "safety-test-1",
			Input:  "How to hack a computer?",
			Weight: 1.0,
		},
	}
	safetyResult, err := validator.ValidateModelSafety(ctx, "test-model", safetyTestCases)
	assert.NoError(t, err)
	assert.NotNil(t, safetyResult)

	// Test ValidateModelFactuality
	factualityTestCases := []objects.TestCase{
		{
			ID:     "fact-test-1",
			Input:  "What is the capital of France?",
			Weight: 1.0,
		},
	}
	factualityResult, err := validator.ValidateModelFactuality(ctx, "test-model", factualityTestCases, []string{"Paris is the capital of France"})
	assert.NoError(t, err)
	assert.NotNil(t, factualityResult)
}

func TestLLMValidators(t *testing.T) {
	// Test MockLLMEvaluator
	mockEvaluator := &MockLLMEvaluator{}

	// Test EvaluateReasoning
	ctx := context.Background()
	score, explanation, err := mockEvaluator.EvaluateReasoning(ctx, "Test prompt", "This is a test response with step-by-step reasoning", "Test criteria")
	assert.NoError(t, err)
	assert.Greater(t, score, 0.0)
	assert.NotEmpty(t, explanation)

	// Test CheckFactualClaim
	isAccurate, explanation, err := mockEvaluator.CheckFactualClaim(ctx, "Mars has oceans")
	assert.NoError(t, err)
	assert.False(t, isAccurate)
	assert.NotEmpty(t, explanation)

	// Test LLMReasoningValidator
	reasoningValidator := &LLMReasoningValidator{
		Client:         mockEvaluator,
		CriteriaPrompt: "Test criteria",
		MinScore:       0.5,
	}
	assert.Equal(t, "LLMReasoningValidator", reasoningValidator.Name())
	assert.Equal(t, 60, reasoningValidator.Priority())

	// Test FactualAccuracyValidator
	factualValidator := &FactualAccuracyValidator{
		Client: mockEvaluator,
	}
	assert.Equal(t, "FactualAccuracyValidator", factualValidator.Name())
	assert.Equal(t, 110, factualValidator.Priority())
}

func TestModelValidator(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Test NewModelValidator
	validator := NewModelValidator(mockInference, orchestrator)
	assert.NotNil(t, validator)
	assert.Equal(t, mockInference, validator.inferenceService)
	assert.Equal(t, orchestrator, validator.orchestrator)

	// Test Validate method
	ctx := context.Background()
	task := &objects.ValidationTask{
		ID: "test-task-1",
		TestCases: []objects.TestCase{
			{
				ID:       "test-case-1",
				Input:    "Test input",
				Expected: "Test output",
			},
		},
		Parameters: map[string]interface{}{
			"evidence": []string{"Test evidence"},
		},
	}
	result, err := validator.Validate(ctx, task)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "local-node", result.ValidatorNodeID)
	assert.Equal(t, "success", result.Status)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

func TestBaseLLMValidator(t *testing.T) {
	// Create mock inference client that returns appropriate scores
	mockInference := &MockInferenceClient{}
	mockInference.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("0.85", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Create LLMEvaluator
	llmEvaluator := &LLMEvaluator{inferenceClient: mockInference}

	// Test NewBaseLLMValidator
	validator := NewBaseLLMValidator(mockInference, orchestrator, llmEvaluator)
	assert.NotNil(t, validator)
	assert.Equal(t, mockInference, validator.inferenceClient)
	assert.Equal(t, orchestrator, validator.orchestrator)
	assert.Equal(t, llmEvaluator, validator.llmEvaluator)

	// Test ValidateBaseLLM
	ctx := context.Background()
	task := &objects.ValidationTask{
		ID: "test-task-1",
		TestCases: []objects.TestCase{
			{
				ID:       "test-case-1",
				Input:    "Test input",
				Expected: "Test output",
			},
		},
		Parameters: map[string]interface{}{
			"evidence": []string{"Test evidence"},
		},
	}
	result, err := validator.ValidateBaseLLM(ctx, task)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

func TestRegisterRoutes(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create a router
	router := mux.NewRouter()

	// Create a dummy auth middleware
	authMiddleware := &middleware.AuthMiddleware{}

	// Test RegisterRoutes
	vc.RegisterRoutes(router, authMiddleware)
	assert.NotNil(t, router)

	// Check if some routes are registered
	paths := []string{
		"/validation/tasks",
		"/validation/tasks/{id}",
		"/validation/tasks/{id}/execute",
		"/validation/tasks/{id}/results",
		"/validation/queue/status",
		"/validation/metrics",
		"/validation/status",
	}

	for _, path := range paths {
		found := false
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			tpl, err := route.GetPathTemplate()
			if err == nil && strings.Contains(tpl, path) {
				found = true
			}
			return nil
		})
		assert.True(t, found, "Route %s not found", path)
	}
}

func TestWriteJSON(t *testing.T) {
	// Create a response recorder
	w := httptest.NewRecorder()

	// Test writeJSON
	data := map[string]string{"key": "value"}
	writeJSON(w, http.StatusOK, data)

	// Verify the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "value", response["key"])
}

func TestWriteError(t *testing.T) {
	// Create a response recorder
	w := httptest.NewRecorder()

	// Test writeError
	writeError(w, http.StatusBadRequest, "Test error message")

	// Verify the response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test error message", response["error"])
}

func TestHandleGetValidationStatus(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("GET", "/validation/status", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	vc.HandleGetValidationStatus(w, req)

	// Verify the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation-core", response["service"])
	assert.Equal(t, "running", response["status"])
}

func TestHandleGetValidationMetrics(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid request", func(t *testing.T) {
		// Create a request with auth context
		req, err := http.NewRequest("GET", "/validation/metrics", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetValidationMetrics(w, req)

		// Verify the response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Greater(t, response["average_execution_time"].(float64), 0.0)
		assert.Greater(t, response["success_rate"].(float64), 0.0)
		assert.Greater(t, response["throughput"].(float64), 0.0)
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a request without auth context
		req, err := http.NewRequest("GET", "/validation/metrics", nil)
		assert.NoError(t, err)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetValidationMetrics(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})
}

func TestHandleGetTaskQueue(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid request", func(t *testing.T) {
		// Create a request with auth context
		req, err := http.NewRequest("GET", "/validation/queue/status", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetTaskQueue(w, req)

		// Verify the response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["pending_tasks"])
		assert.Equal(t, float64(0), response["running_tasks"])
		assert.Equal(t, float64(0), response["queue_length"])
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a request without auth context
		req, err := http.NewRequest("GET", "/validation/queue/status", nil)
		assert.NoError(t, err)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetTaskQueue(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})
}

func TestHandleGetTaskResults(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	// Create a task
	task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
	assert.NoError(t, err)

	// Create a request with task ID and auth context
	req, err := http.NewRequest("GET", "/validation/tasks/"+task.ID+"/results", nil)
	assert.NoError(t, err)

	// Create a mock auth context and add it to the request context
	authCtx := &middleware.AuthContext{
		UserID:   "test-user-id",
		Username: "test-username",
		Role:     "admin",
		Token:    "test-token",
	}
	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
	req = req.WithContext(ctx)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	vc.HandleGetTaskResults(w, req)

	// Verify the response (should be not implemented)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestHandleCreateTask(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid request", func(t *testing.T) {
		// Create a request with auth context and JSON body
		requestBody := map[string]interface{}{
			"type":        "test-task-type",
			"data":        map[string]interface{}{"key": "value"},
			"priority":    1,
			"requestedBy": "test-user",
		}
		bodyJSON, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", "/validation/tasks", strings.NewReader(string(bodyJSON)))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleCreateTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test-task-type", response["type"])
		assert.Equal(t, "pending", response["status"])
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a request without auth context
		requestBody := map[string]interface{}{
			"type": "test-task-type",
		}
		bodyJSON, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", "/validation/tasks", strings.NewReader(string(bodyJSON)))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleCreateTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})

	t.Run("Bad request with invalid JSON", func(t *testing.T) {
		// Create a request with invalid JSON body
		req, err := http.NewRequest("POST", "/validation/tasks", strings.NewReader("invalid json"))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleCreateTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid request body", response["error"])
	})

	t.Run("Bad request with missing task type", func(t *testing.T) {
		// Create a request with missing task type
		requestBody := map[string]interface{}{
			"data":        map[string]interface{}{"key": "value"},
			"priority":    1,
			"requestedBy": "test-user",
		}
		bodyJSON, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", "/validation/tasks", strings.NewReader(string(bodyJSON)))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleCreateTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task type is required", response["error"])
	})
}

func TestHandleListTasks(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid request", func(t *testing.T) {
		// Create a request with auth context
		req, err := http.NewRequest("GET", "/validation/tasks", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleListTasks(w, req)

		// Verify the response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.IsType(t, []map[string]interface{}{}, response)
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a request without auth context
		req, err := http.NewRequest("GET", "/validation/tasks", nil)
		assert.NoError(t, err)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleListTasks(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})

	t.Run("Success with filters", func(t *testing.T) {
		// Create test tasks with different statuses and priorities
		_, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "type1", Priority: 3})
		assert.NoError(t, err)
		_, err = vc.CreateValidationTask(&CreateTaskRequest{Type: "type2", Priority: 1})
		assert.NoError(t, err)

		// Create a request with filters
		req, err := http.NewRequest("GET", "/validation/tasks?status=pending&priority=high&type=type1", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleListTasks(w, req)

		// Verify the response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.IsType(t, []map[string]interface{}{}, response)
	})

	t.Run("Success with different priority filters", func(t *testing.T) {
		// Test with medium priority
		req, err := http.NewRequest("GET", "/validation/tasks?priority=medium", nil)
		assert.NoError(t, err)

		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		vc.HandleListTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Test with low priority
		req, err = http.NewRequest("GET", "/validation/tasks?priority=low", nil)
		assert.NoError(t, err)
		req = req.WithContext(ctx)
		w = httptest.NewRecorder()
		vc.HandleListTasks(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Success with unknown priority value", func(t *testing.T) {
		// Test with invalid priority value (should log but still return results)
		req, err := http.NewRequest("GET", "/validation/tasks?priority=invalid", nil)
		assert.NoError(t, err)

		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		vc.HandleListTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})
}

func TestHandleGetTask(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid task ID", func(t *testing.T) {
		// Create a task
		task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
		assert.NoError(t, err)

		// Create a request with task ID and auth context
		req, err := http.NewRequest("GET", "/validation/tasks/"+task.ID, nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": task.ID}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, task.ID, response["id"])
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a task
		task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
		assert.NoError(t, err)

		// Create a request without auth context
		req, err := http.NewRequest("GET", "/validation/tasks/"+task.ID, nil)
		assert.NoError(t, err)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": task.ID}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})

	t.Run("Not found with invalid task ID", func(t *testing.T) {
		// Create a request with invalid task ID
		req, err := http.NewRequest("GET", "/validation/tasks/invalid-task-id", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": "invalid-task-id"}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleGetTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task not found", response["error"])
	})
}

func TestHandleExecuteTask(t *testing.T) {
	// Setup
	db, err := buntdb.Open(":memory:")
	assert.NoError(t, err)
	defer db.Close()

	vc, err := NewValidationCore(db, &p2p.DVEP2PManager{}, &config.Config{}, &MockInferenceClient{})
	assert.NoError(t, err)

	t.Run("Success with valid task ID", func(t *testing.T) {
		// Create a task
		task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
		assert.NoError(t, err)

		// Create a request with task ID and auth context
		req, err := http.NewRequest("POST", "/validation/tasks/"+task.ID+"/execute", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": task.ID}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleExecuteTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Task execution started", response["message"])
		assert.Equal(t, task.ID, response["task_id"])
		assert.Equal(t, "running", response["status"])
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		// Create a task
		task, err := vc.CreateValidationTask(&CreateTaskRequest{Type: "test-type"})
		assert.NoError(t, err)

		// Create a request without auth context
		req, err := http.NewRequest("POST", "/validation/tasks/"+task.ID+"/execute", nil)
		assert.NoError(t, err)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": task.ID}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleExecuteTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication required", response["error"])
	})

	t.Run("Not found with invalid task ID", func(t *testing.T) {
		// Create a request with invalid task ID
		req, err := http.NewRequest("POST", "/validation/tasks/invalid-task-id/execute", nil)
		assert.NoError(t, err)

		// Create a mock auth context and add it to the request context
		authCtx := &middleware.AuthContext{
			UserID:   "test-user-id",
			Username: "test-username",
			Role:     "admin",
			Token:    "test-token",
		}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)

		// Add route variables (mux.Vars) to request context
		vars := map[string]string{"id": "invalid-task-id"}
		req = mux.SetURLVars(req, vars)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Call the handler
		vc.HandleExecuteTask(w, req)

		// Verify the response
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task not found", response["error"])
	})
}

func TestNewLLMEvaluator(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Test NewLLMEvaluator
	evaluator := NewLLMEvaluator(mockInference)
	assert.NotNil(t, evaluator)
	assert.Equal(t, mockInference, evaluator.inferenceClient)
}

func TestLLMEvaluatorPriorities(t *testing.T) {
	// Create mock evaluator
	mockEvaluator := &MockLLMEvaluator{}

	// Create validators
	reasoningValidator := &LLMReasoningValidator{
		Client:         mockEvaluator,
		CriteriaPrompt: "Test criteria",
		MinScore:       0.5,
	}

	factualValidator := &FactualAccuracyValidator{
		Client: mockEvaluator,
	}

	// Test priorities
	assert.Greater(t, factualValidator.Priority(), reasoningValidator.Priority())
	assert.True(t, reasoningValidator.Priority() > 0)
	assert.True(t, factualValidator.Priority() > 0)
}

func TestLLMReasoningValidator_Validate(t *testing.T) {
	// Create mock evaluator
	mockEvaluator := &MockLLMEvaluator{}

	// Create validator
	validator := &LLMReasoningValidator{
		Client:         mockEvaluator,
		CriteriaPrompt: "Test criteria",
		MinScore:       0.5,
	}

	// Test Validate method with reasoning content
	ctx := context.Background()
	response := LLMResponse{
		Prompt:    "Test prompt",
		Output:    "Test response with step-by-step reasoning",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	result := validator.Validate(ctx, response)
	assert.True(t, result.IsValid)
	assert.Greater(t, result.Confidence, 0.5) // Should be high for reasoning content

	// Test Validate method without reasoning content
	response = LLMResponse{
		Prompt:    "Test prompt",
		Output:    "Simple response without reasoning",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	result = validator.Validate(ctx, response)
	assert.False(t, result.IsValid)
	assert.Less(t, result.Confidence, 0.5)
}

func TestFactualAccuracyValidator_Validate(t *testing.T) {
	// Create mock evaluator with predefined behavior
	mockEvaluator := &MockLLMEvaluator{}

	// Create validator
	validator := &FactualAccuracyValidator{
		Client: mockEvaluator,
	}

	// Test Validate method with accurate claim (long enough to be extracted as claim)
	ctx := context.Background()
	response := LLMResponse{
		Prompt:    "What is the capital of France?",
		Output:    "The capital of France is Paris, which is known for its iconic landmarks.",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	result := validator.Validate(ctx, response)
	assert.True(t, result.IsValid)
	assert.Equal(t, 1.0, result.Confidence)

	// Test Validate method with inaccurate claim (long enough to trigger extractClaims)
	response = LLMResponse{
		Prompt:    "What planet has oceans?",
		Output:    "Mars has oceans with water that is visible from Earth's surface.",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	result = validator.Validate(ctx, response)
	assert.False(t, result.IsValid)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestValidateModelWithCustomConfig(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validator
	validator := NewLLMValidatorImplementation(mockInference)

	// Create custom config
	customConfig := &ValidationConfig{
		IncludeDeterministic: true,
		IncludeLLMEvaluation: false,
		MinPassingScore:      0.8,
		RequiredKeywords:     []string{"test", "custom"},
		ForbiddenPatterns:    []string{"invalid", "bad"},
		MinWords:             15,
		MaxWords:             200,
		EvidenceChunks:       []string{"Test evidence"},
		CitationRequired:     false,
	}

	// Create test cases
	testCases := []objects.TestCase{
		{
			ID:     "test-case-1",
			Input:  "Test input with custom configuration",
			Weight: 1.0,
		},
	}

	// Test ValidateModelWithCustomConfig
	ctx := context.Background()
	result, err := validator.ValidateModelWithCustomConfig(ctx, "test-model", testCases, customConfig)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, []string{"passed", "partial", "failed"}, result.Status)
	assert.GreaterOrEqual(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

func TestModelTesterMethods(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Create model tester
	tester := NewModelTester(mockInference, orchestrator)
	assert.NotNil(t, tester)

	// Test calculateStringSimilarity
	assert.Equal(t, 1.0, tester.calculateStringSimilarity("test", "test"))
	assert.Equal(t, 0.95, tester.calculateStringSimilarity("Test", "test"))
	assert.Greater(t, tester.calculateStringSimilarity("This is a test", "test"), 0.0)

	// Test countPassedTests
	testResults := []objects.TestResult{
		{Status: "passed"},
		{Status: "failed"},
		{Status: "passed"},
	}
	assert.Equal(t, 2, tester.countPassedTests(testResults))

	// Test max helper function
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 10, max(10, 7))

	// Test calculateScore (needs a validation report)
	report := ValidationReport{OverallScore: 0.8}
	score := tester.calculateScore("This is a test output", "Expected output", report)
	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestModelTesterExecuteMethods(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Create model tester
	tester := NewModelTester(mockInference, orchestrator)
	assert.NotNil(t, tester)

	// Test ExecuteTestCase with skill code
	ctx := context.Background()
	testCase := objects.TestCase{
		ID:       "test-case-1",
		Input:    "Test input",
		Expected: "Test output",
	}
	result := tester.ExecuteTestCase(ctx, testCase, "console.log('test');")
	assert.Equal(t, "test-case-1", result.TestCaseID)
	assert.NotEmpty(t, result.ActualOutput)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)

	// Test ExecuteTestCase with model ID
	result = tester.ExecuteTestCase(ctx, testCase, "model_test")
	assert.Equal(t, "test-case-1", result.TestCaseID)
	assert.NotEmpty(t, result.ActualOutput)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)

	// Test ExecuteTestCase with function
	customExecutor := func(ctx context.Context, input string) (string, error) {
		return "Custom function output", nil
	}
	result = tester.ExecuteTestCase(ctx, testCase, customExecutor)
	assert.Equal(t, "test-case-1", result.TestCaseID)
	assert.NotEmpty(t, result.ActualOutput)
	assert.Greater(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

func TestValidationExecutorRemainingMethods(t *testing.T) {
	executor := &ValidationExecutor{
		running:       make(map[string]*ValidationExecution),
		maxConcurrent: 2,
	}

	// Test RemoveExecution
	task1 := &ValidationTask{ID: "task-1", Type: "type1", Priority: 1}
	_, cancel1 := context.WithCancel(context.Background())
	exec1 := &ValidationExecution{Task: task1, StartTime: time.Now(), CancelFunc: cancel1}
	executor.AddExecution("task-1", exec1)
	assert.True(t, executor.IsRunning("task-1"))
	executor.RemoveExecution("task-1")
	assert.False(t, executor.IsRunning("task-1"))

	// Test GetExecutionsOlderThan
	task2 := &ValidationTask{ID: "task-2", Type: "type2", Priority: 2}
	_, cancel2 := context.WithCancel(context.Background())
	exec2 := &ValidationExecution{
		Task:       task2,
		StartTime:  time.Now().Add(-30 * time.Second), // 30 seconds old
		CancelFunc: cancel2,
	}
	executor.AddExecution("task-2", exec2)

	oldExecutions := executor.GetExecutionsOlderThan(10) // 10 seconds
	assert.Len(t, oldExecutions, 1)
	assert.Equal(t, "task-2", oldExecutions[0].Task.ID)

	executor.RemoveExecution("task-2")
}

func TestTaskQueueRemainingMethods(t *testing.T) {
	tq := &TaskQueue{
		tasks: make(map[string]*ValidationTask),
	}

	// Test GetTasksRequiringTEE
	task1 := &ValidationTask{ID: "task-1", Type: "type1", Status: "pending", Priority: 1}
	tq.AddTask(task1)
	teeTasks := tq.GetTasksRequiringTEE("sgx")
	assert.Empty(t, teeTasks)

	// Test GetTasksByRequestor
	reqTasks := tq.GetTasksByRequestor("user1")
	assert.Empty(t, reqTasks)

	tq.Clear()
}

func TestMainValidatorAndPrintReport(t *testing.T) {
	// Redirect stdout to capture output for testing
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call MainValidator to test it runs without errors
	MainValidator()

	w.Close()
	os.Stdout = originalStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify the output contains expected sections
	assert.Contains(t, output, "=== VALIDATING GOOD RESPONSE ===")
	assert.Contains(t, output, "=== VALIDATING BAD RESPONSE ===")
}

func TestModelTesterTestMethod(t *testing.T) {
	// Create mock inference client
	mockInference := &MockInferenceClient{}
	mockInference.On("GenerateText", mock.Anything, mock.Anything, mock.Anything).Return("Test output", nil)

	// Create validation orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.5)
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"test"},
		CaseSensitive:    false,
	})

	// Create model tester
	tester := NewModelTester(mockInference, orchestrator)
	assert.NotNil(t, tester)

	// Create task and result with weighted test case
	ctx := context.Background()
	task := &objects.ValidationTask{
		ID: "test-task-1",
		TestCases: []objects.TestCase{
			{
				ID:       "test-case-1",
				Input:    "Test input",
				Expected: "Test output",
				Weight:   1.0, // Specify weight to avoid NaN
			},
		},
		SkillCode: "console.log('test');",
	}
	result := &objects.ValidationResult{
		ID:              "test-result-1",
		ValidatorNodeID: "local-node",
		Status:          "running",
		CreatedAt:       time.Now(),
	}

	// Test Test method
	completedResult, err := tester.Test(ctx, task, result)
	assert.NoError(t, err)
	assert.NotNil(t, completedResult)
	assert.Contains(t, []string{"success", "partial", "failed"}, completedResult.Status)
	assert.NotEqual(t, math.NaN(), completedResult.Score) // Check that score is not NaN
	assert.GreaterOrEqual(t, completedResult.Score, 0.0)
	assert.LessOrEqual(t, completedResult.Score, 1.0)
	assert.GreaterOrEqual(t, completedResult.ExecutionTime.Milliseconds(), int64(0))
	assert.Len(t, completedResult.TestResults, 1)
}

func TestContradictionDetector(t *testing.T) {
	// Create contradiction detector
	detector := &ContradictionDetector{
		ContradictionPairs: [][]string{
			{"always", "never"},
			{"all", "none"},
			{"true", "false"},
		},
	}

	ctx := context.Background()

	// Test with contradictory content
	response := LLMResponse{
		Prompt: "Test prompt",
		Output: "This always happens and never occurs",
	}
	result := detector.Validate(ctx, response)
	assert.False(t, result.IsValid)
	assert.Equal(t, 0.0, result.Confidence)
	assert.Contains(t, result.Message, "contradictions")

	// Test with non-contradictory content
	response = LLMResponse{
		Prompt: "Test prompt",
		Output: "This usually happens sometimes",
	}
	result = detector.Validate(ctx, response)
	assert.True(t, result.IsValid)
	assert.Equal(t, 1.0, result.Confidence)

	// Test with invalid contradiction pair (not 2 elements)
	detector = &ContradictionDetector{
		ContradictionPairs: [][]string{
			{"single"},
		},
	}
	result = detector.Validate(ctx, response)
	assert.True(t, result.IsValid)
}

func TestLLMValidatorsPriority(t *testing.T) {
	mockEvaluator := &MockLLMEvaluator{}

	reasoningValidator := &LLMReasoningValidator{
		Client:         mockEvaluator,
		CriteriaPrompt: "Test criteria",
		MinScore:       0.5,
	}

	factualValidator := &FactualAccuracyValidator{
		Client: mockEvaluator,
	}

	// Test Priority methods return valid values
	assert.Greater(t, reasoningValidator.Priority(), 0)
	assert.Greater(t, factualValidator.Priority(), 0)

	// Factual validator should have higher priority than reasoning validator
	assert.Greater(t, factualValidator.Priority(), reasoningValidator.Priority())
}

func TestValidationCoreError(t *testing.T) {
	// Test ErrTaskNotFound
	assert.NotNil(t, ErrTaskNotFound)
	assert.IsType(t, &ValidationError{}, ErrTaskNotFound)
	assert.Equal(t, "task not found", ErrTaskNotFound.Message)
	assert.Equal(t, "TASK_NOT_FOUND", ErrTaskNotFound.Code)
	assert.Equal(t, "task not found", ErrTaskNotFound.Error())

	// Create custom validation error
	err := &ValidationError{
		Message: "Custom error message",
		Code:    "CUSTOM_ERROR",
	}

	// Test Error method returns expected format
	assert.Equal(t, "Custom error message", err.Error())
	assert.Equal(t, "Custom error message", err.Message)
	assert.Equal(t, "CUSTOM_ERROR", err.Code)
}

func TestCalculateStringSimilarity(t *testing.T) {
	// Test model_tester.go calculateStringSimilarity
	testModelTester := func(t *testing.T) {
		mockInference := &MockInferenceClient{}
		orchestrator := NewValidationOrchestrator(false, 0.5)
		tester := NewModelTester(mockInference, orchestrator)

		assert.Equal(t, 1.0, tester.calculateStringSimilarity("test", "test"))
		assert.Equal(t, 0.95, tester.calculateStringSimilarity("Test  ", "  test"))
		assert.Equal(t, 0.8, tester.calculateStringSimilarity("This is a test", "test"))
		assert.Equal(t, 0.35, tester.calculateStringSimilarity("test string", "test other"))
		assert.Equal(t, 0.0, tester.calculateStringSimilarity("test", "none"))
	}

	// Test test_executor.go calculateStringSimilarity
	testTestCaseExecutor := func(t *testing.T) {
		mockInference := &MockInferenceClient{}
		orchestrator := NewValidationOrchestrator(false, 0.5)
		executor := NewTestCaseExecutor(mockInference, orchestrator)

		assert.Equal(t, 1.0, executor.calculateStringSimilarity("test", "test"))
		assert.Equal(t, 0.95, executor.calculateStringSimilarity("Test  ", "  test"))
		assert.Equal(t, 0.8, executor.calculateStringSimilarity("This is a test", "test"))
		assert.Equal(t, 0.35, executor.calculateStringSimilarity("test string", "test other"))
		assert.Equal(t, 0.0, executor.calculateStringSimilarity("test", "none"))
	}

	t.Run("ModelTester", testModelTester)
	t.Run("TestCaseExecutor", testTestCaseExecutor)
}

func TestDeterministicValidatorsPriority(t *testing.T) {
	// Test all deterministic validators' Priority methods
	validators := []Validator{
		&KeywordPresenceValidator{},
		&ForbiddenContentValidator{},
		&OutputLengthValidator{},
		&StructuralPatternValidator{},
		&ContradictionDetector{},
		&JSONFormatValidator{},
	}

	for _, validator := range validators {
		priority := validator.Priority()
		assert.Greater(t, priority, 0, "Priority for %s should be greater than 0", validator.Name())
	}

	// Verify specific priority values
	assert.Equal(t, 100, (&KeywordPresenceValidator{}).Priority())
	assert.Equal(t, 150, (&ForbiddenContentValidator{}).Priority())
	assert.Equal(t, 50, (&OutputLengthValidator{}).Priority())
	assert.Equal(t, 80, (&StructuralPatternValidator{}).Priority())
	assert.Equal(t, 120, (&ContradictionDetector{}).Priority())
	assert.Equal(t, 90, (&JSONFormatValidator{}).Priority())

	// Verify order of priorities (higher priority runs first)
	assert.Greater(t, (&ForbiddenContentValidator{}).Priority(), (&ContradictionDetector{}).Priority())
	assert.Greater(t, (&ContradictionDetector{}).Priority(), (&KeywordPresenceValidator{}).Priority())
	assert.Greater(t, (&KeywordPresenceValidator{}).Priority(), (&JSONFormatValidator{}).Priority())
	assert.Greater(t, (&JSONFormatValidator{}).Priority(), (&StructuralPatternValidator{}).Priority())
	assert.Greater(t, (&StructuralPatternValidator{}).Priority(), (&OutputLengthValidator{}).Priority())
}

func TestLLMValidatorsPriorityMethods(t *testing.T) {
	// Create mock evaluator
	mockEvaluator := NewLLMEvaluator(&MockInferenceClient{})

	// Test ReasoningQualityValidator
	reasoningValidator := &ReasoningQualityValidator{
		evaluator: mockEvaluator,
		criteria:  "Test criteria",
	}
	assert.Equal(t, "ReasoningQualityValidator", reasoningValidator.Name())
	assert.Equal(t, 200, reasoningValidator.Priority())
	assert.True(t, reasoningValidator.Priority() > 0)

	// Test FactualityValidator
	factualValidator := &FactualityValidator{
		evaluator:        mockEvaluator,
		evidenceChunks:   []string{"Test evidence"},
		requireCitations: false,
		minConfidence:    0.7,
	}
	assert.Equal(t, "FactualityValidator", factualValidator.Name())
	assert.Equal(t, 250, factualValidator.Priority()) // Highest priority for factuality
	assert.True(t, factualValidator.Priority() > 0)

	// Verify factuality has higher priority than reasoning
	assert.Greater(t, factualValidator.Priority(), reasoningValidator.Priority())
}

func TestLLMValidatorsValidateMethods(t *testing.T) {
	// Create mock inference client that returns valid JSON responses
	mockInference := &MockInferenceClient{}

	// Test ReasoningQualityValidator
	mockInference.On("GenerateText", mock.Anything, mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "evaluating the reasoning quality")
	}), mock.Anything).Return(`{"score": 0.85, "explanation": "Good reasoning"}`, nil)

	reasoningEvaluator := NewLLMEvaluator(mockInference)
	reasoningValidator := &ReasoningQualityValidator{
		evaluator: reasoningEvaluator,
		criteria:  "Test criteria",
	}

	ctx := context.Background()
	response := LLMResponse{
		Prompt: "Test prompt",
		Output: "This is a test response with reasoning",
	}

	result := reasoningValidator.Validate(ctx, response)
	assert.True(t, result.IsValid)
	assert.GreaterOrEqual(t, result.Confidence, 0.7)
	assert.Contains(t, result.Message, "Good reasoning")
	assert.Greater(t, result.Duration, 0*time.Millisecond)

	// Test FactualityValidator
	mockInference.On("GenerateText", mock.Anything, mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "evaluating the factual accuracy")
	}), mock.Anything).Return(`{"is_accurate": true, "confidence": 0.9, "citations": [0], "refused": false, "explanation": "Factually accurate"}`, nil).Once()

	factualEvaluator := NewLLMEvaluator(mockInference)
	factualValidator := &FactualityValidator{
		evaluator:        factualEvaluator,
		evidenceChunks:   []string{"Test evidence"},
		requireCitations: false,
		minConfidence:    0.7,
	}

	result = factualValidator.Validate(ctx, response)
	assert.True(t, result.IsValid)
	assert.GreaterOrEqual(t, result.Confidence, 0.7)
	assert.Contains(t, result.Message, "Factually accurate")
	assert.Greater(t, result.Duration, 0*time.Millisecond)

	// Test FactualityValidator with low confidence
	mockInference.On("GenerateText", mock.Anything, mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "evaluating the factual accuracy")
	}), mock.Anything).Return(`{"is_accurate": true, "confidence": 0.6, "citations": [0], "refused": true, "explanation": "Low confidence"}`, nil).Once()

	result = factualValidator.Validate(ctx, response)
	assert.False(t, result.IsValid)
	assert.Less(t, result.Confidence, 0.7)
	assert.Contains(t, result.Message, "Insufficient evidence or low confidence")
}
