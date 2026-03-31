package pluginmanagement

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

// PluginManagementService provides comprehensive plugin unit administration
type PluginManagementService struct {
	db      *database.BuntDBManager
	mu      sync.RWMutex
	running bool

	// Plugin storage and tracking
	objects     map[string]*objects.Plugin
	deployments map[string]*objects.PluginDeployment
	templates   map[string]*objects.PluginTemplate

	// Runtime monitoring
	runtimeInstances map[string]*objects.PluginRuntimeInstance
	metrics          map[string][]*objects.PluginMetrics
	logs             map[string][]*objects.PluginLog
	events           []*objects.PluginEvent

	// Configuration
	maxPlugins            int
	maxInstancesPerPlugin int
	defaultResourceLimits *objects.PluginResourceLimits
	monitoringInterval    time.Duration

	// Lifecycle management
	pluginServer interface{}
}

type PluginInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// NewPluginManagementService creates a new plugin management service
func NewPluginManagementService(db *database.BuntDBManager) *PluginManagementService {
	service := &PluginManagementService{
		db:                    db,
		objects:               make(map[string]*objects.Plugin),
		deployments:           make(map[string]*objects.PluginDeployment),
		templates:             make(map[string]*objects.PluginTemplate),
		runtimeInstances:      make(map[string]*objects.PluginRuntimeInstance),
		metrics:               make(map[string][]*objects.PluginMetrics),
		logs:                  make(map[string][]*objects.PluginLog),
		events:                make([]*objects.PluginEvent, 0),
		maxPlugins:            100,
		maxInstancesPerPlugin: 10,
		monitoringInterval:    30 * time.Second,
		defaultResourceLimits: &objects.PluginResourceLimits{
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
	service.loadPluginData()

	return service
}

// SetPluginServerReference sets reference to the plugin server for integration
func (ams *PluginManagementService) SetPluginServerReference(pluginServer interface{}) {
	ams.mu.Lock()
	defer ams.mu.Unlock()
	ams.pluginServer = pluginServer
}

// Start begins the plugin management service
func (ams *PluginManagementService) Start() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if ams.running {
		return fmt.Errorf("plugin management service already running")
	}

	ams.running = true

	log.Println("Starting plugin management service...")

	// Start monitoring goroutines
	go ams.monitoringLoop()
	go ams.metricsCollectionLoop()
	go ams.healthCheckLoop()

	log.Println("Plugin management service started successfully")
	return nil
}

// Stop stops the plugin management service
func (ams *PluginManagementService) Stop() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !ams.running {
		return fmt.Errorf("plugin management service not running")
	}

	ams.running = false

	log.Println("Plugin management service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (ams *PluginManagementService) IsRunning() bool {
	ams.mu.RLock()
	defer ams.mu.RUnlock()
	return ams.running
}

// GetAllPlugins returns all plugin units with optional filtering
func (ams *PluginManagementService) GetAllPlugins(filter *objects.PluginFilter) ([]*objects.Plugin, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var result []*objects.Plugin
	for _, plugin := range ams.objects {
		if ams.matchesFilter(plugin, filter) {
			result = append(result, plugin)
		}
	}

	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(result) {
			return []*objects.Plugin{}, nil
		}
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

// GetPlugin returns a specific plugin item by ID
func (ams *PluginManagementService) GetPlugin(pluginID string) (*objects.Plugin, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	plugin, exists := ams.objects[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin item not found: %s", pluginID)
	}

	return plugin, nil
}

// CreatePlugin creates a new plugin item
func (ams *PluginManagementService) CreatePlugin(plugin *objects.Plugin) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !plugin.IsValid() {
		return fmt.Errorf("invalid plugin data")
	}

	if _, exists := ams.objects[plugin.ID]; exists {
		return fmt.Errorf("plugin item already exists: %s", plugin.ID)
	}

	if len(ams.objects) >= ams.maxPlugins {
		return fmt.Errorf("maximum number of plugin units reached: %d", ams.maxPlugins)
	}

	// Set default values
	plugin.UploadedAt = time.Now()
	plugin.LastModified = time.Now()
	plugin.Status = "uploaded"

	if plugin.ResourceLimits == nil {
		plugin.ResourceLimits = ams.defaultResourceLimits
	}

	ams.objects[plugin.ID] = plugin

	// Store in database
	ams.storePlugin(plugin)

	// Record event
	ams.recordEvent(&objects.PluginEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID:    plugin.ID,
		Type:        "uploaded",
		Description: fmt.Sprintf("Plugin unit %s uploaded", plugin.Name),
		Timestamp:   time.Now(),
		UserID:      plugin.UploadedBy,
	})

	log.Printf("Plugin item created: %s (%s)", plugin.Name, plugin.ID)
	return nil
}

