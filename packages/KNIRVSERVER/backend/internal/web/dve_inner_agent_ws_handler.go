package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type InnerAgentManagerInterface interface {
	InnerAgentClient(dveID string) (*http.Client, string, error)
}

type DVEInnerAgentWSHandler struct {
	agentManager InnerAgentManagerInterface
	mu           sync.RWMutex
	active       map[string]*innerWSConn
}

type innerWSConn struct {
	dveID      string
	conn       *websocket.Conn
	send       chan []byte
	socketPath string
	lastSeen   time.Time
	mu         sync.Mutex
}

type innerWSMessage struct {
	Type    string `json:"type"`
	Session string `json:"session,omitempty"`
	Data    string `json:"data,omitempty"`
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

	_, socketPath, err := h.agentManager.InnerAgentClient(dveID)
	if err != nil {
		log.Printf("[DVE Inner Agent WS] No agent for DVE %s: %v", dveID, err)
		http.Error(w, "no agent for DVE: "+err.Error(), http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[DVE Inner Agent WS] Upgrade error DVE=%s: %v", dveID, err)
		return
	}

	wsConn := &innerWSConn{
		dveID:      dveID,
		conn:       conn,
		send:       make(chan []byte, 256),
		socketPath: socketPath,
		lastSeen:   time.Now(),
	}

	h.mu.Lock()
	if existing, ok := h.active[dveID]; ok {
		existing.conn.Close()
	}
	h.active[dveID] = wsConn
	h.mu.Unlock()

	log.Printf("[DVE Inner Agent WS] Connected: DVE=%s socket=%s session=%s", dveID, socketPath, sessionID)

	if sessionID != "" {
		go h.streamSession(wsConn, sessionID)
	}
	go h.writePump(wsConn)
	go h.readPump(wsConn)
}

func (h *DVEInnerAgentWSHandler) streamSession(conn *innerWSConn, sessionID string) {
	unixDialer := net.Dialer{Timeout: 5 * time.Second}
	unixConn, err := unixDialer.Dial("unix", conn.socketPath)
	if err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "cannot connect to agent: " + err.Error()})
		return
	}
	defer unixConn.Close()

	httpReq := "GET /api/inner/" + sessionID + "/stream HTTP/1.1\r\nHost: unix\r\n\r\n"
	if _, err := unixConn.Write([]byte(httpReq)); err != nil {
		conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "stream request failed: " + err.Error()})
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := unixConn.Read(buf)
		if n > 0 {
			// Skip HTTP headers on first read
			data := buf[:n]
			if len(data) > 0 {
				conn.send <- mustMarshal(innerWSResponse{Type: "data", Data: string(data)})
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[DVE Inner Agent WS] Stream read error DVE=%s: %v", conn.dveID, err)
			}
			conn.send <- mustMarshal(innerWSResponse{Type: "stream_end", Data: ""})
			return
		}
	}
}

func (h *DVEInnerAgentWSHandler) readPump(conn *innerWSConn) {
	defer func() {
		h.mu.Lock()
		delete(h.active, conn.dveID)
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

		switch msg.Type {
		case "input":
			if msg.Session == "" {
				conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "session required"})
				continue
			}
			go h.forwardInput(conn, msg.Session, msg.Data)
		case "ping":
			conn.send <- mustMarshal(innerWSResponse{Type: "pong"})
		default:
			conn.send <- mustMarshal(innerWSResponse{Type: "error", Data: "unknown type: " + msg.Type})
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
