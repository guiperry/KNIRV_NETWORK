// desktop/data_migration.go
// Data migration utilities for desktop application upgrades

package desktop

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"KNIRVENGINE/desktop-client/internal/utils"
)

// MigrationManager handles data migration between application versions
type MigrationManager struct {
	currentVersion string
	appDataPath    string
}

// MigrationInfo contains information about a migration
type MigrationInfo struct {
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Date        time.Time `json:"date"`
	Success     bool      `json:"success"`
	BackupPath  string    `json:"backup_path,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(currentVersion string) (*MigrationManager, error) {
	appDataPath, err := utils.GetAppDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get app data directory: %v", err)
	}

	return &MigrationManager{
		currentVersion: currentVersion,
		appDataPath:    appDataPath,
	}, nil
}

// CheckMigrationNeeded checks if data migration is needed
func (mm *MigrationManager) CheckMigrationNeeded() (bool, string, error) {
	// Check for version file
	versionFile := filepath.Join(mm.appDataPath, "version.json")

	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		// No version file exists, this might be a fresh install or very old version
		return mm.checkLegacyData()
	}

	// Read version file
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return false, "", fmt.Errorf("failed to read version file: %v", err)
	}

	var versionInfo struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(data, &versionInfo); err != nil {
		return false, "", fmt.Errorf("failed to parse version file: %v", err)
	}

	// Compare versions
	if versionInfo.Version != mm.currentVersion {
		return true, versionInfo.Version, nil
	}

	return false, "", nil
}

// checkLegacyData checks for legacy data that needs migration
func (mm *MigrationManager) checkLegacyData() (bool, string, error) {
	// Check for common legacy files/directories
	legacyPaths := []string{
		filepath.Join(mm.appDataPath, "data", "auth.db"),
		filepath.Join(mm.appDataPath, "data", "domain.db"),
		filepath.Join(mm.appDataPath, "config"),
		filepath.Join(mm.appDataPath, "plugins"),
	}

	for _, path := range legacyPaths {
		if _, err := os.Stat(path); err == nil {
			// Legacy data found
			return true, "legacy", nil
		}
	}

	return false, "", nil
}

// PerformMigration performs data migration from old version to current version
func (mm *MigrationManager) PerformMigration(fromVersion string) error {
	migrationInfo := &MigrationInfo{
		FromVersion: fromVersion,
		ToVersion:   mm.currentVersion,
		Date:        time.Now(),
	}

	// Create backup before migration
	backupPath, err := mm.createBackup(fromVersion)
	if err != nil {
		migrationInfo.Success = false
		migrationInfo.Error = fmt.Sprintf("backup failed: %v", err)
		mm.saveMigrationInfo(migrationInfo)
		return fmt.Errorf("failed to create backup: %v", err)
	}

	migrationInfo.BackupPath = backupPath

	// Perform version-specific migrations
	switch {
	case fromVersion == "legacy":
		err = mm.migrateLegacyData()
	case strings.HasPrefix(fromVersion, "0."):
		err = mm.migrateFromV0(fromVersion)
	case strings.HasPrefix(fromVersion, "1.0"):
		err = mm.migrateFromV1_0(fromVersion)
	default:
		err = fmt.Errorf("unsupported migration from version %s", fromVersion)
	}

	if err != nil {
		migrationInfo.Success = false
		migrationInfo.Error = err.Error()
		mm.saveMigrationInfo(migrationInfo)
		return fmt.Errorf("migration failed: %v", err)
	}

	// Update version file
	if err := mm.updateVersionFile(); err != nil {
		migrationInfo.Success = false
		migrationInfo.Error = fmt.Sprintf("version update failed: %v", err)
		mm.saveMigrationInfo(migrationInfo)
		return fmt.Errorf("failed to update version file: %v", err)
	}

	migrationInfo.Success = true
	mm.saveMigrationInfo(migrationInfo)

	return nil
}

// createBackup creates a backup of the current data
func (mm *MigrationManager) createBackup(fromVersion string) (string, error) {
	backupDir, err := utils.GetBackupDir()
	if err != nil {
		return "", err
	}

	if err := utils.EnsureDir(backupDir); err != nil {
		return "", err
	}

	// Create timestamped backup directory
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("migration_%s_to_%s_%s", fromVersion, mm.currentVersion, timestamp))

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Copy important directories
	dirsToBackup := []string{"data", "config", "plugins", "mcp-data", "plugin-data"}

	for _, dir := range dirsToBackup {
		srcPath := filepath.Join(mm.appDataPath, dir)
		dstPath := filepath.Join(backupPath, dir)

		if _, err := os.Stat(srcPath); err == nil {
			if err := mm.copyDir(srcPath, dstPath); err != nil {
				return "", fmt.Errorf("failed to backup %s: %v", dir, err)
			}
		}
	}

	return backupPath, nil
}

// copyDir recursively copies a directory
func (mm *MigrationManager) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return mm.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (mm *MigrationManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// migrateLegacyData migrates data from legacy versions
func (mm *MigrationManager) migrateLegacyData() error {
	// Ensure new directory structure exists
	if err := utils.EnsureAppDataDirs(); err != nil {
		return fmt.Errorf("failed to create new directory structure: %v", err)
	}

	// Migrate database files
	oldDataDir := filepath.Join(mm.appDataPath, "data")
	newDataDir, err := utils.GetDatabaseDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(oldDataDir); err == nil {
		// Move database files to new location if they're not already there
		if oldDataDir != newDataDir {
			dbFiles := []string{"auth.db", "domain.db", "inference_engine.db"}
			for _, dbFile := range dbFiles {
				oldPath := filepath.Join(oldDataDir, dbFile)
				newPath := filepath.Join(newDataDir, dbFile)

				if _, err := os.Stat(oldPath); err == nil {
					if _, err := os.Stat(newPath); os.IsNotExist(err) {
						if err := mm.copyFile(oldPath, newPath); err != nil {
							return fmt.Errorf("failed to migrate %s: %v", dbFile, err)
						}
					}
				}
			}
		}
	}

	// Migrate config files
	oldConfigDir := filepath.Join(mm.appDataPath, "config")
	newConfigDir, err := utils.GetConfigDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(oldConfigDir); err == nil && oldConfigDir != newConfigDir {
		if err := mm.copyDir(oldConfigDir, newConfigDir); err != nil {
			return fmt.Errorf("failed to migrate config: %v", err)
		}
	}

	return nil
}

// migrateFromV0 migrates data from version 0.x
func (mm *MigrationManager) migrateFromV0(_ string) error {
	// Implement specific migrations for v0.x versions
	return mm.migrateLegacyData()
}

// migrateFromV1_0 migrates data from version 1.0.x
func (mm *MigrationManager) migrateFromV1_0(_ string) error {
	// Implement specific migrations for v1.0.x versions
	// For now, just ensure directory structure is up to date
	return utils.EnsureAppDataDirs()
}

// updateVersionFile updates the version file with current version
func (mm *MigrationManager) updateVersionFile() error {
	versionFile := filepath.Join(mm.appDataPath, "version.json")

	versionInfo := struct {
		Version   string    `json:"version"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		Version:   mm.currentVersion,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(versionInfo, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(versionFile, data, 0644)
}

// saveMigrationInfo saves migration information to a log file
func (mm *MigrationManager) saveMigrationInfo(info *MigrationInfo) {
	logsDir, err := utils.GetLogsDir()
	if err != nil {
		return
	}

	if err := utils.EnsureDir(logsDir); err != nil {
		return
	}

	migrationLog := filepath.Join(logsDir, "migrations.json")

	// Read existing migrations
	var migrations []MigrationInfo
	if data, err := os.ReadFile(migrationLog); err == nil {
		json.Unmarshal(data, &migrations)
	}

	// Add new migration
	migrations = append(migrations, *info)

	// Save updated migrations
	if data, err := json.MarshalIndent(migrations, "", "  "); err == nil {
		os.WriteFile(migrationLog, data, 0644)
	}
}

// GetMigrationHistory returns the history of migrations
func (mm *MigrationManager) GetMigrationHistory() ([]MigrationInfo, error) {
	logsDir, err := utils.GetLogsDir()
	if err != nil {
		return nil, err
	}

	migrationLog := filepath.Join(logsDir, "migrations.json")

	data, err := os.ReadFile(migrationLog)
	if err != nil {
		if os.IsNotExist(err) {
			return []MigrationInfo{}, nil
		}
		return nil, err
	}

	var migrations []MigrationInfo
	if err := json.Unmarshal(data, &migrations); err != nil {
		return nil, err
	}

	return migrations, nil
}
