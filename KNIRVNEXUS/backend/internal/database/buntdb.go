package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
)

// getAppDataDir returns the OS-specific application data directory for KNIRV-NEXUS
func getAppDataDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	appDataDir := filepath.Join(usr.HomeDir, ".local", "share", "knirvnexus", "backend_server")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory: %v", err)
	}
	return appDataDir, nil
}

// BuntDBManager manages BuntDB operations with custom indexes and corruption prevention
type BuntDBManager struct {
	db     *buntdb.DB
	dbPath string
	mu     sync.RWMutex

	// Corruption prevention
	backupDir      string
	autoBackup     bool
	backupInterval time.Duration
	maxBackups     int

	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc

	// Health monitoring
	lastHealthCheck time.Time
	isHealthy       bool
}

// NewBuntDB creates a new BuntDB instance with custom indexes and corruption prevention
func NewBuntDB(path string) (*BuntDBManager, error) {
	// Validate database file before opening
	if err := validateDatabaseFile(path); err != nil {
		log.Printf("Database validation failed, attempting recovery: %v", err)
		if err := recoverDatabase(path); err != nil {
			return nil, fmt.Errorf("failed to recover database: %w", err)
		}
	}

	db, err := buntdb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open BuntDB: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Use OS application data directory for backups
	appDataDir, err := getAppDataDir()
	var backupDir string
	if err != nil {
		log.Printf("Warning: Failed to get app data directory, falling back to relative path: %v", err)
		backupDir = filepath.Dir(filepath.Dir(path)) + "/backups"
	} else {
		backupDir = filepath.Join(appDataDir, "backups")
	}

	log.Printf("Database backup directory set to: %s (for db path: %s)", backupDir, path)

	manager := &BuntDBManager{
		db:              db,
		dbPath:          path,
		backupDir:       backupDir,
		autoBackup:      true,
		backupInterval:  30 * time.Minute,
		maxBackups:      10,
		ctx:             ctx,
		cancel:          cancel,
		lastHealthCheck: time.Now(),
		isHealthy:       true,
	}

	// Create backup directory
	if err := os.MkdirAll(manager.backupDir, 0755); err != nil {
		log.Printf("Warning: Failed to create backup directory: %v", err)
		manager.autoBackup = false
	}

	// Create custom indexes
	if err := manager.createIndexes(); err != nil {
		db.Close()
		cancel()
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	// Start background health monitoring and backup
	go manager.backgroundMaintenance()

	return manager, nil
}

// createIndexes creates all necessary indexes for DVE operations
func (bm *BuntDBManager) createIndexes() error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		// DVE Node indexes
		tx.CreateIndex("nodes_by_status", "dve:nodes:*", buntdb.IndexJSON("status"))
		tx.CreateIndex("nodes_by_tee_type", "dve:nodes:*", buntdb.IndexJSON("tee_type"))
		tx.CreateIndex("nodes_by_reputation", "dve:nodes:*", buntdb.IndexJSON("reputation_score"))
		tx.CreateIndex("nodes_by_stake", "dve:nodes:*", buntdb.IndexJSON("stake_amount"))
		tx.CreateIndex("nodes_by_location", "dve:nodes:*", buntdb.IndexJSON("location"))
		tx.CreateIndex("nodes_by_created_at", "dve:nodes:*", buntdb.IndexJSON("created_at"))

		// Validation Task indexes
		tx.CreateIndex("tasks_by_status", "validation:tasks:*", buntdb.IndexJSON("status"))
		tx.CreateIndex("tasks_by_priority", "validation:tasks:*", buntdb.IndexJSON("priority"))
		tx.CreateIndex("tasks_by_type", "validation:tasks:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("tasks_by_created_at", "validation:tasks:*", buntdb.IndexJSON("created_at"))
		tx.CreateIndex("tasks_by_assigned_node", "validation:tasks:*", buntdb.IndexJSON("assigned_node_id"))

		// Validation Proof indexes
		tx.CreateIndex("proofs_by_task_id", "validation:proofs:*", buntdb.IndexJSON("task_id"))
		tx.CreateIndex("proofs_by_validator", "validation:proofs:*", buntdb.IndexJSON("validator_node_id"))
		tx.CreateIndex("proofs_by_status", "validation:proofs:*", buntdb.IndexJSON("status"))

		// TEE Attestation indexes
		tx.CreateIndex("attestations_by_node", "tee:attestations:*", buntdb.IndexJSON("node_id"))
		tx.CreateIndex("attestations_by_type", "tee:attestations:*", buntdb.IndexJSON("tee_type"))
		tx.CreateIndex("attestations_by_status", "tee:attestations:*", buntdb.IndexJSON("status"))

		// User Management indexes
		tx.CreateIndex("users_by_email", "users:profiles:*", buntdb.IndexJSON("email"))
		tx.CreateIndex("users_by_username", "users:profiles:*", buntdb.IndexJSON("username"))
		tx.CreateIndex("users_by_role", "users:profiles:*", buntdb.IndexJSON("role"))
		tx.CreateIndex("users_by_created_at", "users:profiles:*", buntdb.IndexJSON("created_at"))

		// Session indexes
		tx.CreateIndex("sessions_by_user_id", "users:sessions:*", buntdb.IndexJSON("user_id"))
		tx.CreateIndex("sessions_by_expires_at", "users:sessions:*", buntdb.IndexJSON("expires_at"))

		// Report indexes
		tx.CreateIndex("reports_by_type", "reports:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("reports_by_created_at", "reports:*", buntdb.IndexJSON("created_at"))
		tx.CreateIndex("reports_by_user", "reports:*", buntdb.IndexJSON("generated_by"))
		tx.CreateIndex("reports_by_format", "reports:*", buntdb.IndexJSON("format"))

		// Report Template indexes
		tx.CreateIndex("templates_by_type", "report:templates:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("templates_by_created_by", "report:templates:*", buntdb.IndexJSON("created_by"))
		tx.CreateIndex("templates_by_public", "report:templates:*", buntdb.IndexJSON("is_public"))

		// Metrics indexes
		tx.CreateIndex("metrics_by_timestamp", "metrics:historical:*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("metrics_by_type", "metrics:historical:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("metrics_by_node_id", "metrics:historical:*", buntdb.IndexJSON("node_id"))

		// Spatial index for geographic node distribution
		tx.CreateSpatialIndex("nodes_by_coordinates", "dve:nodes:*", buntdb.IndexRect)

		return nil
	})
}

// GetDB returns the underlying BuntDB instance
func (bm *BuntDBManager) GetDB() *buntdb.DB {
	return bm.db
}

// Close closes the database connection with graceful shutdown
func (bm *BuntDBManager) Close() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Cancel background operations
	if bm.cancel != nil {
		bm.cancel()
	}

	// Create final backup before closing
	if bm.autoBackup && bm.isHealthy {
		if err := bm.createBackup("final"); err != nil {
			log.Printf("Warning: Failed to create final backup: %v", err)
		}
	}

	// Close database
	if bm.db != nil {
		return bm.db.Close()
	}

	return nil
}

// StoreJSON stores a JSON object with the given key
func (bm *BuntDBManager) StoreJSON(key string, data interface{}) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(key, string(jsonData), nil)
		return err
	})
}

