package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var loopbackFrontendURL = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1|\[::1\])(?::[0-9]{1,5})?(?:/[^\s"']*)?`)

// SandboxSessionStatus is the lifecycle state of a sandbox session.
type SandboxSessionStatus string

const (
	SandboxStatusIdle         SandboxSessionStatus = "idle"
	SandboxStatusProvisioning SandboxSessionStatus = "provisioning"
	SandboxStatusRunning      SandboxSessionStatus = "running"
	SandboxStatusStopping     SandboxSessionStatus = "stopping"
	SandboxStatusStopped      SandboxSessionStatus = "stopped"
	SandboxStatusError        SandboxSessionStatus = "error"
)

// SandboxBind describes a single bubblewrap bind mount.
type SandboxBind struct {
	Mode string `json:"mode"`
	Src  string `json:"src"`
	Dst  string `json:"dst"`
}

// SandboxLaunchConfig carries the parameters for launching a sandboxed target.
type SandboxLaunchConfig struct {
	TargetLabel   string        `json:"targetLabel"`
	TargetCommand string        `json:"targetCommand"`
	TargetArgs    []string      `json:"targetArgs"`
	Display       string        `json:"display"`
	Binds         []SandboxBind `json:"binds"`
	UnshareAll    bool          `json:"unshareAll"`
	ShareNet      bool          `json:"shareNet"`
	DieWithParent bool          `json:"dieWithParent"`
}

// SandboxSession represents a single active sandbox (one app-wide).
type SandboxSession struct {
	ID            string               `json:"id"`
	CreatedAt     time.Time            `json:"createdAt"`
	LastActivity  time.Time            `json:"lastActivity"`
	UserID        int64                `json:"userId"`
	TargetLabel   string               `json:"targetLabel"`
	TargetCommand string               `json:"targetCommand"`
	Status        SandboxSessionStatus `json:"status"`
	Error         string               `json:"error,omitempty"`
	Pid           int                  `json:"pid,omitempty"`
	Display       string               `json:"display"`
	NetnsID       string               `json:"netnsId"`
	VncPort       int                  `json:"vncPort"`
	VncWsPath     string               `json:"vncWsPath,omitempty"`
	StatusWsPath  string               `json:"statusWsPath"`
	FrontendURL   string               `json:"frontendUrl,omitempty"`

	// Private process/lifecycle state
	manager         *SandboxManager
	ctx             context.Context
	cancelFunc      context.CancelFunc
	mutex           sync.RWMutex
	clients         map[*websocket.Conn]bool
	xvfbCmd         *exec.Cmd
	bwrapCmd        *exec.Cmd
	vncCmd          *exec.Cmd
	frontendCmd     *exec.Cmd
	frontendURL     string
	frontendProfile string
	frontendOnce    sync.Once
	proxyListener   net.Listener
	proxyServer     *http.Server
	proxyPort       int
	proxyFlows      []SandboxProxyFlow
	proxyFlowID     int
	started         bool
	log             []string

	// Launch configuration snapshot used to build the bwrap argv.
	binds         []SandboxBind
	unshareAll    bool
	shareNet      bool
	dieWithParent bool
	targetArgs    []string
}

// SandboxProxyFlow is a captured HTTP request made by the noVNC browser.
type SandboxProxyFlow struct {
	ID          int    `json:"id"`
	Method      string `json:"method"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType,omitempty"`
	Size        int64  `json:"size"`
	DurationMs  int64  `json:"durationMs"`
	Error       string `json:"error,omitempty"`
}

// SandboxManager manages the single active sandbox session.
type SandboxManager struct {
	sessions      map[string]*SandboxSession
	mutex         sync.RWMutex
	maxSessions   int
	sessionExpiry time.Duration

	// Binary names are overridable for tests (defaults: Xvfb, bwrap, x11vnc).
	xvfbBin   string
	bwrapBin  string
	x11vncBin string
	vncPort   int
	// screen geometry for the in-sandbox Xvfb display
	screen string

	// Cached result of EnsureSandboxDependencies so the install runs at most
	// once per process.
	depsOnce   sync.Once
	depsMutex  sync.Mutex
	depsStatus []DependencyStatus
	depsErr    error
}

// NewSandboxManager creates a new sandbox manager.
func NewSandboxManager() *SandboxManager {
	return &SandboxManager{
		sessions:      make(map[string]*SandboxSession),
		maxSessions:   1,
		sessionExpiry: 12 * time.Hour,
		xvfbBin:       "Xvfb",
		bwrapBin:      "bwrap",
		x11vncBin:     "x11vnc",
		vncPort:       5999,
		screen:        "1280x800x24",
	}
}

// resolveBins points the manager at the helper binaries, preferring bundled
// copies in tools/ over whatever is on PATH. Explicit overrides (e.g. the
// "sleep" bins used by tests) are preserved because we only re-resolve the
// default bare-name binaries.
func (m *SandboxManager) resolveBins() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.xvfbBin == "" || m.xvfbBin == "Xvfb" {
		m.xvfbBin = resolveSandboxTool("Xvfb")
	}
	if m.bwrapBin == "" || m.bwrapBin == "bwrap" {
		m.bwrapBin = resolveSandboxTool("bwrap")
	}
	if m.x11vncBin == "" || m.x11vncBin == "x11vnc" {
		m.x11vncBin = resolveSandboxTool("x11vnc")
	}
}

// CreateSession creates a new sandbox session (single-session scope).
func (m *SandboxManager) CreateSession(userID int64, cfg SandboxLaunchConfig) (*SandboxSession, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("a sandbox session is already running (single-session mode)")
	}

	if cfg.TargetCommand == "" {
		return nil, fmt.Errorf("targetCommand is required to launch a sandbox")
	}

	display := cfg.Display
	if display == "" {
		display = ":99"
	}

	sessionID := uuid.New().String()
	ctx, cancelFunc := context.WithCancel(context.Background())

	session := &SandboxSession{
		ID:            sessionID,
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
		UserID:        userID,
		TargetLabel:   cfg.TargetLabel,
		TargetCommand: cfg.TargetCommand,
		Status:        SandboxStatusIdle,
		Display:       display,
		NetnsID:       sessionID,
		VncPort:       m.vncPort,
		VncWsPath:     fmt.Sprintf("/api/v1/sandboxes/%s/vnc", sessionID),
		StatusWsPath:  fmt.Sprintf("/api/v1/sandboxes/%s/ws", sessionID),
		manager:       m,
		ctx:           ctx,
		cancelFunc:    cancelFunc,
		clients:       make(map[*websocket.Conn]bool),
		binds:         cfg.Binds,
		unshareAll:    cfg.UnshareAll,
		shareNet:      cfg.ShareNet,
		dieWithParent: cfg.DieWithParent,
	}
	if cfg.TargetArgs != nil {
		session.targetArgs = make([]string, len(cfg.TargetArgs))
		copy(session.targetArgs, cfg.TargetArgs)
	}

	m.sessions[sessionID] = session
	return session, nil
}

