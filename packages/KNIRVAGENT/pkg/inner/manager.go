package inner

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

func (m *InnerAgentManager) Resize(sessionID string, cols, rows uint16) error {
	sess := m.get(sessionID)
	if sess == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return pty.Setsize(sess.PTY, &pty.Winsize{Cols: cols, Rows: rows})
}

type InnerAgentManager struct {
	mu           sync.RWMutex
	sessions     map[string]*InnerAgentSession
	catalog      *ToolCatalog
	workspaceDir string
	dveID        string
	logDir       string
}

func NewInnerAgentManager(dveID, workspaceDir string) *InnerAgentManager {
	logDir := workspaceDir + "/.supervisor-logs"
	os.MkdirAll(logDir, 0755)
	return &InnerAgentManager{
		sessions:     make(map[string]*InnerAgentSession),
		catalog:      DefaultCatalog(),
		workspaceDir: workspaceDir,
		dveID:        dveID,
		logDir:       logDir,
	}
}

func (m *InnerAgentManager) Spawn(toolName string, extraEnv []string) (*InnerAgentSession, error) {
	tool, ok := m.catalog.Lookup(toolName)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
	if !tool.IsInstalled() {
		return nil, fmt.Errorf("tool %s not installed; call Install first", toolName)
	}

	sessionID := uuid.NewString()
	logFile, err := os.Create(fmt.Sprintf("%s/%s.log", m.logDir, sessionID))
	if err != nil {
		return nil, err
	}

	cmd := tool.BuildCmd(m.workspaceDir, extraEnv)
	DefaultSpawnHooks(cmd, m.dveID, sessionID)

	// Do NOT set Setpgid here. pty.Start internally sets Setsid=true, which
	// makes the child a session leader. A session leader cannot call setpgid,
	// so combining Setpgid+Setsid causes EPERM propagated as a fork/exec error.

	ptmx, err := pty.Start(cmd)
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("pty.Start: %w", err)
	}

	sess := &InnerAgentSession{
		ID:            sessionID,
		ToolName:      toolName,
		Cmd:           cmd,
		PTY:           ptmx,
		Status:        StatusRunning,
		StartedAt:     time.Now(),
		WorkspaceDir:  m.workspaceDir,
		DVEID:         m.dveID,
		supervisorLog: logFile,
		done:          make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sessionID] = sess
	m.mu.Unlock()

	go sess.PumpOutput()
	return sess, nil
}

func (m *InnerAgentManager) SendInput(sessionID string, data []byte) error {
	sess := m.get(sessionID)
	if sess == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}
	_, err := sess.PTY.Write(data)
	return err
}

func (m *InnerAgentManager) Install(toolName string) error {
	tool, ok := m.catalog.Lookup(toolName)
	if !ok {
		return fmt.Errorf("unknown tool: %s", toolName)
	}
	return tool.Install(m.workspaceDir)
}

func (m *InnerAgentManager) Terminate(sessionID string) error {
	sess := m.get(sessionID)
	if sess == nil {
		return nil
	}
	if sess.Cmd != nil && sess.Cmd.Process != nil {
		// Kill the entire process group to prevent orphans
		syscall.Kill(-sess.Cmd.Process.Pid, syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		syscall.Kill(-sess.Cmd.Process.Pid, syscall.SIGKILL)
	}
	sess.PTY.Close()
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	return nil
}

func (m *InnerAgentManager) GetSession(id string) (*InnerAgentSession, error) {
	sess := m.get(id)
	if sess == nil {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return sess, nil
}

func (m *InnerAgentManager) ListSessions() []*InnerAgentSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*InnerAgentSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

func (m *InnerAgentManager) ListTools() []Tool { return m.catalog.All() }

func (m *InnerAgentManager) get(id string) *InnerAgentSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}
