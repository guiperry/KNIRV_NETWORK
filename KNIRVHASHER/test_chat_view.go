package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

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

	// Handle server shutdown
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		server.ShutdownServer(serverCmd, serverStarted)
		os.Exit(0)
	}()

	// Create UI model and pass log channel
	model := ui.NewModel()
	model.ServerCmd = serverCmd
	model.ServerReady = true
	// Skip main menu and go directly to chat view
	model.CurrentView = ui.ChatView

	// Start the Bubble Tea UI with alternate screen for clean display
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		server.ShutdownServer(serverCmd, serverStarted)
		os.Exit(1)
	}
}
