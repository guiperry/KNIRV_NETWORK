package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// SystemMetrics represents overall system metrics
type SystemMetrics struct {
	Timestamp      time.Time              `json:"timestamp"`
	CPUUsage       float64                `json:"cpu_usage"`
	MemoryUsage    int64                  `json:"memory_usage"`
	MemoryTotal    int64                  `json:"memory_total"`
	MemoryPercent  float64                `json:"memory_percent"`
	GoroutineCount int                    `json:"goroutine_count"`
	HeapSize       int64                  `json:"heap_size"`
	HeapInUse      int64                  `json:"heap_in_use"`
	GCCount        uint32                 `json:"gc_count"`
	LastGCTime     time.Time              `json:"last_gc_time"`
	Uptime         int64                  `json:"uptime"`
	RequestCount   int64                  `json:"request_count"`
	ErrorCount     int64                  `json:"error_count"`
	ActiveSessions int                    `json:"active_sessions"`
	CustomMetrics  map[string]interface{} `json:"custom_metrics"`
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SystemMonitoringService manages comprehensive system monitoring
type SystemMonitoringService struct {
	metrics         *SystemMetrics
	alerts          []PerformanceAlert
	mutex           sync.RWMutex
	startTime       time.Time
	requestCounter  int64
	errorCounter    int64
	sessionCounter  int
	stopChan        chan bool
	running         bool
	alertThresholds map[string]float64
}

// NewSystemMonitoringService creates a new system monitoring service
func NewSystemMonitoringService() *SystemMonitoringService {
	return &SystemMonitoringService{
		metrics: &SystemMetrics{
			CustomMetrics: make(map[string]interface{}),
		},
		alerts:    make([]PerformanceAlert, 0),
		startTime: time.Now(),
		stopChan:  make(chan bool),
		alertThresholds: map[string]float64{
			"cpu_usage":       80.0, // 80% CPU usage
			"memory_percent":  85.0, // 85% memory usage
			"goroutine_count": 1000, // 1000 goroutines
			"error_rate":      5.0,  // 5% error rate
		},
	}
}

// Start begins the monitoring service
func (s *SystemMonitoringService) Start() error {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return fmt.Errorf("monitoring service is already running")
	}
	s.running = true
	s.mutex.Unlock()

	// Start monitoring loop
	go s.monitoringLoop()
	log.Printf("System monitoring service started")
	return nil
}

// Stop stops the monitoring service
func (s *SystemMonitoringService) Stop() error {
	s.mutex.Lock()
	if !s.running {
		s.mutex.Unlock()
		return fmt.Errorf("monitoring service is not running")
	}
	s.running = false
	s.mutex.Unlock()

	s.stopChan <- true
	log.Printf("System monitoring service stopped")
	return nil
}

// monitoringLoop runs the main monitoring loop
func (s *SystemMonitoringService) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.collectMetrics()
			s.checkAlerts()
		}
	}
}

// collectMetrics collects system metrics
func (s *SystemMonitoringService) collectMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	now := time.Now()
	s.metrics.Timestamp = now
	s.metrics.GoroutineCount = runtime.NumGoroutine()
	s.metrics.HeapSize = int64(memStats.HeapSys)
	s.metrics.HeapInUse = int64(memStats.HeapInuse)
	s.metrics.GCCount = memStats.NumGC
	s.metrics.MemoryUsage = int64(memStats.Alloc)
	s.metrics.MemoryTotal = int64(memStats.Sys)
	s.metrics.MemoryPercent = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	s.metrics.Uptime = int64(now.Sub(s.startTime).Seconds())
	s.metrics.RequestCount = s.requestCounter
	s.metrics.ErrorCount = s.errorCounter
	s.metrics.ActiveSessions = s.sessionCounter

	if memStats.NumGC > 0 {
		s.metrics.LastGCTime = time.Unix(0, int64(memStats.LastGC))
	}

	// Simulate CPU usage (in production, use actual CPU monitoring)
	s.metrics.CPUUsage = float64(s.metrics.GoroutineCount) * 0.1
	if s.metrics.CPUUsage > 100 {
		s.metrics.CPUUsage = 100
	}

	// Update custom metrics
	s.metrics.CustomMetrics["gc_pause_ns"] = memStats.PauseTotalNs
	s.metrics.CustomMetrics["heap_objects"] = memStats.HeapObjects
	s.metrics.CustomMetrics["stack_inuse"] = memStats.StackInuse
}

// checkAlerts checks for alert conditions
func (s *SystemMonitoringService) checkAlerts() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check CPU usage
	if s.metrics.CPUUsage > s.alertThresholds["cpu_usage"] {
		s.createAlert("cpu_high", "warning", "High CPU Usage",
			fmt.Sprintf("CPU usage is %.1f%%", s.metrics.CPUUsage),
			map[string]interface{}{
				"cpu_usage": s.metrics.CPUUsage,
				"threshold": s.alertThresholds["cpu_usage"],
			})
	}

	// Check memory usage
	if s.metrics.MemoryPercent > s.alertThresholds["memory_percent"] {
		s.createAlert("memory_high", "warning", "High Memory Usage",
			fmt.Sprintf("Memory usage is %.1f%%", s.metrics.MemoryPercent),
			map[string]interface{}{
				"memory_percent": s.metrics.MemoryPercent,
				"memory_usage":   s.metrics.MemoryUsage,
				"threshold":      s.alertThresholds["memory_percent"],
			})
	}

	// Check goroutine count
	if float64(s.metrics.GoroutineCount) > s.alertThresholds["goroutine_count"] {
		s.createAlert("goroutine_high", "warning", "High Goroutine Count",
			fmt.Sprintf("Goroutine count is %d", s.metrics.GoroutineCount),
			map[string]interface{}{
				"goroutine_count": s.metrics.GoroutineCount,
				"threshold":       s.alertThresholds["goroutine_count"],
			})
	}

	// Check error rate
	if s.metrics.RequestCount > 100 {
		errorRate := float64(s.metrics.ErrorCount) / float64(s.metrics.RequestCount) * 100
		if errorRate > s.alertThresholds["error_rate"] {
			s.createAlert("error_rate_high", "error", "High Error Rate",
				fmt.Sprintf("Error rate is %.1f%%", errorRate),
				map[string]interface{}{
					"error_rate":    errorRate,
					"error_count":   s.metrics.ErrorCount,
					"request_count": s.metrics.RequestCount,
					"threshold":     s.alertThresholds["error_rate"],
				})
		}
	}
}

