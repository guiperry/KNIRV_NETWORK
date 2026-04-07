package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents a UI screen
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View() string
}

// App is the main application model
type App struct {
	styles      Styles
	screen      Screen
	width       int
	height      int
	initialized bool
	help        bool
	debug       bool
	theme       string
}

// NewApp creates a new application
func NewApp(theme string, debug bool) *App {
	return &App{
		styles:      DefaultStyles(GetThemeByName(theme)),
		screen:      nil,
		width:       80,
		height:      24,
		initialized: false,
		help:        false,
		debug:       debug,
		theme:       theme,
	}
}

// SetScreen sets the current screen
func (a *App) SetScreen(screen Screen) {
	a.screen = screen
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		a.screen.Init(),
	)
}

// Update handles user input
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "ctrl+h":
			a.help = !a.help
			return a, nil
		case "f1":
			// Toggle theme
			switch a.theme {
			case "default":
				a.theme = "dark"
			case "dark":
				a.theme = "light"
			case "light":
				a.theme = "high-contrast"
			case "high-contrast":
				a.theme = "default"
			default:
				a.theme = "default"
			}
			a.styles = DefaultStyles(GetThemeByName(a.theme))
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.initialized = true
	}

	// Update the current screen
	if a.screen != nil {
		newScreen, cmd := a.screen.Update(msg)
		a.screen = newScreen
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View renders the application
func (a *App) View() string {
	if !a.initialized {
		return "Initializing..."
	}

	var content string
	if a.screen != nil {
		content = a.screen.View()
	} else {
		content = "No screen set"
	}

	// Add help text if enabled
	if a.help {
		helpText := a.styles.HelpText.Render(`
Global Keybindings:
  q, Ctrl+c: Quit
  Ctrl+h: Toggle help
  F1: Toggle theme
`)
		content = lipgloss.JoinVertical(lipgloss.Left, content, helpText)
	}

	// Add debug info if enabled
	if a.debug {
		debugInfo := a.styles.Subtle.Render(fmt.Sprintf(
			"Window: %dx%d | Theme: %s",
			a.width, a.height, a.theme,
		))
		content = lipgloss.JoinVertical(lipgloss.Left, content, debugInfo)
	}

	return a.styles.App.Render(content)
}

// Run runs the application
func Run(theme string, debug bool, initialScreen Screen) error {
	app := NewApp(theme, debug)
	app.SetScreen(initialScreen)

	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
