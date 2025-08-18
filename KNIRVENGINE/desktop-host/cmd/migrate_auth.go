package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Agentic_Engine/database"
)

func main() {
	var (
		sqliteDBPath  = flag.String("sqlite-db", "data/auth.db", "Path to SQLite authentication database")
		chromemDBPath = flag.String("chromem-db", "data/auth_chromem.db", "Path to chromem-go authentication database")
		dryRun        = flag.Bool("dry-run", false, "Perform a dry run without actual migration")
		force         = flag.Bool("force", false, "Force migration even if target database exists")
		reportPath    = flag.String("report", "", "Path to save migration report (default: auto-generated)")
		help          = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Validate paths
	if err := validatePaths(*sqliteDBPath, *chromemDBPath, *force); err != nil {
		log.Fatalf("Path validation failed: %v", err)
	}

	// Set default report path if not provided
	if *reportPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		*reportPath = fmt.Sprintf("reports/auth_migration_report_%s.json", timestamp)
	}

	// Ensure report directory exists
	reportDir := filepath.Dir(*reportPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Fatalf("Failed to create report directory: %v", err)
	}

	if *dryRun {
		log.Println("Performing dry run migration...")
		if err := performDryRun(*sqliteDBPath, *chromemDBPath); err != nil {
			log.Fatalf("Dry run failed: %v", err)
		}
		log.Println("Dry run completed successfully")
		return
	}

	// Perform actual migration
	log.Println("Starting authentication database migration...")
	report, err := performMigration(*sqliteDBPath, *chromemDBPath)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Save migration report
	if err := saveMigrationReport(report, *reportPath); err != nil {
		log.Printf("Failed to save migration report: %v", err)
	} else {
		log.Printf("Migration report saved to: %s", *reportPath)
	}

	// Print summary
	printMigrationSummary(report)

	if !report.Success {
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Authentication Database Migration Tool")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("This tool migrates authentication data from SQLite to chromem-go.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate_auth.go [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Basic migration with default paths")
	fmt.Println("  go run cmd/migrate_auth.go")
	fmt.Println()
	fmt.Println("  # Dry run to preview migration")
	fmt.Println("  go run cmd/migrate_auth.go --dry-run")
	fmt.Println()
	fmt.Println("  # Custom paths with force overwrite")
	fmt.Println("  go run cmd/migrate_auth.go --sqlite-db /path/to/auth.db --chromem-db /path/to/auth_chromem.db --force")
	fmt.Println()
	fmt.Println("  # Save report to specific location")
	fmt.Println("  go run cmd/migrate_auth.go --report /path/to/report.json")
}

func validatePaths(sqliteDBPath, chromemDBPath string, force bool) error {
	// Check if SQLite database exists
	if _, err := os.Stat(sqliteDBPath); os.IsNotExist(err) {
		return fmt.Errorf("SQLite database not found: %s", sqliteDBPath)
	}

	// Check if chromem-go database already exists
	if _, err := os.Stat(chromemDBPath); err == nil && !force {
		return fmt.Errorf("chromem-go database already exists: %s (use --force to overwrite)", chromemDBPath)
	}

	// Ensure chromem-go database directory exists
	chromemDir := filepath.Dir(chromemDBPath)
	if err := os.MkdirAll(chromemDir, 0755); err != nil {
		return fmt.Errorf("failed to create chromem-go database directory: %v", err)
	}

	return nil
}

func performDryRun(sqliteDBPath, chromemDBPath string) error {
	log.Printf("Dry run: SQLite DB: %s", sqliteDBPath)
	log.Printf("Dry run: chromem-go DB: %s", chromemDBPath)

	// Open SQLite database for analysis
	sqliteDB, err := database.NewAuthDB(sqliteDBPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	// Count users
	var userCount int
	err = sqliteDB.GetDB().QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %v", err)
	}

	// Count roles
	var roleCount int
	err = sqliteDB.GetDB().QueryRow("SELECT COUNT(*) FROM roles").Scan(&roleCount)
	if err != nil {
		return fmt.Errorf("failed to count roles: %v", err)
	}

	// Count tokens
	var tokenCount int
	err = sqliteDB.GetDB().QueryRow("SELECT COUNT(*) FROM api_tokens").Scan(&tokenCount)
	if err != nil {
		return fmt.Errorf("failed to count tokens: %v", err)
	}

	log.Printf("Dry run results:")
	log.Printf("  Users to migrate: %d", userCount)
	log.Printf("  Roles to migrate: %d", roleCount)
	log.Printf("  Tokens to migrate: %d", tokenCount)
	log.Printf("  Estimated migration time: %v", estimateMigrationTime(userCount, roleCount, tokenCount))

	return nil
}

func estimateMigrationTime(users, roles, tokens int) time.Duration {
	// Rough estimation: 100ms per user, 50ms per role, 25ms per token
	totalMs := users*100 + roles*50 + tokens*25
	return time.Duration(totalMs) * time.Millisecond
}

func performMigration(sqliteDBPath, chromemDBPath string) (*database.MigrationReport, error) {
	// Create migrator
	migrator, err := database.NewAuthMigrator(sqliteDBPath, chromemDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Perform migration
	ctx := context.Background()
	report, err := migrator.Migrate(ctx)
	if err != nil {
		return report, fmt.Errorf("migration failed: %v", err)
	}

	return report, nil
}

func saveMigrationReport(report *database.MigrationReport, reportPath string) error {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %v", err)
	}

	if err := os.WriteFile(reportPath, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %v", err)
	}

	return nil
}

func printMigrationSummary(report *database.MigrationReport) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("MIGRATION SUMMARY")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Start Time: %s\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("End Time: %s\n", report.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %v\n", report.Duration)
	fmt.Printf("Success: %t\n", report.Success)

	fmt.Println("\nUSERS:")
	fmt.Printf("  Total: %d\n", report.UsersTotal)
	fmt.Printf("  Migrated: %d\n", report.UsersMigrated)
	fmt.Printf("  Failed: %d\n", report.UsersFailed)

	if len(report.FailedUsers) > 0 {
		fmt.Printf("  Failed Users: %v\n", report.FailedUsers)
	}

	fmt.Println("\nTOKENS:")
	fmt.Printf("  Total: %d\n", report.TokensTotal)
	fmt.Printf("  Migrated: %d\n", report.TokensMigrated)

	if len(report.Errors) > 0 {
		fmt.Println("\nERRORS:")
		for i, err := range report.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}

	if report.BackupPath != "" {
		fmt.Printf("\nBackup created at: %s\n", report.BackupPath)
	}

	fmt.Println("\nRECOMMENDations:")
	if report.Success {
		fmt.Println("  ✓ Migration completed successfully")
		fmt.Println("  ✓ Update application configuration to use chromem-go authentication")
		fmt.Println("  ✓ Test authentication functionality thoroughly")
		if report.TokensTotal > 0 && report.TokensMigrated == 0 {
			fmt.Println("  ⚠ API tokens were not migrated - users will need to recreate them")
		}
	} else {
		fmt.Println("  ✗ Migration failed - check errors above")
		fmt.Println("  ✗ Restore from backup if needed")
		fmt.Println("  ✗ Fix issues and retry migration")
	}

	fmt.Println(strings.Repeat("=", 50))
}
