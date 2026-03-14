package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	automation "knirv-testnet-orchestrator"
)

func main() {
	// Command line flags
	var (
		configFile = flag.String("config", "orchestrator.json", "Configuration file path")
		testType   = flag.String("type", "all", "Test type to run (all, e2e, performance, security, cortex)")
		duration   = flag.Duration("duration", 30*time.Minute, "Maximum test duration")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		dryRun     = flag.Bool("dry-run", false, "Perform dry run without executing tests")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Println("Starting KNIRV Testnet Orchestrator...")

	// Create orchestrator configuration
	config := &automation.OrchestratorConfig{
		TestnetEndpoint:    "http://localhost:8888",
		MaxConcurrentTests: 5,
		DefaultTimeout:     5 * time.Minute,
		RetryAttempts:      3,
		MetricsInterval:    10 * time.Second,
		ReportingEnabled:   true,
	}

	// Load configuration from file if provided
	if *configFile != "" {
		if err := loadConfigFromFile(*configFile, config); err != nil {
			log.Printf("Warning: Could not load config file %s: %v", *configFile, err)
			log.Println("Using default configuration")
		}
	}

	// Create orchestrator
	orchestrator := automation.NewTestnetOrchestrator(config)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping orchestrator...")
		cancel()
	}()

	// Initialize orchestrator
	if err := orchestrator.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	if *dryRun {
		log.Println("Dry run mode - skipping actual test execution")
		orchestrator.PrintTestPlan(*testType)
		return
	}

	// Execute tests based on type
	var err error
	switch *testType {
	case "all":
		err = orchestrator.RunAllTests(ctx)
	case "e2e":
		err = orchestrator.RunE2ETests(ctx)
	case "performance":
		err = orchestrator.RunPerformanceTests(ctx)
	case "security":
		err = orchestrator.RunSecurityTests(ctx)
	case "cortex":
		err = orchestrator.RunCortexTests(ctx)
	default:
		log.Fatalf("Unknown test type: %s", *testType)
	}

	if err != nil {
		log.Fatalf("Test execution failed: %v", err)
	}

	// Generate final report
	if err := orchestrator.GenerateReport(); err != nil {
		log.Printf("Warning: Failed to generate report: %v", err)
	}

	log.Println("Orchestrator completed successfully")
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(filename string, config *automation.OrchestratorConfig) error {
	// Implementation would load from JSON file
	// For now, just return nil to use defaults
	return nil
}
