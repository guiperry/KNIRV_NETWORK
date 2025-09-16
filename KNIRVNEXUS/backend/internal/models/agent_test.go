package models

import (
	"testing"
	"time"
)

func TestAgent_StructFields(t *testing.T) {
	now := time.Now()
	deployedAt := now.Add(time.Hour)
	lastActivity := now.Add(time.Minute * 30)

	resourceLimits := &AgentResourceLimits{
		MaxMemoryMB:      512,
		MaxCPUPercent:    80.0,
		MaxExecutionTime: 300,
		MaxConcurrency:   10,
		MaxDiskMB:        1024,
		NetworkAccess:    true,
		FileSystemAccess: false,
	}

	resourceUsage := &AgentResourceUsage{
		MemoryUsageMB:   256.5,
		CPUUsagePercent: 45.2,
		DiskUsageMB:     512.0,
		NetworkBytesIn:  1024000,
		NetworkBytesOut: 2048000,
		ExecutionTime:   150,
		RequestCount:    1000,
		ErrorCount:      5,
		LastUpdated:     now,
	}

	runtimeInstance := &AgentRuntimeInstance{
		InstanceID:      "instance-123",
		ProcessID:       12345,
		StartedAt:       now,
		Status:          "running",
		ResourceUsage:   resourceUsage,
		Configuration:   map[string]interface{}{"debug": true},
		Environment:     map[string]string{"ENV": "production"},
		Port:            8080,
		HealthCheckURL:  "/health",
		LastHealthCheck: &now,
		HealthStatus:    "healthy",
		RestartCount:    0,
		ErrorMessage:    "",
	}

	agent := Agent{
		ID:              "agent-123",
		Name:            "Test Agent",
		Description:     "A test WASM agent",
		Version:         "1.0.0",
		Author:          "test-author",
		Type:            "WASM",
		Status:          "running",
		FilePath:        "/agents/test-agent.wasm",
		FileSize:        1024000,
		FileHash:        "sha256:abcd1234",
		Capabilities:    []string{"inference", "validation"},
		Dependencies:    []string{"libmath", "libcrypto"},
		ResourceLimits:  resourceLimits,
		Configuration:   map[string]interface{}{"timeout": 30},
		Metadata:        map[string]interface{}{"category": "ml"},
		Tags:            []string{"ml", "inference"},
		UploadedAt:      now,
		DeployedAt:      &deployedAt,
		LastModified:    now,
		LastActivity:    &lastActivity,
		UploadedBy:      "user-123",
		DeployedBy:      "user-456",
		RuntimeInstance: runtimeInstance,
	}

	if agent.ID != "agent-123" {
		t.Errorf("Expected ID 'agent-123', got '%s'", agent.ID)
	}
	if agent.Type != "WASM" {
		t.Errorf("Expected Type 'WASM', got '%s'", agent.Type)
	}
	if agent.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", agent.Status)
	}
	if agent.FileSize != 1024000 {
		t.Errorf("Expected FileSize 1024000, got %d", agent.FileSize)
	}
	if len(agent.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(agent.Capabilities))
	}
	if agent.ResourceLimits.MaxMemoryMB != 512 {
		t.Errorf("Expected MaxMemoryMB 512, got %d", agent.ResourceLimits.MaxMemoryMB)
	}
	if agent.RuntimeInstance.Status != "running" {
		t.Errorf("Expected RuntimeInstance Status 'running', got '%s'", agent.RuntimeInstance.Status)
	}
}