// GetSession retrieves a sandbox session by ID.
func (m *SandboxManager) GetSession(sessionID string) (*SandboxSession, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("sandbox session not found: %s", sessionID)
	}
	return session, nil
}

// ListSessions lists all sandbox sessions for a user.
func (m *SandboxManager) ListSessions(userID int64) []*SandboxSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var sessions []*SandboxSession
	for _, session := range m.sessions {
		if userID == 0 || session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// CloseSession stops and removes a sandbox session.
func (m *SandboxManager) CloseSession(sessionID string) error {
	m.mutex.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.mutex.Unlock()
		return fmt.Errorf("sandbox session not found: %s", sessionID)
	}
	delete(m.sessions, sessionID)
	m.mutex.Unlock()

	if err := session.Close(); err != nil {
		return fmt.Errorf("failed to close sandbox session: %v", err)
	}
	return nil
}

// CloseAll stops every managed sandbox during engine shutdown. Sessions are
// removed before process termination so concurrent HTTP requests cannot reuse
// a target that is in the process of being torn down.
func (m *SandboxManager) CloseAll() {
	m.mutex.Lock()
	sessions := make([]*SandboxSession, 0, len(m.sessions))
	for id, session := range m.sessions {
		delete(m.sessions, id)
		sessions = append(sessions, session)
	}
	m.mutex.Unlock()
	for _, session := range sessions {
		if err := session.Close(); err != nil {
			log.Printf("failed to close sandbox %s during shutdown: %v", session.ID, err)
		}
	}
}

