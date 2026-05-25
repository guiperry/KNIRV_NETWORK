package web

import (
	"encoding/json"
	"log"
	"net/http"

	"backend_server/internal/services/inferencer"
)

type GenerateBadgeConfigHandlers struct {
	inferenceService *inferencer.InferenceService
}

func NewGenerateBadgeConfigHandlers(inferenceService *inferencer.InferenceService) *GenerateBadgeConfigHandlers {
	return &GenerateBadgeConfigHandlers{inferenceService: inferenceService}
}

type badgeConfigGenerationRequest struct {
	BadgeID      string   `json:"badge_id"`
	OntologyTags []string `json:"ontology_tags"`
	Description  string   `json:"description,omitempty"`
}

type badgeConfigGenerationResponse struct {
	BadgeID      string   `json:"badge_id"`
	ErrorClassID uint32   `json:"error_class_id"`
	AutoResolve  bool     `json:"auto_resolve"`
	OntologyTags []string `json:"ontology_tags"`
	ConfigJSON   string   `json:"config_json,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func (h *GenerateBadgeConfigHandlers) GenerateBadgeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req badgeConfigGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.BadgeID == "" {
		http.Error(w, `{"error":"badge_id is required"}`, http.StatusBadRequest)
		return
	}

	instruction := "Generate ONLY a JSON badge configuration object. " +
		"Return {\"error_class_id\": <uint32>, \"auto_resolve\": <bool>}. " +
		"error_class_id: 0=none, 1=resource_exhaustion, 2=latency, 3=security, 4=crash. " +
		"No markdown, no explanation."
	configJSON, err := h.inferenceService.GenerateTextWithContext(
		r.Context(),
		"",
		req.Description,
		instruction,
	)
	if err != nil {
		log.Printf("[generate-badge-config] inference error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(badgeConfigGenerationResponse{
			BadgeID: req.BadgeID,
			Error:   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(badgeConfigGenerationResponse{
		BadgeID:      req.BadgeID,
		OntologyTags: req.OntologyTags,
		ConfigJSON:   configJSON,
	})
}
