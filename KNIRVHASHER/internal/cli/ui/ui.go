package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	psutil "github.com/shirou/gopsutil/v3/cpu"
	psmem "github.com/shirou/gopsutil/v3/mem"

	"hasher/internal/cli/server"
)

// View states
const (
	MainMenuView = iota
	ChatView
	ProgressView
	LogView
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFFF00")).
			Padding(0, 2).
			Bold(true).
			Width(80)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4B5563")).
			Padding(0, 2).
			Width(80)

	chatViewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#9CA3AF"))

	logViewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#9CA3AF"))

	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#60A5FA")).
				Bold(true)

	llmMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399"))

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2563EB")).
			Padding(0, 1)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2563EB"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#2563EB")).
				Bold(true)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))
)

// Menu item definitions
type menuItem struct {
	title       string
	description string
	view        int
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.description }
func (i menuItem) FilterValue() string { return i.title }

var menuItems = []list.Item{
	menuItem{
		title:       "1. Discovery",
		description: "Discover ASIC devices on the network",
		view:        MainMenuView,
	},
	menuItem{
		title:       "2. Probe",
		description: "Probe connected ASIC device",
		view:        MainMenuView,
	},
	menuItem{
		title:       "3. Protocol",
		description: "Detect ASIC device protocol",
		view:        MainMenuView,
	},
	menuItem{
		title:       "4. Provision",
		description: "Deploy pixie-server to ASIC device",
		view:        MainMenuView,
	},
	menuItem{
		title:       "5. Troubleshoot",
		description: "Run troubleshooting diagnostics",
		view:        MainMenuView,
	},
	menuItem{
		title:       "6. Configure",
		description: "Configure hasher inference service",
		view:        MainMenuView,
	},
	menuItem{
		title:       "7. Test",
		description: "Test hasher validation service",
		view:        MainMenuView,
	},
	menuItem{
		title:       "8. Chat",
		description: "Chat with hasher inference service (-llama for LLM)",
		view:        ChatView,
	},
	menuItem{
		title:       "9. Quit",
		description: "Exit the application",
		view:        MainMenuView,
	},
}

// Model represents the application state
type Model struct {
	CurrentView    int
	MainMenu       list.Model
	ChatView       viewport.Model
	LogView        viewport.Model
	Input          textarea.Model
	ServerCmd      *exec.Cmd
	ServerLogs     []string
	ChatHistory    []string
	ServerReady    bool
	ResourceData   string
	Width          int
	Height         int
	ProgressText   string
	ProgressStatus string
}

// NewModel creates a new UI model
func NewModel() Model {
	// Initialize menu
	menuList := list.New(menuItems, list.NewDefaultDelegate(), 0, 0)
	menuList.Title = "Hasher CLI - Main Menu"
	menuList.SetShowStatusBar(false)
	menuList.SetFilteringEnabled(false)

	// Initialize chat view
	chatView := viewport.New(78, 20)

	// Initialize log view
	logView := viewport.New(78, 20)

	// Initialize input area
	input := textarea.New()
	input.Placeholder = "Type your message here (or /quit to exit)..."
	input.Focus()
	input.Prompt = ""
	input.SetHeight(1)
	input.SetWidth(76)
	input.ShowLineNumbers = false
	input.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB"))

	// Create model with initial data
	model := Model{
		CurrentView:    MainMenuView,
		MainMenu:       menuList,
		ChatView:       chatView,
		LogView:        logView,
		Input:          input,
		ServerLogs:     []string{"Logs will appear here..."},
		ChatHistory:    []string{"Welcome to Hasher CLI!\n\nType your message below. Use -llama for LLM."},
		ServerReady:    false,
		Width:          80,
		Height:         24,
		ProgressText:   "",
		ProgressStatus: "",
	}

	// Initialize views
	model.updateChatView()
	model.updateLogView()

	return model
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.ClearScreen,
		m.updateResourceData(),
	)
}

