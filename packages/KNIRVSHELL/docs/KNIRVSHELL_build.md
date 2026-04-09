## Specification: KNIRVSHELL (Layer 2)
**Base Layer:** `KNIRVCLI`  
**Framework:** Golang 1.26+, Cobra, Bubble Tea (Charm.sh)  
**Primary Output:** `.md` (Obsidian-optimized)  
**Economy Integration:** NRN Staging via KNIRVARENA  
**Module path:** `github.com/knirv/knirvshell`

---

## Project Structure

```
packages/KNIRVSHELL/
├── go.mod
├── go.sum
├── main.go                          # Cobra root, wires subcommands
├── cmd/
│   ├── root.go                      # Root command + persistent flags
│   ├── shell.go                     # `knirv shell` — Bubble Tea TUI
│   ├── register.go                  # `knirv register` — add agent to registry.db
│   ├── run.go                       # `knirv run [agent]` — supervisory launch
│   ├── stage.go                     # `knirv stage [file]` — KNIRVGRAPH submission
│   └── monitor.go                   # `knirv monitor` — list active pids/containers
├── internal/
│   ├── runner/
│   │   ├── runner.go                # Runner interface + AgentProcess type
│   │   ├── local.go                 # LocalRunner (os/exec)
│   │   ├── docker.go                # DockerRunner (docker/go-sdk)
│   │   └── ssh.go                   # SSHRunner (golang.org/x/crypto/ssh)
│   ├── watchtower/
│   │   ├── watchtower.go            # Supervisor goroutine + Incident dispatch
│   │   ├── signatures.go            # Regex error patterns (Python, Go, Node, AI APIs)
│   │   └── ringbuffer.go            # Fixed-size circular log buffer
│   ├── incident/
│   │   ├── incident.go              # Incident struct + .md template renderer
│   │   └── templates/
│   │       └── incident.md.tmpl     # Go text/template for INC-*.md files
│   ├── registry/
│   │   ├── registry.go              # SQLite-backed agent registry (registry.db)
│   │   └── schema.go                # DB schema + migration
│   ├── staging/
│   │   └── staging.go               # HTTP client for KNIRVGRAPH + KNIRVARENA
│   ├── config/
│   │   └── config.go                # Viper-based config (agents.yaml)
│   └── tui/
│       ├── model.go                 # Bubble Tea root Model
│       ├── agentlist.go             # Agent list panel component
│       ├── logviewport.go           # Scrollable log viewport
│       └── errorpanel.go            # Real-time error highlight panel
├── config/
│   └── agents.yaml                  # Default agent registry config
└── registry.db                      # SQLite agent store (gitignored)
```

---

## 1. Core Architecture: The "Parent-Observer" Model
KNIRVSHELL acts as the persistent execution environment. It does not just run commands; it **wraps** them in a supervisory loop.

* **Supervisory Wrapper:** Uses `os/exec` (local) or `docker/go-sdk` to spawn agents. It captures the `stdin`, `stdout`, and `stderr` as structured streams.
* **The Watchtower (Concurrency):** A dedicated goroutine for every active agent that performs regex-based "hot-scanning" on the log stream.
* **TUI Dashboard:** A Bubble Tea-powered interface that visualizes the "health" of the agentic tree and provides a viewport into real-time logs.

### Core Types

