package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type AnalyticsService interface {
	PredictLoad(name string, horizon time.Duration) *cognitiveengine.LoadPrediction
	GetRecommendedAction() string
	GetTrendDirection(name string) string
	GetCapacityForecast(horizon time.Duration) (currentUtil, projectedUtil float64)
	ShouldTriggerProactiveScaling() bool
	RecordMetric(name string, value float64)
	ExportMetrics() map[string][]float64
}

type AnalyticsHandlers struct {
	analytics AnalyticsService
}

func NewAnalyticsHandlers(analytics AnalyticsService) *AnalyticsHandlers {
	return &AnalyticsHandlers{
		analytics: analytics,
	}
}

func (h *AnalyticsHandlers) RegisterRoutes(r *mux.Router) {
	analyticsRouter := r.PathPrefix("/api/analytics").Subrouter()

	analyticsRouter.HandleFunc("/predict", h.PredictLoad).Methods("GET", "OPTIONS")
	analyticsRouter.HandleFunc("/recommendations", h.GetRecommendedAction).Methods("GET", "OPTIONS")
	analyticsRouter.HandleFunc("/trend", h.GetTrendDirection).Methods("GET", "OPTIONS")
	analyticsRouter.HandleFunc("/forecast", h.GetCapacityForecast).Methods("GET", "OPTIONS")
	analyticsRouter.HandleFunc("/metrics", h.GetMetrics).Methods("GET", "OPTIONS")
	analyticsRouter.HandleFunc("/record", h.RecordMetric).Methods("POST", "OPTIONS")
}

func (h *AnalyticsHandlers) PredictLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := r.URL.Query().Get("metric")
	if name == "" {
		name = "tasks_processed"
	}

	horizon := 5 * time.Minute
	horizonStr := r.URL.Query().Get("horizon")
	if horizonStr != "" {
		if parsed, err := time.ParseDuration(horizonStr); err == nil {
			horizon = parsed
		}
	}

	prediction := h.analytics.PredictLoad(name, horizon)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metric":            name,
		"predicted_load":    prediction.PredictedLoad,
		"confidence":        prediction.Confidence,
		"predicted_at":      prediction.PredictedAt.Format(time.RFC3339),
		"prediction_window": prediction.PredictionWindow.String(),
		"is_anomalous":      prediction.IsAnomalous,
		"anomaly_score":     prediction.AnomalyScore,
		"linear_regression": map[string]interface{}{
			"slope":     prediction.LinearRegression.Slope,
			"intercept": prediction.LinearRegression.Intercept,
			"r2":        prediction.LinearRegression.R2,
		},
	})
}

func (h *AnalyticsHandlers) GetRecommendedAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	action := h.analytics.GetRecommendedAction()
	shouldScale := h.analytics.ShouldTriggerProactiveScaling()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommended_action":       action,
		"should_trigger_proactive": shouldScale,
		"timestamp":                time.Now().Format(time.RFC3339),
	})
}

func (h *AnalyticsHandlers) GetTrendDirection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := r.URL.Query().Get("metric")
	if name == "" {
		name = "tasks_processed"
	}

	direction := h.analytics.GetTrendDirection(name)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metric":    name,
		"direction": direction,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *AnalyticsHandlers) GetCapacityForecast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	horizon := 10 * time.Minute
	horizonStr := r.URL.Query().Get("horizon")
	if horizonStr != "" {
		if parsed, err := time.ParseDuration(horizonStr); err == nil {
			horizon = parsed
		}
	}

	currentUtil, projectedUtil := h.analytics.GetCapacityForecast(horizon)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_utilization":   currentUtil,
		"projected_utilization": projectedUtil,
		"horizon":               horizon.String(),
		"timestamp":             time.Now().Format(time.RFC3339),
	})
}

func (h *AnalyticsHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := h.analytics.ExportMetrics()

	response := make(map[string]interface{})
	for name, values := range metrics {
		if len(values) > 0 {
			response[name] = map[string]interface{}{
				"values": values,
				"count":  len(values),
				"latest": values[len(values)-1],
				"min":    minFloat(values),
				"max":    maxFloat(values),
				"avg":    avgFloat(values),
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AnalyticsHandlers) RecordMetric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Metric string  `json:"metric"`
		Value  float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Metric == "" {
		http.Error(w, `{"error":"metric name required"}`, http.StatusBadRequest)
		return
	}

	h.analytics.RecordMetric(req.Metric, req.Value)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "recorded",
		"metric":    req.Metric,
		"value":     req.Value,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func avgFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