// CleanupExpiredSessions removes expired sessions.
func (m *SandboxManager) CleanupExpiredSessions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		session.mutex.RLock()
		expired := now.Sub(session.LastActivity) > m.sessionExpiry
		session.mutex.RUnlock()
		if expired {
			log.Printf("Closing expired sandbox session: %s", id)
			session.Close()
			delete(m.sessions, id)
		}
	}
}

// Start provisions the sandbox: launches Xvfb, the bwrap target, and x11vnc.
func (s *SandboxSession) Start() error {
	s.mutex.Lock()
	if s.started {
		s.mutex.Unlock()
		return fmt.Errorf("sandbox session already started")
	}
	s.started = true
	s.Status = SandboxStatusProvisioning
	s.LastActivity = time.Now()
	s.mutex.Unlock()
	s.broadcastStatus()

	// Resolve the helper binaries, preferring a copy bundled in the engine's
	// tools/ directory over whatever is on PATH. Preserve any explicit
	// override (e.g. the "sleep" test bins) by only resolving when the
	// manager still holds the default bare-name binary.
	s.manager.resolveBins()

	// Automatically verify (and, if needed, install) the required system
	// dependencies so the end user never has to provision bubblewrap/Xvfb/
	// x11vnc by hand.
	depStatuses, depErr := s.manager.GetDependencyStatus()
	if depErr != nil {
		msg := fmt.Sprintf("sandbox dependencies unavailable: %v", depErr)
		s.setStatus(SandboxStatusError, msg)
		s.manager.removeSession(s.ID)
		return fmt.Errorf("%s", msg)
	}
	var missing []string
	for _, st := range depStatuses {
		if !st.Present {
			missing = append(missing, fmt.Sprintf("%s (install: %s)", st.Binary, st.InstallCommand))
		}
	}
	if len(missing) > 0 {
		msg := "missing sandbox dependencies: " + strings.Join(missing, "; ")
		s.setStatus(SandboxStatusError, msg)
		s.manager.removeSession(s.ID)
		return fmt.Errorf("%s", msg)
	}

	// 1. Xvfb display
	xvfbCmd, err := s.spawn(s.manager.xvfbBin, s.Display, "-screen", "0", s.manager.screen)
	if err != nil {
		return s.failStart(fmt.Sprintf("failed to start Xvfb: %v", err))
	}
	s.mutex.Lock()
	s.xvfbCmd = xvfbCmd
	s.mutex.Unlock()

	// 2. bwrap target (the actual sandboxed command)
	bwrapArgs := buildBwrapArgs(s)
	bwrapCmd, err := s.spawn(s.manager.bwrapBin, bwrapArgs...)
	if err != nil {
		return s.failStart(fmt.Sprintf("failed to start bwrap target: %v", err))
	}
	s.mutex.Lock()
	s.bwrapCmd = bwrapCmd
	s.Pid = bwrapCmd.Process.Pid
	s.mutex.Unlock()

	// 3. x11vnc bridging the framebuffer (shared so dock + standalone can both connect)
	vncCmd, err := s.spawn(s.manager.x11vncBin,
		"-display", s.Display,
		"-rfbport", strconv.Itoa(s.VncPort),
		"-forever", "-shared", "-nopw",
	)
	if err != nil {
		return s.failStart(fmt.Sprintf("failed to start x11vnc: %v", err))
	}
	s.mutex.Lock()
	s.vncCmd = vncCmd
	s.mutex.Unlock()

	s.setStatus(SandboxStatusRunning, "")
	return nil
}

// failStart tears down whatever started and records the error.
func (s *SandboxSession) failStart(msg string) error {
	s.Close()
	s.setStatus(SandboxStatusError, msg)
	s.manager.removeSession(s.ID)
	return fmt.Errorf("%s", msg)
}

// removeSession deletes a session from the manager without re-invoking Close.
func (m *SandboxManager) removeSession(sessionID string) {
	m.mutex.Lock()
	delete(m.sessions, sessionID)
	m.mutex.Unlock()
}

