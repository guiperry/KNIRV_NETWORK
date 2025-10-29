package modelmanagement

import (
	"backend_server/internal/objects"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
)

// ModelManagementService provides comprehensive model administration
type ModelManagementService struct {
	db      *buntdb.DB
	mu      sync.RWMutex
	running bool

	// Model storage and tracking
	objects     map[string]*objects.Model
	deployments map[string]*objects.ModelDeployment
	templates   map[string]*objects.ModelTemplate

	// Runtime monitoring
	runtimeInstances map[string]*objects.ModelRuntimeInstance
	metrics          map[string][]*objects.ModelMetrics
	logs             map[string][]*objects.ModelLog
	events           []*objects.ModelEvent

	// Configuration
	maxModels             int
	maxInstancesPerModel  int
	defaultResourceLimits *objects.ModelResourceLimits
	monitoringInterval    time.Duration

	// Lifecycle management
	modelServer interface{}
}

type ModelInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// NewModelManagementService creates a new model management service
func NewModelManagementService(db *buntdb.DB) *ModelManagementService {
	service := &ModelManagementService{
		db:                   db,
		objects:              make(map[string]*objects.Model),
		deployments:          make(map[string]*objects.ModelDeployment),
		templates:            make(map[string]*objects.ModelTemplate),
		runtimeInstances:     make(map[string]*objects.ModelRuntimeInstance),
		metrics:              make(map[string][]*objects.ModelMetrics),
		logs:                 make(map[string][]*objects.ModelLog),
		events:               make([]*objects.ModelEvent, 0),
		maxModels:            100,
		maxInstancesPerModel: 10,
		monitoringInterval:   30 * time.Second,
		defaultResourceLimits: &objects.ModelResourceLimits{
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
	service.loadModelData()

	return service
}

// SetModelServerReference sets reference to the model server for integration
func (ams *ModelManagementService) SetModelServerReference(modelServer interface{}) {
	ams.mu.Lock()
	defer ams.mu.Unlock()
	ams.modelServer = modelServer
}

// Start begins the model management service
func (ams *ModelManagementService) Start() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if ams.running {
		return fmt.Errorf("model management service already running")
	}

	ams.running = true

	log.Println("Starting model management service...")

	// Start monitoring goroutines
	go ams.monitoringLoop()
	go ams.metricsCollectionLoop()
	go ams.healthCheckLoop()

	log.Println("Model management service started successfully")
	return nil
}

// Stop stops the model management service
func (ams *ModelManagementService) Stop() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !ams.running {
		return fmt.Errorf("model management service not running")
	}

	ams.running = false

	log.Println("Model management service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (ams *ModelManagementService) IsRunning() bool {
	ams.mu.RLock()
	defer ams.mu.RUnlock()
	return ams.running
}

// GetAllModels returns all objects with optional filtering
func (ams *ModelManagementService) GetAllModels(filter *objects.ModelFilter) ([]*objects.Model, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var result []*objects.Model
	for _, model := range ams.objects {
		if ams.matchesFilter(model, filter) {
			result = append(result, model)
		}
	}

	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(result) {
			return []*objects.Model{}, nil
		}
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

// GetModel returns a specific model by ID
func (ams *ModelManagementService) GetModel(modelID string) (*objects.Model, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	model, exists := ams.objects[modelID]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	return model, nil
}

// CreateModel creates a new model
func (ams *ModelManagementService) CreateModel(model *objects.Model) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !model.IsValid() {
		return fmt.Errorf("invalid model data")
	}

	if _, exists := ams.objects[model.ID]; exists {
		return fmt.Errorf("model already exists: %s", model.ID)
	}

	if len(ams.objects) >= ams.maxModels {
		return fmt.Errorf("maximum number of objects reached: %d", ams.maxModels)
	}

	// Set default values
	model.UploadedAt = time.Now()
	model.LastModified = time.Now()
	model.Status = "uploaded"

	if model.ResourceLimits == nil {
		model.ResourceLimits = ams.defaultResourceLimits
	}

	ams.objects[model.ID] = model

	// Store in database
	ams.storeModel(model)

	// Record event
	ams.recordEvent(&objects.ModelEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID:     model.ID,
		Type:        "uploaded",
		Description: fmt.Sprintf("Model %s uploaded", model.Name),
		Timestamp:   time.Now(),
		UserID:      model.UploadedBy,
	})

	log.Printf("Model created: %s (%s)", model.Name, model.ID)
	return nil
}

