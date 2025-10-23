package main

import (
	"flag"
	"log"
	"os"

	"KNIRVORACLE/economics"
)

func main() {
	// Command line flags
	var (
		port        = flag.String("port", "8090", "Port to run the economics service on")
		configFile  = flag.String("config", "", "Path to configuration file")
		showHelp    = flag.Bool("help", false, "Show help message")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showHelp {
		showHelpMessage()
		return
	}

	if *showVersion {
		showVersionInfo()
		return
	}

	// Load configuration
	var config *economics.ServiceConfig
	if *configFile != "" {
		// In a real implementation, you would load from file
		log.Printf("Loading configuration from file: %s", *configFile)
		config = economics.GetDefaultConfig()
	} else {
		config = economics.LoadConfigFromEnv()
	}

	// Override port if specified
	if *port != "8090" {
		config.Port = *port
	}

	log.Printf("Starting KNIRV Economics Service with configuration:")
	log.Printf("  Port: %s", config.Port)
	log.Printf("  NRN Contract: %s", config.NRNContract)
	log.Printf("  XION RPC: %s", config.XionRPC)
	log.Printf("  KNIRVCHAIN URL: %s", config.ComponentConfig.KNIRVChainURL)
	log.Printf("  KNIRVNEXUS URL: %s", config.ComponentConfig.KNIRVNexusURL)
	log.Printf("  KNIRVORACLE URL: %s", config.ComponentConfig.KNIRVRootURL)
	log.Printf("  KNIRVGRAPH URL: %s", config.ComponentConfig.KNIRVGraphURL)

	// Create and start the service
	service, err := economics.NewEconomicsService(config)
	if err != nil {
		log.Fatalf("Failed to create economics service: %v", err)
	}

	// Start the service (this blocks until shutdown)
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start economics service: %v", err)
	}
}

func showHelpMessage() {
	log.Println("KNIRV Economics Service")
	log.Println("")
	log.Println("Usage:")
	log.Println("  main [options]")
	log.Println("")
	log.Println("Options:")
	log.Println("  -port string")
	log.Println("        Port to run the economics service on (default \"8090\")")
	log.Println("  -config string")
	log.Println("        Path to configuration file")
	log.Println("  -help")
	log.Println("        Show this help message")
	log.Println("  -version")
	log.Println("        Show version information")
	log.Println("")
	log.Println("Environment Variables:")
	log.Println("  ECONOMICS_PORT        Port for the service (default: 8090)")
	log.Println("  NRN_CONTRACT          NRN token contract address")
	log.Println("  XION_RPC              XION RPC endpoint")
	log.Println("  KNIRVCHAIN_URL        KNIRVCHAIN service URL")
	log.Println("  KNIRVNEXUS_URL        KNIRVNEXUS service URL")
	log.Println("  KNIRVORACLE_URL         KNIRVORACLE service URL")
	log.Println("  KNIRVGRAPH_URL        KNIRVGRAPH service URL")
	log.Println("  ECONOMICS_DB_PATH     Database path for economics data")
	log.Println("")
	log.Println("API Endpoints:")
	log.Println("  POST /economics/skill/invoke           - Process skill invocation")
	log.Println("  POST /economics/llm/register           - Process LLM registration")
	log.Println("  POST /economics/validation/reward      - Process validation reward")
	log.Println("  POST /economics/fees/calculate         - Calculate network fees")
	log.Println("  GET  /economics/metrics                - Get economic metrics")
	log.Println("  GET  /economics/transaction/{id}       - Get transaction details")
	log.Println("  GET  /economics/transactions           - Get transaction list")
	log.Println("  GET  /economics/burn/history           - Get burn event history")
	log.Println("  GET  /economics/burn/total             - Get total burned amount")
	log.Println("  GET  /economics/rules                  - Get economic rules")
	log.Println("  PUT  /economics/rules                  - Update economic rules")
	log.Println("  GET  /economics/service/{service}/metrics - Get service-specific metrics")
	log.Println("  GET  /economics/integration/status     - Get integration status")
	log.Println("  GET  /economics/info                   - Get service information")
	log.Println("  GET  /economics/health                 - Health check")
}

func showVersionInfo() {
	log.Println("KNIRV Economics Service")
	log.Println("Version: 1.0.0")
	log.Println("Build: Month 11 Implementation")
	log.Println("Go Version:", os.Getenv("GO_VERSION"))
	log.Println("Description: Unified token economics system for KNIRV network")
	log.Println("")
	log.Println("Features:")
	log.Println("  - Token economics management")
	log.Println("  - Skill invocation cost processing")
	log.Println("  - LLM registration fee handling")
	log.Println("  - Validation reward distribution")
	log.Println("  - Network fee calculation")
	log.Println("  - Burn tracking and metrics")
	log.Println("  - Performance-based reward calculation")
	log.Println("  - Integration with all KNIRV components")
	log.Println("  - Real-time economic metrics")
	log.Println("  - RESTful API interface")
}
