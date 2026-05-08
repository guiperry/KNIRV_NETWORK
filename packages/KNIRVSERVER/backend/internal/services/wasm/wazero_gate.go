// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package wasm

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WazeroGate wraps the wazero WASM runtime for KNIRVSERVER.  It loads
// pre-compiled .wasm binaries (produced by KNIRVHASHER's TinyGo pipeline)
// from disk and executes their exported functions.
//
// Expected exports by WASM type:
//
//	rules.wasm      → GuardrailClass() uint32
//	resolution.wasm → resolveError() bool, ErrorClass() uint32
//	patch.wasm      → applyPatch() bool (future)
//
// Each gate instance is single-use in the sense that it binds to one .wasm
// module.  For concurrent resolution across multiple DVEs, use WazeroPool.
type WazeroGate struct {
	runtime  wazero.Runtime
	module   api.Module
	compiled wazero.CompiledModule
	wasmPath string
	wasmType WASMType
	mu       sync.Mutex
	closed   bool
}

// NewWazeroGate creates a gate that loads and validates a .wasm binary.
func NewWazeroGate(ctx context.Context, wasmPath string) (*WazeroGate, error) {
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("wazero: read %s: %w", wasmPath, err)
	}

	runtime := wazero.NewRuntime(ctx)
	compiled, err := runtime.CompileModule(ctx, data)
	if err != nil {
		runtime.Close(ctx)
		return nil, fmt.Errorf("wazero: compile %s: %w", wasmPath, err)
	}

	return &WazeroGate{
		runtime:  runtime,
		compiled: compiled,
		wasmPath: wasmPath,
	}, nil
}

// Instantiate finalizes the module so exports can be called.
func (wg *WazeroGate) Instantiate(ctx context.Context) error {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	if wg.closed {
		return fmt.Errorf("wazero: gate closed")
	}
	if wg.module != nil {
		return nil // already instantiated
	}

	module, err := wg.runtime.InstantiateModule(ctx, wg.compiled,
		wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return fmt.Errorf("wazero: instantiate %s: %w", wg.wasmPath, err)
	}
	wg.module = module
	return nil
}

// GuardrailClass calls the GuardrailClass() export and returns the
// guardrail class ID (uint32).  Requires a rules.wasm module.
func (wg *WazeroGate) GuardrailClass(ctx context.Context) (uint32, error) {
	if err := wg.Instantiate(ctx); err != nil {
		return 0, err
	}

	fn := wg.module.ExportedFunction(ExportGuardrailClass)
	if fn == nil {
		return 0, fmt.Errorf("wazero: export %s not found in %s", ExportGuardrailClass, wg.wasmPath)
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return 0, fmt.Errorf("wazero: %s(): %w", ExportGuardrailClass, err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("wazero: %s() returned no values", ExportGuardrailClass)
	}

	// wazero returns uint64 for all integer types; cast down.
	return uint32(results[0]), nil
}

// ResolveError calls the resolveError() export and returns whether
// resolution succeeded.  Requires a resolution.wasm module.
func (wg *WazeroGate) ResolveError(ctx context.Context) (bool, error) {
	if err := wg.Instantiate(ctx); err != nil {
		return false, err
	}

	fn := wg.module.ExportedFunction(ExportResolveError)
	if fn == nil {
		return false, fmt.Errorf("wazero: export %s not found in %s", ExportResolveError, wg.wasmPath)
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return false, fmt.Errorf("wazero: %s(): %w", ExportResolveError, err)
	}
	if len(results) == 0 {
		return false, fmt.Errorf("wazero: %s() returned no values", ExportResolveError)
	}

	return results[0] != 0, nil
}

// ErrorClass calls the ErrorClass() export and returns the error class ID.
// Requires a resolution.wasm module.
func (wg *WazeroGate) ErrorClass(ctx context.Context) (uint32, error) {
	if err := wg.Instantiate(ctx); err != nil {
		return 0, err
	}

	fn := wg.module.ExportedFunction(ExportErrorClass)
	if fn == nil {
		return 0, fmt.Errorf("wazero: export %s not found in %s", ExportErrorClass, wg.wasmPath)
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return 0, fmt.Errorf("wazero: %s(): %w", ExportErrorClass, err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("wazero: %s() returned no values", ExportErrorClass)
	}

	return uint32(results[0]), nil
}

// ApplyPatch calls the applyPatch() export (future patch.wasm support).
func (wg *WazeroGate) ApplyPatch(ctx context.Context) (bool, error) {
	if err := wg.Instantiate(ctx); err != nil {
		return false, err
	}

	fn := wg.module.ExportedFunction("applyPatch")
	if fn == nil {
		return false, fmt.Errorf("wazero: export applyPatch not found in %s", wg.wasmPath)
	}

	results, err := fn.Call(ctx)
	if err != nil {
		return false, fmt.Errorf("wazero: applyPatch(): %w", err)
	}
	if len(results) == 0 {
		return false, fmt.Errorf("wazero: applyPatch() returned no values")
	}

	return results[0] != 0, nil
}

// ListExports returns the names of all exported functions.
func (wg *WazeroGate) ListExports(ctx context.Context) ([]string, error) {
	if err := wg.Instantiate(ctx); err != nil {
		return nil, err
	}
	defs := wg.module.ExportedFunctionDefinitions()
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	return names, nil
}

// Close releases the wazero runtime and module resources.
func (wg *WazeroGate) Close() error {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	if wg.closed {
		return nil
	}
	wg.closed = true

	if wg.module != nil {
		wg.module.Close(context.Background())
	}
	if wg.runtime != nil {
		return wg.runtime.Close(context.Background())
	}
	return nil
}

// WasmPath returns the source .wasm file path.
func (wg *WazeroGate) WasmPath() string { return wg.wasmPath }

// ── WazeroPool ──────────────────────────────────────────────────────────

// WazeroPool maintains a bounded pool of reusable WazeroGate instances
// for a specific .wasm module.  This avoids re-compilation on every
// guardrail check or resolution attempt.
type WazeroPool struct {
	wasmPath string
	gates    []*WazeroGate
	size     int
	next     int
	mu       sync.Mutex
}

// NewWazeroPool creates a pool that lazily instantiates up to size gates.
func NewWazeroPool(wasmPath string, size int) *WazeroPool {
	if size < 1 {
		size = 4
	}
	return &WazeroPool{
		wasmPath: wasmPath,
		gates:    make([]*WazeroGate, size),
		size:     size,
	}
}

// Acquire returns a ready-to-use gate from the pool.
func (wp *WazeroPool) Acquire(ctx context.Context) (*WazeroGate, error) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	idx := wp.next
	wp.next = (wp.next + 1) % wp.size

	if wp.gates[idx] == nil {
		gate, err := NewWazeroGate(ctx, wp.wasmPath)
		if err != nil {
			return nil, err
		}
		wp.gates[idx] = gate
	}

	return wp.gates[idx], nil
}

// CloseAll shuts down every gate in the pool.
func (wp *WazeroPool) CloseAll() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	for i, g := range wp.gates {
		if g != nil {
			g.Close()
			wp.gates[i] = nil
		}
	}
}
