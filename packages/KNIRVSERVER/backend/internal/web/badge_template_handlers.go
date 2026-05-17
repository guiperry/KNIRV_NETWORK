// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"backend_server/internal/services/badge"

	"github.com/gorilla/mux"
)

// BadgeTemplateHandlers provides HTTP handlers for badge template CRUD + mint.
type BadgeTemplateHandlers struct {
	registry *badge.BadgeTemplateRegistry
}

// NewBadgeTemplateHandlers creates handlers wired to the template registry.
func NewBadgeTemplateHandlers(registry *badge.BadgeTemplateRegistry) *BadgeTemplateHandlers {
	return &BadgeTemplateHandlers{
		registry: registry,
	}
}

// RegisterRoutes attaches badge template endpoints to the router.
func (h *BadgeTemplateHandlers) RegisterRoutes(r *mux.Router) {
	sub := r.PathPrefix("/api/badge/templates").Subrouter()
	sub.HandleFunc("", h.ListTemplates).Methods("GET", "OPTIONS")
	sub.HandleFunc("", h.CreateTemplate).Methods("POST", "OPTIONS")
	sub.HandleFunc("/{id}", h.GetTemplate).Methods("GET", "OPTIONS")
	sub.HandleFunc("/{id}", h.UpdateTemplate).Methods("PUT", "OPTIONS")
	sub.HandleFunc("/{id}", h.DeleteTemplate).Methods("DELETE", "OPTIONS")
	sub.HandleFunc("/{id}/mint", h.MintFromTemplate).Methods("POST", "OPTIONS")

	// DVE badge attachment management
	sub.HandleFunc("/dve/{dveId}/badges", h.ListBadgesForDVE).Methods("GET", "OPTIONS")
	sub.HandleFunc("/dve/{dveId}/badges/{badgeId}", h.DetachBadgeFromDVE).Methods("DELETE", "OPTIONS")

	log.Println("[badge-templates] routes registered: /api/badge/templates/*")
}

// ── Create ──

// CreateTemplate handles POST /api/badge/templates.
// Accepts a BadgeTemplate body (without ID/timestamps), generates SVG,
// persists to registry, and returns the full template.
func (h *BadgeTemplateHandlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl badge.BadgeTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	if tmpl.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	// Generate SVG from the template parameters.
	svgParams := badge.BadgeSVGParams{
		Name:             tmpl.Name,
		BadgeType:        tmpl.BadgeType,
		Description:      tmpl.Description,
		ValueSignals:     tmpl.ValueSignals,
		OntologySignals:  tmpl.OntologySignals,
		AITextTag:        tmpl.AITextTag,
		OutputSize:       400,
		PrimaryColor:     tmpl.PrimaryColor,
		SecondaryColor:   tmpl.SecondaryColor,
		BackgroundColor:  tmpl.BackgroundColor,
	}
	if svgParams.PrimaryColor == "" {
		svgParams.PrimaryColor = "#FBBF24"
	}
	if svgParams.SecondaryColor == "" {
		svgParams.SecondaryColor = "#D97706"
	}
	if svgParams.BackgroundColor == "" {
		svgParams.BackgroundColor = "#02040a"
	}
	tmpl.SVGDesign = badge.GenerateBadgeSVG(svgParams)

	if err := h.registry.Create(&tmpl); err != nil {
		log.Printf("[badge-templates] create error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create template"})
		return
	}

	log.Printf("[badge-templates] created template %s (%s)", tmpl.ID, tmpl.Name)
	writeJSON(w, http.StatusCreated, tmpl)
}

// ── List ──

// ListTemplates handles GET /api/badge/templates.
// Supports ?active=true to filter only active templates.
func (h *BadgeTemplateHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	templates, err := h.registry.List(activeOnly)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list templates"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// ── Get ──

// GetTemplate handles GET /api/badge/templates/{id}.
func (h *BadgeTemplateHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	tmpl, err := h.registry.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, tmpl)
}

// ── Update ──

// UpdateTemplate handles PUT /api/badge/templates/{id}.
func (h *BadgeTemplateHandlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var tmpl badge.BadgeTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	// If signals or colors changed, regenerate SVG.
	if tmpl.ValueSignals != nil || tmpl.OntologySignals != nil ||
		tmpl.PrimaryColor != "" || tmpl.SecondaryColor != "" || tmpl.BackgroundColor != "" {
		svgParams := badge.BadgeSVGParams{
			Name:             tmpl.Name,
			BadgeType:        tmpl.BadgeType,
			Description:      tmpl.Description,
			ValueSignals:     tmpl.ValueSignals,
			OntologySignals:  tmpl.OntologySignals,
			AITextTag:        tmpl.AITextTag,
			OutputSize:       400,
			PrimaryColor:     tmpl.PrimaryColor,
			SecondaryColor:   tmpl.SecondaryColor,
			BackgroundColor:  tmpl.BackgroundColor,
		}
		if svgParams.PrimaryColor == "" {
			svgParams.PrimaryColor = "#FBBF24"
		}
		if svgParams.SecondaryColor == "" {
			svgParams.SecondaryColor = "#D97706"
		}
		if svgParams.BackgroundColor == "" {
			svgParams.BackgroundColor = "#02040a"
		}
		tmpl.SVGDesign = badge.GenerateBadgeSVG(svgParams)
	}

	if err := h.registry.Update(id, &tmpl); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Return updated template.
	updated, _ := h.registry.Get(id)
	writeJSON(w, http.StatusOK, updated)
}

