package fabricserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleStartFabric handles the /runtime/start endpoint
func (as *FabricServer) HandleStartFabric(w http.ResponseWriter, r *http.Request) {
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

	fabric, err := as.runtimeManager.StartFabric(request.Name, request.Binary, request.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting fabric: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fabric)

	log.Printf("Started fabric %s (%s) from %s", request.Name, request.Binary, r.RemoteAddr)
}

// HandleStopFabric handles the /runtime/stop/{id} endpoint
func (as *FabricServer) HandleStopFabric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	fabricID := strings.TrimPrefix(r.URL.Path, "/runtime/stop/")
	if fabricID == "" {
		http.Error(w, "Fabric ID is required", http.StatusBadRequest)
		return
	}

	if err := as.runtimeManager.StopFabric(fabricID); err != nil {
		http.Error(w, fmt.Sprintf("Error stopping fabric unit: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Fabric unit stopped successfully"}`)

	log.Printf("Stopped fabric %s from %s", fabricID, r.RemoteAddr)
}

// HandleListRunningFabrics handles the /runtime/objects endpoint
func (as *FabricServer) HandleListRunningFabrics(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	fabricUnits := as.runtimeManager.GetFabricList()

	response := struct {
		Fabrics []interface{} `json:"objects"`
		Count   int           `json:"count"`
	}{
		Fabrics: make([]interface{}, len(fabricUnits)),
		Count:   len(fabricUnits),
	}

	for i, fabric := range fabricUnits {
		response.Fabrics[i] = fabric
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetFabric handles the /runtime/fabric/{id} endpoint
func (as *FabricServer) HandleGetFabric(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	fabricID := strings.TrimPrefix(r.URL.Path, "/runtime/fabric/")
	if fabricID == "" {
		http.Error(w, "Fabric ID is required", http.StatusBadRequest)
		return
	}

	fabric, err := as.runtimeManager.GetFabric(fabricID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Fabric unit not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fabric)
}

// HandleRuntimeStatus handles the /runtime/status endpoint
func (as *FabricServer) HandleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	fabricUnits := as.runtimeManager.GetFabricList()

	status := map[string]interface{}{
		"running":      true,
		"fabric_count": len(fabricUnits),
		"max_objects":  as.maxFabrics,
		"uptime":       time.Since(as.startTime).String(),
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
