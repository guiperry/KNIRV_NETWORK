package agentmanagement

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-backend/internal/models"

	"github.com/tidwall/buntdb"
)

// AgentManagementService provides comprehensive agent administration
type AgentManagementService struct {
	db      *buntdb.DB
	mu      sync.RWMutex
	running bool

	// Agent storage and tracking
	agents      map[string]*models.Agent
	deployments map[string]*models.AgentDeployment
	templates   map[string]*models.AgentTemplate

	// Runtime monitoring
	runtimeInstances map[string]*models.AgentRuntimeInstance
	metrics          map[string][]*models.AgentMetrics
	logs             map[string][]*models.AgentLog
	events           []*models.AgentEvent

	// Configuration
	maxAgents             int
	maxInstancesPerAgent  int
	defaultResourceLimits *models.AgentResourceLimits
	monitoringInterval    time.Duration

	// Lifecycle management
	agentServer interface{}
}

type AgentInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// NewAgentManagementService creates a new agent management service
func NewAgentManagementService(db *buntdb.DB) *AgentManagementService {
	service := &AgentManagementService{
		db:                   db,
		agents:               make(map[string]*models.Agent),
		deployments:          make(map[string]*models.AgentDeployment),
		templates:            make(map[string]*models.AgentTemplate),
		runtimeInstances:     make(map[string]*models.AgentRuntimeInstance),
		metrics:              make(map[string][]*models.AgentMetrics),
		logs:                 make(map[string][]*models.AgentLog),
		events:               make([]*models.AgentEvent, 0),
		maxAgents:            100,
		maxInstancesPerAgent: 10,
		monitoringInterval:   30 * time.Second,
		defaultResourceLimits: &models.AgentResourceLimits{
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
	service.loadAgentData()

	return service
}

// SetAgentServerReference sets reference to the agent server for integration
func (ams *AgentManagementService) SetAgentServerReference(agentServer interface{}) {
	ams.mu.Lock()
	defer ams.mu.Unlock()
	ams.agentServer = agentServer
}

// Start begins the agent management service
func (ams *AgentManagementService) Start() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if ams.running {
		return fmt.Errorf("agent management service already running")
	}

	ams.running = true

	log.Println("Starting agent management service...")

	// Start monitoring goroutines
	go ams.monitoringLoop()
	go ams.metricsCollectionLoop()
	go ams.healthCheckLoop()

	log.Println("Agent management service started successfully")
	return nil
}

// Stop stops the agent management service
func (ams *AgentManagementService) Stop() error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !ams.running {
		return fmt.Errorf("agent management service not running")
	}

	ams.running = false

	log.Println("Agent management service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (ams *AgentManagementService) IsRunning() bool {
	ams.mu.RLock()
	defer ams.mu.RUnlock()
	return ams.running
}

// GetAllAgents returns all agents with optional filtering
func (ams *AgentManagementService) GetAllAgents(filter *models.AgentFilter) ([]*models.Agent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var agents []*models.Agent
	for _, agent := range ams.agents {
		if ams.matchesFilter(agent, filter) {
			agents = append(agents, agent)
		}
	}

	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(agents) {
			return []*models.Agent{}, nil
		}
		if end > len(agents) {
			end = len(agents)
		}
		agents = agents[start:end]
	}

	return agents, nil
}

// GetAgent returns a specific agent by ID
func (ams *AgentManagementService) GetAgent(agentID string) (*models.Agent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	agent, exists := ams.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// CreateAgent creates a new agent
func (ams *AgentManagementService) CreateAgent(agent *models.Agent) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !agent.IsValid() {
		return fmt.Errorf("invalid agent data")
	}

	if _, exists := ams.agents[agent.ID]; exists {
		return fmt.Errorf("agent already exists: %s", agent.ID)
	}

	if len(ams.agents) >= ams.maxAgents {
		return fmt.Errorf("maximum number of agents reached: %d", ams.maxAgents)
	}

	// Set default values
	agent.UploadedAt = time.Now()
	agent.LastModified = time.Now()
	agent.Status = "uploaded"

	if agent.ResourceLimits == nil {
		agent.ResourceLimits = ams.defaultResourceLimits
	}

	ams.agents[agent.ID] = agent

	// Store in database
	ams.storeAgent(agent)

	// Record event
	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		Type:        "uploaded",
		Description: fmt.Sprintf("Agent %s uploaded", agent.Name),
		Timestamp:   time.Now(),
		UserID:      agent.UploadedBy,
	})

	log.Printf("Agent created: %s (%s)", agent.Name, agent.ID)
	return nil
}

