package xion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"KNIRVORACLE/economics"
)

// XIONNetworkMonitorDemo demonstrates integration with existing KNIRV Network Monitor
func XIONNetworkMonitorDemo() {
	log.Println("🚀 Starting XION Payment Gateway Integration with KNIRV Network Monitor")
	log.Println("=====================================================================")

	// 1. Load configuration
	config, err := loadNetworkMonitorConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize Economics API
	economicsAPI := initializeEconomicsAPI()

	// 3. Initialize XION Payment Gateway
	paymentGateway, err := initializeXIONPaymentGateway(config)
	if err != nil {
		log.Fatalf("Failed to initialize XION payment gateway: %v", err)
	}

	// 4. Initialize XION Integration Service
	integrationService, err := initializeXIONIntegrationService(config, economicsAPI, paymentGateway)
	if err != nil {
		log.Fatalf("Failed to initialize XION integration service: %v", err)
	}

	// 5. Initialize Network Monitor Integration
	networkMonitorIntegration := initializeNetworkMonitorIntegration(
		paymentGateway,
		integrationService,
		config.NetworkMonitor.Endpoint,
	)

	// 6. Start all services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := startServices(ctx, paymentGateway, integrationService, networkMonitorIntegration); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}

	// 7. Register with Network Monitor
	if config.NetworkMonitor.Registration.AutoRegister {
		if err := networkMonitorIntegration.RegisterWithNetworkMonitor(); err != nil {
			log.Printf("Warning: Failed to register with network monitor: %v", err)
		}
	}

	// 8. Demonstrate integration features
	demonstrateIntegrationFeatures(integrationService, networkMonitorIntegration)

	// 9. Wait for shutdown signal
	waitForShutdown(cancel, paymentGateway, integrationService, networkMonitorIntegration)

	log.Println("✅ XION Payment Gateway Integration with Network Monitor completed successfully!")
}

// NetworkMonitorConfig represents the configuration for network monitor integration
type NetworkMonitorConfig struct {
	XIONPaymentGateway XIONGatewayConfig `json:"xion_payment_gateway"`
	KNIRVIntegration   struct {
		Router struct {
			Enabled  bool   `json:"enabled"`
			Endpoint string `json:"endpoint"`
		} `json:"router"`
		Oracle struct {
			Enabled  bool   `json:"enabled"`
			Endpoint string `json:"endpoint"`
		} `json:"oracle"`
	} `json:"knirv_integration"`
	NetworkMonitor struct {
		Enabled      bool   `json:"enabled"`
		Endpoint     string `json:"endpoint"`
		Registration struct {
			AutoRegister bool     `json:"auto_register"`
			ServiceName  string   `json:"service_name"`
			ServiceType  string   `json:"service_type"`
			Critical     bool     `json:"critical"`
			Tags         []string `json:"tags"`
		} `json:"registration"`
		Metrics struct {
			Enabled            bool   `json:"enabled"`
			CollectionInterval string `json:"collection_interval"`
			PrometheusEndpoint string `json:"prometheus_endpoint"`
		} `json:"metrics"`
		HealthChecks struct {
			Enabled  bool   `json:"enabled"`
			Interval string `json:"interval"`
			Endpoint string `json:"endpoint"`
		} `json:"health_checks"`
		StatusReporting struct {
			Enabled                bool   `json:"enabled"`
			Interval               string `json:"interval"`
			Endpoint               string `json:"endpoint"`
			IncludeDetailedMetrics bool   `json:"include_detailed_metrics"`
		} `json:"status_reporting"`
	} `json:"network_monitor"`
}

// loadNetworkMonitorConfig loads the network monitor configuration
func loadNetworkMonitorConfig() (*NetworkMonitorConfig, error) {
	configFile := "config/xion_network_monitor_config.json"

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NetworkMonitorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	log.Printf("✅ Loaded configuration from %s", configFile)
	return &config, nil
}

