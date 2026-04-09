package screens

import (
	"fmt"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CapabilityItem represents a capability item
type CapabilityItem struct {
	id          string
	name        string
	version     string
	description string
	author      string
	status      string
}

// Title returns the item title
func (i CapabilityItem) Title() string {
	return fmt.Sprintf("%s (v%s)", i.name, i.version)
}

// Description returns the item description
func (i CapabilityItem) Description() string {
	return fmt.Sprintf("%s • By: %s • Status: %s", i.description, i.author, i.status)
}

// FilterValue returns the filter value
func (i CapabilityItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.name, i.version, i.description)
}

// CapabilityKeyMap defines keybindings for the capability screen
type CapabilityKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Register key.Binding
	Generate key.Binding
	Invoke   key.Binding
	Details  key.Binding
	Back     key.Binding
	Refresh  key.Binding
}

// DefaultCapabilityKeyMap returns the default keybindings
func DefaultCapabilityKeyMap() CapabilityKeyMap {
	return CapabilityKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Register: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "register"),
		),
		Generate: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "generate"),
		),
		Invoke: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "invoke"),
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
			key.WithKeys("f5"),
			key.WithHelp("f5", "refresh"),
		),
	}
}

// CapabilityScreen represents the capability management screen
type CapabilityScreen struct {
	styles    ui.Styles
	list      list.Model
	keyMap    CapabilityKeyMap
	apiClient *core.APIClient
	width     int
	height    int
	loading   bool
	error     string

	// Modals and forms
	registerForm components.Form
	generateForm components.Form
	invokeForm   components.Form
	detailsModal components.Modal

	// State
	showRegisterForm   bool
	showGenerateForm   bool
	showInvokeForm     bool
	showDetailsModal   bool
	selectedCapability CapabilityItem

	// Parent screen
	parent ui.Screen
}

// NewCapabilityScreen creates a new capability management screen
func NewCapabilityScreen(styles ui.Styles, apiClient *core.APIClient, parent ui.Screen) *CapabilityScreen {
	keyMap := DefaultCapabilityKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	capList := list.New([]list.Item{}, listDelegate, 80, 20)
	capList.Title = "Capability Management"
	capList.SetShowStatusBar(false)
	capList.SetFilteringEnabled(true)
	capList.Styles.Title = styles.Title
	capList.Styles.TitleBar = styles.Header

	// Create forms
	registerForm := components.NewForm(styles, 60)
	registerForm.AddField(components.NewFormField("Plugin Path", "Path to plugin file (.so)", "", true))
	registerForm.AddField(components.NewFormField("Manifest Path", "Path to manifest file (.json)", "", true))
	registerForm.AddField(components.NewFormField("Wallet", "Wallet address", "", true))
	registerForm.AddField(components.NewPasswordField("Password", "Wallet password", "", true))
	registerForm.SetSubmitLabel("Register")
	registerForm.SetCancelLabel("Cancel")

	generateForm := components.NewForm(styles, 60)
	generateForm.AddField(components.NewFormField("Name", "Capability name", "", true))
	generateForm.AddField(components.NewFormField("Description", "Capability description", "", true))
	generateForm.AddField(components.NewFormField("Type", "Capability type", "tool", true))
	generateForm.AddField(components.NewFormField("Output Path", "Output directory", "", true))
	generateForm.SetSubmitLabel("Generate")
	generateForm.SetCancelLabel("Cancel")

	invokeForm := components.NewForm(styles, 60)
	invokeForm.AddField(components.NewFormField("Parameters", "JSON parameters", "{}", true))
	invokeForm.AddField(components.NewFormField("Wallet", "Wallet address", "", true))
	invokeForm.AddField(components.NewPasswordField("Password", "Wallet password", "", true))
	invokeForm.SetSubmitLabel("Invoke")
	invokeForm.SetCancelLabel("Cancel")

	// Create modals
	detailsModal := components.NewModal(styles, "Capability Details", "", 70, 20)
	detailsModal.SetButtons([]string{"Close"})

	return &CapabilityScreen{
		styles:             styles,
		list:               capList,
		keyMap:             keyMap,
		apiClient:          apiClient,
		width:              80,
		height:             24,
		loading:            false,
		error:              "",
		registerForm:       registerForm,
		generateForm:       generateForm,
		invokeForm:         invokeForm,
		detailsModal:       detailsModal,
		showRegisterForm:   false,
		showGenerateForm:   false,
		showInvokeForm:     false,
		showDetailsModal:   false,
		selectedCapability: CapabilityItem{},
		parent:             parent,
	}
}

