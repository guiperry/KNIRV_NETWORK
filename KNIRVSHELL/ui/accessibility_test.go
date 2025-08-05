package ui

import (
	"fmt"
	"math"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// TestColorContrast tests the color contrast of the UI themes
func TestColorContrast(t *testing.T) {
	// Test all themes
	themes := []struct {
		name  string
		theme Theme
	}{
		{"default", DefaultTheme},
		{"dark", DarkTheme},
		{"light", LightTheme},
		{"high-contrast", HighContrastTheme},
	}

	for _, tc := range themes {
		t.Run(tc.name, func(t *testing.T) {
			// Test text on background
			textContrast := getContrastRatio(tc.theme.Text, tc.theme.Background)
			assert.GreaterOrEqual(t, textContrast, 4.5, "Text on background contrast ratio should be at least 4.5:1")

			// Test primary on background
			primaryContrast := getContrastRatio(tc.theme.Primary, tc.theme.Background)
			assert.GreaterOrEqual(t, primaryContrast, 3.0, "Primary on background contrast ratio should be at least 3:1")

			// Test error on background
			errorContrast := getContrastRatio(tc.theme.Error, tc.theme.Background)
			assert.GreaterOrEqual(t, errorContrast, 3.0, "Error on background contrast ratio should be at least 3:1")
		})
	}
}

// TestKeyboardNavigation tests keyboard navigation in the UI
func TestKeyboardNavigation(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)

	// Test tab navigation
	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, _ = app.Update(msg)

	// Test shift+tab navigation
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	_, _ = app.Update(msg)

	// Test arrow key navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	_, _ = app.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyDown}
	_, _ = app.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyLeft}
	_, _ = app.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyRight}
	_, _ = app.Update(msg)

	// No assertions here, just making sure the app doesn't crash
	// In a real test, we would check that focus moves correctly
}

// TestScreenReaderCompatibility tests screen reader compatibility
func TestScreenReaderCompatibility(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)
	app.initialized = true

	// Get the view
	view := app.View()

	// Check that the view doesn't contain any ASCII art or other non-text content
	assert.NotContains(t, view, "█")
	assert.NotContains(t, view, "▓")
	assert.NotContains(t, view, "▒")
	assert.NotContains(t, view, "░")

	// Check that the view doesn't rely solely on color for conveying information
	// This is hard to test automatically, but we can check for common patterns
	assert.NotContains(t, view, "red")
	assert.NotContains(t, view, "green")
	assert.NotContains(t, view, "blue")
	assert.NotContains(t, view, "yellow")
}

// TestResponsiveLayout tests responsive layout
func TestResponsiveLayout(t *testing.T) {
	app := NewApp("default", false)
	screen := NewMockScreen()
	app.SetScreen(screen)

	// Test different terminal sizes
	sizes := []struct {
		width  int
		height int
	}{
		{40, 10},  // Very small
		{80, 24},  // Standard
		{120, 40}, // Large
		{200, 60}, // Very large
	}

	for _, size := range sizes {
		t.Run(
			"size_"+fmt.Sprintf("%dx%d", size.width, size.height),
			func(t *testing.T) {
				msg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
				model, _ := app.Update(msg)
				updatedApp := model.(*App)

				assert.Equal(t, size.width, updatedApp.width)
				assert.Equal(t, size.height, updatedApp.height)
				assert.True(t, updatedApp.initialized)

				// No assertions on the view content, just making sure it doesn't crash
				_ = updatedApp.View()
			},
		)
	}
}

// TestColorModes tests different color modes
func TestColorModes(t *testing.T) {
	// Test all color modes
	modes := []struct {
		name    string
		mode    string
		profile termenv.Profile
	}{
		{"16 colors", "16", termenv.ANSI},
		{"256 colors", "256", termenv.ANSI256},
		{"truecolor", "truecolor", termenv.TrueColor},
	}

	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			profile := GetColorProfile(tc.mode)
			assert.Equal(t, tc.profile, profile)

			// Test that styles work with this profile
			styles := DefaultStyles(DefaultTheme)
			_ = styles.Text.Render("Test")
			_ = styles.Bold.Render("Test")
			_ = styles.Error.Render("Test")
		})
	}
}

// Helper function to calculate contrast ratio between two colors
func getContrastRatio(color1, color2 lipgloss.Color) float64 {
	// Convert lipgloss.Color to RGB
	rgb1 := hexToRGB(string(color1))
	rgb2 := hexToRGB(string(color2))

	// Calculate luminance
	l1 := getLuminance(rgb1[0], rgb1[1], rgb1[2])
	l2 := getLuminance(rgb2[0], rgb2[1], rgb2[2])

	// Calculate contrast ratio
	if l1 > l2 {
		return (l1 + 0.05) / (l2 + 0.05)
	}
	return (l2 + 0.05) / (l1 + 0.05)
}

// Helper function to convert hex color to RGB
func hexToRGB(hex string) [3]float64 {
	if hex[0] == '#' {
		hex = hex[1:]
	}

	// Handle shorthand hex
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	}

	// Parse RGB values
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	return [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0}
}

// Helper function to calculate luminance
func getLuminance(r, g, b float64) float64 {
	// Convert RGB to linear RGB
	r = toLinear(r)
	g = toLinear(g)
	b = toLinear(b)

	// Calculate luminance
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// Helper function to convert RGB to linear RGB
func toLinear(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
