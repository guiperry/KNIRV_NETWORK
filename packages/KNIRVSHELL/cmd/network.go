package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	networkCmd = &cobra.Command{
		Use:   "network",
		Short: "KNIRV Network operations and management",
		Long: `Network command provides operations for managing and monitoring the KNIRV Network ecosystem.
It includes service discovery, health monitoring, and network-wide operations.`,
	}

	networkStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show network status and service health",
		Long:  `Display the status of all KNIRV Network services including health, connectivity, and metrics.`,
		RunE:  runNetworkStatus,
	}

	networkDiscoverCmd = &cobra.Command{
		Use:   "discover",
		Short: "Discover and register KNIRV Network services",
		Long:  `Perform service discovery to find and register available KNIRV Network services.`,
		RunE:  runNetworkDiscover,
	}

	networkConnectCmd = &cobra.Command{
		Use:   "connect",
		Short: "Connect to all KNIRV Network services",
		Long:  `Establish connections to all configured KNIRV Network services.`,
		RunE:  runNetworkConnect,
	}

	networkDisconnectCmd = &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect from all KNIRV Network services",
		Long:  `Close connections to all KNIRV Network services.`,
		RunE:  runNetworkDisconnect,
	}

	networkSwitchCmd = &cobra.Command{
		Use:   "switch [environment]",
		Short: "Switch between different network environments",
		Long: `Switch between different KNIRV Network environments:
  - public-testnet: Public testnet environment
  - public-production: Public production network
  - local-testnet: Local testnet environment
  - local-production: Local production network

Examples:
  knirv network switch public-testnet
  knirv network switch local-production
  knirv network switch --list  # List available environments`,
		Args: cobra.MaximumNArgs(1),
		RunE: runNetworkSwitch,
	}

	networkListCmd = &cobra.Command{
		Use:   "list",
		Short: "List available network environments",
		Long:  `Display all available network environments and their configurations.`,
		RunE:  runNetworkList,
	}

	// Flags
	allServices      bool
	includeMetrics   bool
	updateConfig     bool
	filterByStatus   string
	outputFormat     string
	listEnvironments bool
)

func init() {
	rootCmd.AddCommand(networkCmd)

	// Add subcommands
	networkCmd.AddCommand(networkStatusCmd)
	networkCmd.AddCommand(networkDiscoverCmd)
	networkCmd.AddCommand(networkConnectCmd)
	networkCmd.AddCommand(networkDisconnectCmd)
	networkCmd.AddCommand(networkSwitchCmd)
	networkCmd.AddCommand(networkListCmd)

	// Status command flags
	networkStatusCmd.Flags().BoolVar(&allServices, "all-services", true, "Show status for all services")
	networkStatusCmd.Flags().BoolVar(&includeMetrics, "include-metrics", false, "Include detailed metrics")
	networkStatusCmd.Flags().StringVar(&filterByStatus, "filter-by-status", "", "Filter services by status (healthy, unhealthy, unknown)")
	networkStatusCmd.Flags().StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")

	// Discover command flags
	networkDiscoverCmd.Flags().BoolVar(&updateConfig, "update-config", false, "Update configuration with discovered services")

	// Connect command flags
	networkConnectCmd.Flags().StringVar(&filterByStatus, "service-type", "all", "Connect to specific service type")

	// Switch command flags
	networkSwitchCmd.Flags().BoolVar(&listEnvironments, "list", false, "List available environments")
}

func runNetworkStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create service registry
	registry := core.NewServiceRegistry(cfg, log)

	// Create health monitor
	healthMonitor := core.NewHealthMonitor(registry, log)

	// Start registry and health monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service registry: %w", err)
	}
	defer registry.Stop()
	healthMonitor.Start(ctx)
	defer healthMonitor.Stop()

	if err := waitForCondition(ctx, 100*time.Millisecond, func() bool {
		return len(healthMonitor.GetHealthStatus()) >= len(registry.GetAllServices())
	}); err != nil && ctx.Err() == nil {
		log.Warnf("Timed out waiting for health results: %v", err)
	}

	// Get health status
	healthStatus := healthMonitor.GetHealthStatus()
	overallHealth := healthMonitor.GetOverallHealth()

	// Filter services if requested
	if filterByStatus != "" {
		filteredStatus := make(map[string]*core.HealthCheckResult)
		for name, result := range healthStatus {
			if string(result.Status) == filterByStatus {
				filteredStatus[name] = result
			}
		}
		healthStatus = filteredStatus
	}

	// Output results
	switch outputFormat {
	case "json":
		return outputJSON(map[string]interface{}{
			"overall_status": overallHealth,
			"services":       healthStatus,
			"timestamp":      time.Now().Format(time.RFC3339),
		})
	case "yaml":
		return outputYAML(map[string]interface{}{
			"overall_status": overallHealth,
			"services":       healthStatus,
			"timestamp":      time.Now().Format(time.RFC3339),
		})
	default:
		return outputTable(healthStatus, overallHealth, includeMetrics)
	}
}

func runNetworkDiscover(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create service registry
	registry := core.NewServiceRegistry(cfg, log)

	// Start discovery
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Info("Starting service discovery...")

	if err := registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service discovery: %w", err)
	}
	defer registry.Stop()

	// Get discovered services
	services := registry.GetAllServices()

	fmt.Printf("Discovered %d services:\n\n", len(services))
	for name, service := range services {
		fmt.Printf("Service: %s\n", name)
		fmt.Printf("  URL: %s\n", service.URL)
		fmt.Printf("  Status: %s\n", service.Status)
		fmt.Printf("  Capabilities: %v\n", service.Capabilities)
		fmt.Printf("  Last Seen: %s\n", service.LastSeen.Format(time.RFC3339))
		fmt.Println()
	}

	if updateConfig {
		log.Info("Updating configuration with discovered services...")
		// TODO: Implement configuration update
		fmt.Println("Configuration update feature coming soon...")
	}

	return nil
}

func runNetworkConnect(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create service registry and clients
	registry := core.NewServiceRegistry(cfg, log)

	// Start registry
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service registry: %w", err)
	}
	defer registry.Stop()

	clientManager := newClientManager(registry)

	// Connect to all services
	log.Info("Connecting to KNIRV Network services...")

	if err := clientManager.ConnectAll(ctx); err != nil {
		return fmt.Errorf("failed to connect to services: %w", err)
	}
	defer func() {
		if err := clientManager.DisconnectAll(); err != nil {
			log.Warnf("Failed to disconnect from services cleanly: %v", err)
		}
	}()

	// Show connection status
	connected := clientManager.GetConnectedClients()
	fmt.Printf("Successfully connected to %d services:\n", len(connected))
	for name := range connected {
		fmt.Printf("  ✓ %s\n", name)
	}

	return nil
}

func runNetworkDisconnect(cmd *cobra.Command, args []string) error {
	log.Info("Disconnecting from KNIRV Network services...")

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	registry := core.NewServiceRegistry(cfg, log)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service registry: %w", err)
	}
	defer registry.Stop()

	clientManager := newClientManager(registry)
	if err := clientManager.DisconnectAll(); err != nil {
		return err
	}

	fmt.Printf("Disconnected %d configured services\n", len(registry.GetAllServices()))

	return nil
}

// NetworkEnvironment represents a network environment configuration
type NetworkEnvironment struct {
	Name        string                 `json:"name"`
	Environment string                 `json:"environment"`
	Description string                 `json:"description"`
	Services    map[string]string      `json:"services"`
	Features    map[string]interface{} `json:"features"`
}

