package enroll

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
	"go.uber.org/zap"
)

// Step names for the DEC-2 boot-node enrollment order: transfer-first, then
// bond, then register (transfer-first-bond-then-register). The server refuses a
// register that has not been preceded by a transfer and a bond for the same
// enrollment session.
const (
	StepTransfer = "transfer"
	StepBond     = "bond"
	StepRegister = "register"
)

// IssuePath is the internal-only challenge issuance route. It deliberately does
// not sit under /enroll/, so KNIRVGATEWAY's public POST /enroll/ proxy never
// forwards it.
const IssuePath = "/internal/approval/issue"

// orderedSteps is the canonical enrollment order the CLI must follow.
var orderedSteps = []string{StepTransfer, StepBond, StepRegister}

// Server is the D14 server half: it accepts an enrollment request carrying a
// signed challenge, root approval, and nonce, and enforces the DEC-2 ordering
// (transfer -> bond -> register) plus replay prevention via a NonceStore.
//
// In production it is served over HTTP (see RegisterRoutes); tests drive it
// in-process through HandleStep.
type Server struct {
	verifier   Verifier
	store      signing.NonceStore
	rootPubKey string
	operator   string
	log        *zap.Logger

	expectedChainID string
	registrations   RegistrationStore
	admitter        ValidatorAdmitter
	issuer          *Issuer

	mu         sync.Mutex
	steps      map[string]map[string]bool // session -> step -> completed
	order      map[string][]string        // session -> sequence of completed steps
	regist     map[string]bool            // session -> registered
	usedNonces map[string]bool

	accepted   uint64
	rejections map[string]uint64
}

// NewServer constructs the enrollment server with the given verifier, nonce
// store, and trusted root identity/operator (which MUST come from configuration,
// never from the request).
func NewServer(verifier Verifier, store signing.NonceStore, rootPubKey, operator string, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	return &Server{
		verifier:   verifier,
		store:      store,
		rootPubKey: rootPubKey,
		operator:   operator,
		log:        log,
		steps:      map[string]map[string]bool{},
		order:      map[string][]string{},
		regist:     map[string]bool{},
		usedNonces: map[string]bool{},
		rejections: map[string]uint64{},
	}
}

// ConfigureAdmission enables automatic validator-set admission. expectedChainID
// is the chain the node belongs to (empty means use the request's chain ID, and
// the backend admission service still enforces its own configured chain).
// registrations is the approver-owned registration->validator binding store; the
// admitter writes the verified entry to the failover validator set.
func (s *Server) ConfigureAdmission(expectedChainID string, registrations RegistrationStore, admitter ValidatorAdmitter) *Server {
	s.expectedChainID = strings.TrimSpace(expectedChainID)
	s.registrations = registrations
	s.admitter = admitter
	return s
}

// WithIssuer requires every admitted enrollment to redeem a nonce that this server
// issued (see Issuer). Admission without an issuer fails closed.
func (s *Server) WithIssuer(issuer *Issuer) *Server {
	s.issuer = issuer
	return s
}

// Stats returns a snapshot of accepted/rejected enrollment outcomes.
func (s *Server) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := Stats{Accepted: s.accepted, Rejections: make(map[string]uint64, len(s.rejections))}
	for k, v := range s.rejections {
		snapshot.Rejections[k] = v
	}
	return snapshot
}

func (s *Server) reject(step, reason, message string) stepResponse {
	s.rejections[reason]++
	s.log.Warn("enrollment rejected", zap.String("step", step), zap.String("reason", reason), zap.String("error", message))
	return stepResponse{OK: false, Step: step, Error: message}
}

// sessionKey scopes enrollment state by chain and operator.
func sessionKey(chainID, operator string) string {
	return chainID + "\x00" + operator
}

// StepRequest is the body accepted by every step endpoint. Transfer and bond
// carry only the session identity; register additionally carries the approved
// enrollment and the nonce to be consumed.
type StepRequest struct {
	ChainID    string                      `json:"chain_id"`
	Operator   string                      `json:"operator"`
	Nonce      string                      `json:"nonce,omitempty"`
	Enrollment *signing.BootnodeEnrollment `json:"enrollment,omitempty"`
}

type stepResponse struct {
	OK    bool     `json:"ok"`
	Step  string   `json:"step,omitempty"`
	Order []string `json:"order,omitempty"`
	Error string   `json:"error,omitempty"`
}

