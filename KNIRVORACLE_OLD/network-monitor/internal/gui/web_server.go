package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/knirv/network-monitor/internal/config"
	"github.com/knirv/network-monitor/internal/logging"
	"github.com/knirv/network-monitor/internal/monitoring"
	"github.com/sirupsen/logrus"
)

// WebServer provides a web interface for headless mode
type WebServer struct {
	config     *config.Config
	monitor    *monitoring.Monitor
	logManager *logging.Manager
	logger     *logrus.Logger
	server     *http.Server
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewWebServer creates a new web server
func NewWebServer(cfg *config.Config, monitor *monitoring.Monitor, logManager *logging.Manager, logger *logrus.Logger) *WebServer {
	return &WebServer{
		config:     cfg,
		monitor:    monitor,
		logManager: logManager,
		logger:     logger,
	}
}

// Start starts the web server
func (ws *WebServer) Start(port int) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", ws.handleHealth)
	mux.HandleFunc("/api/status", ws.handleNetworkStatus)
	mux.HandleFunc("/api/services", ws.handleServices)
	mux.HandleFunc("/api/service/", ws.handleServiceDetail)
	mux.HandleFunc("/api/metrics", ws.handleMetrics)
	mux.HandleFunc("/api/logs", ws.handleLogs)
	mux.HandleFunc("/api/logs/search", ws.handleLogSearch)
	mux.HandleFunc("/api/config", ws.handleConfig)
	mux.HandleFunc("/webhook/alerts", ws.handleAlertWebhook)

	// Static files and dashboard
	mux.HandleFunc("/", ws.handleDashboard)
	mux.HandleFunc("/dashboard", ws.handleDashboard)
	mux.HandleFunc("/static/", ws.handleStatic)

	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: ws.corsMiddleware(ws.loggingMiddleware(mux)),
	}

	ws.logger.Infof("Starting web server on port %d", port)
	return ws.server.ListenAndServe()
}

