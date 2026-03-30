package cognitiveengine

import (
	"context"
	"log"
	"sync"
	"time"
)

type PolicyRefinementLoop struct {
	engine           *GuardrailEngine
	detector         *ProactiveDetector
	ctx              context.Context
	cancel           context.CancelFunc
	stopCh           chan struct{}
	refinementWindow time.Duration
	minSamples       int
	cooldown         time.Duration
	lastRefinement   map[string]time.Time
	mu               sync.RWMutex
	eventBus         *EventBus
}

func NewPolicyRefinementLoop(engine *GuardrailEngine, detector *ProactiveDetector, eventBus *EventBus) *PolicyRefinementLoop {
	ctx, cancel := context.WithCancel(context.Background())

	return &PolicyRefinementLoop{
		engine:           engine,
		detector:         detector,
		ctx:              ctx,
		cancel:           cancel,
		stopCh:           make(chan struct{}),
		refinementWindow: 1 * time.Hour,
		minSamples:       50,
		cooldown:         30 * time.Minute,
		lastRefinement:   make(map[string]time.Time),
		eventBus:         eventBus,
	}
}

func (prl *PolicyRefinementLoop) Start() {
	go prl.refinementLoop()
	log.Println("PolicyRefinementLoop: started")
}

func (prl *PolicyRefinementLoop) Stop() {
	prl.cancel()
	close(prl.stopCh)
	log.Println("PolicyRefinementLoop: stopped")
}

func (prl *PolicyRefinementLoop) refinementLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-prl.stopCh:
			return
		case <-ticker.C:
			prl.runRefinement()
		}
	}
}

func (prl *PolicyRefinementLoop) runRefinement() {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	violations := prl.engine.GetActiveViolations()

	dveMetrics := make(map[string]map[string][]float64)
	for _, v := range violations {
		if dveMetrics[v.DVEID] == nil {
			dveMetrics[v.DVEID] = make(map[string][]float64)
		}
		dveMetrics[v.DVEID][v.RuleID] = append(dveMetrics[v.DVEID][v.RuleID], v.MetricValue)
	}

	for dveID, rules := range dveMetrics {
		for ruleID, values := range rules {
			if len(values) < prl.minSamples {
				continue
			}

			if prl.shouldRefine(ruleID) {
				prl.refineBasedOnPattern(dveID, ruleID, values)
			}
		}
	}

	prl.detectAndRefineStablePatterns()
}

func (prl *PolicyRefinementLoop) shouldRefine(ruleID string) bool {
	if lastTime, exists := prl.lastRefinement[ruleID]; exists {
		if time.Since(lastTime) < prl.cooldown {
			return false
		}
	}
	return true
}

func (prl *PolicyRefinementLoop) refineBasedOnPattern(dveID, ruleID string, values []float64) {
	policy, exists := prl.engine.GetPolicy(ruleID)
	if !exists {
		return
	}

	mean := calculateMean(values)
	stdDev := calculateStdDev(values)
	triggerRate := float64(len(values)) / float64(prl.minSamples)

	oldThreshold := policy.Threshold

	switch policy.Severity {
	case "warning":
		if triggerRate > 0.3 {
			prl.adjustThreshold(policy, mean, stdDev, 0.15)
			prl.recordRefinement(ruleID, oldThreshold, policy.Threshold, "high_trigger_rate")
		}
	case "critical":
		if triggerRate > 0.2 {
			prl.adjustThreshold(policy, mean, stdDev, 0.1)
			prl.recordRefinement(ruleID, oldThreshold, policy.Threshold, "critical_high_trigger")
		}
	case "panic":
		if triggerRate > 0.1 {
			prl.adjustThreshold(policy, mean, stdDev, 0.05)
			prl.recordRefinement(ruleID, oldThreshold, policy.Threshold, "panic_safe_guard")
		}
	}
}

func (prl *PolicyRefinementLoop) adjustThreshold(policy *PolicyRule, mean, stdDev float64, adjustmentFactor float64) {
	switch policy.Operator {
	case "gt":
		newThreshold := mean + (stdDev * adjustmentFactor * 2)
		if newThreshold < policy.Threshold {
			policy.Threshold = newThreshold
		}
	case "lt":
		newThreshold := mean - (stdDev * adjustmentFactor * 2)
		if newThreshold > policy.Threshold {
			policy.Threshold = newThreshold
		}
	}
}