```go
// internal/runner/runner.go

package runner

import (
	"context"
	"io"
	"time"
)

// AgentType discriminates the execution backend.
type AgentType string

const (
	AgentTypeLocal  AgentType = "local"
	AgentTypeDocker AgentType = "docker"
	AgentTypeSSH    AgentType = "ssh"
)

// AgentProcess is the live handle to a running agent.
// It is created by a Runner and consumed by the Watchtower.
type AgentProcess struct {
	ID        string    // Unique run ID (UUID v4)
	AgentName string    // Matches registry entry
	Type      AgentType
	StartedAt time.Time

	Stdout io.Reader // Combined log stream for Watchtower consumption
	Stderr io.Reader // Separate stderr for error-priority scanning

	// StopFn sends SIGTERM (or Docker stop / SSH session close).
	// Implementations must make this idempotent.
	StopFn func(ctx context.Context) error

	// WaitFn blocks until the agent exits and returns the exit code.
	WaitFn func() (exitCode int, err error)
}

// Runner abstracts the execution environment.
// Implementations: LocalRunner, DockerRunner, SSHRunner.
type Runner interface {
	// Start launches the agent and returns a live AgentProcess.
	// The caller (Watchtower) owns the lifecycle from this point.
	Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error)
}

// AgentConfig is the per-agent launch descriptor, populated from registry.db
// or agents.yaml at runtime.
type AgentConfig struct {
	Name    string    `yaml:"name"`
	Type    AgentType `yaml:"type"`
	// Local
	Path    string   `yaml:"path,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Env     []string `yaml:"env,omitempty"`
	// Docker
	Image   string            `yaml:"image,omitempty"`
	Mounts  map[string]string `yaml:"mounts,omitempty"` // host:container
	// SSH
	Host       string `yaml:"host,omitempty"`
	User       string `yaml:"user,omitempty"`
	KeyPath    string `yaml:"key_path,omitempty"`
	RemoteCmd  string `yaml:"remote_cmd,omitempty"`
}
```

---

## The "Runner" Architecture
The key to supporting both local and Docker/Remote environments is to abstract the execution of the agent. You define a `Runner` interface in Go that the KNIRVSHELL core interacts with, regardless of where the agent actually lives.

### 1. Local Runner (`os/exec`)

```go
// internal/runner/local.go

package runner

import (
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/google/uuid"
	"time"
)

type LocalRunner struct{}

func (r *LocalRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	cmd := exec.CommandContext(ctx, cfg.Path, cfg.Args...)
	cmd.Env = append(cmd.Environ(), cfg.Env...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	proc := &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeLocal,
		StartedAt: time.Now(),
		Stdout:    stdoutPipe,
		Stderr:    stderrPipe,
		StopFn: func(ctx context.Context) error {
			if cmd.Process == nil {
				return nil
			}
			return cmd.Process.Signal(syscall.SIGTERM)
		},
		WaitFn: func() (int, error) {
			err := cmd.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return exitErr.ExitCode(), nil
				}
				return -1, err
			}
			return 0, nil
		},
	}
	return proc, nil
}
```

**Key detail — `io.TeeReader` fan-out:** The Watchtower uses `io.TeeReader` to simultaneously feed the raw stream to the TUI viewport *and* the error scanner without blocking either consumer:

```go
// internal/watchtower/watchtower.go (excerpt)

func (w *Watchtower) fanOut(proc *AgentProcess) {
	var buf bytes.Buffer

	// TeeReader: every byte read by the scanner also goes into buf
	tee := io.TeeReader(proc.Stdout, &buf)

	scanner := bufio.NewScanner(tee)
	for scanner.Scan() {
		line := scanner.Text()
		w.ring.Push(line)               // ring buffer for forensics
		w.uiCh <- UILogLine{            // non-blocking send to TUI
			AgentID: proc.ID,
			Line:    line,
		}
		w.scanLine(proc, line)          // hot-scan for error signatures
	}
}
```

### 2. Docker Runner (`docker/go-sdk`)

```go
// internal/runner/docker.go

package runner

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRunner{cli: cli}, nil
}

func (r *DockerRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	// Build mount binds: "host/path:/container/path"
	var binds []string
	for host, cont := range cfg.Mounts {
		binds = append(binds, host+":"+cont)
	}

	resp, err := r.cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.Image,
			Env:   cfg.Env,
		},
		&container.HostConfig{Binds: binds},
		nil, nil, "",
	)
	if err != nil {
		return nil, err
	}

	if err := r.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	// Attach multiplexed stdout/stderr stream
	attach, err := r.cli.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		return nil, err
	}

	// docker SDK multiplexes stdout/stderr into one stream with 8-byte headers
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	go func() {
		defer stdoutW.Close()
		defer stderrW.Close()
		// stdcopy.StdCopy demuxes the Docker multiplexed stream
		stdcopy.StdCopy(stdoutW, stderrW, attach.Reader)
	}()

	containerID := resp.ID

	return &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeDocker,
		StartedAt: time.Now(),
		Stdout:    stdoutR,
		Stderr:    stderrR,
		StopFn: func(ctx context.Context) error {
			timeout := 10 // seconds
			return r.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		},
		WaitFn: func() (int, error) {
			statusCh, errCh := r.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				return -1, err
			case status := <-statusCh:
				return int(status.StatusCode), nil
			}
		},
	}, nil
}
```

### 3. Remote/SSH Runner (`golang.org/x/crypto/ssh`)

```go
// internal/runner/ssh.go

package runner

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type SSHRunner struct{}