// initializeEconomicsAPI initializes the economics API
func initializeEconomicsAPI() *economics.EconomicsAPI {
	log.Println("🔧 Initializing Economics API...")
	// In a real implementation, this would initialize the actual economics API
	// For demo purposes, we'll return nil
	return nil
}

// initializeXIONPaymentGateway initializes the XION payment gateway
func initializeXIONPaymentGateway(config *NetworkMonitorConfig) (*XIONPaymentGateway, error) {
	log.Println("🔧 Initializing XION Payment Gateway...")

	gatewayConfig := &config.XIONPaymentGateway
	gateway := NewXIONPaymentGateway(gatewayConfig, nil)

	log.Println("✅ XION Payment Gateway initialized")
	return gateway, nil
}

// initializeXIONIntegrationService initializes the integration service
func initializeXIONIntegrationService(
	config *NetworkMonitorConfig,
	economicsAPI *economics.EconomicsAPI,
	paymentGateway *XIONPaymentGateway,
) (*XIONIntegrationService, error) {
	log.Println("🔧 Initializing XION Integration Service...")

	// Create integration config from network monitor config
	integrationConfig := &XIONIntegrationConfig{
		XIONGateway: config.XIONPaymentGateway,
		// Add other config sections as needed
	}

	service := &XIONIntegrationService{
		config:         integrationConfig,
		paymentGateway: paymentGateway,
		economicsAPI:   economicsAPI,
		activePayments: make(map[string]*PaymentFlow),
		paymentFlows:   make([]PaymentFlowRecord, 0),
	}

	log.Println("✅ XION Integration Service initialized")
	return service, nil
}

// initializeNetworkMonitorIntegration initializes the network monitor integration
func initializeNetworkMonitorIntegration(
	paymentGateway *XIONPaymentGateway,
	integrationService *XIONIntegrationService,
	networkMonitorURL string,
) *XIONNetworkMonitorIntegration {
	log.Println("🔧 Initializing Network Monitor Integration...")

	integration := NewXIONNetworkMonitorIntegration(
		paymentGateway,
		integrationService,
		networkMonitorURL,
	)

	log.Println("✅ Network Monitor Integration initialized")
	return integration
}

// startServices starts all the services
func startServices(
	_ context.Context,
	paymentGateway *XIONPaymentGateway,
	integrationService *XIONIntegrationService,
	networkMonitorIntegration *XIONNetworkMonitorIntegration,
) error {
	log.Println("🚀 Starting all services...")

	// Start payment gateway
	if err := paymentGateway.Start(); err != nil {
		return fmt.Errorf("failed to start payment gateway: %w", err)
	}
	log.Println("✅ XION Payment Gateway started")

	// Start integration service
	if err := integrationService.Start(); err != nil {
		return fmt.Errorf("failed to start integration service: %w", err)
	}
	log.Println("✅ XION Integration Service started")

	// Start network monitor integration
	if err := networkMonitorIntegration.Start(); err != nil {
		return fmt.Errorf("failed to start network monitor integration: %w", err)
	}
	log.Println("✅ Network Monitor Integration started")

	log.Println("🎉 All services started successfully!")
	return nil
}

// demonstrateIntegrationFeatures demonstrates key integration features
func demonstrateIntegrationFeatures(
	integrationService *XIONIntegrationService,
	networkMonitorIntegration *XIONNetworkMonitorIntegration,
) {
	log.Println("\n🌟 Demonstrating Network Monitor Integration Features")
	log.Println("===================================================")

	// 1. Demonstrate payment flow with monitoring
	log.Println("1. 💳 Payment Flow with Real-time Monitoring")
	demonstrateMonitoredPaymentFlow(integrationService)

	// 2. Demonstrate metrics collection
	log.Println("\n2. 📊 Metrics Collection and Reporting")
	demonstrateMetricsCollection(networkMonitorIntegration)

	// 3. Demonstrate health monitoring
	log.Println("\n3. 🏥 Health Monitoring and Status Reporting")
	demonstrateHealthMonitoring(networkMonitorIntegration)

	// 4. Demonstrate alerting integration
	log.Println("\n4. 🚨 Alerting and Notification Integration")
	demonstrateAlertingIntegration()
}

