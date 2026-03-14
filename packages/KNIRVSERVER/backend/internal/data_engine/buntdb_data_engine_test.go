package data_engine

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewBuntDBDataEngine(t *testing.T) {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:     dbPath,
		WindowSize:       time.Minute,
		MetricsInterval:  time.Second * 30,
		MetricsRetention: time.Hour * 24,
		AlertsRetention:  time.Hour * 24,
		EventsRetention:  time.Hour * 24,
		BatchSize:        100,
		FlushInterval:    time.Second * 10,
		MaxMemoryUsage:   100 * 1024 * 1024, // 100MB
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}
	if engine == nil {
		t.Fatal("NewBuntDBDataEngine() returned nil")
	}
	if engine.config.DatabasePath != dbPath {
		t.Errorf("DatabasePath = %v, want %v", engine.config.DatabasePath, dbPath)
	}
	if engine.alertChan == nil {
		t.Error("alertChan not initialized")
	}
	if engine.metricsChan == nil {
		t.Error("metricsChan not initialized")
	}
}

func TestBuntDBDataEngine_Start(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:     dbPath,
		WindowSize:       time.Minute,
		MetricsInterval:  time.Second * 30,
		MetricsRetention: time.Hour * 24,
		AlertsRetention:  time.Hour * 24,
		EventsRetention:  time.Hour * 24,
		BatchSize:        100,
		FlushInterval:    time.Second * 10,
		MaxMemoryUsage:   100 * 1024 * 1024,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !engine.IsRunning() {
		t.Error("Engine should be running after Start()")
	}

	if engine.aggregator == nil {
		t.Error("Aggregator not initialized")
	}

	if engine.alerting == nil {
		t.Error("Alerting system not initialized")
	}

	// Test starting already running engine
	err = engine.Start()
	if err == nil {
		t.Error("Start() should fail when engine is already running")
	}
}

func TestBuntDBDataEngine_Stop(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:     dbPath,
		WindowSize:       time.Minute,
		MetricsInterval:  time.Second * 30,
		MetricsRetention: time.Hour * 24,
		AlertsRetention:  time.Hour * 24,
		EventsRetention:  time.Hour * 24,
		BatchSize:        100,
		FlushInterval:    time.Second * 10,
		MaxMemoryUsage:   100 * 1024 * 1024,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	// Start the engine
	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Stop the engine
	err = engine.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if engine.IsRunning() {
		t.Error("Engine should not be running after Stop()")
	}

	// Test stopping already stopped engine
	err = engine.Stop()
	if err != nil {
		t.Fatalf("Stop() should succeed when engine is already stopped: %v", err)
	}
}

func TestBuntDBDataEngine_ProcessEvent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:     dbPath,
		WindowSize:       time.Minute,
		MetricsInterval:  time.Second * 30,
		MetricsRetention: time.Hour * 24,
		AlertsRetention:  time.Hour * 24,
		EventsRetention:  time.Hour * 24,
		BatchSize:        100,
		FlushInterval:    time.Second * 10,
		MaxMemoryUsage:   100 * 1024 * 1024,
	}

	// Ensure intervals are not zero to avoid panic
	if config.MetricsInterval == 0 {
		config.MetricsInterval = time.Second * 30
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = time.Second * 10
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	event := Event{
		Type:      "cpu_usage",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"percentage": 75.5,
		},
	}

	err = engine.ProcessEvent(event)
	if err != nil {
		t.Fatalf("ProcessEvent() failed: %v", err)
	}

	// Test processing event when engine is not running
	engine.Stop()
	err = engine.ProcessEvent(event)
	if err == nil {
		t.Error("ProcessEvent() should fail when engine is not running")
	}
}

