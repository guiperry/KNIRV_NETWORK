package cortexdemo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// DemoOrchestrator manages automated CORTEX demonstrations
type DemoOrchestrator struct {
	Config       OrchestratorConfig
	Scenarios    map[string]*DemoScenario
	ActiveDemos  map[string]*DemoExecution
	CortexAgents map[string]*CortexAgent
	Services     map[string]*ServiceClient
	EventBus     *DemoEventBus
	Metrics      *DemoMetrics
	mu           sync.RWMutex
}

// OrchestratorConfig holds orchestrator configuration
type OrchestratorConfig struct {
	CortexEndpoint   string
	ServicesEndpoint map[string]string
	DefaultTimeout   time.Duration
	MaxConcurrentDemos int
	MetricsEnabled   bool
	RecordingEnabled bool
}

// DemoScenario represents a complete demo scenario
type DemoScenario struct {
	Name         string                 `yaml:"name"`
	Description  string                 `yaml:"description"`
	Duration     string                 `yaml:"duration"`
	Type         string                 `yaml:"type"`
	Participants []DemoParticipant      `yaml:"participants"`
	Workflow     []DemoStep             `yaml:"workflow"`
	SuccessCriteria map[string]interface{} `yaml:"success_criteria"`
}

// DemoParticipant represents a participant in the demo
type DemoParticipant struct {
	Type         string   `yaml:"type"`
	ID           string   `yaml:"id"`
	Role         string   `yaml:"role"`
	Capabilities []string `yaml:"capabilities"`
}