// HandleStep records completion of one enrollment step for a session and, for
// register, runs the full D14 verification. It returns an error value suitable
// for HTTP translation; callers should use RecordStep.
func (s *Server) RecordStep(step string, req StepRequest, now time.Time) stepResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.ChainID == "" || req.Operator == "" {
		return stepResponse{OK: false, Step: step, Error: "chain_id and operator are required"}
	}
	key := sessionKey(req.ChainID, req.Operator)

	switch step {
	case StepTransfer, StepBond:
		if s.steps[key] == nil {
			s.steps[key] = map[string]bool{}
		}
		if s.steps[key][step] {
			return s.reject(step, "step_already_completed", "step already completed")
		}
		s.steps[key][step] = true
		s.order[key] = append(s.order[key], step)
		return stepResponse{OK: true, Step: step, Order: s.order[key]}

	case StepRegister:
		// Enforce transfer-first-bond-then-register.
		if !s.steps[key][StepTransfer] {
			return s.reject(step, "missing_transfer", "register requires transfer first")
		}
		if !s.steps[key][StepBond] {
			return s.reject(step, "missing_bond", "register requires bond before register")
		}
		if s.regist[key] {
			return s.reject(step, "already_registered", "already registered")
		}
		if req.Enrollment == nil {
			return s.reject(step, "missing_enrollment", "missing enrollment")
		}
		if s.admitter != nil {
			if reason, err := s.admitLocked(*req.Enrollment, req, now); err != nil {
				return s.reject(step, reason, err.Error())
			}
		} else {
			valid, code, err := s.verifyEnrollment(*req.Enrollment, req.Nonce, now)
			if err != nil {
				return s.reject(step, "verification_error", "verification error: "+err.Error())
			}
			if !valid {
				return s.reject(step, rejectionReason(code), "enrollment rejected (core code "+codeOrNone(code)+")")
			}
		}
		s.regist[key] = true
		s.accepted++
		s.order[key] = append(s.order[key], step)
		return stepResponse{OK: true, Step: step, Order: s.order[key]}

	default:
		return s.reject(step, "unknown_step", "unknown step")
	}
}

// verifyEnrollment runs the configured verifier and returns the core refusal code
// when the verifier reports one.
func (s *Server) verifyEnrollment(e signing.BootnodeEnrollment, nonce string, now time.Time) (bool, string, error) {
	if coded, ok := s.verifier.(CodedVerifier); ok {
		return coded.VerifyCoded(context.Background(), e, nonce, s.rootPubKey, s.operator, s.store, now)
	}
	valid, err := s.verifier.Verify(context.Background(), e, nonce, s.rootPubKey, s.operator, s.store, now)
	return valid, "", err
}

// codeOrNone renders a core refusal code for an error message.
func codeOrNone(code string) string {
	if code == "" {
		return "none"
	}
	return code
}

// admitLocked authenticates the enrollment without consuming its nonce, checks
// the chain/registration/key binding, appends the validator through the admission
// service, and only then durably consumes the nonce. Holding s.mu across the
// whole sequence serializes register attempts in this process; the backend
// admission service is itself idempotent, so a cross-process retry cannot create
// a duplicate validator entry.
func (s *Server) admitLocked(enrollment signing.BootnodeEnrollment, req StepRequest, now time.Time) (string, error) {
	if s.registrations == nil {
		return "admission_unconfigured", errAdmissionUnconfigured
	}
	authenticator, ok := s.verifier.(Authenticator)
	if !ok {
		return "admission_unconfigured", errDeferredAuthUnsupported
	}
	var nonce string
	var valid bool
	var code string
	var err error
	if coded, ok := s.verifier.(CodedAuthenticator); ok {
		nonce, valid, code, err = coded.AuthenticateCoded(context.Background(), enrollment, s.rootPubKey, s.operator, now)
	} else {
		nonce, valid, err = authenticator.Authenticate(context.Background(), enrollment, s.rootPubKey, s.operator, now)
	}
	if err != nil {
		return "verification_error", err
	}
	if !valid || nonce == "" {
		return rejectionReason(code), errEnrollmentRejected
	}
	if s.usedNonces[nonce] {
		return "replayed", errReplayed
	}

	parsed, err := signing.ParseMessageEnvelope(enrollment.Challenge.Envelope)
	if err != nil {
		return "invalid_challenge", err
	}
	if req.ChainID != parsed.ChainID {
		return "wrong_chain", errWrongChain
	}
	if s.expectedChainID != "" && s.expectedChainID != parsed.ChainID {
		return "wrong_chain", errWrongChain
	}

	claims, err := ParseEnrollmentClaims(parsed.Payload)
	if err != nil {
		return "unknown_registration", err
	}
	registration, ok, err := s.registrations.Lookup(context.Background(), claims.RegistrationID)
	if err != nil {
		return "unknown_registration", err
	}
	if !ok {
		return "unknown_registration", errUnknownRegistration
	}
	if registration.ChainID != "" && registration.ChainID != parsed.ChainID {
		return "wrong_chain", errWrongChain
	}

	validatorID, err := signing.Address(enrollment.Challenge.PublicKey, signing.DefaultAddressPrefix)
	if err != nil {
		return "invalid_public_key", err
	}
	if !strings.EqualFold(validatorID, registration.ValidatorID) {
		return "validator_id_mismatch", errValidatorIDMismatch
	}
	if claims.ValidatorID != "" && !strings.EqualFold(validatorID, claims.ValidatorID) {
		return "validator_id_mismatch", errValidatorIDMismatch
	}

	// The nonce must be one this server issued for this registration and chain,
	// still unexpired and unredeemed. Check does not change issuance state, so a
	// rejection here leaves a valid nonce redeemable.
	if s.issuer == nil {
		return "issuance_unconfigured", errIssuanceUnconfigured
	}
	if err := s.issuer.Check(nonce, claims.RegistrationID, parsed.ChainID, now); err != nil {
		return issuanceReason(err), err
	}

	// Consume before the membership write: a replay must never be able to alter
	// the validator set, even if the backend would otherwise treat it as an
	// idempotent no-op. An admission failure after this point burns the nonce; the
	// root must issue a fresh approval, which is safer than allowing replay.
	consumed, err := s.store.Consume(nonce)
	if err != nil {
		return "nonce_store_error", err
	}
	if !consumed {
		return "replayed", errReplayed
	}
	s.usedNonces[nonce] = true
	s.issuer.Redeem(nonce)

	admission := ValidatorAdmission{
		ChainID:        parsed.ChainID,
		RegistrationID: claims.RegistrationID,
		ValidatorID:    validatorID,
		PublicKey:      hex.EncodeToString(enrollment.Challenge.PublicKey),
	}
	if err := s.admitter.Admit(context.Background(), admission); err != nil {
		return "admission_failed", err
	}
	return "", nil
}

