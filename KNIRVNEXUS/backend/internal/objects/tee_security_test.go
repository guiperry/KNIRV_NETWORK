package objects

import (
	"testing"
	"time"
)

func TestTEESecurityStatus_StructFields(t *testing.T) {
	threatAlert := &ThreatAlert{
		ID:          "threat-123",
		Type:        "memory_corruption",
		Severity:    "high",
		Description: "Potential memory corruption detected",
		DetectedAt:  "2024-09-16T10:00:00Z",
		Status:      "active",
	}

	securityAudit := &SecurityAudit{
		ID:        "audit-123",
		Timestamp: "2024-09-16T09:00:00Z",
		Type:      "attestation_check",
		Result:    "passed",
		Details:   "All attestations verified successfully",
	}

	performanceMetrics := &TEEPerformanceMetrics{
		AttestationLatency:      45.5,
		VerificationSuccessRate: 98.5,
		EnclaveUptime:           99.8,
		ThroughputOpsPerSecond:  1500.0,
		MemoryUtilization:       65.2,
		CPUUtilization:          42.3,
	}

	status := TEESecurityStatus{
		AttestationStatus:  "verified",
		EnclaveCount:       5,
		SecurityScore:      95.5,
		LastAudit:          "2024-09-16T09:00:00Z",
		ThreatsDetected:    1,
		ActiveThreats:      []*ThreatAlert{threatAlert},
		AuditHistory:       []*SecurityAudit{securityAudit},
		PerformanceMetrics: performanceMetrics,
		TEEType:            "SGX",
		LastAttestation:    "2024-09-16T10:30:00Z",
		MonitoringEnabled:  true,
	}

	if status.AttestationStatus != "verified" {
		t.Errorf("Expected AttestationStatus 'verified', got '%s'", status.AttestationStatus)
	}
	if status.LastAudit != "2024-09-16T09:00:00Z" {
		t.Errorf("Expected LastAudit '2024-09-16T09:00:00Z', got '%s'", status.LastAudit)
	}
	if len(status.AuditHistory) != 1 {
		t.Errorf("Expected 1 audit history entry, got %d", len(status.AuditHistory))
	}
	if status.PerformanceMetrics.AttestationLatency != 45.5 {
		t.Errorf("Expected AttestationLatency 45.5, got %f", status.PerformanceMetrics.AttestationLatency)
	}
	if status.LastAttestation != "2024-09-16T10:30:00Z" {
		t.Errorf("Expected LastAttestation '2024-09-16T10:30:00Z', got '%s'", status.LastAttestation)
	}
	if status.EnclaveCount != 5 {
		t.Errorf("Expected EnclaveCount 5, got %d", status.EnclaveCount)
	}
	if status.SecurityScore != 95.5 {
		t.Errorf("Expected SecurityScore 95.5, got %f", status.SecurityScore)
	}
	if status.ThreatsDetected != 1 {
		t.Errorf("Expected ThreatsDetected 1, got %d", status.ThreatsDetected)
	}
	if len(status.ActiveThreats) != 1 {
		t.Errorf("Expected 1 active threat, got %d", len(status.ActiveThreats))
	}
	if !status.MonitoringEnabled {
		t.Error("Expected MonitoringEnabled to be true")
	}
	if status.TEEType != "SGX" {
		t.Errorf("Expected TEEType 'SGX', got '%s'", status.TEEType)
	}
}

func TestThreatAlert_StructFields(t *testing.T) {
	threat := ThreatAlert{
		ID:          "threat-456",
		Type:        "side_channel_attack",
		Severity:    "critical",
		Description: "Potential side-channel attack detected on enclave",
		DetectedAt:  "2024-09-16T11:15:00Z",
		Status:      "investigating",
	}

	if threat.ID != "threat-456" {
		t.Errorf("Expected ID 'threat-456', got '%s'", threat.ID)
	}
	if threat.Description != "Potential side-channel attack detected on enclave" {
		t.Errorf("Expected Description 'Potential side-channel attack detected on enclave', got '%s'", threat.Description)
	}
	if threat.DetectedAt != "2024-09-16T11:15:00Z" {
		t.Errorf("Expected DetectedAt '2024-09-16T11:15:00Z', got '%s'", threat.DetectedAt)
	}
	if threat.Type != "side_channel_attack" {
		t.Errorf("Expected Type 'side_channel_attack', got '%s'", threat.Type)
	}
	if threat.Severity != "critical" {
		t.Errorf("Expected Severity 'critical', got '%s'", threat.Severity)
	}
	if threat.Status != "investigating" {
		t.Errorf("Expected Status 'investigating', got '%s'", threat.Status)
	}
}

