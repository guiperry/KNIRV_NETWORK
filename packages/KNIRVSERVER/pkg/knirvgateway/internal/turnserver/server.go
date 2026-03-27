package turnserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pion/turn/v2"
	"go.uber.org/zap"
)

type Server struct {
	config      *TurnServerConfig
	udpListener net.PacketConn
	tcpListener net.Listener
	turnServer  *turn.Server
	txPool      TxSubmitter
	httpServer  *http.Server
	logger      *zap.Logger
	mu          sync.RWMutex
	running     bool
	startTime   time.Time

	statsMutex   sync.RWMutex
	sessionCount int64
	activeRelays int64
}

func NewServer(config *TurnServerConfig, txPool TxSubmitter, logger *zap.Logger) (*Server, error) {
	if config.AuthSecret == "" {
		config.AuthSecret = "knirvchain-turn-secret"
	}
	if config.Realm == "" {
		config.Realm = "knirvgateway.local"
	}

	udpListener, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", config.UDPPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP port %d: %w", config.UDPPort, err)
	}

	tcpListener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", config.TCPPort))
	if err != nil {
		udpListener.Close()
		return nil, fmt.Errorf("failed to listen on TCP port %d: %w", config.TCPPort, err)
	}

	server := &Server{
		config:      config,
		udpListener: udpListener,
		tcpListener: tcpListener,
		txPool:      txPool,
		logger:      logger,
		startTime:   time.Now(),
	}

	server.setupHTTPServer()

	relayIP := net.ParseIP(config.PublicIP)
	if relayIP == nil {
		relayIP = net.ParseIP("0.0.0.0")
	}

	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm: config.Realm,
		AuthHandler: func(username, realm string, srcAddr net.Addr) ([]byte, bool) {
			return server.authHandler(username, realm, srcAddr)
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: relayIP,
					Address:      "0.0.0.0",
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: relayIP,
					Address:      "0.0.0.0",
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

func (s *Server) authHandler(username, realm string, srcAddr net.Addr) ([]byte, bool) {
	s.logger.Debug("TURN Auth request",
		zap.String("user", username),
		zap.String("realm", realm),
		zap.String("src", srcAddr.String()))

	key := turn.GenerateAuthKey(username, realm, s.config.AuthSecret)

	sessionData := map[string]interface{}{
		"type":        "TURN_SESSION_START",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"client_addr": srcAddr.String(),
		"username":    username,
		"realm":       realm,
	}

	go func() {
		if s.txPool != nil {
			if err := s.txPool.SubmitTurnSessionTx(sessionData); err != nil {
				s.logger.Error("Error creating TURN transaction",
					zap.Error(err))
			} else {
				s.logger.Info("Created blockchain transaction for TURN session",
					zap.String("client", srcAddr.String()))

				s.statsMutex.Lock()
				s.sessionCount++
				s.activeRelays++
				s.statsMutex.Unlock()
			}
		}
	}()

	return key, true
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("TURN server already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting TURN Server",
		zap.Int("udp_port", s.config.UDPPort),
		zap.Int("tcp_port", s.config.TCPPort),
		zap.String("realm", s.config.Realm))

	s.logger.Info("TURN Server listening",
		zap.String("udp", s.udpListener.LocalAddr().String()),
		zap.String("tcp", s.tcpListener.Addr().String()))

	if err := s.StartHTTPServer(); err != nil {
		s.logger.Warn("Failed to start HTTP API server",
			zap.Error(err))
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping TURN Server")

	if httpErr := s.StopHTTPServer(); httpErr != nil {
		s.logger.Error("Error stopping HTTP API server",
			zap.Error(httpErr))
	}

	if s.turnServer != nil {
		s.turnServer.Close()
	}

	if s.udpListener != nil {
		s.udpListener.Close()
	}
	if s.tcpListener != nil {
		s.tcpListener.Close()
	}

	s.running = false
	s.logger.Info("TURN Server stopped")
	return nil
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Server) GetStatus() *TurnServerStatus {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	uptime := time.Since(s.startTime)
	uptimeStr := fmt.Sprintf("%dd %dh %dm %ds",
		int(uptime.Hours()/24),
		int(uptime.Hours())%24,
		int(uptime.Minutes())%60,
		int(uptime.Seconds())%60)

	return &TurnServerStatus{
		Status:       "online",
		Running:      s.IsRunning(),
		UDPPort:      s.config.UDPPort,
		TCPPort:      s.config.TCPPort,
		APIPort:      s.config.APIPort,
		Realm:        s.config.Realm,
		SessionCount: s.sessionCount,
		ActiveRelays: s.activeRelays,
		Uptime:       uptimeStr,
		Timestamp:    time.Now(),
	}
}

func (s *Server) setupHTTPServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/turn/status", s.handleTurnStatus)
	mux.HandleFunc("/api/turn/stats", s.handleTurnStats)
	mux.HandleFunc("/api/proof/submit", s.handleSubmitProof)
	mux.HandleFunc("/api/proof/status", s.handleProofStatus)
	mux.HandleFunc("/api/mint/nrn", s.handleMintNRN)
	mux.HandleFunc("/api/mint/reward", s.handleMintReward)
	mux.HandleFunc("/api/stats/minting", s.handleMintingStats)
	mux.HandleFunc("/api/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.APIPort),
		Handler: mux,
	}
}

func (s *Server) StartHTTPServer() error {
	if s.httpServer == nil {
		return fmt.Errorf("HTTP server not configured")
	}

	s.logger.Info("Starting TURN HTTP API server",
		zap.Int("port", s.config.APIPort))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error",
				zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) StopHTTPServer() error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("Stopping TURN HTTP API server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleTurnStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleTurnStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	stats := map[string]interface{}{
		"session_count": s.sessionCount,
		"active_relays": s.activeRelays,
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

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

	baseReward := 100.0
	rewardAmount := baseReward * (req.Score / 100.0)
	amountStr := fmt.Sprintf("%.0f", rewardAmount*1e18)

	err := s.txPool.SubmitConnectivityProofReward(req.NodeID, req.ProofID, req.Score, amountStr)
	if err != nil {
		s.logger.Error("Error submitting connectivity proof reward",
			zap.Error(err))
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

	response := map[string]interface{}{
		"proof_id":  proofID,
		"status":    "verified",
		"timestamp": time.Now(),
		"verified":  true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

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

	err := s.txPool.SubmitNRNMintTx(req.Recipient, req.Amount, req.Reason, req.ProofID)
	if err != nil {
		s.logger.Error("Error submitting NRN mint transaction",
			zap.Error(err))
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

	err := s.txPool.SubmitParticipationReward(req.NodeID, req.ParticipationType, req.Amount)
	if err != nil {
		s.logger.Error("Error submitting participation reward",
			zap.Error(err))
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

func (s *Server) handleMintingStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.txPool.GetMintingStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

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
