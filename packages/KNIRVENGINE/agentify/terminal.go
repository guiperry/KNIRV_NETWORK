// terminal.go
package agentify

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

// LogMessage represents a log message with metadata
type LogMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "stdout", "stderr", "pty"
	Data      []byte    `json:"data"`
	ProcessID int       `json:"process_id,omitempty"`
	Source    string    `json:"source"` // "terminal", "agent", "system"
}

// TerminalSession represents an interactive terminal session for an agent
type TerminalSession struct {
	// The ID of the terminal session
	ID string

	// The agent plugin associated with the terminal
	Agent AgentPluginInterface

	// The PTY for the terminal
	PTY *os.File

	// The process running in the terminal
	Process *exec.Cmd

	// The size of the terminal
	Rows int
	Cols int

	// The input/output buffers
	inputBuffer  []byte
	outputBuffer []byte

	// The mutex for synchronizing access to the terminal
	mutex sync.RWMutex

	// Whether the terminal is active
	active bool

	// Channel for terminal output
	outputChan chan []byte

	// Channel for terminal close
	closeChan chan struct{}

	// Log buffer for comprehensive process output
	logBuffer []LogMessage

	// Channel for log messages
	logChan chan LogMessage
}

// TerminalManager manages terminal sessions for agents
type TerminalManager struct {
	// The active terminal sessions
	sessions map[string]*TerminalSession

	// The mutex for synchronizing access to the sessions
	mutex sync.RWMutex
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	return &TerminalManager{
		sessions: make(map[string]*TerminalSession),
	}
}

// NewTerminalSession creates a new terminal session for an agent
func (m *TerminalManager) NewTerminalSession(agent AgentPluginInterface, rows, cols int) (*TerminalSession, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate a unique session ID
	sessionID := uuid.New().String()

	// Create the terminal session
	session := &TerminalSession{
		ID:           sessionID,
		Agent:        agent,
		Rows:         rows,
		Cols:         cols,
		inputBuffer:  make([]byte, 0),
		outputBuffer: make([]byte, 0),
		active:       false,
		outputChan:   make(chan []byte, 100),
		closeChan:    make(chan struct{}),
		logBuffer:    make([]LogMessage, 0),
		logChan:      make(chan LogMessage, 200), // Larger buffer for comprehensive logging
	}

	// Start a shell process with PTY
	cmd := exec.Command("/bin/bash")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Create a PTY
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %v", err)
	}

	session.PTY = ptyFile
	session.Process = cmd
	session.active = true

	// Set the terminal size
	if err := pty.Setsize(ptyFile, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}); err != nil {
		return nil, fmt.Errorf("failed to set terminal size: %v", err)
	}

	// Start reading from the PTY
	go session.readFromPTY()

	// Start log relay service
	go session.relayLogs()

	// Store the session
	m.sessions[sessionID] = session

	return session, nil
}

// readFromPTY reads output from the PTY and sends it to the output channel
func (s *TerminalSession) readFromPTY() {
	defer close(s.outputChan)

	reader := bufio.NewReader(s.PTY)
	buffer := make([]byte, 1024)

	for s.active {
		n, err := reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading from PTY: %v\n", err)
			}
			break
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])

			// Add to output buffer
			s.mutex.Lock()
			s.outputBuffer = append(s.outputBuffer, data...)
			s.mutex.Unlock()

			// Create log message for PTY output
			logMsg := LogMessage{
				Timestamp: time.Now(),
				Type:      "pty",
				Data:      data,
				Source:    "terminal",
			}

			// Send to log channel
			select {
			case s.logChan <- logMsg:
			default:
				// Log channel is full, skip this message
			}

			// Send to output channel
			select {
			case s.outputChan <- data:
			case <-s.closeChan:
				return
			default:
				// Channel is full, skip this data
			}
		}
	}
}

// relayLogs processes log messages and maintains the log buffer
func (s *TerminalSession) relayLogs() {
	defer func() {
		// Only close logChan if it hasn't been closed yet
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if s.active {
			close(s.logChan)
			s.active = false
		}
	}()

	for {
		select {
		case logMsg, ok := <-s.logChan:
			if !ok {
				return
			}

			// Add to log buffer
			s.mutex.Lock()
			s.logBuffer = append(s.logBuffer, logMsg)

			// Keep only the last 1000 log messages to prevent memory issues
			if len(s.logBuffer) > 1000 {
				s.logBuffer = s.logBuffer[len(s.logBuffer)-1000:]
			}
			s.mutex.Unlock()

		case <-s.closeChan:
			return
		}
	}
}

