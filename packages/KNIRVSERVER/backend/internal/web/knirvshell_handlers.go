package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/web/middleware"

	knirvshell "github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL"

	"github.com/gorilla/mux"
)

type KNIRVSHELLHandlers struct {
	knirvshellService *knirvshell.KNIRVSHELLService
}

func NewKNIRVSHELLHandlers(svc *knirvshell.KNIRVSHELLService) *KNIRVSHELLHandlers {
	return &KNIRVSHELLHandlers{
		knirvshellService: svc,
	}
}

func (h *KNIRVSHELLHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	knirvshellRouter := r.PathPrefix("/api/knirvshell").Subrouter()

	if authMiddleware != nil {
		protectedRouter := knirvshellRouter.PathPrefix("").Subrouter()
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
		protectedRouter.HandleFunc("/chain/badge/create", h.CreateBadge).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/chain/badge/mint", h.MintBadge).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/chain/badge/{id}", h.GetBadge).Methods("GET", "OPTIONS")
	} else {
		knirvshellRouter.HandleFunc("/execute", h.ExecuteCommand).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/sessions", h.ListSessions).Methods("GET", "OPTIONS")
		knirvshellRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET", "OPTIONS")
		knirvshellRouter.HandleFunc("/sessions/{id}/stop", h.StopSession).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/sessions/{id}/input", h.SendInput).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/wallet/info", h.GetWalletInfo).Methods("GET", "OPTIONS")
		knirvshellRouter.HandleFunc("/wallet/send", h.SendToken).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/validation/execute", h.ExecuteValidation).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/tee/execute", h.ExecuteTEECommand).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/p2p/execute", h.ExecuteP2PCommand).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/chain/execute", h.ExecuteChainCommand).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/chain/badge/create", h.CreateBadge).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/chain/badge/mint", h.MintBadge).Methods("POST", "OPTIONS")
		knirvshellRouter.HandleFunc("/chain/badge/{id}", h.GetBadge).Methods("GET", "OPTIONS")
	}
}

func (h *KNIRVSHELLHandlers) ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var req knirvshell.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, `{"error":"command is required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.ExecuteCommand(r.Context(), &req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.knirvshellService.ListSessions()
	json.NewEncoder(w).Encode(sessions)
}

func (h *KNIRVSHELLHandlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command  string `json:"command"`
		NodeID   string `json:"node_id"`
		Streaming bool  `json:"streaming"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Default to "terminal:start" if no command specified
	command := req.Command
	if command == "" {
		command = "terminal:start"
	}

	// ExecuteCommand creates a session and returns a CommandResult with session_id
	result, err := h.knirvshellService.ExecuteCommand(r.Context(), &knirvshell.CommandRequest{
		Command: command,
		NodeID:  req.NodeID,
	})
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"session_id": result.SessionID,
	})
}

func (h *KNIRVSHELLHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := h.knirvshellService.GetSession(sessionID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(session)
}

func (h *KNIRVSHELLHandlers) StopSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if err := h.knirvshellService.StopSession(sessionID); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "stopped",
		"id":     sessionID,
	})
}

func (h *KNIRVSHELLHandlers) SendInput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.knirvshellService.SendInput(sessionID, req.Input); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "input sent",
	})
}

func (h *KNIRVSHELLHandlers) GetWalletInfo(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")

	info, err := h.knirvshellService.GetWalletInfo(r.Context(), address)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(info)
}

func (h *KNIRVSHELLHandlers) SendToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.knirvshellService.SendToken(r.Context(), req.From, req.To, req.Amount)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(tx)
}

func (h *KNIRVSHELLHandlers) ExecuteValidation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "validation_requested",
		"task_id": req.TaskID,
		"node_id": req.NodeID,
	})
}

func (h *KNIRVSHELLHandlers) ExecuteTEECommand(w http.ResponseWriter, r *http.Request) {
	var req knirvshell.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.ExecuteTEECommand(r.Context(), req.Command, req.NodeID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) ExecuteP2PCommand(w http.ResponseWriter, r *http.Request) {
	var req knirvshell.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.ExecuteP2PCommand(r.Context(), req.Command, req.NodeID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) ExecuteChainCommand(w http.ResponseWriter, r *http.Request) {
	var req knirvshell.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.ExecuteChainCommand(r.Context(), req.Command)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) CreateBadge(w http.ResponseWriter, r *http.Request) {
	var req knirvshell.BadgeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.CreateBadge(r.Context(), &req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) MintBadge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BadgeID string `json:"badge_id"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.knirvshellService.MintBadge(r.Context(), req.BadgeID, req.AgentID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (h *KNIRVSHELLHandlers) GetBadge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	badgeID := vars["id"]

	badge, err := h.knirvshellService.GetBadge(r.Context(), badgeID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(badge)
}