// Update handles UI updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m, cmd = m.handleResize(msg)
		cmds = append(cmds, cmd)

	case updateResourceDataMsg:
		m.ResourceData = msg.data
		cmds = append(cmds, m.updateResourceData())

	case AppendLogMsg:
		m.ServerLogs = append(m.ServerLogs, msg.Log)
		if len(m.ServerLogs) > 50 {
			m.ServerLogs = m.ServerLogs[len(m.ServerLogs)-50:]
		}
		m.updateLogView()

	case AppendChatMsg:
		m.ChatHistory = append(m.ChatHistory, msg.Msg)
		m.updateChatView()

	case CombinedLogChatMsg:
		m.ServerLogs = append(m.ServerLogs, msg.Log)
		if len(m.ServerLogs) > 50 {
			m.ServerLogs = m.ServerLogs[len(m.ServerLogs)-50:]
		}
		m.updateLogView()
		m.ChatHistory = append(m.ChatHistory, msg.Chat)
		m.updateChatView()

	case ProgressUpdateMsg:
		m.ProgressText = msg.text
		m.ProgressStatus = msg.status
	}

	switch m.CurrentView {
	case MainMenuView:
		m.MainMenu, cmd = m.MainMenu.Update(msg)
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.Type {
			case tea.KeyEnter:
				if i, ok := m.MainMenu.SelectedItem().(menuItem); ok {
					switch i.title {
					case "1. Discovery":
						cmds = append(cmds, m.runDiscovery)
					case "2. Probe":
						cmds = append(cmds, m.runProbe)
					case "3. Protocol":
						cmds = append(cmds, m.runProtocol)
					case "4. Provision":
						cmds = append(cmds, m.runProvision)
					case "5. Troubleshoot":
						cmds = append(cmds, m.runTroubleshoot)
					case "6. Configure":
						cmds = append(cmds, m.runConfigure)
					case "7. Test":
						cmds = append(cmds, m.runTest)
					case "8. Chat":
						m.CurrentView = ChatView
					case "9. Quit":
						return m, tea.Quit
					}
				}
			}
		}

	case ChatView:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				if m.Input.Value() != "" {
					cmds = append(cmds, m.handleInput(m.Input.Value()))
					m.Input.Reset()
				}
			case tea.KeyEsc:
				m.CurrentView = MainMenuView
			}
		}

		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)

		m.ChatView, cmd = m.ChatView.Update(msg)
		cmds = append(cmds, cmd)

		m.LogView, cmd = m.LogView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	switch m.CurrentView {
	case MainMenuView:
		return m.renderMainMenu()
	case ChatView:
		return m.renderChatView()
	case ProgressView:
		return m.renderProgressView()
	}

	return m.renderMainMenu()
}

// renderMainMenu renders the main menu
func (m Model) renderMainMenu() string {
	serverStatus := "Server: Stopped"
	if m.ServerReady {
		serverStatus = "Server: Ready"
	}
	headerContent := fmt.Sprintf(" Hasher CLI Tool | %s", serverStatus)
	header := headerStyle.Copy().Width(m.Width).Render(headerContent)

	footer := footerStyle.Copy().Width(m.Width).Render(m.ResourceData)

	menuContent := listStyle.Copy().Width(m.Width - 4).Height(m.Height - 6).Render(m.MainMenu.View())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		menuContent,
		footer,
	)
}

// renderChatView renders the chat interface
func (m Model) renderChatView() string {
	serverStatus := "Server: Stopped"
	if m.ServerReady {
		serverStatus = "Server: Ready"
	}
	headerContent := fmt.Sprintf(" Hasher Chat | %s | Press ESC for menu", serverStatus)
	header := headerStyle.Copy().Width(m.Width).Render(headerContent)

	footer := footerStyle.Copy().Width(m.Width).Render(m.ResourceData)

	// Calculate available height: total height - header(1) - footer(1) - input(3 lines with border)
	availableHeight := m.Height - 1 - 1 - 3
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Vertical layout: Chat view on top, Log view below (better for text selection)
	chatHeight := availableHeight / 2
	logHeight := availableHeight - chatHeight

	m.ChatView.Width = m.Width - 4
	m.ChatView.Height = chatHeight - 2
	m.LogView.Width = m.Width - 4
	m.LogView.Height = logHeight - 2

	chatContent := chatViewStyle.Copy().
		Width(m.Width - 2).
		Height(chatHeight).
		Render(m.ChatView.View())

	logContent := logViewStyle.Copy().
		Width(m.Width - 2).
		Height(logHeight).
		Render(m.LogView.View())

	columns := lipgloss.JoinVertical(
		lipgloss.Left,
		chatContent,
		logContent,
	)

	input := inputStyle.Copy().Width(m.Width - 4).Render(m.Input.View())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		columns,
		input,
		footer,
	)
}

