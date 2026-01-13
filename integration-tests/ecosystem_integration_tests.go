package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// EcosystemTestResult represents the result of a single ecosystem test
type EcosystemTestResult struct {
	TestName  string                 `json:"test_name"`
	Status    string                 `json:"status"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Ecosystem Integration Test Suite
type EcosystemTestSuite struct {
	baseURL    string
	httpClient *http.Client
	results    []EcosystemTestResult
}

// Ecosystem test structures
type EcosystemTestInput struct {
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Config      map[string]interface{} `json:"config"`
	TestData    interface{}            `json:"test_data,omitempty"`
	ExpectedMin float64                `json:"expected_min_success,omitempty"`
}

type EcosystemResponse struct {
	Success        bool                   `json:"success"`
	Component      string                 `json:"component"`
	Operation      string                 `json:"operation"`
	ProcessingTime int                    `json:"processing_time"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// Create new ecosystem test suite
func NewEcosystemTestSuite(baseURL string) *EcosystemTestSuite {
	return &EcosystemTestSuite{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		results: make([]EcosystemTestResult, 0),
	}
}

// Add test result
func (suite *EcosystemTestSuite) addResult(testName, status string, duration time.Duration, err error, metrics map[string]interface{}) {
	result := EcosystemTestResult{
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

// Make HTTP request to ecosystem endpoint
func (suite *EcosystemTestSuite) makeEcosystemRequest(endpoint string, payload interface{}) (*EcosystemResponse, error) {
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

	var ecosystemResp EcosystemResponse
	if err := json.NewDecoder(resp.Body).Decode(&ecosystemResp); err != nil {
		return nil, err
	}

	return &ecosystemResp, nil
}

// Test KNIRV-WALLET integration
func (suite *EcosystemTestSuite) testWalletIntegration() error {
	walletTests := []EcosystemTestInput{
		{
			Component: "knirv-wallet",
			Operation: "initialize",
			Config: map[string]interface{}{
				"api_base_url":          "http://localhost:8083/api/v1",
				"chain_id":              "knirv-mainnet-1",
				"enable_cross_platform": true,
			},
		},
		{
			Component: "knirv-wallet",
			Operation: "get_accounts",
			Config:    map[string]interface{}{},
		},
		{
			Component: "knirv-wallet",
			Operation: "get_balance",
			Config: map[string]interface{}{
				"account_id": "test_account_1",
			},
		},
		{
			Component: "knirv-wallet",
			Operation: "create_transaction",
			Config: map[string]interface{}{
				"from":       "knirv1test1234567890abcdef",
				"to":         "knirv1test0987654321fedcba",
				"amount":     "10.0",
				"nrn_amount": "5.0",
			},
		},
		{
			Component: "knirv-wallet",
			Operation: "invoke_skill",
			Config: map[string]interface{}{
				"skill_id":   "test_skill_001",
				"skill_name": "Test Cognitive Skill",
				"nrn_cost":   "2.5",
				"parameters": map[string]interface{}{
					"input": "Test skill invocation",
				},
			},
		},
	}

	for i, test := range walletTests {
		start := time.Now()
		testName := fmt.Sprintf("wallet_%s_%d", test.Operation, i)

		resp, err := suite.makeEcosystemRequest("/api/ecosystem/wallet/test", test)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("wallet operation failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"processing_time": resp.ProcessingTime,
			"component":       resp.Component,
			"operation":       resp.Operation,
		}

		if resp.Metrics != nil {
			for k, v := range resp.Metrics {
				metrics[k] = v
			}
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test KNIRV-CHAIN integration
func (suite *EcosystemTestSuite) testChainIntegration() error {
	chainTests := []EcosystemTestInput{
		{
			Component: "knirv-chain",
			Operation: "initialize",
			Config: map[string]interface{}{
				"rpc_url":  "http://localhost:8080",
				"chain_id": "knirv-chain-1",
				"contracts": map[string]string{
					"nrn_token":      "0x1234567890123456789012345678901234567890",
					"skill_registry": "0x2345678901234567890123456789012345678901",
					"llm_registry":   "0x3456789012345678901234567890123456789012",
				},
			},
		},
		{
			Component: "knirv-chain",
			Operation: "get_network_consensus",
			Config:    map[string]interface{}{},
		},
		{
			Component: "knirv-chain",
			Operation: "verify_skill",
			Config: map[string]interface{}{
				"skill_id": "test_skill_001",
			},
		},
		{
			Component: "knirv-chain",
			Operation: "get_nrn_balance",
			Config: map[string]interface{}{
				"address": "knirv1test1234567890abcdef",
			},
		},
		{
			Component: "knirv-chain",
			Operation: "invoke_skill_on_chain",
			Config: map[string]interface{}{
				"skill_id":     "test_skill_001",
				"user_address": "knirv1test1234567890abcdef",
				"nrn_amount":   "2.5",
				"parameters": map[string]interface{}{
					"input": "Test blockchain skill invocation",
				},
			},
		},
		{
			Component: "knirv-chain",
			Operation: "register_skill",
			Config: map[string]interface{}{
				"name":         "Test AI Skill",
				"skill_type":   "cognitive_processing",
				"capabilities": []string{"reasoning", "analysis"},
				"owner":        "knirv1test1234567890abcdef",
				"usage_fee":    "1.0",
			},
		},
	}

	for i, test := range chainTests {
		start := time.Now()
		testName := fmt.Sprintf("chain_%s_%d", test.Operation, i)

		resp, err := suite.makeEcosystemRequest("/api/ecosystem/chain/test", test)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("chain operation failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"processing_time": resp.ProcessingTime,
			"component":       resp.Component,
			"operation":       resp.Operation,
		}

		if resp.Metrics != nil {
			for k, v := range resp.Metrics {
				metrics[k] = v
			}
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test Visual Processing integration
func (suite *EcosystemTestSuite) testVisualProcessingIntegration() error {
	visualTests := []EcosystemTestInput{
		{
			Component: "visual-processor",
			Operation: "initialize",
			Config: map[string]interface{}{
				"object_detection":     true,
				"face_recognition":     true,
				"scene_analysis":       true,
				"gesture_recognition":  true,
				"text_recognition":     true,
				"enable_hrm_guidance":  true,
				"confidence_threshold": 0.5,
			},
		},
		{
			Component: "visual-processor",
			Operation: "process_image",
			Config: map[string]interface{}{
				"image_type": "synthetic",
				"image_size": []int{640, 480},
				"channels":   3,
			},
			TestData: suite.generateSyntheticImageData(640, 480, 3),
		},
		{
			Component: "visual-processor",
			Operation: "object_detection",
			Config: map[string]interface{}{
				"confidence_threshold": 0.5,
				"max_objects":          10,
			},
			TestData: suite.generateSyntheticImageData(224, 224, 3),
		},
		{
			Component: "visual-processor",
			Operation: "face_recognition",
			Config: map[string]interface{}{
				"detect_emotions": true,
				"detect_age":      true,
				"detect_gender":   true,
			},
			TestData: suite.generateSyntheticImageData(224, 224, 3),
		},
		{
			Component: "visual-processor",
			Operation: "scene_analysis",
			Config: map[string]interface{}{
				"analyze_lighting": true,
				"analyze_setting":  true,
				"analyze_mood":     true,
			},
			TestData: suite.generateSyntheticImageData(512, 512, 3),
		},
	}

	for i, test := range visualTests {
		start := time.Now()
		testName := fmt.Sprintf("visual_%s_%d", test.Operation, i)

		resp, err := suite.makeEcosystemRequest("/api/ecosystem/visual/test", test)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("visual processing operation failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"processing_time": resp.ProcessingTime,
			"component":       resp.Component,
			"operation":       resp.Operation,
		}

		if resp.Metrics != nil {
			for k, v := range resp.Metrics {
				metrics[k] = v
			}
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test Ecosystem Communication Layer
func (suite *EcosystemTestSuite) testEcosystemCommunication() error {
	communicationTests := []EcosystemTestInput{
		{
			Component: "ecosystem-communication",
			Operation: "initialize",
			Config: map[string]interface{}{
				"enable_wallet_integration":  true,
				"enable_chain_integration":   true,
				"enable_nexus_integration":   true,
				"enable_gateway_integration": true,
				"communication_protocol":     "http",
				"heartbeat_interval":         30000,
			},
		},
		{
			Component: "ecosystem-communication",
			Operation: "get_components",
			Config:    map[string]interface{}{},
		},
		{
			Component: "ecosystem-communication",
			Operation: "send_message",
			Config: map[string]interface{}{
				"to":                "knirv-wallet",
				"type":              "query",
				"payload":           map[string]interface{}{"action": "get_status"},
				"priority":          "normal",
				"requires_response": true,
			},
		},
		{
			Component: "ecosystem-communication",
			Operation: "unified_skill_execution",
			Config: map[string]interface{}{
				"skill_id":   "test_skill_001",
				"parameters": map[string]interface{}{"input": "Test unified execution"},
				"nrn_amount": "2.5",
			},
		},
		{
			Component: "ecosystem-communication",
			Operation: "cross_platform_transaction",
			Config: map[string]interface{}{
				"from":       "knirv1test1234567890abcdef",
				"to":         "knirv1test0987654321fedcba",
				"amount":     "5.0",
				"nrn_amount": "2.0",
			},
		},
	}

	for i, test := range communicationTests {
		start := time.Now()
		testName := fmt.Sprintf("communication_%s_%d", test.Operation, i)

		resp, err := suite.makeEcosystemRequest("/api/ecosystem/communication/test", test)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("ecosystem communication operation failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"processing_time": resp.ProcessingTime,
			"component":       resp.Component,
			"operation":       resp.Operation,
		}

		if resp.Metrics != nil {
			for k, v := range resp.Metrics {
				metrics[k] = v
			}
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Test cross-component integration scenarios
func (suite *EcosystemTestSuite) testCrossComponentIntegration() error {
	crossTests := []EcosystemTestInput{
		{
			Component: "cross-component",
			Operation: "skill_with_payment",
			Config: map[string]interface{}{
				"skill_id":   "test_skill_001",
				"account_id": "test_account_1",
				"nrn_cost":   "2.5",
				"parameters": map[string]interface{}{
					"input": "Test cross-component skill execution with payment",
				},
			},
		},
		{
			Component: "cross-component",
			Operation: "visual_cognitive_processing",
			Config: map[string]interface{}{
				"enable_hrm_enhancement": true,
				"image_size":             []int{224, 224},
				"cognitive_analysis":     true,
			},
			TestData: suite.generateSyntheticImageData(224, 224, 3),
		},
		{
			Component: "cross-component",
			Operation: "adaptive_learning_with_feedback",
			Config: map[string]interface{}{
				"interaction_type": "multi_modal",
				"feedback_score":   4.5,
				"hrm_guidance":     true,
			},
			TestData: map[string]interface{}{
				"text_input":   "Test adaptive learning",
				"visual_input": suite.generateSyntheticImageData(64, 64, 3),
			},
		},
		{
			Component: "cross-component",
			Operation: "ecosystem_health_check",
			Config: map[string]interface{}{
				"check_all_components": true,
				"include_performance":  true,
			},
		},
	}

	for i, test := range crossTests {
		start := time.Now()
		testName := fmt.Sprintf("cross_component_%s_%d", test.Operation, i)

		resp, err := suite.makeEcosystemRequest("/api/ecosystem/cross-component/test", test)
		if err != nil {
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		if !resp.Success {
			err = fmt.Errorf("cross-component operation failed: %s", resp.Error)
			suite.addResult(testName, "FAILED", time.Since(start), err, nil)
			continue
		}

		metrics := map[string]interface{}{
			"processing_time": resp.ProcessingTime,
			"component":       resp.Component,
			"operation":       resp.Operation,
		}

		if resp.Metrics != nil {
			for k, v := range resp.Metrics {
				metrics[k] = v
			}
		}

		suite.addResult(testName, "PASSED", time.Since(start), nil, metrics)
	}

	return nil
}

// Helper function to generate synthetic image data
func (suite *EcosystemTestSuite) generateSyntheticImageData(width, height, channels int) [][]float64 {
	data := make([][]float64, height)
	for i := 0; i < height; i++ {
		data[i] = make([]float64, width*channels)
		for j := 0; j < width*channels; j++ {
			// Generate synthetic image pattern
			data[i][j] = float64((i*width+j)%256) / 255.0
		}
	}
	return data
}

// Get test results
func (suite *EcosystemTestSuite) getResults() []EcosystemTestResult {
	return suite.results
}

// Run all ecosystem integration tests
func TestEcosystemIntegrations(t *testing.T) {
	baseURL := "http://localhost:3001"
	suite := NewEcosystemTestSuite(baseURL)

	fmt.Println("Starting Ecosystem Integration Tests...")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"KNIRV-WALLET Integration", suite.testWalletIntegration},
		{"KNIRV-CHAIN Integration", suite.testChainIntegration},
		{"Visual Processing Integration", suite.testVisualProcessingIntegration},
		{"Ecosystem Communication", suite.testEcosystemCommunication},
		{"Cross-Component Integration", suite.testCrossComponentIntegration},
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

	fmt.Printf("\nEcosystem Integration Tests Summary:\n")
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)

	if failed > 0 {
		t.Errorf("%d ecosystem integration tests failed", failed)
	}
}