// UpdateAgent updates an existing agent
func (ams *AgentManagementService) UpdateAgent(agentID string, updates *models.Agent) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	agent, exists := ams.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Update allowed fields
	if updates.Name != "" {
		agent.Name = updates.Name
	}
	if updates.Description != "" {
		agent.Description = updates.Description
	}
	if updates.Version != "" {
		agent.Version = updates.Version
	}
	if updates.Configuration != nil {
		agent.Configuration = updates.Configuration
	}
	if updates.ResourceLimits != nil {
		agent.ResourceLimits = updates.ResourceLimits
	}
	if updates.Tags != nil {
		agent.Tags = updates.Tags
	}

	agent.LastModified = time.Now()

	// Store updated agent
	ams.storeAgent(agent)

	// Record event
	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		Type:        "updated",
		Description: fmt.Sprintf("Agent %s updated", agent.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Agent updated: %s (%s)", agent.Name, agent.ID)
	return nil
}

// DeleteAgent deletes an agent
func (ams *AgentManagementService) DeleteAgent(agentID string) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	agent, exists := ams.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Stop agent if running
	if agent.IsRunning() {
		if err := ams.stopAgentInternal(agentID); err != nil {
			return fmt.Errorf("failed to stop agent before deletion: %w", err)
		}
	}

	// Remove from memory
	delete(ams.agents, agentID)
	delete(ams.runtimeInstances, agentID)
	delete(ams.metrics, agentID)
	delete(ams.logs, agentID)

	// Remove from database
	ams.db.Update(func(tx *buntdb.Tx) error {
		tx.Delete("agent:" + agentID)
		return nil
	})

	// Record event
	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		Type:        "deleted",
		Description: fmt.Sprintf("Agent %s deleted", agent.Name),
		Timestamp:   time.Now(),
	})

	log.Printf("Agent deleted: %s (%s)", agent.Name, agent.ID)
	return nil
}

// ExecuteAgentAction executes an action on an agent
func (ams *AgentManagementService) ExecuteAgentAction(agentID string, action *models.AgentAction) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	_, exists := ams.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	switch action.Action {
	case "deploy":
		return ams.deployAgentInternal(agentID, action.Parameters)
	case "start":
		return ams.startAgentInternal(agentID, action.Parameters)
	case "stop":
		return ams.stopAgentInternal(agentID)
	case "restart":
		if err := ams.stopAgentInternal(agentID); err != nil {
			return err
		}
		return ams.startAgentInternal(agentID, action.Parameters)
	case "scale":
		return ams.scaleAgentInternal(agentID, action.Parameters)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// GetAgentSummary returns a summary of all agents
func (ams *AgentManagementService) GetAgentSummary() *models.AgentSummary {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	summary := &models.AgentSummary{
		TotalAgents: len(ams.agents),
	}

	for _, agent := range ams.agents {
		switch agent.Status {
		case "running":
			summary.RunningAgents++
		case "stopped":
			summary.StoppedAgents++
		case "error":
			summary.ErrorAgents++
		case "deployed":
			summary.DeployedAgents++
		case "uploaded":
			summary.UploadedAgents++
		}
	}

	return summary
}

// Private methods for internal operations
func (ams *AgentManagementService) initializeDatabase() {
	ams.db.Update(func(tx *buntdb.Tx) error {
		tx.CreateIndex("agents", "agent:*", buntdb.IndexString)
		tx.CreateIndex("deployments", "deployment:*", buntdb.IndexString)
		tx.CreateIndex("templates", "template:*", buntdb.IndexString)
		tx.CreateIndex("events", "event:*", buntdb.IndexString)
		return nil
	})
}

func (ams *AgentManagementService) loadAgentData() {
	// Load agents from database
	ams.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("agents", func(key, value string) bool {
			var agent models.Agent
			if json.Unmarshal([]byte(value), &agent) == nil {
				ams.agents[agent.ID] = &agent
			}
			return true
		})
		return nil
	})

	// Load deployments from database
	ams.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("deployments", func(key, value string) bool {
			var deployment models.AgentDeployment
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
			var template models.AgentTemplate
			if json.Unmarshal([]byte(value), &template) == nil {
				ams.templates[template.ID] = &template
			}
			return true
		})
		return nil
	})
}

