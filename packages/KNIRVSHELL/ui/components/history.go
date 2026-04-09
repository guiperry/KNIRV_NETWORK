package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
)

// CommandEntry represents a command history entry
type CommandEntry struct {
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Timestamp time.Time `json:"timestamp"`
	Favorite  bool      `json:"favorite"`
}

// Title returns the command as a title
func (c CommandEntry) Title() string {
	if c.Favorite {
		return fmt.Sprintf("★ %s", c.Command)
	}
	return c.Command
}

// Description returns the command arguments as a description
func (c CommandEntry) Description() string {
	if len(c.Args) > 0 {
		return strings.Join(c.Args, " ")
	}
	return fmt.Sprintf("Last used: %s", c.Timestamp.Format("2006-01-02 15:04:05"))
}

// FilterValue returns the command as a filter value
func (c CommandEntry) FilterValue() string {
	return c.Command
}

// CommandHistoryKeyMap defines keybindings for the command history
type CommandHistoryKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Execute  key.Binding
	Favorite key.Binding
	Delete   key.Binding
	Close    key.Binding
	Filter   key.Binding
}

// DefaultCommandHistoryKeyMap returns the default keybindings
func DefaultCommandHistoryKeyMap() CommandHistoryKeyMap {
	return CommandHistoryKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Execute: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute"),
		),
		Favorite: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle favorite"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc/q", "close"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
	}
}

// CommandHistory represents a command history manager
type CommandHistory struct {
	styles        ui.Styles
	list          list.Model
	keyMap        CommandHistoryKeyMap
	entries       []CommandEntry
	historyFile   string
	maxEntries    int
	visible       bool
	width         int
	height        int
	showFavorites bool
}

// NewCommandHistory creates a new command history manager
func NewCommandHistory(styles ui.Styles, historyFile string, maxEntries int) (*CommandHistory, error) {
	keyMap := DefaultCommandHistoryKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	historyList := list.New([]list.Item{}, listDelegate, 80, 20)
	historyList.Title = "Command History"
	historyList.SetShowStatusBar(true)
	historyList.SetFilteringEnabled(true)
	historyList.Styles.Title = styles.Title
	historyList.Styles.TitleBar = styles.Header

	history := &CommandHistory{
		styles:        styles,
		list:          historyList,
		keyMap:        keyMap,
		entries:       []CommandEntry{},
		historyFile:   historyFile,
		maxEntries:    maxEntries,
		visible:       false,
		width:         80,
		height:        24,
		showFavorites: false,
	}

	// Create history directory if it doesn't exist
	historyDir := filepath.Dir(historyFile)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// Load history from file
	if err := history.Load(); err != nil {
		// If the file doesn't exist, that's fine
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load history: %w", err)
		}
	}

	return history, nil
}

// Load loads the command history from file
func (h *CommandHistory) Load() error {
	// Check if the file exists
	if _, err := os.Stat(h.historyFile); os.IsNotExist(err) {
		return err
	}

	// Read the file
	data, err := os.ReadFile(h.historyFile)
	if err != nil {
		return err
	}

	// Unmarshal the data
	if err := json.Unmarshal(data, &h.entries); err != nil {
		return err
	}

	// Sort entries by timestamp (newest first)
	sort.Slice(h.entries, func(i, j int) bool {
		return h.entries[i].Timestamp.After(h.entries[j].Timestamp)
	})

	// Update the list
	h.updateList()

	return nil
}

// Save saves the command history to file
func (h *CommandHistory) Save() error {
	// Marshal the data
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return err
	}

	// Write the file
	return os.WriteFile(h.historyFile, data, 0644)
}

// AddEntry adds a command to the history
func (h *CommandHistory) AddEntry(command string, args []string) error {
	// Check if the command already exists
	for i, entry := range h.entries {
		if entry.Command == command && strings.Join(entry.Args, " ") == strings.Join(args, " ") {
			// Update the timestamp
			h.entries[i].Timestamp = time.Now()

			// Move to the top
			h.entries = append([]CommandEntry{h.entries[i]}, append(h.entries[:i], h.entries[i+1:]...)...)

			// Update the list
			h.updateList()

			// Save the history
			return h.Save()
		}
	}

	// Add the command
	entry := CommandEntry{
		Command:   command,
		Args:      args,
		Timestamp: time.Now(),
		Favorite:  false,
	}

	h.entries = append([]CommandEntry{entry}, h.entries...)

	// Trim the history
	if len(h.entries) > h.maxEntries {
		// Keep all favorites
		favorites := []CommandEntry{}
		nonFavorites := []CommandEntry{}

		for _, entry := range h.entries {
			if entry.Favorite {
				favorites = append(favorites, entry)
			} else {
				nonFavorites = append(nonFavorites, entry)
			}
		}

		// Keep only the most recent non-favorites
		if len(nonFavorites) > h.maxEntries {
			nonFavorites = nonFavorites[:h.maxEntries]
		}

		// Combine favorites and non-favorites
		h.entries = append(favorites, nonFavorites...)

		// Sort by timestamp
		sort.Slice(h.entries, func(i, j int) bool {
			return h.entries[i].Timestamp.After(h.entries[j].Timestamp)
		})
	}

	// Update the list
	h.updateList()

	// Save the history
	return h.Save()
}

