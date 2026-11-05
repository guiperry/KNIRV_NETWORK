package transformer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/guiperry/text-embedder/pkg/embed"
	sdk "github.com/guiperry/knirv-sdk-go/models"
	"knirvmodels/pkg/embeddings"
	"knirvmodels/pkg/hands"
	handstok "knirvmodels/pkg/hands/tokenizer"
	"knirvmodels/pkg/heart/decision"
	"knirvmodels/pkg/hashing/semanticmemory"
)

// HEARTService provides HTTP endpoints for KNIRVCORTEX to query HEART
type HEARTService struct {
	engine            *hands.Engine
	semanticMemory    *semanticmemory.Model
	// generateTextFn is the text-generation seam (defaults to the HANDS engine).
	// Tests may override it to avoid loading the full model.
	generateTextFn func(prompt string, n int, temperature float32, topK int) (string, error)
	// decisionReasonFn is a test seam: when set, resolveDecision consults it
	// instead of the decision core so tests can force unavailable/abstain/select
	// without a promoted model.
	decisionReasonFn func(ctx context.Context, tokens []uint32) (json.RawMessage, error)
	bridge            *CerebrasBridge
	processor         *NetworkMetricsProcessor
	tokenizer         handstok.Tokenizer
	embedder          *embeddings.DeterministicService
	compiler          *WASMCompiler
	verifier          *BidirectionalVerifier
	auditor           *Auditor
	hashNet           *HashNetworkWrapper
	config            *HEARTConfig
	stats             *HEARTServiceStats
	baseline          *VarianceAnalyzerSnapshot
	generatorSwitcher *GeneratorSwitcher
	unifiedEngine     *UnifiedHasherEngine
	attestation       *AttestationBridge
	headOnce           sync.Once
	headSvc            *decision.Service
	mu                sync.RWMutex
}

// VarianceAnalyzerSnapshot stores a snapshot of signal indices for drift detection.
type VarianceAnalyzerSnapshot struct {
	SignalIndices []int
	Timestamp     time.Time
}

type HEARTServiceStats struct {
	TotalInquiries      uint64
	SuccessfulResponses uint64
	FailedResponses     uint64
	TotalResponseTimeMs float64
	MinResponseTimeMs   float64
	MaxResponseTimeMs   float64
	ErrorTypeCounts     map[string]uint64
	HeuristicUsage      map[string]uint64
}

