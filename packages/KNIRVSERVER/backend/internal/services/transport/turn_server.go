package transport

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/pion/turn/v4"
)

// TURNServer represents an integrated TURN server for NAT traversal in Nexus
type TURNServer struct {
	udpListener net.PacketConn
	tcpListener net.Listener
	turnServer  *turn.Server
	mu          sync.Mutex
	running     bool
	
	address string
	port    int
	realm   string
}

// NewTURNServer creates a new TURN server instance
func NewTURNServer(address string, port int, realm string) (*TURNServer, error) {
	// Configure UDP listener
	udpAddress := fmt.Sprintf("%s:%d", address, port)
	udpListener, err := net.ListenPacket("udp4", udpAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP address %s: %w", udpAddress, err)
	}

	// Configure TCP listener on the same port
	tcpAddress := fmt.Sprintf("%s:%d", address, port)
	tcpListener, err := net.Listen("tcp4", tcpAddress)
	if err != nil {
		udpListener.Close()
		return nil, fmt.Errorf("failed to listen on TCP address %s: %w", tcpAddress, err)
	}

	s := &TURNServer{
		udpListener: udpListener,
		tcpListener: tcpListener,
		address:     address,
		port:        port,
		realm:       realm,
	}

	// Create the TURN server with static auth for the prototype
	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm: realm,
		AuthHandler: func(username, realm string, srcAddr net.Addr) ([]byte, bool) {
			// Static credential for Nexus P2P Mesh
			return turn.GenerateAuthKey(username, realm, "nexus-p2p-secret"), true
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(address),
					Address:      address,
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(address),
					Address:      address,
				},
			},
		},
	})

	if err != nil {
		udpListener.Close()
		tcpListener.Close()
		return nil, fmt.Errorf("failed to create TURN server: %w", err)
	}

	s.turnServer = turnServer
	return s, nil
}

// Start begins the TURN server operation
func (s *TURNServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.running = true
	log.Printf("Nexus TURN Server started on %s:%d (Realm: %s)", s.address, s.port, s.realm)
	return nil
}

// Stop shuts down the TURN server
func (s *TURNServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	err := s.turnServer.Close()
	s.running = false
	return err
}
