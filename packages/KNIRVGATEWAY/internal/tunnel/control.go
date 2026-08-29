package tunnel

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ControlListener handles TCP connections from internal nodes
type ControlListener struct {
	port            int
	server          net.Listener
	registryManager *RegistryManager
	tunnelManager   *TunnelManager
	logger          *zap.Logger
	config          *Config
	httpClient      *http.Client
}

// NewControlListener creates a new control listener
func NewControlListener(port int, registryManager *RegistryManager, tunnelManager *TunnelManager, config *Config, logger *zap.Logger) *ControlListener {
	cl := &ControlListener{
		port:            port,
		registryManager: registryManager,
		tunnelManager:   tunnelManager,
		logger:          logger,
		config:          config,
	}
	if config != nil && strings.TrimSpace(config.BackendSocketPath) != "" {
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", config.BackendSocketPath)
			},
		}
		cl.httpClient = &http.Client{Transport: transport, Timeout: 5 * time.Second}
	}
	return cl
}

// Start starts the control listener
func (cl *ControlListener) Start() error {
	var err error
	cl.server, err = net.Listen("tcp", fmt.Sprintf(":%d", cl.port))
	if err != nil {
		return fmt.Errorf("failed to start control listener: %w", err)
	}

	cl.logger.Info("Control listener started",
		zap.Int("port", cl.port))

	go cl.acceptConnections()
	return nil
}

// validateControlToken validates the bearer token from an IDENTIFY message.
// It calls the backend's /api/auth/me endpoint over the backend unix socket
// and returns the authenticated user ID.  The caller must verify that this
// user ID matches the claimed DevID.
func (cl *ControlListener) validateControlToken(ctx context.Context, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", errors.New("token is empty")
	}

	parts := strings.Fields(token)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("invalid authorization format")
	}

	if cl.httpClient == nil {
		return "", errors.New("authorization service is not configured")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://backend/api/auth/me", nil)
	if err != nil {
		return "", fmt.Errorf("authorization service unavailable: %w", err)
	}
	request.Header.Set("Authorization", token)
	response, err := cl.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("authorization service unavailable: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return "", errors.New("invalid or expired session token")
		}
		return "", fmt.Errorf("authorization service returned %d", response.StatusCode)
	}

	var meResp struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(response.Body).Decode(&meResp); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}
	if meResp.UserID == "" {
		return "", errors.New("authorization service returned empty user id")
	}
	return meResp.UserID, nil
}

// Stop stops the control listener
func (cl *ControlListener) Stop() error {
	if cl.server != nil {
		return cl.server.Close()
	}
	return nil
}

func (cl *ControlListener) acceptConnections() {
	for {
		conn, err := cl.server.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			cl.logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		go cl.handleConnection(conn)
	}
}