func (ams *AgentManagementService) storeAgent(agent *models.Agent) {
	if data, err := json.Marshal(agent); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("agent:"+agent.ID, string(data), nil)
			return nil
		})
	}
}

func (ams *AgentManagementService) recordEvent(event *models.AgentEvent) {
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

func (ams *AgentManagementService) matchesFilter(agent *models.Agent, filter *models.AgentFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if agent.Status == status {
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
		for _, agentType := range filter.Type {
			if agent.Type == agentType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Author filter
	if filter.Author != "" && agent.Author != filter.Author {
		return false
	}

	// Date filters
	if filter.CreatedAfter != nil && agent.UploadedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && agent.UploadedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (ams *AgentManagementService) deployAgentInternal(agentID string, parameters map[string]interface{}) error {
	agent := ams.agents[agentID]
	if !agent.CanDeploy() {
		return fmt.Errorf("agent cannot be deployed in current state: %s", agent.Status)
	}

	agent.Status = "deployed"
	agent.DeployedAt = &time.Time{}
	*agent.DeployedAt = time.Now()

	ams.storeAgent(agent)

	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		Type:        "deployed",
		Description: fmt.Sprintf("Agent %s deployed", agent.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *AgentManagementService) startAgentInternal(agentID string, parameters map[string]interface{}) error {
	agent := ams.agents[agentID]
	if !agent.CanStart() {
		return fmt.Errorf("agent cannot be started in current state: %s", agent.Status)
	}

	// Create runtime instance
	instance := &models.AgentRuntimeInstance{
		InstanceID:    fmt.Sprintf("instance_%s_%d", agentID, time.Now().UnixNano()),
		StartedAt:     time.Now(),
		Status:        "running",
		Configuration: parameters,
		Environment:   make(map[string]string),
		HealthStatus:  "healthy",
		RestartCount:  0,
		ResourceUsage: &models.AgentResourceUsage{
			LastUpdated: time.Now(),
		},
	}

	agent.Status = "running"
	agent.RuntimeInstance = instance
	agent.LastActivity = &time.Time{}
	*agent.LastActivity = time.Now()

	ams.runtimeInstances[agentID] = instance
	ams.storeAgent(agent)

	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		InstanceID:  instance.InstanceID,
		Type:        "started",
		Description: fmt.Sprintf("Agent %s started", agent.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *AgentManagementService) stopAgentInternal(agentID string) error {
	agent := ams.agents[agentID]
	if !agent.CanStop() {
		return fmt.Errorf("agent cannot be stopped in current state: %s", agent.Status)
	}

	if instance, exists := ams.runtimeInstances[agentID]; exists {
		instance.Status = "stopped"
		delete(ams.runtimeInstances, agentID)
	}

	agent.Status = "stopped"
	agent.RuntimeInstance = nil

	ams.storeAgent(agent)

	ams.recordEvent(&models.AgentEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		AgentID:     agent.ID,
		Type:        "stopped",
		Description: fmt.Sprintf("Agent %s stopped", agent.Name),
		Timestamp:   time.Now(),
	})

	return nil
}

func (ams *AgentManagementService) scaleAgentInternal(agentID string, parameters map[string]interface{}) error {
	// Placeholder for scaling logic
	return fmt.Errorf("scaling not yet implemented")
}

func (ams *AgentManagementService) monitoringLoop() {
	ticker := time.NewTicker(ams.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !ams.running {
				return
			}

			ams.updateAgentStatuses()
		}
	}
}

func (ams *AgentManagementService) metricsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !ams.running {
				return
			}

			ams.collectAgentMetrics()
		}
	}
}

func (ams *AgentManagementService) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !ams.running {
				return
			}

			ams.performHealthChecks()
		}
	}
}

func (ams *AgentManagementService) updateAgentStatuses() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for agentID, agent := range ams.agents {
		if instance, exists := ams.runtimeInstances[agentID]; exists {
			// Update instance health status
			instance.LastHealthCheck = &time.Time{}
			*instance.LastHealthCheck = time.Now()

			// Simulate health check (in real implementation, this would check actual process)
			if time.Since(instance.StartedAt) > 5*time.Minute {
				instance.HealthStatus = "healthy"
			}
		}

		// Update agent last activity
		if agent.IsRunning() {
			agent.LastActivity = &time.Time{}
			*agent.LastActivity = time.Now()
		}
	}
}

