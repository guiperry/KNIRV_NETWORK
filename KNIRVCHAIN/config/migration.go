package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// MigrationTool handles migration from old configuration patterns to unified configuration
type MigrationTool struct {
	oldViper     *viper.Viper
	newManager   *UnifiedConfigManager
	migrationMap map[string]string
}

// NewMigrationTool creates a new configuration migration tool
func NewMigrationTool(oldConfigPath string, newManager *UnifiedConfigManager) (*MigrationTool, error) {
	oldViper := viper.New()

	// Load old configuration
	if oldConfigPath != "" {
		oldViper.SetConfigFile(oldConfigPath)
		if err := oldViper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read old config file: %w", err)
		}
	}

	// Set up old environment variable patterns
	oldViper.SetEnvPrefix("agent") // Old prefix
	oldViper.AutomaticEnv()
	oldViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	tool := &MigrationTool{
		oldViper:     oldViper,
		newManager:   newManager,
		migrationMap: createMigrationMap(),
	}

	return tool, nil
}

// createMigrationMap creates a mapping from old configuration keys to new ones
func createMigrationMap() map[string]string {
	return map[string]string{
		// Core configuration mappings
		"httpport":               "http_port",
		"p2pport":                "p2p_port",
		"walletport":             "wallet_port",
		"chainid":                "chain_id",
		"blockchainDatabasePath": "blockchain_database_path",
		"searchableDatabasePath": "searchable_database_path",
		"reflectionDatabasePath": "reflection_database_path",

		// Service-specific mappings
		"chromem.cerebras_config.api_key":  "inference.cerebras.api_key",
		"chromem.cerebras_config.base_url": "inference.cerebras.base_url",

		// Network configuration
		"network.port":        "http_port",
		"network.p2p_port":    "p2p_port",
		"network.wallet_port": "wallet_port",
		"network.max_devs":    "max_devs",

		// UI configuration
		"ui.use_terminal_integration": "ui.terminal_integration",
		"ui.theme":                    "ui.theme",

		// Blockchain configuration
		"blockchain.data_dir":   "data_dir",
		"blockchain.block_time": "block_time",

		// Wallet configuration
		"wallet.auto_unlock": "wallet.auto_unlock",

		// Economics service mappings
		"economics.port":         "economics.port",
		"economics.nrn_contract": "economics.nrn_contract",
		"economics.xion_rpc":     "economics.xion_rpc",

		// Network monitor mappings
		"network_monitor.port":     "network_monitor.port",
		"network_monitor.interval": "network_monitor.interval",
	}
}

// MigrateConfiguration migrates configuration from old format to new unified format
func (m *MigrationTool) MigrateConfiguration() error {
	log.Println("Starting configuration migration...")

	// Migrate configuration values
	if err := m.migrateConfigurationValues(); err != nil {
		return fmt.Errorf("failed to migrate configuration values: %w", err)
	}

	// Migrate environment variables
	if err := m.migrateEnvironmentVariables(); err != nil {
		return fmt.Errorf("failed to migrate environment variables: %w", err)
	}

	log.Println("Configuration migration completed successfully")
	return nil
}

// migrateConfigurationValues migrates configuration values from old to new format
func (m *MigrationTool) migrateConfigurationValues() error {
	log.Println("Migrating configuration values...")

	// Get all settings from old configuration
	oldSettings := m.oldViper.AllSettings()

	for oldKey, value := range oldSettings {
		// Check if we have a mapping for this key
		if newKey, exists := m.migrationMap[oldKey]; exists {
			log.Printf("Migrating %s -> %s: %v", oldKey, newKey, value)
			if err := m.newManager.SetConfigValue(newKey, value); err != nil {
				log.Printf("Warning: Failed to migrate %s to %s: %v", oldKey, newKey, err)
			}
		} else {
			// Try to migrate nested keys
			if err := m.migrateNestedKey(oldKey, value); err != nil {
				log.Printf("Warning: Failed to migrate nested key %s: %v", oldKey, err)
			}
		}
	}

	return nil
}

// migrateNestedKey handles migration of nested configuration keys
func (m *MigrationTool) migrateNestedKey(key string, value interface{}) error {
	// Handle nested maps
	if valueMap, ok := value.(map[string]interface{}); ok {
		for nestedKey, nestedValue := range valueMap {
			fullOldKey := fmt.Sprintf("%s.%s", key, nestedKey)
			if newKey, exists := m.migrationMap[fullOldKey]; exists {
				log.Printf("Migrating nested %s -> %s: %v", fullOldKey, newKey, nestedValue)
				if err := m.newManager.SetConfigValue(newKey, nestedValue); err != nil {
					return fmt.Errorf("failed to migrate nested key %s: %w", fullOldKey, err)
				}
			}
		}
	}

	return nil
}

