package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui/components"
)

// ProcedureItem represents a procedure item
type ProcedureItem struct {
	name        string
	version     string
	description string
	author      string
	steps       int
	lastRun     time.Time
}

// Title returns the item title
func (i ProcedureItem) Title() string {
	return fmt.Sprintf("%s (v%s)", i.name, i.version)
}

// Description returns the item description
func (i ProcedureItem) Description() string {
	lastRunStr := "Never"
	if !i.lastRun.IsZero() {
		lastRunStr = i.lastRun.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s • By: %s • Steps: %d • Last Run: %s", i.description, i.author, i.steps, lastRunStr)
}

// FilterValue returns the filter value
func (i ProcedureItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", i.name, i.version, i.description, i.author)
}

// ProcedureKeyMap defines keybindings for the procedure screen
type ProcedureKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Add     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Run     key.Binding
	Details key.Binding
	Back    key.Binding
	Refresh key.Binding
}

// DefaultProcedureKeyMap returns the default keybindings
func DefaultProcedureKeyMap() ProcedureKeyMap {
	return ProcedureKeyMap{
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
			key.WithHelp("a", "add procedure"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit procedure"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete procedure"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run procedure"),
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

// ProcedureScreen represents the procedure management screen
type ProcedureScreen struct {
	styles           ui.Styles
	list             list.Model
	keyMap           ProcedureKeyMap
	procedureManager *core.OpProcedureManager
	width            int
	height           int
	loading          bool
	error            string

	// Modals and forms
	addProcedureForm   components.Form
	editProcedureForm  components.Form
	runProcedureForm   components.Form
	confirmDeleteModal components.ConfirmModal
	detailsModal       components.Modal
	runningModal       components.Modal

	// State
	showAddProcedureForm  bool
	showEditProcedureForm bool
	showRunProcedureForm  bool
	showConfirmDelete     bool
	showDetailsModal      bool
	showRunningModal      bool
	selectedProcedure     ProcedureItem
	runningInProgress     bool
	runningResult         string
	runningProgress       float64

	// Parent screen
	parent ui.Screen
}

// NewProcedureScreen creates a new procedure management screen
func NewProcedureScreen(styles ui.Styles, procedureManager *core.OpProcedureManager, parent ui.Screen) *ProcedureScreen {
	keyMap := DefaultProcedureKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	procList := list.New([]list.Item{}, listDelegate, 80, 20)
	procList.Title = "Procedure Management"
	procList.SetShowStatusBar(false)
	procList.SetFilteringEnabled(true)
	procList.Styles.Title = styles.Title
	procList.Styles.TitleBar = styles.Header

	// Create forms
	addProcedureForm := components.NewForm(styles, 60)
	addProcedureForm.AddField(components.NewFormField("Name", "Procedure name", "", true))
	addProcedureForm.AddField(components.NewFormField("Version", "Procedure version", "1.0.0", true))
	addProcedureForm.AddField(components.NewFormField("Description", "Procedure description", "", true))
	addProcedureForm.AddField(components.NewFormField("Author", "Procedure author", "", true))
	addProcedureForm.AddField(components.NewFormField("License", "Procedure license", "MIT", false))
	addProcedureForm.SetSubmitLabel("Add")
	addProcedureForm.SetCancelLabel("Cancel")

	editProcedureForm := components.NewForm(styles, 60)
	editProcedureForm.AddField(components.NewFormField("Name", "Procedure name", "", true))
	editProcedureForm.AddField(components.NewFormField("Version", "Procedure version", "", true))
	editProcedureForm.AddField(components.NewFormField("Description", "Procedure description", "", true))
	editProcedureForm.AddField(components.NewFormField("Author", "Procedure author", "", true))
	editProcedureForm.AddField(components.NewFormField("License", "Procedure license", "", false))
	editProcedureForm.SetSubmitLabel("Update")
	editProcedureForm.SetCancelLabel("Cancel")

	runProcedureForm := components.NewForm(styles, 60)
	runProcedureForm.AddField(components.NewFormField("Parameters", "JSON parameters", "{}", false))
	runProcedureForm.SetSubmitLabel("Run")
	runProcedureForm.SetCancelLabel("Cancel")

	// Create modals
	confirmDeleteModal := components.NewConfirmModal(
		styles,
		"Confirm Delete",
		"Are you sure you want to delete this procedure? This action cannot be undone.",
		60,
		10,
	)

	detailsModal := components.NewModal(styles, "Procedure Details", "", 70, 20)
	detailsModal.SetButtons([]string{"Close"})

	runningModal := components.NewModal(styles, "Running Procedure", "", 60, 15)
	runningModal.SetButtons([]string{"Close"})

	return &ProcedureScreen{
		styles:                styles,
		list:                  procList,
		keyMap:                keyMap,
		procedureManager:      procedureManager,
		width:                 80,
		height:                24,
		loading:               false,
		error:                 "",
		addProcedureForm:      addProcedureForm,
		editProcedureForm:     editProcedureForm,
		runProcedureForm:      runProcedureForm,
		confirmDeleteModal:    confirmDeleteModal,
		detailsModal:          detailsModal,
		runningModal:          runningModal,
		showAddProcedureForm:  false,
		showEditProcedureForm: false,
		showRunProcedureForm:  false,
		showConfirmDelete:     false,
		showDetailsModal:      false,
		showRunningModal:      false,
		selectedProcedure:     ProcedureItem{},
		runningInProgress:     false,
		runningResult:         "",
		runningProgress:       0,
		parent:                parent,
	}
}

// Init initializes the screen
func (p *ProcedureScreen) Init() tea.Cmd {
	return p.loadProcedures()
}

// loadProcedures loads the procedures
func (p *ProcedureScreen) loadProcedures() tea.Cmd {
	return func() tea.Msg {
		p.loading = true

		// In a real implementation, this would load procedures from the procedure manager
		// For now, we'll create some sample procedures
		items := []list.Item{
			ProcedureItem{
				name:        "Capability Registration",
				version:     "1.0.0",
				description: "Registers a new capability",
				author:      "KNIRVCHAIN Team",
				steps:       5,
				lastRun:     time.Now().Add(-24 * time.Hour),
			},
			ProcedureItem{
				name:        "Server Setup",
				version:     "2.1.0",
				description: "Sets up a new KNIRVCHAIN server",
				author:      "KNIRVCHAIN Team",
				steps:       8,
				lastRun:     time.Now().Add(-48 * time.Hour),
			},
			ProcedureItem{
				name:        "Wallet Backup",
				version:     "1.2.0",
				description: "Backs up wallet to secure storage",
				author:      "Security Team",
				steps:       3,
				lastRun:     time.Now().Add(-12 * time.Hour),
			},
			ProcedureItem{
				name:        "Network Diagnostics",
				version:     "0.9.5",
				description: "Runs diagnostics on the KNIRVCHAIN network",
				author:      "DevOps Team",
				steps:       10,
				lastRun:     time.Time{}, // Never run
			},
		}

		return ProceduresLoadedMsg{items: items}
	}
}

// ProceduresLoadedMsg is sent when procedures are loaded
type ProceduresLoadedMsg struct {
	items []list.Item
}

// RunProcedureProgressMsg is sent to update the procedure execution progress
type RunProcedureProgressMsg struct {
	step     int
	total    int
	message  string
	progress float64
}

// RunProcedureResultMsg is sent when a procedure execution completes
type RunProcedureResultMsg struct {
	success bool
	message string
}

// runProcedure runs a procedure
func (p *ProcedureScreen) runProcedure(procedure ProcedureItem) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, this would run the procedure
		// For now, we'll simulate a procedure execution

		// Send progress updates
		for i := 1; i <= procedure.steps; i++ {
			time.Sleep(500 * time.Millisecond)
			progress := float64(i) / float64(procedure.steps)
			p.runningProgress = progress

			// In a real implementation, we would use a channel to send progress updates
			// For now, we'll just update the state directly
		}

		time.Sleep(500 * time.Millisecond)

		return RunProcedureResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully executed procedure '%s' (v%s)", procedure.name, procedure.version),
		}
	}
}

