package tunnel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// relayBindTimeout bounds how long Relay waits for the dev to open its
// dedicated second connection after a RELAY_REQUEST is pushed down the
// control channel, and how long a bind-ready pendingRelay entry is kept
// before it is reaped as abandoned.
const relayBindTimeout = 10 * time.Second

// TunnelManager manages active control sockets and tunneling
type TunnelManager struct {
	activeControlSockets map[string]net.Conn

	pendingRelaysMu sync.Mutex
	pendingRelays   map[string]chan net.Conn

	mu     sync.RWMutex
	logger *zap.Logger
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(logger *zap.Logger) *TunnelManager {
	return &TunnelManager{
		activeControlSockets: make(map[string]net.Conn),
		pendingRelays:        make(map[string]chan net.Conn),
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

// GetControlSocket gets the control socket for a dev ID. It deliberately does
// not probe the socket with a Read: that control connection is permanently
// owned by control.go's IDENTIFY/PING scan loop for its entire lifetime, so a
// liveness Read here would either race that loop for incoming bytes (the same
// class of bug RequestRelayDataConn/Relay fix below) or, since the control
// connection is idle between heartbeats far more often than not, misreport a
// healthy idle connection as closed by observing a read timeout. The map is
// kept accurate by AddControlSocket/RemoveControlSocket's own lifecycle
// hooks, which is sufficient here.
func (tm *TunnelManager) GetControlSocket(devId string) net.Conn {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.activeControlSockets[devId]
}

// RequestRelayDataConn asks devId's control channel to open a dedicated
// second connection for one relay session, and returns a channel that
// receives that connection once BindRelayDataConn delivers it. The control
// socket itself is only ever written to here (one JSON line) — it is never
// handed to io.Copy, which is what previously raced against control.go's
// IDENTIFY/PING scanner reading the same socket.
func (tm *TunnelManager) RequestRelayDataConn(devId string) (string, <-chan net.Conn, error) {
	controlSocket := tm.GetControlSocket(devId)
	if controlSocket == nil {
		return "", nil, fmt.Errorf("no active control socket for devId %q", devId)
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate relay session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	ch := make(chan net.Conn, 1)
	tm.pendingRelaysMu.Lock()
	tm.pendingRelays[token] = ch
	tm.pendingRelaysMu.Unlock()

	request := ControlMessage{Action: "RELAY_REQUEST", RelaySessionToken: token}
	payload, err := json.Marshal(request)
	if err != nil {
		tm.cancelPendingRelay(token)
		return "", nil, fmt.Errorf("failed to marshal relay request: %w", err)
	}
	if _, err := controlSocket.Write(append(payload, '\n')); err != nil {
		tm.cancelPendingRelay(token)
		return "", nil, fmt.Errorf("failed to push relay request to devId %q: %w", devId, err)
	}

	// Reap this entry if nothing ever binds it, so a dev that never responds
	// doesn't leak an entry forever.
	time.AfterFunc(relayBindTimeout, func() { tm.cancelPendingRelay(token) })

	return token, ch, nil
}

// cancelPendingRelay removes a pending relay entry, closing a connection that
// arrived too late to be observed by the original waiter, if any.
func (tm *TunnelManager) cancelPendingRelay(token string) {
	tm.pendingRelaysMu.Lock()
	ch, ok := tm.pendingRelays[token]
	if ok {
		delete(tm.pendingRelays, token)
	}
	tm.pendingRelaysMu.Unlock()
	if !ok {
		return
	}
	select {
	case conn := <-ch:
		conn.Close()
	default:
	}
}

// BindRelayDataConn binds conn — the dev's dedicated second connection — to
// the pending relay session identified by token. Returns false when token is
// unknown or already expired/bound, in which case the caller owns conn and
// must close it.
func (tm *TunnelManager) BindRelayDataConn(token string, conn net.Conn) bool {
	tm.pendingRelaysMu.Lock()
	ch, ok := tm.pendingRelays[token]
	if ok {
		delete(tm.pendingRelays, token)
	}
	tm.pendingRelaysMu.Unlock()
	if !ok {
		return false
	}
	ch <- conn
	return true
}

// Relay establishes a bidirectional relay between an external client and
// targetInternalPeerId's dev process, over a dedicated data connection
// requested via the dev's control channel (see RequestRelayDataConn). The
// control socket itself never carries relay bytes.
func (tm *TunnelManager) Relay(externalClientSocket net.Conn, targetInternalPeerId string) bool {
	token, dataConnCh, err := tm.RequestRelayDataConn(targetInternalPeerId)
	if err != nil {
		tm.logger.Warn("Could not request relay data connection",
			zap.String("targetPeerId", targetInternalPeerId),
			zap.Error(err))
		externalClientSocket.Close()
		return false
	}

	tm.logger.Info("Relay requested, awaiting dedicated data connection",
		zap.String("targetPeerId", targetInternalPeerId))

	select {
	case dataConn := <-dataConnCh:
		tm.logger.Info("Relaying data between external client and internal peer",
			zap.String("targetPeerId", targetInternalPeerId))

		// Both legs are single-use, dedicated to this one relay session, so
		// closing both unconditionally once either direction ends is correct
		// — Close is safe to call more than once. No liveness probe is needed
		// or wanted here: by this point io.Copy has already observed the
		// terminal error/EOF on whichever side ended the session.
		closeBoth := func() {
			externalClientSocket.Close()
			dataConn.Close()
		}
		go func() {
			defer func() {
				tm.logger.Info("Cleaning up relay for external client",
					zap.String("targetPeerId", targetInternalPeerId))
				closeBoth()
			}()
			io.Copy(dataConn, externalClientSocket)
		}()

		go func() {
			defer closeBoth()
			io.Copy(externalClientSocket, dataConn)
		}()

		return true
	case <-time.After(relayBindTimeout):
		tm.cancelPendingRelay(token)
		tm.logger.Warn("Timed out waiting for dev's relay data connection",
			zap.String("targetPeerId", targetInternalPeerId))
		externalClientSocket.Close()
		return false
	}
}

// isSocketClosed probes conn with a near-zero-deadline Read to guess whether
// it is still open. Only still used by service.go's periodic pruning pass —
// GetControlSocket and Relay's cleanup above no longer call this; see the
// comment on GetControlSocket for why a Read-based probe is unsafe on a
// socket that has another legitimate concurrent reader (e.g. a dev's control
// connection, still owned by control.go's scan loop). A read timeout means
// only "no data arrived in this instant," not "closed" — a connection sits
// idle between heartbeats far more often than it has a byte in flight, so
// treating every timeout as closed (the previous behavior) misclassified
// healthy idle connections as dead.
func isSocketClosed(conn net.Conn) bool {
	if conn == nil {
		return true
	}

	conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	var buf [1]byte
	_, err := conn.Read(buf[:])
	conn.SetReadDeadline(time.Time{}) // Reset deadline

	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return false
	}
	return true
}
