package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCognitiveEngineService is a mock implementation of CognitiveEngineService
type MockCognitiveEngineService struct {
	mock.Mock
}

func (m *MockCognitiveEngineService) GetCognitiveMetrics(nodeID string) *cognitiveengine.CognitiveMetrics {
	args := m.Called(nodeID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*cognitiveengine.CognitiveMetrics)
}

func (m *MockCognitiveEngineService) GetLearningState() *cognitiveengine.LearningState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*cognitiveengine.LearningState)
}

func (m *MockCognitiveEngineService) GetAdaptationHistory(limit int) []cognitiveengine.AdaptationEvent {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]cognitiveengine.AdaptationEvent)
}

func TestNewCognitiveEngineHandlers(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	assert.NotNil(t, handlers)
	assert.Equal(t, mockService, handlers.cognitiveEngine)
}

func TestCognitiveEngineHandlers_GetCognitiveMetrics(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	// Test successful case
	expectedMetrics := &cognitiveengine.CognitiveMetrics{
		NodeID:                "test-node",
		TasksProcessed:        100,
		AverageProcessingTime: 50.0,
		SuccessRate:           0.85,
		AdaptationScore:       0.9,
		LearningProgress:      0.75,
		ResourceUtilization:   0.6,
		Timestamp:             time.Now(),
	}

	mockService.On("GetCognitiveMetrics", "test-node").Return(expectedMetrics)

	req := httptest.NewRequest("GET", "/api/cognitive/metrics/test-node", nil)
	w := httptest.NewRecorder()

	// Set up mux vars
	vars := map[string]string{"nodeId": "test-node"}
	req = mux.SetURLVars(req, vars)

	handlers.GetCognitiveMetrics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response cognitiveengine.CognitiveMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Compare individual fields instead of the whole struct due to timestamp precision
	assert.Equal(t, expectedMetrics.NodeID, response.NodeID)
	assert.Equal(t, expectedMetrics.TasksProcessed, response.TasksProcessed)
	assert.Equal(t, expectedMetrics.SuccessRate, response.SuccessRate)
	assert.Equal(t, expectedMetrics.AdaptationScore, response.AdaptationScore)
	assert.Equal(t, expectedMetrics.LearningProgress, response.LearningProgress)
	assert.Equal(t, expectedMetrics.ResourceUtilization, response.ResourceUtilization)

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_GetCognitiveMetrics_MissingNodeID(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	req := httptest.NewRequest("GET", "/api/cognitive/metrics/", nil)
	w := httptest.NewRecorder()

	handlers.GetCognitiveMetrics(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Node ID is required")
}

func TestCognitiveEngineHandlers_GetLearningState(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	expectedState := &cognitiveengine.LearningState{
		LearningProgress:     0.75,
		ConfidenceLevel:      0.9,
		TotalTasksProcessed:  1000,
		SuccessRate:          0.85,
		AverageProcessingTime: 150.0,
		LastUpdated:          time.Now(),
		TaskTypePerformance:  make(map[string]*cognitiveengine.TaskMetrics),
		NodePerformance:      make(map[string]*cognitiveengine.NodeMetrics),
		AdaptationHistory:    []cognitiveengine.AdaptationEvent{},
	}

	mockService.On("GetLearningState").Return(expectedState)

	req := httptest.NewRequest("GET", "/api/cognitive/learning-state", nil)
	w := httptest.NewRecorder()

	handlers.GetLearningState(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response cognitiveengine.LearningState
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Compare individual fields instead of the whole struct due to timestamp precision
	assert.Equal(t, expectedState.TotalTasksProcessed, response.TotalTasksProcessed)
	assert.Equal(t, expectedState.SuccessRate, response.SuccessRate)
	assert.Equal(t, expectedState.AverageProcessingTime, response.AverageProcessingTime)
	assert.Equal(t, expectedState.LearningProgress, response.LearningProgress)
	assert.Equal(t, expectedState.ConfidenceLevel, response.ConfidenceLevel)
	assert.Equal(t, len(expectedState.TaskTypePerformance), len(response.TaskTypePerformance))
	assert.Equal(t, len(expectedState.NodePerformance), len(response.NodePerformance))
	assert.Equal(t, len(expectedState.AdaptationHistory), len(response.AdaptationHistory))

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_GetAdaptationHistory(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	expectedHistory := []cognitiveengine.AdaptationEvent{
		{
			ID:             "test-adaptation-1",
			Timestamp:      time.Now(),
			TriggerReason:  "performance_drop",
			AdaptationType: "resource_adjustment",
			Changes:        map[string]interface{}{"cpu_boost": 0.2},
			ExpectedImpact: "Improved processing speed",
		},
	}

	mockService.On("GetAdaptationHistory", 10).Return(expectedHistory)

	req := httptest.NewRequest("GET", "/api/cognitive/adaptations", nil)
	w := httptest.NewRecorder()

	handlers.GetAdaptationHistory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 1, int(response["count"].(float64)))
	assert.Equal(t, 10, int(response["limit"].(float64)))

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_GetAdaptationHistory_CustomLimit(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	expectedHistory := []cognitiveengine.AdaptationEvent{}

	mockService.On("GetAdaptationHistory", 5).Return(expectedHistory)

	req := httptest.NewRequest("GET", "/api/cognitive/adaptations?limit=5", nil)
	w := httptest.NewRecorder()

	handlers.GetAdaptationHistory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 0, int(response["count"].(float64)))
	assert.Equal(t, 5, int(response["limit"].(float64)))

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_GetFailurePatterns(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	req := httptest.NewRequest("GET", "/api/cognitive/patterns", nil)
	w := httptest.NewRecorder()

	handlers.GetFailurePatterns(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 0, int(response["count"].(float64)))
	assert.Contains(t, response["note"].(string), "Pattern analysis is performed internally")
}

func TestCognitiveEngineHandlers_GetTaskPerformance(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	taskPerformance := map[string]*cognitiveengine.TaskMetrics{
		"inference": {
			TaskType:          "inference",
			TasksProcessed:    100,
			SuccessRate:       0.85,
			AvgProcessingTime: 50.0,
			AvgScore:          0.88,
			LastProcessed:     time.Now(),
		},
		"training": {
			TaskType:          "training",
			TasksProcessed:    50,
			SuccessRate:       0.92,
			AvgProcessingTime: 120.0,
			AvgScore:          0.95,
			LastProcessed:     time.Now(),
		},
	}

	expectedState := &cognitiveengine.LearningState{
		TaskTypePerformance: taskPerformance,
		LastUpdated:         time.Now(),
	}

	mockService.On("GetLearningState").Return(expectedState)

	req := httptest.NewRequest("GET", "/api/cognitive/performance/tasks", nil)
	w := httptest.NewRecorder()

	handlers.GetTaskPerformance(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// The response contains JSON-marshaled data, so we need to compare the structure differently
	taskPerfResponse := response["task_performance"].(map[string]interface{})
	assert.Equal(t, 2, len(taskPerfResponse))
	assert.Contains(t, taskPerfResponse, "inference")
	assert.Contains(t, taskPerfResponse, "training")
	assert.Equal(t, 2, int(response["total_task_types"].(float64)))

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_GetNodePerformance(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	nodePerformance := map[string]*cognitiveengine.NodeMetrics{
		"node-1": {
			NodeID:            "node-1",
			TasksProcessed:    200,
			SuccessRate:       0.88,
			AvgProcessingTime: 45.0,
			ReliabilityScore:  0.85,
			Specializations:   []string{"inference", "validation"},
			LastActive:        time.Now(),
		},
		"node-2": {
			NodeID:            "node-2",
			TasksProcessed:    150,
			SuccessRate:       0.91,
			AvgProcessingTime: 38.0,
			ReliabilityScore:  0.92,
			Specializations:   []string{"training", "analysis"},
			LastActive:        time.Now(),
		},
	}

	expectedState := &cognitiveengine.LearningState{
		NodePerformance: nodePerformance,
		LastUpdated:     time.Now(),
	}

	mockService.On("GetLearningState").Return(expectedState)

	req := httptest.NewRequest("GET", "/api/cognitive/performance/nodes", nil)
	w := httptest.NewRecorder()

	handlers.GetNodePerformance(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// The response contains JSON-marshaled data, so we need to compare the structure differently
	nodePerfResponse := response["node_performance"].(map[string]interface{})
	assert.Equal(t, 2, len(nodePerfResponse))
	assert.Contains(t, nodePerfResponse, "node-1")
	assert.Contains(t, nodePerfResponse, "node-2")
	assert.Equal(t, 2, int(response["total_nodes"].(float64)))

	mockService.AssertExpectations(t)
}

func TestCognitiveEngineHandlers_TriggerLearningCycle(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	req := httptest.NewRequest("POST", "/api/cognitive/learning/trigger", nil)
	w := httptest.NewRecorder()

	handlers.TriggerLearningCycle(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "learning_cycle_triggered", response["status"])
	assert.Contains(t, response["message"].(string), "Learning cycle has been triggered manually")
}

func TestCognitiveEngineHandlers_GetStatus(t *testing.T) {
	mockService := &MockCognitiveEngineService{}
	handlers := NewCognitiveEngineHandlers(mockService)

	taskPerformance := map[string]*cognitiveengine.TaskMetrics{
		"inference": {
			TaskType:          "inference",
			TasksProcessed:    100,
			SuccessRate:       0.85,
			AvgProcessingTime: 50.0,
			AvgScore:          0.88,
			LastProcessed:     time.Now(),
		},
	}
	nodePerformance := map[string]*cognitiveengine.NodeMetrics{
		"node-1": {
			NodeID:            "node-1",
			TasksProcessed:    200,
			SuccessRate:       0.88,
			AvgProcessingTime: 45.0,
			ReliabilityScore:  0.85,
			Specializations:   []string{"inference"},
			LastActive:        time.Now(),
		},
	}
	adaptationHistory := []cognitiveengine.AdaptationEvent{
		{
			ID:             "adaptation-1",
			Timestamp:      time.Now(),
			TriggerReason:  "test_adaptation",
			AdaptationType: "parameter_tuning",
			Changes:        map[string]interface{}{"test_param": "test_value"},
			ExpectedImpact: "Test adaptation impact",
		},
	}

	expectedState := &cognitiveengine.LearningState{
		LearningProgress:     0.75,
		ConfidenceLevel:      0.9,
		TotalTasksProcessed:  1000,
		SuccessRate:          0.85,
		AverageProcessingTime: 150.0,
		LastUpdated:          time.Now(),
		TaskTypePerformance:  taskPerformance,
		NodePerformance:      nodePerformance,
		AdaptationHistory:    adaptationHistory,
	}

	mockService.On("GetLearningState").Return(expectedState)

	req := httptest.NewRequest("GET", "/api/cognitive/status", nil)
	w := httptest.NewRecorder()

	handlers.GetStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "active", response["status"])
	assert.Equal(t, 0.75, response["learning_progress"])
	assert.Equal(t, 0.9, response["confidence_level"])
	assert.Equal(t, float64(1000), response["total_tasks_processed"])
	assert.Equal(t, 0.85, response["success_rate"])
	assert.Equal(t, 150.0, response["average_processing_time"])
	assert.Equal(t, 1, int(response["task_types_tracked"].(float64)))
	assert.Equal(t, 1, int(response["nodes_tracked"].(float64)))
	assert.Equal(t, 1, int(response["adaptations_applied"].(float64)))

	mockService.AssertExpectations(t)
}