// buildBwrapArgs assembles the bubblewrap argument vector from the session.
func buildBwrapArgs(s *SandboxSession) []string {
	args := []string{}
	for _, b := range s.binds {
		if b.Mode == "tmpfs" {
			args = append(args, "--tmpfs", b.Dst)
		} else {
			args = append(args, "--"+b.Mode, b.Src, b.Dst)
		}
	}
	args = append(args, "--proc", "/proc", "--dev", "/dev")
	if s.unshareAll {
		args = append(args, "--unshare-all")
	}
	if s.shareNet {
		args = append(args, "--share-net")
	}
	if s.dieWithParent {
		args = append(args, "--die-with-parent")
	}
	args = append(args, "--setenv", "DISPLAY", s.Display, "--")
	// Explicit arguments preserve spaces in script paths and avoid invoking a
	// shell. Keep strings.Fields for clients created before targetArgs existed.
	if s.targetArgs != nil {
		args = append(args, s.TargetCommand)
		args = append(args, s.targetArgs...)
	} else {
		args = append(args, strings.Fields(s.TargetCommand)...)
	}
	return args
}

// toolEnv builds the subprocess environment for a managed tool, appending
// LD_LIBRARY_PATH to the bundled lib/ directory when the tool is a bundled
// binary so it finds its shipped shared-library dependencies.
func (s *SandboxSession) toolEnv(name string) []string {
	env := os.Environ()
	if isBundledTool(name) {
		if libDir := bundledToolsLibDir(); libDir != "" {
			env = append(env, "LD_LIBRARY_PATH="+libDir)
		}
	}
	return env
}

// spawn starts a managed subprocess, attaches its output to the log stream,
// and watches it for premature exit.
func (s *SandboxSession) spawn(name string, args ...string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(s.ctx, name, args...)
	cmd.Env = s.toolEnv(name)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go s.handleOutput(stdout, name, false)
	go s.handleOutput(stderr, name, true)
	go s.watchProcess(cmd, name)

	return cmd, nil
}

// watchProcess flips the session to error if a managed process dies
// unexpectedly (e.g. a typo'd target command that exits immediately).
func (s *SandboxSession) watchProcess(cmd *exec.Cmd, label string) {
	err := cmd.Wait()
	s.mutex.Lock()
	status := s.Status
	stopping := s.Status == SandboxStatusStopping || s.Status == SandboxStatusStopped
	s.mutex.Unlock()

	if err != nil && !stopping && status != SandboxStatusError {
		s.setStatus(SandboxStatusError, fmt.Sprintf("%s process exited: %v", label, err))
	}
}

// Close terminates the sandbox subprocesses in reverse dependency order.
func (s *SandboxSession) Close() error {
	s.mutex.Lock()
	if s.Status == SandboxStatusStopped {
		s.mutex.Unlock()
		return nil
	}
	if s.Status != SandboxStatusStopping {
		s.Status = SandboxStatusStopping
	}
	s.LastActivity = time.Now()
	s.mutex.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	s.mutex.Lock()
	vncPid := 0
	bwrapPid := s.Pid
	xvfbPid := 0
	frontendPid := 0
	frontendProfile := s.frontendProfile
	proxyServer := s.proxyServer
	if s.vncCmd != nil && s.vncCmd.Process != nil {
		vncPid = s.vncCmd.Process.Pid
	}
	if s.xvfbCmd != nil && s.xvfbCmd.Process != nil {
		xvfbPid = s.xvfbCmd.Process.Pid
	}
	if s.frontendCmd != nil && s.frontendCmd.Process != nil {
		frontendPid = s.frontendCmd.Process.Pid
	}
	s.mutex.Unlock()

	// Reverse dependency order: browser -> x11vnc -> bwrap -> Xvfb
	s.killProcess("frontend browser", frontendPid)
	if proxyServer != nil {
		_ = proxyServer.Close()
	}
	s.killProcess("x11vnc", vncPid)
	s.killProcess("bwrap", bwrapPid)
	s.killProcess("Xvfb", xvfbPid)

	s.mutex.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)
	s.Status = SandboxStatusStopped
	s.LastActivity = time.Now()
	s.mutex.Unlock()
	if frontendProfile != "" {
		_ = os.RemoveAll(frontendProfile)
	}

	return nil
}

func (s *SandboxSession) killProcess(label string, pid int) {
	if pid == 0 {
		return
	}
	if !isProcessAlive(pid) {
		return
	}
	if err := terminateProcess(pid, true); err != nil {
		log.Printf("sandbox %s: failed to terminate %s (pid %d): %v", s.ID, label, pid, err)
	}
}