func (r *SSHRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	key, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: known_hosts in prod
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", cfg.Host+":22", sshCfg)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	stdoutPipe, _ := session.StdoutPipe()
	stderrPipe, _ := session.StderrPipe()

	if err := session.Start(cfg.RemoteCmd); err != nil {
		return nil, err
	}

	return &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeSSH,
		StartedAt: time.Now(),
		Stdout:    stdoutPipe,
		Stderr:    stderrPipe,
		StopFn: func(ctx context.Context) error {
			return session.Signal(ssh.SIGTERM)
		},
		WaitFn: func() (int, error) {
			err := session.Wait()
			client.Close()
			if err != nil {
				if exitErr, ok := err.(*ssh.ExitError); ok {
					return exitErr.ExitStatus(), nil
				}
				return -1, err
			}
			return 0, nil
		},
	}, nil
}
```

---

## The Watchtower

The Watchtower is the supervisory engine. One goroutine per active `AgentProcess`. It owns the ring buffer, fires error signatures, and publishes `Incident` objects.

```go
// internal/watchtower/watchtower.go

package watchtower

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/knirv/knirvshell/internal/incident"
	"github.com/knirv/knirvshell/internal/runner"
)

// UILogLine is sent to the Bubble Tea update channel.
type UILogLine struct {
	AgentID string
	Line    string
	IsError bool // true when a signature matched
}

type Watchtower struct {
	mu       sync.Mutex
	agents   map[string]*watchedAgent // keyed by AgentProcess.ID
	uiCh     chan<- UILogLine
	incidentCh chan<- *incident.Incident
	sigs     *SignatureSet
}

type watchedAgent struct {
	proc   *runner.AgentProcess
	ring   *RingBuffer
	cancel context.CancelFunc
}

func New(uiCh chan<- UILogLine, incCh chan<- *incident.Incident) *Watchtower {
	return &Watchtower{
		agents:     make(map[string]*watchedAgent),
		uiCh:       uiCh,
		incidentCh: incCh,
		sigs:       DefaultSignatures(),
	}
}

// Watch starts a supervisory goroutine for the given process.
func (w *Watchtower) Watch(ctx context.Context, proc *runner.AgentProcess) {
	ctx, cancel := context.WithCancel(ctx)
	wa := &watchedAgent{proc: proc, ring: NewRingBuffer(500), cancel: cancel}

	w.mu.Lock()
	w.agents[proc.ID] = wa
	w.mu.Unlock()

	go w.supervise(ctx, wa)
}

func (w *Watchtower) supervise(ctx context.Context, wa *watchedAgent) {
	defer func() {
		w.mu.Lock()
		delete(w.agents, wa.proc.ID)
		w.mu.Unlock()
	}()

	// Fan-out stdout via TeeReader
	var forensicBuf bytes.Buffer
	tee := io.TeeReader(wa.proc.Stdout, &forensicBuf)
	scanner := bufio.NewScanner(tee)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		wa.ring.Push(line)

		isErr, sigName := w.sigs.Match(line)
		select {
		case w.uiCh <- UILogLine{AgentID: wa.proc.ID, Line: line, IsError: isErr}:
		default: // drop if TUI is blocked — never block the supervisor
		}

		if isErr {
			go w.handleIncident(wa, line, sigName)
		}
	}

	// Agent exited — wait for exit code
	exitCode, _ := wa.proc.WaitFn()
	if exitCode != 0 {
		go w.handleIncident(wa, "non-zero exit: process terminated", "EXIT_NON_ZERO")
	}
}

func (w *Watchtower) handleIncident(wa *watchedAgent, triggerLine, sigName string) {
	inc := incident.New(wa.proc, wa.ring.Snapshot(), triggerLine, sigName)
	path, err := inc.WriteToVault()
	if err == nil {
		incident.OpenInObsidian(path)
	}
	select {
	case w.incidentCh <- inc:
	default:
	}
}
```

---

## Ring Buffer

```go
// internal/watchtower/ringbuffer.go

package watchtower

import "sync"

// RingBuffer is a fixed-capacity circular log store.
// Oldest entries are overwritten when capacity is exceeded.
type RingBuffer struct {
	mu   sync.RWMutex
	buf  []string
	cap  int
	head int // index of the next write slot
	full bool
}

func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{buf: make([]string, capacity), cap: capacity}
}

func (r *RingBuffer) Push(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head] = line
	r.head = (r.head + 1) % r.cap
	if r.head == 0 {
		r.full = true
	}
}

