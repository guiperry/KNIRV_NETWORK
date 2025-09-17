package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// UnifiedConfigManager provides centralized configuration management
// replacing multiple Viper instances with a single, consistent interface
type UnifiedConfigManager struct {
	viper            *viper.Viper
	components       map[string]ComponentConfig
	validators       map[string]ConfigValidator
	mutex            sync.RWMutex
	hotReloadEnabled bool
	configPath       string
	role             Role
}

// ComponentConfig represents configuration for a specific component
type ComponentConfig interface {
	// GetConfigKey returns the configuration key prefix for this component
	GetConfigKey() string
	// Validate validates the component configuration
	Validate() error
	// GetDefaults returns default configuration values
	GetDefaults() map[string]interface{}
	// GetEnvironmentMappings returns environment variable mappings
	GetEnvironmentMappings() map[string]string
}

// ConfigValidator validates configuration values
type ConfigValidator func(key string, value interface{}) error

// UnifiedConfigOptions configures the UnifiedConfigManager
type UnifiedConfigOptions struct {
	Role              Role
	ConfigPath        string
	EnableHotReload   bool
	EnvironmentPrefix string
}

// NewUnifiedConfigManager creates a new centralized configuration manager
func NewUnifiedConfigManager(opts UnifiedConfigOptions) (*UnifiedConfigManager, error) {
	v := viper.New()

	// Set up environment variable handling with KNIRV_ prefix
	envPrefix := opts.EnvironmentPrefix
	if envPrefix == "" {
		envPrefix = "KNIRV"
	}
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	manager := &UnifiedConfigManager{
		viper:            v,
		components:       make(map[string]ComponentConfig),
		validators:       make(map[string]ConfigValidator),
		role:             opts.Role,
		hotReloadEnabled: opts.EnableHotReload,
		configPath:       opts.ConfigPath,
	}

	// Initialize configuration loading
	if err := manager.initializeConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Set up hot reload if enabled
	if opts.EnableHotReload {
		manager.setupHotReload()
	}

	return manager, nil
}

// RegisterComponent registers a component configuration
func (m *UnifiedConfigManager) RegisterComponent(name string, component ComponentConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Register the component
	m.components[name] = component

	// Set default values
	defaults := component.GetDefaults()
	configKey := component.GetConfigKey()

	for key, value := range defaults {
		fullKey := fmt.Sprintf("%s.%s", configKey, key)
		m.viper.SetDefault(fullKey, value)
	}

	// Bind environment variables
	envMappings := component.GetEnvironmentMappings()
	for configKey, envVar := range envMappings {
		if err := m.viper.BindEnv(configKey, envVar); err != nil {
			log.Printf("Warning: Failed to bind environment variable %s to %s: %v", envVar, configKey, err)
		}
	}

	// Validate the component configuration
	if err := component.Validate(); err != nil {
		return fmt.Errorf("component %s validation failed: %w", name, err)
	}

	log.Printf("Registered component: %s with config key: %s", name, configKey)
	return nil
}

// RegisterValidator registers a configuration validator
func (m *UnifiedConfigManager) RegisterValidator(key string, validator ConfigValidator) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.validators[key] = validator
}

