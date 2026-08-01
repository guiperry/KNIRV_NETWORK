package graphrag

// Server exposes the in-process graphrag-rs engine over an HTTP API bound to
// a Unix domain socket.
//
// graphrag-rs is linked into KNIRVSERVER directly via CGo (graphrag.go) and
// is never spawned as a subprocess — Init()/Query()/IndexDocument() etc. all
// call straight into this process's own linked copy of the Rust static
// library. backend_server (KNIRV_CORP) runs as a *separate* OS process
// (extracted from //go:embed bin/backend_server and exec'd — see
// KNIRVSERVER's main.go), so a direct Go import cannot reach this engine:
// backend_server would link its own, independently-uninitialized copy of
// the same static library. This socket is the only bridge between the two,
// mirroring the pattern already used for Validation Chain and Transaction
// Chain (see pkg/embedded/validationchain, pkg/embedded/transactionchain) —
// KNIRVSERVER owns the resource's lifecycle; other processes only ever hold
// an HTTP client dialing this socket. Client (client.go) is that dialer.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// Server wraps the Unix-socket HTTP listener for the embedded graphrag
// engine.
type Server struct {
	listener net.Listener
	httpSrv  *http.Server
}

// StartServer binds an HTTP server to socketPath and serves it in the
// background. Init() must have already succeeded — handlers call directly
// into the package-level FFI wrappers, which operate on this process's
// single global Rust instance.
func StartServer(ctx context.Context, socketPath string) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return nil, fmt.Errorf("create graphrag socket directory: %w", err)
	}
	_ = os.Remove(socketPath) // clear a stale socket file from a previous run

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on graphrag socket %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("chmod graphrag socket: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/query", handleQuery)
	mux.HandleFunc("/index-with-result", handleIndexWithResult)
	mux.HandleFunc("/embed", handleEmbed)

	httpSrv := &http.Server{Handler: recoverMiddleware(mux)}

	s := &Server{listener: listener, httpSrv: httpSrv}

	go func() {
		if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "graphrag socket server error: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	return s, nil
}

// Close shuts down the HTTP server and removes the socket file.
func (s *Server) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.httpSrv.Shutdown(shutdownCtx)
	if unixAddr, ok := s.listener.Addr().(*net.UnixAddr); ok {
		_ = os.Remove(unixAddr.Name)
	}
	return err
}

// recoverMiddleware guards every request against a panic unwinding out of a
// handler (e.g. a future CGo/Rust panic surfacing as a Go panic across the
// FFI boundary) and turns it into a 500 instead of taking down the whole
// KNIRVSERVER process — the risk the integration plan's risk register
// flagged and that was never actually mitigated.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Fprintf(os.Stderr, "graphrag socket handler panic: %v\n%s\n", rec, debug.Stack())
				http.Error(w, fmt.Sprintf("internal error: %v", rec), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := HealthCheck(); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type socketQueryRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req socketQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	raw, err := Query(req.Query, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

type socketIndexRequest struct {
	DocID   string `json:"doc_id"`
	Content string `json:"content"`
}

func handleIndexWithResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req socketIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	raw, err := IndexDocumentWithResult(req.DocID, []byte(req.Content))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

type socketEmbedRequest struct {
	Texts []string `json:"texts"`
}

func handleEmbed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req socketEmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	embeddings, err := EmbedTexts(req.Texts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payload, err := json.Marshal(embeddings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}