// Snapshot returns entries in chronological order (oldest first).
func (r *RingBuffer) Snapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.full {
		out := make([]string, r.head)
		copy(out, r.buf[:r.head])
		return out
	}
	out := make([]string, r.cap)
	copy(out, r.buf[r.head:])
	copy(out[r.cap-r.head:], r.buf[:r.head])
	return out
}

// Last returns the most recent n entries.
func (r *RingBuffer) Last(n int) []string {
	snap := r.Snapshot()
	if len(snap) <= n {
		return snap
	}
	return snap[len(snap)-n:]
}
```

---

## Error Signature Library

```go
// internal/watchtower/signatures.go

package watchtower

import "regexp"

type Signature struct {
	Name    string
	Pattern *regexp.Regexp
}

type SignatureSet struct {
	sigs []Signature
}

// Match returns (true, signatureName) on the first pattern match.
func (s *SignatureSet) Match(line string) (bool, string) {
	for _, sig := range s.sigs {
		if sig.Pattern.MatchString(line) {
			return true, sig.Name
		}
	}
	return false, ""
}

// DefaultSignatures covers Python, Go, Node.js, and major AI API errors.
func DefaultSignatures() *SignatureSet {
	patterns := []struct {
		name    string
		pattern string
	}{
		// Python
		{"PY_TRACEBACK", `Traceback \(most recent call last\)`},
		{"PY_EXCEPTION", `^[A-Za-z]+Error: .+`},
		{"PY_UNHANDLED", `unhandled exception in thread`},
		// Go
		{"GO_PANIC", `^goroutine \d+ \[`},
		{"GO_FATAL", `fatal error:`},
		{"GO_SEGFAULT", `SIGSEGV`},
		// Node.js
		{"NODE_UNHANDLED", `UnhandledPromiseRejectionWarning`},
		{"NODE_EXCEPTION", `^[A-Za-z]+Error: .+\n\s+at `},
		{"NODE_CRASH", `node: internal/process`},
		// OpenAI / Anthropic API
		{"API_RATE_LIMIT", `rate.limit|RateLimitError|429`},
		{"API_CONTEXT", `context.length.exceeded|max_tokens|context window`},
		{"API_AUTH", `AuthenticationError|401 Unauthorized`},
		{"API_OVERLOAD", `overloaded_error|529|ServiceUnavailable`},
		// Agentic / LLM loop patterns
		{"AGENT_LOOP", `infinite loop detected|reasoning loop|stuck in loop`},
		{"AGENT_TOOL_FAIL", `Tool execution failed|tool_use.*error`},
		{"AGENT_HALLUCINATION", `hallucination detected|confidence below threshold`},
		// Generic
		{"OOM", `out of memory|OOMKilled|cannot allocate`},
		{"TIMEOUT", `context deadline exceeded|operation timed out`},
		{"DISK_FULL", `no space left on device`},
	}

	set := &SignatureSet{}
	for _, p := range patterns {
		set.sigs = append(set.sigs, Signature{
			Name:    p.name,
			Pattern: regexp.MustCompile(`(?i)` + p.pattern),
		})
	}
	return set
}
```

---

## Real-Time Monitoring with "Ultraviolet"

Bubble Tea + Ultraviolet (Charm's 2026 cell-diffing renderer) powers the TUI. Three panels:
1. **Agent List** — name, type, status, uptime, error count
2. **Log Viewport** — scrollable, color-coded by stream (stdout=white, stderr=yellow, error=red)
3. **Error Monitor** — live feed of matched incidents, deduped by signature

```go
// internal/tui/model.go

package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knirv/knirvshell/internal/watchtower"
)

// UILogMsg is the Bubble Tea message wrapping a log line from Watchtower.
type UILogMsg watchtower.UILogLine

// IncidentMsg signals a new incident file was written.
type IncidentMsg struct{ Path string }

var (
	styleNormal  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleHeading = lipgloss.NewStyle().Foreground(lipgloss.Color("87")).Bold(true)
	styleBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
)

type Model struct {
	width, height int
	logVP         viewport.Model
	errorVP       viewport.Model
	agents        []AgentRow
	logLines      []string
	errorLines    []string
	ready         bool
}