// Stop stops the web server
func (ws *WebServer) Stop() error {
	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

// handleHealth handles health check requests
func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success:   true,
		Data:      map[string]string{"status": "healthy"},
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleNetworkStatus handles network status requests
func (ws *WebServer) handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	status := ws.monitor.GetNetworkStatus()
	response := APIResponse{
		Success:   true,
		Data:      status,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleServices handles services list requests
func (ws *WebServer) handleServices(w http.ResponseWriter, r *http.Request) {
	status := ws.monitor.GetNetworkStatus()
	response := APIResponse{
		Success:   true,
		Data:      status.Services,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleServiceDetail handles individual service detail requests
func (ws *WebServer) handleServiceDetail(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Path[len("/api/service/"):]
	if serviceName == "" {
		ws.writeError(w, "Service name required", http.StatusBadRequest)
		return
	}

	serviceStatus, err := ws.monitor.GetServiceStatus(serviceName)
	if err != nil {
		ws.writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	response := APIResponse{
		Success:   true,
		Data:      serviceStatus,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleMetrics handles metrics requests
func (ws *WebServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := ws.monitor.GetMetrics()
	response := APIResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleLogs handles log requests
func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	logs := ws.logManager.GetRecentLogs(limit)
	response := APIResponse{
		Success:   true,
		Data:      logs,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleLogSearch handles log search requests
func (ws *WebServer) handleLogSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ws.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var query logging.LogQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		ws.writeError(w, "Invalid query format", http.StatusBadRequest)
		return
	}

	result, err := ws.logManager.SearchLogs(query)
	if err != nil {
		ws.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success:   true,
		Data:      result,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleConfig handles configuration requests
func (ws *WebServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Return safe configuration (without sensitive data)
	safeConfig := map[string]interface{}{
		"active_network": ws.config.ActiveNetwork,
		"networks":       ws.config.Networks,
		"gui":           ws.config.GUI,
	}

	response := APIResponse{
		Success:   true,
		Data:      safeConfig,
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleAlertWebhook handles incoming alert webhooks
func (ws *WebServer) handleAlertWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ws.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alertData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
		ws.writeError(w, "Invalid alert format", http.StatusBadRequest)
		return
	}

	// Process the alert
	ws.logger.Infof("Received alert webhook: %+v", alertData)

	// In a real implementation, this would:
	// 1. Parse the alert data
	// 2. Store it in the alert manager
	// 3. Trigger notifications
	// 4. Update the GUI

	response := APIResponse{
		Success:   true,
		Data:      map[string]string{"status": "alert received"},
		Timestamp: time.Now(),
	}
	ws.writeJSON(w, response)
}

// handleDashboard serves the web dashboard
func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KNIRV Network Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #1a1a1a; color: #fff; }
        .header { text-align: center; margin-bottom: 30px; }
        .status-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: #2a2a2a; border-radius: 8px; padding: 20px; border: 1px solid #444; }
        .status-up { color: #4CAF50; }
        .status-down { color: #f44336; }
        .status-degraded { color: #ff9800; }
        .refresh-btn { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
        .logs { max-height: 300px; overflow-y: auto; background: #1e1e1e; padding: 10px; border-radius: 4px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 KNIRV Network Monitor</h1>
        <button class="refresh-btn" onclick="refreshData()">Refresh</button>
    </div>
    
    <div id="content">
        <div class="status-grid">
            <div class="card">
                <h3>Network Status</h3>
                <div id="network-status">Loading...</div>
            </div>
            
            <div class="card">
                <h3>Services</h3>
                <div id="services-status">Loading...</div>
            </div>
            
            <div class="card">
                <h3>System Metrics</h3>
                <div id="metrics">Loading...</div>
            </div>
            
            <div class="card">
                <h3>Recent Logs</h3>
                <div id="logs" class="logs">Loading...</div>
            </div>
        </div>
    </div>

    <script>
        async function fetchData(endpoint) {
            try {
                const response = await fetch('/api/' + endpoint);
                const data = await response.json();
                return data.success ? data.data : null;
            } catch (error) {
                console.error('Error fetching ' + endpoint + ':', error);
                return null;
            }
        }

        async function refreshData() {
            // Network status
            const networkStatus = await fetchData('status');
            if (networkStatus) {
                document.getElementById('network-status').innerHTML = 
                    '<div class="status-' + networkStatus.overall_status + '">' +
                    'Status: ' + networkStatus.overall_status.toUpperCase() + '<br>' +
                    'Services: ' + networkStatus.services_up + '/' + networkStatus.services_total + ' up<br>' +
                    'Last Update: ' + new Date(networkStatus.last_update).toLocaleTimeString() +
                    '</div>';
            }

            // Services
            const services = await fetchData('services');
            if (services) {
                let servicesHtml = '';
                for (const [name, service] of Object.entries(services)) {
                    servicesHtml += '<div class="status-' + service.status + '">' +
                        name + ': ' + service.status.toUpperCase() + 
                        ' (' + service.response_time + ')' +
                        '</div>';
                }
                document.getElementById('services-status').innerHTML = servicesHtml;
            }

            // Metrics
            const metrics = await fetchData('metrics');
            if (metrics && metrics.system) {
                const sys = metrics.system;
                document.getElementById('metrics').innerHTML = 
                    'CPU: ' + (sys.cpu ? sys.cpu.usage_percent.toFixed(1) + '%' : 'N/A') + '<br>' +
                    'Memory: ' + (sys.memory ? sys.memory.usage_percent.toFixed(1) + '%' : 'N/A') + '<br>' +
                    'Disk: ' + (sys.disk ? sys.disk.usage_percent.toFixed(1) + '%' : 'N/A') + '<br>' +
                    'IPFS Storage: ' + (sys.ipfs ? sys.ipfs.storage_percent.toFixed(1) + '%' : 'N/A');
            }

            // Logs
            const logs = await fetchData('logs?limit=10');
            if (logs) {
                let logsHtml = '';
                logs.forEach(log => {
                    logsHtml += '<div>' +
                        new Date(log['@timestamp']).toLocaleTimeString() + ' ' +
                        '[' + log.level.toUpperCase() + '] ' +
                        log.service + ': ' + log.message +
                        '</div>';
                });
                document.getElementById('logs').innerHTML = logsHtml;
            }
        }

        // Initial load and auto-refresh
        refreshData();
        setInterval(refreshData, 5000);
    </script>
</body>
</html>
    `
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleStatic serves static files
func (ws *WebServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Placeholder for static file serving
	http.NotFound(w, r)
}

// writeJSON writes a JSON response
func (ws *WebServer) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (ws *WebServer) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := APIResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now(),
	}
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers
func (ws *WebServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (ws *WebServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		ws.logger.Debugf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
