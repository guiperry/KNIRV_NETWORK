package agentify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TEESecurityProfile defines predefined security profiles for different use cases
type TEESecurityProfile struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	SecurityLevel  TEESecurityLevel `json:"security_level"`
	ResourceLimits ResourceLimits   `json:"resource_limits"`
	SecurityPolicy SecurityPolicy   `json:"security_policy"`
	RecommendedFor []string         `json:"recommended_for"`
}

// DefaultSecurityProfiles provides predefined security profiles
var DefaultSecurityProfiles = map[string]TEESecurityProfile{
	"development": {
		Name:          "Development",
		Description:   "Relaxed security for development and testing",
		SecurityLevel: SecurityLevelLow,
		ResourceLimits: ResourceLimits{
			MemoryMB:             1024,
			CPUCores:             2.0,
			DiskSpaceMB:          500,
			NetworkBandwidthMBps: 100,
			MaxProcesses:         10,
			ExecutionTimeout:     60 * time.Second,
		},
		SecurityPolicy: SecurityPolicy{
			AllowNetworkAccess:   true,
			AllowFileSystemWrite: true,
			AllowedCommands:      []string{}, // Allow all commands
			BlockedCommands:      []string{"rm", "rmdir", "format", "fdisk"},
			RequireSignature:     false,
			MaxExecutionTime:     120 * time.Second,
		},
		RecommendedFor: []string{"development", "testing", "debugging"},
	},
	"production": {
		Name:          "Production",
		Description:   "Balanced security for production environments",
		SecurityLevel: SecurityLevelMedium,
		ResourceLimits: ResourceLimits{
			MemoryMB:             512,
			CPUCores:             1.0,
			DiskSpaceMB:          200,
			NetworkBandwidthMBps: 50,
			MaxProcesses:         5,
			ExecutionTimeout:     30 * time.Second,
		},
		SecurityPolicy: SecurityPolicy{
			AllowNetworkAccess:   true,
			AllowFileSystemWrite: false,
			AllowedCommands:      []string{"python", "node", "java", "go", "curl", "wget"},
			BlockedCommands:      []string{"rm", "rmdir", "mv", "cp", "dd", "sudo", "su", "mount"},
			RequireSignature:     true,
			MaxExecutionTime:     60 * time.Second,
		},
		RecommendedFor: []string{"production", "staging", "user-facing"},
	},
	"high_security": {
		Name:          "High Security",
		Description:   "Maximum security for sensitive operations",
		SecurityLevel: SecurityLevelHigh,
		ResourceLimits: ResourceLimits{
			MemoryMB:             256,
			CPUCores:             0.5,
			DiskSpaceMB:          100,
			NetworkBandwidthMBps: 10,
			MaxProcesses:         3,
			ExecutionTimeout:     15 * time.Second,
		},
		SecurityPolicy: SecurityPolicy{
			AllowNetworkAccess:   false,
			AllowFileSystemWrite: false,
			AllowedCommands:      []string{"python", "node", "echo", "cat"},
			BlockedCommands: []string{
				"rm", "rmdir", "mv", "cp", "dd", "sudo", "su", "mount", "umount",
				"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync", "ping",
				"chmod", "chown", "mkdir", "touch", "tee",
			},
			RequireSignature: true,
			MaxExecutionTime: 30 * time.Second,
		},
		RecommendedFor: []string{"financial", "healthcare", "government", "sensitive_data"},
	},
	"sandbox": {
		Name:          "Sandbox",
		Description:   "Isolated sandbox for untrusted code",
		SecurityLevel: SecurityLevelHigh,
		ResourceLimits: ResourceLimits{
			MemoryMB:             128,
			CPUCores:             0.25,
			DiskSpaceMB:          50,
			NetworkBandwidthMBps: 0,
			MaxProcesses:         2,
			ExecutionTimeout:     10 * time.Second,
		},
		SecurityPolicy: SecurityPolicy{
			AllowNetworkAccess:   false,
			AllowFileSystemWrite: false,
			AllowedCommands:      []string{"python", "node", "echo"},
			BlockedCommands: []string{
				"rm", "rmdir", "mv", "cp", "dd", "sudo", "su", "mount", "umount",
				"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync", "ping",
				"chmod", "chown", "mkdir", "touch", "tee", "cat", "ls", "ps",
			},
			RequireSignature: true,
			MaxExecutionTime: 15 * time.Second,
		},
		RecommendedFor: []string{"untrusted_code", "user_uploads", "experimental"},
	},
}

// TEESecurityConfigManager manages TEE security configurations
type TEESecurityConfigManager struct {
	configPath    string
	profiles      map[string]TEESecurityProfile
	globalConfig  TEESecurityConfig
	agentProfiles map[string]string // agentID -> profileName
}

// NewTEESecurityConfigManager creates a new configuration manager
func NewTEESecurityConfigManager(configPath string) *TEESecurityConfigManager {
	manager := &TEESecurityConfigManager{
		configPath:    configPath,
		profiles:      make(map[string]TEESecurityProfile),
		agentProfiles: make(map[string]string),
		globalConfig: TEESecurityConfig{
			DefaultSecurityLevel:    SecurityLevelMedium,
			MaxConcurrentTEEs:       50,
			AuditLogRetentionDays:   30,
			IntegrityCheckInterval:  5 * time.Minute,
			EnableRuntimeMonitoring: true,
			GlobalResourceLimits: ResourceLimits{
				MemoryMB:             2048,
				CPUCores:             4.0,
				DiskSpaceMB:          1000,
				NetworkBandwidthMBps: 200,
				MaxProcesses:         20,
				ExecutionTimeout:     300 * time.Second,
			},
		},
	}

	// Load default profiles
	for name, profile := range DefaultSecurityProfiles {
		manager.profiles[name] = profile
	}

	// Load configuration from file if it exists
	manager.LoadConfiguration()

	return manager
}

