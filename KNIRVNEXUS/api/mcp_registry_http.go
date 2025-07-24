package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// MCPServer represents an MCP server from the registry
type MCPServer struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Type           string            `json:"type"` // "typescript" | "python"
	Category       string            `json:"category"`
	Repository     string            `json:"repository"`
	InstallCommand string            `json:"install_command"`
	RunCommand     string            `json:"run_command"`
	Status         string            `json:"status"` // "available" | "installing" | "installed" | "running" | "error"
	Version        string            `json:"version"`
	Dependencies   []string          `json:"dependencies"`
	Configuration  map[string]string `json:"configuration"`
	Tags           []string          `json:"tags"`
	Rating         float64           `json:"rating"`
	Downloads      int               `json:"downloads"`
	LastUpdated    time.Time         `json:"last_updated"`
	OwnerID        *int64            `json:"owner_id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// MCPServerFilter represents filtering options for MCP servers
type MCPServerFilter struct {
	Category string   `json:"category"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Search   string   `json:"search"`
	Tags     []string `json:"tags"`
}

// MCPRegistryService manages MCP server registry
type MCPRegistryService struct {
	servers      map[string]*MCPServer
	mutex        sync.RWMutex
	lastSync     time.Time
	syncInterval time.Duration
	httpClient   *http.Client
}

// NewMCPRegistryService creates a new MCP registry service
func NewMCPRegistryService() *MCPRegistryService {
	return &MCPRegistryService{
		servers:      make(map[string]*MCPServer),
		syncInterval: 1 * time.Hour, // Sync every hour
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins the registry service with periodic syncing
func (s *MCPRegistryService) Start(ctx context.Context) error {
	// Initial sync
	if err := s.SyncWithGitHub(ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Start periodic sync
	go s.periodicSync(ctx)

	return nil
}

// periodicSync runs periodic synchronization with GitHub
func (s *MCPRegistryService) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.SyncWithGitHub(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Periodic sync failed: %v\n", err)
			}
		}
	}
}

// SyncWithGitHub fetches the latest MCP servers from GitHub repository
func (s *MCPRegistryService) SyncWithGitHub(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Fetch README content from GitHub
	readme, err := s.fetchGitHubReadme(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch README: %w", err)
	}

	// Parse servers from README
	servers, err := s.parseServersFromReadme(readme)
	if err != nil {
		return fmt.Errorf("failed to parse servers: %w", err)
	}

	// Update local registry
	for _, server := range servers {
		if existing, exists := s.servers[server.ID]; exists {
			// Update existing server
			server.Status = existing.Status
			server.OwnerID = existing.OwnerID
			server.CreatedAt = existing.CreatedAt
		} else {
			// New server
			server.Status = "available"
			server.CreatedAt = time.Now()
		}
		server.UpdatedAt = time.Now()
		s.servers[server.ID] = server
	}

	s.lastSync = time.Now()
	return nil
}

// fetchGitHubReadme fetches the README content from GitHub
func (s *MCPRegistryService) fetchGitHubReadme(ctx context.Context) (string, error) {
	url := "https://raw.githubusercontent.com/modelcontextprotocol/servers/main/README.md"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parseServersFromReadme parses MCP servers from README content
func (s *MCPRegistryService) parseServersFromReadme(readme string) ([]*MCPServer, error) {
	var servers []*MCPServer

	// Parse reference servers
	referenceServers := s.parseReferenceServers(readme)
	servers = append(servers, referenceServers...)

	// Parse third-party servers
	thirdPartyServers := s.parseThirdPartyServers(readme)
	servers = append(servers, thirdPartyServers...)

	return servers, nil
}

// parseReferenceServers parses reference servers from README
func (s *MCPRegistryService) parseReferenceServers(readme string) []*MCPServer {
	var servers []*MCPServer

	// Find reference servers section
	refSection := s.extractSection(readme, "🌟 Reference Servers", "### Archived")
	if refSection == "" {
		return servers
	}

	// Parse server entries
	serverRegex := regexp.MustCompile(`\*\*\[([^\]]+)\]\([^)]+\)\*\* - (.+)`)
	matches := serverRegex.FindAllStringSubmatch(refSection, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1]
			description := match[2]

			server := &MCPServer{
				ID:          s.generateServerID(name),
				Name:        name,
				Description: description,
				Type:        s.detectServerType(name, description),
				Category:    s.categorizeServer(name, description),
				Repository:  "https://github.com/modelcontextprotocol/servers",
				Tags:        s.extractTags(description),
				Rating:      4.5,  // Default rating for reference servers
				Downloads:   1000, // Default downloads
				LastUpdated: time.Now(),
			}

			// Set install and run commands based on type
			s.setServerCommands(server)

			servers = append(servers, server)
		}
	}

	return servers
}

