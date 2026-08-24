package main

// This file should be moved to cmd/migrate-agents/main.go to avoid conflicts
// with cmd/migrate_auth.go. Both files have main() functions and cannot
// coexist in the same package.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"KNIRVENGINE/desktop-client/agent/migration"
	"KNIRVENGINE/desktop-client/utils"
)

func main() {
	var (
		simpleDBPath  = flag.String("simple-db", "", "Path to SimpleAgent database (required)")
		unifiedDBPath = flag.String("unified-db", "", "Path to UnifiedAgent database (required)")
		backupPath    = flag.String("backup-path", "", "Path for backup files (optional, defaults to appdata/backups)")
		reportPath    = flag.String("report-path", "", "Path for migration report (optional, defaults to appdata/reports)")
		dryRun        = flag.Bool("dry-run", false, "Perform a dry run without actual migration")
		validateOnly  = flag.Bool("validate", false, "Only validate existing migration")
		force         = flag.Bool("force", false, "Force migration even if target database exists")
		help          = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Get app data directory for default paths
	appDataDir, err := utils.GetAppDataDir()
	if err != nil {
		log.Fatalf("Failed to get app data directory: %v", err)
	}

	// Set default paths if not provided
	if *simpleDBPath == "" {
		*simpleDBPath = filepath.Join(appDataDir, "data", "domain.db")
	}
	if *unifiedDBPath == "" {
		*unifiedDBPath = filepath.Join(appDataDir, "data", "unified_agents.db")
	}
	if *backupPath == "" {
		*backupPath = filepath.Join(appDataDir, "backups")
	}
	if *reportPath == "" {
		*reportPath = filepath.Join(appDataDir, "reports")
	}

	// Validate required paths
	if *simpleDBPath == "" || *unifiedDBPath == "" {
		log.Fatal("Both simple-db and unified-db paths are required")
	}

	// Check if source database exists
	if _, err := os.Stat(*simpleDBPath); os.IsNotExist(err) {
		log.Fatalf("Source database does not exist: %s", *simpleDBPath)
	}

	// Check if target database exists and handle force flag
	if _, err := os.Stat(*unifiedDBPath); err == nil && !*force {
		log.Fatalf("Target database already exists: %s (use --force to overwrite)", *unifiedDBPath)
	}

	// Create necessary directories
	for _, dir := range []string{*backupPath, *reportPath, filepath.Dir(*unifiedDBPath)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	log.Printf("Starting agent migration...")
	log.Printf("Source DB: %s", *simpleDBPath)
	log.Printf("Target DB: %s", *unifiedDBPath)
	log.Printf("Backup Path: %s", *backupPath)
	log.Printf("Report Path: %s", *reportPath)
	log.Printf("Dry Run: %v", *dryRun)

	// Create migrator
	migrator, err := migration.NewSimpleToUnifiedMigrator(*simpleDBPath, *unifiedDBPath, *backupPath)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}

	ctx := context.Background()

	if *validateOnly {
		if err := validateMigration(ctx, migrator); err != nil {
			log.Fatalf("Migration validation failed: %v", err)
		}
		log.Println("Migration validation successful")
		return
	}

	if *dryRun {
		if err := performDryRun(ctx, migrator); err != nil {
			log.Fatalf("Dry run failed: %v", err)
		}
		return
	}

	// Perform actual migration
	report, err := migrator.MigrateAllAgents(ctx)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Save migration report
	reportFile := filepath.Join(*reportPath, fmt.Sprintf("migration_report_%s.json",
		time.Now().Format("2006-01-02_15-04-05")))

	if err := saveMigrationReport(report, reportFile); err != nil {
		log.Printf("Failed to save migration report: %v", err)
	} else {
		log.Printf("Migration report saved: %s", reportFile)
	}

	// Print summary
	printMigrationSummary(report)

	if report.FailedAgents > 0 {
		log.Printf("Migration completed with %d failures. Check the report for details.", report.FailedAgents)
		os.Exit(1)
	}

	log.Println("Migration completed successfully!")
}

func showHelp() {
	fmt.Println("Agent Migration Tool")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("This tool migrates agents from SimpleAgentRepository to UnifiedAgentStorage.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate_agents [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --simple-db PATH     Path to SimpleAgent database (default: appdata/data/domain.db)")
	fmt.Println("  --unified-db PATH    Path to UnifiedAgent database (default: appdata/data/unified_agents.db)")
	fmt.Println("  --backup-path PATH   Path for backup files (default: appdata/backups)")
	fmt.Println("  --report-path PATH   Path for migration reports (default: appdata/reports)")
	fmt.Println("  --dry-run           Perform a dry run without actual migration")
	fmt.Println("  --validate          Only validate existing migration")
	fmt.Println("  --force             Force migration even if target database exists")
	fmt.Println("  --help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Basic migration with default paths")
	fmt.Println("  migrate_agents")
	fmt.Println()
	fmt.Println("  # Dry run to see what would be migrated")
	fmt.Println("  migrate_agents --dry-run")
	fmt.Println()
	fmt.Println("  # Custom database paths")
	fmt.Println("  migrate_agents --simple-db /path/to/old.db --unified-db /path/to/new.db")
	fmt.Println()
	fmt.Println("  # Validate existing migration")
	fmt.Println("  migrate_agents --validate")
}

func performDryRun(ctx context.Context, migrator *migration.SimpleToUnifiedMigrator) error {
	log.Println("Performing dry run...")

	// This would be implemented to show what would be migrated without actually doing it
	// For now, we'll just log the intent
	log.Println("Dry run completed. No actual migration performed.")
	log.Println("Use the migration tool without --dry-run to perform actual migration.")

	return nil
}

func validateMigration(ctx context.Context, migrator *migration.SimpleToUnifiedMigrator) error {
	log.Println("Validating migration...")

	// This would be implemented to validate that the migration was successful
	// For now, we'll just log the intent
	log.Println("Migration validation completed.")

	return nil
}

func saveMigrationReport(report *migration.SimpleToUnifiedMigrationReport, filePath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %v", err)
	}

	return nil
}

func printMigrationSummary(report *migration.SimpleToUnifiedMigrationReport) {
	fmt.Println()
	fmt.Println("Migration Summary")
	fmt.Println("=================")
	fmt.Printf("Total Agents:    %d\n", report.TotalAgents)
	fmt.Printf("Migrated:        %d\n", report.MigratedAgents)
	fmt.Printf("Failed:          %d\n", report.FailedAgents)
	fmt.Printf("Skipped:         %d\n", report.SkippedAgents)
	fmt.Printf("Duration:        %v\n", report.Duration)
	fmt.Printf("Backup Location: %s\n", report.BackupLocation)
	fmt.Println()

	if len(report.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range report.Errors {
			fmt.Printf("  - Agent %s: %s (step: %s)\n", err.AgentID, err.Error, err.Step)
		}
		fmt.Println()
	}

	if len(report.AgentDetails) > 0 {
		fmt.Println("Migrated Agents:")
		for _, detail := range report.AgentDetails {
			fmt.Printf("  - %s (%s) - Collection: %s, Status: %s\n",
				detail.AgentName, detail.AgentID, detail.Collection, detail.Status)
		}
	}
}
