package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Migration represents a database schema change
type Migration struct {
	Version     int
	Description string
	UpSQL       string
	DownSQL     string
}

// SQLiteMigrator handles database schema migrations
type SQLiteMigrator struct {
	db         *sql.DB
	migrations []Migration
}

// NewSQLiteMigrator creates a new migrator for SQLite
func NewSQLiteMigrator(db *sql.DB) (*SQLiteMigrator, error) {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	return &SQLiteMigrator{
		db:         db,
		migrations: make([]Migration, 0),
	}, nil
}

// AddMigration adds a migration to the migrator
func (m *SQLiteMigrator) AddMigration(version int, description, upSQL, downSQL string) {
	m.migrations = append(m.migrations, Migration{
		Version:     version,
		Description: description,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
	})
}

// GetCurrentVersion gets the current database schema version
func (m *SQLiteMigrator) GetCurrentVersion() (int, error) {
	var version int
	err := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, nil
}

// MigrateUp applies all pending migrations
func (m *SQLiteMigrator) MigrateUp() error {
	currentVersion, err := m.GetCurrentVersion()
	if err != nil {
		return err
	}

	log.Printf("Current database version: %d", currentVersion)

	// Sort migrations by version (ascending)
	sortMigrations(m.migrations)

	// Apply pending migrations
	for _, migration := range m.migrations {
		if migration.Version <= currentVersion {
			continue // Skip already applied migrations
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Description)

		// Begin transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Apply migration
		_, err = tx.Exec(migration.UpSQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		// Record migration
		_, err = tx.Exec(
			"INSERT INTO migrations (version, description, applied_at) VALUES (?, ?, ?)",
			migration.Version,
			migration.Description,
			time.Now(),
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("Successfully applied migration %d", migration.Version)
	}

	newVersion, err := m.GetCurrentVersion()
	if err != nil {
		return err
	}

	log.Printf("Database migrated from version %d to %d", currentVersion, newVersion)
	return nil
}

// MigrateDown rolls back migrations to the specified version
func (m *SQLiteMigrator) MigrateDown(targetVersion int) error {
	currentVersion, err := m.GetCurrentVersion()
	if err != nil {
		return err
	}

	if targetVersion >= currentVersion {
		log.Printf("Current version (%d) is already <= target version (%d). Nothing to do.", currentVersion, targetVersion)
		return nil
	}

	log.Printf("Rolling back database from version %d to %d", currentVersion, targetVersion)

	// Sort migrations by version (descending for rollback)
	sortMigrationsDesc(m.migrations)

	// Apply rollbacks
	for _, migration := range m.migrations {
		if migration.Version <= targetVersion || migration.Version > currentVersion {
			continue // Skip migrations that are not in the rollback range
		}

		log.Printf("Rolling back migration %d: %s", migration.Version, migration.Description)

		// Begin transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Apply rollback
		_, err = tx.Exec(migration.DownSQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to roll back migration %d: %w", migration.Version, err)
		}

		// Remove migration record
		_, err = tx.Exec("DELETE FROM migrations WHERE version = ?", migration.Version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to remove migration record %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit rollback %d: %w", migration.Version, err)
		}

		log.Printf("Successfully rolled back migration %d", migration.Version)
	}

	newVersion, err := m.GetCurrentVersion()
	if err != nil {
		return err
	}

	log.Printf("Database rolled back from version %d to %d", currentVersion, newVersion)
	return nil
}

// ListAppliedMigrations lists all applied migrations
func (m *SQLiteMigrator) ListAppliedMigrations() ([]struct {
	Version     int
	Description string
	AppliedAt   time.Time
}, error) {
	rows, err := m.db.Query("SELECT version, description, applied_at FROM migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []struct {
		Version     int
		Description string
		AppliedAt   time.Time
	}

	for rows.Next() {
		var migration struct {
			Version     int
			Description string
			AppliedAt   time.Time
		}
		if err := rows.Scan(&migration.Version, &migration.Description, &migration.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return migrations, nil
}

// Helper functions for sorting migrations
func sortMigrations(migrations []Migration) {
	// Simple bubble sort for small number of migrations
	for i := 0; i < len(migrations); i++ {
		for j := i + 1; j < len(migrations); j++ {
			if migrations[i].Version > migrations[j].Version {
				migrations[i], migrations[j] = migrations[j], migrations[i]
			}
		}
	}
}

func sortMigrationsDesc(migrations []Migration) {
	// Simple bubble sort for small number of migrations (descending)
	for i := 0; i < len(migrations); i++ {
		for j := i + 1; j < len(migrations); j++ {
			if migrations[i].Version < migrations[j].Version {
				migrations[i], migrations[j] = migrations[j], migrations[i]
			}
		}
	}
}
