package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SyncMonitor handles monitoring and validation of synchronization
type SyncMonitor struct {
	TestnetRoot    string
	ProductionRoot string
	ReportsDir     string
	Logger         *log.Logger
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Component     string    `json:"component"`
	Pattern       string    `json:"pattern"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	Timestamp     time.Time `json:"timestamp"`
	FilesChecked  int       `json:"files_checked"`
	FilesMissing  int       `json:"files_missing"`
	FilesOutdated int       `json:"files_outdated"`
}

// MonitoringReport represents a comprehensive monitoring report
type MonitoringReport struct {
	Timestamp        time.Time          `json:"timestamp"`
	TotalComponents  int                `json:"total_components"`
	ValidComponents  int                `json:"valid_components"`
	InvalidComponents int               `json:"invalid_components"`
	ValidationResults []ValidationResult `json:"validation_results"`
	Recommendations  []string           `json:"recommendations"`
}

// NewSyncMonitor creates a new synchronization monitor
func NewSyncMonitor(testnetRoot, productionRoot, reportsDir string) *SyncMonitor {
	logger := log.New(os.Stdout, "[MONITOR] ", log.LstdFlags)
	
	return &SyncMonitor{
		TestnetRoot:    testnetRoot,
		ProductionRoot: productionRoot,
		ReportsDir:     reportsDir,
		Logger:         logger,
	}
}

// ValidateSync validates the current synchronization state
func (sm *SyncMonitor) ValidateSync(config *SyncConfig) (*MonitoringReport, error) {
	sm.Logger.Println("Starting synchronization validation...")
	
	report := &MonitoringReport{
		Timestamp:         time.Now(),
		TotalComponents:   len(config.Components),
		ValidationResults: []ValidationResult{},
		Recommendations:   []string{},
	}
	
	for _, component := range config.Components {
		if !component.Enabled {
			continue
		}
		
		sm.Logger.Printf("Validating component: %s", component.Name)
		
		// Validate script patterns
		for _, pattern := range config.ScriptPatterns {
			if !pattern.Enabled {
				continue
			}
			
			result := sm.validateScriptPattern(component, pattern)
			report.ValidationResults = append(report.ValidationResults, result)
			
			if result.Status == "valid" {
				report.ValidComponents++
			} else {
				report.InvalidComponents++
			}
		}
		
		// Validate test patterns
		for _, pattern := range config.TestPatterns {
			if !pattern.Enabled {
				continue
			}
			
			result := sm.validateTestPattern(component, pattern)
			report.ValidationResults = append(report.ValidationResults, result)
		}
	}
	
	// Generate recommendations
	report.Recommendations = sm.generateRecommendations(report.ValidationResults)
	
	sm.Logger.Printf("Validation completed. %d valid, %d invalid components",
		report.ValidComponents, report.InvalidComponents)
	
	return report, nil
}

// validateScriptPattern validates a script pattern for a component
func (sm *SyncMonitor) validateScriptPattern(component Component, pattern ScriptPattern) ValidationResult {
	result := ValidationResult{
		Component: component.Name,
		Pattern:   pattern.Name,
		Timestamp: time.Now(),
		Status:    "valid",
	}
	
	// Check if production files exist
	productionPath := filepath.Join(sm.ProductionRoot, component.ProductionPath)
	testnetPath := filepath.Join(sm.TestnetRoot, component.TestnetPath)
	
	productionFiles, err := sm.findMatchingFiles(productionPath, pattern.Pattern)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to scan production files: %v", err)
		return result
	}
	
	testnetFiles, err := sm.findMatchingFiles(testnetPath, pattern.Pattern)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to scan testnet files: %v", err)
		return result
	}
	
	result.FilesChecked = len(productionFiles)
	
	// Check for missing files
	for _, prodFile := range productionFiles {
		relPath, _ := filepath.Rel(productionPath, prodFile)
		testnetFile := filepath.Join(testnetPath, relPath)
		
		if _, err := os.Stat(testnetFile); os.IsNotExist(err) {
			result.FilesMissing++
		} else {
			// Check if file is outdated
			if sm.isFileOutdated(prodFile, testnetFile) {
				result.FilesOutdated++
			}
		}
	}
	
	if result.FilesMissing > 0 || result.FilesOutdated > 0 {
		result.Status = "outdated"
		result.Message = fmt.Sprintf("%d missing, %d outdated files", result.FilesMissing, result.FilesOutdated)
	} else {
		result.Message = "All files synchronized"
	}
	
	return result
}

// validateTestPattern validates a test pattern for a component
func (sm *SyncMonitor) validateTestPattern(component Component, pattern TestPattern) ValidationResult {
	result := ValidationResult{
		Component: component.Name,
		Pattern:   pattern.Name,
		Timestamp: time.Now(),
		Status:    "valid",
	}
	
	// Check if production test files exist
	productionPath := filepath.Join(sm.ProductionRoot, component.ProductionPath)
	testnetPath := filepath.Join(sm.TestnetRoot, component.TestnetPath, "tests")
	
	productionFiles, err := sm.findMatchingFiles(productionPath, pattern.Pattern)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to scan production test files: %v", err)
		return result
	}
	
	testnetFiles, err := sm.findMatchingFiles(testnetPath, pattern.Pattern)
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Failed to scan testnet test files: %v", err)
		return result
	}
	
	result.FilesChecked = len(productionFiles)
	
	// Check for missing test files
	for _, prodFile := range productionFiles {
		relPath, _ := filepath.Rel(productionPath, prodFile)
		testnetFile := filepath.Join(testnetPath, relPath)
		
		if _, err := os.Stat(testnetFile); os.IsNotExist(err) {
			result.FilesMissing++
		} else {
			// Check if test file is outdated
			if sm.isFileOutdated(prodFile, testnetFile) {
				result.FilesOutdated++
			}
		}
	}
	
	if result.FilesMissing > 0 || result.FilesOutdated > 0 {
		result.Status = "outdated"
		result.Message = fmt.Sprintf("%d missing, %d outdated test files", result.FilesMissing, result.FilesOutdated)
	} else {
		result.Message = "All test files synchronized"
	}
	
	return result
}

// findMatchingFiles finds files matching a pattern in a directory
func (sm *SyncMonitor) findMatchingFiles(dir, pattern string) ([]string, error) {
	var files []string
	
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return files, nil
	}
	
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		if matched, _ := filepath.Match(pattern, d.Name()); matched {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

// isFileOutdated checks if a testnet file is outdated compared to production
func (sm *SyncMonitor) isFileOutdated(productionFile, testnetFile string) bool {
	prodInfo, err := os.Stat(productionFile)
	if err != nil {
		return false
	}
	
	testnetInfo, err := os.Stat(testnetFile)
	if err != nil {
		return true
	}
	
	return prodInfo.ModTime().After(testnetInfo.ModTime())
}

// generateRecommendations generates recommendations based on validation results
func (sm *SyncMonitor) generateRecommendations(results []ValidationResult) []string {
	var recommendations []string
	
	outdatedCount := 0
	missingCount := 0
	errorCount := 0
	
	for _, result := range results {
		switch result.Status {
		case "outdated":
			outdatedCount++
		case "error":
			errorCount++
		}
		missingCount += result.FilesMissing
	}
	
	if outdatedCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Run synchronization to update %d outdated patterns", outdatedCount))
	}
	
	if missingCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Synchronize %d missing files from production components", missingCount))
	}
	
	if errorCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Investigate %d validation errors", errorCount))
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "All components are synchronized and up to date")
	}
	
	return recommendations
}

// GenerateMonitoringReport generates a comprehensive monitoring report
func (sm *SyncMonitor) GenerateMonitoringReport(report *MonitoringReport, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring report: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write monitoring report: %w", err)
	}
	
	sm.Logger.Printf("Monitoring report generated: %s", outputPath)
	return nil
}

// WatchForChanges monitors for changes in production components
func (sm *SyncMonitor) WatchForChanges(config *SyncConfig, interval time.Duration, callback func(*MonitoringReport)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	sm.Logger.Printf("Starting change monitoring with %v interval", interval)
	
	for {
		select {
		case <-ticker.C:
			sm.Logger.Println("Checking for changes...")
			
			report, err := sm.ValidateSync(config)
			if err != nil {
				sm.Logger.Printf("Validation failed: %v", err)
				continue
			}
			
			// Check if any components are outdated
			hasChanges := false
			for _, result := range report.ValidationResults {
				if result.Status == "outdated" || result.FilesMissing > 0 || result.FilesOutdated > 0 {
					hasChanges = true
					break
				}
			}
			
			if hasChanges {
				sm.Logger.Println("Changes detected, triggering callback")
				callback(report)
			}
		}
	}
}
