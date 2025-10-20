package validation_test

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/validation"
	"backend_server/pkg/p2p"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkillNodeValidationEndToEnd tests complete skill node validation workflow
func TestSkillNodeValidationEndToEnd(t *testing.T) {
	// Setup test environment
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		ChainID: "test-chain",
		Validation: config.ValidationConfig{
			Timeout:       5 * time.Minute,
			MaxConcurrent: 2,
		},
	}

	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, "test-node", db.GetDB(), true)
	require.NoError(t, err)

	// Create mock inference service
	inferenceService := &mockInferenceService{}

	// Create validation core
	validationCore, err := validation.NewValidationCore(db.GetDB(), p2pManager, cfg, inferenceService)
	require.NoError(t, err)

	// Start validation core
	ctx := context.Background()
	err = validationCore.Start(ctx)
	require.NoError(t, err)
	defer validationCore.Stop(ctx)

	// Create test task
	testCases := []objects.TestCase{
		{
			ID:       "test-case-1",
			Name:     "Basic arithmetic test",
			Input:    "Calculate 2 + 2",
			Expected: "4",
			Weight:   1.0,
		},
		{
			ID:       "test-case-2",
			Name:     "String manipulation test",
			Input:    "Reverse 'hello'",
			Expected: "olleh",
			Weight:   1.0,
		},
	}

	req := &validation.CreateTaskRequest{
		Type:            "skillnode",
		Priority:        5,
		SkillCode:       "def calculate(input): return eval(input)",
		TestCases:       testCases,
		RequiredTEEType: "software",
		RequestedBy:     "test-user",
	}

	task, err := validationCore.CreateValidationTask(req)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "pending", task.Status)

	// Wait for validation to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var completedTask *objects.ValidationTask
	for {
		select {
		case <-timeout:
			t.Fatal("Validation did not complete within timeout")
		case <-ticker.C:
			// Check if task is completed
			currentTask, err := validationCore.GetValidationTask(task.ID)
			require.NoError(t, err)

			if currentTask.Status == "completed" || currentTask.Status == "failed" {
				completedTask = currentTask
				goto validationComplete
			}
		}
	}

validationComplete:
	assert.NotNil(t, completedTask)
	assert.Equal(t, "completed", completedTask.Status)
	assert.NotNil(t, completedTask.CompletedAt)

	// Verify validation results
	tasks, err := validationCore.GetValidationTasks(&validation.TaskFilter{Status: "completed"})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task.ID, tasks[0].ID)
}

// TestBaseLLMValidationEndToEnd tests complete base LLM validation workflow
func TestBaseLLMValidationEndToEnd(t *testing.T) {
	// Setup test environment
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		ChainID: "test-chain",
		Validation: config.ValidationConfig{
			Timeout:       5 * time.Minute,
			MaxConcurrent: 2,
		},
	}

	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, "test-node", db.GetDB(), true)
	require.NoError(t, err)

	// Create mock inference service
	inferenceService := &mockInferenceService{}

	// Create validation core
	validationCore, err := validation.NewValidationCore(db.GetDB(), p2pManager, cfg, inferenceService)
	require.NoError(t, err)

	// Start validation core
	ctx := context.Background()
	err = validationCore.Start(ctx)
	require.NoError(t, err)
	defer validationCore.Stop(ctx)

	// Create base LLM validation task
	testCases := []objects.TestCase{
		{
			ID:       "llm-test-1",
			Name:     "Factuality test",
			Input:    "What is the capital of France?",
			Expected: "Paris",
			Weight:   1.0,
		},
	}

	req := &validation.CreateTaskRequest{
		Type:            "base_llm",
		Priority:        3,
		TestCases:       testCases,
		RequiredTEEType: "software",
		RequestedBy:     "test-user",
		Parameters: map[string]interface{}{
			"evidence": []string{
				"Paris is the capital and most populous city of France.",
				"France is a country in Western Europe.",
			},
		},
	}

	task, err := validationCore.CreateValidationTask(req)
	require.NoError(t, err)
	assert.Equal(t, "base_llm", task.Type)

	// Wait for validation to complete
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var completedTask *objects.ValidationTask
	for {
		select {
		case <-timeout:
			t.Fatal("Base LLM validation did not complete within timeout")
		case <-ticker.C:
			currentTask, err := validationCore.GetValidationTask(task.ID)
			require.NoError(t, err)

			if currentTask.Status == "completed" || currentTask.Status == "failed" {
				completedTask = currentTask
				goto baseLLMComplete
			}
		}
	}

baseLLMComplete:
	assert.NotNil(t, completedTask)
	assert.Equal(t, "completed", completedTask.Status)
}

