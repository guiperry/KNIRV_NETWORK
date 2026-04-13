package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type mockAnalyticsService struct {
	predictions     map[string]*cognitiveengine.LoadPrediction
	recommended     string
	capacityCurrent float64
	capacityProject float64
	metrics         map[string][]float64
}

func (m *mockAnalyticsService) PredictLoad(name string, horizon time.Duration) *cognitiveengine.LoadPrediction {
	if pred, ok := m.predictions[name]; ok {
		return pred
	}
	return &cognitiveengine.LoadPrediction{
		PredictedLoad:    0.5,
		Confidence:       0.8,
		PredictedAt:      time.Now(),
		PredictionWindow: horizon,
	}
}

func (m *mockAnalyticsService) GetRecommendedAction() string {
	return m.recommended
}

func (m *mockAnalyticsService) GetTrendDirection(name string) string {
	return "increasing"
}

func (m *mockAnalyticsService) GetCapacityForecast(horizon time.Duration) (float64, float64) {
	return m.capacityCurrent, m.capacityProject
}

func (m *mockAnalyticsService) ShouldTriggerProactiveScaling() bool {
	return m.recommended == "scale_up"
}

func (m *mockAnalyticsService) RecordMetric(name string, value float64) {
	if m.metrics == nil {
		m.metrics = make(map[string][]float64)
	}
	m.metrics[name] = append(m.metrics[name], value)
}

func (m *mockAnalyticsService) ExportMetrics() map[string][]float64 {
	return m.metrics
}

func TestAnalyticsHandlers_PredictLoad(t *testing.T) {
	service := &mockAnalyticsService{
		predictions: map[string]*cognitiveengine.LoadPrediction{
			"tasks_processed": {
				PredictedLoad:    0.75,
				Confidence:       0.9,
				PredictedAt:      time.Now(),
				PredictionWindow: 5 * time.Minute,
				LinearRegression: cognitiveengine.LinearRegression{
					Slope:     0.02,
					Intercept: 0.5,
					R2:        0.85,
				},
				IsAnomalous:  false,
				AnomalyScore: 0.3,
			},
		},
	}

	handler := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/analytics/predict?metric=tasks_processed", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["predicted_load"].(float64) != 0.75 {
		t.Errorf("expected predicted_load 0.75, got %v", response["predicted_load"])
	}

	if response["confidence"].(float64) != 0.9 {
		t.Errorf("expected confidence 0.9, got %v", response["confidence"])
	}
}

func TestAnalyticsHandlers_GetRecommendedAction(t *testing.T) {
	service := &mockAnalyticsService{
		recommended: "scale_up",
	}

	handler := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/analytics/recommendations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["recommended_action"] != "scale_up" {
		t.Errorf("expected recommended_action scale_up, got %v", response["recommended_action"])
	}

	if response["should_trigger_proactive"] != true {
		t.Errorf("expected should_trigger_proactive true, got %v", response["should_trigger_proactive"])
	}
}

func TestAnalyticsHandlers_GetCapacityForecast(t *testing.T) {
	service := &mockAnalyticsService{
		capacityCurrent: 0.6,
		capacityProject: 0.85,
	}

	handler := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/analytics/forecast?horizon=10m", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["current_utilization"].(float64) != 0.6 {
		t.Errorf("expected current_utilization 0.6, got %v", response["current_utilization"])
	}

	if response["projected_utilization"].(float64) != 0.85 {
		t.Errorf("expected projected_utilization 0.85, got %v", response["projected_utilization"])
	}
}

func TestAnalyticsHandlers_GetMetrics(t *testing.T) {
	service := &mockAnalyticsService{
		metrics: map[string][]float64{
			"tasks_processed": {10, 20, 30, 40, 50},
			"cpu_usage":       {0.3, 0.4, 0.5, 0.6, 0.7},
		},
	}

	handler := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/analytics/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	tasksData := response["tasks_processed"].(map[string]interface{})
	if tasksData["count"].(float64) != 5 {
		t.Errorf("expected count 5, got %v", tasksData["count"])
	}

	if tasksData["latest"].(float64) != 50 {
		t.Errorf("expected latest 50, got %v", tasksData["latest"])
	}

	if tasksData["min"].(float64) != 10 {
		t.Errorf("expected min 10, got %v", tasksData["min"])
	}

	if tasksData["max"].(float64) != 50 {
		t.Errorf("expected max 50, got %v", tasksData["max"])
	}

	if tasksData["avg"].(float64) != 30 {
		t.Errorf("expected avg 30, got %v", tasksData["avg"])
	}
}

func TestAnalyticsHandlers_GetTrendDirection(t *testing.T) {
	service := &mockAnalyticsService{}

	handler := NewAnalyticsHandlers(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/analytics/trend?metric=cpu_usage", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["direction"] != "increasing" {
		t.Errorf("expected direction increasing, got %v", response["direction"])
	}

	if response["metric"] != "cpu_usage" {
		t.Errorf("expected metric cpu_usage, got %v", response["metric"])
	}
}
