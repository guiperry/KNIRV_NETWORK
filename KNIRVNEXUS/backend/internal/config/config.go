package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// PaymentProcessorConfig defines payment processor configuration
type PaymentProcessorConfig struct {
	Enabled            bool    `mapstructure:"enabled"`
	NodeRPC            string  `mapstructure:"node_rpc"`
	WebhookPort        int     `mapstructure:"webhook_port"`
	TokenSymbol        string  `mapstructure:"token_symbol"`
	TokenDecimals      int     `mapstructure:"token_decimals"`
	USDPerToken        float64 `mapstructure:"usd_per_token"`
	ETHPerToken        float64 `mapstructure:"eth_per_token"`
}

// BootnodeConfig defines bootnode configuration
type BootnodeConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// TunnelClientConfig defines tunnel client configuration
type TunnelClientConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	ServerAddress  string `mapstructure:"server_address"`
	ControlPort    uint   `mapstructure:"control_port"`
	PingInterval   uint   `mapstructure:"ping_interval"`
	ReconnectDelay uint   `mapstructure:"reconnect_delay"`
}

// ReverseProxyConfig defines reverse proxy configuration
type ReverseProxyConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	ListenAddr string `mapstructure:"listen_addr"`
}

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
	DVE         DVEConfig        `mapstructure:"dve"`
	Failover    FailoverConfig   `mapstructure:"failover"`
	ModelServer ModelServerConfig `mapstructure:"model_server"`
	IsRoot      bool             `json:"is_root" mapstructure:"is_root,IsRoot"`
	IsBootnode  bool             `json:"is_bootnode" mapstructure:"is_bootnode,IsBootnode"`                                          // General bootnode flag
	IsPeer      bool             `json:"is_dev" mapstructure:"is_dev,IsPeer"`
	ClientOnly  bool             `json:"client_only" mapstructure:"client_only,clientOnly"`

	// Additional fields for role-based settings
	PaymentProcessor PaymentProcessorConfig `mapstructure:"payment_processor"`
	Bootnode         BootnodeConfig         `mapstructure:"bootnode"`
	TunnelClient     TunnelClientConfig     `mapstructure:"tunnel_client"`
	ReverseProxy     ReverseProxyConfig     `mapstructure:"reverse_proxy"`

	// Legacy fields
	Port        uint64 `mapstructure:"port"`
	P2PPort     uint64 `mapstructure:"p2p_port"`
	WalletPort  uint64 `mapstructure:"wallet_port"`
	NoWalletServer bool `mapstructure:"no_wallet_server"`
	UseGUI      bool   `mapstructure:"use_gui"`
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
	DHTEnabled bool     `mapstructure:"dht_enabled"`
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

// DVEConfig represents DVE (Decentralized Validation Environment) configuration
type DVEConfig struct {
	Discovery DVEDiscoveryConfig `mapstructure:"discovery"`
}

// DVEDiscoveryConfig represents DVE discovery configuration
type DVEDiscoveryConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	BootstrapNodes       []string      `mapstructure:"bootstrap_nodes"`
	DiscoveryInterval    time.Duration `mapstructure:"discovery_interval"`
	ConnectionTimeout    time.Duration `mapstructure:"connection_timeout"`
	HealthCheckInterval  time.Duration `mapstructure:"health_check_interval"`
	HeartbeatTimeout     time.Duration `mapstructure:"heartbeat_timeout"`
	MaxRetries           int           `mapstructure:"max_retries"`
	RetryBackoffDuration time.Duration `mapstructure:"retry_backoff_duration"`
	ConnectionPoolSize   int           `mapstructure:"connection_pool_size"`
}

// FailoverConfig represents failover service configuration
type FailoverConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	HealthCheckInterval  time.Duration `mapstructure:"health_check_interval"`
	HealthCheckTimeout   time.Duration `mapstructure:"health_check_timeout"`
	FailoverThreshold    int           `mapstructure:"failover_threshold"`
	EnableAutoFailover   bool          `mapstructure:"enable_auto_failover"`
	RootNodes            []string      `mapstructure:"root_nodes"`
	Bootnodes            []string      `mapstructure:"bootnodes"`
}

// ModelServerConfig represents model server configuration
type ModelServerConfig struct {
	StoragePath string `mapstructure:"storage_path"`
	MaxModels   int    `mapstructure:"max_models"`
	EnableCORS  bool   `mapstructure:"enable_cors"`
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
	viper.SetConfigName("production")
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

	// Expand all paths in configuration
	if err := config.expandPaths(); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// getAppDataDir returns the XDG Base Directory for app data
// Following XDG Base Directory Specification: ~/.local/share/knirvnexus/backend_server
func getAppDataDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	appDataDir := filepath.Join(usr.HomeDir, ".local", "share", "knirvnexus", "backend_server")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %v", appDataDir, err)
	}

	return appDataDir, nil
}

