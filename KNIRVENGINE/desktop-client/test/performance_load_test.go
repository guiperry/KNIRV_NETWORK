package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"KNIRV_Engine/agent"
	"KNIRV_Engine/agentify"
	"KNIRV_Engine/database"
)

// LoadTestMetrics tracks performance metrics during load testing
type LoadTestMetrics struct {
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	MemoryUsageBefore   runtime.MemStats
	MemoryUsageAfter    runtime.MemStats
	MemoryAllocated     uint64
	GoroutinesBefore    int
	GoroutinesAfter     int
	SuccessfulOps       int64
	FailedOps           int64
	TotalOps            int64
	OperationsPerSecond float64
}

// TestAgentPluginCompilationLoad tests agent plugin compilation under load
func TestAgentPluginCompilationLoad(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "load_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	templatesPath := "agent/templates"
	outputPath := filepath.Join(tempDir, "plugins")

	// Create agent builder
	builder, err := agent.NewAgentBuilder(dbPath, templatesPath, outputPath)
	if err != nil {
		t.Fatalf("Failed to create agent builder: %v", err)
	}
	defer builder.Close()

	// Test configurations
	testCases := []struct {
		name           string
		concurrency    int
		agentsPerBatch int
		batches        int
	}{
		{"Low Load", 5, 10, 3},
		{"Medium Load", 10, 20, 5},
		{"High Load", 20, 30, 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := runAgentCompilationLoadTest(t, builder, tc.concurrency, tc.agentsPerBatch, tc.batches)

			// Validate performance metrics
			if metrics.OperationsPerSecond < 1.0 {
				t.Errorf("Performance too low: %.2f ops/sec", metrics.OperationsPerSecond)
			}

			if metrics.FailedOps > metrics.TotalOps/10 { // Allow up to 10% failure rate
				t.Errorf("Too many failures: %d/%d (%.1f%%)",
					metrics.FailedOps, metrics.TotalOps,
					float64(metrics.FailedOps)/float64(metrics.TotalOps)*100)
			}

			// Check memory usage
			memoryIncrease := metrics.MemoryUsageAfter.Alloc - metrics.MemoryUsageBefore.Alloc
			if memoryIncrease > 500*1024*1024 { // 500MB threshold
				t.Errorf("Memory usage increased too much: %d bytes", memoryIncrease)
			}

			t.Logf("Performance Results for %s:", tc.name)
			t.Logf("  Duration: %v", metrics.Duration)
			t.Logf("  Operations/sec: %.2f", metrics.OperationsPerSecond)
			t.Logf("  Success rate: %.1f%%", float64(metrics.SuccessfulOps)/float64(metrics.TotalOps)*100)
			t.Logf("  Memory allocated: %d bytes", metrics.MemoryAllocated)
			t.Logf("  Goroutines created: %d", metrics.GoroutinesAfter-metrics.GoroutinesBefore)
		})
	}
}

// TestConcurrentAgentOperations tests concurrent agent operations
func TestConcurrentAgentOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize agent inferencer
	inferencer := agentify.NewAgentInferencer(tempDir)

	testCases := []struct {
		name        string
		concurrency int
		operations  int
	}{
		{"Low Concurrency", 10, 100},
		{"Medium Concurrency", 50, 500},
		{"High Concurrency", 100, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := runConcurrentOperationsTest(t, inferencer, tc.concurrency, tc.operations)

			// Validate performance
			if metrics.OperationsPerSecond < 10.0 {
				t.Errorf("Performance too low: %.2f ops/sec", metrics.OperationsPerSecond)
			}

			if metrics.FailedOps > metrics.TotalOps/20 { // Allow up to 5% failure rate
				t.Errorf("Too many failures: %d/%d", metrics.FailedOps, metrics.TotalOps)
			}

			t.Logf("Concurrent Operations Results for %s:", tc.name)
			t.Logf("  Duration: %v", metrics.Duration)
			t.Logf("  Operations/sec: %.2f", metrics.OperationsPerSecond)
			t.Logf("  Success rate: %.1f%%", float64(metrics.SuccessfulOps)/float64(metrics.TotalOps)*100)
		})
	}
}

