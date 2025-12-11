package tunnel

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

// STUNServer implements a basic STUN server for NAT traversal
type STUNServer struct {
	port   int
	server *net.UDPConn
	logger *zap.Logger
}

// NewSTUNServer creates a new STUN server
func NewSTUNServer(port int, logger *zap.Logger) *STUNServer {
	return &STUNServer{
		port:   port,
		logger: logger,
	}
}

// Start starts the STUN server
func (s *STUNServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	s.server, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start STUN server: %w", err)
	}

	s.logger.Info("STUN server started",
		zap.Int("port", s.port))

	go s.handleConnections()
	return nil
}

// Stop stops the STUN server
func (s *STUNServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *STUNServer) handleConnections() {
	buffer := make([]byte, 1024)

	for {
		n, remoteAddr, err := s.server.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			s.logger.Error("Error reading from UDP", zap.Error(err))
			break
		}

		s.logger.Debug("Received STUN request",
			zap.Int("bytes", n),
			zap.String("remoteAddr", remoteAddr.String()))

		// Create a simple response with the client's address information
		response := map[string]interface{}{
			"type":         "stun-response",
			"remoteAddress": remoteAddr.IP.String(),
			"remotePort":   remoteAddr.Port,
			"timestamp":    time.Now().Unix(),
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			s.logger.Error("Failed to marshal STUN response", zap.Error(err))
			continue
		}

		_, err = s.server.WriteToUDP(responseBytes, remoteAddr)
		if err != nil {
			s.logger.Error("Failed to send STUN response",
				zap.String("remoteAddr", remoteAddr.String()),
				zap.Error(err))
		} else {
			s.logger.Debug("Sent STUN response",
				zap.String("remoteAddr", remoteAddr.String()))
		}
	}
}