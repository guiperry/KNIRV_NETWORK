package vault

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SolutionExecutor handles the execution of logic found in SolutionNodes
type SolutionExecutor struct {
	// Add config for allowed languages/paths
}

// NewSolutionExecutor creates a new executor
func NewSolutionExecutor() *SolutionExecutor {
	return &SolutionExecutor{}
}

// Execute runs the code block within a SolutionNode
func (e *SolutionExecutor) Execute(ctx context.Context, sol *SolutionNode, params map[string]interface{}) (string, error) {
	switch strings.ToLower(sol.Language) {
	case "shell", "bash":
		return e.executeShell(ctx, sol.Code, params)
	case "go":
		// For Go, in a real scenario we might compile or use an interpreter
		// For now, we'll demonstrate a placeholder
		return "", fmt.Errorf("go execution requires dynamic compilation (implementation pending)")
	default:
		return "", fmt.Errorf("unsupported language: %s", sol.Language)
	}
}

func (e *SolutionExecutor) executeShell(ctx context.Context, code string, params map[string]interface{}) (string, error) {
	// VERY DANGEROUS - In production this should be sandboxed (e.g., using the Nexus CDE/eBPF)
	// For this prototype, we'll execute it with strict context timeout
	
	cmd := exec.CommandContext(ctx, "bash", "-c", code)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("shell execution failed: %w", err)
	}
	
	return string(output), nil
}
