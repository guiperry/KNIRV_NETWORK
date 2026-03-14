package systemhealth

import (
	"testing"

	"github.com/tidwall/buntdb"
)

func TestNewSystemHealthService(t *testing.T) {
	// Create temporary database
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.db != db {
		t.Error("Expected database to be set correctly")
	}

	if service.running {
		t.Error("Expected service to not be running initially")
	}

	if !service.monitoringEnabled {
		t.Error("Expected monitoring to be enabled initially")
	}

	if service.systemHealth == nil {
		t.Error("Expected system health to be initialized")
	}

	if service.alerts == nil {
		t.Error("Expected alerts slice to be initialized")
	}

	if service.alertThresholds == nil {
		t.Error("Expected alert thresholds to be initialized")
	}
}

func TestSystemHealthService_Start(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)

	// Test starting the service
	err = service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Error("Expected service to be running after start")
	}

	// Test starting already running service
	err = service.Start()
	if err == nil {
		t.Error("Expected error when starting already running service")
	}

	// Clean up
	service.Stop()
}

func TestSystemHealthService_Stop(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)

	// Test stopping non-running service
	err = service.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running service")
	}

	// Start and then stop
	service.Start()
	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Error("Expected service to not be running after stop")
	}
}

func TestSystemHealthService_IsRunning(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)

	// Initially not running
	if service.IsRunning() {
		t.Error("Expected service to not be running initially")
	}

	// Start service
	service.Start()
	if !service.IsRunning() {
		t.Error("Expected service to be running after start")
	}

	// Stop service
	service.Stop()
	if service.IsRunning() {
		t.Error("Expected service to not be running after stop")
	}
}

func TestSystemHealthService_GetSystemHealth(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)
	service.Start()
	defer service.Stop()

	// Test getting basic health
	health := service.GetSystemHealth(false)
	if health == nil {
		t.Fatal("Expected health to be returned")
	}

	if health.OverallStatus == "" {
		t.Error("Expected overall status to be set")
	}

	if health.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	// Test getting detailed health
	detailedHealth := service.GetSystemHealth(true)
	if detailedHealth == nil {
		t.Fatal("Expected detailed health to be returned")
	}

	if detailedHealth.Components == nil {
		t.Error("Expected components to be included in detailed health")
	}

	if detailedHealth.Metrics == nil {
		t.Error("Expected metrics to be included in detailed health")
	}
}

func TestSystemHealthService_AddAlert(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)
	service.Start()
	defer service.Stop()

	// Add an alert
	metadata := map[string]interface{}{
		"cpu_usage": 85.5,
	}
	alert := service.AddAlert("warning", "system", "High CPU usage detected", metadata)

	if alert == nil {
		t.Fatal("Expected alert to be created")
	}

	if alert.ID == "" {
		t.Error("Expected alert ID to be set")
	}

	if alert.Severity != "warning" {
		t.Error("Expected alert severity to be 'warning'")
	}

	if alert.Component != "system" {
		t.Error("Expected alert component to be 'system'")
	}

	if alert.Message != "High CPU usage detected" {
		t.Error("Expected alert message to match")
	}

	if alert.Resolved {
		t.Error("Expected alert to not be resolved initially")
	}

	if alert.Metadata["cpu_usage"] != 85.5 {
		t.Error("Expected alert metadata to be preserved")
	}

	// Check that alert was added to the service
	alerts := service.GetAlerts(nil, "")
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
}

func TestSystemHealthService_GetAlerts(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)
	service.Start()
	defer service.Stop()

	// Add some test alerts
	service.AddAlert("warning", "system", "Warning alert", nil)
	service.AddAlert("critical", "database", "Critical alert", nil)
	service.AddAlert("info", "network", "Info alert", nil)

	// Test getting all alerts
	allAlerts := service.GetAlerts(nil, "")
	if len(allAlerts) != 3 {
		t.Errorf("Expected 3 alerts, got %d", len(allAlerts))
	}

	// Test filtering by severity
	warningAlerts := service.GetAlerts(nil, "warning")
	if len(warningAlerts) != 1 {
		t.Errorf("Expected 1 warning alert, got %d", len(warningAlerts))
	}

	criticalAlerts := service.GetAlerts(nil, "critical")
	if len(criticalAlerts) != 1 {
		t.Errorf("Expected 1 critical alert, got %d", len(criticalAlerts))
	}

	// Test filtering by resolved status
	resolved := false
	unresolvedAlerts := service.GetAlerts(&resolved, "")
	if len(unresolvedAlerts) != 3 {
		t.Errorf("Expected 3 unresolved alerts, got %d", len(unresolvedAlerts))
	}
}

func TestSystemHealthService_ResolveAlert(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)
	service.Start()
	defer service.Stop()

	// Add an alert
	alert := service.AddAlert("warning", "system", "Test alert", nil)

	// Resolve the alert
	err = service.ResolveAlert(alert.ID)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Check that alert is resolved
	alerts := service.GetAlerts(nil, "")
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(alerts))
	}

	if !alerts[0].Resolved {
		t.Error("Expected alert to be resolved")
	}

	if alerts[0].ResolvedAt == "" {
		t.Error("Expected resolved timestamp to be set")
	}

	// Test resolving non-existent alert
	err = service.ResolveAlert("non-existent-id")
	if err == nil {
		t.Error("Expected error when resolving non-existent alert")
	}
}

func TestSystemHealthService_RunDiagnostics(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)
	service.Start()
	defer service.Stop()

	// Run diagnostics
	result := service.RunDiagnostics()

	if result == nil {
		t.Fatal("Expected diagnostics result to be returned")
	}

	if result.Timestamp == "" {
		t.Error("Expected diagnostics timestamp to be set")
	}

	if result.Summary == "" {
		t.Error("Expected diagnostics summary to be set")
	}

	if len(result.Tests) == 0 {
		t.Error("Expected diagnostics tests to be included")
	}

	// Check that database connectivity test is included
	foundDBTest := false
	for _, test := range result.Tests {
		if test.Name == "Database Connectivity" {
			foundDBTest = true
			break
		}
	}

	if !foundDBTest {
		t.Error("Expected database connectivity test to be included")
	}
}

func TestSystemHealthService_SetServiceReferences(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewSystemHealthService(db)

	// Mock service references
	mockDVEManager := &struct{}{}
	mockValidationCore := &struct{}{}
	mockInferenceService := &struct{}{}
	mockTEESecurityService := &struct{}{}

	// Set service references
	service.SetServiceReferences(
		mockDVEManager,
		mockValidationCore,
		mockInferenceService,
		mockTEESecurityService,
	)

	// Verify references are set (we can't directly access private fields,
	// but we can test that the service doesn't panic when using them)
	service.Start()
	health := service.GetSystemHealth(true)
	service.Stop()

	if health == nil {
		t.Error("Expected health to be returned after setting service references")
	}
}
