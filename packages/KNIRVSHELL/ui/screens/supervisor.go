package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/incident"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/registry"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/watchtower"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type supervisorLoadedMsg struct {
	agents []registry.AgentRecord
	err    error
}

type supervisorLogMsg struct {
	line watchtower.UILogLine
}

type supervisorIncidentMsg struct {
	incident *incident.Incident
}

type supervisorResultMsg struct {
	result watchtower.AgentResult
}

type SupervisorModel struct {
	styles      ui.Styles
	store       *registry.Store
	agents      []registry.AgentRecord
	selected    int
	logs        []string
	incidents   []string
	status      string
	runningName string
	runningStop func()
	events      chan tea.Msg
	width       int
	height      int
}

func NewSupervisorModel(themeName, registryPath string) *SupervisorModel {
	return &SupervisorModel{
		styles: ui.DefaultStyles(ui.GetThemeByName(themeName)),
		store:  registry.NewStore(registryPath),
		status: "Loading registry...",
		events: make(chan tea.Msg, 128),
		width:  120,
		height: 30,
	}
}

func (m *SupervisorModel) Init() tea.Cmd {
	return tea.Batch(loadSupervisorAgents(m.store), waitForSupervisorEvent(m.events))
}

func (m *SupervisorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.agents)-1 {
				m.selected++
			}
		case "r":
			m.status = "Refreshing registry..."
			return m, tea.Batch(loadSupervisorAgents(m.store), waitForSupervisorEvent(m.events))
		case "enter":
			if m.runningName == "" {
				return m, tea.Batch(m.startSelectedAgent(), waitForSupervisorEvent(m.events))
			}
		case "s":
			if m.runningStop != nil {
				m.runningStop()
				m.status = fmt.Sprintf("Stopping %s...", m.runningName)
			}
		}
	case supervisorLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.agents = msg.agents
			if m.selected >= len(m.agents) {
				m.selected = max(0, len(m.agents)-1)
			}
			m.status = fmt.Sprintf("Loaded %d agent(s) from %s", len(m.agents), m.store.Path())
		}
		return m, waitForSupervisorEvent(m.events)
	case supervisorLogMsg:
		m.logs = appendTrimmed(m.logs, msg.line.Line, 16)
		return m, waitForSupervisorEvent(m.events)
	case supervisorIncidentMsg:
		label := fmt.Sprintf("%s (%s)", msg.incident.ID, msg.incident.ErrorSignature)
		m.incidents = appendTrimmed(m.incidents, label, 8)
		return m, waitForSupervisorEvent(m.events)
	case supervisorResultMsg:
		m.runningName = ""
		m.runningStop = nil
		if msg.result.Err != nil {
			m.status = fmt.Sprintf("Agent exited with error: %v", msg.result.Err)
		} else {
			m.status = fmt.Sprintf("Agent exited with code %d", msg.result.ExitCode)
		}
		return m, waitForSupervisorEvent(m.events)
	}

	return m, nil
}

func (m *SupervisorModel) View() string {
	title := m.styles.Title.Render("KNIRVSHELL Supervisor")
	status := m.styles.Subtle.Render(m.status)

	agents := m.renderAgents()
	logs := m.renderPanel("Logs", m.logs)
	incidents := m.renderPanel("Incidents", m.incidents)

	bottom := lipgloss.JoinHorizontal(lipgloss.Top, logs, incidents)
	help := m.styles.HelpText.Render("enter: run  s: stop  r: refresh  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, status, agents, bottom, help)
}

func (m *SupervisorModel) startSelectedAgent() tea.Cmd {
	if len(m.agents) == 0 {
		m.status = "No registered agents available"
		return nil
	}

	record := m.agents[m.selected]
	m.runningName = record.Name
	m.status = fmt.Sprintf("Starting %s...", record.Name)

	return func() tea.Msg {
		executor, err := runner.ForType(record.Type)
		if err != nil {
			return supervisorResultMsg{result: watchtower.AgentResult{Err: err}}
		}

		ctx, cancel := context.WithCancel(context.Background())
		proc, err := executor.Start(ctx, record.RunnerConfig())
		if err != nil {
			cancel()
			return supervisorResultMsg{result: watchtower.AgentResult{Err: err}}
		}

		m.runningStop = func() {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer stopCancel()
			_ = proc.StopFn(stopCtx)
			cancel()
		}

		uiCh := make(chan watchtower.UILogLine, 128)
		incCh := make(chan *incident.Incident, 16)
		tower := watchtower.New(uiCh, incCh)
		resultCh := tower.Watch(ctx, proc)

		go func() {
			for line := range uiCh {
				m.events <- supervisorLogMsg{line: line}
			}
		}()

		go func() {
			for inc := range incCh {
				m.events <- supervisorIncidentMsg{incident: inc}
			}
		}()

		go func() {
			result := <-resultCh
			m.events <- supervisorResultMsg{result: result}
		}()

		return supervisorLoadedMsg{agents: m.agents}
	}
}

func (m *SupervisorModel) renderAgents() string {
	lines := make([]string, 0, len(m.agents))
	for i, agent := range m.agents {
		line := fmt.Sprintf("%s [%s]", agent.Name, agent.Type)
		if agent.Description != "" {
			line = fmt.Sprintf("%s - %s", line, agent.Description)
		}
		if i == m.selected {
			line = m.styles.ListItemSelected.Render("> " + line)
		} else {
			line = m.styles.ListItem.Render("  " + line)
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.Subtle.Render("No agents registered yet. Use `knirv register` first."))
	}

	return m.styles.Panel.Width(max(40, m.width-6)).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Header.Render("Agents"),
			strings.Join(lines, "\n"),
		),
	)
}

func (m *SupervisorModel) renderPanel(title string, lines []string) string {
	body := strings.Join(lines, "\n")
	if body == "" {
		body = m.styles.Subtle.Render("No entries yet.")
	}

	return m.styles.Panel.Width(max(40, (m.width/2)-4)).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Header.Render(title),
			body,
		),
	)
}

func loadSupervisorAgents(store *registry.Store) tea.Cmd {
	return func() tea.Msg {
		agents, err := store.List()
		return supervisorLoadedMsg{agents: agents, err: err}
	}
}

func waitForSupervisorEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-events
	}
}

func appendTrimmed(values []string, next string, limit int) []string {
	values = append(values, next)
	if len(values) > limit {
		values = values[len(values)-limit:]
	}
	return values
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
