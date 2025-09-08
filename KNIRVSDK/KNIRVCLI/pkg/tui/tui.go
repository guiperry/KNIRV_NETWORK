package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the TUI
type Theme struct {
	Primary      lipgloss.Color
	Secondary    lipgloss.Color
	Tertiary     lipgloss.Color
	Success      lipgloss.Color
	Warning      lipgloss.Color
	Error        lipgloss.Color
	Text         lipgloss.Color
	TextDim      lipgloss.Color
	Background   lipgloss.Color
	BorderColor  lipgloss.Color
	SpinnerColor lipgloss.Color
}

// DefaultTheme is the default color scheme
var DefaultTheme = Theme{
	Primary:      lipgloss.Color("69"),
	Secondary:    lipgloss.Color("39"),
	Tertiary:     lipgloss.Color("99"),
	Success:      lipgloss.Color("42"),
	Warning:      lipgloss.Color("208"),
	Error:        lipgloss.Color("196"),
	Text:         lipgloss.Color("252"),
	TextDim:      lipgloss.Color("246"),
	Background:   lipgloss.Color("236"),
	BorderColor:  lipgloss.Color("240"),
	SpinnerColor: lipgloss.Color("69"),
}

// DarkTheme is a dark color scheme
var DarkTheme = Theme{
	Primary:      lipgloss.Color("69"),
	Secondary:    lipgloss.Color("39"),
	Tertiary:     lipgloss.Color("99"),
	Success:      lipgloss.Color("42"),
	Warning:      lipgloss.Color("208"),
	Error:        lipgloss.Color("196"),
	Text:         lipgloss.Color("252"),
	TextDim:      lipgloss.Color("246"),
	Background:   lipgloss.Color("234"),
	BorderColor:  lipgloss.Color("238"),
	SpinnerColor: lipgloss.Color("69"),
}

// LightTheme is a light color scheme
var LightTheme = Theme{
	Primary:      lipgloss.Color("27"),
	Secondary:    lipgloss.Color("33"),
	Tertiary:     lipgloss.Color("90"),
	Success:      lipgloss.Color("28"),
	Warning:      lipgloss.Color("166"),
	Error:        lipgloss.Color("160"),
	Text:         lipgloss.Color("234"),
	TextDim:      lipgloss.Color("240"),
	Background:   lipgloss.Color("255"),
	BorderColor:  lipgloss.Color("250"),
	SpinnerColor: lipgloss.Color("27"),
}

// HighContrastTheme is a high contrast color scheme
var HighContrastTheme = Theme{
	Primary:      lipgloss.Color("51"),
	Secondary:    lipgloss.Color("201"),
	Tertiary:     lipgloss.Color("226"),
	Success:      lipgloss.Color("46"),
	Warning:      lipgloss.Color("214"),
	Error:        lipgloss.Color("196"),
	Text:         lipgloss.Color("15"),
	TextDim:      lipgloss.Color("251"),
	Background:   lipgloss.Color("16"),
	BorderColor:  lipgloss.Color("15"),
	SpinnerColor: lipgloss.Color("51"),
}

// GetTheme returns the theme based on the name
func GetTheme(name string) Theme {
	switch name {
	case "dark":
		return DarkTheme
	case "light":
		return LightTheme
	case "high-contrast":
		return HighContrastTheme
	default:
		return DefaultTheme
	}
}

// Styles defines the styles for the TUI
type Styles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Text        lipgloss.Style
	TextDim     lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	Border      lipgloss.Style
	BorderRound lipgloss.Style
	Spinner     lipgloss.Style
	TextInput   lipgloss.Style
	Button      lipgloss.Style
	ButtonHover lipgloss.Style
}

// NewStyles creates a new Styles instance with the given theme
func NewStyles(theme Theme) Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true),
		Text: lipgloss.NewStyle().
			Foreground(theme.Text),
		TextDim: lipgloss.NewStyle().
			Foreground(theme.TextDim),
		Success: lipgloss.NewStyle().
			Foreground(theme.Success),
		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),
		Error: lipgloss.NewStyle().
			Foreground(theme.Error),
		Border: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.BorderColor).
			Padding(1),
		BorderRound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderColor).
			Padding(1),
		Spinner: lipgloss.NewStyle().
			Foreground(theme.SpinnerColor),
		TextInput: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.BorderColor).
			Padding(0, 1),
		Button: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Primary).
			Padding(0, 3).
			Margin(1, 1, 0, 0),
		ButtonHover: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Secondary).
			Padding(0, 3).
			Margin(1, 1, 0, 0),
	}
}

// NewSpinner creates a new spinner with the given theme
func NewSpinner(theme Theme) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.SpinnerColor)
	return s
}

// NewTextInput creates a new text input with the given theme
func NewTextInput(theme Theme, placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.TextDim)
	return ti
}

// RunTUI runs a bubbletea program
func RunTUI(model tea.Model) error {
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

// ConfirmationModel is a model for confirmation dialogs
type ConfirmationModel struct {
	Styles    Styles
	Question  string
	YesText   string
	NoText    string
	YesAction func() error
	NoAction  func() error
	Result    bool
	Quitting  bool
}

// NewConfirmationModel creates a new confirmation model
func NewConfirmationModel(theme Theme, question, yesText, noText string, yesAction, noAction func() error) ConfirmationModel {
	return ConfirmationModel{
		Styles:    NewStyles(theme),
		Question:  question,
		YesText:   yesText,
		NoText:    noText,
		YesAction: yesAction,
		NoAction:  noAction,
		Result:    false,
		Quitting:  false,
	}
}

// Init initializes the model
func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update updates the model
func (m ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.Result = true
			m.Quitting = true
			return m, tea.Quit
		case "n", "N", "q", "Q", "ctrl+c", "esc":
			m.Result = false
			m.Quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m ConfirmationModel) View() string {
	if m.Quitting {
		return ""
	}
	return fmt.Sprintf(
		"%s\n\n%s / %s",
		m.Styles.Text.Render(m.Question),
		m.Styles.Button.Render(m.YesText),
		m.Styles.Button.Render(m.NoText),
	)
}
