package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// Config represents the application configuration
type Config struct {
	Environment string           `mapstructure:"environment"`
	ChainID     string           `mapstructure:"chain_id"`
	NodeRole    string           `mapstructure:"node_role"`
	Mode        string           `mapstructure:"mode"` // "headless" or "gui"
	Testnet     bool             `mapstructure:"testnet"`
	Database    DatabaseConfig   `mapstructure:"database"`
	API         APIConfig        `mapstructure:"api"`
	GUI         GUIConfig        `mapstructure:"gui"`
	P2P         P2PConfig        `mapstructure:"p2p"`
	Auth        AuthConfig       `mapstructure:"auth"`
	Security    SecurityConfig   `mapstructure:"security"`
	Roles       RolesConfig      `mapstructure:"roles"`
	Network     NetworkConfig    `mapstructure:"network"`
	Validation  ValidationConfig `mapstructure:"validation"`
	TEE         TEEConfig        `mapstructure:"tee"`
	CDE         CDEConfig        `mapstructure:"cde"`
	Reports     ReportsConfig    `mapstructure:"reports"`
	Log         LogConfig        `mapstructure:"log"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Address     string `mapstructure:"address"`
	Port        int    `mapstructure:"port"`
	BindAddress string `mapstructure:"bind_address"`
}

// GUIConfig represents GUI mode configuration
type GUIConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Port         int    `mapstructure:"port"`
	FrontendPath string `mapstructure:"frontend_path"`
	BindAddress  string `mapstructure:"bind_address"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	AuthRequired bool   `mapstructure:"auth_required"`
	TLSEnabled   bool   `mapstructure:"tls_enabled"`
	AuditLogging bool   `mapstructure:"audit_logging"`
	JWTSecret    string `mapstructure:"jwt_secret"`
}

// RolesConfig represents user roles and permissions configuration
type RolesConfig struct {
	Validator ValidatorRole `mapstructure:"validator"`
	Admin     AdminRole     `mapstructure:"admin"`
	Observer  ObserverRole  `mapstructure:"observer"`
}

// ValidatorRole represents validator role configuration
type ValidatorRole struct {
	Permissions  []string `mapstructure:"permissions"`
	ScopedAccess bool     `mapstructure:"scoped_access"`
}

// AdminRole represents admin role configuration
type AdminRole struct {
	Permissions  []string `mapstructure:"permissions"`
	ScopedAccess bool     `mapstructure:"scoped_access"`
}

// ObserverRole represents observer role configuration
type ObserverRole struct {
	Permissions  []string `mapstructure:"permissions"`
	ScopedAccess bool     `mapstructure:"scoped_access"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	ChainID          string `mapstructure:"chain_id"`
	P2PPort          int    `mapstructure:"p2p_port"`
	DiscoveryEnabled bool   `mapstructure:"discovery_enabled"`
}

// P2PConfig represents P2P networking configuration
type P2PConfig struct {
	Port       int      `mapstructure:"port"`
	Bootstraps []string `mapstructure:"bootstraps"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret      string        `mapstructure:"jwt_secret"`
	TokenExpiry    time.Duration `mapstructure:"token_expiry"`
	EnableOAuth    bool          `mapstructure:"enable_oauth"`
	OAuthProviders []string      `mapstructure:"oauth_providers"`
}

// ValidationConfig represents validation engine configuration
type ValidationConfig struct {
	Timeout       time.Duration `mapstructure:"timeout"`
	MaxConcurrent int           `mapstructure:"max_concurrent"`
	EnableTEE     bool          `mapstructure:"enable_tee"`
}

// TEEConfig represents TEE (Trusted Execution Environment) configuration
type TEEConfig struct {
	Type           string         `mapstructure:"type"` // "sgx", "sev-snp", "tdx", "software"
	SGXConfig      SGXConfig      `mapstructure:"sgx"`
	SEVConfig      SEVConfig      `mapstructure:"sev"`
	TDXConfig      TDXConfig      `mapstructure:"tdx"`
	SoftwareConfig SoftwareConfig `mapstructure:"software"`
}

// SGXConfig represents Intel SGX configuration
type SGXConfig struct {
	EnclaveFile string `mapstructure:"enclave_file"`
	SpidFile    string `mapstructure:"spid_file"`
	IASUrl      string `mapstructure:"ias_url"`
}

// SEVConfig represents AMD SEV-SNP configuration
type SEVConfig struct {
	PolicyFile string `mapstructure:"policy_file"`
	CertChain  string `mapstructure:"cert_chain"`
}

// TDXConfig represents Intel TDX configuration
type TDXConfig struct {
	TDReportFile string `mapstructure:"td_report_file"`
	QuoteFile    string `mapstructure:"quote_file"`
}

// SoftwareConfig represents software-based TEE simulation
type SoftwareConfig struct {
	EnableAttestation bool   `mapstructure:"enable_attestation"`
	KeyFile           string `mapstructure:"key_file"`
}

// CDEConfig represents Cloud Development Environment configuration
type CDEConfig struct {
	BaseImagePath          string        `mapstructure:"base_image_path"`
	WorkspaceRoot          string        `mapstructure:"workspace_root"`
	MaxEnvironments        int           `mapstructure:"max_environments"`
	DefaultTimeout         time.Duration `mapstructure:"default_timeout"`
	MaxCPUPerEnv           float64       `mapstructure:"max_cpu_per_env"`
	MaxMemoryPerEnv        uint64        `mapstructure:"max_memory_per_env"`
	MaxDiskPerEnv          uint64        `mapstructure:"max_disk_per_env"`
	EnableSandboxing       bool          `mapstructure:"enable_sandboxing"`
	EnableNetworkIsolation bool          `mapstructure:"enable_network_isolation"`
	AllowedPorts           []int         `mapstructure:"allowed_ports"`
	SessionTimeout         time.Duration `mapstructure:"session_timeout"`
	MaxSessionsPerUser     int           `mapstructure:"max_sessions_per_user"`
	MaxProjectsPerUser     int           `mapstructure:"max_projects_per_user"`
	ProjectStoragePath     string        `mapstructure:"project_storage_path"`
}

