package cognitiveengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/ebpf"
	"backend_server/internal/objects"
	"backend_server/internal/services/icme"
	"backend_server/internal/services/validation"

	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

// CognitiveEngine manages AI-driven learning and adaptation for the DVE network.
//
// New in this revision:
//   - Configurable loop intervals via EngineConfig
//   - Internal EventBus for event-driven (not just timed) adaptation triggers
//   - Worker pool for concurrent validation-result ingestion
//   - Real eBPF resource telemetry via EBPFBridge
//   - Per-DVE guardrail policy enforcement via GuardrailEngine
//   - DVE ontology management + KNIRVGRAPH integration via DVEOntologyManager
type CognitiveEngine struct {
	db               *database.BuntDBManager
	validationCore   ValidationClient
	inferenceService InferenceClient
	modelManager     ModelManagerClient
	learningState    *LearningState
	adaptationEngine *AdaptationEngine
	metricsCollector *MetricsCollector
	patternAnalyzer  *PatternAnalyzer
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	running          bool
	startedAt        time.Time

	// ── Enhancement subsystems ────────────────────────────────────────────────
	cfg             *EngineConfig
	eventBus        *EventBus
	workerPool      *TaskWorkerPool
	ebpfBridge      *EBPFBridge
	guardrailEngine *GuardrailEngine
	ontologyManager *DVEOntologyManager

	// ── Per-DVE agent runtime telemetry (implements runtime.CognitiveEngineInterface) ─
	agentMetrics   map[string]*AgentDVEMetrics
	agentMetricsMu sync.RWMutex

	// ── GraphRAG knowledge retrieval client (wired after construction) ─────────
	graphRAGClient any
}

// AgentDVEMetrics holds the most recent resource and complexity snapshot for a
// single DVE's oh-my-pi agent container.  This is the bridge between the
// inner (per-DVE) cognitive engine callbacks and the outer (server-wide)
// CognitiveEngine learning & guardrail subsystems.
type AgentDVEMetrics struct {
	DVEID       string
	Complexity  int
	CPUPercent  float64
	MemoryMB    int64
	LastUpdated time.Time
}

// LearningState represents the current state of the cognitive engine's learning
type LearningState struct {
	TotalTasksProcessed   int64                   `json:"total_tasks_processed"`
	SuccessRate           float64                 `json:"success_rate"`
	AverageProcessingTime float64                 `json:"average_processing_time"`
	TaskTypePerformance   map[string]*TaskMetrics `json:"task_type_performance"`
	NodePerformance       map[string]*NodeMetrics `json:"node_performance"`
	AdaptationHistory     []AdaptationEvent       `json:"adaptation_history"`
	LastUpdated           time.Time               `json:"last_updated"`
	LearningProgress      float64                 `json:"learning_progress"`
	ConfidenceLevel       float64                 `json:"confidence_level"`
}

// TaskMetrics tracks performance metrics for different task types
type TaskMetrics struct {
	TaskType          string    `json:"task_type"`
	TasksProcessed    int64     `json:"tasks_processed"`
	SuccessRate       float64   `json:"success_rate"`
	AvgProcessingTime float64   `json:"avg_processing_time"`
	AvgScore          float64   `json:"avg_score"`
	FailurePatterns   []string  `json:"failure_patterns"`
	LastProcessed     time.Time `json:"last_processed"`
}

// NodeMetrics tracks performance metrics for different nodes
type NodeMetrics struct {
	NodeID            string    `json:"node_id"`
	TasksProcessed    int64     `json:"tasks_processed"`
	SuccessRate       float64   `json:"success_rate"`
	AvgProcessingTime float64   `json:"avg_processing_time"`
	ReliabilityScore  float64   `json:"reliability_score"`
	Specializations   []string  `json:"specializations"`
	LastActive        time.Time `json:"last_active"`
}

// AdaptationEvent represents a system adaptation triggered by learning
type AdaptationEvent struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	TriggerReason  string                 `json:"trigger_reason"`
	AdaptationType string                 `json:"adaptation_type"`
	Changes        map[string]interface{} `json:"changes"`
	ExpectedImpact string                 `json:"expected_impact"`
	ActualImpact   *AdaptationResult      `json:"actual_impact,omitempty"`
}

// AdaptationResult represents the measured impact of an adaptation
type AdaptationResult struct {
	MeasuredAt           time.Time `json:"measured_at"`
	SuccessRateChange    float64   `json:"success_rate_change"`
	ProcessingTimeChange float64   `json:"processing_time_change"`
	OverallImprovement   float64   `json:"overall_improvement"`
}

// AdaptationEngine handles system parameter adjustments
type AdaptationEngine struct {
	currentParams   map[string]interface{}
	adaptationRules []AdaptationRule
}

// AdaptationRule defines when and how to adapt system parameters
type AdaptationRule struct {
	ID          string
	Condition   string
	Action      string
	Parameters  map[string]interface{}
	Priority    int
	LastApplied time.Time
}

// MetricsCollector collects and aggregates cognitive engine metrics
type MetricsCollector struct {
	metrics map[string]*CognitiveMetrics
	mu      sync.RWMutex
}

// CognitiveMetrics represents comprehensive cognitive engine performance metrics
type CognitiveMetrics struct {
	NodeID                string    `json:"node_id"`
	TasksProcessed        int64     `json:"tasks_processed"`
	AverageProcessingTime float64   `json:"average_processing_time"`
	SuccessRate           float64   `json:"success_rate"`
	AdaptationScore       float64   `json:"adaptation_score"`
	LearningProgress      float64   `json:"learning_progress"`
	ResourceUtilization   float64   `json:"resource_utilization"`
	Timestamp             time.Time `json:"timestamp"`
}

// PatternAnalyzer identifies patterns in validation results
type PatternAnalyzer struct {
	patterns map[string]*FailurePattern
	mu       sync.RWMutex
}

// FailurePattern represents a detected failure pattern
type FailurePattern struct {
	PatternID       string    `json:"pattern_id"`
	Description     string    `json:"description"`
	TaskTypes       []string  `json:"task_types"`
	Frequency       int       `json:"frequency"`
	AvgImpact       float64   `json:"avg_impact"`
	SuggestedAction string    `json:"suggested_action"`
	LastSeen        time.Time `json:"last_seen"`
}

// Client interfaces for dependencies
type ValidationClient interface {
	GetValidationResults(limit int) ([]*objects.ValidationResult, error)
	GetValidationTasks(filter *validation.TaskFilter) ([]*objects.ValidationTask, error)
}

