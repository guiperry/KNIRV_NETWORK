package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestResult represents the result of a single HRM cognitive test
type TestResult struct {
	TestName  string                 `json:"test_name"`
	Status    string                 `json:"status"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// HRM Cognitive Test Suite
type HRMCognitiveTestSuite struct {
	baseURL    string
	httpClient *http.Client
	results    []TestResult
}

// HRM test input structures
type HRMTestInput struct {
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
	Config      HRMConfig   `json:"config"`
	ExpectedMin float64     `json:"expected_min_confidence"`
}

type HRMConfig struct {
	LModuleCount      int  `json:"l_module_count"`
	HModuleCount      int  `json:"h_module_count"`
	EnableAdaptation  bool `json:"enable_adaptation"`
	ProcessingTimeout int  `json:"processing_timeout"`
}

type HRMResponse struct {
	Success           bool                   `json:"success"`
	Confidence        float64                `json:"confidence"`
	LModuleOutputs    []float64              `json:"l_module_outputs"`
	HModuleOutputs    []float64              `json:"h_module_outputs"`
	ProcessingTime    int                    `json:"processing_time"`
	AdaptationApplied bool                   `json:"adaptation_applied"`
	Metadata          map[string]interface{} `json:"metadata"`
	Error             string                 `json:"error,omitempty"`
}

// Create new HRM test suite
func NewHRMCognitiveTestSuite(baseURL string) *HRMCognitiveTestSuite {
	return &HRMCognitiveTestSuite{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		results: make([]TestResult, 0),
	}
}

// Add test result
func (suite *HRMCognitiveTestSuite) addResult(testName, status string, duration time.Duration, err error, metrics map[string]interface{}) {
	result := TestResult{
		TestName:  testName,
		Status:    status,
		Duration:  duration,
		Metrics:   metrics,
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	suite.results = append(suite.results, result)
}

// Make HTTP request to HRM endpoint
func (suite *HRMCognitiveTestSuite) makeHRMRequest(endpoint string, payload interface{}) (*HRMResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", suite.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var hrmResp HRMResponse
	if err := json.NewDecoder(resp.Body).Decode(&hrmResp); err != nil {
		return nil, err
	}

	return &hrmResp, nil
}

// Test HRM WASM module initialization
func (suite *HRMCognitiveTestSuite) testHRMInitialization() error {
	start := time.Now()
	testName := "hrm_wasm_initialization"

	payload := map[string]interface{}{
		"action": "initialize",
		"config": HRMConfig{
			LModuleCount:      8,
			HModuleCount:      4,
			EnableAdaptation:  true,
			ProcessingTimeout: 30000,
		},
	}

	resp, err := suite.makeHRMRequest("/api/hrm/initialize", payload)
	if err != nil {
		suite.addResult(testName, "FAILED", time.Since(start), err, nil)
		return err
	}

	if !resp.Success {
		err = fmt.Errorf("HRM initialization failed: %s", resp.Error)
		suite.addResult(testName, "FAILED", time.Since(start), err, nil)
		return err
	}

	metrics := map[string]interface{}{
		"processing_time":    resp.ProcessingTime,
		"l_module_count":     len(resp.LModuleOutputs),
		"h_module_count":     len(resp.HModuleOutputs),
		"adaptation_applied": resp.AdaptationApplied,
	}

	suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	return nil
}

// Test HRM text processing
func (suite *HRMCognitiveTestSuite) testHRMTextProcessing() error {
	testCases := []HRMTestInput{
		{
			Type: "text",
			Data: "Analyze this complex problem and provide a cognitive solution",
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.5,
		},
		{
			Type: "text",
			Data: "Process this multi-modal input with reasoning and analysis",
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.4,
		},
		{
			Type: "text",
			Data: "Perform temporal reasoning about causal relationships",
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.3,
		},
	}

	for i, testCase := range testCases {
		start := time.Now()
		testName := fmt.Sprintf("hrm_text_processing_%d", i)

		payload := map[string]interface{}{
			"action": "process_cognitive_input",
			"input": map[string]interface{}{
				"sensory_data": suite.textToSensoryData(testCase.Data.(string)),
				"context":      fmt.Sprintf(`{"type":"text","content":"%s"}`, testCase.Data),
				"task_type":    "text_analysis",
			},
			"config": testCase.Config,
		}

		resp, err := suite.makeHRMRequest("/api/hrm/process", payload)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("HRM processing failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if resp.Confidence < testCase.ExpectedMin {
			err = fmt.Errorf("confidence too low: %f < %f", resp.Confidence, testCase.ExpectedMin)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"confidence":         resp.Confidence,
			"processing_time":    resp.ProcessingTime,
			"l_module_outputs":   len(resp.LModuleOutputs),
			"h_module_outputs":   len(resp.HModuleOutputs),
			"adaptation_applied": resp.AdaptationApplied,
			"input_length":       len(testCase.Data.(string)),
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test HRM sensory data processing
func (suite *HRMCognitiveTestSuite) testHRMSensoryProcessing() error {
	testCases := []HRMTestInput{
		{
			Type: "sensory",
			Data: suite.generateSensoryData(512, "random"),
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.3,
		},
		{
			Type: "sensory",
			Data: suite.generateSensoryData(512, "pattern"),
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.4,
		},
		{
			Type: "sensory",
			Data: suite.generateSensoryData(512, "structured"),
			Config: HRMConfig{
				LModuleCount:      8,
				HModuleCount:      4,
				EnableAdaptation:  true,
				ProcessingTimeout: 30000,
			},
			ExpectedMin: 0.5,
		},
	}

	for i, testCase := range testCases {
		start := time.Now()
		testName := fmt.Sprintf("hrm_sensory_processing_%d", i)

		payload := map[string]interface{}{
			"action": "process_cognitive_input",
			"input": map[string]interface{}{
				"sensory_data": testCase.Data,
				"context":      `{"type":"sensory","modality":"multi"}`,
				"task_type":    "sensory_analysis",
			},
			"config": testCase.Config,
		}

		resp, err := suite.makeHRMRequest("/api/hrm/process", payload)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("HRM sensory processing failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if resp.Confidence < testCase.ExpectedMin {
			err = fmt.Errorf("confidence too low: %f < %f", resp.Confidence, testCase.ExpectedMin)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"confidence":         resp.Confidence,
			"processing_time":    resp.ProcessingTime,
			"l_module_outputs":   len(resp.LModuleOutputs),
			"h_module_outputs":   len(resp.HModuleOutputs),
			"adaptation_applied": resp.AdaptationApplied,
			"sensory_data_size":  len(testCase.Data.([]float64)),
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test HRM module configuration
func (suite *HRMCognitiveTestSuite) testHRMModuleConfiguration() error {
	configurations := []HRMConfig{
		{LModuleCount: 4, HModuleCount: 2, EnableAdaptation: true, ProcessingTimeout: 30000},
		{LModuleCount: 8, HModuleCount: 4, EnableAdaptation: true, ProcessingTimeout: 30000},
		{LModuleCount: 16, HModuleCount: 8, EnableAdaptation: true, ProcessingTimeout: 30000},
		{LModuleCount: 8, HModuleCount: 4, EnableAdaptation: false, ProcessingTimeout: 30000},
	}

	for i, config := range configurations {
		start := time.Now()
		testName := fmt.Sprintf("hrm_module_config_%d", i)

		payload := map[string]interface{}{
			"action": "process_cognitive_input",
			"input": map[string]interface{}{
				"sensory_data": suite.generateSensoryData(512, "test"),
				"context":      `{"type":"configuration_test"}`,
				"task_type":    "module_test",
			},
			"config": config,
		}

		resp, err := suite.makeHRMRequest("/api/hrm/process", payload)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("HRM module configuration test failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		// Validate module outputs match configuration
		expectedLModules := config.LModuleCount
		expectedHModules := config.HModuleCount

		if len(resp.LModuleOutputs) != expectedLModules {
			err = fmt.Errorf("L-module count mismatch: expected %d, got %d", expectedLModules, len(resp.LModuleOutputs))
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if len(resp.HModuleOutputs) != expectedHModules {
			err = fmt.Errorf("H-module count mismatch: expected %d, got %d", expectedHModules, len(resp.HModuleOutputs))
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"confidence":         resp.Confidence,
			"processing_time":    resp.ProcessingTime,
			"l_module_count":     len(resp.LModuleOutputs),
			"h_module_count":     len(resp.HModuleOutputs),
			"adaptation_applied": resp.AdaptationApplied,
			"config":             config,
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test HRM performance under load
func (suite *HRMCognitiveTestSuite) testHRMPerformance() error {
	start := time.Now()
	testName := "hrm_performance_load_test"

	concurrentRequests := 5
	requestsPerWorker := 10

	type performanceResult struct {
		success        bool
		processingTime int
		confidence     float64
		error          error
	}

	resultsChan := make(chan performanceResult, concurrentRequests*requestsPerWorker)

	// Launch concurrent workers
	for worker := 0; worker < concurrentRequests; worker++ {
		go func(workerID int) {
			for req := 0; req < requestsPerWorker; req++ {
				payload := map[string]interface{}{
					"action": "process_cognitive_input",
					"input": map[string]interface{}{
						"sensory_data": suite.generateSensoryData(512, "performance"),
						"context":      fmt.Sprintf(`{"type":"performance_test","worker":%d,"request":%d}`, workerID, req),
						"task_type":    "performance_analysis",
					},
					"config": HRMConfig{
						LModuleCount:      8,
						HModuleCount:      4,
						EnableAdaptation:  true,
						ProcessingTimeout: 30000,
					},
				}

				resp, err := suite.makeHRMRequest("/api/hrm/process", payload)

				result := performanceResult{
					error: err,
				}

				if err == nil && resp.Success {
					result.success = true
					result.processingTime = resp.ProcessingTime
					result.confidence = resp.Confidence
				}

				resultsChan <- result
			}
		}(worker)
	}

	// Collect results
	totalRequests := concurrentRequests * requestsPerWorker
	successCount := 0
	totalProcessingTime := 0
	totalConfidence := 0.0

	for i := 0; i < totalRequests; i++ {
		result := <-resultsChan
		if result.success {
			successCount++
			totalProcessingTime += result.processingTime
			totalConfidence += result.confidence
		}
	}

	successRate := float64(successCount) / float64(totalRequests)
	avgProcessingTime := 0
	avgConfidence := 0.0

	if successCount > 0 {
		avgProcessingTime = totalProcessingTime / successCount
		avgConfidence = totalConfidence / float64(successCount)
	}

	metrics := map[string]interface{}{
		"total_requests":      totalRequests,
		"successful_requests": successCount,
		"success_rate":        successRate,
		"avg_processing_time": avgProcessingTime,
		"avg_confidence":      avgConfidence,
		"concurrent_workers":  concurrentRequests,
		"requests_per_worker": requestsPerWorker,
	}

	if successRate < 0.8 {
		err := fmt.Errorf("success rate too low: %f", successRate)
		suite.addResult(testName, "FAILED", time.Since(start), err, metrics)
		return err
	}

	suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	return nil
}

// Helper function to convert text to sensory data
func (suite *HRMCognitiveTestSuite) textToSensoryData(text string) []float64 {
	data := make([]float64, 512)

	// Simple text encoding to sensory data
	for i, char := range text {
		if i >= 512 {
			break
		}
		data[i] = float64(char) / 255.0
	}

	return data
}

// Helper function to generate sensory data
func (suite *HRMCognitiveTestSuite) generateSensoryData(size int, pattern string) []float64 {
	data := make([]float64, size)

	switch pattern {
	case "random":
		for i := 0; i < size; i++ {
			data[i] = float64(i%256) / 255.0
		}
	case "pattern":
		for i := 0; i < size; i++ {
			data[i] = float64((i*7)%256) / 255.0
		}
	case "structured":
		for i := 0; i < size; i++ {
			data[i] = float64((i*i)%256) / 255.0
		}
	default:
		for i := 0; i < size; i++ {
			data[i] = 0.5
		}
	}

	return data
}

// Get test results
func (suite *HRMCognitiveTestSuite) getResults() []TestResult {
	return suite.results
}

// Run all HRM cognitive tests
func TestHRMCognitiveFunctionality(t *testing.T) {
	baseURL := "http://localhost:3001"
	suite := NewHRMCognitiveTestSuite(baseURL)

	fmt.Println("Starting HRM Cognitive Core Tests...")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"HRM WASM Initialization", suite.testHRMInitialization},
		{"HRM Text Processing", suite.testHRMTextProcessing},
		{"HRM Sensory Processing", suite.testHRMSensoryProcessing},
		{"HRM Module Configuration", suite.testHRMModuleConfiguration},
		{"HRM Performance Load Test", suite.testHRMPerformance},
	}

	for _, test := range tests {
		fmt.Printf("Running %s...\n", test.name)
		if err := test.fn(); err != nil {
			fmt.Printf("Test %s encountered errors: %v\n", test.name, err)
		}
	}

	// Print summary
	results := suite.getResults()
	passed := 0
	failed := 0

	for _, result := range results {
		if result.Status == "PASSED" {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\nHRM Cognitive Tests Summary:\n")
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)

	if failed > 0 {
		t.Errorf("%d HRM cognitive tests failed", failed)
	}
}
