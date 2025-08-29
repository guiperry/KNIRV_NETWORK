package agentify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTEESecurityManager(t *testing.T) {
	// Create a test configuration
	config := TEESecurityConfig{
		DefaultSecurityLevel:    SecurityLevelMedium,
		MaxConcurrentTEEs:       5,
		AuditLogRetentionDays:   7,
		IntegrityCheckInterval:  1 * time.Second,
		EnableRuntimeMonitoring: false, // Disable for testing
		GlobalResourceLimits: ResourceLimits{
			MemoryMB:         1024,
			CPUCores:         2.0,
			MaxProcesses:     10,
			ExecutionTimeout: 30 * time.Second,
		},
	}

	manager := NewTEESecurityManager(config)

	t.Run("CreateSecureTEE", func(t *testing.T) {
		teeConfig := TEEConfig{
			WorkingDir: "/tmp/test_tee",
			ResourceLimits: ResourceLimits{
				MemoryMB:         512,
				CPUCores:         1.0,
				ExecutionTimeout: 15 * time.Second,
			},
			SecurityPolicy: SecurityPolicy{
				AllowNetworkAccess:   true,
				AllowFileSystemWrite: false,
				AllowedCommands:      []string{"echo", "cat"},
				BlockedCommands:      []string{"rm", "sudo"},
				MaxExecutionTime:     30 * time.Second,
			},
		}

		tee, err := manager.CreateSecureTEE("test-agent-1", SecurityLevelLow, teeConfig)
		if err != nil {
			t.Fatalf("Failed to create secure TEE: %v", err)
		}

		if tee == nil {
			t.Fatal("TEE should not be nil")
		}

		// Verify TEE is registered
		if len(manager.activeTEEs) != 1 {
			t.Errorf("Expected 1 active TEE, got %d", len(manager.activeTEEs))
		}

		// Clean up
		err = manager.DestroyTEE("test-agent-1")
		if err != nil {
			t.Errorf("Failed to destroy TEE: %v", err)
		}
	})

	t.Run("ValidateAgentExecution", func(t *testing.T) {
		teeConfig := TEEConfig{
			WorkingDir: "/tmp/test_tee",
			SecurityPolicy: SecurityPolicy{
				AllowedCommands: []string{"echo", "cat"},
				BlockedCommands: []string{"rm", "sudo"},
			},
		}

		_, err := manager.CreateSecureTEE("test-agent-2", SecurityLevelLow, teeConfig)
		if err != nil {
			t.Fatalf("Failed to create secure TEE: %v", err)
		}

		// Test allowed command
		err = manager.ValidateAgentExecution("test-agent-2", "echo", []string{"hello"})
		if err != nil {
			t.Errorf("Expected allowed command to pass validation: %v", err)
		}

		// Test blocked command
		err = manager.ValidateAgentExecution("test-agent-2", "rm", []string{"-rf", "/"})
		if err == nil {
			t.Error("Expected blocked command to fail validation")
		}

		// Clean up
		manager.DestroyTEE("test-agent-2")
	})

	t.Run("MonitorTEEHealth", func(t *testing.T) {
		teeConfig := TEEConfig{
			WorkingDir: "/tmp/test_tee",
			ResourceLimits: ResourceLimits{
				MemoryMB:         256,
				CPUCores:         0.5,
				ExecutionTimeout: 10 * time.Second,
			},
		}

		_, err := manager.CreateSecureTEE("test-agent-3", SecurityLevelLow, teeConfig)
		if err != nil {
			t.Fatalf("Failed to create secure TEE: %v", err)
		}

		healthStatus := manager.MonitorTEEHealth()
		if len(healthStatus) != 1 {
			t.Errorf("Expected 1 health status, got %d", len(healthStatus))
		}

		status, exists := healthStatus["test-agent-3"]
		if !exists {
			t.Error("Expected health status for test-agent-3")
		}

		if status.AgentID != "test-agent-3" {
			t.Errorf("Expected agent ID 'test-agent-3', got '%s'", status.AgentID)
		}

		// Clean up
		manager.DestroyTEE("test-agent-3")
	})

	t.Run("ConcurrentTEELimit", func(t *testing.T) {
		teeConfig := TEEConfig{
			WorkingDir: "/tmp/test_tee",
		}

		// Create TEEs up to the limit
		for i := 0; i < config.MaxConcurrentTEEs; i++ {
			agentID := fmt.Sprintf("test-agent-%d", i)
			_, err := manager.CreateSecureTEE(agentID, SecurityLevelLow, teeConfig)
			if err != nil {
				t.Fatalf("Failed to create TEE %d: %v", i, err)
			}
		}

		// Try to create one more (should fail)
		_, err := manager.CreateSecureTEE("test-agent-overflow", SecurityLevelLow, teeConfig)
		if err == nil {
			t.Error("Expected TEE creation to fail due to concurrent limit")
		}

		// Clean up all TEEs
		for i := 0; i < config.MaxConcurrentTEEs; i++ {
			agentID := fmt.Sprintf("test-agent-%d", i)
			manager.DestroyTEE(agentID)
		}
	})

	t.Run("SecurityAuditLog", func(t *testing.T) {
		initialLogCount := len(manager.GetSecurityAuditLog())

		teeConfig := TEEConfig{
			WorkingDir: "/tmp/test_tee",
		}

		// Create and destroy a TEE to generate audit events
		_, err := manager.CreateSecureTEE("test-agent-audit", SecurityLevelLow, teeConfig)
		if err != nil {
			t.Fatalf("Failed to create secure TEE: %v", err)
		}

		err = manager.DestroyTEE("test-agent-audit")
		if err != nil {
			t.Fatalf("Failed to destroy TEE: %v", err)
		}

		auditLog := manager.GetSecurityAuditLog()
		if len(auditLog) <= initialLogCount {
			t.Error("Expected audit log to contain new events")
		}

		// Check for expected event types
		foundCreation := false
		foundDestruction := false
		for _, event := range auditLog {
			if event.Type == "tee_created" && event.AgentID == "test-agent-audit" {
				foundCreation = true
			}
			if event.Type == "tee_destroyed" && event.AgentID == "test-agent-audit" {
				foundDestruction = true
			}
		}

		if !foundCreation {
			t.Error("Expected to find TEE creation event in audit log")
		}
		if !foundDestruction {
			t.Error("Expected to find TEE destruction event in audit log")
		}
	})
}

