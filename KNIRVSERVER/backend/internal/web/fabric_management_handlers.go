package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/fabricmanagement"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type FabricManagementHandlers struct {
	fabricManagementService *fabricmanagement.FabricManagementService
}

func NewFabricManagementHandlers(fabricManagementService *fabricmanagement.FabricManagementService) *FabricManagementHandlers {
	return &FabricManagementHandlers{fabricManagementService: fabricManagementService}
}

type FabricManagementResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetFabrics handles GET /api/fabric-management/objects
func (h *FabricManagementHandlers) GetFabrics(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &objects.FabricFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}
	if fabricType := r.URL.Query().Get("type"); fabricType != "" {
		filter.Type = []string{fabricType}
	}
	if author := r.URL.Query().Get("author"); author != "" {
		filter.Author = author
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

	objects, err := h.fabricManagementService.GetAllFabrics(filter)
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve fabric objects: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      objects,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabric handles GET /api/fabric-management/objects/{id}
func (h *FabricManagementHandlers) GetFabric(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	fabric, err := h.fabricManagementService.GetFabric(fabricID)
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve fabric unit: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      fabric,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostFabric handles POST /api/fabric-management/objects
func (h *FabricManagementHandlers) PostFabric(w http.ResponseWriter, r *http.Request) {
	var fabric objects.Fabric
	if err := json.NewDecoder(r.Body).Decode(&fabric); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.fabricManagementService.CreateFabric(&fabric); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to create fabric item: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      &fabric,
		Message:   "Fabric item created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// PutFabric handles PUT /api/fabric-management/objects/{id}
func (h *FabricManagementHandlers) PutFabric(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var updates objects.Fabric
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.fabricManagementService.UpdateFabric(fabricID, &updates); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to update fabric item: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Message:   "Fabric item updated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteFabric handles DELETE /api/fabric-management/objects/{id}
func (h *FabricManagementHandlers) DeleteFabric(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.fabricManagementService.DeleteFabric(fabricID); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to delete fabric unit: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Message:   "Fabric unit deleted successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostFabricAction handles POST /api/fabric-management/objects/{id}/actions
func (h *FabricManagementHandlers) PostFabricAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var action objects.FabricAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.fabricManagementService.ExecuteFabricAction(fabricID, &action); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Message:   "Action executed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabricSummary handles GET /api/fabric-management/summary
func (h *FabricManagementHandlers) GetFabricSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.fabricManagementService.GetFabricSummary()

	response := FabricManagementResponse{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabricMetrics handles GET /api/fabric-management/objects/{id}/metrics
func (h *FabricManagementHandlers) GetFabricMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
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

	metrics, err := h.fabricManagementService.GetFabricMetrics(fabricID, limit)
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve metrics: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabricLogs handles GET /api/fabric-management/objects/{id}/logs
func (h *FabricManagementHandlers) GetFabricLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	if fabricID == "" {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Fabric ID is required",
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

	logs, err := h.fabricManagementService.GetFabricLogs(fabricID, limit)
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve logs: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      logs,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabricEvents handles GET /api/fabric-management/objects/{id}/events
func (h *FabricManagementHandlers) GetFabricEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fabricID := vars["id"]

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	events, err := h.fabricManagementService.GetFabricEvents(fabricID, limit)
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve events: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      events,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFabricTemplates handles GET /api/fabric-management/templates
func (h *FabricManagementHandlers) GetFabricTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.fabricManagementService.GetFabricTemplates()
	if err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve templates: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      templates,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostFabricTemplate handles POST /api/fabric-management/templates
func (h *FabricManagementHandlers) PostFabricTemplate(w http.ResponseWriter, r *http.Request) {
	var template objects.FabricTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.fabricManagementService.CreateFabricTemplate(&template); err != nil {
		response := FabricManagementResponse{
			Success:   false,
			Error:     "Failed to create template: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FabricManagementResponse{
		Success:   true,
		Data:      &template,
		Message:   "Template created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the fabric management routes with the router
func (h *FabricManagementHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for fabric management endpoints
	fabricRouter := r.PathPrefix("/api/fabric-management").Subrouter()

	// Handle OPTIONS preflight requests globally for this subrouter
	fabricRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Fabric CRUD operations (public for development)
	fabricRouter.HandleFunc("/objects", h.GetFabrics).Methods("GET")
	fabricRouter.HandleFunc("/objects", h.PostFabric).Methods("POST")
	fabricRouter.HandleFunc("/objects/{id}", h.GetFabric).Methods("GET")
	fabricRouter.HandleFunc("/objects/{id}", h.PutFabric).Methods("PUT")
	fabricRouter.HandleFunc("/objects/{id}", h.DeleteFabric).Methods("DELETE")

	// Fabric actions
	fabricRouter.HandleFunc("/objects/{id}/actions", h.PostFabricAction).Methods("POST")

	// Fabric monitoring (public for development)
	fabricRouter.HandleFunc("/objects/{id}/metrics", h.GetFabricMetrics).Methods("GET")
	fabricRouter.HandleFunc("/objects/{id}/logs", h.GetFabricLogs).Methods("GET")
	fabricRouter.HandleFunc("/objects/{id}/events", h.GetFabricEvents).Methods("GET")

	// Templates (public for development)
	fabricRouter.HandleFunc("/templates", h.GetFabricTemplates).Methods("GET")
	fabricRouter.HandleFunc("/templates", h.PostFabricTemplate).Methods("POST")

	// Summary (public for development)
	fabricRouter.HandleFunc("/summary", h.GetFabricSummary).Methods("GET")
}
