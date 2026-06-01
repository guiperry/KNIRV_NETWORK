package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/pluginmanagement"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type PluginManagementHandlers struct {
	pluginManagementService *pluginmanagement.PluginManagementService
}

func NewPluginManagementHandlers(pluginManagementService *pluginmanagement.PluginManagementService) *PluginManagementHandlers {
	return &PluginManagementHandlers{pluginManagementService: pluginManagementService}
}

type PluginManagementResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetPlugins handles GET /api/plugin-management/objects
func (h *PluginManagementHandlers) GetPlugins(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &objects.PluginFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}
	if pluginType := r.URL.Query().Get("type"); pluginType != "" {
		filter.Type = []string{pluginType}
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

	objects, err := h.pluginManagementService.GetAllPlugins(filter)
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve plugin objects: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      objects,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPlugin handles GET /api/plugin-management/objects/{id}
func (h *PluginManagementHandlers) GetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	plugin, err := h.pluginManagementService.GetPlugin(pluginID)
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve plugin unit: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      plugin,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostPlugin handles POST /api/plugin-management/objects
func (h *PluginManagementHandlers) PostPlugin(w http.ResponseWriter, r *http.Request) {
	var plugin objects.Plugin
	if err := json.NewDecoder(r.Body).Decode(&plugin); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.pluginManagementService.CreatePlugin(&plugin); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to create plugin item: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      &plugin,
		Message:   "Plugin item created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// PutPlugin handles PUT /api/plugin-management/objects/{id}
func (h *PluginManagementHandlers) PutPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var updates objects.Plugin
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.pluginManagementService.UpdatePlugin(pluginID, &updates); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to update plugin item: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Message:   "Plugin item updated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePlugin handles DELETE /api/plugin-management/objects/{id}
func (h *PluginManagementHandlers) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.pluginManagementService.DeletePlugin(pluginID); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to delete plugin unit: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Message:   "Plugin unit deleted successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostPluginAction handles POST /api/plugin-management/objects/{id}/actions
func (h *PluginManagementHandlers) PostPluginAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var action objects.PluginAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.pluginManagementService.ExecutePluginAction(pluginID, &action); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Message:   "Action executed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPluginSummary handles GET /api/plugin-management/summary
func (h *PluginManagementHandlers) GetPluginSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.pluginManagementService.GetPluginSummary()

	response := PluginManagementResponse{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPluginMetrics handles GET /api/plugin-management/objects/{id}/metrics
func (h *PluginManagementHandlers) GetPluginMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
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

	metrics, err := h.pluginManagementService.GetPluginMetrics(pluginID, limit)
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve metrics: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPluginLogs handles GET /api/plugin-management/objects/{id}/logs
func (h *PluginManagementHandlers) GetPluginLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if pluginID == "" {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Plugin ID is required",
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

	logs, err := h.pluginManagementService.GetPluginLogs(pluginID, limit)
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve logs: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      logs,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPluginEvents handles GET /api/plugin-management/objects/{id}/events
func (h *PluginManagementHandlers) GetPluginEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	events, err := h.pluginManagementService.GetPluginEvents(pluginID, limit)
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve events: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      events,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPluginTemplates handles GET /api/plugin-management/templates
func (h *PluginManagementHandlers) GetPluginTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.pluginManagementService.GetPluginTemplates()
	if err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve templates: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      templates,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostPluginTemplate handles POST /api/plugin-management/templates
func (h *PluginManagementHandlers) PostPluginTemplate(w http.ResponseWriter, r *http.Request) {
	var template objects.PluginTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.pluginManagementService.CreatePluginTemplate(&template); err != nil {
		response := PluginManagementResponse{
			Success:   false,
			Error:     "Failed to create template: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PluginManagementResponse{
		Success:   true,
		Data:      &template,
		Message:   "Template created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleListBuiltins handles GET /api/plugins/builtins
func (h *PluginManagementHandlers) HandleListBuiltins(w http.ResponseWriter, r *http.Request) {
	builtins := h.pluginManagementService.ListBuiltins()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"builtins": builtins,
	})
}

// HandleToggleBuiltin handles POST /api/plugins/builtin/{id}/toggle
func (h *PluginManagementHandlers) HandleToggleBuiltin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var body struct {
		Enable bool `json:"enable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	if err := h.pluginManagementService.ToggleBuiltin(r.Context(), id, body.Enable); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(PluginManagementResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugin_id": id,
		"enabled":   body.Enable,
	})
}

// HandleUpdateBuiltinConfig handles PATCH /api/plugins/builtin/{id}/config
func (h *PluginManagementHandlers) HandleUpdateBuiltinConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PluginManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	if err := h.pluginManagementService.UpdateBuiltinConfig(id, patch); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PluginManagementResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugin_id": id,
		"updated":   true,
	})
}

// RegisterRoutes registers the plugin management routes with the router
func (h *PluginManagementHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for plugin management endpoints
	pluginRouter := r.PathPrefix("/api/plugin-management").Subrouter()

	// Handle OPTIONS preflight requests globally for this subrouter
	pluginRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Plugin CRUD operations (public for development)
	pluginRouter.HandleFunc("/objects", h.GetPlugins).Methods("GET")
	pluginRouter.HandleFunc("/objects", h.PostPlugin).Methods("POST")
	pluginRouter.HandleFunc("/objects/{id}", h.GetPlugin).Methods("GET")
	pluginRouter.HandleFunc("/objects/{id}", h.PutPlugin).Methods("PUT")
	pluginRouter.HandleFunc("/objects/{id}", h.DeletePlugin).Methods("DELETE")

	// Plugin actions
	pluginRouter.HandleFunc("/objects/{id}/actions", h.PostPluginAction).Methods("POST")

	// Plugin monitoring (public for development)
	pluginRouter.HandleFunc("/objects/{id}/metrics", h.GetPluginMetrics).Methods("GET")
	pluginRouter.HandleFunc("/objects/{id}/logs", h.GetPluginLogs).Methods("GET")
	pluginRouter.HandleFunc("/objects/{id}/events", h.GetPluginEvents).Methods("GET")

	// Templates (public for development)
	pluginRouter.HandleFunc("/templates", h.GetPluginTemplates).Methods("GET")
	pluginRouter.HandleFunc("/templates", h.PostPluginTemplate).Methods("POST")

	// Summary (public for development)
	pluginRouter.HandleFunc("/summary", h.GetPluginSummary).Methods("GET")

	// Builtin plugin routes under /api/plugins
	apiRouter := r.PathPrefix("/api/plugins").Subrouter()
	apiRouter.HandleFunc("/builtins", h.HandleListBuiltins).Methods("GET")
	apiRouter.HandleFunc("/builtin/{id}/toggle", h.HandleToggleBuiltin).Methods("POST")
	apiRouter.HandleFunc("/builtin/{id}/config", h.HandleUpdateBuiltinConfig).Methods("PATCH")
}