type InferenceClient interface {
	// Inference service interface
}

type ModelManagerClient interface {
	// Model management interface
}

// NewCognitiveEngine creates a new cognitive engine instance with default configuration.
func NewCognitiveEngine(db *database.BuntDBManager, validationClient ValidationClient, inferenceClient InferenceClient, modelManager ModelManagerClient) *CognitiveEngine {
	return NewCognitiveEngineWithConfig(db, validationClient, inferenceClient, modelManager, DefaultEngineConfig())
}

// NewCognitiveEngineWithConfig creates a cognitive engine using the supplied EngineConfig.
// This is the primary constructor for production use where intervals are
// controlled via YAML / environment variables.
func NewCognitiveEngineWithConfig(
	db *database.BuntDBManager,
	validationClient ValidationClient,
	inferenceClient InferenceClient,
	modelManager ModelManagerClient,
	cfg *EngineConfig,
) *CognitiveEngine {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg == nil {
		cfg = DefaultEngineConfig()
	}

	bus := NewEventBus(64)

	ce := &CognitiveEngine{
		db:               db,
		validationCore:   validationClient,
		inferenceService: inferenceClient,
		modelManager:     modelManager,
		agentMetrics:     make(map[string]*AgentDVEMetrics),
		learningState: &LearningState{
			TaskTypePerformance: make(map[string]*TaskMetrics),
			NodePerformance:     make(map[string]*NodeMetrics),
			AdaptationHistory:   make([]AdaptationEvent, 0),
			LastUpdated:         time.Now(),
		},
		adaptationEngine: &AdaptationEngine{
			currentParams:   make(map[string]interface{}),
			adaptationRules: make([]AdaptationRule, 0),
		},
		metricsCollector: &MetricsCollector{
			metrics: make(map[string]*CognitiveMetrics),
		},
		patternAnalyzer: &PatternAnalyzer{
			patterns: make(map[string]*FailurePattern),
		},
		ctx:    ctx,
		cancel: cancel,

		cfg:             cfg,
		eventBus:        bus,
		workerPool:      NewTaskWorkerPool(cfg.WorkerPoolSize, cfg.TaskQueueCapacity),
		ebpfBridge:      NewEBPFBridge(nil, bus), // eBPF wired later via SetEBPFManager
		guardrailEngine: NewGuardrailEngine(bus),
		ontologyManager: NewDVEOntologyManager(nil, nil), // KNIRVGRAPH wired later
	}

	ce.initializeAdaptationRules()
	ce.loadLearningState()

	return ce
}

// ─── Dependency injection setters ────────────────────────────────────────────

// SetEBPFManager wires a live eBPF manager into the engine.
// Must be called before Start().
func (ce *CognitiveEngine) SetEBPFManager(mgr ebpf.ManagerInterface) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.ebpfBridge = NewEBPFBridge(mgr, ce.eventBus)
}

// SetGraphRAGClient wires the GraphRAG knowledge retrieval client.
// Must be called before Start().
func (ce *CognitiveEngine) SetGraphRAGClient(client any) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.graphRAGClient = client
}

// SetKNIRVGRAPHEngine wires the KNIRVGRAPH temporal hypergraph and zap logger
// into the ontology manager.  Must be called before Start().
func (ce *CognitiveEngine) SetKNIRVGRAPHEngine(hg *icme.TemporalHypergraph, logger *zap.Logger) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.ontologyManager = NewDVEOntologyManager(hg, logger)
}

// RegisterRemediator registers a remediation function for the given action
// name in the guardrail engine.  This is how the WASMRemediator bridge or
// ProductionRemediationSink actions are wired into the policy enforcement loop.
func (ce *CognitiveEngine) RegisterRemediator(action string, fn func(ctx context.Context, violation *PolicyViolation) error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	if ce.guardrailEngine != nil {
		ce.guardrailEngine.RegisterRemediator(action, fn)
	}
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────

// Start starts the cognitive engine and all subsystem goroutines.
func (ce *CognitiveEngine) Start() error {
	if ce.running {
		return fmt.Errorf("cognitive engine is already running")
	}

	log.Println("Starting Cognitive Engine...")

	// Start worker pool for concurrent validation result processing
	ce.workerPool.Start(ce.ctx, ce.processWorkItem)

	// Start eBPF telemetry bridge
	ce.ebpfBridge.Start(ce.ctx, ce.cfg.EBPFTelemetryInterval)

	// Start background loops (timed + event-driven)
	go ce.learningLoop()
	go ce.metricsCollectionLoop()
	go ce.patternAnalysisLoop()
	go ce.guardrailCheckLoop()
	go ce.ontologyUpdateLoop()
	go ce.securityEventHandlerLoop()

	ce.running = true
	ce.startedAt = time.Now()
	log.Println("Cognitive Engine started successfully")
	return nil
}

// Stop gracefully stops the cognitive engine.
func (ce *CognitiveEngine) Stop() error {
	if !ce.running {
		return nil
	}

	log.Println("Stopping Cognitive Engine...")
	ce.cancel()
	ce.running = false

	ce.ebpfBridge.Stop()
	ce.workerPool.Stop()
	ce.eventBus.Close()

	// Save final state
	ce.saveLearningState()

	log.Println("Cognitive Engine stopped")
	return nil
}

// ─── Public API ──────────────────────────────────────────────────────────────

// ProcessValidationResult enqueues a validation result for learning.
// The actual state update is performed by a worker goroutine via the pool.
// Falls back to synchronous processing when the pool is full.
func (ce *CognitiveEngine) ProcessValidationResult(result *objects.ValidationResult, task *objects.ValidationTask) {
	item := ValidationWorkItem{Result: result, Task: task}

	// Publish event so event-driven subscribers can react immediately
	ce.eventBus.Publish(EngineEvent{
		Type:      EventValidationResult,
		Source:    "validation_core",
		Payload:   item,
		Timestamp: time.Now(),
	})

	if !ce.workerPool.Submit(item) {
		// Pool queue full – process inline to avoid dropping data
		ce.processWorkItem(item)
	}
}

// processWorkItem is the worker function executed by each pool goroutine.
// It acquires ce.mu for state mutation, then releases before saving.
func (ce *CognitiveEngine) processWorkItem(item ValidationWorkItem) {
	ce.mu.Lock()
	ce.updateTaskMetrics(item.Task, item.Result)
	ce.updateNodeMetrics(item.Result)
	ce.updateOverallMetricsLocked()   // no internal lock – caller holds write lock
	ce.analyzeFailurePatterns(item.Result, item.Task)
	ce.evaluateAdaptationsLocked()    // no internal lock – caller holds write lock
	ce.updateLearningProgressLocked() // no internal lock – caller holds write lock
	shouldSave := ce.shouldSaveState()
	ce.mu.Unlock()

	// saveLearningState takes its own read lock – must be called after Unlock
	if shouldSave {
		ce.saveLearningState()
	}
}

// GetCognitiveMetrics returns current cognitive engine metrics for a node.
func (ce *CognitiveEngine) GetCognitiveMetrics(nodeID string) *CognitiveMetrics {
	ce.metricsCollector.mu.RLock()
	defer ce.metricsCollector.mu.RUnlock()

	if metrics, exists := ce.metricsCollector.metrics[nodeID]; exists {
		return metrics
	}

	return &CognitiveMetrics{
		NodeID:           nodeID,
		TasksProcessed:   0,
		SuccessRate:      0.0,
		AdaptationScore:  0.0,
		LearningProgress: 0.0,
		Timestamp:        time.Now(),
	}
}

// GetLearningState returns a snapshot copy of the current learning state.
func (ce *CognitiveEngine) GetLearningState() *LearningState {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	stateCopy := *ce.learningState
	return &stateCopy
}

// GetLearningStateRaw returns key learning metrics as primitives
// (satisfies web.CognitiveEngineReader).
func (ce *CognitiveEngine) GetLearningStateRaw() (totalTasks int64, successRate float64, learningProgress float64, confidenceLevel float64) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.learningState.TotalTasksProcessed,
		ce.learningState.SuccessRate,
		ce.learningState.LearningProgress,
		ce.learningState.ConfidenceLevel
}

