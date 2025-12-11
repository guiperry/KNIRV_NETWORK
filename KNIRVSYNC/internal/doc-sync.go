package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DocSyncConfig defines documentation synchronization configuration
type DocSyncConfig struct {
	Services           []ServiceDocConfig `json:"services"`
	CentralDocPaths    []string           `json:"central_doc_paths"`
	APIDocPaths        []string           `json:"api_doc_paths"`
	UnifiedAPIPath     string             `json:"unified_api_path"`
	SyncREADMEs        bool               `json:"sync_readmes"`
	SyncAPISpecs       bool               `json:"sync_api_specs"`
	GenerateUnifiedAPI bool               `json:"generate_unified_api"`
}

// ServiceDocConfig defines documentation configuration for a service
type ServiceDocConfig struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HasAPI      bool   `json:"has_api"`
	APISpecPath string `json:"api_spec_path"`
	Enabled     bool   `json:"enabled"`
}

// DocSyncResult represents the result of a documentation sync operation
type DocSyncResult struct {
	Service         string    `json:"service"`
	SyncType        string    `json:"sync_type"` // "readme", "api_spec", "unified_api"
	SourcePath      string    `json:"source_path"`
	TargetPaths     []string  `json:"target_paths"`
	FilesUpdated    int       `json:"files_updated"`
	Errors          []string  `json:"errors"`
	Timestamp       time.Time `json:"timestamp"`
	Success         bool      `json:"success"`
	GapsFound       int       `json:"gaps_found"`
	GapsFixed       int       `json:"gaps_fixed"`
}

// DocSyncManager handles documentation synchronization
type DocSyncManager struct {
	RootPath   string
	Config     *DocSyncConfig
	Scanner    *DocScanner
	Logger     Logger
}

// NewDocSyncManager creates a new documentation sync manager
func NewDocSyncManager(rootPath string, config *DocSyncConfig, logger Logger) *DocSyncManager {
	return &DocSyncManager{
		RootPath: rootPath,
		Config:   config,
		Scanner:  NewDocScanner(rootPath, logger),
		Logger:   logger,
	}
}

// SyncAllDocumentation synchronizes all documentation
func (dsm *DocSyncManager) SyncAllDocumentation() ([]DocSyncResult, error) {
	var results []DocSyncResult

	dsm.Logger.Println("Starting documentation synchronization...")

	// First, scan for gaps
	gaps, err := dsm.Scanner.ScanAllServices()
	if err != nil {
		dsm.Logger.Printf("Warning: failed to scan for gaps: %v", err)
	} else {
		dsm.Logger.Printf("Found documentation gaps in %d services", len(gaps))

		// Generate gap report
		reportPath := filepath.Join(dsm.RootPath, "KNIRVSYNC", "reports", "doc-gaps-report.md")
		if err := dsm.Scanner.GenerateGapReport(gaps, reportPath); err != nil {
			dsm.Logger.Printf("Warning: failed to generate gap report: %v", err)
		} else {
			dsm.Logger.Printf("Gap report generated: %s", reportPath)
		}
	}

	// Sync README files
	if dsm.Config.SyncREADMEs {
		readmeResults, err := dsm.SyncREADMEs()
		if err != nil {
			dsm.Logger.Printf("Error syncing READMEs: %v", err)
		}
		results = append(results, readmeResults...)
	}

	// Sync API specifications
	if dsm.Config.SyncAPISpecs {
		apiResults, err := dsm.SyncAPISpecs()
		if err != nil {
			dsm.Logger.Printf("Error syncing API specs: %v", err)
		}
		results = append(results, apiResults...)
	}

	// Generate unified OpenAPI spec
	if dsm.Config.GenerateUnifiedAPI {
		unifiedResult, err := dsm.GenerateUnifiedAPI()
		if err != nil {
			dsm.Logger.Printf("Error generating unified API: %v", err)
		} else {
			results = append(results, unifiedResult)
		}
	}

	dsm.Logger.Printf("Documentation synchronization completed. %d operations performed", len(results))
	return results, nil
}