// GetJSON retrieves and unmarshals a JSON object
func (bm *BuntDBManager) GetJSON(key string, dest interface{}) error {
	return bm.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), dest)
	})
}

// DeleteKey deletes a key from the database
func (bm *BuntDBManager) DeleteKey(key string) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

// ListKeys returns all keys matching a pattern
func (bm *BuntDBManager) ListKeys(pattern string) ([]string, error) {
	var keys []string

	// If pattern is empty, iterate through all keys
	if pattern == "" {
		err := bm.db.View(func(tx *buntdb.Tx) error {
			return tx.Ascend("", func(key, value string) bool {
				keys = append(keys, key)
				return true
			})
		})
		return keys, err
	}

	// Otherwise use the pattern
	err := bm.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(pattern, func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
	})
	return keys, err
}

// CountKeys returns the count of keys matching a pattern
func (bm *BuntDBManager) CountKeys(pattern string) (int, error) {
	count := 0
	err := bm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(pattern, func(key, value string) bool {
			count++
			return true
		})
	})
	return count, err
}

// SetWithTTL stores a key-value pair with a time-to-live
func (bm *BuntDBManager) SetWithTTL(key, value string, ttl time.Duration) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		opts := &buntdb.SetOptions{Expires: true, TTL: ttl}
		_, _, err := tx.Set(key, value, opts)
		return err
	})
}

// Transaction executes a function within a database transaction
func (bm *BuntDBManager) Transaction(fn func(*buntdb.Tx) error) error {
	return bm.db.Update(fn)
}

// ViewTransaction executes a read-only function within a database transaction
func (bm *BuntDBManager) ViewTransaction(fn func(*buntdb.Tx) error) error {
	return bm.db.View(fn)
}

// GetValue retrieves a value from the database by key
func (bm *BuntDBManager) GetValue(key string) (string, error) {
	var value string
	err := bm.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		value = val
		return nil
	})
	return value, err
}

// SetValue stores a value in the database with a given key
func (bm *BuntDBManager) SetValue(key, value string) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, nil)
		return err
	})
}

// backgroundMaintenance runs periodic health checks and backups
func (bm *BuntDBManager) backgroundMaintenance() {
	healthTicker := time.NewTicker(5 * time.Minute)
	backupTicker := time.NewTicker(bm.backupInterval)
	defer healthTicker.Stop()
	defer backupTicker.Stop()

	for {
		select {
		case <-bm.ctx.Done():
			return
		case <-healthTicker.C:
			bm.performHealthCheck()
		case <-backupTicker.C:
			if bm.autoBackup {
				if err := bm.createBackup("auto"); err != nil {
					log.Printf("Auto backup failed: %v", err)
				}
			}
		}
	}
}

