package config

import (
	"fmt"
	"log"

	"KNIRVCHAIN/config/components"
)

// UnifiedConfigIntegration demonstrates how to integrate the unified configuration manager
// with all KNIRVCHAIN components
type UnifiedConfigIntegration struct {
	manager *UnifiedConfigManager
}

// NewUnifiedConfigIntegration creates a new configuration integration
func NewUnifiedConfigIntegration(role Role, configPath string) (*UnifiedConfigIntegration, error) {
	// Create unified configuration manager
	manager, err := NewUnifiedConfigManager(UnifiedConfigOptions{
		Role:              role,
		ConfigPath:        configPath,
		EnableHotReload:   true,
		EnvironmentPrefix: "KNIRV",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create unified config manager: %w", err)
	}

	integration := &UnifiedConfigIntegration{
		manager: manager,
	}

	// Register all components
	if err := integration.registerAllComponents(); err != nil {
		return nil, fmt.Errorf("failed to register components: %w", err)
	}

	// Set up configuration validators
	integration.setupValidators()

	return integration, nil
}

// registerAllComponents registers all KNIRVCHAIN components with the unified manager
func (i *UnifiedConfigIntegration) registerAllComponents() error {
	log.Println("Registering all components with unified configuration manager...")

	// Register MCP component
	mcpConfig := components.NewMCPConfig()
	if err := i.manager.RegisterComponent("mcp", mcpConfig); err != nil {
		return fmt.Errorf("failed to register MCP component: %w", err)
	}

	// Register Inference component
	inferenceConfig := components.NewInferenceConfig()
	if err := i.manager.RegisterComponent("inference", inferenceConfig); err != nil {
		return fmt.Errorf("failed to register Inference component: %w", err)
	}

	// Register Economics component
	economicsConfig := components.NewEconomicsConfig()
	if err := i.manager.RegisterComponent("economics", economicsConfig); err != nil {
		return fmt.Errorf("failed to register Economics component: %w", err)
	}

	// Register Network Monitor component
	networkMonitorConfig := components.NewNetworkMonitorConfig()
	if err := i.manager.RegisterComponent("network_monitor", networkMonitorConfig); err != nil {
		return fmt.Errorf("failed to register Network Monitor component: %w", err)
	}

	log.Println("All components registered successfully")
	return nil
}

// setupValidators sets up configuration validators for critical values
func (i *UnifiedConfigIntegration) setupValidators() {
	log.Println("Setting up configuration validators...")

	// Port range validator
	portValidator := func(key string, value interface{}) error {
		if port, ok := value.(int); ok {
			if port <= 0 || port > 65535 {
				return fmt.Errorf("port %d out of valid range (1-65535)", port)
			}
		}
		return nil
	}

	// Register port validators
	i.manager.RegisterValidator("port", portValidator)
	i.manager.RegisterValidator("p2p_port", portValidator)
	i.manager.RegisterValidator("wallet_port", portValidator)
	i.manager.RegisterValidator("mcp.port", portValidator)
	i.manager.RegisterValidator("economics.port", portValidator)
	i.manager.RegisterValidator("network_monitor.port", portValidator)

	// Chain ID validator
	chainIDValidator := func(key string, value interface{}) error {
		if chainID, ok := value.(string); ok {
			if chainID == "" {
				return fmt.Errorf("chain ID cannot be empty")
			}
			if len(chainID) < 3 {
				return fmt.Errorf("chain ID too short: %s (minimum 3 characters)", chainID)
			}
		}
		return nil
	}
	i.manager.RegisterValidator("chain_id", chainIDValidator)

	// API key validator
	apiKeyValidator := func(key string, value interface{}) error {
		if apiKey, ok := value.(string); ok {
			if len(apiKey) > 0 && len(apiKey) < 10 {
				return fmt.Errorf("API key too short: %d characters (minimum 10)", len(apiKey))
			}
		}
		return nil
	}
	i.manager.RegisterValidator("inference.api_key", apiKeyValidator)
	i.manager.RegisterValidator("inference.cerebras.api_key", apiKeyValidator)
	i.manager.RegisterValidator("inference.deepseek.api_key", apiKeyValidator)
	i.manager.RegisterValidator("inference.gemini.api_key", apiKeyValidator)

	log.Println("Configuration validators set up successfully")
}

// GetManager returns the unified configuration manager
func (i *UnifiedConfigIntegration) GetManager() *UnifiedConfigManager {
	return i.manager
}

// GetMainConfig returns the main KNIRVCHAIN configuration
func (i *UnifiedConfigIntegration) GetMainConfig() (*Config, error) {
	return i.manager.GetConfig()
}

// GetMCPConfig returns the MCP component configuration
func (i *UnifiedConfigIntegration) GetMCPConfig() (*components.MCPConfig, error) {
	configMap, err := i.manager.GetComponentConfig("mcp")
	if err != nil {
		return nil, err
	}

	var mcpConfig components.MCPConfig
	// In a real implementation, you would use a proper mapping library
	// For now, this is a simplified example
	if enabled, ok := configMap["enabled"].(bool); ok {
		mcpConfig.Enabled = enabled
	}
	if port, ok := configMap["port"].(int); ok {
		mcpConfig.Port = port
	}
	if host, ok := configMap["host"].(string); ok {
		mcpConfig.Host = host
	}

	return &mcpConfig, nil
}

// GetInferenceConfig returns the Inference component configuration
func (i *UnifiedConfigIntegration) GetInferenceConfig() (*components.InferenceConfig, error) {
	configMap, err := i.manager.GetComponentConfig("inference")
	if err != nil {
		return nil, err
	}

	var inferenceConfig components.InferenceConfig
	// Simplified mapping - in practice, use a proper mapping library
	if enabled, ok := configMap["enabled"].(bool); ok {
		inferenceConfig.Enabled = enabled
	}
	if provider, ok := configMap["provider"].(string); ok {
		inferenceConfig.Provider = provider
	}

	return &inferenceConfig, nil
}

// GetEconomicsConfig returns the Economics component configuration
func (i *UnifiedConfigIntegration) GetEconomicsConfig() (*components.EconomicsConfig, error) {
	configMap, err := i.manager.GetComponentConfig("economics")
	if err != nil {
		return nil, err
	}

	var economicsConfig components.EconomicsConfig
	// Simplified mapping
	if enabled, ok := configMap["enabled"].(bool); ok {
		economicsConfig.Enabled = enabled
	}
	if port, ok := configMap["port"].(string); ok {
		economicsConfig.Port = port
	}

	return &economicsConfig, nil
}

// GetNetworkMonitorConfig returns the Network Monitor component configuration
func (i *UnifiedConfigIntegration) GetNetworkMonitorConfig() (*components.NetworkMonitorConfig, error) {
	configMap, err := i.manager.GetComponentConfig("network_monitor")
	if err != nil {
		return nil, err
	}

	var networkMonitorConfig components.NetworkMonitorConfig
	// Simplified mapping
	if enabled, ok := configMap["enabled"].(bool); ok {
		networkMonitorConfig.Enabled = enabled
	}
	if port, ok := configMap["port"].(string); ok {
		networkMonitorConfig.Port = port
	}

	return &networkMonitorConfig, nil
}

// ValidateAllConfigurations validates all component configurations
func (i *UnifiedConfigIntegration) ValidateAllConfigurations() error {
	log.Println("Validating all component configurations...")

	// Get and validate main config
	mainConfig, err := i.GetMainConfig()
	if err != nil {
		return fmt.Errorf("failed to get main config: %w", err)
	}

	// Basic main config validation
	if mainConfig.ChainID == "" {
		return fmt.Errorf("main config validation failed: chain ID cannot be empty")
	}

	// Validate MCP config
	mcpConfig, err := i.GetMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to get MCP config: %w", err)
	}
	if err := mcpConfig.Validate(); err != nil {
		return fmt.Errorf("MCP config validation failed: %w", err)
	}

	// Validate Inference config
	inferenceConfig, err := i.GetInferenceConfig()
	if err != nil {
		return fmt.Errorf("failed to get Inference config: %w", err)
	}
	if err := inferenceConfig.Validate(); err != nil {
		return fmt.Errorf("inference config validation failed: %w", err)
	}

	// Validate Economics config
	economicsConfig, err := i.GetEconomicsConfig()
	if err != nil {
		return fmt.Errorf("failed to get Economics config: %w", err)
	}
	if err := economicsConfig.Validate(); err != nil {
		return fmt.Errorf("economics config validation failed: %w", err)
	}

	// Validate Network Monitor config
	networkMonitorConfig, err := i.GetNetworkMonitorConfig()
	if err != nil {
		return fmt.Errorf("failed to get Network Monitor config: %w", err)
	}
	if err := networkMonitorConfig.Validate(); err != nil {
		return fmt.Errorf("network Monitor config validation failed: %w", err)
	}

	log.Println("All component configurations validated successfully")
	return nil
}

