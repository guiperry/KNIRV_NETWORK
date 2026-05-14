package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// AgentManagerInterface defines what the DVEAgentWSHandler needs from the
// KNIRVAGENT manager for per-DVE socket routing.
type AgentManagerInterface interface {
	IsRunning() bool
	GetBaseURL() string
	HealthCheck(ctx context.Context) error
	GetSocketPathForDVE(dveID string) (string, error)
}

// DVEAgentWSHandler proxies WebSocket I/O between the frontend terminal
// and the KNIRVAGENT DVE Supervisor subprocess (via Unix socket).
type DVEAgentWSHandler struct {
	agentManager AgentManagerInterface
	mu           sync.RWMutex
	active       map[string]*agentWSConn // nodeId -> connection
}

// agentWSConn represents a single WebSocket connection for a DVE agent terminal.
type agentWSConn struct {
	nodeID     string
	conn       *websocket.Conn
	send       chan []byte
	socketPath string
	lastSeen   time.Time
	mu         sync.Mutex
	agentHTTP  *http.Client
}

// agentWSMessage is the JSON frame exchanged over the WebSocket.
type agentWSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// agentWSResponse is the JSON frame sent to the client.
type agentWSResponse struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// NewDVEAgentWSHandler creates a new WebSocket handler for DVE agent terminal sessions.
func NewDVEAgentWSHandler(mgr AgentManagerInterface) *DVEAgentWSHandler {
	return &DVEAgentWSHandler{
		agentManager: mgr,
		active:       make(map[string]*agentWSConn),
	}
}

// HandleWebSocket handles the HTTP->WebSocket upgrade for agent terminal connections.
func (h *DVEAgentWSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["nodeId"]
	if nodeID == "" {
		http.Error(w, "nodeId required", http.StatusBadRequest)
		return
	}

	if h.agentManager == nil || !h.agentManager.IsRunning() {
		http.Error(w, "KNIRVAGENT supervisor not running", http.StatusServiceUnavailable)
		return
	}

	// Look up the per-DVE agent socket path from the AgentManager
	socketPath, err := h.agentManager.GetSocketPathForDVE(nodeID)
	if err != nil {
		log.Printf("[DVE Agent WS] No agent socket for node %s: %v", nodeID, err)
		http.Error(w, "No KNIRVAGENT running for this DVE: "+err.Error(), http.StatusNotFound)
		return
	}

	// Create an HTTP client that dials the agent's Unix socket.
	// No client-level Timeout: LLM calls via the fallback chain can easily take
	// more than 30 seconds.  Each forwardInput call uses its own per-request
	// context with a 5-minute deadline instead.
	unixClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 5*time.Second)
			},
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[DVE Agent WS] Upgrade error for node %s: %v", nodeID, err)
		return
	}

	wsConn := &agentWSConn{
		nodeID:     nodeID,
		conn:       conn,
		send:       make(chan []byte, 256),
		socketPath: socketPath,
		lastSeen:   time.Now(),
		agentHTTP:  unixClient,
	}

	h.mu.Lock()
	if existing, ok := h.active[nodeID]; ok {
		existing.conn.Close()
	}
	h.active[nodeID] = wsConn
	h.mu.Unlock()

	log.Printf("[DVE Agent WS] Connected: node=%s socket=%s", nodeID, socketPath)

	// Send hello message and prompt to the frontend terminal
	wsConn.send <- mustMarshal(agentWSResponse{
		Type: "data",
		Data: "\x1b[1;35m╔═══════════════════════════════════════════════════════════╗\x1b[0m\r\n" +
			"\x1b[1;35m║  KNIRVAGENT DVE Supervisor v1.1.0                        ║\x1b[0m\r\n" +
			"\x1b[1;35m║  Status: ACTIVE                                           ║\x1b[0m\r\n" +
			"\x1b[1;35m║  DVE: " + nodeID + "\x1b[0m\r\n" +
			"\x1b[1;35m╚═══════════════════════════════════════════════════════════╝\x1b[0m\r\n" +
			"\x1b[35mType 'help' for available commands or 'exit' to disconnect.\x1b[0m\r\n",
	})
	wsConn.send <- mustMarshal(agentWSResponse{Type: "prompt", Data: ""})

	go h.writePump(wsConn)
	go h.readPump(wsConn)
}