// GetConfig returns the main configuration
func (m *UnifiedConfigManager) GetConfig() (*Config, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var cfg Config

	// Unmarshal the configuration
	if err := m.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate the configuration
	if err := m.validateConfiguration(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// GetComponentConfig returns configuration for a specific component
func (m *UnifiedConfigManager) GetComponentConfig(componentName string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	component, exists := m.components[componentName]
	if !exists {
		return nil, fmt.Errorf("component %s not registered", componentName)
	}

	configKey := component.GetConfigKey()
	config := m.viper.Sub(configKey)
	if config == nil {
		return make(map[string]interface{}), nil
	}

	return config.AllSettings(), nil
}

// SetConfigValue sets a configuration value
func (m *UnifiedConfigManager) SetConfigValue(key string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate the value if a validator exists
	if validator, exists := m.validators[key]; exists {
		if err := validator(key, value); err != nil {
			return fmt.Errorf("validation failed for key %s: %w", key, err)
		}
	}

	m.viper.Set(key, value)
	return nil
}

// GetConfigValue gets a configuration value
func (m *UnifiedConfigManager) GetConfigValue(key string) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.viper.Get(key)
}

// GetString gets a string configuration value
func (m *UnifiedConfigManager) GetString(key string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.viper.GetString(key)
}

// GetInt gets an integer configuration value
func (m *UnifiedConfigManager) GetInt(key string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.viper.GetInt(key)
}

// GetBool gets a boolean configuration value
func (m *UnifiedConfigManager) GetBool(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.viper.GetBool(key)
}

// IsSet checks if a configuration key is set
func (m *UnifiedConfigManager) IsSet(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.viper.IsSet(key)
}

// ReloadConfiguration reloads the configuration from files
func (m *UnifiedConfigManager) ReloadConfiguration() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Println("Reloading configuration...")

	// Re-read configuration files
	if err := m.loadConfigurationFiles(); err != nil {
		return fmt.Errorf("failed to reload configuration files: %w", err)
	}

	// Re-validate all components
	for name, component := range m.components {
		if err := component.Validate(); err != nil {
			log.Printf("Warning: Component %s validation failed after reload: %v", name, err)
		}
	}

	log.Println("Configuration reloaded successfully")
	return nil
}

// initializeConfiguration sets up the initial configuration
func (m *UnifiedConfigManager) initializeConfiguration() error {
	// Set configuration type
	m.viper.SetConfigType("json")

	// Set up search paths
	m.setupConfigPaths()

	// Load configuration files
	if err := m.loadConfigurationFiles(); err != nil {
		return fmt.Errorf("failed to load configuration files: %w", err)
	}

	return nil
}

// setupConfigPaths configures the search paths for configuration files
func (m *UnifiedConfigManager) setupConfigPaths() {
	// Add current directory
	m.viper.AddConfigPath(".")

	// Add user config directory
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		m.viper.AddConfigPath(filepath.Join(userConfigDir, AppName))
	}

	// Add system config directory
	m.viper.AddConfigPath(filepath.Join("/etc", AppName))

	// Add role-specific directory
	if roleDataDir, err := GetDataDir(m.role); err == nil {
		m.viper.AddConfigPath(roleDataDir)
	}
}

// loadConfigurationFiles loads configuration from files
func (m *UnifiedConfigManager) loadConfigurationFiles() error {
	if m.configPath != "" {
		// Use specific config file
		m.viper.SetConfigFile(m.configPath)
		if err := m.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %s: %w", m.configPath, err)
		}
		log.Printf("Loaded configuration from: %s", m.viper.ConfigFileUsed())
	} else {
		// Load default configuration
		m.viper.SetConfigName("default_config")
		if err := m.viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Printf("Warning: Error reading default_config.json: %v", err)
			}
		} else {
			log.Printf("Loaded base configuration from: %s", m.viper.ConfigFileUsed())
		}

		// Load role-specific configuration
		roleConfigName := fmt.Sprintf("%s_config", strings.ToLower(m.role.String()))
		m.viper.SetConfigName(roleConfigName)
		if err := m.viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Printf("Role-specific config file '%s.json' not found. Using defaults.", roleConfigName)
			} else {
				log.Printf("Warning: Error reading role-specific config file '%s.json': %v", roleConfigName, err)
			}
		} else {
			log.Printf("Merged role-specific configuration from: %s", m.viper.ConfigFileUsed())
		}
	}

	return nil
}

// validateConfiguration validates the entire configuration
func (m *UnifiedConfigManager) validateConfiguration(cfg *Config) error {
	// Run registered validators
	for key, validator := range m.validators {
		value := m.viper.Get(key)
		if value != nil {
			if err := validator(key, value); err != nil {
				return fmt.Errorf("validation failed for %s: %w", key, err)
			}
		}
	}

	// Validate all registered components
	for name, component := range m.components {
		if err := component.Validate(); err != nil {
			return fmt.Errorf("component %s validation failed: %w", name, err)
		}
	}

	return nil
}

// setupHotReload sets up configuration hot reloading
func (m *UnifiedConfigManager) setupHotReload() {
	m.viper.WatchConfig()
	m.viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Configuration file changed: %s", e.Name)
		if err := m.ReloadConfiguration(); err != nil {
			log.Printf("Error reloading configuration: %v", err)
		}
	})
	log.Println("Configuration hot reload enabled")
}
