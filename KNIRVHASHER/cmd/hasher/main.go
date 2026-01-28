package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"hasher/internal/cli/server"
	"hasher/internal/cli/ui"
)

func main() {
	var serverCmd *exec.Cmd
	serverStarted := false

	// Clean up any existing llama-server processes
	server.CleanupExistingServer()

	// Set up signal handler for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create log channel for server output
	logChan := make(chan string, 100)

	// Handle server shutdown
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		server.ShutdownServer(serverCmd, serverStarted)
		os.Exit(0)
	}()

	// Check if Llama-LLM server is running
	if !server.IsServerRunning() {
		fmt.Println("Llama-LLM server is not running.")

		// Check if we have existing configuration
		config, err := server.LoadConfig()
		if err == nil {
			// Configuration exists, check if model file and server exist
			if _, err := os.Stat(config.ModelPath); err == nil {
				var serverPath string
				if config.ServerPath != "" {
					serverPath = config.ServerPath
				} else {
					serverPath = server.FindServerExecutable()
				}

				if serverPath != "" {
					fmt.Println("Found existing installation, starting server...")
					serverCmd = server.StartServerWithPath(config.ModelPath, serverPath, logChan)
					if serverCmd == nil {
						fmt.Println("Error starting server.")
						os.Exit(1)
					}
					serverStarted = true
				} else {
					fmt.Println("Server executable not found.")
					os.Exit(1)
				}
			} else {
				fmt.Println("Model file not found at configured path.")
				os.Exit(1)
			}
		} else {
			// No configuration, check if we can find existing installation
			foundModelPath := server.FindModelFile()
			foundServerPath := server.FindServerExecutable()

			if foundModelPath != "" && foundServerPath != "" {
				// Found existing installation, create config and start server
				fmt.Println("Found existing Llama-LLM installation, creating configuration...")
				config = &server.Config{ModelPath: foundModelPath, ServerPath: foundServerPath}
				if err := server.SaveConfig(config); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
					os.Exit(1)
				} else {
					fmt.Println("Starting server...")
					serverCmd = server.StartServerWithPath(foundModelPath, foundServerPath, logChan)
					if serverCmd == nil {
						fmt.Println("Error starting server.")
						os.Exit(1)
					}
					serverStarted = true
				}
			} else {
				fmt.Println("No existing installation found.")
				os.Exit(1)
			}
		}
	}

	// Wait for server to be ready
	if serverStarted {
		fmt.Println("Waiting for server to start...")
		startTime := time.Now()
		for time.Since(startTime) < server.ServerTimeout {
			if server.IsServerRunning() {
				fmt.Println("Server is ready!")
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !server.IsServerRunning() {
			fmt.Println("Server failed to start within timeout.")
			server.ShutdownServer(serverCmd, serverStarted)
			os.Exit(1)
		}
	}

	// Verify server is running after attempting to start
	if !server.IsServerRunning() {
		fmt.Println("Failed to start Llama-LLM server.")
		server.ShutdownServer(serverCmd, serverStarted)
		os.Exit(1)
	}

	// Create UI model and pass log channel
	model := ui.NewModel()
	model.ServerCmd = serverCmd
	// Set server ready flag
	model.ServerReady = true

	// Start the Bubble Tea UI with alternate screen and mouse support for clean display and text selection
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	// Start log listener and send log messages to program
	go func() {
		for log := range logChan {
			p.Send(ui.AppendLogMsg{Log: log})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		server.ShutdownServer(serverCmd, serverStarted)
		os.Exit(1)
	}

	// Ensure server is shut down if we exit the loop
	server.ShutdownServer(serverCmd, serverStarted)
}
