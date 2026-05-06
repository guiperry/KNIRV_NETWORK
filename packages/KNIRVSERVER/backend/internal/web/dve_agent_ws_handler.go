package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// DVEAgentWSHandler proxies WebSocket I/O between the frontend terminal
// and the KNIRVAGENT DVE Supervisor subprocess.
type DVEAgentWSHandler struct {
	knirvagentManager interface {
		IsRunning() bool
		GetBaseURL() string
		HealthCheck(ctx context.Context) error
	}
	mu     sync.RWMutex
	active map[string]*agentWSConn // nodeId -> connection
}

// agentWSConn represents a single WebSocket connection for a DVE agent terminal.
type agentWSConn struct {
	nodeID    string
	conn      *websocket.Conn
	send      chan []byte
	agentURL  string
	lastSeen  time.Time
	mu        sync.Mutex
	agentHTTP *http.Client
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
func NewDVEAgentWSHandler(mgr interface {
	IsRunning() bool
	GetBaseURL() string
	HealthCheck(ctx context.Context) error
}) *DVEAgentWSHandler {
	return &DVEAgentWSHandler{
		knirvagentManager: mgr,
		active:            make(map[string]*agentWSConn),
	}
}

// HandleWebSocket handles the HTTP→WebSocket upgrade for agent terminal connections.
func (h *DVEAgentWSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["nodeId"]
	if nodeID == "" {
		http.Error(w, "nodeId required", http.StatusBadRequest)
		return
	}

	if h.knirvagentManager == nil || !h.knirvagentManager.IsRunning() {
		http.Error(w, "KNIRVAGENT supervisor not running", http.StatusServiceUnavailable)
		return
	}

	baseURL := h.knirvagentManager.GetBaseURL()
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[DVE Agent WS] Upgrade error for node %s: %v", nodeID, err)
		return
	}

	wsConn := &agentWSConn{
		nodeID:   nodeID,
		conn:     conn,
		send:     make(chan []byte, 256),
		agentURL: baseURL,
		lastSeen: time.Now(),
		agentHTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	h.mu.Lock()
	// Close any existing connection for this node
	if existing, ok := h.active[nodeID]; ok {
		existing.conn.Close()
	}
	h.active[nodeID] = wsConn
	h.mu.Unlock()

	log.Printf("[DVE Agent WS] Connected: node=%s agent=%s", nodeID, baseURL)

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
			// Forward terminal input to the KNIRVAGENT command execution endpoint
			go h.forwardInput(conn, string(msg.Data))
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

// forwardInput sends a command to the KNIRVAGENT's execute endpoint and
// writes the response back to the WebSocket send channel.
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

	// POST command to KNIRVAGENT's execute endpoint
	resp, err := conn.agentHTTP.Post(
		conn.agentURL+"/api/execute",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		conn.send <- mustMarshal(agentWSResponse{
			Type: "data",
			Data: fmt.Sprintf("\x1b[31m[ERROR] KNIRVAGENT unreachable: %v\x1b[0m\r\n", err),
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

	// Parse KNIRVAGENT response for output
	var agentResp struct {
		Output  string `json:"output"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &agentResp); err != nil {
		// Raw text response — send it directly
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

	// Send prompt
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