func (ams *AgentManagementService) collectAgentMetrics() {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	for agentID, instance := range ams.runtimeInstances {
		// Simulate metrics collection
		metrics := &models.AgentMetrics{
			AgentID:           agentID,
			InstanceID:        instance.InstanceID,
			Timestamp:         time.Now(),
			RequestsPerSecond: 10.0 + float64(time.Now().Unix()%50),
			AverageLatency:    50.0 + float64(time.Now().Unix()%100),
			ErrorRate:         0.1 + float64(time.Now().Unix()%5)/100.0,
			Throughput:        1.5 + float64(time.Now().Unix()%10)/10.0,
			ResourceUsage: &models.AgentResourceUsage{
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
		if ams.metrics[agentID] == nil {
			ams.metrics[agentID] = make([]*models.AgentMetrics, 0)
		}
		ams.metrics[agentID] = append(ams.metrics[agentID], metrics)

		// Keep only last 100 metrics per agent
		if len(ams.metrics[agentID]) > 100 {
			ams.metrics[agentID] = ams.metrics[agentID][len(ams.metrics[agentID])-100:]
		}

		// Update instance resource usage
		instance.ResourceUsage = metrics.ResourceUsage
	}
}

func (ams *AgentManagementService) performHealthChecks() {
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

// GetAgentMetrics returns metrics for a specific agent
func (ams *AgentManagementService) GetAgentMetrics(agentID string, limit int) ([]*models.AgentMetrics, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	metrics, exists := ams.metrics[agentID]
	if !exists {
		return []*models.AgentMetrics{}, nil
	}

	if limit > 0 && len(metrics) > limit {
		return metrics[len(metrics)-limit:], nil
	}

	return metrics, nil
}

// GetAgentLogs returns logs for a specific agent
func (ams *AgentManagementService) GetAgentLogs(agentID string, limit int) ([]*models.AgentLog, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	logs, exists := ams.logs[agentID]
	if !exists {
		return []*models.AgentLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

// GetAgentEvents returns events for a specific agent or all events
func (ams *AgentManagementService) GetAgentEvents(agentID string, limit int) ([]*models.AgentEvent, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var filteredEvents []*models.AgentEvent
	for _, event := range ams.events {
		if agentID == "" || event.AgentID == agentID {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > limit {
		return filteredEvents[len(filteredEvents)-limit:], nil
	}

	return filteredEvents, nil
}

// GetAgentTemplates returns all agent templates
func (ams *AgentManagementService) GetAgentTemplates() ([]*models.AgentTemplate, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var templates []*models.AgentTemplate
	for _, template := range ams.templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// CreateAgentTemplate creates a new agent template
func (ams *AgentManagementService) CreateAgentTemplate(template *models.AgentTemplate) error {
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

	log.Printf("Agent template created: %s (%s)", template.Name, template.ID)
	return nil
}

// GetAgentDeployments returns all deployments for an agent
func (ams *AgentManagementService) GetAgentDeployments(agentID string) ([]*models.AgentDeployment, error) {
	ams.mu.RLock()
	defer ams.mu.RUnlock()

	var deployments []*models.AgentDeployment
	for _, deployment := range ams.deployments {
		if agentID == "" || deployment.AgentID == agentID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// CreateAgentDeployment creates a new deployment for an agent
func (ams *AgentManagementService) CreateAgentDeployment(deployment *models.AgentDeployment) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if deployment.ID == "" || deployment.AgentID == "" {
		return fmt.Errorf("deployment ID and agent ID are required")
	}

	if _, exists := ams.deployments[deployment.ID]; exists {
		return fmt.Errorf("deployment already exists: %s", deployment.ID)
	}

	// Verify agent exists
	if _, exists := ams.agents[deployment.AgentID]; !exists {
		return fmt.Errorf("agent not found: %s", deployment.AgentID)
	}

	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()
	deployment.Status = "pending"
	deployment.Instances = make([]*models.AgentRuntimeInstance, 0)

	ams.deployments[deployment.ID] = deployment

	// Store in database
	if data, err := json.Marshal(deployment); err == nil {
		ams.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("deployment:"+deployment.ID, string(data), nil)
			return nil
		})
	}

	log.Printf("Agent deployment created: %s for agent %s", deployment.ID, deployment.AgentID)
	return nil
}
