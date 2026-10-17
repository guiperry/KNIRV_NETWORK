package knirvmodels

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// InferenceBinaryName is the file the inference instance runs from. It differs from
// "knirvmodels" on purpose; see ExtractEmbeddedBinaryAs.
const InferenceBinaryName = "knirvmodels-infer"

// InferenceConfig describes the instance of the knirvmodels binary that serves the
// pre-trained models: Colibri as the primary harness, llama.cpp as its fallback, both
// behind SocketPath.
type InferenceConfig struct {
	// BinaryPath runs a specific binary; empty extracts the embedded one into BinDir.
	BinaryPath string
	BinDir     string
	// SocketPath is the one inference socket callers use (llama.sock).
	SocketPath string
	// Env is added to the child's environment (model selection, thread counts, ...).
	Env []string
	// Stdout and Stderr receive the child's output. Nil means the process's own.
	Stdout, Stderr io.Writer
	// RestartBackoff is the wait before restart 1, 2, ... (default 2s, 10s, 30s, 2m; the
	// last repeats). It starts over once a child has stayed up for StableAfter.
	RestartBackoff []time.Duration
	StableAfter    time.Duration
	// StopTimeout is how long a stop waits for the child to shut down cleanly before
	// killing it (default 45s: Colibri has to stop its engine first).
	StopTimeout time.Duration
	Logf        func(format string, args ...any)
}

// Inference keeps the inference instance of the knirvmodels binary running.
type Inference struct {
	cfg InferenceConfig

	mu  sync.Mutex
	cmd *exec.Cmd
}

// NewInference fills defaults.
func NewInference(cfg InferenceConfig) *Inference {
	if cfg.RestartBackoff == nil {
		cfg.RestartBackoff = []time.Duration{2 * time.Second, 10 * time.Second, 30 * time.Second, 2 * time.Minute}
	}
	if cfg.StableAfter <= 0 {
		cfg.StableAfter = 2 * time.Minute
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 45 * time.Second
	}
	if cfg.Logf == nil {
		cfg.Logf = func(f string, a ...any) { log.Printf("KNIRVMODELS inference: "+f, a...) }
	}
	return &Inference{cfg: cfg}
}

// SocketPath is the inference socket.
func (i *Inference) SocketPath() string { return i.cfg.SocketPath }

// Run starts the inference instance and restarts it whenever it exits, until ctx is
// cancelled; it then stops the child and returns. A child that cannot even be started
// (missing binary) ends Run with that error, since retrying will not help.
func (i *Inference) Run(ctx context.Context) error {
	binary := i.cfg.BinaryPath
	if binary == "" {
		var err error
		binary, err = ExtractEmbeddedBinaryAs(i.cfg.BinDir, InferenceBinaryName)
		if err != nil {
			return fmt.Errorf("extract embedded knirvmodels binary: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(i.cfg.SocketPath), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}

	failures := 0
	for {
		began := time.Now()
		exited, err := i.runOnce(ctx, binary)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		if time.Since(began) >= i.cfg.StableAfter {
			failures = 0
		}
		failures++
		wait := i.backoff(failures)
		i.cfg.Logf("inference process exited (%v); restarting in %s (restart %d)", exited, wait, failures)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (i *Inference) backoff(n int) time.Duration {
	s := i.cfg.RestartBackoff
	if len(s) == 0 {
		return 0
	}
	if n-1 < len(s) {
		return s[n-1]
	}
	return s[len(s)-1]
}

// runOnce runs one child to completion. err is non-nil only when the child could not
// be started at all; its exit status comes back as exited.
func (i *Inference) runOnce(ctx context.Context, binary string) (exited error, err error) {
	cmd := exec.Command(binary, "--inference", "--inference-socket", i.cfg.SocketPath)
	cmd.Env = append(os.Environ(), i.cfg.Env...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if i.cfg.Stdout != nil {
		cmd.Stdout = i.cfg.Stdout
	}
	if i.cfg.Stderr != nil {
		cmd.Stderr = i.cfg.Stderr
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", binary, err)
	}
	i.mu.Lock()
	i.cmd = cmd
	i.mu.Unlock()
	i.cfg.Logf("inference process started (pid %d)", cmd.Process.Pid)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case werr := <-done:
		i.clear(cmd)
		return werr, nil
	case <-ctx.Done():
		i.stop(cmd, done)
		i.clear(cmd)
		return nil, nil
	}
}

func (i *Inference) clear(cmd *exec.Cmd) {
	i.mu.Lock()
	if i.cmd == cmd {
		i.cmd = nil
	}
	i.mu.Unlock()
}

// stop asks the child's process group to shut down and kills it after StopTimeout.
func (i *Inference) stop(cmd *exec.Cmd, done <-chan error) {
	pgid := -cmd.Process.Pid
	syscall.Kill(pgid, syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(i.cfg.StopTimeout):
		i.cfg.Logf("inference process did not stop in %s; killing it", i.cfg.StopTimeout)
		syscall.Kill(pgid, syscall.SIGKILL)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}

// PID is the running child's process id, or 0.
func (i *Inference) PID() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cmd != nil && i.cmd.Process != nil {
		return i.cmd.Process.Pid
	}
	return 0
}
