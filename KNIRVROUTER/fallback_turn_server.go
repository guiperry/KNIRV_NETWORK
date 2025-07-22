package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"KNIRVCHAIN_GO_Verifyer/config"

	"github.com/gorilla/mux"
	"github.com/pion/turn/v2"
)

// TURNServer represents a TURN server instance with advanced functionality
type TURNServer struct {
	udpListener net.PacketConn
	tcpListener net.Listener
	turnServer  *turn.Server
	mu          sync.Mutex
	running     bool
	router      *mux.Router
}

// authHandler authenticates TURN requests
func (s *TURNServer) authHandler(username, realm string, srcAddr net.Addr) ([]byte, bool) {
	log.Printf("TURN Auth request: user=%s, realm=%s, src=%s", username, realm, srcAddr.String())

	// Generate authentication key - in production, use a proper credential mechanism
	// This is a simple static authentication method
	key := turn.GenerateAuthKey(username, realm, "knirvchain-turn-secret")

	// Log the session information
	log.Printf("TURN session established: client=%s, username=%s", srcAddr.String(), username)

	return key, true
}

// NewTURNServer creates a new TURN server instance
func NewTURNServer() (*TURNServer, error) {
	config.LoadConfig()

	// Configure UDP listener
	udpAddress := fmt.Sprintf("%s:%s", config.AppConfig.TURNAddress, config.AppConfig.TURNPort)
	udpListener, err := net.ListenPacket("udp4", udpAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP address %s: %w", udpAddress, err)
	}

	// Configure TCP listener on the same port
	tcpAddress := fmt.Sprintf("%s:%s", config.AppConfig.TURNAddress, config.AppConfig.TURNPort)
	tcpListener, err := net.Listen("tcp4", tcpAddress)
	if err != nil {
		udpListener.Close() // Clean up UDP listener if TCP fails
		return nil, fmt.Errorf("failed to listen on TCP address %s: %w", tcpAddress, err)
	}

	server := &TURNServer{
		udpListener: udpListener,
		tcpListener: tcpListener,
		router:      mux.NewRouter(),
	}

	// Create the dynamic realm based on the verifyer's chain:// URI
	// Format: chain://{serviceAddress}
	realm := fmt.Sprintf("%s", config.AppConfig.ServiceAddress)
	log.Printf("Setting TURN server realm to: %s", realm)

	// Create the TURN server with our auth handler
	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm:       realm,
		AuthHandler: server.authHandler,
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(config.AppConfig.TURNAddress),
					Address:      config.AppConfig.TURNAddress,
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(config.AppConfig.TURNAddress),
					Address:      config.AppConfig.TURNAddress,
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
func (s *TURNServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("TURN Server already running.")
		return nil
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting TURN Server...")
	log.Printf("TURN Server listening on UDP %s and TCP %s",
		s.udpListener.LocalAddr().String(),
		s.tcpListener.Addr().String())

	// Setup HTTP routes
	s.router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Verifier is running with TURN server\nRealm: chain://%s\n",
			config.AppConfig.ServiceAddress)
	})

	s.router.HandleFunc("/turn/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "TURN Server Statistics\n")
		fmt.Fprintf(w, "UDP Address: %s\n", s.udpListener.LocalAddr().String())
		fmt.Fprintf(w, "TCP Address: %s\n", s.tcpListener.Addr().String())
		fmt.Fprintf(w, "Running: %v\n", s.IsRunning())
	})

	// Start the HTTP service
	port := config.AppConfig.Port
	log.Printf("Serving HTTP service on port %s\n", port)

	// Start HTTP server in a goroutine so it doesn't block
	go func() {
		if err := http.ListenAndServe(":"+port, s.router); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop shuts down the TURN server
func (s *TURNServer) Stop() error {
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
func (s *TURNServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