// getNetworkEnvironments returns all available network environments
func getNetworkEnvironments() map[string]NetworkEnvironment {
	return map[string]NetworkEnvironment{
		"public-testnet": {
			Name:        "Public Testnet",
			Environment: "public-testnet",
			Description: "KNIRV Network public testnet environment",
			Services: map[string]string{
				"knirvoracle":     "https://testnet.knirv.com:1317",
				"knirvgateway":    "https://testnet-gateway.knirv.com",
				"knirvserver":     "https://testnet-nexus.knirv.com",
				"knirvgraph":      "https://testnet-graph.knirv.com",
				"knirvrouter":     "https://testnet-router.knirv.com",
				"knirvcontroller": "https://testnet-controller.knirv.com",
			},
			Features: map[string]interface{}{
				"testnet":    true,
				"faucet":     true,
				"debugging":  true,
				"monitoring": true,
			},
		},
		"public-production": {
			Name:        "Public Production",
			Environment: "public-production",
			Description: "KNIRV Network main production environment",
			Services: map[string]string{
				"knirvoracle":     "https://oracle.knirv.com:1317",
				"knirvgateway":    "https://gateway.knirv.com",
				"knirvserver":     "https://nexus.knirv.com",
				"knirvgraph":      "https://graph.knirv.com",
				"knirvrouter":     "https://router.knirv.com",
				"knirvcontroller": "https://controller.knirv.com",
			},
			Features: map[string]interface{}{
				"production":        true,
				"mainnet":           true,
				"monitoring":        true,
				"high_availability": true,
			},
		},
		"local-testnet": {
			Name:        "Local Testnet",
			Environment: "local-testnet",
			Description: "Local development testnet environment",
			Services: map[string]string{
				"knirvoracle":     "http://localhost:1317",
				"knirvgateway":    "http://localhost:8888",
				"knirvserver":     "http://localhost:8084",
				"knirvgraph":      "http://localhost:8082",
				"knirvrouter":     "http://localhost:8086",
				"knirvcontroller": "http://localhost:8088",
			},
			Features: map[string]interface{}{
				"development": true,
				"testnet":     true,
				"local":       true,
				"debugging":   true,
			},
		},
		"local-production": {
			Name:        "Local Production",
			Environment: "local-production",
			Description: "Local production-like environment",
			Services: map[string]string{
				"knirvoracle":     "http://localhost:9317",
				"knirvgateway":    "http://localhost:9888",
				"knirvserver":     "http://localhost:9084",
				"knirvgraph":      "http://localhost:9082",
				"knirvrouter":     "http://localhost:9086",
				"knirvcontroller": "http://localhost:9088",
			},
			Features: map[string]interface{}{
				"production": true,
				"local":      true,
				"testing":    true,
			},
		},
	}
}

func runNetworkSwitch(cmd *cobra.Command, args []string) error {
	environments := getNetworkEnvironments()

	// If --list flag is provided, show available environments
	if listEnvironments {
		return runNetworkList(cmd, args)
	}

	// If no environment specified, show current and available
	if len(args) == 0 {
		// Load current configuration
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		currentEnv := cfg.KNIRV.Network.Environment
		if currentEnv == "" {
			currentEnv = "development" // default
		}

		fmt.Printf("Current environment: %s\n\n", currentEnv)
		fmt.Println("Available environments:")
		for envID, env := range environments {
			marker := "  "
			if envID == currentEnv {
				marker = "* "
			}
			fmt.Printf("%s%s - %s\n", marker, envID, env.Description)
		}
		fmt.Println("\nUse 'knirv network switch <environment>' to switch networks")
		return nil
	}

	targetEnv := args[0]
	env, exists := environments[targetEnv]
	if !exists {
		fmt.Printf("Error: Unknown environment '%s'\n\n", targetEnv)
		fmt.Println("Available environments:")
		for envID, env := range environments {
			fmt.Printf("  %s - %s\n", envID, env.Description)
		}
		return fmt.Errorf("invalid environment: %s", targetEnv)
	}

	// Load current configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update network configuration
	cfg.KNIRV.Network.Environment = targetEnv

	// Update service URLs
	for serviceName, serviceURL := range env.Services {
		switch serviceName {
		case "knirvoracle":
			cfg.KNIRV.Services.KNIRVRoot.URL = serviceURL
		case "knirvgateway":
			cfg.KNIRV.Services.KNIRVGateway.URL = serviceURL
		case "knirvserver":
			cfg.KNIRV.Services.KNIRVNexus.URL = serviceURL
		case "knirvgraph":
			cfg.KNIRV.Services.KNIRVGraph.URL = serviceURL
		}
	}

	// Save updated configuration
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Switched to %s environment\n", env.Name)
	fmt.Printf("  Environment: %s\n", env.Environment)
	fmt.Printf("  Description: %s\n", env.Description)

	// Show updated service URLs
	fmt.Println("\nService endpoints:")
	for serviceName, serviceURL := range env.Services {
		fmt.Printf("  %s: %s\n", serviceName, serviceURL)
	}

	// Show features
	if len(env.Features) > 0 {
		fmt.Println("\nFeatures:")
		for feature, enabled := range env.Features {
			if enabled == true {
				fmt.Printf("  ✓ %s\n", feature)
			}
		}
	}

	return nil
}