// LoadConfiguration loads configuration from file
func (m *TEESecurityConfigManager) LoadConfiguration() error {
	configFile := filepath.Join(m.configPath, "tee_security_config.json")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults and create it
		return m.SaveConfiguration()
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		GlobalConfig  TEESecurityConfig             `json:"global_config"`
		Profiles      map[string]TEESecurityProfile `json:"profiles"`
		AgentProfiles map[string]string             `json:"agent_profiles"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	m.globalConfig = config.GlobalConfig

	// Merge custom profiles with defaults
	for name, profile := range config.Profiles {
		m.profiles[name] = profile
	}

	m.agentProfiles = config.AgentProfiles

	return nil
}

// SaveConfiguration saves configuration to file
func (m *TEESecurityConfigManager) SaveConfiguration() error {
	// Ensure config directory exists
	if err := os.MkdirAll(m.configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	config := struct {
		GlobalConfig  TEESecurityConfig             `json:"global_config"`
		Profiles      map[string]TEESecurityProfile `json:"profiles"`
		AgentProfiles map[string]string             `json:"agent_profiles"`
	}{
		GlobalConfig:  m.globalConfig,
		Profiles:      m.profiles,
		AgentProfiles: m.agentProfiles,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configFile := filepath.Join(m.configPath, "tee_security_config.json")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfileForAgent returns the security profile for an agent
func (m *TEESecurityConfigManager) GetProfileForAgent(agentID string) (TEESecurityProfile, error) {
	profileName, exists := m.agentProfiles[agentID]
	if !exists {
		// Use default profile based on global config
		switch m.globalConfig.DefaultSecurityLevel {
		case SecurityLevelLow:
			profileName = "development"
		case SecurityLevelMedium:
			profileName = "production"
		case SecurityLevelHigh:
			profileName = "high_security"
		default:
			profileName = "production"
		}
	}

	profile, exists := m.profiles[profileName]
	if !exists {
		return TEESecurityProfile{}, fmt.Errorf("security profile '%s' not found", profileName)
	}

	return profile, nil
}

// SetProfileForAgent sets the security profile for an agent
func (m *TEESecurityConfigManager) SetProfileForAgent(agentID, profileName string) error {
	if _, exists := m.profiles[profileName]; !exists {
		return fmt.Errorf("security profile '%s' not found", profileName)
	}

	m.agentProfiles[agentID] = profileName
	return m.SaveConfiguration()
}

// AddCustomProfile adds a custom security profile
func (m *TEESecurityConfigManager) AddCustomProfile(name string, profile TEESecurityProfile) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	profile.Name = name
	m.profiles[name] = profile
	return m.SaveConfiguration()
}

// GetAvailableProfiles returns all available security profiles
func (m *TEESecurityConfigManager) GetAvailableProfiles() map[string]TEESecurityProfile {
	profiles := make(map[string]TEESecurityProfile)
	for name, profile := range m.profiles {
		profiles[name] = profile
	}
	return profiles
}

// GetGlobalConfig returns the global TEE security configuration
func (m *TEESecurityConfigManager) GetGlobalConfig() TEESecurityConfig {
	return m.globalConfig
}

// UpdateGlobalConfig updates the global TEE security configuration
func (m *TEESecurityConfigManager) UpdateGlobalConfig(config TEESecurityConfig) error {
	m.globalConfig = config
	return m.SaveConfiguration()
}

// ValidateProfile validates a security profile
func (m *TEESecurityConfigManager) ValidateProfile(profile TEESecurityProfile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	if profile.ResourceLimits.MemoryMB <= 0 {
		return fmt.Errorf("memory limit must be positive")
	}

	if profile.ResourceLimits.CPUCores <= 0 {
		return fmt.Errorf("CPU cores must be positive")
	}

	if profile.SecurityPolicy.MaxExecutionTime <= 0 {
		return fmt.Errorf("max execution time must be positive")
	}

	// Validate that resource limits don't exceed global limits
	if profile.ResourceLimits.MemoryMB > m.globalConfig.GlobalResourceLimits.MemoryMB {
		return fmt.Errorf("profile memory limit exceeds global limit")
	}

	if profile.ResourceLimits.CPUCores > m.globalConfig.GlobalResourceLimits.CPUCores {
		return fmt.Errorf("profile CPU limit exceeds global limit")
	}

	return nil
}

// GetRecommendedProfile returns a recommended profile based on agent type and use case
func (m *TEESecurityConfigManager) GetRecommendedProfile(agentType, useCase string) (string, error) {
	for name, profile := range m.profiles {
		for _, recommended := range profile.RecommendedFor {
			if recommended == useCase || recommended == agentType {
				return name, nil
			}
		}
	}

	// Default recommendation based on use case
	switch useCase {
	case "development", "testing", "debugging":
		return "development", nil
	case "production", "staging":
		return "production", nil
	case "financial", "healthcare", "government", "sensitive_data":
		return "high_security", nil
	case "untrusted_code", "user_uploads", "experimental":
		return "sandbox", nil
	default:
		return "production", nil
	}
}
