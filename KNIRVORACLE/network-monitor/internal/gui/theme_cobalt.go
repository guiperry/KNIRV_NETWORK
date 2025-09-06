package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// CobaltTheme represents a dark cobalt blue theme
type CobaltTheme struct{}

func (t *CobaltTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Dark cobalt blue color palette
	cobaltDark := &color.NRGBA{R: 0, G: 29, B: 51, A: 255}     // #001D33 - Primary dark
	cobaltPrimary := &color.NRGBA{R: 0, G: 71, B: 119, A: 255} // #004777 - Primary
	cobaltAccent := &color.NRGBA{R: 0, G: 120, B: 191, A: 255} // #0078BF - Accent
	cobaltLight := &color.NRGBA{R: 102, G: 153, B: 204, A: 255} // #6699CC - Light

	switch name {
	case theme.ColorNameBackground:
		return cobaltDark
	case theme.ColorNameForeground:
		return &color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light text
	case theme.ColorNamePrimary:
		return cobaltPrimary
	case theme.ColorNameFocus:
		return cobaltAccent
	case theme.ColorNameHover:
		return cobaltLight
	case theme.ColorNameInputBackground:
		return &color.NRGBA{R: 20, G: 49, B: 71, A: 255} // #143147 - Input background
	case theme.ColorNamePlaceHolder:
		return &color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	case theme.ColorNamePressed:
		return cobaltAccent
	case theme.ColorNameSelection:
		return cobaltAccent
	case theme.ColorNameScrollBar:
		return cobaltPrimary
	case theme.ColorNameShadow:
		return &color.NRGBA{R: 0, G: 0, B: 0, A: 128}
	case theme.ColorNameSuccess:
		return &color.NRGBA{R: 46, G: 204, B: 113, A: 255} // Green for success
	case theme.ColorNameWarning:
		return &color.NRGBA{R: 241, G: 196, B: 15, A: 255} // Yellow for warning
	case theme.ColorNameError:
		return &color.NRGBA{R: 231, G: 76, B: 60, A: 255} // Red for error
	case theme.ColorNameDisabled:
		return &color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	case theme.ColorNameDisabledButton:
		return &color.NRGBA{R: 80, G: 80, B: 80, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *CobaltTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *CobaltTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *CobaltTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// NewCobaltTheme creates a new cobalt blue theme instance
func NewCobaltTheme() fyne.Theme {
	return &CobaltTheme{}
}