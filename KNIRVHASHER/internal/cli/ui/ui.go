package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	psutil "github.com/shirou/gopsutil/v3/cpu"
	psmem "github.com/shirou/gopsutil/v3/mem"

	"github.com/gperry/devtools/claude-cli/internal/server"
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
)

// Model represents the application state
type Model struct {
	ChatView     viewport.Model
	LogView      viewport.Model
	Input        textarea.Model
	ServerCmd    *exec.Cmd
	ServerLogs   []string
	ChatHistory  []string
	ServerReady  bool
	ResourceData string
	Width        int
	Height       int
}

// NewModel creates a new UI model
func NewModel() Model {
	// Initialize chat view with default dimensions (use full available space)
	chatView := viewport.New(78, 20) // Full width minus borders, more height

	// Initialize log view with default dimensions (use full available space)
	logView := viewport.New(78, 20) // Full width minus borders, more height

	// Initialize input area with fixed height
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
		ChatView:    chatView,
		LogView:     logView,
		Input:       input,
		ServerLogs:  []string{"Logs will appear here..."},
		ChatHistory: []string{"Welcome to Tiny-LLM CLI!\n\nType your message below. Available commands:\n  /file <path> - Attach a file to your message\n  /sudo <command> - Execute a command with sudo\n  /quit - Exit the tool\n  /reset - Clear configuration and reconfigure"},
		ServerReady: false,
		Width:       80,
		Height:      24,
	}

	// Initialize styles with default dimensions
	headerStyle = headerStyle.Width(80)
	footerStyle = footerStyle.Width(80)

	// Initialize chat view content
	model.updateChatView()
	// Initialize log view content
	model.updateLogView()

	return model
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Clear screen and start resource monitoring
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
		case tea.KeyEnter:
			if m.Input.Value() != "" {
				cmds = append(cmds, m.handleInput(m.Input.Value()))
				m.Input.Reset()
			}
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
		// Append log message
		m.ServerLogs = append(m.ServerLogs, msg.Log)
		if len(m.ServerLogs) > 50 {
			m.ServerLogs = m.ServerLogs[len(m.ServerLogs)-50:]
		}
		m.updateLogView()

		// Append chat message
		m.ChatHistory = append(m.ChatHistory, msg.Chat)
		m.updateChatView()
	}

	// Update input
	m.Input, cmd = m.Input.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewports
	m.ChatView, cmd = m.ChatView.Update(msg)
	cmds = append(cmds, cmd)

	m.LogView, cmd = m.LogView.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	// Build header - ensure it's visible and properly formatted
	serverStatus := "Server: Stopped"
	if m.ServerReady {
		serverStatus = "Server: Ready"
	}
	headerContent := fmt.Sprintf(" Tiny-LLM CLI Tool | %s", serverStatus)
	header := headerStyle.Copy().Width(m.Width).Render(headerContent)

	// Build footer first so we know its height
	footer := footerStyle.Copy().Width(m.Width).Render(m.ResourceData)

	// Calculate dimensions for the content area
	// Header: 1 line, Footer: 1 line, Input: 3 lines (with border), Column borders: 2 lines each
	availableHeight := m.Height - 1 - 1 - 3 - 2
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate column widths (split screen in half, accounting for spacing and borders)
	halfWidth := (m.Width - 3) / 2 // -3 for spacing between columns
	chatWidth := halfWidth
	logWidth := halfWidth

	// Update viewport dimensions (subtract 2 for borders)
	m.ChatView.Width = chatWidth - 2
	m.ChatView.Height = availableHeight - 2
	m.LogView.Width = logWidth - 2
	m.LogView.Height = availableHeight - 2

	// Build chat view with border
	chatContent := chatViewStyle.Copy().
		Width(chatWidth).
		Height(availableHeight).
		Render(m.ChatView.View())

	// Build log view with border
	logContent := logViewStyle.Copy().
		Width(logWidth).
		Height(availableHeight).
		Render(m.LogView.View())

	// Build columns with spacing
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		chatContent,
		" ",
		logContent,
	)

	// Build input area with border
	input := inputStyle.Copy().Width(m.Width - 4).Render(m.Input.View())

	// Combine all parts vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		columns,
		input,
		footer,
	)
}