// TestMemoryUsageUnderLoad tests memory usage patterns under load
func TestMemoryUsageUnderLoad(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "memory_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewSimpleDomainDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test memory usage with different data sizes
	testCases := []struct {
		name      string
		dataSize  int // Number of agents to create
		batchSize int
	}{
		{"Small Dataset", 100, 10},
		{"Medium Dataset", 1000, 50},
		{"Large Dataset", 5000, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := runMemoryUsageTest(t, db, tc.dataSize, tc.batchSize)

			// Check memory efficiency
			avgMemoryPerAgent := float64(metrics.MemoryAllocated) / float64(tc.dataSize)
			if avgMemoryPerAgent > 1024*1024 { // 1MB per agent threshold
				t.Errorf("Memory usage per agent too high: %.2f bytes", avgMemoryPerAgent)
			}

			t.Logf("Memory Usage Results for %s:", tc.name)
			t.Logf("  Total memory allocated: %d bytes", metrics.MemoryAllocated)
			t.Logf("  Memory per agent: %.2f bytes", avgMemoryPerAgent)
			t.Logf("  Duration: %v", metrics.Duration)
		})
	}
}

// runAgentCompilationLoadTest runs a load test for agent compilation
func runAgentCompilationLoadTest(t *testing.T, builder *agent.AgentBuilder, concurrency, agentsPerBatch, batches int) LoadTestMetrics {
	var metrics LoadTestMetrics

	// Collect initial metrics
	runtime.ReadMemStats(&metrics.MemoryUsageBefore)
	metrics.GoroutinesBefore = runtime.NumGoroutine()
	metrics.StartTime = time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for batch := 0; batch < batches; batch++ {
		// Create a semaphore to limit concurrency
		sem := make(chan struct{}, concurrency)

		for i := 0; i < agentsPerBatch; i++ {
			wg.Add(1)
			go func(batchNum, agentNum int) {
				defer wg.Done()

				// Acquire semaphore
				sem <- struct{}{}
				defer func() { <-sem }()

				// Create agent configuration
				config := agent.AgentConfig{
					AgentType:   "llm",
					Name:        fmt.Sprintf("LoadTest_Agent_%d_%d", batchNum, agentNum),
					Description: "Load test agent",
					Model:       "gpt-4",
					Instruction: "You are a test agent for load testing.",
				}

				// Build agent
				agentID, err := builder.BuildAgent(config)

				mu.Lock()
				metrics.TotalOps++
				if err != nil {
					metrics.FailedOps++
					t.Logf("Agent build failed for %s: %v", config.Name, err)
				} else {
					metrics.SuccessfulOps++
					t.Logf("Agent build succeeded for %s (ID: %s)", config.Name, agentID)
				}
				mu.Unlock()

				if err == nil && agentID != "" {
					// Optionally clean up the agent to save space
					// This would be done in a real scenario to prevent disk space issues
				}
			}(batch, i)
		}

		// Wait for batch to complete before starting next batch
		wg.Wait()

		// Small delay between batches to prevent overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// Collect final metrics
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	runtime.ReadMemStats(&metrics.MemoryUsageAfter)
	metrics.GoroutinesAfter = runtime.NumGoroutine()
	metrics.MemoryAllocated = metrics.MemoryUsageAfter.TotalAlloc - metrics.MemoryUsageBefore.TotalAlloc

	if metrics.Duration.Seconds() > 0 {
		metrics.OperationsPerSecond = float64(metrics.TotalOps) / metrics.Duration.Seconds()
	}

	return metrics
}

// runConcurrentOperationsTest runs concurrent operations test
func runConcurrentOperationsTest(t *testing.T, inferencer *agentify.AgentInferencer, concurrency, totalOps int) LoadTestMetrics {
	var metrics LoadTestMetrics

	// Collect initial metrics
	runtime.ReadMemStats(&metrics.MemoryUsageBefore)
	metrics.StartTime = time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, concurrency)

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(opNum int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create session ID
			sessionID := fmt.Sprintf("load-test-session-%d", opNum)

			// Perform operations
			ctx := context.Background()

			// Test memory operations
			key := fmt.Sprintf("test-key-%d", opNum)
			value := fmt.Sprintf("test-value-%d", opNum)

			err := inferencer.SetAgentMemory(ctx, sessionID, key, value)

			mu.Lock()
			metrics.TotalOps++
			if err != nil {
				metrics.FailedOps++
				t.Logf("Memory operation failed for session %s: %v", sessionID, err)
			} else {
				metrics.SuccessfulOps++
				t.Logf("Memory operation succeeded for session %s", sessionID)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Collect final metrics
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	runtime.ReadMemStats(&metrics.MemoryUsageAfter)

	if metrics.Duration.Seconds() > 0 {
		metrics.OperationsPerSecond = float64(metrics.TotalOps) / metrics.Duration.Seconds()
	}

	return metrics
}

// runMemoryUsageTest runs memory usage test
func runMemoryUsageTest(t *testing.T, db *database.SimpleDomainDB, dataSize, batchSize int) LoadTestMetrics {
	var metrics LoadTestMetrics

	// Collect initial metrics
	runtime.ReadMemStats(&metrics.MemoryUsageBefore)
	metrics.StartTime = time.Now()

	// Get or create collection for agents
	collection, err := db.GetOrCreateCollection("memory_test_agents")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Create agents in batches to test memory usage
	for i := 0; i < dataSize; i += batchSize {
		end := i + batchSize
		if end > dataSize {
			end = dataSize
		}

		// Create batch of agents
		for j := i; j < end; j++ {
			// Create a simple agent for memory testing
			agent := &database.SimpleAgent{
				ID:           fmt.Sprintf("memory-test-agent-%d", j),
				Name:         fmt.Sprintf("MemoryTest_Agent_%d", j),
				Collection:   "memory_test",
				ImageURL:     "",
				Status:       "active",
				Capabilities: []string{"llm"},
				TokenID:      "0",
				ContractAddr: "0x000...",
				OwnerID:      1,
				CreatedAt:    time.Now(),
			}

			// Create repository and store agent
			repo := database.NewSimpleAgentRepository(collection)
			err := repo.CreateAgent(context.Background(), agent)
			if err != nil {
				t.Logf("Failed to create agent %d: %v", j, err)
			}
		}

		// Force garbage collection to get accurate memory measurements
		runtime.GC()

		// Small delay to prevent overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	// Collect final metrics
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	runtime.ReadMemStats(&metrics.MemoryUsageAfter)
	metrics.MemoryAllocated = metrics.MemoryUsageAfter.TotalAlloc - metrics.MemoryUsageBefore.TotalAlloc

	return metrics
}

// TestDatabasePerformanceUnderLoad tests database performance under concurrent load
func TestDatabasePerformanceUnderLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db_perf_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "perf_test.db")
	db, err := database.NewSimpleDomainDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	testCases := []struct {
		name        string
		concurrency int
		operations  int
	}{
		{"Low DB Load", 5, 100},
		{"Medium DB Load", 15, 500},
		{"High DB Load", 30, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := runDatabaseLoadTest(t, db, tc.concurrency, tc.operations)

			// Validate database performance
			if metrics.OperationsPerSecond < 5.0 {
				t.Errorf("Database performance too low: %.2f ops/sec", metrics.OperationsPerSecond)
			}

			if metrics.FailedOps > metrics.TotalOps/10 { // Allow up to 10% failure rate
				t.Errorf("Too many database failures: %d/%d", metrics.FailedOps, metrics.TotalOps)
			}

			t.Logf("Database Performance Results for %s:", tc.name)
			t.Logf("  Duration: %v", metrics.Duration)
			t.Logf("  Operations/sec: %.2f", metrics.OperationsPerSecond)
			t.Logf("  Success rate: %.1f%%", float64(metrics.SuccessfulOps)/float64(metrics.TotalOps)*100)
		})
	}
}

