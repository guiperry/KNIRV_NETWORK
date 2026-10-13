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
	"strings"
	"time"
)

type Server struct {
	Command *exec.Cmd
	exited  chan struct{} // closed when the process has been reaped (one Wait, owned here)
}

// Exited is closed once the process has ended.
func (s *Server) Exited() <-chan struct{} { return s.exited }

// LoRAAdapter is an adapter to load alongside the base model.
//
// This is what lets a DVE actually execute a minted uLoRA: the bundle's
// projected weights are projected onto the target architecture by KNIRVULORA's
// binder and written as a llama.cpp-compatible adapter, which llama-server can
// then load at startup. Without this the adapter exists on chain but there is no
// way to run a model with it applied.
type LoRAAdapter struct {
	// Path is the adapter file on disk.
	Path string
	// Scale overrides the adapter's scale. Zero means "use llama-server's
	// default", passed as a plain --lora rather than --lora-scaled.
	Scale float64
}

// Options controls the deterministic tunables passed to llama-server.
// Zero values are treated as "let llama-server choose" (except Parallel, which
// defaults to 1 to keep the embedded CPU model uncontended — see the llama
// cognitive-engine plan §2.1, §4 Phase A).
type Options struct {
	Parallel int
	CtxSize  int
	Threads  int
	APIKey   string
	// LoRA lists adapters to apply, in order.
	LoRA []LoRAAdapter
	// LoRAInitWithoutApply passes --lora-init-without-apply. NOTE: against the
	// provisioned llama.cpp (9e0e220) a request WITHOUT a `lora` field still runs
	// every loaded adapter at its default scale (GET /lora-adapters reports 1.0),
	// so this flag alone does not make adapters inert. Callers must also call
	// ZeroLoRAAdapters after start and send an explicit `lora` list per request.
	LoRAInitWithoutApply bool
}

// Validate rejects options llama-server would abort on, before it is started.
func (o Options) Validate() error {
	for _, a := range o.LoRA {
		if strings.TrimSpace(a.Path) == "" {
			continue
		}
		if a.Scale > 0 && strings.Contains(a.Path, ":") {
			return fmt.Errorf("adapter path %q contains ':', which --lora-scaled uses to separate the scale", a.Path)
		}
	}
	return nil
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
	// Adapters last so the base-model flags above stay stable in the argv the
	// manager asserts against.
	hasAdapter := false
	for _, adapter := range o.LoRA {
		path := strings.TrimSpace(adapter.Path)
		if path == "" {
			// An empty path must be skipped, not emitted: llama-server would
			// consume the following flag as the filename.
			continue
		}
		hasAdapter = true
		if adapter.Scale > 0 {
			// llama-server takes FNAME:SCALE as ONE argument (verified against the
			// provisioned build); "--lora-scaled path scale" aborts at startup.
			// Options.Validate refuses a ':' in the path; the scale is the last field.
			args = append(args, "--lora-scaled", path+":"+strconv.FormatFloat(adapter.Scale, 'g', -1, 64))
			continue
		}
		args = append(args, "--lora", path)
	}
	// Only meaningful alongside at least one adapter; emitting it alone would
	// imply an intent the command line cannot express.
	if o.LoRAInitWithoutApply && hasAdapter {
		args = append(args, "--lora-init-without-apply")
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
	srv := &Server{Command: cmd, exited: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(srv.exited) }()
	return srv, nil
}

func (s *Server) Stop() {
	if s == nil || s.Command == nil || s.Command.Process == nil {
		return
	}
	select {
	case <-s.exited:
		return
	default:
	}
	// llama-server receives SIGINT when KNIRVLLAMA is asked to stop, matching
	// an operator interrupt and allowing it to finish its own shutdown path.
	if err := s.Command.Process.Signal(os.Interrupt); err != nil {
		_ = s.Command.Process.Kill()
		<-s.exited
		return
	}
	select {
	case <-s.exited:
	case <-time.After(10 * time.Second):
		_ = s.Command.Process.Kill()
		<-s.exited
	}
}