// AddClient registers a WebSocket client and pushes current session info.
func (s *SandboxSession) AddClient(conn *websocket.Conn) {
	s.mutex.Lock()
	s.clients[conn] = true
	s.LastActivity = time.Now()
	flows := append([]SandboxProxyFlow(nil), s.proxyFlows...)
	frontendURL, proxyPort := s.frontendURL, s.proxyPort
	s.mutex.Unlock()

	conn.WriteJSON(map[string]interface{}{
		"type":        "session_info",
		"id":          s.ID,
		"status":      s.Status,
		"label":       s.TargetLabel,
		"display":     s.Display,
		"vncPort":     s.VncPort,
		"error":       s.Error,
		"frontendUrl": frontendURL,
		"proxyPort":   proxyPort,
	})
	for _, flow := range flows {
		_ = conn.WriteJSON(map[string]interface{}{"type": "proxy_flow", "flow": flow})
	}
}

// RemoveClient deregisters a WebSocket client.
func (s *SandboxSession) RemoveClient(conn *websocket.Conn) {
	s.mutex.Lock()
	delete(s.clients, conn)
	s.mutex.Unlock()
}

// handleOutput reads a process pipe and fans log lines out to clients.
func (s *SandboxSession) handleOutput(pipe io.ReadCloser, stream string, isStderr bool) {
	buffer := make([]byte, 4096)
	for {
		n, err := pipe.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("sandbox %s: error reading %s: %v", s.ID, stream, err)
			}
			break
		}
		if n > 0 {
			line := string(buffer[:n])
			if stream == s.manager.bwrapBin {
				if frontendURL := extractLoopbackFrontendURL(line); frontendURL != "" {
					s.frontendOnce.Do(func() { go s.openFrontend(frontendURL) })
				}
			}
			s.mutex.Lock()
			s.LastActivity = time.Now()
			s.log = append(s.log, line)
			s.mutex.Unlock()
			s.broadcastToClients(map[string]interface{}{
				"type":   "log",
				"stream": stream,
				"stderr": isStderr,
				"data":   line,
			})
		}
	}
}

// extractLoopbackFrontendURL accepts only a target's loopback HTTP endpoint.
// This keeps automatic browser launch scoped to a server the sandboxed target
// has explicitly announced, not arbitrary URLs that happen to appear in logs.
func extractLoopbackFrontendURL(output string) string {
	lower := strings.ToLower(output)
	// Dependency failures can mention unrelated local services (for example a
	// ChromaDB URL on :8000). Only accept a URL from an affirmative web-server
	// announcement, never from an error line.
	if strings.Contains(lower, "failed") || strings.Contains(lower, "error") ||
		!(strings.Contains(lower, "server started") || strings.Contains(lower, "server listening") ||
			strings.Contains(lower, "listening on") || strings.Contains(lower, "listen ") || strings.Contains(lower, "dashboard:") ||
			strings.Contains(lower, "opening dashboard")) {
		return ""
	}
	match := loopbackFrontendURL.FindString(output)
	if match == "" {
		return ""
	}
	match = strings.TrimRight(match, ".,;)}]")
	parsed, err := url.Parse(match)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return parsed.String()
	default:
		return ""
	}
}