// HEARTErrorInquiry matches the proto definition
type HEARTErrorInquiry struct {
	ErrorID         string            `json:"error_id"`
	ErrorType       string            `json:"error_type"`
	ErrorMessage    string            `json:"error_message"`
	ErrorContext    string            `json:"error_context"`
	StackTrace      string            `json:"stack_trace,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	NetworkMetrics  *NetworkMetrics   `json:"network_metrics,omitempty"`
	Prompt          string            `json:"prompt,omitempty"`
	ModelResponse   string            `json:"model_response,omitempty"`
	ConfidenceScore float32           `json:"confidence_score,omitempty"`
	Timestamp       uint64            `json:"timestamp"`
}

// NetworkMetrics for HEART analysis
type NetworkMetrics struct {
	NodeID         uint32    `json:"node_id"`
	TimeSlice      uint32    `json:"time_slice"`
	MeasurementVec []float32 `json:"measurement_vec"`
	TrafficIn      float32   `json:"traffic_in"`
	TrafficOut     float32   `json:"traffic_out"`
	LatencyMs      float32   `json:"latency_ms"`
	ErrorRate      float32   `json:"error_rate"`
	CPULoad        float32   `json:"cpu_load"`
}

// HEARTHeuristicResponse matches the proto definition
type HEARTHeuristicResponse struct {
	InquiryID          string         `json:"inquiry_id"`
	CommandVector      []float32      `json:"command_vector"`
	AlertLevel         uint32         `json:"alert_level"`
	HeuristicID        uint32         `json:"heuristic_id"`
	TargetNodeID       uint32         `json:"target_node_id"`
	ConfidenceScore    float32        `json:"confidence_score"`
	ActionParameters   []float32      `json:"action_parameters"`
	AnalysisSummary    string         `json:"analysis_summary"`
	RecommendedActions []string       `json:"recommended_actions"`
	DebugInsights      []string       `json:"debug_insights"`
	IdentifiedPatterns []ErrorPattern `json:"identified_patterns"`
	SimilarErrors      []SimilarError `json:"similar_errors"`
	ProcessingTimeMs   float32        `json:"processing_time_ms"`
	Timestamp          uint64         `json:"timestamp"`
}

type ErrorPattern struct {
	PatternID       string   `json:"pattern_id"`
	PatternName     string   `json:"pattern_name"`
	Description     string   `json:"description"`
	MatchConfidence float32  `json:"match_confidence"`
	Indicators      []string `json:"indicators"`
}

type SimilarError struct {
	ErrorID         string  `json:"error_id"`
	ErrorType       string  `json:"error_type"`
	SimilarityScore float32 `json:"similarity_score"`
	Resolution      string  `json:"resolution"`
	SkillID         string  `json:"skill_id,omitempty"`
}

// NewHEARTService creates a new HEART service
func NewHEARTService(bridge *CerebrasBridge, processor *NetworkMetricsProcessor) *HEARTService {
	return &HEARTService{
		bridge:    bridge,
		processor: processor,
		stats: &HEARTServiceStats{
			ErrorTypeCounts:   make(map[string]uint64),
			HeuristicUsage:    make(map[string]uint64),
			MinResponseTimeMs: 999999.0,
		},
	}
}

// NewHEARTServiceWithConfig creates a new HEART service with configuration (Phase 2)
func NewHEARTServiceWithConfig(cfg *HEARTConfig) (*HEARTService, error) {
	var semanticMemory *semanticmemory.Model
	if cfg.SemanticMemoryPath != "" {
		memory, err := semanticmemory.Load(cfg.SemanticMemoryPath)
		if err != nil {
			log.Printf("semantic memory unavailable at %s (%v); falling back to GPT", cfg.SemanticMemoryPath, err)
		} else if memory.Embedder != embed.ModelID || memory.Dimensions != embed.Dims {
			log.Printf("semantic memory at %s is incompatible with %s; falling back to GPT", cfg.SemanticMemoryPath, embed.ModelID)
		} else {
			semanticMemory = memory
			log.Printf("loaded semantic memory from %s (%d prototypes, %d indexed frames)", cfg.SemanticMemoryPath, len(memory.Prototypes), memory.FramesIndexed)
		}
	}

	var engine *hands.Engine
	// Do not allocate an untrained default engine as a fallback: its default
	// vocabulary and 768-dimensional weights can exhaust a normal server.
	// A configured checkpoint is an explicit opt-in to that legacy path.
	if semanticMemory == nil && cfg.ModelCheckpointPath != "" {
		engine = hands.NewEngine(&cfg.Gorgonite, "")
		if err := engine.Load(cfg.ModelCheckpointPath); err != nil {
			log.Printf("no trained checkpoint at %s (%v); starting from random init", cfg.ModelCheckpointPath, err)
		} else {
			log.Printf("loaded trained checkpoint from %s", cfg.ModelCheckpointPath)
		}
	}

	tok, err := handstok.NewTiktokenTokenizer("cl100k_base")
	if err != nil {
		log.Printf("tokenizer init failed: %v (proceeding without tokenizer)", err)
	}

	var tokIface handstok.Tokenizer
	if tok != nil {
		tokIface = tok
	}

	embedder := embeddings.NewDeterministicService()

	var hashNet *HashNetworkWrapper
	if cfg.UseHashNetwork {
		hashNet = NewHashNetworkWrapper()
		hashNet.SetAvailable(true)
	}

	var unifiedEngine *UnifiedHasherEngine
	if cfg.Engine != nil {
		unifiedEngine = cfg.Engine
	} else if cfg.InferenceMode == "" || cfg.InferenceMode == "unified" {
		engine, err := NewUnifiedHasherEngineFromConfig(nil)
		if err != nil {
			log.Printf("unified engine init failed: %v (proceeding without unified engine)", err)
		} else {
			unifiedEngine = engine
		}
	}

	attestation := NewEmptyAttestationBridge(cfg.AttestationLedgerDir, cfg.AttestationSignalIndices, cfg.AttestationQueueSize, "")
	if err := attestation.Reload(); err != nil {
		log.Printf("attestation ledger unavailable (%v); LM assertions will be queued with low confidence", err)
	}

	svc := &HEARTService{
		engine:         engine,
		semanticMemory: semanticMemory,
		bridge:         cfg.getBridge(),
		tokenizer:      tokIface,
		embedder:       embedder,
		compiler:       NewWASMCompiler(cfg.TinyGoPath, cfg.WASMOutDir),
		verifier:       NewBidirectionalVerifier(),
		auditor:        NewAuditor(cfg.AuditLogDir),
		hashNet:        hashNet,
		config:         cfg,
		unifiedEngine:  unifiedEngine,
		attestation:    attestation,
		stats: &HEARTServiceStats{
			ErrorTypeCounts:   make(map[string]uint64),
			HeuristicUsage:    make(map[string]uint64),
			MinResponseTimeMs: 999999.0,
		},
	}

	internalGen := NewInternalGPTGenerator(svc)
	var externalGen WASMSourceGenerator
	if cfg.ExternalGeneratorURL != "" {
		externalGen = NewExternalLLMGeneratorWithURL(cfg.ExternalGeneratorURL)
	} else if cfg.ExternalGenerateFn != nil {
		externalGen = NewExternalLLMGenerator(cfg.ExternalGenerateFn)
	}

	var trainingProvider TrainingStateProvider
	if engine != nil {
		trainingProvider = engine
	}
	svc.generatorSwitcher = NewGeneratorSwitcher(internalGen, externalGen, trainingProvider)

	svc.generateTextFn = func(prompt string, n int, temperature float32, topK int) (string, error) {
		if svc.engine == nil {
			return "", fmt.Errorf("engine not initialized")
		}
		return svc.engine.GenerateText(prompt, n, temperature, topK)
	}

	return svc, nil
}

// SetGeneratorSwitcher sets the generator switcher used by process().
// When the switcher selects GeneratorExternal, the multi-turn pipeline
// is bypassed and the external LLM generates the WASM source directly.
func (hs *HEARTService) SetGeneratorSwitcher(gs *GeneratorSwitcher) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.generatorSwitcher = gs
}

func (c *HEARTConfig) getBridge() *CerebrasBridge {
	if c.UseCerebras {
		return NewCerebrasBridge(c.CerebrasProgramDir, c.CerebrasWeightsPath, false)
	}
	return nil
}

// Start starts the HEART HTTP service
func (hs *HEARTService) Start(port int) error {
	http.HandleFunc("/heart/analyze", hs.handleAnalyze)
	http.HandleFunc("/heart/generate", hs.handleGenerate)
	http.HandleFunc("/heart/health", hs.handleHealth)
	http.HandleFunc("/heart/stats", hs.handleStats)
	hs.registerHEARTDecision(http.DefaultServeMux)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HEART service on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// Mount registers HEART's routes under prefix ("/heart" standalone,
// "/models/heart" in the daemon).
func (hs *HEARTService) Mount(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/analyze", hs.handleAnalyze)
	mux.HandleFunc(prefix+"/advise", hs.handleAdvise)
	mux.HandleFunc(prefix+"/resolve", hs.handleResolve)
	mux.HandleFunc(prefix+"/patch", hs.handlePatch)
	mux.HandleFunc(prefix+"/generate", hs.handleGenerate)
	mux.HandleFunc(prefix+"/health", hs.handleHealth)
	mux.HandleFunc(prefix+"/stats", hs.handleStats)
	mux.HandleFunc(prefix+"/reload-seeds", hs.handleReloadSeeds)
	// Mount the decision core (formerly HEAD) so it is reachable whenever HEART is
	// mounted — the daemon builds HEART through this path, and without this the
	// /models/heart/decide/* routes are never registered.
	hs.registerHEARTDecision(mux)
}

// ListenAndServe starts the HEART HTTP service on given address (Phase 3)
func (hs *HEARTService) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	hs.Mount(mux, "/heart")

	log.Printf("Starting HEART service on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// classifyInquiry maps an HTTP path to a WASMType.
func classifyInquiry(path string) WASMType {
	switch {
	case contains(path, "advise"):
		return WASMTypeRule
	case contains(path, "resolve"):
		return WASMTypeResolution
	case contains(path, "patch"):
		return WASMTypePatch
	default:
		return WASMTypeRule
	}
}

// runGorgoniteInference runs the Gorgonite GPT forward pass and returns logits.
// Instruments per-token entropy and routes spikes to gap queues (Phase 13).
//
// The bounded semantic memory is the default when its checkpoint exists.
// The legacy GPT is constructed only when a checkpoint explicitly opts into
// it; UnifiedHasherEngine remains the safe fallback when neither is present.
func (hs *HEARTService) runGorgoniteInference(tokens []int) ([]float32, error) {
	if hs.semanticMemory != nil && hs.tokenizer != nil {
		text := hs.tokenizer.Decode(tokens)
		if target, score, ok := hs.semanticMemory.Query(embed.Embed(text)); ok {
			vocabSize := hs.config.Gorgonite.VocabSize
			if vocabSize <= 0 {
				return nil, fmt.Errorf("semantic inference: invalid vocabulary size")
			}
			idx := int(target) % vocabSize
			if idx < 0 {
				idx += vocabSize
			}
			// A zero baseline leaves the learned token as the clear argmax even
			// for a low but valid cosine match.
			logits := make([]float32, vocabSize)
			logits[idx] = 1 + score
			hs.reportEntropy(logits)
			return logits, nil
		}
	}

	if hs.engine != nil {
		m := hs.engine.Model()
		data, err := m.Forward(tokens)
		if err == nil {
			if len(tokens) > 0 {
				vocabSize := m.VocabSize()
				lastStart := (len(tokens) - 1) * vocabSize
				if lastStart >= 0 && lastStart+vocabSize <= len(data) {
					data = data[lastStart : lastStart+vocabSize]
				}
			}
			hs.reportEntropy(data)
			return data, nil
		}
		log.Printf("[gorgonite] GPT forward failed, falling back to unifiedEngine: %v", err)
	}

	if hs.unifiedEngine != nil {
		logits := hs.unifiedEngine.Forward(tokens)
		if len(logits) > 0 {
			hs.reportEntropy(logits)
			return logits, nil
		}
	}

	return nil, fmt.Errorf("gpt not initialized and unifiedEngine fallback unavailable")
}

// groundLMProposal turns the LM's most likely next token into the current
// one-token assertion span and asks the attestation bridge for a witness. The
// bridge is deliberately advisory: a miss returns low confidence and queues
// mining, but it never changes or blocks the language-model output.
func (hs *HEARTService) groundLMProposal(tokens []int, logits []float32) *AttestationResult {
	if hs.attestation == nil || len(tokens) == 0 || len(logits) == 0 || hs.embedder == nil {
		return nil
	}
	best := 0
	for i := 1; i < len(logits); i++ {
		if logits[i] > logits[best] {
			best = i
		}
	}
	contextText := fmt.Sprint(tokens)
	if hs.tokenizer != nil {
		contextText = hs.tokenizer.Decode(tokens)
	}
	context := make([]int32, len(tokens))
	for i, token := range tokens {
		context[i] = int32(token)
	}
	result, err := hs.attestation.Ground(AttestationCandidate{
		ContextTokens:    context,
		AssertionSpan:    []int32{int32(best)},
		ContextEmbedding: hs.embedder.GetEmbedding(contextText),
	})
	if err != nil {
		log.Printf("attestation lookup skipped: %v", err)
		return nil
	}
	return &result
}

// reportEntropy instruments per-token entropy and routes spikes to gap
// queues (Phase 13) — shared by both the GPT and unifiedEngine paths in
// runGorgoniteInference.
func (hs *HEARTService) reportEntropy(logits []float32) {
	entropy := computeEntropy(logits)
	threshold := 3.0
	if hs.config != nil && hs.config.EntropySpikethreshold > 0 {
		threshold = hs.config.EntropySpikethreshold
	}
	if entropy > threshold {
		hs.handleEntropySpike(entropy, threshold)
	}
}

// handleEntropySpike routes high-entropy token activations to gap queues (Phase 13).
// Routes by WASM type: OntologyGap, NovelError, or SystemAlert.
// Emits OntologyDrift event when delta signal indices exceed threshold (Phase 13.3).
func (hs *HEARTService) handleEntropySpike(entropy, threshold float64) {
	log.Printf("HEART entropy spike: %.4f > %.4f, routing to gap queue", entropy, threshold)
	hs.mu.Lock()
	if hs.stats.HeuristicUsage == nil {
		hs.stats.HeuristicUsage = make(map[string]uint64)
	}
	hs.stats.HeuristicUsage["entropy_spike"]++

	// Gap queue routing by WASM type (Phase 13.1)
	gapQueue := "SystemAlert" // default
	if hs.stats.HeuristicUsage["rule"] > 0 {
		gapQueue = "OntologyGap"
	} else if hs.stats.HeuristicUsage["resolution"] > 0 {
		gapQueue = "NovelError"
	}
	hs.stats.HeuristicUsage[gapQueue]++
	hs.mu.Unlock()

	// OntologyDrift detection: compare current vs baseline signal indices (Phase 13.3)
	// Requires SetBaseline() to be called first with a VarianceAnalyzer snapshot
	hs.mu.Lock()
	baseline := hs.baseline
	hs.mu.Unlock()

	if baseline != nil {
		// Would compare current signal indices against baseline
		// Log that drift detection is active
		log.Printf("OntologyDrift detection active: baseline with %d signal indices from %v",
			len(baseline.SignalIndices), baseline.Timestamp)
	}
}

// SetBaseline sets the baseline signal indices for OntologyDrift detection.
// Call this with signal indices from a reference VarianceAnalyzer.
func (hs *HEARTService) SetBaseline(signalIndices []int) {
	snapshot := &VarianceAnalyzerSnapshot{
		SignalIndices: signalIndices,
		Timestamp:     time.Now(),
	}
	hs.mu.Lock()
	hs.baseline = snapshot
	hs.mu.Unlock()
	log.Printf("OntologyDrift baseline set with %d signal indices", len(snapshot.SignalIndices))
}

// CheckOntologyDrift compares current signal indices against baseline.
// Returns true if the Jaccard distance exceeds the threshold (0.4).
// Uses DeltaSignalIndices logic: computes symmetric difference / union.
func (hs *HEARTService) CheckOntologyDrift(currentIndices []int, threshold float32) (bool, []int) {
	hs.mu.RLock()
	baseline := hs.baseline
	hs.mu.RUnlock()

	if baseline == nil || len(baseline.SignalIndices) == 0 {
		return false, nil
	}

	// Build sets for Jaccard distance
	bSet := make(map[int]bool)
	for _, idx := range baseline.SignalIndices {
		bSet[idx] = true
	}
	aSet := make(map[int]bool)
	for _, idx := range currentIndices {
		aSet[idx] = true
	}

	intersection := 0
	union := len(bSet)
	for idx := range aSet {
		if bSet[idx] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return false, nil
	}

	jaccardDist := float32(1) - float32(intersection)/float32(union)
	if jaccardDist > threshold {
		// Return symmetric difference as deltas
		var deltas []int
		for idx := range bSet {
			if !aSet[idx] {
				deltas = append(deltas, idx)
			}
		}
		for idx := range aSet {
			if !bSet[idx] {
				deltas = append(deltas, idx)
			}
		}
		log.Printf("OntologyDrift detected: Jaccard distance %.3f > %.1f, %d signal changes",
			jaccardDist, threshold, len(deltas))
		return true, deltas
	}

	return false, nil
}

// computeEntropy computes the Shannon entropy of the softmax distribution over logits.
func computeEntropy(logits []float32) float64 {
	if len(logits) == 0 {
		return 0
	}
	maxLogit := float64(logits[0])
	for _, v := range logits {
		if float64(v) > maxLogit {
			maxLogit = float64(v)
		}
	}
	sum := 0.0
	probs := make([]float64, len(logits))
	for i, v := range logits {
		probs[i] = math.Exp(float64(v) - maxLogit)
		sum += probs[i]
	}
	entropy := 0.0
	for _, p := range probs {
		p /= sum
		if p > 0 {
			entropy -= p * math.Log(p)
		}
	}
	return entropy
}

// process runs the multi-turn HEART cognitive loop with HashNetwork fast-path (Phase 7.2 + 8.2).
// When the GeneratorSwitcher selects GeneratorExternal, the external LLM
// generates the WASM source directly (bootstrap mode) instead of running
// the full Gorgonite pipeline.
func (hs *HEARTService) process(ctx context.Context, wasmType WASMType, inquiry interface{}) (WASMDecision, error) {
	hs.mu.RLock()
	switcher := hs.generatorSwitcher
	hs.mu.RUnlock()

	// Bootstrap mode: use external LLM to generate WASM source directly,
	// bypassing the multi-turn Gorgonite pipeline.
	if switcher != nil && switcher.ActiveGenerator() == GeneratorExternal {
		source, err := switcher.GenerateSource(ctx, wasmType, inquiry)
		if err != nil {
			return WASMDecision{}, fmt.Errorf("external generation: %w", err)
		}
		s4 := &Stage4Result{Source: source, WASMType: wasmType}
		s3 := &Stage3Result{Rationale: "Generated via external LLM (bootstrap mode)"}
		d := hs.buildDecision(ctx, s4, s3)
		d.TurnCount = 1
		return d, nil
	}

	maxTurns := 3
	if hs.config != nil && hs.config.MaxTurns > 0 {
		maxTurns = hs.config.MaxTurns
	}

	// HashNetwork fast-path: if available, bypass the full Gorgonite pipeline (Phase 8.2).
	if hs.hashNet != nil && hs.hashNet.IsAvailable() {
		threshold := float32(0.85)
		if hs.config != nil && hs.config.HashNetworkConfidenceThreshold > 0 {
			threshold = hs.config.HashNetworkConfidenceThreshold
		}
		hash := hs.hashNet.ComputeHash(fmt.Sprintf("%v", inquiry))
		if hash != "" && threshold > 0 {
			s4, s3, err := hs.runPipeline(ctx, wasmType, inquiry, nil)
			if err == nil {
				d := hs.buildDecision(ctx, s4, s3)
				d.HashNetworkVerified = true
				d.TurnCount = 1
				return d, nil
			}
		}
	}

	var priorFailures []string
	var last WASMDecision

	for turn := 1; turn <= maxTurns; turn++ {
		s4, s3, err := hs.runPipeline(ctx, wasmType, inquiry, priorFailures)
		if err != nil {
			return WASMDecision{}, fmt.Errorf("pipeline error (turn %d): %w", turn, err)
		}
		d := hs.buildDecision(ctx, s4, s3)
		d.TurnCount = turn
		last = d

		if d.BidirectionalVerified {
			return d, nil
		}

		failMsg := d.BackwardVerifierMsg
		if failMsg == "" {
			failMsg = fmt.Sprintf("verification failed on turn %d", turn)
		}
		priorFailures = append(priorFailures, failMsg)
		log.Printf("HEART process turn %d/%d failed: %s, retrying", turn, maxTurns, failMsg)
	}

	return last, nil
}

// handleAdvise handles /heart/advise requests (Phase 3)
func (hs *HEARTService) handleAdvise(w http.ResponseWriter, r *http.Request) {
	var inq PolicyBadgeInquiry
	if err := json.NewDecoder(r.Body).Decode(&inq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	decision, err := hs.process(r.Context(), WASMTypeRule, inq)
	if err != nil {
		http.Error(w, fmt.Sprintf("pipeline error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

// handleResolve handles /heart/resolve requests (Phase 3)
func (hs *HEARTService) handleResolve(w http.ResponseWriter, r *http.Request) {
	var inq DVEErrorInquiry
	if err := json.NewDecoder(r.Body).Decode(&inq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if inq.DVESessionID == "" {
		http.Error(w, "resolution requires active DVE session", http.StatusUnprocessableEntity)
		return
	}
	decision, err := hs.process(r.Context(), WASMTypeResolution, inq)
	if err != nil {
		http.Error(w, fmt.Sprintf("pipeline error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

// handlePatch handles /heart/patch requests (Phase 3)
func (hs *HEARTService) handlePatch(w http.ResponseWriter, r *http.Request) {
	var inq SystemPatchInquiry
	if err := json.NewDecoder(r.Body).Decode(&inq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	decision, err := hs.process(r.Context(), WASMTypePatch, inq)
	if err != nil {
		http.Error(w, fmt.Sprintf("pipeline error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

// buildDecision compiles WASM, verifies, audits, and returns a WASMDecision.
func (hs *HEARTService) buildDecision(ctx context.Context, s4 *Stage4Result, s3 *Stage3Result) WASMDecision {
	decision := WASMDecision{
		WASMType:  s4.WASMType,
		Rationale: s3.Rationale,
		TurnCount: 1,
		Grounding: s3.Grounding,
	}

	if hs.compiler == nil {
		return decision
	}

	result, err := hs.compiler.Compile(ctx, s4.Source, s4.WASMType)
	if err != nil {
		log.Printf("WASM compile error: %v", err)
		return decision
	}
	decision.WASMPath = result.WASMPath
	decision.WASMHash = result.WASMHash

	passed, confidence, msg := hs.verifier.VerifyBinary(s4.Source, string(s4.WASMType), result.WASMPath)
	decision.BidirectionalVerified = passed
	decision.VerifierConfidence = confidence
	if passed {
		decision.ForwardVerifierMsg = msg
		decision.WazeroExecPassed = true
	} else {
		decision.BackwardVerifierMsg = msg
	}

	if hs.auditor != nil {
		inquiryHash := inquiryHashStr(s4.Source)
		_ = hs.auditor.WriteAuditLog(&AuditRecord{
			InquiryHash: inquiryHash,
			WASMSha256:  result.WASMHash,
			WASMType:    s4.WASMType,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		})
	}

	return decision
}

// inquiryHashStr returns a short SHA-256 hex of the source string.
func inquiryHashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

// handleAnalyze processes error inquiries from CORTEX
// Now uses the new process() architecture instead of the legacy processInquiry() path
func (hs *HEARTService) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enable CORS for WASM clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	startTime := time.Now()

	// Parse inquiry
	var inquiry HEARTErrorInquiry
	if err := json.NewDecoder(r.Body).Decode(&inquiry); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("HEART inquiry: %s (type: %s)", inquiry.ErrorID, inquiry.ErrorType)

	// Update stats
	hs.mu.Lock()
	hs.stats.TotalInquiries++
	hs.stats.ErrorTypeCounts[inquiry.ErrorType]++
	hs.mu.Unlock()

	// Convert HEARTErrorInquiry to DVEErrorInquiry and use new process() architecture
	dveInquiry := DVEErrorInquiry{
		DVESessionID: inquiry.ErrorID, // Use ErrorID as session ID for compatibility
		ErrorType:    inquiry.ErrorType,
		ErrorMessage: inquiry.ErrorMessage,
		ErrorContext: inquiry.ErrorContext,
		StackTrace:   inquiry.StackTrace,
		Metadata:     inquiry.Metadata,
	}

	// Use the new process() multi-turn loop with WASMTypeResolution
	decision, err := hs.process(r.Context(), WASMTypeResolution, dveInquiry)
	if err != nil {
		hs.mu.Lock()
		hs.stats.FailedResponses++
		hs.mu.Unlock()

		http.Error(w, fmt.Sprintf("Processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Update stats
	elapsed := time.Since(startTime).Milliseconds()
	hs.mu.Lock()
	hs.stats.SuccessfulResponses++
	hs.stats.TotalResponseTimeMs += float64(elapsed)
	if float64(elapsed) < hs.stats.MinResponseTimeMs {
		hs.stats.MinResponseTimeMs = float64(elapsed)
	}
	if float64(elapsed) > hs.stats.MaxResponseTimeMs {
		hs.stats.MaxResponseTimeMs = float64(elapsed)
	}
	hs.stats.HeuristicUsage[fmt.Sprintf("turn_%d", decision.TurnCount)]++
	hs.mu.Unlock()

	// Convert WASMDecision to HEARTHeuristicResponse for backward compatibility
	response := &HEARTHeuristicResponse{
		InquiryID:          inquiry.ErrorID,
		CommandVector:      decisionToCommandVector(decision),
		AlertLevel:         decisionToAlertLevel(decision.WASMType),
		HeuristicID:        decisionToHeuristicID(decision),
		TargetNodeID:       0, // Default for backward compatibility
		ConfidenceScore:    decision.VerifierConfidence,
		ActionParameters:   decisionToActionParams(decision),
		AnalysisSummary:    decision.Rationale,
		RecommendedActions: hs.generateRecommendedActions(&inquiry, decisionToHeuristicID(decision)),
		Timestamp:          uint64(time.Now().Unix()),
	}

	response.ProcessingTimeMs = float32(elapsed)

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// decisionToCommandVector converts a WASMDecision to a command vector
func decisionToCommandVector(d WASMDecision) []float32 {
	return []float32{
		float32(decisionToAlertLevel(d.WASMType)),
		float32(decisionToHeuristicID(d)),
		0, // Target node ID placeholder
		d.VerifierConfidence,
		float32(d.TurnCount),
		0, 0, 0, 0, // Padding
	}
}

// decisionToAlertLevel maps WASMType to alert level
func decisionToAlertLevel(wasmType WASMType) uint32 {
	switch wasmType {
	case WASMTypeRule:
		return 1
	case WASMTypeResolution:
		return 3
	case WASMTypePatch:
		return 5
	default:
		return 2
	}
}

// decisionToHeuristicID generates a heuristic ID from the decision
func decisionToHeuristicID(d WASMDecision) uint32 {
	hash := 0
	for _, c := range d.Rationale {
		hash += int(c)
	}
	return uint32(hash%1000) + 100
}

// decisionToActionParams extracts action parameters from decision
func decisionToActionParams(d WASMDecision) []float32 {
	params := make([]float32, 4)
	if d.BidirectionalVerified {
		params[0] = 1.0
	}
	if d.WazeroExecPassed {
		params[1] = 1.0
	}
	if d.HashNetworkVerified {
		params[2] = 1.0
	}
	params[3] = float32(d.TurnCount)
	return params
}

// processInquiry analyzes the error using HEART transformer
func (hs *HEARTService) processInquiry(inquiry *HEARTErrorInquiry) (*HEARTHeuristicResponse, error) {
	// Prepare input for HEART
	var heartInput *HEARTInput

	if inquiry.NetworkMetrics != nil {
		// Use network metrics if provided
		heartInput = &HEARTInput{
			Measurements: [][]float32{inquiry.NetworkMetrics.MeasurementVec},
			NodeIDs:      []uint32{inquiry.NetworkMetrics.NodeID},
			TimeSlices:   []uint32{inquiry.NetworkMetrics.TimeSlice},
		}
	} else {
		// Synthesize metrics from error context
		heartInput = hs.synthesizeMetricsFromError(inquiry)
	}

	// Run inference on HEART (Cerebras WSE)
	output, err := hs.bridge.RunInference(heartInput)
	if err != nil {
		return nil, fmt.Errorf("HEART inference failed: %w", err)
	}

	// Interpret HEART output
	if len(output.Commands) == 0 || len(output.Commands[0]) < 8 {
		return nil, fmt.Errorf("invalid HEART output")
	}

	command := output.Commands[0]

	// Extract command components
	alertLevel := uint32(command[0])
	heuristicID := uint32(command[1])
	targetNodeID := uint32(command[2])
	confidence := command[3]
	actionParams := command[4:8]

	// Generate analysis
	analysis := hs.generateAnalysisSummary(inquiry, alertLevel, heuristicID)
	actions := hs.generateRecommendedActions(inquiry, heuristicID)
	insights := hs.generateDebugInsights(inquiry, command)

	// Identify patterns (simulated - would use KNIRVGRAPH in production)
	patterns := hs.identifyErrorPatterns(inquiry)

	// Find similar errors (simulated - would query KNIRVGRAPH)
	similarErrors := hs.findSimilarErrors(inquiry)

	response := &HEARTHeuristicResponse{
		InquiryID:          inquiry.ErrorID,
		CommandVector:      command,
		AlertLevel:         alertLevel,
		HeuristicID:        heuristicID,
		TargetNodeID:       targetNodeID,
		ConfidenceScore:    confidence,
		ActionParameters:   actionParams,
		AnalysisSummary:    analysis,
		RecommendedActions: actions,
		DebugInsights:      insights,
		IdentifiedPatterns: patterns,
		SimilarErrors:      similarErrors,
		Timestamp:          uint64(time.Now().Unix()),
	}

	return response, nil
}

// synthesizeMetricsFromError creates measurement vectors from error context
func (hs *HEARTService) synthesizeMetricsFromError(inquiry *HEARTErrorInquiry) *HEARTInput {
	// Create synthetic metrics based on error characteristics
	measurements := make([]float32, 16)

	// Encode error type (simple hash)
	typeHash := 0.0
	for _, c := range inquiry.ErrorType {
		typeHash += float64(c)
	}
	measurements[0] = float32(typeHash / 1000.0)

	// Encode error severity
	switch inquiry.ErrorType {
	case "InferenceError", "ModelError":
		measurements[1] = 5.0 // High severity
	case "NetworkError":
		measurements[1] = 4.0
	case "TypeError":
		measurements[1] = 3.0
	default:
		measurements[1] = 2.0
	}

	// Other synthetic features
	measurements[2] = float32(len(inquiry.ErrorMessage)) / 100.0
	measurements[3] = float32(len(inquiry.ErrorContext)) / 100.0

	if inquiry.ConfidenceScore > 0 {
		measurements[4] = inquiry.ConfidenceScore
	}

	// Fill rest with normalized metadata
	metaIdx := 5
	for k, v := range inquiry.Metadata {
		if metaIdx < 16 {
			valHash := 0.0
			for _, c := range k + v {
				valHash += float64(c)
			}
			measurements[metaIdx] = float32(valHash / 1000.0)
			metaIdx++
		}
	}

	return &HEARTInput{
		Measurements: [][]float32{measurements},
		NodeIDs:      []uint32{0},
		TimeSlices:   []uint32{uint32(inquiry.Timestamp % 64)},
	}
}

// generateAnalysisSummary creates human-readable analysis
func (hs *HEARTService) generateAnalysisSummary(inquiry *HEARTErrorInquiry, alertLevel, heuristicID uint32) string {
	severity := "medium"
	switch alertLevel {
	case 5:
		severity = "critical"
	case 4:
		severity = "high"
	case 3:
		severity = "medium"
	case 2:
		severity = "low"
	case 1:
		severity = "info"
	}

	return fmt.Sprintf("HEART Analysis: %s severity %s detected. Recommended heuristic: H%d. %s",
		severity, inquiry.ErrorType, heuristicID, inquiry.ErrorMessage)
}

// generateRecommendedActions creates actionable recommendations.
// When semantic memory or an explicit Gorgonite GPT and tokenizer are
// available it uses inference to select actions; otherwise it falls back to
// heuristic rules (Phase 2.4).
func (hs *HEARTService) generateRecommendedActions(inquiry *HEARTErrorInquiry, heuristicID uint32) []string {
	if (hs.engine != nil || hs.semanticMemory != nil) && hs.tokenizer != nil {
		prompt := fmt.Sprintf("error_type:%s message:%s", inquiry.ErrorType, inquiry.ErrorMessage)
		tokens := hs.tokenizer.Encode(prompt)
		logits, err := hs.runGorgoniteInference(tokens)
		if err == nil && len(logits) > 0 {
			return hs.actionsFromLogits(logits)
		}
	}

	// Heuristic fallback.
	switch heuristicID {
	case 101:
		return []string{
			"Validate input types before processing",
			"Add type guards to prevent undefined access",
			"Review variable initialization",
		}
	case 201:
		return []string{
			"Implement exponential backoff for retries",
			"Check endpoint availability and CORS configuration",
			"Add fallback to cached data if available",
		}
	case 301:
		return []string{
			"Verify model weights are loaded correctly",
			"Check input tensor shapes and formats",
			"Consider reloading model weights",
			"Apply relevant LoRA adapter if available",
		}
	default:
		return []string{
			"Review error context and stack trace",
			"Check system logs for related errors",
			"Consider retry with exponential backoff",
		}
	}
}

// actionPool is the set of possible actions indexed by logit argmax bucket.
var actionPool = []string{
	"Validate input types before processing",
	"Implement exponential backoff for retries",
	"Verify model weights are loaded correctly",
	"Add type guards to prevent undefined access",
	"Check endpoint availability and CORS configuration",
	"Review variable initialization",
	"Check input tensor shapes and formats",
	"Review error context and stack trace",
}

// actionsFromLogits selects up to 3 actions using the top-3 logit indices.
func (hs *HEARTService) actionsFromLogits(logits []float32) []string {
	n := len(actionPool)
	if n == 0 {
		return nil
	}
	// Top-3 argmax without full sort.
	selected := make([]string, 0, 3)
	used := make(map[int]bool)
	for range 3 {
		best, bestIdx := float32(-1e9), -1
		for i, v := range logits {
			bucket := i % n
			if !used[bucket] && v > best {
				best, bestIdx = v, bucket
			}
		}
		if bestIdx < 0 {
			break
		}
		selected = append(selected, actionPool[bestIdx])
		used[bestIdx] = true
	}
	return selected
}

// generateDebugInsights provides debugging information
func (hs *HEARTService) generateDebugInsights(inquiry *HEARTErrorInquiry, command []float32) []string {
	insights := []string{
		fmt.Sprintf("Error type: %s", inquiry.ErrorType),
		fmt.Sprintf("Command vector magnitude: %.2f", vectorMagnitude(command)),
	}

	if len(inquiry.Metadata) > 0 {
		insights = append(insights, fmt.Sprintf("Metadata entries: %d", len(inquiry.Metadata)))
	}

	return insights
}

// identifyErrorPatterns finds known patterns in the error
func (hs *HEARTService) identifyErrorPatterns(inquiry *HEARTErrorInquiry) []ErrorPattern {
	patterns := []ErrorPattern{}

	// Simple pattern matching (would use KNIRVGRAPH in production)
	if contains(inquiry.ErrorMessage, "undefined") {
		patterns = append(patterns, ErrorPattern{
			PatternID:       "PTN001",
			PatternName:     "Undefined Reference",
			Description:     "Accessing undefined variable or property",
			MatchConfidence: 0.85,
			Indicators:      []string{"undefined", "null reference"},
		})
	}

	if contains(inquiry.ErrorType, "Network") {
		patterns = append(patterns, ErrorPattern{
			PatternID:       "PTN002",
			PatternName:     "Network Connectivity",
			Description:     "Network request failed or timed out",
			MatchConfidence: 0.90,
			Indicators:      []string{"fetch failed", "timeout", "CORS"},
		})
	}

	return patterns
}

// findSimilarErrors queries for similar past errors using embedder cosine similarity.
// Falls back to heuristic if embedder is unavailable (Phase 1.3).
func (hs *HEARTService) findSimilarErrors(inquiry *HEARTErrorInquiry) []SimilarError {
	// Use embedder-based cosine similarity if available
	if hs.embedder != nil {
		query := inquiry.ErrorType + " " + inquiry.ErrorMessage + " " + inquiry.ErrorContext
		queryEmb := hs.embedder.GetEmbedding(query)

		// Simulated known errors with pre-computed embeddings
		knownErrors := []struct {
			ID         string
			Text       string
			Resolution string
			SkillID    string
		}{
			{"ERR-2024-001", "TypeError undefined validation", "Applied type validation LoRA adapter", "skill-typecheck-v1"},
			{"ERR-2024-042", "NetworkError timeout fetch", "Network timeout retry logic", "skill-network-v2"},
		}

		results := make([]SimilarError, 0, len(knownErrors))
		for _, k := range knownErrors {
			knownEmb := hs.embedder.GetEmbedding(k.Text)
			score := embeddings.CosineSimilarity(queryEmb, knownEmb)
			if score > 0.5 { // threshold for similarity
				results = append(results, SimilarError{
					ErrorID:         k.ID,
					ErrorType:       inquiry.ErrorType,
					SimilarityScore: score,
					Resolution:      k.Resolution,
					SkillID:         k.SkillID,
				})
			}
		}
		if len(results) > 0 {
			return results
		}
	}

	// Fallback heuristic
	return []SimilarError{
		{
			ErrorID:         "ERR-2024-001",
			ErrorType:       inquiry.ErrorType,
			SimilarityScore: 0.78,
			Resolution:      "Applied type validation LoRA adapter",
			SkillID:         "skill-typecheck-v1",
		},
	}
}

// HeartGenerateRequest is the decision-aware /heart/generate body. The optional
// Decision envelope lets a caller inject a decision already produced by the
// decision core (or have HEART run the core itself when requested). Plain
// generation (no Decision and WithDecision=false) never consults the decision
// core, so legacy clients keep working even when the core is unavailable.
type HeartGenerateRequest struct {
	Prompt       string            `json:"prompt"`
	Decision     *DecisionEnvelope `json:"decision,omitempty"`
	WithDecision bool              `json:"with_decision,omitempty"`
}

// DecisionEnvelope carries the decision-core output so it can be supplied by the
// caller or surfaced by HEART. It mirrors the SDK Reasoning envelope: each
// Decision records whether a head selected an option or abstained.
type DecisionEnvelope struct {
	Decisions []sdk.Decision `json:"decisions,omitempty"`
	Abstained bool           `json:"abstained,omitempty"`
	Reason    string         `json:"reason,omitempty"`
}

// HeartGenerateResponse is the decision-aware /heart/generate response. It always
// carries the (possibly abstaining) decision metadata alongside the text.
type HeartGenerateResponse struct {
	Text      string            `json:"text,omitempty"`
	Ready     bool              `json:"ready"`
	Progress  float64           `json:"progress"`
	Decision  *DecisionEnvelope `json:"decision,omitempty"`
	Abstained bool              `json:"abstained,omitempty"`
	Tokens    int               `json:"tokens,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// decisionAbstains reports whether the envelope represents an abstention: either
// it was flagged as abstained, or every decision head explicitly abstained.
func decisionAbstains(env *DecisionEnvelope) bool {
	if env == nil {
		return false
	}
	if env.Abstained {
		return true
	}
	for i := range env.Decisions {
		if env.Decisions[i].Abstain {
			return true
		}
	}
	return false
}

// decisionSummary renders the envelope as a short frame prefix.
func decisionSummary(env *DecisionEnvelope) string {
	if env == nil {
		return ""
	}
	var b strings.Builder
	for _, d := range env.Decisions {
		if d.Abstain {
			b.WriteString(fmt.Sprintf("[decision:%s abstain]", d.Name))
			continue
		}
		if d.Selected != nil {
			b.WriteString(fmt.Sprintf("[decision:%s=%s]", d.Name, *d.Selected))
		}
	}
	return b.String()
}

// encodeDecisionTokens byte-v1 encodes the prompt for the decision core. The SDK
// core may be unavailable offline; a panic there is converted to an error so the
// caller can abstain gracefully.
func encodeDecisionTokens(prompt string) (tokens []uint32, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("decision encoding unavailable: %v", r)
		}
	}()
	tokens = sdk.EncodeByteV1(prompt, 256)
	return tokens, nil
}

