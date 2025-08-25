package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// RollbackManager handles rollback and recovery operations
type RollbackManager struct {
	TestnetRoot string
	BackupDir   string
	Logger      *log.Logger
}

// Snapshot represents a backup snapshot of the synchronization state
type Snapshot struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Components  []string  `json:"components"`
	BackupPath  string    `json:"backup_path"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
}

// RollbackOperation represents a rollback operation
type RollbackOperation struct {
	SnapshotID    string    `json:"snapshot_id"`
	Timestamp     time.Time `json:"timestamp"`
	FilesRestored int       `json:"files_restored"`
	Success       bool      `json:"success"`
	Errors        []string  `json:"errors"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(testnetRoot, backupDir string) *RollbackManager {
	logger := log.New(os.Stdout, "[ROLLBACK] ", log.LstdFlags)
	
	return &RollbackManager{
		TestnetRoot: testnetRoot,
		BackupDir:   backupDir,
		Logger:      logger,
	}
}

// CreateSnapshot creates a backup snapshot of the current synchronization state
func (rm *RollbackManager) CreateSnapshot(description string, components []string) (*Snapshot, error) {
	rm.Logger.Printf("Creating snapshot: %s", description)
	
	// Generate snapshot ID
	snapshotID := fmt.Sprintf("snapshot-%s", time.Now().Format("20060102-150405"))
	snapshotPath := filepath.Join(rm.BackupDir, snapshotID)
	
	// Create snapshot directory
	if err := os.MkdirAll(snapshotPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	
	snapshot := &Snapshot{
		ID:          snapshotID,
		Timestamp:   time.Now(),
		Description: description,
		Components:  components,
		BackupPath:  snapshotPath,
	}
	
	// Backup sync patterns
	syncPatternsPath := filepath.Join(rm.TestnetRoot, "sync", "patterns")
	if err := rm.backupDirectory(syncPatternsPath, filepath.Join(snapshotPath, "patterns")); err != nil {
		return nil, fmt.Errorf("failed to backup sync patterns: %w", err)
	}
	
	// Backup scripts
	scriptsPath := filepath.Join(rm.TestnetRoot, "scripts")
	if err := rm.backupDirectory(scriptsPath, filepath.Join(snapshotPath, "scripts")); err != nil {
		return nil, fmt.Errorf("failed to backup scripts: %w", err)
	}
	
	// Backup tests
	testsPath := filepath.Join(rm.TestnetRoot, "tests")
	if err := rm.backupDirectory(testsPath, filepath.Join(snapshotPath, "tests")); err != nil {
		return nil, fmt.Errorf("failed to backup tests: %w", err)
	}
	
	// Calculate snapshot size and checksum
	size, checksum, err := rm.calculateSnapshotMetrics(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate snapshot metrics: %w", err)
	}
	
	snapshot.Size = size
	snapshot.Checksum = checksum
	
	// Save snapshot metadata
	metadataPath := filepath.Join(snapshotPath, "metadata.json")
	if err := rm.saveSnapshotMetadata(snapshot, metadataPath); err != nil {
		return nil, fmt.Errorf("failed to save snapshot metadata: %w", err)
	}
	
	rm.Logger.Printf("Snapshot created successfully: %s", snapshotID)
	return snapshot, nil
}

// ListSnapshots lists all available snapshots
func (rm *RollbackManager) ListSnapshots() ([]*Snapshot, error) {
	var snapshots []*Snapshot
	
	if _, err := os.Stat(rm.BackupDir); os.IsNotExist(err) {
		return snapshots, nil
	}
	
	entries, err := os.ReadDir(rm.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		metadataPath := filepath.Join(rm.BackupDir, entry.Name(), "metadata.json")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			continue
		}
		
		snapshot, err := rm.loadSnapshotMetadata(metadataPath)
		if err != nil {
			rm.Logger.Printf("Failed to load snapshot metadata for %s: %v", entry.Name(), err)
			continue
		}
		
		snapshots = append(snapshots, snapshot)
	}
	
	return snapshots, nil
}