// parseThirdPartyServers parses third-party servers from README
func (s *MCPRegistryService) parseThirdPartyServers(readme string) []*MCPServer {
	var servers []*MCPServer

	// Find third-party servers section
	thirdPartySection := s.extractSection(readme, "🤝 Third-Party Servers", "📚 Frameworks")
	if thirdPartySection == "" {
		return servers
	}

	// Parse server entries with GitHub links
	serverRegex := regexp.MustCompile(`\*\*\[([^\]]+)\]\(([^)]+)\)\*\* - (.+)`)
	matches := serverRegex.FindAllStringSubmatch(thirdPartySection, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			name := match[1]
			repository := match[2]
			description := match[3]

			server := &MCPServer{
				ID:          s.generateServerID(name),
				Name:        name,
				Description: description,
				Type:        s.detectServerType(name, description),
				Category:    s.categorizeServer(name, description),
				Repository:  repository,
				Tags:        s.extractTags(description),
				Rating:      4.0, // Default rating for third-party servers
				Downloads:   500, // Default downloads
				LastUpdated: time.Now(),
			}

			// Set install and run commands based on repository
			s.setThirdPartyCommands(server)

			servers = append(servers, server)
		}
	}

	return servers
}

// extractSection extracts a section from README between two headers
func (s *MCPRegistryService) extractSection(readme, startHeader, endHeader string) string {
	startIdx := strings.Index(readme, startHeader)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(readme[startIdx:], endHeader)
	if endIdx == -1 {
		return readme[startIdx:]
	}

	return readme[startIdx : startIdx+endIdx]
}

// generateServerID generates a unique ID for a server
func (s *MCPRegistryService) generateServerID(name string) string {
	// Convert name to lowercase and replace spaces with hyphens
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	// Remove special characters
	id = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(id, "")
	return id
}

// detectServerType detects if server is TypeScript or Python based
func (s *MCPRegistryService) detectServerType(name, description string) string {
	lowerDesc := strings.ToLower(description)
	lowerName := strings.ToLower(name)

	if strings.Contains(lowerDesc, "python") || strings.Contains(lowerName, "python") {
		return "python"
	}
	if strings.Contains(lowerDesc, "typescript") || strings.Contains(lowerName, "typescript") {
		return "typescript"
	}

	// Default to typescript for reference servers
	return "typescript"
}

// categorizeServer categorizes server based on name and description
func (s *MCPRegistryService) categorizeServer(name, description string) string {
	lowerDesc := strings.ToLower(description)
	lowerName := strings.ToLower(name)

	categories := map[string][]string{
		"web":      {"web", "browser", "fetch", "http", "api"},
		"file":     {"file", "filesystem", "git", "directory"},
		"data":     {"database", "sql", "postgres", "redis", "data"},
		"ai":       {"vision", "image", "nlp", "openai", "anthropic"},
		"system":   {"system", "terminal", "command", "process"},
		"security": {"security", "auth", "token", "encryption"},
		"cloud":    {"aws", "azure", "gcp", "cloud", "s3"},
		"social":   {"slack", "discord", "twitter", "social"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(lowerDesc, keyword) || strings.Contains(lowerName, keyword) {
				return category
			}
		}
	}

	return "general"
}

// extractTags extracts tags from description
func (s *MCPRegistryService) extractTags(description string) []string {
	var tags []string

	// Common tag patterns
	tagPatterns := []string{
		"web", "api", "database", "file", "git", "vision", "ai", "security",
		"cloud", "social", "automation", "monitoring", "analytics",
	}

	lowerDesc := strings.ToLower(description)
	for _, pattern := range tagPatterns {
		if strings.Contains(lowerDesc, pattern) {
			tags = append(tags, pattern)
		}
	}

	return tags
}