// openFrontend starts a trusted host browser on Xvfb, so noVNC displays a
// frontend served by the sandboxed process. The browser is deliberately not
// put inside bwrap: it only visits the loopback URL the target announced and
// must share the host network when ShareNet is enabled.
func (s *SandboxSession) openFrontend(frontendURL string) {
	if !s.shareNet {
		s.appendLog("[sandbox] frontend detected at " + frontendURL + ", but automatic browser launch requires --share-net")
		return
	}
	browser, err := findSandboxBrowser()
	if err != nil {
		s.appendLog("[sandbox] frontend detected at " + frontendURL + ", but no supported browser is installed")
		return
	}
	proxyPort, err := s.startFrontendProxy(frontendURL)
	if err != nil {
		s.appendLog("[sandbox] could not start frontend proxy: " + err.Error())
		return
	}
	profile, err := os.MkdirTemp("", "knirv-sandbox-browser-")
	if err != nil {
		s.appendLog("[sandbox] frontend detected at " + frontendURL + ", but could not create a browser profile: " + err.Error())
		return
	}
	cmd := exec.CommandContext(s.ctx, browser,
		"--no-first-run", "--no-default-browser-check", "--user-data-dir="+profile,
		"--proxy-server=http://127.0.0.1:"+strconv.Itoa(proxyPort), "--proxy-bypass-list=<-loopback>", frontendURL)
	cmd.Env = append(os.Environ(), "DISPLAY="+s.Display)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(profile)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = os.RemoveAll(profile)
		return
	}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(profile)
		s.appendLog("[sandbox] could not open frontend " + frontendURL + ": " + err.Error())
		return
	}
	s.mutex.Lock()
	s.frontendCmd = cmd
	s.frontendURL = frontendURL
	s.frontendProfile = profile
	s.mutex.Unlock()
	s.broadcastToClients(map[string]interface{}{"type": "frontend", "url": frontendURL})
	s.appendLog("[sandbox] opening frontend in noVNC: " + frontendURL)
	go s.handleOutput(stdout, browser, false)
	go s.handleOutput(stderr, browser, true)
	go func() {
		if err := cmd.Wait(); err != nil && s.ctx.Err() == nil {
			s.appendLog("[sandbox] frontend browser exited: " + err.Error())
		}
	}()
}

func (s *SandboxSession) startFrontendProxy(frontendURL string) (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.proxyFrontendRequest(w, r, frontendURL)
	})}
	s.mutex.Lock()
	s.proxyListener, s.proxyServer = listener, server
	s.proxyPort = listener.Addr().(*net.TCPAddr).Port
	port := s.proxyPort
	s.mutex.Unlock()
	go func() { _ = server.Serve(listener) }()
	return port, nil
}

func (s *SandboxSession) proxyFrontendRequest(w http.ResponseWriter, r *http.Request, frontendURL string) {
	started := time.Now()
	target, _ := url.Parse(frontendURL)
	requestURL := r.URL
	if requestURL.Scheme == "" {
		requestURL = &url.URL{Scheme: target.Scheme, Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
	}
	flow := SandboxProxyFlow{Method: r.Method, Host: requestURL.Host, Path: requestURL.RequestURI()}
	if r.Method == http.MethodConnect {
		connectURL := &url.URL{Host: r.Host}
		flow.Host, flow.Path = connectURL.Host, "CONNECT"
		if connectURL.Hostname() != target.Hostname() || connectURL.Port() != target.Port() {
			flow.Error = "blocked: proxy is scoped to the sandbox frontend"
			http.Error(w, flow.Error, http.StatusForbidden)
			s.recordProxyFlow(flow, started)
			return
		}
		s.proxyConnect(w, r, flow, started)
		return
	}
	if requestURL.Hostname() != target.Hostname() || requestURL.Port() != target.Port() {
		flow.Error = "blocked: proxy is scoped to the sandbox frontend"
		http.Error(w, flow.Error, http.StatusForbidden)
		s.recordProxyFlow(flow, started)
		return
	}
	request := r.Clone(r.Context())
	request.URL = requestURL
	request.RequestURI = ""
	request.Host = requestURL.Host
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		flow.Error = err.Error()
		http.Error(w, err.Error(), http.StatusBadGateway)
		s.recordProxyFlow(flow, started)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	bytes, _ := io.Copy(w, response.Body)
	flow.Status, flow.ContentType, flow.Size = response.StatusCode, response.Header.Get("Content-Type"), bytes
	s.recordProxyFlow(flow, started)
}

// proxyConnect passes WebSocket/TLS CONNECT traffic through unchanged. It is
// restricted to the detected frontend host and port, so it cannot become a
// general-purpose host proxy. CONNECT payloads are intentionally not decoded.
func (s *SandboxSession) proxyConnect(w http.ResponseWriter, r *http.Request, flow SandboxProxyFlow, started time.Time) {
	upstream, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		flow.Error = err.Error()
		http.Error(w, err.Error(), http.StatusBadGateway)
		s.recordProxyFlow(flow, started)
		return
	}
	defer upstream.Close()
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		flow.Error = "proxy response does not support connection hijacking"
		http.Error(w, flow.Error, http.StatusInternalServerError)
		s.recordProxyFlow(flow, started)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		flow.Error = err.Error()
		s.recordProxyFlow(flow, started)
		return
	}
	defer client.Close()
	_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	flow.Status = http.StatusOK
	s.recordProxyFlow(flow, started)
	done := make(chan struct{})
	go func() { _, _ = io.Copy(upstream, client); close(done) }()
	_, _ = io.Copy(client, upstream)
	<-done
}

