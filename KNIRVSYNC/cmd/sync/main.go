package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Import the sync manager
import (
	"knirv-sync/internal"
)

func main() {
	var (
		configPath     = flag.String("config", "sync-config.json", "Path to synchronization configuration file")
		testnetRoot    = flag.String("testnet", ".", "Path to KNIRVTESTNET root directory")
		productionRoot = flag.String("production", "..", "Path to production network root directory")
		outputDir      = flag.String("output", "reports", "Output directory for reports")
		dryRun         = flag.Bool("dry-run", false, "Perform a dry run without making changes")
		scriptsOnly    = flag.Bool("scripts-only", false, "Synchronize only script patterns")
		testsOnly      = flag.Bool("tests-only", false, "Synchronize only test patterns")
		skipUnitTests  = flag.Bool("skip-unit-tests", false, "Skip unit testing synchronization")
		verbose        = flag.Bool("verbose", false, "Enable verbose logging")
		watch          = flag.Bool("watch", false, "Watch for changes and auto-synchronize")
		interval       = flag.Duration("interval", 5*time.Minute, "Watch interval for auto-synchronization")
	)
	flag.Parse()

	// Setup logging
	logLevel := log.LstdFlags
	if *verbose {
		logLevel = log.LstdFlags | log.Lshortfile
	}
	logger := log.New(os.Stdout, "[SYNC] ", logLevel)

	// Validate paths
	if err := validatePaths(*testnetRoot, *productionRoot, *outputDir); err != nil {
		logger.Fatalf("Path validation failed: %v", err)
	}

	// Create sync manager
	syncManager := sync.NewSyncManager(*testnetRoot, *productionRoot)
	syncManager.Logger = logger

	// Load configuration
	configFile := *configPath
	if !filepath.IsAbs(configFile) {
		// Try the config subdirectory first, then the testnet root
		configConfigPath := filepath.Join(*testnetRoot, "config", configFile)
		if _, err := os.Stat(configConfigPath); os.IsNotExist(err) {
			configFile = filepath.Join(*testnetRoot, configFile)
		} else {
			configFile = configConfigPath
		}
	}

	if err := syncManager.LoadConfig(configFile); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Fatalf("Failed to create output directory: %v", err)
	}

	if *dryRun {
		logger.Println("DRY RUN MODE: Not yet implemented in refactored version")
	}

	// Perform synchronization
	if *watch {
		logger.Printf("Starting watch mode with %v interval", *interval)
		runWatchMode(syncManager, *outputDir, *scriptsOnly, *testsOnly, *skipUnitTests, *interval)
	} else {
		if err := runSingleSync(syncManager, *outputDir, *scriptsOnly, *testsOnly, *skipUnitTests); err != nil {
			logger.Fatalf("Synchronization failed: %v", err)
		}
	}
}

// validatePaths validates that required paths exist
func validatePaths(testnetRoot, productionRoot, outputDir string) error {
	if _, err := os.Stat(testnetRoot); os.IsNotExist(err) {
		return fmt.Errorf("testnet root directory does not exist: %s", testnetRoot)
	}

	if _, err := os.Stat(productionRoot); os.IsNotExist(err) {
		return fmt.Errorf("production root directory does not exist: %s", productionRoot)
	}

	return nil
}

// runSingleSync performs a single synchronization run
func runSingleSync(syncManager *sync.SyncManager, outputDir string, scriptsOnly, testsOnly, skipUnitTests bool) error {
	var allResults []sync.SyncResult
	timestamp := time.Now().Format("20060102-150405")

	// Synchronize scripts if not tests-only
	if !testsOnly {
		syncManager.Logger.Println("Starting script synchronization...")
		scriptResults, err := syncManager.SyncScriptPatterns()
		if err != nil {
			return fmt.Errorf("script synchronization failed: %w", err)
		}
		allResults = append(allResults, scriptResults...)
	}

	// Synchronize tests if not scripts-only and not skip-unit-tests
	if !scriptsOnly && !skipUnitTests {
		syncManager.Logger.Println("Starting test synchronization...")
		testResults, err := syncManager.SyncTestPatterns()
		if err != nil {
			return fmt.Errorf("test synchronization failed: %w", err)
		}
		allResults = append(allResults, testResults...)
	}

	// Synchronize API references
	syncManager.Logger.Println("Starting API synchronization...")
	apiResults, err := syncManager.SyncAPIPatterns()
	if err != nil {
		return fmt.Errorf("API synchronization failed: %w", err)
	}
	allResults = append(allResults, apiResults...)

	// Generate report
	reportPath := filepath.Join(outputDir, fmt.Sprintf("sync-report-%s.json", timestamp))
	if err := syncManager.GenerateReport(allResults, reportPath); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Print summary
	printSummary(allResults)
	return nil
}

// runWatchMode runs continuous synchronization with file watching
func runWatchMode(syncManager *sync.SyncManager, outputDir string, scriptsOnly, testsOnly, skipUnitTests bool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	syncManager.Logger.Println("Initial synchronization...")
	if err := runSingleSync(syncManager, outputDir, scriptsOnly, testsOnly, skipUnitTests); err != nil {
		syncManager.Logger.Printf("Initial sync failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			syncManager.Logger.Println("Performing scheduled synchronization...")
			if err := runSingleSync(syncManager, outputDir, scriptsOnly, testsOnly, skipUnitTests); err != nil {
				syncManager.Logger.Printf("Scheduled sync failed: %v", err)
			}
		}
	}
}

// printSummary prints a summary of synchronization results
func printSummary(results []sync.SyncResult) {
	successful := 0
	failed := 0
	totalFiles := 0

	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalFiles += result.FilesSync
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SYNCHRONIZATION SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total Patterns: %d\n", len(results))
	fmt.Printf("Successful: %d\n", successful)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Files Synchronized: %d\n", totalFiles)

	if failed > 0 {
		fmt.Println("\nFAILED PATTERNS:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("- %s (%s): %v\n", result.Pattern, result.Component, result.Errors)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}

// Use the imported SyncManager from internal package

// Types are now imported from the internal package