// setServerCommands sets install and run commands for reference servers
func (s *MCPRegistryService) setServerCommands(server *MCPServer) {
	serverPath := strings.ToLower(strings.ReplaceAll(server.Name, " ", ""))

	if server.Type == "python" {
		server.InstallCommand = fmt.Sprintf("uvx mcp-server-%s", serverPath)
		server.RunCommand = fmt.Sprintf("uvx mcp-server-%s", serverPath)
	} else {
		server.InstallCommand = fmt.Sprintf("npx -y @modelcontextprotocol/server-%s", serverPath)
		server.RunCommand = fmt.Sprintf("npx @modelcontextprotocol/server-%s", serverPath)
	}
}

// setThirdPartyCommands sets install and run commands for third-party servers
func (s *MCPRegistryService) setThirdPartyCommands(server *MCPServer) {
	// Extract package name from repository URL
	if strings.Contains(server.Repository, "github.com") {
		parts := strings.Split(server.Repository, "/")
		if len(parts) >= 2 {
			packageName := parts[len(parts)-1]

			if server.Type == "python" {
				server.InstallCommand = fmt.Sprintf("uvx %s", packageName)
				server.RunCommand = fmt.Sprintf("uvx %s", packageName)
			} else {
				server.InstallCommand = fmt.Sprintf("npx -y %s", packageName)
				server.RunCommand = fmt.Sprintf("npx %s", packageName)
			}
		}
	}
}

// GetServers returns all servers with optional filtering
func (s *MCPRegistryService) GetServers(filter MCPServerFilter) []*MCPServer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*MCPServer

	for _, server := range s.servers {
		if s.matchesFilter(server, filter) {
			result = append(result, server)
		}
	}

	return result
}

// GetServerByID returns a server by ID
func (s *MCPRegistryService) GetServerByID(id string) (*MCPServer, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	server, exists := s.servers[id]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", id)
	}

	return server, nil
}

// matchesFilter checks if server matches the given filter
func (s *MCPRegistryService) matchesFilter(server *MCPServer, filter MCPServerFilter) bool {
	if filter.Category != "" && server.Category != filter.Category {
		return false
	}

	if filter.Type != "" && server.Type != filter.Type {
		return false
	}

	if filter.Status != "" && server.Status != filter.Status {
		return false
	}

	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(server.Name), searchLower) &&
			!strings.Contains(strings.ToLower(server.Description), searchLower) {
			return false
		}
	}

	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, serverTag := range server.Tags {
				if strings.EqualFold(filterTag, serverTag) {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

// HTTP Handlers

// GetServersHandler handles GET /api/v1/mcp/servers
func (s *MCPRegistryService) GetServersHandler(w http.ResponseWriter, r *http.Request) {
	filter := MCPServerFilter{
		Category: r.URL.Query().Get("category"),
		Type:     r.URL.Query().Get("type"),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
	}

	// Parse tags from query parameter
	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		filter.Tags = strings.Split(tagsParam, ",")
	}

	servers := s.GetServers(filter)

	response := map[string]interface{}{
		"servers":   servers,
		"count":     len(servers),
		"last_sync": s.lastSync,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServerHandler handles GET /api/v1/mcp/servers/{id}
func (s *MCPRegistryService) GetServerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	server, err := s.GetServerByID(id)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"server": server,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncServersHandler handles POST /api/v1/mcp/servers/sync
func (s *MCPRegistryService) SyncServersHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.SyncWithGitHub(r.Context()); err != nil {
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "Sync completed successfully",
		"last_sync": s.lastSync,
		"count":     len(s.servers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ClearDemoData removes all demo/sample MCP servers from the registry
func (s *MCPRegistryService) ClearDemoData() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear all servers from the registry
	s.servers = make(map[string]*MCPServer)
	s.lastSync = time.Time{} // Reset sync time
	log.Println("All MCP demo data cleared from registry")
	return nil
}

// RegisterHandlers registers the MCP registry HTTP handlers
func (s *MCPRegistryService) RegisterHandlers(router *mux.Router) {
	mcpRouter := router.PathPrefix("/api/v1/mcp").Subrouter()
	mcpRouter.HandleFunc("/servers", s.GetServersHandler).Methods("GET")
	mcpRouter.HandleFunc("/servers/{id}", s.GetServerHandler).Methods("GET")
	mcpRouter.HandleFunc("/servers/sync", s.SyncServersHandler).Methods("POST")
}