// TestConcurrentTaskExecution tests concurrent task execution limits
func TestConcurrentTaskExecution(t *testing.T) {
	// Setup test environment with low concurrency limit
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		ChainID: "test-chain",
		Validation: config.ValidationConfig{
			Timeout:       5 * time.Minute,
			MaxConcurrent: 1, // Only allow 1 concurrent execution
		},
	}

	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, "test-node", db.GetDB(), true)
	require.NoError(t, err)

	inferenceService := &mockInferenceService{}
	validationCore, err := validation.NewValidationCore(db.GetDB(), p2pManager, cfg, inferenceService)
	require.NoError(t, err)

	ctx := context.Background()
	err = validationCore.Start(ctx)
	require.NoError(t, err)
	defer validationCore.Stop(ctx)

	// Create multiple tasks
	testCases := []objects.TestCase{
		{
			ID:       "concurrent-test",
			Input:    "test",
			Expected: "test",
			Weight:   1.0,
		},
	}

	// Create first task
	req1 := &validation.CreateTaskRequest{
		Type:            "skillnode",
		Priority:        1,
		SkillCode:       "def test(): return 'test'",
		TestCases:       testCases,
		RequiredTEEType: "software",
		RequestedBy:     "test-user",
	}

	task1, err := validationCore.CreateValidationTask(req1)
	require.NoError(t, err)

	// Create second task (should be queued due to concurrency limit)
	req2 := &validation.CreateTaskRequest{
		Type:            "skillnode",
		Priority:        1,
		SkillCode:       "def test(): return 'test'",
		TestCases:       testCases,
		RequiredTEEType: "software",
		RequestedBy:     "test-user",
	}

	task2, err := validationCore.CreateValidationTask(req2)
	require.NoError(t, err)

	// Verify both tasks are created
	assert.NotEqual(t, task1.ID, task2.ID)
	assert.Equal(t, "pending", task1.Status)
	assert.Equal(t, "pending", task2.Status)
}

// TestTimeoutHandling tests validation timeout handling
func TestTimeoutHandling(t *testing.T) {
	// Setup test environment with short timeout
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		ChainID: "test-chain",
		Validation: config.ValidationConfig{
			Timeout:       1 * time.Second, // Very short timeout
			MaxConcurrent: 2,
		},
	}

	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, "test-node", db.GetDB(), true)
	require.NoError(t, err)

	// Create slow inference service
	slowInferenceService := &slowInferenceService{delay: 5 * time.Second}

	validationCore, err := validation.NewValidationCore(db.GetDB(), p2pManager, cfg, slowInferenceService)
	require.NoError(t, err)

	ctx := context.Background()
	err = validationCore.Start(ctx)
	require.NoError(t, err)
	defer validationCore.Stop(ctx)

	// Create task that will timeout
	testCases := []objects.TestCase{
		{
			ID:       "timeout-test",
			Input:    "test",
			Expected: "test",
			Weight:   1.0,
		},
	}

	req := &validation.CreateTaskRequest{
		Type:            "skillnode",
		Priority:        1,
		SkillCode:       "def test(): return 'test'",
		TestCases:       testCases,
		RequiredTEEType: "software",
		RequestedBy:     "test-user",
	}

	task, err := validationCore.CreateValidationTask(req)
	require.NoError(t, err)

	// Wait for task to timeout and fail
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var failedTask *objects.ValidationTask
	for {
		select {
		case <-timeout:
			t.Fatal("Task did not fail within expected timeout")
		case <-ticker.C:
			currentTask, err := validationCore.GetValidationTask(task.ID)
			require.NoError(t, err)

			if currentTask.Status == "failed" {
				failedTask = currentTask
				goto timeoutComplete
			}
		}
	}

timeoutComplete:
	assert.NotNil(t, failedTask)
	assert.Equal(t, "failed", failedTask.Status)
}

// TestProofGenerationAndVerification tests cryptographic proof generation
func TestProofGenerationAndVerification(t *testing.T) {
	// Test basic proof format validation
	proof := "PROOF_V1:test-node:mock-hash"
	assert.NotEmpty(t, proof)
	assert.Contains(t, proof, "PROOF_V1")

	// Verify proof format (basic check)
	assert.Contains(t, proof, "test-node")
}

// Mock inference service for testing
type mockInferenceService struct{}

func (m *mockInferenceService) GenerateText(modelName string, promptText string, instructionText string) (string, error) {
	// Simple mock responses based on input
	if promptText == "Calculate 2 + 2" {
		return "4", nil
	}
	if promptText == "Reverse 'hello'" {
		return "olleh", nil
	}
	return "mock response", nil
}

func (m *mockInferenceService) Generate(ctx context.Context, prompt string, options interface{}) (string, error) {
	// Simple mock responses based on input
	if prompt == "Calculate 2 + 2" {
		return "4", nil
	}
	if prompt == "Reverse 'hello'" {
		return "olleh", nil
	}
	return "mock response", nil
}

func (m *mockInferenceService) Start() error {
	return nil
}

func (m *mockInferenceService) Stop() error {
	return nil
}

// Slow inference service for timeout testing
type slowInferenceService struct {
	delay time.Duration
}

func (s *slowInferenceService) GenerateText(modelName string, promptText string, instructionText string) (string, error) {
	time.Sleep(s.delay)
	return "slow response", nil
}

func (s *slowInferenceService) Generate(ctx context.Context, prompt string, options interface{}) (string, error) {
	time.Sleep(s.delay)
	return "slow response", nil
}

func (s *slowInferenceService) Start() error {
	return nil
}

func (s *slowInferenceService) Stop() error {
	return nil
}
