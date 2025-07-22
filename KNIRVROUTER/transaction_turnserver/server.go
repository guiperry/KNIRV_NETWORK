package transaction_turnserver

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pion/turn/v2"
)

// Server represents a TURN server instance
type Server struct {
	udpListener net.PacketConn
	tcpListener net.Listener
	turnServer  *turn.Server
	txPool      TxSubmitter // Interface for transaction submission
	mu          sync.Mutex
	running     bool
}

// TxSubmitter is an interface for submitting transactions
// This allows decoupling from the specific blockchain implementation
type TxSubmitter interface {
	SubmitTurnSessionTx(sessionData map[string]interface{}) error
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
func NewServer(udpPort, tcpPort int, txPool TxSubmitter) (*Server, error) {
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
	}

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
