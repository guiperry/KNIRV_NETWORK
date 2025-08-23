package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme defines the color scheme for the UI
type Theme struct {
	Primary      lipgloss.Color
	Secondary    lipgloss.Color
	Tertiary     lipgloss.Color
	Success      lipgloss.Color
	Warning      lipgloss.Color
	Error        lipgloss.Color
	Subtle       lipgloss.Color
	HighlightLow lipgloss.Color
	HighlightMed lipgloss.Color
	HighlightHi  lipgloss.Color
	Background   lipgloss.Color
	Text         lipgloss.Color
}

// Predefined themes
var (
	DefaultTheme = Theme{
		Primary:      lipgloss.Color("#7D56F4"),
		Secondary:    lipgloss.Color("#2D79C7"),
		Tertiary:     lipgloss.Color("#5A36EC"),
		Success:      lipgloss.Color("#73F59F"),
		Warning:      lipgloss.Color("#F2C94C"),
		Error:        lipgloss.Color("#F25757"),
		Subtle:       lipgloss.Color("#9B9B9B"),
		HighlightLow: lipgloss.Color("#3B3B3B"),
		HighlightMed: lipgloss.Color("#5C5C5C"),
		HighlightHi:  lipgloss.Color("#7D7D7D"),
		Background:   lipgloss.Color("#1E1E1E"),
		Text:         lipgloss.Color("#FFFFFF"),
	}

	DarkTheme = Theme{
		Primary:      lipgloss.Color("#BB86FC"),
		Secondary:    lipgloss.Color("#03DAC6"),
		Tertiary:     lipgloss.Color("#3700B3"),
		Success:      lipgloss.Color("#03DAC6"),
		Warning:      lipgloss.Color("#FFB74D"),
		Error:        lipgloss.Color("#CF6679"),
		Subtle:       lipgloss.Color("#6E6E6E"),
		HighlightLow: lipgloss.Color("#2D2D2D"),
		HighlightMed: lipgloss.Color("#3D3D3D"),
		HighlightHi:  lipgloss.Color("#4D4D4D"),
		Background:   lipgloss.Color("#121212"),
		Text:         lipgloss.Color("#FFFFFF"),
	}

	LightTheme = Theme{
		Primary:      lipgloss.Color("#6200EE"),
		Secondary:    lipgloss.Color("#03DAC6"),
		Tertiary:     lipgloss.Color("#3700B3"),
		Success:      lipgloss.Color("#4CAF50"),
		Warning:      lipgloss.Color("#FB8C00"),
		Error:        lipgloss.Color("#B00020"),
		Subtle:       lipgloss.Color("#9E9E9E"),
		HighlightLow: lipgloss.Color("#E0E0E0"),
		HighlightMed: lipgloss.Color("#BDBDBD"),
		HighlightHi:  lipgloss.Color("#9E9E9E"),
		Background:   lipgloss.Color("#FFFFFF"),
		Text:         lipgloss.Color("#000000"),
	}

	HighContrastTheme = Theme{
		Primary:      lipgloss.Color("#FFFF00"),
		Secondary:    lipgloss.Color("#00FFFF"),
		Tertiary:     lipgloss.Color("#FF00FF"),
		Success:      lipgloss.Color("#00FF00"),
		Warning:      lipgloss.Color("#FFFF00"),
		Error:        lipgloss.Color("#FF0000"),
		Subtle:       lipgloss.Color("#FFFFFF"),
		HighlightLow: lipgloss.Color("#333333"),
		HighlightMed: lipgloss.Color("#666666"),
		HighlightHi:  lipgloss.Color("#999999"),
		Background:   lipgloss.Color("#000000"),
		Text:         lipgloss.Color("#FFFFFF"),
	}
)

