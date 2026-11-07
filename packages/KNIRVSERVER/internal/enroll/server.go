package enroll

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

	mu     sync.Mutex
	steps  map[string]map[string]bool // session -> step -> completed
	order  map[string][]string        // session -> sequence of completed steps
	regist map[string]bool            // session -> registered
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
	}
}

// sessionKey scopes enrollment state by chain and operator.
func sessionKey(chainID, operator string) string {
	return chainID + "\x00" + operator
}

// StepRequest is the body accepted by every step endpoint. Transfer and bond
// carry only the session identity; register additionally carries the approved
// enrollment and the nonce to be consumed.
type StepRequest struct {
	ChainID     string                 `json:"chain_id"`
	Operator    string                 `json:"operator"`
	Nonce       string                 `json:"nonce,omitempty"`
	Enrollment  *signing.BootnodeEnrollment `json:"enrollment,omitempty"`
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
			return stepResponse{OK: false, Step: step, Error: "step already completed"}
		}
		s.steps[key][step] = true
		s.order[key] = append(s.order[key], step)
		return stepResponse{OK: true, Step: step, Order: s.order[key]}

	case StepRegister:
		// Enforce transfer-first-bond-then-register.
		if !s.steps[key][StepTransfer] {
			return stepResponse{OK: false, Step: step, Error: "register requires transfer first"}
		}
		if !s.steps[key][StepBond] {
			return stepResponse{OK: false, Step: step, Error: "register requires bond before register"}
		}
		if s.regist[key] {
			return stepResponse{OK: false, Step: step, Error: "already registered"}
		}
		if req.Enrollment == nil {
			return stepResponse{OK: false, Step: step, Error: "missing enrollment"}
		}
		valid, err := s.verifier.Verify(context.Background(), *req.Enrollment, req.Nonce, s.rootPubKey, s.operator, s.store, now)
		if err != nil {
			return stepResponse{OK: false, Step: step, Error: "verification error: " + err.Error()}
		}
		if !valid {
			return stepResponse{OK: false, Step: step, Error: "enrollment rejected (unsigned, missing approval, or replayed)"}
		}
		s.regist[key] = true
		s.order[key] = append(s.order[key], step)
		return stepResponse{OK: true, Step: step, Order: s.order[key]}

	default:
		return stepResponse{OK: false, Step: step, Error: "unknown step"}
	}
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
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
