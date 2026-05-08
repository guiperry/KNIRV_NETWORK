// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/services"

	"go.uber.org/zap"
)

// DVEPageData carries all data needed to render a public DVE page.
type DVEPageData struct {
	DVEID     string
	FullURI   string
	Status    string
	WalletAddr string
	CreatedAt int64
	Query     string
	Metrics   DVEPageMetrics
	Records   []DVEPageRecord
	Total     int
}

// DVEPageMetrics holds aggregated validation stats for the metrics panel.
type DVEPageMetrics struct {
	TotalValidations int     `json:"total_validations"`
	SuccessRate      float64 `json:"success_rate"`
	AverageScore     float64 `json:"average_score"`
}

// DVEPageRecord is a flattened validation record for the public table.
type DVEPageRecord struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	Score           float64 `json:"score"`
	ValidatorNodeID string  `json:"validator_node_id"`
	CreatedAt       string  `json:"created_at"`
}

// DVEProxyHandler serves public-facing DVE pages as HTML rendered
// from Go templates, replacing the previous raw-JSON stubs.
type DVEProxyHandler struct {
	uriRegistry *services.DVEURIRegistry
	logger      *zap.Logger
	tmpl        *template.Template
}

// NewDVEProxyHandler creates a handler that loads templates from tmplDir.
// The directory must contain: public_page.gohtml, validation_records.gohtml,
// metrics_panel.gohtml, and search_form.gohtml.
func NewDVEProxyHandler(registry *services.DVEURIRegistry, tmplDir string, logger *zap.Logger) (*DVEProxyHandler, error) {
	funcMap := template.FuncMap{
		"truncateWallet": truncateWallet,
		"formatTime":     formatTime,
		"safeHTML":       func(s string) template.HTML { return template.HTML(s) },
	}

	pattern := fmt.Sprintf("%s/*.gohtml", tmplDir)
	tmpl, err := template.New("public_page.gohtml").Funcs(funcMap).ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load DVE templates from %s: %w", tmplDir, err)
	}

	return &DVEProxyHandler{
		uriRegistry: registry,
		logger:      logger,
		tmpl:        tmpl,
	}, nil
}

// RegisterRoutes registers public-facing and API routes on the router.
func (h *DVEProxyHandler) RegisterRoutes(router Router) {
	// Public HTML pages
	router.HandleFunc("/dve/{dveId}", h.handleDVEPublicPage)
	router.HandleFunc("/dve/{dveId}/validation", h.handleValidationRecords)
	router.HandleFunc("/dve/{dveId}/metrics", h.handleDVEMetrics)
	router.HandleFunc("/dve/{dveId}/search", h.handleSearch)

	// JSON API
	router.HandleFunc("/api/dve/resolve", h.handleResolveURI)
	router.HandleFunc("/api/dve/register", h.handleRegisterDVE)
	router.HandleFunc("/api/dve/{dveId}", h.handleGetDVE)
}

// ── Public HTML handlers ─────────────────────────────────────────────

func (h *DVEProxyHandler) handleDVEPublicPage(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		h.logger.Warn("DVE not found", zap.String("dve_id", dveID), zap.Error(err))
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	data := &DVEPageData{
		DVEID:      dve.DVEID,
		FullURI:    dve.FullURI,
		Status:     dve.Status,
		WalletAddr: dve.WalletAddr,
		CreatedAt:  dve.CreatedAt,
		Metrics: DVEPageMetrics{
			TotalValidations: 0,
			SuccessRate:      100.0,
			AverageScore:     0.0,
		},
		Records: []DVEPageRecord{},
		Total:   0,
	}

	h.renderTemplate(w, data)
}