// Styles contains all the UI styles
type Styles struct {
	Theme Theme

	// Common styles
	App      lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Header   lipgloss.Style
	Text     lipgloss.Style
	Bold     lipgloss.Style
	Italic   lipgloss.Style
	Subtle   lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Info     lipgloss.Style

	// Container styles
	Panel         lipgloss.Style
	Dialog        lipgloss.Style
	DialogTitle   lipgloss.Style
	DialogContent lipgloss.Style
	DialogFooter  lipgloss.Style

	// Form styles
	Input           lipgloss.Style
	InputFocused    lipgloss.Style
	InputError      lipgloss.Style
	Label           lipgloss.Style
	Button          lipgloss.Style
	ButtonPrimary   lipgloss.Style
	ButtonSecondary lipgloss.Style
	ButtonDisabled  lipgloss.Style

	// List styles
	List             lipgloss.Style
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListTitle        lipgloss.Style
	ListDescription  lipgloss.Style

	// Table styles
	Table       lipgloss.Style
	TableHeader lipgloss.Style
	TableCell   lipgloss.Style
	TableRow    lipgloss.Style
	TableRowAlt lipgloss.Style
	TableBorder lipgloss.Style

	// Help styles
	HelpText   lipgloss.Style
	KeyBinding lipgloss.Style
}

// DefaultStyles returns the default styles
func DefaultStyles(theme Theme) Styles {
	s := Styles{
		Theme: theme,

		// Common styles
		App: lipgloss.NewStyle().
			Background(theme.Background).
			Foreground(theme.Text).
			Padding(1),

		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			MarginBottom(1).
			PaddingLeft(1).
			PaddingRight(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			MarginBottom(1),

		Header: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			PaddingBottom(1),

		Text: lipgloss.NewStyle().
			Foreground(theme.Text),

		Bold: lipgloss.NewStyle().
			Foreground(theme.Text).
			Bold(true),

		Italic: lipgloss.NewStyle().
			Foreground(theme.Text).
			Italic(true),

		Subtle: lipgloss.NewStyle().
			Foreground(theme.Subtle),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),

		Info: lipgloss.NewStyle().
			Foreground(theme.Secondary),

		// Container styles
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.HighlightMed).
			Padding(1).
			MarginRight(2),

		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(1),

		DialogTitle: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			PaddingBottom(1).
			Width(50),

		DialogContent: lipgloss.NewStyle().
			PaddingTop(1).
			PaddingBottom(1),

		DialogFooter: lipgloss.NewStyle().
			PaddingTop(1),

		// Form styles
		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.HighlightMed).
			Padding(0, 1),

		InputFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(0, 1),

		InputError: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Error).
			Padding(0, 1),

		Label: lipgloss.NewStyle().
			Foreground(theme.Text).
			Bold(true),

		Button: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.HighlightMed).
			Padding(0, 3).
			Margin(0, 1, 0, 0).
			Bold(true),

		ButtonPrimary: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Primary).
			Padding(0, 3).
			Margin(0, 1, 0, 0).
			Bold(true),

		ButtonSecondary: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Secondary).
			Padding(0, 3).
			Margin(0, 1, 0, 0).
			Bold(true),

		ButtonDisabled: lipgloss.NewStyle().
			Foreground(theme.Subtle).
			Background(theme.HighlightLow).
			Padding(0, 3).
			Margin(0, 1, 0, 0),

		// List styles
		List: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.HighlightMed).
			Padding(0),

		ListItem: lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2),

		ListItemSelected: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Primary).
			PaddingLeft(2).
			PaddingRight(2),

		ListTitle: lipgloss.NewStyle().
			Foreground(theme.Text).
			Bold(true),

		ListDescription: lipgloss.NewStyle().
			Foreground(theme.Subtle),

		// Table styles
		Table: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()),

		TableHeader: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1),

		TableCell: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1),

		TableRow: lipgloss.NewStyle(),

		TableRowAlt: lipgloss.NewStyle().
			Background(theme.HighlightLow),

		TableBorder: lipgloss.NewStyle().
			Foreground(theme.HighlightMed),

		// Help styles
		HelpText: lipgloss.NewStyle().
			Foreground(theme.Subtle),

		KeyBinding: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true),
	}

	return s
}

// GetThemeByName returns a theme by name
func GetThemeByName(name string) Theme {
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

// GetColorProfile returns the terminal color profile
func GetColorProfile(colorMode string) termenv.Profile {
	switch colorMode {
	case "16":
		return termenv.ANSI
	case "256":
		return termenv.ANSI256
	case "truecolor":
		return termenv.TrueColor
	default:
		return termenv.ColorProfile()
	}
}
