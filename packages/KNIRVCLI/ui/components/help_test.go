package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewHelpModel(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	assert.Equal(t, styles, help.styles)
	assert.Equal(t, 0, len(help.sections))
	assert.False(t, help.visible)
	assert.Equal(t, 80, help.width)
	assert.Equal(t, 24, help.height)
}

func TestAddSection(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	bindings := []KeyBinding{
		{Key: "ctrl+c", Description: "Quit"},
		{Key: "enter", Description: "Select"},
	}

	help.AddSection("Global", bindings)

	assert.Equal(t, 1, len(help.sections))
	assert.Equal(t, "Global", help.sections[0].Title)
	assert.Equal(t, 2, len(help.sections[0].Bindings))
	assert.Equal(t, "ctrl+c", help.sections[0].Bindings[0].Key)
	assert.Equal(t, "Quit", help.sections[0].Bindings[0].Description)
	assert.Equal(t, "enter", help.sections[0].Bindings[1].Key)
	assert.Equal(t, "Select", help.sections[0].Bindings[1].Description)
}

func TestAddKeyBindingMap(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	// Create a key map
	keyMap := map[string]key.Binding{
		"quit": key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		"select": key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	}

	help.AddKeyBindingMap("Global", keyMap)

	assert.Equal(t, 1, len(help.sections))
	assert.Equal(t, "Global", help.sections[0].Title)
	assert.Equal(t, 2, len(help.sections[0].Bindings))
}

func TestShow(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	help.Show()
	assert.True(t, help.visible)
}

func TestHide(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	help.visible = true
	help.Hide()
	assert.False(t, help.visible)
}

func TestToggle(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	// Toggle from false to true
	help.Toggle()
	assert.True(t, help.visible)

	// Toggle from true to false
	help.Toggle()
	assert.False(t, help.visible)
}

func TestIsVisible(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	assert.False(t, help.IsVisible())

	help.visible = true
	assert.True(t, help.IsVisible())
}

func TestHelpSetSize(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	help.SetSize(100, 50)
	assert.Equal(t, 100, help.width)
	assert.Equal(t, 50, help.height)
}

func TestHelpUpdate(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	// Test update when not visible
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedHelp, cmd := help.Update(msg)

	assert.False(t, updatedHelp.visible)
	assert.Nil(t, cmd)

	// Test update when visible
	help.visible = true

	// Test escape key
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedHelp, cmd = help.Update(msg)

	assert.False(t, updatedHelp.visible)
	assert.Nil(t, cmd)

	// Test q key
	help.visible = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedHelp, cmd = help.Update(msg)

	assert.False(t, updatedHelp.visible)
	assert.Nil(t, cmd)

	// Test ? key
	help.visible = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedHelp, cmd = help.Update(msg)

	assert.False(t, updatedHelp.visible)
	assert.Nil(t, cmd)

	// Test h key
	help.visible = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updatedHelp, cmd = help.Update(msg)

	assert.False(t, updatedHelp.visible)
	assert.Nil(t, cmd)
}

func TestHelpView(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)
	help := NewHelpModel(styles)

	// Test view when not visible
	view := help.View()
	assert.Equal(t, "", view)

	// Test view when visible
	help.visible = true

	// Add some sections
	help.AddSection("Global", []KeyBinding{
		{Key: "ctrl+c", Description: "Quit"},
		{Key: "enter", Description: "Select"},
	})

	help.AddSection("Navigation", []KeyBinding{
		{Key: "up/k", Description: "Move up"},
		{Key: "down/j", Description: "Move down"},
	})

	view = help.View()

	// Check that the view contains all the expected elements
	assert.Contains(t, view, "Keyboard Shortcuts")
	assert.Contains(t, view, "Global")
	assert.Contains(t, view, "ctrl+c")
	assert.Contains(t, view, "Quit")
	assert.Contains(t, view, "enter")
	assert.Contains(t, view, "Select")
	assert.Contains(t, view, "Navigation")
	assert.Contains(t, view, "up/k")
	assert.Contains(t, view, "Move up")
	assert.Contains(t, view, "down/j")
	assert.Contains(t, view, "Move down")
	assert.Contains(t, view, "Press ESC, q, ?, or h to close")
}
