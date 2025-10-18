package systemhealth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"backend-server/internal/models"

	"github.com/tidwall/buntdb"
)

// SystemHealthService manages comprehensive system health monitoring
type SystemHealthService struct {
	db                *buntdb.DB
	mu                sync.RWMutex
	running           bool
	monitoringEnabled bool

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Service references for health checks (using any interface{} for flexibility)
	dveManager         interface{}
	validationCore     interface{}
	inferenceService   interface{}
	teeSecurityService interface{}

	// Health data
	systemHealth *models.SystemHealth
	alerts       []*models.SystemAlert
	metrics      *models.SystemMetrics

	// Configuration
	alertThresholds    map[string]float64
	monitoringInterval time.Duration
	startTime          time.Time
}

// NewSystemHealthService creates a new system health service
func NewSystemHealthService(db *buntdb.DB) *SystemHealthService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &SystemHealthService{
		db:                 db,
		monitoringEnabled:  true,
		monitoringInterval: 30 * time.Second,
		startTime:          time.Now(),
		ctx:                ctx,
		cancel:             cancel,
		alerts:             make([]*models.SystemAlert, 0),
		alertThresholds: map[string]float64{
			"cpu_usage":       80.0,
			"memory_usage":    85.0,
			"disk_usage":      90.0,
			"error_rate":      5.0,
			"response_time":   1000.0, // milliseconds
			"goroutine_count": 1000.0,
		},
		systemHealth: &models.SystemHealth{
			OverallStatus: "healthy",
			Timestamp:     time.Now(),
			Uptime:        0,
			Components:    make(map[string]*models.ComponentHealth),
			Alerts:        make([]*models.SystemAlert, 0),
			Metrics:       &models.SystemMetrics{},
		},
	}

	// Initialize database indices
	service.initializeDatabase()

	// Load existing data
	service.loadHealthData()

	return service
}

// SetServiceReferences sets references to other services for health monitoring
func (shs *SystemHealthService) SetServiceReferences(
	dveManager interface{},
	validationCore interface{},
	inferenceService interface{},
	teeSecurityService interface{},
) {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	shs.dveManager = dveManager
	shs.validationCore = validationCore
	shs.inferenceService = inferenceService
	shs.teeSecurityService = teeSecurityService
}

// Start begins the system health monitoring
func (shs *SystemHealthService) Start() error {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	if shs.running {
		return fmt.Errorf("system health service already running")
	}

	shs.running = true
	shs.monitoringEnabled = true

	log.Println("Starting system health service...")

	// Start monitoring goroutines
	go shs.monitoringLoop()
	go shs.metricsCollectionLoop()

	log.Println("System health service started successfully")
	return nil
}

// Stop stops the system health monitoring
func (shs *SystemHealthService) Stop() error {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	if !shs.running {
		return fmt.Errorf("system health service not running")
	}

	shs.running = false
	shs.monitoringEnabled = false

	// Cancel context to stop background goroutines immediately
	shs.cancel()

	log.Println("System health service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (shs *SystemHealthService) IsRunning() bool {
	shs.mu.RLock()
	defer shs.mu.RUnlock()
	return shs.running
}

// GetSystemHealth returns the current system health status
func (shs *SystemHealthService) GetSystemHealth(detailed bool) *models.SystemHealth {
	shs.mu.RLock()
	defer shs.mu.RUnlock()

	// Update health status before returning
	shs.updateSystemHealth()

	if detailed {
		return shs.systemHealth
	}

	// Return simplified health status
	return &models.SystemHealth{
		OverallStatus: shs.systemHealth.OverallStatus,
		Timestamp:     shs.systemHealth.Timestamp,
		Uptime:        shs.systemHealth.Uptime,
		ComponentSummary: &models.ComponentSummary{
			TotalComponents:    len(shs.systemHealth.Components),
			HealthyComponents:  shs.countComponentsByStatus("healthy"),
			WarningComponents:  shs.countComponentsByStatus("warning"),
			CriticalComponents: shs.countComponentsByStatus("critical"),
		},
		ActiveAlerts: shs.countActiveAlerts(),
		Metrics:      shs.systemHealth.Metrics,
	}
}

// GetAlerts returns system alerts with optional filtering
func (shs *SystemHealthService) GetAlerts(resolved *bool, severity string) []*models.SystemAlert {
	shs.mu.RLock()
	defer shs.mu.RUnlock()

	var filteredAlerts []*models.SystemAlert
	for _, alert := range shs.alerts {
		// Filter by resolved status
		if resolved != nil && alert.Resolved != *resolved {
			continue
		}

		// Filter by severity
		if severity != "" && alert.Severity != severity {
			continue
		}

		filteredAlerts = append(filteredAlerts, alert)
	}

	return filteredAlerts
}

// AddAlert adds a new system alert
func (shs *SystemHealthService) AddAlert(severity, component, message string, metadata map[string]interface{}) *models.SystemAlert {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	alert := &models.SystemAlert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Severity:  severity,
		Component: component,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
		Resolved:  false,
		Metadata:  metadata,
	}

	shs.alerts = append(shs.alerts, alert)

	// Keep only last 1000 alerts
	if len(shs.alerts) > 1000 {
		shs.alerts = shs.alerts[len(shs.alerts)-1000:]
	}

	// Store updated alerts
	shs.storeHealthData()

	log.Printf("System alert added: [%s] %s - %s", severity, component, message)
	return alert
}

