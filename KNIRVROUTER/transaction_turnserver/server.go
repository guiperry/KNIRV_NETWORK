package transaction_turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pion/turn/v2"
)

// Server represents a TURN server instance
type Server struct {
	udpListener net.PacketConn
	tcpListener net.Listener
	turnServer  *turn.Server
	txPool      TxSubmitter  // Interface for transaction submission
	httpServer  *http.Server // HTTP server for API endpoints
	apiPort     int
	mu          sync.Mutex
	running     bool
}

// TxSubmitter is an interface for submitting transactions
// This allows decoupling from the specific blockchain implementation
type TxSubmitter interface {
	SubmitTurnSessionTx(sessionData map[string]interface{}) error
	SubmitNRNMintTx(recipient, amount, reason, proofID string) error
	SubmitConnectivityProofReward(nodeID, proofID string, score float64, amount string) error
	SubmitParticipationReward(nodeID, participationType, amount string) error
	GetMintingStats() map[string]interface{}
}

// authHandler authenticates TURN requests and creates transactions for successful allocations
func (s *Server) authHandler(username, realm string, srcAddr net.Addr) ([]byte, bool) {
	log.Printf("TURN Auth request: user=%s, realm=%s, src=%s", username, realm, srcAddr.String())

	// Simple static authentication - in production, use a proper credential mechanism
	key := turn.GenerateAuthKey(username, realm, "knirvchain-turn-secret")

	// Create transaction data for the session
	sessionData := map[string]interface{}{
		"type":        "TURN_SESSION_START",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"client_addr": srcAddr.String(),
		"username":    username,
		"realm":       realm,
	}

	// Submit transaction asynchronously to avoid blocking auth
	go func() {
		if s.txPool != nil {
			if err := s.txPool.SubmitTurnSessionTx(sessionData); err != nil {
				log.Printf("Error creating TURN transaction: %v", err)
			} else {
				log.Printf("Created blockchain transaction for TURN session: %s", srcAddr.String())
			}
		} else {
			log.Printf("Warning: Transaction pool not available, session not recorded on blockchain")
		}
	}()

	return key, true
}

// NewServer creates a new TURN server instance
func NewServer(udpPort, tcpPort, apiPort int, txPool TxSubmitter) (*Server, error) {
	udpListener, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", udpPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP port %d: %w", udpPort, err)
	}

	tcpListener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", tcpPort))
	if err != nil {
		udpListener.Close() // Clean up UDP listener if TCP fails
		return nil, fmt.Errorf("failed to listen on TCP port %d: %w", tcpPort, err)
	}

	server := &Server{
		udpListener: udpListener,
		tcpListener: tcpListener,
		txPool:      txPool,
		apiPort:     apiPort,
	}

	// Setup HTTP server for API endpoints
	server.setupHTTPServer()

	// Create the TURN server with our auth handler
	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm: "knirvchain.local",
		// Use the server instance's method as the auth handler
		AuthHandler: server.authHandler,
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP("0.0.0.0"), // Use appropriate IP
					Address:      "0.0.0.0",              // Should be external IP ideally
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP("0.0.0.0"), // Use appropriate IP
					Address:      "0.0.0.0",              // Should be external IP ideally
				},
			},
		},
	})

	if err != nil {
		udpListener.Close()
		tcpListener.Close()
		return nil, fmt.Errorf("failed to create TURN server: %w", err)
	}

	server.turnServer = turnServer
	return server, nil
}

// Start begins the TURN server operation
func (s *Server) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("TURN Server already running.")
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting TURN Server...")
	// The pion server runs implicitly when created with listeners.
	// We just need to keep the process alive and handle shutdown.
	log.Printf("TURN Server listening on UDP %s and TCP %s",
		s.udpListener.LocalAddr().String(),
		s.tcpListener.Addr().String())

	// Start HTTP API server
	if err := s.StartHTTPServer(); err != nil {
		log.Printf("Warning: Failed to start HTTP API server: %v", err)
	}
}