// issuanceReason maps an Issuer.Check failure to its rejection counter.
func issuanceReason(err error) string {
	switch {
	case errors.Is(err, errNonceNotIssued):
		return "nonce_not_issued"
	case errors.Is(err, errNonceExpired):
		return "nonce_expired"
	case errors.Is(err, errNonceReused):
		return "replayed"
	case errors.Is(err, errNonceRegistration):
		return "nonce_registration_mismatch"
	case errors.Is(err, errWrongChain):
		return "wrong_chain"
	default:
		return "nonce_rejected"
	}
}

// issueChallenge handles POST /internal/approval/issue: it issues a fresh nonce
// for an approved registration. It is served only on the internal socket and is
// deliberately outside the /enroll/ prefix that KNIRVGATEWAY proxies publicly.
func (s *Server) issueChallenge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RegistrationID string `json:"registration_id"`
		ChainID        string `json:"chain_id"`
		TTLSeconds     int64  `json:"ttl_seconds,omitempty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad request: " + err.Error()})
		return
	}
	s.mu.Lock()
	issuer := s.issuer
	operator := s.operator
	expectedChainID := s.expectedChainID
	s.mu.Unlock()

	fail := func(code int, reason string, err error) {
		s.mu.Lock()
		s.rejections["issue_"+reason]++
		s.mu.Unlock()
		s.log.Warn("challenge issuance rejected", zap.String("reason", reason), zap.Error(err))
		writeJSON(w, code, map[string]any{"ok": false, "error": err.Error()})
	}
	if issuer == nil {
		fail(http.StatusServiceUnavailable, "unconfigured", errIssuanceUnconfigured)
		return
	}
	chainID := strings.TrimSpace(req.ChainID)
	if chainID == "" || (expectedChainID != "" && chainID != expectedChainID) {
		fail(http.StatusBadRequest, "wrong_chain", errWrongChain)
		return
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	issued, err := issuer.Issue(r.Context(), req.RegistrationID, time.Now(), ttl)
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, errTooManyPendingIssuance) {
			code = http.StatusTooManyRequests
		}
		fail(code, "rejected", err)
		return
	}
	// The issuer's chain is the request chain (checked above against the node's
	// configured chain), and the operator is the trusted root operator.
	issued.Operator = operator
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "issued": issued})
}

// RecordedOrder returns the completed-step order captured for a session (tests).
func (s *Server) RecordedOrder(chainID, operator string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.order[sessionKey(chainID, operator)]...)
}

// RegisterRoutes wires the three enrollment step endpoints onto mux. It is the
// internal handler registration used by ServeUnix; do NOT call it on the public
// HTTP listener (Rule 2 — enrollment is reachable only via the internal Unix
// socket, never on the public mux).
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	for _, step := range orderedSteps {
		step := step
		mux.HandleFunc("/enroll/"+step, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req StepRequest
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, stepResponse{OK: false, Step: step, Error: "bad request: " + err.Error()})
				return
			}
			resp := s.RecordStep(step, req, time.Now())
			code := http.StatusOK
			if !resp.OK {
				code = http.StatusBadRequest
			}
			writeJSON(w, code, resp)
		})
	}
	mux.HandleFunc(IssuePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.issueChallenge(w, r)
	})
	mux.HandleFunc("/enroll/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, s.Stats())
	})
}

// ServeUnix binds the enrollment server exclusively to an internal Unix-domain
// socket. Enrollment is NOT reachable on the public HTTP listener (Rule 2): call
// this, never RegisterRoutes on the public mux. Lifecycle is owned by ctx.
func (s *Server) ServeUnix(ctx context.Context, socketPath string) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return err
	}
	_ = os.Remove(socketPath) // clear a stale socket file from a previous run
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	srv := &http.Server{Handler: mux}
	if s.log != nil {
		s.log.Info("enrollment socket listening", zap.String("socket", socketPath))
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		_ = os.Remove(socketPath)
	}()
	return srv.Serve(ln)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
