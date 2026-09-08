// Package runtime manages the child llama-server process.
package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Server struct{ Command *exec.Cmd }

// Options controls the deterministic tunables passed to llama-server.
// Zero values are treated as "let llama-server choose" (except Parallel, which
// defaults to 1 to keep the embedded CPU model uncontended — see the llama
// cognitive-engine plan §2.1, §4 Phase A).
type Options struct {
	Parallel int
	CtxSize  int
	Threads  int
	APIKey   string
}

func (o Options) args(model, port string) []string {
	args := []string{"-m", model, "--host", "127.0.0.1", "--port", port}
	if o.Parallel > 0 {
		args = append(args, "--parallel", strconv.Itoa(o.Parallel))
	}
	if o.CtxSize > 0 {
		args = append(args, "--ctx-size", strconv.Itoa(o.CtxSize))
	}
	if o.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(o.Threads))
	}
	if o.APIKey != "" {
		args = append(args, "--api-key", o.APIKey)
	}
	return args
}

// Args returns the fully-resolved llama-server command line for the supplied
// model + port, applying any non-zero Options. Exported so the manager layer
// (and tests) can assert the exact flag set without spawning a process.
func Args(path, model, address string, opts Options) []string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		port = address
	}
	return append([]string{path}, opts.args(model, port)...)
}

func Healthy(ctx context.Context, address string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/health", nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// Start launches llama-server with the supplied tunables. Pass Options{} to
// preserve the previous default behaviour; the caller is responsible for
// deciding whether to override Parallel/CtxSize/Threads/APIKey (the manager
// does this from ManagerConfig).
func Start(path, model, address string, opts Options) (*Server, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid llama address %q: %w", address, err)
	}
	cmd := exec.Command(path, opts.args(model, port)...)
	// Keep llama-server diagnostics in the KNIRVSERVER log stream. These logs
	// explain model-load failures and make first-run progress observable.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	return &Server{Command: cmd}, nil
}

func (s *Server) Stop() {
	if s == nil || s.Command == nil || s.Command.Process == nil {
		return
	}
	// llama-server receives SIGINT when KNIRVLLAMA is asked to stop, matching
	// an operator interrupt and allowing it to finish its own shutdown path.
	if err := s.Command.Process.Signal(os.Interrupt); err != nil {
		_ = s.Command.Process.Kill()
		_ = s.Command.Wait()
		return
	}
	done := make(chan error, 1)
	go func() { done <- s.Command.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = s.Command.Process.Kill()
		<-done
	}
}