func (prl *PolicyRefinementLoop) detectAndRefineStablePatterns() {
	prl.mu.RLock()
	dveIDs := make(map[string]bool)
	for _, v := range prl.engine.GetActiveViolations() {
		dveIDs[v.DVEID] = true
	}
	prl.mu.RUnlock()

	for dveID := range dveIDs {
		metrics := prl.detector.GetPredictiveMetrics(dveID)
		for metricName, metricData := range metrics {
			if data, ok := metricData.(map[string]interface{}); ok {
				if health, ok := data["health"].(string); ok && health == "healthy" {
					prl.refineForStableMetric(dveID, metricName, data)
				}
			}
		}
	}
}

func (prl *PolicyRefinementLoop) refineForStableMetric(dveID, metric string, data map[string]interface{}) {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	for _, policy := range prl.engine.policies {
		if policy.DVEID != "" && policy.DVEID != dveID {
			continue
		}
		if policy.Metric != metric {
			continue
		}

		if !prl.shouldRefine(policy.ID) {
			continue
		}

		if confidence, ok := data["confidence"].(float64); ok && confidence > 0.9 {
			oldThreshold := policy.Threshold
			prl.tightenThreshold(policy, data)
			if oldThreshold != policy.Threshold {
				prl.recordRefinement(policy.ID, oldThreshold, policy.Threshold, "stable_pattern")
			}
		}
	}
}

func (prl *PolicyRefinementLoop) tightenThreshold(policy *PolicyRule, data map[string]interface{}) {
	if current, ok := data["current"].(float64); ok {
		switch policy.Operator {
		case "gt":
			buffer := current * 0.05
			newThreshold := current + buffer
			if newThreshold < policy.Threshold {
				policy.Threshold = newThreshold
			}
		case "lt":
			buffer := current * 0.05
			newThreshold := current - buffer
			if newThreshold > policy.Threshold {
				policy.Threshold = newThreshold
			}
		}
	}
}

func (prl *PolicyRefinementLoop) recordRefinement(ruleID string, oldThreshold, newThreshold float64, reason string) {
	prl.lastRefinement[ruleID] = time.Now()

	log.Printf("[REFINEMENT] Policy %s refined: %.4f -> %.4f (reason: %s)",
		ruleID, oldThreshold, newThreshold, reason)

	if prl.eventBus != nil {
		prl.eventBus.Publish(EngineEvent{
			Type:   EventPatternDetected,
			Source: "policy_refinement",
			Payload: map[string]interface{}{
				"rule_id":       ruleID,
				"old_threshold": oldThreshold,
				"new_threshold": newThreshold,
				"reason":        reason,
			},
			Timestamp: time.Now(),
		})
	}
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateMean(values)
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return sqrt(sumSq / float64(len(values)-1))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func (prl *PolicyRefinementLoop) GetRefinementHistory() map[string]time.Time {
	prl.mu.RLock()
	defer prl.mu.RUnlock()

	history := make(map[string]time.Time)
	for k, v := range prl.lastRefinement {
		history[k] = v
	}
	return history
}

func (prl *PolicyRefinementLoop) ForceRefinement(ruleID string) {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	delete(prl.lastRefinement, ruleID)

	violations := prl.engine.GetActiveViolations()
	var values []float64
	for _, v := range violations {
		if v.RuleID == ruleID {
			values = append(values, v.MetricValue)
		}
	}

	if len(values) >= prl.minSamples {
		prl.refineBasedOnPattern("", ruleID, values)
	}
}

func (prl *PolicyRefinementLoop) SetRefinementWindow(d time.Duration) {
	prl.mu.Lock()
	defer prl.mu.Unlock()
	prl.refinementWindow = d
}

func (prl *PolicyRefinementLoop) SetCooldown(d time.Duration) {
	prl.mu.Lock()
	defer prl.mu.Unlock()
	prl.cooldown = d
}

func (prl *PolicyRefinementLoop) SetMinSamples(n int) {
	prl.mu.Lock()
	defer prl.mu.Unlock()
	prl.minSamples = n
}

func (prl *PolicyRefinementLoop) ExportState() map[string]interface{} {
	prl.mu.RLock()
	defer prl.mu.RUnlock()

	return map[string]interface{}{
		"refinement_window_minutes": prl.refinementWindow.Minutes(),
		"cooldown_minutes":          prl.cooldown.Minutes(),
		"min_samples":               prl.minSamples,
		"last_refinements":          len(prl.lastRefinement),
	}
}