func TestBuntDBDataEngine_convertEventToMetric(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	tests := []struct {
		name         string
		event        Event
		expectMetric bool
		expectedType string
		expectedUnit string
	}{
		{
			name: "cpu_usage event",
			event: Event{
				Type:   "cpu_usage",
				Source: "test",
				Data: map[string]interface{}{
					"percentage": 85.0,
				},
			},
			expectMetric: true,
			expectedType: "cpu_usage",
			expectedUnit: "percent",
		},
		{
			name: "memory_usage event",
			event: Event{
				Type:   "memory_usage",
				Source: "test",
				Data: map[string]interface{}{
					"bytes": 1024.0,
				},
			},
			expectMetric: true,
			expectedType: "memory_usage",
			expectedUnit: "bytes",
		},
		{
			name: "network_latency event",
			event: Event{
				Type:   "network_latency",
				Source: "test",
				Data: map[string]interface{}{
					"latency_ms": 150.0,
				},
			},
			expectMetric: true,
			expectedType: "network_latency",
			expectedUnit: "milliseconds",
		},
		{
			name: "generic numeric event",
			event: Event{
				Type:   "custom_metric",
				Source: "test",
				Data: map[string]interface{}{
					"value": 42.0,
				},
			},
			expectMetric: true,
			expectedType: "custom_metric",
			expectedUnit: "value",
		},
		{
			name: "no numeric data",
			event: Event{
				Type:   "text_event",
				Source: "test",
				Data: map[string]interface{}{
					"message": "hello world",
				},
			},
			expectMetric: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metric := engine.convertEventToMetric(test.event)
			if test.expectMetric {
				if metric == nil {
					t.Fatal("Expected metric, got nil")
				}
				if metric.Type != test.expectedType {
					t.Errorf("Metric type = %v, want %v", metric.Type, test.expectedType)
				}
				if metric.Unit != test.expectedUnit {
					t.Errorf("Metric unit = %v, want %v", metric.Unit, test.expectedUnit)
				}
				if metric.Source != test.event.Source {
					t.Errorf("Metric source = %v, want %v", metric.Source, test.event.Source)
				}
			} else {
				if metric != nil {
					t.Errorf("Expected no metric, got %v", metric)
				}
			}
		})
	}
}

func TestBuntDBDataEngine_GetActiveAlerts(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:    dbPath,
		MetricsInterval: time.Second * 30,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	alerts := engine.GetActiveAlerts()
	// GetActiveAlerts can return nil if alerting system is not initialized
	// Just check that it doesn't panic
	_ = alerts
}

func TestBuntDBDataEngine_ResolveAlert(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// Test resolving non-existent alert
	resolved := engine.ResolveAlert("non-existent")
	if resolved {
		t.Error("ResolveAlert() should return false for non-existent alert")
	}
}

func TestBuntDBDataEngine_GetMetrics(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	metrics := engine.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics() should not return nil")
	}
}

func TestBuntDBDataEngine_GetDatabaseStats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	stats, err := engine.GetDatabaseStats()
	if err != nil {
		t.Fatalf("GetDatabaseStats() failed: %v", err)
	}
	if stats == nil {
		t.Error("GetDatabaseStats() should not return nil")
	}
}