// runDatabaseLoadTest runs a database load test
func runDatabaseLoadTest(t *testing.T, db *database.SimpleDomainDB, concurrency, totalOps int) LoadTestMetrics {
	var metrics LoadTestMetrics

	runtime.ReadMemStats(&metrics.MemoryUsageBefore)
	metrics.StartTime = time.Now()

	// Get or create collection for load testing
	collection, err := db.GetOrCreateCollection("load_test_agents")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, concurrency)

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(opNum int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			// Create agent for database testing
			agent := &database.SimpleAgent{
				ID:           fmt.Sprintf("load-test-agent-%d", opNum),
				Name:         fmt.Sprintf("PerfTest_Agent_%d", opNum),
				Collection:   "load_test",
				Status:       "active",
				Capabilities: []string{"llm"},
				TokenID:      "0",
				ContractAddr: "0x000...",
				OwnerID:      1,
				CreatedAt:    time.Now(),
			}

			// Create repository and perform operations
			repo := database.NewSimpleAgentRepository(collection)
			err := repo.CreateAgent(context.Background(), agent)

			mu.Lock()
			metrics.TotalOps++
			if err != nil {
				metrics.FailedOps++
			} else {
				metrics.SuccessfulOps++

				// Also test retrieval
				_, getErr := repo.GetAgentByID(context.Background(), agent.ID)
				if getErr != nil {
					metrics.FailedOps++
				}
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	runtime.ReadMemStats(&metrics.MemoryUsageAfter)

	if metrics.Duration.Seconds() > 0 {
		metrics.OperationsPerSecond = float64(metrics.TotalOps) / metrics.Duration.Seconds()
	}

	return metrics
}
