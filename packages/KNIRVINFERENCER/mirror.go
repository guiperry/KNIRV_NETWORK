package inferencer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// Mirrored generation: HEAD (the VLM / mirrored text generator) and HEART (the base
// generator) each produce a candidate for the same prompt, and the harness arbitrates
// between them. This is the migration's answer to "HEAD generates text": HEAD no longer
// owns a generator; it owns the candidate and the arbitration. The arbiter weights by
// HASHER validation when HASHER returns a verdict; absent a verdict it falls back to the
// caller-supplied weights (or the hard-coded defaults) and, below the tie margin, HEART
// wins ties (HEART is the trained generator). A REFUTED candidate is disqualified. The
// exact weighting policy and the head.mirror.generate frame stage are owner-level
// (provisional).
//
// Phase 3 contract:
//   - FIX 6: both candidates are verified by HASHER (verifyCandidate); the weights and
//     refutation are derived from those verdicts, never from the hard-coded 1.0/0.9 when
//     HASHER returns a verdict, and the per-candidate verification is recorded in the
//     MirrorTrace.
//   - FIX 7: the HEART mirror candidate is generated only after a HEART decision; the
//     resulting DecisionEnvelope is sent in the (decision-aware) HEART generate request,
//     and never in the HEAD request. The decision is recorded in the MirrorTrace.

// MirroredResult is the outcome of one mirrored generation.
type MirroredResult struct {
	Winner     string                              // model whose text won
	Text       string                              // winning text
	Version    string                              // winning model version
	Refuted    bool                                // winning candidate was flagged REFUTED (none clean survived)
	Candidates map[string]models.HeadGenerateResponse // model -> its candidate (HEAD's shape; HEART mapped into it)
	Trace      *MirrorTrace                        // HASHER verification + HEART decision record
}

// MirrorTrace records what the mirror flow did: the HEART decision that conditioned the
// HEART candidate, and the HASHER verification (verdict, weight, refutation) of each
// candidate. It copies no candidate text.
type MirrorTrace struct {
	Decision   *heartDecisionEnvelope `json:"decision,omitempty"`
	Candidates []MirrorCandidateTrace `json:"candidates"`
}

// MirrorCandidateTrace is the HASHER verification outcome for one candidate.
type MirrorCandidateTrace struct {
	Model      string  `json:"model"`
	Verified   bool    `json:"verified"`              // HASHER returned a usable verdict
	Unverified bool    `json:"unverified,omitempty"`  // cache negative / no verdict yet
	Degraded   bool    `json:"degraded,omitempty"`    // HASHER degraded (judge down, etc.)
	Refuted    bool    `json:"refuted"`              // HASHER REFUTED this candidate
	Weight     float64 `json:"weight"`               // weight derived from the verdict (or fallback)
	Verdict    string  `json:"verdict,omitempty"`    // HASHER response-level verdict
	Error      string  `json:"error,omitempty"`      // verification failure (degrades gracefully)
}

// mirrorCandidate is an internal, uniform view of one model's answer.
type mirrorCandidate struct {
	model   string
	text    string
	refuted bool
	weight  float64
	version string
}

// mirrorOutcome is the per-goroutine result of one candidate's generation and
// HASHER verification. The HEAD and HEART goroutines each produce one; the
// results are collected after the wait and merged into the trace exactly once,
// so the concurrent goroutines never append to a shared slice.
type mirrorOutcome struct {
	cand  mirrorCandidate
	trace MirrorCandidateTrace
}

// heartDecisionEnvelope mirrors the HEART generate handler's DecisionEnvelope: the
// decision-core output that conditions a HEART generation. The Decisions reuse the SDK's
// models.Decision so the envelope is a faithful projection of the daemon contract.
type heartDecisionEnvelope struct {
	Decisions []models.Decision `json:"decisions,omitempty"`
	Abstained bool              `json:"abstained,omitempty"`
	Reason    string            `json:"reason,omitempty"`
}

// heartGenerateRequest is the decision-aware /heart/generate body we send (FIX 7).
type heartGenerateRequest struct {
	Prompt   string                  `json:"prompt"`
	Decision *heartDecisionEnvelope  `json:"decision,omitempty"`
}

