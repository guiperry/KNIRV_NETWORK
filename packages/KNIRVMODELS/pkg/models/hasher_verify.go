package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/factslice"
	"knirvmodels/pkg/heartqwen"
	"knirvmodels/pkg/livesearch"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Verification limits. Provisional (D-H2 is open): small, because the judge shares one
// local llama slot with everything else.
const (
	defaultInlineBudget = 8 * time.Second
	minInlineBudget     = 1 * time.Second
	maxInlineBudget     = 30 * time.Second
	defaultMaxClaims    = 8
	maxVerifyBody       = 1 << 20
	maxVerifyTextBytes  = 64 << 10
	verifyJobTTL        = 15 * time.Minute
	verifyJobCap        = 256
	verifyQueueDepth    = 16
	verifyRunLimit      = 10 * time.Minute // one job never runs longer than this
)

// VerifierConfig wires HASHER's verification to the local judge.
type VerifierConfig struct {
	LLM   heartqwen.LLM
	Reg   *frame.Registry // nil: frame.Default()
	Facts func() frame.LiveFacts
	// Live reports whether the llama wrapper answers. Verification never calls a
	// model that is not up; it degrades instead.
	Live func(ctx context.Context) bool
	// Judge identifies the model file and adapter set, for the response and the cache.
	Judge func(ctx context.Context) sdk.VerifyJudge
	// Cal is the calibration fitted on owner-labelled gold data; nil until there is one.
	Cal        *factslice.Calibrator
	Thresholds factslice.Thresholds
	Witness    factslice.Witness
}

type verifier struct {
	cfg   VerifierConfig
	cache *factslice.ClaimCache
	// retriever builds the live retriever on first use (nil error, nil retriever: none).
	retriever func() (*livesearch.Retriever, error)
	sem       chan struct{} // one judge run at a time: llama has one slot
	queue     chan struct{} // bounds waiting jobs

	mu   sync.Mutex
	jobs map[string]*verifyJob
	seq  []string
}

type verifyJob struct {
	id      string
	req     sdk.VerifyRequest
	tenant  string
	created time.Time
	done    chan struct{}
	resp    sdk.VerifyResponse
}

// UseVerifier switches POST /verify from the interim exact-containment check to the
// model-backed pipeline (pkg/factslice).
func (h *Hasher) UseVerifier(cfg VerifierConfig) {
	if cfg.Reg == nil {
		cfg.Reg = frame.Default()
	}
	v := &verifier{cfg: cfg, cache: factslice.NewClaimCache(1024, time.Hour), sem: make(chan struct{}, 1),
		queue: make(chan struct{}, verifyQueueDepth), jobs: map[string]*verifyJob{}, retriever: h.Retriever}
	h.mu.Lock()
	h.verifier = v
	h.mu.Unlock()
}

func (h *Hasher) verifierOrNil() *verifier {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.verifier
}

// VerifyResult is the interim verifier's answer, used only when no judge is
// configured: a result is supported only when the complete canonical claim occurs in a
// tenant-visible evidence span. Retrieval alone is otherwise NEI; it never becomes a
// guessed factual verdict. It is always marked degraded, since no judge ran.
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
	Status       string         `json:"status"`
	Degraded     bool           `json:"degraded"`
	Mode         string         `json:"mode"`
	Notes        []string       `json:"notes,omitempty"`
}