// Update handles user input
func (p *ProcedureScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ProceduresLoadedMsg:
		p.loading = false
		p.list.SetItems(msg.items)
		return p, nil

	case RunProcedureProgressMsg:
		p.runningProgress = msg.progress
		return p, nil

	case RunProcedureResultMsg:
		p.runningInProgress = false
		p.runningResult = msg.message

		if msg.success {
			p.runningModal.Content = p.styles.Success.Render(msg.message)
		} else {
			p.runningModal.Content = p.styles.Error.Render(msg.message)
		}

		return p, nil

	case tea.KeyMsg:
		// Handle form and modal inputs first
		if p.showAddProcedureForm {
			form, cmd := p.addProcedureForm.Update(msg)
			p.addProcedureForm = form

			switch msg.String() {
			case "enter":
				if p.addProcedureForm.Validate() {
					// Add procedure
					values := p.addProcedureForm.GetValues()
					name := values["Name"]
					version := values["Version"]
					description := values["Description"]
					author := values["Author"]
					license := values["License"]

					// In a real implementation, this would add the procedure
					_ = name        // Unused for now
					_ = version     // Unused for now
					_ = description // Unused for now
					_ = author      // Unused for now
					_ = license     // Unused for now

					p.showAddProcedureForm = false
					return p, p.loadProcedures()
				}
			case "esc":
				p.showAddProcedureForm = false
				return p, nil
			}

			return p, cmd
		}

		if p.showEditProcedureForm {
			form, cmd := p.editProcedureForm.Update(msg)
			p.editProcedureForm = form

			switch msg.String() {
			case "enter":
				if p.editProcedureForm.Validate() {
					// Update procedure
					values := p.editProcedureForm.GetValues()
					name := values["Name"]
					version := values["Version"]
					description := values["Description"]
					author := values["Author"]
					license := values["License"]

					// In a real implementation, this would update the procedure
					_ = name        // Unused for now
					_ = version     // Unused for now
					_ = description // Unused for now
					_ = author      // Unused for now
					_ = license     // Unused for now

					p.showEditProcedureForm = false
					return p, p.loadProcedures()
				}
			case "esc":
				p.showEditProcedureForm = false
				return p, nil
			}

			return p, cmd
		}

		if p.showRunProcedureForm {
			form, cmd := p.runProcedureForm.Update(msg)
			p.runProcedureForm = form

			switch msg.String() {
			case "enter":
				if p.runProcedureForm.Validate() {
					// Run procedure
					values := p.runProcedureForm.GetValues()
					parameters := values["Parameters"]

					p.showRunProcedureForm = false
					p.runningInProgress = true
					p.runningProgress = 0
					p.runningResult = ""
					p.showRunningModal = true
					p.runningModal.Show()

					return p, func() tea.Cmd {
						var (
							procedure ProcedureItem = p.selectedProcedure
							_         string        = parameters
						)
						return p.runProcedure(procedure)
					}()
				}
			case "esc":
				p.showRunProcedureForm = false
				return p, nil
			}

			return p, cmd
		}

		if p.showConfirmDelete {
			modal, cmd := p.confirmDeleteModal.Update(msg)
			p.confirmDeleteModal = modal

			if !p.confirmDeleteModal.IsVisible() {
				p.showConfirmDelete = false

				// If confirmed, delete the procedure
				if p.confirmDeleteModal.SelectedButton() == "Confirm" {
					// In a real implementation, this would delete the procedure
					return p, p.loadProcedures()
				}
			}

			return p, cmd
		}

		if p.showDetailsModal {
			modal, cmd := p.detailsModal.Update(msg)
			p.detailsModal = modal

			if !p.detailsModal.IsVisible() {
				p.showDetailsModal = false
			}

			return p, cmd
		}

		if p.showRunningModal {
			modal, cmd := p.runningModal.Update(msg)
			p.runningModal = modal

			if !p.runningModal.IsVisible() {
				p.showRunningModal = false
			}

			return p, cmd
		}

		// Handle main screen inputs
		switch {
		case key.Matches(msg, p.keyMap.Back):
			return p.parent, nil

		case key.Matches(msg, p.keyMap.Add):
			p.showAddProcedureForm = true
			p.addProcedureForm.Focus()
			return p, nil

		case key.Matches(msg, p.keyMap.Edit):
			i, ok := p.list.SelectedItem().(ProcedureItem)
			if ok {
				p.selectedProcedure = i

				// Populate form with procedure data
				p.editProcedureForm.Fields[0].Input.SetValue(i.name)
				p.editProcedureForm.Fields[1].Input.SetValue(i.version)
				p.editProcedureForm.Fields[2].Input.SetValue(i.description)
				p.editProcedureForm.Fields[3].Input.SetValue(i.author)
				p.editProcedureForm.Fields[4].Input.SetValue("MIT") // Default license

				p.showEditProcedureForm = true
				p.editProcedureForm.Focus()
			}
			return p, nil

		case key.Matches(msg, p.keyMap.Delete):
			i, ok := p.list.SelectedItem().(ProcedureItem)
			if ok {
				p.selectedProcedure = i
				p.showConfirmDelete = true
				p.confirmDeleteModal.Show()
			}
			return p, nil

		case key.Matches(msg, p.keyMap.Run):
			i, ok := p.list.SelectedItem().(ProcedureItem)
			if ok {
				p.selectedProcedure = i
				p.showRunProcedureForm = true
				p.runProcedureForm.Focus()
			}
			return p, nil

		case key.Matches(msg, p.keyMap.Details):
			i, ok := p.list.SelectedItem().(ProcedureItem)
			if ok {
				p.selectedProcedure = i

				// Create details content
				lastRunStr := "Never"
				if !i.lastRun.IsZero() {
					lastRunStr = i.lastRun.Format("2006-01-02 15:04:05")
				}

				// Create a sample procedure steps list
				var stepsStr strings.Builder
				for j := 1; j <= i.steps; j++ {
					stepsStr.WriteString(fmt.Sprintf("%d. Step %d: %s\n", j, j, getStepDescription(i.name, j)))
				}

				details := fmt.Sprintf(
					"Name: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nSteps: %d\nLast Run: %s\n\nSteps:\n%s",
					i.name, i.version, i.description, i.author, i.steps, lastRunStr, stepsStr.String(),
				)

				p.detailsModal.Title = fmt.Sprintf("Procedure: %s (v%s)", i.name, i.version)
				p.detailsModal.Content = details
				p.detailsModal.Show()
				p.showDetailsModal = true
			}
			return p, nil

		case key.Matches(msg, p.keyMap.Refresh):
			return p, p.loadProcedures()
		}

	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.list.SetWidth(msg.Width)
		p.list.SetHeight(msg.Height - 10)
	}

	// Update the list
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