// heartGenerateResponse is the decision-aware /heart/generate response we read back.
type heartGenerateResponse struct {
	Text      string                 `json:"text,omitempty"`
	Ready     bool                   `json:"ready"`
	Progress  float64                `json:"progress"`
	Decision  *heartDecisionEnvelope `json:"decision,omitempty"`
	Abstained bool                   `json:"abstained,omitempty"`
	Tokens    int                    `json:"tokens,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// mirrorTieMargin is read from KNIRV_MIRROR_TIE_MARGIN; below it HEART wins ties.
func mirrorTieMargin() float64 {
	if v := os.Getenv("KNIRV_MIRROR_TIE_MARGIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.05
}

// GenerateMirrored runs HEAD's HeadGenerate and HEART's decision-aware HeartGenerate
// concurrently, verifies BOTH candidates with HASHER, and arbitrates. weights, when
// non-nil, overrides the default per-model weight ONLY when HASHER does not return a
// verdict (the arbiter is otherwise given HASHER-derived scores); a missing entry falls
// back to the default. A model whose generation or HASHER verification errors (HEAD has
// no backend, HEART not ready / abstained) contributes no candidate, so a single
// surviving model wins cleanly.
func GenerateMirrored(ctx context.Context, c *models.Client, prompt, perceptual string, images []models.ImageRef, frame string, weights map[string]float64) (*MirroredResult, error) {
	// FIX 7: decide first so the HEART candidate is decision-conditioned. A decision
	// failure degrades gracefully (no envelope sent); HEAD is never given one.
	env := decideEnvelope(ctx, c, prompt)

	trace := &MirrorTrace{Decision: env}

	var wg sync.WaitGroup
	// Each goroutine produces its candidate plus its HASHER trace; the results are
	// merged into trace.Candidates exactly once after the wait, so the concurrent
	// HEAD/HEART goroutines never append to a shared slice (data race) and each
	// candidate appears exactly once (no duplicate entries).
	ch := make(chan mirrorOutcome, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := c.HeadGenerate(ctx, models.HeadGenerateRequest{Prompt: prompt, PerceptualContext: perceptual, Images: images, Frame: frame})
		if err != nil {
			return // HEAD unavailable -> no candidate; HEART may still win
		}
		cand := mirrorCandidate{model: "head", text: r.Text, refuted: r.Refuted, version: r.Version}
		v, ct := verifyCandidate(ctx, c, "head", r.Text, weights)
		applyVerification(&cand, v)
		ch <- mirrorOutcome{cand: cand, trace: ct}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// FIX 7: decision-aware HEART generate. The envelope is sent here and only here.
		t, err := heartGenerateDecision(ctx, c, prompt, env)
		if err != nil {
			return // HEART unavailable / abstained -> no candidate; HEAD may still win
		}
		cand := mirrorCandidate{model: "heart", text: t, version: heartVersion(ctx, c)}
		v, ct := verifyCandidate(ctx, c, "heart", t, weights)
		applyVerification(&cand, v)
		ch <- mirrorOutcome{cand: cand, trace: ct}
	}()

	wg.Wait()
	close(ch)

	var cands []mirrorCandidate
	for o := range ch {
		cands = append(cands, o.cand)
		trace.Candidates = append(trace.Candidates, o.trace)
	}
	return arbitrate(cands, trace)
}

// applyVerification folds a HASHER verdict into a candidate (refutation + derived weight).
func applyVerification(cand *mirrorCandidate, v hashVerdict) {
	if v.refuted {
		cand.refuted = true
	}
	cand.weight = v.weight
}

// heartVersion best-effort reads HEART's registry version for the candidate record.
func heartVersion(ctx context.Context, c *models.Client) string {
	rows, err := c.Registry(ctx)
	if err != nil {
		return ""
	}
	for _, r := range rows {
		if r.Name == "heart" {
			return r.Version
		}
	}
	return ""
}

// decideEnvelope consults the HEART decision core (now the migrated route head) and
// returns the envelope that must condition the HEART generate. Any failure returns nil
// so the flow degrades gracefully rather than blocking on the decision core.
func decideEnvelope(ctx context.Context, c *models.Client, prompt string) *heartDecisionEnvelope {
	m, err := c.Manifest(ctx)
	if err != nil {
		return nil
	}
	width, err := m.InputWidth()
	if err != nil {
		return nil
	}
	tokens := models.EncodeByteV1(prompt, width)
	r, err := c.Decide(ctx, tokens)
	if err != nil {
		return nil
	}
	return &heartDecisionEnvelope{Decisions: r.Decisions}
}

// heartGenerateDecision sends a decision-aware /heart/generate request (FIX 7). When env
// is non-nil the envelope is included; the daemon trusts it and conditions generation on
// the decision. An abstaining decision or a 503 yields no candidate.
func heartGenerateDecision(ctx context.Context, c *models.Client, prompt string, env *heartDecisionEnvelope) (string, error) {
	body, err := json.Marshal(heartGenerateRequest{Prompt: prompt, Decision: env})
	if err != nil {
		return "", fmt.Errorf("heart: encode: %w", err)
	}
	res, err := c.Do(ctx, http.MethodPost, models.HeartPrefix+"/generate", body, http.Header{})
	if err != nil {
		return "", fmt.Errorf("heart unavailable: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		return "", models.ErrNotReady
	default:
		return "", fmt.Errorf("heart: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out heartGenerateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("heart: malformed response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("heart: %s", out.Error)
	}
	if out.Abstained && out.Text == "" {
		// The decision core abstained: there is no HEART candidate to arbitrate.
		return "", models.ErrNotReady
	}
	return out.Text, nil
}

// hashVerdict is the normalized HASHER outcome for one candidate.
type hashVerdict struct {
	refuted    bool
	weight     float64
	verified   bool   // HASHER returned a usable (non-cache-negative) verdict
	unverified bool   // cache negative / no verdict yet
	degraded   bool
	verdict    string
}

// verifyCandidate invokes HASHER verification for one candidate text (FIX 6) and derives
// its weight and refutation from the verdict. When HASHER is unavailable, returns an
// unverified verdict and falls back to the supplied/default weight (graceful degradation,
// never a hard error). The outcome is returned to the caller, which merges it into the
// trace exactly once (no shared-slice append from the concurrent HEAD/HEART goroutines).
func verifyCandidate(ctx context.Context, c *models.Client, model, text string, weights map[string]float64) (hashVerdict, MirrorCandidateTrace) {
	fallback := weightOf(weights, model)
	ct := MirrorCandidateTrace{Model: model, Weight: fallback}

	resp, err := c.Verify(ctx, models.VerifyRequest{Text: text})
	if err != nil {
		ct.Error = err.Error()
		return hashVerdict{weight: fallback}, ct
	}

	ct.Verified = true
	ct.Verdict = resp.Verdict
	hv := hashVerdict{verified: true, verdict: resp.Verdict}

	switch resp.Verdict {
	case "refuted":
		ct.Refuted = true
		hv.refuted = true
		hv.weight = 0
	case "supported":
		w := resp.Confidence
		if w <= 0 {
			w = fallback
		}
		hv.weight = w
	default: // mixed, nei, unchecked
		hv.weight = 0.5
	}

	// Cache negative / no verdict yet: degrade gracefully and do NOT trust the candidate
	// with a HASHER-derived score; fall back to the supplied/default weight and clear any
	// refutation derived above.
	if resp.Unverified {
		ct.Unverified = true
		hv.unverified = true
		hv.verified = false
		hv.refuted = false
		hv.weight = fallback
	}
	if resp.Degraded {
		ct.Degraded = true
		hv.degraded = true
	}

	ct.Weight = hv.weight
	ct.Refuted = hv.refuted
	return hv, ct
}

func weightOf(weights map[string]float64, model string) float64 {
	if weights != nil {
		if w, ok := weights[model]; ok {
			return w
		}
	}
	switch model {
	case "heart":
		return 1.0
	case "head":
		return 0.9
	default:
		return 0.5
	}
}

// arbitrate picks the winner from the surviving candidates. Refuted candidates are
// disqualified; among the rest the highest weight wins, and HEART wins any tie within
// mirrorTieMargin. If no candidate survives, it returns an error (the caller degrades).
func arbitrate(cands []mirrorCandidate, trace *MirrorTrace) (*MirroredResult, error) {
	margin := mirrorTieMargin()
	var live []mirrorCandidate
	for _, c := range cands {
		if !c.refuted {
			live = append(live, c)
		}
	}
	if len(live) == 0 {
		return nil, models.ErrNoModel
	}
	best := live[0]
	for _, c := range live[1:] {
		switch {
		case c.weight > best.weight+margin:
			best = c
		case c.model == "heart" && c.weight >= best.weight:
			best = c // HEART wins ties
		}
	}

	out := &MirroredResult{Winner: best.model, Text: best.text, Version: best.version, Trace: trace}
	if best.refuted {
		out.Refuted = true
	}
	out.Candidates = map[string]models.HeadGenerateResponse{}
	for _, c := range cands {
		out.Candidates[c.model] = models.HeadGenerateResponse{Text: c.text, Model: c.model, Version: c.version, Refuted: c.refuted}
	}
	return out, nil
}