// LogProcessOutput logs stdout/stderr from agent processes
func (s *TerminalSession) LogProcessOutput(outputType string, data []byte, processID int) {
	if !s.active {
		return
	}

	logMsg := LogMessage{
		Timestamp: time.Now(),
		Type:      outputType, // "stdout" or "stderr"
		Data:      data,
		ProcessID: processID,
		Source:    "agent",
	}

	// Send to log channel
	select {
	case s.logChan <- logMsg:
	default:
		// Log channel is full, skip this message
	}

	// Also send to output channel for real-time display
	select {
	case s.outputChan <- data:
	default:
		// Output channel is full, skip this data
	}
}

// GetLogBuffer returns the current log buffer
func (s *TerminalSession) GetLogBuffer() []LogMessage {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy of the log buffer
	logs := make([]LogMessage, len(s.logBuffer))
	copy(logs, s.logBuffer)
	return logs
}

// ResizeTerminal resizes a terminal session
func (m *TerminalManager) ResizeTerminal(sessionID string, rows, cols int) error {
	m.mutex.RLock()
	session, ok := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("terminal session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Rows = rows
	session.Cols = cols

	if session.PTY != nil {
		return pty.Setsize(session.PTY, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		})
	}

	return nil
}

// WriteToTerminal writes data to a terminal session
func (m *TerminalManager) WriteToTerminal(sessionID string, data []byte) error {
	m.mutex.RLock()
	session, ok := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("terminal session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.PTY != nil && session.active {
		_, err := session.PTY.Write(data)
		return err
	}

	return fmt.Errorf("terminal session not active: %s", sessionID)
}

// ReadFromTerminal reads data from a terminal session
func (m *TerminalManager) ReadFromTerminal(sessionID string) ([]byte, error) {
	m.mutex.RLock()
	session, ok := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// Return and clear the output buffer
	data := make([]byte, len(session.outputBuffer))
	copy(data, session.outputBuffer)
	session.outputBuffer = session.outputBuffer[:0]

	return data, nil
}

// GetTerminalOutput gets the output channel for a terminal session
func (m *TerminalManager) GetTerminalOutput(sessionID string) (<-chan []byte, error) {
	m.mutex.RLock()
	session, ok := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", sessionID)
	}

	return session.outputChan, nil
}

// CloseTerminalSession closes a terminal session
func (m *TerminalManager) CloseTerminalSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("terminal session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// Mark as inactive
	session.active = false

	// Close the close channel
	close(session.closeChan)

	// Close the log channel
	close(session.logChan)

	// Close the PTY
	if session.PTY != nil {
		session.PTY.Close()
	}

	// Kill the process
	if session.Process != nil && session.Process.Process != nil {
		session.Process.Process.Kill()
		session.Process.Wait()
	}

	// Remove from sessions map
	delete(m.sessions, sessionID)

	return nil
}

// GetTerminalSession gets a terminal session by ID
func (m *TerminalManager) GetTerminalSession(sessionID string) (*TerminalSession, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", sessionID)
	}

	return session, nil
}

// ListTerminalSessions lists all active terminal sessions
func (m *TerminalManager) ListTerminalSessions() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessions := make([]string, 0, len(m.sessions))
	for sessionID := range m.sessions {
		sessions = append(sessions, sessionID)
	}

	return sessions
}

// LogAgentProcessOutput logs output from an agent process to all its terminal sessions
func (m *TerminalManager) LogAgentProcessOutput(agentID string, outputType string, data []byte, processID int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Log to all sessions (in a real implementation, you'd filter by agent ID)
	for _, session := range m.sessions {
		session.LogProcessOutput(outputType, data, processID)
	}
}

// GetTerminalLogs returns the log buffer for a specific terminal session
func (m *TerminalManager) GetTerminalLogs(sessionID string) ([]LogMessage, error) {
	m.mutex.RLock()
	session, ok := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", sessionID)
	}

	return session.GetLogBuffer(), nil
}

// GetTerminalSessions returns all terminal sessions for a specific agent/sub-agent
// This method is used by agent templates for sub-agent terminal management
func (m *TerminalManager) GetTerminalSessions(agentID string) []*TerminalSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var sessions []*TerminalSession
	for _, session := range m.sessions {
		// For now, we'll return all sessions since we don't have agent-specific mapping
		// This can be enhanced later to filter by agent ID
		sessions = append(sessions, session)
	}

	return sessions
}