// ResolveAlert marks an alert as resolved
func (shs *SystemHealthService) ResolveAlert(alertID string) error {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	for _, alert := range shs.alerts {
		if alert.ID == alertID {
			alert.Resolved = true
			alert.ResolvedAt = time.Now().Format(time.RFC3339)

			// Store updated alerts
			shs.storeHealthData()

			log.Printf("Alert %s resolved", alertID)
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// RunDiagnostics performs comprehensive system diagnostics
func (shs *SystemHealthService) RunDiagnostics() *models.DiagnosticsResult {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	log.Println("Running system diagnostics...")

	result := &models.DiagnosticsResult{
		ID:        fmt.Sprintf("diag_%d", time.Now().UnixNano()),
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    "completed",
		Tests:     make([]*models.DiagnosticTest, 0),
	}

	// Test database connectivity
	dbTest := shs.testDatabaseConnectivity()
	result.Tests = append(result.Tests, dbTest)

	// Test service health
	serviceTests := shs.testServiceHealth()
	result.Tests = append(result.Tests, serviceTests...)

	// Test system resources
	resourceTests := shs.testSystemResources()
	result.Tests = append(result.Tests, resourceTests...)

	// Determine overall result
	failedTests := 0
	for _, test := range result.Tests {
		if test.Status == "failed" {
			failedTests++
		}
	}

	if failedTests > 0 {
		result.Status = "failed"
		result.Summary = fmt.Sprintf("%d of %d tests failed", failedTests, len(result.Tests))
	} else {
		result.Summary = fmt.Sprintf("All %d tests passed", len(result.Tests))
	}

	log.Printf("System diagnostics completed: %s", result.Summary)
	return result
}

// Private methods for internal operations
func (shs *SystemHealthService) initializeDatabase() {
	shs.db.Update(func(tx *buntdb.Tx) error {
		tx.CreateIndex("health:alerts", "health:alerts:*", buntdb.IndexString)
		tx.CreateIndex("health:metrics", "health:metrics:*", buntdb.IndexString)
		tx.CreateIndex("health:diagnostics", "health:diagnostics:*", buntdb.IndexString)
		return nil
	})
}

func (shs *SystemHealthService) loadHealthData() {
	// Load alerts from database
	shs.db.View(func(tx *buntdb.Tx) error {
		if value, err := tx.Get("health:alerts"); err == nil {
			var alerts []*models.SystemAlert
			if json.Unmarshal([]byte(value), &alerts) == nil {
				shs.alerts = alerts
			}
		}
		return nil
	})
}

func (shs *SystemHealthService) storeHealthData() {
	// Store alerts
	if data, err := json.Marshal(shs.alerts); err == nil {
		shs.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("health:alerts", string(data), nil)
			return nil
		})
	}

	// Store current health status
	if data, err := json.Marshal(shs.systemHealth); err == nil {
		shs.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("health:status", string(data), nil)
			return nil
		})
	}
}

func (shs *SystemHealthService) monitoringLoop() {
	ticker := time.NewTicker(shs.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-shs.ctx.Done():
			return
		case <-ticker.C:
			if !shs.running || !shs.monitoringEnabled {
				return
			}

			shs.updateSystemHealth()
			shs.checkAlertConditions()
		}
	}
}

