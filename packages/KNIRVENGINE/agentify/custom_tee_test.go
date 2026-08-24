package agentify

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCustomTEEWithVariousWorkloads(t *testing.T) {
	// Skip on Windows as some commands won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "custom-tee-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	if err := ioutil.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create TEE config
	config := TEEConfig{
		WorkingDir: tempDir,
		ResourceLimits: ResourceLimits{
			MemoryMB:            256,
			CPUCores:            1.0,
			ExecutionTimeout:    30 * time.Second,
			MemoryAlertThreshold: 0.8,
			CPUAlertThreshold:    0.9,
			DiskAlertThreshold:   0.7,
		},
		SecurityPolicy: SecurityPolicy{
			AllowedCommands:    []string{"echo", "cat", "ls", "sleep", "dd", "find", "grep", "sort"},
			BlockedCommands:    []string{"rm", "mv", "cp"},
			AllowNetworkAccess: false,
			MaxExecutionTime:   60 * time.Second,
		},
		Env: map[string]string{
			"TEST_ENV": "test_value",
		},
	}

	// Create CustomTEE
	tee := NewCustomTEE(config)
	if err := tee.Start(); err != nil {
		t.Fatalf("Failed to start CustomTEE: %v", err)
	}
	defer tee.Stop()

	// Test cases for different workload types
	testCases := []struct {
		name     string
		command  string
		args     []string
		validate func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE)
	}{
		{
			name:    "Simple Echo Command",
			command: "echo",
			args:    []string{"Hello, World!"},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				if err != nil {
					t.Errorf("Echo command failed: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				if strings.TrimSpace(stdout) != "Hello, World!" {
					t.Errorf("Expected 'Hello, World!', got '%s'", strings.TrimSpace(stdout))
				}

				// Check resource usage
				usage, err := tee.GetResourceUsage()
				if err != nil {
					t.Errorf("Failed to get resource usage: %v", err)
				} else {
					t.Logf("Echo command resource usage: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
						usage.MemoryMB, usage.CPUPercent, usage.Processes)
				}
			},
		},
		{
			name:    "File Read Command",
			command: "cat",
			args:    []string{testFile},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				if err != nil {
					t.Errorf("Cat command failed: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				if strings.TrimSpace(stdout) != "test content" {
					t.Errorf("Expected 'test content', got '%s'", strings.TrimSpace(stdout))
				}
			},
		},
		{
			name:    "CPU Intensive Workload",
			command: "dd",
			args:    []string{"if=/dev/zero", "of=/dev/null", "bs=1M", "count=100"},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				if err != nil {
					t.Errorf("DD command failed: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}

				// Check resource usage
				usage, err := tee.GetResourceUsage()
				if err != nil {
					t.Errorf("Failed to get resource usage: %v", err)
				} else {
					t.Logf("CPU intensive workload resource usage: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
						usage.MemoryMB, usage.CPUPercent, usage.Processes)
				}
			},
		},
		{
			name:    "Memory Intensive Workload",
			command: "sort",
			args:    []string{"-R", "/dev/urandom", "--buffer-size=50M", "-o", "/dev/null"},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				// This command might fail due to resource constraints, which is expected
				// We're more interested in monitoring the resource usage
				
				// Check resource usage
				usage, err := tee.GetResourceUsage()
				if err != nil {
					t.Logf("Failed to get resource usage: %v", err)
				} else {
					t.Logf("Memory intensive workload resource usage: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
						usage.MemoryMB, usage.CPUPercent, usage.Processes)
				}
			},
		},
		{
			name:    "Long Running Command",
			command: "sleep",
			args:    []string{"2"},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				if err != nil {
					t.Errorf("Sleep command failed: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}

				// Check resource usage
				usage, err := tee.GetResourceUsage()
				if err != nil {
					t.Errorf("Failed to get resource usage: %v", err)
				} else {
					t.Logf("Long running command resource usage: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
						usage.MemoryMB, usage.CPUPercent, usage.Processes)
				}
			},
		},
		{
			name:    "Blocked Command",
			command: "rm",
			args:    []string{"-rf", "/"},
			validate: func(t *testing.T, stdout, stderr string, exitCode int, err error, tee *CustomTEE) {
				if err == nil {
					t.Errorf("Expected blocked command to fail, but it succeeded")
				}
				if !strings.Contains(stderr, "blocked by security policy") {
					t.Errorf("Expected security policy error, got: %s", stderr)
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exitCode, err := tee.Execute(tc.command, tc.args)
			tc.validate(t, stdout, stderr, exitCode, err, tee)
			
			// Allow time for resource stats to update
			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestCustomTEEResourceLimits(t *testing.T) {
	// Skip on Windows as some commands won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "custom-tee-resource-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create TEE config with tight resource limits
	config := TEEConfig{
		WorkingDir: tempDir,
		ResourceLimits: ResourceLimits{
			MemoryMB:            50,  // Very low memory limit
			CPUCores:            0.5, // Half a CPU core
			ExecutionTimeout:    5 * time.Second,
			MemoryAlertThreshold: 0.5,
			CPUAlertThreshold:    0.5,
			DiskAlertThreshold:   0.5,
		},
		SecurityPolicy: SecurityPolicy{
			AllowedCommands:    []string{"echo", "cat", "ls", "sleep", "dd", "find", "grep", "sort"},
			MaxExecutionTime:   10 * time.Second,
		},
	}

	// Create CustomTEE
	tee := NewCustomTEE(config)
	if err := tee.Start(); err != nil {
		t.Fatalf("Failed to start CustomTEE: %v", err)
	}
	defer tee.Stop()

	// Test with a command that should exceed memory limits
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This command attempts to use more memory than allowed
	stdout, stderr, exitCode, err := tee.ExecuteWithContext(ctx, "sort", []string{"-R", "/dev/urandom", "--buffer-size=100M", "-o", "/dev/null"})
	
	// Log the results
	t.Logf("Memory limit test results: exitCode=%d, err=%v", exitCode, err)
	t.Logf("stdout: %s", stdout)
	t.Logf("stderr: %s", stderr)

	// Check resource usage
	usage, err := tee.GetResourceUsage()
	if err != nil {
		t.Logf("Failed to get resource usage: %v", err)
	} else {
		t.Logf("Resource usage during memory limit test: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
			usage.MemoryMB, usage.CPUPercent, usage.Processes)
	}

	// Test with a command that should exceed execution time limits
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This command attempts to run longer than the context timeout
	stdout, stderr, exitCode, err = tee.ExecuteWithContext(ctx, "sleep", []string{"30"})
	
	// The command should be terminated by the context timeout
	if err == nil {
		t.Errorf("Expected timeout error, but command succeeded")
	}
	
	t.Logf("Timeout test results: exitCode=%d, err=%v", exitCode, err)
}

func TestCustomTEEConcurrentCommands(t *testing.T) {
	// Skip on Windows as some commands won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "custom-tee-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create TEE config
	config := TEEConfig{
		WorkingDir: tempDir,
		ResourceLimits: ResourceLimits{
			MemoryMB:            512,
			CPUCores:            2.0,
			ExecutionTimeout:    30 * time.Second,
		},
		SecurityPolicy: SecurityPolicy{
			AllowedCommands:    []string{"echo", "sleep"},
			MaxExecutionTime:   60 * time.Second,
		},
	}

	// Create CustomTEE
	tee := NewCustomTEE(config)
	if err := tee.Start(); err != nil {
		t.Fatalf("Failed to start CustomTEE: %v", err)
	}
	defer tee.Stop()

	// Run multiple commands concurrently
	concurrency := 5
	results := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			// Each goroutine runs a command with a different sleep time
			sleepTime := fmt.Sprintf("%d", id+1)
			stdout, stderr, exitCode, err := tee.Execute("sleep", []string{sleepTime})
			
			result := fmt.Sprintf("Command %d: exitCode=%d, err=%v, stdout=%s, stderr=%s", 
				id, exitCode, err, strings.TrimSpace(stdout), strings.TrimSpace(stderr))
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < concurrency; i++ {
		t.Log(<-results)
	}

	// Check resource usage after concurrent commands
	usage, err := tee.GetResourceUsage()
	if err != nil {
		t.Errorf("Failed to get resource usage: %v", err)
	} else {
		t.Logf("Resource usage after concurrent commands: Memory=%.2fMB, CPU=%.2f%%, Processes=%d", 
			usage.MemoryMB, usage.CPUPercent, usage.Processes)
	}
}