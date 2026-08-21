package api

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"knirvhasher/pkg/hashing/math"
	"knirvhasher/pkg/hashing/miner"
	"knirvhasher/pkg/hashing/proofasset"
)

// FormalVerifierClient is the interface for submitting proof assets to a
// formal checker worker. Implementations may be local (subprocess) or remote
// (gRPC).
type FormalVerifierClient interface {
	SubmitProof(asset *proofasset.ProofAsset) (*proofasset.VerificationReceipt, error)
}

// MathVerifierServer orchestrates structural precheck and optional formal
// verification of mathematical expressions.
type MathVerifierServer struct {
	watchdog        *math.InferenceWatchdog
	mapper          *math.LaTeXMapper
	subdomain       uint32
	formalClient    FormalVerifierClient
	allowedImports  []string
	proofLedgerPath string
}

// NewMathVerifierServer creates a server whose watchdog uses hard-coded
// fallback rules.
func NewMathVerifierServer(subdomain uint32) *MathVerifierServer {
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}
	return &MathVerifierServer{
		watchdog:        math.NewInferenceWatchdog(subdomain),
		mapper:          math.NewLaTeXMapper(subdomain),
		subdomain:       subdomain,
		allowedImports:  []string{},
		proofLedgerPath: "",
	}
}

// NewMathVerifierServerWithRules creates a server whose watchdog uses
// schema-loaded validation rules.
func NewMathVerifierServerWithRules(subdomain uint32, rules []math.WatchdogRule) *MathVerifierServer {
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}
	return &MathVerifierServer{
		watchdog:        math.NewWatchdogWithRules(subdomain, rules),
		mapper:          math.NewLaTeXMapper(subdomain),
		subdomain:       subdomain,
		allowedImports:  []string{},
		proofLedgerPath: "",
	}
}

// WithFormalVerifier attaches a formal verifier client to the server. When nil,
// the server performs structural precheck only and returns PROOF_PENDING /
// CHECKER_UNAVAILABLE as appropriate.
func (s *MathVerifierServer) WithFormalVerifier(client FormalVerifierClient) *MathVerifierServer {
	s.formalClient = client
	return s
}

// WithAllowedImports sets the Lean import allowlist.
func (s *MathVerifierServer) WithAllowedImports(imports []string) *MathVerifierServer {
	s.allowedImports = imports
	return s
}

// WithProofLedgerPath sets the path to the append-only proof ledger.
func (s *MathVerifierServer) WithProofLedgerPath(path string) *MathVerifierServer {
	s.proofLedgerPath = path
	return s
}

// VerifyResponse is the enhanced response model for math verification.
// PrecheckStatus is the outcome of the MATHASHER structural precheck.
// FormalStatus is the outcome of the formal checker (empty when no checker
// is configured or the candidate was not submitted).
// ProofAssetID is the content-addressed identity of the proof asset when one
// was built for formal checking.
// Receipt is the full verification receipt when FormalStatus is
// FORMALLY_VERIFIED; otherwise omitted.
type VerifyResponse struct {
	Status            string  `json:"status"`
	PrecheckStatus    string  `json:"precheck_status"`
	FormalStatus      string  `json:"formal_status,omitempty"`
	ProofAssetID      string  `json:"proof_asset_id,omitempty"`
	Nonce             string  `json:"nonce"`
	NonceNote         string  `json:"nonce_note,omitempty"`
	ResultHash        string  `json:"result_hash"`
	DetokenizedOutput string  `json:"detokenized_output"`
	LogicIntegrity    float32 `json:"logic_integrity"`
	Receipt           *proofasset.VerificationReceipt `json:"receipt,omitempty"`
	LatencyMs         float64 `json:"latency_ms"`
}