func TestSecurityAudit_StructFields(t *testing.T) {
	audit := SecurityAudit{
		ID:        "audit-456",
		Timestamp: "2024-09-16T12:00:00Z",
		Type:      "enclave_integrity_check",
		Result:    "failed",
		Details:   "Enclave measurement mismatch detected",
	}

	if audit.ID != "audit-456" {
		t.Errorf("Expected ID 'audit-456', got '%s'", audit.ID)
	}
	if audit.Timestamp != "2024-09-16T12:00:00Z" {
		t.Errorf("Expected Timestamp '2024-09-16T12:00:00Z', got '%s'", audit.Timestamp)
	}
	if audit.Details != "Enclave measurement mismatch detected" {
		t.Errorf("Expected Details 'Enclave measurement mismatch detected', got '%s'", audit.Details)
	}
	if audit.Type != "enclave_integrity_check" {
		t.Errorf("Expected Type 'enclave_integrity_check', got '%s'", audit.Type)
	}
	if audit.Result != "failed" {
		t.Errorf("Expected Result 'failed', got '%s'", audit.Result)
	}
}

func TestTEEPerformanceMetrics_StructFields(t *testing.T) {
	metrics := TEEPerformanceMetrics{
		AttestationLatency:      75.2,
		VerificationSuccessRate: 97.8,
		EnclaveUptime:           99.5,
		ThroughputOpsPerSecond:  2500.0,
		MemoryUtilization:       72.1,
		CPUUtilization:          55.8,
	}

	if metrics.AttestationLatency != 75.2 {
		t.Errorf("Expected AttestationLatency 75.2, got %f", metrics.AttestationLatency)
	}
	if metrics.VerificationSuccessRate != 97.8 {
		t.Errorf("Expected VerificationSuccessRate 97.8, got %f", metrics.VerificationSuccessRate)
	}
	if metrics.EnclaveUptime != 99.5 {
		t.Errorf("Expected EnclaveUptime 99.5, got %f", metrics.EnclaveUptime)
	}
	if metrics.ThroughputOpsPerSecond != 2500.0 {
		t.Errorf("Expected ThroughputOpsPerSecond 2500.0, got %f", metrics.ThroughputOpsPerSecond)
	}
	if metrics.MemoryUtilization != 72.1 {
		t.Errorf("Expected MemoryUtilization 72.1, got %f", metrics.MemoryUtilization)
	}
	if metrics.CPUUtilization != 55.8 {
		t.Errorf("Expected CPUUtilization 55.8, got %f", metrics.CPUUtilization)
	}
}

func TestTEESecurityMetrics_StructFields(t *testing.T) {
	metrics := TEESecurityMetrics{
		AttestationStatus:   "verified",
		SecurityScore:       92.3,
		ThreatsDetected:     3,
		LastAudit:           "2024-09-16T08:00:00Z",
		ActiveAttestations:  10,
		ExpiredAttestations: 2,
		FailedVerifications: 1,
	}

	if metrics.AttestationStatus != "verified" {
		t.Errorf("Expected AttestationStatus 'verified', got '%s'", metrics.AttestationStatus)
	}
	if metrics.SecurityScore != 92.3 {
		t.Errorf("Expected SecurityScore 92.3, got %f", metrics.SecurityScore)
	}
	if metrics.ThreatsDetected != 3 {
		t.Errorf("Expected ThreatsDetected 3, got %d", metrics.ThreatsDetected)
	}
	if metrics.LastAudit != "2024-09-16T08:00:00Z" {
		t.Errorf("Expected LastAudit '2024-09-16T08:00:00Z', got '%s'", metrics.LastAudit)
	}
	if metrics.ActiveAttestations != 10 {
		t.Errorf("Expected ActiveAttestations 10, got %d", metrics.ActiveAttestations)
	}
	if metrics.ExpiredAttestations != 2 {
		t.Errorf("Expected ExpiredAttestations 2, got %d", metrics.ExpiredAttestations)
	}
	if metrics.FailedVerifications != 1 {
		t.Errorf("Expected FailedVerifications 1, got %d", metrics.FailedVerifications)
	}
}

