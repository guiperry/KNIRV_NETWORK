package authority

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Server serves the authority API over a Unix socket. KNIRVSERVER binds the
// socket and backend_server calls it over the socket (BS cannot import internal
// packages, and internal calls use sockets, not URLs).
//
//	POST /authority/authorize     issue a record for an action
//	GET  /authority/records/{hash} fetch a stored record and its issue-time verdict
//	GET  /authority/status        mode, state, counters, anchor
type Server struct {
	Svc    *Service
	Anchor *Anchor // optional
	// Token, when non-empty, is required as "Authorization: Bearer <token>", the
	// same internal token KNIRVSERVER hands its other embedded services. The
	// socket is also created owner-only. The token is a second check, not a substitute.
	Token string
}

const maxRequestBytes = 2 << 20

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/authority/authorize", s.guard(http.MethodPost, s.authorize))
	mux.HandleFunc("/authority/records/", s.guard(http.MethodGet, s.record))
	mux.HandleFunc("/authority/status", s.guard(http.MethodGet, s.status))
	mux.HandleFunc("/authority/session/rotate", s.guard(http.MethodPost, s.rotateSession))
	// Everything else is a JSON 404, so clients never have to parse plain text.
	mux.HandleFunc("/", s.guard("", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}))
	return mux
}

func (s *Server) guard(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		if method != "" && r.Method != method {
			w.Header().Set("Allow", method)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		next(w, r)
	}
}

type authorizeRequest struct {
	Tier   uint8           `json:"tier"`
	Domain string          `json:"domain"`
	Scope  string          `json:"scope"`
	Action json.RawMessage `json:"action"`
}

type authorizeResponse struct {
	Hash      string     `json:"hash"`
	Record    RecordJSON `json:"record"`
	Mode      string     `json:"mode"`
	Verified  bool       `json:"verified"`
	ErrorCode string     `json:"error_code,omitempty"`
	// BreakGlass flags a record the long-term key signed itself (Task 2c.1).
	BreakGlass    bool           `json:"break_glass,omitempty"`
	RegistryEntry *RegistryEntry `json:"registry_entry,omitempty"`
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	var req authorizeRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request: " + err.Error()})
		return
	}
	res, err := s.Svc.Authorize(r.Context(), Action{Tier: req.Tier, Domain: req.Domain, Scope: req.Scope, Body: req.Action})
	if err != nil {
		var refusal *Refusal
		switch {
		case errors.As(err, &refusal):
			status := http.StatusForbidden
			if refusal.Code == RefusedAction {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]string{"error": refusal.Reason, "code": refusal.Code})
		case errors.Is(err, ErrDegraded):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error(), "code": RefusedDegraded})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, authorizeResponse{
		Hash: res.Stored.Record.Hash, Record: res.Stored.Record, Mode: res.Stored.Mode,
		Verified: res.Stored.Verified, ErrorCode: res.Stored.ErrorCode,
		BreakGlass: res.Stored.BreakGlass, RegistryEntry: s.Svc.RegistryEntry(),
	})
}

func (s *Server) record(w http.ResponseWriter, r *http.Request) {
	if s.Svc.Store() == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authority service is degraded", "code": RefusedDegraded})
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/authority/records/")
	if raw, err := hex.DecodeString(hash); err != nil || len(raw) != 32 || hash != strings.ToLower(hash) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hash must be 64 lowercase hex characters"})
		return
	}
	sr, ok := s.Svc.Store().Get(hash)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such record"})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		StoredRecord
		RegistryEntry *RegistryEntry `json:"registry_entry,omitempty"`
		Note          string         `json:"note"`
	}{sr, s.Svc.RegistryEntry(), "issued_at is the service clock and is not authenticated by the record signature; verified/error_code are the verdict at issue time (spec sections 2.3, 5.2)"})
}

// rotateSession forces a new session ceremony now, retiring the current key. It is
// the operator's tool for a suspected key theft (plan test "stolen-session
// revocation by rotation"). It only rotates the routine record-signing session and
// retires the old session key; the Oracle chain author stays the long-term key
// (D10: Oracle-author rotation is deprecated), so no Oracle rotation is involved.
func (s *Server) rotateSession(w http.ResponseWriter, r *http.Request) {
	sm := s.Svc.Sessions()
	if sm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authority service has no session manager", "code": RefusedDegraded})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength != 0 {
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request: " + err.Error()})
			return
		}
	}
	if len(req.Reason) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reason is longer than 200 bytes"})
		return
	}
	_, err := sm.Rotate(r.Context(), "manual: "+req.Reason)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sm.Status(s.Svc.cfg.Now()))
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	st := s.Svc.Status()
	if s.Anchor != nil {
		a := s.Anchor.Status()
		st.Anchor = &a
	}
	writeJSON(w, http.StatusOK, st)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Serve listens on the Unix socket at path until ctx ends. A stale socket file
// is removed, but a socket that still answers means another instance is running
// and Serve refuses rather than steal it.
func (s *Server) Serve(ctx context.Context, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		if conn, derr := net.DialTimeout("unix", path, time.Second); derr == nil {
			conn.Close()
			return fmt.Errorf("authority socket %s is already served by another process", path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale socket: %w", err)
		}
	}
	old := setUmask(0o077) // the socket is created owner-only (mode 0700)
	ln, err := net.Listen("unix", path)
	setUmask(old)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", path, err)
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Minute, IdleTimeout: 2 * time.Minute}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