func TestAgentResourceLimits_StructFields(t *testing.T) {
	limits := AgentResourceLimits{
		MaxMemoryMB:      1024,
		MaxCPUPercent:    90.5,
		MaxExecutionTime: 600,
		MaxConcurrency:   20,
		MaxDiskMB:        2048,
		NetworkAccess:    true,
		FileSystemAccess: true,
	}

	if limits.MaxMemoryMB != 1024 {
		t.Errorf("Expected MaxMemoryMB 1024, got %d", limits.MaxMemoryMB)
	}
	if limits.MaxCPUPercent != 90.5 {
		t.Errorf("Expected MaxCPUPercent 90.5, got %f", limits.MaxCPUPercent)
	}
	if limits.MaxExecutionTime != 600 {
		t.Errorf("Expected MaxExecutionTime 600, got %d", limits.MaxExecutionTime)
	}
	if !limits.NetworkAccess {
		t.Error("Expected NetworkAccess to be true")
	}
	if !limits.FileSystemAccess {
		t.Error("Expected FileSystemAccess to be true")
	}
}

func TestAgentResourceUsage_StructFields(t *testing.T) {
	now := time.Now()
	usage := AgentResourceUsage{
		MemoryUsageMB:   512.75,
		CPUUsagePercent: 65.3,
		DiskUsageMB:     1024.5,
		NetworkBytesIn:  5000000,
		NetworkBytesOut: 3000000,
		ExecutionTime:   450,
		RequestCount:    5000,
		ErrorCount:      25,
		LastUpdated:     now,
	}

	if usage.MemoryUsageMB != 512.75 {
		t.Errorf("Expected MemoryUsageMB 512.75, got %f", usage.MemoryUsageMB)
	}
	if usage.CPUUsagePercent != 65.3 {
		t.Errorf("Expected CPUUsagePercent 65.3, got %f", usage.CPUUsagePercent)
	}
	if usage.NetworkBytesIn != 5000000 {
		t.Errorf("Expected NetworkBytesIn 5000000, got %d", usage.NetworkBytesIn)
	}
	if usage.RequestCount != 5000 {
		t.Errorf("Expected RequestCount 5000, got %d", usage.RequestCount)
	}
	if usage.ErrorCount != 25 {
		t.Errorf("Expected ErrorCount 25, got %d", usage.ErrorCount)
	}
}

func TestAgentDeployment_StructFields(t *testing.T) {
	now := time.Now()

	healthCheck := &AgentHealthCheck{
		Enabled:          true,
		Path:             "/health",
		IntervalSeconds:  30,
		TimeoutSeconds:   5,
		FailureThreshold: 3,
		SuccessThreshold: 1,
	}

	deployment := AgentDeployment{
		ID:             "deployment-123",
		AgentID:        "agent-123",
		Name:           "Production Deployment",
		Description:    "Production deployment of test agent",
		Environment:    "production",
		Replicas:       3,
		Strategy:       "rolling",
		Configuration:  map[string]interface{}{"replicas": 3},
		ResourceLimits: &AgentResourceLimits{MaxMemoryMB: 512, MaxCPUPercent: 80.0, MaxExecutionTime: 300},
		HealthCheck:    healthCheck,
		AutoRestart:    true,
		RestartPolicy:  "always",
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "user-123",
		Status:         "deployed",
		Instances:      []*AgentRuntimeInstance{},
	}

	if deployment.ID != "deployment-123" {
		t.Errorf("Expected ID 'deployment-123', got '%s'", deployment.ID)
	}
	if deployment.Environment != "production" {
		t.Errorf("Expected Environment 'production', got '%s'", deployment.Environment)
	}
	if deployment.Replicas != 3 {
		t.Errorf("Expected Replicas 3, got %d", deployment.Replicas)
	}
	if deployment.Strategy != "rolling" {
		t.Errorf("Expected Strategy 'rolling', got '%s'", deployment.Strategy)
	}
	if !deployment.AutoRestart {
		t.Error("Expected AutoRestart to be true")
	}
	if deployment.HealthCheck.IntervalSeconds != 30 {
		t.Errorf("Expected HealthCheck IntervalSeconds 30, got %d", deployment.HealthCheck.IntervalSeconds)
	}
}

