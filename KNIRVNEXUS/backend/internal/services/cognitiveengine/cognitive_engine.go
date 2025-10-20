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

	"backend_server/internal/objects"
	"backend_server/internal/services/validation"

	"github.com/tidwall/buntdb"
)

// CognitiveEngine manages AI-driven learning and adaptation for the DVE network
type CognitiveEngine struct {
	db                    *buntdb.DB
	validationCore        ValidationClient
	inferenceService      InferenceClient
	modelManager          ModelManagerClient
	learningState         *LearningState
	adaptationEngine      *AdaptationEngine
	metricsCollector      *MetricsCollector
	patternAnalyzer       *PatternAnalyzer
	ctx                   context.Context
	cancel                context.CancelFunc
	mu                    sync.RWMutex
	running               bool
}

// LearningState represents the current state of the cognitive engine's learning
type LearningState struct {
	TotalTasksProcessed    int64                    `json:"total_tasks_processed"`
	SuccessRate            float64                  `json:"success_rate"`
	AverageProcessingTime  float64                  `json:"average_processing_time"`
	TaskTypePerformance    map[string]*TaskMetrics `json:"task_type_performance"`
	NodePerformance        map[string]*NodeMetrics `json:"node_performance"`
	AdaptationHistory      []AdaptationEvent       `json:"adaptation_history"`
	LastUpdated            time.Time                `json:"last_updated"`
	LearningProgress       float64                  `json:"learning_progress"`
	ConfidenceLevel        float64                  `json:"confidence_level"`
}

// TaskMetrics tracks performance metrics for different task types
type TaskMetrics struct {
	TaskType         string  `json:"task_type"`
	TasksProcessed   int64   `json:"tasks_processed"`
	SuccessRate      float64 `json:"success_rate"`
	AvgProcessingTime float64 `json:"avg_processing_time"`
	AvgScore         float64 `json:"avg_score"`
	FailurePatterns  []string `json:"failure_patterns"`
	LastProcessed    time.Time `json:"last_processed"`
}

// NodeMetrics tracks performance metrics for different nodes
type NodeMetrics struct {
	NodeID           string  `json:"node_id"`
	TasksProcessed   int64   `json:"tasks_processed"`
	SuccessRate      float64 `json:"success_rate"`
	AvgProcessingTime float64 `json:"avg_processing_time"`
	ReliabilityScore float64 `json:"reliability_score"`
	Specializations  []string `json:"specializations"`
	LastActive       time.Time `json:"last_active"`
}

// AdaptationEvent represents a system adaptation triggered by learning
type AdaptationEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	TriggerReason   string                 `json:"trigger_reason"`
	AdaptationType  string                 `json:"adaptation_type"`
	Changes         map[string]interface{} `json:"changes"`
	ExpectedImpact  string                 `json:"expected_impact"`
	ActualImpact    *AdaptationResult      `json:"actual_impact,omitempty"`
}

// AdaptationResult represents the measured impact of an adaptation
type AdaptationResult struct {
	MeasuredAt      time.Time `json:"measured_at"`
	SuccessRateChange float64  `json:"success_rate_change"`
	ProcessingTimeChange float64 `json:"processing_time_change"`
	OverallImprovement float64 `json:"overall_improvement"`
}

// AdaptationEngine handles system parameter adjustments
type AdaptationEngine struct {
	currentParams   map[string]interface{}
	adaptationRules []AdaptationRule
	mu              sync.RWMutex
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
	// Inference service interface - simplified for now
}

type ModelManagerClient interface {
	// Model management interface - simplified for now
}

// NewCognitiveEngine creates a new cognitive engine instance
func NewCognitiveEngine(db *buntdb.DB, validationClient ValidationClient, inferenceClient InferenceClient, modelManager ModelManagerClient) *CognitiveEngine {
	ctx, cancel := context.WithCancel(context.Background())

	ce := &CognitiveEngine{
		db:               db,
		validationCore:   validationClient,
		inferenceService: inferenceClient,
		modelManager:     modelManager,
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
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}

	ce.initializeAdaptationRules()
	ce.loadLearningState()

	return ce
}

// Start starts the cognitive engine
func (ce *CognitiveEngine) Start() error {
	if ce.running {
		return fmt.Errorf("cognitive engine is already running")
	}

	log.Println("Starting Cognitive Engine...")

	// Start background learning loop
	go ce.learningLoop()
	go ce.metricsCollectionLoop()
	go ce.patternAnalysisLoop()

	ce.running = true
	log.Println("Cognitive Engine started successfully")
	return nil
}

