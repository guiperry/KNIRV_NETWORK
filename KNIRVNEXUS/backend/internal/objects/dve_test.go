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