// UpdateModel updates an existing model
func (ams *ModelManagementService) UpdateModel(modelID string, updates *objects.Model) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	model, exists := ams.objects[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	// Update allowed fields
	if updates.Name != "" {
		model.Name = updates.Name
	}
	if updates.Description != "" {
		model.Description = updates.Description
	}
	if updates.Version != "" {
		model.Version = updates.Version
	}
	if updates.Configuration != nil {
		model.Configuration = updates.Configuration
	}
	if updates.ResourceLimits != nil {
		model.ResourceLimits = updates.ResourceLimits
	}
	if updates.Tags != nil {
		model.Tags = updates.Tags
	}

	model.LastModified = time.Now()

	// Store updated model
	ams.storeModel(model)

	// Record event
	ams.recordEvent(&objects.ModelEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID:     model.ID,
		Type:        "updated",
		Description: fmt.Sprintf("Model %s updated", model.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Model updated: %s (%s)", model.Name, model.ID)
	return nil
}

// DeleteModel deletes an model
func (ams *ModelManagementService) DeleteModel(modelID string) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	model, exists := ams.objects[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	// Stop model if running
	if model.IsRunning() {
		if err := ams.stopModelInternal(modelID); err != nil {
			return fmt.Errorf("failed to stop model before deletion: %w", err)
		}
	}

	// Remove from memory
	delete(ams.objects, modelID)
	delete(ams.runtimeInstances, modelID)
	delete(ams.metrics, modelID)
	delete(ams.logs, modelID)

	// Remove from database
	ams.db.Update(func(tx *buntdb.Tx) error {
		tx.Delete("model:" + modelID)
		return nil
	})

	// Record event
	ams.recordEvent(&objects.ModelEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID:     model.ID,
		Type:        "deleted",
		Description: fmt.Sprintf("Model %s deleted", model.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Model deleted: %s (%s)", model.Name, model.ID)
	return nil
}

// ExecuteModelAction executes an action on a model
func (ams *ModelManagementService) ExecuteModelAction(modelID string, action *objects.ModelAction) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	_, exists := ams.objects[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	switch action.Action {
	case "deploy":
		return ams.deployModelInternal(modelID, action.Parameters)
	case "start":
		return ams.startModelInternal(modelID, action.Parameters)
	case "stop":
		return ams.stopModelInternal(modelID)
	case "restart":
		if err := ams.stopModelInternal(modelID); err != nil {
			return fmt.Errorf("failed to stop model for restart: %w", err)
		}
		return ams.startModelInternal(modelID, action.Parameters)
	case "scale":
		return ams.scaleModelInternal(modelID, action.Parameters)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// GetModelSummary returns a summary of all objects
func (ams *ModelManagementService) GetModelSummary() *objects.ModelSummary {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	summary := &objects.ModelSummary{
		TotalModels: len(ams.objects),
	}

	for _, model := range ams.objects {
		switch model.Status {
		case "running":
			summary.RunningModels++
		case "stopped":
			summary.StoppedModels++
		case "error":
			summary.ErrorModels++
		case "deployed":
			summary.DeployedModels++
		case "uploaded":
			summary.UploadedModels++
		}
	}

	return summary
}

// Private methods for internal operations
func (ams *ModelManagementService) initializeDatabase() {
	ams.db.Update(func(tx *buntdb.Tx) error {
		tx.CreateIndex("objects", "model:*", buntdb.IndexString)
		tx.CreateIndex("deployments", "deployment:*", buntdb.IndexString)
		tx.CreateIndex("templates", "template:*", buntdb.IndexString)
		tx.CreateIndex("events", "event:*", buntdb.IndexString)
		return nil
	})
}

func (ams *ModelManagementService) loadModelData() {
	// Load objects from database
	ams.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("objects", func(key, value string) bool {
			var model objects.Model
			if json.Unmarshal([]byte(value), &model) == nil {
				ams.objects[model.ID] = &model
			}
			return true
		})
		return nil
	})

	// Load deployments from database
	ams.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("deployments", func(key, value string) bool {
			var deployment objects.ModelDeployment
			if json.Unmarshal([]byte(value), &deployment) == nil {
				ams.deployments[deployment.ID] = &deployment
			}
			return true
		})
		return nil
	})

	// Load templates from database
	ams.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("templates", func(key, value string) bool {
			var template objects.ModelTemplate
			if json.Unmarshal([]byte(value), &template) == nil {
				ams.templates[template.ID] = &template
			}
			return true
		})
		return nil
	})
}

