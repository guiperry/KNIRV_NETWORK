package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// Embed the compiled binaries
//
//go:embed bin/dve-manager
var dveManagerBinary []byte

//go:embed bin/validation-core
var validationCoreBinary []byte

//go:embed bin/nexus-server
var nexusServerBinary []byte

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// BackendOrchestrator manages the three domain-specific services
type BackendOrchestrator struct {
	tempDir string

	// Binary paths
	dveManagerPath     string
	validationCorePath string
	nexusServerPath    string

	// Process handles
	dveManagerCmd     *exec.Cmd
	validationCoreCmd *exec.Cmd
	nexusServerCmd    *exec.Cmd
}

// NewBackendOrchestrator creates a new backend orchestrator
func NewBackendOrchestrator() (*BackendOrchestrator, error) {
	// Create temporary directory for binaries
	tempDir, err := os.MkdirTemp("", "nexus-backend-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	orchestrator := &BackendOrchestrator{
		tempDir:            tempDir,
		dveManagerPath:     filepath.Join(tempDir, "dve-manager"),
		validationCorePath: filepath.Join(tempDir, "validation-core"),
		nexusServerPath:    filepath.Join(tempDir, "nexus-server"),
	}

	// Extract binaries to temp directory
	if err := orchestrator.extractBinaries(); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to extract binaries: %w", err)
	}

	return orchestrator, nil
}

// extractBinaries extracts embedded binaries to the temp directory
func (bo *BackendOrchestrator) extractBinaries() error {
	// Extract DVE Manager
	if err := bo.extractBinary(dveManagerBinary, bo.dveManagerPath); err != nil {
		return fmt.Errorf("failed to extract dve-manager: %w", err)
	}

	// Extract Validation Core
	if err := bo.extractBinary(validationCoreBinary, bo.validationCorePath); err != nil {
		return fmt.Errorf("failed to extract validation-core: %w", err)
	}

	// Extract Nexus Server
	if err := bo.extractBinary(nexusServerBinary, bo.nexusServerPath); err != nil {
		return fmt.Errorf("failed to extract nexus-server: %w", err)
	}

	return nil
}

// extractBinary extracts a binary to the specified path
func (bo *BackendOrchestrator) extractBinary(data []byte, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	// Make executable
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}

	return nil
}

// Start starts all the backend services
func (bo *BackendOrchestrator) Start() error {
	log.Printf("Starting KNIRV-NEXUS Backend v%s", Version)

	// Start DVE Manager on port 8082
	log.Println("Starting DVE Manager...")
	bo.dveManagerCmd = exec.Command(bo.dveManagerPath, "--port", "8082")
	bo.dveManagerCmd.Stdout = os.Stdout
	bo.dveManagerCmd.Stderr = os.Stderr
	if err := bo.dveManagerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start DVE Manager: %w", err)
	}
	log.Printf("DVE Manager started (PID: %d)", bo.dveManagerCmd.Process.Pid)

	// Start Validation Core on port 8083
	log.Println("Starting Validation Core...")
	bo.validationCoreCmd = exec.Command(bo.validationCorePath, "--port", "8083")
	bo.validationCoreCmd.Stdout = os.Stdout
	bo.validationCoreCmd.Stderr = os.Stderr
	if err := bo.validationCoreCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Validation Core: %w", err)
	}
	log.Printf("Validation Core started (PID: %d)", bo.validationCoreCmd.Process.Pid)

	// Wait a moment for services to initialize
	time.Sleep(2 * time.Second)

	// Start Nexus Server (API Gateway) on port 8081
	log.Println("Starting Nexus Server (API Gateway)...")
	bo.nexusServerCmd = exec.Command(bo.nexusServerPath)
	bo.nexusServerCmd.Stdout = os.Stdout
	bo.nexusServerCmd.Stderr = os.Stderr
	if err := bo.nexusServerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Nexus Server: %w", err)
	}
	log.Printf("Nexus Server started (PID: %d)", bo.nexusServerCmd.Process.Pid)

	log.Println("All backend services started successfully")
	return nil
}

// Stop stops all the backend services
func (bo *BackendOrchestrator) Stop() error {
	log.Println("Stopping backend services...")

	// Stop in reverse order
	if bo.nexusServerCmd != nil && bo.nexusServerCmd.Process != nil {
		log.Printf("Stopping Nexus Server (PID: %d)", bo.nexusServerCmd.Process.Pid)
		bo.nexusServerCmd.Process.Signal(syscall.SIGTERM)
		bo.nexusServerCmd.Wait()
	}

	if bo.validationCoreCmd != nil && bo.validationCoreCmd.Process != nil {
		log.Printf("Stopping Validation Core (PID: %d)", bo.validationCoreCmd.Process.Pid)
		bo.validationCoreCmd.Process.Signal(syscall.SIGTERM)
		bo.validationCoreCmd.Wait()
	}

	if bo.dveManagerCmd != nil && bo.dveManagerCmd.Process != nil {
		log.Printf("Stopping DVE Manager (PID: %d)", bo.dveManagerCmd.Process.Pid)
		bo.dveManagerCmd.Process.Signal(syscall.SIGTERM)
		bo.dveManagerCmd.Wait()
	}

	// Clean up temp directory
	if bo.tempDir != "" {
		os.RemoveAll(bo.tempDir)
	}

	log.Println("All backend services stopped")
	return nil
}

// Wait waits for all services to complete
func (bo *BackendOrchestrator) Wait() {
	// Wait for any service to exit
	done := make(chan error, 3)

	if bo.dveManagerCmd != nil {
		go func() {
			done <- bo.dveManagerCmd.Wait()
		}()
	}

	if bo.validationCoreCmd != nil {
		go func() {
			done <- bo.validationCoreCmd.Wait()
		}()
	}

	if bo.nexusServerCmd != nil {
		go func() {
			done <- bo.nexusServerCmd.Wait()
		}()
	}

	// Wait for first service to exit
	err := <-done
	if err != nil {
		log.Printf("Service exited with error: %v", err)
	}
}

// HealthCheck checks if all services are running
func (bo *BackendOrchestrator) HealthCheck() error {
	// Simple check - verify processes are still running
	if bo.dveManagerCmd != nil && bo.dveManagerCmd.Process != nil {
		if err := bo.dveManagerCmd.Process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("DVE Manager not running: %w", err)
		}
	}

	if bo.validationCoreCmd != nil && bo.validationCoreCmd.Process != nil {
		if err := bo.validationCoreCmd.Process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("Validation Core not running: %w", err)
		}
	}

	if bo.nexusServerCmd != nil && bo.nexusServerCmd.Process != nil {
		if err := bo.nexusServerCmd.Process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("Nexus Server not running: %w", err)
		}
	}

	return nil
}

func main() {
	// Print version information
	fmt.Printf("KNIRV-NEXUS Backend Orchestrator v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Create orchestrator
	orchestrator, err := NewBackendOrchestrator()
	if err != nil {
		log.Fatalf("Failed to create backend orchestrator: %v", err)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
		orchestrator.Stop()
	}()

	// Start all services
	if err := orchestrator.Start(); err != nil {
		log.Fatalf("Failed to start backend services: %v", err)
	}

	// Wait for shutdown signal or service exit
	select {
	case <-ctx.Done():
		log.Println("Shutdown requested")
	default:
		orchestrator.Wait()
	}

	log.Println("Backend orchestrator exited")
}