// DemoStep represents a step in the demo workflow
type DemoStep struct {
	Step        string                 `yaml:"step"`
	Description string                 `yaml:"description"`
	Action      string                 `yaml:"action"`
	Target      string                 `yaml:"target"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Timeout     string                 `yaml:"timeout"`
	Validation  []DemoValidation       `yaml:"validation"`
}

// DemoValidation represents validation criteria for a step
type DemoValidation struct {
	Type     string      `yaml:"type"`
	Expected interface{} `yaml:"expected"`
}

// DemoExecution represents an active demo execution
type DemoExecution struct {
	ID           string
	ScenarioName string
	StartTime    time.Time
	EndTime      time.Time
	Status       DemoStatus
	CurrentStep  int
	Results      map[string]*StepResult
	Participants map[string]*ParticipantState
	Metrics      ExecutionMetrics
	Context      context.Context
	Cancel       context.CancelFunc
}

// DemoStatus represents the status of a demo execution
type DemoStatus int

const (
	DemoStatusPending DemoStatus = iota
	DemoStatusRunning
	DemoStatusCompleted
	DemoStatusFailed
	DemoStatusCancelled
)

// StepResult represents the result of a demo step
type StepResult struct {
	StepName    string
	StartTime   time.Time
	EndTime     time.Time
	Status      string
	Output      interface{}
	Error       string
	Validations map[string]bool
}

// ParticipantState represents the state of a demo participant
type ParticipantState struct {
	ID       string
	Type     string
	Status   string
	Metrics  ParticipantMetrics
	LastSeen time.Time
}

// ExecutionMetrics holds metrics for demo execution
type ExecutionMetrics struct {
	TotalSteps      int
	CompletedSteps  int
	FailedSteps     int
	TotalDuration   time.Duration
	StepDurations   map[string]time.Duration
	ResourceUsage   ResourceMetrics
}

// ParticipantMetrics holds metrics for demo participants
type ParticipantMetrics struct {
	TasksCompleted  int
	ResponseTime    time.Duration
	SuccessRate     float64
	ErrorCount      int
}

// ResourceMetrics holds resource usage metrics
type ResourceMetrics struct {
	CPUUsage    float64
	MemoryUsage int64
	NetworkIO   int64
}

// DemoEventBus manages demo events
type DemoEventBus struct {
	events   []DemoEvent
	handlers map[string][]DemoEventHandler
	mu       sync.RWMutex
}

// DemoEvent represents an event during demo execution
type DemoEvent struct {
	ID        string
	DemoID    string
	Type      string
	Source    string
	Data      map[string]interface{}
	Timestamp time.Time
}

// DemoEventHandler handles demo events
type DemoEventHandler func(event DemoEvent) error

// DemoMetrics aggregates demo metrics
type DemoMetrics struct {
	TotalDemos      int
	SuccessfulDemos int
	FailedDemos     int
	AverageRuntime  time.Duration
	SuccessRate     float64
	LastUpdated     time.Time
}

// CortexAgent represents a CORTEX agent for demos
type CortexAgent struct {
	ID           string
	Type         string
	Capabilities []string
	Status       string
	Connection   *AgentConnection
	Performance  AgentPerformance
}

// AgentConnection represents connection to CORTEX
type AgentConnection struct {
	Endpoint    string
	SessionID   string
	Connected   bool
	LastPing    time.Time
}

// AgentPerformance tracks agent performance
type AgentPerformance struct {
	TasksCompleted int
	SuccessRate    float64
	AvgLatency     time.Duration
	ErrorCount     int
}

// ServiceClient represents a testnet service client
type ServiceClient struct {
	Name     string
	BaseURL  string
	Healthy  bool
	LastPing time.Time
}

// NewDemoOrchestrator creates a new demo orchestrator
func NewDemoOrchestrator(config OrchestratorConfig) *DemoOrchestrator {
	return &DemoOrchestrator{
		Config:       config,
		Scenarios:    make(map[string]*DemoScenario),
		ActiveDemos:  make(map[string]*DemoExecution),
		CortexAgents: make(map[string]*CortexAgent),
		Services:     make(map[string]*ServiceClient),
		EventBus:     NewDemoEventBus(),
		Metrics:      &DemoMetrics{},
	}
}

// NewDemoEventBus creates a new demo event bus
func NewDemoEventBus() *DemoEventBus {
	return &DemoEventBus{
		events:   make([]DemoEvent, 0),
		handlers: make(map[string][]DemoEventHandler),
	}
}

// Initialize initializes the demo orchestrator
func (do *DemoOrchestrator) Initialize(ctx context.Context) error {
	log.Println("Initializing CORTEX Demo Orchestrator...")

	// Load demo scenarios
	if err := do.LoadScenarios("../config/demo-scenarios.yaml"); err != nil {
		return fmt.Errorf("failed to load scenarios: %w", err)
	}

	// Initialize CORTEX agents
	if err := do.InitializeCortexAgents(ctx); err != nil {
		return fmt.Errorf("failed to initialize CORTEX agents: %w", err)
	}

	// Initialize service clients
	if err := do.InitializeServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Start metrics collection
	if do.Config.MetricsEnabled {
		go do.collectMetrics(ctx)
	}

	log.Println("CORTEX Demo Orchestrator initialized successfully")
	return nil
}

// LoadScenarios loads demo scenarios from YAML file
func (do *DemoOrchestrator) LoadScenarios(filename string) error {
	// In a real implementation, this would read from file
	// For now, create sample scenarios programmatically
	
	skillDevScenario := &DemoScenario{
		Name:        "CORTEX Skill Development Demo",
		Description: "Demonstrates CORTEX agent creating and registering a new skill",
		Duration:    "10m",
		Type:        "demo",
		Participants: []DemoParticipant{
			{Type: "cortex_agent", ID: "cortex-dev-001", Role: "developer"},
			{Type: "service", ID: "knirvchain", Role: "skill-registry"},
			{Type: "service", ID: "knirvgraph", Role: "knowledge-storage"},
			{Type: "service", ID: "knirv-nexus", Role: "validation"},
		},
		Workflow: []DemoStep{
			{
				Step:        "initialize_agent",
				Description: "Initialize CORTEX developer agent",
				Action:      "agent_init",
				Target:      "cortex-dev-001",
				Timeout:     "30s",
				Validation: []DemoValidation{
					{Type: "agent_status", Expected: "active"},
				},
			},
			{
				Step:        "create_skill",
				Description: "Agent creates new coding skill",
				Action:      "skill_create",
				Target:      "cortex-dev-001",
				Parameters: map[string]interface{}{
					"skill_type":  "code_generation",
					"language":    "python",
					"complexity":  "intermediate",
					"description": "Python web scraping automation",
				},
				Timeout: "2m",
				Validation: []DemoValidation{
					{Type: "skill_created", Expected: true},
				},
			},
		},
		SuccessCriteria: map[string]interface{}{
			"all_steps_completed": true,
			"skill_registered":    true,
			"execution_time":      "<10m",
		},
	}

	do.Scenarios["skill-development"] = skillDevScenario

	log.Printf("Loaded %d demo scenarios", len(do.Scenarios))
	return nil
}

// InitializeCortexAgents initializes CORTEX agents
func (do *DemoOrchestrator) InitializeCortexAgents(ctx context.Context) error {
	agents := []struct {
		id   string
		typ  string
		caps []string
	}{
		{"cortex-dev-001", "Developer", []string{"skill-creation", "code-generation", "testing"}},
		{"cortex-collab-001", "Collaborator", []string{"task-coordination", "knowledge-sharing"}},
		{"cortex-learner-001", "Learner", []string{"adaptation", "pattern-recognition", "optimization"}},
	}

	for _, agentDef := range agents {
		agent := &CortexAgent{
			ID:           agentDef.id,
			Type:         agentDef.typ,
			Capabilities: agentDef.caps,
			Status:       "idle",
			Connection: &AgentConnection{
				Endpoint:  do.Config.CortexEndpoint,
				Connected: false,
			},
			Performance: AgentPerformance{},
		}

		// Connect to CORTEX
		if err := do.connectAgent(ctx, agent); err != nil {
			return fmt.Errorf("failed to connect agent %s: %w", agent.ID, err)
		}

		do.CortexAgents[agent.ID] = agent
		log.Printf("Initialized CORTEX agent: %s (%s)", agent.ID, agent.Type)
	}

	return nil
}

// InitializeServices initializes service clients
func (do *DemoOrchestrator) InitializeServices(ctx context.Context) error {
	services := map[string]string{
		"knirv-root":    "http://localhost:1317",
		"knirvchain":    "http://localhost:8090",
		"knirvgraph":    "http://localhost:8082",
		"knirv-nexus":   "http://localhost:8084",
		"knirv-router":  "http://localhost:5001",
		"knirv-gateway": "http://localhost:8087",
	}

	for name, url := range services {
		client := &ServiceClient{
			Name:    name,
			BaseURL: url,
			Healthy: false,
		}

		// Health check
		if err := do.healthCheckService(ctx, client); err != nil {
			log.Printf("Warning: Service %s not healthy: %v", name, err)
		} else {
			client.Healthy = true
			client.LastPing = time.Now()
		}

		do.Services[name] = client
		log.Printf("Initialized service client: %s", name)
	}

	return nil
}

// RunDemo executes a demo scenario
func (do *DemoOrchestrator) RunDemo(ctx context.Context, scenarioName string) (*DemoExecution, error) {
	do.mu.Lock()
	defer do.mu.Unlock()

	scenario, exists := do.Scenarios[scenarioName]
	if !exists {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	// Check concurrent demo limit
	if len(do.ActiveDemos) >= do.Config.MaxConcurrentDemos {
		return nil, fmt.Errorf("maximum concurrent demos reached")
	}

	// Create demo execution
	demoCtx, cancel := context.WithCancel(ctx)
	execution := &DemoExecution{
		ID:           fmt.Sprintf("demo_%s_%d", scenarioName, time.Now().Unix()),
		ScenarioName: scenarioName,
		StartTime:    time.Now(),
		Status:       DemoStatusPending,
		CurrentStep:  0,
		Results:      make(map[string]*StepResult),
		Participants: make(map[string]*ParticipantState),
		Context:      demoCtx,
		Cancel:       cancel,
		Metrics: ExecutionMetrics{
			TotalSteps:    len(scenario.Workflow),
			StepDurations: make(map[string]time.Duration),
		},
	}

	do.ActiveDemos[execution.ID] = execution

	// Start demo execution
	go do.executeDemo(execution, scenario)

	log.Printf("Started demo execution: %s (scenario: %s)", execution.ID, scenarioName)
	return execution, nil
}

// executeDemo executes a demo scenario
func (do *DemoOrchestrator) executeDemo(execution *DemoExecution, scenario *DemoScenario) {
	defer func() {
		do.mu.Lock()
		delete(do.ActiveDemos, execution.ID)
		do.mu.Unlock()
	}()

	execution.Status = DemoStatusRunning

	// Execute each step
	for i, step := range scenario.Workflow {
		execution.CurrentStep = i

		stepResult, err := do.executeStep(execution.Context, step, execution)
		if err != nil {
			log.Printf("Demo %s failed at step %s: %v", execution.ID, step.Step, err)
			execution.Status = DemoStatusFailed
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
		} else {
			stepResult.Status = "completed"
			execution.Metrics.CompletedSteps++
		}

		execution.Results[step.Step] = stepResult

		// Check if demo should continue
		if execution.Status == DemoStatusFailed {
			break
		}

		// Publish step completion event
		do.EventBus.PublishEvent(DemoEvent{
			ID:     fmt.Sprintf("step_%s_%d", execution.ID, i),
			DemoID: execution.ID,
			Type:   "step_completed",
			Source: "orchestrator",
			Data: map[string]interface{}{
				"step":   step.Step,
				"status": stepResult.Status,
			},
			Timestamp: time.Now(),
		})
	}

	// Finalize demo
	execution.EndTime = time.Now()
	execution.Metrics.TotalDuration = execution.EndTime.Sub(execution.StartTime)

	if execution.Status != DemoStatusFailed {
		execution.Status = DemoStatusCompleted
		do.Metrics.SuccessfulDemos++
	} else {
		do.Metrics.FailedDemos++
	}

	do.Metrics.TotalDemos++
	do.updateMetrics()

	log.Printf("Demo %s completed with status: %v", execution.ID, execution.Status)
}

// executeStep executes a single demo step
func (do *DemoOrchestrator) executeStep(ctx context.Context, step DemoStep, execution *DemoExecution) (*StepResult, error) {
	result := &StepResult{
		StepName:    step.Step,
		StartTime:   time.Now(),
		Validations: make(map[string]bool),
	}

	// Parse timeout
	timeout, err := time.ParseDuration(step.Timeout)
	if err != nil {
		timeout = do.Config.DefaultTimeout
	}

	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute step action
	switch step.Action {
	case "agent_init":
		result.Output, err = do.executeAgentInit(stepCtx, step)
	case "skill_create":
		result.Output, err = do.executeSkillCreate(stepCtx, step)
	case "blockchain_register":
		result.Output, err = do.executeBlockchainRegister(stepCtx, step)
	case "graph_update":
		result.Output, err = do.executeGraphUpdate(stepCtx, step)
	case "skill_validate":
		result.Output, err = do.executeSkillValidate(stepCtx, step)
	default:
		err = fmt.Errorf("unknown action: %s", step.Action)
	}

	result.EndTime = time.Now()
	execution.Metrics.StepDurations[step.Step] = result.EndTime.Sub(result.StartTime)

	if err != nil {
		return result, err
	}

	// Validate step results
	for _, validation := range step.Validation {
		passed := do.validateStepResult(result.Output, validation)
		result.Validations[validation.Type] = passed
		if !passed {
			return result, fmt.Errorf("validation failed: %s", validation.Type)
		}
	}

	return result, nil
}

// Helper methods for step execution
func (do *DemoOrchestrator) executeAgentInit(ctx context.Context, step DemoStep) (interface{}, error) {
	// Implementation would initialize CORTEX agent
	return map[string]interface{}{
		"agent_id": step.Target,
		"status":   "active",
		"message":  "Agent initialized successfully",
	}, nil
}

func (do *DemoOrchestrator) executeSkillCreate(ctx context.Context, step DemoStep) (interface{}, error) {
	// Implementation would create skill through CORTEX
	return map[string]interface{}{
		"skill_id":   "skill_001",
		"created":    true,
		"parameters": step.Parameters,
	}, nil
}

func (do *DemoOrchestrator) executeBlockchainRegister(ctx context.Context, step DemoStep) (interface{}, error) {
	// Implementation would register on blockchain
	return map[string]interface{}{
		"transaction_id": "tx_001",
		"confirmed":      true,
		"block_height":   12345,
	}, nil
}

func (do *DemoOrchestrator) executeGraphUpdate(ctx context.Context, step DemoStep) (interface{}, error) {
	// Implementation would update knowledge graph
	return map[string]interface{}{
		"node_id": "node_001",
		"updated": true,
		"edges":   3,
	}, nil
}

func (do *DemoOrchestrator) executeSkillValidate(ctx context.Context, step DemoStep) (interface{}, error) {
	// Implementation would validate skill
	return map[string]interface{}{
		"validation_id": "val_001",
		"passed":        true,
		"score":         0.95,
	}, nil
}

// validateStepResult validates step output against criteria
func (do *DemoOrchestrator) validateStepResult(output interface{}, validation DemoValidation) bool {
	// Implementation would validate based on type and expected value
	// For now, return true for basic validation
	return true
}

// connectAgent connects to a CORTEX agent
func (do *DemoOrchestrator) connectAgent(ctx context.Context, agent *CortexAgent) error {
	// Implementation would establish connection to CORTEX
	agent.Connection.Connected = true
	agent.Connection.SessionID = fmt.Sprintf("session_%s", agent.ID)
	agent.Status = "connected"
	return nil
}

// healthCheckService performs health check on a service
func (do *DemoOrchestrator) healthCheckService(ctx context.Context, client *ServiceClient) error {
	// Implementation would make HTTP health check
	// For now, assume all services are healthy
	return nil
}

// collectMetrics collects demo metrics
func (do *DemoOrchestrator) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			do.updateMetrics()
		}
	}
}

// updateMetrics updates demo metrics
func (do *DemoOrchestrator) updateMetrics() {
	do.mu.RLock()
	defer do.mu.RUnlock()

	if do.Metrics.TotalDemos > 0 {
		do.Metrics.SuccessRate = float64(do.Metrics.SuccessfulDemos) / float64(do.Metrics.TotalDemos)
	}
	do.Metrics.LastUpdated = time.Now()
}

// PublishEvent publishes an event to the demo event bus
func (deb *DemoEventBus) PublishEvent(event DemoEvent) error {
	deb.mu.Lock()
	defer deb.mu.Unlock()

	deb.events = append(deb.events, event)

	// Trigger handlers
	if handlers, exists := deb.handlers[event.Type]; exists {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	return nil
}

// GetDemoStatus returns the status of a demo execution
func (do *DemoOrchestrator) GetDemoStatus(demoID string) (*DemoExecution, error) {
	do.mu.RLock()
	defer do.mu.RUnlock()

	execution, exists := do.ActiveDemos[demoID]
	if !exists {
		return nil, fmt.Errorf("demo %s not found", demoID)
	}

	return execution, nil
}

// ListActiveDemo returns all active demos
func (do *DemoOrchestrator) ListActiveDemos() []*DemoExecution {
	do.mu.RLock()
	defer do.mu.RUnlock()

	demos := make([]*DemoExecution, 0, len(do.ActiveDemos))
	for _, demo := range do.ActiveDemos {
		demos = append(demos, demo)
	}

	return demos
}

// Shutdown gracefully shuts down the orchestrator
func (do *DemoOrchestrator) Shutdown(ctx context.Context) error {
	log.Println("Shutting down CORTEX Demo Orchestrator...")

	// Cancel all active demos
	do.mu.Lock()
	for _, execution := range do.ActiveDemos {
		execution.Cancel()
	}
	do.mu.Unlock()

	// Disconnect CORTEX agents
	for _, agent := range do.CortexAgents {
		agent.Connection.Connected = false
		agent.Status = "disconnected"
	}

	log.Println("CORTEX Demo Orchestrator shutdown complete")
	return nil
}
