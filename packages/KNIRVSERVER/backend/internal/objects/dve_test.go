package objects

import (
	"testing"
	"time"
)

func TestDVENode_StructFields(t *testing.T) {
	node := DVENode{
		ID:              "node-123",
		Name:            "Test Node",
		Status:          "online",
		TEEType:         "sgx",
		StakeAmount:     1000000,
		ReputationScore: 95,
		Capabilities:    []string{"validation", "inference"},
		CPUUsage:        75.5,
		Latitude:        40.7128,
	}

	if node.Name != "Test Node" {
		t.Errorf("Expected Name 'Test Node', got '%s'", node.Name)
	}

	if node.ID != "node-123" {
		t.Errorf("Expected ID 'node-123', got '%s'", node.ID)
	}
	if node.Status != "online" {
		t.Errorf("Expected Status 'online', got '%s'", node.Status)
	}
	if node.TEEType != "sgx" {
		t.Errorf("Expected TEEType 'sgx', got '%s'", node.TEEType)
	}
	if node.StakeAmount != 1000000 {
		t.Errorf("Expected StakeAmount 1000000, got %d", node.StakeAmount)
	}
	if node.ReputationScore != 95 {
		t.Errorf("Expected ReputationScore 95, got %d", node.ReputationScore)
	}
	if len(node.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(node.Capabilities))
	}
	if node.CPUUsage != 75.5 {
		t.Errorf("Expected CPUUsage 75.5, got %f", node.CPUUsage)
	}
	if node.Latitude != 40.7128 {
		t.Errorf("Expected Latitude 40.7128, got %f", node.Latitude)
	}
}

func TestBrowserDVENode_StructFields(t *testing.T) {
	node := DVENode{
		ID:              "browser-node-001",
		Name:            "Browser DVE Node",
		Status:          "online",
		TEEType:         "browser-extension",
		ExtensionID:     "chrome-extension-abc123",
		BrowserVersion:  "1.0.0",
		WSConnectionID:  "ws-conn-001",
		BadgeNFTIDs:     []string{"badge-nft-1", "badge-nft-2"},
		IsRemote:        true,
		Connected:       true,
		Capabilities:    []string{"validation", "light-attestation", "dve-identity"},
		StakeAmount:     10000,
		Location:        "global",
	}

	if node.TEEType != "browser-extension" {
		t.Errorf("Expected TEEType 'browser-extension', got '%s'", node.TEEType)
	}
	if node.ExtensionID != "chrome-extension-abc123" {
		t.Errorf("Expected ExtensionID 'chrome-extension-abc123', got '%s'", node.ExtensionID)
	}
	if node.BrowserVersion != "1.0.0" {
		t.Errorf("Expected BrowserVersion '1.0.0', got '%s'", node.BrowserVersion)
	}
	if node.WSConnectionID != "ws-conn-001" {
		t.Errorf("Expected WSConnectionID 'ws-conn-001', got '%s'", node.WSConnectionID)
	}
	if len(node.BadgeNFTIDs) != 2 {
		t.Errorf("Expected 2 badge NFT IDs, got %d", len(node.BadgeNFTIDs))
	}
	if node.BadgeNFTIDs[0] != "badge-nft-1" {
		t.Errorf("Expected first badge NFT ID 'badge-nft-1', got '%s'", node.BadgeNFTIDs[0])
	}
	if !node.IsRemote {
		t.Error("Expected IsRemote to be true for browser-extension nodes")
	}
	if !node.Connected {
		t.Error("Expected Connected to be true for browser-extension nodes")
	}
}