// IsRunning reports whether the cognitive engine is currently active.
func (ce *CognitiveEngine) IsRunning() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.running
}

// StartedAt returns the time the engine was last started, or zero if never started.
func (ce *CognitiveEngine) StartedAt() time.Time {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.startedAt
}

// GetAverageProcessingTime returns the current rolling average task processing time in seconds.
func (ce *CognitiveEngine) GetAverageProcessingTime() float64 {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.learningState.AverageProcessingTime
}

// GetAdaptationHistory returns the most recent `limit` adaptation events.
func (ce *CognitiveEngine) GetAdaptationHistory(limit int) []AdaptationEvent {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	history := ce.learningState.AdaptationHistory
	if len(history) <= limit {
		return history
	}
	return history[len(history)-limit:]
}

// GetGuardrailViolations returns recent policy violations from the guardrail engine.
func (ce *CognitiveEngine) GetGuardrailViolations(limit int) []PolicyViolation {
	return ce.guardrailEngine.GetViolations(limit)
}

// EventBus returns the internal event bus for subscribing to engine events.
func (ce *CognitiveEngine) EventBus() *EventBus {
	return ce.eventBus
}

// ── runtime.CognitiveEngineInterface implementation ──────────────────────────
// These two methods bridge the inner (per-DVE) agent runtime telemetry into the
// outer (server-wide) cognitive engine learning and guardrail subsystems.
// The outer CognitiveEngine satisfies runtime.CognitiveEngineInterface structurally
// (Go duck-typing), so no import of the runtime package is needed here.

// OnAgentTaskComplexity records the complexity estimate for a DVE's oh-my-pi
// agent container.  High-complexity tasks trigger the adaptation loop so that
// the outer engine can pre-allocate resources or re-balance the network.
func (ce *CognitiveEngine) OnAgentTaskComplexity(dveID string, complexity int) error {
	ce.agentMetricsMu.Lock()
	m, ok := ce.agentMetrics[dveID]
	if !ok {
		m = &AgentDVEMetrics{DVEID: dveID}
		ce.agentMetrics[dveID] = m
	}
	m.Complexity = complexity
	m.LastUpdated = time.Now()
	ce.agentMetricsMu.Unlock()

	// Reflect in the outer learning state's node performance map.
	ce.mu.Lock()
	if ce.learningState.NodePerformance == nil {
		ce.learningState.NodePerformance = make(map[string]*NodeMetrics)
	}
	nm, exists := ce.learningState.NodePerformance[dveID]
	if !exists {
		nm = &NodeMetrics{
			NodeID:          dveID,
			Specializations: []string{"agent"},
		}
		ce.learningState.NodePerformance[dveID] = nm
	}
	nm.TasksProcessed++
	nm.LastActive = time.Now()
	ce.mu.Unlock()

	// Publish an internal event so subsystems (e.g. guardrail loop) can react.
	if ce.eventBus != nil {
		ce.eventBus.Publish(EngineEvent{
			Type:      EventAdaptationRequired,
			Source:    dveID,
			Payload:   complexity,
			Timestamp: time.Now(),
		})
	}

	log.Printf("[CognitiveEngine] DVE %s agent task complexity: %d", dveID, complexity)
	return nil
}

// OnAgentResourceUsage records real-time CPU and memory consumption for a DVE's
// agent container.  The measurements are forwarded to the metrics collector and
// evaluated against guardrail policies so the outer engine can trigger automated
// remediation (e.g. resource scaling) if thresholds are breached.
func (ce *CognitiveEngine) OnAgentResourceUsage(dveID string, cpuPercent float64, memoryMB int64) error {
	ce.agentMetricsMu.Lock()
	m, ok := ce.agentMetrics[dveID]
	if !ok {
		m = &AgentDVEMetrics{DVEID: dveID}
		ce.agentMetrics[dveID] = m
	}
	m.CPUPercent = cpuPercent
	m.MemoryMB = memoryMB
	m.LastUpdated = time.Now()
	ce.agentMetricsMu.Unlock()

	// Update the outer metrics collector so dashboard queries reflect agent load.
	// utilization is a 0-1 score combining CPU and memory pressure.
	utilization := cpuPercent/100.0 + float64(memoryMB)/65536.0
	if utilization > 1.0 {
		utilization = 1.0
	}
	ce.metricsCollector.mu.Lock()
	cm, ok := ce.metricsCollector.metrics[dveID]
	if !ok {
		cm = &CognitiveMetrics{NodeID: dveID}
		ce.metricsCollector.metrics[dveID] = cm
	}
	cm.ResourceUtilization = utilization
	cm.Timestamp = time.Now()
	ce.metricsCollector.mu.Unlock()

	// Run guardrail evaluation with the resource_utilization metric.
	if ce.guardrailEngine != nil {
		ce.guardrailEngine.Evaluate(
			context.Background(),
			dveID, dveID,
			map[string]float64{"resource_utilization": utilization},
		)
	}

	// Publish a resource-pressure event for event-driven subscribers.
	if ce.eventBus != nil {
		ce.eventBus.Publish(EngineEvent{
			Type:      EventResourcePressure,
			Source:    dveID,
			Payload:   map[string]float64{"cpu_percent": cpuPercent, "memory_mb": float64(memoryMB)},
			Timestamp: time.Now(),
		})
	}

	log.Printf("[CognitiveEngine] DVE %s agent resources: CPU=%.1f%% Mem=%dMB", dveID, cpuPercent, memoryMB)
	return nil
}