// View renders the screen
func (p ProcedureScreen) View() string {
	if p.loading {
		spinner := components.NewSpinner(p.styles)
		spinner.SetLabel("Loading procedures...")
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			spinner.View(),
		)
	}

	if p.showAddProcedureForm {
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					p.styles.DialogTitle.Render("Add Procedure"),
					"",
					p.addProcedureForm.View(),
					"",
					p.styles.Error.Render(p.error),
				),
			),
		)
	}

	if p.showEditProcedureForm {
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					p.styles.DialogTitle.Render(fmt.Sprintf("Edit Procedure: %s", p.selectedProcedure.name)),
					"",
					p.editProcedureForm.View(),
					"",
					p.styles.Error.Render(p.error),
				),
			),
		)
	}

	if p.showRunProcedureForm {
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					p.styles.DialogTitle.Render(fmt.Sprintf("Run Procedure: %s", p.selectedProcedure.name)),
					"",
					p.runProcedureForm.View(),
					"",
					p.styles.Error.Render(p.error),
				),
			),
		)
	}

	if p.showConfirmDelete {
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.confirmDeleteModal.View(),
		)
	}

	if p.showDetailsModal {
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.detailsModal.View(),
		)
	}

	if p.showRunningModal {
		var content string
		if p.runningInProgress {
			// Create progress bar
			progressBar := components.NewProgressBar(p.styles, 50)
			progressBar.SetPercent(p.runningProgress)

			// Calculate current step
			currentStep := int(p.runningProgress*float64(p.selectedProcedure.steps)) + 1
			if currentStep > p.selectedProcedure.steps {
				currentStep = p.selectedProcedure.steps
			}

			stepDesc := getStepDescription(p.selectedProcedure.name, currentStep)

			content = lipgloss.JoinVertical(
				lipgloss.Left,
				fmt.Sprintf("Running procedure: %s (v%s)", p.selectedProcedure.name, p.selectedProcedure.version),
				fmt.Sprintf("Step %d of %d: %s", currentStep, p.selectedProcedure.steps, stepDesc),
				"",
				progressBar.View(),
			)
		} else {
			content = p.runningResult
		}

		p.runningModal.Content = content

		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			p.runningModal.View(),
		)
	}

	// Main view
	var sb strings.Builder

	// Title
	sb.WriteString(p.list.View())

	// Help
	help := p.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: add • %s: edit • %s: delete • %s: run • %s: details • %s: refresh • %s: back",
		p.styles.KeyBinding.Render("↑/↓"),
		p.styles.KeyBinding.Render("a"),
		p.styles.KeyBinding.Render("e"),
		p.styles.KeyBinding.Render("d"),
		p.styles.KeyBinding.Render("r"),
		p.styles.KeyBinding.Render("enter"),
		p.styles.KeyBinding.Render("f5"),
		p.styles.KeyBinding.Render("esc"),
	))

	sb.WriteString("\n\n")
	sb.WriteString(help)

	// Error
	if p.error != "" {
		sb.WriteString("\n\n")
		sb.WriteString(p.styles.Error.Render(p.error))
	}

	return sb.String()
}

