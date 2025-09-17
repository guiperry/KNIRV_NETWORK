package payment

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"KNIRVORACLE/pkg/services/interfaces"
)

//go:embed agent-payment-gateway
var paymentGatewayFS embed.FS

// EmbeddedPaymentGateway represents the embedded payment gateway service
type EmbeddedPaymentGateway struct {
	name        string
	port        uint64
	enabled     bool
	running     bool
	process     *exec.Cmd
	mutex       sync.RWMutex
	startTime   time.Time
	workingDir  string
	environment map[string]string
}

// NewEmbeddedPaymentGateway creates a new embedded payment gateway service
func NewEmbeddedPaymentGateway(port uint64, enabled bool) *EmbeddedPaymentGateway {
	return &EmbeddedPaymentGateway{
		name:        "payment-gateway",
		port:        port,
		enabled:     enabled,
		running:     false,
		environment: make(map[string]string),
	}
}

// Name returns the service name
func (e *EmbeddedPaymentGateway) Name() string {
	return e.name
}

// IsRunning returns true if the service is currently running
func (e *EmbeddedPaymentGateway) IsRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.running
}

// GetPort returns the port the service is running on
func (e *EmbeddedPaymentGateway) GetPort() uint64 {
	return e.port
}

// GetPID returns the process ID if applicable
func (e *EmbeddedPaymentGateway) GetPID() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.process != nil && e.process.Process != nil {
		return e.process.Process.Pid
	}
	return 0
}

// Status returns the current service status
func (e *EmbeddedPaymentGateway) Status() interfaces.ServiceStatus {
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

// extractEmbeddedFiles extracts the embedded payment gateway files to a temporary directory
func (e *EmbeddedPaymentGateway) extractEmbeddedFiles() error {
	// Create temporary directory for extracted files
	tempDir, err := os.MkdirTemp("", "payment-gateway-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	e.workingDir = tempDir

	// Extract embedded files
	err = fs.WalkDir(paymentGatewayFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "." {
			return nil
		}

		// Create the target path
		targetPath := filepath.Join(tempDir, path)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		} else {
			// Create file
			data, err := paymentGatewayFS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read embedded file %s: %w", path, err)
			}

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Write file
			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
		}

		return nil
	})

	if err != nil {
		// Clean up on error
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to extract embedded files: %w", err)
	}

	log.Printf("[%s] Extracted embedded files to %s", e.name, tempDir)
	return nil
}

// Start starts the payment gateway service
func (e *EmbeddedPaymentGateway) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.running {
		return fmt.Errorf("payment gateway service is already running")
	}

	if !e.enabled {
		return fmt.Errorf("payment gateway service is disabled")
	}

	// Extract embedded files
	if err := e.extractEmbeddedFiles(); err != nil {
		return fmt.Errorf("failed to extract embedded files: %w", err)
	}

	// Set up environment variables with KNIRV_ prefix for consistency
	env := os.Environ()
	env = append(env, fmt.Sprintf("KNIRV_NODEJS_PORT=%d", e.port))
	env = append(env, fmt.Sprintf("PORT=%d", e.port)) // Keep for backward compatibility
	env = append(env, "KNIRV_NODEJS_ENV=production")
	env = append(env, "NODE_ENV=production") // Keep for backward compatibility

	// Add custom environment variables
	for key, value := range e.environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Check if dependencies are already installed
	nodeModulesPath := filepath.Join(e.workingDir, "agent-payment-gateway", "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		// Install dependencies only if node_modules doesn't exist
		log.Printf("[%s] Installing dependencies...", e.name)
		installCmd := exec.CommandContext(ctx, "npm", "install")
		installCmd.Dir = filepath.Join(e.workingDir, "agent-payment-gateway")
		installCmd.Env = env

		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
	} else {
		log.Printf("[%s] Dependencies already installed, skipping npm install", e.name)
	}

	// Start the service
	log.Printf("[%s] Starting payment gateway service on port %d...", e.name, e.port)

	e.process = exec.CommandContext(ctx, "node", "server.js")
	e.process.Dir = filepath.Join(e.workingDir, "agent-payment-gateway")
	e.process.Env = env

	// Start the process
	if err := e.process.Start(); err != nil {
		return fmt.Errorf("failed to start payment gateway service: %w", err)
	}

	e.running = true
	e.startTime = time.Now()

	log.Printf("[%s] Payment gateway service started with PID %d", e.name, e.process.Process.Pid)

	// Monitor the process in a goroutine
	go e.monitorProcess()

	return nil
}

// Stop stops the payment gateway service
func (e *EmbeddedPaymentGateway) Stop(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.running {
		return fmt.Errorf("payment gateway service is not running")
	}

	log.Printf("[%s] Stopping payment gateway service...", e.name)

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

	log.Printf("[%s] Payment gateway service stopped", e.name)
	return nil
}

// Restart restarts the payment gateway service
func (e *EmbeddedPaymentGateway) Restart(ctx context.Context) error {
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
func (e *EmbeddedPaymentGateway) SetEnvironment(env map[string]string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for key, value := range env {
		e.environment[key] = value
	}
}

// monitorProcess monitors the running process and updates status
func (e *EmbeddedPaymentGateway) monitorProcess() {
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