func TestTEESecurityUpdate_StructFields(t *testing.T) {
	update := TEESecurityUpdate{
		AttestationStatus: "pending",
		SecurityScore:     88.7,
		ThreatsDetected:   2,
		LastAudit:         "2024-09-16T07:30:00Z",
	}

	if update.AttestationStatus != "pending" {
		t.Errorf("Expected AttestationStatus 'pending', got '%s'", update.AttestationStatus)
	}
	if update.SecurityScore != 88.7 {
		t.Errorf("Expected SecurityScore 88.7, got %f", update.SecurityScore)
	}
	if update.ThreatsDetected != 2 {
		t.Errorf("Expected ThreatsDetected 2, got %d", update.ThreatsDetected)
	}
	if update.LastAudit != "2024-09-16T07:30:00Z" {
		t.Errorf("Expected LastAudit '2024-09-16T07:30:00Z', got '%s'", update.LastAudit)
	}
}

func TestTEESecurityAction_StructFields(t *testing.T) {
	action := TEESecurityAction{
		Action:     "force_attestation",
		Parameters: map[string]interface{}{"enclave_id": "enclave-123", "timeout": 30},
	}

	if action.Action != "force_attestation" {
		t.Errorf("Expected Action 'force_attestation', got '%s'", action.Action)
	}
	if action.Parameters["enclave_id"] != "enclave-123" {
		t.Errorf("Expected enclave_id 'enclave-123', got %v", action.Parameters["enclave_id"])
	}
	if action.Parameters["timeout"] != 30 {
		t.Errorf("Expected timeout 30, got %v", action.Parameters["timeout"])
	}
}

func TestTEEEnclave_StructFields(t *testing.T) {
	now := time.Now()
	lastActivity := now.Add(-time.Minute * 5)

	enclave := TEEEnclave{
		ID:           "enclave-789",
		Name:         "ML Inference Enclave",
		Status:       "running",
		TEEType:      "SGX",
		CreatedAt:    now,
		LastActivity: lastActivity,
		MemoryUsage:  1073741824, // 1GB in bytes
		CPUUsage:     65.4,
		Measurements: []string{"measurement1", "measurement2"},
	}

	if enclave.ID != "enclave-789" {
		t.Errorf("Expected ID 'enclave-789', got '%s'", enclave.ID)
	}
	if enclave.TEEType != "SGX" {
		t.Errorf("Expected TEEType 'SGX', got '%s'", enclave.TEEType)
	}
	if enclave.CreatedAt != now {
		t.Errorf("Expected CreatedAt %v, got %v", now, enclave.CreatedAt)
	}
	if enclave.LastActivity != lastActivity {
		t.Errorf("Expected LastActivity %v, got %v", lastActivity, enclave.LastActivity)
	}
	if enclave.Name != "ML Inference Enclave" {
		t.Errorf("Expected Name 'ML Inference Enclave', got '%s'", enclave.Name)
	}
	if enclave.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", enclave.Status)
	}
	if enclave.MemoryUsage != 1073741824 {
		t.Errorf("Expected MemoryUsage 1073741824, got %d", enclave.MemoryUsage)
	}
	if enclave.CPUUsage != 65.4 {
		t.Errorf("Expected CPUUsage 65.4, got %f", enclave.CPUUsage)
	}
	if len(enclave.Measurements) != 2 {
		t.Errorf("Expected 2 measurements, got %d", len(enclave.Measurements))
	}
}