// getStepDescription returns a description for a procedure step
func getStepDescription(procedureName string, step int) string {
	// In a real implementation, this would get the actual step description
	// For now, we'll return some sample descriptions

	switch procedureName {
	case "Capability Registration":
		steps := []string{
			"Validate plugin file",
			"Validate manifest file",
			"Generate file references",
			"Prepare capability registration",
			"Submit transaction",
		}
		if step <= len(steps) {
			return steps[step-1]
		}
	case "Server Setup":
		steps := []string{
			"Check system requirements",
			"Install dependencies",
			"Configure network settings",
			"Initialize database",
			"Generate server keys",
			"Configure security settings",
			"Start services",
			"Verify installation",
		}
		if step <= len(steps) {
			return steps[step-1]
		}
	case "Wallet Backup":
		steps := []string{
			"Decrypt wallet",
			"Encrypt with backup key",
			"Store in secure location",
		}
		if step <= len(steps) {
			return steps[step-1]
		}
	case "Network Diagnostics":
		steps := []string{
			"Check network connectivity",
			"Verify DNS resolution",
			"Test API endpoints",
			"Measure latency",
			"Check transaction throughput",
			"Verify block propagation",
			"Test consensus mechanism",
			"Check validator status",
			"Verify capability registry",
			"Generate diagnostic report",
		}
		if step <= len(steps) {
			return steps[step-1]
		}
	}

	return fmt.Sprintf("Execute step %d", step)
}