// PrintConfigurationSummary prints a summary of all configurations
func (i *UnifiedConfigIntegration) PrintConfigurationSummary() {
	log.Println("=== KNIRVCHAIN Unified Configuration Summary ===")

	// Main configuration
	if mainConfig, err := i.GetMainConfig(); err == nil {
		log.Printf("Main Config - Chain ID: %s, HTTP Port: %d, P2P Port: %d",
			mainConfig.ChainID, mainConfig.Port, mainConfig.P2PPort)
	}

	// MCP configuration
	if mcpConfig, err := i.GetMCPConfig(); err == nil {
		log.Printf("MCP Config - Enabled: %t, Port: %d, Host: %s",
			mcpConfig.Enabled, mcpConfig.Port, mcpConfig.Host)
	}

	// Inference configuration
	if inferenceConfig, err := i.GetInferenceConfig(); err == nil {
		log.Printf("Inference Config - Enabled: %t, Provider: %s",
			inferenceConfig.Enabled, inferenceConfig.Provider)
	}

	// Economics configuration
	if economicsConfig, err := i.GetEconomicsConfig(); err == nil {
		log.Printf("Economics Config - Enabled: %t, Port: %s",
			economicsConfig.Enabled, economicsConfig.Port)
	}

	// Network Monitor configuration
	if networkMonitorConfig, err := i.GetNetworkMonitorConfig(); err == nil {
		log.Printf("Network Monitor Config - Enabled: %t, Port: %s, Interval: %d",
			networkMonitorConfig.Enabled, networkMonitorConfig.Port, networkMonitorConfig.Interval)
	}

	log.Println("=== End Configuration Summary ===")
}