// SyncREADMEs synchronizes README files to central documentation locations
func (dsm *DocSyncManager) SyncREADMEs() ([]DocSyncResult, error) {
	var results []DocSyncResult

	dsm.Logger.Println("Syncing README files to central documentation...")

	for _, service := range dsm.Config.Services {
		if !service.Enabled {
			continue
		}

		result := DocSyncResult{
			Service:   service.Name,
			SyncType:  "readme",
			Timestamp: time.Now(),
		}

		readmePath := filepath.Join(dsm.RootPath, service.Path, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("README.md not found: %s", readmePath))
			result.Success = false
			results = append(results, result)
			continue
		}

		result.SourcePath = readmePath

		// Read README content
		content, err := os.ReadFile(readmePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read README: %v", err))
			result.Success = false
			results = append(results, result)
			continue
		}

		// Sync to all central documentation paths
		for _, docPath := range dsm.Config.CentralDocPaths {
			targetPath := filepath.Join(dsm.RootPath, docPath, strings.ToLower(service.Name)+".md")

			// Ensure target directory exists
			targetDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create directory %s: %v", targetDir, err))
				continue
			}

			// Write content
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to write %s: %v", targetPath, err))
			} else {
				result.TargetPaths = append(result.TargetPaths, targetPath)
				result.FilesUpdated++
				dsm.Logger.Printf("Synced README: %s -> %s", readmePath, targetPath)
			}
		}

		result.Success = len(result.Errors) == 0
		results = append(results, result)
	}

	return results, nil
}

// SyncAPISpecs synchronizes API specifications to central locations
func (dsm *DocSyncManager) SyncAPISpecs() ([]DocSyncResult, error) {
	var results []DocSyncResult

	dsm.Logger.Println("Syncing API specifications to central locations...")

	for _, service := range dsm.Config.Services {
		if !service.Enabled || !service.HasAPI {
			continue
		}

		result := DocSyncResult{
			Service:   service.Name,
			SyncType:  "api_spec",
			Timestamp: time.Now(),
		}

		apiSpecPath := filepath.Join(dsm.RootPath, service.Path, "api.yaml")
		if service.APISpecPath != "" {
			apiSpecPath = filepath.Join(dsm.RootPath, service.Path, service.APISpecPath)
		}

		if _, err := os.Stat(apiSpecPath); os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("API spec not found: %s", apiSpecPath))
			result.Success = false
			results = append(results, result)
			continue
		}

		result.SourcePath = apiSpecPath

		// Read API spec content
		content, err := os.ReadFile(apiSpecPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read API spec: %v", err))
			result.Success = false
			results = append(results, result)
			continue
		}

		// Sync to all API documentation paths
		for _, apiDocPath := range dsm.Config.APIDocPaths {
			targetPath := filepath.Join(dsm.RootPath, apiDocPath, strings.ToLower(service.Name)+"-api.yaml")

			// Ensure target directory exists
			targetDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create directory %s: %v", targetDir, err))
				continue
			}

			// Write content
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to write %s: %v", targetPath, err))
			} else {
				result.TargetPaths = append(result.TargetPaths, targetPath)
				result.FilesUpdated++
				dsm.Logger.Printf("Synced API spec: %s -> %s", apiSpecPath, targetPath)
			}
		}

		result.Success = len(result.Errors) == 0
		results = append(results, result)
	}

	return results, nil
}

