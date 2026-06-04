package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type InnerAgentManagerInterface interface {
	InnerAgentClient(dveID string) (*http.Client, string, error)
	StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error
}

type DVEInnerAgentWSHandler struct {
	agentManager InnerAgentManagerInterface
	mu           sync.RWMutex
	active       map[string]*innerWSConn
}

type innerWSConn struct {
	dveID     string
	sessionID string // from URL — default session for input/resize when msg.Session is empty
	conn      *websocket.Conn
	send      chan []byte
	lastSeen  time.Time
	mu        sync.Mutex
}

// innerWSMessage is sent from the browser to the backend.
type innerWSMessage struct {
	Type    string `json:"type"`
	Session string `json:"session,omitempty"` // optional; URL sessionId is used when absent
	Data    string `json:"data,omitempty"`
	Cols    int    `json:"cols,omitempty"`
	Rows    int    `json:"rows,omitempty"`
}

type innerWSResponse struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}

func NewDVEInnerAgentWSHandler(mgr InnerAgentManagerInterface) *DVEInnerAgentWSHandler {
	return &DVEInnerAgentWSHandler{
		agentManager: mgr,
		active:       make(map[string]*innerWSConn),
	}
}

func (h *DVEInnerAgentWSHandler) ensureInnerAgentReady(ctx context.Context, dveID string) error {
	if h.agentManager == nil {
		return errors.New("agent manager not available")
	}

	if _, _, err := h.agentManager.InnerAgentClient(dveID); err == nil {
		return nil
	}

	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := h.agentManager.StartAgent(startCtx, dveID, 30*time.Second); err != nil {
		return err
	}

	return nil
}

func (h *DVEInnerAgentWSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["dveId"]
	if dveID == "" {
		http.Error(w, "dveId required", http.StatusBadRequest)
		return
	}
	sessionID := vars["sessionId"]

	if h.agentManager == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}

	if err := h.ensureInnerAgentReady(r.Context(), dveID); err != nil {
		log.Printf("[DVE Inner Agent WS] No agent for DVE %s: %v", dveID, err)
		http.Error(w, "KNIRVAGENT inner agent not available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[DVE Inner Agent WS] Upgrade error DVE=%s: %v", dveID, err)
		return
	}

	wsConn := &innerWSConn{
		dveID:     dveID,
		sessionID: sessionID,
		conn:      conn,
		send:      make(chan []byte, 256),
		lastSeen:  time.Now(),
	}

	h.mu.Lock()
	if existing, ok := h.active[dveID+"/"+sessionID]; ok {
		existing.conn.Close()
	}
	h.active[dveID+"/"+sessionID] = wsConn
	h.mu.Unlock()

	log.Printf("[DVE Inner Agent WS] Connected: DVE=%s session=%s", dveID, sessionID)

	if sessionID != "" {
		go h.streamSession(wsConn, sessionID)
	}
	go h.writePump(wsConn)
	go h.readPump(wsConn)
}

// streamSession streams PTY output from the KNIRVAGENT to the WebSocket client.
// Uses http.Client so Go's HTTP stack strips the response headers and handles
// chunked transfer encoding — only the raw PTY bytes reach the browser.
func (h *DVEInnerAgentWSHandler) streamSession(conn *innerWSConn, sessionID string) {
	client, _, err := h.agentManager.InnerAgentClient(conn.dveID)
	if err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "cannot connect to agent: " + err.Error()})
		return
	}

	// The default InnerAgentClient has a 30 s overall timeout which would kill the
	// streaming body as soon as the terminal goes idle. Use a no-timeout clone that
	// shares the same Unix-socket transport.
	streamClient := &http.Client{Transport: client.Transport}
	resp, err := streamClient.Get("http://unix/api/inner/" + sessionID + "/stream")
	if err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "stream request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			conn.send <- mustMarshal(innerWSResponse{Type: "data", Data: string(buf[:n])})
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[DVE Inner Agent WS] Stream read error DVE=%s session=%s: %v", conn.dveID, sessionID, err)
			}
			conn.send <- mustMarshal(innerWSResponse{Type: "stream_end"})
			return
		}
	}
}

func (h *DVEInnerAgentWSHandler) readPump(conn *innerWSConn) {
	defer func() {
		h.mu.Lock()
		delete(h.active, conn.dveID+"/"+conn.sessionID)
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
				log.Printf("[DVE Inner Agent WS] Read error DVE=%s: %v", conn.dveID, err)
			}
			break
		}

		var msg innerWSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[DVE Inner Agent WS] Invalid message from DVE=%s: %v", conn.dveID, err)
			continue
		}

		// Use the URL session ID as default when the message doesn't include one.
		targetSession := msg.Session
		if targetSession == "" {
			targetSession = conn.sessionID
		}

		switch msg.Type {
		case "input":
			if targetSession == "" {
				conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "session required"})
				continue
			}
			go h.forwardInput(conn, targetSession, msg.Data)
		case "resize":
			if targetSession != "" && (msg.Cols > 0 || msg.Rows > 0) {
				go h.forwardResize(conn, targetSession, msg.Cols, msg.Rows)
			}
		case "ping":
			conn.send <- mustMarshal(innerWSResponse{Type: "pong"})
		default:
			log.Printf("[DVE Inner Agent WS] Ignoring unknown message type %q from DVE=%s", msg.Type, conn.dveID)
		}
	}
}

func (h *DVEInnerAgentWSHandler) forwardInput(conn *innerWSConn, sessionID, input string) {
	client, _, err := h.agentManager.InnerAgentClient(conn.dveID)
	if err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "agent unavailable: " + err.Error()})
		return
	}

	body, _ := json.Marshal(map[string]string{"data": input})
	resp, err := client.Post("http://unix/api/inner/"+sessionID+"/input", "application/json", bytes.NewReader(body))
	if err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "input failed: " + err.Error()})
		return
	}
	resp.Body.Close()
}

func (h *DVEInnerAgentWSHandler) forwardResize(conn *innerWSConn, sessionID string, cols, rows int) {
	client, _, err := h.agentManager.InnerAgentClient(conn.dveID)
	if err != nil {
		return
	}
	body, _ := json.Marshal(map[string]int{"cols": cols, "rows": rows})
	resp, err := client.Post("http://unix/api/inner/"+sessionID+"/resize", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (h *DVEInnerAgentWSHandler) writePump(conn *innerWSConn) {
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

func (h *DVEInnerAgentWSHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws/dve/{dveId}/inner/stream/{sessionId}", h.HandleWebSocket)
	r.HandleFunc("/ws/dve/{dveId}/inner", h.HandleWebSocket)
}
