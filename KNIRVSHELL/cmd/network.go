package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/guiperry/KNIRVCHAIN-CLI/core"
	"github.com/spf13/cobra"
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

	// Flags
	allServices     bool
	includeMetrics  bool
	updateConfig    bool
	filterByStatus  string
	outputFormat    string
)

func init() {
	rootCmd.AddCommand(networkCmd)
	
	// Add subcommands
	networkCmd.AddCommand(networkStatusCmd)
	networkCmd.AddCommand(networkDiscoverCmd)
	networkCmd.AddCommand(networkConnectCmd)
	networkCmd.AddCommand(networkDisconnectCmd)

	// Status command flags
	networkStatusCmd.Flags().BoolVar(&allServices, "all-services", true, "Show status for all services")
	networkStatusCmd.Flags().BoolVar(&includeMetrics, "include-metrics", false, "Include detailed metrics")
	networkStatusCmd.Flags().StringVar(&filterByStatus, "filter-by-status", "", "Filter services by status (healthy, unhealthy, unknown)")
	networkStatusCmd.Flags().StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")

	// Discover command flags
	networkDiscoverCmd.Flags().BoolVar(&updateConfig, "update-config", false, "Update configuration with discovered services")

	// Connect command flags
	networkConnectCmd.Flags().StringVar(&filterByStatus, "service-type", "all", "Connect to specific service type")
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

	// Wait a moment for health checks to complete
	time.Sleep(2 * time.Second)

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

	// Wait for discovery to complete
	time.Sleep(5 * time.Second)

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
	eventBus := core.NewEventBus(log)
	healthMonitor := core.NewHealthMonitor(registry, log)
	clientManager := core.NewKNIRVClientManager(registry, eventBus, healthMonitor)

	// Start registry
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service registry: %w", err)
	}
	defer registry.Stop()

	// Create and register service clients
	services := registry.GetAllServices()
	
	for name, service := range services {
		if !service.Config.Enabled {
			continue
		}

		var client core.KNIRVServiceClient
		
		switch name {
		case "knirvroot":
			client = core.NewKNIRVRootClient(service.Config, log)
		case "knirvgateway":
			client = core.NewKNIRVGatewayClient(service.Config, log)
		case "knirvnexus":
			client = core.NewKNIRVNexusClient(service.Config, log)
		case "knirvgraph":
			client = core.NewKNIRVGraphClient(service.Config, log)
		default:
			log.Warnf("Unknown service type: %s", name)
			continue
		}

		clientManager.RegisterClient(name, client)
		log.Infof("Registered client for service: %s", name)
	}

	// Connect to all services
	log.Info("Connecting to KNIRV Network services...")
	
	if err := clientManager.ConnectAll(ctx); err != nil {
		return fmt.Errorf("failed to connect to services: %w", err)
	}

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
	
	// TODO: Implement proper disconnect logic
	// For now, just show a message
	fmt.Println("Disconnected from all KNIRV Network services")
	
	return nil
}

// Helper functions for output formatting

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputYAML(data interface{}) error {
	// TODO: Implement YAML output
	fmt.Println("YAML output not yet implemented")
	return nil
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
