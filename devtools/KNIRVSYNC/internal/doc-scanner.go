package sync

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DocGap represents a detected gap between code and documentation
type DocGap struct {
	Service     string   `json:"service"`
	GapType     string   `json:"gap_type"` // "missing_endpoint", "undocumented_function", "outdated_api"
	Location    string   `json:"location"`
	Description string   `json:"description"`
	Suggestions []string `json:"suggestions"`
}

// DocScanner scans for documentation gaps
type DocScanner struct {
	RootPath string
	Logger   Logger
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// NewDocScanner creates a new documentation scanner
func NewDocScanner(rootPath string, logger Logger) *DocScanner {
	return &DocScanner{
		RootPath: rootPath,
		Logger:   logger,
	}
}

// ScanService scans a specific service for documentation gaps
func (ds *DocScanner) ScanService(serviceName string) ([]DocGap, error) {
	var gaps []DocGap

	servicePath := filepath.Join(ds.RootPath, serviceName)
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("service path does not exist: %s", servicePath)
	}

	ds.Logger.Printf("Scanning service: %s", serviceName)

	// Scan for API endpoints in code vs api.yaml
	apiGaps, err := ds.scanAPIEndpoints(serviceName, servicePath)
	if err != nil {
		ds.Logger.Printf("Warning: failed to scan API endpoints for %s: %v", serviceName, err)
	} else {
		gaps = append(gaps, apiGaps...)
	}

	// Scan for undocumented functions in README
	funcGaps, err := ds.scanUndocumentedFunctions(serviceName, servicePath)
	if err != nil {
		ds.Logger.Printf("Warning: failed to scan functions for %s: %v", serviceName, err)
	} else {
		gaps = append(gaps, funcGaps...)
	}

	// Scan for missing API spec
	if apiSpecGap := ds.checkAPISpec(serviceName, servicePath); apiSpecGap != nil {
		gaps = append(gaps, *apiSpecGap)
	}

	ds.Logger.Printf("Found %d documentation gaps in %s", len(gaps), serviceName)
	return gaps, nil
}

// ScanAllServices scans all KNIRV services for documentation gaps
func (ds *DocScanner) ScanAllServices() (map[string][]DocGap, error) {
	results := make(map[string][]DocGap)

	entries, err := os.ReadDir(ds.RootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "KNIRV") {
			continue
		}

		// Skip KNIRVSYNC itself
		if entry.Name() == "KNIRVSYNC" {
			continue
		}

		gaps, err := ds.ScanService(entry.Name())
		if err != nil {
			ds.Logger.Printf("Error scanning %s: %v", entry.Name(), err)
			continue
		}

		if len(gaps) > 0 {
			results[entry.Name()] = gaps
		}
	}

	return results, nil
}

// scanAPIEndpoints compares API endpoints in code with api.yaml
func (ds *DocScanner) scanAPIEndpoints(serviceName, servicePath string) ([]DocGap, error) {
	var gaps []DocGap

	// Read api.yaml if it exists
	apiYAMLPath := filepath.Join(servicePath, "api.yaml")
	documentedEndpoints := make(map[string]bool)

	if _, err := os.Stat(apiYAMLPath); err == nil {
		endpoints, err := ds.extractEndpointsFromAPIYAML(apiYAMLPath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract endpoints from api.yaml: %w", err)
		}
		for _, ep := range endpoints {
			documentedEndpoints[ep] = true
		}
	}

	// Extract endpoints from code
	codeEndpoints, err := ds.extractEndpointsFromCode(servicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract endpoints from code: %w", err)
	}

	// Find undocumented endpoints
	for endpoint, files := range codeEndpoints {
		if !documentedEndpoints[endpoint] {
			gaps = append(gaps, DocGap{
				Service:     serviceName,
				GapType:     "missing_endpoint",
				Location:    strings.Join(files, ", "),
				Description: fmt.Sprintf("API endpoint '%s' exists in code but not documented in api.yaml", endpoint),
				Suggestions: []string{
					fmt.Sprintf("Add '%s' to api.yaml with proper OpenAPI specification", endpoint),
					"Include request/response schemas and authentication requirements",
				},
			})
		}
	}

	return gaps, nil
}

// extractEndpointsFromAPIYAML extracts endpoint paths from OpenAPI YAML
func (ds *DocScanner) extractEndpointsFromAPIYAML(yamlPath string) ([]string, error) {
	var endpoints []string

	file, err := os.Open(yamlPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inPathsSection := false
	pathRegex := regexp.MustCompile(`^  (/[^\s:]+):`)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "paths:" {
			inPathsSection = true
			continue
		}

		if inPathsSection {
			// Check if we've exited the paths section
			if len(line) > 0 && line[0] != ' ' {
				break
			}

			if matches := pathRegex.FindStringSubmatch(line); len(matches) > 1 {
				endpoints = append(endpoints, matches[1])
			}
		}
	}

	return endpoints, scanner.Err()
}