type AgentRow struct {
	ID        string
	Name      string
	Type      string
	Status    string
	ErrorCount int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		half := (m.height - 6) / 2
		m.logVP = viewport.New(m.width-2, half)
		m.errorVP = viewport.New(m.width-2, half)
		m.ready = true

	case UILogMsg:
		style := styleNormal
		if msg.IsError {
			style = styleError
		}
		line := style.Render(fmt.Sprintf("[%s] %s", msg.AgentID[:8], msg.Line))
		m.logLines = append(m.logLines, line)
		m.logVP.SetContent(joinLines(m.logLines))
		m.logVP.GotoBottom()

		if msg.IsError {
			m.errorLines = append(m.errorLines, styleError.Render("⚡ "+msg.Line))
			m.errorVP.SetContent(joinLines(m.errorLines))
			m.errorVP.GotoBottom()
		}

	case IncidentMsg:
		notice := styleError.Render("📁 INCIDENT WRITTEN: " + msg.Path)
		m.errorLines = append(m.errorLines, notice)
		m.errorVP.SetContent(joinLines(m.errorLines))

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.logVP, cmd = m.logVP.Update(msg)
	cmds = append(cmds, cmd)
	m.errorVP, cmd = m.errorVP.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing KNIRVSHELL..."
	}
	header := styleHeading.Render("◈ KNIRVSHELL — Agentic Supervisor")
	logPanel := styleBorder.Render(styleHeading.Render("📡 Live Logs") + "\n" + m.logVP.View())
	errPanel := styleBorder.Render(styleHeading.Render("🚨 Error Monitor") + "\n" + m.errorVP.View())
	help := styleNormal.Render("q: quit  ↑↓: scroll")
	return header + "\n" + logPanel + "\n" + errPanel + "\n" + help
}

func joinLines(lines []string) string {
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}
```

---

## 2. The "DNA of an Error" (.md) Specification
When the Watchtower detects a failure, it halts the flow and compiles a **Forensic Error Bundle** into your Obsidian Vault.

### File Naming Convention
`INC-[YYYYMMDD]-[UUID].md`

### YAML Frontmatter (Machine Readable)
```yaml
---
id: "INC-2026-0409-A7X"
agent: "[[AGENT_NAME]]"
status: "DRAFT" # Becomes 'STAGED' upon submission to KNIRVGRAPH
exit_code: 1
timestamp: 2026-04-09T03:00:00Z
environment: "Local/Docker/Remote"
error_signature: "REF_ERROR_04"
nrn_bounty_suggested: 25.0
tags: [ergo, error-node, knirvshell-capture]
---
```

### Markdown Body (Human/Agent Readable)
* **## 🚨 Incident Summary:** High-level description of the crash.
* **## 🧠 Brain State:** The last 5 LLM prompt/response pairs (scraped from the buffer).
* **## 🛠 Tool Context:** If a tool was active (e.g., `KNIRVGRAPH`), include the specific JSON arguments used.
* **## 📜 Raw Forensic Trace:**
    > [!ABSTRACT] Standard Error (stderr)
    > ```bash
    > [Capture of the last 50 lines leading to crash]
    > ```

### Incident Generation Code

```go
// internal/incident/incident.go

package incident

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/knirv/knirvshell/internal/runner"
)

type Incident struct {
	ID              string
	AgentName       string
	AgentType       string
	Status          string
	ExitCode        int
	Timestamp       time.Time
	Environment     string
	ErrorSignature  string
	TriggerLine     string
	NRNBounty       float64
	LogSnapshot     []string // last 50 lines from ring buffer
	Tags            []string
}

func New(proc *runner.AgentProcess, snapshot []string, triggerLine, sigName string) *Incident {
	id := fmt.Sprintf("INC-%s-%s", time.Now().Format("20060102"), uuid.NewString()[:8])
	last50 := snapshot
	if len(last50) > 50 {
		last50 = last50[len(last50)-50:]
	}
	return &Incident{
		ID:             id,
		AgentName:      proc.AgentName,
		AgentType:      string(proc.Type),
		Status:         "DRAFT",
		ExitCode:       1,
		Timestamp:      time.Now().UTC(),
		Environment:    string(proc.Type),
		ErrorSignature: sigName,
		TriggerLine:    triggerLine,
		NRNBounty:      25.0,
		LogSnapshot:    last50,
		Tags:           []string{"ergo", "error-node", "knirvshell-capture"},
	}
}

