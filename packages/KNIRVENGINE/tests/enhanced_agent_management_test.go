package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"KNIRVENGINE/desktop-client/internal/agent"
)

func TestEnhancedAgentManagement(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "enhanced_agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock registry and builder
	registry := createMockRegistry(t, tempDir)
	builder := createMockBuilder(t, tempDir)

	// Create enhanced agent manager
	manager, err := agent.NewEnhancedAgentManager(registry, builder, tempDir)
	if err != nil {
		t.Fatalf("Failed to create enhanced agent manager: %v", err)
	}

	// Test agent versioning
	t.Run("Agent Versioning", func(t *testing.T) {
		testAgentVersioning(t, manager)
	})

	// Test agent backup/restore
	t.Run("Agent Backup/Restore", func(t *testing.T) {
		testAgentBackupRestore(t, manager)
	})

	// Test health monitoring
	t.Run("Health Monitoring", func(t *testing.T) {
		testHealthMonitoring(t, manager)
	})

	// Test performance analytics
	t.Run("Performance Analytics", func(t *testing.T) {
		testPerformanceAnalytics(t, manager)
	})
}

func testAgentVersioning(t *testing.T, manager *agent.EnhancedAgentManager) {
	agentID := "test-agent-1"

	// Create a version
	version, err := manager.CreateAgentVersion(
		agentID,
		"v1.0.0",
		"Initial version with basic functionality",
		[]string{"stable", "production"},
	)
	if err != nil {
		t.Fatalf("Failed to create agent version: %v", err)
	}

	// Verify version properties
	if version.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", version.Version)
	}
	if version.AgentID != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, version.AgentID)
	}
	if len(version.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(version.Tags))
	}

	// Create another version
	_, err = manager.CreateAgentVersion(
		agentID,
		"v1.1.0",
		"Added new features and bug fixes",
		[]string{"stable", "feature-update"},
	)
	if err != nil {
		t.Fatalf("Failed to create second agent version: %v", err)
	}

	// List versions
	versions, err := manager.ListAgentVersions(agentID)
	if err != nil {
		t.Fatalf("Failed to list agent versions: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Verify versions are sorted by creation time (newest first)
	if versions[0].Version != "v1.1.0" {
		t.Errorf("Expected newest version v1.1.0 first, got %s", versions[0].Version)
	}
	if versions[1].Version != "v1.0.0" {
		t.Errorf("Expected older version v1.0.0 second, got %s", versions[1].Version)
	}

	t.Logf("✅ Agent versioning test passed - created %d versions", len(versions))
}

func testAgentBackupRestore(t *testing.T, manager *agent.EnhancedAgentManager) {
	agentID := "test-agent-2"

	// Create a backup
	backup, err := manager.CreateAgentBackup(
		agentID,
		"Pre-deployment backup",
		"manual",
	)
	if err != nil {
		t.Fatalf("Failed to create agent backup: %v", err)
	}

	// Verify backup properties
	if backup.AgentID != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, backup.AgentID)
	}
	if backup.Type != "manual" {
		t.Errorf("Expected backup type manual, got %s", backup.Type)
	}
	if backup.Size <= 0 {
		t.Errorf("Expected backup size > 0, got %d", backup.Size)
	}

	// Wait a moment to ensure different timestamps
	time.Sleep(1100 * time.Millisecond)

	// Create another backup
	_, err = manager.CreateAgentBackup(
		agentID,
		"Scheduled backup",
		"automatic",
	)
	if err != nil {
		t.Fatalf("Failed to create second agent backup: %v", err)
	}

	// List backups
	backups, err := manager.ListAgentBackups(agentID)
	if err != nil {
		t.Fatalf("Failed to list agent backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}

	// Test restore (this is a simplified test since actual restore is complex)
	err = manager.RestoreAgentFromBackup(backup.BackupID)
	if err != nil {
		t.Fatalf("Failed to restore agent from backup: %v", err)
	}

	t.Logf("✅ Agent backup/restore test passed - created %d backups", len(backups))
}

func testHealthMonitoring(t *testing.T, manager *agent.EnhancedAgentManager) {
	agentID := "test-agent-3"

	// Perform health check
	health, err := manager.PerformHealthCheck(agentID)
	if err != nil {
		t.Fatalf("Failed to perform health check: %v", err)
	}

	// Verify health check properties
	if health.AgentID != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, health.AgentID)
	}
	if health.Status == "" {
		t.Error("Expected health status to be set")
	}
	if health.HealthScore < 0 || health.HealthScore > 100 {
		t.Errorf("Expected health score between 0-100, got %f", health.HealthScore)
	}

	// Perform another health check
	time.Sleep(100 * time.Millisecond) // Small delay to ensure different timestamps
	_, err = manager.PerformHealthCheck(agentID)
	if err != nil {
		t.Fatalf("Failed to perform second health check: %v", err)
	}

	// Get health history
	history, err := manager.GetAgentHealthHistory(agentID)
	if err != nil {
		t.Fatalf("Failed to get health history: %v", err)
	}

	if len(history) < 1 {
		t.Errorf("Expected at least 1 health check in history, got %d", len(history))
	}

	// Verify history is sorted by check time (newest first)
	if len(history) >= 2 {
		if history[0].LastHealthCheck.Before(history[1].LastHealthCheck) {
			t.Error("Expected health history to be sorted by check time (newest first)")
		}
	}

	t.Logf("✅ Health monitoring test passed - performed %d health checks", len(history))
}

