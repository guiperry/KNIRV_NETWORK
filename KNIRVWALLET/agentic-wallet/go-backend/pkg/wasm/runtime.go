package wasm

import (
	"context"
	"crypto-wallet-backend/internal/config"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Runtime struct {
	config *config.Config
}

type ExecutionLimits struct {
	MemoryLimit    int64
	CPULimit       float64
	NetworkTimeout time.Duration
	Permissions    []string
}

type ExecutionResult struct {
	Output      interface{}
	MemoryUsed  int64
	CPUUsed     float64
	Duration    time.Duration
}

func NewRuntime(cfg *config.Config) *Runtime {
	return &Runtime{
		config: cfg,
	}
}

func (r *Runtime) ValidateCode(code []byte) (string, error) {
	// Validate WASM bytecode
	_, err := wasmer.NewModule(wasmer.NewStore(wasmer.NewEngine()), code)
	if err != nil {
		return "", fmt.Errorf("invalid WASM bytecode: %w", err)
	}

	// Generate hash
	hash := sha256.Sum256(code)
	return hex.EncodeToString(hash[:]), nil
}

func (r *Runtime) Execute(ctx context.Context, code []byte, input map[string]interface{}, limits *ExecutionLimits) (*ExecutionResult, error) {
	startTime := time.Now()
	
	// Create WASM engine and store
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compile module
	module, err := wasmer.NewModule(store, code)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	// Create import object with limited capabilities
	importObject := r.createImportObject(store, limits)

	// Instantiate module
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	// Get the main function
	mainFunc, err := instance.Exports.GetFunction("main")
	if err != nil {
		return nil, fmt.Errorf("main function not found: %w", err)
	}

	// Execute with timeout
	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("WASM execution panic: %v", r)
			}
		}()

		// Execute the function
		result, err := mainFunc()
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		duration := time.Since(startTime)
		return &ExecutionResult{
			Output:     result,
			MemoryUsed: r.getMemoryUsage(instance),
			CPUUsed:    duration.Seconds(),
			Duration:   duration,
		}, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(limits.NetworkTimeout):
		return nil, fmt.Errorf("execution timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Runtime) createImportObject(store *wasmer.Store, limits *ExecutionLimits) *wasmer.ImportObject {
	importObject := wasmer.NewImportObject()

	// Add limited system functions based on permissions
	if r.hasPermission(limits.Permissions, "network_access") {
		// Add network functions
		r.addNetworkFunctions(store, importObject, limits)
	}

	if r.hasPermission(limits.Permissions, "storage_access") {
		// Add storage functions
		r.addStorageFunctions(store, importObject, limits)
	}

	if r.hasPermission(limits.Permissions, "read_market_data") {
		// Add market data functions
		r.addMarketDataFunctions(store, importObject, limits)
	}

	return importObject
}

func (r *Runtime) hasPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (r *Runtime) addNetworkFunctions(store *wasmer.Store, importObject *wasmer.ImportObject, limits *ExecutionLimits) {
	// Add HTTP request function with restrictions
	httpGet := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32),
		),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			// Implement restricted HTTP GET
			// This would include URL validation, rate limiting, etc.
			return []wasmer.Value{wasmer.NewI32(0)}, nil
		},
	)

	importObject.Register("env", map[string]wasmer.IntoExtern{
		"http_get": httpGet,
	})
}

func (r *Runtime) addStorageFunctions(store *wasmer.Store, importObject *wasmer.ImportObject, limits *ExecutionLimits) {
	// Add storage functions with quota limits
	// Implementation would include key-value storage with size limits
}

func (r *Runtime) addMarketDataFunctions(store *wasmer.Store, importObject *wasmer.ImportObject, limits *ExecutionLimits) {
	// Add market data access functions
	// Implementation would provide access to real-time price data
}

func (r *Runtime) getMemoryUsage(instance *wasmer.Instance) int64 {
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return 0
	}
	return int64(memory.DataSize())
}