// Stop stops the cognitive engine
func (ce *CognitiveEngine) Stop() error {
	if !ce.running {
		return nil
	}

	log.Println("Stopping Cognitive Engine...")
	ce.cancel()
	ce.running = false

	// Save final state
	ce.saveLearningState()

	log.Println("Cognitive Engine stopped")
	return nil
}

// ProcessValidationResult processes a validation result for learning
func (ce *CognitiveEngine) ProcessValidationResult(result *objects.ValidationResult, task *objects.ValidationTask) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Update learning state
	ce.updateTaskMetrics(task, result)
	ce.updateNodeMetrics(result)
	ce.updateOverallMetrics()

	// Analyze for patterns
	ce.analyzeFailurePatterns(result, task)

	// Check for adaptation opportunities
	ce.evaluateAdaptations()

	// Update learning progress
	ce.updateLearningProgress()

	// Save state periodically
	if ce.shouldSaveState() {
		ce.saveLearningState()
	}
}

// GetCognitiveMetrics returns current cognitive engine metrics
func (ce *CognitiveEngine) GetCognitiveMetrics(nodeID string) *CognitiveMetrics {
	ce.metricsCollector.mu.RLock()
	defer ce.metricsCollector.mu.RUnlock()

	if metrics, exists := ce.metricsCollector.metrics[nodeID]; exists {
		return metrics
	}

	// Return default metrics if none exist
	return &CognitiveMetrics{
		NodeID:          nodeID,
		TasksProcessed:  0,
		SuccessRate:     0.0,
		AdaptationScore: 0.0,
		LearningProgress: 0.0,
		Timestamp:       time.Now(),
	}
}

// GetLearningState returns the current learning state
func (ce *CognitiveEngine) GetLearningState() *LearningState {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Return a copy to prevent external modification
	stateCopy := *ce.learningState
	return &stateCopy
}

// GetAdaptationHistory returns recent adaptation events
func (ce *CognitiveEngine) GetAdaptationHistory(limit int) []AdaptationEvent {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	history := ce.learningState.AdaptationHistory
	if len(history) <= limit {
		return history
	}

	// Return most recent events
	return history[len(history)-limit:]
}

// learningLoop runs the main learning algorithm
func (ce *CognitiveEngine) learningLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.performLearningCycle()
		}
	}
}

// performLearningCycle executes one cycle of the learning algorithm
func (ce *CognitiveEngine) performLearningCycle() {
	// Fetch recent validation results
	results, err := ce.validationCore.GetValidationResults(100)
	if err != nil {
		log.Printf("Failed to fetch validation results for learning: %v", err)
		return
	}

	// Process results for learning
	for _, result := range results {
		// Get associated task
		tasks, err := ce.validationCore.GetValidationTasks(nil)
		if err != nil || len(tasks) == 0 {
			continue
		}

		task := tasks[0]
		ce.ProcessValidationResult(result, task)
	}

	// Perform periodic adaptations
	ce.performPeriodicAdaptations()
}

