package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	sync "knirv-sync/internal"
)

func main() {
	var (
		configPath   = flag.String("config", "config/doc-sync-config.json", "Path to documentation sync configuration file")
		scanOnly     = flag.Bool("scan", false, "Only scan for documentation gaps, don't sync")
		syncREADMEs  = flag.Bool("readme", false, "Sync only README files")
		syncAPISpecs = flag.Bool("api", false, "Sync only API specifications")
		unifiedAPI   = flag.Bool("unified", false, "Generate only unified API specification")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		reportPath   = flag.String("report", "reports/doc-sync-report.json", "Path for sync report output")
	)

	flag.Parse()

	logger := log.New(os.Stdout, "[DOC-SYNC] ", log.LstdFlags)

	if *verbose {
		logger.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Get root path (parent of KNIRVSYNC)
	rootPath, err := filepath.Abs("..")
	if err != nil {
		logger.Fatalf("Failed to determine root path: %v", err)
	}

	logger.Printf("Root path: %s", rootPath)
	logger.Printf("Config file: %s", *configPath)

	// Load configuration
	configData, err := os.ReadFile(*configPath)
	if err != nil {
		logger.Fatalf("Failed to read config file: %v", err)
	}

	var config sync.DocSyncConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		logger.Fatalf("Failed to parse config: %v", err)
	}

	logger.Printf("Loaded configuration for %d services", len(config.Services))

	// Override config based on flags
	if *syncREADMEs {
		config.SyncREADMEs = true
		config.SyncAPISpecs = false
		config.GenerateUnifiedAPI = false
	} else if *syncAPISpecs {
		config.SyncREADMEs = false
		config.SyncAPISpecs = true
		config.GenerateUnifiedAPI = false
	} else if *unifiedAPI {
		config.SyncREADMEs = false
		config.SyncAPISpecs = false
		config.GenerateUnifiedAPI = true
	}

	// Create documentation sync manager
	docSyncManager := sync.NewDocSyncManager(rootPath, &config, logger)

	// Scan for gaps
	if *scanOnly {
		logger.Println("Scanning for documentation gaps...")

		gaps, err := docSyncManager.Scanner.ScanAllServices()
		if err != nil {
			logger.Fatalf("Failed to scan services: %v", err)
		}

		totalGaps := 0
		for service, serviceGaps := range gaps {
			totalGaps += len(serviceGaps)
			logger.Printf("%s: %d gaps found", service, len(serviceGaps))
		}

		logger.Printf("Total gaps found: %d", totalGaps)

		// Generate gap report
		reportDir := filepath.Dir(*reportPath)
		gapReportPath := filepath.Join(reportDir, "doc-gaps-report.md")

		if err := os.MkdirAll(reportDir, 0755); err != nil {
			logger.Fatalf("Failed to create report directory: %v", err)
		}

		if err := docSyncManager.Scanner.GenerateGapReport(gaps, gapReportPath); err != nil {
			logger.Fatalf("Failed to generate gap report: %v", err)
		}

		logger.Printf("Gap report generated: %s", gapReportPath)
		return
	}

	// Perform synchronization
	logger.Println("Starting documentation synchronization...")

	results, err := docSyncManager.SyncAllDocumentation()
	if err != nil {
		logger.Printf("Warning: synchronization completed with errors: %v", err)
	}

	// Generate sync report
	reportData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		logger.Fatalf("Failed to marshal report: %v", err)
	}

	reportDir := filepath.Dir(*reportPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Fatalf("Failed to create report directory: %v", err)
	}

	if err := os.WriteFile(*reportPath, reportData, 0644); err != nil {
		logger.Fatalf("Failed to write report: %v", err)
	}

	logger.Printf("Synchronization report saved: %s", *reportPath)

	// Summary
	successful := 0
	failed := 0
	totalFiles := 0

	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalFiles += result.FilesUpdated
	}

	logger.Printf("=== Synchronization Summary ===")
	logger.Printf("Total operations: %d", len(results))
	logger.Printf("Successful: %d", successful)
	logger.Printf("Failed: %d", failed)
	logger.Printf("Files updated: %d", totalFiles)

	if failed > 0 {
		logger.Println("Some operations failed. Check the report for details.")
		os.Exit(1)
	}

	logger.Println("Documentation synchronization completed successfully!")
}