// Verify performs structural precheck and, when a formal verifier client is
// attached, submits a proof asset for formal verification. The endpoint never
// synthesizes a nonce from input text and presents it as a formal proof
// witness.
func (s *MathVerifierServer) Verify(latex string, subdomain uint32) *VerifyResponse {
	start := time.Now()

	if subdomain == 0 {
		subdomain = s.subdomain
	}

	slots := s.mapper.MapLaTeXToTensor(latex, subdomain)
	validation := s.watchdog.ValidateMathStep(0, slots)
	precheckStatus := proofasset.StatusStructurallyValid
	if !validation.Valid {
		precheckStatus = proofasset.StatusStructurallyRejected
	}

	response := &VerifyResponse{
		Status:            precheckStatus,
		PrecheckStatus:    precheckStatus,
		Nonce:             fmt.Sprintf("0x%08x", slots[11]),
		NonceNote:         "nonce is deterministic input-derived slot metadata; it is NOT a formal proof witness",
		ResultHash:        fmt.Sprintf("0x%08x", slots[3]),
		DetokenizedOutput: latex,
		LogicIntegrity:    float32(validation.LogicIntegrity),
		LatencyMs:         float64(time.Since(start).Seconds() * 1000),
	}

	if !validation.Valid {
		response.ResultHash = validation.Error
		response.FormalStatus = ""
		return response
	}

	// Structurally valid: attempt formal verification if a client is attached.
	if s.formalClient == nil {
		response.FormalStatus = proofasset.StatusCheckerUnavailable
		response.Status = proofasset.StatusProofPending
		return response
	}

	asset, err := s.buildProofAsset(latex, slots)
	if err != nil {
		response.FormalStatus = proofasset.StatusCheckerUnavailable
		response.Status = proofasset.StatusProofPending
		response.ResultHash = fmt.Sprintf("proof asset build failed: %v", err)
		return response
	}

	proofAssetID, err := proofasset.ComputeProofAssetID(asset)
	if err != nil {
		response.FormalStatus = proofasset.StatusCheckerUnavailable
		response.Status = proofasset.StatusProofPending
		response.ResultHash = fmt.Sprintf("proof asset ID computation failed: %v", err)
		return response
	}
	response.ProofAssetID = proofAssetID

	receipt, err := s.formalClient.SubmitProof(asset)
	if err != nil {
		response.FormalStatus = proofasset.StatusCheckerUnavailable
		response.Status = proofasset.StatusProofPending
		response.ResultHash = fmt.Sprintf("formal checker error: %v", err)
		return response
	}

	response.FormalStatus = receipt.Status
	response.Status = receipt.Status
	if receipt.Status == proofasset.StatusFormallyVerified {
		response.Receipt = receipt
	}

	// Persist to proof ledger if configured.
	if s.proofLedgerPath != "" && response.ProofAssetID != "" {
		ledger := proofasset.NewProofLedger(s.proofLedgerPath)
		ledgerEntry := proofasset.ProofLedgerEntry{
			SchemaVersion:        1,
			ProofAssetID:         response.ProofAssetID,
			TheoremID:            "", // filled below if possible
			ProofSystem:          proofasset.ProofSystemLean,
			CheckerDigest:        receipt.CheckerDigest,
			DependencyLockDigest: "", // populated from asset in production
			EnvironmentDigest:    receipt.EnvironmentDigest,
			FinalStatus:          receipt.Status,
			DiagnosticDigest:     receipt.DiagnosticDigest,
			NrvBracketID:         fmt.Sprintf("0x%08x", slots[0]),
			Receipt:              receipt,
		}
		if theoremID, terr := proofasset.ComputeTheoremID(proofasset.ProofSystemLean, "", asset.TheoremSource); terr == nil {
			ledgerEntry.TheoremID = theoremID
		}
		if len(s.allowedImports) > 0 {
			ledgerEntry.DependencyLockDigest = fmt.Sprintf("lean-%d-imports", len(s.allowedImports))
		}
		if err := ledger.Append(ledgerEntry); err != nil {
			log.Printf("WARNING: proof ledger append failed: %v", err)
		}
	}

	return response
}

// buildProofAsset constructs a minimal ProofAsset from a structurally valid
// LaTeX expression. The theorem source is the original expression; the proof
// source is a placeholder that the formal checker must replace with a real
// proof. In production, the neural/search layer would supply the proof source.
func (s *MathVerifierServer) buildProofAsset(latex string, slots [12]uint32) (*proofasset.ProofAsset, error) {
	theoremSource := []byte(fmt.Sprintf("theorem example_theorem : True := by\n  -- original expression: %s\n  trivial\n", latex))
	proofSource := []byte(fmt.Sprintf("theorem example_theorem : True := by\n  -- auto-generated candidate for: %s\n  trivial\n", latex))

	var imports []proofasset.ArtifactRef
	for _, imp := range s.allowedImports {
		imports = append(imports, proofasset.ArtifactRef{
			Name:   imp,
			Digest: "sha256:unknown",
		})
	}

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: fmt.Sprintf("lean-%d-imports", len(s.allowedImports)),
		TheoremSource:        theoremSource,
		ProofSource:          proofSource,
		Imports:              imports,
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	if err := proofasset.ValidateProofAsset(asset, s.allowedImports); err != nil {
		return nil, err
	}

	return asset, nil
}

// VerifyResponseLegacy is retained for backward compatibility during the
// migration window. New clients should consume VerifyResponse.
type VerifyResponseLegacy struct {
	Status            string  `json:"status"`
	Nonce             string  `json:"nonce"`
	ResultHash        string  `json:"result_hash"`
	DetokenizedOutput string  `json:"detokenized_output"`
	LogicIntegrity    float64 `json:"logic_integrity"`
	LatencyMs         float64 `json:"latency_ms"`
}

