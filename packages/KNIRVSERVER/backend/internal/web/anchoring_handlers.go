package web

import (
	"encoding/json"
	"net/http"

	evidencesvc "backend_server/internal/services/evidence"

	"github.com/gorilla/mux"
)

type AnchoringHandlers struct {
	anchoringService *evidencesvc.AnchoringService
}

func NewAnchoringHandlers(as *evidencesvc.AnchoringService) *AnchoringHandlers {
	return &AnchoringHandlers{
		anchoringService: as,
	}
}

func (h *AnchoringHandlers) RegisterRoutes(r *mux.Router) {
	anchoringRouter := r.PathPrefix("/api/anchoring").Subrouter()

	anchoringRouter.HandleFunc("/evidence/create", h.CreateEvidencePack).Methods("POST", "OPTIONS")
	anchoringRouter.HandleFunc("/evidence/list", h.ListEvidencePacks).Methods("GET", "OPTIONS")
	anchoringRouter.HandleFunc("/evidence/{id}", h.GetEvidencePack).Methods("GET", "OPTIONS")
	anchoringRouter.HandleFunc("/evidence/{id}/sign", h.SignEvidencePack).Methods("POST", "OPTIONS")
	anchoringRouter.HandleFunc("/evidence/{id}/anchor", h.AnchorToChain).Methods("POST", "OPTIONS")
	anchoringRouter.HandleFunc("/evidence/{id}/verify", h.VerifyEvidencePack).Methods("POST", "OPTIONS")
	anchoringRouter.HandleFunc("/statistics", h.GetStatistics).Methods("GET", "OPTIONS")
}

func (h *AnchoringHandlers) CreateEvidencePack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type         string                 `json:"type"`
		NodeID       string                 `json:"node_id"`
		ValidationID string                 `json:"validation_id"`
		Evidence     map[string]interface{} `json:"evidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Type == "" || req.NodeID == "" {
		http.Error(w, `{"error":"type and node_id required"}`, http.StatusBadRequest)
		return
	}

	validationID := req.ValidationID
	if validationID == "" {
		validationID = "default"
	}

	evidence := req.Evidence
	if evidence == nil {
		evidence = make(map[string]interface{})
	}

	pack, err := h.anchoringService.CreateEvidencePack(req.Type, req.NodeID, validationID, evidence)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"pack":   pack,
	})
}

func (h *AnchoringHandlers) GetEvidencePack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packID := vars["id"]

	pack, ok := h.anchoringService.GetEvidencePack(packID)
	if !ok {
		http.Error(w, `{"error":"evidence pack not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(pack)
}

func (h *AnchoringHandlers) ListEvidencePacks(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	limit := 100

	packs := h.anchoringService.ListEvidencePacks(nodeID, limit)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"packs": packs,
		"count": len(packs),
	})
}

func (h *AnchoringHandlers) SignEvidencePack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packID := vars["id"]

	pack, err := h.anchoringService.SignEvidencePack(packID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "signed",
		"pack":   pack,
	})
}

func (h *AnchoringHandlers) AnchorToChain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packID := vars["id"]

	pack, err := h.anchoringService.AnchorToChain(packID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "anchored",
		"pack":         pack,
		"tx_hash":      pack.ChainTxHash,
		"block_height": pack.BlockHeight,
	})
}

func (h *AnchoringHandlers) VerifyEvidencePack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packID := vars["id"]

	valid, err := h.anchoringService.VerifyEvidencePack(packID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   valid,
		"pack_id": packID,
	})
}

func (h *AnchoringHandlers) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats := h.anchoringService.GetStatistics()
	json.NewEncoder(w).Encode(stats)
}
