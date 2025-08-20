package dataengine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BuntDBDataEngine is the main data engine using BuntDB
type BuntDBDataEngine struct {
	db         *BuntDBManager
	aggregator *WindowedAggregator
	alerting   *AlertingSystem
	websocket  *WebSocketServer
	restAPI    *RESTAPIServer

	config    BuntDBDataEngineConfig
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
	mutex     sync.RWMutex

	// Channels for communication with UI
	alertChan   chan Alert
	metricsChan chan *MetricsSnapshot
}

// BuntDBDataEngineConfig contains configuration for the BuntDB data engine
type BuntDBDataEngineConfig struct {
	DatabasePath    string
	EnableWebSocket bool
	EnableRESTAPI   bool
	WebSocketPort   int
	RESTAPIPort     int
	WindowSize      time.Duration
	MetricsInterval time.Duration

	// Data retention settings
	MetricsRetention time.Duration
	AlertsRetention  time.Duration
	EventsRetention  time.Duration

	// Performance settings
	BatchSize      int
	FlushInterval  time.Duration
	MaxMemoryUsage int64
}

// NewBuntDBDataEngine creates a new BuntDB-based data engine
func NewBuntDBDataEngine(config BuntDBDataEngineConfig) (*BuntDBDataEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize BuntDB manager
	db, err := NewBuntDBManager(config.DatabasePath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create BuntDB manager: %w", err)
	}

	return &BuntDBDataEngine{
		db:          db,
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		alertChan:   make(chan Alert, 100),
		metricsChan: make(chan *MetricsSnapshot, 10),
	}, nil
}

// Start starts the BuntDB data engine
func (d *BuntDBDataEngine) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.isRunning {
		return fmt.Errorf("data engine is already running")
	}

	// Create windowed aggregator
	d.aggregator = NewWindowedAggregator(SlidingWindow, d.config.WindowSize)

	// Create alerting system
	d.alerting = NewAlertingSystem(1000)

	// Register alert handler
	d.alerting.RegisterHandler(d.handleAlert)

	// Register default alert rules
	d.registerDefaultAlertRules()

	// Create WebSocket server if enabled
	if d.config.EnableWebSocket {
		d.websocket = NewWebSocketServer(WebSocketConfig{
			Port:            d.config.WebSocketPort,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     true,
		}, d)

		// Start WebSocket server
		err := d.websocket.Start()
		if err != nil {
			return fmt.Errorf("failed to start WebSocket server: %w", err)
		}
	}

	// Create REST API server if enabled
	if d.config.EnableRESTAPI {
		d.restAPI = NewRESTAPIServer(RESTAPIConfig{
			Port:           d.config.RESTAPIPort,
			EnableCORS:     true,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1MB
		}, d)

		// Start REST API server
		err := d.restAPI.Start()
		if err != nil {
			return fmt.Errorf("failed to start REST API server: %w", err)
		}
	}

	// Start background tasks
	go d.reportMetrics()
	go d.cleanupOldData()
	go d.processMetricsBatch()

	d.isRunning = true
	return nil
}

// Stop stops the BuntDB data engine
func (d *BuntDBDataEngine) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.isRunning {
		return nil
	}

	// Cancel context
	d.cancel()

	// Stop components
	if d.alerting != nil {
		d.alerting.Close()
	}

	// Stop WebSocket server
	if d.websocket != nil {
		err := d.websocket.Stop()
		if err != nil {
			fmt.Printf("Failed to stop WebSocket server: %s\n", err.Error())
		}
	}

	// Stop REST API server
	if d.restAPI != nil {
		err := d.restAPI.Stop()
		if err != nil {
			fmt.Printf("Failed to stop REST API server: %s\n", err.Error())
		}
	}

	// Close database
	if d.db != nil {
		d.db.Close()
	}

	d.isRunning = false
	return nil
}