func testPerformanceAnalytics(t *testing.T, manager *agent.EnhancedAgentManager) {
	agentID := "test-agent-4"

	// Generate analytics for different periods
	periods := []string{"1h", "24h", "7d", "30d"}

	for _, period := range periods {
		analytics, err := manager.GeneratePerformanceAnalytics(agentID, period)
		if err != nil {
			t.Fatalf("Failed to generate analytics for period %s: %v", period, err)
		}

		// Verify analytics properties
		if analytics.AgentID != agentID {
			t.Errorf("Expected agent ID %s, got %s", agentID, analytics.AgentID)
		}
		if analytics.AnalyticsPeriod != period {
			t.Errorf("Expected period %s, got %s", period, analytics.AnalyticsPeriod)
		}
		if analytics.TotalRequests <= 0 {
			t.Errorf("Expected total requests > 0, got %d", analytics.TotalRequests)
		}
		if analytics.SuccessRate < 0 || analytics.SuccessRate > 100 {
			t.Errorf("Expected success rate between 0-100, got %f", analytics.SuccessRate)
		}
		if analytics.AverageResponseTime <= 0 {
			t.Errorf("Expected average response time > 0, got %v", analytics.AverageResponseTime)
		}

		// Verify error distribution
		if len(analytics.ErrorDistribution) == 0 {
			t.Error("Expected error distribution to have entries")
		}

		// Verify resource utilization
		if len(analytics.ResourceUtilization) == 0 {
			t.Error("Expected resource utilization to have entries")
		}

		// Verify orchestration metrics
		if len(analytics.OrchestrationMetrics) == 0 {
			t.Error("Expected orchestration metrics to have entries")
		}
	}

	t.Logf("✅ Performance analytics test passed - generated analytics for %d periods", len(periods))
}

// Helper functions to create mock objects

func createMockRegistry(t *testing.T, tempDir string) *agent.AgentRegistry {
	// Create a simple mock registry
	// In a real test, this would be a proper mock or test double
	dbPath := filepath.Join(tempDir, "test.db")
	registry, err := agent.NewAgentRegistry(dbPath)
	if err != nil {
		t.Fatalf("Failed to create mock registry: %v", err)
	}

	// Add some test agents
	testAgents := []map[string]interface{}{
		{
			"id":          "test-agent-1",
			"agent_type":  "llm",
			"name":        "Test Agent 1",
			"description": "Test agent for versioning",
			"plugin_path": filepath.Join(tempDir, "test-agent-1.so"),
			"version":     "1.0.0",
		},
		{
			"id":          "test-agent-2",
			"agent_type":  "llm",
			"name":        "Test Agent 2",
			"description": "Test agent for backup/restore",
			"plugin_path": filepath.Join(tempDir, "test-agent-2.so"),
			"version":     "1.0.0",
		},
		{
			"id":          "test-agent-3",
			"agent_type":  "llm",
			"name":        "Test Agent 3",
			"description": "Test agent for health monitoring",
			"plugin_path": filepath.Join(tempDir, "test-agent-3.so"),
			"version":     "1.0.0",
		},
		{
			"id":          "test-agent-4",
			"agent_type":  "llm",
			"name":        "Test Agent 4",
			"description": "Test agent for performance analytics",
			"plugin_path": filepath.Join(tempDir, "test-agent-4.so"),
			"version":     "1.0.0",
		},
	}

	for _, agentConfig := range testAgents {
		// Create dummy plugin files
		pluginPath := agentConfig["plugin_path"].(string)
		os.WriteFile(pluginPath, []byte("dummy plugin content"), 0644)

		// Register agent
		registry.RegisterAgent(agentConfig["id"].(string), agentConfig)
	}

	return registry
}

func createMockBuilder(t *testing.T, tempDir string) *agent.AgentBuilder {
	// Create a simple mock builder
	// In a real test, this would be a proper mock or test double
	builder, err := agent.NewAgentBuilder(tempDir, "templates", "plugins")
	if err != nil {
		t.Fatalf("Failed to create mock builder: %v", err)
	}
	return builder
}
