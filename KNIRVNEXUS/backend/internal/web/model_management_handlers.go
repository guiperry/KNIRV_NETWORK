package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/modelmanagement"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type ModelManagementHandlers struct {
	modelManagementService *modelmanagement.ModelManagementService
}

func NewModelManagementHandlers(modelManagementService *modelmanagement.ModelManagementService) *ModelManagementHandlers {
	return &ModelManagementHandlers{modelManagementService: modelManagementService}
}

type ModelManagementResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetModels handles GET /api/model-management/objects
func (h *ModelManagementHandlers) GetModels(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &objects.ModelFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}
	if modelType := r.URL.Query().Get("type"); modelType != "" {
		filter.Type = []string{modelType}
	}
	if author := r.URL.Query().Get("author"); author != "" {
		filter.Author = author
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	objects, err := h.modelManagementService.GetAllModels(filter)
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve objects: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      objects,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModel handles GET /api/model-management/objects/{id}
func (h *ModelManagementHandlers) GetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	model, err := h.modelManagementService.GetModel(modelID)
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve model: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      model,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostModel handles POST /api/model-management/objects
func (h *ModelManagementHandlers) PostModel(w http.ResponseWriter, r *http.Request) {
	var model objects.Model
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.modelManagementService.CreateModel(&model); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to create model: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      &model,
		Message:   "Model created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// PutModel handles PUT /api/model-management/objects/{id}
func (h *ModelManagementHandlers) PutModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var updates objects.Model
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.modelManagementService.UpdateModel(modelID, &updates); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to update model: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Message:   "Model updated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteModel handles DELETE /api/model-management/objects/{id}
func (h *ModelManagementHandlers) DeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.modelManagementService.DeleteModel(modelID); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to delete model: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Message:   "Model deleted successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostModelAction handles POST /api/model-management/objects/{id}/actions
func (h *ModelManagementHandlers) PostModelAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var action objects.ModelAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.modelManagementService.ExecuteModelAction(modelID, &action); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Message:   "Action executed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModelSummary handles GET /api/model-management/summary
func (h *ModelManagementHandlers) GetModelSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.modelManagementService.GetModelSummary()

	response := ModelManagementResponse{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModelMetrics handles GET /api/model-management/objects/{id}/metrics
func (h *ModelManagementHandlers) GetModelMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	metrics, err := h.modelManagementService.GetModelMetrics(modelID, limit)
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve metrics: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModelLogs handles GET /api/model-management/objects/{id}/logs
func (h *ModelManagementHandlers) GetModelLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	if modelID == "" {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Model ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := h.modelManagementService.GetModelLogs(modelID, limit)
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve logs: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      logs,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModelEvents handles GET /api/model-management/objects/{id}/events
func (h *ModelManagementHandlers) GetModelEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	events, err := h.modelManagementService.GetModelEvents(modelID, limit)
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve events: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      events,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModelTemplates handles GET /api/model-management/templates
func (h *ModelManagementHandlers) GetModelTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.modelManagementService.GetModelTemplates()
	if err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve templates: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      templates,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostModelTemplate handles POST /api/model-management/templates
func (h *ModelManagementHandlers) PostModelTemplate(w http.ResponseWriter, r *http.Request) {
	var template objects.ModelTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.modelManagementService.CreateModelTemplate(&template); err != nil {
		response := ModelManagementResponse{
			Success:   false,
			Error:     "Failed to create template: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ModelManagementResponse{
		Success:   true,
		Data:      &template,
		Message:   "Template created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the model management routes with the router
func (h *ModelManagementHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for model management endpoints
	modelRouter := r.PathPrefix("/api/model-management").Subrouter()

	// For development, make routes public (remove auth requirement)
	// TODO: Re-enable authentication for production
	// if authMiddleware != nil {
	//	protectedModelRouter := modelRouter.PathPrefix("").Subrouter()
	//	protectedModelRouter.Use(authMiddleware.RequireAuth)

	// Model CRUD operations (public for development)
	modelRouter.HandleFunc("/objects", h.GetModels).Methods("GET")
	modelRouter.HandleFunc("/objects", h.PostModel).Methods("POST")
	modelRouter.HandleFunc("/objects/{id}", h.GetModel).Methods("GET")
	modelRouter.HandleFunc("/objects/{id}", h.PutModel).Methods("PUT")
	modelRouter.HandleFunc("/objects/{id}", h.DeleteModel).Methods("DELETE")

	// Model actions
	modelRouter.HandleFunc("/objects/{id}/actions", h.PostModelAction).Methods("POST")

	// Model monitoring (public for development)
	modelRouter.HandleFunc("/objects/{id}/metrics", h.GetModelMetrics).Methods("GET")
	modelRouter.HandleFunc("/objects/{id}/logs", h.GetModelLogs).Methods("GET")
	modelRouter.HandleFunc("/objects/{id}/events", h.GetModelEvents).Methods("GET")

	// Templates (public for development)
	modelRouter.HandleFunc("/templates", h.GetModelTemplates).Methods("GET")
	modelRouter.HandleFunc("/templates", h.PostModelTemplate).Methods("POST")

	// Summary (public for development)
	modelRouter.HandleFunc("/summary", h.GetModelSummary).Methods("GET")
	// }

	// Handle OPTIONS requests for CORS (ensure CORS headers are set)
	modelRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
	})
}
