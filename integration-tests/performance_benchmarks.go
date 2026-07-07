package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// Performance Benchmark Suite
type PerformanceBenchmarkSuite struct {
	baseURL    string
	httpClient *http.Client
	results    []BenchmarkResult
	mutex      sync.Mutex
}

// Benchmark result structure
type BenchmarkResult struct {
	BenchmarkName      string                   `json:"benchmark_name"`
	TotalRequests      int                      `json:"total_requests"`
	SuccessfulRequests int                      `json:"successful_requests"`
	FailedRequests     int                      `json:"failed_requests"`
	SuccessRate        float64                  `json:"success_rate"`
	TotalDuration      time.Duration            `json:"total_duration"`
	AverageLatency     time.Duration            `json:"average_latency"`
	MinLatency         time.Duration            `json:"min_latency"`
	MaxLatency         time.Duration            `json:"max_latency"`
	RequestsPerSecond  float64                  `json:"requests_per_second"`
	Percentiles        map[string]time.Duration `json:"percentiles"`
	Metrics            map[string]interface{}   `json:"metrics"`
	Timestamp          time.Time                `json:"timestamp"`
}

// Performance test configuration
type PerformanceTestConfig struct {
	ConcurrentUsers int           `json:"concurrent_users"`
	RequestsPerUser int           `json:"requests_per_user"`
	TestDuration    time.Duration `json:"test_duration"`
	RampUpTime      time.Duration `json:"ramp_up_time"`
	ThinkTime       time.Duration `json:"think_time"`
	MaxLatency      time.Duration `json:"max_latency"`
	MinSuccessRate  float64       `json:"min_success_rate"`
}

// Create new performance benchmark suite
func NewPerformanceBenchmarkSuite(baseURL string) *PerformanceBenchmarkSuite {
	return &PerformanceBenchmarkSuite{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		results: make([]BenchmarkResult, 0),
	}
}

// Add benchmark result
func (suite *PerformanceBenchmarkSuite) addResult(result BenchmarkResult) {
	suite.mutex.Lock()
	defer suite.mutex.Unlock()
	suite.results = append(suite.results, result)
}

// Execute performance test
func (suite *PerformanceBenchmarkSuite) executePerformanceTest(
	benchmarkName string,
	endpoint string,
	payload interface{},
	config PerformanceTestConfig,
) BenchmarkResult {

	fmt.Printf("Running benchmark: %s\n", benchmarkName)

	totalRequests := config.ConcurrentUsers * config.RequestsPerUser
	results := make(chan time.Duration, totalRequests)
	errors := make(chan error, totalRequests)

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch concurrent users
	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)

		go func(userID int) {
			defer wg.Done()

			// Ramp-up delay
			rampDelay := time.Duration(userID) * (config.RampUpTime / time.Duration(config.ConcurrentUsers))
			time.Sleep(rampDelay)

			// Execute requests for this user
			for req := 0; req < config.RequestsPerUser; req++ {
				requestStart := time.Now()

				err := suite.makeRequest(endpoint, payload)
				requestDuration := time.Since(requestStart)

				if err != nil {
					errors <- err
				} else {
					results <- requestDuration
				}

				// Think time between requests
				if config.ThinkTime > 0 && req < config.RequestsPerUser-1 {
					time.Sleep(config.ThinkTime)
				}
			}
		}(user)
	}

	// Wait for all users to complete
	wg.Wait()
	close(results)
	close(errors)

	totalDuration := time.Since(startTime)

	// Collect results
	var latencies []time.Duration
	successCount := 0
	errorCount := 0

	for latency := range results {
		latencies = append(latencies, latency)
		successCount++
	}

	for range errors {
		errorCount++
	}

	// Calculate statistics
	result := BenchmarkResult{
		BenchmarkName:      benchmarkName,
		TotalRequests:      totalRequests,
		SuccessfulRequests: successCount,
		FailedRequests:     errorCount,
		SuccessRate:        float64(successCount) / float64(totalRequests),
		TotalDuration:      totalDuration,
		RequestsPerSecond:  float64(successCount) / totalDuration.Seconds(),
		Percentiles:        make(map[string]time.Duration),
		Metrics:            make(map[string]interface{}),
		Timestamp:          time.Now(),
	}

	if len(latencies) > 0 {
		// Sort latencies for percentile calculation
		for i := 0; i < len(latencies)-1; i++ {
			for j := i + 1; j < len(latencies); j++ {
				if latencies[i] > latencies[j] {
					latencies[i], latencies[j] = latencies[j], latencies[i]
				}
			}
		}

		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[len(latencies)-1]

		// Calculate average
		var totalLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
		}
		result.AverageLatency = totalLatency / time.Duration(len(latencies))

		// Calculate percentiles
		result.Percentiles["p50"] = latencies[len(latencies)*50/100]
		result.Percentiles["p90"] = latencies[len(latencies)*90/100]
		result.Percentiles["p95"] = latencies[len(latencies)*95/100]
		result.Percentiles["p99"] = latencies[len(latencies)*99/100]
	}

	// Add configuration metrics
	result.Metrics["concurrent_users"] = config.ConcurrentUsers
	result.Metrics["requests_per_user"] = config.RequestsPerUser
	result.Metrics["test_duration"] = config.TestDuration.Seconds()
	result.Metrics["ramp_up_time"] = config.RampUpTime.Seconds()

	return result
}

