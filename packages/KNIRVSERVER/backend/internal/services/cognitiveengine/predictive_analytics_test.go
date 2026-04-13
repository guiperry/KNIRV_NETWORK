package cognitiveengine

import (
	"math"
	"testing"
	"time"
)

func TestPredictiveAnalyticsNew(t *testing.T) {
	pa := NewPredictiveAnalytics(50)
	if pa == nil {
		t.Fatal("expected non-nil PredictiveAnalytics")
	}
	if pa.maxWindowSize != 50 {
		t.Errorf("expected window size 50, got %d", pa.maxWindowSize)
	}
	if pa.history == nil {
		t.Error("expected non-nil history map")
	}
}

func TestPredictiveAnalyticsDefaultWindowSize(t *testing.T) {
	pa := NewPredictiveAnalytics(0)
	if pa.maxWindowSize < 10 {
		t.Errorf("expected window size >= 10, got %d", pa.maxWindowSize)
	}
}

func TestPredictiveAnalyticsRecordMetricValues(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	pa.RecordMetric("cpu_usage", 0.5)
	pa.RecordMetric("cpu_usage", 0.6)
	pa.RecordMetric("cpu_usage", 0.7)

	if len(pa.history) != 1 {
		t.Errorf("expected 1 metric series, got %d", len(pa.history))
	}

	series := pa.history["cpu_usage"]
	if series == nil || len(series.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(series.Values))
	}
}

func TestPredictiveAnalyticsLinearRegression(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	values := []float64{1, 2, 3, 4, 5}
	for i, v := range values {
		pa.RecordMetric("linear", v)
		_ = i
	}

	lr := pa.linearRegression(pa.history["linear"])
	if math.Abs(lr.Slope-1.0) > 0.01 {
		t.Errorf("expected slope ~1.0, got %f", lr.Slope)
	}
	if math.Abs(lr.Intercept-1.0) > 0.01 {
		t.Errorf("expected intercept ~1.0, got %f", lr.Intercept)
	}
	if lr.R2 < 0.99 {
		t.Errorf("expected R2 > 0.99, got %f", lr.R2)
	}
}

func TestPredictiveAnalyticsLinearRegressionInsufficientData(t *testing.T) {
	pa := NewPredictiveAnalytics(100)
	pa.RecordMetric("sparse", 10.0)

	lr := pa.linearRegression(pa.history["sparse"])
	if lr.Slope != 0 {
		t.Errorf("expected slope 0 for single point, got %f", lr.Slope)
	}
	if lr.R2 != 0.0 {
		t.Errorf("expected R2 0 for single point, got %f", lr.R2)
	}
}

func TestPredictiveAnalyticsPredictLoadInsufficientData(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	prediction := pa.PredictLoad("nonexistent", 5*time.Minute)

	if prediction.PredictedLoad != 0.5 {
		t.Errorf("expected default prediction 0.5, got %f", prediction.PredictedLoad)
	}
	if prediction.Confidence != 0.0 {
		t.Errorf("expected 0 confidence for missing data, got %f", prediction.Confidence)
	}
}

func TestPredictiveAnalyticsPredictLoadSufficientData(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 15; i++ {
		pa.RecordMetric("tasks", float64(i*10))
	}

	prediction := pa.PredictLoad("tasks", 5*time.Minute)

	if prediction.PredictedLoad <= 0 {
		t.Errorf("expected positive prediction, got %f", prediction.PredictedLoad)
	}
	if prediction.Confidence <= 0 {
		t.Errorf("expected positive confidence, got %f", prediction.Confidence)
	}
	if prediction.PredictionWindow != 5*time.Minute {
		t.Errorf("expected 5m window, got %v", prediction.PredictionWindow)
	}
}

func TestPredictiveAnalyticsAnomalyDetection(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("cpu", 0.5+float64(i)*0.01)
	}

	pa.RecordMetric("cpu", 0.95)

	anomalyScore := pa.detectAnomaly(pa.history["cpu"])
	if anomalyScore < 2.0 {
		t.Errorf("expected anomaly score > 2.0 for outlier, got %f", anomalyScore)
	}
}

func TestPredictiveAnalyticsCalculateStats(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	mean, stdDev := pa.calculateStats([]float64{1, 2, 3, 4, 5})
	if math.Abs(mean-3.0) > 0.01 {
		t.Errorf("expected mean 3.0, got %f", mean)
	}
	if math.Abs(stdDev-1.414) > 0.01 {
		t.Errorf("expected stdDev ~1.414, got %f", stdDev)
	}
}

func TestPredictiveAnalyticsCalculateStatsEmpty(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	mean, stdDev := pa.calculateStats([]float64{})
	if mean != 0 || stdDev != 0 {
		t.Errorf("expected 0 for empty, got mean=%f stdDev=%f", mean, stdDev)
	}
}