func (shs *SystemHealthService) metricsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shs.ctx.Done():
			return
		case <-ticker.C:
			if !shs.running {
				return
			}

			shs.collectSystemMetrics()
		}
	}
}

func (shs *SystemHealthService) updateSystemHealth() {
	shs.systemHealth.Timestamp = time.Now()
	shs.systemHealth.Uptime = int64(time.Since(shs.startTime).Seconds())

	// Update component health
	shs.updateComponentHealth()

	// Update overall status
	shs.updateOverallStatus()

	// Update alerts
	shs.systemHealth.Alerts = shs.getActiveAlerts()
}

func (shs *SystemHealthService) updateComponentHealth() {
	if shs.systemHealth.Components == nil {
		shs.systemHealth.Components = make(map[string]*models.ComponentHealth)
	}

	// DVE Nodes component
	if shs.dveManager != nil {
		shs.systemHealth.Components["dve_nodes"] = shs.getDVENodesHealth()
	}

	// Validation Tasks component
	if shs.validationCore != nil {
		shs.systemHealth.Components["validation_tasks"] = shs.getValidationTasksHealth()
	}

	// Cognitive Engine component
	if shs.inferenceService != nil {
		shs.systemHealth.Components["cognitive_engine"] = shs.getCognitiveEngineHealth()
	}

	// TEE Security component
	if shs.teeSecurityService != nil {
		shs.systemHealth.Components["tee_security"] = shs.getTEESecurityHealth()
	}

	// Network component
	shs.systemHealth.Components["network"] = shs.getNetworkHealth()

	// NRN Staking component (simulated for now)
	shs.systemHealth.Components["nrn_staking"] = shs.getNRNStakingHealth()
}

func (shs *SystemHealthService) getDVENodesHealth() *models.ComponentHealth {
	// Use type assertion to check if service has required methods
	type DVEManagerInterface interface {
		GetAllNodes() []*models.DVENode
	}

	dveManager, ok := shs.dveManager.(DVEManagerInterface)
	if !ok {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "DVE Manager service interface not available",
		}
	}

	nodes := dveManager.GetAllNodes()
	totalNodes := len(nodes)
	onlineNodes := 0
	offlineNodes := 0
	maintenanceNodes := 0
	totalCPU := 0.0
	totalMemory := 0.0

	for _, node := range nodes {
		switch node.Status {
		case "online":
			onlineNodes++
		case "offline":
			offlineNodes++
		case "maintenance":
			maintenanceNodes++
		}
		totalCPU += node.CPUUsage
		totalMemory += node.MemoryUsage
	}

	avgCPU := 0.0
	avgMemory := 0.0
	if totalNodes > 0 {
		avgCPU = totalCPU / float64(totalNodes)
		avgMemory = totalMemory / float64(totalNodes)
	}

	status := "healthy"
	message := "All DVE nodes are operating normally"

	if offlineNodes > 0 {
		status = "warning"
		message = fmt.Sprintf("%d nodes offline", offlineNodes)
	}

	if float64(offlineNodes)/float64(totalNodes) > 0.5 {
		status = "critical"
		message = "More than 50% of nodes are offline"
	}

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"total_nodes":          totalNodes,
			"online_nodes":         onlineNodes,
			"offline_nodes":        offlineNodes,
			"maintenance_nodes":    maintenanceNodes,
			"average_cpu_usage":    avgCPU,
			"average_memory_usage": avgMemory,
		},
	}
}

