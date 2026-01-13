package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TunnelManager manages active control sockets and tunneling
type TunnelManager struct {
	activeControlSockets map[string]net.Conn
	mu                   sync.RWMutex
	logger               *zap.Logger
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(logger *zap.Logger) *TunnelManager {
	return &TunnelManager{
		activeControlSockets: make(map[string]net.Conn),
		logger:               logger,
	}
}

// AddControlSocket adds a control socket for a dev ID
func (tm *TunnelManager) AddControlSocket(devId string, socket net.Conn) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Handle if a dev reconnects, close old socket
	if existingSocket, exists := tm.activeControlSockets[devId]; exists && existingSocket != nil {
		tm.logger.Warn("Peer re-established control, closing old socket",
			zap.String("devId", devId))
		existingSocket.Close()
	}

	tm.activeControlSockets[devId] = socket

	// Update connection registry
	connectionRegistryMu.Lock()
	if connectionRegistry[devId] == nil {
		connectionRegistry[devId] = &ConnectionInfo{
			ID:         devId,
			Type:       "dev", // Default type, will be updated by IDENTIFY message
			SourceIP:   socket.RemoteAddr().(*net.TCPAddr).IP.String(),
			SourcePort: socket.RemoteAddr().(*net.TCPAddr).Port,
			LastSeen:   time.Now(),
			Socket:     socket,
		}
	} else {
		connectionRegistry[devId].LastSeen = time.Now()
		connectionRegistry[devId].Socket = socket
		connectionRegistry[devId].SourceIP = socket.RemoteAddr().(*net.TCPAddr).IP.String()
		connectionRegistry[devId].SourcePort = socket.RemoteAddr().(*net.TCPAddr).Port
	}
	connectionRegistryMu.Unlock()

	tm.logger.Info("Control socket added",
		zap.String("devId", devId))
}

// RemoveControlSocket removes a control socket for a dev ID
func (tm *TunnelManager) RemoveControlSocket(devId, socketIdToMatch string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	currentSocket := tm.activeControlSockets[devId]
	if currentSocket != nil {
		// Only remove if the socket ID matches (to handle rapid reconnects)
		if socketIdToMatch == "" || fmt.Sprintf("%p", currentSocket) == socketIdToMatch {
			delete(tm.activeControlSockets, devId)

			// Mark socket as destroyed in connection registry
			connectionRegistryMu.Lock()
			if connInfo := connectionRegistry[devId]; connInfo != nil {
				connInfo.Socket = nil
				tm.logger.Info("Marked socket as destroyed in registry",
					zap.String("devId", devId))
			}
			connectionRegistryMu.Unlock()

			tm.logger.Info("Control socket removed",
				zap.String("devId", devId))
		} else {
			tm.logger.Info("Stale removeControlSocket call",
				zap.String("devId", devId),
				zap.String("currentSocketId", fmt.Sprintf("%p", currentSocket)),
				zap.String("socketIdToMatch", socketIdToMatch))
		}
	}
}

// GetControlSocket gets the control socket for a dev ID
func (tm *TunnelManager) GetControlSocket(devId string) net.Conn {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	socket := tm.activeControlSockets[devId]
	if socket != nil && !isSocketClosed(socket) {
		return socket
	}
	return nil
}

// Relay establishes a bidirectional relay between external client and internal control socket
func (tm *TunnelManager) Relay(externalClientSocket net.Conn, targetInternalPeerId string) bool {
	internalControlSocket := tm.GetControlSocket(targetInternalPeerId)

	if internalControlSocket == nil {
		tm.logger.Warn("No active control socket for target",
			zap.String("targetPeerId", targetInternalPeerId))
		externalClientSocket.Close()
		return false
	}

	tm.logger.Info("Relaying data between external client and internal peer",
		zap.String("targetPeerId", targetInternalPeerId))

	// Bidirectional pipe - don't end control socket if external client disconnects
	go func() {
		defer func() {
			tm.logger.Info("Cleaning up relay for external client",
				zap.String("targetPeerId", targetInternalPeerId))
			if !isSocketClosed(externalClientSocket) {
				externalClientSocket.Close()
			}
			// Don't close internal control socket - it might be reused
		}()

		io.Copy(internalControlSocket, externalClientSocket)
	}()

	go func() {
		defer func() {
			if !isSocketClosed(internalControlSocket) {
				// Don't actually close the control socket here
			}
		}()

		io.Copy(externalClientSocket, internalControlSocket)
	}()

	return true
}

// isSocketClosed checks if a socket is closed
func isSocketClosed(conn net.Conn) bool {
	if conn == nil {
		return true
	}

	// Try to set a read deadline and read a byte
	conn.SetReadDeadline(time.Now().Add(time.Nanosecond))
	var buf [1]byte
	_, err := conn.Read(buf[:])
	conn.SetReadDeadline(time.Time{}) // Reset deadline

	return err != nil
}
