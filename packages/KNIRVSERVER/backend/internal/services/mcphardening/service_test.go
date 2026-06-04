package mcphardening

import (
	"testing"
	"time"
)

func TestNewMCPGateway(t *testing.T) {
	gw := NewMCPGateway()
	if gw == nil {
		t.Fatal("Expected non-nil MCPGateway")
	}
	if gw.endpoints == nil {
		t.Error("Expected endpoints map to be initialized")
	}
	if gw.auditLog == nil {
		t.Error("Expected auditLog to be initialized")
	}
}

func TestRegisterTool(t *testing.T) {
	validator := NewToolCallValidator()
	tool := &ToolDefinition{
		Name:        "read_file",
		Description: "Read file contents",
		Parameters:  []string{"path"},
		MaxArgs:     2,
	}
	validator.RegisterTool(tool)

	retrieved, ok := validator.GetTool("read_file")
	if !ok {
		t.Error("Expected to retrieve registered tool")
	}
	if retrieved.Name != "read_file" {
		t.Errorf("Expected name read_file, got %s", retrieved.Name)
	}
}

func TestListTools(t *testing.T) {
	validator := NewToolCallValidator()
	validator.RegisterTool(&ToolDefinition{Name: "tool-1"})
	validator.RegisterTool(&ToolDefinition{Name: "tool-2"})

	tools := validator.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestToolCallValidationAllowed(t *testing.T) {
	validator := NewToolCallValidator()
	validator.RegisterTool(&ToolDefinition{
		Name:    "safe_tool",
		MaxArgs: 3,
	})

	status, reason := validator.ValidateCall("agent-1", "safe_tool", map[string]interface{}{
		"arg1": "value1",
	})
	if status != ToolCallStatusAllowed {
		t.Errorf("Expected allowed, got %s: %s", status, reason)
	}
}

func TestToolCallValidationUnknownTool(t *testing.T) {
	validator := NewToolCallValidator()
	status, reason := validator.ValidateCall("agent-1", "unknown_tool", nil)
	if status != ToolCallStatusFlagged {
		t.Errorf("Expected flagged for unknown tool, got %s", status)
	}
	if reason == "" {
		t.Error("Expected non-empty reason for unknown tool")
	}
}

func TestToolCallValidationExceedsMaxArgs(t *testing.T) {
	validator := NewToolCallValidator()
	validator.RegisterTool(&ToolDefinition{
		Name:    "limited_tool",
		MaxArgs: 1,
	})

	status, _ := validator.ValidateCall("agent-1", "limited_tool", map[string]interface{}{
		"arg1": "v1",
		"arg2": "v2",
	})
	if status != ToolCallStatusDenied {
		t.Errorf("Expected denied for exceeding max args, got %s", status)
	}
}

func TestToolCallRateLimiting(t *testing.T) {
	validator := NewToolCallValidator()
	validator.SetMaxCallsPerMin(2)
	validator.RegisterTool(&ToolDefinition{Name: "tool"})

	for i := 0; i < 2; i++ {
		status, _ := validator.ValidateCall("agent-1", "tool", nil)
		if status != ToolCallStatusAllowed {
			t.Errorf("Expected allowed on call %d, got %s", i+1, status)
		}
	}

	status, _ := validator.ValidateCall("agent-1", "tool", nil)
	if status != ToolCallStatusBlocked {
		t.Errorf("Expected blocked after rate limit, got %s", status)
	}
}

func TestAgentCallStats(t *testing.T) {
	validator := NewToolCallValidator()
	validator.RegisterTool(&ToolDefinition{Name: "tool"})

	validator.ValidateCall("agent-1", "tool", nil)
	validator.ValidateCall("agent-1", "unknown", nil)

	stats := validator.GetAgentCallStats("agent-1")
	if stats["allowed"] != 2 {
		t.Errorf("Expected 2 allowed, got %d", stats["allowed"])
	}
}

func TestNewPoisoningDetector(t *testing.T) {
	pd := NewPoisoningDetector()
	if pd == nil {
		t.Fatal("Expected non-nil PoisoningDetector")
	}
	if pd.signatures == nil {
		t.Error("Expected signatures slice to be initialized")
	}
}

func TestPoisoningDetectionNoMatch(t *testing.T) {
	pd := NewPoisoningDetector()
	pd.RegisterSignature(PoisoningSignature{
		Pattern:    "dangerous_tool",
		Confidence: 0.9,
		Indicators: []string{"known_bad_pattern"},
		Severity:   PoisoningLevelConfirmed,
	})

	level, indicators := pd.AnalyzeCall("agent-1", "safe_tool", nil)
	if level != PoisoningLevelNone {
		t.Errorf("Expected no poisoning for safe tool, got %d", level)
	}
	if indicators != nil {
		t.Errorf("Expected nil indicators for safe tool, got %v", indicators)
	}
}

func TestPoisoningDetectionMatch(t *testing.T) {
	pd := NewPoisoningDetector()
	pd.RegisterSignature(PoisoningSignature{
		Pattern:    "dangerous_tool",
		Confidence: 0.9,
		Indicators: []string{"known_bad_pattern", "high_risk_action"},
		Severity:   PoisoningLevelConfirmed,
	})

	level, indicators := pd.AnalyzeCall("agent-1", "dangerous_tool", nil)
	if level != PoisoningLevelConfirmed {
		t.Errorf("Expected confirmed poisoning, got %d", level)
	}
	if len(indicators) == 0 {
		t.Error("Expected non-empty indicators")
	}
}

func TestPoisoningSuspiciousCallCount(t *testing.T) {
	pd := NewPoisoningDetector()
	pd.RegisterSignature(PoisoningSignature{
		Pattern:    "bad_tool",
		Confidence: 0.9,
		Indicators: []string{"bad"},
		Severity:   PoisoningLevelSuspicious,
	})

	pd.AnalyzeCall("agent-1", "bad_tool", nil)
	pd.AnalyzeCall("agent-1", "bad_tool", nil)

	count := pd.GetSuspiciousCallCount("agent-1")
	if count != 2 {
		t.Errorf("Expected 2 suspicious calls, got %d", count)
	}
}

func TestSetThreshold(t *testing.T) {
	pd := NewPoisoningDetector()
	pd.SetThreshold(0.5)
	if pd.threshold != 0.5 {
		t.Errorf("Expected threshold 0.5, got %f", pd.threshold)
	}
}

func TestRegisterEndpoint(t *testing.T) {
	gw := NewMCPGateway()
	gw.RegisterEndpoint("main-api", &MCPEndpoint{
		URL:          "https://api.example.com",
		AllowedTools: []string{"tool-1", "tool-2"},
		RateLimit:    100,
		AuthRequired: true,
	})

	ep, ok := gw.GetEndpoint("main-api")
	if !ok {
		t.Error("Expected to retrieve registered endpoint")
	}
	if ep.URL != "https://api.example.com" {
		t.Errorf("Expected URL https://api.example.com, got %s", ep.URL)
	}
}

func TestListEndpoints(t *testing.T) {
	gw := NewMCPGateway()
	gw.RegisterEndpoint("ep-1", &MCPEndpoint{URL: "https://ep1.example.com"})
	gw.RegisterEndpoint("ep-2", &MCPEndpoint{URL: "https://ep2.example.com"})

	eps := gw.ListEndpoints()
	if len(eps) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(eps))
	}
}

