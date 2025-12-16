package economics

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"KNIRVCHAIN/pkg/services/interfaces"
)

//go:embed economics-service
var economicsServiceBinary embed.FS

// EmbeddedEconomicsService represents the embedded economics service binary
type EmbeddedEconomicsService struct {
	name        string
	port        uint64
	enabled     bool
	running     bool
	process     *exec.Cmd
	mutex       sync.RWMutex
	startTime   time.Time
	workingDir  string
	binaryPath  string
	environment map[string]string
}

// NewEmbeddedEconomicsService creates a new embedded economics service
func NewEmbeddedEconomicsService(port uint64, enabled bool) *EmbeddedEconomicsService {
	return &EmbeddedEconomicsService{
		name:        "economics-service",
		port:        port,
		enabled:     enabled,
		running:     false,
		environment: make(map[string]string),
	}
}

// Name returns the service name
func (e *EmbeddedEconomicsService) Name() string {
	return e.name
}

// IsRunning returns true if the service is currently running
func (e *EmbeddedEconomicsService) IsRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.running
}

// GetPort returns the port the service is running on
func (e *EmbeddedEconomicsService) GetPort() uint64 {
	return e.port
}

// GetPID returns the process ID if applicable
func (e *EmbeddedEconomicsService) GetPID() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.process != nil && e.process.Process != nil {
		return e.process.Process.Pid
	}
	return 0
}

// Status returns the current service status
func (e *EmbeddedEconomicsService) Status() interfaces.ServiceStatus {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	status := interfaces.ServiceStatus{
		Name:    e.name,
		Running: e.running,
		Port:    e.port,
	}

	if e.running {
		status.StartTime = e.startTime
		if e.process != nil && e.process.Process != nil {
			status.PID = e.process.Process.Pid
		}
	}

	return status
}

// extractEmbeddedBinary extracts the embedded economics service binary to a temporary directory
func (e *EmbeddedEconomicsService) extractEmbeddedBinary() error {
	// Create temporary directory for extracted binary
	tempDir, err := os.MkdirTemp("", "economics-service-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	e.workingDir = tempDir

	// Extract embedded binary
	binaryData, err := economicsServiceBinary.ReadFile("economics-service")
	if err != nil {
		return fmt.Errorf("failed to read embedded binary: %w", err)
	}

	// Write binary to temporary location
	e.binaryPath = filepath.Join(tempDir, "economics-service")
	if err := os.WriteFile(e.binaryPath, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	log.Printf("[%s] Extracted embedded binary to %s", e.name, e.binaryPath)
	return nil
}

// Start starts the economics service
func (e *EmbeddedEconomicsService) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.running {
		return fmt.Errorf("economics service is already running")
	}

	if !e.enabled {
		return fmt.Errorf("economics service is disabled")
	}

	// Extract embedded binary
	if err := e.extractEmbeddedBinary(); err != nil {
		return fmt.Errorf("failed to extract embedded binary: %w", err)
	}

	// Set up environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("ECONOMICS_PORT=%d", e.port))
	env = append(env, "ECONOMICS_LOCAL_MODE=true")

	// Add custom environment variables
	for key, value := range e.environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Start the service
	log.Printf("[%s] Starting economics service on port %d...", e.name, e.port)

	e.process = exec.CommandContext(ctx, e.binaryPath, "-port", fmt.Sprintf("%d", e.port))
	e.process.Dir = e.workingDir
	e.process.Env = env

	// Start the process
	if err := e.process.Start(); err != nil {
		return fmt.Errorf("failed to start economics service: %w", err)
	}

	e.running = true
	e.startTime = time.Now()

	log.Printf("[%s] Economics service started with PID %d", e.name, e.process.Process.Pid)

	// Monitor the process in a goroutine
	go e.monitorProcess()

	return nil
}

// Stop stops the economics service
func (e *EmbeddedEconomicsService) Stop(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.running {
		return fmt.Errorf("economics service is not running")
	}

	log.Printf("[%s] Stopping economics service...", e.name)

	if e.process != nil && e.process.Process != nil {
		// Send termination signal
		if err := e.process.Process.Signal(os.Interrupt); err != nil {
			log.Printf("[%s] Failed to send interrupt signal: %v", e.name, err)
			// Force kill if interrupt fails
			if err := e.process.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			done <- e.process.Wait()
		}()

		select {
		case <-ctx.Done():
			// Context cancelled, force kill
			e.process.Process.Kill()
			return ctx.Err()
		case err := <-done:
			if err != nil {
				log.Printf("[%s] Process exited with error: %v", e.name, err)
			}
		case <-time.After(10 * time.Second):
			// Timeout, force kill
			log.Printf("[%s] Process did not exit gracefully, force killing", e.name)
			e.process.Process.Kill()
		}
	}

	e.running = false
	e.process = nil

	// Clean up temporary directory
	if e.workingDir != "" {
		if err := os.RemoveAll(e.workingDir); err != nil {
			log.Printf("[%s] Warning: Failed to clean up working directory %s: %v", e.name, e.workingDir, err)
		}
		e.workingDir = ""
	}

	log.Printf("[%s] Economics service stopped", e.name)
	return nil
}

// Restart restarts the economics service
func (e *EmbeddedEconomicsService) Restart(ctx context.Context) error {
	if err := e.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait a moment before restarting
	time.Sleep(2 * time.Second)

	if err := e.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// SetEnvironment sets environment variables for the service
func (e *EmbeddedEconomicsService) SetEnvironment(env map[string]string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for key, value := range env {
		e.environment[key] = value
	}
}

// monitorProcess monitors the running process and updates status
func (e *EmbeddedEconomicsService) monitorProcess() {
	if e.process == nil {
		return
	}

	err := e.process.Wait()

	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.running = false
	e.process = nil

	if err != nil {
		log.Printf("[%s] Process exited with error: %v", e.name, err)
	} else {
		log.Printf("[%s] Process exited normally", e.name)
	}
}
