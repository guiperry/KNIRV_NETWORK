package security

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"KNIRVGRAPH/internal/drq" // For Solution
)

// ResourceLimits defines resource constraints for sandbox execution
type ResourceLimits struct {
	MaxCPU       float64 // CPU cores
	MaxMemory    uint64  // Bytes
	MaxDiskIO    uint64  // Bytes/sec
	MaxNetworkIO uint64  // Bytes/sec
	MaxProcesses int
}

// ExecutionResult captures the outcome of a sandboxed execution
type ExecutionResult struct {
	Output           []byte
	Error            error
	MemoryPeak       uint64
	InstructionCount uint64
	CPUTimeUsed      time.Duration
	NetworkBytes     uint64
	DiskBytes        uint64
}

// ContainerInterface defines the interface for an isolated execution container
type ContainerInterface interface {
	Cleanup()
	InjectContext(errorContext []byte) error
	LoadCode(codePackage string) error
	Execute(result *ExecutionResult) error
	Terminate()
}

// SafeExecutionSandbox isolates solution execution
type SafeExecutionSandbox struct {
	resourceLimits  ResourceLimits
	timeLimit       time.Duration
	memoryLimit     uint64
	instructionCap  uint64
	retrieval       RetrievalContextProvider
	maxContextBytes int
}

type RetrievalContextProvider interface {
	Process(context.Context, types.QueryRequest) (*types.QueryResponse, error)
}

func (ses *SafeExecutionSandbox) SetRetrievalContextProvider(provider RetrievalContextProvider, maxBytes int) {
	ses.retrieval = provider
	if maxBytes <= 0 {
		maxBytes = 64 << 10
	}
	ses.maxContextBytes = maxBytes
}

// ExecuteSolutionWithRetrieval obtains bounded context in the trusted host and
// injects it into the sandbox. Skills never receive network credentials or
// direct access to the retrieval service.
func (ses *SafeExecutionSandbox) ExecuteSolutionWithRetrieval(ctx context.Context, solution *drq.Solution, errorContext []byte) (*ExecutionResult, error) {
	if ses.retrieval == nil {
		return ses.ExecuteSolution(solution, errorContext)
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := ses.retrieval.Process(queryCtx, types.QueryRequest{Query: string(errorContext), TopK: 5, UseHybrid: true})
	if err != nil {
		return nil, fmt.Errorf("retrieve skill context: %w", err)
	}
	raw, err := json.Marshal(map[string]interface{}{"error": string(errorContext), "retrieval": resp.Results, "sources": resp.Sources})
	if err != nil {
		return nil, err
	}
	limit := ses.maxContextBytes
	if limit <= 0 {
		limit = 64 << 10
	}
	if len(raw) > limit {
		raw = raw[:limit]
	}
	return ses.ExecuteSolution(solution, raw)
}

// ExecuteSolution runs code in isolated environment
func (ses *SafeExecutionSandbox) ExecuteSolution(
	solution *drq.Solution, // Use drq.Solution
	errorContext []byte,
) (*ExecutionResult, error) {
	// Create isolated container (stub)
	container := ses.createContainer()
	defer container.Cleanup()

	// Inject error context
	err := container.InjectContext(errorContext)
	if err != nil {
		return nil, fmt.Errorf("failed to inject context: %w", err)
	}

	// Load solution code
	err = container.LoadCode(solution.CodePackage)
	if err != nil {
		return nil, fmt.Errorf("failed to load code: %w", err)
	}

	// Execute with monitoring
	result := &ExecutionResult{}

	executionChan := make(chan error)
	go func() {
		executionChan <- container.Execute(result)
	}()

	// Enforce timeout
	select {
	case err := <-executionChan:
		if err != nil {
			return nil, err
		}
	case <-time.After(ses.timeLimit):
		container.Terminate()
		return nil, errors.New("execution timeout")
	}

	// Validate resource usage
	if result.MemoryPeak > ses.memoryLimit {
		return nil, errors.New("memory limit exceeded")
	}

	if result.InstructionCount > ses.instructionCap {
		return nil, errors.New("instruction limit exceeded")
	}

	return result, nil
}

// createContainer is a stub for creating an isolated container
func (ses *SafeExecutionSandbox) createContainer() ContainerInterface {
	// TODO: Implement actual container creation logic (e.g., WASM runtime, Docker, seccomp)
	return &StubContainer{}
}

// StubContainer is a mock implementation of ContainerInterface for testing
type StubContainer struct{}

func (sc *StubContainer) Cleanup() {
	fmt.Println("StubContainer: Cleanup called")
}

func (sc *StubContainer) InjectContext(errorContext []byte) error {
	fmt.Printf("StubContainer: InjectContext called with %d bytes\n", len(errorContext))
	return nil
}

func (sc *StubContainer) LoadCode(codePackage string) error {
	fmt.Printf("StubContainer: LoadCode called with %s\n", codePackage)
	return nil
}

func (sc *StubContainer) Execute(result *ExecutionResult) error {
	fmt.Println("StubContainer: Execute called")
	// Simulate some execution
	time.Sleep(10 * time.Millisecond)
	result.MemoryPeak = 100 * 1024 * 1024 // 100MB
	result.InstructionCount = 1000000
	return nil
}

func (sc *StubContainer) Terminate() {
	fmt.Println("StubContainer: Terminate called")
}
