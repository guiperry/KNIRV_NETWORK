package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ServerItem represents a server item
type ServerItem struct {
	name        string
	url         string
	description string
	status      string
	region      string
	lastChecked time.Time
}

// Title returns the item title
func (i ServerItem) Title() string {
	return fmt.Sprintf("%s (%s)", i.name, i.status)
}

// Description returns the item description
func (i ServerItem) Description() string {
	return fmt.Sprintf("%s • %s • Region: %s", i.description, i.url, i.region)
}

// FilterValue returns the filter value
func (i ServerItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", i.name, i.description, i.url, i.region)
}

// ServerKeyMap defines keybindings for the server screen
type ServerKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Add     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Test    key.Binding
	Details key.Binding
	Back    key.Binding
	Refresh key.Binding
}

// DefaultServerKeyMap returns the default keybindings
func DefaultServerKeyMap() ServerKeyMap {
	return ServerKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add server"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit server"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete server"),
		),
		Test: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "test connection"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// ServerScreen represents the server management screen
type ServerScreen struct {
	styles        ui.Styles
	list          list.Model
	keyMap        ServerKeyMap
	serverManager *core.MCPServerManager
	width         int
	height        int
	loading       bool
	error         string

	// Modals and forms
	addServerForm      components.Form
	editServerForm     components.Form
	confirmDeleteModal components.ConfirmModal
	detailsModal       components.Modal
	testingModal       components.Modal

	// State
	showAddServerForm  bool
	showEditServerForm bool
	showConfirmDelete  bool
	showDetailsModal   bool
	showTestingModal   bool
	selectedServer     ServerItem
	testingInProgress  bool
	testingResult      string

	// Parent screen
	parent ui.Screen
}

// NewServerScreen creates a new server management screen
func NewServerScreen(styles ui.Styles, serverManager *core.MCPServerManager, parent ui.Screen) *ServerScreen {
	keyMap := DefaultServerKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	serverList := list.New([]list.Item{}, listDelegate, 80, 20)
	serverList.Title = "Server Management"
	serverList.SetShowStatusBar(false)
	serverList.SetFilteringEnabled(true)
	serverList.Styles.Title = styles.Title
	serverList.Styles.TitleBar = styles.Header

	// Create forms
	addServerForm := components.NewForm(styles, 60)
	addServerForm.AddField(components.NewFormField("Name", "Server name", "", true))
	addServerForm.AddField(components.NewFormField("URL", "Server URL", "http://", true))
	addServerForm.AddField(components.NewFormField("Description", "Server description", "", false))
	addServerForm.AddField(components.NewFormField("API Key", "API key (optional)", "", false))
	addServerForm.AddField(components.NewFormField("Region", "Server region", "", false))
	addServerForm.SetSubmitLabel("Add")
	addServerForm.SetCancelLabel("Cancel")

	editServerForm := components.NewForm(styles, 60)
	editServerForm.AddField(components.NewFormField("Name", "Server name", "", true))
	editServerForm.AddField(components.NewFormField("URL", "Server URL", "", true))
	editServerForm.AddField(components.NewFormField("Description", "Server description", "", false))
	editServerForm.AddField(components.NewFormField("API Key", "API key (optional)", "", false))
	editServerForm.AddField(components.NewFormField("Region", "Server region", "", false))
	editServerForm.SetSubmitLabel("Update")
	editServerForm.SetCancelLabel("Cancel")

	// Create modals
	confirmDeleteModal := components.NewConfirmModal(
		styles,
		"Confirm Delete",
		"Are you sure you want to delete this server? This action cannot be undone.",
		60,
		10,
	)

	detailsModal := components.NewModal(styles, "Server Details", "", 70, 20)
	detailsModal.SetButtons([]string{"Close"})

	testingModal := components.NewModal(styles, "Testing Connection", "", 60, 10)
	testingModal.SetButtons([]string{"Close"})

	return &ServerScreen{
		styles:             styles,
		list:               serverList,
		keyMap:             keyMap,
		serverManager:      serverManager,
		width:              80,
		height:             24,
		loading:            false,
		error:              "",
		addServerForm:      addServerForm,
		editServerForm:     editServerForm,
		confirmDeleteModal: confirmDeleteModal,
		detailsModal:       detailsModal,
		testingModal:       testingModal,
		showAddServerForm:  false,
		showEditServerForm: false,
		showConfirmDelete:  false,
		showDetailsModal:   false,
		showTestingModal:   false,
		selectedServer:     ServerItem{},
		testingInProgress:  false,
		testingResult:      "",
		parent:             parent,
	}
}

// Init initializes the screen
func (s *ServerScreen) Init() tea.Cmd {
	return s.loadServers()
}

