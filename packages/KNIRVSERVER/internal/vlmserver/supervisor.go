// Package vlmserver runs the MiniCPM-V server that serves HEAD's vision and mirror-text
// requests. It is started by the launcher, after the local text harness (Colibri, with
// llama.cpp as the fallback) has settled, so that the RAM it plans against is the RAM
// that is actually free. Everything it runs is chosen at start from this host's
// resources (see PlanFor); nothing about the machine is hard-coded.
package vlmserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Gate blocks until the local text harness has settled (Colibri serving, or ruled out
// with llama.cpp serving) and returns true. It returns false when ctx ends first.
type Gate func(ctx context.Context) bool

// Config wires a Supervisor. Zero values take the documented defaults.
type Config struct {
	DataDir   string          // <app data>/vlm: binaries, logs, models (default)
	ModelDir  string          // where the weights live (default DataDir/models)
	Files     []ProvisionFile // what to install (default MiniCPMFiles)
	ModelName string          // reported to HEAD (default: the language model's file name)
	Binary    string          // llama-server path; empty resolves via ResolveBinary
	Host      string          // listen address (default 127.0.0.1; loopback only)
	Port      int             // listen port (default 8093)
	Gate      Gate            // nil starts at once
	Probe     func(dataDir string) Host
	Shape     ModelShape // default MiniCPMShape
	HTTP      *http.Client
	// StartupTimeout bounds one load to healthy. A first load reads 18 GB, so the
	// default is 30 minutes.
	StartupTimeout time.Duration
	// StableAfter is how long a server must stay healthy before its restart counter
	// resets (default 2m).
	StableAfter    time.Duration
	RestartBackoff []time.Duration // default 10s, 30s, 2m, 5m
	StopTimeout    time.Duration   // default 20s
	Logf           func(format string, args ...any)
}

// Supervisor keeps one llama-server running with the planned arguments.
type Supervisor struct {
	cfg Config

	mu       sync.Mutex
	state    string
	note     string
	plan     *Plan
	model    string
	cmd      *exec.Cmd
	cmdDone  chan struct{}
	stopping bool
}

// New fills defaults and returns a Supervisor that has not started anything yet.
func New(cfg Config) *Supervisor {
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(os.TempDir(), "knirvserver", "vlm")
	}
	if cfg.ModelDir == "" {
		cfg.ModelDir = filepath.Join(cfg.DataDir, "models")
	}
	if cfg.Files == nil {
		cfg.Files = MiniCPMFiles
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8093
	}
	if cfg.Probe == nil {
		cfg.Probe = ProbeHost
	}
	if cfg.Shape.Layers == 0 {
		cfg.Shape = MiniCPMShape
	}
	if cfg.HTTP == nil {
		cfg.HTTP = defaultHTTPClient()
	}
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 30 * time.Minute
	}
	if cfg.StableAfter <= 0 {
		cfg.StableAfter = 2 * time.Minute
	}
	if cfg.RestartBackoff == nil {
		cfg.RestartBackoff = []time.Duration{10 * time.Second, 30 * time.Second, 2 * time.Minute, 5 * time.Minute}
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 20 * time.Second
	}
	if cfg.Logf == nil {
		cfg.Logf = func(format string, args ...any) { fmt.Fprintf(os.Stderr, "[vlm] "+format+"\n", args...) }
	}
	if cfg.ModelName == "" {
		cfg.ModelName = MiniCPMModelName
	}
	return &Supervisor{cfg: cfg, state: "idle"}
}

// defaultHTTPClient bounds connection setup and the wait for response headers, but
// never the whole request: a first run downloads an 18.7 GB weight file. A stalled
// body is caught by the idle-body watchdog in Provision instead.
func defaultHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
	}
	clone := transport.Clone()
	clone.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	clone.TLSHandshakeTimeout = 10 * time.Second
	clone.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: clone}
}

// MiniCPMModelName is the name HEAD reports for the default weights.
const MiniCPMModelName = "MiniCPM-V-4_7-35B-A3B-IQ4_XS"

// Endpoint is the address HEAD's backend calls (KNIRV_VLM_MINICPM_ENDPOINT).
func (s *Supervisor) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", s.cfg.Host, s.cfg.Port)
}

// Env is what the launcher exports so HEAD (KNIRVMODELS) uses this server. Variables the
// operator already set are left alone.
func (s *Supervisor) Env() []string {
	var out []string
	add := func(k, v string) {
		if os.Getenv(k) == "" {
			out = append(out, k+"="+v)
		}
	}
	add("KNIRV_VLM_BACKEND", "minicpm")
	add("KNIRV_VLM_MINICPM_ENDPOINT", s.Endpoint())
	add("KNIRV_VLM_MINICPM_MODEL", s.cfg.ModelName)
	return out
}