func (s *SandboxSession) recordProxyFlow(flow SandboxProxyFlow, started time.Time) {
	flow.DurationMs = time.Since(started).Milliseconds()
	s.mutex.Lock()
	s.proxyFlowID++
	flow.ID = s.proxyFlowID
	s.proxyFlows = append(s.proxyFlows, flow)
	if len(s.proxyFlows) > 200 {
		s.proxyFlows = s.proxyFlows[len(s.proxyFlows)-200:]
	}
	s.mutex.Unlock()
	s.broadcastToClients(map[string]interface{}{"type": "proxy_flow", "flow": flow})
}

func findSandboxBrowser() (string, error) {
	for _, candidate := range []string{"google-chrome", "chromium", "chromium-browser", "firefox"} {
		if browser, err := exec.LookPath(candidate); err == nil {
			return browser, nil
		}
	}
	return "", fmt.Errorf("no supported browser found")
}

func (s *SandboxSession) appendLog(line string) {
	s.mutex.Lock()
	s.LastActivity = time.Now()
	s.log = append(s.log, line)
	s.mutex.Unlock()
	s.broadcastToClients(map[string]interface{}{"type": "log", "stream": "sandbox", "stderr": false, "data": line})
}

// setStatus updates the session status and broadcasts the change.
func (s *SandboxSession) setStatus(status SandboxSessionStatus, errMsg string) {
	s.mutex.Lock()
	s.Status = status
	s.Error = errMsg
	s.LastActivity = time.Now()
	s.mutex.Unlock()
	s.broadcastStatus()
}

// broadcastStatus sends a status update to all connected clients.
func (s *SandboxSession) broadcastStatus() {
	s.mutex.Lock()
	status := s.Status
	errMsg := s.Error
	s.mutex.Unlock()
	s.broadcastToClients(map[string]interface{}{
		"type":   "status",
		"status": status,
		"error":  errMsg,
	})
}

// broadcastToClients sends a message to every connected client.
func (s *SandboxSession) broadcastToClients(message map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for client := range s.clients {
		if err := client.WriteJSON(message); err != nil {
			log.Printf("sandbox %s: error sending to client: %v", s.ID, err)
			client.Close()
			delete(s.clients, client)
		}
	}
}

// GetInfo returns public information about the sandbox session.
func (s *SandboxSession) GetInfo() SandboxSessionInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SandboxSessionInfo{
		ID:            s.ID,
		CreatedAt:     s.CreatedAt,
		LastActivity:  s.LastActivity,
		TargetLabel:   s.TargetLabel,
		TargetCommand: s.TargetCommand,
		Status:        s.Status,
		Error:         s.Error,
		Pid:           s.Pid,
		Display:       s.Display,
		NetnsID:       s.NetnsID,
		VncPort:       s.VncPort,
		VncWsPath:     s.VncWsPath,
		StatusWsPath:  s.StatusWsPath,
		FrontendURL:   s.frontendURL,
		ProxyPort:     s.proxyPort,
		ClientCount:   len(s.clients),
	}
}

// SandboxSessionInfo is the public wire representation of a session.
type SandboxSessionInfo struct {
	ID            string               `json:"id"`
	CreatedAt     time.Time            `json:"createdAt"`
	LastActivity  time.Time            `json:"lastActivity"`
	TargetLabel   string               `json:"targetLabel"`
	TargetCommand string               `json:"targetCommand"`
	Status        SandboxSessionStatus `json:"status"`
	Error         string               `json:"error,omitempty"`
	Pid           int                  `json:"pid"`
	Display       string               `json:"display"`
	NetnsID       string               `json:"netnsId"`
	VncPort       int                  `json:"vncPort"`
	VncWsPath     string               `json:"vncWsPath,omitempty"`
	StatusWsPath  string               `json:"statusWsPath,omitempty"`
	FrontendURL   string               `json:"frontendUrl,omitempty"`
	ProxyPort     int                  `json:"proxyPort,omitempty"`
	ClientCount   int                  `json:"clientCount"`
}