// expandPath expands paths with ~ to the user's home directory and creates them if needed
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		expanded := filepath.Join(usr.HomeDir, path[1:])
		return expanded, nil
	}
	return path, nil
}

// expandPaths expands all paths in the configuration
func (c *Config) expandPaths() error {
	var err error

	// Expand database path
	if c.Database.Path != "" {
		c.Database.Path, err = expandPath(c.Database.Path)
		if err != nil {
			return err
		}
		// Ensure directory exists
		dbDir := filepath.Dir(c.Database.Path)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %v", err)
		}
	}

	// Expand CDE paths
	if c.CDE.BaseImagePath != "" {
		c.CDE.BaseImagePath, err = expandPath(c.CDE.BaseImagePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(c.CDE.BaseImagePath, 0755); err != nil {
			return fmt.Errorf("failed to create base image directory: %v", err)
		}
	}

	if c.CDE.WorkspaceRoot != "" {
		c.CDE.WorkspaceRoot, err = expandPath(c.CDE.WorkspaceRoot)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(c.CDE.WorkspaceRoot, 0755); err != nil {
			return fmt.Errorf("failed to create workspace root directory: %v", err)
		}
	}

	if c.CDE.ProjectStoragePath != "" {
		c.CDE.ProjectStoragePath, err = expandPath(c.CDE.ProjectStoragePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(c.CDE.ProjectStoragePath, 0755); err != nil {
			return fmt.Errorf("failed to create project storage directory: %v", err)
		}
	}

	// Expand reports path
	if c.Reports.StoragePath != "" {
		c.Reports.StoragePath, err = expandPath(c.Reports.StoragePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(c.Reports.StoragePath, 0755); err != nil {
			return fmt.Errorf("failed to create reports directory: %v", err)
		}
	}

	// Expand log output path
	if c.Log.Output != "" {
		c.Log.Output, err = expandPath(c.Log.Output)
		if err != nil {
			return err
		}
		logDir := filepath.Dir(c.Log.Output)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create logs directory: %v", err)
		}
	}

	// Expand model server storage path
	if c.ModelServer.StoragePath != "" {
		c.ModelServer.StoragePath, err = expandPath(c.ModelServer.StoragePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(c.ModelServer.StoragePath, 0755); err != nil {
			return fmt.Errorf("failed to create model server storage directory: %v", err)
		}
	}

	return nil
}

// Role type (if not already defined elsewhere, or import if it is)
type Role string

const (
	Root           Role = "Root"
	RoleBootnode   Role = "Bootnode"
	RoleReflection Role = "Reflection"
	RolePeer       Role = "Peer"
	RoleClient     Role = "Client"
	// Add other roles if any
)

func (r Role) String() string {
	return string(r)
}

// DetermineRole determines the node's role based on flags.
// This function can be used both with command-line flags and with Config struct fields
func DetermineRole(IsRoot, IsBootnode, IsPeer, IsClientOnly bool) Role {
	if IsRoot {
		return Root
	}
	if IsBootnode {
		return RoleBootnode
	}
	if IsPeer {
		return RolePeer
	}
	if IsClientOnly {
		return RoleClient
	}
	return RoleClient // Default to Client if no specific role flag is set
}

// DetermineRoleFromConfig determines the node's role based on build flags first, then Config struct fields
func DetermineRoleFromConfig(cfg *Config) Role {
	// Check for build flags first (these will be set at compile time)
	// The actual build flag checking is done in the role-specific entry point files

	// If no build flags are set, fall back to config values
	return DetermineRole(cfg.IsRoot, cfg.IsBootnode, cfg.IsPeer, cfg.ClientOnly)
}


// setDefaults sets default configuration values
func setDefaults() {
	appDataDir, err := getAppDataDir()
	if err != nil {
		// Fallback to relative paths if we can't get app data directory
		appDataDir = "."
	}

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
	viper.SetDefault("security.auth_required", false) // false for testnet
	viper.SetDefault("security.tls_enabled", false)   // false for development
	viper.SetDefault("security.audit_logging", true)

	// User roles configuration
	viper.SetDefault("roles.validator.permissions", []string{"node:read", "node:update", "tasks:read", "results:read"})
	viper.SetDefault("roles.validator.scoped_access", true)
	viper.SetDefault("roles.admin.permissions", []string{"*:*"})
	viper.SetDefault("roles.admin.scoped_access", false)
	viper.SetDefault("roles.observer.permissions", []string{"*:read"})
	viper.SetDefault("roles.observer.scoped_access", false)

	// Database defaults - use XDG Base Directory
	viper.SetDefault("database.path", filepath.Join(appDataDir, "data", "nexus.db"))

	// Network configuration
	viper.SetDefault("network.chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("network.p2p_port", 4001)
	viper.SetDefault("network.discovery_enabled", true)

	// Logging configuration - use XDG Base Directory
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", filepath.Join(appDataDir, "logs", "nexus.log"))

	// CDE configuration defaults - use XDG Base Directory
	viper.SetDefault("cde.base_image_path", filepath.Join(appDataDir, "images"))
	viper.SetDefault("cde.workspace_root", filepath.Join(appDataDir, "workspaces"))
	viper.SetDefault("cde.max_environments", 50)
	viper.SetDefault("cde.default_timeout", "1h")
	viper.SetDefault("cde.max_cpu_per_env", 2.0)
	viper.SetDefault("cde.max_memory_per_env", 2147483648) // 2GB
	viper.SetDefault("cde.max_disk_per_env", 10737418240)  // 10GB
	viper.SetDefault("cde.enable_sandboxing", true)
	viper.SetDefault("cde.enable_network_isolation", false)
	viper.SetDefault("cde.allowed_ports", []int{8082, 3000, 5000})
	viper.SetDefault("cde.session_timeout", "2h")
	viper.SetDefault("cde.max_sessions_per_user", 5)
	viper.SetDefault("cde.max_projects_per_user", 20)
	viper.SetDefault("cde.project_storage_path", filepath.Join(appDataDir, "projects"))

	// Reports configuration - use XDG Base Directory
	viper.SetDefault("reports.storage_path", filepath.Join(appDataDir, "reports"))
	viper.SetDefault("reports.max_file_size", int64(104857600)) // 100MB
	viper.SetDefault("reports.retention_days", 30)
	viper.SetDefault("reports.enable_sharing", true)
	viper.SetDefault("reports.enable_scheduling", true)

	// DVE Discovery configuration defaults
	viper.SetDefault("dve.discovery.enabled", true)
	viper.SetDefault("dve.discovery.bootstrap_nodes", []string{})
	viper.SetDefault("dve.discovery.discovery_interval", "30s")
	viper.SetDefault("dve.discovery.connection_timeout", "5s")
	viper.SetDefault("dve.discovery.health_check_interval", "60s")
	viper.SetDefault("dve.discovery.heartbeat_timeout", "2m")
	viper.SetDefault("dve.discovery.max_retries", 3)
	viper.SetDefault("dve.discovery.retry_backoff_duration", "2s")
	viper.SetDefault("dve.discovery.connection_pool_size", 10)

	// Failover configuration defaults
	viper.SetDefault("failover.enabled", false)
	viper.SetDefault("failover.health_check_interval", "30s")
	viper.SetDefault("failover.health_check_timeout", "5s")
	viper.SetDefault("failover.failover_threshold", 3)
	viper.SetDefault("failover.enable_auto_failover", false)
	viper.SetDefault("failover.root_nodes", []string{})
	viper.SetDefault("failover.bootnodes", []string{})

	// Model server configuration defaults - use XDG Base Directory
	viper.SetDefault("model_server.storage_path", filepath.Join(appDataDir, "models"))
	viper.SetDefault("model_server.max_models", 10)
	viper.SetDefault("model_server.enable_cors", true)

	// Legacy defaults for backward compatibility
	viper.SetDefault("chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("node_role", "dve-manager")
	viper.SetDefault("p2p.port", 4001)
	viper.SetDefault("p2p.dht_enabled", false) // DHT disabled by default to reduce noise
	viper.SetDefault("auth.jwt_secret", "")
}

// GetConfigDir returns the base configuration directory (e.g., ~/.config/knirv-nexus)
func GetConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	appConfigDir := filepath.Join(userConfigDir, "knirv-nexus")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app config directory %s: %w", appConfigDir, err)
	}
	return appConfigDir, nil
}

// GetDataDir returns the data directory path based on role
func GetDataDir(role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	baseDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Determine role-specific data directory name
	var dataDirName string
	switch currentRole {
	case Root:
		dataDirName = "root_data"
	case RoleBootnode:
		dataDirName = "bootnode_data"
	case RolePeer:
		dataDirName = "peer_data"
	case RoleClient:
		dataDirName = "client_data"
	default:
		dataDirName = "data" // Fallback
	}

	dataDirPath := filepath.Join(baseDir, dataDirName)
	if err := os.MkdirAll(dataDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory %s: %w", currentRole, dataDirPath, err)
	}

	return dataDirPath, nil
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