func TestBuntDBDataEngine_ProcessMetricEvent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	err = engine.ProcessMetricEvent("test-source", "cpu_usage", 85.0, "percent", map[string]string{"host": "localhost"})
	if err != nil {
		t.Fatalf("ProcessMetricEvent() failed: %v", err)
	}

	// Test with uninitialized database
	engine.db = nil
	err = engine.ProcessMetricEvent("test-source", "cpu_usage", 85.0, "percent", nil)
	if err == nil {
		t.Error("ProcessMetricEvent() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_ProcessAlertEvent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	err = engine.ProcessAlertEvent("High CPU", "CPU usage is high", "warning", "monitor", map[string]string{"host": "localhost"})
	if err != nil {
		t.Fatalf("ProcessAlertEvent() failed: %v", err)
	}

	// Test with uninitialized database
	engine.db = nil
	err = engine.ProcessAlertEvent("High CPU", "CPU usage is high", "warning", "monitor", nil)
	if err == nil {
		t.Error("ProcessAlertEvent() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_Getters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// Test GetBuntDBManager
	if engine.GetBuntDBManager() == nil {
		t.Error("GetBuntDBManager() should not return nil")
	}

	// Test GetAggregator
	if engine.GetAggregator() == nil {
		t.Error("GetAggregator() should not return nil")
	}

	// Test GetAlerting
	if engine.GetAlerting() == nil {
		t.Error("GetAlerting() should not return nil")
	}

	// Test GetDatabase
	if engine.GetDatabase() == nil {
		t.Error("GetDatabase() should not return nil")
	}

	// Test channels
	alertChan := engine.GetAlertChannel()
	if alertChan == nil {
		t.Error("GetAlertChannel() should not return nil")
	}

	metricsChan := engine.GetMetricsChannel()
	if metricsChan == nil {
		t.Error("GetMetricsChannel() should not return nil")
	}
}

func TestBuntDBDataEngine_StoreReports(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	report := &ReportEntry{
		ID:          "test-report",
		Timestamp:   time.Now(),
		Type:        "performance",
		Title:       "Test Report",
		Description: "Test report description",
		Data:        map[string]interface{}{"metric": 100.0},
	}

	// Test StoreUserReport
	err = engine.StoreUserReport(report)
	if err != nil {
		t.Fatalf("StoreUserReport() failed: %v", err)
	}

	// Test StoreSystemReport
	err = engine.StoreSystemReport(report)
	if err != nil {
		t.Fatalf("StoreSystemReport() failed: %v", err)
	}

	// Test with uninitialized database
	engine.db = nil
	err = engine.StoreUserReport(report)
	if err == nil {
		t.Error("StoreUserReport() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_GetReports(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// Test GetUserReports
	reports, err := engine.GetUserReports("performance", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("GetUserReports() failed: %v", err)
	}
	// Reports can be empty slice, just check it doesn't panic
	_ = reports

	// Test GetSystemReports
	reports, err = engine.GetSystemReports("performance", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("GetSystemReports() failed: %v", err)
	}
	// Reports can be empty slice, just check it doesn't panic
	_ = reports

	// Test with uninitialized database
	engine.db = nil
	_, err = engine.GetUserReports("performance", time.Now().Add(-time.Hour), 10)
	if err == nil {
		t.Error("GetUserReports() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_GetMetricsFromDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	metrics, err := engine.GetMetricsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("GetMetricsFromDB() failed: %v", err)
	}
	// Metrics can be empty slice, just check it doesn't panic
	_ = metrics

	// Test with uninitialized database
	engine.db = nil
	_, err = engine.GetMetricsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err == nil {
		t.Error("GetMetricsFromDB() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_GetAlertsFromDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	alerts, err := engine.GetAlertsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("GetAlertsFromDB() failed: %v", err)
	}
	// Alerts can be empty slice, just check it doesn't panic
	_ = alerts

	// Test with uninitialized database
	engine.db = nil
	_, err = engine.GetAlertsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err == nil {
		t.Error("GetAlertsFromDB() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_GetEventsFromDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	events, err := engine.GetEventsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("GetEventsFromDB() failed: %v", err)
	}
	// Events can be empty slice, just check it doesn't panic
	_ = events

	// Test with uninitialized database
	engine.db = nil
	_, err = engine.GetEventsFromDB("", "", time.Now().Add(-time.Hour), 10)
	if err == nil {
		t.Error("GetEventsFromDB() should fail with uninitialized database")
	}
}

func TestBuntDBDataEngine_handleAlert(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	alert := Alert{
		ID:          "test-alert",
		Level:       WarningAlert,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test",
		Timestamp:   time.Now(),
		Data:        map[string]interface{}{"key": "value"},
		Resolved:    false,
	}

	// Test handleAlert - this should not panic
	engine.handleAlert(alert)

	// Test with uninitialized database - should not panic, just log error
	originalDB := engine.db
	engine.db = nil
	// This should not panic, just log an error
	engine.handleAlert(alert)
	engine.db = originalDB
}

func TestBuntDBDataEngine_generateMetricsSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath:    dbPath,
		MetricsInterval: time.Second * 30,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// Test generateMetricsSnapshot
	snapshot := engine.generateMetricsSnapshot()
	if snapshot == nil {
		t.Error("generateMetricsSnapshot() should not return nil")
		return
	}
	if snapshot.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// Test with uninitialized database
	originalDB := engine.db
	engine.db = nil
	snapshot = engine.generateMetricsSnapshot()
	if snapshot == nil {
		t.Error("generateMetricsSnapshot() should return a snapshot even with uninitialized database")
	}
	engine.db = originalDB
}

func TestBuntDBDataEngine_performCleanup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// Test performCleanup - this should not panic
	engine.performCleanup()
}

func TestBuntDBDataEngine_registerDefaultAlertRules(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	engine, err := NewBuntDBDataEngine(config)
	if err != nil {
		t.Fatalf("NewBuntDBDataEngine() failed: %v", err)
	}

	err = engine.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer engine.Stop()

	// registerDefaultAlertRules is called in Start(), but let's test it directly
	// by creating a new alerting system and calling it
	alerting := NewAlertingSystem(100)
	engine.alerting = alerting

	// Test registerDefaultAlertRules - this should not panic
	engine.registerDefaultAlertRules()

	// Verify rules were registered
	if len(engine.alerting.rules) == 0 {
		t.Error("Default alert rules should have been registered")
	}

	// Check for specific rules
	expectedRules := []string{"error-rate", "high-memory", "high-cpu"}
	for _, ruleID := range expectedRules {
		if _, exists := engine.alerting.rules[ruleID]; !exists {
			t.Errorf("Expected rule %s to be registered", ruleID)
		}
	}
}
