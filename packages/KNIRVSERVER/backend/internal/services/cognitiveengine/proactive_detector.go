package cognitiveengine

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

type ProactiveDetector struct {
	mu               sync.RWMutex
	windowSize       time.Duration
	anomalyThreshold float64
	trendWindow      int
	historicalData   map[string][]MetricPoint
	alertCallback    func(ViolationAlert)
	ctx              context.Context
	cancel           context.CancelFunc
	stopCh           chan struct{}
	eventBus         *EventBus
	eventCh          <-chan EngineEvent
	metricsEventCh   <-chan EngineEvent
}

type MetricPoint struct {
	Timestamp time.Time
	Value     float64
}

type ViolationAlert struct {
	DVEID             string
	PolicyID          string
	Metric            string
	PredictedValue    float64
	CurrentValue      float64
	Threshold         float64
	TrendDirection    string
	Confidence        float64
	RecommendedAction string
	Timestamp         time.Time
}

type TrendResult struct {
	Direction string
	Slope     float64
	R2        float64
	Velocity  float64
	Forecast  []float64
}

func NewProactiveDetector(cfg *ProactiveDetectorConfig, eventBus *EventBus) *ProactiveDetector {
	ctx, cancel := context.WithCancel(context.Background())

	pd := &ProactiveDetector{
		windowSize:       cfg.WindowSize,
		anomalyThreshold: cfg.AnomalyThreshold,
		trendWindow:      cfg.TrendWindow,
		historicalData:   make(map[string][]MetricPoint),
		ctx:              ctx,
		cancel:           cancel,
		stopCh:           make(chan struct{}),
		eventBus:         eventBus,
	}

	if eventBus != nil {
		pd.eventCh = eventBus.Subscribe(EventPatternDetected)
		pd.metricsEventCh = eventBus.Subscribe(EventNodeMetricsUpdated)
	}

	return pd
}

func DefaultProactiveDetectorConfig() *ProactiveDetectorConfig {
	return &ProactiveDetectorConfig{
		WindowSize:       10 * time.Minute,
		AnomalyThreshold: 2.0,
		TrendWindow:      30,
	}
}

func (pd *ProactiveDetector) Start() {
	go pd.analysisLoop()
	go pd.cleanupLoop()
	go pd.eventLoop()

	log.Println("ProactiveDetector: started")
}

func (pd *ProactiveDetector) Stop() {
	pd.cancel()
	close(pd.stopCh)
	log.Println("ProactiveDetector: stopped")
}

func (pd *ProactiveDetector) RecordMetric(dveID, metric string, value float64) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	key := fmt.Sprintf("%s:%s", dveID, metric)

	point := MetricPoint{
		Timestamp: time.Now(),
		Value:     value,
	}

	pd.historicalData[key] = append(pd.historicalData[key], point)

	cutoff := time.Now().Add(-pd.windowSize)
	var filtered []MetricPoint
	for _, p := range pd.historicalData[key] {
		if p.Timestamp.After(cutoff) {
			filtered = append(filtered, p)
		}
	}
	pd.historicalData[key] = filtered
}