// Init initializes the screen
func (c *CapabilityScreen) Init() tea.Cmd {
	return c.loadCapabilities()
}

// loadCapabilities loads the capabilities
func (c *CapabilityScreen) loadCapabilities() tea.Cmd {
	return func() tea.Msg {
		c.loading = true

		// In a real implementation, this would load capabilities from the API client
		// For now, we'll create some sample capabilities
		items := []list.Item{
			CapabilityItem{
				id:          "cap-1234567890",
				name:        "Image Generator",
				version:     "1.0.0",
				description: "Generates images from text descriptions",
				author:      "AI Labs",
				status:      "Active",
			},
			CapabilityItem{
				id:          "cap-0987654321",
				name:        "Text Translator",
				version:     "2.1.0",
				description: "Translates text between languages",
				author:      "Language Tech",
				status:      "Active",
			},
			CapabilityItem{
				id:          "cap-5678901234",
				name:        "Code Generator",
				version:     "0.9.5",
				description: "Generates code from natural language",
				author:      "DevTools Inc",
				status:      "Pending",
			},
			CapabilityItem{
				id:          "cap-4321098765",
				name:        "Data Analyzer",
				version:     "1.2.3",
				description: "Analyzes data and generates insights",
				author:      "Data Science Co",
				status:      "Active",
			},
		}

		return CapabilitiesLoadedMsg{items: items}
	}
}

// CapabilitiesLoadedMsg is sent when capabilities are loaded
type CapabilitiesLoadedMsg struct {
	items []list.Item
}

