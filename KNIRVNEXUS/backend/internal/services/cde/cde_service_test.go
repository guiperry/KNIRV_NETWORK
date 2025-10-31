package cde

import (
	"testing"
	"time"

	dataengine "backend_server/internal/data-engine"
	"backend_server/internal/services/teesecurity"
)

func TestNewCDEService(t *testing.T) {
	// Mock dependencies
	teeService := &teesecurity.TEESecurityService{}
	dataEngine := &dataengine.BuntDBDataEngine{}

	config := CDEConfig{
		BaseImagePath:       "/tmp/images",
		WorkspaceRoot:       "/tmp/workspaces",
		MaxEnvironments:     10,
		DefaultTimeout:      30 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     4 * 1024 * 1024 * 1024, // 4GB
		MaxDiskPerEnv:       20 * 1024 * 1024 * 1024, // 20GB
		SessionTimeout:      2 * time.Hour,
		MaxSessionsPerUser:  5,
		MaxProjectsPerUser:  10,
		ProjectStoragePath:  "/tmp/projects",
	}

	service, err := NewCDEService(teeService, dataEngine, config)
	if err != nil {
		t.Fatalf("Failed to create CDE service: %v", err)
	}

	if service == nil {
		t.Fatal("Service is nil")
	}

	if service.teeSecurityService != teeService {
		t.Error("TEE security service not set correctly")
	}

	if service.dataEngine != dataEngine {
		t.Error("Data engine not set correctly")
	}

	if service.config.MaxEnvironments != config.MaxEnvironments {
		t.Errorf("Expected max environments %d, got %d", config.MaxEnvironments, service.config.MaxEnvironments)
	}
}

func TestEnvironmentStatus(t *testing.T) {
	status := EnvStatusRunning
	if status != "running" {
		t.Errorf("Expected status 'running', got '%s'", status)
	}

	if EnvStatusCreating != "creating" {
		t.Error("EnvironmentStatus constants not set correctly")
	}
}

func TestEnvironmentType(t *testing.T) {
	envType := EnvTypePython
	if envType != "python" {
		t.Errorf("Expected type 'python', got '%s'", envType)
	}

	if EnvTypeNodeJS != "nodejs" {
		t.Error("EnvironmentType constants not set correctly")
	}
}

func TestSessionStatus(t *testing.T) {
	status := SessionStatusActive
	if status != "active" {
		t.Errorf("Expected status 'active', got '%s'", status)
	}

	if SessionStatusIdle != "idle" {
		t.Error("SessionStatus constants not set correctly")
	}
}

func TestProjectType(t *testing.T) {
	projectType := ProjectTypeWebApp
	if projectType != "webapp" {
		t.Errorf("Expected type 'webapp', got '%s'", projectType)
	}

	if ProjectTypeAPI != "api" {
		t.Error("ProjectType constants not set correctly")
	}
}

func TestGetBaseImageForType(t *testing.T) {
	service, _ := NewCDEService(nil, nil, CDEConfig{})

	tests := []struct {
		envType EnvironmentType
		expected string
	}{
		{EnvTypePython, "python:3.11-slim"},
		{EnvTypeNodeJS, "node:18-alpine"},
		{EnvTypeGo, "golang:1.21-alpine"},
		{EnvTypeRust, "rust:1.70-slim"},
		{EnvTypeJava, "openjdk:17-jdk-slim"},
		{EnvTypeGeneral, "ubuntu:22.04"},
	}

	for _, test := range tests {
		result := service.getBaseImageForType(test.envType)
		if result != test.expected {
			t.Errorf("Expected base image %s for type %s, got %s", test.expected, test.envType, result)
		}
	}
}

func TestGetRequiredEnvType(t *testing.T) {
	service, _ := NewCDEService(nil, nil, CDEConfig{})

	tests := []struct {
		language string
		expected EnvironmentType
	}{
		{"python", EnvTypePython},
		{"javascript", EnvTypeNodeJS},
		{"typescript", EnvTypeNodeJS},
		{"go", EnvTypeGo},
		{"rust", EnvTypeRust},
		{"java", EnvTypeJava},
		{"kotlin", EnvTypeJava},
		{"scala", EnvTypeJava},
		{"unknown", EnvTypeGeneral},
	}

	for _, test := range tests {
		result := service.getRequiredEnvType(test.language)
		if result != test.expected {
			t.Errorf("Expected env type %s for language %s, got %s", test.expected, test.language, result)
		}
	}
}