// GenerateUnifiedAPI generates a unified OpenAPI specification from all service API specs
func (dsm *DocSyncManager) GenerateUnifiedAPI() (DocSyncResult, error) {
	result := DocSyncResult{
		Service:   "UNIFIED",
		SyncType:  "unified_api",
		Timestamp: time.Now(),
	}

	dsm.Logger.Println("Generating unified OpenAPI specification...")

	var unifiedSpec strings.Builder

	// Write OpenAPI header
	unifiedSpec.WriteString("openapi: 3.1.0\n")
	unifiedSpec.WriteString("info:\n")
	unifiedSpec.WriteString("  title: KNIRV Network Unified API\n")
	unifiedSpec.WriteString("  description: |\n")
	unifiedSpec.WriteString("    Unified API specification for the entire KNIRV Network ecosystem.\n")
	unifiedSpec.WriteString("    \n")
	unifiedSpec.WriteString("    This specification aggregates all service APIs into a single source of truth\n")
	unifiedSpec.WriteString("    for KNIRV SDK implementations and CLI tools.\n")
	unifiedSpec.WriteString(fmt.Sprintf("  version: \"1.0.0\"\n"))
	unifiedSpec.WriteString("  contact:\n")
	unifiedSpec.WriteString("    name: KNIRV Network Support\n")
	unifiedSpec.WriteString("    email: support@knirv.com\n\n")

	// Collect servers from all specs
	unifiedSpec.WriteString("servers:\n")
	unifiedSpec.WriteString("  - url: http://localhost:3000/api\n")
	unifiedSpec.WriteString("    description: Local development\n")
	unifiedSpec.WriteString("  - url: https://api.knirv.com/v1\n")
	unifiedSpec.WriteString("    description: Production API Gateway\n\n")

	// Collect tags and paths from all service API specs
	allTags := make(map[string]bool)
	allPaths := strings.Builder{}

	for _, service := range dsm.Config.Services {
		if !service.Enabled || !service.HasAPI {
			continue
		}

		apiSpecPath := filepath.Join(dsm.RootPath, service.Path, "api.yaml")
		if service.APISpecPath != "" {
			apiSpecPath = filepath.Join(dsm.RootPath, service.Path, service.APISpecPath)
		}

		if _, err := os.Stat(apiSpecPath); os.IsNotExist(err) {
			dsm.Logger.Printf("Warning: API spec not found for %s: %s", service.Name, apiSpecPath)
			continue
		}

		// Read and parse API spec
		content, err := os.ReadFile(apiSpecPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", service.Name, err))
			continue
		}

		// Extract tags
		tags := extractYAMLSection(string(content), "tags:")
		for _, tag := range tags {
			allTags[tag] = true
		}

		// Extract paths with service prefix comment
		paths := extractYAMLSection(string(content), "paths:")
		if len(paths) > 0 {
			allPaths.WriteString(fmt.Sprintf("\n  # ========================================\n"))
			allPaths.WriteString(fmt.Sprintf("  # %s API ENDPOINTS\n", service.Name))
			allPaths.WriteString(fmt.Sprintf("  # ========================================\n"))
			allPaths.WriteString(strings.Join(paths, "\n"))
			allPaths.WriteString("\n")
		}

		result.TargetPaths = append(result.TargetPaths, apiSpecPath)
	}

	// Write tags
	unifiedSpec.WriteString("tags:\n")
	for tag := range allTags {
		unifiedSpec.WriteString(fmt.Sprintf("  - name: %s\n", tag))
	}
	unifiedSpec.WriteString("\n")

	// Write paths
	unifiedSpec.WriteString("paths:")
	unifiedSpec.WriteString(allPaths.String())

	// Write unified spec to configured path
	unifiedPath := filepath.Join(dsm.RootPath, dsm.Config.UnifiedAPIPath)

	// Ensure directory exists
	unifiedDir := filepath.Dir(unifiedPath)
	if err := os.MkdirAll(unifiedDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create directory: %v", err))
		result.Success = false
		return result, err
	}

	if err := os.WriteFile(unifiedPath, []byte(unifiedSpec.String()), 0644); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to write unified API spec: %v", err))
		result.Success = false
		return result, err
	}

	result.FilesUpdated = 1
	result.Success = true
	dsm.Logger.Printf("Generated unified API specification: %s", unifiedPath)

	return result, nil
}

// extractYAMLSection extracts a section from YAML content
func extractYAMLSection(content, sectionHeader string) []string {
	var lines []string
	contentLines := strings.Split(content, "\n")
	inSection := false
	baseIndent := ""

	for _, line := range contentLines {
		trimmed := strings.TrimSpace(line)

		if trimmed == sectionHeader {
			inSection = true
			continue
		}

		if inSection {
			// Determine base indentation from first line in section
			if baseIndent == "" && len(line) > 0 && line[0] == ' ' {
				// Count leading spaces
				for i, char := range line {
					if char != ' ' {
						baseIndent = line[:i]
						break
					}
				}
			}

			// Check if we've exited the section (line with no indentation or different top-level key)
			if len(line) > 0 && line[0] != ' ' && line[0] != '#' {
				break
			}

			// Add line if it's part of the section
			if strings.HasPrefix(line, baseIndent) || strings.TrimSpace(line) == "" || strings.HasPrefix(trimmed, "#") {
				lines = append(lines, line)
			}
		}
	}

	return lines
}

// UpdateREADMEWithGapFixes updates a service README to include missing documentation
func (dsm *DocSyncManager) UpdateREADMEWithGapFixes(serviceName string, gaps []DocGap) error {
	readmePath := filepath.Join(dsm.RootPath, serviceName, "README.md")

	content, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("failed to read README: %w", err)
	}

	readmeStr := string(content)

	// Add missing API endpoints section if needed
	endpointGaps := []DocGap{}
	for _, gap := range gaps {
		if gap.GapType == "missing_endpoint" {
			endpointGaps = append(endpointGaps, gap)
		}
	}

	if len(endpointGaps) > 0 && !strings.Contains(readmeStr, "## API Endpoints") {
		apiSection := "\n## API Endpoints\n\n"
		apiSection += "The following endpoints are available:\n\n"

		for _, gap := range endpointGaps {
			endpoint := extractEndpointFromDescription(gap.Description)
			apiSection += fmt.Sprintf("- `%s` - TODO: Add description\n", endpoint)
		}

		apiSection += "\nFor detailed API documentation, see [api.yaml](./api.yaml).\n"

		readmeStr += apiSection
	}

	return os.WriteFile(readmePath, []byte(readmeStr), 0644)
}

// extractEndpointFromDescription extracts endpoint path from gap description
func extractEndpointFromDescription(description string) string {
	start := strings.Index(description, "'")
	if start == -1 {
		return ""
	}
	end := strings.Index(description[start+1:], "'")
	if end == -1 {
		return ""
	}
	return description[start+1 : start+1+end]
}