// Update handles user input
func (c *CapabilityScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CapabilitiesLoadedMsg:
		c.loading = false
		c.list.SetItems(msg.items)
		return c, nil

	case tea.KeyMsg:
		// Handle form and modal inputs first
		if c.showRegisterForm {
			form, cmd := c.registerForm.Update(msg)
			c.registerForm = form

			switch msg.String() {
			case "enter":
				if c.registerForm.Validate() {
					// Register capability
					values := c.registerForm.GetValues()
					pluginPath := values["Plugin Path"]
					manifestPath := values["Manifest Path"]
					wallet := values["Wallet"]
					password := values["Password"]

					// In a real implementation, this would register the capability
					_ = pluginPath   // Unused for now
					_ = manifestPath // Unused for now
					_ = wallet       // Unused for now
					_ = password     // Unused for now

					c.showRegisterForm = false
					return c, c.loadCapabilities()
				}
			case "esc":
				c.showRegisterForm = false
				return c, nil
			}

			return c, cmd
		}

		if c.showGenerateForm {
			form, cmd := c.generateForm.Update(msg)
			c.generateForm = form

			switch msg.String() {
			case "enter":
				if c.generateForm.Validate() {
					// Generate capability
					values := c.generateForm.GetValues()
					name := values["Name"]
					description := values["Description"]
					capType := values["Type"]
					outputPath := values["Output Path"]

					// In a real implementation, this would generate the capability
					_ = name        // Unused for now
					_ = description // Unused for now
					_ = capType     // Unused for now
					_ = outputPath  // Unused for now

					c.showGenerateForm = false
					return c, nil
				}
			case "esc":
				c.showGenerateForm = false
				return c, nil
			}

			return c, cmd
		}

		if c.showInvokeForm {
			form, cmd := c.invokeForm.Update(msg)
			c.invokeForm = form

			switch msg.String() {
			case "enter":
				if c.invokeForm.Validate() {
					// Invoke capability
					values := c.invokeForm.GetValues()
					parameters := values["Parameters"]
					wallet := values["Wallet"]
					password := values["Password"]

					// In a real implementation, this would invoke the capability
					_ = parameters // Unused for now
					_ = wallet     // Unused for now
					_ = password   // Unused for now

					c.showInvokeForm = false
					return c, nil
				}
			case "esc":
				c.showInvokeForm = false
				return c, nil
			}

			return c, cmd
		}

		if c.showDetailsModal {
			modal, cmd := c.detailsModal.Update(msg)
			c.detailsModal = modal

			if !c.detailsModal.IsVisible() {
				c.showDetailsModal = false
			}

			return c, cmd
		}

		// Handle main screen inputs
		switch {
		case key.Matches(msg, c.keyMap.Back):
			return c.parent, nil

		case key.Matches(msg, c.keyMap.Register):
			c.showRegisterForm = true
			c.registerForm.Focus()
			return c, nil

		case key.Matches(msg, c.keyMap.Generate):
			c.showGenerateForm = true
			c.generateForm.Focus()
			return c, nil

		case key.Matches(msg, c.keyMap.Invoke):
			i, ok := c.list.SelectedItem().(CapabilityItem)
			if ok {
				c.selectedCapability = i
				c.showInvokeForm = true
				c.invokeForm.Focus()
			}
			return c, nil

		case key.Matches(msg, c.keyMap.Details):
			i, ok := c.list.SelectedItem().(CapabilityItem)
			if ok {
				c.selectedCapability = i

				// Create details content
				details := fmt.Sprintf(
					"ID: %s\nName: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nStatus: %s\n\nParameters:\n- param1: string\n- param2: number\n- param3: boolean\n\nReturns:\n- result: object",
					i.id, i.name, i.version, i.description, i.author, i.status,
				)

				c.detailsModal.Title = fmt.Sprintf("Capability: %s (v%s)", i.name, i.version)
				c.detailsModal.Content = details
				c.detailsModal.Show()
				c.showDetailsModal = true
			}
			return c, nil

		case key.Matches(msg, c.keyMap.Refresh):
			return c, c.loadCapabilities()
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.list.SetWidth(msg.Width)
		c.list.SetHeight(msg.Height - 10)
	}

	// Update the list
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

// View renders the screen
func (c CapabilityScreen) View() string {
	if c.loading {
		spinner := components.NewSpinner(c.styles)
		spinner.SetLabel("Loading capabilities...")
		return lipgloss.Place(
			c.width,
			c.height,
			lipgloss.Center,
			lipgloss.Center,
			spinner.View(),
		)
	}

	if c.showRegisterForm {
		return lipgloss.Place(
			c.width,
			c.height,
			lipgloss.Center,
			lipgloss.Center,
			c.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					c.styles.DialogTitle.Render("Register Capability"),
					"",
					c.registerForm.View(),
					"",
					c.styles.Error.Render(c.error),
				),
			),
		)
	}

	if c.showGenerateForm {
		return lipgloss.Place(
			c.width,
			c.height,
			lipgloss.Center,
			lipgloss.Center,
			c.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					c.styles.DialogTitle.Render("Generate Capability"),
					"",
					c.generateForm.View(),
					"",
					c.styles.Error.Render(c.error),
				),
			),
		)
	}

	if c.showInvokeForm {
		return lipgloss.Place(
			c.width,
			c.height,
			lipgloss.Center,
			lipgloss.Center,
			c.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					c.styles.DialogTitle.Render(fmt.Sprintf("Invoke Capability: %s", c.selectedCapability.name)),
					"",
					c.invokeForm.View(),
					"",
					c.styles.Error.Render(c.error),
				),
			),
		)
	}

	if c.showDetailsModal {
		return lipgloss.Place(
			c.width,
			c.height,
			lipgloss.Center,
			lipgloss.Center,
			c.detailsModal.View(),
		)
	}

	// Main view
	var sb strings.Builder

	// Title
	sb.WriteString(c.list.View())

	// Help
	help := c.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: register • %s: generate • %s: invoke • %s: details • %s: refresh • %s: back",
		c.styles.KeyBinding.Render("↑/↓"),
		c.styles.KeyBinding.Render("r"),
		c.styles.KeyBinding.Render("g"),
		c.styles.KeyBinding.Render("i"),
		c.styles.KeyBinding.Render("enter"),
		c.styles.KeyBinding.Render("f5"),
		c.styles.KeyBinding.Render("esc"),
	))

	sb.WriteString("\n\n")
	sb.WriteString(help)

	// Error
	if c.error != "" {
		sb.WriteString("\n\n")
		sb.WriteString(c.styles.Error.Render(c.error))
	}

	return sb.String()
}
