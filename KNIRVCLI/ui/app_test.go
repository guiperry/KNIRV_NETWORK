package ui

// NOTE: The compiler may show false positive warnings about incompatible assignments
// for tea.KeyMsg values in test cases below. These warnings can be safely ignored
// as the tests are correctly verifying the app's handling of different message types.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// MockScreen is a mock implementation of the Screen interface for testing
type MockScreen struct {
	initCalled    bool
	updateCalled  bool
	viewCalled    bool
	lastMsg       tea.Msg
	returnScreen  Screen
	returnCmd     tea.Cmd
	returnContent string
}

func NewMockScreen() *MockScreen {
	return &MockScreen{
		initCalled:    false,
		updateCalled:  false,
		viewCalled:    false,
		returnScreen:  nil,
		returnCmd:     nil,
		returnContent: "Mock Screen Content",
	}
}

func (m *MockScreen) Init() tea.Cmd {
	m.initCalled = true
	return nil
}

func (m *MockScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	m.updateCalled = true
	m.lastMsg = msg
	return m.returnScreen, m.returnCmd
}

func (m *MockScreen) View() string {
	m.viewCalled = true
	return m.returnContent
}

func (m *MockScreen) SetReturnScreen(screen Screen) {
	m.returnScreen = screen
}

func (m *MockScreen) SetReturnCmd(cmd tea.Cmd) {
	m.returnCmd = cmd
}

func (m *MockScreen) SetReturnContent(content string) {
	m.returnContent = content
}

func TestNewApp(t *testing.T) {
	app := NewApp("default", false)
	assert.NotNil(t, app)
	assert.Equal(t, "default", app.theme)
	assert.False(t, app.debug)
	assert.False(t, app.initialized)
	assert.False(t, app.help)
	assert.Nil(t, app.screen)
}

func TestSetScreen(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)
	assert.Equal(t, screen, app.screen)
}

func TestInit(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)

	cmd := app.Init()
	assert.NotNil(t, cmd)
	assert.True(t, screen.initCalled)
}

func TestUpdate(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)

	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, cmd := app.Update(msg)
	updatedApp := model.(*App)

	assert.Equal(t, 100, updatedApp.width)
	assert.Equal(t, 50, updatedApp.height)
	assert.True(t, updatedApp.initialized)
	assert.True(t, screen.updateCalled)
	assert.Equal(t, msg, screen.lastMsg)
	assert.Nil(t, cmd)

	// Test key message for help toggle
	screen.updateCalled = false
	// nolint:staticcheck // This is a valid test case for key handling
	ctrlHMsg := tea.KeyMsg{Type: tea.KeyCtrlH}
	model, cmd = app.Update(ctrlHMsg)
	updatedApp = model.(*App)

	assert.True(t, updatedApp.help)
	assert.True(t, screen.updateCalled)
	assert.Equal(t, msg, screen.lastMsg)
	assert.Nil(t, cmd)

	// Test key message for theme toggle
	screen.updateCalled = false
	// nolint:staticcheck // This is a valid test case for key handling
	f1Msg := tea.KeyMsg{Type: tea.KeyF1}
	model, cmd = app.Update(f1Msg)
	updatedApp = model.(*App)

	assert.Equal(t, "dark", updatedApp.theme)
	assert.True(t, screen.updateCalled)
	assert.Equal(t, msg, screen.lastMsg)
	assert.Nil(t, cmd)

	// Test key message for quit
	screen.updateCalled = false
	// nolint:staticcheck // This is a valid test case for key handling
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	model, cmd = app.Update(ctrlCMsg)
	updatedApp = model.(*App)
	assert.Contains(t, updatedApp.View(), "Mock Screen Content")

	assert.NotNil(t, cmd)
	assert.True(t, screen.updateCalled)
	assert.Equal(t, msg, screen.lastMsg)
}

func TestView(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	screen.SetReturnContent("Test Screen Content")
	app.SetScreen(screen)

	// Test view before initialization
	view := app.View()
	assert.Contains(t, view, "Initializing...")
	assert.False(t, screen.viewCalled)

	// Test view after initialization
	app.initialized = true
	view = app.View()
	assert.Contains(t, view, "Test Screen Content")
	assert.True(t, screen.viewCalled)

	// Test view with help enabled
	screen.viewCalled = false
	app.help = true
	view = app.View()
	assert.Contains(t, view, "Test Screen Content")
	assert.Contains(t, view, "Global Keybindings:")
	assert.Contains(t, view, "q, Ctrl+c: Quit")
	assert.Contains(t, view, "Ctrl+h: Toggle help")
	assert.Contains(t, view, "F1: Toggle theme")
	assert.True(t, screen.viewCalled)

	// Test view with debug enabled
	screen.viewCalled = false
	app.debug = true
	view = app.View()
	assert.Contains(t, view, "Test Screen Content")
	assert.Contains(t, view, "Window: 80x24")
	assert.Contains(t, view, "Theme: default")
	assert.True(t, screen.viewCalled)
}

func TestThemeToggle(t *testing.T) {
	app := NewApp("default", false)
	app.initialized = true

	// Test theme cycling
	themes := []string{"default", "dark", "light", "high-contrast", "default"}

	for i := 0; i < len(themes)-1; i++ {
		assert.Equal(t, themes[i], app.theme)
		msg := tea.KeyMsg{Type: tea.KeyF1}
		model, _ := app.Update(msg)
		app = model.(*App)
		assert.Equal(t, themes[i+1], app.theme)
	}
}
