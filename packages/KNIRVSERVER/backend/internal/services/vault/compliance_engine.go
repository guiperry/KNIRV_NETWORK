package vault

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"backend_server/internal/fintech/ontology"
	"backend_server/internal/storage/pqc"
)

// ComplianceEngine runs regulatory scenarios and calculates compliance scores
type ComplianceEngine struct {
	repository       *ScenarioRepository
	executor         *ComplianceScriptExecutor
	ontologyRegistry *ontology.OntologyRegistry

	// Configuration
	config ComplianceConfig

	// State
	mu             sync.RWMutex
	executionCache map[string]*ScenarioResult
	statistics     *ComplianceStatistics
}

// ComplianceConfig configures the compliance engine
type ComplianceConfig struct {
	// Execution
	MaxConcurrentExecutions int           `json:"max_concurrent_executions"`
	DefaultTimeout          time.Duration `json:"default_timeout"`
	EnableCaching           bool          `json:"enable_caching"`
	CacheTTL                time.Duration `json:"cache_ttl"`

	// Scoring weights
	GuardrailWeight  float64 `json:"guardrail_weight"`
	ViolationWeight  float64 `json:"violation_weight"`
	MandatoryPenalty float64 `json:"mandatory_penalty"` // Score reduction for mandatory guardrail failure

	// Thresholds
	PassThreshold   float64 `json:"pass_threshold"`   // Minimum score to pass
	ReviewThreshold float64 `json:"review_threshold"` // Minimum score for review (below = fail)
}

// DefaultComplianceConfig returns default configuration
func DefaultComplianceConfig() ComplianceConfig {
	return ComplianceConfig{
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          30 * time.Second,
		EnableCaching:           true,
		CacheTTL:                1 * time.Hour,
		GuardrailWeight:         0.4,
		ViolationWeight:         0.4,
		MandatoryPenalty:        0.2,
		PassThreshold:           80.0,
		ReviewThreshold:         60.0,
	}
}

// ComplianceStatistics tracks execution statistics
type ComplianceStatistics struct {
	TotalExecutions      int64            `json:"total_executions"`
	PassedExecutions     int64            `json:"passed_executions"`
	FailedExecutions     int64            `json:"failed_executions"`
	ErrorExecutions      int64            `json:"error_executions"`
	TimeoutExecutions    int64            `json:"timeout_executions"`
	ScenarioExecutions   map[string]int64 `json:"scenario_executions"`
	AverageExecutionTime int64            `json:"average_execution_time_ms"`
	LastUpdated          time.Time        `json:"last_updated"`
}

// NewComplianceEngine creates a new compliance engine
func NewComplianceEngine(
	repository *ScenarioRepository,
	config *ComplianceConfig,
	validator *pqc.SolutionNodeValidator,
) *ComplianceEngine {
	cfg := DefaultComplianceConfig()
	if config != nil {
		cfg = *config
	}

	return &ComplianceEngine{
		repository:     repository,
		executor:       NewComplianceScriptExecutor(validator),
		config:         cfg,
		executionCache: make(map[string]*ScenarioResult),
		statistics: &ComplianceStatistics{
			ScenarioExecutions: make(map[string]int64),
			LastUpdated:        time.Now(),
		},
	}
}

// SetOntologyRegistry sets the ontology registry for rule validation
func (ce *ComplianceEngine) SetOntologyRegistry(registry *ontology.OntologyRegistry) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.ontologyRegistry = registry
}

// ValidateAgent runs all applicable regulatory scenarios against an agent's output
func (ce *ComplianceEngine) ValidateAgent(
	ctx context.Context,
	agentID string,
	agentOutput string,
	filter *ScenarioFilter,
) (*ValidationReport, error) {
	// Get scenarios to run
	scenarios, err := ce.repository.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}

	if len(scenarios) == 0 {
		// Try to get defaults if no scenarios found
		scenarios = GetDefaultRegulatoryScenarios()
	}

	// Run validation
	return ce.runValidation(ctx, agentID, agentOutput, scenarios)
}