// UpdatePlugin updates an existing plugin item
func (ams *PluginManagementService) UpdatePlugin(pluginID string, updates *objects.Plugin) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	plugin, exists := ams.objects[pluginID]
	if !exists {
		return fmt.Errorf("plugin item not found: %s", pluginID)
	}

	// Update allowed fields
	if updates.Name != "" {
		plugin.Name = updates.Name
	}
	if updates.Description != "" {
		plugin.Description = updates.Description
	}
	if updates.Version != "" {
		plugin.Version = updates.Version
	}
	if updates.Configuration != nil {
		plugin.Configuration = updates.Configuration
	}
	if updates.ResourceLimits != nil {
		plugin.ResourceLimits = updates.ResourceLimits
	}
	if updates.Tags != nil {
		plugin.Tags = updates.Tags
	}

	plugin.LastModified = time.Now()

	// Store updated plugin unit
	ams.storePlugin(plugin)

	// Record event
	ams.recordEvent(&objects.PluginEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID:    plugin.ID,
		Type:        "updated",
		Description: fmt.Sprintf("Plugin unit %s updated", plugin.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Plugin item updated: %s (%s)", plugin.Name, plugin.ID)
	return nil
}

// DeletePlugin deletes a plugin item
func (ams *PluginManagementService) DeletePlugin(pluginID string) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	plugin, exists := ams.objects[pluginID]
	if !exists {
		return fmt.Errorf("plugin item not found: %s", pluginID)
	}

	// Stop plugin unit if running
	if plugin.IsRunning() {
		if err := ams.stopPluginInternal(pluginID); err != nil {
			return fmt.Errorf("failed to stop plugin unit before deletion: %w", err)
		}
	}

	// Remove from memory
	delete(ams.objects, pluginID)
	delete(ams.runtimeInstances, pluginID)
	delete(ams.metrics, pluginID)
	delete(ams.logs, pluginID)

	// Remove from database
	ams.db.Transaction(func(tx *buntdb.Tx) error {
		tx.Delete("plugin:" + pluginID)
		return nil
	})

	// Record event
	ams.recordEvent(&objects.PluginEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID:    plugin.ID,
		Type:        "deleted",
		Description: fmt.Sprintf("Plugin unit %s deleted", plugin.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Plugin item deleted: %s (%s)", plugin.Name, plugin.ID)
	return nil
}

// ExecutePluginAction executes an action on a plugin item
func (ams *PluginManagementService) ExecutePluginAction(pluginID string, action *objects.PluginAction) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	_, exists := ams.objects[pluginID]
	if !exists {
		return fmt.Errorf("plugin item not found: %s", pluginID)
	}

	switch action.Action {
	case "deploy":
		return ams.deployPluginInternal(pluginID, action.Parameters)
	case "start":
		return ams.startPluginInternal(pluginID, action.Parameters)
	case "stop":
		return ams.stopPluginInternal(pluginID)
	case "restart":
		if err := ams.stopPluginInternal(pluginID); err != nil {
			return fmt.Errorf("failed to stop plugin unit for restart: %w", err)
		}
		return ams.startPluginInternal(pluginID, action.Parameters)
	case "scale":
		return ams.scalePluginInternal(pluginID, action.Parameters)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// GetPluginSummary returns a summary of all plugin units
func (ams *PluginManagementService) GetPluginSummary() *objects.PluginSummary {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	summary := &objects.PluginSummary{
		TotalPlugins: len(ams.objects),
	}

	for _, plugin := range ams.objects {
		switch plugin.Status {
		case "running":
			summary.RunningPlugins++
		case "stopped":
			summary.StoppedPlugins++
		case "error":
			summary.ErrorPlugins++
		case "deployed":
			summary.DeployedPlugins++
		case "uploaded":
			summary.UploadedPlugins++
		}
	}

	return summary
}

// Private methods for internal operations
func (ams *PluginManagementService) initializeDatabase() {
	ams.db.Transaction(func(tx *buntdb.Tx) error {
		tx.CreateIndex("plugins", "plugin:*", buntdb.IndexString)
		tx.CreateIndex("plugin_deployments", "plugin_deployment:*", buntdb.IndexString)
		tx.CreateIndex("plugin_templates", "plugin_template:*", buntdb.IndexString)
		tx.CreateIndex("plugin_events", "plugin_event:*", buntdb.IndexString)
		return nil
	})
}

func (ams *PluginManagementService) loadPluginData() {
	// Load plugins from database
	ams.db.GetObjectsByPrefix("plugin:", func(key string, value []byte) bool {
		var plugin objects.Plugin
		if json.Unmarshal(value, &plugin) == nil {
			ams.objects[plugin.ID] = &plugin
		}
		return true
	})

	// Load deployments from database
	ams.db.GetObjectsByPrefix("plugin_deployment:", func(key string, value []byte) bool {
		var deployment objects.PluginDeployment
		if json.Unmarshal(value, &deployment) == nil {
			ams.deployments[deployment.ID] = &deployment
		}
		return true
	})

	// Load templates from database
	ams.db.GetObjectsByPrefix("plugin_template:", func(key string, value []byte) bool {
		var template objects.PluginTemplate
		if json.Unmarshal(value, &template) == nil {
			ams.templates[template.ID] = &template
		}
		return true
	})
}

func (ams *PluginManagementService) storePlugin(plugin *objects.Plugin) {
	if data, err := json.Marshal(plugin); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("plugin:"+plugin.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *PluginManagementService) recordEvent(event *objects.PluginEvent) {
	ams.events = append(ams.events, event)

	// Keep only last 1000 events
	if len(ams.events) > 1000 {
		ams.events = ams.events[len(ams.events)-1000:]
	}

	// Store in database
	if data, err := json.Marshal(event); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("plugin_event:"+event.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *PluginManagementService) matchesFilter(plugin *objects.Plugin, filter *objects.PluginFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if plugin.Status == status {
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
		for _, pluginType := range filter.Type {
			if plugin.Type == pluginType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Author filter
	if filter.Author != "" && plugin.Author != filter.Author {
		return false
	}

	// Date filters
	if filter.CreatedAfter != nil && plugin.UploadedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && plugin.UploadedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (ams *PluginManagementService) deployPluginInternal(pluginID string, parameters map[string]interface{}) error {
	plugin := ams.objects[pluginID]
	if !plugin.CanDeploy() {
		return fmt.Errorf("plugin unit cannot be deployed in current state: %s", plugin.Status)
	}

	// Extract and apply deployment parameters
	replicas, _ := parameters["replicas"].(float64)
	if replicas == 0 {
		replicas = 1 // Default to 1 replica
	}

	resourceLimits := &objects.PluginResourceLimits{}
	if cpuLimit, ok := parameters["cpu_limit"].(float64); ok {
		resourceLimits.MaxCPUPercent = cpuLimit
	}
	if memoryLimit, ok := parameters["memory_limit"].(float64); ok {
		resourceLimits.MaxMemoryMB = int(memoryLimit)
	}
	if executionTime, ok := parameters["execution_time"].(float64); ok {
		resourceLimits.MaxExecutionTime = int(executionTime)
	}

	// Update plugin status with deployment configuration
	plugin.Status = "deployed"
	plugin.DeployedAt = &time.Time{}
	*plugin.DeployedAt = time.Now()

	// Create deployment with parameters
	deployment := &objects.PluginDeployment{
		ID:          fmt.Sprintf("plugin_deployment_%d", time.Now().UnixNano()),
		PluginID:    pluginID,
		Name:        fmt.Sprintf("plugin-deployment-%s", pluginID),
		Description: fmt.Sprintf("Deployment for plugin unit %s", plugin.Name),
		Status:      "deployed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Config: &objects.PluginDeploymentConfig{
			ResourceLimits: resourceLimits,
		},
		Replicas:  int(replicas),
		Instances: []*objects.PluginRuntimeInstance{},
	}

	// Store deployment
	ams.deployments[pluginID] = deployment
	ams.storePlugin(plugin)

	// Record deployment event with parameters
	ams.recordEvent(&objects.PluginEvent{
		ID:       fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID: plugin.ID,
		Type:     "deployed",
		Description: fmt.Sprintf("Plugin unit %s deployed with %d replicas (CPU: %.1f%%, Memory: %dMB)",
			plugin.Name, int(replicas), resourceLimits.MaxCPUPercent, resourceLimits.MaxMemoryMB),
		Timestamp: time.Now(),
	})

	log.Printf("Plugin item %s deployed successfully with parameters: %v", pluginID, parameters)
	return nil
}

func (ams *PluginManagementService) startPluginInternal(pluginID string, parameters map[string]interface{}) error {
	plugin := ams.objects[pluginID]
	if !plugin.CanStart() {
		return fmt.Errorf("plugin unit cannot be started in current state: %s", plugin.Status)
	}

	// Create runtime instance
	instance := &objects.PluginRuntimeInstance{
		InstanceID:    fmt.Sprintf("instance_%s_%d", pluginID, time.Now().UnixNano()),
		StartedAt:     time.Now(),
		Status:        "running",
		Configuration: parameters,
		Environment:   make(map[string]string),
		HealthStatus:  "healthy",
		RestartCount:  0,
		ResourceUsage: &objects.PluginResourceUsage{
			LastUpdated: time.Now(),
		},
	}

	plugin.Status = "running"
	plugin.RuntimeInstance = instance
	plugin.LastActivity = &time.Time{}
	*plugin.LastActivity = time.Now()

	ams.runtimeInstances[pluginID] = instance
	ams.storePlugin(plugin)

	ams.recordEvent(&objects.PluginEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID:    plugin.ID,
		InstanceID:  instance.InstanceID,
		Type:        "started",
		Description: fmt.Sprintf("Plugin unit %s started", plugin.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *PluginManagementService) stopPluginInternal(pluginID string) error {
	plugin := ams.objects[pluginID]
	if !plugin.CanStop() {
		return fmt.Errorf("plugin unit cannot be stopped in current state: %s", plugin.Status)
	}

	if instance, exists := ams.runtimeInstances[pluginID]; exists {
		instance.Status = "stopped"
		delete(ams.runtimeInstances, pluginID)
	}

	plugin.Status = "stopped"
	plugin.RuntimeInstance = nil

	ams.storePlugin(plugin)

	ams.recordEvent(&objects.PluginEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		PluginID:    plugin.ID,
		Type:        "stopped",
		Description: fmt.Sprintf("Plugin unit %s stopped", plugin.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *PluginManagementService) scalePluginInternal(pluginID string, parameters map[string]interface{}) error {
	// Extract scaling parameters
	replicas, ok := parameters["replicas"].(float64)
	if !ok {
		return fmt.Errorf("invalid replicas parameter for plugin item %s", pluginID)
	}

	cpuLimit, _ := parameters["cpu_limit"].(float64)
	memoryLimit, _ := parameters["memory_limit"].(float64)

	// Implement actual scaling logic
	log.Printf("Scaling plugin unit %s to %d replicas (CPU: %.1f%%, Memory: %dMB)",
		pluginID, int(replicas), cpuLimit, int(memoryLimit))

	// Update deployment with scaling information
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment, exists := ams.deployments[pluginID]; exists {
		deployment.Replicas = int(replicas)
		if deployment.Config == nil {
			deployment.Config = &objects.PluginDeploymentConfig{}
		}
		if deployment.Config.ResourceLimits == nil {
			deployment.Config.ResourceLimits = &objects.PluginResourceLimits{}
		}
		deployment.Config.ResourceLimits.MaxCPUPercent = cpuLimit
		deployment.Config.ResourceLimits.MaxMemoryMB = int(memoryLimit)
	} else {
		// Create new deployment if it doesn't exist
		ams.deployments[pluginID] = &objects.PluginDeployment{
			ID:        fmt.Sprintf("plugin_deployment_%d", time.Now().UnixNano()),
			PluginID:  pluginID,
			Name:      fmt.Sprintf("plugin-deployment-%s", pluginID),
			Status:    "deployed",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config: &objects.PluginDeploymentConfig{
				ResourceLimits: &objects.PluginResourceLimits{
					MaxCPUPercent: cpuLimit,
					MaxMemoryMB:   int(memoryLimit),
				},
			},
			Replicas:  int(replicas),
			Instances: []*objects.PluginRuntimeInstance{},
		}
	}

	return nil
}

func (ams *PluginManagementService) monitoringLoop() {
	ticker := time.NewTicker(ams.monitoringInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.updatePluginStatuses()
	}
}

func (ams *PluginManagementService) metricsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.collectPluginMetrics()
	}
}

func (ams *PluginManagementService) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !ams.running {
			return
		}

		ams.performHealthChecks()
	}
}

func (ams *PluginManagementService) updatePluginStatuses() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for pluginID, plugin := range ams.objects {
		if instance, exists := ams.runtimeInstances[pluginID]; exists {
			// Update instance health status
			instance.LastHealthCheck = &time.Time{}
			*instance.LastHealthCheck = time.Now()

			// Simulate health check (in real implementation, this would check actual process)
			if time.Since(instance.StartedAt) > 5*time.Minute {
				instance.HealthStatus = "healthy"
			}
		}

		// Update plugin unit last activity
		if plugin.IsRunning() {
			plugin.LastActivity = &time.Time{}
			*plugin.LastActivity = time.Now()
		}
	}
}

func (ams *PluginManagementService) collectPluginMetrics() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for pluginID, instance := range ams.runtimeInstances {
		// Simulate metrics collection
		metrics := &objects.PluginMetrics{
			PluginID:          pluginID,
			InstanceID:        instance.InstanceID,
			Timestamp:         time.Now(),
			RequestsPerSecond: 10.0 + float64(time.Now().Unix()%50),
			AverageLatency:    50.0 + float64(time.Now().Unix()%100),
			ErrorRate:         0.1 + float64(time.Now().Unix()%5)/100.0,
			Throughput:        1.5 + float64(time.Now().Unix()%10)/10.0,
			ResourceUsage: &objects.PluginResourceUsage{
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
		if ams.metrics[pluginID] == nil {
			ams.metrics[pluginID] = make([]*objects.PluginMetrics, 0)
		}
		ams.metrics[pluginID] = append(ams.metrics[pluginID], metrics)

		// Keep only last 100 metrics per plugin unit
		if len(ams.metrics[pluginID]) > 100 {
			ams.metrics[pluginID] = ams.metrics[pluginID][len(ams.metrics[pluginID])-100:]
		}

		// Update instance resource usage
		instance.ResourceUsage = metrics.ResourceUsage
	}
}

func (ams *PluginManagementService) performHealthChecks() {
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

// GetPluginMetrics returns metrics for a specific plugin unit
func (ams *PluginManagementService) GetPluginMetrics(pluginID string, limit int) ([]*objects.PluginMetrics, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	metrics, exists := ams.metrics[pluginID]
	if !exists {
		return []*objects.PluginMetrics{}, nil
	}

	if limit > 0 && len(metrics) > limit {
		return metrics[len(metrics)-limit:], nil
	}

	return metrics, nil
}

// GetPluginLogs returns logs for a specific plugin unit
func (ams *PluginManagementService) GetPluginLogs(pluginID string, limit int) ([]*objects.PluginLog, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	logs, exists := ams.logs[pluginID]
	if !exists {
		return []*objects.PluginLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

// GetPluginEvents returns events for a specific plugin item or all events
func (ams *PluginManagementService) GetPluginEvents(pluginID string, limit int) ([]*objects.PluginEvent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var filteredEvents []*objects.PluginEvent
	for _, event := range ams.events {
		if pluginID == "" || event.PluginID == pluginID {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > limit {
		return filteredEvents[len(filteredEvents)-limit:], nil
	}

	return filteredEvents, nil
}

// GetPluginTemplates returns all plugin templates
func (ams *PluginManagementService) GetPluginTemplates() ([]*objects.PluginTemplate, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var templates []*objects.PluginTemplate
	for _, template := range ams.templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// CreatePluginTemplate creates a new plugin template
func (ams *PluginManagementService) CreatePluginTemplate(template *objects.PluginTemplate) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if template.ID == "" || template.Name == "" {
		return fmt.Errorf("template ID and name are required")
	}

	if _, exists := ams.templates[template.ID]; exists {
		return fmt.Errorf("plugin template already exists: %s", template.ID)
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.UsageCount = 0

	ams.templates[template.ID] = template

	// Store in database
	if data, err := json.Marshal(template); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("plugin_template:"+template.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Plugin template created: %s (%s)", template.Name, template.ID)
	return nil
}

// GetPluginDeployments returns all deployments for an plugin unit
func (ams *PluginManagementService) GetPluginDeployments(pluginID string) ([]*objects.PluginDeployment, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var deployments []*objects.PluginDeployment
	for _, deployment := range ams.deployments {
		if pluginID == "" || deployment.PluginID == pluginID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// CreatePluginDeployment creates a new deployment for a plugin unit
func (ams *PluginManagementService) CreatePluginDeployment(deployment *objects.PluginDeployment) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment.ID == "" || deployment.PluginID == "" {
		return fmt.Errorf("deployment ID and plugin ID are required")
	}

	if _, exists := ams.deployments[deployment.ID]; exists {
		return fmt.Errorf("plugin deployment already exists: %s", deployment.ID)
	}

	// Verify plugin exists
	if _, exists := ams.objects[deployment.PluginID]; !exists {
		return fmt.Errorf("plugin unit not found: %s", deployment.PluginID)
	}

	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()
	deployment.Status = "pending"
	deployment.Instances = make([]*objects.PluginRuntimeInstance, 0)

	ams.deployments[deployment.ID] = deployment

	// Store in database
	if data, err := json.Marshal(deployment); err == nil {
		ams.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("plugin_deployment:"+deployment.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Plugin deployment created: %s for plugin item %s", deployment.ID, deployment.PluginID)
	return nil
}
