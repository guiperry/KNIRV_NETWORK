package transformer

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type WazeroGate struct {
	runtime     wazero.Runtime
	module     api.Module
	compiled   wazero.CompiledModule
	execResults map[string]interface{}
}

func NewWazeroGate() *WazeroGate {
	return &WazeroGate{
		execResults: make(map[string]interface{}),
	}
}

func (wg *WazeroGate) Compile(source string) error {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)

	compiled, err := runtime.CompileModule(ctx, []byte(source))
	if err != nil {
		return fmt.Errorf("failed to compile WASM: %w", err)
	}

	wg.runtime = runtime
	wg.compiled = compiled
	return nil
}

func (wg *WazeroGate) CompileAndRun(source string, args ...string) (map[string]interface{}, error) {
	if err := wg.Compile(source); err != nil {
		return nil, err
	}
	return wg.Run(args...)
}

func (wg *WazeroGate) Run(args ...string) (map[string]interface{}, error) {
	ctx := context.Background()

	if wg.compiled == nil {
		return nil, fmt.Errorf("no compiled module")
	}

	module, err := wg.runtime.InstantiateModule(ctx, wg.compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	wg.module = module

	results := make(map[string]interface{})

	fnDefs := module.ExportedFunctionDefinitions()
	for name := range fnDefs {
		results[name] = nil
	}

	wg.execResults = results
	return results, nil
}

func (wg *WazeroGate) CallFunction(name string, params ...interface{}) (interface{}, error) {
	if wg.module == nil {
		return nil, fmt.Errorf("no module instantiated")
	}

	fnDef := wg.module.ExportedFunctionDefinitions()[name]
	if fnDef == nil {
		return nil, fmt.Errorf("function %s not found", name)
	}

	return fnDef, nil
}

func (wg *WazeroGate) GetResult(key string) interface{} {
	return wg.execResults[key]
}

func (wg *WazeroGate) Close() error {
	if wg.module != nil {
		ctx := context.Background()
		_ = wg.module.Close(ctx)
	}
	if wg.runtime != nil {
		ctx := context.Background()
		return wg.runtime.Close(ctx)
	}
	return nil
}

// CompileFromBinary loads a pre-compiled .wasm binary from path and prepares it for execution.
func (wg *WazeroGate) CompileFromBinary(wasmPath string) error {
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("failed to read wasm binary %s: %w", wasmPath, err)
	}
	ctx := context.Background()
	if wg.runtime == nil {
		wg.runtime = wazero.NewRuntime(ctx)
	}
	compiled, err := wg.runtime.CompileModule(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to compile wasm binary: %w", err)
	}
	wg.compiled = compiled
	return nil
}

func (wg *WazeroGate) Validate(source string) (bool, string) {
	err := wg.Compile(source)
	if err != nil {
		return false, err.Error()
	}

	results, err := wg.Run()
	if err != nil {
		return false, err.Error()
	}

	if len(results) > 0 {
		return true, ""
	}

	return false, "no exports found"
}

// ValidateBinary loads a .wasm binary from path and validates it can be instantiated.
func (wg *WazeroGate) ValidateBinary(wasmPath string) (bool, string) {
	if err := wg.CompileFromBinary(wasmPath); err != nil {
		return false, err.Error()
	}
	results, err := wg.Run()
	if err != nil {
		return false, err.Error()
	}
	if len(results) > 0 {
		return true, ""
	}
	return false, "no exports found"
}

type WazeroPool struct {
	gates  []*WazeroGate
	size  int
	index int
}

func NewWazeroPool(size int) *WazeroPool {
	return &WazeroPool{
		gates: make([]*WazeroGate, size),
		size:  size,
		index: 0,
	}
}

func (wp *WazeroPool) Acquire() *WazeroGate {
	if wp.gates[wp.index] == nil {
		wp.gates[wp.index] = NewWazeroGate()
	}

	gate := wp.gates[wp.index]
	wp.index = (wp.index + 1) % wp.size

	return gate
}

func (wp *WazeroPool) Release(gate *WazeroGate) {
	_ = gate.Close()
}

func (wp *WazeroPool) CloseAll() {
	for _, gate := range wp.gates {
		if gate != nil {
			_ = gate.Close()
		}
	}
}