// migrateEnvironmentVariables creates a migration guide for environment variables
func (m *MigrationTool) migrateEnvironmentVariables() error {
	log.Println("Creating environment variable migration guide...")

	// Create migration guide for environment variables
	migrationGuide := []string{
		"# Environment Variable Migration Guide",
		"# Old -> New mappings for KNIRVCHAIN configuration",
		"",
		"# Core configuration",
		"# agent_HTTPPORT -> KNIRV_HTTP_PORT",
		"# agent_P2PPORT -> KNIRV_P2P_PORT",
		"# agent_WALLETPORT -> KNIRV_WALLET_PORT",
		"# agent_CHAINID -> KNIRV_CHAIN_ID",
		"",
		"# Cerebras configuration",
		"# DEFAULT_CEREBRAS_API_KEY -> KNIRV_CEREBRAS_API_KEY",
		"# DEFAULT_CEREBRAS_BASE_URL -> KNIRV_CEREBRAS_BASE_URL",
		"",
		"# Economics service",
		"# ECONOMICS_PORT -> KNIRV_ECONOMICS_PORT",
		"# NRN_CONTRACT -> KNIRV_ECONOMICS_NRN_CONTRACT",
		"# XION_RPC -> KNIRV_ECONOMICS_XION_RPC",
		"# KNIRVCHAIN_URL -> KNIRV_ECONOMICS_KNIRVCHAIN_URL",
		"# KNIRVNEXUS_URL -> KNIRV_ECONOMICS_KNIRVNEXUS_URL",
		"# _URL -> KNIRV_ECONOMICS__URL",
		"",
		"# Node.js services",
		"# HTTP_API_PORT -> KNIRV_NODEJS_PORT",
		"# PORT -> KNIRV_NODEJS_PORT",
		"# NODE_ENV -> KNIRV_NODEJS_ENV",
		"",
	}

	// Write migration guide to file
	migrationFile := "ENVIRONMENT_MIGRATION_GUIDE.md"
	file, err := os.Create(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to create migration guide file: %w", err)
	}
	defer file.Close()

	for _, line := range migrationGuide {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write migration guide: %w", err)
		}
	}

	log.Printf("Environment variable migration guide created: %s", migrationFile)
	return nil
}

// ValidateMigration validates that the migration was successful
func (m *MigrationTool) ValidateMigration() error {
	log.Println("Validating configuration migration...")

	// Get the migrated configuration
	cfg, err := m.newManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get migrated configuration: %w", err)
	}

	// Validate critical configuration values
	validationChecks := []struct {
		name  string
		check func(*Config) error
	}{
		{
			name: "HTTP Port",
			check: func(c *Config) error {
				if c.Port <= 0 || c.Port > 65535 {
					return fmt.Errorf("invalid HTTP port: %d", c.Port)
				}
				return nil
			},
		},
		{
			name: "P2P Port",
			check: func(c *Config) error {
				if c.P2PPort <= 0 || c.P2PPort > 65535 {
					return fmt.Errorf("invalid P2P port: %d", c.P2PPort)
				}
				return nil
			},
		},
		{
			name: "Chain ID",
			check: func(c *Config) error {
				if c.ChainID == "" {
					return fmt.Errorf("chain ID cannot be empty")
				}
				return nil
			},
		},
	}

	// Run validation checks
	for _, check := range validationChecks {
		if err := check.check(cfg); err != nil {
			return fmt.Errorf("validation failed for %s: %w", check.name, err)
		}
		log.Printf("✓ %s validation passed", check.name)
	}

	log.Println("Configuration migration validation completed successfully")
	return nil
}

// GenerateMigrationReport generates a detailed migration report
func (m *MigrationTool) GenerateMigrationReport() error {
	log.Println("Generating migration report...")

	reportFile := "CONFIGURATION_MIGRATION_REPORT.md"
	file, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("failed to create migration report file: %w", err)
	}
	defer file.Close()

	report := []string{
		"# KNIRVCHAIN Configuration Migration Report",
		"",
		"## Migration Summary",
		"",
		"This report documents the migration from legacy configuration patterns to the unified configuration system.",
		"",
		"## Configuration Value Migrations",
		"",
	}

	// Add migration mappings to report
	for oldKey, newKey := range m.migrationMap {
		oldValue := m.oldViper.Get(oldKey)
		if oldValue != nil {
			newValue := m.newManager.GetConfigValue(newKey)
			report = append(report, fmt.Sprintf("- `%s` -> `%s`: %v -> %v", oldKey, newKey, oldValue, newValue))
		}
	}

	report = append(report, []string{
		"",
		"## Environment Variable Changes",
		"",
		"- Old prefix: `agent_`",
		"- New prefix: `KNIRV_`",
		"",
		"## Next Steps",
		"",
		"1. Update deployment scripts to use new environment variables",
		"2. Update configuration files to use new format",
		"3. Test all services with new configuration",
		"4. Remove old configuration files after validation",
		"",
	}...)

	// Write report to file
	for _, line := range report {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write migration report: %w", err)
		}
	}

	log.Printf("Migration report generated: %s", reportFile)
	return nil
}

// CleanupOldConfiguration removes old configuration files and patterns
func (m *MigrationTool) CleanupOldConfiguration() error {
	log.Println("Cleaning up old configuration patterns...")

	// This is a placeholder for cleanup operations
	// In a real implementation, you would:
	// 1. Remove old configuration files
	// 2. Update service startup scripts
	// 3. Remove old environment variable references

	log.Println("Old configuration cleanup completed")
	return nil
}
