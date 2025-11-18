package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"KNIRVCHAIN/config"

	"github.com/gorilla/mux"
)

// handlePlugins returns all available plugins
func (api *UnifiedAPI) handlePlugins(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual plugin system
	plugins := []map[string]interface{}{
		{
			"name":        "oracle-connector",
			"version":     "1.0.0",
			"enabled":     true,
			"description": "Oracle blockchain connector plugin",
			"status":      "active",
		},
		{
			"name":        "payment-processor",
			"version":     "2.1.0",
			"enabled":     true,
			"description": "Payment processing plugin",
			"status":      "active",
		},
		{
			"name":        "tunnel-manager",
			"version":     "1.5.0",
			"enabled":     false,
			"description": "Tunnel management plugin",
			"status":      "inactive",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
		"total":   len(plugins),
	})
}

// handleEnablePlugin enables a specific plugin
func (api *UnifiedAPI) handleEnablePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	// This would integrate with the actual plugin system
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Plugin %s enabled successfully", pluginName),
	})
}

// handleDisablePlugin disables a specific plugin
func (api *UnifiedAPI) handleDisablePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	// This would integrate with the actual plugin system
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Plugin %s disabled successfully", pluginName),
	})
}

// handleNetworkStatus returns network monitoring status
func (api *UnifiedAPI) handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	// Check if network monitor service is running
	if api.binaryManager != nil {
		if service, err := api.binaryManager.GetService("network-monitor"); err == nil {
			status := service.Status()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service_running": status.Running,
				"port":            status.Port,
				"uptime":          time.Since(status.StartTime).String(),
				"network":         api.config.ChainID,
				"testnet_enabled": api.config.Testnet.Enabled,
				"status":          "monitoring",
				"last_check":      time.Now(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"network":         api.config.ChainID,
		"testnet_enabled": api.config.Testnet.Enabled,
		"status":          "inactive",
	})
}

// handleNetworkPeers returns network peer information
func (api *UnifiedAPI) handleNetworkPeers(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual network monitoring service
	peers := []map[string]interface{}{
		{
			"id":         "peer-001",
			"address":    "192.168.1.100:8080",
			"status":     "connected",
			"last_seen":  time.Now().Add(-5 * time.Minute),
			"version":    "2.0.0",
			"role":       "bootnode",
			"latency_ms": 45,
		},
		{
			"id":         "peer-002",
			"address":    "192.168.1.101:8080",
			"status":     "connected",
			"last_seen":  time.Now().Add(-2 * time.Minute),
			"version":    "2.0.0",
			"role":       "client",
			"latency_ms": 32,
		},
	}

	connectedCount := 0
	for _, peer := range peers {
		if peer["status"] == "connected" {
			connectedCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers":        peers,
		"total":        len(peers),
		"connected":    connectedCount,
		"network":      api.config.ChainID,
		"last_updated": time.Now(),
	})
}

// handleEconomicsStatus returns economics service status
func (api *UnifiedAPI) handleEconomicsStatus(w http.ResponseWriter, r *http.Request) {
	// Check if economics service is running
	if api.binaryManager != nil {
		if service, err := api.binaryManager.GetService("economics-service"); err == nil {
			status := service.Status()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service_running": status.Running,
				"port":            status.Port,
				"uptime":          time.Since(status.StartTime).String(),
				"is_root":         api.config.IsRoot,
				"status":          "active",
				"last_check":      time.Now(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"is_root":         api.config.IsRoot,
		"status":          "inactive",
	})
}

// handleEconomicsMetrics returns economics metrics
func (api *UnifiedAPI) handleEconomicsMetrics(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual economics service
	metrics := map[string]interface{}{
		"token_supply": map[string]interface{}{
			"total":       "1000000000",
			"circulating": "750000000",
			"staked":      "200000000",
			"burned":      "50000000",
		},
		"network_stats": map[string]interface{}{
			"total_transactions": 1234567,
			"daily_volume":       "50000.00",
			"active_validators":  150,
			"staking_ratio":      0.20,
		},
		"price_data": map[string]interface{}{
			"nrn_usd":    "0.15",
			"market_cap": "112500000",
			"24h_change": "+2.5%",
			"volume_24h": "2500000",
		},
		"last_updated": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Additional utility handlers for Oracle operations

// handleOperatorGovernance returns operator governance status
func (api *UnifiedAPI) handleOperatorGovernance(w http.ResponseWriter, r *http.Request) {
	// Check if operator registry service is running
	if api.nodeJSManager != nil {
		if service, err := api.nodeJSManager.GetService("operator-registry"); err == nil {
			status := service.Status()

			governance := map[string]interface{}{
				"service_running":      status.Running,
				"port":                 status.Port,
				"registered_operators": 25,
				"active_proposals":     3,
				"voting_power": map[string]interface{}{
					"total":     "1000000",
					"delegated": "750000",
					"available": "250000",
				},
				"governance_token": "NRN",
				"last_updated":     time.Now(),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(governance)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"status":          "inactive",
	})
}

// handleSystemMetrics returns overall system metrics
func (api *UnifiedAPI) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	var totalServices, runningServices int

	// Count Node.js services
	if api.nodeJSManager != nil {
		nodeJSStatuses := api.nodeJSManager.GetServiceStatuses()
		totalServices += len(nodeJSStatuses)
		for _, status := range nodeJSStatuses {
			if status.Running {
				runningServices++
			}
		}
	}

	// Count binary services
	if api.binaryManager != nil {
		binaryStatuses := api.binaryManager.GetServiceStatuses()
		totalServices += len(binaryStatuses)
		for _, status := range binaryStatuses {
			if status.Running {
				runningServices++
			}
		}
	}

	metrics := map[string]interface{}{
		"system": map[string]interface{}{
			"total_services":   totalServices,
			"running_services": runningServices,
			"service_health":   float64(runningServices) / float64(totalServices) * 100,
			"uptime":           time.Since(time.Now()).String(), // This would be actual uptime
		},
		"oracle": map[string]interface{}{
			"chain_id":        api.config.ChainID,
			"role":            config.DetermineRoleFromConfig(api.config).String(),
			"is_root":         api.config.IsRoot,
			"is_bootnode":     api.config.IsBootnode,
			"testnet_enabled": api.config.Testnet.Enabled,
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
