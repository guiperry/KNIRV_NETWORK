package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/knirvcli"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// KNIRVCLIHandlers provides HTTP handlers for KNIRVCLI integration
type KNIRVCLIHandlers struct {
	knirvcliService *knirvcli.KNIRVCLIService
}

// NewKNIRVCLIHandlers creates new KNIRVCLI handlers
func NewKNIRVCLIHandlers(knirvcliService *knirvcli.KNIRVCLIService) *KNIRVCLIHandlers {
	return &KNIRVCLIHandlers{
		knirvcliService: knirvcliService,
	}
}

// RegisterRoutes registers KNIRVCLI routes
func (h *KNIRVCLIHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	knirvcliRouter := r.PathPrefix("/api/knirvcli").Subrouter()

	// Protected routes
	if authMiddleware != nil {
		protectedRouter := knirvcliRouter.PathPrefix("").Subrouter()
		protectedRouter.Use(authMiddleware.RequireAuth)
		protectedRouter.HandleFunc("/execute", h.ExecuteCommand).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/sessions", h.ListSessions).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/sessions/{id}/stop", h.StopSession).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/sessions/{id}/input", h.SendInput).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/wallet/info", h.GetWalletInfo).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/wallet/send", h.SendToken).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/validation/execute", h.ExecuteValidation).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/tee/execute", h.ExecuteTEECommand).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/p2p/execute", h.ExecuteP2PCommand).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/chain/execute", h.ExecuteChainCommand).Methods("POST", "OPTIONS")
	} else {
		// No auth required for testnet mode
		knirvcliRouter.HandleFunc("/execute", h.ExecuteCommand).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/sessions", h.ListSessions).Methods("GET", "OPTIONS")
		knirvcliRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET", "OPTIONS")
		knirvcliRouter.HandleFunc("/sessions/{id}/stop", h.StopSession).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/sessions/{id}/input", h.SendInput).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/wallet/info", h.GetWalletInfo).Methods("GET", "OPTIONS")
		knirvcliRouter.HandleFunc("/wallet/send", h.SendToken).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/validation/execute", h.ExecuteValidation).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/tee/execute", h.ExecuteTEECommand).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/p2p/execute", h.ExecuteP2PCommand).Methods("POST", "OPTIONS")
		knirvcliRouter.HandleFunc("/chain/execute", h.ExecuteChainCommand).Methods("POST", "OPTIONS")
	}
}

// ExecuteCommand handles POST /api/knirvcli/execute
func (h *KNIRVCLIHandlers) ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var req knirvcli.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, `{"error":"command is required"}`, http.StatusBadRequest)
		return
	}

	// Get user from auth context
	authCtx := middleware.GetAuthContext(r)
	if authCtx != nil {
		req.UserID = authCtx.UserID
		req.Username = authCtx.Username
	}

	result, err := h.knirvcliService.ExecuteCommand(r.Context(), &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListSessions handles GET /api/knirvcli/sessions
func (h *KNIRVCLIHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.knirvcliService.ListSessions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// GetSession handles GET /api/knirvcli/sessions/{id}
func (h *KNIRVCLIHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := h.knirvcliService.GetSession(sessionID)
	if err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// StopSession handles POST /api/knirvcli/sessions/{id}/stop
func (h *KNIRVCLIHandlers) StopSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if err := h.knirvcliService.StopSession(sessionID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "stopped",
		"session_id": sessionID,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// SendInput handles POST /api/knirvcli/sessions/{id}/input
func (h *KNIRVCLIHandlers) SendInput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.knirvcliService.SendInput(sessionID, req.Input); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "sent",
		"session_id": sessionID,
	})
}

// GetWalletInfo handles GET /api/knirvcli/wallet/info
func (h *KNIRVCLIHandlers) GetWalletInfo(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, `{"error":"address parameter required"}`, http.StatusBadRequest)
		return
	}

	info, err := h.knirvcliService.GetWalletInfo(r.Context(), address)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// SendToken handles POST /api/knirvcli/wallet/send
func (h *KNIRVCLIHandlers) SendToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.From == "" || req.To == "" || req.Amount <= 0 {
		http.Error(w, `{"error":"from, to, and amount are required"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.knirvcliService.SendToken(r.Context(), req.From, req.To, req.Amount)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
}

// ExecuteValidation handles POST /api/knirvcli/validation/execute
func (h *KNIRVCLIHandlers) ExecuteValidation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.knirvcliService.ExecuteValidation(r.Context(), req.TaskID, req.NodeID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "submitted",
		"task_id": req.TaskID,
	})
}

// ExecuteTEECommand handles POST /api/knirvcli/tee/execute
func (h *KNIRVCLIHandlers) ExecuteTEECommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
		NodeID  string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvcliService.ExecuteTEECommand(r.Context(), req.Command, req.NodeID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExecuteP2PCommand handles POST /api/knirvcli/p2p/execute
func (h *KNIRVCLIHandlers) ExecuteP2PCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
		NodeID  string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvcliService.ExecuteP2PCommand(r.Context(), req.Command, req.NodeID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExecuteChainCommand handles POST /api/knirvcli/chain/execute
func (h *KNIRVCLIHandlers) ExecuteChainCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvcliService.ExecuteChainCommand(r.Context(), req.Command)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