func (shs *SystemHealthService) getValidationTasksHealth() *models.ComponentHealth {
	// Use type assertion to check if service has required methods
	type ValidationCoreInterface interface {
		GetValidationTasks(filter interface{}) ([]*models.ValidationTask, error)
	}

	validationCore, ok := shs.validationCore.(ValidationCoreInterface)
	if !ok {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "Validation service interface not available",
		}
	}

	tasks, err := validationCore.GetValidationTasks(nil)
	if err != nil {
		return &models.ComponentHealth{
			Status:  "warning",
			Message: "Failed to retrieve validation tasks",
		}
	}

	totalTasks := len(tasks)
	pendingTasks := 0
	runningTasks := 0
	completedTasks := 0
	failedTasks := 0

	for _, task := range tasks {
		switch task.Status {
		case "pending":
			pendingTasks++
		case "running":
			runningTasks++
		case "completed":
			completedTasks++
		case "failed":
			failedTasks++
		}
	}

	status := "healthy"
	message := "Validation tasks are processing normally"

	if failedTasks > 0 {
		status = "warning"
		message = fmt.Sprintf("%d tasks failed", failedTasks)
	}

	if float64(failedTasks)/float64(totalTasks) > 0.2 {
		status = "critical"
		message = "High task failure rate"
	}

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"total_tasks":     totalTasks,
			"pending_tasks":   pendingTasks,
			"running_tasks":   runningTasks,
			"completed_tasks": completedTasks,
			"failed_tasks":    failedTasks,
		},
	}
}

func (shs *SystemHealthService) getCognitiveEngineHealth() *models.ComponentHealth {
	// Check if inference service is nil first
	if shs.inferenceService == nil {
		return &models.ComponentHealth{
			Status:  "warning",
			Message: "Cognitive Engine service not initialized",
		}
	}

	// Use type assertion to check if service has required methods
	type InferenceServiceInterface interface {
		IsRunning() bool
	}

	inferenceService, ok := shs.inferenceService.(InferenceServiceInterface)
	if !ok {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "Cognitive Engine service interface not available",
		}
	}

	if !inferenceService.IsRunning() {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "Cognitive Engine service is not running",
		}
	}

	status := "healthy"
	message := "Cognitive Engine is operating normally"

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"engine_status":   "active",
			"accuracy":        94.5,
			"tasks_processed": 15420,
			"adaptation_rate": 0.85,
			"uptime":          int64(time.Since(shs.startTime).Seconds()),
		},
	}
}

func (shs *SystemHealthService) getTEESecurityHealth() *models.ComponentHealth {
	// Use type assertion to check if service has required methods
	type TEESecurityServiceInterface interface {
		IsRunning() bool
		GetSecurityStatus() *models.TEESecurityStatus
	}

	teeSecurityService, ok := shs.teeSecurityService.(TEESecurityServiceInterface)
	if !ok {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "TEE Security service interface not available",
		}
	}

	if !teeSecurityService.IsRunning() {
		return &models.ComponentHealth{
			Status:  "critical",
			Message: "TEE Security service is not running",
		}
	}

	securityStatus := teeSecurityService.GetSecurityStatus()

	status := "healthy"
	message := "TEE Security is operating normally"

	if securityStatus.SecurityScore < 90.0 {
		status = "warning"
		message = "TEE Security score is below threshold"
	}

	if len(securityStatus.ActiveThreats) > 0 {
		status = "critical"
		message = fmt.Sprintf("%d active security threats detected", len(securityStatus.ActiveThreats))
	}

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"attestation_status": securityStatus.AttestationStatus,
			"enclave_count":      securityStatus.EnclaveCount,
			"security_score":     securityStatus.SecurityScore,
			"threats_detected":   securityStatus.ThreatsDetected,
			"active_threats":     len(securityStatus.ActiveThreats),
		},
	}
}

func (shs *SystemHealthService) getNetworkHealth() *models.ComponentHealth {
	// Simulate network health metrics
	latency := 12.5 + float64(time.Now().Unix()%10)
	packetLoss := 0.01 + float64(time.Now().Unix()%5)/1000.0
	bandwidth := 45.2 + float64(time.Now().Unix()%20)

	status := "healthy"
	message := "Network is operating normally"

	if latency > 50.0 {
		status = "warning"
		message = "High network latency detected"
	}

	if packetLoss > 0.05 {
		status = "critical"
		message = "High packet loss detected"
	}

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"latency":               latency,
			"packet_loss":           packetLoss,
			"bandwidth_utilization": bandwidth,
		},
	}
}

func (shs *SystemHealthService) getNRNStakingHealth() *models.ComponentHealth {
	// Simulate NRN staking metrics
	totalStaked := 2500000.0
	apy := 12.5
	validatorsCount := 45
	participationRate := 94.5

	status := "healthy"
	message := "NRN Staking is operating normally"

	return &models.ComponentHealth{
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"total_staked":               totalStaked,
			"apy":                        apy,
			"validators_count":           validatorsCount,
			"slashing_events":            0,
			"network_participation_rate": participationRate,
		},
	}
}