// ValidateAgentWithScenarios validates against specific scenarios
func (ce *ComplianceEngine) ValidateAgentWithScenarios(
	ctx context.Context,
	agentID string,
	agentOutput string,
	scenarioIDs []string,
) (*ValidationReport, error) {
	var scenarios []*RegulatoryScenario

	for _, id := range scenarioIDs {
		scenario, err := ce.repository.Get(id)
		if err != nil {
			return nil, fmt.Errorf("failed to load scenario %s: %w", id, err)
		}
		scenarios = append(scenarios, scenario)
	}

	return ce.runValidation(ctx, agentID, agentOutput, scenarios)
}

// RunScenario executes a single scenario
func (ce *ComplianceEngine) RunScenario(
	ctx context.Context,
	scenarioID string,
	agentOutput string,
) (*ScenarioResult, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", scenarioID, hashString(agentOutput))
	if ce.config.EnableCaching {
		if cached := ce.getCachedResult(cacheKey); cached != nil {
			return cached, nil
		}
	}

	// Load scenario
	scenario, err := ce.repository.Get(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenario: %w", err)
	}

	// Check if valid
	if !scenario.IsValid() {
		return nil, fmt.Errorf("scenario is not currently valid (inactive or expired)")
	}

	// Execute
	result, err := ce.executor.ExecuteScenario(ctx, scenario, agentOutput)
	if err != nil {
		return nil, err
	}

	// Calculate compliance score
	result.ComplianceScore = ce.calculateComplianceScore(scenario, result)
	result.OverallScore = ce.calculateOverallScore(result)

	// Update scenario stats
	ce.repository.IncrementExecution(scenarioID)

	// Update statistics
	ce.updateStatistics(result)

	// Cache result
	if ce.config.EnableCaching {
		ce.cacheResult(cacheKey, result)
	}

	return result, nil
}

// ValidationReport contains the complete validation results
type ValidationReport struct {
	AgentID     string    `json:"agent_id"`
	ExecutionID string    `json:"execution_id"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`

	// Results
	ScenarioResults []*ScenarioResult `json:"scenario_results"`

	// Aggregated scores
	OverallScore    float64 `json:"overall_score"`
	ComplianceScore float64 `json:"compliance_score"`
	GuardrailScore  float64 `json:"guardrail_score"`
	ViolationScore  float64 `json:"violation_score"`

	// Status
	Status            string `json:"status"` // passed, failed, review
	CriticalFailures  int    `json:"critical_failures"`
	MandatoryFailures int    `json:"mandatory_failures"`

	// Summary
	Summary         string   `json:"summary"`
	Recommendations []string `json:"recommendations"`
}

// runValidation runs all scenarios and generates a report
func (ce *ComplianceEngine) runValidation(
	ctx context.Context,
	agentID string,
	agentOutput string,
	scenarios []*RegulatoryScenario,
) (*ValidationReport, error) {
	start := time.Now()

	report := &ValidationReport{
		AgentID:         agentID,
		ExecutionID:     generateExecutionID(),
		StartedAt:       start,
		ScenarioResults: make([]*ScenarioResult, 0),
	}

	// Execute scenarios
	var wg sync.WaitGroup
	resultChan := make(chan *ScenarioResult, len(scenarios))

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, ce.config.MaxConcurrentExecutions)

	for _, scenario := range scenarios {
		if !scenario.IsValid() {
			continue
		}

		wg.Add(1)
		go func(s *RegulatoryScenario) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := ce.RunScenario(ctx, s.ID, agentOutput)
			if err != nil {
				// Log error but continue
				return
			}

			resultChan <- result
		}(scenario)
	}

	// Close channel when all done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var totalGuardrailScore, totalComplianceScore float64
	for result := range resultChan {
		report.ScenarioResults = append(report.ScenarioResults, result)

		if result.GuardrailScore > 0 {
			totalGuardrailScore += result.GuardrailScore
		}
		if result.ComplianceScore > 0 {
			totalComplianceScore += result.ComplianceScore
		}

		// Count failures
		if result.Status == "failed" {
			if isCriticalScenario(result.ScenarioID) {
				report.CriticalFailures++
			}
			for _, gr := range result.GuardrailResults {
				if gr.Status == "failed" {
					report.MandatoryFailures++
				}
			}
		}
	}

	// Calculate aggregate scores
	if len(report.ScenarioResults) > 0 {
		report.GuardrailScore = totalGuardrailScore / float64(len(report.ScenarioResults))
		report.ComplianceScore = totalComplianceScore / float64(len(report.ScenarioResults))
		report.ViolationScore = ce.calculateViolationScore(report.ScenarioResults)
		report.OverallScore = ce.calculateOverallReportScore(report)
	}

	// Determine status
	report.Status = ce.determineStatus(report)

	// Generate summary
	report.Summary = ce.generateSummary(report)
	report.Recommendations = ce.generateRecommendations(report)

	report.CompletedAt = time.Now()
	report.DurationMs = report.CompletedAt.Sub(start).Milliseconds()

	return report, nil
}

