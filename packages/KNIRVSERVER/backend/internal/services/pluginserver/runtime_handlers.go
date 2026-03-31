package pluginserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleStartPlugin handles the /runtime/start endpoint
func (as *PluginServer) HandleStartPlugin(w http.ResponseWriter, r *http.Request) {
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

	plugin, err := as.runtimeManager.StartPlugin(request.Name, request.Binary, request.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugin)

	log.Printf("Started plugin %s (%s) from %s", request.Name, request.Binary, r.RemoteAddr)
}

// HandleStopPlugin handles the /runtime/stop/{id} endpoint
func (as *PluginServer) HandleStopPlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	pluginID := strings.TrimPrefix(r.URL.Path, "/runtime/stop/")
	if pluginID == "" {
		http.Error(w, "Plugin ID is required", http.StatusBadRequest)
		return
	}

	if err := as.runtimeManager.StopPlugin(pluginID); err != nil {
		http.Error(w, fmt.Sprintf("Error stopping plugin unit: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Plugin unit stopped successfully"}`)

	log.Printf("Stopped plugin %s from %s", pluginID, r.RemoteAddr)
}

// HandleListRunningPlugins handles the /runtime/objects endpoint
func (as *PluginServer) HandleListRunningPlugins(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	pluginUnits := as.runtimeManager.GetPluginList()

	response := struct {
		Plugins []interface{} `json:"objects"`
		Count   int           `json:"count"`
	}{
		Plugins: make([]interface{}, len(pluginUnits)),
		Count:   len(pluginUnits),
	}

	for i, plugin := range pluginUnits {
		response.Plugins[i] = plugin
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetPlugin handles the /runtime/plugin/{id} endpoint
func (as *PluginServer) HandleGetPlugin(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	pluginID := strings.TrimPrefix(r.URL.Path, "/runtime/plugin/")
	if pluginID == "" {
		http.Error(w, "Plugin ID is required", http.StatusBadRequest)
		return
	}

	plugin, err := as.runtimeManager.GetPlugin(pluginID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Plugin unit not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugin)
}

// HandleRuntimeStatus handles the /runtime/status endpoint
func (as *PluginServer) HandleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	if as.runtimeManager == nil {
		http.Error(w, "Runtime management not enabled", http.StatusServiceUnavailable)
		return
	}

	pluginUnits := as.runtimeManager.GetPluginList()

	status := map[string]interface{}{
		"running":      true,
		"plugin_count": len(pluginUnits),
		"max_objects":  as.maxPlugins,
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
