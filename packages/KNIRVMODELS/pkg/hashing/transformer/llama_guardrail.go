package transformer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClaimVerification is the non-blocking result attached to a local-model
// claim. Math verification is deterministic when the claim is mathematical;
// ledger grounding is best-effort and never stops text generation.
type ClaimVerification struct {
	MathVerified bool               `json:"math_verified"`
	MathStatus   string             `json:"math_status,omitempty"`
	MathError    string             `json:"math_error,omitempty"`
	Attestation  *AttestationResult `json:"attestation,omitempty"`
}

// LlamaGuardrail connects an OpenAI-compatible local generator to the two
// KNIRV verification boundaries. It intentionally has no generation method:
// model output can guide a request, but cannot define acceptance.
type LlamaGuardrail struct {
	mathURL string
	bridge  *AttestationBridge
	client  *http.Client
}

func NewLlamaGuardrail(mathURL string, bridge *AttestationBridge) *LlamaGuardrail {
	return &LlamaGuardrail{mathURL: strings.TrimRight(mathURL, "/"), bridge: bridge, client: &http.Client{Timeout: 15 * time.Second}}
}

// VerifyClaim verifies a mathematical derivation when supplied and grounds
// its structured candidate against the assertion ledger. Transport failure is
// represented in the result rather than returned as a generation-blocking error.
func (g *LlamaGuardrail) VerifyClaim(ctx context.Context, latex string, subdomain uint32, candidate *AttestationCandidate) ClaimVerification {
	result := ClaimVerification{}
	if candidate != nil && g.bridge != nil {
		grounded, err := g.bridge.Ground(*candidate)
		if err != nil {
			result.MathError = fmt.Sprintf("attestation: %v", err)
		} else {
			result.Attestation = &grounded
		}
	}
	if latex == "" || g.mathURL == "" {
		return result
	}
	payload, err := json.Marshal(map[string]interface{}{"latex": latex, "subdomain": subdomain})
	if err != nil {
		result.MathError = err.Error()
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.mathURL+"/v1/verify/math", bytes.NewReader(payload))
	if err != nil {
		result.MathError = err.Error()
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		result.MathError = fmt.Sprintf("math verifier unavailable: %v", err)
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.MathError = fmt.Sprintf("math verifier status %d: %s", resp.StatusCode, body)
		return result
	}
	var body struct {
		Status         string  `json:"status"`
		Verified       bool    `json:"verified"`
		LogicIntegrity float32 `json:"logic_integrity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		result.MathError = fmt.Sprintf("decode math verifier: %v", err)
		return result
	}
	result.MathStatus = body.Status
	// Current MATHASHER responses use a status value; retain the explicit field
	// for newer servers without making an LLM's own confidence authoritative.
	result.MathVerified = body.Verified || body.Status == "FORMALLY_VERIFIED" || body.Status == "STRUCTURALLY_VALID"
	return result
}
