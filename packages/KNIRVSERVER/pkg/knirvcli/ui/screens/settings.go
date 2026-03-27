package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/ui/components"
)

// SettingsKeyMap defines keybindings for the settings screen
type SettingsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Save   key.Binding
	Reset  key.Binding
	Back   key.Binding
}

// DefaultSettingsKeyMap returns the default keybindings
func DefaultSettingsKeyMap() SettingsKeyMap {
	return SettingsKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", "space"),
			key.WithHelp("enter/space", "toggle"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
	}
}

// SettingItem represents a setting item
type SettingItem struct {
	name        string
	description string
	value       interface{}
	valueType   string // "bool", "string", "select"
	options     []string
	selected    int
}

// SettingsScreen represents the settings screen
type SettingsScreen struct {
	styles      ui.Styles
	keyMap      SettingsKeyMap
	width       int
	height      int
	cursor      int
	settings    []SettingItem
	saved       bool
	saveMessage string

	// Theme selector
	themeSelect components.Select

	// Parent screen
	parent ui.Screen
}

// NewSettingsScreen creates a new settings screen
func NewSettingsScreen(styles ui.Styles, parent ui.Screen) *SettingsScreen {
	keyMap := DefaultSettingsKeyMap()

	// Create theme selector
	themeOptions := []components.SelectOption{
		{Value: "default", Label: "Default"},
		{Value: "dark", Label: "Dark"},
		{Value: "light", Label: "Light"},
		{Value: "high-contrast", Label: "High Contrast"},
	}

	themeSelect := components.NewSelect(styles, "Theme", themeOptions, false)
	themeSelect.SetDescription("Select the UI theme")

	// Create settings
	settings := []SettingItem{
		{
			name:        "API URL",
			description: "URL of the KNIRVCHAIN API",
			value:       "https://api.knirvchain.net",
			valueType:   "string",
		},
		{
			name:        "Wallet Directory",
			description: "Directory for wallet storage",
			value:       "~/.knirvchain/wallets",
			valueType:   "string",
		},
		{
			name:        "Log Level",
			description: "Logging verbosity level",
			value:       "info",
			valueType:   "select",
			options:     []string{"debug", "info", "warn", "error"},
			selected:    1,
		},
		{
			name:        "Auto-Connect",
			description: "Automatically connect to the API on startup",
			value:       true,
			valueType:   "bool",
		},
		{
			name:        "Show Confirmations",
			description: "Show confirmation dialogs for destructive actions",
			value:       true,
			valueType:   "bool",
		},
		{
			name:        "Cache Timeout",
			description: "Cache timeout in seconds",
			value:       300,
			valueType:   "string",
		},
		{
			name:        "Enable Analytics",
			description: "Send anonymous usage data",
			value:       false,
			valueType:   "bool",
		},
		{
			name:        "Check for Updates",
			description: "Automatically check for updates",
			value:       true,
			valueType:   "bool",
		},
	}

	return &SettingsScreen{
		styles:      styles,
		keyMap:      keyMap,
		width:       80,
		height:      24,
		cursor:      0,
		settings:    settings,
		saved:       false,
		saveMessage: "",
		themeSelect: themeSelect,
		parent:      parent,
	}
}