func runNetworkList(cmd *cobra.Command, args []string) error {
	environments := getNetworkEnvironments()

	// Load current configuration to show which is active
	cfg, err := config.LoadConfig(cfgFile)
	currentEnv := "development" // default
	if err == nil {
		currentEnv = cfg.KNIRV.Network.Environment
		if currentEnv == "" {
			currentEnv = "development"
		}
	}

	switch outputFormat {
	case "json":
		return outputJSON(environments)
	case "yaml":
		return outputYAML(environments)
	default:
		fmt.Println("Available Network Environments:")
		fmt.Println("==============================")

		for envID, env := range environments {
			marker := "   "
			if envID == currentEnv {
				marker = " * "
			}

			fmt.Printf("%s%s\n", marker, env.Name)
			fmt.Printf("     ID: %s\n", envID)
			fmt.Printf("     Description: %s\n", env.Description)
			fmt.Printf("     Services: %d configured\n", len(env.Services))

			// Show key features
			features := []string{}
			for feature, enabled := range env.Features {
				if enabled == true {
					features = append(features, feature)
				}
			}
			if len(features) > 0 {
				fmt.Printf("     Features: %s\n", strings.Join(features, ", "))
			}
			fmt.Println()
		}

		if currentEnv != "" {
			fmt.Printf("Current environment: %s (marked with *)\n", currentEnv)
		}
		fmt.Println("\nUse 'knirv network switch <environment>' to change networks")
	}

	return nil
}

// Helper functions for output formatting

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputYAML(data interface{}) error {
	body, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}

func outputTable(healthStatus map[string]*core.HealthCheckResult, overallHealth core.ServiceStatus, includeMetrics bool) error {
	fmt.Printf("KNIRV Network Status: %s\n\n", overallHealth)

	if len(healthStatus) == 0 {
		fmt.Println("No services found")
		return nil
	}

	// Table header
	fmt.Printf("%-15s %-10s %-15s %-20s\n", "SERVICE", "STATUS", "RESPONSE TIME", "LAST CHECK")
	fmt.Println("------------------------------------------------------------")

	// Table rows
	for name, result := range healthStatus {
		responseTime := result.ResponseTime.String()
		if result.ResponseTime == 0 {
			responseTime = "N/A"
		}

		lastCheck := result.Timestamp.Format("15:04:05")

		fmt.Printf("%-15s %-10s %-15s %-20s\n",
			name,
			result.Status,
			responseTime,
			lastCheck)

		if includeMetrics && result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
	}

	return nil
}

func newClientManager(registry *core.ServiceRegistry) *core.KNIRVClientManager {
	eventBus := core.NewEventBus(log)
	healthMonitor := core.NewHealthMonitor(registry, log)
	clientManager := core.NewKNIRVClientManager(registry, eventBus, healthMonitor)

	for name, service := range registry.GetAllServices() {
		if service.Config == nil || !service.Config.Enabled {
			continue
		}

		var client core.KNIRVServiceClient
		switch name {
		case "knirvoracle":
			client = core.NewKNIRVRootClient(service.Config, log)
		case "knirvgateway":
			client = core.NewKNIRVGatewayClient(service.Config, log)
		case "knirvserver":
			client = core.NewKNIRVNexusClient(service.Config, log)
		case "knirvgraph":
			client = core.NewKNIRVGraphClient(service.Config, log)
		default:
			log.Warnf("Unknown service type: %s", name)
			continue
		}

		clientManager.RegisterClient(name, client)
	}

	return clientManager
}

func waitForCondition(ctx context.Context, interval time.Duration, condition func() bool) error {
	if condition() {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}