// GetOntologyStats returns the count of entities and relations currently indexed.
func (ce *CognitiveEngine) GetOntologyStats() (entities int, relations int) {
	return ce.ontologyManager.Stats()
}

// InjectEBPFSecurityFeedback feeds a kernel-level security event into the
// cognitive engine's learning loop and guardrail evaluation.
func (ce *CognitiveEngine) InjectEBPFSecurityFeedback(event SecurityEventFeedback) {
	ce.ebpfBridge.InjectSecurityFeedback(event)
}

// GetLatestTelemetry returns the most recent eBPF/runtime resource snapshot.
func (ce *CognitiveEngine) GetLatestTelemetry() *SystemResourceSnapshot {
	return ce.ebpfBridge.LatestTelemetry()
}

// ─── Background loops ────────────────────────────────────────────────────────

// learningLoop runs the timed learning cycle at the configured interval.
// It also reacts to EventAdaptationRequired events published on the bus.
func (ce *CognitiveEngine) learningLoop() {
	ticker := time.NewTicker(ce.cfg.LearningInterval)
	defer ticker.Stop()

	adaptCh := ce.eventBus.Subscribe(EventAdaptationRequired)
	highFailCh := ce.eventBus.Subscribe(EventHighFailureRate)

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.performLearningCycle()
		case <-adaptCh:
			// Event-driven trigger: kick off an immediate learning cycle
			ce.performLearningCycle()
		case <-highFailCh:
			// High failure rate detected – run learning immediately
			ce.performLearningCycle()
		}
	}
}

// metricsCollectionLoop collects and updates metrics at the configured interval.
func (ce *CognitiveEngine) metricsCollectionLoop() {
	ticker := time.NewTicker(ce.cfg.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.collectMetrics()
		}
	}
}

// patternAnalysisLoop analyses failure patterns at the configured interval.
func (ce *CognitiveEngine) patternAnalysisLoop() {
	ticker := time.NewTicker(ce.cfg.PatternAnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.analyzePatterns()
		}
	}
}

// guardrailCheckLoop evaluates DVE policy guardrails for all tracked nodes.
func (ce *CognitiveEngine) guardrailCheckLoop() {
	ticker := time.NewTicker(ce.cfg.GuardrailCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.evaluateGuardrails()
		}
	}
}

// ontologyUpdateLoop periodically indexes the learning state into the ontology / KNIRVGRAPH.
func (ce *CognitiveEngine) ontologyUpdateLoop() {
	ticker := time.NewTicker(ce.cfg.OntologyUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.indexOntology()
		}
	}
}

// securityEventHandlerLoop consumes eBPF security alert events and incorporates
// them into the learning state as negative signals.
func (ce *CognitiveEngine) securityEventHandlerLoop() {
	secCh := ce.eventBus.Subscribe(EventEBPFSecurityAlert)
	resPressureCh := ce.eventBus.Subscribe(EventResourcePressure)

	for {
		select {
		case <-ce.ctx.Done():
			return
		case event, ok := <-secCh:
			if !ok {
				return
			}
			ce.handleSecurityEvent(event)
		case event, ok := <-resPressureCh:
			if !ok {
				return
			}
			ce.handleResourcePressureEvent(event)
		}
	}
}

// ─── Internal implementation ─────────────────────────────────────────────────

// performLearningCycle executes one cycle of the learning algorithm.
func (ce *CognitiveEngine) performLearningCycle() {
	if ce.validationCore == nil {
		return
	}
	results, err := ce.validationCore.GetValidationResults(100)
	if err != nil {
		log.Printf("CognitiveEngine: failed to fetch validation results: %v", err)
		return
	}

	for _, result := range results {
		tasks, err := ce.validationCore.GetValidationTasks(nil)
		if err != nil || len(tasks) == 0 {
			continue
		}
		task := tasks[0]
		ce.workerPool.Submit(ValidationWorkItem{Result: result, Task: task})
	}

	ce.performPeriodicAdaptations()
}

// collectMetrics collects current system metrics for each tracked node.
func (ce *CognitiveEngine) collectMetrics() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	for nodeID, nodeMetrics := range ce.learningState.NodePerformance {
		metrics := &CognitiveMetrics{
			NodeID:                nodeID,
			TasksProcessed:        nodeMetrics.TasksProcessed,
			AverageProcessingTime: nodeMetrics.AvgProcessingTime,
			SuccessRate:           nodeMetrics.SuccessRate,
			AdaptationScore:       ce.calculateAdaptationScore(nodeID),
			LearningProgress:      ce.learningState.LearningProgress,
			ResourceUtilization:   ce.calculateResourceUtilization(nodeID),
			Timestamp:             time.Now(),
		}

		ce.metricsCollector.mu.Lock()
		ce.metricsCollector.metrics[nodeID] = metrics
		ce.metricsCollector.mu.Unlock()
	}
}

// analyzePatterns performs pattern analysis on recent data.
func (ce *CognitiveEngine) analyzePatterns() {
	ce.patternAnalyzer.mu.Lock()
	defer ce.patternAnalyzer.mu.Unlock()

	if ce.validationCore == nil {
		return
	}
	results, err := ce.validationCore.GetValidationResults(1000)
	if err != nil {
		log.Printf("CognitiveEngine: failed to fetch results for pattern analysis: %v", err)
		return
	}

	failureGroups := make(map[string][]*objects.ValidationResult)
	for _, result := range results {
		if result.Status == "failure" || result.Status == "error" {
			key := ce.generateFailureKey(result)
			failureGroups[key] = append(failureGroups[key], result)
		}
	}

	for key, failures := range failureGroups {
		if len(failures) >= 5 {
			pattern := ce.createFailurePattern(key, failures)
			ce.patternAnalyzer.patterns[key] = pattern

			ce.eventBus.Publish(EngineEvent{
				Type:      EventPatternDetected,
				Source:    "pattern_analyzer",
				Payload:   pattern,
				Timestamp: time.Now(),
			})
		}
	}
}

