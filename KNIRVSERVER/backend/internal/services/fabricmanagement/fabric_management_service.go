package fabricmanagement

import (
	"backend_server/internal/database"
	"backend_server/internal/objects"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
)

// FabricManagementService provides comprehensive fabric unit administration
type FabricManagementService struct {
	db      *database.BuntDBManager
	mu      sync.RWMutex
	running bool

	// Fabric storage and tracking
	objects     map[string]*objects.Fabric
	deployments map[string]*objects.FabricDeployment
	templates   map[string]*objects.FabricTemplate

	// Runtime monitoring
	runtimeInstances map[string]*objects.FabricRuntimeInstance
	metrics          map[string][]*objects.FabricMetrics
	logs             map[string][]*objects.FabricLog
	events           []*objects.FabricEvent

	// Configuration
	maxFabrics            int
	maxInstancesPerFabric int
	defaultResourceLimits *objects.FabricResourceLimits
	monitoringInterval    time.Duration

	// Lifecycle management
	fabricServer interface{}
}

type FabricInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// NewFabricManagementService creates a new fabric management service
func NewFabricManagementService(db *database.BuntDBManager) *FabricManagementService {
	service := &FabricManagementService{
		db:                   db,
		objects:              make(map[string]*objects.Fabric),
		deployments:          make(map[string]*objects.FabricDeployment),
		templates:            make(map[string]*objects.FabricTemplate),
		runtimeInstances:     make(map[string]*objects.FabricRuntimeInstance),
		metrics:              make(map[string][]*objects.FabricMetrics),
		logs:                 make(map[string][]*objects.FabricLog),
		events:               make([]*objects.FabricEvent, 0),
		maxFabrics:           100,
		maxInstancesPerFabric: 10,
		monitoringInterval:   30 * time.Second,
		defaultResourceLimits: &objects.FabricResourceLimits{
			MaxMemoryMB:      512,
			MaxCPUPercent:    80.0,
			MaxExecutionTime: 300,
			MaxConcurrency:   10,
			MaxDiskMB:        1024,
			NetworkAccess:    true,
			FileSystemAccess: false,
		},
	}

	// Initialize database indices
	service.initializeDatabase()

	// Load existing data
	service.loadFabricData()

	return service
}

// SetFabricServerReference sets reference to the fabric server for integration
func (ams *FabricManagementService) SetFabricServerReference(fabricServer interface{}) {
	ams.mu.Lock()
	defer ams.mu.Unlock()
	ams.fabricServer = fabricServer
}

// Start begins the fabric management service
func (ams *FabricManagementService) Start() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if ams.running {
		return fmt.Errorf("fabric management service already running")
	}

	ams.running = true

	log.Println("Starting fabric management service...")

	// Start monitoring goroutines
	go ams.monitoringLoop()
	go ams.metricsCollectionLoop()
	go ams.healthCheckLoop()

	log.Println("Fabric management service started successfully")
	return nil
}

// Stop stops the fabric management service
func (ams *FabricManagementService) Stop() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !ams.running {
		return fmt.Errorf("fabric management service not running")
	}

	ams.running = false

	log.Println("Fabric management service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (ams *FabricManagementService) IsRunning() bool {
	ams.mu.RLock()
	defer ams.mu.RUnlock()
	return ams.running
}

