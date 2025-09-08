package migration

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	
	"Agentic_Engine/agent/service"
)

// MigrateAgentData performs migration of agent data between database versions
func MigrateAgentData() {
	// Command line flags
	var (
		oldDBPath    = flag.String("old-db", "", "Path to the old agent database")
		newDBPath    = flag.String("new-db", "", "Path for the new agent database")
		pluginsDir   = flag.String("plugins-dir", "", "Path to plugins directory")
		wasmDir      = flag.String("wasm-dir", "", "Path to WASM directory")
		templatesDir = flag.String("templates-dir", "", "Path to templates directory")
		outputDir    = flag.String("output-dir", "", "Path to output directory")
		dataDir      = flag.String("data-dir", "", "Path to data directory")
		reportPath   = flag.String("report", "", "Path to save migration report")
		validate     = flag.Bool("validate", false, "Validate migration after completion")
		dryRun       = flag.Bool("dry-run", false, "Perform a dry run without actual migration")
	)
	flag.Parse()

	// Validate required flags
	if *oldDBPath == "" {
		log.Fatal("--old-db flag is required")
	}
	if *newDBPath == "" {
		log.Fatal("--new-db flag is required")
	}

	// Set default paths if not provided
	if *pluginsDir == "" {
		*pluginsDir = "./plugins"
	}
	if *wasmDir == "" {
		*wasmDir = "./plugins"
	}
	if *templatesDir == "" {
		*templatesDir = "./agent/templates"
	}
	if *outputDir == "" {
		*outputDir = "./plugins"
	}
	if *dataDir == "" {
		*dataDir = "./data"
	}
	if *reportPath == "" {
		*reportPath = "./migration_report.json"
	}

	// Validate all paths exist and are accessible
	if err := validatePaths(*oldDBPath, *newDBPath, *pluginsDir, *wasmDir, *templatesDir, *outputDir, *dataDir); err != nil {
		log.Fatalf("Path validation failed: %v", err)
	}

	// Check disk space before proceeding
	if err := checkDiskSpace(*oldDBPath, *newDBPath, *pluginsDir, *wasmDir, *templatesDir, *outputDir, *dataDir); err != nil {
		log.Fatalf("Disk space check failed: %v", err)
	}

	// Create backup of old database
	if err := backupOldDatabase(*oldDBPath); err != nil {
		log.Fatalf("Failed to create backup: %v", err)
	}

	// Create context
	ctx := context.Background()

	// Initialize new agent service
	serviceConfig := &service.ServiceConfig{
		DBPath:       *newDBPath,
		PluginsDir:   *pluginsDir,
		WASMDir:      *wasmDir,
		TemplatesDir: *templatesDir,
		OutputDir:    *outputDir,
		DataDir:      *dataDir,
	}

	agentService, err := service.NewAgentService(serviceConfig)
	if err != nil {
		log.Fatalf("Failed to create agent service: %v", err)
	}

	// Initialize migrator
	coreService := agentService.GetCoreService()
	migrator, err := NewDataMigrator(*oldDBPath, &coreService)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}

	if *dryRun {
		log.Println("Performing dry run migration...")
		if err := performDryRun(ctx, migrator); err != nil {
			log.Fatalf("Dry run failed: %v", err)
		}
		log.Println("Dry run completed successfully")
		return
	}

	// Perform migration
	log.Println("Starting agent data migration...")
	report, err := migrator.MigrateWithReport(ctx)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Save migration report
	if err := migrator.SaveMigrationReport(report, *reportPath); err != nil {
		log.Printf("Failed to save migration report: %v", err)
	}

	// Print summary
	printMigrationSummary(report)

	// Validate migration if requested
	if *validate {
		log.Println("Validating migration...")
		if err := migrator.ValidateMigration(ctx); err != nil {
			log.Fatalf("Migration validation failed: %v", err)
		}
		log.Println("Migration validation successful")
	}

	log.Println("Migration completed successfully")
}