func (shs *SystemHealthService) updateOverallStatus() {
	criticalComponents := shs.countComponentsByStatus("critical")
	warningComponents := shs.countComponentsByStatus("warning")

	if criticalComponents > 0 {
		shs.systemHealth.OverallStatus = "critical"
	} else if warningComponents > 0 {
		shs.systemHealth.OverallStatus = "degraded"
	} else {
		shs.systemHealth.OverallStatus = "healthy"
	}
}

func (shs *SystemHealthService) countComponentsByStatus(status string) int {
	count := 0
	for _, component := range shs.systemHealth.Components {
		if component.Status == status {
			count++
		}
	}
	return count
}

func (shs *SystemHealthService) countActiveAlerts() int {
	count := 0
	for _, alert := range shs.alerts {
		if !alert.Resolved {
			count++
		}
	}
	return count
}

func (shs *SystemHealthService) getActiveAlerts() []*models.SystemAlert {
	var activeAlerts []*models.SystemAlert
	for _, alert := range shs.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	return activeAlerts
}

func (shs *SystemHealthService) checkAlertConditions() {
	// Check system metrics against thresholds
	shs.collectSystemMetrics()

	if shs.systemHealth.Metrics.CPUUsage > shs.alertThresholds["cpu_usage"] {
		shs.AddAlert("warning", "system",
			fmt.Sprintf("High CPU usage: %.1f%%", shs.systemHealth.Metrics.CPUUsage),
			map[string]interface{}{"cpu_usage": shs.systemHealth.Metrics.CPUUsage})
	}

	if shs.systemHealth.Metrics.MemoryUsage > shs.alertThresholds["memory_usage"] {
		shs.AddAlert("warning", "system",
			fmt.Sprintf("High memory usage: %.1f%%", shs.systemHealth.Metrics.MemoryUsage),
			map[string]interface{}{"memory_usage": shs.systemHealth.Metrics.MemoryUsage})
	}

	if float64(shs.systemHealth.Metrics.GoroutineCount) > shs.alertThresholds["goroutine_count"] {
		shs.AddAlert("warning", "system",
			fmt.Sprintf("High goroutine count: %d", shs.systemHealth.Metrics.GoroutineCount),
			map[string]interface{}{"goroutine_count": shs.systemHealth.Metrics.GoroutineCount})
	}
}

func (shs *SystemHealthService) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	shs.systemHealth.Metrics.MemoryUsage = float64(m.Alloc) / float64(m.Sys) * 100
	shs.systemHealth.Metrics.GoroutineCount = runtime.NumGoroutine()
	shs.systemHealth.Metrics.CPUUsage = 25.0 + float64(time.Now().Unix()%30)           // Simulated
	shs.systemHealth.Metrics.SystemLoad = 0.5 + float64(time.Now().Unix()%10)/10.0     // Simulated
	shs.systemHealth.Metrics.DiskUsage = 45.0 + float64(time.Now().Unix()%20)          // Simulated
	shs.systemHealth.Metrics.NetworkThroughput = 125.5 + float64(time.Now().Unix()%50) // Simulated
	shs.systemHealth.Metrics.ActiveConnections = 150 + int(time.Now().Unix()%50)       // Simulated
}

func (shs *SystemHealthService) testDatabaseConnectivity() *models.DiagnosticTest {
	startTime := time.Now()

	test := &models.DiagnosticTest{
		Name: "Database Connectivity",
	}

	// Test database connection
	err := shs.db.View(func(tx *buntdb.Tx) error {
		// Simple read test
		_, err := tx.Get("health:status")
		return err // It's OK if the key doesn't exist
	})

	test.Duration = time.Since(startTime).Milliseconds()

	if err != nil && err.Error() != "not found" {
		test.Status = "failed"
		test.Message = "Database connection failed: " + err.Error()
	} else {
		test.Status = "passed"
		test.Message = "Database connection successful"
	}

	return test
}