func TestTEESecurityConfig_StructFields(t *testing.T) {
	config := TEESecurityConfig{
		AttestationInterval: time.Minute * 5,
		ScanInterval:        time.Minute * 10,
		MonitoringEnabled:   true,
		TEEType:             "TrustZone",
		RequireAttestation:  true,
		SecurityThreshold:   85.0,
	}

	if config.AttestationInterval != time.Minute*5 {
		t.Errorf("Expected AttestationInterval 5m, got %v", config.AttestationInterval)
	}
	if config.ScanInterval != time.Minute*10 {
		t.Errorf("Expected ScanInterval 10m, got %v", config.ScanInterval)
	}
	if !config.MonitoringEnabled {
		t.Error("Expected MonitoringEnabled to be true")
	}
	if config.TEEType != "TrustZone" {
		t.Errorf("Expected TEEType 'TrustZone', got '%s'", config.TEEType)
	}
	if !config.RequireAttestation {
		t.Error("Expected RequireAttestation to be true")
	}
	if config.SecurityThreshold != 85.0 {
		t.Errorf("Expected SecurityThreshold 85.0, got %f", config.SecurityThreshold)
	}
}

func TestTEESecurityEvent_StructFields(t *testing.T) {
	now := time.Now()

	event := TEESecurityEvent{
		ID:        "event-123",
		Type:      "attestation_failure",
		Severity:  "high",
		Message:   "Attestation verification failed for enclave",
		Timestamp: now,
		NodeID:    "node-456",
		EnclaveID: "enclave-789",
		Metadata:  map[string]interface{}{"error_code": 500, "retry_count": 3},
	}

	if event.ID != "event-123" {
		t.Errorf("Expected ID 'event-123', got '%s'", event.ID)
	}
	if event.Message != "Attestation verification failed for enclave" {
		t.Errorf("Expected Message 'Attestation verification failed for enclave', got '%s'", event.Message)
	}
	if event.Timestamp != now {
		t.Errorf("Expected Timestamp %v, got %v", now, event.Timestamp)
	}
	if event.Type != "attestation_failure" {
		t.Errorf("Expected Type 'attestation_failure', got '%s'", event.Type)
	}
	if event.Severity != "high" {
		t.Errorf("Expected Severity 'high', got '%s'", event.Severity)
	}
	if event.NodeID != "node-456" {
		t.Errorf("Expected NodeID 'node-456', got '%s'", event.NodeID)
	}
	if event.EnclaveID != "enclave-789" {
		t.Errorf("Expected EnclaveID 'enclave-789', got '%s'", event.EnclaveID)
	}
	if event.Metadata["error_code"] != 500 {
		t.Errorf("Expected error_code 500, got %v", event.Metadata["error_code"])
	}
}