// metricsCollectionLoop collects and updates metrics
func (ce *CognitiveEngine) metricsCollectionLoop() {
	ticker := time.NewTicker(60 * time.Second)
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

// collectMetrics collects current system metrics
func (ce *CognitiveEngine) collectMetrics() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Collect metrics for each node
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

// patternAnalysisLoop analyzes patterns in validation data
func (ce *CognitiveEngine) patternAnalysisLoop() {
	ticker := time.NewTicker(5 * time.Minute)
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

// analyzePatterns performs pattern analysis on recent data
func (ce *CognitiveEngine) analyzePatterns() {
	ce.patternAnalyzer.mu.Lock()
	defer ce.patternAnalyzer.mu.Unlock()

	// Analyze failure patterns
	results, err := ce.validationCore.GetValidationResults(1000)
	if err != nil {
		log.Printf("Failed to fetch results for pattern analysis: %v", err)
		return
	}

	// Group failures by characteristics
	failureGroups := make(map[string][]*objects.ValidationResult)

	for _, result := range results {
		if result.Status == "failure" || result.Status == "error" {
			key := ce.generateFailureKey(result)
			failureGroups[key] = append(failureGroups[key], result)
		}
	}

	// Identify significant patterns
	for key, failures := range failureGroups {
		if len(failures) >= 5 { // Minimum threshold for pattern recognition
			pattern := ce.createFailurePattern(key, failures)
			ce.patternAnalyzer.patterns[key] = pattern
		}
	}
}

// updateTaskMetrics updates metrics for a specific task type
func (ce *CognitiveEngine) updateTaskMetrics(task *objects.ValidationTask, result *objects.ValidationResult) {
	taskType := task.Type
	if taskType == "" {
		taskType = "unknown"
	}

	metrics, exists := ce.learningState.TaskTypePerformance[taskType]
	if !exists {
		metrics = &TaskMetrics{
			TaskType:       taskType,
			LastProcessed:  time.Now(),
		}
		ce.learningState.TaskTypePerformance[taskType] = metrics
	}

	// Update metrics
	metrics.TasksProcessed++
	metrics.LastProcessed = time.Now()

	// Update success rate using exponential moving average
	success := 0.0
	if result.Status == "success" || result.Score >= 0.8 {
		success = 1.0
	}
	metrics.SuccessRate = ce.exponentialMovingAverage(metrics.SuccessRate, success, 0.1)

	// Update average processing time
	if result.ExecutionTime > 0 {
		processingTime := result.ExecutionTime.Seconds()
		metrics.AvgProcessingTime = ce.exponentialMovingAverage(metrics.AvgProcessingTime, processingTime, 0.1)
	}

	// Update average score
	metrics.AvgScore = ce.exponentialMovingAverage(metrics.AvgScore, result.Score, 0.1)

	// Track failure patterns
	if result.Status == "failure" || result.Score < 0.6 {
		pattern := ce.extractFailurePattern(result, task)
		metrics.FailurePatterns = append(metrics.FailurePatterns, pattern)
		// Keep only recent patterns
		if len(metrics.FailurePatterns) > 10 {
			metrics.FailurePatterns = metrics.FailurePatterns[1:]
		}
	}
}

// updateNodeMetrics updates metrics for a specific node
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

	// Update metrics
	metrics.TasksProcessed++
	metrics.LastActive = time.Now()

	// Update success rate
	success := 0.0
	if result.Status == "success" || result.Score >= 0.8 {
		success = 1.0
	}
	metrics.SuccessRate = ce.exponentialMovingAverage(metrics.SuccessRate, success, 0.1)

	// Update processing time
	if result.ExecutionTime > 0 {
		processingTime := result.ExecutionTime.Seconds()
		metrics.AvgProcessingTime = ce.exponentialMovingAverage(metrics.AvgProcessingTime, processingTime, 0.1)
	}

	// Update reliability score based on consistency
	metrics.ReliabilityScore = ce.calculateReliabilityScore(metrics)
}

// updateOverallMetrics updates overall learning state metrics
func (ce *CognitiveEngine) updateOverallMetrics() {
	ce.learningState.TotalTasksProcessed++

	// Calculate overall success rate
	totalSuccess := 0.0
	totalTasks := 0

	for _, taskMetrics := range ce.learningState.TaskTypePerformance {
		totalSuccess += taskMetrics.SuccessRate * float64(taskMetrics.TasksProcessed)
		totalTasks += int(taskMetrics.TasksProcessed)
	}

	if totalTasks > 0 {
		ce.learningState.SuccessRate = totalSuccess / float64(totalTasks)
	}

	// Calculate average processing time
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

// analyzeFailurePatterns analyzes failure patterns from results
func (ce *CognitiveEngine) analyzeFailurePatterns(result *objects.ValidationResult, task *objects.ValidationTask) {
	if result.Status == "success" && result.Score >= 0.8 {
		return // Only analyze failures
	}

	pattern := ce.extractFailurePattern(result, task)
	log.Printf("Detected failure pattern: %s for task %s", pattern, task.ID)
}

// evaluateAdaptations checks if adaptations should be triggered
func (ce *CognitiveEngine) evaluateAdaptations() {
	// Check adaptation rules
	for _, rule := range ce.adaptationEngine.adaptationRules {
		if ce.shouldApplyRule(rule) {
			ce.applyAdaptationRule(rule)
		}
	}
}

// performPeriodicAdaptations performs scheduled adaptations
func (ce *CognitiveEngine) performPeriodicAdaptations() {
	// Check if it's time for periodic adaptations
	now := time.Now()
	lastAdaptation := ce.getLastAdaptationTime()

	if now.Sub(lastAdaptation) > 24*time.Hour {
		ce.performLoadBalancingAdaptation()
		ce.performResourceOptimizationAdaptation()
	}
}

// updateLearningProgress calculates and updates learning progress
func (ce *CognitiveEngine) updateLearningProgress() {
	// Learning progress based on:
	// 1. Improvement in success rates over time
	// 2. Reduction in processing times
	// 3. Number of successful adaptations
	// 4. Pattern recognition accuracy

	baseProgress := 0.1 // Minimum progress

	// Factor 1: Success rate improvement
	if ce.learningState.SuccessRate > 0.7 {
		baseProgress += 0.3
	} else if ce.learningState.SuccessRate > 0.5 {
		baseProgress += 0.2
	}

	// Factor 2: Processing time optimization
	if ce.learningState.AverageProcessingTime < 30.0 {
		baseProgress += 0.2
	} else if ce.learningState.AverageProcessingTime < 60.0 {
		baseProgress += 0.1
	}

	// Factor 3: Adaptation success
	adaptationSuccess := ce.calculateAdaptationSuccessRate()
	baseProgress += adaptationSuccess * 0.2

	// Factor 4: Pattern recognition
	patternsRecognized := len(ce.patternAnalyzer.patterns)
	if patternsRecognized > 5 {
		baseProgress += 0.2
	} else if patternsRecognized > 2 {
		baseProgress += 0.1
	}

	// Cap at 1.0 and apply smoothing
	ce.learningState.LearningProgress = math.Min(1.0, ce.exponentialMovingAverage(ce.learningState.LearningProgress, baseProgress, 0.05))

	// Update confidence level based on data volume and consistency
	dataPoints := ce.learningState.TotalTasksProcessed
	if dataPoints > 1000 {
		ce.learningState.ConfidenceLevel = 0.9
	} else if dataPoints > 100 {
		ce.learningState.ConfidenceLevel = 0.7
	} else {
		ce.learningState.ConfidenceLevel = 0.5
	}
}

// shouldSaveState determines if the learning state should be saved
func (ce *CognitiveEngine) shouldSaveState() bool {
	return time.Since(ce.learningState.LastUpdated) > 5*time.Minute
}

// initializeAdaptationRules sets up the initial adaptation rules
func (ce *CognitiveEngine) initializeAdaptationRules() {
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
			ID:        "node_overload",
			Condition: "node_tasks_per_hour > 100",
			Action:    "redistribute_load",
			Priority:  3,
		},
	}

	ce.adaptationEngine.adaptationRules = rules
}