func TestProcessToolCallAllow(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{
		Name: "safe_tool",
	})

	record := &ToolCallRecord{
		ID:       "call-1",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "safe_tool",
	}

	status := gw.ProcessToolCall(record)
	if status != ToolCallStatusAllowed {
		t.Errorf("Expected allowed, got %s", status)
	}

	log := gw.GetAuditLog("", 0)
	if len(log) != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", len(log))
	}
}

func TestProcessToolCallBlockPoisoning(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{
		Name: "bad_tool",
	})
	gw.Detector().RegisterSignature(PoisoningSignature{
		Pattern:    "bad_tool",
		Confidence: 0.9,
		Indicators: []string{"malicious_pattern"},
		Severity:   PoisoningLevelConfirmed,
	})

	record := &ToolCallRecord{
		ID:       "call-2",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "bad_tool",
	}

	status := gw.ProcessToolCall(record)
	if status != ToolCallStatusBlocked {
		t.Errorf("Expected blocked for poisoned call, got %s", status)
	}
}

func TestProcessToolCallFlaggedUnknown(t *testing.T) {
	gw := NewMCPGateway()
	record := &ToolCallRecord{
		ID:       "call-3",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "nonexistent_tool",
	}

	status := gw.ProcessToolCall(record)
	if status != ToolCallStatusFlagged {
		t.Errorf("Expected flagged for unknown tool, got %s", status)
	}
}

