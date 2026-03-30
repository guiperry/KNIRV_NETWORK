package cognitiveengine

import (
	"math"
	"sync"
	"time"
)

type MetricSeries struct {
	Values     []float64
	Timestamps []time.Time
	WindowSize int
}

type LinearRegression struct {
	Slope     float64
	Intercept float64
	R2        float64
}

type LoadPrediction struct {
	PredictedLoad    float64
	Confidence       float64
	PredictedAt      time.Time
	PredictionWindow time.Duration
	LinearRegression LinearRegression
	IsAnomalous      bool
	AnomalyScore     float64
}

type PredictiveAnalytics struct {
	history       map[string]*MetricSeries
	maxWindowSize int
	mu            sync.RWMutex
}

func NewPredictiveAnalytics(windowSize int) *PredictiveAnalytics {
	if windowSize < 10 {
		windowSize = 100
	}
	return &PredictiveAnalytics{
		history:       make(map[string]*MetricSeries),
		maxWindowSize: windowSize,
	}
}

func (pa *PredictiveAnalytics) RecordMetric(name string, value float64) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	series, exists := pa.history[name]
	if !exists {
		series = &MetricSeries{
			Values:     make([]float64, 0, pa.maxWindowSize),
			Timestamps: make([]time.Time, 0, pa.maxWindowSize),
			WindowSize: pa.maxWindowSize,
		}
		pa.history[name] = series
	}

	series.Values = append(series.Values, value)
	series.Timestamps = append(series.Timestamps, time.Now())

	if len(series.Values) > pa.maxWindowSize {
		series.Values = series.Values[len(series.Values)-pa.maxWindowSize:]
		series.Timestamps = series.Timestamps[len(series.Timestamps)-pa.maxWindowSize:]
	}
}

func (pa *PredictiveAnalytics) PredictLoad(name string, horizon time.Duration) *LoadPrediction {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	series, exists := pa.history[name]
	if !exists || len(series.Values) < 10 {
		return &LoadPrediction{
			PredictedLoad:    0.5,
			Confidence:       0.0,
			PredictedAt:      time.Now(),
			PredictionWindow: horizon,
		}
	}

	lr := pa.linearRegression(series)
	predictedValue := lr.Slope*float64(len(series.Values)+int(horizon.Seconds())) + lr.Intercept

	anomalyScore := pa.detectAnomaly(series)
	isAnomalous := anomalyScore > 2.0

	confidence := math.Min(1.0, lr.R2)
	if isAnomalous {
		confidence *= 0.5
	}

	return &LoadPrediction{
		PredictedLoad:    math.Max(0.0, math.Min(1.0, predictedValue)),
		Confidence:       confidence,
		PredictedAt:      time.Now(),
		PredictionWindow: horizon,
		LinearRegression: lr,
		IsAnomalous:      isAnomalous,
		AnomalyScore:     anomalyScore,
	}
}

func (pa *PredictiveAnalytics) linearRegression(series *MetricSeries) LinearRegression {
	n := float64(len(series.Values))
	if n < 2 {
		return LinearRegression{Intercept: series.Values[0], R2: 0.0}
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i, v := range series.Values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
		sumY2 += v * v
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		mean := sumY / n
		return LinearRegression{Slope: 0, Intercept: mean, R2: 0.0}
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	ssTot := 0.0
	meanY := sumY / n
	for _, v := range series.Values {
		ssTot += (v - meanY) * (v - meanY)
	}

	ssRes := 0.0
	for i, v := range series.Values {
		predicted := slope*float64(i) + intercept
		ssRes += (v - predicted) * (v - predicted)
	}

	r2 := 0.0
	if ssTot > 0 {
		r2 = 1.0 - ssRes/ssTot
	}

	return LinearRegression{
		Slope:     slope,
		Intercept: intercept,
		R2:        math.Max(0.0, r2),
	}
}

func (pa *PredictiveAnalytics) detectAnomaly(series *MetricSeries) float64 {
	if len(series.Values) < 10 {
		return 0.0
	}

	mean, stdDev := pa.calculateStats(series.Values)
	if stdDev < 1e-10 {
		return 0.0
	}

	lastValue := series.Values[len(series.Values)-1]
	zScore := math.Abs(lastValue-mean) / stdDev

	return zScore
}

func (pa *PredictiveAnalytics) calculateStats(values []float64) (mean, stdDev float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / n

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= n
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func (pa *PredictiveAnalytics) ShouldTriggerProactiveScaling() bool {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	tasksProcessed := pa.history["tasks_processed"]
	if tasksProcessed == nil || len(tasksProcessed.Values) < 10 {
		return false
	}

	prediction := pa.PredictLoad("tasks_processed", 5*time.Minute)

	return prediction.IsAnomalous || prediction.PredictedLoad > 0.8
}

func (pa *PredictiveAnalytics) GetRecommendedAction() string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	tasksPrediction := pa.PredictLoad("tasks_processed", 5*time.Minute)
	cpuPrediction := pa.PredictLoad("cpu_usage", 5*time.Minute)
	memPrediction := pa.PredictLoad("memory_usage", 5*time.Minute)

	if tasksPrediction.PredictedLoad > 0.9 || cpuPrediction.PredictedLoad > 0.9 || memPrediction.PredictedLoad > 0.9 {
		return "scale_up"
	}

	if tasksPrediction.PredictedLoad < 0.2 && cpuPrediction.PredictedLoad < 0.3 && memPrediction.PredictedLoad < 0.3 {
		return "scale_down"
	}

	return "maintain"
}

func (pa *PredictiveAnalytics) GetTrendDirection(name string) string {
	prediction := pa.PredictLoad(name, 5*time.Minute)

	if math.Abs(prediction.LinearRegression.Slope) < 0.01 {
		return "stable"
	}

	if prediction.LinearRegression.Slope > 0.01 {
		return "increasing"
	}

	return "decreasing"
}

func (pa *PredictiveAnalytics) GetCapacityForecast(horizon time.Duration) (currentUtil, projectedUtil float64) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	currentTasks := pa.history["tasks_processed"]

	if currentTasks != nil && len(currentTasks.Values) > 0 {
		currentUtil = currentTasks.Values[len(currentTasks.Values)-1]
	}

	tasksPrediction := pa.PredictLoad("tasks_processed", horizon)
	projectedUtil = tasksPrediction.PredictedLoad

	if pa.history["cpu_usage"] != nil && len(pa.history["cpu_usage"].Values) > 0 {
		cpuPrediction := pa.PredictLoad("cpu_usage", horizon)
		projectedUtil = (projectedUtil + cpuPrediction.PredictedLoad) / 2.0
	}

	return currentUtil, projectedUtil
}

func (pa *PredictiveAnalytics) ExportMetrics() map[string][]float64 {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	result := make(map[string][]float64)
	for name, series := range pa.history {
		result[name] = make([]float64, len(series.Values))
		copy(result[name], series.Values)
	}
	return result
}