// loadLearningState loads the learning state from database
func (ce *CognitiveEngine) loadLearningState() {
	err := ce.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("cognitive:learning_state")
		if err != nil {
			return err
		}

		return json.Unmarshal([]byte(val), ce.learningState)
	})

	if err != nil {
		log.Printf("Failed to load learning state, starting fresh: %v", err)
	}
}

// saveLearningState saves the learning state to database
func (ce *CognitiveEngine) saveLearningState() {
	ce.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(ce.learningState)
		if err != nil {
			return err
		}

		_, _, err = tx.Set("cognitive:learning_state", string(data), nil)
		return err
	})
}

// Helper methods

func (ce *CognitiveEngine) exponentialMovingAverage(current, new float64, alpha float64) float64 {
	if current == 0 {
		return new
	}
	return alpha*new + (1-alpha)*current
}

func (ce *CognitiveEngine) calculateAdaptationScore(_ string) float64 {
	// Calculate how well adaptations have worked overall
	adaptations := 0
	successfulAdaptations := 0

	for _, event := range ce.learningState.AdaptationHistory {
		if event.ActualImpact != nil && event.ActualImpact.OverallImprovement > 0 {
			successfulAdaptations++
		}
		adaptations++
	}

	if adaptations == 0 {
		return 0.5 // Neutral score
	}

	return float64(successfulAdaptations) / float64(adaptations)
}

func (ce *CognitiveEngine) calculateResourceUtilization(nodeID string) float64 {
	// Calculate resource utilization based on task processing patterns
	nodeMetrics, exists := ce.learningState.NodePerformance[nodeID]
	if !exists {
		return 0.0
	}

	// Simple utilization based on tasks processed and success rate
	baseUtilization := math.Min(1.0, float64(nodeMetrics.TasksProcessed)/100.0)
	utilization := baseUtilization * nodeMetrics.SuccessRate

	return utilization
}

