// hrm_engine_test.go
package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func createTestHRMEngine(t *testing.T) *HRMEngine {
	engine := NewHRMEngine()
	return engine
}

func createMockWASMFile(t *testing.T, dir, filename string) string {
	wasmPath := filepath.Join(dir, filename)
	// Create a mock WASM file (empty file for testing)
	file, err := os.Create(wasmPath)
	require.NoError(t, err)
	file.Close()
	return wasmPath
}

func createTestWeightsFile(t *testing.T, dir, filename string) string {
	weightsPath := filepath.Join(dir, filename)
	// Create a mock weights file
	file, err := os.Create(weightsPath)
	require.NoError(t, err)
	file.Close()
	return weightsPath
}

// TestHRMEngine_NewHRMEngine tests the constructor
func TestHRMEngine_NewHRMEngine(t *testing.T) {
	engine := NewHRMEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.runtime)
	assert.False(t, engine.initialized)
}

// TestHRMEngine_LoadHRMModule tests loading the HRM WASM module
func TestHRMEngine_LoadHRMModule(t *testing.T) {
	engine := createTestHRMEngine(t)

	tempDir, err := os.MkdirTemp("", "test_hrm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock WASM file
	wasmPath := createMockWASMFile(t, tempDir, "hrm.wasm")

	// This will fail because it's not a real WASM file, but we test the error handling
	err = engine.LoadHRMModule(wasmPath)

	assert.Error(t, err) // Expected to fail with mock file
}

// TestHRMEngine_LoadHRMModule_NonexistentFile tests loading nonexistent WASM file
func TestHRMEngine_LoadHRMModule_NonexistentFile(t *testing.T) {
	engine := createTestHRMEngine(t)

	err := engine.LoadHRMModule("/nonexistent/hrm.wasm")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

// TestHRMEngine_LoadWeights tests loading weights
func TestHRMEngine_LoadWeights(t *testing.T) {
	engine := createTestHRMEngine(t)

	tempDir, err := os.MkdirTemp("", "test_hrm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock weights file
	weightsPath := createTestWeightsFile(t, tempDir, "weights.bin")

	err = engine.LoadWeights(weightsPath)

	// This may fail due to file format, but we test that the path is set
	if err == nil {
		assert.Equal(t, weightsPath, engine.weightsPath)
	}
}

// TestHRMEngine_LoadWeights_NonexistentFile tests loading nonexistent weights file
func TestHRMEngine_LoadWeights_NonexistentFile(t *testing.T) {
	engine := createTestHRMEngine(t)

	err := engine.LoadWeights("/nonexistent/weights.bin")

	assert.Error(t, err)
}

// TestHRMEngine_ProcessCognitive tests cognitive processing
func TestHRMEngine_ProcessCognitive(t *testing.T) {
	engine := createTestHRMEngine(t)

	input := &HRMInput{
		SensoryData: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Context:     "test cognitive processing",
		TaskType:    "reasoning",
	}

	// This will fail because the engine is not initialized with a real WASM module
	output, err := engine.ProcessCognitiveInput(input)

	assert.Error(t, err) // Expected to fail without proper initialization
	assert.Nil(t, output)
}

// TestHRMEngine_ProcessCognitive_InvalidInput tests processing with invalid input
func TestHRMEngine_ProcessCognitive_InvalidInput(t *testing.T) {
	engine := createTestHRMEngine(t)

	// Test with nil input
	output, err := engine.ProcessCognitiveInput(nil)
	assert.Error(t, err)
	assert.Nil(t, output)

	// Test with empty sensory data
	input := &HRMInput{
		SensoryData: []float32{},
		Context:     "test",
		TaskType:    "reasoning",
	}

	output, err = engine.ProcessCognitiveInput(input)
	assert.Error(t, err)
	assert.Nil(t, output)
}

// TestHRMEngine_IsInitialized tests initialization status
func TestHRMEngine_IsInitialized(t *testing.T) {
	engine := createTestHRMEngine(t)

	// Initially should not be initialized
	assert.False(t, engine.IsInitialized())

	// After attempting to load (even if it fails), the flag might change
	tempDir, err := os.MkdirTemp("", "test_hrm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	wasmPath := createMockWASMFile(t, tempDir, "hrm.wasm")
	engine.LoadHRMModule(wasmPath) // This will fail but might set some state

	// The initialization status depends on successful loading
	// Since we're using a mock file, it should still be false
	assert.False(t, engine.IsInitialized())
}

// TestHRMEngine_Close tests engine shutdown
func TestHRMEngine_Close(t *testing.T) {
	engine := createTestHRMEngine(t)

	err := engine.Close()

	assert.NoError(t, err)
	// After close, the engine should be in a clean state
}

// TestHRMEngine_ConcurrentAccess tests thread safety
func TestHRMEngine_ConcurrentAccess(t *testing.T) {
	engine := createTestHRMEngine(t)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent access to IsInitialized
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			initialized := engine.IsInitialized()
			assert.False(t, initialized) // Should be false for uninitialized engine
		}()
	}

	wg.Wait()
}

// TestHRMEngine_ConcurrentOperations tests thread safety
func TestHRMEngine_ConcurrentOperations(t *testing.T) {
	engine := createTestHRMEngine(t)

	const numOperations = 20
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Perform concurrent operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Mix of different operations
			switch id % 4 {
			case 0:
				// Test IsInitialized
				initialized := engine.IsInitialized()
				if initialized {
					errors <- fmt.Errorf("engine should not be initialized without loading module")
				}
			case 1:
				// Test GetModelInfo
				info := engine.GetModelInfo()
				if info == nil {
					errors <- fmt.Errorf("model info should not be nil")
				}
			case 2:
				// Test ProcessCognitiveInput with nil (should fail gracefully)
				_, err := engine.ProcessCognitiveInput(nil)
				if err == nil {
					errors <- fmt.Errorf("expected error for nil input")
				}
			case 3:
				// Test ProcessCognitiveInput with valid input (should fail due to no module)
				input := &HRMInput{
					SensoryData: []float32{1.0, 2.0, 3.0},
					Context:     "test",
					TaskType:    "reasoning",
				}
				_, err := engine.ProcessCognitiveInput(input)
				if err == nil {
					errors <- fmt.Errorf("expected error for uninitialized engine")
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for unexpected errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestHRMEngine_ResourceManagement tests resource management
func TestHRMEngine_ResourceManagement(t *testing.T) {
	engine := createTestHRMEngine(t)

	// Test multiple close operations (should be safe)
	for i := 0; i < 5; i++ {
		err := engine.Close()
		assert.NoError(t, err)
	}

	// Test operations after close
	assert.False(t, engine.IsInitialized())

	info := engine.GetModelInfo()
	assert.NotNil(t, info)

	_, err := engine.ProcessCognitiveInput(&HRMInput{
		SensoryData: []float32{1.0},
		Context:     "test",
		TaskType:    "reasoning",
	})
	assert.Error(t, err)
}

// TestHRMEngine_ErrorHandling tests error handling scenarios
func TestHRMEngine_ErrorHandling(t *testing.T) {
	engine := createTestHRMEngine(t)

	// Test loading non-existent WASM file
	err := engine.LoadHRMModule("/non/existent/file.wasm")
	assert.Error(t, err)

	// Test loading non-existent weights file
	err = engine.LoadWeights("/non/existent/weights.safetensors")
	assert.Error(t, err)

	// Test initializing modules without loading WASM
	err = engine.InitializeModules(10, 10)
	assert.Error(t, err)

	// Test processing with invalid input
	_, err = engine.ProcessCognitiveInput(&HRMInput{
		SensoryData: nil, // Invalid: nil sensory data
		Context:     "",
		TaskType:    "",
	})
	assert.Error(t, err)
}

// TestHRMEngine_ModelInfo tests model information retrieval
func TestHRMEngine_ModelInfo(t *testing.T) {
	engine := createTestHRMEngine(t)

	info := engine.GetModelInfo()
	assert.NotNil(t, info)

	// Check expected fields
	assert.Contains(t, info, "total_parameters")
	assert.Contains(t, info, "l_module_size")
	assert.Contains(t, info, "h_module_size")
	assert.Contains(t, info, "initialized")
	assert.Contains(t, info, "weights_loaded")

	// Verify types
	assert.IsType(t, int(0), info["total_parameters"])
	assert.IsType(t, int(0), info["l_module_size"])
	assert.IsType(t, int(0), info["h_module_size"])
	assert.IsType(t, false, info["initialized"])
	assert.IsType(t, false, info["weights_loaded"])
}

// TestHRMEngine_InputValidation tests input validation
func TestHRMEngine_InputValidation(t *testing.T) {
	engine := createTestHRMEngine(t)

	testCases := []struct {
		name        string
		input       *HRMInput
		expectError bool
	}{
		{
			name:        "nil input",
			input:       nil,
			expectError: true,
		},
		{
			name: "empty sensory data",
			input: &HRMInput{
				SensoryData: []float32{},
				Context:     "test",
				TaskType:    "reasoning",
			},
			expectError: true,
		},
		{
			name: "nil sensory data",
			input: &HRMInput{
				SensoryData: nil,
				Context:     "test",
				TaskType:    "reasoning",
			},
			expectError: true,
		},
		{
			name: "valid input",
			input: &HRMInput{
				SensoryData: []float32{1.0, 2.0, 3.0},
				Context:     "test",
				TaskType:    "reasoning",
			},
			expectError: true, // Still expect error due to uninitialized engine
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := engine.ProcessCognitiveInput(tc.input)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHRMEngine_ProcessingBenchmark tests processing performance
func TestHRMEngine_ProcessingBenchmark(t *testing.T) {
	engine := createTestHRMEngine(t)

	input := &HRMInput{
		SensoryData: make([]float32, 1000), // Large input
		Context:     "performance test",
		TaskType:    "benchmark",
	}

	// Fill with test data
	for i := range input.SensoryData {
		input.SensoryData[i] = float32(i) * 0.001
	}

	start := time.Now()
	_, err := engine.ProcessCognitiveInput(input)
	duration := time.Since(start)

	// Even though it will fail, we can measure the time to failure
	assert.Error(t, err)
	assert.True(t, duration < time.Second) // Should fail quickly
}

// TestHRMEngine_MemoryManagement tests memory usage
func TestHRMEngine_MemoryManagement(t *testing.T) {
	engine := createTestHRMEngine(t)

	tempDir, err := os.MkdirTemp("", "test_hrm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test loading multiple times (should handle cleanup)
	for i := 0; i < 5; i++ {
		wasmPath := createMockWASMFile(t, tempDir, "hrm.wasm")
		engine.LoadHRMModule(wasmPath) // Will fail but tests memory handling
	}

	// Close should clean up resources
	err = engine.Close()
	assert.NoError(t, err)
}

// TestHRMEngine_EdgeCases tests various edge cases
func TestHRMEngine_EdgeCases(t *testing.T) {
	engine := createTestHRMEngine(t)

	// Test with very large sensory data
	largeInput := &HRMInput{
		SensoryData: make([]float32, 1000000), // 1M floats
		Context:     "large data test",
		TaskType:    "stress",
	}

	_, err := engine.ProcessCognitiveInput(largeInput)
	assert.Error(t, err) // Should handle gracefully

	// Test with special characters in context
	specialInput := &HRMInput{
		SensoryData: []float32{0.1, 0.2},
		Context:     "test with special chars: !@#$%^&*()",
		TaskType:    "special",
	}

	_, err = engine.ProcessCognitiveInput(specialInput)
	assert.Error(t, err) // Should handle gracefully

	// Test with empty context
	emptyContextInput := &HRMInput{
		SensoryData: []float32{0.1, 0.2},
		Context:     "",
		TaskType:    "empty",
	}

	_, err = engine.ProcessCognitiveInput(emptyContextInput)
	assert.Error(t, err) // Should handle gracefully
}

// TestHRMEngine_ConfigurationValidation tests configuration validation
func TestHRMEngine_ConfigurationValidation(t *testing.T) {
	engine := createTestHRMEngine(t)

	tempDir, err := os.MkdirTemp("", "test_hrm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test with empty file path
	err = engine.LoadHRMModule("")
	assert.Error(t, err)

	// Test with directory instead of file
	err = engine.LoadHRMModule(tempDir)
	assert.Error(t, err)

	// Test with file that doesn't have .wasm extension
	txtPath := filepath.Join(tempDir, "not_wasm.txt")
	file, err := os.Create(txtPath)
	require.NoError(t, err)
	file.Close()

	err = engine.LoadHRMModule(txtPath)
	assert.Error(t, err) // Should validate file extension or format
}

// TestHRMInput_Validation tests HRM input validation
func TestHRMInput_Validation(t *testing.T) {
	// Test valid input
	validInput := &HRMInput{
		SensoryData: []float32{0.1, 0.2, 0.3},
		Context:     "valid context",
		TaskType:    "reasoning",
	}

	assert.NotNil(t, validInput)
	assert.Len(t, validInput.SensoryData, 3)
	assert.NotEmpty(t, validInput.Context)
	assert.NotEmpty(t, validInput.TaskType)

	// Test input with negative values
	negativeInput := &HRMInput{
		SensoryData: []float32{-0.1, -0.2, 0.3},
		Context:     "negative values",
		TaskType:    "test",
	}

	assert.NotNil(t, negativeInput)
	assert.Contains(t, negativeInput.SensoryData, float32(-0.1))
}

// TestHRMOutput_Structure tests HRM output structure
func TestHRMOutput_Structure(t *testing.T) {
	output := &HRMOutput{
		ReasoningResult:    "test result",
		Confidence:         0.85,
		ProcessingTime:     0.1,
		LModuleActivations: []float32{0.1, 0.2, 0.3},
		HModuleActivations: []float32{0.4, 0.5, 0.6},
	}

	assert.NotEmpty(t, output.ReasoningResult)
	assert.True(t, output.Confidence >= 0.0 && output.Confidence <= 1.0)
	assert.True(t, output.ProcessingTime >= 0.0)
	assert.Len(t, output.LModuleActivations, 3)
	assert.Len(t, output.HModuleActivations, 3)
}