func TestCDEResourceAllocation(t *testing.T) {
	resources := &CDEResourceAllocation{
		CPUCores:         2.0,
		MemoryBytes:      4 * 1024 * 1024 * 1024,
		DiskBytes:        20 * 1024 * 1024 * 1024,
		NetworkBandwidth: 100 * 1024 * 1024,
		CPULimit:         2.0,
		MemoryLimit:      4 * 1024 * 1024 * 1024,
		DiskLimit:        20 * 1024 * 1024 * 1024,
	}

	if resources.CPUCores != 2.0 {
		t.Errorf("Expected CPU cores 2.0, got %f", resources.CPUCores)
	}

	if resources.MemoryBytes != 4*1024*1024*1024 {
		t.Errorf("Expected memory bytes %d, got %d", 4*1024*1024*1024, resources.MemoryBytes)
	}
}

func TestCDEEnvironment(t *testing.T) {
	now := time.Now()
	env := &CDEEnvironment{
		ID:              "env-123",
		Name:            "test-env",
		UserID:          "user-123",
		Status:          EnvStatusRunning,
		CreatedAt:       now,
		LastAccessed:    now,
		EnvironmentType: EnvTypePython,
		BaseImage:       "python:3.11-slim",
		WorkspacePath:   "/tmp/workspace",
		Resources: &CDEResourceAllocation{
			CPUCores:    2.0,
			MemoryBytes: 4 * 1024 * 1024 * 1024,
		},
		Ports:     make(map[string]int),
		Config:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	if env.ID != "env-123" {
		t.Errorf("Expected ID 'env-123', got '%s'", env.ID)
	}

	if env.Status != EnvStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", env.Status)
	}

	if env.EnvironmentType != EnvTypePython {
		t.Errorf("Expected type 'python', got '%s'", env.EnvironmentType)
	}
}

func TestCDESession(t *testing.T) {
	now := time.Now()
	session := &CDESession{
		ID:            "sess-123",
		EnvironmentID: "env-123",
		UserID:        "user-123",
		Status:        SessionStatusActive,
		StartTime:     now,
		LastActivity:  now,
		ExpiresAt:     now.Add(2 * time.Hour),
		ConnectionType: "ssh",
		ConnectionInfo: make(map[string]string),
		WorkingDirectory: "/workspace",
		OpenFiles:        []string{"main.py"},
		RunningProcesses: []string{"python"},
	}

	if session.ID != "sess-123" {
		t.Errorf("Expected ID 'sess-123', got '%s'", session.ID)
	}

	if session.Status != SessionStatusActive {
		t.Errorf("Expected status 'active', got '%s'", session.Status)
	}

	if session.ConnectionType != "ssh" {
		t.Errorf("Expected connection type 'ssh', got '%s'", session.ConnectionType)
	}
}

func TestCDEProject(t *testing.T) {
	now := time.Now()
	project := &CDEProject{
		ID:              "proj-123",
		Name:            "test-project",
		Description:     "A test project",
		UserID:          "user-123",
		CreatedAt:       now,
		UpdatedAt:       now,
		ProjectType:     ProjectTypeWebApp,
		Language:        "python",
		Framework:       "django",
		StoragePath:     "/tmp/projects/test-project",
		RequiredEnvType: EnvTypePython,
		Dependencies:    []string{"django", "psycopg2"},
		Tags:            make(map[string]string),
		Config:          make(map[string]interface{}),
		TotalSessions:   5,
		LastAccessed:    now,
	}

	if project.ID != "proj-123" {
		t.Errorf("Expected ID 'proj-123', got '%s'", project.ID)
	}

	if project.ProjectType != ProjectTypeWebApp {
		t.Errorf("Expected project type 'webapp', got '%s'", project.ProjectType)
	}

	if project.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", project.Language)
	}
}

func TestCDEConfig(t *testing.T) {
	config := CDEConfig{
		BaseImagePath:       "/opt/cde/images",
		WorkspaceRoot:       "/opt/cde/workspaces",
		MaxEnvironments:     50,
		DefaultTimeout:      60 * time.Minute,
		MaxCPUPerEnv:        4.0,
		MaxMemoryPerEnv:     8 * 1024 * 1024 * 1024,
		MaxDiskPerEnv:       100 * 1024 * 1024 * 1024,
		EnableSandboxing:       true,
		EnableNetworkIsolation: true,
		AllowedPorts:           []int{22, 80, 443, 8080},
		SessionTimeout:         4 * time.Hour,
		MaxSessionsPerUser:     10,
		MaxProjectsPerUser:     20,
		ProjectStoragePath:     "/opt/cde/projects",
	}

	if config.MaxEnvironments != 50 {
		t.Errorf("Expected max environments 50, got %d", config.MaxEnvironments)
	}

	if config.EnableSandboxing != true {
		t.Error("Expected sandboxing to be enabled")
	}

	if len(config.AllowedPorts) != 4 {
		t.Errorf("Expected 4 allowed ports, got %d", len(config.AllowedPorts))
	}
}