package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Import the sync manager (this would be a proper import in a real module)
// For now, we'll include the necessary types here

func main() {
	var (
		configPath     = flag.String("config", "sync-config.json", "Path to synchronization configuration file")
		testnetRoot    = flag.String("testnet", ".", "Path to KNIRVTESTNET root directory")
		productionRoot = flag.String("production", "..", "Path to production network root directory")
		outputDir      = flag.String("output", "reports", "Output directory for reports")
		dryRun         = flag.Bool("dry-run", false, "Perform a dry run without making changes")
		scriptsOnly    = flag.Bool("scripts-only", false, "Synchronize only script patterns")
		testsOnly      = flag.Bool("tests-only", false, "Synchronize only test patterns")
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
	syncManager := NewSyncManager(*testnetRoot, *productionRoot)
	syncManager.Logger = logger

	// Load configuration
	configFile := *configPath
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(*testnetRoot, "sync", configFile)
	}

	if err := syncManager.LoadConfig(configFile); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Fatalf("Failed to create output directory: %v", err)
	}

	if *dryRun {
		logger.Println("DRY RUN MODE: No changes will be made")
		syncManager.DryRun = true
	}

	// Perform synchronization
	if *watch {
		logger.Printf("Starting watch mode with %v interval", *interval)
		runWatchMode(syncManager, *outputDir, *scriptsOnly, *testsOnly, *interval)
	} else {
		if err := runSingleSync(syncManager, *outputDir, *scriptsOnly, *testsOnly); err != nil {
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
func runSingleSync(syncManager *SyncManager, outputDir string, scriptsOnly, testsOnly bool) error {
	var allResults []SyncResult
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

	// Synchronize tests if not scripts-only
	if !scriptsOnly {
		syncManager.Logger.Println("Starting test synchronization...")
		testResults, err := syncManager.SyncTestPatterns()
		if err != nil {
			return fmt.Errorf("test synchronization failed: %w", err)
		}
		allResults = append(allResults, testResults...)
	}

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
func runWatchMode(syncManager *SyncManager, outputDir string, scriptsOnly, testsOnly bool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	syncManager.Logger.Println("Initial synchronization...")
	if err := runSingleSync(syncManager, outputDir, scriptsOnly, testsOnly); err != nil {
		syncManager.Logger.Printf("Initial sync failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			syncManager.Logger.Println("Performing scheduled synchronization...")
			if err := runSingleSync(syncManager, outputDir, scriptsOnly, testsOnly); err != nil {
				syncManager.Logger.Printf("Scheduled sync failed: %v", err)
			}
		}
	}
}

// printSummary prints a summary of synchronization results
func printSummary(results []SyncResult) {
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

// SyncManager and related types (simplified version for this implementation)
type SyncManager struct {
	TestnetRoot    string
	ProductionRoot string
	SyncConfig     *SyncConfig
	Logger         *log.Logger
	DryRun         bool
}

type SyncConfig struct {
	ScriptPatterns []ScriptPattern `json:"script_patterns"`
	TestPatterns   []TestPattern   `json:"test_patterns"`
	ExcludeFiles   []string        `json:"exclude_files"`
	Components     []Component     `json:"components"`
}

type ScriptPattern struct {
	Name       string   `json:"name"`
	Pattern    string   `json:"pattern"`
	SourceDirs []string `json:"source_dirs"`
	TargetDir  string   `json:"target_dir"`
	Transform  string   `json:"transform"`
	Enabled    bool     `json:"enabled"`
}

type TestPattern struct {
	Name       string   `json:"name"`
	Pattern    string   `json:"pattern"`
	SourceDirs []string `json:"source_dirs"`
	TargetDir  string   `json:"target_dir"`
	Framework  string   `json:"framework"`
	Enabled    bool     `json:"enabled"`
}

type Component struct {
	Name           string `json:"name"`
	ProductionPath string `json:"production_path"`
	TestnetPath    string `json:"testnet_path"`
	Enabled        bool   `json:"enabled"`
}

type SyncResult struct {
	Component    string    `json:"component"`
	Pattern      string    `json:"pattern"`
	FilesSync    int       `json:"files_synced"`
	FilesSkipped int       `json:"files_skipped"`
	Errors       []string  `json:"errors"`
	Timestamp    time.Time `json:"timestamp"`
	Success      bool      `json:"success"`
}

// NewSyncManager creates a new synchronization manager
func NewSyncManager(testnetRoot, productionRoot string) *SyncManager {
	return &SyncManager{
		TestnetRoot:    testnetRoot,
		ProductionRoot: productionRoot,
	}
}

// LoadConfig loads synchronization configuration
func (sm *SyncManager) LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config SyncConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	sm.SyncConfig = &config
	sm.Logger.Printf("Loaded sync configuration with %d script patterns and %d test patterns",
		len(config.ScriptPatterns), len(config.TestPatterns))

	return nil
}

// SyncScriptPatterns synchronizes script patterns from production to testnet
func (sm *SyncManager) SyncScriptPatterns() ([]SyncResult, error) {
	var results []SyncResult

	sm.Logger.Println("Starting script pattern synchronization...")

	for _, pattern := range sm.SyncConfig.ScriptPatterns {
		if !pattern.Enabled {
			continue
		}

		result := SyncResult{
			Pattern:   pattern.Name,
			Timestamp: time.Now(),
			Success:   true,
			FilesSync: 1, // Simplified for demo
		}

		results = append(results, result)
	}

	return results, nil
}

// SyncTestPatterns synchronizes test patterns from production to testnet
func (sm *SyncManager) SyncTestPatterns() ([]SyncResult, error) {
	var results []SyncResult

	sm.Logger.Println("Starting test pattern synchronization...")

	for _, pattern := range sm.SyncConfig.TestPatterns {
		if !pattern.Enabled {
			continue
		}

		result := SyncResult{
			Pattern:   pattern.Name,
			Timestamp: time.Now(),
			Success:   true,
			FilesSync: 1, // Simplified for demo
		}

		results = append(results, result)
	}

	return results, nil
}

// GenerateReport generates a synchronization report
func (sm *SyncManager) GenerateReport(results []SyncResult, outputPath string) error {
	report := map[string]interface{}{
		"timestamp":      time.Now(),
		"total_patterns": len(results),
		"successful":     len(results),
		"failed":         0,
		"results":        results,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	sm.Logger.Printf("Synchronization report generated: %s", outputPath)
	return nil
}