// ReportsConfig represents report generation configuration
type ReportsConfig struct {
	StoragePath      string `mapstructure:"storage_path"`
	MaxFileSize      int64  `mapstructure:"max_file_size"`
	RetentionDays    int    `mapstructure:"retention_days"`
	EnableSharing    bool   `mapstructure:"enable_sharing"`
	EnableScheduling bool   `mapstructure:"enable_scheduling"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	return LoadWithDefaults()
}

// LoadWithDefaults loads configuration with default values
func LoadWithDefaults() (*Config, error) {
	// Load .env file if it exists (for API keys and other environment variables)
	// Try multiple locations: current directory, parent directory
	envPaths := []string{".env", "../.env"}
	for _, envPath := range envPaths {
		if err := gotenv.Load(envPath); err == nil {
			break // Successfully loaded .env file
		}
		// Continue trying other paths if this one fails
	}

	// Set default values
	setDefaults()

	// Set configuration file name and paths
	viper.SetConfigName("knirv-nexus")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("KNIRV")
	viper.AutomaticEnv()

	// Read configuration file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue with environment variables and defaults
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Operational mode defaults
	viper.SetDefault("mode", "headless")
	viper.SetDefault("gui.enabled", false)
	viper.SetDefault("gui.port", 9080)
	viper.SetDefault("gui.frontend_path", "./dist")
	viper.SetDefault("gui.bind_address", "127.0.0.1")

	// Service defaults
	viper.SetDefault("api.port", 8082)
	viper.SetDefault("api.bind_address", "0.0.0.0")
	viper.SetDefault("api.address", "0.0.0.0")

	// Security defaults
	viper.SetDefault("security.auth_required", true)
	viper.SetDefault("security.tls_enabled", true)
	viper.SetDefault("security.audit_logging", true)

	// User roles configuration
	viper.SetDefault("roles.validator.permissions", []string{"node:read", "node:update", "tasks:read", "results:read"})
	viper.SetDefault("roles.validator.scoped_access", true)
	viper.SetDefault("roles.admin.permissions", []string{"*:*"})
	viper.SetDefault("roles.admin.scoped_access", false)
	viper.SetDefault("roles.observer.permissions", []string{"*:read"})
	viper.SetDefault("roles.observer.scoped_access", false)

	// Database defaults
	viper.SetDefault("database.path", "./data/nexus.db")

	// Network configuration
	viper.SetDefault("network.chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("network.p2p_port", 4001)
	viper.SetDefault("network.discovery_enabled", true)

	// Logging configuration
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "./logs/nexus.log")

	// CDE configuration defaults
	viper.SetDefault("cde.base_image_path", "images")
	viper.SetDefault("cde.workspace_root", "workspaces")
	viper.SetDefault("cde.max_environments", 50)
	viper.SetDefault("cde.default_timeout", "1h")
	viper.SetDefault("cde.max_cpu_per_env", 2.0)
	viper.SetDefault("cde.max_memory_per_env", 2147483648) // 2GB
	viper.SetDefault("cde.max_disk_per_env", 10737418240)  // 10GB
	viper.SetDefault("cde.enable_sandboxing", true)
	viper.SetDefault("cde.enable_network_isolation", false)
	viper.SetDefault("cde.allowed_ports", []int{8080, 3000, 5000})
	viper.SetDefault("cde.session_timeout", "2h")
	viper.SetDefault("cde.max_sessions_per_user", 5)
	viper.SetDefault("cde.max_projects_per_user", 20)
	viper.SetDefault("cde.project_storage_path", "projects")

	// Legacy defaults for backward compatibility
	viper.SetDefault("chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("node_role", "dve-manager")
	viper.SetDefault("p2p.port", 4001)
	viper.SetDefault("auth.jwt_secret", "")
}

// validateConfig validates the loaded configuration
func validateConfig(cfg *Config) error {
	// Validate mode
	if cfg.Mode != "headless" && cfg.Mode != "gui" {
		return fmt.Errorf("mode must be 'headless' or 'gui', got: %s", cfg.Mode)
	}

	// Validate chain ID (use Network.ChainID if available, fallback to ChainID)
	chainID := cfg.Network.ChainID
	if chainID == "" {
		chainID = cfg.ChainID
	}
	if chainID == "" {
		return fmt.Errorf("chain_id is required")
	}

	if cfg.NodeRole == "" {
		return fmt.Errorf("node_role is required")
	}

	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}

	// JWT secret validation - only required in headless mode or when auth is required
	jwtSecret := cfg.Security.JWTSecret
	if jwtSecret == "" {
		jwtSecret = cfg.Auth.JWTSecret
	}
	if cfg.Mode == "headless" && cfg.Security.AuthRequired && jwtSecret == "" {
		return fmt.Errorf("security.jwt_secret is required in headless mode with authentication")
	}

	// GUI mode specific validation
	if cfg.Mode == "gui" && cfg.GUI.Enabled {
		if cfg.GUI.Port <= 0 {
			return fmt.Errorf("gui.port must be positive")
		}
		if cfg.GUI.FrontendPath == "" {
			return fmt.Errorf("gui.frontend_path is required when GUI is enabled")
		}
	}

	return nil
}
