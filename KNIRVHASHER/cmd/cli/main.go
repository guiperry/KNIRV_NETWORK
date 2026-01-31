package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"hasher/internal/analyzer"
	"hasher/internal/cli/embedded"
	"hasher/internal/cli/ui"
)

func main() {
	var hasherHostCmd *exec.Cmd
	hasherHostStarted := false

	// Initialize embedded binaries
	fmt.Println("Hasher CLI starting...")
	initEmbeddedBinaries()

	// Set up signal handler for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create log channel for server output
	logChan := make(chan string, 100)

	// Create UI model and pass log channel
	model := ui.NewModel()

	// Handle server shutdown with ASIC cleanup
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		cleanupASICDevice(model.Deployer)
		shutdownHasherHost(hasherHostCmd, hasherHostStarted)
		os.Exit(0)
	}()

	// Try to start hasher-host orchestrator
	hasherHostCmd, hasherHostStarted = startHasherHost(logChan)

	model.ServerCmd = hasherHostCmd
	model.ServerReady = hasherHostStarted

	// Start the Bubble Tea UI with alternate screen and mouse support
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Start log listener and send log messages to program
	go func() {
		for log := range logChan {
			p.Send(ui.AppendLogMsg{Log: log})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cleanupASICDevice(model.Deployer)
		os.Exit(1)
	}

	// Ensure cleanup when exiting normally
	cleanupASICDevice(model.Deployer)
}

// initEmbeddedBinaries extracts embedded binaries to app data directory
func initEmbeddedBinaries() {
	binDir, err := embedded.GetBinDir()
	if err != nil {
		fmt.Printf("Warning: Could not determine binary directory: %v\n", err)
		return
	}

	// Check for embedded binaries
	binaries := embedded.CheckEmbeddedBinaries()
	embeddedCount := 0
	for _, b := range binaries {
		if b.Embedded {
			embeddedCount++
		}
	}

	if embeddedCount > 0 {
		fmt.Printf("Extracting %d embedded binaries to %s...\n", embeddedCount, binDir)
		if err := embedded.EnsureExtracted(); err != nil {
			fmt.Printf("Warning: Could not extract binaries: %v\n", err)
		} else {
			fmt.Println("Binaries extracted successfully.")
		}
	} else {
		fmt.Printf("No embedded binaries found. Binary directory: %s\n", binDir)
		fmt.Println("Build binaries with: make build-all")
	}
}

// startHasherHost attempts to start the hasher-host orchestrator
func startHasherHost(logChan chan string) (*exec.Cmd, bool) {
	// Try to find hasher-host binary
	hostPath, err := embedded.GetHasherHostPath()
	if err != nil {
		// Try looking in the working directory
		hostPath = findHasherHostExecutable()
		if hostPath == "" {
			fmt.Println("hasher-host not found. Run Discovery to provision ASIC devices.")
			return nil, false
		}
	}

	// Check if hasher-host is already running on the API port
	if isHasherHostRunning() {
		fmt.Println("hasher-host is already running.")
		return nil, true
	}

	fmt.Printf("Starting hasher-host from %s...\n", hostPath)

	// Start hasher-host in API mode
	cmd := exec.Command(hostPath, "--mode=api", "--port=8080")

	// Create pipes to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logChan <- fmt.Sprintf("Error creating stdout pipe: %v", err)
		return nil, false
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logChan <- fmt.Sprintf("Error creating stderr pipe: %v", err)
		return nil, false
	}

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("Error starting hasher-host: %v", err)
		return nil, false
	}

	logChan <- fmt.Sprintf("hasher-host started with PID %d", cmd.Process.Pid)

	// Capture stdout
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logChan <- string(buf[:n])
			}
		}
	}()

	// Capture stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logChan <- string(buf[:n])
			}
		}
	}()

	// Wait for server to be ready
	fmt.Println("Waiting for hasher-host to start...")
	startTime := time.Now()
	timeout := 10 * time.Second
	for time.Since(startTime) < timeout {
		if isHasherHostRunning() {
			fmt.Println("hasher-host is ready!")
			return cmd, true
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("hasher-host startup timed out, continuing without orchestrator.")
	return cmd, false
}

// isHasherHostRunning checks if hasher-host API is responding
func isHasherHostRunning() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8008/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// findHasherHostExecutable searches for hasher-host in common locations
func findHasherHostExecutable() string {
	// Check app data directory
	binDir, err := embedded.GetBinDir()
	if err == nil {
		paths := []string{
			binDir + "/hasher-host",
			binDir + "/hasher-host-linux-amd64",
			binDir + "/hasher-host-darwin-amd64",
			binDir + "/hasher-host-darwin-arm64",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Check current directory and common paths
	paths := []string{
		"./hasher-host",
		"./bin/hasher-host",
		"../hasher-host",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// cleanupASICDevice removes deployed binaries from the ASIC device
func cleanupASICDevice(deployer *analyzer.Deployer) {
	if deployer == nil {
		return
	}

	device := deployer.GetActiveDevice()
	if device == nil {
		return
	}

	fmt.Printf("Cleaning up binaries from ASIC device %s...\n", device.IPAddress)

	// Try to connect and cleanup
	if err := deployer.Connect(); err != nil {
		fmt.Printf("Could not connect to device for cleanup: %v\n", err)
		return
	}
	defer deployer.Disconnect()

	// Cleanup removes hasher-server and any temporary files
	if err := deployer.Cleanup(); err != nil {
		fmt.Printf("Cleanup warning: %v\n", err)
	} else {
		fmt.Println("ASIC device cleanup complete.")
	}
}

// shutdownHasherHost gracefully shuts down the hasher-host process
func shutdownHasherHost(cmd *exec.Cmd, started bool) {
	if started && cmd != nil && cmd.Process != nil {
		fmt.Println("Shutting down hasher-host...")
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Force kill if SIGTERM fails
			cmd.Process.Kill()
		}

		// Wait for process to terminate with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			fmt.Println("hasher-host shut down successfully.")
		case <-time.After(5 * time.Second):
			fmt.Println("hasher-host shutdown timeout, force killing...")
			cmd.Process.Kill()
		}
	}
}