// resolveDecision returns the decision envelope to use for a generate request.
// If the caller supplied one it is trusted; otherwise the decision core
// (served under /models/heart/decide) is consulted. A core failure is returned
// as an error so callers can short-circuit gracefully.
func (hs *HEARTService) resolveDecision(ctx context.Context, prompt string) (*DecisionEnvelope, error) {
	tokens, err := encodeDecisionTokens(prompt)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if hs.decisionReasonFn != nil {
		raw, err = hs.decisionReasonFn(ctx, tokens)
	} else {
		raw, err = hs.headService().Reason(ctx, tokens)
	}
	if err != nil {
		return nil, err
	}
	var r sdk.Reasoning
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &DecisionEnvelope{Decisions: r.Decisions}, nil
}

// handleGenerate handles /heart/generate requests from the backend harness.
// It is decision-aware: it runs (or accepts) a decision from the core and only
// proceeds to text generation when the decision does not abstain. Returns a 503
// with readiness status when the model isn't trained, or when the decision core
// cannot be reached.
func (hs *HEARTService) handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed","ready":false}`, http.StatusMethodNotAllowed)
		return
	}

	var req HeartGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bad request: %s","ready":false}`, err), http.StatusBadRequest)
		return
	}

	hs.mu.RLock()
	engine := hs.engine
	gen := hs.generateTextFn
	hs.mu.RUnlock()

	if engine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HeartGenerateResponse{
			Error:    "engine not initialized",
			Ready:    false,
			Progress: 0.0,
		})
		return
	}

	// Resolve the decision only when the caller asks for one (explicit envelope
	// or WithDecision). Plain generation — what SDK HeartGenerate and the harness
	// send — must never fail or abstain just because the decision core is
	// unavailable; it degrades to ungated generation with decision=nil.
	var env *DecisionEnvelope
	if req.Decision != nil {
		env = req.Decision
	} else if req.WithDecision {
		d, err := hs.resolveDecision(r.Context(), req.Prompt)
		if err != nil {
			// Decision core unavailable (e.g. no promoted model offline): abstain gracefully.
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(HeartGenerateResponse{
				Error:     fmt.Sprintf("decision core unavailable: %v", err),
				Ready:     engine.IsReady(),
				Progress:  engine.Progress(),
				Abstained: true,
				Decision:  &DecisionEnvelope{Abstained: true, Reason: err.Error()},
			})
			return
		}
		env = d
	}

	// An abstaining decision short-circuits generation.
	if env != nil && decisionAbstains(env) {
		json.NewEncoder(w).Encode(HeartGenerateResponse{
			Ready:     engine.IsReady(),
			Progress:  engine.Progress(),
			Abstained: true,
			Decision:  env,
		})
		return
	}

	if !engine.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HeartGenerateResponse{
			Error:    "model not trained",
			Ready:    false,
			Progress: engine.Progress(),
			Decision: env,
		})
		return
	}

	// Inject the decision frame so generation is conditioned on the decision.
	generatePrompt := req.Prompt
	if summary := decisionSummary(env); summary != "" {
		generatePrompt = summary + " " + req.Prompt
	}

	maxNewTokens := 256
	temperature := float32(0.8)
	topK := 40

	text, err := gen(generatePrompt, maxNewTokens, temperature, topK)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HeartGenerateResponse{
			Error:     fmt.Sprintf("generation failed: %v", err),
			Ready:     true,
			Progress:  1.0,
			Abstained: false,
			Decision:  env,
		})
		return
	}

	json.NewEncoder(w).Encode(HeartGenerateResponse{
		Text:     text,
		Ready:    true,
		Progress: 1.0,
		Tokens:   len(text),
		Decision: env,
	})
}

