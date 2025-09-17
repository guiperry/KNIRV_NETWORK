package docsify

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"KNIRVORACLE/pkg/services/interfaces"
)

//go:embed docsify
var docsifyFS embed.FS

// EmbeddedDocsify represents the embedded docsify documentation service
type EmbeddedDocsify struct {
	name      string
	port      uint64
	enabled   bool
	running   bool
	mutex     sync.RWMutex
	startTime time.Time
	server    *http.Server
}

// NewEmbeddedDocsify creates a new embedded docsify service
func NewEmbeddedDocsify(port uint64, enabled bool) *EmbeddedDocsify {
	return &EmbeddedDocsify{
		name:    "docsify",
		port:    port,
		enabled: enabled,
		running: false,
	}
}

// GetName returns the service name
func (e *EmbeddedDocsify) GetName() string {
	return e.name
}

// GetPort returns the service port
func (e *EmbeddedDocsify) GetPort() uint64 {
	return e.port
}

// SetPort sets the service port
func (e *EmbeddedDocsify) SetPort(port uint64) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.port = port
}

// IsEnabled returns whether the service is enabled
func (e *EmbeddedDocsify) IsEnabled() bool {
	return e.enabled
}

// SetEnabled sets whether the service is enabled
func (e *EmbeddedDocsify) SetEnabled(enabled bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.enabled = enabled
}

// IsRunning returns whether the service is running
func (e *EmbeddedDocsify) IsRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.running
}

// GetStatus returns the service status
func (e *EmbeddedDocsify) GetStatus() interfaces.ServiceStatus {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	status := interfaces.ServiceStatus{
		Name:    e.name,
		Running: e.running,
		Port:    e.port,
	}

	if e.running {
		status.StartTime = e.startTime
	}

	return status
}

// Start starts the docsify service
func (e *EmbeddedDocsify) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.running {
		return fmt.Errorf("docsify service is already running")
	}

	if !e.enabled {
		return fmt.Errorf("docsify service is disabled")
	}

	// Create a sub-filesystem for the docsify directory
	docsFS, err := fs.Sub(docsifyFS, "docsify")
	if err != nil {
		return fmt.Errorf("failed to create docsify filesystem: %w", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Create a file server for the documentation
	fileServer := http.FileServer(http.FS(docsFS))

	// Register the handler for the /docs/ path
	mux.Handle("/docs/", http.StripPrefix("/docs/", fileServer))

	// Also serve at root for direct access
	mux.Handle("/", fileServer)

	e.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", e.port),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Docsify documentation server starting on port %d", e.port)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Docsify server error: %v", err)
		}
	}()

	e.running = true
	e.startTime = time.Now()

	log.Printf("Docsify service started on port %d", e.port)
	log.Printf("Documentation available at http://localhost:%d/docs/", e.port)

	return nil
}

// Stop stops the docsify service
func (e *EmbeddedDocsify) Stop() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.running {
		return fmt.Errorf("docsify service is not running")
	}

	if e.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := e.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown docsify server: %w", err)
		}
	}

	e.running = false
	e.server = nil

	log.Printf("Docsify service stopped")

	return nil
}

// Restart restarts the docsify service
func (e *EmbeddedDocsify) Restart(ctx context.Context) error {
	if err := e.Stop(); err != nil {
		return fmt.Errorf("failed to stop docsify service: %w", err)
	}

	// Wait a moment before restarting
	time.Sleep(100 * time.Millisecond)

	return e.Start(ctx)
}

// GetMetrics returns service metrics
func (e *EmbeddedDocsify) GetMetrics() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	metrics := map[string]interface{}{
		"name":    e.name,
		"running": e.running,
		"enabled": e.enabled,
		"port":    e.port,
	}

	if e.running {
		metrics["uptime"] = time.Since(e.startTime).String()
		metrics["start_time"] = e.startTime.Format(time.RFC3339)
	}

	return metrics
}

// SetEnvironment sets environment variables (not used for docsify)
func (e *EmbeddedDocsify) SetEnvironment(env map[string]string) {
	// Docsify is static content, no environment variables needed
}

// GetLogs returns service logs (not implemented for docsify)
func (e *EmbeddedDocsify) GetLogs() ([]string, error) {
	return []string{"Docsify service logs not implemented"}, nil
}