const incidentTemplate = `---
id: "{{.ID}}"
agent: "[[{{.AgentName}}]]"
status: "{{.Status}}"
exit_code: {{.ExitCode}}
timestamp: {{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}
environment: "{{.Environment}}"
error_signature: "{{.ErrorSignature}}"
nrn_bounty_suggested: {{.NRNBounty}}
tags: [{{range $i, $t := .Tags}}{{if $i}}, {{end}}{{$t}}{{end}}]
---

## 🚨 Incident Summary

**Agent:** {{.AgentName}}  
**Signature:** ` + "`{{.ErrorSignature}}`" + `  
**Trigger:** ` + "`{{.TriggerLine}}`" + `  
**Captured:** {{.Timestamp.Format "2006-01-02 15:04:05 UTC"}}

## 🧠 Brain State

> _LLM prompt/response pairs — populate from agent memory if available._

## 🛠 Tool Context

> _Active tool JSON arguments at time of failure — populate from agent context._

## 📜 Raw Forensic Trace

> [!ABSTRACT] Standard Error (stderr — last {{len .LogSnapshot}} lines)
> ` + "```" + `bash
{{range .LogSnapshot}}{{.}}
{{end}}` + "```" + `
`

// WriteToVault renders the incident to the configured Obsidian vault directory.
func (inc *Incident) WriteToVault() (string, error) {
	vaultDir := vaultPath() // reads from config
	filename := inc.ID + ".md"
	path := filepath.Join(vaultDir, "Incidents", filename)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	tmpl, err := template.New("incident").Parse(incidentTemplate)
	if err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return path, tmpl.Execute(f, inc)
}

// OpenInObsidian uses the obsidian:// URI scheme to pull focus to the vault.
func OpenInObsidian(path string) {
	uri := "obsidian://open?path=" + strings.ReplaceAll(path, " ", "%20")
	exec.Command("xdg-open", uri).Start() // Linux; swap for `open` on macOS
}

func vaultPath() string {
	// TODO: read from ~/.config/knirvshell/config.yaml via Viper
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "KNIRVVault")
}
```

---

## 3. The Functional Workflow

### Phase A: Capture & Auto-Open
1.  **Detection:** Watchtower hits a "Panic" or "Traceback."
2.  **Compilation:** Go writes the `.md` file using a predefined template.
3.  **Handoff:** `knirvshell` executes `open [filename].md`, instantly pulling focus to Obsidian for the dev/user to review the "Crime Scene."

### Phase B: Economic Staging
Once the user reviews the error in Obsidian and confirms it's a network-level issue:
1.  **Command:** `knirv stage [inc-file].md --bounty [NRN_AMOUNT]`
2.  **Private Graph:** The file is pushed to the user's **Private KNIRVGRAPH** for indexing.
3.  **The Bounty:** The incident is broadcast to the **KNIRVARENA** as an "Ergo" node.

### Staging Client

```go
// internal/staging/staging.go

package staging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// KNIRVGraphClient submits incidents to the private KNIRVGRAPH node.
type KNIRVGraphClient struct {
	GraphURL string
	ArenaURL string
	APIKey   string
	HTTP     *http.Client
}

func New(graphURL, arenaURL, apiKey string) *KNIRVGraphClient {
	return &KNIRVGraphClient{
		GraphURL: graphURL,
		ArenaURL: arenaURL,
		APIKey:   apiKey,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
	}
}

type ErgoSubmission struct {
	IncidentID     string  `json:"incident_id"`
	AgentName      string  `json:"agent_name"`
	ErrorSignature string  `json:"error_signature"`
	NRNBounty      float64 `json:"nrn_bounty"`
	MDContent      string  `json:"md_content"`
	Tags           []string `json:"tags"`
}

