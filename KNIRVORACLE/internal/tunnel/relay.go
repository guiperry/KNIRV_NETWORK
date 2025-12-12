package tunnel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

// PublicRelayListener handles TCP connections from external clients
type PublicRelayListener struct {
	port          int
	server        net.Listener
	tunnelManager *TunnelManager
	logger        *zap.Logger
	config        *Config
}

// NewPublicRelayListener creates a new public relay listener
func NewPublicRelayListener(port int, tunnelManager *TunnelManager, config *Config, logger *zap.Logger) *PublicRelayListener {
	return &PublicRelayListener{
		port:          port,
		tunnelManager: tunnelManager,
		logger:        logger,
		config:        config,
	}
}

// Start starts the public relay listener
func (prl *PublicRelayListener) Start() error {
	var err error
	prl.server, err = net.Listen("tcp", fmt.Sprintf(":%d", prl.port))
	if err != nil {
		return fmt.Errorf("failed to start public relay listener: %w", err)
	}

	prl.logger.Info("Public relay listener started",
		zap.Int("port", prl.port))

	go prl.acceptConnections()
	return nil
}

// Stop stops the public relay listener
func (prl *PublicRelayListener) Stop() error {
	if prl.server != nil {
		return prl.server.Close()
	}
	return nil
}

func (prl *PublicRelayListener) acceptConnections() {
	for {
		conn, err := prl.server.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			prl.logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		go prl.handleConnection(conn)
	}
}

func (prl *PublicRelayListener) handleConnection(externalClientConn net.Conn) {
	defer externalClientConn.Close()

	tcpAddr, ok := externalClientConn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		prl.logger.Error("Failed to get TCP address")
		return
	}

	prl.logger.Info("Incoming external connection",
		zap.String("remoteAddr", tcpAddr.String()))

	var targetPeerId string

	scanner := bufio.NewScanner(externalClientConn)

	// Read first line to get target peer ID
	if scanner.Scan() {
		dataStr := strings.TrimSpace(scanner.Text())

		// Check if this is an HTTP request
		if strings.HasPrefix(dataStr, "GET ") || strings.HasPrefix(dataStr, "POST ") ||
			strings.HasPrefix(dataStr, "HEAD ") || strings.HasPrefix(dataStr, "PUT ") ||
			strings.HasPrefix(dataStr, "DELETE ") || strings.HasPrefix(dataStr, "OPTIONS ") {

			prl.logger.Info("Received HTTP request on relay port, sending service info")

			httpResponse := fmt.Sprintf(`HTTP/1.1 200 OK
Content-Type: text/plain
Connection: close

KNIRV Tunnel Registry Public Relay Service
This port (%d) is for external client tunnel connections.
Use the HTTP API on port %d for web requests.
Protocol: Send target PeerID as first line (JSON or plain text)
Example: {"targetPeerId": "Qm..."} or QmExamplePeerId
`, prl.port, prl.config.HTTPAPIPort)

			externalClientConn.Write([]byte(httpResponse))
			return
		}

		// Try to parse as JSON first
		var relayMessage RelayMessage
		if err := json.Unmarshal([]byte(dataStr), &relayMessage); err == nil && relayMessage.TargetPeerID != "" {
			targetPeerId = relayMessage.TargetPeerID
		} else {
			// Fallback: assume the line is the PeerID directly
			// Basic PeerID check (starts with Qm or 12D3Koo)
			if strings.HasPrefix(dataStr, "Qm") || strings.HasPrefix(dataStr, "12D3Koo") {
				targetPeerId = dataStr
			} else {
				prl.logger.Warn("Invalid initial message from external client",
					zap.String("message", dataStr))
				externalClientConn.Write([]byte("ERROR: Invalid initial message. Expecting target PeerID.\n"))
				return
			}
		}

		prl.logger.Info("External client wants to connect",
			zap.String("targetPeerId", targetPeerId))

		// Attempt to establish relay
		if !prl.tunnelManager.Relay(externalClientConn, targetPeerId) {
			externalClientConn.Write([]byte(fmt.Sprintf("ERROR: Could not establish relay to %s.\n", targetPeerId)))
			return
		}
		// Relay established, connection will be handled by tunnel manager
	}

	if err := scanner.Err(); err != nil {
		prl.logger.Error("Scanner error", zap.Error(err))
	}

	prl.logger.Info("External client connection closed")
}
