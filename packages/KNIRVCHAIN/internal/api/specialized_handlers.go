package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handlePlugins returns all available plugins (stub - services moved elsewhere)
func (api *UnifiedAPI) handlePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": []map[string]interface{}{},
		"total":   0,
		"message": "Plugin services moved to separate components",
	})
}

// handleEnablePlugin enables a specific plugin (stub - services moved elsewhere)
func (api *UnifiedAPI) handleEnablePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": fmt.Sprintf("Plugin %s cannot be enabled - services moved to separate components", pluginName),
	})
}

// handleDisablePlugin disables a specific plugin (stub - services moved elsewhere)
func (api *UnifiedAPI) handleDisablePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": fmt.Sprintf("Plugin %s cannot be disabled - services moved to separate components", pluginName),
	})
}

// handleNetworkStatus returns network monitoring status (stub - services moved elsewhere)
func (api *UnifiedAPI) handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"network":         api.config.ChainID,
		"testnet_enabled": api.config.Testnet.Enabled,
		"status":          "inactive",
		"message":         "Network monitoring services moved to separate components",
	})
}

// handleNetworkPeers returns network peer information (stub - services moved elsewhere)
func (api *UnifiedAPI) handleNetworkPeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers":        []map[string]interface{}{},
		"total":        0,
		"connected":    0,
		"network":      api.config.ChainID,
		"last_updated": time.Now(),
		"message":      "Network monitoring services moved to separate components",
	})
}

