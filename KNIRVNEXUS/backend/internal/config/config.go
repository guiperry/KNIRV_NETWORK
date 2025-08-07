package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Environment string       `mapstructure:"environment"`
	ChainID     string       `mapstructure:"chain_id"`
	NodeRole    string       `mapstructure:"node_role"`
	Database    DatabaseConfig `mapstructure:"database"`
	API         APIConfig      `mapstructure:"api"`
	P2P         P2PConfig      `mapstructure:"p2p"`
	Auth        AuthConfig     `mapstructure:"auth"`
	Validation  ValidationConfig `mapstructure:"validation"`
	TEE         TEEConfig      `mapstructure:"tee"`
	Reports     ReportsConfig  `mapstructure:"reports"`
	Log         LogConfig      `mapstructure:"log"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// P2PConfig represents P2P networking configuration
type P2PConfig struct {
	Port       int      `mapstructure:"port"`
	Bootstraps []string `mapstructure:"bootstraps"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret    string        `mapstructure:"jwt_secret"`
	TokenExpiry  time.Duration `mapstructure:"token_expiry"`
	EnableOAuth  bool          `mapstructure:"enable_oauth"`
	OAuthProviders []string    `mapstructure:"oauth_providers"`
}

// ValidationConfig represents validation engine configuration
type ValidationConfig struct {
	Timeout       time.Duration `mapstructure:"timeout"`
	MaxConcurrent int           `mapstructure:"max_concurrent"`
	EnableTEE     bool          `mapstructure:"enable_tee"`
}

// TEEConfig represents TEE (Trusted Execution Environment) configuration
type TEEConfig struct {
	Type           string `mapstructure:"type"` // "sgx", "sev-snp", "tdx", "software"
	SGXConfig      SGXConfig `mapstructure:"sgx"`
	SEVConfig      SEVConfig `mapstructure:"sev"`
	TDXConfig      TDXConfig `mapstructure:"tdx"`
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
	KeyFile          string `mapstructure:"key_file"`
}

// ReportsConfig represents report generation configuration
type ReportsConfig struct {
	StoragePath    string        `mapstructure:"storage_path"`
	MaxFileSize    int64         `mapstructure:"max_file_size"`
	RetentionDays  int           `mapstructure:"retention_days"`
	EnableSharing  bool          `mapstructure:"enable_sharing"`
	EnableScheduling bool        `mapstructure:"enable_scheduling"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	// Set configuration file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app/config")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Enable environment variable reading
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

// validateConfig validates the loaded configuration
func validateConfig(cfg *Config) error {
	if cfg.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}

	if cfg.NodeRole == "" {
		return fmt.Errorf("node_role is required")
	}

	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}

	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}

	return nil
}