func TestRegisterNodeRequest_BrowserDVEFields(t *testing.T) {
	req := RegisterNodeRequest{
		Name:           "Test Browser DVE",
		TEEType:        "browser-extension",
		StakeAmount:    5000,
		Location:       "browser",
		Capabilities:   []string{"policy-check", "signature-verify"},
		ExtensionID:    "ext-abc-123",
		BrowserVersion: "2.1.0",
		WalletAddress:  "0x1234567890abcdef",
		BadgeNFTIDs:    []string{"badge-nft-001", "badge-nft-002"},
	}

	if req.TEEType != "browser-extension" {
		t.Errorf("Expected TEEType 'browser-extension', got '%s'", req.TEEType)
	}
	if req.ExtensionID != "ext-abc-123" {
		t.Errorf("Expected ExtensionID 'ext-abc-123', got '%s'", req.ExtensionID)
	}
	if req.BrowserVersion != "2.1.0" {
		t.Errorf("Expected BrowserVersion '2.1.0', got '%s'", req.BrowserVersion)
	}
	if req.WalletAddress != "0x1234567890abcdef" {
		t.Errorf("Expected WalletAddress '0x1234567890abcdef', got '%s'", req.WalletAddress)
	}
	if len(req.BadgeNFTIDs) != 2 {
		t.Errorf("Expected 2 badge NFT IDs, got %d", len(req.BadgeNFTIDs))
	}
	if req.BadgeNFTIDs[0] != "badge-nft-001" {
		t.Errorf("Expected first badge NFT ID 'badge-nft-001', got '%s'", req.BadgeNFTIDs[0])
	}
}

func TestValidationTask_StructFields(t *testing.T) {
	startedAt := time.Now().Add(time.Minute)
	completedAt := time.Now().Add(time.Hour)

	parameters := map[string]interface{}{
		"model":      "test-model",
		"batch_size": 32,
	}

	testCases := []TestCase{
		{
			ID:          "test-1",
			Name:        "Basic Test",
			Description: "Test basic functionality",
			Input:       "test input",
			Expected:    "expected output",
			Weight:      1.0,
		},
	}

	if testCases[0].Description != "Test basic functionality" {
		t.Errorf("Expected Description 'Test basic functionality', got '%s'", testCases[0].Description)
	}
	if testCases[0].Name != "Basic Test" {
		t.Errorf("Expected Name 'Basic Test', got '%s'", testCases[0].Name)
	}

	task := ValidationTask{
		ID:              "task-123",
		Type:            "skillnode",
		Status:          "completed",
		Priority:        5,
		TestCases:       testCases,
		RequiredTEEType: "sgx",
		Parameters:      parameters,
		StartedAt:       &startedAt,
		CompletedAt:     &completedAt,
	}

	if task.ID != "task-123" {
		t.Errorf("Expected ID 'task-123', got '%s'", task.ID)
	}
	if task.Type != "skillnode" {
		t.Errorf("Expected Type 'skillnode', got '%s'", task.Type)
	}
	if task.Status != "completed" {
		t.Errorf("Expected Status 'completed', got '%s'", task.Status)
	}
	if task.Priority != 5 {
		t.Errorf("Expected Priority 5, got %d", task.Priority)
	}
	if task.RequiredTEEType != "sgx" {
		t.Errorf("Expected RequiredTEEType 'sgx', got '%s'", task.RequiredTEEType)
	}
	if len(task.TestCases) != 1 {
		t.Errorf("Expected 1 test case, got %d", len(task.TestCases))
	}
	if task.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if task.Parameters["model"] != "test-model" {
		t.Errorf("Expected parameter model 'test-model', got '%v'", task.Parameters["model"])
	}
}

func TestTestCase_StructFields(t *testing.T) {
	input := "test prompt with temperature 0.7"
	expected := "expected response with confidence 0.95"

	testCase := TestCase{
		ID:          "test-case-1",
		Name:        "Temperature Test",
		Description: "Test with specific temperature",
		Input:       input,
		Expected:    expected,
		Weight:      0.8,
	}

	if testCase.ID != "test-case-1" {
		t.Errorf("Expected ID 'test-case-1', got '%s'", testCase.ID)
	}
	if testCase.Name != "Temperature Test" {
		t.Errorf("Expected Name 'Temperature Test', got '%s'", testCase.Name)
	}
	if testCase.Weight != 0.8 {
		t.Errorf("Expected Weight 0.8, got %f", testCase.Weight)
	}
	if testCase.Input != "test prompt with temperature 0.7" {
		t.Errorf("Expected input 'test prompt with temperature 0.7', got '%s'", testCase.Input)
	}
	if testCase.Expected != "expected response with confidence 0.95" {
		t.Errorf("Expected response 'expected response with confidence 0.95', got '%s'", testCase.Expected)
	}
	if testCase.Description != "Test with specific temperature" {
		t.Errorf("Expected Description 'Test with specific temperature', got '%s'", testCase.Description)
	}
}

