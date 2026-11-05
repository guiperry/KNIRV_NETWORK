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
}

func NewManager(config Config, source SealSource) *Manager {
	return &Manager{config: config, source: source, baoBinary: embedded.BaoBinary, pluginBinary: embedded.PluginBinary, pluginSHA: embedded.PluginSHA256, status: Status{State: Absent}}
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
	wait       <-chan error
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
	backoff := time.Second
	started := time.Now()
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
				if time.Since(started) < time.Minute {
					backoff *= 2
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
				} else {
					backoff = time.Second
				}
				m.setStatus(Status{State: Degraded, Reason: "OpenBao exited; restarting"})
				log.Print("WARN: OpenBao exited; attempting restart")
			}
		}
		for {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				m.setStatus(Status{State: Absent})
				return
			case <-timer.C:
			}
			next, absent, err := m.startChild(ctx)
			if err == nil && !absent {
				current = next
				started = time.Now()
				m.setStatus(Status{State: Ready})
				break
			}
			if absent {
				m.setStatus(Status{State: Absent})
				return
			}
			m.setStatus(Status{State: Degraded, Reason: err.Error()})
			log.Printf("WARN: OpenBao restart failed: %v", err)
			if backoff < 30*time.Second {
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
		}
	}
}

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