func (ce *CognitiveEngine) calculateReliabilityScore(metrics *NodeMetrics) float64 {
	// Calculate reliability based on consistency of performance
	if metrics.TasksProcessed < 10 {
		return 0.5 // Not enough data
	}

	// Higher reliability for consistent performance
	consistency := 1.0 - (metrics.SuccessRate - 0.8) // Penalize deviation from 80%
	if consistency < 0 {
		consistency = 0
	}

	return math.Min(1.0, consistency)
}

func (ce *CognitiveEngine) generateFailureKey(result *objects.ValidationResult) string {
	// Generate a key based on failure characteristics
	key := fmt.Sprintf("%s_%.2f", result.Status, result.Score)
	if result.ErrorMessage != "" {
		// Include error type in key
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
	// Provide suggestions based on pattern characteristics
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

func (ce *CognitiveEngine) shouldApplyRule(rule AdaptationRule) bool {
	// Check if rule conditions are met
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

func (ce *CognitiveEngine) applyAdaptationRule(rule AdaptationRule) {
	log.Printf("Applying adaptation rule: %s", rule.ID)

	event := AdaptationEvent{
		ID:             fmt.Sprintf("adaptation_%d", time.Now().Unix()),
		Timestamp:      time.Now(),
		TriggerReason:  rule.Condition,
		AdaptationType: rule.Action,
		Changes:        rule.Parameters,
		ExpectedImpact: fmt.Sprintf("Expected improvement from rule: %s", rule.ID),
	}

	// Apply the adaptation
	switch rule.Action {
	case "increase_priority":
		ce.adaptTaskPriorities(rule.Parameters)
	case "optimize_resources":
		ce.adaptResourceAllocation(rule.Parameters)
	case "redistribute_load":
		ce.adaptLoadDistribution()
	}

	rule.LastApplied = time.Now()
	ce.learningState.AdaptationHistory = append(ce.learningState.AdaptationHistory, event)

	// Keep only recent history
	if len(ce.learningState.AdaptationHistory) > 100 {
		ce.learningState.AdaptationHistory = ce.learningState.AdaptationHistory[1:]
	}
}

func (ce *CognitiveEngine) adaptTaskPriorities(params map[string]interface{}) {
	boost, _ := params["priority_boost"].(int)
	if boost <= 0 {
		boost = 1
	}

	// Increase priority for underperforming task types
	for taskType, metrics := range ce.learningState.TaskTypePerformance {
		if metrics.SuccessRate < 0.6 {
			log.Printf("Increasing priority for task type: %s", taskType)
			// In a real implementation, this would update the validation core
		}
	}
}

func (ce *CognitiveEngine) adaptResourceAllocation(params map[string]interface{}) {
	boost, _ := params["cpu_boost"].(float64)
	log.Printf("Adapting resource allocation with CPU boost: %.2f", boost)
	// In a real implementation, this would adjust system resources
}

func (ce *CognitiveEngine) adaptLoadDistribution() {
	log.Println("Adapting load distribution across nodes")
	// In a real implementation, this would redistribute tasks
}

func (ce *CognitiveEngine) performLoadBalancingAdaptation() {
	// Analyze node performance and redistribute load
	nodePerformances := make([]NodeMetrics, 0, len(ce.learningState.NodePerformance))
	for _, metrics := range ce.learningState.NodePerformance {
		nodePerformances = append(nodePerformances, *metrics)
	}

	// Sort by utilization
	sort.Slice(nodePerformances, func(i, j int) bool {
		return ce.calculateResourceUtilization(nodePerformances[i].NodeID) <
			ce.calculateResourceUtilization(nodePerformances[j].NodeID)
	})

	// Balance load by adjusting task assignments
	log.Println("Performing load balancing adaptation")
}

func (ce *CognitiveEngine) performResourceOptimizationAdaptation() {
	// Optimize resource allocation based on usage patterns
	log.Println("Performing resource optimization adaptation")
}

func (ce *CognitiveEngine) getLastAdaptationTime() time.Time {
	if len(ce.learningState.AdaptationHistory) == 0 {
		return time.Now().Add(-25 * time.Hour) // Force adaptation if none recent
	}

	lastEvent := ce.learningState.AdaptationHistory[len(ce.learningState.AdaptationHistory)-1]
	return lastEvent.Timestamp
}

func (ce *CognitiveEngine) calculateAdaptationSuccessRate() float64 {
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

// Utility functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}