func TestAgentHealthCheck_StructFields(t *testing.T) {
	healthCheck := AgentHealthCheck{
		Enabled:          true,
		Path:             "/api/health",
		IntervalSeconds:  60,
		TimeoutSeconds:   10,
		FailureThreshold: 5,
		SuccessThreshold: 2,
	}

	if !healthCheck.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if healthCheck.Path != "/api/health" {
		t.Errorf("Expected Path '/api/health', got '%s'", healthCheck.Path)
	}
	if healthCheck.IntervalSeconds != 60 {
		t.Errorf("Expected IntervalSeconds 60, got %d", healthCheck.IntervalSeconds)
	}
	if healthCheck.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold 5, got %d", healthCheck.FailureThreshold)
	}
}

func TestAgentLog_StructFields(t *testing.T) {
	now := time.Now()
	log := AgentLog{
		ID:         "log-123",
		AgentID:    "agent-123",
		InstanceID: "instance-123",
		Level:      "error",
		Message:    "Connection timeout",
		Timestamp:  now,
		Source:     "network-module",
		Metadata:   map[string]interface{}{"retry_count": 3},
	}

	if log.ID != "log-123" {
		t.Errorf("Expected ID 'log-123', got '%s'", log.ID)
	}
	if log.Level != "error" {
		t.Errorf("Expected Level 'error', got '%s'", log.Level)
	}
	if log.Message != "Connection timeout" {
		t.Errorf("Expected Message 'Connection timeout', got '%s'", log.Message)
	}
	if log.Source != "network-module" {
		t.Errorf("Expected Source 'network-module', got '%s'", log.Source)
	}
	if log.Metadata["retry_count"] != 3 {
		t.Errorf("Expected retry_count 3, got '%v'", log.Metadata["retry_count"])
	}
}

func TestAgentMetrics_StructFields(t *testing.T) {
	now := time.Now()
	resourceUsage := &AgentResourceUsage{
		MemoryUsageMB:   256.0,
		CPUUsagePercent: 45.0,
		DiskUsageMB:     512.0,
		LastUpdated:     now,
	}

	metrics := AgentMetrics{
		AgentID:           "agent-123",
		InstanceID:        "instance-123",
		Timestamp:         now,
		RequestsPerSecond: 150.5,
		AverageLatency:    25.3,
		ErrorRate:         2.1,
		Throughput:        10.5,
		ResourceUsage:     resourceUsage,
	}

	if metrics.AgentID != "agent-123" {
		t.Errorf("Expected AgentID 'agent-123', got '%s'", metrics.AgentID)
	}
	if metrics.RequestsPerSecond != 150.5 {
		t.Errorf("Expected RequestsPerSecond 150.5, got %f", metrics.RequestsPerSecond)
	}
	if metrics.AverageLatency != 25.3 {
		t.Errorf("Expected AverageLatency 25.3, got %f", metrics.AverageLatency)
	}
	if metrics.ErrorRate != 2.1 {
		t.Errorf("Expected ErrorRate 2.1, got %f", metrics.ErrorRate)
	}
	if metrics.ResourceUsage.MemoryUsageMB != 256.0 {
		t.Errorf("Expected ResourceUsage MemoryUsageMB 256.0, got %f", metrics.ResourceUsage.MemoryUsageMB)
	}
}

func TestAgentEvent_StructFields(t *testing.T) {
	now := time.Now()
	event := AgentEvent{
		ID:          "event-123",
		AgentID:     "agent-123",
		InstanceID:  "instance-123",
		Type:        "deployed",
		Description: "Agent successfully deployed to production",
		Timestamp:   now,
		UserID:      "user-123",
		Metadata:    map[string]interface{}{"deployment_id": "deploy-456"},
	}

	if event.ID != "event-123" {
		t.Errorf("Expected ID 'event-123', got '%s'", event.ID)
	}
	if event.Type != "deployed" {
		t.Errorf("Expected Type 'deployed', got '%s'", event.Type)
	}
	if event.Description != "Agent successfully deployed to production" {
		t.Errorf("Expected Description 'Agent successfully deployed to production', got '%s'", event.Description)
	}
	if event.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", event.UserID)
	}
	if event.Metadata["deployment_id"] != "deploy-456" {
		t.Errorf("Expected deployment_id 'deploy-456', got '%v'", event.Metadata["deployment_id"])
	}
}