func (pd *ProactiveDetector) DetectViolations(dveID string, policies []*PolicyRule) []ViolationAlert {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	var alerts []ViolationAlert

	for _, policy := range policies {
		key := fmt.Sprintf("%s:%s", dveID, policy.Metric)
		points, ok := pd.historicalData[key]
		if !ok || len(points) < 5 {
			continue
		}

		trend := pd.calculateTrend(points)
		if trend == nil {
			continue
		}

		forecastValue := trend.Forecast[len(trend.Forecast)-1]

		var willViolate bool
		switch policy.Operator {
		case "gt":
			willViolate = forecastValue > policy.Threshold
		case "lt":
			willViolate = forecastValue < policy.Threshold
		case "gte":
			willViolate = forecastValue >= policy.Threshold
		case "lte":
			willViolate = forecastValue <= policy.Threshold
		case "eq":
			willViolate = math.Abs(forecastValue-policy.Threshold) < 0.001
		}

		if willViolate && trend.R2 > 0.7 {
			alert := ViolationAlert{
				DVEID:             dveID,
				PolicyID:          policy.ID,
				Metric:            policy.Metric,
				PredictedValue:    forecastValue,
				CurrentValue:      points[len(points)-1].Value,
				Threshold:         policy.Threshold,
				TrendDirection:    trend.Direction,
				Confidence:        trend.R2,
				RecommendedAction: pd.getRecommendedAction(policy, trend),
				Timestamp:         time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func (pd *ProactiveDetector) AnalyzeTrend(dveID, metric string) *TrendResult {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", dveID, metric)
	points, ok := pd.historicalData[key]
	if !ok || len(points) < 5 {
		return nil
	}

	return pd.calculateTrend(points)
}

func (pd *ProactiveDetector) calculateTrend(points []MetricPoint) *TrendResult {
	if len(points) < 5 {
		return nil
	}

	xValues := make([]float64, len(points))
	yValues := make([]float64, len(points))

	for i, p := range points {
		xValues[i] = float64(p.Timestamp.Unix())
		yValues[i] = p.Value
	}

	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	for i := range points {
		sumX += xValues[i]
		sumY += yValues[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	var numerator, denominator float64
	for i := range points {
		xDiff := xValues[i] - meanX
		yDiff := yValues[i] - meanY
		numerator += xDiff * yDiff
		denominator += xDiff * xDiff
	}

	var slope float64
	if denominator != 0 {
		slope = numerator / denominator
	}

	var ssRes, ssTot float64
	for i := range points {
		predicted := meanY + slope*(xValues[i]-meanX)
		ssRes += (yValues[i] - predicted) * (yValues[i] - predicted)
		ssTot += (yValues[i] - meanY) * (yValues[i] - meanY)
	}

	var r2 float64
	if ssTot != 0 {
		r2 = 1 - (ssRes / ssTot)
	}

	velocity := slope * 60.0

	var direction string
	if math.Abs(slope) < 0.01 {
		direction = "stable"
	} else if slope > 0 {
		direction = "increasing"
	} else {
		direction = "decreasing"
	}

	forecast := pd.forecast(points, 5)

	return &TrendResult{
		Direction: direction,
		Slope:     slope,
		R2:        r2,
		Velocity:  velocity,
		Forecast:  forecast,
	}
}

func (pd *ProactiveDetector) forecast(points []MetricPoint, horizon int) []float64 {
	trend := pd.calculateTrend(points)
	if trend == nil {
		return nil
	}

	lastTime := float64(points[len(points)-1].Timestamp.Unix())
	interval := float64(time.Minute / time.Nanosecond)

	meanVal := 0.0
	for _, p := range points {
		meanVal += p.Value
	}
	meanVal /= float64(len(points))

	var forecasts []float64
	for i := 1; i <= horizon; i++ {
		forecastTime := lastTime + (float64(i) * interval)
		forecast := trend.Slope*(forecastTime-float64(points[0].Timestamp.Unix())) + meanVal
		forecasts = append(forecasts, forecast)
	}

	return forecasts
}

func (pd *ProactiveDetector) DetectAnomalies(dveID, metric string) bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", dveID, metric)
	points, ok := pd.historicalData[key]
	if !ok || len(points) < 10 {
		return false
	}

	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}

	mean := mean(values)
	stdDev := stdDev(values)

	if stdDev == 0 {
		return false
	}

	lastValue := points[len(points)-1].Value
	zScore := math.Abs((lastValue - mean) / stdDev)

	return zScore > pd.anomalyThreshold
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := mean(values)
	sumSq := 0.0
	for _, v := range values {
		diff := v - m
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func (pd *ProactiveDetector) GetMetricHealth(dveID, metric string) string {
	if pd.DetectAnomalies(dveID, metric) {
		return "anomaly"
	}

	trend := pd.AnalyzeTrend(dveID, metric)
	if trend == nil {
		return "unknown"
	}

	if trend.R2 < 0.5 {
		return "volatile"
	}

	return "healthy"
}

func (pd *ProactiveDetector) getRecommendedAction(policy *PolicyRule, trend *TrendResult) string {
	switch policy.Severity {
	case "panic":
		return "immediate_intervention"
	case "critical":
		if trend.Direction == "increasing" {
			return "scale_up_preemptively"
		}
		return "prepare_scaling"
	case "warning":
		if trend.Direction == "increasing" {
			return "monitor_closely"
		}
		return "continue_monitoring"
	default:
		return "log_and_continue"
	}
}

func (pd *ProactiveDetector) analysisLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pd.stopCh:
			return
		case <-ticker.C:
			pd.runAnalysis()
		}
	}
}

func (pd *ProactiveDetector) runAnalysis() {
	pd.mu.RLock()
	keys := make([]string, 0, len(pd.historicalData))
	for key := range pd.historicalData {
		keys = append(keys, key)
	}
	pd.mu.RUnlock()

	for _, key := range keys {
		parts := splitKey(key)
		if len(parts) != 2 {
			continue
		}
		dveID, metric := parts[0], parts[1]

		if pd.DetectAnomalies(dveID, metric) {
			if pd.eventBus != nil {
				pd.eventBus.Publish(EngineEvent{
					Type:      EventPatternDetected,
					Source:    "proactive_detector",
					Payload:   map[string]interface{}{"metric": metric, "dve_id": dveID, "event": "anomaly"},
					Timestamp: time.Now(),
				})
			}
		}
	}
}

func (pd *ProactiveDetector) eventLoop() {
	if pd.metricsEventCh == nil {
		return
	}

	for {
		select {
		case <-pd.stopCh:
			return
		case event := <-pd.metricsEventCh:
			pd.handleMetricEvent(event)
		}
	}
}

func (pd *ProactiveDetector) handleMetricEvent(event EngineEvent) {
	if payload, ok := event.Payload.(map[string]interface{}); ok {
		if dveID, ok := payload["dve_id"].(string); ok {
			if metrics, ok := payload["metrics"].(map[string]float64); ok {
				for metric, value := range metrics {
					pd.RecordMetric(dveID, metric, value)
				}
			}
		}
	}
}

func (pd *ProactiveDetector) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pd.stopCh:
			return
		case <-ticker.C:
			pd.cleanupOldData()
		}
	}
}