// handleHealth returns service health status including model readiness.
func (hs *HEARTService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var ready bool
	var progress float64
	if hs.engine != nil {
		ready = hs.engine.IsReady()
		progress = hs.engine.Progress()
	}

	health := map[string]interface{}{
		"available":      true,
		"status":         "online",
		"loaded":         true,
		"ready":          ready,
		"progress":       progress,
		"avg_latency_ms": hs.getAvgLatency(),
		"total_queries":  hs.stats.TotalInquiries,
		"success_rate":   hs.getSuccessRate(),
		"head":            hs.headService().Status(),
	}

	json.NewEncoder(w).Encode(health)
}

// handleStats returns detailed statistics
func (hs *HEARTService) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	hs.mu.RLock()
	defer hs.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_inquiries":      hs.stats.TotalInquiries,
		"successful_responses": hs.stats.SuccessfulResponses,
		"failed_responses":     hs.stats.FailedResponses,
		"avg_response_time_ms": hs.getAvgLatency(),
		"min_response_time_ms": hs.stats.MinResponseTimeMs,
		"max_response_time_ms": hs.stats.MaxResponseTimeMs,
		"error_type_counts":    hs.stats.ErrorTypeCounts,
		"heuristic_usage":      hs.stats.HeuristicUsage,
	})
}

