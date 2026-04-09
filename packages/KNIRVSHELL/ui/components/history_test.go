package components

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
	"github.com/stretchr/testify/assert"
)

func TestCommandEntry(t *testing.T) {
	entry := CommandEntry{
		Command:   "wallet",
		Args:      []string{"new", "--password", "secret"},
		Timestamp: time.Now(),
		Favorite:  false,
	}

	assert.Equal(t, "wallet", entry.Title())
	assert.Equal(t, "new --password secret", entry.Description())
	assert.Equal(t, "wallet", entry.FilterValue())

	entry.Favorite = true
	assert.Equal(t, "★ wallet", entry.Title())
}

func TestNewCommandHistory(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)
	assert.NotNil(t, history)

	assert.Equal(t, styles, history.styles)
	assert.Equal(t, historyFile, history.historyFile)
	assert.Equal(t, 100, history.maxEntries)
	assert.False(t, history.visible)
	assert.Equal(t, 80, history.width)
	assert.Equal(t, 24, history.height)
	assert.False(t, history.showFavorites)
	assert.Equal(t, 0, len(history.entries))
}

func TestAddEntry(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Add an entry
	err = history.AddEntry("wallet", []string{"new", "--password", "secret"})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(history.entries))
	assert.Equal(t, "wallet", history.entries[0].Command)
	assert.Equal(t, []string{"new", "--password", "secret"}, history.entries[0].Args)
	assert.False(t, history.entries[0].Favorite)

	// Add another entry
	err = history.AddEntry("wallet", []string{"list"})
	assert.NoError(t, err)

	assert.Equal(t, 2, len(history.entries))
	assert.Equal(t, "wallet", history.entries[0].Command)
	assert.Equal(t, []string{"list"}, history.entries[0].Args)

	// Add a duplicate entry (should update timestamp and move to top)
	time.Sleep(10 * time.Millisecond) // Ensure timestamp is different
	err = history.AddEntry("wallet", []string{"new", "--password", "secret"})
	assert.NoError(t, err)

	assert.Equal(t, 2, len(history.entries))
	assert.Equal(t, "wallet", history.entries[0].Command)
	assert.Equal(t, []string{"new", "--password", "secret"}, history.entries[0].Args)

	// Verify the file was created
	_, err = os.Stat(historyFile)
	assert.NoError(t, err)
}

func TestToggleFavorite(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Add entries
	history.AddEntry("wallet", []string{"new"})
	history.AddEntry("wallet", []string{"list"})

	// Toggle favorite
	err = history.ToggleFavorite(0)
	assert.NoError(t, err)

	assert.True(t, history.entries[0].Favorite)

	// Toggle back
	err = history.ToggleFavorite(0)
	assert.NoError(t, err)

	assert.False(t, history.entries[0].Favorite)

	// Test invalid index
	err = history.ToggleFavorite(100)
	assert.Error(t, err)
}

func TestDeleteEntry(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Add entries
	history.AddEntry("wallet", []string{"new"})
	history.AddEntry("wallet", []string{"list"})

	// Delete entry
	err = history.DeleteEntry(0)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(history.entries))
	assert.Equal(t, "wallet", history.entries[0].Command)
	assert.Equal(t, []string{"new"}, history.entries[0].Args)

	// Test invalid index
	err = history.DeleteEntry(100)
	assert.Error(t, err)
}

func TestVisibility(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Test show
	history.Show()
	assert.True(t, history.visible)
	assert.True(t, history.IsVisible())

	// Test hide
	history.Hide()
	assert.False(t, history.visible)
	assert.False(t, history.IsVisible())

	// Test toggle
	history.ToggleVisibility()
	assert.True(t, history.visible)
	assert.True(t, history.IsVisible())

	history.ToggleVisibility()
	assert.False(t, history.visible)
	assert.False(t, history.IsVisible())
}

func TestToggleFavorites(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Test toggle
	history.ToggleFavorites()
	assert.True(t, history.showFavorites)

	history.ToggleFavorites()
	assert.False(t, history.showFavorites)
}

func TestHistorySetSize(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Test set size
	history.SetSize(100, 50)
	assert.Equal(t, 100, history.width)
	assert.Equal(t, 50, history.height)
}

func TestHistoryUpdate(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Add entries
	history.AddEntry("wallet", []string{"new"})
	history.AddEntry("wallet", []string{"list"})

	// Test update when not visible
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd := history.Update(msg)

	h := model.(*CommandHistory)
	assert.False(t, h.visible)
	assert.Nil(t, cmd)

	// Test update when visible
	history.Show()

	// Test escape key
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd = history.Update(msg)

	h = model.(*CommandHistory)
	assert.False(t, h.visible)
	assert.Nil(t, cmd)
}

func TestHistoryView(t *testing.T) {
	styles := ui.DefaultStyles(ui.DefaultTheme)

	// Create a temporary directory for the history file
	tempDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	historyFile := filepath.Join(tempDir, "history.json")

	history, err := NewCommandHistory(styles, historyFile, 100)
	assert.NoError(t, err)

	// Test view when not visible
	view := history.View()
	assert.Equal(t, "", view)

	// Test view when visible
	history.Show()

	// Add entries
	history.AddEntry("wallet", []string{"new"})
	history.AddEntry("wallet", []string{"list"})

	view = history.View()

	// Check that the view contains all the expected elements
	assert.Contains(t, view, "Command History")
	assert.Contains(t, view, "wallet")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "execute")
	assert.Contains(t, view, "toggle favorite")
	assert.Contains(t, view, "delete")
	assert.Contains(t, view, "filter")
	assert.Contains(t, view, "close")

	// Test view with favorites only
	history.ToggleFavorites()
	view = history.View()

	assert.Contains(t, view, "Favorite Commands")
}
