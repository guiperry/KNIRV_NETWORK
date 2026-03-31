package vault

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"backend_server/internal/storage/pqc"

	"github.com/robertkrimen/otto"
)

// ComplianceScriptExecutor runs guardrail rules as compliance checks
type ComplianceScriptExecutor struct {
	solutionExecutor *SolutionExecutor
	enabledLanguages map[string]bool
	defaultTimeout   time.Duration
}

// NewComplianceScriptExecutor creates a new compliance executor
func NewComplianceScriptExecutor(validator *pqc.SolutionNodeValidator) *ComplianceScriptExecutor {
	return &ComplianceScriptExecutor{
		solutionExecutor: NewSolutionExecutor(validator),
		enabledLanguages: map[string]bool{
			"javascript": true,
			"python":     false, // Disabled - requires sandbox
			"shell":      false, // Disabled - security risk
			"rego":       false, // Open Policy Agent - not implemented
			"cel":        false, // Common Expression Language - not implemented
		},
		defaultTimeout: 5 * time.Second,
	}
}

// SetLanguageEnabled enables or disables a language
func (e *ComplianceScriptExecutor) SetLanguageEnabled(language string, enabled bool) {
	e.enabledLanguages[language] = enabled
}

// IsLanguageEnabled checks if a language is enabled
func (e *ComplianceScriptExecutor) IsLanguageEnabled(language string) bool {
	return e.enabledLanguages[strings.ToLower(language)]
}

// ExecuteGuardrail runs a single guardrail rule
func (e *ComplianceScriptExecutor) ExecuteGuardrail(
	ctx context.Context,
	guardrail *GuardrailRule,
	testData map[string]interface{},
	agentOutput string,
) (*GuardrailResult, error) {
	start := time.Now()

	result := &GuardrailResult{
		GuardrailID:    guardrail.ID,
		GuardrailName:  guardrail.Name,
		Status:         "pending",
		ExpectedResult: guardrail.ExpectedResult,
	}

	// Check language support
	if !e.IsLanguageEnabled(guardrail.Language) {
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("language '%s' is not enabled", guardrail.Language)
		result.ExecutionTime = time.Since(start).Milliseconds()
		return result, nil
	}

	// Set timeout
	timeout := e.defaultTimeout
	if guardrail.TimeoutSec > 0 {
		timeout = time.Duration(guardrail.TimeoutSec) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute based on language
	var actualResult interface{}
	var err error

	switch strings.ToLower(guardrail.Language) {
	case "javascript":
		actualResult, err = e.executeJavaScript(ctx, guardrail.Code, testData, agentOutput)
	case "python":
		actualResult, err = e.executePython(ctx, guardrail.Code, testData, agentOutput)
	case "shell", "bash":
		actualResult, err = e.executeShell(ctx, guardrail.Code, testData, agentOutput)
	default:
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("unsupported language: %s", guardrail.Language)
		result.ExecutionTime = time.Since(start).Milliseconds()
		return result, nil
	}

	result.ExecutionTime = time.Since(start).Milliseconds()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "timeout"
			result.ErrorMessage = "execution timed out"
		} else {
			result.Status = "error"
			result.ErrorMessage = err.Error()
		}
		return result, nil
	}

	result.ActualResult = actualResult

	// Compare result with expected
	if e.resultsMatch(actualResult, guardrail.ExpectedResult) {
		result.Status = "passed"
	} else {
		result.Status = "failed"
	}

	return result, nil
}

// ExecuteScenario runs all guardrails for a scenario
func (e *ComplianceScriptExecutor) ExecuteScenario(
	ctx context.Context,
	scenario *RegulatoryScenario,
	agentOutput string,
) (*ScenarioResult, error) {
	start := time.Now()

	result := &ScenarioResult{
		ScenarioID:       scenario.ID,
		ExecutionID:      generateExecutionID(),
		Status:           "running",
		StartedAt:        start,
		GuardrailResults: make([]GuardrailResult, 0),
	}

	// Execute each guardrail
	allPassed := true
	var guardrailScore float64

	for _, guardrail := range scenario.Guardrails {
		guardrailResult, err := e.ExecuteGuardrail(ctx, &guardrail, scenario.TestData, agentOutput)
		if err != nil {
			// Log error but continue
			continue
		}

		result.GuardrailResults = append(result.GuardrailResults, *guardrailResult)

		if guardrailResult.Status != "passed" {
			allPassed = false
			if guardrail.IsMandatory {
				// Mandatory guardrail failure is critical
				result.Status = "failed"
			}
		} else {
			guardrailScore += 1.0
		}
	}

	// Calculate guardrail score
	if len(scenario.Guardrails) > 0 {
		result.GuardrailScore = (guardrailScore / float64(len(scenario.Guardrails))) * 100
	}

	// Update final status
	if result.Status != "failed" {
		if allPassed {
			result.Status = "passed"
		} else {
			result.Status = "failed"
		}
	}

	result.CompletedAt = time.Now()
	result.DurationMs = result.CompletedAt.Sub(start).Milliseconds()
	result.AgentOutput = agentOutput

	return result, nil
}

