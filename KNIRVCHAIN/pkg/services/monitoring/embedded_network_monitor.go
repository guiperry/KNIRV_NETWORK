package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"KNIRVCHAIN/config"
)

// EmbeddedNetworkMonitor provides a simplified network monitor for testnet mode
type EmbeddedNetworkMonitor struct {
	config    *config.Config
	server    *http.Server
	mu        sync.RWMutex
	isRunning bool
	port      int
	services  map[string]*ServiceStatus
	stopCh    chan struct{}
}

// ServiceStatus represents the status of a monitored service
type ServiceStatus struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Healthy      bool      `json:"healthy"`
	Status       int       `json:"status"`
	ResponseTime int64     `json:"responseTime"` // in milliseconds
	LastCheck    time.Time `json:"lastCheck"`
	Error        string    `json:"error,omitempty"`
}

// HealthResponse represents the overall health status response
type HealthResponse struct {
	Overall   string                    `json:"overall"`
	Timestamp time.Time                 `json:"timestamp"`
	Services  map[string]*ServiceStatus `json:"services"`
}

// NewEmbeddedNetworkMonitor creates a new embedded network monitor
func NewEmbeddedNetworkMonitor(cfg *config.Config) *EmbeddedNetworkMonitor {
	port := 8091
	if cfg.NetworkMonitor.Port > 0 {
		port = cfg.NetworkMonitor.Port
	}

	monitor := &EmbeddedNetworkMonitor{
		config:   cfg,
		port:     port,
		services: make(map[string]*ServiceStatus),
		stopCh:   make(chan struct{}),
	}

	// Initialize default testnet services
	monitor.initializeTestnetServices()

	return monitor
}

// initializeTestnetServices sets up the default services to monitor in testnet mode
func (enm *EmbeddedNetworkMonitor) initializeTestnetServices() {
	enm.mu.Lock()
	defer enm.mu.Unlock()

	// Define testnet services to monitor
	testnetServices := map[string]string{
		"knirvoracle": "http://localhost:1317/health",
		"knirvchain":  "http://localhost:8080/health",
		"knirvgraph":  "http://localhost:8081/health",
		"knirvnexus":  "http://localhost:8082/health",
		"knirvrouter": "http://localhost:8086/api/connectivity/status",
	}

	for name, url := range testnetServices {
		enm.services[name] = &ServiceStatus{
			Name:      name,
			URL:       url,
			Healthy:   false,
			Status:    0,
			LastCheck: time.Now(),
		}
	}
}

// Start starts the embedded network monitor web server
func (enm *EmbeddedNetworkMonitor) Start() error {
	enm.mu.Lock()
	defer enm.mu.Unlock()

	if enm.isRunning {
		return fmt.Errorf("embedded network monitor is already running")
	}

	// Create fresh stop channel per start
	enm.stopCh = make(chan struct{})

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/status", enm.handleStatus)
	mux.HandleFunc("/health", enm.handleHealth)
	mux.HandleFunc("/", enm.handleRoot)

	enm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", enm.port),
		Handler: enm.corsMiddleware(mux),
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Embedded Network Monitor starting on port %d", enm.port)
		if err := enm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Embedded Network Monitor server error: %v", err)
		}
	}()

	// Start health checking goroutine
	go enm.startHealthChecking()

	enm.isRunning = true
	log.Printf("Embedded Network Monitor started successfully on port %d", enm.port)
	return nil
}

// Stop stops the embedded network monitor
func (enm *EmbeddedNetworkMonitor) Stop() error {
	enm.mu.Lock()
	defer enm.mu.Unlock()

	if !enm.isRunning {
		return nil
	}

	// Stop health checking goroutine
	close(enm.stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := enm.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown embedded network monitor: %w", err)
	}

	enm.isRunning = false
	log.Printf("Embedded Network Monitor stopped")
	return nil
}

// IsRunning returns whether the monitor is currently running
func (enm *EmbeddedNetworkMonitor) IsRunning() bool {
	enm.mu.RLock()
	defer enm.mu.RUnlock()
	return enm.isRunning
}

// GetPort returns the port the monitor is running on
func (enm *EmbeddedNetworkMonitor) GetPort() int {
	return enm.port
}

// corsMiddleware adds CORS headers to allow cross-origin requests
func (enm *EmbeddedNetworkMonitor) corsMiddleware(next http.Handler) http.Handler {
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

// handleRoot handles requests to the root path
func (enm *EmbeddedNetworkMonitor) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"service": "KNIRV Embedded Network Monitor",
		"version": "1.0.0",
		"status":  "running",
		"port":    enm.port,
	}
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (enm *EmbeddedNetworkMonitor) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// handleStatus handles status requests (main API endpoint for health-monitor page)
func (enm *EmbeddedNetworkMonitor) handleStatus(w http.ResponseWriter, r *http.Request) {
	enm.mu.RLock()
	services := make(map[string]*ServiceStatus)
	for k, v := range enm.services {
		services[k] = v
	}
	enm.mu.RUnlock()

	// Determine overall status
	overall := "healthy"
	for _, service := range services {
		if !service.Healthy {
			overall = "degraded"
			break
		}
	}

	response := HealthResponse{
		Overall:   overall,
		Timestamp: time.Now(),
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// startHealthChecking starts the background health checking routine
func (enm *EmbeddedNetworkMonitor) startHealthChecking() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	// Initial check
	enm.checkAllServices()

	for {
		select {
		case <-ticker.C:
			enm.checkAllServices()
		case <-enm.stopCh:
			return
		}
	}
}

// checkAllServices checks the health of all monitored services
func (enm *EmbeddedNetworkMonitor) checkAllServices() {
	enm.mu.Lock()
	defer enm.mu.Unlock()

	for name, service := range enm.services {
		enm.checkService(name, service)
	}
}

// checkService checks the health of a single service
func (enm *EmbeddedNetworkMonitor) checkService(_ string, service *ServiceStatus) {
	start := time.Now()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(service.URL)

	service.LastCheck = time.Now()
	service.ResponseTime = time.Since(start).Milliseconds()

	if err != nil {
		service.Healthy = false
		service.Status = 0
		service.Error = err.Error()
		return
	}
	defer resp.Body.Close()

	service.Status = resp.StatusCode
	service.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	service.Error = ""

	if !service.Healthy {
		service.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}
