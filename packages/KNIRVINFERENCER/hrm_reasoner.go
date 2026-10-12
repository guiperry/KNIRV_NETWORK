package inferencer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HRMDecision / HRMReasoning mirror the knirv-hrm crate's Reasoning JSON
// (KNIRVARENA/rust-wasm/src/model.rs).
type HRMDecision struct {
	Name          string    `json:"name"`
	Kind          string    `json:"kind"`
	Options       []string  `json:"options"`
	Probabilities []float32 `json:"probabilities"`
	Selected      *string   `json:"selected"`
	Value         *float32  `json:"value"`
	Confidence    *float32  `json:"confidence"`
	Abstain       bool      `json:"abstain"`
}

type HRMReasoning struct {
	PlanTokens          []uint32      `json:"plan_tokens"`
	HaltingSteps        int           `json:"halting_steps"`
	Confidence          float32       `json:"confidence"`
	Decisions           []HRMDecision `json:"decisions"`
	ModelVersion        string        `json:"model_version"`
	DatasetManifestHash string        `json:"dataset_manifest_hash"`
	PolicyBundleHash    string        `json:"policy_bundle_hash"`
}

// Decision returns the named head, or nil.
func (r *HRMReasoning) Decision(name string) *HRMDecision {
	for i := range r.Decisions {
		if r.Decisions[i].Name == name {
			return &r.Decisions[i]
		}
	}
	return nil
}

// PlanText decodes the plan (byte-v1: bytes up to the first EOS; specials dropped).
func (r *HRMReasoning) PlanText() string {
	var out []byte
	for _, t := range r.PlanTokens {
		if t == 259 { // EOS
			break
		}
		if t < 256 {
			out = append(out, byte(t))
		}
	}
	return string(out)
}

// HRMReasoner calls the HRM through HEART (no cross-package import). The model
// advises; callers keep every enforcement decision.
type HRMReasoner struct {
	baseURL string
	client  *http.Client
}

func NewHRMReasoner(heartBaseURL string) *HRMReasoner {
	return &HRMReasoner{baseURL: heartBaseURL, client: &http.Client{Timeout: 30 * time.Second}}
}

// Reason takes already-encoded byte-v1 tokens at the model's fixed width.
func (h *HRMReasoner) Reason(ctx context.Context, tokens []uint32) (*HRMReasoning, error) {
	body, _ := json.Marshal(map[string]any{"tokens": tokens})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/heart/hrm/reason", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hrm unavailable: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hrm: HTTP %d: %s", res.StatusCode, bytes.TrimSpace(raw))
	}
	var out HRMReasoning
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("hrm: malformed response: %w", err)
	}
	return &out, nil
}