// Status is the document the launcher and status endpoints report.
func (s *Supervisor) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := map[string]any{
		"state":    s.state,
		"endpoint": s.Endpoint(),
		"model":    s.cfg.ModelName,
	}
	if s.note != "" {
		doc["note"] = s.note
	}
	if s.plan != nil {
		doc["plan"] = map[string]any{
			"mode":             s.plan.Mode,
			"ngl":              s.plan.NGPULayers,
			"n_cpu_moe":        s.plan.NCPUMoE,
			"projector_on_gpu": s.plan.ProjectorOnGPU,
			"load_mode":        s.plan.LoadMode,
			"threads":          s.plan.Threads,
			"ctx":              s.plan.Ctx,
			"notes":            s.plan.Notes,
			"warnings":         s.plan.Warnings,
		}
	}
	return doc
}

func (s *Supervisor) setState(state, note string) {
	s.mu.Lock()
	s.state, s.note = state, note
	s.mu.Unlock()
}

// Run blocks until ctx ends, keeping the server up. It returns ctx.Err().
func (s *Supervisor) Run(ctx context.Context) error {
	if s.cfg.Gate != nil {
		s.setState("waiting", "waiting for the local text harness to settle (Colibri first)")
		if !s.cfg.Gate(ctx) {
			s.setState("stopped", "")
			return ctx.Err()
		}
	}
	failures := 0
	for {
		if ctx.Err() != nil {
			s.setState("stopped", "")
			return ctx.Err()
		}
		readyFor, err := s.runOnce(ctx)
		if ctx.Err() != nil {
			s.stopChild()
			s.setState("stopped", "")
			return ctx.Err()
		}
		if readyFor >= s.cfg.StableAfter {
			failures = 0
		}
		failures++
		wait := s.backoff(failures)
		msg := "server exited"
		if err != nil {
			msg = err.Error()
		}
		s.setState("restarting", fmt.Sprintf("%s; restarting in %s", msg, wait))
		s.cfg.Logf("%s; restarting in %s (restart %d)", msg, wait, failures)
		if !sleepCtx(ctx, wait) {
			s.stopChild()
			s.setState("stopped", "")
			return ctx.Err()
		}
	}
}

func (s *Supervisor) backoff(n int) time.Duration {
	sched := s.cfg.RestartBackoff
	if len(sched) == 0 {
		return 0
	}
	if n-1 < len(sched) {
		return sched[n-1]
	}
	return sched[len(sched)-1]
}

// runOnce provisions, plans and starts one server, waits for it to become healthy and
// returns when it exits. readyFor is how long it was healthy.
func (s *Supervisor) runOnce(ctx context.Context) (time.Duration, error) {
	s.setState("provisioning", "")
	paths, err := Provision(ctx, s.cfg.ModelDir, s.cfg.Files, s.cfg.HTTP, s.cfg.Logf)
	if err != nil {
		return 0, err
	}
	model := paths[s.cfg.Files[0].Name]
	mmproj := ""
	if len(s.cfg.Files) > 1 {
		mmproj = paths[s.cfg.Files[1].Name]
	}

	host := s.cfg.Probe(s.cfg.DataDir)
	bin, err := s.resolveServer(ctx, host)
	if err != nil {
		return 0, err
	}
	plan := PlanFor(host, s.cfg.Shape, hasCUDA(bin))
	s.mu.Lock()
	s.plan = &plan
	s.model = model
	s.mu.Unlock()
	for _, w := range plan.Warnings {
		s.cfg.Logf("warning: %s", w)
	}
	for _, n := range plan.Notes {
		s.cfg.Logf("plan: %s", n)
	}
	s.cfg.Logf("starting %s on %s (%s mode, %d expert layers on CPU)", bin, s.Endpoint(), plan.Mode, plan.NCPUMoE)

	if s.healthy() {
		// A server already answers on this port (an earlier run that was not stopped).
		// Adopt it rather than fight it for the port.
		s.setState("ready", "adopted an existing server on "+s.Endpoint())
		return s.watchAdopted(ctx)
	}

	args := []string{"-m", model}
	if mmproj != "" {
		args = append(args, "--mmproj", mmproj)
	}
	args = append(args, "--host", s.cfg.Host, "--port", strconv.Itoa(s.cfg.Port))
	args = append(args, plan.Args()...)

	logPath := filepath.Join(s.cfg.DataDir, "logs", "llama-server.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return 0, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer logFile.Close()

	cmd := exec.Command(bin, args...)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start llama-server: %w", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	s.mu.Lock()
	s.cmd, s.cmdDone = cmd, done
	s.mu.Unlock()
	s.setState("starting", fmt.Sprintf("loading %s", s.cfg.ModelName))

	if err := s.waitHealthy(ctx, done); err != nil {
		s.stopChild()
		return 0, err
	}
	s.setState("ready", "")
	s.cfg.Logf("ready at %s", s.Endpoint())
	start := time.Now()
	select {
	case <-done:
		return time.Since(start), errors.New("llama-server exited; see " + logPath)
	case <-ctx.Done():
		return time.Since(start), ctx.Err()
	}
}

// waitHealthy polls /health until the server answers, the child exits, or the startup
// timeout passes.
func (s *Supervisor) waitHealthy(ctx context.Context, exited <-chan struct{}) error {
	deadline := time.Now().Add(s.cfg.StartupTimeout)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		if s.healthy() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not healthy after %s", s.cfg.StartupTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-exited:
			return errors.New("llama-server exited while loading")
		case <-tick.C:
		}
	}
}