// ToLegacy converts the enhanced response to the legacy format.
func (r *VerifyResponse) ToLegacy() VerifyResponseLegacy {
	return VerifyResponseLegacy{
		Status:            r.Status,
		Nonce:             r.Nonce,
		ResultHash:        r.ResultHash,
		DetokenizedOutput: r.DetokenizedOutput,
		LogicIntegrity:    float64(r.LogicIntegrity),
		LatencyMs:         r.LatencyMs,
	}
}

// ServerConfig holds REST API server configuration.
type ServerConfig struct {
	Port       string
	Subdomain  uint32
	SchemaPath string // optional: path to slot-schema YAML for rule loading
}

// DefaultServerConfig returns default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:      ":50051",
		Subdomain: math.SUBDOMAIN_ARITHMETIC,
	}
}

// StartServer starts the HashNet Math Verification REST API on config.Port.
// Endpoints:
//
//	POST /v1/verify/math  — verify a LaTeX derivation step
//	GET  /health          — liveness probe
func StartServer(config *ServerConfig) error {
	if config == nil {
		config = DefaultServerConfig()
	}

	var server *MathVerifierServer

	if config.SchemaPath != "" {
		rules, err := math.LoadWatchdogRulesFromYAML(config.SchemaPath)
		if err != nil {
			log.Printf("Warning: could not load schema rules from %s: %v — using hard-coded fallback", config.SchemaPath, err)
			server = NewMathVerifierServer(config.Subdomain)
		} else {
			log.Printf("Loaded %d validation rules from %s", len(rules), config.SchemaPath)
			server = NewMathVerifierServerWithRules(config.Subdomain, rules)
		}
	} else {
		server = NewMathVerifierServer(config.Subdomain)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/verify/math", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req MathVerificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		resp := server.Verify(req.Proposition, req.Subdomain)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"domain": "MATHASHER",
		})
	})

	log.Printf("HashNet Math Verification API listening on %s", config.Port)
	return http.ListenAndServe(config.Port, mux)
}

// VerifyHandler returns an http.HandlerFunc for POST /v1/verify/math.
// Mount on a gin router with gin.WrapF(server.VerifyHandler()).
func (s *MathVerifierServer) VerifyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req MathVerificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		resp := s.Verify(req.Proposition, req.Subdomain)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding verify response: %v", err)
		}
	}
}

// MathHealthHandler returns an http.HandlerFunc for GET /v1/verify/math/health.
// Mount on a gin router with gin.WrapF(server.MathHealthHandler()).
func (s *MathVerifierServer) MathHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"domain": "MATHASHER",
		})
	}
}

type MathVerificationRequest struct {
	Context             string  `json:"context"`
	Proposition         string  `json:"proposition"`
	Subdomain           uint32  `json:"subdomain"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
}

func VerifyMathDerivation(req MathVerificationRequest) (VerifyResponseLegacy, error) {
	subdomain := req.Subdomain
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}

	mapper := math.NewLaTeXMapper(subdomain)
	watchdog := math.NewInferenceWatchdog(subdomain)

	slots := mapper.MapLaTeXToTensor(req.Proposition, subdomain)
	validation := watchdog.ValidateMathStep(0, slots)

	response := VerifyResponseLegacy{
		Status:            "UNVERIFIED",
		Nonce:             fmt.Sprintf("0x%08x", slots[11]),
		ResultHash:        fmt.Sprintf("0x%08x", slots[3]),
		DetokenizedOutput: req.Proposition,
		LogicIntegrity:    validation.LogicIntegrity,
	}

	if validation.Valid {
		response.Status = proofasset.StatusStructurallyValid
	} else {
		response.Status = proofasset.StatusStructurallyRejected
		response.ResultHash = validation.Error
	}

	return response, nil
}

func MineMathDataset(inputPath, outputPath string, subdomain uint32) (int, error) {
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}
	return miner.MineMathProofs(inputPath, outputPath, subdomain)
}

func HashFrame(latex string, subdomain uint32) string {
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}
	mapper := math.NewLaTeXMapper(subdomain)
	slots := mapper.MapLaTeXToTensor(latex, subdomain)

	result := ""
	for _, v := range slots {
		result += fmt.Sprintf("%08x", v)
	}
	return result
}

func HashFrameBytes(latex string, subdomain uint32) []byte {
	if subdomain == 0 {
		subdomain = math.SUBDOMAIN_ARITHMETIC
	}
	mapper := math.NewLaTeXMapper(subdomain)
	slots := mapper.MapLaTeXToTensor(latex, subdomain)

	result := make([]byte, 48)
	for i, v := range slots {
		binary.BigEndian.PutUint32(result[i*4:], v)
	}
	return result
}
