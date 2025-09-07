package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiperry/KNIRVCHAIN-CLI/ui"
)

// KeyBinding represents a key binding with description
type KeyBinding struct {
	Key         string
	Description string
}

// HelpSection represents a section of help text
type HelpSection struct {
	Title    string
	Bindings []KeyBinding
}

// HelpModel represents a help overlay
type HelpModel struct {
	styles   ui.Styles
	sections []HelpSection
	visible  bool
	width    int
	height   int
}

// NewHelpModel creates a new help model
func NewHelpModel(styles ui.Styles) HelpModel {
	return HelpModel{
		styles:   styles,
		sections: []HelpSection{},
		visible:  false,
		width:    80,
		height:   24,
	}
}

// AddSection adds a section to the help model
func (m *HelpModel) AddSection(title string, bindings []KeyBinding) {
	m.sections = append(m.sections, HelpSection{
		Title:    title,
		Bindings: bindings,
	})
}

// AddKeyBindingMap adds a key binding map to the help model
func (m *HelpModel) AddKeyBindingMap(title string, keyMap interface{}) {
	var bindings []KeyBinding

	// Use reflection to extract key bindings from the key map
	switch km := keyMap.(type) {
	case map[string]key.Binding:
		for _, binding := range km {
			bindings = append(bindings, KeyBinding{
				Key:         binding.Help().Key,
				Description: binding.Help().Desc,
			})
		}
	default:
		// Try to handle other types of key maps
		// This is a simplified implementation
		return
	}

	m.AddSection(title, bindings)
}

// Show shows the help overlay
func (m *HelpModel) Show() {
	m.visible = true
}

// Hide hides the help overlay
func (m *HelpModel) Hide() {
	m.visible = false
}

// Toggle toggles the visibility of the help overlay
func (m *HelpModel) Toggle() {
	m.visible = !m.visible
}

// IsVisible returns whether the help overlay is visible
func (m *HelpModel) IsVisible() bool {
	return m.visible
}

// SetSize sets the size of the help overlay
func (m *HelpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles user input
func (m *HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?", "h":
			m.visible = false
			return *m, nil
		}
	}

	return *m, nil
}

// View renders the help overlay
func (m HelpModel) View() string {
	if !m.visible {
		return ""
	}

	var sb strings.Builder

	// Title
	sb.WriteString(m.styles.DialogTitle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	// Sections
	for _, section := range m.sections {
		sb.WriteString(m.styles.Bold.Render(section.Title))
		sb.WriteString("\n")

		// Calculate the maximum key length for alignment
		maxKeyLen := 0
		for _, binding := range section.Bindings {
			if len(binding.Key) > maxKeyLen {
				maxKeyLen = len(binding.Key)
			}
		}

		// Bindings
		for _, binding := range section.Bindings {
			key := m.styles.KeyBinding.Render(binding.Key)
			desc := m.styles.Text.Render(binding.Description)
			padding := strings.Repeat(" ", maxKeyLen-len(binding.Key)+2)
			sb.WriteString(fmt.Sprintf("  %s%s%s\n", key, padding, desc))
		}

		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(m.styles.Subtle.Render("Press ESC, q, ?, or h to close"))

	// Render in a dialog
	content := sb.String()
	dialog := m.styles.Dialog.Copy().Width(m.width - 20).Render(content)

	// Center the dialog
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}