// ProcessEvent processes an event through the data engine
func (d *BuntDBDataEngine) ProcessEvent(event Event) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.isRunning {
		return fmt.Errorf("data engine is not running")
	}

	// Process through windowed aggregator
	err := d.aggregator.ProcessEvent(event)
	if err != nil {
		return fmt.Errorf("failed to process event through aggregator: %w", err)
	}

	// Process through alerting system
	d.alerting.ProcessEvent(event)

	// Store event in BuntDB
	eventEntry := &EventEntry{
		ID:        fmt.Sprintf("%d-%s", time.Now().UnixNano(), event.Type),
		Timestamp: time.Now(),
		Type:      event.Type,
		Source:    event.Source,
		Data:      event.Data,
		Tags:      event.Tags,
	}

	if err := d.db.StoreEvent(eventEntry); err != nil {
		return fmt.Errorf("failed to store event in BuntDB: %w", err)
	}

	// Convert event to metric if applicable
	if metric := d.convertEventToMetric(event); metric != nil {
		if err := d.db.StoreMetric(metric); err != nil {
			return fmt.Errorf("failed to store metric in BuntDB: %w", err)
		}
	}

	// Broadcast to WebSocket clients if enabled
	if d.config.EnableWebSocket && d.websocket != nil && d.websocket.IsRunning() {
		d.websocket.Broadcast(map[string]interface{}{
			"type":  "event",
			"event": event,
		})
	}

	return nil
}

// convertEventToMetric converts an event to a metric entry
func (d *BuntDBDataEngine) convertEventToMetric(event Event) *MetricEntry {
	// Extract numeric values from event data
	var value float64
	var unit string

	switch event.Type {
	case "cpu_usage":
		if v, ok := event.Data["percentage"].(float64); ok {
			value = v
			unit = "percent"
		}
	case "memory_usage":
		if v, ok := event.Data["bytes"].(float64); ok {
			value = v
			unit = "bytes"
		}
	case "network_latency":
		if v, ok := event.Data["latency_ms"].(float64); ok {
			value = v
			unit = "milliseconds"
		}
	case "transaction_count":
		if v, ok := event.Data["count"].(float64); ok {
			value = v
			unit = "count"
		}
	default:
		// Try to find any numeric value
		for key, val := range event.Data {
			if v, ok := val.(float64); ok {
				value = v
				unit = key
				break
			}
		}
	}

	if value == 0 && unit == "" {
		return nil // No numeric data found
	}

	return &MetricEntry{
		ID:        fmt.Sprintf("%d-%s", time.Now().UnixNano(), event.Type),
		Timestamp: time.Now(),
		Source:    event.Source,
		Type:      event.Type,
		Value:     value,
		Unit:      unit,
		Tags:      event.Tags,
		Metadata:  event.Data,
	}
}

// handleAlert handles an alert
func (d *BuntDBDataEngine) handleAlert(alert Alert) {
	// Store alert in BuntDB
	alertEntry := &AlertEntry{
		ID:          alert.ID,
		Timestamp:   alert.Timestamp,
		Title:       alert.Title,
		Description: alert.Description,
		Severity:    string(alert.Level),
		Source:      alert.Source,
		Status:      "active",
		Tags:        alert.Tags,
		Metadata:    alert.Data,
	}

	if err := d.db.StoreAlert(alertEntry); err != nil {
		fmt.Printf("Failed to store alert in BuntDB: %v\n", err)
	}

	// Send to alert channel
	select {
	case d.alertChan <- alert:
		// Alert sent successfully
	default:
		// Channel is full, log and continue
		fmt.Printf("Alert channel is full, dropping alert: %s\n", alert.Title)
	}

	// Broadcast to WebSocket clients if enabled
	if d.config.EnableWebSocket && d.websocket != nil && d.websocket.IsRunning() {
		d.websocket.Broadcast(map[string]interface{}{
			"type":  "alert",
			"alert": alert,
		})
	}
}

// reportMetrics periodically reports metrics
func (d *BuntDBDataEngine) reportMetrics() {
	ticker := time.NewTicker(d.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// Generate metrics snapshot
			metrics := d.generateMetricsSnapshot()

			// Send to metrics channel
			select {
			case d.metricsChan <- metrics:
				// Metrics sent successfully
			default:
				// Channel is full, skip this update
			}

			// Broadcast to WebSocket clients if enabled
			if d.config.EnableWebSocket && d.websocket != nil && d.websocket.IsRunning() {
				d.websocket.Broadcast(map[string]interface{}{
					"type":    "metrics",
					"metrics": metrics,
				})
			}
		}
	}
}

