// Package seedwitness exposes the durable root-assertion witness queue over a
// Unix socket. It is deliberately cgo-free: it depends only on the cgo-free
// rootwitness package and a WitnessStore interface, never on the CUDA-backed
// storage writer. That keeps the server and its negative tests linkable in the
// normal non-CUDA CI target (audit finding 4); the concrete store is a
// *storage.DualSeedWriter, wired in cmd/data-seeder.
package seedwitness

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	rootwitness "github.com/lab/hasher/data-seeder/pkg/rootwitness"
)

// SocketPath returns the canonical seed-witness Unix socket path the server binds,
// matching pkg/evidence's DefaultWitnessSocketPath so the verifier's Witnesser (which
// dials DefaultWitnessSocketPath) can reach this server end to end. appDataDir is
// KNIRV_APP_DATA_DIR; when empty the server falls back to /var/lib/knirvserver, the
// SAME default pkg/evidence.appDataDir() uses (pi_3_9 socket-path alignment). The two
// fallbacks MUST stay identical: if you change one, change the other, or the verifier
// cannot reach the seeder when KNIRV_APP_DATA_DIR is unset.
func SocketPath(appDataDir string) string {
	if strings.TrimSpace(appDataDir) == "" {
		appDataDir = "/var/lib/knirvserver"
	}
	return filepath.Join(appDataDir, "sockets", "seed-witness.sock")
}

// RootAssertionView is the cgo-free view of a committed root assertion returned
// by the /commit and /assertion endpoints. The concrete store (storage.DualSeedWriter)
// maps its ledger record onto this type so the server package stays cgo-free.
type RootAssertionView struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Root string `json:"root"`
	Ref  string `json:"ref"`
	Seed string `json:"seed"`
}

// WitnessStore is the cgo-free contract the seedwitness server needs. The concrete
// implementation is *storage.DualSeedWriter; tests use an in-memory fake. Keeping
// this interface here means the server and its tests never import any CUDA/cgo
// package, so they link and run without CUDA (audit finding 4).
type WitnessStore interface {
	SubmitRoot(root []byte, ref string) string
	Pending() []rootwitness.PendingRoot
	CommitRootAssertion(id string, seed []byte) (string, error)
	RootAssertionView(id string) (RootAssertionView, bool)
}

// Server exposes the witness endpoints for the durable root-assertion queue.
type Server struct {
	socketPath string
	writer     WitnessStore
}

// New creates a root witness server serving the given WitnessStore (typically a
// *storage.DualSeedWriter) over socketPath (Unix socket; "" lets the caller bind
// after New via Serve).
func New(socketPath string, store WitnessStore) *Server {
	return &Server{socketPath: socketPath, writer: store}
}

// Handler builds the HTTP handler for the witness endpoints. It is transport
// agnostic so it can be served over a Unix socket or any mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/witness/submit", s.handleSubmit)
	mux.HandleFunc("/witness/pending", s.handlePending)
	mux.HandleFunc("/witness/commit", s.handleCommit)
	mux.HandleFunc("/witness/assertion", s.handleAssertion)
	return mux
}

// Serve binds the configured Unix socket and serves until the listener closes.
func (s *Server) Serve() error {
	if s.socketPath == "" {
		return fmt.Errorf("seedwitness: empty socket path")
	}
	_ = os.Remove(s.socketPath)
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	return http.Serve(l, s.Handler())
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Root []byte `json:"root"`
		Ref  string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if len(req.Root) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty root"})
		return
	}
	id := s.writer.SubmitRoot(req.Root, req.Ref)
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) handlePending(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"pending": s.writer.Pending()})
}

// handleCommit validates the seed before recording it. The central guard lives in
// the store (storage.CommitRootAssertion: 32-byte + real root-PoW), but the server
// also checks up front and returns a precise error for the three forge attempts:
// wrong/unknown root, wrong seed length, or a non-PoW seed (audit finding 2).
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		ID   string `json:"id"`
		Seed string `json:"best_seed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	seed, err := base64.StdEncoding.DecodeString(req.Seed)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seed must be base64"})
		return
	}
	pendList := s.writer.Pending()
	var pend *rootwitness.PendingRoot
	for i := range pendList {
		if pendList[i].ID == req.ID {
			pend = &pendList[i]
			break
		}
	}
	if pend == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "root not pending"})
		return
	}
	if len(seed) != 32 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seed must be 32 bytes"})
		return
	}
	if !rootwitness.VerifyRootPoW(pend.Root, seed) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seed does not satisfy root PoW"})
		return
	}
	id, err := s.writer.CommitRootAssertion(req.ID, seed)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) handleAssertion(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
		return
	}
	view, ok := s.writer.RootAssertionView(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