func (pd *ProactiveDetector) cleanupOldData() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	cutoff := time.Now().Add(-pd.windowSize * 2)

	for key, points := range pd.historicalData {
		var filtered []MetricPoint
		for _, p := range points {
			if p.Timestamp.After(cutoff) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(pd.historicalData, key)
		} else {
			pd.historicalData[key] = filtered
		}
	}
}

func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return nil
}

type ProactiveDetectorConfig struct {
	WindowSize       time.Duration
	AnomalyThreshold float64
	TrendWindow      int
}

func (pd *ProactiveDetector) SetAlertCallback(callback func(ViolationAlert)) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.alertCallback = callback
}

func (pd *ProactiveDetector) CheckPreemptiveActions(dveID string, policies []*PolicyRule) []string {
	alerts := pd.DetectViolations(dveID, policies)
	var actions []string

	for _, alert := range alerts {
		if alert.Confidence > 0.8 && alert.Severity() == "critical" {
			actions = append(actions, alert.RecommendedAction)
		}
	}

	return actions
}

func (va *ViolationAlert) Severity() string {
	thresholdDiff := math.Abs(va.PredictedValue - va.Threshold)
	currentDiff := math.Abs(va.CurrentValue - va.Threshold)

	if currentDiff < thresholdDiff*0.1 {
		return "critical"
	}
	if currentDiff < thresholdDiff*0.3 {
		return "warning"
	}
	return "info"
}

func (pd *ProactiveDetector) GetPredictiveMetrics(dveID string) map[string]interface{} {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	metrics := make(map[string]interface{})
	prefix := dveID + ":"

	for key, points := range pd.historicalData {
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			continue
		}

		metric := key[len(prefix):]
		if len(points) < 5 {
			continue
		}

		trend := pd.calculateTrend(points)
		if trend == nil {
			continue
		}

		metrics[metric] = map[string]interface{}{
			"current":    points[len(points)-1].Value,
			"trend":      trend.Direction,
			"velocity":   trend.Velocity,
			"confidence": trend.R2,
			"forecast":   trend.Forecast,
			"health":     pd.GetMetricHealth(dveID, metric),
		}
	}

	return metrics
}

func (pd *ProactiveDetector) ExportState() map[string]interface{} {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	state := make(map[string]interface{})

	count := 0
	for _, points := range pd.historicalData {
		count += len(points)
	}

	state["metrics_tracked"] = len(pd.historicalData)
	state["total_data_points"] = count
	state["window_size_seconds"] = pd.windowSize.Seconds()
	state["anomaly_threshold"] = pd.anomalyThreshold

	return state
}

func (pd *ProactiveDetector) GetPercentile(points []MetricPoint, percentile float64) float64 {
	if len(points) == 0 {
		return 0
	}

	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}

	sort.Float64s(values)

	index := int(math.Ceil(float64(len(values)) * percentile / 100.0))
	if index >= len(values) {
		index = len(values) - 1
	}
	if index < 0 {
		index = 0
	}

	return values[index]
}

func (pd *ProactiveDetector) DetectSeasonality(dveID, metric string, period time.Duration) bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", dveID, metric)
	points, ok := pd.historicalData[key]
	if !ok || len(points) < int(period/time.Minute)*3 {
		return false
	}

	return false
}

func (pd *ProactiveDetector) GetRollingAverage(dveID, metric string, window time.Duration) float64 {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", dveID, metric)
	points, ok := pd.historicalData[key]
	if !ok || len(points) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-window)
	var sum float64
	var count int
	for _, p := range points {
		if p.Timestamp.After(cutoff) {
			sum += p.Value
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}