func TestTEESecurityConfigManager(t *testing.T) {
	// Create temporary directory for test config
	tempDir, err := os.MkdirTemp("", "tee_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewTEESecurityConfigManager(tempDir)

	t.Run("DefaultProfiles", func(t *testing.T) {
		profiles := manager.GetAvailableProfiles()

		expectedProfiles := []string{"development", "production", "high_security", "sandbox"}
		for _, expected := range expectedProfiles {
			if _, exists := profiles[expected]; !exists {
				t.Errorf("Expected default profile '%s' to exist", expected)
			}
		}
	})

	t.Run("GetProfileForAgent", func(t *testing.T) {
		// Test default profile assignment
		profile, err := manager.GetProfileForAgent("test-agent")
		if err != nil {
			t.Fatalf("Failed to get profile for agent: %v", err)
		}

		if profile.Name != "Production" {
			t.Errorf("Expected default profile 'Production', got '%s'", profile.Name)
		}
	})

	t.Run("SetProfileForAgent", func(t *testing.T) {
		err := manager.SetProfileForAgent("test-agent", "high_security")
		if err != nil {
			t.Fatalf("Failed to set profile for agent: %v", err)
		}

		profile, err := manager.GetProfileForAgent("test-agent")
		if err != nil {
			t.Fatalf("Failed to get profile for agent: %v", err)
		}

		if profile.Name != "High Security" {
			t.Errorf("Expected profile 'High Security', got '%s'", profile.Name)
		}
	})

	t.Run("AddCustomProfile", func(t *testing.T) {
		customProfile := TEESecurityProfile{
			Name:          "custom_test",
			Description:   "Custom test profile",
			SecurityLevel: SecurityLevelMedium,
			ResourceLimits: ResourceLimits{
				MemoryMB:         256,
				CPUCores:         1.0,
				ExecutionTimeout: 20 * time.Second,
			},
			SecurityPolicy: SecurityPolicy{
				AllowNetworkAccess:   false,
				AllowFileSystemWrite: false,
				AllowedCommands:      []string{"echo"},
				MaxExecutionTime:     30 * time.Second,
			},
		}

		err := manager.AddCustomProfile("custom_test", customProfile)
		if err != nil {
			t.Fatalf("Failed to add custom profile: %v", err)
		}

		profiles := manager.GetAvailableProfiles()
		if _, exists := profiles["custom_test"]; !exists {
			t.Error("Expected custom profile to be available")
		}
	})

	t.Run("ValidateProfile", func(t *testing.T) {
		validProfile := TEESecurityProfile{
			Name:          "valid_test",
			SecurityLevel: SecurityLevelLow,
			ResourceLimits: ResourceLimits{
				MemoryMB:         128,
				CPUCores:         0.5,
				ExecutionTimeout: 10 * time.Second,
			},
			SecurityPolicy: SecurityPolicy{
				MaxExecutionTime: 15 * time.Second,
			},
		}

		err := manager.ValidateProfile(validProfile)
		if err != nil {
			t.Errorf("Expected valid profile to pass validation: %v", err)
		}

		// Test invalid profile (exceeds global limits)
		invalidProfile := validProfile
		invalidProfile.ResourceLimits.MemoryMB = 10000 // Exceeds global limit

		err = manager.ValidateProfile(invalidProfile)
		if err == nil {
			t.Error("Expected invalid profile to fail validation")
		}
	})

	t.Run("GetRecommendedProfile", func(t *testing.T) {
		testCases := []struct {
			agentType       string
			useCase         string
			expectedProfile string
		}{
			{"llm", "development", "development"},
			{"llm", "production", "production"},
			{"llm", "financial", "high_security"},
			{"llm", "untrusted_code", "sandbox"},
			{"llm", "unknown", "production"},
		}

		for _, tc := range testCases {
			profile, err := manager.GetRecommendedProfile(tc.agentType, tc.useCase)
			if err != nil {
				t.Errorf("Failed to get recommended profile for %s/%s: %v", tc.agentType, tc.useCase, err)
			}

			if profile != tc.expectedProfile {
				t.Errorf("Expected profile '%s' for %s/%s, got '%s'", tc.expectedProfile, tc.agentType, tc.useCase, profile)
			}
		}
	})

	t.Run("SaveAndLoadConfiguration", func(t *testing.T) {
		// Modify configuration
		err := manager.SetProfileForAgent("save-test-agent", "sandbox")
		if err != nil {
			t.Fatalf("Failed to set profile: %v", err)
		}

		// Create new manager to test loading
		newManager := NewTEESecurityConfigManager(tempDir)

		profile, err := newManager.GetProfileForAgent("save-test-agent")
		if err != nil {
			t.Fatalf("Failed to get profile from loaded config: %v", err)
		}

		if profile.Name != "Sandbox" {
			t.Errorf("Expected loaded profile 'Sandbox', got '%s'", profile.Name)
		}

		// Verify config file exists
		configFile := filepath.Join(tempDir, "tee_security_config.json")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Expected config file to be created")
		}
	})
}