// GetAllFabrics returns all fabric units with optional filtering
func (ams *FabricManagementService) GetAllFabrics(filter *objects.FabricFilter) ([]*objects.Fabric, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var result []*objects.Fabric
	for _, fabric := range ams.objects {
		if ams.matchesFilter(fabric, filter) {
			result = append(result, fabric)
		}
	}

	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(result) {
			return []*objects.Fabric{}, nil
		}
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

// GetFabric returns a specific fabric item by ID
func (ams *FabricManagementService) GetFabric(fabricID string) (*objects.Fabric, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	fabric, exists := ams.objects[fabricID]
	if !exists {
		return nil, fmt.Errorf("fabric item not found: %s", fabricID)
	}

	return fabric, nil
}

// CreateFabric creates a new fabric item
func (ams *FabricManagementService) CreateFabric(fabric *objects.Fabric) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !fabric.IsValid() {
		return fmt.Errorf("invalid fabric data")
	}

	if _, exists := ams.objects[fabric.ID]; exists {
		return fmt.Errorf("fabric item already exists: %s", fabric.ID)
	}

	if len(ams.objects) >= ams.maxFabrics {
		return fmt.Errorf("maximum number of fabric units reached: %d", ams.maxFabrics)
	}

	// Set default values
	fabric.UploadedAt = time.Now()
	fabric.LastModified = time.Now()
	fabric.Status = "uploaded"

	if fabric.ResourceLimits == nil {
		fabric.ResourceLimits = ams.defaultResourceLimits
	}

	ams.objects[fabric.ID] = fabric

	// Store in database
	ams.storeFabric(fabric)

	// Record event
	ams.recordEvent(&objects.FabricEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID:    fabric.ID,
		Type:        "uploaded",
		Description: fmt.Sprintf("Fabric unit %s uploaded", fabric.Name),
		Timestamp:   time.Now(),
		UserID:      fabric.UploadedBy,
	})

	log.Printf("Fabric item created: %s (%s)", fabric.Name, fabric.ID)
	return nil
}