// loadServers loads the servers
func (s *ServerScreen) loadServers() tea.Cmd {
	return func() tea.Msg {
		s.loading = true

		// In a real implementation, this would load servers from the server manager
		// For now, we'll create some sample servers
		items := []list.Item{
			ServerItem{
				name:        "Main Node",
				url:         "https://main.knirvchain.net",
				description: "Primary KNIRVCHAIN node",
				status:      "Online",
				region:      "us-west",
				lastChecked: time.Now().Add(-10 * time.Minute),
			},
			ServerItem{
				name:        "Backup Node",
				url:         "https://backup.knirvchain.net",
				description: "Backup KNIRVCHAIN node",
				status:      "Online",
				region:      "us-east",
				lastChecked: time.Now().Add(-15 * time.Minute),
			},
			ServerItem{
				name:        "Development Node",
				url:         "https://dev.knirvchain.net",
				description: "Development KNIRVCHAIN node",
				status:      "Online",
				region:      "eu-west",
				lastChecked: time.Now().Add(-5 * time.Minute),
			},
			ServerItem{
				name:        "Test Node",
				url:         "https://test.knirvchain.net",
				description: "Test KNIRVCHAIN node",
				status:      "Offline",
				region:      "ap-south",
				lastChecked: time.Now().Add(-30 * time.Minute),
			},
		}

		return ServersLoadedMsg{items: items}
	}
}

// ServersLoadedMsg is sent when servers are loaded
type ServersLoadedMsg struct {
	items []list.Item
}

// TestConnectionResultMsg is sent when a connection test completes
type TestConnectionResultMsg struct {
	success bool
	message string
}

// testConnection tests the connection to a server
func (s *ServerScreen) testConnection(server ServerItem) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would test the connection to the server
		// For now, we'll simulate a connection test
		time.Sleep(2 * time.Second)

		if server.status == "Online" {
			return TestConnectionResultMsg{
				success: true,
				message: fmt.Sprintf("Successfully connected to %s (%s)", server.name, server.url),
			}
		}

		return TestConnectionResultMsg{
			success: false,
			message: fmt.Sprintf("Failed to connect to %s (%s): Connection refused", server.name, server.url),
		}
	}
}

// Update handles user input
func (s *ServerScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ServersLoadedMsg:
		s.loading = false
		s.list.SetItems(msg.items)
		return s, nil

	case TestConnectionResultMsg:
		s.testingInProgress = false
		s.testingResult = msg.message

		if msg.success {
			s.testingModal.Content = s.styles.Success.Render(msg.message)
		} else {
			s.testingModal.Content = s.styles.Error.Render(msg.message)
		}

		return s, nil

	case tea.KeyMsg:
		// Handle form and modal inputs first
		if s.showAddServerForm {
			form, cmd := s.addServerForm.Update(msg)
			s.addServerForm = form

			switch msg.String() {
			case "enter":
				if s.addServerForm.Validate() {
					// Add server
					values := s.addServerForm.GetValues()
					name := values["Name"]
					url := values["URL"]
					description := values["Description"]
					apiKey := values["API Key"]
					region := values["Region"]

					// In a real implementation, this would add the server
					_ = name        // Unused for now
					_ = url         // Unused for now
					_ = description // Unused for now
					_ = apiKey      // Unused for now
					_ = region      // Unused for now

					s.showAddServerForm = false
					return s, s.loadServers()
				}
			case "esc":
				s.showAddServerForm = false
				return s, nil
			}

			return s, cmd
		}

		if s.showEditServerForm {
			form, cmd := s.editServerForm.Update(msg)
			s.editServerForm = form

			switch msg.String() {
			case "enter":
				if s.editServerForm.Validate() {
					// Update server
					values := s.editServerForm.GetValues()
					name := values["Name"]
					url := values["URL"]
					description := values["Description"]
					apiKey := values["API Key"]
					region := values["Region"]

					// In a real implementation, this would update the server
					_ = name        // Unused for now
					_ = url         // Unused for now
					_ = description // Unused for now
					_ = apiKey      // Unused for now
					_ = region      // Unused for now

					s.showEditServerForm = false
					return s, s.loadServers()
				}
			case "esc":
				s.showEditServerForm = false
				return s, nil
			}

			return s, cmd
		}

		if s.showConfirmDelete {
			modal, cmd := s.confirmDeleteModal.Update(msg)
			s.confirmDeleteModal = modal

			if !s.confirmDeleteModal.IsVisible() {
				s.showConfirmDelete = false

				// If confirmed, delete the server
				if s.confirmDeleteModal.SelectedButton() == "Confirm" {
					// In a real implementation, this would delete the server
					return s, s.loadServers()
				}
			}

			return s, cmd
		}

		if s.showDetailsModal {
			modal, cmd := s.detailsModal.Update(msg)
			s.detailsModal = modal

			if !s.detailsModal.IsVisible() {
				s.showDetailsModal = false
			}

			return s, cmd
		}

		if s.showTestingModal {
			modal, cmd := s.testingModal.Update(msg)
			s.testingModal = modal

			if !s.testingModal.IsVisible() {
				s.showTestingModal = false
			}

			return s, cmd
		}

		// Handle main screen inputs
		switch {
		case key.Matches(msg, s.keyMap.Back):
			return s.parent, nil

		case key.Matches(msg, s.keyMap.Add):
			s.showAddServerForm = true
			s.addServerForm.Focus()
			return s, nil

		case key.Matches(msg, s.keyMap.Edit):
			i, ok := s.list.SelectedItem().(ServerItem)
			if ok {
				s.selectedServer = i

				// Populate form with server data
				s.editServerForm.Fields[0].Input.SetValue(i.name)
				s.editServerForm.Fields[1].Input.SetValue(i.url)
				s.editServerForm.Fields[2].Input.SetValue(i.description)
				s.editServerForm.Fields[3].Input.SetValue("") // API key not shown
				s.editServerForm.Fields[4].Input.SetValue(i.region)

				s.showEditServerForm = true
				s.editServerForm.Focus()
			}
			return s, nil

		case key.Matches(msg, s.keyMap.Delete):
			i, ok := s.list.SelectedItem().(ServerItem)
			if ok {
				s.selectedServer = i
				s.showConfirmDelete = true
				s.confirmDeleteModal.Show()
			}
			return s, nil

		case key.Matches(msg, s.keyMap.Test):
			i, ok := s.list.SelectedItem().(ServerItem)
			if ok {
				s.selectedServer = i
				s.testingInProgress = true
				s.testingResult = ""
				s.testingModal.Content = "Testing connection..."
				s.testingModal.Show()
				s.showTestingModal = true
				return s, s.testConnection(i)
			}
			return s, nil

		case key.Matches(msg, s.keyMap.Details):
			i, ok := s.list.SelectedItem().(ServerItem)
			if ok {
				s.selectedServer = i

				// Create details content
				lastChecked := i.lastChecked.Format("2006-01-02 15:04:05")
				details := fmt.Sprintf(
					"Name: %s\nURL: %s\nDescription: %s\nStatus: %s\nRegion: %s\nLast Checked: %s\n\nCapabilities: 42\nProcedures: 15\nAverage Response Time: 120ms\nUptime: 99.98%%",
					i.name, i.url, i.description, i.status, i.region, lastChecked,
				)

				s.detailsModal.Title = fmt.Sprintf("Server: %s", i.name)
				s.detailsModal.Content = details
				s.detailsModal.Show()
				s.showDetailsModal = true
			}
			return s, nil

		case key.Matches(msg, s.keyMap.Refresh):
			return s, s.loadServers()
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.list.SetWidth(msg.Width)
		s.list.SetHeight(msg.Height - 10)
	}

	// Update the list
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	cmds = append(cmds, cmd)

	return s, tea.Batch(cmds...)
}