func (ams *ModelManagementService) storeModel(model *objects.Model) {
	if data, err := json.Marshal(model); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("model:"+model.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *ModelManagementService) recordEvent(event *objects.ModelEvent) {
	ams.events = append(ams.events, event)

	// Keep only last 1000 events
	if len(ams.events) > 1000 {
		ams.events = ams.events[len(ams.events)-1000:]
	}

	// Store in database
	if data, err := json.Marshal(event); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("event:"+event.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *ModelManagementService) matchesFilter(model *objects.Model, filter *objects.ModelFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if model.Status == status {
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
		for _, modelType := range filter.Type {
			if model.Type == modelType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Author filter
	if filter.Author != "" && model.Author != filter.Author {
		return false
	}

	// Date filters
	if filter.CreatedAfter != nil && model.UploadedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && model.UploadedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (ams *ModelManagementService) deployModelInternal(modelID string, parameters map[string]interface{}) error {
	model := ams.objects[modelID]
	if !model.CanDeploy() {
		return fmt.Errorf("model cannot be deployed in current state: %s", model.Status)
	}

	// Extract and apply deployment parameters
	replicas, _ := parameters["replicas"].(float64)
	if replicas == 0 {
		replicas = 1 // Default to 1 replica
	}

	resourceLimits := &objects.ModelResourceLimits{}
	if cpuLimit, ok := parameters["cpu_limit"].(float64); ok {
		resourceLimits.MaxCPUPercent = cpuLimit
	}
	if memoryLimit, ok := parameters["memory_limit"].(float64); ok {
		resourceLimits.MaxMemoryMB = int(memoryLimit)
	}
	if executionTime, ok := parameters["execution_time"].(float64); ok {
		resourceLimits.MaxExecutionTime = int(executionTime)
	}

	// Update model status with deployment configuration
	model.Status = "deployed"
	model.DeployedAt = &time.Time{}
	*model.DeployedAt = time.Now()

	// Create deployment with parameters
	deployment := &objects.ModelDeployment{
		ID:             fmt.Sprintf("deployment_%d", time.Now().UnixNano()),
		ModelID:        modelID,
		Name:           fmt.Sprintf("deployment-%s", modelID),
		Description:    fmt.Sprintf("Deployment for model %s", model.Name),
		Status:         "deployed",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Config: &objects.ModelDeploymentConfig{
			ResourceLimits: resourceLimits,
		},
		Replicas: int(replicas),
		Instances: []*objects.ModelRuntimeInstance{},
	}

	// Store deployment
	ams.deployments[modelID] = deployment
	ams.storeModel(model)

	// Record deployment event with parameters
	ams.recordEvent(&objects.ModelEvent{
		ID:      fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID: model.ID,
		Type:    "deployed",
		Description: fmt.Sprintf("Model %s deployed with %d replicas (CPU: %.1f%%, Memory: %dMB)",
			model.Name, int(replicas), resourceLimits.MaxCPUPercent, resourceLimits.MaxMemoryMB),
		Timestamp: time.Now(),
	})

	log.Printf("Model %s deployed successfully with parameters: %v", modelID, parameters)
	return nil
}

func (ams *ModelManagementService) startModelInternal(modelID string, parameters map[string]interface{}) error {
	model := ams.objects[modelID]
	if !model.CanStart() {
		return fmt.Errorf("model cannot be started in current state: %s", model.Status)
	}

	// Create runtime instance
	instance := &objects.ModelRuntimeInstance{
		InstanceID:    fmt.Sprintf("instance_%s_%d", modelID, time.Now().UnixNano()),
		StartedAt:     time.Now(),
		Status:        "running",
		Configuration: parameters,
		Environment:   make(map[string]string),
		HealthStatus:  "healthy",
		RestartCount:  0,
		ResourceUsage: &objects.ModelResourceUsage{
			LastUpdated: time.Now(),
		},
	}

	model.Status = "running"
	model.RuntimeInstance = instance
	model.LastActivity = &time.Time{}
	*model.LastActivity = time.Now()

	ams.runtimeInstances[modelID] = instance
	ams.storeModel(model)

	ams.recordEvent(&objects.ModelEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID:     model.ID,
		InstanceID:  instance.InstanceID,
		Type:        "started",
		Description: fmt.Sprintf("Model %s started", model.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *ModelManagementService) stopModelInternal(modelID string) error {
	model := ams.objects[modelID]
	if !model.CanStop() {
		return fmt.Errorf("model cannot be stopped in current state: %s", model.Status)
	}

	if instance, exists := ams.runtimeInstances[modelID]; exists {
		instance.Status = "stopped"
		delete(ams.runtimeInstances, modelID)
	}

	model.Status = "stopped"
	model.RuntimeInstance = nil

	ams.storeModel(model)

	ams.recordEvent(&objects.ModelEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ModelID:     model.ID,
		Type:        "stopped",
		Description: fmt.Sprintf("Model %s stopped", model.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *ModelManagementService) scaleModelInternal(modelID string, parameters map[string]interface{}) error {
	// Extract scaling parameters
	replicas, ok := parameters["replicas"].(float64)
	if !ok {
		return fmt.Errorf("invalid replicas parameter for model %s", modelID)
	}

	cpuLimit, _ := parameters["cpu_limit"].(float64)
	memoryLimit, _ := parameters["memory_limit"].(float64)

	// Implement actual scaling logic
	log.Printf("Scaling model %s to %d replicas (CPU: %.1f%%, Memory: %dMB)",
		modelID, int(replicas), cpuLimit, int(memoryLimit))

	// Update deployment with scaling information
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment, exists := ams.deployments[modelID]; exists {
		deployment.Replicas = int(replicas)
		if deployment.Config == nil {
			deployment.Config = &objects.ModelDeploymentConfig{}
		}
		if deployment.Config.ResourceLimits == nil {
			deployment.Config.ResourceLimits = &objects.ModelResourceLimits{}
		}
		deployment.Config.ResourceLimits.MaxCPUPercent = cpuLimit
		deployment.Config.ResourceLimits.MaxMemoryMB = int(memoryLimit)
	} else {
		// Create new deployment if it doesn't exist
		ams.deployments[modelID] = &objects.ModelDeployment{
			ID:        fmt.Sprintf("deployment_%d", time.Now().UnixNano()),
			ModelID:   modelID,
			Name:      fmt.Sprintf("deployment-%s", modelID),
			Status:    "deployed",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config: &objects.ModelDeploymentConfig{
				ResourceLimits: &objects.ModelResourceLimits{
					MaxCPUPercent: cpuLimit,
					MaxMemoryMB:   int(memoryLimit),
				},
			},
			Replicas:  int(replicas),
			Instances: []*objects.ModelRuntimeInstance{},
		}
	}

	return nil
}

func (ams *ModelManagementService) monitoringLoop() {
	ticker := time.NewTicker(ams.monitoringInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.updateModelStatuses()
	}
}

func (ams *ModelManagementService) metricsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.collectModelMetrics()
	}
}

func (ams *ModelManagementService) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.performHealthChecks()
	}
}

func (ams *ModelManagementService) updateModelStatuses() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for modelID, model := range ams.objects {
		if instance, exists := ams.runtimeInstances[modelID]; exists {
			// Update instance health status
			instance.LastHealthCheck = &time.Time{}
			*instance.LastHealthCheck = time.Now()

			// Simulate health check (in real implementation, this would check actual process)
			if time.Since(instance.StartedAt) > 5*time.Minute {
				instance.HealthStatus = "healthy"
			}
		}

		// Update model last activity
		if model.IsRunning() {
			model.LastActivity = &time.Time{}
			*model.LastActivity = time.Now()
		}
	}
}

func (ams *ModelManagementService) collectModelMetrics() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for modelID, instance := range ams.runtimeInstances {
		// Simulate metrics collection
		metrics := &objects.ModelMetrics{
			ModelID:           modelID,
			InstanceID:        instance.InstanceID,
			Timestamp:         time.Now(),
			RequestsPerSecond: 10.0 + float64(time.Now().Unix()%50),
			AverageLatency:    50.0 + float64(time.Now().Unix()%100),
			ErrorRate:         0.1 + float64(time.Now().Unix()%5)/100.0,
			Throughput:        1.5 + float64(time.Now().Unix()%10)/10.0,
			ResourceUsage: &objects.ModelResourceUsage{
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
		if ams.metrics[modelID] == nil {
			ams.metrics[modelID] = make([]*objects.ModelMetrics, 0)
		}
		ams.metrics[modelID] = append(ams.metrics[modelID], metrics)

		// Keep only last 100 metrics per model
		if len(ams.metrics[modelID]) > 100 {
			ams.metrics[modelID] = ams.metrics[modelID][len(ams.metrics[modelID])-100:]
		}

		// Update instance resource usage
		instance.ResourceUsage = metrics.ResourceUsage
	}
}

func (ams *ModelManagementService) performHealthChecks() {
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

// GetModelMetrics returns metrics for a specific model
func (ams *ModelManagementService) GetModelMetrics(modelID string, limit int) ([]*objects.ModelMetrics, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	metrics, exists := ams.metrics[modelID]
	if !exists {
		return []*objects.ModelMetrics{}, nil
	}

	if limit > 0 && len(metrics) > limit {
		return metrics[len(metrics)-limit:], nil
	}

	return metrics, nil
}

// GetModelLogs returns logs for a specific model
func (ams *ModelManagementService) GetModelLogs(modelID string, limit int) ([]*objects.ModelLog, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	logs, exists := ams.logs[modelID]
	if !exists {
		return []*objects.ModelLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

// GetModelEvents returns events for a specific model or all events
func (ams *ModelManagementService) GetModelEvents(modelID string, limit int) ([]*objects.ModelEvent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var filteredEvents []*objects.ModelEvent
	for _, event := range ams.events {
		if modelID == "" || event.ModelID == modelID {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > limit {
		return filteredEvents[len(filteredEvents)-limit:], nil
	}

	return filteredEvents, nil
}

// GetModelTemplates returns all model templates
func (ams *ModelManagementService) GetModelTemplates() ([]*objects.ModelTemplate, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var templates []*objects.ModelTemplate
	for _, template := range ams.templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// CreateModelTemplate creates a new model template
func (ams *ModelManagementService) CreateModelTemplate(template *objects.ModelTemplate) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if template.ID == "" || template.Name == "" {
		return fmt.Errorf("template ID and name are required")
	}

	if _, exists := ams.templates[template.ID]; exists {
		return fmt.Errorf("template already exists: %s", template.ID)
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.UsageCount = 0

	ams.templates[template.ID] = template

	// Store in database
	if data, err := json.Marshal(template); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("template:"+template.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Model template created: %s (%s)", template.Name, template.ID)
	return nil
}

// GetModelDeployments returns all deployments for an model
func (ams *ModelManagementService) GetModelDeployments(modelID string) ([]*objects.ModelDeployment, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var deployments []*objects.ModelDeployment
	for _, deployment := range ams.deployments {
		if modelID == "" || deployment.ModelID == modelID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// CreateModelDeployment creates a new deployment for an model
func (ams *ModelManagementService) CreateModelDeployment(deployment *objects.ModelDeployment) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment.ID == "" || deployment.ModelID == "" {
		return fmt.Errorf("deployment ID and model ID are required")
	}

	if _, exists := ams.deployments[deployment.ID]; exists {
		return fmt.Errorf("deployment already exists: %s", deployment.ID)
	}

	// Verify model exists
	if _, exists := ams.objects[deployment.ModelID]; !exists {
		return fmt.Errorf("model not found: %s", deployment.ModelID)
	}

	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()
	deployment.Status = "pending"
	deployment.Instances = make([]*objects.ModelRuntimeInstance, 0)

	ams.deployments[deployment.ID] = deployment

	// Store in database
	if data, err := json.Marshal(deployment); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("deployment:"+deployment.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Model deployment created: %s for model %s", deployment.ID, deployment.ModelID)
	return nil
}