// Make HTTP request
func (suite *PerformanceBenchmarkSuite) makeRequest(endpoint string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", suite.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return nil
}

// Benchmark HRM cognitive processing
func (suite *PerformanceBenchmarkSuite) benchmarkHRMCognitiveProcessing() BenchmarkResult {
	payload := map[string]interface{}{
		"action": "process_cognitive_input",
		"input": map[string]interface{}{
			"sensory_data": suite.generateSensoryData(512),
			"context":      `{"type":"performance_test"}`,
			"task_type":    "cognitive_analysis",
		},
		"config": map[string]interface{}{
			"l_module_count":    8,
			"h_module_count":    4,
			"enable_adaptation": true,
		},
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 10,
		RequestsPerUser: 20,
		TestDuration:    60 * time.Second,
		RampUpTime:      10 * time.Second,
		ThinkTime:       100 * time.Millisecond,
		MaxLatency:      5 * time.Second,
		MinSuccessRate:  0.95,
	}

	result := suite.executePerformanceTest(
		"HRM Cognitive Processing",
		"/api/hrm/process",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Benchmark neural network operations
func (suite *PerformanceBenchmarkSuite) benchmarkNeuralNetworkOperations() BenchmarkResult {
	payload := map[string]interface{}{
		"operation": "tensor_operations_benchmark",
		"config": map[string]interface{}{
			"tensor_size":    []int{100, 100},
			"operations":     []string{"add", "multiply", "matmul"},
			"iterations":     50,
			"measure_memory": true,
		},
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 5,
		RequestsPerUser: 10,
		TestDuration:    45 * time.Second,
		RampUpTime:      5 * time.Second,
		ThinkTime:       200 * time.Millisecond,
		MaxLatency:      3 * time.Second,
		MinSuccessRate:  0.90,
	}

	result := suite.executePerformanceTest(
		"Neural Network Operations",
		"/api/neural/benchmark",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Benchmark LoRA adapter training
func (suite *PerformanceBenchmarkSuite) benchmarkLoRATraining() BenchmarkResult {
	payload := map[string]interface{}{
		"operation": "lora_training_benchmark",
		"config": map[string]interface{}{
			"rank":          16,
			"alpha":         32,
			"dropout":       0.1,
			"learning_rate": 0.001,
			"batch_size":    16,
			"epochs":        3,
			"input_dim":     256,
			"output_dim":    256,
		},
		"test_data": suite.generateTrainingData(100, 256, 256),
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 3,
		RequestsPerUser: 5,
		TestDuration:    90 * time.Second,
		RampUpTime:      10 * time.Second,
		ThinkTime:       500 * time.Millisecond,
		MaxLatency:      15 * time.Second,
		MinSuccessRate:  0.80,
	}

	result := suite.executePerformanceTest(
		"LoRA Adapter Training",
		"/api/neural/lora/benchmark",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Benchmark visual processing
func (suite *PerformanceBenchmarkSuite) benchmarkVisualProcessing() BenchmarkResult {
	payload := map[string]interface{}{
		"operation": "visual_processing_benchmark",
		"config": map[string]interface{}{
			"image_size":          []int{224, 224, 3},
			"object_detection":    true,
			"face_recognition":    true,
			"scene_analysis":      true,
			"enable_hrm_guidance": true,
		},
		"test_data": suite.generateImageData(224, 224, 3),
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 8,
		RequestsPerUser: 15,
		TestDuration:    60 * time.Second,
		RampUpTime:      8 * time.Second,
		ThinkTime:       300 * time.Millisecond,
		MaxLatency:      8 * time.Second,
		MinSuccessRate:  0.85,
	}

	result := suite.executePerformanceTest(
		"Visual Processing",
		"/api/visual/benchmark",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Benchmark ecosystem communication
func (suite *PerformanceBenchmarkSuite) benchmarkEcosystemCommunication() BenchmarkResult {
	payload := map[string]interface{}{
		"operation": "ecosystem_communication_benchmark",
		"config": map[string]interface{}{
			"message_types":     []string{"query", "command", "event"},
			"target_components": []string{"knirv-wallet", "knirv-chain", "visual-processor"},
			"message_size":      "medium",
		},
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 15,
		RequestsPerUser: 25,
		TestDuration:    45 * time.Second,
		RampUpTime:      5 * time.Second,
		ThinkTime:       50 * time.Millisecond,
		MaxLatency:      2 * time.Second,
		MinSuccessRate:  0.95,
	}

	result := suite.executePerformanceTest(
		"Ecosystem Communication",
		"/api/ecosystem/communication/benchmark",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Benchmark unified skill execution
func (suite *PerformanceBenchmarkSuite) benchmarkUnifiedSkillExecution() BenchmarkResult {
	payload := map[string]interface{}{
		"operation": "unified_skill_execution_benchmark",
		"config": map[string]interface{}{
			"skill_id":   "benchmark_skill_001",
			"skill_type": "cognitive_processing",
			"nrn_cost":   "1.0",
			"parameters": map[string]interface{}{
				"input":      "Performance benchmark test input",
				"complexity": "medium",
			},
			"enable_wallet_integration": true,
			"enable_chain_integration":  true,
		},
	}

	config := PerformanceTestConfig{
		ConcurrentUsers: 5,
		RequestsPerUser: 10,
		TestDuration:    120 * time.Second,
		RampUpTime:      15 * time.Second,
		ThinkTime:       1 * time.Second,
		MaxLatency:      20 * time.Second,
		MinSuccessRate:  0.80,
	}

	result := suite.executePerformanceTest(
		"Unified Skill Execution",
		"/api/ecosystem/unified-skill/benchmark",
		payload,
		config,
	)

	suite.addResult(result)
	return result
}

// Helper functions for generating test data
func (suite *PerformanceBenchmarkSuite) generateSensoryData(size int) []float64 {
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = float64((i*7)%256) / 255.0
	}
	return data
}

func (suite *PerformanceBenchmarkSuite) generateTrainingData(samples, inputDim, outputDim int) map[string]interface{} {
	inputs := make([][]float64, samples)
	outputs := make([][]float64, samples)

	for i := 0; i < samples; i++ {
		inputs[i] = make([]float64, inputDim)
		outputs[i] = make([]float64, outputDim)

		for j := 0; j < inputDim; j++ {
			inputs[i][j] = float64((i*j)%256) / 255.0
		}

		for j := 0; j < outputDim; j++ {
			outputs[i][j] = float64((i + j) % 2)
		}
	}

	return map[string]interface{}{
		"inputs":  inputs,
		"outputs": outputs,
		"samples": samples,
	}
}

func (suite *PerformanceBenchmarkSuite) generateImageData(width, height, channels int) [][]float64 {
	data := make([][]float64, height)
	for i := 0; i < height; i++ {
		data[i] = make([]float64, width*channels)
		for j := 0; j < width*channels; j++ {
			data[i][j] = float64((i*width+j)%256) / 255.0
		}
	}
	return data
}

// Generate performance report
func (suite *PerformanceBenchmarkSuite) generatePerformanceReport() map[string]interface{} {
	report := map[string]interface{}{
		"benchmark_suite":  "KNIRV-CORTEX Performance Benchmarks",
		"total_benchmarks": len(suite.results),
		"execution_time":   time.Now(),
		"benchmarks":       suite.results,
		"summary":          make(map[string]interface{}),
	}

	// Calculate summary statistics
	totalRequests := 0
	totalSuccessful := 0
	var totalLatency time.Duration
	var maxRPS float64

	for _, result := range suite.results {
		totalRequests += result.TotalRequests
		totalSuccessful += result.SuccessfulRequests
		totalLatency += result.AverageLatency

		if result.RequestsPerSecond > maxRPS {
			maxRPS = result.RequestsPerSecond
		}
	}

	summary := map[string]interface{}{
		"total_requests":       totalRequests,
		"total_successful":     totalSuccessful,
		"overall_success_rate": float64(totalSuccessful) / float64(totalRequests),
		"average_latency":      totalLatency / time.Duration(len(suite.results)),
		"max_rps":              maxRPS,
	}

	report["summary"] = summary
	return report
}

// Get benchmark results
func (suite *PerformanceBenchmarkSuite) getResults() []BenchmarkResult {
	return suite.results
}

// Run all performance benchmarks
func TestPerformanceBenchmarks(t *testing.T) {
	baseURL := "http://localhost:3001"
	suite := NewPerformanceBenchmarkSuite(baseURL)

	fmt.Println("Starting KNIRV-CORTEX Performance Benchmarks...")

	benchmarks := []struct {
		name string
		fn   func() BenchmarkResult
	}{
		{"HRM Cognitive Processing", suite.benchmarkHRMCognitiveProcessing},
		{"Neural Network Operations", suite.benchmarkNeuralNetworkOperations},
		{"LoRA Adapter Training", suite.benchmarkLoRATraining},
		{"Visual Processing", suite.benchmarkVisualProcessing},
		{"Ecosystem Communication", suite.benchmarkEcosystemCommunication},
		{"Unified Skill Execution", suite.benchmarkUnifiedSkillExecution},
	}

	for _, benchmark := range benchmarks {
		fmt.Printf("Running benchmark: %s...\n", benchmark.name)
		result := benchmark.fn()

		fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
		fmt.Printf("  Success Rate: %.2f%%\n", result.SuccessRate*100)
		fmt.Printf("  Average Latency: %v\n", result.AverageLatency)
		fmt.Printf("  Requests/Second: %.2f\n", result.RequestsPerSecond)
		fmt.Printf("  P95 Latency: %v\n", result.Percentiles["p95"])
		fmt.Println()
	}

	// Generate performance report
	report := suite.generatePerformanceReport()

	// Print summary
	summary := report["summary"].(map[string]interface{})
	fmt.Printf("Performance Benchmarks Summary:\n")
	fmt.Printf("Total Benchmarks: %d\n", len(suite.results))
	fmt.Printf("Total Requests: %d\n", summary["total_requests"])
	fmt.Printf("Overall Success Rate: %.2f%%\n", summary["overall_success_rate"].(float64)*100)
	fmt.Printf("Average Latency: %v\n", summary["average_latency"])
	fmt.Printf("Max RPS: %.2f\n", summary["max_rps"])

	// Check if performance meets requirements
	overallSuccessRate := summary["overall_success_rate"].(float64)
	if overallSuccessRate < 0.85 {
		t.Errorf("Overall success rate too low: %.2f%% < 85%%", overallSuccessRate*100)
	}

	maxRPS := summary["max_rps"].(float64)
	if maxRPS < 10.0 {
		t.Errorf("Maximum RPS too low: %.2f < 10.0", maxRPS)
	}
}
