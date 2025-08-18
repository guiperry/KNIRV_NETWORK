package api

import (
	"Agentic_Engine/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// MCPServerMetrics represents metrics for an MCP server
type MCPServerMetrics struct {
	ServerID        string    `json:"server_id"`
	Status          string    `json:"status"`
	Uptime          int64     `json:"uptime"` // seconds
	RestartCount    int       `json:"restart_count"`
	LastRestart     time.Time `json:"last_restart"`
	MemoryUsage     int64     `json:"memory_usage"` // bytes
	CPUUsage        float64   `json:"cpu_usage"`    // percentage
	RequestCount    int64     `json:"request_count"`
	ErrorCount      int64     `json:"error_count"`
	LastHealthCheck time.Time `json:"last_health_check"`
	HealthStatus    string    `json:"health_status"` // "healthy" | "unhealthy" | "unknown"
	LogLevel        string    `json:"log_level"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// Enhanced production metrics
	ResponseTime       float64                `json:"response_time_ms"`
	ThroughputRPS      float64                `json:"throughput_rps"`
	SuccessRate        float64                `json:"success_rate"`
	DiskUsage          int64                  `json:"disk_usage_bytes"`
	NetworkBytesIn     int64                  `json:"network_bytes_in"`
	NetworkBytesOut    int64                  `json:"network_bytes_out"`
	ActiveConnections  int                    `json:"active_connections"`
	QueueSize          int                    `json:"queue_size"`
	LastError          string                 `json:"last_error,omitempty"`
	LastErrorTime      time.Time              `json:"last_error_time,omitempty"`
	PerformanceHistory []PerformanceSnapshot  `json:"performance_history"`
	CustomMetrics      map[string]interface{} `json:"custom_metrics"`
}

// PerformanceSnapshot represents a point-in-time performance snapshot
type PerformanceSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`
	ResponseTime float64   `json:"response_time_ms"`
	RequestCount int64     `json:"request_count"`
}

// MCPLogEntry represents a log entry for an MCP server
type MCPLogEntry struct {
	ID        string                 `json:"id"`
	ServerID  string                 `json:"server_id"`
	Level     string                 `json:"level"` // "debug" | "info" | "warn" | "error"
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"` // "server" | "system" | "user"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MCPAlert represents an alert for MCP server issues
type MCPAlert struct {
	ID          string                 `json:"id"`
	ServerID    string                 `json:"server_id"`
	Type        string                 `json:"type"` // "error" | "warning" | "info"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"` // "low" | "medium" | "high" | "critical"
	Status      string                 `json:"status"`   // "active" | "resolved" | "acknowledged"
	CreatedAt   time.Time              `json:"created_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MCPMonitoringService manages monitoring and logging for MCP servers
type MCPMonitoringService struct {
	registryService  *MCPRegistryService
	lifecycleService *MCPLifecycleService
	metrics          map[string]*MCPServerMetrics
	logs             []MCPLogEntry
	alerts           []MCPAlert
	mutex            sync.RWMutex
	logDir           string
	maxLogEntries    int
	maxAlerts        int
}

// NewMCPMonitoringService creates a new monitoring service
func NewMCPMonitoringService(registryService *MCPRegistryService, lifecycleService *MCPLifecycleService) *MCPMonitoringService {
	// Create monitoring directory in app data directory
	logDir := "./mcp_monitoring" // Default fallback
	if appDataDir, err := utils.GetMCPMonitoringDir(); err == nil {
		logDir = appDataDir
	}
	os.MkdirAll(logDir, 0755)

	return &MCPMonitoringService{
		registryService:  registryService,
		lifecycleService: lifecycleService,
		metrics:          make(map[string]*MCPServerMetrics),
		logs:             make([]MCPLogEntry, 0),
		alerts:           make([]MCPAlert, 0),
		logDir:           logDir,
		maxLogEntries:    1000, // Keep last 1000 log entries
		maxAlerts:        100,  // Keep last 100 alerts
	}
}

// Start begins the monitoring service
func (s *MCPMonitoringService) Start() error {
	// Start monitoring loop
	go s.monitoringLoop()
	return nil
}

// monitoringLoop runs the main monitoring loop
func (s *MCPMonitoringService) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.collectMetrics()
		s.checkAlerts()
	}
}

// collectMetrics collects metrics from all running servers
func (s *MCPMonitoringService) collectMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get running servers
	runningServers := s.lifecycleService.GetRunningServers()

	for _, process := range runningServers {
		metrics, exists := s.metrics[process.ServerID]
		if !exists {
			metrics = &MCPServerMetrics{
				ServerID:     process.ServerID,
				Status:       process.Status,
				CreatedAt:    time.Now(),
				LogLevel:     "info",
				HealthStatus: "unknown",
			}
			s.metrics[process.ServerID] = metrics
		}

		// Update basic metrics
		metrics.Status = process.Status
		metrics.RestartCount = process.RestartCount
		metrics.LastHealthCheck = process.HealthCheck
		metrics.UpdatedAt = time.Now()

		// Calculate uptime
		if process.Status == "running" {
			metrics.Uptime = int64(time.Since(process.StartedAt).Seconds())
			metrics.HealthStatus = "healthy"
		} else {
			metrics.Uptime = 0
			metrics.HealthStatus = "unhealthy"
		}

		// Enhanced metrics collection
		s.collectEnhancedMetrics(metrics, process)

		// Update performance history
		s.updatePerformanceHistory(metrics)
	}

	// Mark stopped servers as unhealthy
	for serverID, metrics := range s.metrics {
		found := false
		for _, process := range runningServers {
			if process.ServerID == serverID {
				found = true
				break
			}
		}
		if !found {
			metrics.Status = "stopped"
			metrics.HealthStatus = "unhealthy"
			metrics.Uptime = 0
			metrics.UpdatedAt = time.Now()
		}
	}
}

// collectEnhancedMetrics collects enhanced production metrics
func (s *MCPMonitoringService) collectEnhancedMetrics(metrics *MCPServerMetrics, process *MCPServerProcess) {
	// Simulate enhanced metrics collection (in production, these would come from actual monitoring)

	// Performance metrics
	metrics.ResponseTime = 45.2 + float64(process.RestartCount)*2.1   // Simulate degradation with restarts
	metrics.ThroughputRPS = 150.0 - float64(process.RestartCount)*5.0 // Simulate throughput impact

	// Calculate success rate based on error count and request count
	if metrics.RequestCount > 0 {
		metrics.SuccessRate = float64(metrics.RequestCount-metrics.ErrorCount) / float64(metrics.RequestCount) * 100.0
	} else {
		metrics.SuccessRate = 100.0
	}

	// Resource usage metrics
	metrics.MemoryUsage = int64(50*1024*1024) + int64(process.RestartCount*10*1024*1024) // Base 50MB + 10MB per restart
	metrics.CPUUsage = 15.5 + float64(process.RestartCount)*2.5                          // Base 15.5% + 2.5% per restart
	metrics.DiskUsage = int64(100 * 1024 * 1024)                                         // 100MB disk usage

	// Network metrics
	metrics.NetworkBytesIn = metrics.RequestCount * 1024  // 1KB per request
	metrics.NetworkBytesOut = metrics.RequestCount * 2048 // 2KB per response

	// Connection metrics
	metrics.ActiveConnections = int(metrics.RequestCount%10) + 1 // 1-10 active connections
	metrics.QueueSize = int(metrics.RequestCount % 5)            // 0-4 queue size

	// Increment request count
	metrics.RequestCount += 1

	// Simulate occasional errors
	if metrics.RequestCount%50 == 0 {
		metrics.ErrorCount += 1
		metrics.LastError = "Simulated timeout error"
		metrics.LastErrorTime = time.Now()
	}

	// Initialize custom metrics if not exists
	if metrics.CustomMetrics == nil {
		metrics.CustomMetrics = make(map[string]interface{})
	}

	// Add custom metrics
	metrics.CustomMetrics["server_version"] = "1.0.0"
	metrics.CustomMetrics["protocol_version"] = "2023-11-05"
	metrics.CustomMetrics["feature_flags"] = []string{"streaming", "tools", "resources"}
}

// updatePerformanceHistory updates the performance history for a server
func (s *MCPMonitoringService) updatePerformanceHistory(metrics *MCPServerMetrics) {
	// Create performance snapshot
	snapshot := PerformanceSnapshot{
		Timestamp:    time.Now(),
		CPUUsage:     metrics.CPUUsage,
		MemoryUsage:  metrics.MemoryUsage,
		ResponseTime: metrics.ResponseTime,
		RequestCount: metrics.RequestCount,
	}

	// Add to history
	metrics.PerformanceHistory = append(metrics.PerformanceHistory, snapshot)

	// Keep only last 100 snapshots
	if len(metrics.PerformanceHistory) > 100 {
		metrics.PerformanceHistory = metrics.PerformanceHistory[len(metrics.PerformanceHistory)-100:]
	}
}

// checkAlerts checks for alert conditions
func (s *MCPMonitoringService) checkAlerts() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for serverID, metrics := range s.metrics {
		// Check for high restart count
		if metrics.RestartCount > 3 {
			s.createAlert(serverID, "warning", "High Restart Count",
				fmt.Sprintf("Server has restarted %d times", metrics.RestartCount),
				"medium", map[string]interface{}{
					"restart_count": metrics.RestartCount,
				})
		}

		// Check for unhealthy status
		if metrics.HealthStatus == "unhealthy" && metrics.Status != "stopped" {
			s.createAlert(serverID, "error", "Server Unhealthy",
				"Server is running but health check failed",
				"high", map[string]interface{}{
					"status": metrics.Status,
				})
		}

		// Enhanced production alerts

		// Check for high memory usage
		if metrics.MemoryUsage > 100*1024*1024 { // 100MB
			s.createAlert(serverID, "warning", "High Memory Usage",
				fmt.Sprintf("Memory usage: %d MB", metrics.MemoryUsage/(1024*1024)),
				"medium", map[string]interface{}{
					"memory_usage": metrics.MemoryUsage,
				})
		}

		// Check for high CPU usage
		if metrics.CPUUsage > 80.0 {
			s.createAlert(serverID, "warning", "High CPU Usage",
				fmt.Sprintf("CPU usage: %.1f%%", metrics.CPUUsage),
				"medium", map[string]interface{}{
					"cpu_usage": metrics.CPUUsage,
				})
		}

		// Check for low success rate
		if metrics.SuccessRate < 95.0 && metrics.RequestCount > 10 {
			s.createAlert(serverID, "error", "Low Success Rate",
				fmt.Sprintf("Success rate: %.1f%%", metrics.SuccessRate),
				"high", map[string]interface{}{
					"success_rate": metrics.SuccessRate,
					"error_count":  metrics.ErrorCount,
				})
		}

		// Check for high response time
		if metrics.ResponseTime > 1000.0 { // 1 second
			s.createAlert(serverID, "warning", "High Response Time",
				fmt.Sprintf("Average response time: %.1f ms", metrics.ResponseTime),
				"medium", map[string]interface{}{
					"response_time": metrics.ResponseTime,
				})
		}

		// Check for recent errors
		if !metrics.LastErrorTime.IsZero() && time.Since(metrics.LastErrorTime) < 5*time.Minute {
			s.createAlert(serverID, "error", "Recent Error",
				fmt.Sprintf("Recent error: %s", metrics.LastError),
				"high", map[string]interface{}{
					"last_error": metrics.LastError,
					"error_time": metrics.LastErrorTime,
				})
		}

		// Check for high queue size
		if metrics.QueueSize > 10 {
			s.createAlert(serverID, "warning", "High Queue Size",
				fmt.Sprintf("Queue size: %d", metrics.QueueSize),
				"medium", map[string]interface{}{
					"queue_size": metrics.QueueSize,
				})
		}

		// Check for low throughput
		if metrics.ThroughputRPS < 10.0 && metrics.Status == "running" {
			s.createAlert(serverID, "info", "Low Throughput",
				fmt.Sprintf("Throughput: %.1f RPS", metrics.ThroughputRPS),
				"low", map[string]interface{}{
					"throughput": metrics.ThroughputRPS,
				})
		}
	}
}

// createAlert creates a new alert if it doesn't already exist
func (s *MCPMonitoringService) createAlert(serverID, alertType, title, description, severity string, metadata map[string]interface{}) {
	// Check if similar alert already exists
	for _, alert := range s.alerts {
		if alert.ServerID == serverID && alert.Title == title && alert.Status == "active" {
			return // Alert already exists
		}
	}

	alert := MCPAlert{
		ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		ServerID:    serverID,
		Type:        alertType,
		Title:       title,
		Description: description,
		Severity:    severity,
		Status:      "active",
		CreatedAt:   time.Now(),
		Metadata:    metadata,
	}

	s.alerts = append(s.alerts, alert)

	// Keep only the last maxAlerts alerts
	if len(s.alerts) > s.maxAlerts {
		s.alerts = s.alerts[len(s.alerts)-s.maxAlerts:]
	}

	// Log the alert
	s.addLogEntry(serverID, "warn", fmt.Sprintf("Alert created: %s", title), "system", metadata)
}

// AddLogEntry adds a log entry
func (s *MCPMonitoringService) AddLogEntry(serverID, level, message, source string, metadata map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.addLogEntry(serverID, level, message, source, metadata)
}

// addLogEntry adds a log entry (internal, assumes lock is held)
func (s *MCPMonitoringService) addLogEntry(serverID, level, message, source string, metadata map[string]interface{}) {
	entry := MCPLogEntry{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		ServerID:  serverID,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Source:    source,
		Metadata:  metadata,
	}

	s.logs = append(s.logs, entry)

	// Keep only the last maxLogEntries entries
	if len(s.logs) > s.maxLogEntries {
		s.logs = s.logs[len(s.logs)-s.maxLogEntries:]
	}

	// Write to log file
	s.writeLogToFile(entry)
}

// writeLogToFile writes a log entry to file
func (s *MCPMonitoringService) writeLogToFile(entry MCPLogEntry) {
	logFile := filepath.Join(s.logDir, fmt.Sprintf("%s.log", entry.ServerID))

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	logLine := fmt.Sprintf("[%s] %s [%s] %s\n",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		strings.ToUpper(entry.Level),
		entry.Source,
		entry.Message)

	file.WriteString(logLine)
}

// GetMetrics returns metrics for all servers or a specific server
func (s *MCPMonitoringService) GetMetrics(serverID string) map[string]*MCPServerMetrics {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if serverID != "" {
		if metrics, exists := s.metrics[serverID]; exists {
			return map[string]*MCPServerMetrics{serverID: metrics}
		}
		return make(map[string]*MCPServerMetrics)
	}

	// Return all metrics
	result := make(map[string]*MCPServerMetrics)
	for k, v := range s.metrics {
		result[k] = v
	}
	return result
}

// GetLogs returns log entries with optional filtering
func (s *MCPMonitoringService) GetLogs(serverID, level string, limit int) []MCPLogEntry {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []MCPLogEntry
	for i := len(s.logs) - 1; i >= 0; i-- {
		entry := s.logs[i]

		if serverID != "" && entry.ServerID != serverID {
			continue
		}
		if level != "" && entry.Level != level {
			continue
		}

		filtered = append(filtered, entry)

		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

// GetAlerts returns alerts with optional filtering
func (s *MCPMonitoringService) GetAlerts(serverID, status string) []MCPAlert {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []MCPAlert
	for i := len(s.alerts) - 1; i >= 0; i-- {
		alert := s.alerts[i]

		if serverID != "" && alert.ServerID != serverID {
			continue
		}
		if status != "" && alert.Status != status {
			continue
		}

		filtered = append(filtered, alert)
	}

	return filtered
}

// ResolveAlert resolves an alert
func (s *MCPMonitoringService) ResolveAlert(alertID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == alertID {
			now := time.Now()
			s.alerts[i].Status = "resolved"
			s.alerts[i].ResolvedAt = &now
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// HTTP Handlers

// GetMetricsHandler handles GET /api/v1/mcp/metrics
func (s *MCPMonitoringService) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	metrics := s.GetMetrics(serverID)

	response := map[string]interface{}{
		"metrics": metrics,
		"count":   len(metrics),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetLogsHandler handles GET /api/v1/mcp/logs
func (s *MCPMonitoringService) GetLogsHandler(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	level := r.URL.Query().Get("level")
	limitStr := r.URL.Query().Get("limit")

	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs := s.GetLogs(serverID, level, limit)

	response := map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAlertsHandler handles GET /api/v1/mcp/alerts
func (s *MCPMonitoringService) GetAlertsHandler(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	status := r.URL.Query().Get("status")

	alerts := s.GetAlerts(serverID, status)

	response := map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ResolveAlertHandler handles POST /api/v1/mcp/alerts/{id}/resolve
func (s *MCPMonitoringService) ResolveAlertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	if err := s.ResolveAlert(alertID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"message": "Alert resolved successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHandlers registers the monitoring HTTP handlers
func (s *MCPMonitoringService) RegisterHandlers(router *mux.Router) {
	mcpRouter := router.PathPrefix("/api/v1/mcp").Subrouter()
	mcpRouter.HandleFunc("/metrics", s.GetMetricsHandler).Methods("GET")
	mcpRouter.HandleFunc("/logs", s.GetLogsHandler).Methods("GET")
	mcpRouter.HandleFunc("/alerts", s.GetAlertsHandler).Methods("GET")
	mcpRouter.HandleFunc("/alerts/{id}/resolve", s.ResolveAlertHandler).Methods("POST")
}