func TestTEESecurityReport_StructFields(t *testing.T) {
	now := time.Now()

	threatAlert := &ThreatAlert{
		ID:       "threat-report-1",
		Type:     "timing_attack",
		Severity: "medium",
		Status:   "active",
	}

	securityEvent := &TEESecurityEvent{
		ID:        "event-report-1",
		Type:      "security_scan",
		Severity:  "info",
		Message:   "Security scan completed",
		Timestamp: now,
	}

	performanceMetrics := &TEEPerformanceMetrics{
		AttestationLatency:      50.0,
		VerificationSuccessRate: 99.2,
		EnclaveUptime:           99.9,
	}

	report := TEESecurityReport{
		ID:                 "report-123",
		GeneratedAt:        now,
		ReportPeriod:       "24h",
		OverallScore:       94.5,
		AttestationStatus:  "verified",
		TotalThreats:       5,
		ResolvedThreats:    4,
		ActiveThreats:      []*ThreatAlert{threatAlert},
		SecurityEvents:     []*TEESecurityEvent{securityEvent},
		PerformanceMetrics: performanceMetrics,
		Recommendations:    []string{"Update enclave firmware", "Increase monitoring frequency"},
	}

	if report.ID != "report-123" {
		t.Errorf("Expected ID 'report-123', got '%s'", report.ID)
	}
	if report.GeneratedAt != now {
		t.Errorf("Expected GeneratedAt %v, got %v", now, report.GeneratedAt)
	}
	if report.AttestationStatus != "verified" {
		t.Errorf("Expected AttestationStatus 'verified', got '%s'", report.AttestationStatus)
	}
	if report.PerformanceMetrics.AttestationLatency != 50.0 {
		t.Errorf("Expected AttestationLatency 50.0, got %f", report.PerformanceMetrics.AttestationLatency)
	}
	if report.ReportPeriod != "24h" {
		t.Errorf("Expected ReportPeriod '24h', got '%s'", report.ReportPeriod)
	}
	if report.OverallScore != 94.5 {
		t.Errorf("Expected OverallScore 94.5, got %f", report.OverallScore)
	}
	if report.TotalThreats != 5 {
		t.Errorf("Expected TotalThreats 5, got %d", report.TotalThreats)
	}
	if report.ResolvedThreats != 4 {
		t.Errorf("Expected ResolvedThreats 4, got %d", report.ResolvedThreats)
	}
	if len(report.ActiveThreats) != 1 {
		t.Errorf("Expected 1 active threat, got %d", len(report.ActiveThreats))
	}
	if len(report.SecurityEvents) != 1 {
		t.Errorf("Expected 1 security event, got %d", len(report.SecurityEvents))
	}
	if len(report.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(report.Recommendations))
	}
}

func TestTEESecurityFilter_StructFields(t *testing.T) {
	dateFrom := time.Now().Add(-24 * time.Hour)
	dateTo := time.Now()

	filter := TEESecurityFilter{
		Status:   "active",
		TEEType:  "SGX",
		Severity: "high",
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    100,
	}

	if filter.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", filter.Status)
	}
	if filter.DateFrom == nil {
		t.Error("Expected DateFrom to be set")
	}
	if filter.DateTo == nil {
		t.Error("Expected DateTo to be set")
	}
	if filter.TEEType != "SGX" {
		t.Errorf("Expected TEEType 'SGX', got '%s'", filter.TEEType)
	}
	if filter.Severity != "high" {
		t.Errorf("Expected Severity 'high', got '%s'", filter.Severity)
	}
	if filter.Limit != 100 {
		t.Errorf("Expected Limit 100, got %d", filter.Limit)
	}
}