// handleResize adjusts layout for window resizing
func (m Model) handleResize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height

	// Calculate available height: total - header(1) - footer(1) - input(3) - column borders(2)
	availableHeight := msg.Height - 1 - 1 - 3 - 2
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate column widths
	halfWidth := (msg.Width - 3) / 2
	chatWidth := halfWidth
	logWidth := halfWidth

	// Update viewports (subtract 2 for borders)
	m.ChatView.Width = chatWidth - 2
	m.ChatView.Height = availableHeight - 2
	m.LogView.Width = logWidth - 2
	m.LogView.Height = availableHeight - 2

	// Update textarea input
	m.Input.SetWidth(msg.Width - 6)
	m.Input.SetHeight(1)

	// Update styles
	headerStyle = headerStyle.Width(msg.Width)
	footerStyle = footerStyle.Width(msg.Width)

	// Update views
	m.updateChatView()
	m.updateLogView()

	return m, nil
}

// updateChatView updates the chat view with history
func (m *Model) updateChatView() {
	var content string
	for _, msg := range m.ChatHistory {
		content += msg + "\n\n"
	}
	m.ChatView.SetContent(content)
	m.ChatView.GotoBottom()
}

// updateLogView updates the log view with server logs
func (m *Model) updateLogView() {
	var content string
	for _, log := range m.ServerLogs {
		content += log + "\n"
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

	// Log the input
	logMsg := fmt.Sprintf("[%s] User input: %s", time.Now().Format("15:04:05"), input)

	if input == "/reset" {
		return func() tea.Msg {
			return AppendLogMsg{Log: logMsg + " (reset command)"}
		}
	}

	// Handle file attachments
	if strings.HasPrefix(input, "/file ") {
		return func() tea.Msg {
			return AppendLogMsg{Log: logMsg + " (file attachment)"}
		}
	}

	// Handle sudo commands
	if strings.HasPrefix(input, "/sudo ") {
		return func() tea.Msg {
			return AppendLogMsg{Log: logMsg + " (sudo command)"}
		}
	}

	// Add user message to chat history via message
	userMsg := userMessageStyle.Render("You: " + input)
	logStart := fmt.Sprintf("[%s] Sending to LLM: %s", time.Now().Format("15:04:05"), input)
	thinkingMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true).Render("Tiny-LLM is thinking...")

	// Call Tiny-LLM API
	userInput := input // Capture for closure
	return tea.Batch(
		// First send the user message and log the request
		func() tea.Msg {
			return CombinedLogChatMsg{Log: logStart, Chat: userMsg}
		},
		// Show thinking indicator
		func() tea.Msg {
			return AppendChatMsg{Msg: thinkingMsg}
		},
		// Then call the API and send the response
		func() tea.Msg {
			resp, err := server.CallTinyLLMAPI(userInput)
			if err != nil {
				logErr := fmt.Sprintf("[%s] API Error: %v", time.Now().Format("15:04:05"), err)
				// Show error in both log and chat so user sees it
				errMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("Error: " + err.Error())
				return CombinedLogChatMsg{Log: logErr, Chat: errMsg}
			}

			logResp := fmt.Sprintf("[%s] LLM Response received (%d chars)", time.Now().Format("15:04:05"), len(resp))
			llmMsg := llmMessageStyle.Render("Tiny-LLM: " + resp)
			return CombinedLogChatMsg{Log: logResp, Chat: llmMsg}
		},
	)
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

// CombinedLogChatMsg is used to update both log and chat views together
type CombinedLogChatMsg struct {
	Log  string
	Chat string
}