func (cl *ControlListener) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Assign unique ID to socket
	socketID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())
	cl.logger.Info("Internal node connected",
		zap.String("socketId", socketID))

	var identifiedPeerId string

	// Set up connection registry entry
	tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		cl.logger.Error("Failed to get TCP address")
		return
	}

	connectionRegistryMu.Lock()
	connectionRegistry[socketID] = &ConnectionInfo{
		ID:         socketID,
		Type:       "unknown",
		SourceIP:   tcpAddr.IP.String(),
		SourcePort: tcpAddr.Port,
		LastSeen:   time.Now(),
		Socket:     conn,
	}
	connectionRegistryMu.Unlock()

	defer func() {
		// Clean up connection registry
		connectionRegistryMu.Lock()
		delete(connectionRegistry, socketID)
		connectionRegistryMu.Unlock()

		// Deregister node if identified
		if identifiedPeerId != "" {
			cl.tunnelManager.RemoveControlSocket(identifiedPeerId, "")
			cl.registryManager.DeregisterNodeByControlSocket(socketID)
		}
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		dataStr := strings.TrimSpace(scanner.Text())

		// Check if this is an HTTP request
		if strings.HasPrefix(dataStr, "GET ") || strings.HasPrefix(dataStr, "POST ") ||
			strings.HasPrefix(dataStr, "HEAD ") || strings.HasPrefix(dataStr, "PUT ") ||
			strings.HasPrefix(dataStr, "DELETE ") || strings.HasPrefix(dataStr, "OPTIONS ") {

			cl.logger.Info("Received HTTP request on control port, sending service info",
				zap.String("socketId", socketID))

			httpResponse := fmt.Sprintf(`HTTP/1.1 200 OK
Content-Type: text/plain
Connection: close

KNIRV Tunnel Registry Control Service
This port (%d) is for internal node control connections.
Use the HTTP API on port %d for web requests.
Protocol: JSON messages (IDENTIFY, PING, etc.)
`, cl.port, cl.config.HTTPAPIPort)

			conn.Write([]byte(httpResponse))
			return
		}

		// Try to parse as JSON
		var message ControlMessage
		if err := json.Unmarshal([]byte(dataStr), &message); err != nil {
			cl.logger.Error("Error parsing message",
				zap.String("socketId", socketID),
				zap.String("data", dataStr),
				zap.Error(err))
			continue
		}

		// Handle IDENTIFY message
		if message.Action == "IDENTIFY" && message.DevID != "" && message.InternalIP != "" && message.InternalP2PPort != 0 {
			authenticatedUserID, err := cl.validateControlToken(context.Background(), message.Token)
			if err != nil {
				cl.logger.Warn("Control socket auth failed",
					zap.String("socketId", socketID),
					zap.String("devId", message.DevID),
					zap.Error(err))
				response := ControlMessage{
					Action:  "ERROR",
					Message: "authentication failed: " + err.Error(),
				}
				responseBytes, _ := json.Marshal(response)
				conn.Write(append(responseBytes, '\n'))
				conn.Close()
				return
			}
			if authenticatedUserID != message.DevID {
				cl.logger.Warn("Control socket identity mismatch",
					zap.String("socketId", socketID),
					zap.String("claimedDevId", message.DevID),
					zap.String("authenticatedUser", authenticatedUserID))
				response := ControlMessage{
					Action:  "ERROR",
					Message: "identity mismatch: token owner does not match claimed devId",
				}
				responseBytes, _ := json.Marshal(response)
				conn.Write(append(responseBytes, '\n'))
				conn.Close()
				return
			}
			identifiedPeerId = message.DevID
			cl.tunnelManager.AddControlSocket(identifiedPeerId, conn)
			cl.registryManager.RegisterNodeViaControlSocket(
				identifiedPeerId,
				message.ChainID,
				message.InternalIP,
				message.InternalP2PPort,
				message.Type,
				socketID,
				cl.config.ServerPublicHost,
				cl.config.PublicRelayPort,
			)

			response := ControlMessage{
				Action:  "SUCCESS",
				Message: "Identified and control channel active.",
			}
			responseBytes, _ := json.Marshal(response)
			conn.Write(append(responseBytes, '\n'))

			cl.logger.Info("Socket identified",
				zap.String("socketId", socketID),
				zap.String("peerId", identifiedPeerId))

		} else if message.Action == "PING" {
			// Update last seen timestamp
			if identifiedPeerId != "" {
				connectionRegistryMu.Lock()
				if connInfo := connectionRegistry[socketID]; connInfo != nil {
					connInfo.LastSeen = time.Now()
					cl.logger.Debug("Received PING, updated last seen",
						zap.String("peerId", identifiedPeerId))
				} else {
					// Re-register if missing
					cl.logger.Info("Received PING but not in registry, re-registering",
						zap.String("peerId", identifiedPeerId))
					connectionRegistry[socketID] = &ConnectionInfo{
						ID:         socketID,
						Type:       "dev",
						SourceIP:   tcpAddr.IP.String(),
						SourcePort: tcpAddr.Port,
						LastSeen:   time.Now(),
						Socket:     conn,
					}
				}
				connectionRegistryMu.Unlock()
			}

			// Send PONG response
			response := ControlMessage{
				Action:    "PONG",
				Timestamp: time.Now().Unix(),
				Status:    "active",
			}
			responseBytes, _ := json.Marshal(response)
			conn.Write(append(responseBytes, '\n'))

		} else {
			cl.logger.Warn("Unknown message",
				zap.String("socketId", socketID),
				zap.Any("message", message))
		}
	}

	if err := scanner.Err(); err != nil {
		cl.logger.Error("Scanner error",
			zap.String("socketId", socketID),
			zap.Error(err))
	}

	cl.logger.Info("Socket closed",
		zap.String("socketId", socketID),
		zap.String("peerId", identifiedPeerId))
}