// generateMetricsSnapshot generates a current metrics snapshot
func (d *BuntDBDataEngine) generateMetricsSnapshot() *MetricsSnapshot {
	// Get recent metrics from BuntDB
	since := time.Now().Add(-d.config.MetricsInterval)
	metrics, err := d.db.GetMetrics("", "", since, 1000)
	if err != nil {
		fmt.Printf("Failed to get metrics from BuntDB: %v\n", err)
		return &MetricsSnapshot{
			Timestamp: time.Now(),
		}
	}

	// Aggregate metrics by type
	aggregated := make(map[string]float64)
	counts := make(map[string]int)

	for _, metric := range metrics {
		aggregated[metric.Type] += metric.Value
		counts[metric.Type]++
	}

	// Calculate averages
	averages := make(map[string]float64)
	for metricType, total := range aggregated {
		if count := counts[metricType]; count > 0 {
			averages[metricType] = total / float64(count)
		}
	}

	return &MetricsSnapshot{
		Timestamp:         time.Now(),
		EventsProcessed:   int64(len(metrics)),
		AverageLatency:    averages["latency"],
		ThroughputPerSec:  float64(len(metrics)) / d.config.MetricsInterval.Seconds(),
		ErrorRate:         averages["error_rate"],
		ActiveConnections: int64(averages["connections"]),
		MemoryUsage:       averages["memory_usage"],
		CPUUsage:          averages["cpu_usage"],
	}
}

// cleanupOldData periodically cleans up old data
func (d *BuntDBDataEngine) cleanupOldData() {
	ticker := time.NewTicker(1 * time.Hour) // Run cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.performCleanup()
		}
	}
}

// performCleanup removes old data based on retention policies
func (d *BuntDBDataEngine) performCleanup() {
	// This would implement cleanup logic for old metrics, alerts, and events
	// For now, just log that cleanup is running
	fmt.Printf("Running data cleanup at %v\n", time.Now())

	// In a real implementation, this would:
	// 1. Remove metrics older than MetricsRetention
	// 2. Remove resolved alerts older than AlertsRetention
	// 3. Remove events older than EventsRetention
	// 4. Compact the database if needed
}

// processMetricsBatch processes metrics in batches for better performance
func (d *BuntDBDataEngine) processMetricsBatch() {
	ticker := time.NewTicker(d.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// This would implement batch processing logic
			// For now, just a placeholder
		}
	}
}

// registerDefaultAlertRules registers default alert rules
func (d *BuntDBDataEngine) registerDefaultAlertRules() {
	// Register error rate alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "error-rate",
		Name:        "High Error Rate",
		Description: "Error rate exceeds threshold",
		EventType:   ErrorEvent,
		Condition:   ErrorRateCondition(10), // 10 errors per minute
		Level:       ErrorAlert,
		Cooldown:    5 * time.Minute,
	})

	// Register memory usage alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "high-memory",
		Name:        "High Memory Usage",
		Description: "Memory usage exceeds threshold",
		EventType:   "memory_usage",
		Condition:   ThresholdCondition("percentage", 80, true), // >80%
		Level:       WarningAlert,
		Cooldown:    5 * time.Minute,
	})

	// Register CPU usage alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "high-cpu",
		Name:        "High CPU Usage",
		Description: "CPU usage exceeds threshold",
		EventType:   "cpu_usage",
		Condition:   ThresholdCondition("percentage", 90, true), // >90%
		Level:       WarningAlert,
		Cooldown:    5 * time.Minute,
	})
}

// GetAlertChannel returns the alert channel
func (d *BuntDBDataEngine) GetAlertChannel() <-chan Alert {
	return d.alertChan
}

// GetMetricsChannel returns the metrics channel
func (d *BuntDBDataEngine) GetMetricsChannel() <-chan *MetricsSnapshot {
	return d.metricsChan
}

// IsRunning returns whether the data engine is running
func (d *BuntDBDataEngine) IsRunning() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.isRunning
}

// GetActiveAlerts returns all active alerts
func (d *BuntDBDataEngine) GetActiveAlerts() []Alert {
	if d.alerting == nil {
		return nil
	}
	return d.alerting.GetActiveAlerts()
}

// ResolveAlert resolves an alert
func (d *BuntDBDataEngine) ResolveAlert(alertID string) bool {
	if d.alerting == nil {
		return false
	}

	resolved := d.alerting.ResolveAlert(alertID)

	// Update alert status in BuntDB
	if resolved {
		// This would update the alert status to "resolved" in BuntDB
		// For now, just broadcast the resolution
		if d.config.EnableWebSocket && d.websocket != nil && d.websocket.IsRunning() {
			d.websocket.Broadcast(map[string]interface{}{
				"type":     "alert_resolved",
				"alert_id": alertID,
			})
		}
	}

	return resolved
}