// watchAdopted keeps an adopted server under watch until it stops answering.
func (s *Supervisor) watchAdopted(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	bad := 0
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return time.Since(start), ctx.Err()
		case <-tick.C:
		}
		if s.healthy() {
			bad = 0
			continue
		}
		bad++
		if bad >= 3 {
			return time.Since(start), errors.New("adopted server stopped answering")
		}
	}
}

func (s *Supervisor) healthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint()+"/health", nil)
	if err != nil {
		return false
	}
	res, err := s.cfg.HTTP.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<12))
	return res.StatusCode == http.StatusOK
}

// stopChild ends the server's whole process group, politely first.
func (s *Supervisor) stopChild() {
	s.mu.Lock()
	cmd, done := s.cmd, s.cmdDone
	s.cmd, s.cmdDone = nil, nil
	s.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	select {
	case <-done:
		return
	case <-time.After(s.cfg.StopTimeout):
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	<-done
}

// Stop ends the server. Run also stops it when its context ends.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	s.stopping = true
	s.mu.Unlock()
	s.stopChild()
	s.setState("stopped", "")
}

// ResolveBinary finds llama-server: an explicit path, then KNIRV_VLM_LLAMA_SERVER, then
// <data dir>/bin/llama-server, then PATH. It does not build one.
func ResolveBinary(explicit, dataDir string) (string, error) {
	candidates := []string{explicit, os.Getenv("KNIRV_VLM_LLAMA_SERVER"), filepath.Join(dataDir, "bin", "llama-server")}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	if p, err := exec.LookPath("llama-server"); err == nil {
		return p, nil
	}
	return "", errors.New("llama-server not found: set KNIRV_VLM_LLAMA_SERVER to a CUDA build, or copy the build's bin directory to <data dir>/bin")
}

// hasCUDA reports whether the llama-server binary ships its CUDA backend library. A GPU
// is only used when it does.
func hasCUDA(bin string) bool {
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(bin), "libggml-cuda.so*"))
	return len(matches) > 0
}

// resolveServer returns the llama-server to run. An explicit path (Config.Binary or
// KNIRV_VLM_LLAMA_SERVER) is used as given. Otherwise the build for this host is chosen
// from its resources, reused if it already matches the pinned source, or built. A failed
// CUDA build falls back to the CPU build, so the server still starts.
func (s *Supervisor) resolveServer(ctx context.Context, host Host) (string, error) {
	if p := s.cfg.Binary; p != "" {
		return ResolveBinary(p, s.cfg.DataDir)
	}
	if p := os.Getenv("KNIRV_VLM_LLAMA_SERVER"); p != "" {
		return ResolveBinary("", s.cfg.DataDir)
	}
	nvcc := FindNVCC()
	spec := ChooseBuild(host, nvcc)
	s.setState("building", "llama-server: "+spec.Variant)
	dir, err := EnsureBuild(ctx, s.cfg.DataDir, spec, s.cfg.Logf)
	if err != nil && spec.CUDA {
		s.cfg.Logf("warning: CUDA build failed (%v); falling back to the CPU build", err)
		spec = ChooseBuild(host, "")
		dir, err = EnsureBuild(ctx, s.cfg.DataDir, spec, s.cfg.Logf)
	}
	if err != nil {
		// Last resort: a binary already present or on PATH, so an operator can supply one.
		if p, perr := ResolveBinary("", s.cfg.DataDir); perr == nil {
			s.cfg.Logf("warning: no build for this host (%v); using %s", err, p)
			return p, nil
		}
		return "", err
	}
	return filepath.Join(dir, "llama-server"), nil
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