// ToggleFavorite toggles the favorite status of a command
func (h *CommandHistory) ToggleFavorite(index int) error {
	if index < 0 || index >= len(h.entries) {
		return fmt.Errorf("invalid index: %d", index)
	}

	h.entries[index].Favorite = !h.entries[index].Favorite

	// Update the list
	h.updateList()

	// Save the history
	return h.Save()
}

// DeleteEntry deletes a command from the history
func (h *CommandHistory) DeleteEntry(index int) error {
	if index < 0 || index >= len(h.entries) {
		return fmt.Errorf("invalid index: %d", index)
	}

	h.entries = append(h.entries[:index], h.entries[index+1:]...)

	// Update the list
	h.updateList()

	// Save the history
	return h.Save()
}

// GetSelectedEntry returns the selected entry
func (h *CommandHistory) GetSelectedEntry() (CommandEntry, bool) {
	item, ok := h.list.SelectedItem().(CommandEntry)
	return item, ok
}

// Show shows the command history
func (h *CommandHistory) Show() {
	h.visible = true
	h.updateList()
}

// Hide hides the command history
func (h *CommandHistory) Hide() {
	h.visible = false
}

// ToggleVisibility toggles the visibility of the command history
func (h *CommandHistory) ToggleVisibility() {
	h.visible = !h.visible
	if h.visible {
		h.updateList()
	}
}

// IsVisible returns whether the command history is visible
func (h *CommandHistory) IsVisible() bool {
	return h.visible
}

// ToggleFavorites toggles showing only favorites
func (h *CommandHistory) ToggleFavorites() {
	h.showFavorites = !h.showFavorites
	h.updateList()
}

// SetSize sets the size of the command history
func (h *CommandHistory) SetSize(width, height int) {
	h.width = width
	h.height = height
	h.list.SetSize(width, height-4)
}

// updateList updates the list with the current entries
func (h *CommandHistory) updateList() {
	var items []list.Item

	for _, entry := range h.entries {
		if h.showFavorites && !entry.Favorite {
			continue
		}
		items = append(items, entry)
	}

	h.list.SetItems(items)
}

// Init initializes the command history
func (h *CommandHistory) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (h *CommandHistory) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, h.keyMap.Close):
			h.Hide()
			return h, nil

		case key.Matches(msg, h.keyMap.Favorite):
			if i, ok := h.list.SelectedItem().(CommandEntry); ok {
				for j, entry := range h.entries {
					if entry.Command == i.Command && strings.Join(entry.Args, " ") == strings.Join(i.Args, " ") {
						h.ToggleFavorite(j)
						break
					}
				}
			}

		case key.Matches(msg, h.keyMap.Delete):
			if i, ok := h.list.SelectedItem().(CommandEntry); ok {
				for j, entry := range h.entries {
					if entry.Command == i.Command && strings.Join(entry.Args, " ") == strings.Join(i.Args, " ") {
						h.DeleteEntry(j)
						break
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	h.list, cmd = h.list.Update(msg)
	return h, cmd
}

// View renders the command history
func (h *CommandHistory) View() string {
	if !h.visible {
		return ""
	}

	title := "Command History"
	if h.showFavorites {
		title = "Favorite Commands"
	}
	h.list.Title = title

	content := h.list.View()

	help := h.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: execute • %s: toggle favorite • %s: delete • %s: filter • %s: close",
		h.styles.KeyBinding.Render("↑/↓"),
		h.styles.KeyBinding.Render("enter"),
		h.styles.KeyBinding.Render("f"),
		h.styles.KeyBinding.Render("d"),
		h.styles.KeyBinding.Render("/"),
		h.styles.KeyBinding.Render("esc/q"),
	))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		help,
	)
}
