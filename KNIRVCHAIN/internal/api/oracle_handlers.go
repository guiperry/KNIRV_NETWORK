package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handleServiceStatus returns status for a specific service
func (api *UnifiedAPI) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["name"]
	
	// Check Node.js services
	if api.nodeJSManager != nil {
		if service, err := api.nodeJSManager.GetService(serviceName); err == nil {
			status := service.Status()
			serviceStatus := ServiceStatus{
				Name:      status.Name,
				Type:      "nodejs",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(serviceStatus)
			return
		}
	}
	
	// Check binary services
	if api.binaryManager != nil {
		if service, err := api.binaryManager.GetService(serviceName); err == nil {
			status := service.Status()
			serviceStatus := ServiceStatus{
				Name:      status.Name,
				Type:      "binary",
				Running:   status.Running,
				Port:      status.Port,
				PID:       status.PID,
				StartTime: status.StartTime,
				Error:     status.Error,
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(serviceStatus)
			return
		}
	}
	
	http.Error(w, fmt.Sprintf("Service %s not found", serviceName), http.StatusNotFound)
}

// handleNodeJSServices returns all Node.js services
func (api *UnifiedAPI) handleNodeJSServices(w http.ResponseWriter, r *http.Request) {
	if api.nodeJSManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"services": []ServiceStatus{},
			"total":    0,
		})
		return
	}
	
	statuses := api.nodeJSManager.GetServiceStatuses()
	var services []ServiceStatus
	
	for _, status := range statuses {
		services = append(services, ServiceStatus{
			Name:      status.Name,
			Type:      "nodejs",
			Running:   status.Running,
			Port:      status.Port,
			PID:       status.PID,
			StartTime: status.StartTime,
			Error:     status.Error,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
		"total":    len(services),
	})
}

// handleNodeJSStatus returns Node.js manager status
func (api *UnifiedAPI) handleNodeJSStatus(w http.ResponseWriter, r *http.Request) {
	if api.nodeJSManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled": false,
			"running": false,
		})
		return
	}
	
	runningServices := api.nodeJSManager.GetRunningServices()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":          true,
		"running":          api.nodeJSManager.IsRunning(),
		"running_services": len(runningServices),
		"total_services":   api.nodeJSManager.GetServiceCount(),
	})
}

// handleBinaryServices returns all binary services
func (api *UnifiedAPI) handleBinaryServices(w http.ResponseWriter, r *http.Request) {
	if api.binaryManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"services": []ServiceStatus{},
			"total":    0,
		})
		return
	}
	
	statuses := api.binaryManager.GetServiceStatuses()
	var services []ServiceStatus
	
	for _, status := range statuses {
		services = append(services, ServiceStatus{
			Name:      status.Name,
			Type:      "binary",
			Running:   status.Running,
			Port:      status.Port,
			PID:       status.PID,
			StartTime: status.StartTime,
			Error:     status.Error,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
		"total":    len(services),
	})
}

// handleBinaryStatus returns binary manager status
func (api *UnifiedAPI) handleBinaryStatus(w http.ResponseWriter, r *http.Request) {
	if api.binaryManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled": false,
			"running": false,
		})
		return
	}
	
	runningServices := api.binaryManager.GetRunningServices()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":          true,
		"running":          api.binaryManager.IsRunning(),
		"running_services": len(runningServices),
		"total_services":   api.binaryManager.GetServiceCount(),
	})
}

// handleTunnelStatus returns tunnel service status
func (api *UnifiedAPI) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	// Check if tunnel registry service is running
	if api.nodeJSManager != nil {
		if service, err := api.nodeJSManager.GetService("tunnel-registry"); err == nil {
			status := service.Status()
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service_running": status.Running,
				"port":           status.Port,
				"uptime":         time.Since(status.StartTime).String(),
				"status":         "active",
			})
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"status":         "inactive",
	})
}

// handleTunnelConnections returns active tunnel connections
func (api *UnifiedAPI) handleTunnelConnections(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual tunnel registry service
	// For now, return mock data structure
	connections := []map[string]interface{}{
		{
			"id":          "tunnel-001",
			"peer_id":     "peer-abc123",
			"status":      "active",
			"created_at":  time.Now().Add(-1 * time.Hour),
			"data_sent":   1024000,
			"data_recv":   2048000,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections":     connections,
		"total":          len(connections),
		"active_tunnels": len(connections),
	})
}

// handleWalletStatus returns wallet service status
func (api *UnifiedAPI) handleWalletStatus(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual wallet service
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "active",
		"connected":  true,
		"network":    api.config.ChainID,
		"address":    "knirv1abc123...", // Mock address
		"last_sync":  time.Now(),
	})
}

// handleWalletBalance returns wallet balance
func (api *UnifiedAPI) handleWalletBalance(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual wallet service
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"balance": map[string]interface{}{
			"NRN": map[string]interface{}{
				"amount":   "1000.50",
				"currency": "NRN",
				"usd_value": "150.75",
			},
		},
		"last_updated": time.Now(),
	})
}

// handlePaymentStatus returns payment gateway status
func (api *UnifiedAPI) handlePaymentStatus(w http.ResponseWriter, r *http.Request) {
	// Check if payment gateway service is running
	if api.nodeJSManager != nil {
		if service, err := api.nodeJSManager.GetService("payment-gateway"); err == nil {
			status := service.Status()
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service_running": status.Running,
				"port":           status.Port,
				"uptime":         time.Since(status.StartTime).String(),
				"stripe_enabled": true,
				"coinbase_enabled": true,
				"status":         "active",
			})
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_running": false,
		"status":         "inactive",
	})
}

// handlePaymentHistory returns payment history
func (api *UnifiedAPI) handlePaymentHistory(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the actual payment service
	payments := []map[string]interface{}{
		{
			"id":        "pay-001",
			"amount":    "50.00",
			"currency":  "USD",
			"status":    "completed",
			"method":    "stripe",
			"timestamp": time.Now().Add(-2 * time.Hour),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payments": payments,
		"total":    len(payments),
	})
}
