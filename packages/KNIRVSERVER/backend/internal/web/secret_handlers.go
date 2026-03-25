package web

import (
	"encoding/json"
	"net/http"

	secretssvc "backend_server/internal/services/secrets"

	"github.com/gorilla/mux"
)

type SecretHandlers struct {
	secretManager *secretssvc.SecretManager
}

func NewSecretHandlers(sm *secretssvc.SecretManager) *SecretHandlers {
	return &SecretHandlers{
		secretManager: sm,
	}
}

func (h *SecretHandlers) RegisterRoutes(r *mux.Router) {
	secretRouter := r.PathPrefix("/api/secrets").Subrouter()

	secretRouter.HandleFunc("/create", h.CreateSecret).Methods("POST", "OPTIONS")
	secretRouter.HandleFunc("/list", h.ListSecrets).Methods("GET", "OPTIONS")
	secretRouter.HandleFunc("/{id}", h.GetSecret).Methods("GET", "OPTIONS")
	secretRouter.HandleFunc("/{id}", h.DeleteSecret).Methods("DELETE", "OPTIONS")

	secretRouter.HandleFunc("/sessions/create", h.CreateSession).Methods("POST", "OPTIONS")
	secretRouter.HandleFunc("/sessions/list", h.ListSessions).Methods("GET", "OPTIONS")
	secretRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET", "OPTIONS")
	secretRouter.HandleFunc("/sessions/{id}/rotate", h.RotateSession).Methods("POST", "OPTIONS")
	secretRouter.HandleFunc("/sessions/{id}/revoke", h.RevokeSession).Methods("POST", "OPTIONS")
	secretRouter.HandleFunc("/sessions/{id}/add-secret", h.AddSecretToSession).Methods("POST", "OPTIONS")
	secretRouter.HandleFunc("/sessions/{id}/retrieve", h.RetrieveSecrets).Methods("POST", "OPTIONS")
}

func (h *SecretHandlers) CreateSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string                 `json:"name"`
		Type        secretssvc.SecretType  `json:"type"`
		OwnerID     string                 `json:"owner_id"`
		Value       string                 `json:"value"`
		SessionOnly bool                   `json:"session_only"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Value == "" {
		http.Error(w, `{"error":"name and value required"}`, http.StatusBadRequest)
		return
	}

	ownerID := req.OwnerID
	if ownerID == "" {
		ownerID = "system"
	}

	secret, err := h.secretManager.CreateSecret(req.Name, req.Type, ownerID, req.Value, req.SessionOnly, req.Metadata)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"secret": secret,
	})
}

func (h *SecretHandlers) GetSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secretID := vars["id"]

	secretsList := h.secretManager.ListSecrets("")
	var found *secretssvc.Secret
	for _, s := range secretsList {
		if s.ID == secretID {
			found = s
			break
		}
	}

	if found == nil {
		http.Error(w, `{"error":"secret not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(found)
}

func (h *SecretHandlers) ListSecrets(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")

	secretsList := h.secretManager.ListSecrets(ownerID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"secrets": secretsList,
		"count":   len(secretsList),
	})
}

func (h *SecretHandlers) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secretID := vars["id"]

	ownerID := r.URL.Query().Get("owner_id")
	if ownerID == "" {
		ownerID = "system"
	}

	if err := h.secretManager.DeleteSecret(secretID, ownerID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "deleted",
		"secret_id": secretID,
	})
}

func (h *SecretHandlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID       string   `json:"node_id"`
		OwnerID      string   `json:"owner_id"`
		ChainSession string   `json:"chain_session"`
		Permissions  []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" {
		http.Error(w, `{"error":"node_id required"}`, http.StatusBadRequest)
		return
	}

	ownerID := req.OwnerID
	if ownerID == "" {
		ownerID = "system"
	}

	permissions := req.Permissions
	if permissions == nil {
		permissions = []string{"read"}
	}

	session, err := h.secretManager.CreateSession(req.NodeID, ownerID, req.ChainSession, permissions)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "created",
		"session": session,
	})
}

func (h *SecretHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, ok := h.secretManager.GetSession(sessionID)
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(session)
}

func (h *SecretHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")

	sessions := h.secretManager.ListSessions(ownerID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func (h *SecretHandlers) RotateSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := h.secretManager.RotateSession(sessionID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "rotated",
		"session": session,
	})
}

func (h *SecretHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if err := h.secretManager.RevokeSession(sessionID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "revoked",
		"session_id": sessionID,
	})
}

func (h *SecretHandlers) AddSecretToSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		SecretID string `json:"secret_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.SecretID == "" {
		http.Error(w, `{"error":"secret_id required"}`, http.StatusBadRequest)
		return
	}

	if err := h.secretManager.AddSecretToSession(sessionID, req.SecretID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "added",
		"session_id": sessionID,
		"secret_id":  req.SecretID,
	})
}

func (h *SecretHandlers) RetrieveSecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		SecretKeys []string `json:"secret_keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if len(req.SecretKeys) == 0 {
		http.Error(w, `{"error":"secret_keys required"}`, http.StatusBadRequest)
		return
	}

	response, err := h.secretManager.RetrieveSecrets(sessionID, req.SecretKeys)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "retrieved",
		"secrets": response.Secrets,
		"errors":  response.Errors,
	})
}