func TestProcessToolCallDeniedExceedArgs(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{
		Name:    "limited_tool",
		MaxArgs: 1,
	})

	record := &ToolCallRecord{
		ID:       "call-4",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "limited_tool",
		Arguments: map[string]interface{}{
			"arg1": "value1",
			"arg2": "value2",
		},
	}

	status := gw.ProcessToolCall(record)
	if status != ToolCallStatusDenied {
		t.Errorf("Expected denied for exceeding args (%d > %d), got %s", len(record.Arguments), 1, status)
	}
}

func TestGetAuditLogFilter(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{Name: "tool"})

	gw.ProcessToolCall(&ToolCallRecord{ID: "c1", AgentID: "agent-1", ToolName: "tool"})
	gw.ProcessToolCall(&ToolCallRecord{ID: "c2", AgentID: "agent-2", ToolName: "tool"})

	log := gw.GetAuditLog("agent-1", 0)
	if len(log) != 1 {
		t.Errorf("Expected 1 log entry for agent-1, got %d", len(log))
	}

	log = gw.GetAuditLog("", 0)
	if len(log) != 2 {
		t.Errorf("Expected 2 log entries total, got %d", len(log))
	}
}

func TestGetStatistics(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{Name: "tool-1"})
	gw.RegisterEndpoint("ep-1", &MCPEndpoint{URL: "https://example.com"})
	gw.Detector().RegisterSignature(PoisoningSignature{
		Pattern: "test", Confidence: 0.8, Severity: PoisoningLevelSuspicious,
	})

	gw.ProcessToolCall(&ToolCallRecord{
		ID: "c1", AgentID: "agent-1", NodeID: "node-1", ToolName: "tool-1",
	})

	stats := gw.GetStatistics()
	if stats["total_calls"].(int) != 1 {
		t.Errorf("Expected 1 total call, got %d", stats["total_calls"])
	}
	if stats["registered_tools"].(int) != 1 {
		t.Errorf("Expected 1 registered tool, got %d", stats["registered_tools"])
	}
	if stats["registered_endpoints"].(int) != 1 {
		t.Errorf("Expected 1 registered endpoint, got %d", stats["registered_endpoints"])
	}
	if stats["registered_signatures"].(int) != 1 {
		t.Errorf("Expected 1 registered signature, got %d", stats["registered_signatures"])
	}
}

func TestToolCallRecordTimestamp(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{Name: "tool"})

	record := &ToolCallRecord{
		ID:       "c1",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "tool",
	}
	gw.ProcessToolCall(record)

	if record.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestProcessToolCallDuration(t *testing.T) {
	gw := NewMCPGateway()
	gw.Validator().RegisterTool(&ToolDefinition{Name: "tool"})

	record := &ToolCallRecord{
		ID:       "c1",
		AgentID:  "agent-1",
		NodeID:   "node-1",
		ToolName: "tool",
		Duration: 150 * time.Millisecond,
	}
	gw.ProcessToolCall(record)
	if record.Duration != 150*time.Millisecond {
		t.Errorf("Expected duration to be preserved")
	}
}