// readPump reads messages from the WebSocket and forwards them to the KNIRVAGENT.
func (h *DVEAgentWSHandler) readPump(conn *agentWSConn) {
	defer func() {
		h.mu.Lock()
		delete(h.active, conn.nodeID)
		h.mu.Unlock()
		conn.conn.Close()
	}()

	conn.conn.SetReadLimit(65536)
	conn.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	conn.conn.SetPongHandler(func(string) error {
		conn.mu.Lock()
		conn.lastSeen = time.Now()
		conn.mu.Unlock()
		conn.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})

	for {
		_, message, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[DVE Agent WS] Read error node=%s: %v", conn.nodeID, err)
			}
			break
		}

		var msg agentWSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[DVE Agent WS] Invalid message from node=%s: %v", conn.nodeID, err)
			continue
		}

		switch msg.Type {
		case "input":
			var inputStr string
			if err := json.Unmarshal(msg.Data, &inputStr); err != nil {
				log.Printf("[DVE Agent WS] Invalid input data from node=%s: %v", conn.nodeID, err)
				continue
			}
			go h.forwardInput(conn, inputStr)
		case "ping":
			conn.send <- mustMarshal(agentWSResponse{Type: "pong", Data: ""})
		default:
			log.Printf("[DVE Agent WS] Unknown message type from node=%s: %s", conn.nodeID, msg.Type)
		}
	}
}

// writePump writes messages from the send channel to the WebSocket connection.
func (h *DVEAgentWSHandler) writePump(conn *agentWSConn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.send:
			if !ok {
				conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[DVE Agent WS] Write error node=%s: %v", conn.nodeID, err)
				return
			}
		case <-ticker.C:
			conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// forwardInput sends a command to the KNIRVAGENT's /api/execute endpoint via
// Unix socket and writes the response back to the WebSocket send channel.
func (h *DVEAgentWSHandler) forwardInput(conn *agentWSConn, input string) {
	if input == "" {
		return
	}

	payload := map[string]string{"command": input}
	body, err := json.Marshal(payload)
	if err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: fmt.Sprintf("\x1b[31m[ERROR] Failed to encode command: %v\x1b[0m\r\n", err),
		})
		return
	}

	// Give each LLM call up to 5 minutes — enough for a fallback chain and a
	// slow provider response.  The KNIRVAGENT honours request cancellation, so
	// this deadline propagates through to the underlying LLM call.
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		"http://localhost/api/execute", bytes.NewReader(body))
	if err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: fmt.Sprintf("\x1b[31m[ERROR] Failed to build request: %v\x1b[0m\r\n", err),
		})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := conn.agentHTTP.Do(req)
	if err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: fmt.Sprintf("\x1b[31m[ERROR] KNIRVAGENT unreachable (socket: %s): %v\x1b[0m\r\n", conn.socketPath, err),
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: fmt.Sprintf("\x1b[31m[ERROR] Failed to read response: %v\x1b[0m\r\n", err),
		})
		return
	}

	var agentResp struct {
		Output  string `json:"output"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &agentResp); err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: string(respBody) + "\r\n",
		})
	} else {
		if !agentResp.Success && agentResp.Error != "" {
			conn.send <- mustMarshal(agentWSResponse{
				Type: "data",
				Data: fmt.Sprintf("\x1b[31m%s\x1b[0m\r\n", agentResp.Error),
			})
		} else if agentResp.Output != "" {
			conn.send <- mustMarshal(agentWSResponse{
				Type: "data",
				Data: agentResp.Output,
			})
		}
	}

	conn.send <- mustMarshal(agentWSResponse{Type: "prompt", Data: ""})
}

// RegisterRoutes registers the WebSocket endpoint on the router.
func (h *DVEAgentWSHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws/dve/{nodeId}/agent", h.HandleWebSocket)
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