// executeJavaScript runs JavaScript code using Otto VM
func (e *ComplianceScriptExecutor) executeJavaScript(
	ctx context.Context,
	code string,
	testData map[string]interface{},
	agentOutput string,
) (interface{}, error) {
	vm := otto.New()

	// Set up the context
	errChan := make(chan error, 1)
	resultChan := make(chan interface{}, 1)

	go func() {
		// Inject test data as variables
		for key, value := range testData {
			if err := vm.Set(key, value); err != nil {
				errChan <- fmt.Errorf("failed to set variable %s: %w", key, err)
				return
			}
		}

		// Inject agent output
		if err := vm.Set("agentResponse", agentOutput); err != nil {
			errChan <- fmt.Errorf("failed to set agentResponse: %w", err)
			return
		}

		// Execute code - wrap in self-invoking function if it contains return
		execCode := code
		if strings.Contains(strings.TrimSpace(code), "return") {
			execCode = "(function() { " + code + " })()"
		}

		// Execute code
		result, err := vm.Run(execCode)
		if err != nil {
			errChan <- err
			return
		}

		// Extract result
		val, err := result.Export()
		if err != nil {
			errChan <- err
			return
		}

		resultChan <- val
	}()

	select {
	case <-ctx.Done():
		vm.Interrupt <- func() {}
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}

// executePython runs Python code (placeholder - requires sandbox)
func (e *ComplianceScriptExecutor) executePython(
	_ context.Context,
	_ string,
	_ map[string]interface{},
	_ string,
) (interface{}, error) {
	// Python execution requires a secure sandbox (e.g., gVisor, Firecracker)
	// This is a placeholder implementation
	return nil, fmt.Errorf("python execution requires sandbox (not implemented)")
}

// executeShell runs shell code (disabled by default for security)
func (e *ComplianceScriptExecutor) executeShell(
	ctx context.Context,
	code string,
	testData map[string]interface{},
	agentOutput string,
) (interface{}, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", code)

	// Set environment variables for test data
	for key, value := range testData {
		if strVal, ok := value.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strVal))
		}
	}

	// Set agent output
	cmd.Env = append(cmd.Env, fmt.Sprintf("AGENT_RESPONSE=%s", agentOutput))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("shell execution failed: %w - output: %s", err, string(output))
	}

	// Parse output as boolean or string
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "true" || outputStr == "1" {
		return true, nil
	}
	if outputStr == "false" || outputStr == "0" {
		return false, nil
	}

	return outputStr, nil
}

// resultsMatch compares actual and expected results
func (e *ComplianceScriptExecutor) resultsMatch(actual, expected interface{}) bool {
	// Handle nil cases
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}

	// Type-specific comparisons
	switch exp := expected.(type) {
	case bool:
		if act, ok := actual.(bool); ok {
			return act == exp
		}
	case string:
		if act, ok := actual.(string); ok {
			return act == exp
		}
	case int:
		switch act := actual.(type) {
		case int:
			return act == exp
		case int64:
			return int(act) == exp
		case float64:
			return int(act) == exp
		}
	case int64:
		switch act := actual.(type) {
		case int:
			return int64(act) == exp
		case int64:
			return act == exp
		case float64:
			return int64(act) == exp
		}
	case float64:
		switch act := actual.(type) {
		case float64:
			return act == exp
		case int:
			return float64(act) == exp
		case int64:
			return float64(act) == exp
		}
	}

	// Default to string comparison
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

// generateExecutionID creates a unique execution identifier
func generateExecutionID() string {
	return fmt.Sprintf("exec-%d-%s", time.Now().Unix(), randomString(8))
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