// ReloadAllConfigurations reloads all configurations from files
func (i *UnifiedConfigIntegration) ReloadAllConfigurations() error {
	log.Println("Reloading all configurations...")

	if err := i.manager.ReloadConfiguration(); err != nil {
		return fmt.Errorf("failed to reload configurations: %w", err)
	}

	// Re-validate after reload
	if err := i.ValidateAllConfigurations(); err != nil {
		return fmt.Errorf("validation failed after reload: %w", err)
	}

	log.Println("All configurations reloaded and validated successfully")
	return nil
}

// Example usage function
func ExampleUsage() {
	// Create unified configuration integration
	integration, err := NewUnifiedConfigIntegration(RoleClient, "")
	if err != nil {
		log.Fatalf("Failed to create configuration integration: %v", err)
	}

	// Validate all configurations
	if err := integration.ValidateAllConfigurations(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Print configuration summary
	integration.PrintConfigurationSummary()

	// Get specific component configurations
	mcpConfig, _ := integration.GetMCPConfig()
	if mcpConfig.Enabled {
		log.Printf("MCP service will run on %s", mcpConfig.GetEndpoint())
	}

	economicsConfig, _ := integration.GetEconomicsConfig()
	if economicsConfig.Enabled {
		log.Printf("Economics service will run on %s", economicsConfig.GetEndpoint())
	}

	log.Println("Unified configuration integration example completed successfully")
}
