package models

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"knirvmodels/pkg/evidence"
)

// VerifyResult is the conservative verification result available before the
// trained claim extractor/judge is enabled. A result is supported only when the
// complete canonical claim occurs in a tenant-visible evidence span. Retrieval
// alone is otherwise NEI; it never becomes a guessed factual verdict.
type VerifyResult struct {
	Text         string         `json:"text"`
	Verdict      string         `json:"verdict"`
	Verified     bool           `json:"verified"`
	Gate         string         `json:"gate"`
	GateLevel    int            `json:"gate_level"`
	Confidence   float64        `json:"confidence"`
	Citations    []int          `json:"citations"`
	Evidence     []evidence.Hit `json:"evidence"`
	IndexVersion string         `json:"index_version"`
	EmbedderID   string         `json:"embedder_id"`
}

// VerifyText searches only evidence visible to tenant and applies the one
// deterministic rule this interim verifier can justify: exact canonical claim
// containment. The context exists to keep the API compatible with the later
// model-backed verifier and to permit cancellation before retrieval is added.
func (h *Hasher) VerifyText(ctx context.Context, text, tenant string) (VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return VerifyResult{}, err
	}
	claim := strings.TrimSpace(text)
	if claim == "" {
		return VerifyResult{}, indexError("a claim text is required")
	}
	ix, err := h.Index()
	if err != nil {
		return VerifyResult{}, err
	}
	result, err := ix.Search(evidence.Claim{Text: claim}, evidence.Options{K: 10, Tenant: tenant})
	if err != nil {
		return VerifyResult{}, err
	}
	out := VerifyResult{
		Text: claim, Verdict: "NEI", Gate: "uncommitted", GateLevel: 2,
		Citations: []int{}, Evidence: result.Hits, IndexVersion: result.IndexVer, EmbedderID: result.EmbedderID,
	}
	canonicalClaim := evidence.Canon(claim)
	for i, hit := range result.Hits {
		if canonicalClaim != "" && strings.Contains(evidence.Canon(hit.Text), canonicalClaim) {
			// The document itself states the exact claim. Keep the gate below
			// accepted: the calibrated multi-claim judge is not yet available.
			out.Verdict, out.Verified = "SUPPORTED", true
			out.Gate, out.GateLevel, out.Confidence = "seen", 3, 1
			out.Citations = []int{i}
			break
		}
	}
	return out, nil
}

func (h *Hasher) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request: " + err.Error()})
		return
	}
	result, err := h.VerifyText(r.Context(), req.Text, r.Header.Get(TenantHeader))
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "claim text") {
			status = http.StatusBadRequest
		} else if err == errNoIndex {
			status = http.StatusServiceUnavailable
		}
		WriteJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, result)
}