// Init initializes the screen
func (s *SettingsScreen) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (s *SettingsScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.keyMap.Back):
			return s.parent, nil

		case key.Matches(msg, s.keyMap.Up):
			s.cursor--
			if s.cursor < 0 {
				s.cursor = len(s.settings) - 1
			}

		case key.Matches(msg, s.keyMap.Down):
			s.cursor++
			if s.cursor >= len(s.settings) {
				s.cursor = 0
			}

		case key.Matches(msg, s.keyMap.Toggle):
			// Toggle the current setting
			if s.cursor >= 0 && s.cursor < len(s.settings) {
				setting := &s.settings[s.cursor]

				switch setting.valueType {
				case "bool":
					setting.value = !setting.value.(bool)
				case "select":
					setting.selected = (setting.selected + 1) % len(setting.options)
					setting.value = setting.options[setting.selected]
				}

				s.saved = false
			}

		case key.Matches(msg, s.keyMap.Save):
			// Save settings
			s.saved = true
			s.saveMessage = "Settings saved successfully!"

		case key.Matches(msg, s.keyMap.Reset):
			// Reset settings to defaults
			s.settings = []SettingItem{
				{
					name:        "API URL",
					description: "URL of the KNIRVCHAIN API",
					value:       "https://api.knirvchain.net",
					valueType:   "string",
				},
				{
					name:        "Wallet Directory",
					description: "Directory for wallet storage",
					value:       "~/.knirvchain/wallets",
					valueType:   "string",
				},
				{
					name:        "Log Level",
					description: "Logging verbosity level",
					value:       "info",
					valueType:   "select",
					options:     []string{"debug", "info", "warn", "error"},
					selected:    1,
				},
				{
					name:        "Auto-Connect",
					description: "Automatically connect to the API on startup",
					value:       true,
					valueType:   "bool",
				},
				{
					name:        "Show Confirmations",
					description: "Show confirmation dialogs for destructive actions",
					value:       true,
					valueType:   "bool",
				},
				{
					name:        "Cache Timeout",
					description: "Cache timeout in seconds",
					value:       300,
					valueType:   "string",
				},
				{
					name:        "Enable Analytics",
					description: "Send anonymous usage data",
					value:       false,
					valueType:   "bool",
				},
				{
					name:        "Check for Updates",
					description: "Automatically check for updates",
					value:       true,
					valueType:   "bool",
				},
			}

			s.saved = false
			s.saveMessage = "Settings reset to defaults"
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	}

	// Update theme selector
	var cmd tea.Cmd
	s.themeSelect, cmd = s.themeSelect.Update(msg)

	return s, cmd
}

// View renders the screen
func (s SettingsScreen) View() string {
	var sb strings.Builder

	// Title
	sb.WriteString(s.styles.Title.Render("Settings"))
	sb.WriteString("\n\n")

	// Theme selector
	sb.WriteString(s.themeSelect.View())
	sb.WriteString("\n\n")

	// Settings
	for i, setting := range s.settings {
		// Determine if this setting is selected
		var nameStyle, valueStyle lipgloss.Style
		if i == s.cursor {
			nameStyle = s.styles.ListItemSelected
			valueStyle = s.styles.ListItemSelected
		} else {
			nameStyle = s.styles.ListItem
			valueStyle = s.styles.ListItem
		}

		// Render setting name and description
		sb.WriteString(nameStyle.Render(setting.name))
		sb.WriteString("\n")
		sb.WriteString(s.styles.Subtle.Render(setting.description))
		sb.WriteString("\n")

		// Render setting value
		switch setting.valueType {
		case "bool":
			if setting.value.(bool) {
				sb.WriteString(valueStyle.Render("● Enabled"))
			} else {
				sb.WriteString(valueStyle.Render("○ Disabled"))
			}
		case "string":
			sb.WriteString(valueStyle.Render(fmt.Sprintf("%v", setting.value)))
		case "select":
			var optionsStr strings.Builder
			for j, option := range setting.options {
				if j == setting.selected {
					optionsStr.WriteString(fmt.Sprintf("● %s ", option))
				} else {
					optionsStr.WriteString(fmt.Sprintf("○ %s ", option))
				}
			}
			sb.WriteString(valueStyle.Render(optionsStr.String()))
		}

		sb.WriteString("\n\n")
	}

	// Save message
	if s.saved {
		sb.WriteString(s.styles.Success.Render(s.saveMessage))
		sb.WriteString("\n\n")
	}

	// Help
	help := s.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: toggle • %s: save • %s: reset • %s: back",
		s.styles.KeyBinding.Render("↑/↓"),
		s.styles.KeyBinding.Render("enter/space"),
		s.styles.KeyBinding.Render("s"),
		s.styles.KeyBinding.Render("r"),
		s.styles.KeyBinding.Render("esc"),
	))

	sb.WriteString(help)

	return sb.String()
}
