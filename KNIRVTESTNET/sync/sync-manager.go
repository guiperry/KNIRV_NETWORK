package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncManager handles synchronization of scripts and testing patterns
type SyncManager struct {
	TestnetRoot    string
	ProductionRoot string
	SyncConfig     *SyncConfig
	Logger         *log.Logger
}

// SyncConfig defines what patterns to synchronize
type SyncConfig struct {
	ScriptPatterns []ScriptPattern `json:"script_patterns"`
	TestPatterns   []TestPattern   `json:"test_patterns"`
	ExcludeFiles   []string        `json:"exclude_files"`
	Components     []Component     `json:"components"`
}

// ScriptPattern defines a script synchronization pattern
type ScriptPattern struct {
	Name        string   `json:"name"`
	Pattern     string   `json:"pattern"`
	SourceDirs  []string `json:"source_dirs"`
	TargetDir   string   `json:"target_dir"`
	Transform   string   `json:"transform"`
	Enabled     bool     `json:"enabled"`
}

// TestPattern defines a testing pattern synchronization
type TestPattern struct {
	Name       string   `json:"name"`
	Pattern    string   `json:"pattern"`
	SourceDirs []string `json:"source_dirs"`
	TargetDir  string   `json:"target_dir"`
	Framework  string   `json:"framework"`
	Enabled    bool     `json:"enabled"`
}

// Component represents a KNIRV network component
type Component struct {
	Name           string `json:"name"`
	ProductionPath string `json:"production_path"`
	TestnetPath    string `json:"testnet_path"`
	Enabled        bool   `json:"enabled"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Component     string    `json:"component"`
	Pattern       string    `json:"pattern"`
	FilesSync     int       `json:"files_synced"`
	FilesSkipped  int       `json:"files_skipped"`
	Errors        []string  `json:"errors"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `json:"success"`
}

// NewSyncManager creates a new synchronization manager
func NewSyncManager(testnetRoot, productionRoot string) *SyncManager {
	logger := log.New(os.Stdout, "[SYNC] ", log.LstdFlags)
	
	return &SyncManager{
		TestnetRoot:    testnetRoot,
		ProductionRoot: productionRoot,
		Logger:         logger,
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
			sm.Logger.Printf("Skipping disabled pattern: %s", pattern.Name)
			continue
		}
		
		result := SyncResult{
			Pattern:   pattern.Name,
			Timestamp: time.Now(),
		}
		
		sm.Logger.Printf("Synchronizing script pattern: %s", pattern.Name)
		
		for _, component := range sm.SyncConfig.Components {
			if !component.Enabled {
				continue
			}
			
			result.Component = component.Name
			
			// Find matching files in production component
			sourcePath := filepath.Join(sm.ProductionRoot, component.ProductionPath)
			targetPath := filepath.Join(sm.TestnetRoot, component.TestnetPath)
			
			err := sm.syncPatternFiles(sourcePath, targetPath, pattern)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				result.Success = false
			} else {
				result.Success = true
				result.FilesSync++
			}
		}
		
		results = append(results, result)
	}
	
	sm.Logger.Printf("Script pattern synchronization completed. %d patterns processed", len(results))
	return results, nil
}

// SyncTestPatterns synchronizes testing patterns from production to testnet
func (sm *SyncManager) SyncTestPatterns() ([]SyncResult, error) {
	var results []SyncResult
	
	sm.Logger.Println("Starting test pattern synchronization...")
	
	for _, pattern := range sm.SyncConfig.TestPatterns {
		if !pattern.Enabled {
			sm.Logger.Printf("Skipping disabled test pattern: %s", pattern.Name)
			continue
		}
		
		result := SyncResult{
			Pattern:   pattern.Name,
			Timestamp: time.Now(),
		}
		
		sm.Logger.Printf("Synchronizing test pattern: %s", pattern.Name)
		
		for _, component := range sm.SyncConfig.Components {
			if !component.Enabled {
				continue
			}
			
			result.Component = component.Name
			
			// Find matching test files in production component
			sourcePath := filepath.Join(sm.ProductionRoot, component.ProductionPath)
			targetPath := filepath.Join(sm.TestnetRoot, component.TestnetPath)
			
			err := sm.syncTestFiles(sourcePath, targetPath, pattern)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				result.Success = false
			} else {
				result.Success = true
				result.FilesSync++
			}
		}
		
		results = append(results, result)
	}
	
	sm.Logger.Printf("Test pattern synchronization completed. %d patterns processed", len(results))
	return results, nil
}