// createAlert creates a new alert
func (s *SystemMonitoringService) createAlert(alertType, severity, title, description string, metadata map[string]interface{}) {
	// Check if similar alert already exists and is not resolved
	for _, alert := range s.alerts {
		if alert.Type == alertType && !alert.Resolved {
			return // Don't create duplicate alerts
		}
	}

	alert := PerformanceAlert{
		ID:          fmt.Sprintf("%s-%d", alertType, time.Now().UnixNano()),
		Type:        alertType,
		Severity:    severity,
		Title:       title,
		Description: description,
		Timestamp:   time.Now(),
		Resolved:    false,
		Metadata:    metadata,
	}

	s.alerts = append(s.alerts, alert)

	// Keep only last 100 alerts
	if len(s.alerts) > 100 {
		s.alerts = s.alerts[len(s.alerts)-100:]
	}

	log.Printf("System Alert [%s]: %s - %s", severity, title, description)
}

// IncrementRequestCount increments the request counter
func (s *SystemMonitoringService) IncrementRequestCount() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.requestCounter++
}

// IncrementErrorCount increments the error counter
func (s *SystemMonitoringService) IncrementErrorCount() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.errorCounter++
}

// UpdateSessionCount updates the active session count
func (s *SystemMonitoringService) UpdateSessionCount(count int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessionCounter = count
}

// GetMetrics returns current system metrics
func (s *SystemMonitoringService) GetMetrics() *SystemMetrics {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := *s.metrics
	metricsCopy.CustomMetrics = make(map[string]interface{})
	for k, v := range s.metrics.CustomMetrics {
		metricsCopy.CustomMetrics[k] = v
	}

	return &metricsCopy
}

// GetAlerts returns current alerts
func (s *SystemMonitoringService) GetAlerts() []PerformanceAlert {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	alertsCopy := make([]PerformanceAlert, len(s.alerts))
	copy(alertsCopy, s.alerts)
	return alertsCopy
}

// HTTP Handlers

// SystemMetricsHandler handles GET /api/v1/system/metrics
func (s *SystemMonitoringService) SystemMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := s.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"metrics": metrics,
	}); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// SystemAlertsHandler handles GET /api/v1/system/alerts
func (s *SystemMonitoringService) SystemAlertsHandler(w http.ResponseWriter, r *http.Request) {
	alerts := s.GetAlerts()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"alerts": alerts,
		"count":  len(alerts),
	}); err != nil {
		http.Error(w, "Failed to encode alerts", http.StatusInternalServerError)
		return
	}
}

// SystemHealthHandler handles GET /api/v1/system/health
func (s *SystemMonitoringService) SystemHealthHandler(w http.ResponseWriter, r *http.Request) {
	metrics := s.GetMetrics()
	alerts := s.GetAlerts()

	// Determine overall health status
	healthStatus := "healthy"
	criticalAlerts := 0
	warningAlerts := 0

	for _, alert := range alerts {
		if !alert.Resolved {
			if alert.Severity == "error" || alert.Severity == "critical" {
				criticalAlerts++
				healthStatus = "unhealthy"
			} else if alert.Severity == "warning" {
				warningAlerts++
				if healthStatus == "healthy" {
					healthStatus = "degraded"
				}
			}
		}
	}

	response := map[string]interface{}{
		"status":          "success",
		"health_status":   healthStatus,
		"timestamp":       time.Now().UTC(),
		"uptime_seconds":  metrics.Uptime,
		"version":         "1.0.0",
		"critical_alerts": criticalAlerts,
		"warning_alerts":  warningAlerts,
		"system_metrics": map[string]interface{}{
			"cpu_usage":       metrics.CPUUsage,
			"memory_percent":  metrics.MemoryPercent,
			"goroutine_count": metrics.GoroutineCount,
			"request_count":   metrics.RequestCount,
			"error_count":     metrics.ErrorCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
		return
	}
}

// SystemConfigHandler handles GET/POST /api/v1/system/config
func (s *SystemMonitoringService) SystemConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.mutex.RLock()
		config := map[string]interface{}{
			"alert_thresholds":  s.alertThresholds,
			"monitoring_active": s.running,
		}
		s.mutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": config,
		})

	case "POST":
		var request struct {
			AlertThresholds map[string]float64 `json:"alert_thresholds"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		s.mutex.Lock()
		if request.AlertThresholds != nil {
			for key, value := range request.AlertThresholds {
				s.alertThresholds[key] = value
			}
		}
		s.mutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Configuration updated successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// RegisterRoutes registers monitoring routes with the router
func (s *SystemMonitoringService) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/system/metrics", s.SystemMetricsHandler).Methods("GET")
	router.HandleFunc("/system/alerts", s.SystemAlertsHandler).Methods("GET")
	router.HandleFunc("/system/health", s.SystemHealthHandler).Methods("GET")
	router.HandleFunc("/system/config", s.SystemConfigHandler).Methods("GET", "POST")
}