// RollbackToSnapshot rolls back to a specific snapshot
func (rm *RollbackManager) RollbackToSnapshot(snapshotID string) (*RollbackOperation, error) {
	rm.Logger.Printf("Rolling back to snapshot: %s", snapshotID)
	
	operation := &RollbackOperation{
		SnapshotID: snapshotID,
		Timestamp:  time.Now(),
		Success:    false,
	}
	
	// Find snapshot
	snapshotPath := filepath.Join(rm.BackupDir, snapshotID)
	metadataPath := filepath.Join(snapshotPath, "metadata.json")
	
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Snapshot not found: %s", snapshotID))
		return operation, fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	
	snapshot, err := rm.loadSnapshotMetadata(metadataPath)
	if err != nil {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Failed to load snapshot metadata: %v", err))
		return operation, fmt.Errorf("failed to load snapshot metadata: %w", err)
	}
	
	// Verify snapshot integrity
	if err := rm.verifySnapshotIntegrity(snapshot); err != nil {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Snapshot integrity check failed: %v", err))
		return operation, fmt.Errorf("snapshot integrity check failed: %w", err)
	}
	
	// Create backup of current state before rollback
	currentSnapshot, err := rm.CreateSnapshot(fmt.Sprintf("Pre-rollback backup before %s", snapshotID), []string{})
	if err != nil {
		rm.Logger.Printf("Warning: Failed to create pre-rollback backup: %v", err)
	} else {
		rm.Logger.Printf("Created pre-rollback backup: %s", currentSnapshot.ID)
	}
	
	// Restore sync patterns
	patternsBackup := filepath.Join(snapshotPath, "patterns")
	patternsTarget := filepath.Join(rm.TestnetRoot, "sync", "patterns")
	if err := rm.restoreDirectory(patternsBackup, patternsTarget); err != nil {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Failed to restore patterns: %v", err))
	} else {
		operation.FilesRestored += rm.countFiles(patternsBackup)
	}
	
	// Restore scripts
	scriptsBackup := filepath.Join(snapshotPath, "scripts")
	scriptsTarget := filepath.Join(rm.TestnetRoot, "scripts")
	if err := rm.restoreDirectory(scriptsBackup, scriptsTarget); err != nil {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Failed to restore scripts: %v", err))
	} else {
		operation.FilesRestored += rm.countFiles(scriptsBackup)
	}
	
	// Restore tests
	testsBackup := filepath.Join(snapshotPath, "tests")
	testsTarget := filepath.Join(rm.TestnetRoot, "tests")
	if err := rm.restoreDirectory(testsBackup, testsTarget); err != nil {
		operation.Errors = append(operation.Errors, fmt.Sprintf("Failed to restore tests: %v", err))
	} else {
		operation.FilesRestored += rm.countFiles(testsBackup)
	}
	
	if len(operation.Errors) == 0 {
		operation.Success = true
		rm.Logger.Printf("Rollback completed successfully. %d files restored", operation.FilesRestored)
	} else {
		rm.Logger.Printf("Rollback completed with %d errors", len(operation.Errors))
	}
	
	return operation, nil
}

// DeleteSnapshot deletes a specific snapshot
func (rm *RollbackManager) DeleteSnapshot(snapshotID string) error {
	snapshotPath := filepath.Join(rm.BackupDir, snapshotID)
	
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	
	if err := os.RemoveAll(snapshotPath); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	
	rm.Logger.Printf("Snapshot deleted: %s", snapshotID)
	return nil
}

// CleanupOldSnapshots removes snapshots older than the specified duration
func (rm *RollbackManager) CleanupOldSnapshots(maxAge time.Duration) error {
	snapshots, err := rm.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}
	
	cutoff := time.Now().Add(-maxAge)
	deleted := 0
	
	for _, snapshot := range snapshots {
		if snapshot.Timestamp.Before(cutoff) {
			if err := rm.DeleteSnapshot(snapshot.ID); err != nil {
				rm.Logger.Printf("Failed to delete old snapshot %s: %v", snapshot.ID, err)
			} else {
				deleted++
			}
		}
	}
	
	rm.Logger.Printf("Cleaned up %d old snapshots", deleted)
	return nil
}

// backupDirectory recursively backs up a directory
func (rm *RollbackManager) backupDirectory(source, target string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil // Source doesn't exist, skip
	}
	
	return filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		relPath, _ := filepath.Rel(source, path)
		targetPath := filepath.Join(target, relPath)
		
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		
		return rm.copyFile(path, targetPath)
	})
}

// restoreDirectory recursively restores a directory
func (rm *RollbackManager) restoreDirectory(source, target string) error {
	// Remove existing target directory
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to remove target directory: %w", err)
	}
	
	// Restore from backup
	return rm.backupDirectory(source, target)
}

// copyFile copies a single file
func (rm *RollbackManager) copyFile(source, target string) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	
	// Read source file
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	
	// Write target file
	return os.WriteFile(target, data, 0644)
}

// countFiles counts the number of files in a directory
func (rm *RollbackManager) countFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// calculateSnapshotMetrics calculates size and checksum for a snapshot
func (rm *RollbackManager) calculateSnapshotMetrics(snapshotPath string) (int64, string, error) {
	var size int64
	
	err := filepath.WalkDir(snapshotPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		
		return nil
	})
	
	// Simple checksum based on size and file count
	checksum := fmt.Sprintf("%d-%d", size, rm.countFiles(snapshotPath))
	
	return size, checksum, err
}

// verifySnapshotIntegrity verifies the integrity of a snapshot
func (rm *RollbackManager) verifySnapshotIntegrity(snapshot *Snapshot) error {
	currentSize, currentChecksum, err := rm.calculateSnapshotMetrics(snapshot.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to calculate current metrics: %w", err)
	}
	
	if currentSize != snapshot.Size {
		return fmt.Errorf("size mismatch: expected %d, got %d", snapshot.Size, currentSize)
	}
	
	if currentChecksum != snapshot.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", snapshot.Checksum, currentChecksum)
	}
	
	return nil
}

// saveSnapshotMetadata saves snapshot metadata to a file
func (rm *RollbackManager) saveSnapshotMetadata(snapshot *Snapshot, path string) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// loadSnapshotMetadata loads snapshot metadata from a file
func (rm *RollbackManager) loadSnapshotMetadata(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	
	return &snapshot, nil
}
