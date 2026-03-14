package teesecurity

import (
	"testing"
	"time"

	"github.com/tidwall/buntdb"
)

func TestNewTEESecurityService(t *testing.T) {
	// Create in-memory database for testing
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	if service == nil {
		t.Fatal("TEE security service should not be nil")
	}

	if service.kaliProfile == nil {
		t.Error("Kali profile should not be nil")
	}

	if service.runtimeManager == nil {
		t.Error("Runtime manager should not be nil")
	}

	if service.db != db {
		t.Error("Database should be set correctly")
	}
}

func TestTEESecurityService_GetKaliProfile(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	profile := service.GetKaliProfile()
	if profile == nil {
		t.Error("Kali profile should not be nil")
	}

	if profile.OS == "" {
		t.Error("OS should be detected")
	}
}

func TestTEESecurityService_GetRuntimeManager(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	runtimeManager := service.GetRuntimeManager()
	if runtimeManager == nil {
		t.Error("Runtime manager should not be nil")
	}
}

func TestTEESecurityService_Start(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	err = service.Start()
	if err != nil {
		t.Fatalf("Failed to start TEE security service: %v", err)
	}
}

func TestTEESecurityService_Stop(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop TEE security service: %v", err)
	}
}

func TestTEESecurityService_IsRunning(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	// Service should be running after initialization
	if !service.IsRunning() {
		t.Error("Service should be running after initialization")
	}
}

func TestTEESecurityService_GetSecurityStatus(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	status := service.GetSecurityStatus()
	if status == nil {
		t.Fatal("Security status should not be nil")
	}

	if status.AttestationStatus != "verified" {
		t.Errorf("Expected attestation status 'verified', got '%s'", status.AttestationStatus)
	}

	if status.EnclaveCount != 1 {
		t.Errorf("Expected enclave count 1, got %d", status.EnclaveCount)
	}

	if status.SecurityScore != 95.0 {
		t.Errorf("Expected security score 95.0, got %.1f", status.SecurityScore)
	}

	if status.ThreatsDetected != 0 {
		t.Errorf("Expected threats detected 0, got %d", status.ThreatsDetected)
	}

	if status.ActiveThreats == nil {
		t.Error("Active threats should not be nil")
	}

	if status.AuditHistory == nil {
		t.Error("Audit history should not be nil")
	}

	if status.PerformanceMetrics == nil {
		t.Error("Performance metrics should not be nil")
	}

	if !status.MonitoringEnabled {
		t.Error("Monitoring should be enabled")
	}
}

func TestTEESecurityService_RunSecurityScan(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	err = service.RunSecurityScan()
	if err != nil {
		t.Fatalf("Failed to run security scan: %v", err)
	}
}

func TestTEESecurityService_PerformAttestation(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	err = service.PerformAttestation()
	if err != nil {
		t.Fatalf("Failed to perform attestation: %v", err)
	}
}

func TestTEESecurityService_UpdateAttestationStatus(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	newStatus := "attested"
	err = service.UpdateAttestationStatus(newStatus)
	if err != nil {
		t.Fatalf("Failed to update attestation status: %v", err)
	}
}

func TestTEESecurityService_ResolveThreat(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	threatID := "threat_123"
	err = service.ResolveThreat(threatID)
	if err != nil {
		t.Fatalf("Failed to resolve threat: %v", err)
	}
}

func TestTEESecurityService_StoreKaliProfile(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	// Verify profile was stored
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("tee:kali_profile")
		if err != nil {
			return err
		}
		if val == "" {
			t.Error("Kali profile should be stored in database")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to verify Kali profile storage: %v", err)
	}

	// Use service to avoid unused variable error
	_ = service
}

func TestTEESecurityService_SecurityStatusFields(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	status := service.GetSecurityStatus()

	// Test LastAudit field
	if status.LastAudit == "" {
		t.Error("LastAudit should not be empty")
	}

	// Test TEEType field
	if status.TEEType == "" {
		t.Error("TEEType should not be empty")
	}

	// Test LastAttestation field
	if status.LastAttestation == "" {
		t.Error("LastAttestation should not be empty")
	}

	// Test PerformanceMetrics fields
	metrics := status.PerformanceMetrics
	if metrics.AttestationLatency <= 0 {
		t.Error("AttestationLatency should be positive")
	}

	if metrics.VerificationSuccessRate <= 0 {
		t.Error("VerificationSuccessRate should be positive")
	}

	if metrics.EnclaveUptime <= 0 {
		t.Error("EnclaveUptime should be positive")
	}

	if metrics.ThroughputOpsPerSecond <= 0 {
		t.Error("ThroughputOpsPerSecond should be positive")
	}

	if metrics.MemoryUtilization < 0 {
		t.Error("MemoryUtilization should not be negative")
	}

	if metrics.CPUUtilization < 0 {
		t.Error("CPUUtilization should not be negative")
	}
}

func TestTEESecurityService_MultipleOperations(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	// Test multiple operations in sequence
	err = service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	err = service.RunSecurityScan()
	if err != nil {
		t.Fatalf("Failed to run security scan: %v", err)
	}

	err = service.PerformAttestation()
	if err != nil {
		t.Fatalf("Failed to perform attestation: %v", err)
	}

	err = service.UpdateAttestationStatus("verified")
	if err != nil {
		t.Fatalf("Failed to update attestation status: %v", err)
	}

	err = service.ResolveThreat("test_threat")
	if err != nil {
		t.Fatalf("Failed to resolve threat: %v", err)
	}

	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	// Service should still be considered running for compatibility
	if !service.IsRunning() {
		t.Error("Service should still be running after operations")
	}
}

func TestTEESecurityService_StatusConsistency(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	service, err := NewTEESecurityService(db)
	if err != nil {
		t.Fatalf("Failed to create TEE security service: %v", err)
	}

	// Get status multiple times and verify consistency
	status1 := service.GetSecurityStatus()
	time.Sleep(10 * time.Millisecond) // Small delay
	status2 := service.GetSecurityStatus()

	if status1.AttestationStatus != status2.AttestationStatus {
		t.Error("Attestation status should be consistent")
	}

	if status1.EnclaveCount != status2.EnclaveCount {
		t.Error("Enclave count should be consistent")
	}

	if status1.SecurityScore != status2.SecurityScore {
		t.Error("Security score should be consistent")
	}

	// Test that service is used
	if !service.IsRunning() {
		t.Error("Service should be running")
	}

	// Use service variable to avoid unused variable error
	_ = service
}