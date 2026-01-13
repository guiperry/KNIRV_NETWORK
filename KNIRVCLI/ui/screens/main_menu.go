package screens

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/ui"
)

// MainMenuItem represents a main menu item
type MainMenuItem struct {
	title       string
	description string
	screen      ui.Screen
}

// Title returns the item title
func (i MainMenuItem) Title() string { return i.title }

// Description returns the item description
func (i MainMenuItem) Description() string { return i.description }

// FilterValue returns the filter value
func (i MainMenuItem) FilterValue() string { return i.title }

// MainMenuKeyMap defines keybindings for the main menu
type MainMenuKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Quit   key.Binding
}

// DefaultMainMenuKeyMap returns the default keybindings
func DefaultMainMenuKeyMap() MainMenuKeyMap {
	return MainMenuKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
	}
}

// MainMenuScreen represents the main menu screen
type MainMenuScreen struct {
	styles   ui.Styles
	list     list.Model
	keyMap   MainMenuKeyMap
	items    []MainMenuItem
	selected ui.Screen
}

// NewMainMenuScreen creates a new main menu screen
func NewMainMenuScreen(styles ui.Styles, width, height int) *MainMenuScreen {
	keyMap := DefaultMainMenuKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	mainList := list.New([]list.Item{}, listDelegate, width, height-10)
	mainList.Title = "KNIRVCHAIN CLI"
	mainList.SetShowStatusBar(false)
	mainList.SetFilteringEnabled(false)
	mainList.Styles.Title = styles.Title
	mainList.Styles.TitleBar = styles.Header

	return &MainMenuScreen{
		styles:   styles,
		list:     mainList,
		keyMap:   keyMap,
		items:    []MainMenuItem{},
		selected: nil,
	}
}

// AddMenuItem adds a menu item
func (m *MainMenuScreen) AddMenuItem(title, description string, screen ui.Screen) {
	item := MainMenuItem{
		title:       title,
		description: description,
		screen:      screen,
	}
	m.items = append(m.items, item)
	m.list.SetItems(append(m.list.Items(), item))
}

// Init initializes the screen
func (m *MainMenuScreen) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m *MainMenuScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Select):
			i, ok := m.list.SelectedItem().(MainMenuItem)
			if ok && i.screen != nil {
				m.selected = i.screen
				return i.screen, i.screen.Init()
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 10)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the screen
func (m *MainMenuScreen) View() string {
	if m.selected != nil {
		return m.selected.View()
	}

	header := m.styles.Title.Render("KNIRVCHAIN CLI")
	subtitle := m.styles.Subtitle.Render("Multi-Capability Protocol Command Line Interface")

	version := m.styles.Subtle.Render("Version 1.0.0")

	help := m.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: select • %s: quit",
		m.styles.KeyBinding.Render("↑/↓"),
		m.styles.KeyBinding.Render("enter"),
		m.styles.KeyBinding.Render("q"),
	))

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		subtitle,
		"",
		m.list.View(),
		"",
		help,
		version,
	)

	return content
}