// evaluateGuardrails checks all tracked nodes against policy rules and triggers
// panic isolation via the eBPF bridge when the violation count exceeds the
// configured threshold.
func (ce *CognitiveEngine) evaluateGuardrails() {
	ce.mu.RLock()
	nodes := make(map[string]*NodeMetrics, len(ce.learningState.NodePerformance))
	for k, v := range ce.learningState.NodePerformance {
		nodes[k] = v
	}
	ce.mu.RUnlock()

	for nodeID, nm := range nodes {
		if ce.ebpfBridge.IsNodeIsolated(nodeID) {
			continue // already isolated – skip
		}

		metrics := map[string]float64{
			"success_rate":        nm.SuccessRate,
			"avg_processing_time": nm.AvgProcessingTime,
			"resource_utilization": ce.calculateResourceUtilization(nodeID),
			"violation_count":     float64(ce.guardrailEngine.ViolationCountForNode(nodeID)),
		}

		violations := ce.guardrailEngine.Evaluate(ce.ctx, nodeID, "" /* dveID unknown */, metrics)

		for _, v := range violations {
			if v.Severity == "panic" {
				if err := ce.ebpfBridge.TriggerPanicIsolation(nodeID); err != nil {
					log.Printf("CognitiveEngine: panic isolation error for %s: %v", nodeID, err)
				}
			}
		}

		// Feed observed values back to the guardrail engine for threshold refinement
		observed := []float64{nm.SuccessRate}
		ce.guardrailEngine.RefinePolicy("dveguard_low_success", observed)
	}
}

// indexOntology pushes the current learning state into the ontology manager
// and from there into the KNIRVGRAPH temporal hypergraph.
func (ce *CognitiveEngine) indexOntology() {
	ce.mu.RLock()
	stateCopy := *ce.learningState
	ce.mu.RUnlock()

	ce.ontologyManager.IndexLearningState(&stateCopy)

	ce.patternAnalyzer.mu.RLock()
	patterns := make(map[string]*FailurePattern, len(ce.patternAnalyzer.patterns))
	for k, v := range ce.patternAnalyzer.patterns {
		patterns[k] = v
	}
	ce.patternAnalyzer.mu.RUnlock()

	ce.ontologyManager.IndexFailurePatterns(patterns)
}

// handleSecurityEvent processes an eBPF security alert and records it as a
// failure signal so the learning loop can factor kernel denials into adaptation
// decisions.
func (ce *CognitiveEngine) handleSecurityEvent(event EngineEvent) {
	feedback, ok := event.Payload.(SecurityEventFeedback)
	if !ok {
		return
	}

	ce.mu.Lock()
	defer ce.mu.Unlock()

	if feedback.NodeID == "" {
		return
	}

	nm, exists := ce.learningState.NodePerformance[feedback.NodeID]
	if !exists {
		nm = &NodeMetrics{NodeID: feedback.NodeID, LastActive: time.Now()}
		ce.learningState.NodePerformance[feedback.NodeID] = nm
	}

	// Treat a critical kernel security event as a synthetic failure
	if feedback.Severity == "critical" || feedback.Severity == "warning" {
		nm.SuccessRate = ce.exponentialMovingAverage(nm.SuccessRate, 0.0, 0.15)
		nm.ReliabilityScore = ce.calculateReliabilityScore(nm)

		// If high failure rate now, emit event to kick off learning immediately
		if nm.SuccessRate < 0.5 {
			ce.eventBus.Publish(EngineEvent{
				Type:      EventHighFailureRate,
				Source:    "security_event_handler",
				Payload:   feedback.NodeID,
				Timestamp: time.Now(),
			})
		}
	}

	log.Printf("CognitiveEngine: absorbed security event from node %s (type=%s severity=%s)",
		feedback.NodeID, feedback.EventType, feedback.Severity)
}

// handleResourcePressureEvent reacts to resource pressure by potentially
// triggering an adaptation cycle.
func (ce *CognitiveEngine) handleResourcePressureEvent(event EngineEvent) {
	pressures, ok := event.Payload.(map[string]float64)
	if !ok {
		return
	}

	cpuP := pressures["cpu_pressure"]
	memP := pressures["mem_pressure"]

	if cpuP > 0.9 || memP > 0.9 {
		log.Printf("CognitiveEngine: critical resource pressure (cpu=%.2f mem=%.2f) – triggering adaptation", cpuP, memP)
		ce.eventBus.Publish(EngineEvent{
			Type:      EventAdaptationRequired,
			Source:    "resource_pressure_handler",
			Payload:   pressures,
			Timestamp: time.Now(),
		})
	}
}

// ─── Learning state helpers ──────────────────────────────────────────────────

func (ce *CognitiveEngine) updateTaskMetrics(task *objects.ValidationTask, result *objects.ValidationResult) {
	taskType := task.Type
	if taskType == "" {
		taskType = "unknown"
	}

	metrics, exists := ce.learningState.TaskTypePerformance[taskType]
	if !exists {
		metrics = &TaskMetrics{
			TaskType:      taskType,
			LastProcessed: time.Now(),
		}
		ce.learningState.TaskTypePerformance[taskType] = metrics
	}

	metrics.TasksProcessed++
	metrics.LastProcessed = time.Now()

	success := 0.0
	if result.Status == "success" || result.Score >= 0.8 {
		success = 1.0
	}
	metrics.SuccessRate = ce.exponentialMovingAverage(metrics.SuccessRate, success, 0.1)

	if result.ExecutionTime > 0 {
		processingTime := result.ExecutionTime.Seconds()
		metrics.AvgProcessingTime = ce.exponentialMovingAverage(metrics.AvgProcessingTime, processingTime, 0.1)
	}

	metrics.AvgScore = ce.exponentialMovingAverage(metrics.AvgScore, result.Score, 0.1)

	if result.Status == "failure" || result.Score < 0.6 {
		pattern := ce.extractFailurePattern(result, task)
		metrics.FailurePatterns = append(metrics.FailurePatterns, pattern)
		if len(metrics.FailurePatterns) > 10 {
			metrics.FailurePatterns = metrics.FailurePatterns[1:]
		}

		// Emit event so the learning loop can be triggered immediately
		if metrics.SuccessRate < 0.5 {
			ce.eventBus.Publish(EngineEvent{
				Type:      EventHighFailureRate,
				Source:    "task_metrics_updater",
				Payload:   taskType,
				Timestamp: time.Now(),
			})
		}
	}
}

