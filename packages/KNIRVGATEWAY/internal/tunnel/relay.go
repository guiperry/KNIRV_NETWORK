package tunnel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

// bufferedConn wraps a net.Conn whose first line has already been consumed
// through a bufio.Reader, so any bytes the reader buffered past that line
// (common when a sender pipelines its first line and body in one write) are
// served before falling through to the raw connection. Without this, handing
// the raw net.Conn to io.Copy after reading a line through bufio would
// silently drop whatever the reader had already buffered.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

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
	// ownConn is cleared when this connection's ownership transfers elsewhere
	// (bound as a dev's dedicated relay data connection, or handed to
	// TunnelManager.Relay as the external leg of an active relay) so this
	// deferred Close doesn't yank a connection out from under its new owner.
	ownConn := true
	defer func() {
		if ownConn {
			externalClientConn.Close()
		}
	}()

	tcpAddr, ok := externalClientConn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		prl.logger.Error("Failed to get TCP address")
		return
	}

	prl.logger.Info("Incoming external connection",
		zap.String("remoteAddr", tcpAddr.String()))

	var targetPeerId string

	// Use bufio.Reader (not Scanner) and wrap the connection with it below
	// before handing off to Relay/BindRelayDataConn — otherwise any bytes a
	// sender pipelines immediately after its first line would already be
	// consumed into the reader's internal buffer and silently lost to
	// whatever reads the raw net.Conn next.
	reader := bufio.NewReader(externalClientConn)
	wrapped := &bufferedConn{Conn: externalClientConn, reader: reader}

	// Read first line to get target peer ID
	if line, err := reader.ReadString('\n'); err == nil || len(line) > 0 {
		dataStr := strings.TrimSpace(line)

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

		// A dev's dedicated second connection, opened in response to a
		// RELAY_REQUEST pushed down its control channel, identifies itself by
		// session token rather than target peer ID. Try that shape first:
		// once bound, ownership of the connection transfers to the tunnel
		// manager and this listener must not touch it again.
		var bindMessage RelayBindMessage
		if err := json.Unmarshal([]byte(dataStr), &bindMessage); err == nil && bindMessage.RelaySessionToken != "" {
			if !prl.tunnelManager.BindRelayDataConn(bindMessage.RelaySessionToken, wrapped) {
				prl.logger.Warn("Unknown or expired relay session token",
					zap.String("token", bindMessage.RelaySessionToken))
				externalClientConn.Write([]byte("ERROR: Unknown or expired relay session token.\n"))
				return
			}
			// Bound: the tunnel manager now owns this connection's
			// lifecycle (it is one leg of an active or pending relay).
			ownConn = false
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
		if !prl.tunnelManager.Relay(wrapped, targetPeerId) {
			externalClientConn.Write([]byte(fmt.Sprintf("ERROR: Could not establish relay to %s.\n", targetPeerId)))
			return
		}
		// Relay established: TunnelManager.Relay has already taken ownership
		// of externalClientConn (it is one leg of the active relay) and will
		// close it when the relay ends. Returning here must not also close it.
		ownConn = false
		return
	}

	prl.logger.Info("External client connection closed")
}