// View renders the screen
func (s ServerScreen) View() string {
	if s.loading {
		spinner := components.NewSpinner(s.styles)
		spinner.SetLabel("Loading servers...")
		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			spinner.View(),
		)
	}

	if s.showAddServerForm {
		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			s.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					s.styles.DialogTitle.Render("Add Server"),
					"",
					s.addServerForm.View(),
					"",
					s.styles.Error.Render(s.error),
				),
			),
		)
	}

	if s.showEditServerForm {
		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			s.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					s.styles.DialogTitle.Render(fmt.Sprintf("Edit Server: %s", s.selectedServer.name)),
					"",
					s.editServerForm.View(),
					"",
					s.styles.Error.Render(s.error),
				),
			),
		)
	}

	if s.showConfirmDelete {
		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			s.confirmDeleteModal.View(),
		)
	}

	if s.showDetailsModal {
		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			s.detailsModal.View(),
		)
	}

	if s.showTestingModal {
		var content string
		if s.testingInProgress {
			spinner := components.NewSpinner(s.styles)
			spinner.SetLabel("Testing connection...")
			content = spinner.View()
		} else {
			content = s.testingResult
		}

		s.testingModal.Content = content

		return lipgloss.Place(
			s.width,
			s.height,
			lipgloss.Center,
			lipgloss.Center,
			s.testingModal.View(),
		)
	}

	// Main view
	var sb strings.Builder

	// Title
	sb.WriteString(s.list.View())

	// Help
	help := s.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: add • %s: edit • %s: delete • %s: test • %s: details • %s: refresh • %s: back",
		s.styles.KeyBinding.Render("↑/↓"),
		s.styles.KeyBinding.Render("a"),
		s.styles.KeyBinding.Render("e"),
		s.styles.KeyBinding.Render("d"),
		s.styles.KeyBinding.Render("t"),
		s.styles.KeyBinding.Render("enter"),
		s.styles.KeyBinding.Render("r"),
		s.styles.KeyBinding.Render("esc"),
	))

	sb.WriteString("\n\n")
	sb.WriteString(help)

	// Error
	if s.error != "" {
		sb.WriteString("\n\n")
		sb.WriteString(s.styles.Error.Render(s.error))
	}

	return sb.String()
}
