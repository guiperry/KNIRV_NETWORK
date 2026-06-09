package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type PodRegistration struct {
	NodeID      string                 `json:"node_id"`
	TEType      string                 `json:"tee_type"`
	WASMHash    string                 `json:"wasm_hash"`
	Measurement string                 `json:"measurement"`
	PublicKey   string                 `json:"public_key"`
	Timestamp   int64                  `json:"timestamp"`
	Version     string                 `json:"version"`
	Signature   string                 `json:"signature"`
	AuthToken   string                 `json:"auth_token,omitempty"`
}

type PodSession struct {
	SessionID   string          `json:"session_id"`
	NodeID      string          `json:"node_id"`
	WSURL       string          `json:"ws_url"`
	Attestation json.RawMessage `json:"attestation"`
	CreatedAt   time.Time       `json:"created_at"`
	TrustLevel  string          `json:"trust_level"`
}

type DVEPodHandler struct {
	mu       sync.RWMutex
	sessions map[string]*PodSession
	baseURL  string
}

func NewDVEPodHandler(baseURL string) *DVEPodHandler {
	return &DVEPodHandler{
		sessions: make(map[string]*PodSession),
		baseURL:  baseURL,
	}
}

func (h *DVEPodHandler) RegisterRoutes(r *mux.Router) {
	sub := r.PathPrefix("/dve").Subrouter()
	sub.HandleFunc("/pod/register", h.RegisterPod).Methods("POST", "OPTIONS")
	sub.HandleFunc("/pod/sessions", h.ListSessions).Methods("GET", "OPTIONS")
	sub.HandleFunc("/pod/sessions/{sessionID}", h.GetSession).Methods("GET", "OPTIONS")
	sub.HandleFunc("/pod/{podID}/attest", h.AttestPod).Methods("POST", "OPTIONS")
}

func (h *DVEPodHandler) RegisterPod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Attestation PodRegistration `json:"attestation"`
		NodeID      string          `json:"node_id"`
		TEType      string          `json:"tee_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendPodError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	att := req.Attestation
	if att.NodeID == "" {
		att.NodeID = req.NodeID
	}
	if att.TEType == "" {
		att.TEType = req.TEType
	}
	if att.Version == "" {
		att.Version = "dvepod/1.0"
	}

	if att.NodeID == "" {
		sendPodError(w, "Missing node_id in attestation", http.StatusBadRequest)
		return
	}

	sessionID := generateSessionID()
	wsURL := fmt.Sprintf("%s/api/v1/dve/pod/ws/%s", h.baseURL, sessionID)

	attData, _ := json.Marshal(att)

	trustLevel := trustLevelFromTEType(att.TEType)

	session := &PodSession{
		SessionID:   sessionID,
		NodeID:      att.NodeID,
		WSURL:       wsURL,
		Attestation: attData,
		CreatedAt:   time.Now(),
		TrustLevel:  trustLevel,
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	sendPodJSON(w, map[string]interface{}{
		"session_id": sessionID,
		"node_id":    att.NodeID,
		"ws_url":     wsURL,
		"status":     "registered",
	}, "Pod registered successfully", http.StatusCreated)
}

func (h *DVEPodHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var list []map[string]interface{}
	for _, s := range h.sessions {
		list = append(list, map[string]interface{}{
			"session_id": s.SessionID,
			"node_id":    s.NodeID,
			"created_at": s.CreatedAt.Format(time.RFC3339),
		})
	}
	if list == nil {
		list = make([]map[string]interface{}, 0)
	}

	sendPodJSON(w, list, "Sessions retrieved", http.StatusOK)
}

func (h *DVEPodHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		sendPodError(w, "Session not found", http.StatusNotFound)
		return
	}

	sendPodJSON(w, session, "Session retrieved", http.StatusOK)
}

func (h *DVEPodHandler) AttestPod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podID := vars["podID"]

	var req struct {
		TEType      string `json:"tee_type"`
		Measurement string `json:"measurement"`
		PublicKey   string `json:"public_key"`
		Signature   string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendPodError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TEType == "" {
		sendPodError(w, "Missing tee_type in attestation upgrade", http.StatusBadRequest)
		return
	}

	newTrustLevel := trustLevelFromTEType(req.TEType)

	h.mu.Lock()
	defer h.mu.Unlock()

	var targetSessionID string
	var targetSession *PodSession
	for _, s := range h.sessions {
		if s.NodeID == podID {
			targetSessionID = s.SessionID
			targetSession = s
			break
		}
	}
	if targetSession == nil {
		sendPodError(w, "Pod not found", http.StatusNotFound)
		return
	}

	if trustLevelRank(newTrustLevel) <= trustLevelRank(targetSession.TrustLevel) {
		sendPodError(w,
			fmt.Sprintf("Attestation not stronger: current=%s, requested=%s", targetSession.TrustLevel, newTrustLevel),
			http.StatusConflict)
		return
	}

	attData, _ := json.Marshal(req)
	targetSession.Attestation = attData
	targetSession.TrustLevel = newTrustLevel

	sendPodJSON(w, map[string]interface{}{
		"session_id":  targetSessionID,
		"node_id":     podID,
		"trust_level": newTrustLevel,
		"status":      "upgraded",
	}, "Trust level upgraded successfully", http.StatusOK)
}

func trustLevelFromTEType(teeType string) string {
	switch teeType {
	case "sgx", "sevsnp", "sev-snp", "tdx", "nitro":
		return "L3"
	case "browser-wasm":
		return "L1"
	case "wasmer", "wasmtime", "wasi":
		return "L0"
	default:
		return "L0"
	}
}

func trustLevelRank(level string) int {
	switch level {
	case "L3":
		return 3
	case "L2":
		return 2
	case "L1":
		return 1
	case "L0":
		return 0
	default:
		return -1
	}
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "dvepod-" + hex.EncodeToString(b)
}

type PodResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func sendPodError(w http.ResponseWriter, message string, code int) {
	resp := PodResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}

func sendPodJSON(w http.ResponseWriter, data interface{}, message string, code int) {
	resp := PodResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}
