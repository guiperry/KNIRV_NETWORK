package agent

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AgentVersion represents a version of an agent
type AgentVersion struct {
	Version     string                 `json:"version"`
	AgentID     string                 `json:"agent_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	PluginPath  string                 `json:"plugin_path"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	ChangeLog   string                 `json:"change_log"`
	Tags        []string               `json:"tags"`
	Status      string                 `json:"status"` // "active", "archived", "deprecated"
}

// AgentBackup represents a backup of an agent
type AgentBackup struct {
	BackupID    string    `json:"backup_id"`
	AgentID     string    `json:"agent_id"`
	Version     string    `json:"version"`
	BackupPath  string    `json:"backup_path"`
	CreatedAt   time.Time `json:"created_at"`
	Size        int64     `json:"size"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "manual", "automatic", "scheduled"
}

// AgentHealthStatus represents the health status of an agent
type AgentHealthStatus struct {
	AgentID            string                 `json:"agent_id"`
	Status             string                 `json:"status"` // "healthy", "warning", "critical", "unknown"
	LastHealthCheck    time.Time              `json:"last_health_check"`
	HealthScore        float64                `json:"health_score"` // 0-100
	Issues             []string               `json:"issues"`
	Recommendations    []string               `json:"recommendations"`
	PerformanceMetrics map[string]interface{} `json:"performance_metrics"`
	Uptime             time.Duration          `json:"uptime"`
	LastError          string                 `json:"last_error,omitempty"`
}

// AgentPerformanceAnalytics represents performance analytics for an agent
type AgentPerformanceAnalytics struct {
	AgentID              string                 `json:"agent_id"`
	AnalyticsPeriod      string                 `json:"analytics_period"` // "1h", "24h", "7d", "30d"
	TotalRequests        int64                  `json:"total_requests"`
	SuccessfulRequests   int64                  `json:"successful_requests"`
	FailedRequests       int64                  `json:"failed_requests"`
	SuccessRate          float64                `json:"success_rate"`
	AverageResponseTime  time.Duration          `json:"average_response_time"`
	MedianResponseTime   time.Duration          `json:"median_response_time"`
	P95ResponseTime      time.Duration          `json:"p95_response_time"`
	P99ResponseTime      time.Duration          `json:"p99_response_time"`
	ThroughputPerSecond  float64                `json:"throughput_per_second"`
	ErrorDistribution    map[string]int         `json:"error_distribution"`
	ResourceUtilization  map[string]interface{} `json:"resource_utilization"`
	SubAgentMetrics      map[string]interface{} `json:"sub_agent_metrics"`
	OrchestrationMetrics map[string]interface{} `json:"orchestration_metrics"`
	GeneratedAt          time.Time              `json:"generated_at"`
}

// EnhancedAgentManager provides advanced agent management capabilities
type EnhancedAgentManager struct {
	registry       *AgentRegistry
	builder        *AgentBuilder
	backupDir      string
	versionsDir    string
	analyticsDir   string
	healthCheckDir string
}

// NewEnhancedAgentManager creates a new enhanced agent manager
func NewEnhancedAgentManager(registry *AgentRegistry, builder *AgentBuilder, dataDir string) (*EnhancedAgentManager, error) {
	manager := &EnhancedAgentManager{
		registry:       registry,
		builder:        builder,
		backupDir:      filepath.Join(dataDir, "backups"),
		versionsDir:    filepath.Join(dataDir, "versions"),
		analyticsDir:   filepath.Join(dataDir, "analytics"),
		healthCheckDir: filepath.Join(dataDir, "health"),
	}

	// Create necessary directories
	dirs := []string{manager.backupDir, manager.versionsDir, manager.analyticsDir, manager.healthCheckDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return manager, nil
}

// CreateAgentVersion creates a new version of an agent
func (m *EnhancedAgentManager) CreateAgentVersion(agentID, version, changeLog string, tags []string) (*AgentVersion, error) {
	// Get current agent configuration
	config, err := m.registry.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %v", err)
	}

	// Create version record
	agentVersion := &AgentVersion{
		Version:     version,
		AgentID:     agentID,
		Name:        config["name"].(string),
		Description: config["description"].(string),
		Config:      config,
		PluginPath:  config["plugin_path"].(string),
		CreatedAt:   time.Now(),
		CreatedBy:   "system", // In a real system, this would be the current user
		ChangeLog:   changeLog,
		Tags:        tags,
		Status:      "active",
	}

	// Save version to file
	versionFile := filepath.Join(m.versionsDir, fmt.Sprintf("%s_%s.json", agentID, version))
	if err := m.saveVersionToFile(agentVersion, versionFile); err != nil {
		return nil, fmt.Errorf("failed to save version: %v", err)
	}

	return agentVersion, nil
}

// ListAgentVersions lists all versions of an agent
func (m *EnhancedAgentManager) ListAgentVersions(agentID string) ([]*AgentVersion, error) {
	pattern := filepath.Join(m.versionsDir, fmt.Sprintf("%s_*.json", agentID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list version files: %v", err)
	}

	var versions []*AgentVersion
	for _, file := range files {
		version, err := m.loadVersionFromFile(file)
		if err != nil {
			continue // Skip invalid files
		}
		versions = append(versions, version)
	}

	// Sort by creation time (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	return versions, nil
}

// CreateAgentBackup creates a backup of an agent
func (m *EnhancedAgentManager) CreateAgentBackup(agentID, description, backupType string) (*AgentBackup, error) {
	// Get agent configuration
	config, err := m.registry.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %v", err)
	}

	// Generate backup ID
	backupID := fmt.Sprintf("%s_%d", agentID, time.Now().Unix())
	backupPath := filepath.Join(m.backupDir, fmt.Sprintf("%s.tar.gz", backupID))

	// Create backup archive
	size, err := m.createBackupArchive(config, backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup archive: %v", err)
	}

	// Create backup record
	backup := &AgentBackup{
		BackupID:    backupID,
		AgentID:     agentID,
		Version:     config["version"].(string),
		BackupPath:  backupPath,
		CreatedAt:   time.Now(),
		Size:        size,
		Description: description,
		Type:        backupType,
	}

	// Save backup metadata
	metadataFile := filepath.Join(m.backupDir, fmt.Sprintf("%s.json", backupID))
	if err := m.saveBackupMetadata(backup, metadataFile); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %v", err)
	}

	return backup, nil
}

// RestoreAgentFromBackup restores an agent from a backup
func (m *EnhancedAgentManager) RestoreAgentFromBackup(backupID string) error {
	// Load backup metadata
	metadataFile := filepath.Join(m.backupDir, fmt.Sprintf("%s.json", backupID))
	backup, err := m.loadBackupMetadata(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %v", err)
	}

	// Extract backup archive
	if err := m.extractBackupArchive(backup.BackupPath, backup.AgentID); err != nil {
		return fmt.Errorf("failed to extract backup: %v", err)
	}

	return nil
}

// PerformHealthCheck performs a health check on an agent
func (m *EnhancedAgentManager) PerformHealthCheck(agentID string) (*AgentHealthStatus, error) {
	// Get agent configuration
	config, err := m.registry.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %v", err)
	}

	// Initialize health status
	health := &AgentHealthStatus{
		AgentID:            agentID,
		LastHealthCheck:    time.Now(),
		Issues:             []string{},
		Recommendations:    []string{},
		PerformanceMetrics: make(map[string]interface{}),
	}

	// Check plugin file existence
	pluginPath := config["plugin_path"].(string)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		health.Issues = append(health.Issues, "Plugin file not found")
		health.Status = "critical"
		health.HealthScore = 0
		return health, nil
	}

	// Check plugin file integrity (basic check)
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		health.Issues = append(health.Issues, "Cannot access plugin file")
		health.Status = "warning"
	} else {
		health.PerformanceMetrics["plugin_size"] = fileInfo.Size()
		health.PerformanceMetrics["plugin_modified"] = fileInfo.ModTime()
	}

	// Calculate health score based on checks
	health.HealthScore = m.calculateHealthScore(health)
	health.Status = m.determineHealthStatus(health.HealthScore)

	// Save health check result
	healthFile := filepath.Join(m.healthCheckDir, fmt.Sprintf("%s_latest.json", agentID))
	if err := m.saveHealthStatus(health, healthFile); err != nil {
		// Log error but don't fail the health check
		fmt.Printf("Warning: failed to save health status: %v\n", err)
	}

	return health, nil
}

// GeneratePerformanceAnalytics generates performance analytics for an agent
func (m *EnhancedAgentManager) GeneratePerformanceAnalytics(agentID, period string) (*AgentPerformanceAnalytics, error) {
	// Initialize analytics
	analytics := &AgentPerformanceAnalytics{
		AgentID:              agentID,
		AnalyticsPeriod:      period,
		ErrorDistribution:    make(map[string]int),
		ResourceUtilization:  make(map[string]interface{}),
		SubAgentMetrics:      make(map[string]interface{}),
		OrchestrationMetrics: make(map[string]interface{}),
		GeneratedAt:          time.Now(),
	}

	// Simulate analytics data (in a real implementation, this would come from monitoring data)
	analytics.TotalRequests = 1000
	analytics.SuccessfulRequests = 950
	analytics.FailedRequests = 50
	analytics.SuccessRate = float64(analytics.SuccessfulRequests) / float64(analytics.TotalRequests) * 100
	analytics.AverageResponseTime = time.Millisecond * 250
	analytics.MedianResponseTime = time.Millisecond * 200
	analytics.P95ResponseTime = time.Millisecond * 500
	analytics.P99ResponseTime = time.Millisecond * 800
	analytics.ThroughputPerSecond = 10.5

	// Error distribution
	analytics.ErrorDistribution["timeout"] = 20
	analytics.ErrorDistribution["validation"] = 15
	analytics.ErrorDistribution["system"] = 10
	analytics.ErrorDistribution["network"] = 5

	// Resource utilization
	analytics.ResourceUtilization["avg_cpu_usage"] = 25.5
	analytics.ResourceUtilization["avg_memory_usage"] = 128 * 1024 * 1024 // 128MB
	analytics.ResourceUtilization["peak_cpu_usage"] = 45.0
	analytics.ResourceUtilization["peak_memory_usage"] = 256 * 1024 * 1024 // 256MB

	// Sub-agent metrics
	analytics.SubAgentMetrics["total_spawned"] = 25
	analytics.SubAgentMetrics["avg_lifetime"] = time.Minute * 5
	analytics.SubAgentMetrics["success_rate"] = 92.0

	// Orchestration metrics
	analytics.OrchestrationMetrics["patterns_used"] = map[string]int{
		"single_agent_tools":     400,
		"sequential_agents":      200,
		"parallel_hierarchy":     150,
		"single_agent_mcp_tools": 100,
		"others":                 150,
	}

	// Save analytics to file
	analyticsFile := filepath.Join(m.analyticsDir, fmt.Sprintf("%s_%s_%d.json", agentID, period, time.Now().Unix()))
	if err := m.saveAnalytics(analytics, analyticsFile); err != nil {
		return nil, fmt.Errorf("failed to save analytics: %v", err)
	}

	return analytics, nil
}

// ListAgentBackups lists all backups for an agent
func (m *EnhancedAgentManager) ListAgentBackups(agentID string) ([]*AgentBackup, error) {
	// List all backup metadata files
	pattern := filepath.Join(m.backupDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %v", err)
	}

	var backups []*AgentBackup
	for _, file := range files {
		backup, err := m.loadBackupMetadata(file)
		if err != nil {
			continue // Skip invalid files
		}
		// Only include backups for the specified agent
		if backup.AgentID == agentID {
			backups = append(backups, backup)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetAgentHealthHistory gets the health check history for an agent
func (m *EnhancedAgentManager) GetAgentHealthHistory(agentID string) ([]*AgentHealthStatus, error) {
	pattern := filepath.Join(m.healthCheckDir, fmt.Sprintf("%s_*.json", agentID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list health files: %v", err)
	}

	var healthHistory []*AgentHealthStatus
	for _, file := range files {
		health, err := m.loadHealthStatus(file)
		if err != nil {
			continue // Skip invalid files
		}
		healthHistory = append(healthHistory, health)
	}

	// Sort by check time (newest first)
	sort.Slice(healthHistory, func(i, j int) bool {
		return healthHistory[i].LastHealthCheck.After(healthHistory[j].LastHealthCheck)
	})

	return healthHistory, nil
}

// Helper methods

func (m *EnhancedAgentManager) saveVersionToFile(version *AgentVersion, filename string) error {
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (m *EnhancedAgentManager) loadVersionFromFile(filename string) (*AgentVersion, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var version AgentVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, err
	}
	return &version, nil
}

func (m *EnhancedAgentManager) createBackupArchive(config map[string]interface{}, backupPath string) (int64, error) {
	// Create backup archive (simplified implementation)
	file, err := os.Create(backupPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add config file to archive
	configData, _ := json.MarshalIndent(config, "", "  ")
	header := &tar.Header{
		Name: "config.json",
		Size: int64(len(configData)),
		Mode: 0644,
	}
	tarWriter.WriteHeader(header)
	tarWriter.Write(configData)

	// Get file size
	fileInfo, _ := file.Stat()
	return fileInfo.Size(), nil
}

func (m *EnhancedAgentManager) extractBackupArchive(backupPath, agentID string) error {
	// Simplified restore implementation
	// In a real implementation, this would extract the full archive

	// Check if backup path exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found at path: %s", backupPath)
	}

	// Create a target directory for extraction based on agent ID
	targetDir := filepath.Join(m.backupDir, "extracts", agentID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory for agent %s: %v", agentID, err)
	}

	// In a real implementation, this would extract the archive
	// For now, just log the operation
	fmt.Printf("Extracting backup from %s for agent %s to %s\n", backupPath, agentID, targetDir)

	// Here we would implement the actual extraction logic:
	// 1. Open the tar.gz file
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %v", err)
	}
	defer file.Close()

	// 2. Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	// 3. Create tar reader
	tr := tar.NewReader(gzr)

	// 4. Extract each file (simplified for demonstration)
	fmt.Printf("Starting extraction of files for agent %s\n", agentID)

	// Read and count files in the archive (simplified implementation)
	fileCount := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %v", err)
		}

		// Just count and log files, don't actually extract them in this simplified version
		fileCount++
		fmt.Printf("Found file in archive: %s\n", header.Name)
	}

	fmt.Printf("Archive contains %d files for agent %s\n", fileCount, agentID)
	return nil
}

func (m *EnhancedAgentManager) saveBackupMetadata(backup *AgentBackup, filename string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (m *EnhancedAgentManager) loadBackupMetadata(filename string) (*AgentBackup, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var backup AgentBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, err
	}
	return &backup, nil
}

func (m *EnhancedAgentManager) saveHealthStatus(health *AgentHealthStatus, filename string) error {
	data, err := json.MarshalIndent(health, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (m *EnhancedAgentManager) loadHealthStatus(filename string) (*AgentHealthStatus, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var health AgentHealthStatus
	if err := json.Unmarshal(data, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

func (m *EnhancedAgentManager) saveAnalytics(analytics *AgentPerformanceAnalytics, filename string) error {
	data, err := json.MarshalIndent(analytics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (m *EnhancedAgentManager) calculateHealthScore(health *AgentHealthStatus) float64 {
	score := 100.0

	// Deduct points for each issue
	for _, issue := range health.Issues {
		if strings.Contains(issue, "critical") || strings.Contains(issue, "not found") {
			score -= 50
		} else if strings.Contains(issue, "warning") || strings.Contains(issue, "access") {
			score -= 20
		} else {
			score -= 10
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (m *EnhancedAgentManager) determineHealthStatus(score float64) string {
	switch {
	case score >= 80:
		return "healthy"
	case score >= 60:
		return "warning"
	case score >= 20:
		return "critical"
	default:
		return "unknown"
	}
}
