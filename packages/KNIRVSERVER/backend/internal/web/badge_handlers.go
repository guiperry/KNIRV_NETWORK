// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"encoding/json"
	"net/http"

	knirvshell "github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL"

	"backend_server/internal/services/badge"

	"github.com/gorilla/mux"
)

// BadgeHandlers provides HTTP handlers for badge lifecycle operations
// beyond what KNIRVSHELL already covers (create/mint/get).
type BadgeHandlers struct {
	svcService *knirvshell.KNIRVSHELLService
}

// NewBadgeHandlers creates badge-specific HTTP handlers.
func NewBadgeHandlers(svc *knirvshell.KNIRVSHELLService) *BadgeHandlers {
	return &BadgeHandlers{
		svcService: svc,
	}
}

// RegisterRoutes attaches badge endpoints to the router.
func (h *BadgeHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/badge/generate", h.GenerateBadgeSVG).Methods("POST", "OPTIONS")
}

// GenerateBadgeSVG handles POST /api/badge/generate.
// Accepts a JSON body with BadgeSVGParams and returns the generated SVG.
//
// This replaces the client-side SVG stub in badge-lab-panel.tsx with
// server-side rendering that supports AI text tags and signal visualization.
func (h *BadgeHandlers) GenerateBadgeSVG(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var params badge.BadgeSVGParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if params.Name == "" {
		params.Name = "KNIRV Badge"
	}
	if params.OutputSize <= 0 {
		params.OutputSize = 400
	}

	svg := badge.GenerateBadgeSVG(params)

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(svg))
}
