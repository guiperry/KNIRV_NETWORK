package uri

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler represents the HTTP handler for URI generation
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new URI generation handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// RegisterRoutes registers the URI generation routes with the router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/uriGenerator", h.handleGenerateURI).Methods("POST")
}

// URIGenerationRequest represents the request payload for URI generation
type URIGenerationRequest struct {
	ResourceType string                 `json:"resource_type"`
	ContentHash  string                 `json:"content_hash,omitempty"`
	Owner        string                 `json:"owner,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// URIGenerationResponse represents the response payload for URI generation
type URIGenerationResponse struct {
	URI        string `json:"uri"`
	ResourceID string `json:"resource_id"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// handleGenerateURI handles the URI generation endpoint
func (h *Handler) handleGenerateURI(w http.ResponseWriter, r *http.Request) {
	var req URIGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode URI generation request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ResourceType == "" {
		http.Error(w, "resource_type is required", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the resource
	resourceID := uuid.New().String()

	// Generate the URI
	uri := GenerateResourceURI(resourceID, req.ResourceType, "", nil)

	response := URIGenerationResponse{
		URI:        uri,
		ResourceID: resourceID,
		Success:    true,
		Message:    "URI generated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode URI generation response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("URI generated",
		zap.String("resource_type", req.ResourceType),
		zap.String("resource_id", resourceID),
		zap.String("uri", uri))
}