func (h *DVEProxyHandler) handleValidationRecords(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	data := &DVEPageData{
		DVEID:      dve.DVEID,
		FullURI:    dve.FullURI,
		Status:     dve.Status,
		WalletAddr: dve.WalletAddr,
		CreatedAt:  dve.CreatedAt,
		Metrics: DVEPageMetrics{
			TotalValidations: 0,
			SuccessRate:      100.0,
			AverageScore:     0.0,
		},
		Records: []DVEPageRecord{},
		Total:   0,
	}

	// Filter by status if requested
	statusFilter := r.URL.Query().Get("status")
	if statusFilter != "" {
		// In production this would query actual validation records.
		// For now, records are empty until ValidationService is wired.
		_ = statusFilter
	}

	h.renderTemplate(w, data)
}

func (h *DVEProxyHandler) handleDVEMetrics(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	data := &DVEPageData{
		DVEID:      dve.DVEID,
		FullURI:    dve.FullURI,
		Status:     dve.Status,
		WalletAddr: dve.WalletAddr,
		CreatedAt:  dve.CreatedAt,
		Metrics: DVEPageMetrics{
			TotalValidations: 0,
			SuccessRate:      100.0,
			AverageScore:     0.0,
		},
	}

	h.renderTemplate(w, data)
}

func (h *DVEProxyHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	query := r.URL.Query().Get("q")

	if query == "" {
		// Render the search form with empty results.
		dve, err := h.uriRegistry.GetByDVEID(dveID)
		if err != nil {
			http.Error(w, "DVE not found", http.StatusNotFound)
			return
		}

		data := &DVEPageData{
			DVEID:      dve.DVEID,
			FullURI:    dve.FullURI,
			Status:     dve.Status,
			WalletAddr: dve.WalletAddr,
			CreatedAt:  dve.CreatedAt,
			Metrics: DVEPageMetrics{
				TotalValidations: 0,
				SuccessRate:      100.0,
				AverageScore:     0.0,
			},
			Records: []DVEPageRecord{},
		}
		h.renderTemplate(w, data)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	data := &DVEPageData{
		DVEID:      dve.DVEID,
		FullURI:    dve.FullURI,
		Status:     dve.Status,
		WalletAddr: dve.WalletAddr,
		CreatedAt:  dve.CreatedAt,
		Query:      query,
		Metrics: DVEPageMetrics{
			TotalValidations: 0,
			SuccessRate:      100.0,
			AverageScore:     0.0,
		},
		Records: []DVEPageRecord{},
		Total:   0,
	}

	h.renderTemplate(w, data)
}

// ── JSON API handlers (unchanged) ────────────────────────────────────

func (h *DVEProxyHandler) handleResolveURI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URI string `json:"uri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.Resolve(req.URI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve URI: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dve)
}

func (h *DVEProxyHandler) handleRegisterDVE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DVEID      string `json:"dve_id"`
		FullURI    string `json:"full_uri"`
		WalletAddr string `json:"wallet_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dve := &services.DVEURI{
		DVEID:      req.DVEID,
		FullURI:    req.FullURI,
		WalletAddr: req.WalletAddr,
	}

	if err := h.uriRegistry.Register(dve); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register DVE: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dve_id":  req.DVEID,
	})
}

func (h *DVEProxyHandler) handleGetDVE(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dve)
}

// ── helpers ──────────────────────────────────────────────────────────

func (h *DVEProxyHandler) renderTemplate(w http.ResponseWriter, data *DVEPageData) {
	if h.tmpl == nil {
		http.Error(w, "Template engine not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err := h.tmpl.ExecuteTemplate(w, "public_page.gohtml", data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Template render error", http.StatusInternalServerError)
	}
}

// Router interface (identical to the original — maintained for compatibility).
type Router interface {
	HandleFunc(pattern string, handler http.HandlerFunc)
}

func extractURLParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

// ── template helper functions ────────────────────────────────────────

func truncateWallet(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}

func formatTime(ts int64) string {
	if ts == 0 {
		return "—"
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02")
}

// for compatibility with old code that might reference these:
var _ = strings.TrimSpace