func TestPredictiveAnalyticsShouldTriggerProactiveScaling(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("tasks_processed", 0.3)
	}

	result := pa.ShouldTriggerProactiveScaling()
	if result {
		t.Error("expected false for stable load")
	}
}

func TestPredictiveAnalyticsShouldTriggerProactiveScalingAnomalous(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("tasks_processed", 0.3)
	}
	pa.RecordMetric("tasks_processed", 0.95)

	result := pa.ShouldTriggerProactiveScaling()
	if !result {
		t.Error("expected true for anomalous load")
	}
}

func TestPredictiveAnalyticsGetRecommendedActionScaleUp(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("tasks_processed", 0.95)
		pa.RecordMetric("cpu_usage", 0.95)
	}

	action := pa.GetRecommendedAction()
	if action != "scale_up" {
		t.Errorf("expected 'scale_up', got '%s'", action)
	}
}

func TestPredictiveAnalyticsGetRecommendedActionScaleDown(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("tasks_processed", 0.1)
		pa.RecordMetric("cpu_usage", 0.1)
		pa.RecordMetric("memory_usage", 0.1)
	}

	action := pa.GetRecommendedAction()
	if action != "scale_down" {
		t.Errorf("expected 'scale_down', got '%s'", action)
	}
}

func TestPredictiveAnalyticsGetRecommendedActionMaintain(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("tasks_processed", 0.5)
		pa.RecordMetric("cpu_usage", 0.5)
		pa.RecordMetric("memory_usage", 0.5)
	}

	action := pa.GetRecommendedAction()
	if action != "maintain" {
		t.Errorf("expected 'maintain', got '%s'", action)
	}
}

func TestPredictiveAnalyticsGetTrendDirectionStable(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("load", 0.5)
	}

	direction := pa.GetTrendDirection("load")
	if direction != "stable" {
		t.Errorf("expected 'stable', got '%s'", direction)
	}
}

func TestPredictiveAnalyticsGetTrendDirectionIncreasing(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("load", 0.5+float64(i)*0.02)
	}

	direction := pa.GetTrendDirection("load")
	if direction != "increasing" {
		t.Errorf("expected 'increasing', got '%s'", direction)
	}
}

func TestPredictiveAnalyticsGetTrendDirectionDecreasing(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	for i := 0; i < 20; i++ {
		pa.RecordMetric("load", 1.0-float64(i)*0.02)
	}

	direction := pa.GetTrendDirection("load")
	if direction != "decreasing" {
		t.Errorf("expected 'decreasing', got '%s'", direction)
	}
}

func TestPredictiveAnalyticsGetCapacityForecast(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	pa.RecordMetric("tasks_processed", 0.7)
	pa.RecordMetric("cpu_usage", 0.6)

	current, projected := pa.GetCapacityForecast(5 * time.Minute)

	if current != 0.7 {
		t.Errorf("expected current 0.7, got %f", current)
	}
	if projected <= 0 {
		t.Errorf("expected positive projected utilization, got %f", projected)
	}
}

func TestPredictiveAnalyticsExportMetrics(t *testing.T) {
	pa := NewPredictiveAnalytics(100)

	pa.RecordMetric("cpu", 0.5)
	pa.RecordMetric("cpu", 0.6)
	pa.RecordMetric("memory", 0.4)

	exported := pa.ExportMetrics()

	if len(exported) != 2 {
		t.Errorf("expected 2 metric series, got %d", len(exported))
	}
	if len(exported["cpu"]) != 2 {
		t.Errorf("expected 2 CPU values, got %d", len(exported["cpu"]))
	}
	if len(exported["memory"]) != 1 {
		t.Errorf("expected 1 memory value, got %d", len(exported["memory"]))
	}
}

func TestLoadPredictionStruct(t *testing.T) {
	pred := LoadPrediction{
		PredictedLoad:    0.75,
		Confidence:       0.85,
		PredictedAt:      time.Now(),
		PredictionWindow: 5 * time.Minute,
		LinearRegression: LinearRegression{
			Slope:     0.1,
			Intercept: 0.5,
			R2:        0.92,
		},
		IsAnomalous:  false,
		AnomalyScore: 1.2,
	}

	if pred.PredictedLoad != 0.75 {
		t.Errorf("expected 0.75, got %f", pred.PredictedLoad)
	}
	if pred.Confidence != 0.85 {
		t.Errorf("expected 0.85, got %f", pred.Confidence)
	}
	if pred.LinearRegression.Slope != 0.1 {
		t.Errorf("expected slope 0.1, got %f", pred.LinearRegression.Slope)
	}
}

func TestMetricSeriesStruct(t *testing.T) {
	series := &MetricSeries{
		Values:     []float64{0.1, 0.2, 0.3},
		Timestamps: []time.Time{time.Now()},
		WindowSize: 100,
	}

	if len(series.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(series.Values))
	}
	if series.WindowSize != 100 {
		t.Errorf("expected window size 100, got %d", series.WindowSize)
	}
}