// extractEndpointsFromCode searches for HTTP route definitions in code
func (ds *DocScanner) extractEndpointsFromCode(servicePath string) (map[string][]string, error) {
	endpoints := make(map[string][]string)

	// Patterns for different frameworks/languages
	patterns := []*regexp.Regexp{
		// Go: r.HandleFunc("/path", handler) or mux.Get("/path", handler)
		regexp.MustCompile(`(?:HandleFunc|Get|Post|Put|Delete|Patch)\s*\(\s*"(/[^"]+)"`),
		// Go: Route("GET", "/path", handler)
		regexp.MustCompile(`Route\s*\(\s*"[A-Z]+"\s*,\s*"(/[^"]+)"`),
		// Node.js: app.get('/path', handler) or router.post('/path', handler)
		regexp.MustCompile(`(?:app|router)\.(get|post|put|delete|patch)\s*\(\s*['"](/[^'"]+)`),
		// Express: @Get("/path") decorator
		regexp.MustCompile(`@(?:Get|Post|Put|Delete|Patch)\s*\(\s*['"]([^'"]+)`),
	}

	err := filepath.WalkDir(servicePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip common directories
			if d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan code files
		ext := filepath.Ext(d.Name())
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".rs" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		for _, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) >= 2 {
					endpoint := match[len(match)-1]
					// Normalize endpoint
					if !strings.HasPrefix(endpoint, "/") {
						endpoint = "/" + endpoint
					}
					relPath, _ := filepath.Rel(servicePath, path)
					endpoints[endpoint] = append(endpoints[endpoint], relPath)
				}
			}
		}

		return nil
	})

	return endpoints, err
}

// scanUndocumentedFunctions looks for exported functions not mentioned in README
func (ds *DocScanner) scanUndocumentedFunctions(serviceName, servicePath string) ([]DocGap, error) {
	var gaps []DocGap

	// Read README if it exists
	readmePath := filepath.Join(servicePath, "README.md")
	readmeContent := ""

	if data, err := os.ReadFile(readmePath); err == nil {
		readmeContent = strings.ToLower(string(data))
	}

	// Find exported functions in Go files
	exportedFuncs, err := ds.findExportedFunctions(servicePath)
	if err != nil {
		return nil, err
	}

	// Check if functions are documented
	for funcName, location := range exportedFuncs {
		if !strings.Contains(readmeContent, strings.ToLower(funcName)) {
			gaps = append(gaps, DocGap{
				Service:     serviceName,
				GapType:     "undocumented_function",
				Location:    location,
				Description: fmt.Sprintf("Exported function '%s' is not mentioned in README.md", funcName),
				Suggestions: []string{
					fmt.Sprintf("Add documentation for '%s' to README.md", funcName),
					"Include usage examples and parameter descriptions",
				},
			})
		}
	}

	return gaps, nil
}

// findExportedFunctions finds exported functions in Go code
func (ds *DocScanner) findExportedFunctions(servicePath string) (map[string]string, error) {
	functions := make(map[string]string)

	// Pattern for exported Go functions (starts with capital letter)
	funcPattern := regexp.MustCompile(`func\s+([A-Z][a-zA-Z0-9_]*)\s*\(`)

	err := filepath.WalkDir(servicePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(d.Name()) != ".go" {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		matches := funcPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) >= 2 {
				funcName := match[1]
				relPath, _ := filepath.Rel(servicePath, path)
				functions[funcName] = relPath
			}
		}

		return nil
	})

	return functions, err
}

// checkAPISpec checks if api.yaml exists and is valid
func (ds *DocScanner) checkAPISpec(serviceName, servicePath string) *DocGap {
	apiYAMLPath := filepath.Join(servicePath, "api.yaml")

	if _, err := os.Stat(apiYAMLPath); os.IsNotExist(err) {
		// Check if this service should have an API
		hasAPI, _ := ds.serviceHasAPI(servicePath)
		if hasAPI {
			return &DocGap{
				Service:     serviceName,
				GapType:     "missing_api_spec",
				Location:    servicePath,
				Description: "Service appears to have API endpoints but no api.yaml specification",
				Suggestions: []string{
					"Create api.yaml with OpenAPI 3.1.0 specification",
					"Document all endpoints, request/response schemas, and authentication",
					"Use existing api.yaml files from other services as templates",
				},
			}
		}
	}

	return nil
}

// serviceHasAPI checks if a service appears to have API endpoints
func (ds *DocScanner) serviceHasAPI(servicePath string) (bool, error) {
	endpoints, err := ds.extractEndpointsFromCode(servicePath)
	if err != nil {
		return false, err
	}

	return len(endpoints) > 0, nil
}

// GenerateGapReport generates a markdown report of documentation gaps
func (ds *DocScanner) GenerateGapReport(gaps map[string][]DocGap, outputPath string) error {
	var report strings.Builder

	report.WriteString("# KNIRV Network Documentation Gap Analysis\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", filepath.Base(outputPath)))

	totalGaps := 0
	for _, serviceGaps := range gaps {
		totalGaps += len(serviceGaps)
	}

	report.WriteString(fmt.Sprintf("## Summary\n\n"))
	report.WriteString(fmt.Sprintf("- Services Scanned: %d\n", len(gaps)))
	report.WriteString(fmt.Sprintf("- Total Gaps Found: %d\n\n", totalGaps))

	// Group by service
	for service, serviceGaps := range gaps {
		report.WriteString(fmt.Sprintf("## %s (%d gaps)\n\n", service, len(serviceGaps)))

		// Group by gap type
		gapsByType := make(map[string][]DocGap)
		for _, gap := range serviceGaps {
			gapsByType[gap.GapType] = append(gapsByType[gap.GapType], gap)
		}

		for gapType, typeGaps := range gapsByType {
			report.WriteString(fmt.Sprintf("### %s\n\n", strings.Title(strings.ReplaceAll(gapType, "_", " "))))

			for i, gap := range typeGaps {
				report.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, gap.Description))
				report.WriteString(fmt.Sprintf("   - Location: `%s`\n", gap.Location))
				if len(gap.Suggestions) > 0 {
					report.WriteString("   - Suggestions:\n")
					for _, suggestion := range gap.Suggestions {
						report.WriteString(fmt.Sprintf("     - %s\n", suggestion))
					}
				}
				report.WriteString("\n")
			}
		}
	}

	return os.WriteFile(outputPath, []byte(report.String()), 0644)
}