// calculateComplianceScore calculates the compliance score for a scenario result
func (ce *ComplianceEngine) calculateComplianceScore(
	scenario *RegulatoryScenario,
	result *ScenarioResult,
) float64 {
	// Check violation detection
	detectedCount := len(result.DetectedViolations)
	expectedCount := len(scenario.ExpectedViolations)
	mustDetectCount := 0
	mustDetectFound := 0

	for _, v := range scenario.ExpectedViolations {
		if v.MustDetect {
			mustDetectCount++
			// Check if it was detected
			for _, d := range result.DetectedViolations {
				if d.RuleID == v.RuleID {
					mustDetectFound++
					break
				}
			}
		}
	}

	// Calculate violation detection score
	var violationScore float64
	if expectedCount > 0 {
		violationScore = (float64(detectedCount) / float64(expectedCount)) * 100
	} else {
		violationScore = 100.0
	}

	// Penalize missed mandatory violations
	if mustDetectCount > 0 {
		missedRate := float64(mustDetectCount-mustDetectFound) / float64(mustDetectCount)
		violationScore -= missedRate * ce.config.MandatoryPenalty * 100
	}

	if violationScore < 0 {
		violationScore = 0
	}

	return violationScore
}

// calculateOverallScore calculates the weighted overall score for a scenario
func (ce *ComplianceEngine) calculateOverallScore(result *ScenarioResult) float64 {
	score := (result.GuardrailScore * ce.config.GuardrailWeight) +
		(result.ComplianceScore * ce.config.ViolationWeight)

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// calculateViolationScore calculates overall violation detection score
func (ce *ComplianceEngine) calculateViolationScore(results []*ScenarioResult) float64 {
	if len(results) == 0 {
		return 100.0
	}

	var totalScore float64
	for _, r := range results {
		totalScore += r.ComplianceScore
	}

	return totalScore / float64(len(results))
}

// calculateOverallReportScore calculates the overall report score
func (ce *ComplianceEngine) calculateOverallReportScore(report *ValidationReport) float64 {
	score := (report.GuardrailScore * ce.config.GuardrailWeight) +
		(report.ComplianceScore * ce.config.ViolationWeight)

	// Penalize critical failures
	if report.CriticalFailures > 0 {
		score -= float64(report.CriticalFailures) * ce.config.MandatoryPenalty * 100
	}

	// Penalize mandatory guardrail failures
	if report.MandatoryFailures > 0 {
		score -= float64(report.MandatoryFailures) * (ce.config.MandatoryPenalty / 2) * 100
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// determineStatus determines the overall validation status
func (ce *ComplianceEngine) determineStatus(report *ValidationReport) string {
	// Critical failures = automatic fail
	if report.CriticalFailures > 0 {
		return "failed"
	}

	// Check overall score
	if report.OverallScore >= ce.config.PassThreshold {
		return "passed"
	} else if report.OverallScore >= ce.config.ReviewThreshold {
		return "review"
	}

	return "failed"
}

// generateSummary creates a human-readable summary
func (ce *ComplianceEngine) generateSummary(report *ValidationReport) string {
	switch report.Status {
	case "passed":
		return fmt.Sprintf(
			"Agent %s passed validation with score %.1f/100. All %d scenarios completed successfully.",
			report.AgentID, report.OverallScore, len(report.ScenarioResults),
		)
	case "failed":
		return fmt.Sprintf(
			"Agent %s FAILED validation with score %.1f/100. %d critical failures, %d mandatory guardrail failures.",
			report.AgentID, report.OverallScore, report.CriticalFailures, report.MandatoryFailures,
		)
	case "review":
		return fmt.Sprintf(
			"Agent %s requires manual review with score %.1f/100. Some scenarios showed concerning results.",
			report.AgentID, report.OverallScore,
		)
	default:
		return fmt.Sprintf(
			"Validation for agent %s completed with status '%s' and score %.1f/100",
			report.AgentID, report.Status, report.OverallScore,
		)
	}
}

// generateRecommendations creates actionable recommendations
func (ce *ComplianceEngine) generateRecommendations(report *ValidationReport) []string {
	var recommendations []string

	if report.Status == "passed" {
		recommendations = append(recommendations, "Agent meets regulatory compliance requirements")
		if report.OverallScore < 95 {
			recommendations = append(recommendations, "Consider reviewing edge cases to improve score")
		}
		return recommendations
	}

	if report.CriticalFailures > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("URGENT: Address %d critical compliance failures immediately", report.CriticalFailures))
	}

	if report.MandatoryFailures > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d mandatory guardrail failures for agent logic", report.MandatoryFailures))
	}

	if report.GuardrailScore < ce.config.PassThreshold {
		recommendations = append(recommendations,
			"Improve agent's ability to follow mandatory guardrails")
	}

	if report.ComplianceScore < ce.config.PassThreshold {
		recommendations = append(recommendations,
			"Enhance violation detection capabilities")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Review validation results for detailed findings")
	}

	return recommendations
}