func (ce *CognitiveEngine) updateNodeMetrics(result *objects.ValidationResult) {
	nodeID := result.ValidatorNodeID
	if nodeID == "" {
		nodeID = "unknown"
	}

	metrics, exists := ce.learningState.NodePerformance[nodeID]
	if !exists {
		metrics = &NodeMetrics{
			NodeID:     nodeID,
			LastActive: time.Now(),
		}
		ce.learningState.NodePerformance[nodeID] = metrics
	}

	metrics.TasksProcessed++
	metrics.LastActive = time.Now()

	success := 0.0
	if result.Status == "success" || result.Score >= 0.8 {
		success = 1.0
	}
	metrics.SuccessRate = ce.exponentialMovingAverage(metrics.SuccessRate, success, 0.1)

	if result.ExecutionTime > 0 {
		processingTime := result.ExecutionTime.Seconds()
		metrics.AvgProcessingTime = ce.exponentialMovingAverage(metrics.AvgProcessingTime, processingTime, 0.1)
	}

	metrics.ReliabilityScore = ce.calculateReliabilityScore(metrics)

	// Emit node-overload event when the node has processed many tasks and is struggling
	if metrics.TasksProcessed > 100 && metrics.SuccessRate < 0.6 {
		ce.eventBus.Publish(EngineEvent{
			Type:      EventNodeOverload,
			Source:    "node_metrics_updater",
			Payload:   nodeID,
			Timestamp: time.Now(),
		})
	}
}

// updateOverallMetricsLocked updates aggregate metrics.
// Requires ce.mu write lock held by the caller.
func (ce *CognitiveEngine) updateOverallMetricsLocked() {
	ce.learningState.TotalTasksProcessed++

	totalSuccess := 0.0
	totalTasks := 0
	for _, taskMetrics := range ce.learningState.TaskTypePerformance {
		totalSuccess += taskMetrics.SuccessRate * float64(taskMetrics.TasksProcessed)
		totalTasks += int(taskMetrics.TasksProcessed)
	}
	if totalTasks > 0 {
		ce.learningState.SuccessRate = totalSuccess / float64(totalTasks)
	}

	totalTime := 0.0
	timeTasks := 0
	for _, taskMetrics := range ce.learningState.TaskTypePerformance {
		if taskMetrics.AvgProcessingTime > 0 {
			totalTime += taskMetrics.AvgProcessingTime * float64(taskMetrics.TasksProcessed)
			timeTasks += int(taskMetrics.TasksProcessed)
		}
	}
	if timeTasks > 0 {
		ce.learningState.AverageProcessingTime = totalTime / float64(timeTasks)
	}

	ce.learningState.LastUpdated = time.Now()
}

// updateOverallMetrics is the externally safe wrapper (acquires the lock).
func (ce *CognitiveEngine) updateOverallMetrics() {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.updateOverallMetricsLocked()
}

func (ce *CognitiveEngine) analyzeFailurePatterns(result *objects.ValidationResult, task *objects.ValidationTask) {
	if result.Status == "success" && result.Score >= 0.8 {
		return
	}
	pattern := ce.extractFailurePattern(result, task)
	log.Printf("CognitiveEngine: failure pattern – %s (task=%s)", pattern, task.ID)
}

// evaluateAdaptationsLocked checks rules and applies them.
// Requires ce.mu write lock held by the caller.
func (ce *CognitiveEngine) evaluateAdaptationsLocked() {
	for _, rule := range ce.adaptationEngine.adaptationRules {
		if ce.shouldApplyRuleLocked(rule) {
			ce.applyAdaptationRuleLocked(rule)
		}
	}
}

// evaluateAdaptations is the externally safe wrapper.
func (ce *CognitiveEngine) evaluateAdaptations() {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.evaluateAdaptationsLocked()
}

func (ce *CognitiveEngine) performPeriodicAdaptations() {
	now := time.Now()
	if now.Sub(ce.getLastAdaptationTime()) > ce.cfg.AdaptationMinInterval {
		ce.performLoadBalancingAdaptation()
		ce.performResourceOptimizationAdaptation()
	}
}

// updateLearningProgressLocked updates learning progress metrics.
// Requires ce.mu write lock held by the caller.
func (ce *CognitiveEngine) updateLearningProgressLocked() {
	baseProgress := 0.1

	if ce.learningState.SuccessRate > 0.7 {
		baseProgress += 0.3
	} else if ce.learningState.SuccessRate > 0.5 {
		baseProgress += 0.2
	}

	if ce.learningState.AverageProcessingTime < 30.0 {
		baseProgress += 0.2
	} else if ce.learningState.AverageProcessingTime < 60.0 {
		baseProgress += 0.1
	}

	// calculateAdaptationSuccessRateLocked – caller holds the lock already
	adaptationSuccess := ce.calculateAdaptationSuccessRateLocked()
	baseProgress += adaptationSuccess * 0.2

	// patternAnalyzer has its own mutex – safe to read without ce.mu
	ce.patternAnalyzer.mu.RLock()
	patternsRecognized := len(ce.patternAnalyzer.patterns)
	ce.patternAnalyzer.mu.RUnlock()

	if patternsRecognized > 5 {
		baseProgress += 0.2
	} else if patternsRecognized > 2 {
		baseProgress += 0.1
	}

	ce.learningState.LearningProgress = math.Min(1.0,
		ce.exponentialMovingAverage(ce.learningState.LearningProgress, baseProgress, 0.05))

	dataPoints := ce.learningState.TotalTasksProcessed
	if dataPoints > 1000 {
		ce.learningState.ConfidenceLevel = 0.9
	} else if dataPoints > 100 {
		ce.learningState.ConfidenceLevel = 0.7
	} else {
		ce.learningState.ConfidenceLevel = 0.5
	}
}

// updateLearningProgress is the externally safe wrapper (acquires the lock).
func (ce *CognitiveEngine) updateLearningProgress() {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.updateLearningProgressLocked()
}