// UpdateFabric updates an existing fabric item
func (ams *FabricManagementService) UpdateFabric(fabricID string, updates *objects.Fabric) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	fabric, exists := ams.objects[fabricID]
	if !exists {
		return fmt.Errorf("fabric item not found: %s", fabricID)
	}

	// Update allowed fields
	if updates.Name != "" {
		fabric.Name = updates.Name
	}
	if updates.Description != "" {
		fabric.Description = updates.Description
	}
	if updates.Version != "" {
		fabric.Version = updates.Version
	}
	if updates.Configuration != nil {
		fabric.Configuration = updates.Configuration
	}
	if updates.ResourceLimits != nil {
		fabric.ResourceLimits = updates.ResourceLimits
	}
	if updates.Tags != nil {
		fabric.Tags = updates.Tags
	}

	fabric.LastModified = time.Now()

	// Store updated fabric unit
	ams.storeFabric(fabric)

	// Record event
	ams.recordEvent(&objects.FabricEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID:    fabric.ID,
		Type:        "updated",
		Description: fmt.Sprintf("Fabric unit %s updated", fabric.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Fabric item updated: %s (%s)", fabric.Name, fabric.ID)
	return nil
}

// DeleteFabric deletes a fabric item
func (ams *FabricManagementService) DeleteFabric(fabricID string) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	fabric, exists := ams.objects[fabricID]
	if !exists {
		return fmt.Errorf("fabric item not found: %s", fabricID)
	}

	// Stop fabric unit if running
	if fabric.IsRunning() {
		if err := ams.stopFabricInternal(fabricID); err != nil {
			return fmt.Errorf("failed to stop fabric unit before deletion: %w", err)
		}
	}

	// Remove from memory
	delete(ams.objects, fabricID)
	delete(ams.runtimeInstances, fabricID)
	delete(ams.metrics, fabricID)
	delete(ams.logs, fabricID)

	// Remove from database
	ams.db.Transaction(func(tx *buntdb.Tx) error {
		tx.Delete("fabric:" + fabricID)
		return nil
	})

	// Record event
	ams.recordEvent(&objects.FabricEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID:    fabric.ID,
		Type:        "deleted",
		Description: fmt.Sprintf("Fabric unit %s deleted", fabric.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Fabric item deleted: %s (%s)", fabric.Name, fabric.ID)
	return nil
}

// ExecuteFabricAction executes an action on a fabric item
func (ams *FabricManagementService) ExecuteFabricAction(fabricID string, action *objects.FabricAction) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	_, exists := ams.objects[fabricID]
	if !exists {
		return fmt.Errorf("fabric item not found: %s", fabricID)
	}

	switch action.Action {
	case "deploy":
		return ams.deployFabricInternal(fabricID, action.Parameters)
	case "start":
		return ams.startFabricInternal(fabricID, action.Parameters)
	case "stop":
		return ams.stopFabricInternal(fabricID)
	case "restart":
		if err := ams.stopFabricInternal(fabricID); err != nil {
			return fmt.Errorf("failed to stop fabric unit for restart: %w", err)
		}
		return ams.startFabricInternal(fabricID, action.Parameters)
	case "scale":
		return ams.scaleFabricInternal(fabricID, action.Parameters)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// GetFabricSummary returns a summary of all fabric units
func (ams *FabricManagementService) GetFabricSummary() *objects.FabricSummary {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	summary := &objects.FabricSummary{
		TotalFabrics: len(ams.objects),
	}

	for _, fabric := range ams.objects {
		switch fabric.Status {
		case "running":
			summary.RunningFabrics++
		case "stopped":
			summary.StoppedFabrics++
		case "error":
			summary.ErrorFabrics++
		case "deployed":
			summary.DeployedFabrics++
		case "uploaded":
			summary.UploadedFabrics++
		}
	}

	return summary
}

// Private methods for internal operations
func (ams *FabricManagementService) initializeDatabase() {
	ams.db.Transaction(func(tx *buntdb.Tx) error {
		tx.CreateIndex("fabrics", "fabric:*", buntdb.IndexString)
		tx.CreateIndex("fabric_deployments", "fabric_deployment:*", buntdb.IndexString)
		tx.CreateIndex("fabric_templates", "fabric_template:*", buntdb.IndexString)
		tx.CreateIndex("fabric_events", "fabric_event:*", buntdb.IndexString)
		return nil
	})
}

func (ams *FabricManagementService) loadFabricData() {
	// Load fabrics from database
	ams.db.GetObjectsByPrefix("fabric:", func(key string, value []byte) bool {
		var fabric objects.Fabric
		if json.Unmarshal(value, &fabric) == nil {
			ams.objects[fabric.ID] = &fabric
		}
		return true
	})

	// Load deployments from database
	ams.db.GetObjectsByPrefix("fabric_deployment:", func(key string, value []byte) bool {
		var deployment objects.FabricDeployment
		if json.Unmarshal(value, &deployment) == nil {
			ams.deployments[deployment.ID] = &deployment
		}
		return true
	})

	// Load templates from database
	ams.db.GetObjectsByPrefix("fabric_template:", func(key string, value []byte) bool {
		var template objects.FabricTemplate
		if json.Unmarshal(value, &template) == nil {
			ams.templates[template.ID] = &template
		}
		return true
	})
}

func (ams *FabricManagementService) storeFabric(fabric *objects.Fabric) {
	if data, err := json.Marshal(fabric); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("fabric:"+fabric.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *FabricManagementService) recordEvent(event *objects.FabricEvent) {
	ams.events = append(ams.events, event)

	// Keep only last 1000 events
	if len(ams.events) > 1000 {
		ams.events = ams.events[len(ams.events)-1000:]
	}

	// Store in database
	if data, err := json.Marshal(event); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("fabric_event:"+event.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *FabricManagementService) matchesFilter(fabric *objects.Fabric, filter *objects.FabricFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if fabric.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Type filter
	if len(filter.Type) > 0 {
		found := false
		for _, fabricType := range filter.Type {
			if fabric.Type == fabricType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Author filter
	if filter.Author != "" && fabric.Author != filter.Author {
		return false
	}

	// Date filters
	if filter.CreatedAfter != nil && fabric.UploadedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && fabric.UploadedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (ams *FabricManagementService) deployFabricInternal(fabricID string, parameters map[string]interface{}) error {
	fabric := ams.objects[fabricID]
	if !fabric.CanDeploy() {
		return fmt.Errorf("fabric unit cannot be deployed in current state: %s", fabric.Status)
	}

	// Extract and apply deployment parameters
	replicas, _ := parameters["replicas"].(float64)
	if replicas == 0 {
		replicas = 1 // Default to 1 replica
	}

	resourceLimits := &objects.FabricResourceLimits{}
	if cpuLimit, ok := parameters["cpu_limit"].(float64); ok {
		resourceLimits.MaxCPUPercent = cpuLimit
	}
	if memoryLimit, ok := parameters["memory_limit"].(float64); ok {
		resourceLimits.MaxMemoryMB = int(memoryLimit)
	}
	if executionTime, ok := parameters["execution_time"].(float64); ok {
		resourceLimits.MaxExecutionTime = int(executionTime)
	}

	// Update fabric status with deployment configuration
	fabric.Status = "deployed"
	fabric.DeployedAt = &time.Time{}
	*fabric.DeployedAt = time.Now()

	// Create deployment with parameters
	deployment := &objects.FabricDeployment{
		ID:             fmt.Sprintf("fabric_deployment_%d", time.Now().UnixNano()),
		FabricID:       fabricID,
		Name:           fmt.Sprintf("fabric-deployment-%s", fabricID),
		Description:    fmt.Sprintf("Deployment for fabric unit %s", fabric.Name),
		Status:         "deployed",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Config: &objects.FabricDeploymentConfig{
			ResourceLimits: resourceLimits,
		},
		Replicas: int(replicas),
		Instances: []*objects.FabricRuntimeInstance{},
	}

	// Store deployment
	ams.deployments[fabricID] = deployment
	ams.storeFabric(fabric)

	// Record deployment event with parameters
	ams.recordEvent(&objects.FabricEvent{
		ID:      fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID: fabric.ID,
		Type:    "deployed",
		Description: fmt.Sprintf("Fabric unit %s deployed with %d replicas (CPU: %.1f%%, Memory: %dMB)",
			fabric.Name, int(replicas), resourceLimits.MaxCPUPercent, resourceLimits.MaxMemoryMB),
		Timestamp: time.Now(),
	})

	log.Printf("Fabric item %s deployed successfully with parameters: %v", fabricID, parameters)
	return nil
}

func (ams *FabricManagementService) startFabricInternal(fabricID string, parameters map[string]interface{}) error {
	fabric := ams.objects[fabricID]
	if !fabric.CanStart() {
		return fmt.Errorf("fabric unit cannot be started in current state: %s", fabric.Status)
	}

	// Create runtime instance
	instance := &objects.FabricRuntimeInstance{
		InstanceID:    fmt.Sprintf("instance_%s_%d", fabricID, time.Now().UnixNano()),
		StartedAt:     time.Now(),
		Status:        "running",
		Configuration: parameters,
		Environment:   make(map[string]string),
		HealthStatus:  "healthy",
		RestartCount:  0,
		ResourceUsage: &objects.FabricResourceUsage{
			LastUpdated: time.Now(),
		},
	}

	fabric.Status = "running"
	fabric.RuntimeInstance = instance
	fabric.LastActivity = &time.Time{}
	*fabric.LastActivity = time.Now()

	ams.runtimeInstances[fabricID] = instance
	ams.storeFabric(fabric)

	ams.recordEvent(&objects.FabricEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID:    fabric.ID,
		InstanceID:  instance.InstanceID,
		Type:        "started",
		Description: fmt.Sprintf("Fabric unit %s started", fabric.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *FabricManagementService) stopFabricInternal(fabricID string) error {
	fabric := ams.objects[fabricID]
	if !fabric.CanStop() {
		return fmt.Errorf("fabric unit cannot be stopped in current state: %s", fabric.Status)
	}

	if instance, exists := ams.runtimeInstances[fabricID]; exists {
		instance.Status = "stopped"
		delete(ams.runtimeInstances, fabricID)
	}

	fabric.Status = "stopped"
	fabric.RuntimeInstance = nil

	ams.storeFabric(fabric)

	ams.recordEvent(&objects.FabricEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		FabricID:    fabric.ID,
		Type:        "stopped",
		Description: fmt.Sprintf("Fabric unit %s stopped", fabric.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *FabricManagementService) scaleFabricInternal(fabricID string, parameters map[string]interface{}) error {
	// Extract scaling parameters
	replicas, ok := parameters["replicas"].(float64)
	if !ok {
		return fmt.Errorf("invalid replicas parameter for fabric item %s", fabricID)
	}

	cpuLimit, _ := parameters["cpu_limit"].(float64)
	memoryLimit, _ := parameters["memory_limit"].(float64)

	// Implement actual scaling logic
	log.Printf("Scaling fabric unit %s to %d replicas (CPU: %.1f%%, Memory: %dMB)",
		fabricID, int(replicas), cpuLimit, int(memoryLimit))

	// Update deployment with scaling information
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment, exists := ams.deployments[fabricID]; exists {
		deployment.Replicas = int(replicas)
		if deployment.Config == nil {
			deployment.Config = &objects.FabricDeploymentConfig{}
		}
		if deployment.Config.ResourceLimits == nil {
			deployment.Config.ResourceLimits = &objects.FabricResourceLimits{}
		}
		deployment.Config.ResourceLimits.MaxCPUPercent = cpuLimit
		deployment.Config.ResourceLimits.MaxMemoryMB = int(memoryLimit)
	} else {
		// Create new deployment if it doesn't exist
		ams.deployments[fabricID] = &objects.FabricDeployment{
			ID:        fmt.Sprintf("fabric_deployment_%d", time.Now().UnixNano()),
			FabricID:  fabricID,
			Name:      fmt.Sprintf("fabric-deployment-%s", fabricID),
			Status:    "deployed",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config: &objects.FabricDeploymentConfig{
				ResourceLimits: &objects.FabricResourceLimits{
					MaxCPUPercent: cpuLimit,
					MaxMemoryMB:   int(memoryLimit),
				},
			},
			Replicas:  int(replicas),
			Instances: []*objects.FabricRuntimeInstance{},
		}
	}

	return nil
}

func (ams *FabricManagementService) monitoringLoop() {
	ticker := time.NewTicker(ams.monitoringInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.updateFabricStatuses()
	}
}

func (ams *FabricManagementService) metricsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.collectFabricMetrics()
	}
}

func (ams *FabricManagementService) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.performHealthChecks()
	}
}

func (ams *FabricManagementService) updateFabricStatuses() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for fabricID, fabric := range ams.objects {
		if instance, exists := ams.runtimeInstances[fabricID]; exists {
			// Update instance health status
			instance.LastHealthCheck = &time.Time{}
			*instance.LastHealthCheck = time.Now()

			// Simulate health check (in real implementation, this would check actual process)
			if time.Since(instance.StartedAt) > 5*time.Minute {
				instance.HealthStatus = "healthy"
			}
		}

		// Update fabric unit last activity
		if fabric.IsRunning() {
			fabric.LastActivity = &time.Time{}
			*fabric.LastActivity = time.Now()
		}
	}
}

func (ams *FabricManagementService) collectFabricMetrics() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for fabricID, instance := range ams.runtimeInstances {
		// Simulate metrics collection
		metrics := &objects.FabricMetrics{
			FabricID:          fabricID,
			InstanceID:        instance.InstanceID,
			Timestamp:         time.Now(),
			RequestsPerSecond: 10.0 + float64(time.Now().Unix()%50),
			AverageLatency:    50.0 + float64(time.Now().Unix()%100),
			ErrorRate:         0.1 + float64(time.Now().Unix()%5)/100.0,
			Throughput:        1.5 + float64(time.Now().Unix()%10)/10.0,
			ResourceUsage: &objects.FabricResourceUsage{
				MemoryUsageMB:   100.0 + float64(time.Now().Unix()%200),
				CPUUsagePercent: 20.0 + float64(time.Now().Unix()%60),
				DiskUsageMB:     50.0 + float64(time.Now().Unix()%100),
				ExecutionTime:   time.Since(instance.StartedAt).Milliseconds() / 1000,
				RequestCount:    int64(time.Since(instance.StartedAt).Minutes() * 10),
				ErrorCount:      int64(time.Since(instance.StartedAt).Minutes() * 0.1),
				LastUpdated:     time.Now(),
			},
		}

		// Store metrics
		if ams.metrics[fabricID] == nil {
			ams.metrics[fabricID] = make([]*objects.FabricMetrics, 0)
		}
		ams.metrics[fabricID] = append(ams.metrics[fabricID], metrics)

		// Keep only last 100 metrics per fabric unit
		if len(ams.metrics[fabricID]) > 100 {
			ams.metrics[fabricID] = ams.metrics[fabricID][len(ams.metrics[fabricID])-100:]
		}

		// Update instance resource usage
		instance.ResourceUsage = metrics.ResourceUsage
	}
}

func (ams *FabricManagementService) performHealthChecks() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for _, instance := range ams.runtimeInstances {
		// Simulate health check
		if time.Since(instance.StartedAt) > 10*time.Minute {
			instance.HealthStatus = "healthy"
		} else {
			instance.HealthStatus = "unknown"
		}

		instance.LastHealthCheck = &time.Time{}
		*instance.LastHealthCheck = time.Now()
	}
}

// GetFabricMetrics returns metrics for a specific fabric unit
func (ams *FabricManagementService) GetFabricMetrics(fabricID string, limit int) ([]*objects.FabricMetrics, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	metrics, exists := ams.metrics[fabricID]
	if !exists {
		return []*objects.FabricMetrics{}, nil
	}

	if limit > 0 && len(metrics) > limit {
		return metrics[len(metrics)-limit:], nil
	}

	return metrics, nil
}

// GetFabricLogs returns logs for a specific fabric unit
func (ams *FabricManagementService) GetFabricLogs(fabricID string, limit int) ([]*objects.FabricLog, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	logs, exists := ams.logs[fabricID]
	if !exists {
		return []*objects.FabricLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

// GetFabricEvents returns events for a specific fabric item or all events
func (ams *FabricManagementService) GetFabricEvents(fabricID string, limit int) ([]*objects.FabricEvent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var filteredEvents []*objects.FabricEvent
	for _, event := range ams.events {
		if fabricID == "" || event.FabricID == fabricID {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > limit {
		return filteredEvents[len(filteredEvents)-limit:], nil
	}

	return filteredEvents, nil
}

// GetFabricTemplates returns all fabric templates
func (ams *FabricManagementService) GetFabricTemplates() ([]*objects.FabricTemplate, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var templates []*objects.FabricTemplate
	for _, template := range ams.templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// CreateFabricTemplate creates a new fabric template
func (ams *FabricManagementService) CreateFabricTemplate(template *objects.FabricTemplate) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if template.ID == "" || template.Name == "" {
		return fmt.Errorf("template ID and name are required")
	}

	if _, exists := ams.templates[template.ID]; exists {
		return fmt.Errorf("fabric template already exists: %s", template.ID)
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.UsageCount = 0

	ams.templates[template.ID] = template

	// Store in database
	if data, err := json.Marshal(template); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("fabric_template:"+template.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Fabric template created: %s (%s)", template.Name, template.ID)
	return nil
}

// GetFabricDeployments returns all deployments for an fabric unit
func (ams *FabricManagementService) GetFabricDeployments(fabricID string) ([]*objects.FabricDeployment, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var deployments []*objects.FabricDeployment
	for _, deployment := range ams.deployments {
		if fabricID == "" || deployment.FabricID == fabricID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// CreateFabricDeployment creates a new deployment for a fabric unit
func (ams *FabricManagementService) CreateFabricDeployment(deployment *objects.FabricDeployment) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment.ID == "" || deployment.FabricID == "" {
		return fmt.Errorf("deployment ID and fabric ID are required")
	}

	if _, exists := ams.deployments[deployment.ID]; exists {
		return fmt.Errorf("fabric deployment already exists: %s", deployment.ID)
	}

	// Verify fabric exists
	if _, exists := ams.objects[deployment.FabricID]; !exists {
		return fmt.Errorf("fabric unit not found: %s", deployment.FabricID)
	}

	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()
	deployment.Status = "pending"
	deployment.Instances = make([]*objects.FabricRuntimeInstance, 0)

	ams.deployments[deployment.ID] = deployment

	// Store in database
	if data, err := json.Marshal(deployment); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("fabric_deployment:"+deployment.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Fabric deployment created: %s for fabric item %s", deployment.ID, deployment.FabricID)
	return nil
}
