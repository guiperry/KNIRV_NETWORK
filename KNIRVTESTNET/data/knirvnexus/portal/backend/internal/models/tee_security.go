package models

import "time"

// TEESecurityStatus represents the overall TEE security status
type TEESecurityStatus struct {
	AttestationStatus  string                 `json:"attestation_status"`
	EnclaveCount       int                    `json:"enclave_count"`
	SecurityScore      float64                `json:"security_score"`
	LastAudit          string                 `json:"last_audit"`
	ThreatsDetected    int                    `json:"threats_detected"`
	ActiveThreats      []*ThreatAlert         `json:"active_threats"`
	AuditHistory       []*SecurityAudit       `json:"audit_history"`
	PerformanceMetrics *TEEPerformanceMetrics `json:"performance_metrics"`
	TEEType            string                 `json:"tee_type"`
	LastAttestation    string                 `json:"last_attestation"`
	MonitoringEnabled  bool                   `json:"monitoring_enabled"`
}

// ThreatAlert represents a security threat
type ThreatAlert struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Severity    string `json:"severity"` // low, medium, high, critical
	Description string `json:"description"`
	DetectedAt  string `json:"detected_at"`
	Status      string `json:"status"` // active, investigating, resolved
}

// SecurityAudit represents a security audit record
type SecurityAudit struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Result    string `json:"result"` // passed, failed, warning, info
	Details   string `json:"details"`
}

// TEEPerformanceMetrics represents TEE performance metrics
type TEEPerformanceMetrics struct {
	AttestationLatency      float64 `json:"attestation_latency"`       // milliseconds
	VerificationSuccessRate float64 `json:"verification_success_rate"` // percentage
	EnclaveUptime           float64 `json:"enclave_uptime"`            // percentage
	ThroughputOpsPerSecond  float64 `json:"throughput_ops_per_second"` // operations per second
	MemoryUtilization       float64 `json:"memory_utilization"`        // percentage
	CPUUtilization          float64 `json:"cpu_utilization"`           // percentage
}

// TEESecurityMetrics represents aggregated security metrics
type TEESecurityMetrics struct {
	AttestationStatus   string  `json:"attestation_status"`
	SecurityScore       float64 `json:"security_score"`
	ThreatsDetected     int     `json:"threats_detected"`
	LastAudit           string  `json:"last_audit"`
	ActiveAttestations  int     `json:"active_attestations"`
	ExpiredAttestations int     `json:"expired_attestations"`
	FailedVerifications int     `json:"failed_verifications"`
}

// TEESecurityUpdate represents a WebSocket update for TEE security
type TEESecurityUpdate struct {
	AttestationStatus string  `json:"attestation_status"`
	SecurityScore     float64 `json:"security_score"`
	ThreatsDetected   int     `json:"threats_detected"`
	LastAudit         string  `json:"last_audit"`
}

// TEESecurityAction represents an action that can be performed on TEE security
type TEESecurityAction struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// TEEEnclave represents a TEE enclave
type TEEEnclave struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // running, stopped, error
	TEEType      string    `json:"tee_type"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	MemoryUsage  int64     `json:"memory_usage"` // bytes
	CPUUsage     float64   `json:"cpu_usage"`    // percentage
	Measurements []string  `json:"measurements"`
}

// TEESecurityConfig represents TEE security configuration
type TEESecurityConfig struct {
	AttestationInterval time.Duration `json:"attestation_interval"`
	ScanInterval        time.Duration `json:"scan_interval"`
	MonitoringEnabled   bool          `json:"monitoring_enabled"`
	TEEType             string        `json:"tee_type"`
	RequireAttestation  bool          `json:"require_attestation"`
	SecurityThreshold   float64       `json:"security_threshold"`
}

// TEESecurityEvent represents a security event
type TEESecurityEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	NodeID    string                 `json:"node_id,omitempty"`
	EnclaveID string                 `json:"enclave_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TEESecurityReport represents a comprehensive security report
type TEESecurityReport struct {
	ID                 string                 `json:"id"`
	GeneratedAt        time.Time              `json:"generated_at"`
	ReportPeriod       string                 `json:"report_period"`
	OverallScore       float64                `json:"overall_score"`
	AttestationStatus  string                 `json:"attestation_status"`
	TotalThreats       int                    `json:"total_threats"`
	ResolvedThreats    int                    `json:"resolved_threats"`
	ActiveThreats      []*ThreatAlert         `json:"active_threats"`
	SecurityEvents     []*TEESecurityEvent    `json:"security_events"`
	PerformanceMetrics *TEEPerformanceMetrics `json:"performance_metrics"`
	Recommendations    []string               `json:"recommendations"`
}

// TEESecurityFilter represents filters for querying TEE security data
type TEESecurityFilter struct {
	Status   string     `json:"status,omitempty"`
	TEEType  string     `json:"tee_type,omitempty"`
	Severity string     `json:"severity,omitempty"`
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	Limit    int        `json:"limit,omitempty"`
}

// Validation methods
func (ta *ThreatAlert) IsActive() bool {
	return ta.Status == "active" || ta.Status == "investigating"
}

func (ta *ThreatAlert) IsCritical() bool {
	return ta.Severity == "critical"
}

func (sa *SecurityAudit) IsPassed() bool {
	return sa.Result == "passed"
}

func (sa *SecurityAudit) IsFailed() bool {
	return sa.Result == "failed"
}

func (tss *TEESecurityStatus) IsHealthy() bool {
	return tss.SecurityScore >= 90.0 &&
		tss.AttestationStatus == "verified" &&
		len(tss.ActiveThreats) == 0
}

func (tss *TEESecurityStatus) HasCriticalThreats() bool {
	for _, threat := range tss.ActiveThreats {
		if threat.IsCritical() {
			return true
		}
	}
	return false
}

func (tpm *TEEPerformanceMetrics) IsPerformant() bool {
	return tpm.AttestationLatency < 100.0 &&
		tpm.VerificationSuccessRate > 95.0 &&
		tpm.EnclaveUptime > 99.0
}

func (te *TEEEnclave) IsRunning() bool {
	return te.Status == "running"
}

func (te *TEEEnclave) IsHealthy() bool {
	return te.IsRunning() && te.CPUUsage < 80.0
}

// System Health models are now defined in dve.go