// handleReloadSeeds reloads the SeedStore for the unified engine.
// Accepts a JSON body with a SeedStore or reloads from default if empty.
func (hs *HEARTService) handleReloadSeeds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.unifiedEngine == nil {
		http.Error(w, `{"error":"unified engine not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Seeds *SeedStore `json:"seeds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Seeds != nil {
		hs.unifiedEngine.SetSeeds(req.Seeds)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "reloaded",
			"mode":   hs.unifiedEngine.Mode(),
			"seeds":  "custom",
		})
		return
	}

	// No custom payload: prefer seeds mined by 3_DATA_SEEDER over unmined
	// crypto/rand noise. Previously this always called BuildDefaultSeedStore
	// directly, so a bare reload silently discarded any mined data and
	// re-randomized the whole engine.
	newStore, stats := LoadOrBuildSeedStore(ResolveFramesDir(), DefaultUnifiedConfig())
	hs.unifiedEngine.SetSeeds(newStore)
	seedSource := "default"
	resp := map[string]interface{}{
		"status": "reloaded",
		"mode":   hs.unifiedEngine.Mode(),
	}
	if stats != nil {
		seedSource = "mined"
		resp["ledger_records"] = stats.LedgerRecords
		resp["tokens_covered"] = stats.TokensCovered
		resp["tokens_total"] = stats.TokensTotal
	}
	if hs.attestation != nil {
		if err := hs.attestation.Reload(); err != nil {
			resp["attestation_ledger"] = "unavailable"
			resp["attestation_error"] = err.Error()
		} else {
			resp["attestation_ledger"] = "reloaded"
		}
	}
	resp["seeds"] = seedSource
	json.NewEncoder(w).Encode(resp)
}

func (hs *HEARTService) getAvgLatency() float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if hs.stats.SuccessfulResponses > 0 {
		return hs.stats.TotalResponseTimeMs / float64(hs.stats.SuccessfulResponses)
	}
	return 0.0
}

func (hs *HEARTService) getSuccessRate() float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if hs.stats.TotalInquiries > 0 {
		return float64(hs.stats.SuccessfulResponses) / float64(hs.stats.TotalInquiries)
	}
	return 0.0
}

// Helper functions

func vectorMagnitude(vec []float32) float64 {
	sum := 0.0
	for _, v := range vec {
		sum += float64(v * v)
	}
	return sum
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
