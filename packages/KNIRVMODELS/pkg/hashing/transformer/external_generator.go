package transformer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExternalGenerateFn is a function that calls an external LLM to generate
// WASM source code.  The prompt contains the WASM type and inquiry context.
// It must return valid TinyGo source code.
type ExternalGenerateFn func(ctx context.Context, prompt string) (string, error)

// ExternalLLMGenerator uses a provided LLM function or HTTP endpoint to
// generate WASM source code.  This is the bootstrap generator — it
// produces real TinyGo WASM modules via a trained LLM while the
// internal Gorgonite model is still being pre-trained.
type ExternalLLMGenerator struct {
	generateFn ExternalGenerateFn
	httpURL    string
	httpClient *http.Client
}

// NewExternalLLMGenerator creates an external generator that calls the
// given function for each generation request.
func NewExternalLLMGenerator(fn ExternalGenerateFn) *ExternalLLMGenerator {
	return &ExternalLLMGenerator{generateFn: fn}
}

// NewExternalLLMGeneratorWithURL creates an external generator that POSTs
// generation requests to the given HTTP endpoint.  The endpoint receives
// a JSON body with "wasm_type", "prompt" and returns JSON with "source".
func NewExternalLLMGeneratorWithURL(url string) *ExternalLLMGenerator {
	return &ExternalLLMGenerator{
		httpURL: url,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateSource builds a prompt and generates WASM source via the
// configured external LLM.
func (g *ExternalLLMGenerator) GenerateSource(ctx context.Context, wasmType WASMType, inquiry interface{}) (string, error) {
	prompt := BuildWASMGenerationPrompt(wasmType, inquiry)

	if g.generateFn != nil {
		return g.generateFn(ctx, prompt)
	}

	if g.httpURL != "" {
		return g.generateViaHTTP(ctx, prompt, wasmType)
	}

	return "", fmt.Errorf("external generator: no generate function or HTTP URL configured")
}

// generateViaHTTP sends the prompt to the configured HTTP endpoint.
func (g *ExternalLLMGenerator) generateViaHTTP(ctx context.Context, prompt string, wasmType WASMType) (string, error) {
	body := map[string]string{
		"wasm_type": string(wasmType),
		"prompt":    prompt,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("external generator: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.httpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("external generator: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("external generator: HTTP call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("external generator: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("external generator: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("external generator: unmarshal response: %w", err)
	}

	if result.Source == "" {
		return "", fmt.Errorf("external generator: empty source in response")
	}

	return result.Source, nil
}

// BuildWASMGenerationPrompt creates a structured prompt for the external
// LLM to generate a TinyGo WASM module.
func BuildWASMGenerationPrompt(wasmType WASMType, inquiry interface{}) string {
	prompt := fmt.Sprintf(`Generate a TinyGo WASM module for %s policy enforcement.

`, wasmType)

	switch inq := inquiry.(type) {
	case PolicyBadgeInquiry:
		prompt += fmt.Sprintf("Badge: %s (type: %s)\n", inq.Name, inq.BadgeType)
		if len(inq.ValuesSignals) > 0 {
			prompt += fmt.Sprintf("Values: %v\n", inq.ValuesSignals)
		}
		if len(inq.OntologySignals) > 0 {
			prompt += fmt.Sprintf("Ontology: %v\n", inq.OntologySignals)
		}
		if inq.DVEContext != "" {
			prompt += fmt.Sprintf("Context: %s\n", inq.DVEContext)
		}
	case DVEErrorInquiry:
		prompt += fmt.Sprintf("Error: %s - %s\n", inq.ErrorType, inq.ErrorMessage)
		if inq.ErrorContext != "" {
			prompt += fmt.Sprintf("Context: %s\n", inq.ErrorContext)
		}
		if inq.StackTrace != "" {
			prompt += fmt.Sprintf("Stack: %s\n", inq.StackTrace)
		}
		if inq.Metadata != nil {
			prompt += "Metadata:\n"
			for k, v := range inq.Metadata {
				prompt += fmt.Sprintf("  %s: %s\n", k, v)
			}
		}
	case SystemPatchInquiry:
		prompt += fmt.Sprintf("Component: %s\n", inq.ComponentID)
		prompt += fmt.Sprintf("Error code: %s\n", inq.ErrorCode)
		if inq.SystemState != "" {
			prompt += fmt.Sprintf("System state: %s\n", inq.SystemState)
		}
	}

	prompt += "\nRequirements:\n"
	prompt += "- The module MUST be valid TinyGo source compilable with \"tinygo build -target wasm\"\n"
	prompt += "- Use //export directives for all exported functions\n"
	prompt += "- Include func main() {} at the end\n"

	switch wasmType {
	case WASMTypeRule:
		prompt += `- Export: GuardrailClass() uint32 — return 0 to allow, non-zero to block
- Export: PolicyID() uint32
- Export: Severity() uint32
- Export: Action() uint32
- Export: Confidence() float64`
	case WASMTypeResolution:
		prompt += `- Export: ErrorClass() uint32
- Export: ErrorCode() uint32
- Export: Retryable() bool
- export resolveError() bool — return true if this error can be resolved`
	case WASMTypePatch:
		prompt += `- Export: PatchScope() uint32
- Export: TargetComponent() uint32
- var patchApplied bool
- export applyPatch() bool — attempt the patch and return success/failure`
	}

	prompt += "\n\nReturn ONLY the compilable TinyGo source code, no explanation."
	return prompt
}

// WASMGenerationRequest is the JSON body sent to the external generator HTTP endpoint.
type WASMGenerationRequest struct {
	WASMType string `json:"wasm_type"`
	Prompt   string `json:"prompt"`
}

// WASMGenerationResponse is the expected JSON response from the external generator.
type WASMGenerationResponse struct {
	Source string `json:"source"`
}
