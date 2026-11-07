package knirvbao

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	embedded "knirv-server/pkg/embedded/knirvbao"
)

type State string

var errResourceBusy = errors.New("OpenBao resource is already in use")

const (
	Ready    State = "ready"
	Degraded State = "degraded"
	Absent   State = "absent"
)

type Status struct {
	State  State  `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// SealSource exposes only the derived static-seal key, never the key-file
// password or its scrypt output. A nil key means this node has no key file.
type SealSource interface {
	SealKey(context.Context) ([]byte, error)
}

type Manager struct {
	config        Config
	maxRestarts   int
	source        SealSource
	baoBinary     func() []byte
	pluginBinary  func() []byte
	pluginSHA     func() string
	beforeFinish  func(context.Context, *baoAPI, string) error // integration-test seam; nil in production; runs before the bootstrap marker is written
	mu            sync.Mutex
	status        Status
	healthWarning string
	cancel        context.CancelFunc
	done          chan struct{}
	lockFile      *os.File

	// clock, now, spawn and stableWindow are seams for deterministic testing.
	// In production they default to the real clock, time.Now, startChild and one
	// minute. A successful run that lasts at least stableWindow clears the failure
	// budget; a child that crashes sooner (or never starts) keeps accumulating
	// restarts so the manager eventually gives up instead of looping forever.
	clock        clock
	now          func() time.Time
	spawn        func(ctx context.Context) (*child, bool, error)
	stableWindow time.Duration
}

func NewManager(config Config, source SealSource) *Manager {
	m := &Manager{
		config: config, source: source,
		baoBinary: embedded.BaoBinary, pluginBinary: embedded.PluginBinary, pluginSHA: embedded.PluginSHA256,
		status: Status{State: Absent}, maxRestarts: 5,
	}
	m.clock = realClock{}
	m.now = time.Now
	m.spawn = m.startChild
	m.stableWindow = time.Minute
	return m
}

func (m *Manager) Socket() string {
	root, err := m.config.root()
	if err != nil {
		return ""
	}
	return filepath.Join(root, "bao.sock")
}

func (m *Manager) setStatus(status Status) {
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
}

func (m *Manager) Health() Status {
	m.mu.Lock()
	status := m.status
	m.mu.Unlock()
	if status.State != Ready {
		return status
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	api := newBaoAPI(m.Socket())
	defer api.client.CloseIdleConnections()
	state, err := api.sealStatus(ctx)
	if err != nil {
		return m.healthFailure("OpenBao is unreachable")
	}
	if !state.Initialized || state.Sealed {
		return m.healthFailure("OpenBao is sealed or uninitialized")
	}
	if code, err := api.request(ctx, "GET", "sys/health", "", nil, nil); err != nil || code != 200 {
		return m.healthFailure("OpenBao is not active")
	}
	m.mu.Lock()
	m.healthWarning = ""
	status = m.status
	m.mu.Unlock()
	return status
}

func (m *Manager) healthFailure(reason string) Status {
	m.mu.Lock()
	if m.status.State != Ready {
		status := m.status
		m.mu.Unlock()
		return status
	}
	changed := m.healthWarning != reason
	m.healthWarning = reason
	m.mu.Unlock()
	if changed {
		log.Printf("WARN: %s", reason)
	}
	return Status{State: Degraded, Reason: reason}
}

type child struct {
	cmd        *exec.Cmd
	wait       chan error
	socket     string
	socketInfo os.FileInfo
	exited     bool
}

// Start reports the first startup result; supervision continues independently
// until Stop. It never adopts or signals an already-existing process.
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		m.setStatus(Status{State: Absent})
		return nil
	}
	if err := ctx.Err(); err != nil {
		m.setStatus(Status{State: Degraded, Reason: "OpenBao start cancelled"})
		return err
	}
	if m.source == nil {
		m.setStatus(Status{State: Degraded, Reason: "seal source is unavailable"})
		return errors.New("OpenBao seal source is unavailable")
	}
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return errors.New("OpenBao manager already started")
	}
	life, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.cancel, m.done = cancel, done
	m.status = Status{State: Degraded, Reason: "OpenBao is starting"}
	m.mu.Unlock()
	startup, stopStartup := context.WithCancel(life)
	stopCaller := context.AfterFunc(ctx, stopStartup)
	c, absent, err := m.startChild(startup)
	if err == nil && !absent && ctx.Err() != nil {
		stopChild(c)
		err = ctx.Err()
	}
	stopCaller()
	stopStartup()
	if err != nil && !errors.Is(err, errResourceBusy) && ctx.Err() == nil && life.Err() == nil {
		m.setStatus(Status{State: Degraded, Reason: err.Error()})
		go m.supervise(life, nil, done)
		return err
	}
	if err != nil || absent {
		cancel()
		m.releaseLock()
		m.mu.Lock()
		m.cancel = nil
		if absent {
			m.status = Status{State: Absent}
		} else {
			m.status = Status{State: Degraded, Reason: err.Error()}
		}
		m.mu.Unlock()
		close(done)
		return err
	}
	m.setStatus(Status{State: Ready})
	go m.supervise(life, c, done)
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cancel, done := m.cancel, m.done
	m.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) supervise(ctx context.Context, current *child, done chan struct{}) {
	defer close(done)
	defer func() { m.mu.Lock(); m.cancel = nil; m.mu.Unlock() }()
	defer m.releaseLock()
	const initialBackoff = time.Second
	const maxBackoff = 30 * time.Second
	backoff := initialBackoff
	restarts := 0
	lastHealthy := m.now()
	for {
		if current != nil {
			select {
			case <-ctx.Done():
				stopChild(current)
				m.setStatus(Status{State: Absent})
				return
			case <-current.wait:
				current.exited = true
				removeOwnedSocket(current)
				m.setStatus(Status{State: Degraded, Reason: "OpenBao exited; restarting"})
				log.Print("WARN: OpenBao exited; attempting restart")
				restarts++
				if time.Since(lastHealthy) >= m.stableWindow {
					// The child had a sustained healthy run before crashing: clear
					// the consecutive-failure budget so a flaky-but-recovering
					// vault is not declared permanently down.
					restarts = 0
					backoff = initialBackoff
					if !m.waitBackoff(ctx, backoff) {
						m.setStatus(Status{State: Absent})
						return
					}
				} else {
					if restarts > m.maxRestarts {
						m.setStatus(Status{State: Degraded, Reason: "OpenBao restarts exhausted"})
						log.Print("ERROR: OpenBao restarts exhausted; giving up")
						return
					}
					if !m.waitBackoff(ctx, backoff) {
						m.setStatus(Status{State: Absent})
						return
					}
					backoff = nextBackoff(backoff, maxBackoff)
				}
			}
		}
		for {
			next, absent, err := m.spawn(ctx)
			if err == nil && !absent {
				current = next
				lastHealthy = m.now()
				m.setStatus(Status{State: Ready})
				break
			}
			if absent {
				m.setStatus(Status{State: Absent})
				return
			}
			m.setStatus(Status{State: Degraded, Reason: err.Error()})
			log.Printf("WARN: OpenBao restart failed: %v", err)
			restarts++
			if restarts > m.maxRestarts {
				m.setStatus(Status{State: Degraded, Reason: "OpenBao restarts exhausted"})
				log.Print("ERROR: OpenBao restarts exhausted; giving up")
				return
			}
			if !m.waitBackoff(ctx, backoff) {
				m.setStatus(Status{State: Absent})
				return
			}
			backoff = nextBackoff(backoff, maxBackoff)
		}
	}
}

// waitBackoff waits for the backoff interval (via the injectable clock) or context
// cancellation, whichever comes first.
func (m *Manager) waitBackoff(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-m.clock.After(d):
		return true
	}
}

// nextBackoff doubles d, capping it at max.
func nextBackoff(d, max time.Duration) time.Duration {
	d *= 2
	if d > max {
		d = max
	}
	return d
}

// clock is the seam used for backoff waits so tests can run deterministically.
type clock interface {
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func reserveLoopback() (string, string, error) {
	a, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer a.Close()
	b, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer b.Close()
	return a.Addr().String(), b.Addr().String(), nil
}

func childEnvironment(seal []byte) []string {
	// Bao does not need the launcher's cloud credentials, node passwords or
	// other service secrets. Pass only ordinary process settings plus its seal.
	allowed := map[string]bool{"PATH": true, "LANG": true, "LC_ALL": true, "TZ": true}
	env := make([]string, 0, len(allowed)+1)
	for _, item := range os.Environ() {
		name, _, _ := strings.Cut(item, "=")
		if allowed[name] {
			env = append(env, item)
		}
	}
	return append(env, "KNIRV_BAO_SEAL_KEY="+base64.StdEncoding.EncodeToString(seal))
}

func stopChild(c *child) {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return
	}
	if c.exited {
		removeOwnedSocket(c)
		return
	}
	_ = c.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-c.wait:
	case <-time.After(5 * time.Second):
		_ = c.cmd.Process.Kill()
		<-c.wait
	}
	c.exited = true
	removeOwnedSocket(c)
}

func removeOwnedSocket(c *child) {
	if c == nil || c.socketInfo == nil || c.socketInfo.Mode()&os.ModeSocket == 0 {
		return
	}
	current, err := os.Lstat(c.socket)
	if err == nil && current.Mode()&os.ModeSocket != 0 && os.SameFile(c.socketInfo, current) {
		_ = os.Remove(c.socket)
	}
}

func (m *Manager) startChild(ctx context.Context) (*child, bool, error) {
	seal, err := m.source.SealKey(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("derive OpenBao seal: %w", err)
	}
	defer clear(seal)
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if len(seal) == 0 {
		return nil, true, nil
	}
	if len(seal) != 32 {
		return nil, false, errors.New("OpenBao seal must be 32 bytes")
	}
	root, err := m.config.root()
	if err != nil {
		return nil, false, err
	}
	if err := m.lockRoot(root); err != nil {
		return nil, false, err
	}
	socket := filepath.Join(root, "bao.sock")
	if _, err := os.Lstat(socket); err == nil {
		return nil, false, fmt.Errorf("%w: refusing to adopt existing socket", errResourceBusy)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	baoPath, _, err := installBinary(root, "bao", m.baoBinary())
	if err != nil {
		return nil, false, err
	}
	pluginPath, sha, err := installBinary(root, "openbao-plugin-secrets-knirv", m.pluginBinary())
	if err != nil {
		return nil, false, err
	}
	if sha != m.pluginSHA() {
		return nil, false, errors.New("embedded OpenBao plugin digest mismatch")
	}
	api, cluster, err := reserveLoopback()
	if err != nil {
		return nil, false, err
	}
	config, err := renderConfig(root, filepath.Dir(pluginPath), sha, api, cluster, seal)
	if err != nil {
		return nil, false, err
	}
	file, err := os.CreateTemp(root, "bao-config-*.hcl")
	if err != nil {
		return nil, false, err
	}
	if _, err := file.WriteString(config); err != nil {
		_ = file.Close()
		return nil, false, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return nil, false, err
	}
	if err := file.Close(); err != nil {
		return nil, false, err
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	cmd := exec.Command(baoPath, "server", "-config", file.Name())
	cmd.Env = childEnvironment(seal)
	err = cmd.Start()
	clear(cmd.Env)
	cmd.Env = nil
	clear(seal)
	if err != nil {
		return nil, false, fmt.Errorf("start OpenBao: %w", err)
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	c := &child{cmd: cmd, wait: wait, socket: socket}
	if err := m.initializeChild(ctx, root, c); err != nil {
		stopChild(c)
		return nil, false, err
	}
	return c, false, nil
}