// renderProgressView renders the progress indicator
func (m Model) renderProgressView() string {
	header := headerStyle.Copy().Width(m.Width).Render(" Hasher CLI - Processing")
	footer := footerStyle.Copy().Width(m.Width).Render(m.ResourceData)

	progress := fmt.Sprintf("Processing: %s", m.ProgressText)
	if m.ProgressStatus != "" {
		progress += fmt.Sprintf("\nStatus: %s", m.ProgressStatus)
	}

	content := lipgloss.NewStyle().
		Padding(2, 4).
		Width(m.Width - 4).
		Height(m.Height - 6).
		Render(progress)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// handleResize adjusts layout for window resizing
func (m Model) handleResize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height

	m.MainMenu.SetSize(msg.Width-4, msg.Height-6)

	// Calculate available height: total height - header(1) - footer(1) - input(3 lines with border)
	availableHeight := msg.Height - 1 - 1 - 3
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Vertical layout dimensions
	chatHeight := availableHeight / 2
	logHeight := availableHeight - chatHeight

	m.ChatView.Width = msg.Width - 4
	m.ChatView.Height = chatHeight - 2
	m.LogView.Width = msg.Width - 4
	m.LogView.Height = logHeight - 2

	m.Input.SetWidth(msg.Width - 6)
	m.Input.SetHeight(1)

	headerStyle = headerStyle.Width(msg.Width)
	footerStyle = footerStyle.Width(msg.Width)

	m.updateChatView()
	m.updateLogView()

	return m, nil
}

// updateChatView updates the chat view with history
func (m *Model) updateChatView() {
	var content string
	width := m.ChatView.Width
	for _, msg := range m.ChatHistory {
		// Word wrap message to viewport width
		wrappedMsg := ansi.Wordwrap(msg, width, " \t")
		content += wrappedMsg + "\n\n"
	}
	m.ChatView.SetContent(content)
	m.ChatView.GotoBottom()
}

// updateLogView updates the log view with server logs
func (m *Model) updateLogView() {
	var content string
	width := m.LogView.Width
	for _, log := range m.ServerLogs {
		// Word wrap log entry to viewport width
		wrappedLog := ansi.Wordwrap(log, width, " \t")
		content += wrappedLog + "\n"
	}
	m.LogView.SetContent(content)
	m.LogView.GotoBottom()
}

// updateResourceData updates resource usage information
func (m Model) updateResourceData() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		cpuPercent, _ := psutil.Percent(0, false)
		memInfo, _ := psmem.VirtualMemory()

		data := fmt.Sprintf("CPU: %.1f%% | RAM: %.1f%% | Go: %s",
			cpuPercent[0], memInfo.UsedPercent, runtime.Version())
		return updateResourceDataMsg{data}
	})
}