// performHealthCheck checks database integrity
func (bm *BuntDBManager) performHealthCheck() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Simple health check - try to read a key
	err := bm.db.View(func(tx *buntdb.Tx) error {
		// Try to iterate through a small number of keys
		count := 0
		return tx.Ascend("", func(key, value string) bool {
			count++
			return count < 10 // Only check first 10 keys
		})
	})

	bm.lastHealthCheck = time.Now()
	bm.isHealthy = (err == nil)

	if !bm.isHealthy {
		log.Printf("Database health check failed: %v", err)
	}
}

// createBackup creates a backup of the database
func (bm *BuntDBManager) createBackup(suffix string) error {
	if !bm.autoBackup {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(bm.backupDir, fmt.Sprintf("nexus_%s_%s.db", suffix, timestamp))

	// Create backup file
	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer backupFile.Close()

	// Perform backup by copying all key-value pairs
	err = bm.db.View(func(tx *buntdb.Tx) error {
		// Create a temporary database for backup
		tempDB, err := buntdb.Open(":memory:")
		if err != nil {
			return err
		}
		defer tempDB.Close()

		// Copy all data to temp database
		err = tempDB.Update(func(tempTx *buntdb.Tx) error {
			return tx.Ascend("", func(key, value string) bool {
				tempTx.Set(key, value, nil)
				return true
			})
		})
		if err != nil {
			return err
		}

		// Save temp database to file
		return tempDB.Save(backupFile)
	})

	if err != nil {
		os.Remove(backupPath) // Clean up failed backup
		return fmt.Errorf("failed to write backup: %w", err)
	}

	log.Printf("Database backup created: %s", backupPath)

	// Clean up old backups
	bm.cleanupOldBackups()

	return nil
}

// cleanupOldBackups removes old backup files
func (bm *BuntDBManager) cleanupOldBackups() {
	files, err := filepath.Glob(filepath.Join(bm.backupDir, "nexus_*.db"))
	if err != nil {
		log.Printf("Failed to list backup files: %v", err)
		return
	}

	if len(files) <= bm.maxBackups {
		return
	}

	// Sort files by modification time and remove oldest
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		if stat, err := os.Stat(file); err == nil {
			fileInfos = append(fileInfos, fileInfo{
				path:    file,
				modTime: stat.ModTime(),
			})
		}
	}

	// Sort by modification time (oldest first)
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.After(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove oldest files
	toRemove := len(fileInfos) - bm.maxBackups
	for i := 0; i < toRemove; i++ {
		if err := os.Remove(fileInfos[i].path); err != nil {
			log.Printf("Failed to remove old backup %s: %v", fileInfos[i].path, err)
		} else {
			log.Printf("Removed old backup: %s", fileInfos[i].path)
		}
	}
}

// validateDatabaseFile checks if a database file is valid
func validateDatabaseFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, which is fine for new databases
	}

	// Check if file is readable
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open database file: %w", err)
	}
	defer file.Close()

	// Check file size (empty files or very small files might be corrupted)
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat database file: %w", err)
	}

	if stat.Size() > 0 && stat.Size() < 100 {
		return fmt.Errorf("database file too small, possibly corrupted")
	}

	// Try to open with BuntDB to check basic integrity
	testDB, err := buntdb.Open(path)
	if err != nil {
		return fmt.Errorf("database file corrupted: %w", err)
	}
	testDB.Close()

	return nil
}

// recoverDatabase attempts to recover a corrupted database
func recoverDatabase(path string) error {
	log.Printf("Attempting to recover database: %s", path)

	// Use same backup directory logic as NewBuntDB
	appDataDir, err := getAppDataDir()
	var backupDir string
	if err != nil {
		log.Printf("Warning: Failed to get app data directory for recovery, falling back to relative path: %v", err)
		backupDir = filepath.Dir(filepath.Dir(path)) + "/backups"
	} else {
		backupDir = filepath.Join(appDataDir, "backups")
	}

	backupPattern := filepath.Join(backupDir, "nexus_*.db")

	backups, err := filepath.Glob(backupPattern)
	if err != nil || len(backups) == 0 {
		log.Printf("No backup files found, creating new database")
		// Remove corrupted file and let BuntDB create a new one
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove corrupted database: %w", err)
		}
		return nil
	}

	// Find the most recent backup
	var latestBackup string
	var latestTime time.Time

	for _, backup := range backups {
		if stat, err := os.Stat(backup); err == nil {
			if stat.ModTime().After(latestTime) {
				latestTime = stat.ModTime()
				latestBackup = backup
			}
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("no valid backup found")
	}

	log.Printf("Restoring from backup: %s", latestBackup)

	// Copy backup to main database location
	src, err := os.Open(latestBackup)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create new database file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy backup: %w", err)
	}

	log.Printf("Database recovered from backup successfully")
	return nil
}