func TestAgentTemplate_StructFields(t *testing.T) {
	now := time.Now()
	template := AgentTemplate{
		ID:                   "template-123",
		Name:                 "ML Inference Template",
		Description:          "Template for ML inference agents",
		Version:              "2.0.0",
		Type:                 "WASM",
		SourceCode:           "fn main() { ... }",
		BuildScript:          "cargo build --release",
		DefaultConfig:        map[string]interface{}{"model_path": "/models/default"},
		ResourceLimits:       &AgentResourceLimits{MaxMemoryMB: 1024, MaxCPUPercent: 70.0, MaxExecutionTime: 600},
		RequiredCapabilities: []string{"inference", "model-loading"},
		Tags:                 []string{"ml", "template"},
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedBy:            "user-123",
		IsPublic:             true,
		UsageCount:           42,
	}

	if template.ID != "template-123" {
		t.Errorf("Expected ID 'template-123', got '%s'", template.ID)
	}
	if template.Name != "ML Inference Template" {
		t.Errorf("Expected Name 'ML Inference Template', got '%s'", template.Name)
	}
	if template.Type != "WASM" {
		t.Errorf("Expected Type 'WASM', got '%s'", template.Type)
	}
	if !template.IsPublic {
		t.Error("Expected IsPublic to be true")
	}
	if template.UsageCount != 42 {
		t.Errorf("Expected UsageCount 42, got %d", template.UsageCount)
	}
	if len(template.RequiredCapabilities) != 2 {
		t.Errorf("Expected 2 required capabilities, got %d", len(template.RequiredCapabilities))
	}
}

func TestAgentAction_StructFields(t *testing.T) {
	action := AgentAction{
		Action:     "scale",
		Parameters: map[string]interface{}{"replicas": 5, "strategy": "rolling"},
	}

	if action.Action != "scale" {
		t.Errorf("Expected Action 'scale', got '%s'", action.Action)
	}
	if action.Parameters["replicas"] != 5 {
		t.Errorf("Expected replicas 5, got '%v'", action.Parameters["replicas"])
	}
	if action.Parameters["strategy"] != "rolling" {
		t.Errorf("Expected strategy 'rolling', got '%v'", action.Parameters["strategy"])
	}
}