// handleInput processes user input
func (m Model) handleInput(input string) tea.Cmd {
	if input == "/quit" {
		return tea.Quit
	}
	if input == "/menu" {
		return func() tea.Msg {
			m.CurrentView = MainMenuView
			return nil
		}
	}

	userMsg := userMessageStyle.Render("You: " + input)
	logStart := fmt.Sprintf("[%s] Sending to service: %s", time.Now().Format("15:04:05"), input)
	thinkingMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true).Render("Processing...")

	return tea.Batch(
		func() tea.Msg {
			return CombinedLogChatMsg{Log: logStart, Chat: userMsg}
		},
		func() tea.Msg {
			return AppendChatMsg{Msg: thinkingMsg}
		},
		func() tea.Msg {
			var resp string
			var err error

			if strings.Contains(strings.ToLower(input), "-llama") {
				resp, err = server.CallTinyLLMAPI(input)
			} else {
				resp = "Hasher service is not yet implemented"
				err = nil
			}

			if err != nil {
				logErr := fmt.Sprintf("[%s] API Error: %v", time.Now().Format("15:04:05"), err)
				errMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("Error: " + err.Error())
				return CombinedLogChatMsg{Log: logErr, Chat: errMsg}
			}

			logResp := fmt.Sprintf("[%s] Response received (%d chars)", time.Now().Format("15:04:05"), len(resp))
			llmMsg := llmMessageStyle.Render("Hasher: " + resp)
			return CombinedLogChatMsg{Log: logResp, Chat: llmMsg}
		},
	)
}

// runDiscovery runs device discovery
func (m Model) runDiscovery() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Discovering ASIC devices..."
	m.ProgressStatus = "Scanning network..."

	go func() {
		time.Sleep(1 * time.Second)
		// TODO: Implement discovery logic
		m.ProgressStatus = "Scan complete"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Discovering ASIC devices...", status: "Scanning network..."}
}

// runProbe runs device probe
func (m Model) runProbe() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Probing ASIC device..."
	m.ProgressStatus = "Running device diagnostics..."

	go func() {
		time.Sleep(2 * time.Second)
		// TODO: Implement probe logic
		m.ProgressStatus = "Probe complete"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Probing ASIC device...", status: "Running diagnostics..."}
}

// runProtocol runs protocol detection
func (m Model) runProtocol() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Detecting protocol..."
	m.ProgressStatus = "Testing communication protocols..."

	go func() {
		time.Sleep(2 * time.Second)
		// TODO: Implement protocol detection
		m.ProgressStatus = "Protocol detected"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Detecting protocol...", status: "Testing communication..."}
}

// runProvision runs device provisioning
func (m Model) runProvision() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Provisioning device..."
	m.ProgressStatus = "Deploying pixie-server..."

	go func() {
		time.Sleep(3 * time.Second)
		// TODO: Implement provisioning logic
		m.ProgressStatus = "Provisioning complete"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Provisioning device...", status: "Deploying pixie-server..."}
}

// runTroubleshoot runs troubleshooting
func (m Model) runTroubleshoot() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Troubleshooting..."
	m.ProgressStatus = "Running diagnostic tests..."

	go func() {
		time.Sleep(2 * time.Second)
		// TODO: Implement troubleshooting
		m.ProgressStatus = "Diagnostics complete"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Troubleshooting...", status: "Running diagnostics..."}
}

// runConfigure runs configuration
func (m Model) runConfigure() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Configuring service..."
	m.ProgressStatus = "Setting up hasher service..."

	go func() {
		time.Sleep(2 * time.Second)
		// TODO: Implement configuration
		m.ProgressStatus = "Configuration complete"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Configuring service...", status: "Setting up service..."}
}

// runTest runs service tests
func (m Model) runTest() tea.Msg {
	m.CurrentView = ProgressView
	m.ProgressText = "Testing service..."
	m.ProgressStatus = "Running validation tests..."

	go func() {
		time.Sleep(1 * time.Second)
		// TODO: Implement test logic
		m.ProgressStatus = "Tests passed"
		time.Sleep(1 * time.Second)
		m.CurrentView = MainMenuView
	}()

	return ProgressUpdateMsg{text: "Testing service...", status: "Running validation..."}
}

// Messages
type updateResourceDataMsg struct {
	data string
}

type AppendLogMsg struct {
	Log string
}

type AppendChatMsg struct {
	Msg string
}

type CombinedLogChatMsg struct {
	Log  string
	Chat string
}

type ProgressUpdateMsg struct {
	text   string
	status string
}