// syncPatternFiles synchronizes files matching a script pattern
func (sm *SyncManager) syncPatternFiles(sourcePath, targetPath string, pattern ScriptPattern) error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}
	
	// Ensure target directory exists
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	
	// Walk through source directory and find matching files
	return filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Check if file matches pattern
		if matched, _ := filepath.Match(pattern.Pattern, d.Name()); matched {
			relPath, _ := filepath.Rel(sourcePath, path)
			targetFile := filepath.Join(targetPath, relPath)
			
			// Skip excluded files
			for _, exclude := range sm.SyncConfig.ExcludeFiles {
				if strings.Contains(path, exclude) {
					sm.Logger.Printf("Skipping excluded file: %s", path)
					return nil
				}
			}
			
			return sm.copyAndTransformFile(path, targetFile, pattern.Transform)
		}
		
		return nil
	})
}

// syncTestFiles synchronizes test files matching a test pattern
func (sm *SyncManager) syncTestFiles(sourcePath, targetPath string, pattern TestPattern) error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}
	
	// Ensure target directory exists
	testTargetPath := filepath.Join(targetPath, "tests")
	if err := os.MkdirAll(testTargetPath, 0755); err != nil {
		return fmt.Errorf("failed to create test target directory: %w", err)
	}
	
	// Walk through source directory and find matching test files
	return filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Check if file matches test pattern
		if matched, _ := filepath.Match(pattern.Pattern, d.Name()); matched {
			relPath, _ := filepath.Rel(sourcePath, path)
			targetFile := filepath.Join(testTargetPath, relPath)
			
			// Skip excluded files
			for _, exclude := range sm.SyncConfig.ExcludeFiles {
				if strings.Contains(path, exclude) {
					sm.Logger.Printf("Skipping excluded test file: %s", path)
					return nil
				}
			}
			
			return sm.copyAndTransformFile(path, targetFile, "testnet")
		}
		
		return nil
	})
}

// copyAndTransformFile copies a file and applies transformations
func (sm *SyncManager) copyAndTransformFile(sourcePath, targetPath, transform string) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	
	// Read source file
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}
	
	// Apply transformations based on transform type
	transformedContent := sm.applyTransformations(content, transform)
	
	// Write to target file
	if err := os.WriteFile(targetPath, transformedContent, 0755); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}
	
	sm.Logger.Printf("Synchronized: %s -> %s", sourcePath, targetPath)
	return nil
}

// applyTransformations applies content transformations for testnet compatibility
func (sm *SyncManager) applyTransformations(content []byte, transform string) []byte {
	contentStr := string(content)
	
	switch transform {
	case "testnet":
		// Add testnet-specific transformations
		contentStr = strings.ReplaceAll(contentStr, "production", "testnet")
		contentStr = strings.ReplaceAll(contentStr, "-tags \"\"", "-tags \"testnet\"")
		contentStr = strings.ReplaceAll(contentStr, "CGO_ENABLED=1", "CGO_ENABLED=0")
		
	case "build":
		// Add build-specific transformations
		contentStr = strings.ReplaceAll(contentStr, "go build", "go build -tags testnet")
		
	case "test":
		// Add test-specific transformations
		contentStr = strings.ReplaceAll(contentStr, "go test", "go test -tags testnet")
	}
	
	return []byte(contentStr)
}

// GenerateReport generates a synchronization report
func (sm *SyncManager) GenerateReport(results []SyncResult, outputPath string) error {
	report := map[string]interface{}{
		"timestamp":      time.Now(),
		"total_patterns": len(results),
		"successful":     0,
		"failed":         0,
		"results":        results,
	}
	
	for _, result := range results {
		if result.Success {
			report["successful"] = report["successful"].(int) + 1
		} else {
			report["failed"] = report["failed"].(int) + 1
		}
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