// ── Delete ──

// DeleteTemplate handles DELETE /api/badge/templates/{id}.
// Soft-deletes (archives) the template.
func (h *BadgeTemplateHandlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.registry.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ── Mint from Template ──

// mintRequest is the body for POST /api/badge/templates/{id}/mint.
type mintRequest struct {
	DVEID string `json:"dve_id"`
}

// MintFromTemplate handles POST /api/badge/templates/{id}/mint.
// Looks up the template, creates an on-chain badge via KNIRVSHELL,
// and attaches it to the specified DVE via GuardrailInjector.
//
// NOTE: This handler is a scaffold that returns the template data so
// the frontend can call the existing /api/knirvshell/chain/badge/create
// endpoint for the actual on-chain mint. The full automint flow will
// be wired once the KNIRVSHELL svc service is available here.
func (h *BadgeTemplateHandlers) MintFromTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req mintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	tmpl, err := h.registry.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	if !tmpl.IsActive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "template is archived"})
		return
	}

	// Return template data so the frontend can call the existing
	// /api/knirvshell/chain/badge/create endpoint with these params.
	// In a future phase, this handler will call knirvshell.Service.CreateBadge()
	// and GuardrailInjector.AttachBadge() directly.
	// For now, record the attachment so ListBadgesForDVE can return it.
	attachedBadgeMu.Lock()
	dveAttachedBadges[req.DVEID] = append(dveAttachedBadges[req.DVEID], tmpl.ID)
	attachedBadgeMu.Unlock()

	log.Printf("[badge-templates] recorded badge %s attachment to DVE %s", tmpl.ID, req.DVEID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"template_id":   tmpl.ID,
		"template_name": tmpl.Name,
		"dve_id":        req.DVEID,
		"badge_params": map[string]interface{}{
			"name":            tmpl.Name,
			"badge_type":      tmpl.BadgeType,
			"description":     tmpl.Description,
			"image_data":      tmpl.SVGDesign,
			"value_signals":   tmpl.ValueSignals,
			"ontology_signals": tmpl.OntologySignals,
			"ai_text_tag":     tmpl.AITextTag,
			"alignment_threshold": tmpl.AlignmentThreshold,
		},
		"status": "ready_for_mint",
	})
}

// ── helpers ──

// ── DVE Badge Attachment ──

// attachedBadgeStore is a simple in-memory store tracking which badge IDs
// are attached to which DVEs. This complements the MintFromTemplate flow
// by recording the attachment so ListBadgesForDVE can return it.
// In production this would be persisted alongside the GuardrailInjector state.
var (
	attachedBadgeMu     sync.RWMutex
	dveAttachedBadges   = make(map[string][]string) // dveID → []badgeID
)

// ListBadgesForDVE handles GET /api/badge/templates/dve/{dveId}/badges.
// Returns the list of badge IDs currently attached to this DVE.
func (h *BadgeTemplateHandlers) ListBadgesForDVE(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]

	attachedBadgeMu.RLock()
	ids := dveAttachedBadges[dveID]
	attachedBadgeMu.RUnlock()

	if ids == nil {
		ids = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dve_id":          dveID,
		"badge_ids":       ids,
		"attached_badges": ids,
		"count":           len(ids),
	})
}

// DetachBadgeFromDVE handles DELETE /api/badge/templates/dve/{dveId}/badges/{badgeId}.
// Removes a badge from the DVE's attached list.
func (h *BadgeTemplateHandlers) DetachBadgeFromDVE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["dveId"]
	badgeID := vars["badgeId"]

	attachedBadgeMu.Lock()
	ids := dveAttachedBadges[dveID]
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != badgeID {
			filtered = append(filtered, id)
		}
	}
	dveAttachedBadges[dveID] = filtered
	attachedBadgeMu.Unlock()

	log.Printf("[badge-templates] detached badge %s from DVE %s", badgeID, dveID)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "detached",
		"badge_id": badgeID,
		"dve_id":   dveID,
	})
}

// ── helpers ──

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
