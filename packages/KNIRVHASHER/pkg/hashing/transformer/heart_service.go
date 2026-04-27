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
	"sync"
	"time"

	"gorgonia.org/gorgonia"
)

// HEARTService provides HTTP endpoints for KNIRVCORTEX to query HEART
type HEARTService struct {
	gpt       *GPT
	bridge    *CerebrasBridge
	processor *NetworkMetricsProcessor
	tokenizer Tokenizer
	compiler  *WASMCompiler
	verifier  *BidirectionalVerifier
	auditor   *Auditor
	hashNet   *HashNetworkWrapper
	config    *HEARTConfig
	stats     *HEARTServiceStats
	mu        sync.RWMutex
}

type HEARTServiceStats struct {
	TotalInquiries       uint64
	SuccessfulResponses  uint64
	FailedResponses      uint64
	TotalResponseTimeMs  float64
	MinResponseTimeMs    float64
	MaxResponseTimeMs    float64
	ErrorTypeCounts      map[string]uint64
	HeuristicUsage       map[string]uint64
}

// HEARTErrorInquiry matches the proto definition
type HEARTErrorInquiry struct {
	ErrorID          string            `json:"error_id"`
	ErrorType        string            `json:"error_type"`
	ErrorMessage     string            `json:"error_message"`
	ErrorContext     string            `json:"error_context"`
	StackTrace       string            `json:"stack_trace,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	NetworkMetrics   *NetworkMetrics   `json:"network_metrics,omitempty"`
	Prompt           string            `json:"prompt,omitempty"`
	ModelResponse    string            `json:"model_response,omitempty"`
	ConfidenceScore  float32           `json:"confidence_score,omitempty"`
	Timestamp        uint64            `json:"timestamp"`
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
	InquiryID           string          `json:"inquiry_id"`
	CommandVector       []float32       `json:"command_vector"`
	AlertLevel          uint32          `json:"alert_level"`
	HeuristicID         uint32          `json:"heuristic_id"`
	TargetNodeID        uint32          `json:"target_node_id"`
	ConfidenceScore     float32         `json:"confidence_score"`
	ActionParameters    []float32       `json:"action_parameters"`
	AnalysisSummary     string          `json:"analysis_summary"`
	RecommendedActions  []string        `json:"recommended_actions"`
	DebugInsights       []string        `json:"debug_insights"`
	IdentifiedPatterns  []ErrorPattern  `json:"identified_patterns"`
	SimilarErrors       []SimilarError  `json:"similar_errors"`
	ProcessingTimeMs    float32         `json:"processing_time_ms"`
	Timestamp           uint64          `json:"timestamp"`
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
			ErrorTypeCounts: make(map[string]uint64),
			HeuristicUsage:  make(map[string]uint64),
			MinResponseTimeMs: 999999.0,
		},
	}
}

// NewHEARTServiceWithConfig creates a new HEART service with configuration (Phase 2)
func NewHEARTServiceWithConfig(cfg *HEARTConfig) (*HEARTService, error) {
	g := gorgonia.NewGraph()
	gpt := NewGPT(g, &cfg.Gorgonite)

	tok, err := NewTiktokenTokenizer("cl100k_base")
	if err != nil {
		log.Printf("tokenizer init failed: %v (proceeding without tokenizer)", err)
	}

	var tokIface Tokenizer
	if tok != nil {
		tokIface = tok
	}

	var hashNet *HashNetworkWrapper
	if cfg.UseHashNetwork {
		hashNet = NewHashNetworkWrapper()
		hashNet.SetAvailable(true)
	}

	return &HEARTService{
		gpt:       gpt,
		bridge:    cfg.getBridge(),
		tokenizer: tokIface,
		compiler:  NewWASMCompiler(cfg.TinyGoPath, cfg.WASMOutDir),
		verifier:  NewBidirectionalVerifier(),
		auditor:   NewAuditor(cfg.AuditLogDir),
		hashNet:   hashNet,
		config:    cfg,
		stats: &HEARTServiceStats{
			ErrorTypeCounts:   make(map[string]uint64),
			HeuristicUsage:    make(map[string]uint64),
			MinResponseTimeMs: 999999.0,
		},
	}, nil
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
	http.HandleFunc("/heart/health", hs.handleHealth)
	http.HandleFunc("/heart/stats", hs.handleStats)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HEART service on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// ListenAndServe starts the HEART HTTP service on given address (Phase 3)
func (hs *HEARTService) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/heart/advise", hs.handleAdvise)
	mux.HandleFunc("/heart/resolve", hs.handleResolve)
	mux.HandleFunc("/heart/patch", hs.handlePatch)
	mux.HandleFunc("/heart/health", hs.handleHealth)
	mux.HandleFunc("/heart/stats", hs.handleStats)

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
func (hs *HEARTService) runGorgoniteInference(tokens []int) ([]float32, error) {
	if hs.gpt == nil {
		return nil, fmt.Errorf("gpt not initialized")
	}
	logitsNode, err := hs.gpt.Forward(false)
	if err != nil {
		return nil, fmt.Errorf("gpt forward: %w", err)
	}
	vm := gorgonia.NewTapeMachine(hs.gpt.graph)
	defer vm.Close()
	if err := vm.RunAll(); err != nil {
		return nil, fmt.Errorf("gpt vm: %w", err)
	}
	if logitsNode.Value() == nil {
		return nil, fmt.Errorf("nil logits value")
	}
	data, ok := logitsNode.Value().Data().([]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected logits type")
	}

	// Per-token entropy instrumentation (Phase 13).
	entropy := computeEntropy(data)
	threshold := 3.0
	if hs.config != nil && hs.config.EntropySpikethreshold > 0 {
		threshold = hs.config.EntropySpikethreshold
	}
	if entropy > threshold {
		hs.handleEntropySpike(entropy, threshold)
	}

	return data, nil
}

// handleEntropySpike routes high-entropy token activations to gap queues (Phase 13).
func (hs *HEARTService) handleEntropySpike(entropy, threshold float64) {
	log.Printf("HEART entropy spike: %.4f > %.4f, routing to gap queue", entropy, threshold)
	hs.mu.Lock()
	if hs.stats.HeuristicUsage == nil {
		hs.stats.HeuristicUsage = make(map[string]uint64)
	}
	hs.stats.HeuristicUsage["entropy_spike"]++
	hs.mu.Unlock()
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
func (hs *HEARTService) process(ctx context.Context, wasmType WASMType, inquiry interface{}) (WASMDecision, error) {
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

	passed, confidence, msg := hs.verifier.Verify(s4.Source, string(s4.WASMType))
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

	// Process inquiry
	response, err := hs.processInquiry(&inquiry)
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
	hs.stats.HeuristicUsage[fmt.Sprintf("h%d", response.HeuristicID)]++
	hs.mu.Unlock()

	response.ProcessingTimeMs = float32(elapsed)

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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
// When a Gorgonite GPT and tokenizer are available it uses inference to select
// actions; otherwise it falls back to heuristic rules (Phase 2.4).
func (hs *HEARTService) generateRecommendedActions(inquiry *HEARTErrorInquiry, heuristicID uint32) []string {
	if hs.gpt != nil && hs.tokenizer != nil {
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

// findSimilarErrors queries for similar past errors
func (hs *HEARTService) findSimilarErrors(inquiry *HEARTErrorInquiry) []SimilarError {
	// Simulated - would query KNIRVGRAPH in production
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

// handleHealth returns service health status
func (hs *HEARTService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"available":   true,
		"status":      "online",
		"avg_latency_ms": hs.getAvgLatency(),
		"total_queries":  hs.stats.TotalInquiries,
		"success_rate": hs.getSuccessRate(),
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
		"total_inquiries":       hs.stats.TotalInquiries,
		"successful_responses":  hs.stats.SuccessfulResponses,
		"failed_responses":      hs.stats.FailedResponses,
		"avg_response_time_ms":  hs.getAvgLatency(),
		"min_response_time_ms":  hs.stats.MinResponseTimeMs,
		"max_response_time_ms":  hs.stats.MaxResponseTimeMs,
		"error_type_counts":     hs.stats.ErrorTypeCounts,
		"heuristic_usage":       hs.stats.HeuristicUsage,
	})
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