func (ce *CognitiveEngine) shouldSaveState() bool {
	return time.Since(ce.learningState.LastUpdated) > 5*time.Minute
}

func (ce *CognitiveEngine) initializeAdaptationRules() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	rules := []AdaptationRule{
		{
			ID:        "high_failure_rate",
			Condition: "task_type_success_rate < 0.6",
			Action:    "increase_priority",
			Parameters: map[string]interface{}{
				"priority_boost": 2,
			},
			Priority: 1,
		},
		{
			ID:        "slow_processing",
			Condition: "avg_processing_time > 120",
			Action:    "optimize_resources",
			Parameters: map[string]interface{}{
				"cpu_boost": 0.2,
			},
			Priority: 2,
		},
		{
			ID:       "node_overload",
			Condition: "node_tasks_per_hour > 100",
			Action:   "redistribute_load",
			Priority: 3,
		},
	}

	ce.adaptationEngine.adaptationRules = rules
}

func (ce *CognitiveEngine) loadLearningState() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if err := ce.db.GetJSON("cognitive:learning_state", ce.learningState); err != nil {
		log.Printf("CognitiveEngine: failed to load learning state, starting fresh: %v", err)
	}
}

func (ce *CognitiveEngine) saveLearningState() {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	ce.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(ce.learningState)
		if err != nil {
			return err
		}
		_, _, err = tx.Set("cognitive:learning_state", string(data), nil)
		return err
	})
}

// ─── Calculation helpers ─────────────────────────────────────────────────────

func (ce *CognitiveEngine) exponentialMovingAverage(current, new float64, alpha float64) float64 {
	if current == 0 {
		return new
	}
	return alpha*new + (1-alpha)*current
}

func (ce *CognitiveEngine) calculateAdaptationScore(_ string) float64 {
	adaptations := 0
	successfulAdaptations := 0
	for _, event := range ce.learningState.AdaptationHistory {
		if event.ActualImpact != nil && event.ActualImpact.OverallImprovement > 0 {
			successfulAdaptations++
		}
		adaptations++
	}
	if adaptations == 0 {
		return 0.5
	}
	return float64(successfulAdaptations) / float64(adaptations)
}

// calculateResourceUtilization returns a 0.0–1.0 score derived from:
//   - Real eBPF telemetry (when available) via the ResourceTelemetryCollector
//   - Node task-processing statistics as a fallback
func (ce *CognitiveEngine) calculateResourceUtilization(nodeID string) float64 {
	nodeMetrics, exists := ce.learningState.NodePerformance[nodeID]
	if !exists {
		return 0.0
	}

	// Use real telemetry if the eBPF bridge has collected data
	snap := ce.ebpfBridge.LatestTelemetry()
	if snap.TotalCPUTimeNs > 0 || snap.HeapAllocBytes > 0 {
		return ce.ebpfBridge.telemetry.ResourceUtilizationForNode(
			nodeMetrics.TasksProcessed,
			nodeMetrics.SuccessRate,
		)
	}

	// Fallback: task-rate heuristic
	base := math.Min(1.0, float64(nodeMetrics.TasksProcessed)/100.0)
	return base * nodeMetrics.SuccessRate
}

func (ce *CognitiveEngine) calculateReliabilityScore(metrics *NodeMetrics) float64 {
	if metrics.TasksProcessed < 10 {
		return 0.5
	}
	consistency := 1.0 - math.Abs(metrics.SuccessRate-0.8)
	if consistency < 0 {
		consistency = 0
	}
	return math.Min(1.0, consistency)
}

func (ce *CognitiveEngine) generateFailureKey(result *objects.ValidationResult) string {
	key := fmt.Sprintf("%s_%.2f", result.Status, result.Score)
	if result.ErrorMessage != "" {
		if len(result.ErrorMessage) > 50 {
			key += "_" + result.ErrorMessage[:50]
		} else {
			key += "_" + result.ErrorMessage
		}
	}
	return key
}

func (ce *CognitiveEngine) createFailurePattern(key string, failures []*objects.ValidationResult) *FailurePattern {
	avgImpact := 0.0
	for _, failure := range failures {
		avgImpact += (1.0 - failure.Score)
	}
	avgImpact /= float64(len(failures))

	return &FailurePattern{
		PatternID:       key,
		Description:     fmt.Sprintf("Recurring failure pattern: %s", key),
		Frequency:       len(failures),
		AvgImpact:       avgImpact,
		SuggestedAction: ce.suggestActionForPattern(key),
		LastSeen:        time.Now(),
	}
}

func (ce *CognitiveEngine) suggestActionForPattern(patternKey string) string {
	if contains(patternKey, "timeout") {
		return "Increase timeout limits or optimize processing"
	}
	if contains(patternKey, "memory") {
		return "Increase memory allocation or optimize memory usage"
	}
	if contains(patternKey, "validation") {
		return "Review validation criteria or adjust thresholds"
	}
	return "Investigate and optimize based on error details"
}

func (ce *CognitiveEngine) extractFailurePattern(result *objects.ValidationResult, task *objects.ValidationTask) string {
	pattern := fmt.Sprintf("TaskType:%s_Status:%s_Score:%.2f", task.Type, result.Status, result.Score)
	if result.ErrorMessage != "" {
		pattern += fmt.Sprintf("_Error:%s", result.ErrorMessage)
	}
	return pattern
}

// shouldApplyRuleLocked checks whether a rule's conditions are met.
// Requires ce.mu held (read or write) by the caller.
func (ce *CognitiveEngine) shouldApplyRuleLocked(rule AdaptationRule) bool {
	switch rule.Condition {
	case "task_type_success_rate < 0.6":
		for _, metrics := range ce.learningState.TaskTypePerformance {
			if metrics.SuccessRate < 0.6 && time.Since(rule.LastApplied) > time.Hour {
				return true
			}
		}
	case "avg_processing_time > 120":
		if ce.learningState.AverageProcessingTime > 120 && time.Since(rule.LastApplied) > time.Hour {
			return true
		}
	}
	return false
}

// shouldApplyRule is the externally safe wrapper.
func (ce *CognitiveEngine) shouldApplyRule(rule AdaptationRule) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.shouldApplyRuleLocked(rule)
}