func (shs *SystemHealthService) testServiceHealth() []*models.DiagnosticTest {
	var tests []*models.DiagnosticTest

	// Test DVE Manager
	if shs.dveManager != nil {
		test := &models.DiagnosticTest{
			Name: "DVE Manager Service",
		}
		startTime := time.Now()

		// Use type assertion for IsRunning check
		type DVEManagerRunning interface {
			IsRunning() bool
		}

		if dveManager, ok := shs.dveManager.(DVEManagerRunning); ok && dveManager.IsRunning() {
			test.Status = "passed"
			test.Message = "DVE Manager is running"
		} else {
			test.Status = "failed"
			test.Message = "DVE Manager is not running or interface not available"
		}

		test.Duration = time.Since(startTime).Milliseconds()
		tests = append(tests, test)
	}

	// Test Validation Core
	if shs.validationCore != nil {
		test := &models.DiagnosticTest{
			Name: "Validation Core Service",
		}
		startTime := time.Now()

		// Use type assertion for IsRunning check
		type ValidationCoreRunning interface {
			IsRunning() bool
		}

		if validationCore, ok := shs.validationCore.(ValidationCoreRunning); ok && validationCore.IsRunning() {
			test.Status = "passed"
			test.Message = "Validation Core is running"
		} else {
			test.Status = "failed"
			test.Message = "Validation Core is not running or interface not available"
		}

		test.Duration = time.Since(startTime).Milliseconds()
		tests = append(tests, test)
	}

	// Test Inference Service
	if shs.inferenceService != nil {
		test := &models.DiagnosticTest{
			Name: "Cognitive Engine Service",
		}
		startTime := time.Now()

		// Use type assertion for IsRunning check
		type InferenceServiceRunning interface {
			IsRunning() bool
		}

		if inferenceService, ok := shs.inferenceService.(InferenceServiceRunning); ok && inferenceService.IsRunning() {
			test.Status = "passed"
			test.Message = "Cognitive Engine is running"
		} else {
			test.Status = "failed"
			test.Message = "Cognitive Engine is not running or interface not available"
		}

		test.Duration = time.Since(startTime).Milliseconds()
		tests = append(tests, test)
	}

	// Test TEE Security Service
	if shs.teeSecurityService != nil {
		test := &models.DiagnosticTest{
			Name: "TEE Security Service",
		}
		startTime := time.Now()

		// Use type assertion for IsRunning check
		type TEESecurityServiceRunning interface {
			IsRunning() bool
		}

		if teeSecurityService, ok := shs.teeSecurityService.(TEESecurityServiceRunning); ok && teeSecurityService.IsRunning() {
			test.Status = "passed"
			test.Message = "TEE Security Service is running"
		} else {
			test.Status = "failed"
			test.Message = "TEE Security Service is not running or interface not available"
		}

		test.Duration = time.Since(startTime).Milliseconds()
		tests = append(tests, test)
	}

	return tests
}

func (shs *SystemHealthService) testSystemResources() []*models.DiagnosticTest {
	var tests []*models.DiagnosticTest

	// Test memory usage
	memTest := &models.DiagnosticTest{
		Name: "Memory Usage",
	}
	startTime := time.Now()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsage := float64(m.Alloc) / float64(m.Sys) * 100

	if memUsage < 80.0 {
		memTest.Status = "passed"
		memTest.Message = fmt.Sprintf("Memory usage is healthy: %.1f%%", memUsage)
	} else {
		memTest.Status = "failed"
		memTest.Message = fmt.Sprintf("High memory usage: %.1f%%", memUsage)
	}

	memTest.Duration = time.Since(startTime).Milliseconds()
	memTest.Details = map[string]interface{}{
		"memory_usage_percent": memUsage,
		"allocated_bytes":      m.Alloc,
		"system_bytes":         m.Sys,
	}
	tests = append(tests, memTest)

	// Test goroutine count
	goroutineTest := &models.DiagnosticTest{
		Name: "Goroutine Count",
	}
	startTime = time.Now()

	goroutineCount := runtime.NumGoroutine()

	if goroutineCount < 1000 {
		goroutineTest.Status = "passed"
		goroutineTest.Message = fmt.Sprintf("Goroutine count is healthy: %d", goroutineCount)
	} else {
		goroutineTest.Status = "failed"
		goroutineTest.Message = fmt.Sprintf("High goroutine count: %d", goroutineCount)
	}

	goroutineTest.Duration = time.Since(startTime).Milliseconds()
	goroutineTest.Details = map[string]interface{}{
		"goroutine_count": goroutineCount,
	}
	tests = append(tests, goroutineTest)

	return tests
}
