package modelserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleStartModel handles the /runtime/start endpoint
func (as *ModelServer) HandleStartModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Name   string                 `json:"name"`
		Binary string                 `json:"binary"`
		Config map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	if request.Name == "" || request.Binary == "" {
		http.Error(w, "Name and binary are required", http.StatusBadRequest)
		return
	}

	model, err := as.runtimeManager.StartModel(request.Name, request.Binary, request.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting model: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)

	log.Printf("Started model %s (%s) from %s", request.Name, request.Binary, r.RemoteAddr)
}

// HandleStopModel handles the /runtime/stop/{id} endpoint
func (as *ModelServer) HandleStopModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	modelID := strings.TrimPrefix(r.URL.Path, "/runtime/stop/")
	if modelID == "" {
		http.Error(w, "Model ID is required", http.StatusBadRequest)
		return
	}

	if err := as.runtimeManager.StopModel(modelID); err != nil {
		http.Error(w, fmt.Sprintf("Error stopping model: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Model stopped successfully"}`)

	log.Printf("Stopped model %s from %s", modelID, r.RemoteAddr)
}

// HandleListRunningModels handles the /runtime/models endpoint
func (as *ModelServer) HandleListRunningModels(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	models := as.runtimeManager.GetModelList()

	response := struct {
		Models []interface{} `json:"models"`
		Count  int           `json:"count"`
	}{
		Models: make([]interface{}, len(models)),
		Count:  len(models),
	}

	for i, model := range models {
		response.Models[i] = model
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetModel handles the /runtime/model/{id} endpoint
func (as *ModelServer) HandleGetModel(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	modelID := strings.TrimPrefix(r.URL.Path, "/runtime/model/")
	if modelID == "" {
		http.Error(w, "Model ID is required", http.StatusBadRequest)
		return
	}

	model, err := as.runtimeManager.GetModel(modelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Model not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// HandleRuntimeStatus handles the /runtime/status endpoint
func (as *ModelServer) HandleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	models := as.runtimeManager.GetModelList()

	status := map[string]interface{}{
		"running":     true,
		"model_count": len(models),
		"max_models":  as.maxModels,
		"uptime":      time.Since(as.startTime).String(),
	}

	// Add resource pool status if available
	if as.runtimeManager.resourcePool != nil {
		status["resources"] = as.runtimeManager.resourcePool.GetResourceUsage()
	}

	// Add scheduler status if available
	if as.runtimeManager.scheduler != nil {
		status["scheduler"] = as.runtimeManager.scheduler.GetQueueStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