// Stage reads the incident .md file and pushes it to KNIRVGRAPH + KNIRVARENA.
func (c *KNIRVGraphClient) Stage(incidentPath string, bounty float64) error {
	raw, err := os.ReadFile(incidentPath)
	if err != nil {
		return fmt.Errorf("reading incident file: %w", err)
	}

	sub := ErgoSubmission{
		MDContent: string(raw),
		NRNBounty: bounty,
	}

	body, _ := json.Marshal(sub)

	// Step 1: index in private KNIRVGRAPH
	req, _ := http.NewRequest("POST", c.GraphURL+"/api/incidents", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("knirvgraph submit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("knirvgraph returned %d", resp.StatusCode)
	}

	// Step 2: broadcast to KNIRVARENA as Ergo node
	req2, _ := http.NewRequest("POST", c.ArenaURL+"/api/ergo", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+c.APIKey)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := c.HTTP.Do(req2)
	if err != nil {
		return fmt.Errorf("knirvarena submit: %w", err)
	}
	defer resp2.Body.Close()

	return nil
}
```

---

## 4. Extended Command Set (Cobra)

| Command | Action |
| :--- | :--- |
| `knirv shell` | Launches the interactive TUI monitor (Bubble Tea). |
| `knirv register` | Adds an agent binary/image to the local `registry.db`. |
| `knirv run [agent]` | Starts an agent with a supervisory wrapper. |
| `knirv stage [file]` | Submits an error report to the private KNIRVGRAPH for NRN bounty placement. |
| `knirv monitor` | Lists all active pids/containers managed by the shell. |

### Root Command + Wiring

```go
// cmd/root.go

package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "knirv",
	Short: "KNIRVSHELL — Agentic Supervisor for the KNIRV Network",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/knirvshell/config.yaml)")
	rootCmd.AddCommand(shellCmd, registerCmd, runCmd, stageCmd, monitorCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(home + "/.config/knirvshell")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	viper.ReadInConfig()
}
```

### `knirv run` — Supervisory Launch

```go
// cmd/run.go

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/knirv/knirvshell/internal/runner"
	"github.com/knirv/knirvshell/internal/registry"
	"github.com/knirv/knirvshell/internal/tui"
	"github.com/knirv/knirvshell/internal/watchtower"
	"github.com/knirv/knirvshell/internal/incident"
)

var runCmd = &cobra.Command{
	Use:   "run [agent-name]",
	Short: "Start an agent under KNIRVSHELL supervision",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

		reg, err := registry.Open("registry.db")
		if err != nil {
			return err
		}
		cfg, err := reg.Get(agentName)
		if err != nil {
			return fmt.Errorf("agent %q not registered: %w", agentName, err)
		}

		r, err := runner.ForType(cfg.Type)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		proc, err := r.Start(ctx, *cfg)
		if err != nil {
			return err
		}
		fmt.Printf("◈ Started agent %q (id: %s)\n", agentName, proc.ID)

		uiCh := make(chan watchtower.UILogLine, 512)
		incCh := make(chan *incident.Incident, 64)

		wt := watchtower.New(uiCh, incCh)
		wt.Watch(ctx, proc)

		// Launch TUI
		m := tui.Model{}
		p := tea.NewProgram(m, tea.WithAltScreen())

		// Bridge channels → Bubble Tea messages
		go func() {
			for {
				select {
				case line := <-uiCh:
					p.Send(tui.UILogMsg(line))
				case inc := <-incCh:
					p.Send(tui.IncidentMsg{Path: inc.ID + ".md"})
				case <-ctx.Done():
					p.Quit()
					return
				}
			}
		}()

		_, err = p.Run()
		return err
	},
}
```

### `knirv stage` — Economic Submission

```go
// cmd/stage.go

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/knirv/knirvshell/internal/staging"
)

var stageBounty float64

var stageCmd = &cobra.Command{
	Use:   "stage [incident-file]",
	Short: "Submit an incident report to KNIRVGRAPH + KNIRVARENA for NRN bounty",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		incPath := args[0]
		graphURL := viper.GetString("knirvgraph.url")
		arenaURL := viper.GetString("knirvarena.url")
		apiKey   := viper.GetString("api_key")

		client := staging.New(graphURL, arenaURL, apiKey)
		if err := client.Stage(incPath, stageBounty); err != nil {
			return fmt.Errorf("staging failed: %w", err)
		}
		fmt.Printf("✓ Staged %s with %.2f NRN bounty\n", incPath, stageBounty)
		return nil
	},
}

func init() {
	stageCmd.Flags().Float64VarP(&stageBounty, "bounty", "b", 25.0, "NRN bounty amount")
}
```

---

## Config Schema

```yaml
# ~/.config/knirvshell/config.yaml

api_key: "your-knirv-api-key"

knirvgraph:
  url: "http://localhost:8082"

knirvarena:
  url: "http://localhost:8086"

obsidian:
  vault: "~/Documents/KNIRVVault"

agents:
  - name: "researcher-bot"
    type: "docker"
    image: "ai/researcher:latest"
    mounts:
      "./data": "/app/data"
  - name: "coder-local"
    type: "local"
    path: "./bin/coder"
    args: ["--model", "claude-opus-4-6"]
  - name: "remote-analyzer"
    type: "ssh"
    host: "10.0.1.42"
    user: "knirv"
    key_path: "~/.ssh/knirv_ed25519"
    remote_cmd: "/opt/knirv/analyzer --watch"