func TestValidationResult_StructFields(t *testing.T) {
	results := map[string]interface{}{
		"accuracy": 0.95,
		"latency":  150,
	}

	testResults := []TestResult{
		{
			TestCaseID:    "test-1",
			Status:        "passed",
			ActualOutput:  "actual output",
			Score:         0.9,
			ErrorMessage:  "",
			ExecutionTime: time.Millisecond * 100,
		},
	}

	if testResults[0].ErrorMessage != "" {
		t.Errorf("Expected ErrorMessage '', got '%s'", testResults[0].ErrorMessage)
	}

	result := ValidationResult{
		ID:            "result-123",
		TaskID:        "task-123",
		Status:        "success",
		Score:         0.95,
		Results:       results,
		TestResults:   testResults,
		ExecutionTime: time.Second * 5,
	}

	if result.ID != "result-123" {
		t.Errorf("Expected ID 'result-123', got '%s'", result.ID)
	}
	if result.TaskID != "task-123" {
		t.Errorf("Expected TaskID 'task-123', got '%s'", result.TaskID)
	}
	if result.Status != "success" {
		t.Errorf("Expected Status 'success', got '%s'", result.Status)
	}
	if result.Score != 0.95 {
		t.Errorf("Expected Score 0.95, got %f", result.Score)
	}
	if len(result.TestResults) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(result.TestResults))
	}
	if result.ExecutionTime != time.Second*5 {
		t.Errorf("Expected ExecutionTime 5s, got %v", result.ExecutionTime)
	}
	if result.Results["accuracy"] != 0.95 {
		t.Errorf("Expected accuracy 0.95, got '%v'", result.Results["accuracy"])
	}
}

func TestTestResult_StructFields(t *testing.T) {
	actualOutput := "positive prediction with confidence 0.87"

	testResult := TestResult{
		TestCaseID:    "test-case-1",
		Status:        "passed",
		ActualOutput:  actualOutput,
		Score:         0.87,
		ErrorMessage:  "",
		ExecutionTime: time.Millisecond * 250,
	}

	if testResult.TestCaseID != "test-case-1" {
		t.Errorf("Expected TestCaseID 'test-case-1', got '%s'", testResult.TestCaseID)
	}
	if testResult.Status != "passed" {
		t.Errorf("Expected Status 'passed', got '%s'", testResult.Status)
	}
	if testResult.Score != 0.87 {
		t.Errorf("Expected Score 0.87, got %f", testResult.Score)
	}
	if testResult.ExecutionTime != time.Millisecond*250 {
		t.Errorf("Expected ExecutionTime 250ms, got %v", testResult.ExecutionTime)
	}
	if testResult.ActualOutput != "positive prediction with confidence 0.87" {
		t.Errorf("Expected output 'positive prediction with confidence 0.87', got '%s'", testResult.ActualOutput)
	}
	if testResult.ErrorMessage != "" {
		t.Errorf("Expected ErrorMessage '', got '%s'", testResult.ErrorMessage)
	}
}

func TestTEEAttestation_StructFields(t *testing.T) {
	verifiedAt := time.Now().Add(time.Minute)

	attestation := TEEAttestation{
		ID:           "attestation-123",
		NodeID:       "node-123",
		TEEType:      "sgx",
		Status:       "valid",
		Measurements: []string{"measurement1", "measurement2"},
		VerifiedAt:   &verifiedAt,
	}

	if attestation.ID != "attestation-123" {
		t.Errorf("Expected ID 'attestation-123', got '%s'", attestation.ID)
	}
	if attestation.NodeID != "node-123" {
		t.Errorf("Expected NodeID 'node-123', got '%s'", attestation.NodeID)
	}
	if attestation.TEEType != "sgx" {
		t.Errorf("Expected TEEType 'sgx', got '%s'", attestation.TEEType)
	}
	if attestation.Status != "valid" {
		t.Errorf("Expected Status 'valid', got '%s'", attestation.Status)
	}
	if len(attestation.Measurements) != 2 {
		t.Errorf("Expected 2 measurements, got %d", len(attestation.Measurements))
	}
	if attestation.VerifiedAt == nil {
		t.Error("Expected VerifiedAt to be set")
	}
}