// demonstrateMonitoredPaymentFlow shows a payment flow with monitoring
func demonstrateMonitoredPaymentFlow(integrationService *XIONIntegrationService) {
	userAddress := "xion1demo_user_address"
	usdcAmount := "50000000" // 50 USDC

	log.Printf("   Initiating monitored payment flow...")
	log.Printf("   User: %s", userAddress)
	log.Printf("   Amount: %s USDC", usdcAmount)

	flow, err := integrationService.InitiatePaymentFlow(userAddress, usdcAmount, "email", true)
	if err != nil {
		log.Printf("   ❌ Failed to initiate payment flow: %v", err)
		return
	}

	log.Printf("   ✅ Payment flow initiated: %s", flow.FlowID)
	log.Printf("   📊 Flow will be monitored by Network Monitor")
	log.Printf("   📈 Metrics will be collected and reported")

	// Wait a bit for processing
	time.Sleep(3 * time.Second)

	// Check flow status
	updatedFlow, err := integrationService.GetPaymentFlow(flow.FlowID)
	if err != nil {
		log.Printf("   ❌ Failed to get flow status: %v", err)
		return
	}

	log.Printf("   📋 Current status: %s", updatedFlow.Status)
	log.Printf("   🔄 Steps completed: %d", len(updatedFlow.Steps))
}

// demonstrateMetricsCollection shows metrics collection
func demonstrateMetricsCollection(_ *XIONNetworkMonitorIntegration) {
	log.Printf("   📊 Prometheus metrics being collected:")
	log.Printf("   • xion_payments_total")
	log.Printf("   • xion_payment_flows_active")
	log.Printf("   • xion_nrv_minting_total")
	log.Printf("   • xion_treasury_mints_total")
	log.Printf("   • xion_gateway_uptime_seconds")
	log.Printf("   📈 Metrics available at /metrics endpoint")
	log.Printf("   🎯 Integrated with existing Prometheus setup")
}

// demonstrateHealthMonitoring shows health monitoring
func demonstrateHealthMonitoring(_ *XIONNetworkMonitorIntegration) {
	log.Printf("   🏥 Health checks being performed:")
	log.Printf("   • Payment Gateway health")
	log.Printf("   • Integration Service health")
	log.Printf("   • NRV Minting service health")
	log.Printf("   • Treasury service health")
	log.Printf("   📊 Status reported to Network Monitor every 30s")
	log.Printf("   🎯 Integrated with existing monitoring dashboard")
}

// demonstrateAlertingIntegration shows alerting integration
func demonstrateAlertingIntegration() {
	log.Printf("   🚨 Alerting rules configured:")
	log.Printf("   • Payment failure rate > 5%%")
	log.Printf("   • High payment volume (>1000/hour)")
	log.Printf("   • Service down detection")
	log.Printf("   • Treasury balance low")
	log.Printf("   📢 Alerts sent to existing notification channels")
	log.Printf("   🎯 Integrated with existing AlertManager")
}

// waitForShutdown waits for shutdown signal and gracefully stops services
func waitForShutdown(
	cancel context.CancelFunc,
	paymentGateway *XIONPaymentGateway,
	integrationService *XIONIntegrationService,
	networkMonitorIntegration *XIONNetworkMonitorIntegration,
) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n⏳ Services running... Press Ctrl+C to shutdown")
	<-sigChan

	log.Println("\n🛑 Shutdown signal received, stopping services...")

	// Stop services in reverse order
	if err := networkMonitorIntegration.Stop(); err != nil {
		log.Printf("Error stopping network monitor integration: %v", err)
	}

	if err := integrationService.Stop(); err != nil {
		log.Printf("Error stopping integration service: %v", err)
	}

	if err := paymentGateway.Stop(); err != nil {
		log.Printf("Error stopping payment gateway: %v", err)
	}

	cancel()
	log.Println("✅ All services stopped gracefully")
}
