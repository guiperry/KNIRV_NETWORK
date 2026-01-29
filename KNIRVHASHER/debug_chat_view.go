package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"hasher/internal/cli/ui"
)

type debugModel struct {
	ui.Model
}

func (m debugModel) View() string {
	fmt.Printf("Debug: Width=%d, Height=%d\n", m.Width, m.Height)
	return m.Model.View()
}

func main() {
	// Create UI model
	model := ui.NewModel()
	model.ServerReady = true
	// Skip main menu and go directly to chat view
	model.CurrentView = ui.ChatView

	// Set fixed dimensions for debugging
	model.Width = 80
	model.Height = 24

	// Start the Bubble Tea UI
	p := tea.NewProgram(debugModel{model}, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