// Cache operations
func (ce *ComplianceEngine) getCachedResult(key string) *ScenarioResult {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if result, ok := ce.executionCache[key]; ok {
		// Check TTL
		if time.Since(result.CompletedAt) < ce.config.CacheTTL {
			return result
		}
	}
	return nil
}

func (ce *ComplianceEngine) cacheResult(key string, result *ScenarioResult) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.executionCache[key] = result

	// Simple cache cleanup (in production, use LRU)
	if len(ce.executionCache) > 1000 {
		// Clear old entries
		now := time.Now()
		for k, v := range ce.executionCache {
			if now.Sub(v.CompletedAt) > ce.config.CacheTTL {
				delete(ce.executionCache, k)
			}
		}
	}
}

// Statistics operations
func (ce *ComplianceEngine) updateStatistics(result *ScenarioResult) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.statistics.TotalExecutions++
	ce.statistics.ScenarioExecutions[result.ScenarioID]++

	switch result.Status {
	case "passed":
		ce.statistics.PassedExecutions++
	case "failed":
		ce.statistics.FailedExecutions++
	case "error":
		ce.statistics.ErrorExecutions++
	case "timeout":
		ce.statistics.TimeoutExecutions++
	}

	// Update average execution time
	ce.statistics.AverageExecutionTime =
		(ce.statistics.AverageExecutionTime*(ce.statistics.TotalExecutions-1) + result.DurationMs) /
			ce.statistics.TotalExecutions

	ce.statistics.LastUpdated = time.Now()
}

// GetStatistics returns current statistics
func (ce *ComplianceEngine) GetStatistics() *ComplianceStatistics {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Return copy
	stats := *ce.statistics
	return &stats
}

// ResetStatistics resets execution statistics
func (ce *ComplianceEngine) ResetStatistics() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.statistics = &ComplianceStatistics{
		ScenarioExecutions: make(map[string]int64),
		LastUpdated:        time.Now(),
	}
}

// Helper functions
func isCriticalScenario(scenarioID string) bool {
	// Check if scenario is critical based on ID pattern
	criticalTypes := []string{
		"flash-crash", "basel-violation", "sec-abuse",
		"insider-trading", "ponzi",
	}

	for _, t := range criticalTypes {
		if strings.Contains(strings.ToLower(scenarioID), t) {
			return true
		}
	}
	return false
}

func hashString(s string) string {
	// Simple hash for cache key
	var hash int64
	for i, c := range s {
		hash += int64(c) * int64(i+1)
	}
	return fmt.Sprintf("%x", hash)
}