```

---

## Registry (SQLite)

```go
// internal/registry/schema.go

package registry

const schema = `
CREATE TABLE IF NOT EXISTS agents (
    name        TEXT PRIMARY KEY,
    type        TEXT NOT NULL CHECK(type IN ('local','docker','ssh')),
    path        TEXT,
    image       TEXT,
    host        TEXT,
    user        TEXT,
    key_path    TEXT,
    remote_cmd  TEXT,
    args        TEXT,   -- JSON array
    env         TEXT,   -- JSON array
    mounts      TEXT,   -- JSON object
    registered_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
`
// internal/registry/registry.go

package registry

import (
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
	"github.com/knirv/knirvshell/internal/runner"
)

type Registry struct{ db *sql.DB }

func Open(path string) (*Registry, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(schema)
	return &Registry{db: db}, err
}

func (r *Registry) Register(cfg runner.AgentConfig) error {
	args, _ := json.Marshal(cfg.Args)
	env, _  := json.Marshal(cfg.Env)
	mounts, _ := json.Marshal(cfg.Mounts)
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO agents (name, type, path, image, host, user, key_path, remote_cmd, args, env, mounts)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cfg.Name, cfg.Type, cfg.Path, cfg.Image, cfg.Host, cfg.User, cfg.KeyPath, cfg.RemoteCmd,
		string(args), string(env), string(mounts),
	)
	return err
}

func (r *Registry) Get(name string) (*runner.AgentConfig, error) {
	row := r.db.QueryRow(`SELECT name,type,path,image,host,user,key_path,remote_cmd,args,env,mounts FROM agents WHERE name=?`, name)
	var cfg runner.AgentConfig
	var args, env, mounts string
	err := row.Scan(&cfg.Name, &cfg.Type, &cfg.Path, &cfg.Image, &cfg.Host, &cfg.User, &cfg.KeyPath, &cfg.RemoteCmd, &args, &env, &mounts)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(args), &cfg.Args)
	json.Unmarshal([]byte(env), &cfg.Env)
	json.Unmarshal([]byte(mounts), &cfg.Mounts)
	return &cfg, nil
}
```

---

## 5. Technical Implementation Checklist

* [ ] **Signal Passthrough:** `signal.NotifyContext` catches `SIGTERM`/`SIGINT` and calls each `AgentProcess.StopFn` before exiting.
* [ ] **Buffer Management:** `RingBuffer` at 500-line capacity. Watchtower `Last(50)` for forensic snapshots. Zero heap growth at steady state.
* [ ] **Obsidian Path Mapping:** `obsidian.vault` key in config, defaulting to `~/Documents/KNIRVVault`. Override via `KNIRV_VAULT` env var.
* [ ] **Regex Library:** `DefaultSignatures()` in `internal/watchtower/signatures.go` — Python, Go, Node, OpenAI, Anthropic, and generic system errors. Signatures are case-insensitive.
* [ ] **Non-blocking TUI channel:** `uiCh` is buffered at 512. `select { case ch <- msg: default: }` prevents Watchtower from blocking on a lagging UI.
* [ ] **Docker demux:** Use `stdcopy.StdCopy` to split the Docker multiplexed stream into separate stdout/stderr `io.Pipe` pairs.
* [ ] **SSH HostKey:** `InsecureIgnoreHostKey()` is dev-only. Production must use `knownhosts.New(path)`.
* [ ] **SQLite concurrency:** Open registry with `?_journal_mode=WAL` to support concurrent reads from `monitor` while `run` writes.
* [ ] **go.mod dependencies:**
  ```
  github.com/charmbracelet/bubbletea
  github.com/charmbracelet/bubbles
  github.com/charmbracelet/lipgloss
  github.com/docker/docker
  golang.org/x/crypto
  github.com/google/uuid
  github.com/mattn/go-sqlite3
  github.com/spf13/cobra
  github.com/spf13/viper
  ```
* [ ] **P model:** Add `modp/components/shell/knirvshell_machine.p` to formally verify Watchtower → Incident → Stage state transitions.
* [ ] **Integration test:** Add `integration-tests/knirvshell_test.go` using a local mock agent that deliberately emits `Traceback` — verify incident file written and KNIRVGRAPH endpoint hit.

---