func TestAgent_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected bool
	}{
		{
			name: "Valid agent",
			agent: Agent{
				ID:     "agent-123",
				Name:   "Test Agent",
				Type:   "WASM",
				Status: "running",
			},
			expected: true,
		},
		{
			name: "Missing ID",
			agent: Agent{
				Name:   "Test Agent",
				Type:   "WASM",
				Status: "running",
			},
			expected: false,
		},
		{
			name: "Missing Name",
			agent: Agent{
				ID:     "agent-123",
				Type:   "WASM",
				Status: "running",
			},
			expected: false,
		},
		{
			name: "Missing Type",
			agent: Agent{
				ID:     "agent-123",
				Name:   "Test Agent",
				Status: "running",
			},
			expected: false,
		},
		{
			name: "Missing Status",
			agent: Agent{
				ID:   "agent-123",
				Name: "Test Agent",
				Type: "WASM",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.agent.IsValid(); got != tt.expected {
				t.Errorf("Agent.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgent_IsRunning(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected bool
	}{
		{
			name: "Running agent with runtime instance",
			agent: Agent{
				Status: "running",
				RuntimeInstance: &AgentRuntimeInstance{
					Status: "running",
				},
			},
			expected: true,
		},
		{
			name: "Running agent without runtime instance",
			agent: Agent{
				Status: "running",
			},
			expected: false,
		},
		{
			name: "Non-running agent",
			agent: Agent{
				Status: "stopped",
				RuntimeInstance: &AgentRuntimeInstance{
					Status: "running",
				},
			},
			expected: false,
		},
		{
			name: "Running agent with stopped runtime instance",
			agent: Agent{
				Status: "running",
				RuntimeInstance: &AgentRuntimeInstance{
					Status: "stopped",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.agent.IsRunning(); got != tt.expected {
				t.Errorf("Agent.IsRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgent_CanDeploy(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Uploaded agent", "uploaded", true},
		{"Stopped agent", "stopped", true},
		{"Running agent", "running", false},
		{"Deployed agent", "deployed", false},
		{"Error agent", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{Status: tt.status}
			if got := agent.CanDeploy(); got != tt.expected {
				t.Errorf("Agent.CanDeploy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgent_CanStart(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Deployed agent", "deployed", true},
		{"Stopped agent", "stopped", true},
		{"Running agent", "running", false},
		{"Uploaded agent", "uploaded", false},
		{"Error agent", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{Status: tt.status}
			if got := agent.CanStart(); got != tt.expected {
				t.Errorf("Agent.CanStart() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgent_CanStop(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Running agent", "running", true},
		{"Deployed agent", "deployed", false},
		{"Stopped agent", "stopped", false},
		{"Uploaded agent", "uploaded", false},
		{"Error agent", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{Status: tt.status}
			if got := agent.CanStop(); got != tt.expected {
				t.Errorf("Agent.CanStop() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentResourceLimits_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		limits   AgentResourceLimits
		expected bool
	}{
		{
			name: "Valid limits",
			limits: AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: true,
		},
		{
			name: "Zero memory",
			limits: AgentResourceLimits{
				MaxMemoryMB:      0,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: false,
		},
		{
			name: "Zero CPU",
			limits: AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    0,
				MaxExecutionTime: 300,
			},
			expected: false,
		},
		{
			name: "Zero execution time",
			limits: AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limits.IsValid(); got != tt.expected {
				t.Errorf("AgentResourceLimits.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentResourceUsage_IsWithinLimits(t *testing.T) {
	tests := []struct {
		name     string
		usage    AgentResourceUsage
		limits   *AgentResourceLimits
		expected bool
	}{
		{
			name: "Within limits",
			usage: AgentResourceUsage{
				MemoryUsageMB:   256.0,
				CPUUsagePercent: 50.0,
				ExecutionTime:   150,
			},
			limits: &AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: true,
		},
		{
			name: "Memory exceeded",
			usage: AgentResourceUsage{
				MemoryUsageMB:   600.0,
				CPUUsagePercent: 50.0,
				ExecutionTime:   150,
			},
			limits: &AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: false,
		},
		{
			name: "CPU exceeded",
			usage: AgentResourceUsage{
				MemoryUsageMB:   256.0,
				CPUUsagePercent: 90.0,
				ExecutionTime:   150,
			},
			limits: &AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: false,
		},
		{
			name: "Execution time exceeded",
			usage: AgentResourceUsage{
				MemoryUsageMB:   256.0,
				CPUUsagePercent: 50.0,
				ExecutionTime:   400,
			},
			limits: &AgentResourceLimits{
				MaxMemoryMB:      512,
				MaxCPUPercent:    80.0,
				MaxExecutionTime: 300,
			},
			expected: false,
		},
		{
			name: "No limits (nil)",
			usage: AgentResourceUsage{
				MemoryUsageMB:   1000.0,
				CPUUsagePercent: 100.0,
				ExecutionTime:   1000,
			},
			limits:   nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.usage.IsWithinLimits(tt.limits); got != tt.expected {
				t.Errorf("AgentResourceUsage.IsWithinLimits() = %v, want %v", got, tt.expected)
			}
		})
	}
}