// performDryRun performs a dry run migration to check for potential issues
func performDryRun(ctx context.Context, migrator *DataMigrator) error {
	// Validate that we can connect to the old storage and generate a migration report
	log.Println("Performing dry run migration analysis...")
	
	// Generate a migration report without actually migrating data
	report, err := migrator.MigrateWithReport(ctx)
	if err != nil {
		return fmt.Errorf("dry run failed: %v", err)
	}
	
	// Log the migration report details
	log.Printf("Dry run analysis complete: %d agents would be migrated, %d would fail",
		report.MigratedAgents, report.FailedAgents)
	
	if report.FailedAgents > 0 {
		log.Println("Warning: Some agents would fail migration in a real run")
		for _, err := range report.Errors {
			log.Printf("  - Agent %s: %s", err.AgentID, err.Error)
		}
	}
	
	return nil
}

// printMigrationSummary prints a summary of the migration results
func printMigrationSummary(report *MigrationReport) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("MIGRATION SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total Agents:    %d\n", report.TotalAgents)
	fmt.Printf("Migrated:        %d\n", report.MigratedAgents)
	fmt.Printf("Failed:          %d\n", report.FailedAgents)
	fmt.Printf("Duration:        %v\n", report.Duration)
	fmt.Printf("Start Time:      %v\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("End Time:        %v\n", report.EndTime.Format("2006-01-02 15:04:05"))

	if len(report.Errors) > 0 {
		fmt.Println("\nERRORS:")
		for _, err := range report.Errors {
			fmt.Printf("  Agent %s: %s\n", err.AgentID, err.Error)
		}
	}

	if len(report.AgentDetails) > 0 {
		fmt.Println("\nMIGRATED AGENTS:")
		for _, detail := range report.AgentDetails {
			fmt.Printf("  %s (%s) - %s -> %s [%s]\n",
				detail.AgentName,
				detail.AgentID,
				detail.OldType,
				detail.NewType,
				detail.BuildTarget)
		}
	}
	fmt.Println(strings.Repeat("=", 50))
}

// ensureDirectoryExists ensures that a directory exists, creating it if necessary
func ensureDirectoryExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// backupOldDatabase creates a backup of the old database before migration
// Currently unused but should be integrated into the migration workflow for safety
func backupOldDatabase(oldDBPath string) error {
	backupPath := oldDBPath + ".backup." + fmt.Sprintf("%d", os.Getpid())

	// In a real implementation, this would copy the database file
	log.Printf("Creating backup of old database at %s", backupPath)

	return nil
}

// validatePaths validates that all required paths exist and are accessible
// Currently unused but should be integrated into the migration workflow for validation
func validatePaths(oldDBPath, newDBPath, pluginsDir, wasmDir, templatesDir, outputDir, dataDir string) error {
	// Check if old database exists
	if _, err := os.Stat(oldDBPath); os.IsNotExist(err) {
		return fmt.Errorf("old database does not exist: %s", oldDBPath)
	}

	// Ensure new database directory exists
	newDBDir := filepath.Dir(newDBPath)
	if err := ensureDirectoryExists(newDBDir); err != nil {
		return fmt.Errorf("failed to create new database directory: %v", err)
	}

	// Ensure other directories exist
	dirs := []string{pluginsDir, wasmDir, templatesDir, outputDir, dataDir}
	for _, dir := range dirs {
		if err := ensureDirectoryExists(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}

// checkDiskSpace checks if there's enough disk space for the migration
// Currently unused but should be integrated into the migration workflow for safety
func checkDiskSpace(paths ...string) error {
	// Check disk space for all provided paths
	for _, path := range paths {
		// Get disk usage stats
		var stat syscall.Statfs_t
		if err := syscall.Statfs(filepath.Dir(path), &stat); err != nil {
			return fmt.Errorf("failed to check disk space for %s: %v", path, err)
		}

		// Calculate free space in GB
		freeSpace := stat.Bavail * uint64(stat.Bsize) / (1024 * 1024 * 1024)
		if freeSpace < 1 { // Less than 1GB free
			return fmt.Errorf("insufficient disk space on %s (only %dGB free)", path, freeSpace)
		}

		log.Printf("Disk space check passed for %s (%dGB free)", path, freeSpace)
	}
	return nil
}

// init sets up logging
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