// HandleStatusWebSocket streams status + log lines to a single client.
func (m *SandboxManager) HandleStatusWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("sandbox: failed to upgrade status websocket: %v", err)
		return
	}
	defer conn.Close()

	session, err := m.GetSession(sessionID)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{"type": "error", "error": err.Error()})
		return
	}

	session.AddClient(conn)
	defer session.RemoveClient(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// HandleVNCWebSocket bridges a WebSocket connection to the session's local
// x11vnc TCP endpoint (Go-native, no websockify dependency).
func (m *SandboxManager) HandleVNCWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("sandbox: failed to upgrade vnc websocket: %v", err)
		return
	}
	defer conn.Close()

	session, err := m.GetSession(sessionID)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session not found"))
		return
	}

	session.mutex.RLock()
	vncPort := session.VncPort
	session.mutex.RUnlock()

	vncAddr := fmt.Sprintf("127.0.0.1:%d", vncPort)
	vnc, err := net.Dial("tcp", vncAddr)
	if err != nil {
		log.Printf("sandbox %s: cannot dial x11vnc at %s: %v", sessionID, vncAddr, err)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "vnc backend unavailable"))
		return
	}
	defer vnc.Close()

	// ws -> tcp
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				vnc.Close()
				return
			}
			if _, err := vnc.Write(data); err != nil {
				return
			}
		}
	}()

	// tcp -> ws
	buffer := make([]byte, 4096)
	for {
		n, err := vnc.Read(buffer)
		if err != nil {
			break
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
			break
		}
	}
}

// RegisterHandlers registers the sandbox API + WebSocket handlers.
func (m *SandboxManager) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/sandboxes/deps", m.handleGetDeps).Methods("GET")
	router.HandleFunc("/api/v1/sandboxes/deps/install", m.handleInstallDeps).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes", m.handleList).Methods("GET")
	router.HandleFunc("/api/v1/sandboxes", m.handleCreate).Methods("POST")
	router.HandleFunc("/api/v1/sandboxes/{id}", m.handleGet).Methods("GET")
	router.HandleFunc("/api/v1/sandboxes/{id}", m.handleStop).Methods("DELETE")
	router.HandleFunc("/api/v1/sandboxes/{id}/ws", m.handleStatusWebSocket)
	router.HandleFunc("/api/v1/sandboxes/{id}/vnc", m.handleVNCWebSocket)
}

func (m *SandboxManager) handleList(w http.ResponseWriter, r *http.Request) {
	sessions := m.ListSessions(0)
	infos := make([]SandboxSessionInfo, len(sessions))
	for i, s := range sessions {
		infos[i] = s.GetInfo()
	}
	RespondWithSuccess(w, map[string]interface{}{"sandboxes": infos}, MessageListRetrieved)
}

func (m *SandboxManager) handleCreate(w http.ResponseWriter, r *http.Request) {
	var cfg SandboxLaunchConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondWithValidationError(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	session, err := m.CreateSession(1, cfg)
	if err != nil {
		RespondWithConflict(w, err.Error())
		return
	}

	if err := session.Start(); err != nil {
		RespondWithInternalError(w, fmt.Sprintf("Failed to start sandbox: %v", err))
		return
	}

	RespondWithCreated(w, session.GetInfo(), "Sandbox session created successfully")
}

func (m *SandboxManager) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, err := m.GetSession(vars["id"])
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}
	RespondWithSuccess(w, session.GetInfo(), MessageRetrieved)
}

func (m *SandboxManager) handleStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := m.CloseSession(vars["id"]); err != nil {
		if strings.Contains(err.Error(), "not found") {
			RespondWithNotFound(w, "Sandbox session")
			return
		}
		RespondWithInternalError(w, fmt.Sprintf("Failed to stop sandbox: %v", err))
		return
	}
	RespondWithNoContent(w, "Sandbox session stopped successfully")
}

func (m *SandboxManager) handleStatusWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	m.HandleStatusWebSocket(w, r, vars["id"])
}

func (m *SandboxManager) handleVNCWebSocket(w http.ResponseWriter, cmd *http.Request) {
	vars := mux.Vars(cmd)
	m.HandleVNCWebSocket(w, cmd, vars["id"])
}
