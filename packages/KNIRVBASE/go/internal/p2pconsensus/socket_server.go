package p2pconsensus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

// ConsensusHandler processes inbound operations and sync requests from the gateway.
type ConsensusHandler interface {
	HandleOperation(op OperationEnvelope) error
	HandleSyncRequest(req SyncRequest) (*SyncResponse, error)
}

// SocketServer is a lightweight HTTP server on a Unix socket that the gateway calls into.
type SocketServer struct {
	socketPath    string
	networkID     string
	networkSecret string
	server        *http.Server
	handler       ConsensusHandler
	mu            sync.Mutex
	started       bool
	stop          chan struct{}
}

// NewSocketServer creates a new Unix socket server for gateway callbacks.
// networkID/networkSecret authenticate inbound messages from the gateway so an
// arbitrary local process cannot inject CRDT operations without the PSK.
func NewSocketServer(socketPath string, handler ConsensusHandler, networkID, networkSecret string) *SocketServer {
	return &SocketServer{
		socketPath:    socketPath,
		networkID:     networkID,
		networkSecret: networkSecret,
		handler:       handler,
	}
}

// Serve starts the Unix socket HTTP server. The socket is created with
// owner-only permissions (0700) rather than world-writable, and any stale
// socket file is removed before binding so a crash that leaves the socket
// behind does not block restart.
func (s *SocketServer) Serve() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("socket server already started")
	}
	s.started = true
	s.mu.Unlock()

	// Remove existing socket file if present (left over from a previous crash).
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket %s: %w", s.socketPath, err)
	}
	// Restrictive permissions: only the owner may connect to this IPC channel.
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/p2p/received-op", s.handleReceivedOp)
	mux.HandleFunc("/internal/p2p/received-sync-request", s.handleSyncRequest)

	s.server = &http.Server{
		Handler: mux,
	}

	// Best-effort cleanup of the socket file if the server exits unexpectedly.
	go func() {
		<-s.stopChan()
		_ = os.Remove(s.socketPath)
	}()

	return s.server.Serve(listener)
}

// stopChan returns a channel that is closed when Stop is called.
func (s *SocketServer) stopChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop == nil {
		s.stop = make(chan struct{})
	}
	return s.stop
}

// Stop gracefully shuts down the socket server and removes the socket file.
func (s *SocketServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.stop != nil {
		select {
		case <-s.stop:
		default:
			close(s.stop)
		}
	}
	server := s.server
	s.mu.Unlock()

	if server == nil {
		return nil
	}
	err := server.Shutdown(ctx)
	_ = os.Remove(s.socketPath)
	return err
}

// verifyRequest authenticates an inbound gateway request using the network
// secret. When no secret is configured (open network) the check is skipped.
func (s *SocketServer) verifyRequest(r *http.Request, payload []byte) bool {
	if s.networkSecret == "" {
		return true
	}
	sig := r.Header.Get("X-KNIRV-Signature")
	return VerifyMessage(s.networkID, s.networkSecret, sig, payload)
}

func (s *SocketServer) handleReceivedOp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	if !s.verifyRequest(r, body) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var op OperationEnvelope
	if err := json.Unmarshal(body, &op); err != nil {
		http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
		return
	}
	if s.handler == nil {
		http.Error(w, "no handler registered", http.StatusInternalServerError)
		return
	}
	if err := s.handler.HandleOperation(op); err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *SocketServer) handleSyncRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	if !s.verifyRequest(r, body) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req SyncRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
		return
	}
	if s.handler == nil {
		http.Error(w, "no handler registered", http.StatusInternalServerError)
		return
	}
	resp, err := s.handler.HandleSyncRequest(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
