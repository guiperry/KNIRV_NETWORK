// Package head runs the knirv-head Hierarchical Reasoning Model server-side.
//
// The model is the same Rust crate the arena runs in the browser, compiled to
// WASI (head-wasi.wasm, built by `make build-head-wasi`) and executed under
// wazero, so server and browser answers agree to float tolerance. The module
// speaks one JSON object per line over stdin/stdout.
package head

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed head-wasi.wasm
var wasiModule []byte

// Info describes the loaded model.
type Info struct {
	Loaded          bool   `json:"loaded"`
	TotalParameters int    `json:"total_parameters"`
	LModules        int    `json:"l_modules"`
	HModules        int    `json:"h_modules"`
	MaxInputLen     int    `json:"max_input_len"`
	Tokenizer       string `json:"tokenizer"`
	ModelVersion    string `json:"model_version"`
	VocabSize       int    `json:"vocab_size"`
}

// Runner owns one WASI instance. Calls are serialised: the module is single-threaded.
type Runner struct {
	mu      sync.Mutex
	rt      wazero.Runtime
	in      *io.PipeWriter
	out     *bufio.Reader
	outPipe *io.PipeReader
	done    chan error
	dead    error
}

type response struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

// NewRunner starts the module with modelDir mounted read-only at /m.
func NewRunner(ctx context.Context, modelDir string) (*Runner, error) {
	rt := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, err
	}
	compiled, err := rt.CompileModule(ctx, wasiModule)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compile head-wasi: %w", err)
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	cfg := wazero.NewModuleConfig().
		WithStdin(inR).WithStdout(outW).WithStderr(io.Discard).
		WithFSConfig(wazero.NewFSConfig().WithReadOnlyDirMount(modelDir, "/m")).
		WithStartFunctions("_start")
	r := &Runner{rt: rt, in: inW, out: bufio.NewReader(outR), outPipe: outR, done: make(chan error, 1)}
	go func() {
		_, err := rt.InstantiateModule(ctx, compiled, cfg)
		outW.CloseWithError(errors.New("head module exited"))
		r.done <- err
	}()
	return r, nil
}

func (r *Runner) call(req any, into any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dead != nil {
		return r.dead
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := r.in.Write(append(b, '\n')); err != nil {
		r.dead = fmt.Errorf("head module unavailable: %w", err)
		return r.dead
	}
	line, err := r.out.ReadBytes('\n')
	if err != nil {
		r.dead = fmt.Errorf("head module unavailable: %w", err)
		return r.dead
	}
	var resp response
	if err := json.Unmarshal(bytes.TrimSpace(line), &resp); err != nil {
		return fmt.Errorf("bad head response: %w", err)
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	if into != nil {
		return json.Unmarshal(resp.Result, into)
	}
	return nil
}

// Load reads config and weights (file names relative to the model dir).
func (r *Runner) Load(configFile, weightsFile string) (Info, error) {
	var info Info
	err := r.call(map[string]string{"op": "load", "config_path": "/m/" + configFile, "weights_path": "/m/" + weightsFile}, &info)
	return info, err
}

// Reason returns the raw Reasoning JSON for a token sequence.
func (r *Runner) Reason(tokens []uint32) (json.RawMessage, error) {
	var raw json.RawMessage
	err := r.call(map[string]any{"op": "reason", "tokens": tokens}, &raw)
	return raw, err
}

func (r *Runner) Close(ctx context.Context) error {
	r.in.Close()
	r.outPipe.Close()
	return r.rt.Close(ctx)
}