func TestThreatAlert_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Active threat", "active", true},
		{"Investigating threat", "investigating", true},
		{"Resolved threat", "resolved", false},
		{"Unknown status", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threat := ThreatAlert{Status: tt.status}
			if got := threat.IsActive(); got != tt.expected {
				t.Errorf("ThreatAlert.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestThreatAlert_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		expected bool
	}{
		{"Critical threat", "critical", true},
		{"High threat", "high", false},
		{"Medium threat", "medium", false},
		{"Low threat", "low", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threat := ThreatAlert{Severity: tt.severity}
			if got := threat.IsCritical(); got != tt.expected {
				t.Errorf("ThreatAlert.IsCritical() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSecurityAudit_IsPassed(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected bool
	}{
		{"Passed audit", "passed", true},
		{"Failed audit", "failed", false},
		{"Warning audit", "warning", false},
		{"Info audit", "info", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audit := SecurityAudit{Result: tt.result}
			if got := audit.IsPassed(); got != tt.expected {
				t.Errorf("SecurityAudit.IsPassed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSecurityAudit_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected bool
	}{
		{"Failed audit", "failed", true},
		{"Passed audit", "passed", false},
		{"Warning audit", "warning", false},
		{"Info audit", "info", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audit := SecurityAudit{Result: tt.result}
			if got := audit.IsFailed(); got != tt.expected {
				t.Errorf("SecurityAudit.IsFailed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTEESecurityStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		name              string
		securityScore     float64
		attestationStatus string
		activeThreats     []*ThreatAlert
		expected          bool
	}{
		{
			name:              "Healthy status",
			securityScore:     95.0,
			attestationStatus: "verified",
			activeThreats:     []*ThreatAlert{},
			expected:          true,
		},
		{
			name:              "Low security score",
			securityScore:     85.0,
			attestationStatus: "verified",
			activeThreats:     []*ThreatAlert{},
			expected:          false,
		},
		{
			name:              "Unverified attestation",
			securityScore:     95.0,
			attestationStatus: "pending",
			activeThreats:     []*ThreatAlert{},
			expected:          false,
		},
		{
			name:              "Has active threats",
			securityScore:     95.0,
			attestationStatus: "verified",
			activeThreats:     []*ThreatAlert{{ID: "threat-1", Status: "active"}},
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := TEESecurityStatus{
				SecurityScore:     tt.securityScore,
				AttestationStatus: tt.attestationStatus,
				ActiveThreats:     tt.activeThreats,
			}
			if got := status.IsHealthy(); got != tt.expected {
				t.Errorf("TEESecurityStatus.IsHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTEESecurityStatus_HasCriticalThreats(t *testing.T) {
	tests := []struct {
		name          string
		activeThreats []*ThreatAlert
		expected      bool
	}{
		{
			name:          "No threats",
			activeThreats: []*ThreatAlert{},
			expected:      false,
		},
		{
			name: "Has critical threat",
			activeThreats: []*ThreatAlert{
				{ID: "threat-1", Severity: "high"},
				{ID: "threat-2", Severity: "critical"},
			},
			expected: true,
		},
		{
			name: "No critical threats",
			activeThreats: []*ThreatAlert{
				{ID: "threat-1", Severity: "high"},
				{ID: "threat-2", Severity: "medium"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := TEESecurityStatus{ActiveThreats: tt.activeThreats}
			if got := status.HasCriticalThreats(); got != tt.expected {
				t.Errorf("TEESecurityStatus.HasCriticalThreats() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTEEPerformanceMetrics_IsPerformant(t *testing.T) {
	tests := []struct {
		name                    string
		attestationLatency      float64
		verificationSuccessRate float64
		enclaveUptime           float64
		expected                bool
	}{
		{
			name:                    "Performant metrics",
			attestationLatency:      50.0,
			verificationSuccessRate: 98.0,
			enclaveUptime:           99.5,
			expected:                true,
		},
		{
			name:                    "High latency",
			attestationLatency:      150.0,
			verificationSuccessRate: 98.0,
			enclaveUptime:           99.5,
			expected:                false,
		},
		{
			name:                    "Low success rate",
			attestationLatency:      50.0,
			verificationSuccessRate: 90.0,
			enclaveUptime:           99.5,
			expected:                false,
		},
		{
			name:                    "Low uptime",
			attestationLatency:      50.0,
			verificationSuccessRate: 98.0,
			enclaveUptime:           95.0,
			expected:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := TEEPerformanceMetrics{
				AttestationLatency:      tt.attestationLatency,
				VerificationSuccessRate: tt.verificationSuccessRate,
				EnclaveUptime:           tt.enclaveUptime,
			}
			if got := metrics.IsPerformant(); got != tt.expected {
				t.Errorf("TEEPerformanceMetrics.IsPerformant() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTEEEnclave_IsRunning(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Running enclave", "running", true},
		{"Stopped enclave", "stopped", false},
		{"Error enclave", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enclave := TEEEnclave{Status: tt.status}
			if got := enclave.IsRunning(); got != tt.expected {
				t.Errorf("TEEEnclave.IsRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTEEEnclave_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		cpuUsage float64
		expected bool
	}{
		{"Healthy running enclave", "running", 50.0, true},
		{"High CPU running enclave", "running", 85.0, false},
		{"Stopped enclave", "stopped", 50.0, false},
		{"Error enclave", "error", 50.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enclave := TEEEnclave{
				Status:   tt.status,
				CPUUsage: tt.cpuUsage,
			}
			if got := enclave.IsHealthy(); got != tt.expected {
				t.Errorf("TEEEnclave.IsHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}