// VerifyText is the interim verifier (see VerifyResult).
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
		Text: claim, Verdict: "NEI", Gate: "uncommitted", GateLevel: 2, Status: sdk.VerifyDone, Degraded: true, Mode: sdk.ModeIndex,
		Notes:     []string{"no judge is configured: only an exact statement in the evidence is recognised"},
		Citations: []int{}, Evidence: result.Hits, IndexVersion: result.IndexVer, EmbedderID: result.EmbedderID,
	}
	canonicalClaim := evidence.Canon(claim)
	for i, hit := range result.Hits {
		if canonicalClaim != "" && strings.Contains(evidence.Canon(hit.Text), canonicalClaim) {
			// The document itself states the exact claim. Keep the gate below
			// accepted: nothing here is calibrated.
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
	var req sdk.VerifyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxVerifyBody)).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request: " + err.Error()})
		return
	}
	tenant := r.Header.Get(TenantHeader)
	v := h.verifierOrNil()
	if v == nil {
		result, err := h.VerifyText(r.Context(), req.Text, tenant)
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
		return
	}
	resp, status, err := h.startVerify(v, req, tenant)
	if err != nil {
		WriteJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

type verifyReject struct{ msg string }

func (e verifyReject) Error() string { return e.msg }

// normalize validates a request and fills the defaults. Deferred is the default
// delivery (plan D2).
func normalizeVerify(req sdk.VerifyRequest) (sdk.VerifyRequest, time.Duration, error) {
	if strings.TrimSpace(req.Text) == "" && len(req.Claims) == 0 {
		return req, 0, verifyReject{"a text or claims are required"}
	}
	if len(req.Text) > maxVerifyTextBytes {
		return req, 0, verifyReject{"text is larger than 64 KiB"}
	}
	switch req.Mode {
	case "":
		req.Mode = sdk.ModeIndex
	case sdk.ModeIndex, sdk.ModeFlash, sdk.ModeHybrid:
	default:
		return req, 0, verifyReject{"mode must be index, flash or hybrid"}
	}
	switch req.Delivery {
	case "":
		req.Delivery = sdk.DeliveryDeferred
	case sdk.DeliveryInline, sdk.DeliveryDeferred:
	default:
		return req, 0, verifyReject{"delivery must be inline or deferred"}
	}
	if req.MaxClaims <= 0 {
		req.MaxClaims = defaultMaxClaims
	}
	if req.MaxClaims > factslice.MaxClaims {
		req.MaxClaims = factslice.MaxClaims
	}
	if len(req.Claims) > req.MaxClaims {
		return req, 0, verifyReject{"more claims than max_claims"}
	}
	if len(req.Domains) > 16 {
		return req, 0, verifyReject{"at most 16 domains"}
	}
	for _, d := range req.Domains {
		if len(d) == 0 || len(d) > 64 {
			return req, 0, verifyReject{"a domain must be 1 to 64 characters"}
		}
	}
	budget := defaultInlineBudget
	if req.BudgetMS > 0 {
		budget = time.Duration(req.BudgetMS) * time.Millisecond
	}
	if budget < minInlineBudget {
		budget = minInlineBudget
	}
	if budget > maxInlineBudget {
		budget = maxInlineBudget
	}
	return req, budget, nil
}

// startVerify admits a job and answers according to the delivery mode.
func (h *Hasher) startVerify(v *verifier, req sdk.VerifyRequest, tenant string) (sdk.VerifyResponse, int, error) {
	req, budget, err := normalizeVerify(req)
	if err != nil {
		return sdk.VerifyResponse{}, http.StatusBadRequest, err
	}
	ix, err := h.Index()
	if err != nil {
		return sdk.VerifyResponse{}, http.StatusServiceUnavailable, err
	}
	job, err := v.admit(tenant, req)
	if err != nil {
		return sdk.VerifyResponse{}, http.StatusServiceUnavailable, err
	}
	go v.run(job, ix, req, tenant)

	pending := v.pending(ix, req, job)
	if req.Delivery == sdk.DeliveryDeferred {
		return pending, http.StatusOK, nil
	}
	select {
	case <-job.done:
		return job.resp, http.StatusOK, nil
	case <-time.After(budget):
		// Out of budget: the caller gets the unverified answer and can read the job.
		p := pending
		p.Notes = append(p.Notes, "the verdict was not ready within budget_ms; read it from the job")
		return p, http.StatusOK, nil
	}
}

func (v *verifier) pending(ix *evidence.Index, req sdk.VerifyRequest, job *verifyJob) sdk.VerifyResponse {
	st := ix.Status()
	embedder := ""
	if len(st.EmbedderIDs) > 0 {
		embedder = st.EmbedderIDs[0]
	}
	resp := sdk.VerifyResponse{
		Status: sdk.VerifyPending, JobID: jobID(job), Gate: factslice.GateUncommitted.String(), GateLevel: int(factslice.GateUncommitted),
		Verdict: "unchecked", Claims: []sdk.VerifyClaim{}, Unverified: true, Calibrated: v.cfg.Cal.IsFitted(),
		IndexVersion: st.Version, EmbedderID: embedder, Mode: appliedMode(req, false), Delivery: req.Delivery, Domains: req.Domains,
		Notes: modeNotes(req, false), Citations: []int{},
	}
	return resp
}

func jobID(j *verifyJob) string { return j.id }

// admit registers a job, evicting old finished ones. It refuses when the queue is full.
func (v *verifier) admit(tenant string, req sdk.VerifyRequest) (*verifyJob, error) {
	select {
	case v.queue <- struct{}{}:
	default:
		return nil, errors.New("verification is busy; try again shortly")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now()
	kept := v.seq[:0]
	for _, id := range v.seq {
		j := v.jobs[id]
		finished := false
		select {
		case <-j.done:
			finished = true
		default:
		}
		if finished && now.Sub(j.created) > verifyJobTTL {
			delete(v.jobs, id)
			continue
		}
		kept = append(kept, id)
	}
	v.seq = kept
	for len(v.jobs) >= verifyJobCap {
		evicted := false
		for i, id := range v.seq {
			j := v.jobs[id]
			select {
			case <-j.done:
				delete(v.jobs, id)
				v.seq = append(v.seq[:i], v.seq[i+1:]...)
				evicted = true
			default:
			}
			if evicted {
				break
			}
		}
		if !evicted {
			<-v.queue
			return nil, errors.New("verification is busy; too many jobs are in flight")
		}
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	j := &verifyJob{id: id, req: req, tenant: tenant, created: now, done: make(chan struct{})}
	v.jobs[id] = j
	v.seq = append(v.seq, id)
	return j, nil
}

func (v *verifier) job(id, tenant string) (*verifyJob, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	j, ok := v.jobs[id]
	if !ok || j.tenant != tenant {
		return nil, false
	}
	return j, true
}

// appliedMode is the retrieval mode that actually ran: the requested flash or hybrid
// only when live lookup is switched on, otherwise the index.
func appliedMode(req sdk.VerifyRequest, live bool) string {
	if live && (req.Mode == sdk.ModeFlash || req.Mode == sdk.ModeHybrid) {
		return req.Mode
	}
	return sdk.ModeIndex
}

func modeNotes(req sdk.VerifyRequest, live bool) []string {
	var notes []string
	if (req.Mode == sdk.ModeFlash || req.Mode == sdk.ModeHybrid) && !live {
		notes = append(notes, "live lookup is not enabled: answered from the evidence index only")
	}
	if len(req.Domains) > 0 {
		notes = append(notes, "domains are recorded but not applied: the index carries no domain field yet")
	}
	return notes
}

func (h *Hasher) mountVerifyJobs(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/verify/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		v := h.verifierOrNil()
		if v == nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no verifier is configured"})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, prefix+"/verify/jobs/")
		j, ok := v.job(id, r.Header.Get(TenantHeader))
		if !ok {
			WriteJSON(w, http.StatusNotFound, map[string]string{"error": "no such job"})
			return
		}
		select {
		case <-j.done:
			WriteJSON(w, http.StatusOK, j.resp)
		default:
			ix, err := h.Index()
			if err != nil {
				WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
				return
			}
			WriteJSON(w, http.StatusOK, v.pending(ix, j.req, j))
		}
	})
}