func TestCognitiveEngineMetrics_StructFields(t *testing.T) {

	metrics := CognitiveEngineMetrics{
		ID:                "metrics-123",
		NodeID:            "node-123",
		TasksProcessed:    1000,
		SuccessRate:       0.95,
		AdaptationScore:   0.87,
	}

	if metrics.ID != "metrics-123" {
		t.Errorf("Expected ID 'metrics-123', got '%s'", metrics.ID)
	}
	if metrics.NodeID != "node-123" {
		t.Errorf("Expected NodeID 'node-123', got '%s'", metrics.NodeID)
	}
	if metrics.TasksProcessed != 1000 {
		t.Errorf("Expected TasksProcessed 1000, got %d", metrics.TasksProcessed)
	}
	if metrics.SuccessRate != 0.95 {
		t.Errorf("Expected SuccessRate 0.95, got %f", metrics.SuccessRate)
	}
	if metrics.AdaptationScore != 0.87 {
		t.Errorf("Expected AdaptationScore 0.87, got %f", metrics.AdaptationScore)
	}
}

func TestSystemHealth_StructFields(t *testing.T) {
	now := time.Now()

	components := map[string]*ComponentHealth{
		"database": {
			Status:  "healthy",
			Message: "All connections active",
			Metrics: map[string]interface{}{"connections": 10},
		},
	}

	alerts := []*SystemAlert{
		{
			ID:        "alert-1",
			Severity:  "warning",
			Component: "cpu",
			Message:   "High CPU usage",
			Timestamp: now.Format(time.RFC3339),
			Resolved:  false,
		},
	}

	health := SystemHealth{
		ID:            "health-123",
		OverallStatus: "healthy",
		ActiveNodes:   5,
		TotalNodes:    6,
		Components:    components,
		Alerts:        alerts,
	}

	if health.ID != "health-123" {
		t.Errorf("Expected ID 'health-123', got '%s'", health.ID)
	}
	if health.OverallStatus != "healthy" {
		t.Errorf("Expected OverallStatus 'healthy', got '%s'", health.OverallStatus)
	}
	if health.ActiveNodes != 5 {
		t.Errorf("Expected ActiveNodes 5, got %d", health.ActiveNodes)
	}
	if health.TotalNodes != 6 {
		t.Errorf("Expected TotalNodes 6, got %d", health.TotalNodes)
	}
	if len(health.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(health.Components))
	}
	if len(health.Alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(health.Alerts))
	}
	if health.Components["database"].Status != "healthy" {
		t.Errorf("Expected database status 'healthy', got '%s'", health.Components["database"].Status)
	}
}

func TestAlert_StructFields(t *testing.T) {
	now := time.Now()
	resolvedAt := now.Add(time.Hour)

	metadata := map[string]interface{}{
		"cpu_usage": 95.5,
		"threshold": 90.0,
	}

	alert := Alert{
		ID:         "alert-123",
		Type:       "warning",
		Severity:   "high",
		Title:      "High CPU Usage",
		Message:    "CPU usage exceeded threshold",
		Source:     "system-monitor",
		NodeID:     "node-123",
		Metadata:   metadata,
		Status:     "resolved",
		ResolvedAt: &resolvedAt,
	}

	if alert.ID != "alert-123" {
		t.Errorf("Expected ID 'alert-123', got '%s'", alert.ID)
	}
	if alert.Title != "High CPU Usage" {
		t.Errorf("Expected Title 'High CPU Usage', got '%s'", alert.Title)
	}
	if alert.Message != "CPU usage exceeded threshold" {
		t.Errorf("Expected Message 'CPU usage exceeded threshold', got '%s'", alert.Message)
	}
	if alert.Source != "system-monitor" {
		t.Errorf("Expected Source 'system-monitor', got '%s'", alert.Source)
	}
	if alert.NodeID != "node-123" {
		t.Errorf("Expected NodeID 'node-123', got '%s'", alert.NodeID)
	}
	if alert.Type != "warning" {
		t.Errorf("Expected Type 'warning', got '%s'", alert.Type)
	}
	if alert.Severity != "high" {
		t.Errorf("Expected Severity 'high', got '%s'", alert.Severity)
	}
	if alert.Status != "resolved" {
		t.Errorf("Expected Status 'resolved', got '%s'", alert.Status)
	}
	if alert.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set")
	}
	if alert.Metadata["cpu_usage"] != 95.5 {
		t.Errorf("Expected cpu_usage 95.5, got '%v'", alert.Metadata["cpu_usage"])
	}
}