// Stop shuts down the TURN server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		log.Println("TURN Server is not running.")
		return nil
	}

	log.Println("Stopping TURN Server...")

	// Stop HTTP API server
	if httpErr := s.StopHTTPServer(); httpErr != nil {
		log.Printf("Error stopping HTTP API server: %v", httpErr)
	}

	err := s.turnServer.Close()

	// Explicitly close listeners too
	if s.udpListener != nil {
		s.udpListener.Close()
	}
	if s.tcpListener != nil {
		s.tcpListener.Close()
	}

	s.running = false
	log.Println("TURN Server stopped.")
	return err
}

// IsRunning returns the current running state of the server
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// setupHTTPServer configures the HTTP server for API endpoints
func (s *Server) setupHTTPServer() {
	mux := http.NewServeMux()

	// Proof and minting endpoints
	mux.HandleFunc("/api/proof/submit", s.handleSubmitProof)
	mux.HandleFunc("/api/proof/status", s.handleProofStatus)
	mux.HandleFunc("/api/mint/nrn", s.handleMintNRN)
	mux.HandleFunc("/api/mint/reward", s.handleMintReward)
	mux.HandleFunc("/api/stats/minting", s.handleMintingStats)
	mux.HandleFunc("/api/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.apiPort),
		Handler: mux,
	}
}

// handleSubmitProof handles connectivity proof submissions
func (s *Server) handleSubmitProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NodeID    string  `json:"node_id"`
		ProofID   string  `json:"proof_id"`
		Score     float64 `json:"score"`
		ProofData string  `json:"proof_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Calculate reward amount based on score
	baseReward := 100.0
	rewardAmount := baseReward * (req.Score / 100.0)
	amountStr := fmt.Sprintf("%.0f", rewardAmount*1e18) // Convert to wei

	// Submit connectivity proof reward
	err := s.txPool.SubmitConnectivityProofReward(req.NodeID, req.ProofID, req.Score, amountStr)
	if err != nil {
		log.Printf("Error submitting connectivity proof reward: %v", err)
		http.Error(w, "Failed to submit proof reward", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":       true,
		"proof_id":      req.ProofID,
		"reward_amount": amountStr,
		"message":       "Connectivity proof submitted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProofStatus handles proof status queries
func (s *Server) handleProofStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	proofID := r.URL.Query().Get("proof_id")
	if proofID == "" {
		http.Error(w, "proof_id parameter required", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would query the blockchain for proof status
	response := map[string]interface{}{
		"proof_id":  proofID,
		"status":    "verified",
		"timestamp": time.Now(),
		"verified":  true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMintNRN handles direct NRN minting requests
func (s *Server) handleMintNRN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
		Reason    string `json:"reason"`
		ProofID   string `json:"proof_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Submit NRN mint transaction
	err := s.txPool.SubmitNRNMintTx(req.Recipient, req.Amount, req.Reason, req.ProofID)
	if err != nil {
		log.Printf("Error submitting NRN mint transaction: %v", err)
		http.Error(w, "Failed to mint NRN tokens", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"recipient": req.Recipient,
		"amount":    req.Amount,
		"reason":    req.Reason,
		"message":   "NRN tokens minted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMintReward handles participation reward minting
func (s *Server) handleMintReward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NodeID            string `json:"node_id"`
		ParticipationType string `json:"participation_type"`
		Amount            string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Submit participation reward
	err := s.txPool.SubmitParticipationReward(req.NodeID, req.ParticipationType, req.Amount)
	if err != nil {
		log.Printf("Error submitting participation reward: %v", err)
		http.Error(w, "Failed to mint participation reward", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":            true,
		"node_id":            req.NodeID,
		"participation_type": req.ParticipationType,
		"amount":             req.Amount,
		"message":            "Participation reward minted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMintingStats handles minting statistics requests
func (s *Server) handleMintingStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.txPool.GetMintingStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":      "healthy",
		"turn_server": s.IsRunning(),
		"api_server":  s.httpServer != nil,
		"timestamp":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// StartHTTPServer starts the HTTP API server
func (s *Server) StartHTTPServer() error {
	if s.httpServer == nil {
		return fmt.Errorf("HTTP server not configured")
	}

	log.Printf("Starting HTTP API server on port %d", s.apiPort)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// StopHTTPServer stops the HTTP API server
func (s *Server) StopHTTPServer() error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Stopping HTTP API server...")
	return s.httpServer.Close()
}
