package agentify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// TEESecurityLevel defines the security isolation level
type TEESecurityLevel int

const (
	SecurityLevelLow    TEESecurityLevel = iota // Process-based TEE
	SecurityLevelMedium                         // Container-based TEE
	SecurityLevelHigh                           // VM-based TEE
)

// TEESecurityManager manages TEE security across all agent plugins
type TEESecurityManager struct {
	activeTEEs       map[string]TEE
	securityPolicies map[string]SecurityPolicy
	auditLog         []TEESecurityEvent
	mutex            sync.RWMutex
	config           TEESecurityConfig
}

// TEESecurityConfig defines global TEE security configuration
type TEESecurityConfig struct {
	DefaultSecurityLevel    TEESecurityLevel `json:"default_security_level"`
	MaxConcurrentTEEs       int              `json:"max_concurrent_tees"`
	GlobalResourceLimits    ResourceLimits   `json:"global_resource_limits"`
	AuditLogRetentionDays   int              `json:"audit_log_retention_days"`
	IntegrityCheckInterval  time.Duration    `json:"integrity_check_interval"`
	EnableRuntimeMonitoring bool             `json:"enable_runtime_monitoring"`
	TrustedSignatures       []string         `json:"trusted_signatures"`
}

// TEESecurityEvent represents a security event in the TEE system
type TEESecurityEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	AgentID     string                 `json:"agent_id"`
	TEEType     string                 `json:"tee_type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Remediation string                 `json:"remediation,omitempty"`
}

// NewTEESecurityManager creates a new TEE security manager
func NewTEESecurityManager(config TEESecurityConfig) *TEESecurityManager {
	// Set default values
	if config.MaxConcurrentTEEs == 0 {
		config.MaxConcurrentTEEs = 50
	}
	if config.AuditLogRetentionDays == 0 {
		config.AuditLogRetentionDays = 30
	}
	if config.IntegrityCheckInterval == 0 {
		config.IntegrityCheckInterval = 5 * time.Minute
	}

	manager := &TEESecurityManager{
		activeTEEs:       make(map[string]TEE),
		securityPolicies: make(map[string]SecurityPolicy),
		auditLog:         make([]TEESecurityEvent, 0),
		config:           config,
	}

	// Start background monitoring if enabled
	if config.EnableRuntimeMonitoring {
		go manager.startRuntimeMonitoring()
	}

	return manager
}

// CreateSecureTEE creates a TEE with appropriate security level for an agent
func (m *TEESecurityManager) CreateSecureTEE(agentID string, securityLevel TEESecurityLevel, teeConfig TEEConfig) (TEE, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check concurrent TEE limit
	if len(m.activeTEEs) >= m.config.MaxConcurrentTEEs {
		return nil, fmt.Errorf("maximum concurrent TEEs (%d) reached", m.config.MaxConcurrentTEEs)
	}

	// Apply global resource limits
	if teeConfig.ResourceLimits.MemoryMB > m.config.GlobalResourceLimits.MemoryMB {
		teeConfig.ResourceLimits.MemoryMB = m.config.GlobalResourceLimits.MemoryMB
	}
	if teeConfig.ResourceLimits.CPUCores > m.config.GlobalResourceLimits.CPUCores {
		teeConfig.ResourceLimits.CPUCores = m.config.GlobalResourceLimits.CPUCores
	}

	// Create TEE based on security level
	var tee TEE
	var teeType string

	switch securityLevel {
	case SecurityLevelLow:
		tee = NewProcessTEE(teeConfig)
		teeType = "process"
	case SecurityLevelMedium:
		tee = NewContainerTEE(teeConfig)
		teeType = "container"
	case SecurityLevelHigh:
		tee = NewVMTEE(teeConfig)
		teeType = "vm"
	default:
		tee = NewProcessTEE(teeConfig)
		teeType = "process"
	}

	// Start the TEE
	if err := tee.Start(); err != nil {
		m.logSecurityEvent("tee_creation_failed", agentID, teeType, "error",
			fmt.Sprintf("Failed to start TEE: %v", err), map[string]interface{}{
				"security_level": securityLevel,
				"error":          err.Error(),
			})
		return nil, fmt.Errorf("failed to start TEE: %w", err)
	}

	// Register TEE
	m.activeTEEs[agentID] = tee
	m.securityPolicies[agentID] = teeConfig.SecurityPolicy

	m.logSecurityEvent("tee_created", agentID, teeType, "info",
		"TEE created successfully", map[string]interface{}{
			"security_level":  securityLevel,
			"resource_limits": teeConfig.ResourceLimits,
		})

	return tee, nil
}

// ValidateAgentExecution validates if an agent can execute a command
func (m *TEESecurityManager) ValidateAgentExecution(agentID, command string, args []string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tee, exists := m.activeTEEs[agentID]
	if !exists {
		return fmt.Errorf("no active TEE found for agent %s", agentID)
	}

	// Validate command through TEE
	if err := tee.ValidateCommand(command, args); err != nil {
		m.logSecurityEvent("command_blocked", agentID, "unknown", "warning",
			fmt.Sprintf("Command blocked: %s", command), map[string]interface{}{
				"command": command,
				"args":    args,
				"reason":  err.Error(),
			})
		return err
	}

	return nil
}

// MonitorTEEHealth monitors the health and security of all active TEEs
func (m *TEESecurityManager) MonitorTEEHealth() map[string]TEEHealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	healthStatus := make(map[string]TEEHealthStatus)

	for agentID, tee := range m.activeTEEs {
		status := TEEHealthStatus{
			AgentID:   agentID,
			Timestamp: time.Now(),
		}

		// Get resource usage
		if usage, err := tee.GetResourceUsage(); err == nil {
			status.ResourceUsage = usage
			status.IsHealthy = true

			// Check for resource violations
			policy := m.securityPolicies[agentID]
			if usage.MemoryMB > float64(m.config.GlobalResourceLimits.MemoryMB) {
				status.IsHealthy = false
				status.Issues = append(status.Issues, "Memory usage exceeds global limits")
			}
			if usage.CPUPercent > float64(m.config.GlobalResourceLimits.CPUCores)*100 {
				status.IsHealthy = false
				status.Issues = append(status.Issues, "CPU usage exceeds global limits")
			}

			// Check execution time limits
			if policy.MaxExecutionTime > 0 && time.Since(usage.Timestamp) > policy.MaxExecutionTime {
				status.IsHealthy = false
				status.Issues = append(status.Issues, "Execution time exceeds policy limits")
			}
		} else {
			status.IsHealthy = false
			status.Issues = append(status.Issues, fmt.Sprintf("Failed to get resource usage: %v", err))
		}

		healthStatus[agentID] = status

		// Log health issues
		if !status.IsHealthy {
			m.logSecurityEvent("tee_health_issue", agentID, "unknown", "warning",
				"TEE health check failed", map[string]interface{}{
					"issues":         status.Issues,
					"resource_usage": status.ResourceUsage,
				})
		}
	}

	return healthStatus
}

// TEEHealthStatus represents the health status of a TEE
type TEEHealthStatus struct {
	AgentID       string        `json:"agent_id"`
	IsHealthy     bool          `json:"is_healthy"`
	ResourceUsage ResourceUsage `json:"resource_usage"`
	Issues        []string      `json:"issues"`
	Timestamp     time.Time     `json:"timestamp"`
}

// DestroyTEE safely destroys a TEE and cleans up resources
func (m *TEESecurityManager) DestroyTEE(agentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	tee, exists := m.activeTEEs[agentID]
	if !exists {
		return fmt.Errorf("no active TEE found for agent %s", agentID)
	}

	// Stop the TEE
	if err := tee.Stop(); err != nil {
		m.logSecurityEvent("tee_destruction_failed", agentID, "unknown", "error",
			fmt.Sprintf("Failed to stop TEE: %v", err), map[string]interface{}{
				"error": err.Error(),
			})
		return fmt.Errorf("failed to stop TEE: %w", err)
	}

	// Clean up
	delete(m.activeTEEs, agentID)
	delete(m.securityPolicies, agentID)

	m.logSecurityEvent("tee_destroyed", agentID, "unknown", "info",
		"TEE destroyed successfully", nil)

	return nil
}

// GetSecurityAuditLog returns the security audit log
func (m *TEESecurityManager) GetSecurityAuditLog() []TEESecurityEvent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modification
	auditCopy := make([]TEESecurityEvent, len(m.auditLog))
	copy(auditCopy, m.auditLog)
	return auditCopy
}

// logSecurityEvent logs a security event
func (m *TEESecurityManager) logSecurityEvent(eventType, agentID, teeType, severity, message string, details map[string]interface{}) {
	event := TEESecurityEvent{
		ID:        generateEventID(),
		Type:      eventType,
		AgentID:   agentID,
		TEEType:   teeType,
		Severity:  severity,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}

	m.auditLog = append(m.auditLog, event)

	// Log to system log as well
	log.Printf("TEE Security Event [%s]: %s - %s", severity, eventType, message)

	// Clean up old audit logs
	m.cleanupAuditLog()
}

// generateEventID generates a unique event ID
func generateEventID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:8])
}

// cleanupAuditLog removes old audit log entries
func (m *TEESecurityManager) cleanupAuditLog() {
	cutoff := time.Now().AddDate(0, 0, -m.config.AuditLogRetentionDays)

	var filteredLog []TEESecurityEvent
	for _, event := range m.auditLog {
		if event.Timestamp.After(cutoff) {
			filteredLog = append(filteredLog, event)
		}
	}

	m.auditLog = filteredLog
}

// startRuntimeMonitoring starts background runtime monitoring
func (m *TEESecurityManager) startRuntimeMonitoring() {
	ticker := time.NewTicker(m.config.IntegrityCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		healthStatus := m.MonitorTEEHealth()

		// Process health status and take action if needed
		for agentID, status := range healthStatus {
			if !status.IsHealthy {
				log.Printf("TEE Security: Health issues detected for agent %s: %v", agentID, status.Issues)

				// Could implement automatic remediation here
				// For now, just log the issue
			}
		}
	}
}