// applyAdaptationRuleLocked applies a rule and records the event.
// Requires ce.mu write lock held by the caller.
func (ce *CognitiveEngine) applyAdaptationRuleLocked(rule AdaptationRule) {
	log.Printf("CognitiveEngine: applying adaptation rule %s", rule.ID)

	event := AdaptationEvent{
		ID:             fmt.Sprintf("adaptation_%d", time.Now().Unix()),
		Timestamp:      time.Now(),
		TriggerReason:  rule.Condition,
		AdaptationType: rule.Action,
		Changes:        rule.Parameters,
		ExpectedImpact: fmt.Sprintf("Expected improvement from rule: %s", rule.ID),
	}

	switch rule.Action {
	case "increase_priority":
		ce.adaptTaskPriorities(rule.Parameters)
	case "optimize_resources":
		ce.adaptResourceAllocation(rule.Parameters)
	case "redistribute_load":
		ce.adaptLoadDistribution()
	}

	for i, r := range ce.adaptationEngine.adaptationRules {
		if r.ID == rule.ID {
			ce.adaptationEngine.adaptationRules[i].LastApplied = time.Now()
			break
		}
	}

	ce.learningState.AdaptationHistory = append(ce.learningState.AdaptationHistory, event)
	if len(ce.learningState.AdaptationHistory) > 100 {
		ce.learningState.AdaptationHistory = ce.learningState.AdaptationHistory[1:]
	}
}

func (ce *CognitiveEngine) adaptTaskPriorities(params map[string]interface{}) {
	boost, _ := params["priority_boost"].(int)
	if boost <= 0 {
		boost = 1
	}
	for taskType, metrics := range ce.learningState.TaskTypePerformance {
		if metrics.SuccessRate < 0.6 {
			log.Printf("CognitiveEngine: increasing priority for task type %s (boost=%d)", taskType, boost)
		}
	}
}

func (ce *CognitiveEngine) adaptResourceAllocation(params map[string]interface{}) {
	boost, _ := params["cpu_boost"].(float64)
	snap := ce.ebpfBridge.LatestTelemetry()
	log.Printf("CognitiveEngine: resource allocation adaptation – cpu_boost=%.2f current_pressure=(cpu=%.2f mem=%.2f)",
		boost, snap.CPUPressure, snap.MemoryPressure)
	// Production hook: adjust GOMAXPROCS or notify an orchestrator
}

func (ce *CognitiveEngine) adaptLoadDistribution() {
	log.Println("CognitiveEngine: redistributing load across nodes")
}

func (ce *CognitiveEngine) performLoadBalancingAdaptation() {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	nodePerformances := make([]NodeMetrics, 0, len(ce.learningState.NodePerformance))
	for _, metrics := range ce.learningState.NodePerformance {
		nodePerformances = append(nodePerformances, *metrics)
	}
	sort.Slice(nodePerformances, func(i, j int) bool {
		return ce.calculateResourceUtilization(nodePerformances[i].NodeID) <
			ce.calculateResourceUtilization(nodePerformances[j].NodeID)
	})
	log.Println("CognitiveEngine: performed load balancing adaptation")
}

func (ce *CognitiveEngine) performResourceOptimizationAdaptation() {
	snap := ce.ebpfBridge.LatestTelemetry()
	log.Printf("CognitiveEngine: resource optimisation – heap=%dMB goroutines=%d cpu_pressure=%.2f",
		snap.HeapAllocBytes/1024/1024, snap.GoRoutines, snap.CPUPressure)
}

func (ce *CognitiveEngine) getLastAdaptationTime() time.Time {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if len(ce.learningState.AdaptationHistory) == 0 {
		return time.Now().Add(-25 * time.Hour)
	}
	lastEvent := ce.learningState.AdaptationHistory[len(ce.learningState.AdaptationHistory)-1]
	return lastEvent.Timestamp
}

// calculateAdaptationSuccessRateLocked computes the adaptation success rate.
// Requires ce.mu held by the caller.
func (ce *CognitiveEngine) calculateAdaptationSuccessRateLocked() float64 {
	if len(ce.learningState.AdaptationHistory) == 0 {
		return 0.5
	}
	successful := 0
	for _, event := range ce.learningState.AdaptationHistory {
		if event.ActualImpact != nil && event.ActualImpact.OverallImprovement > 0 {
			successful++
		}
	}
	return float64(successful) / float64(len(ce.learningState.AdaptationHistory))
}

// calculateAdaptationSuccessRate is the externally safe wrapper.
func (ce *CognitiveEngine) calculateAdaptationSuccessRate() float64 {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.calculateAdaptationSuccessRateLocked()
}

// applyAdaptationRule is the externally safe wrapper used by existing callers and tests.
func (ce *CognitiveEngine) applyAdaptationRule(rule AdaptationRule) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.applyAdaptationRuleLocked(rule)
}

// InjectOrganizationContext feeds an organization's value system into the
// cognitive engine's learning state as an adaptation event.  This is called
// by the OnboardingService once a ValueSystem + Ontology have been processed
// so that the engine can weight its guardrail evaluations accordingly.
func (ce *CognitiveEngine) InjectOrganizationContext(orgID, orgName string, guidelines, values []string, riskLevel string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Map risk level to a confidence offset so high-risk orgs start with
	// tighter guardrail thresholds.
	riskOffset := 0.0
	switch riskLevel {
	case "low":
		riskOffset = 0.05
	case "high":
		riskOffset = -0.05
	}
	if ce.learningState.ConfidenceLevel+riskOffset >= 0 && ce.learningState.ConfidenceLevel+riskOffset <= 1.0 {
		ce.learningState.ConfidenceLevel += riskOffset
	}

	changes := map[string]interface{}{
		"org_id":     orgID,
		"org_name":   orgName,
		"guidelines": len(guidelines),
		"values":     len(values),
		"risk_level": riskLevel,
	}

	event := AdaptationEvent{
		ID:             fmt.Sprintf("org-context-%s", orgID),
		Timestamp:      time.Now(),
		TriggerReason:  fmt.Sprintf("Organization onboarded: %s", orgName),
		AdaptationType: "organization_context",
		Changes:        changes,
		ExpectedImpact: "Guardrail thresholds aligned to organizational risk appetite",
	}

	ce.learningState.AdaptationHistory = append(ce.learningState.AdaptationHistory, event)
	ce.learningState.LastUpdated = time.Now()

	log.Printf("CognitiveEngine: injected org context for %s (%s) — guidelines=%d values=%d risk=%s",
		orgName, orgID, len(guidelines), len(values), riskLevel)
}

// ─── Utilities ───────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && strings.Contains(s, substr)))
}