// GetMetrics returns metrics from BuntDB
func (d *BuntDBDataEngine) GetMetrics() *MetricsSnapshot {
	return d.generateMetricsSnapshot()
}

// GetWebSocketClientCount returns the number of connected WebSocket clients
func (d *BuntDBDataEngine) GetWebSocketClientCount() int {
	if d.websocket == nil {
		return 0
	}
	return d.websocket.GetClientCount()
}

// IsWebSocketRunning returns whether the WebSocket server is running
func (d *BuntDBDataEngine) IsWebSocketRunning() bool {
	if d.websocket == nil {
		return false
	}
	return d.websocket.IsRunning()
}

// IsRESTAPIRunning returns whether the REST API server is running
func (d *BuntDBDataEngine) IsRESTAPIRunning() bool {
	if d.restAPI == nil {
		return false
	}
	return d.restAPI.IsRunning()
}

// GetDatabaseStats returns database statistics
func (d *BuntDBDataEngine) GetDatabaseStats() (map[string]interface{}, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetDatabaseStats()
}

// StoreUserReport stores a user-generated report
func (d *BuntDBDataEngine) StoreUserReport(report *ReportEntry) error {
	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return d.db.StoreReport(report, true)
}

// StoreSystemReport stores a system-generated report
func (d *BuntDBDataEngine) StoreSystemReport(report *ReportEntry) error {
	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return d.db.StoreReport(report, false)
}

// GetUserReports retrieves user reports
func (d *BuntDBDataEngine) GetUserReports(reportType string, since time.Time, limit int) ([]*ReportEntry, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetReports(reportType, true, since, limit)
}

// GetSystemReports retrieves system reports
func (d *BuntDBDataEngine) GetSystemReports(reportType string, since time.Time, limit int) ([]*ReportEntry, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetReports(reportType, false, since, limit)
}

// GetMetricsFromDB retrieves metrics from the database
func (d *BuntDBDataEngine) GetMetricsFromDB(source, metricType string, since time.Time, limit int) ([]*MetricEntry, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetMetrics(source, metricType, since, limit)
}

// GetAlertsFromDB retrieves alerts from the database
func (d *BuntDBDataEngine) GetAlertsFromDB(severity, status string, since time.Time, limit int) ([]*AlertEntry, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetAlerts(severity, status, since, limit)
}

// GetEventsFromDB retrieves events from the database
func (d *BuntDBDataEngine) GetEventsFromDB(eventType, source string, since time.Time, limit int) ([]*EventEntry, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return d.db.GetEvents(eventType, source, since, limit)
}

// GetAggregator returns the windowed aggregator
func (d *BuntDBDataEngine) GetAggregator() *WindowedAggregator {
	return d.aggregator
}

// GetAlerting returns the alerting system
func (d *BuntDBDataEngine) GetAlerting() *AlertingSystem {
	return d.alerting
}

// GetDatabase returns the BuntDB manager
func (d *BuntDBDataEngine) GetDatabase() *BuntDBManager {
	return d.db
}

// ProcessMetricEvent processes a metric event specifically
func (d *BuntDBDataEngine) ProcessMetricEvent(source, metricType string, value float64, unit string, tags map[string]string) error {
	metric := &MetricEntry{
		ID:        fmt.Sprintf("%d-%s-%s", time.Now().UnixNano(), source, metricType),
		Timestamp: time.Now(),
		Source:    source,
		Type:      metricType,
		Value:     value,
		Unit:      unit,
		Tags:      tags,
		Metadata:  make(map[string]interface{}),
	}

	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return d.db.StoreMetric(metric)
}

// ProcessAlertEvent processes an alert event specifically
func (d *BuntDBDataEngine) ProcessAlertEvent(title, description, severity, source string, tags map[string]string) error {
	alert := &AlertEntry{
		ID:          fmt.Sprintf("%d-%s", time.Now().UnixNano(), source),
		Timestamp:   time.Now(),
		Title:       title,
		Description: description,
		Severity:    severity,
		Source:      source,
		Status:      "active",
		Tags:        tags,
		Metadata:    make(map[string]interface{}),
	}

	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return d.db.StoreAlert(alert)
}
