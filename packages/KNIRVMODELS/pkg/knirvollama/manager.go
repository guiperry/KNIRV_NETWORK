package knirvollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Phases a Manager reports. They are visible in Status so a first run, which can
// download a release and pull a model, is not a silent wait.
const (
	PhaseIdle        = "idle"        // created, Start not called
	PhaseDisabled    = "disabled"    // not running, not enabled, nothing to adopt
	PhaseAdopting    = "adopting"    // probing an existing instance
	PhaseDownloading = "downloading" // fetching the pinned release
	PhaseStarting    = "starting"    // child started, waiting for it to answer
	PhasePulling     = "pulling"     // pulling the embedding model
	PhaseReady       = "ready"
	PhaseDegraded    = "degraded" // running, but something is wrong; see Error
	PhaseFailed      = "failed"   // could not be provisioned or started
	PhaseStopped     = "stopped"
)

// Status is the block published in the models status document (SDK
// models.OllamaStatus). It carries no path and no secret.
type Status struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
	Digest   string `json:"digest,omitempty"`
	// Pinned is true when a digest pin is configured and matched.
	Pinned  bool   `json:"pinned"`
	Version string `json:"version,omitempty"`
	Phase   string `json:"phase"`
	Ready   bool   `json:"ready"`
	// Adopted is true when an instance that was already running is in use and this
	// manager did not start it (and so will not stop it).
	Adopted bool   `json:"adopted"`
	Bind    string `json:"bind,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Manager provisions, starts, adopts and watches Ollama and its embedding model.
type Manager struct {
	cfg Config
	hc  *http.Client

	startMu sync.Mutex // serializes Start so mu is never held across a long wait
	mu      sync.RWMutex
	st      Status
	cmd     *exec.Cmd
	exited  chan struct{}
	lastTry time.Time // last Start from Run, so a failing install is not retried every tick
}

// New builds a Manager. It does nothing until Start or Run.
func New(cfg Config) *Manager {
	cfg = cfg.normalize()
	m := &Manager{cfg: cfg, hc: &http.Client{Timeout: 0}}
	m.st = Status{Enabled: cfg.Enabled, Endpoint: cfg.BaseURL, Model: cfg.EmbeddingModel, Version: cfg.Version, Phase: PhaseIdle}
	return m
}

// Status returns a copy of the current block.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.st
}

// Endpoint is the base URL callers should use once Status().Ready is true.
func (m *Manager) Endpoint() string { return m.cfg.BaseURL }

// Embed is the daemon-owned embedding operation. Callers use the inference
// socket rather than this endpoint, so model selection and readiness remain
// under the supervising manager.
func (m *Manager) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if !m.Status().Ready {
		return nil, fmt.Errorf("ollama embedding service is not ready")
	}
	body, err := json.Marshal(struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{m.cfg.EmbeddingModel, texts})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.BaseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed: HTTP %d", resp.StatusCode)
	}
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) != len(texts) {
		return nil, fmt.Errorf("ollama embed returned %d vectors for %d inputs", len(out.Embeddings), len(texts))
	}
	for _, v := range out.Embeddings {
		if len(v) == 0 {
			return nil, fmt.Errorf("ollama embed returned an empty vector")
		}
	}
	return out.Embeddings, nil
}

func (m *Manager) set(f func(*Status)) {
	m.mu.Lock()
	f(&m.st)
	m.mu.Unlock()
}

func (m *Manager) phase(p string) {
	m.set(func(s *Status) { s.Phase = p; s.Ready = false })
	m.cfg.Logf("ollama: %s", p)
}

// degrade records that the instance runs but must not be relied on.
func (m *Manager) degrade(err error) error {
	m.set(func(s *Status) { s.Phase = PhaseDegraded; s.Ready = false; s.Error = err.Error() })
	m.cfg.Logf("ollama: degraded: %v", err)
	return err
}

func (m *Manager) fail(err error) error {
	m.set(func(s *Status) { s.Phase = PhaseFailed; s.Ready = false; s.Error = err.Error() })
	m.cfg.Logf("ollama: failed: %v", err)
	return err
}

// Start brings Ollama and the embedding model up. It is meant to run in the
// background and its error is advisory: the caller logs it and carries on, and the
// failure is also in Status (Degraded or Failed). Start never panics the daemon.
func (m *Manager) Start(ctx context.Context) error {
	m.startMu.Lock()
	defer m.startMu.Unlock()
	m.set(func(s *Status) { s.Error = ""; s.Adopted = false })

	// 1. Adopt a running instance.
	m.phase(PhaseAdopting)
	if _, ok := probe(ctx, m.hc, m.cfg.BaseURL); ok {
		bind, err := checkBind(m.cfg.BaseURL)
		m.set(func(s *Status) { s.Adopted = true; s.Bind = bind })
		if err != nil {
			return m.degrade(err)
		}
		m.refreshVersion(ctx)
		return m.ensureModel(ctx)
	}
	if !m.cfg.Enabled {
		m.phase(PhaseDisabled)
		return nil
	}

	// 2. Refuse to start where it would listen beyond this machine.
	bind, err := checkBind(m.cfg.BaseURL)
	m.set(func(s *Status) { s.Bind = bind })
	if err != nil {
		return m.degrade(err)
	}

	// 3. Install the pinned release, then start and supervise a child on loopback.
	m.phase(PhaseDownloading)
	bin, err := EnsureBinary(ctx, m.cfg, m.hc)
	if err != nil {
		return m.fail(err)
	}
	if err := m.startChild(ctx, bin); err != nil {
		return m.fail(err)
	}
	m.refreshVersion(ctx)

	// 4. The embedding model.
	return m.ensureModel(ctx)
}

// startChild runs `ollama serve` and waits until it answers, retrying when the child
// exits during startup. A timeout is never retried.
func (m *Manager) startChild(ctx context.Context, bin string) error {
	var last error
	for attempt := 1; attempt <= m.cfg.MaxStartAttempts; attempt++ {
		m.phase(PhaseStarting)
		cmd, exited, err := m.spawn(bin)
		if err != nil {
			return err
		}
		m.mu.Lock()
		m.cmd, m.exited = cmd, exited
		m.mu.Unlock()

		deadline := time.NewTimer(m.cfg.StartTimeout)
		tick := time.NewTicker(250 * time.Millisecond)
		var outcome string
	wait:
		for {
			select {
			case <-ctx.Done():
				outcome = "cancelled"
				break wait
			case <-exited:
				outcome = "exited"
				break wait
			case <-deadline.C:
				outcome = "timeout"
				break wait
			case <-tick.C:
				if _, ok := probe(ctx, m.hc, m.cfg.BaseURL); ok {
					outcome = "ready"
					break wait
				}
			}
		}
		deadline.Stop()
		tick.Stop()
		switch outcome {
		case "ready":
			return nil
		case "cancelled":
			m.Stop()
			return ctx.Err()
		case "timeout":
			m.Stop()
			return fmt.Errorf("ollama did not answer at %s within %s", m.cfg.BaseURL, m.cfg.StartTimeout)
		}
		last = fmt.Errorf("ollama exited during start (attempt %d of %d)", attempt, m.cfg.MaxStartAttempts)
		m.cfg.Logf("ollama: %v", last)
		if attempt < m.cfg.MaxStartAttempts {
			wait := m.cfg.RetryBackoff[min(attempt-1, len(m.cfg.RetryBackoff)-1)]
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return last
}

func (m *Manager) spawn(bin string) (*exec.Cmd, chan struct{}, error) {
	host, port, err := hostOf(m.cfg.BaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(m.cfg.modelsDir(), 0o755); err != nil {
		return nil, nil, err
	}
	cmd := exec.Command(bin, "serve")
	env := os.Environ()
	set := func(k, v string) { env = append(env, k+"="+v) }
	set("OLLAMA_HOST", fmt.Sprintf("%s:%d", host, port)) // loopback only, by construction
	set("OLLAMA_MODELS", m.cfg.modelsDir())
	if m.cfg.NumParallel > 0 {
		set("OLLAMA_NUM_PARALLEL", strconv.Itoa(m.cfg.NumParallel))
	}
	if m.cfg.Threads > 0 {
		set("OLLAMA_NUM_THREAD", strconv.Itoa(m.cfg.Threads))
	}
	if m.cfg.KeepAlive != "" {
		set("OLLAMA_KEEP_ALIVE", m.cfg.KeepAlive)
	}
	cmd.Env = env
	w := &lineLog{logf: m.cfg.Logf}
	cmd.Stdout, cmd.Stderr = w, w
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start %s: %w", bin, err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	return cmd, exited, nil
}

// lineLog forwards a child's output to Logf one line at a time with a prefix.
type lineLog struct {
	logf func(string, ...any)
	buf  bytes.Buffer
	mu   sync.Mutex
}

func (w *lineLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.logf("[OLLAMA] %s", strings.TrimRight(line, "\r\n"))
	}
	return len(p), nil
}

// Stop ends a child this manager started. An adopted instance is left alone.
func (m *Manager) Stop() {
	m.mu.Lock()
	cmd, exited := m.cmd, m.exited
	m.cmd, m.exited = nil, nil
	m.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-exited:
		case <-time.After(m.cfg.StopTimeout):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-exited
		}
	}
	m.set(func(s *Status) { s.Phase = PhaseStopped; s.Ready = false })
}

func (m *Manager) refreshVersion(ctx context.Context) {
	var v struct {
		Version string `json:"version"`
	}
	if m.getJSON(ctx, "/api/version", &v) == nil && v.Version != "" {
		m.set(func(s *Status) { s.Version = v.Version })
	}
}

func (m *Manager) getJSON(ctx context.Context, p string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.BaseURL+p, nil)
	if err != nil {
		return err
	}
	resp, err := m.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", p, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

// modelDigest returns the digest of the embedding model, or "" when it is absent.
func (m *Manager) modelDigest(ctx context.Context) (string, error) {
	var tags struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := m.getJSON(ctx, "/api/tags", &tags); err != nil {
		return "", err
	}
	want := m.cfg.EmbeddingModel
	for _, t := range tags.Models {
		if t.Name == want || t.Name == want+":latest" || strings.TrimSuffix(t.Name, ":latest") == want {
			return t.Digest, nil
		}
	}
	return "", nil
}

// ensureModel makes sure the embedding model is present and matches its pin.
func (m *Manager) ensureModel(ctx context.Context) error {
	digest, err := m.modelDigest(ctx)
	if err != nil {
		return m.degrade(fmt.Errorf("list models: %w", err))
	}
	pin := m.cfg.EmbeddingDigest
	if digest == "" || (pin != "" && !digestMatches(digest, pin)) {
		m.phase(PhasePulling)
		if err := m.pull(ctx); err != nil {
			return m.degrade(fmt.Errorf("pull %s: %w", m.cfg.EmbeddingModel, err))
		}
		if digest, err = m.modelDigest(ctx); err != nil {
			return m.degrade(fmt.Errorf("list models: %w", err))
		}
		if digest == "" {
			return m.degrade(fmt.Errorf("%s is not present after the pull", m.cfg.EmbeddingModel))
		}
	}
	pinned := pin != "" && digestMatches(digest, pin)
	m.set(func(s *Status) { s.Digest = digest; s.Pinned = pinned })
	if pin != "" && !pinned {
		// Never silent: a different model must not be used under this id.
		return m.degrade(fmt.Errorf("%s digest %s does not match the pinned %s", m.cfg.EmbeddingModel, short(digest), short(pin)))
	}
	m.set(func(s *Status) { s.Phase = PhaseReady; s.Ready = true; s.Error = "" })
	m.cfg.Logf("ollama: ready at %s (%s %s, pinned=%t)", m.cfg.BaseURL, m.cfg.EmbeddingModel, short(digest), pinned)
	return nil
}

func short(d string) string {
	d = strings.TrimPrefix(d, "sha256:")
	if len(d) > 12 {
		return d[:12]
	}
	return d
}

// pull runs POST /api/pull and reads its progress stream to completion.
func (m *Manager) pull(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.PullTimeout)
	defer cancel()
	body, _ := json.Marshal(map[string]any{"name": m.cfg.EmbeddingModel, "stream": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.BaseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	success := false
	for sc.Scan() {
		var ev struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		if ev.Error != "" {
			return errors.New(ev.Error)
		}
		if ev.Status == "success" {
			success = true
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if !success {
		return errors.New("the pull ended without success")
	}
	return nil
}

// Run starts the manager and then watches it until ctx is cancelled: a child this
// manager started is restarted when it exits, and an adopted instance that goes
// away is reported not ready. It stops its own child on the way out.
func (m *Manager) Run(ctx context.Context, every time.Duration) {
	if every <= 0 {
		every = 10 * time.Second
	}
	m.lastTry = time.Now()
	_ = m.Start(ctx)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			m.Stop()
			return
		case <-t.C:
			st := m.Status()
			if st.Phase == PhaseDisabled || st.Phase == PhaseStopped {
				continue
			}
			if _, ok := probe(ctx, m.hc, m.cfg.BaseURL); ok {
				if !st.Ready && st.Phase != PhaseDegraded && st.Phase != PhaseFailed {
					_ = m.Start(ctx)
				}
				continue
			}
			if st.Ready {
				m.phase(PhaseDegraded)
				m.set(func(s *Status) { s.Error = "Ollama stopped answering" })
			}
			if m.cfg.Enabled && time.Since(m.lastTry) >= m.retryEvery() {
				m.lastTry = time.Now()
				_ = m.Start(ctx) // restarts a child we own, or re-adopts
			}
		}
	}
}

// retryEvery is the least time between restarts attempted by Run.
func (m *Manager) retryEvery() time.Duration {
	if n := len(m.cfg.RetryBackoff); n > 0 {
		return m.cfg.RetryBackoff[n-1]
	}
	return 30 * time.Second
}
