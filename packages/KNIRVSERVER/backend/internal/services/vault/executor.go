package vault

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"backend_server/internal/storage/pqc"
)

// SolutionExecutor handles the execution of logic found in SolutionNodes
type SolutionExecutor struct {
	validator *pqc.SolutionNodeValidator
}

// NewSolutionExecutor creates a new executor with security validation
func NewSolutionExecutor(validator *pqc.SolutionNodeValidator) *SolutionExecutor {
	return &SolutionExecutor{
		validator: validator,
	}
}

// Execute runs the code block within a SolutionNode after verifying its PQC signature
func (e *SolutionExecutor) Execute(ctx context.Context, sol *SolutionNode, params map[string]interface{}) (string, error) {
	// Security Enforcement: Verify Dilithium-3 signature before execution
	if e.validator != nil && e.validator.EnforcePQCSigning() {
		log.Printf("SolutionExecutor: Verifying PQC signature for node %s", sol.ID)
		if err := e.validator.ValidateAndEnforce(sol.ID); err != nil {
			return "", fmt.Errorf("PQC security enforcement failed: %w", err)
		}
		log.Printf("SolutionExecutor: PQC signature verified for node %s", sol.ID)
	}

	switch strings.ToLower(sol.Language) {
	case "shell", "bash":
		return e.executeShell(ctx, sol.Code, params)
	case "go":
		return "", fmt.Errorf("go execution requires dynamic compilation (implementation pending)")
	default:
		return "", fmt.Errorf("unsupported language: %s", sol.Language)
	}
}

func (e *SolutionExecutor) executeShell(ctx context.Context, code string, _ map[string]interface{}) (string, error) {
	// VERY DANGEROUS - In production this should be sandboxed (e.g., using the Nexus CDE/eBPF)
	// For this prototype, we'll execute it with strict context timeout
	
	cmd := exec.CommandContext(ctx, "bash", "-c", code)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("shell execution failed: %w", err)
	}
	
	return string(output), nil
}
