package daemon

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"ulora/internal/api"
	"ulora/internal/compiler"
	"ulora/internal/config"
)

const socketProtocol = "unix"

type Server struct {
	cfg       *config.Config
	compiler  *compiler.Compiler
	ln        net.Listener
	server    *http.Server
	startTime time.Time
}

func NewServer(cfg *config.Config) (*Server, error) {
	if err := os.RemoveAll(cfg.SocketPath); err != nil {
		return nil, fmt.Errorf("remove stale socket: %w", err)
	}

	ln, err := net.Listen(socketProtocol, cfg.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on socket %s: %w", cfg.SocketPath, err)
	}

	mux := http.NewServeMux()
	s := &Server{
		cfg:       cfg,
		compiler:  compiler.NewCompiler(cfg),
		ln:        ln,
		startTime: time.Now(),
		server:    &http.Server{Handler: mux},
	}

	mux.HandleFunc("/ulora/v1/health", s.handleHealth)
	mux.Handle("/ulora/v1/validate-manifest", s.authMiddleware(http.HandlerFunc(s.handleValidateManifest)))
	mux.Handle("/ulora/v1/compile-cluster", s.authMiddleware(http.HandlerFunc(s.handleCompileCluster)))
	mux.Handle("/ulora/v1/transfer", s.authMiddleware(http.HandlerFunc(s.handleTransfer)))
	// Per-request adapter projection: a caller names a bundle and a target model
	// and gets the LoRA A/B matrices. Authenticated like the other routes that
	// do real work.
	mux.Handle("/ulora/v1/bind", s.authMiddleware(http.HandlerFunc(s.handleBind)))
	// Connectors are derived once per model and cached, so a skill binds to any
	// model the cache can serve without the bundle knowing the model existed.
	mux.Handle("/ulora/v1/connectors/derive", s.authMiddleware(http.HandlerFunc(s.handleDeriveConnector)))

	return s, nil
}

func (s *Server) SetEnginesDir(dir string) {
	s.compiler.SetEnginesDir(dir)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.AuthToken)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Serve() error {
	log.Printf("ulorad listening on %s", s.cfg.SocketPath)
	if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	if s.ln != nil {
		s.ln.Close()
	}
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := api.HealthResponseNow()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleValidateManifest(w http.ResponseWriter, r *http.Request) {
	var m api.Manifest
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if err := m.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "errors": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"valid": true, "manifest": m})
}

func (s *Server) handleCompileCluster(w http.ResponseWriter, r *http.Request) {
	var req api.CompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if len(req.Dataset) == 0 {
		http.Error(w, "dataset cannot be empty", http.StatusBadRequest)
		return
	}
	if len(req.TargetModels) == 0 {
		http.Error(w, "target_models cannot be empty", http.StatusBadRequest)
		return
	}

	rank := req.Rank
	if rank <= 0 {
		rank = 16
	}
	alpha := req.Alpha
	if alpha <= 0 {
		alpha = 32.0
	}
	epochs := req.NumEpochs
	if epochs <= 0 {
		epochs = 50
	}
	lr := req.LearningRate
	if lr <= 0 {
		lr = 0.01
	}

	provenance := api.CompileProvenance{
		SourceID:         req.Provenance.SourceID,
		SourceDatasetIDs: req.Provenance.SourceDatasetIDs,
		Extensions:       req.Provenance.Extensions,
	}

	clusterID := provenance.SourceID
	if clusterID == "" {
		clusterID = fmt.Sprintf("cluster-%d", time.Now().UnixNano())
	}

	resp, err := s.compiler.CompileCluster(clusterID, req.Dataset, req.TargetModels, provenance, compiler.CompileParams{
		Rank:         rank,
		Alpha:        alpha,
		LearningRate: lr,
		Epochs:       epochs,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("compile failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req api.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.SourceBundlePath == "" {
		http.Error(w, "source_bundle_path is required", http.StatusBadRequest)
		return
	}
	if len(req.TargetModels) == 0 {
		http.Error(w, "target_models cannot be empty", http.StatusBadRequest)
		return
	}

	resp, err := s.compiler.Transfer(req.SourceBundlePath, req.TargetModels)
	if err != nil {
		http.Error(w, fmt.Sprintf("transfer failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
