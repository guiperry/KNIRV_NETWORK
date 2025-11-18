package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"KNIRVORACLE/config"
)

// NetworkMonitorManager manages the embedded network-monitor binary or embedded monitor
type NetworkMonitorManager struct {
	config          *config.Config
	cmd             *exec.Cmd
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.Mutex
	isRunning       bool
	port            int
	binaryPath      string
	embeddedMonitor *EmbeddedNetworkMonitor
	useEmbedded     bool
}

// NewNetworkMonitorManager creates a new network monitor manager
func NewNetworkMonitorManager(cfg *config.Config) *NetworkMonitorManager {
	// Default port for network monitor web interface
	port := 8090
	if cfg.Testnet.Enabled {
		// Use a different port for testnet to avoid conflicts
		port = 8091
	}

	// Determine binary path and whether to use embedded monitor
	binaryPath := filepath.Join("embedded", "binaries", "network", "knirv-network-monitor")
	useEmbedded := false

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try fallback paths
		binaryPath = filepath.Join("bin", "knirv-network-monitor")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			binaryPath = filepath.Join(".", "knirv-network-monitor")
			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				// Binary not found, use embedded monitor
				useEmbedded = true
				log.Printf("Network monitor binary not found, using embedded monitor")
			}
		}
	}

	nmm := &NetworkMonitorManager{
		config:      cfg,
		port:        port,
		binaryPath:  binaryPath,
		useEmbedded: useEmbedded,
	}

	if useEmbedded {
		nmm.embeddedMonitor = NewEmbeddedNetworkMonitor(cfg)
	}

	return nmm
}

// Start initializes and starts the network monitor in web mode
func (nmm *NetworkMonitorManager) Start() error {
	nmm.mu.Lock()
	defer nmm.mu.Unlock()

	if nmm.isRunning {
		return fmt.Errorf("network monitor is already running")
	}

	// Only start in testnet mode
	if !nmm.config.Testnet.Enabled {
		log.Printf("Network monitor: Skipping start (not in testnet mode)")
		return nil
	}

	if nmm.useEmbedded {
		// Use embedded monitor
		if err := nmm.embeddedMonitor.Start(); err != nil {
			return fmt.Errorf("failed to start embedded network monitor: %w", err)
		}
		nmm.isRunning = true
		log.Printf("Embedded network monitor started on port %d", nmm.embeddedMonitor.GetPort())
		return nil
	}

	// Check if binary exists
	if _, err := os.Stat(nmm.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("network monitor binary not found at %s", nmm.binaryPath)
	}

	// Create context for the network monitor process
	nmm.ctx, nmm.cancel = context.WithCancel(context.Background())

	// Prepare command arguments
	args := []string{
		"--headless",                     // Run in web mode without GUI
		"--port", strconv.Itoa(nmm.port), // Web interface port
		"--network", "testnet", // Monitor testnet
		"--debug", // Enable debug logging
	}

	// Create the command
	nmm.cmd = exec.CommandContext(nmm.ctx, nmm.binaryPath, args...)

	// Set environment variables
	nmm.cmd.Env = append(os.Environ(),
		"KNIRV_NETWORK=testnet",
		fmt.Sprintf("KNIRV_MONITOR_PORT=%d", nmm.port),
	)

	// Set working directory to KNIRVORACLE directory
	nmm.cmd.Dir = "."

	// Start the process
	if err := nmm.cmd.Start(); err != nil {
		nmm.cancel()
		return fmt.Errorf("failed to start network monitor: %w", err)
	}

	nmm.isRunning = true
	log.Printf("Network monitor started in web mode on port %d (PID: %d)", nmm.port, nmm.cmd.Process.Pid)

	// Monitor the process in a goroutine
	go nmm.monitorProcess()

	// Wait a moment for the web server to start
	time.Sleep(2 * time.Second)

	return nil
}

// Stop gracefully stops the network monitor
func (nmm *NetworkMonitorManager) Stop() error {
	nmm.mu.Lock()
	defer nmm.mu.Unlock()

	if !nmm.isRunning {
		return nil
	}

	log.Printf("Stopping network monitor...")

	if nmm.useEmbedded {
		// Stop embedded monitor
		if err := nmm.embeddedMonitor.Stop(); err != nil {
			log.Printf("Error stopping embedded network monitor: %v", err)
		}
		nmm.isRunning = false
		return nil
	}

	// Cancel the context to signal shutdown
	if nmm.cancel != nil {
		nmm.cancel()
	}

	// Wait for process to exit gracefully
	if nmm.cmd != nil && nmm.cmd.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- nmm.cmd.Wait()
		}()

		select {
		case <-done:
			log.Printf("Network monitor stopped gracefully")
		case <-time.After(10 * time.Second):
			log.Printf("Network monitor did not stop gracefully, forcing termination")
			if err := nmm.cmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill network monitor process: %v", err)
			}
		}
	}

	nmm.isRunning = false
	nmm.cmd = nil
	nmm.cancel = nil

	return nil
}

// IsRunning returns whether the network monitor is currently running
func (nmm *NetworkMonitorManager) IsRunning() bool {
	nmm.mu.Lock()
	defer nmm.mu.Unlock()
	return nmm.isRunning
}

// GetPort returns the port the network monitor web interface is running on
func (nmm *NetworkMonitorManager) GetPort() int {
	if nmm.useEmbedded && nmm.embeddedMonitor != nil {
		return nmm.embeddedMonitor.GetPort()
	}
	return nmm.port
}

// GetWebURL returns the URL for the network monitor web interface
func (nmm *NetworkMonitorManager) GetWebURL() string {
	return fmt.Sprintf("http://localhost:%d", nmm.port)
}

// monitorProcess monitors the network monitor process and handles unexpected exits
func (nmm *NetworkMonitorManager) monitorProcess() {
	if nmm.cmd == nil {
		return
	}

	err := nmm.cmd.Wait()

	nmm.mu.Lock()
	defer nmm.mu.Unlock()

	if nmm.isRunning {
		if err != nil {
			log.Printf("Network monitor process exited with error: %v", err)
		} else {
			log.Printf("Network monitor process exited normally")
		}
		nmm.isRunning = false
	}
}

// Restart stops and starts the network monitor
func (nmm *NetworkMonitorManager) Restart() error {
	if err := nmm.Stop(); err != nil {
		return fmt.Errorf("failed to stop network monitor: %w", err)
	}

	// Wait a moment before restarting
	time.Sleep(1 * time.Second)

	if err := nmm.Start(); err != nil {
		return fmt.Errorf("failed to start network monitor: %w", err)
	}

	return nil
}

// GetStatus returns the current status of the network monitor
func (nmm *NetworkMonitorManager) GetStatus() map[string]interface{} {
	nmm.mu.Lock()
	defer nmm.mu.Unlock()

	status := map[string]interface{}{
		"running":     nmm.isRunning,
		"port":        nmm.port,
		"web_url":     nmm.GetWebURL(),
		"binary_path": nmm.binaryPath,
	}

	if nmm.cmd != nil && nmm.cmd.Process != nil {
		status["pid"] = nmm.cmd.Process.Pid
	}

	return status
}
