// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/validation"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var timeNow = time.Now

func getTimestamp() string {
	return timeNow().UTC().Format("15:04:05")
}

type DVEPageData struct {
	DVEID      string
	FullURI    string
	Status     string
	WalletAddr string
	CreatedAt  int64
	Query      string
	Metrics    DVEPageMetrics
	Records    []DVEPageRecord
	Total      int

	RewardPoolAddress string
	NodeIP            string
	TEEType           string
	ReputationScore   int
	LastHeartbeat     string
	UptimeSeconds     int64
}

type DVEPageMetrics struct {
	TotalValidations int     `json:"total_validations"`
	SuccessRate      float64 `json:"success_rate"`
	AverageScore     float64 `json:"average_score"`
}

type DVEPageRecord struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	Score           float64 `json:"score"`
	ValidatorNodeID string  `json:"validator_node_id"`
	CreatedAt       string  `json:"created_at"`
}

type DVEProxyHandlerOption func(*DVEProxyHandler)

func WithDVEManager(dm *dvemanager.DVEManager) DVEProxyHandlerOption {
	return func(h *DVEProxyHandler) { h.dveManager = dm }
}

func WithValidationCore(vc *validation.ValidationCore) DVEProxyHandlerOption {
	return func(h *DVEProxyHandler) { h.validationCore = vc }
}

func WithRewardPoolAddress(addr string) DVEProxyHandlerOption {
	return func(h *DVEProxyHandler) { h.rewardPoolAddr = addr }
}

type DVEProxyHandler struct {
	uriRegistry    *services.DVEURIRegistry
	dveManager     *dvemanager.DVEManager
	validationCore *validation.ValidationCore
	rewardPoolAddr string
	logger         *zap.Logger
	tmpl           *template.Template
}

func NewDVEProxyHandler(registry *services.DVEURIRegistry, tmplFS fs.FS, logger *zap.Logger, opts ...DVEProxyHandlerOption) (*DVEProxyHandler, error) {
	funcMap := template.FuncMap{
		"truncateWallet": truncateWallet,
		"formatTime":     formatTime,
		"safeHTML":       func(s string) template.HTML { return template.HTML(s) },
		"slicestr":       func(s string, start, end int) string { if start > len(s) { return "" }; if end > len(s) { end = len(s) }; return s[start:end] },
		"GetTimestamp":   getTimestamp,
	}
	tmpl, err := template.New("public_page.gohtml").Funcs(funcMap).ParseFS(tmplFS, "*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("failed to load DVE templates: %w", err)
	}
	h := &DVEProxyHandler{uriRegistry: registry, logger: logger, tmpl: tmpl}
	for _, opt := range opts {
		opt(h)
	}
	return h, nil
}

func (h *DVEProxyHandler) RegisterRoutes(router interface {
	HandleFunc(string, func(http.ResponseWriter, *http.Request)) *mux.Route
}) {
	router.HandleFunc("/dve/{dveId}", h.handleDVEPublicPage)
	router.HandleFunc("/dve/{dveId}/validation", h.handleValidationRecords)
	router.HandleFunc("/dve/{dveId}/metrics", h.handleDVEMetrics)
	router.HandleFunc("/dve/{dveId}/search", h.handleSearch)
	router.HandleFunc("/dve/{dveId}/validate", h.handleInitiatePublicValidation)
	router.HandleFunc("/api/dve/{dveId}/verify-proof/{resultId}", h.handleVerifyProof)
	router.HandleFunc("/api/dve/resolve", h.handleResolveURI)
	router.HandleFunc("/api/dve/register", h.handleRegisterDVE)
	router.HandleFunc("/api/dve/{dveId}", h.handleGetDVE)
}

// resolveDVE looks up a DVE first in the URI registry (fast path), then falls
// back to dveManager.GetNode so the page works even when the registry is empty.
func (h *DVEProxyHandler) resolveDVE(dveID string) (*services.DVEURI, error) {
	if dve, err := h.uriRegistry.GetByDVEID(dveID); err == nil {
		return dve, nil
	}
	if h.dveManager == nil {
		return nil, fmt.Errorf("DVE not found: %s", dveID)
	}
	node, err := h.dveManager.GetNode(dveID)
	if err != nil {
		return nil, fmt.Errorf("DVE not found: %s", dveID)
	}
	return &services.DVEURI{
		ID:        node.ID,
		DVEID:     node.ID,
		FullURI:   fmt.Sprintf("knirv://%s", node.ID),
		Status:    node.Status,
		CreatedAt: node.CreatedAt.Unix(),
	}, nil
}

func (h *DVEProxyHandler) handleDVEPublicPage(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}
	dve, err := h.resolveDVE(dveID)
	if err != nil {
		h.logger.Warn("DVE not found", zap.String("dve_id", dveID), zap.Error(err))
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}
	h.renderTemplate(w, h.buildPageData(dve, ""))
}

func (h *DVEProxyHandler) handleValidationRecords(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}
	dve, err := h.resolveDVE(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}
	data := h.buildPageData(dve, "")
	statusFilter := r.URL.Query().Get("status")
	if statusFilter != "" && h.validationCore != nil {
		tasks, err := h.validationCore.GetValidationTasks(&validation.TaskFilter{Status: statusFilter})
		if err == nil {
			var filtered []*objects.ValidationTask
			for _, t := range tasks {
				if t.AssignedNodeID == dveID {
					filtered = append(filtered, t)
				}
			}
			data.Records = buildPageRecords(filtered)
			data.Total = len(filtered)
		}
	}
	h.renderTemplate(w, data)
}

func (h *DVEProxyHandler) handleDVEMetrics(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}
	dve, err := h.resolveDVE(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}
	h.renderTemplate(w, h.buildPageData(dve, ""))
}

func (h *DVEProxyHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	query := r.URL.Query().Get("q")
	dve, err := h.resolveDVE(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}
	h.renderTemplate(w, h.buildPageData(dve, query))
}

func (h *DVEProxyHandler) buildPageData(dve *services.DVEURI, query string) *DVEPageData {
	data := &DVEPageData{
		DVEID: dve.DVEID, FullURI: dve.FullURI, Status: dve.Status,
		WalletAddr: dve.WalletAddr, CreatedAt: dve.CreatedAt, Query: query,
		Metrics: DVEPageMetrics{TotalValidations: 0, SuccessRate: 100.0, AverageScore: 0.0},
		Records: []DVEPageRecord{}, Total: 0,
	}
	if h.dveManager != nil {
		if node, err := h.dveManager.GetNode(dve.DVEID); err == nil && node != nil {
			data.NodeIP = node.IPAddress
			data.TEEType = node.TEEType
			data.ReputationScore = node.ReputationScore
			data.LastHeartbeat = node.LastHeartbeat.Format(time.RFC3339)
			if data.Status == "" {
				data.Status = node.Status
			}
		}
	}
	if h.validationCore != nil {
		if tasks, err := h.validationCore.GetValidationTasks(&validation.TaskFilter{}); err == nil {
			var filtered []*objects.ValidationTask
			for _, t := range tasks {
				if t.AssignedNodeID == dve.DVEID {
					filtered = append(filtered, t)
				}
			}
			data.Records = buildPageRecords(filtered)
			data.Total = len(filtered)
			data.Metrics.TotalValidations = len(filtered)
		}
	}
	data.RewardPoolAddress = h.rewardPoolAddr
	return data
}

func buildPageRecords(tasks []*objects.ValidationTask) []DVEPageRecord {
	if len(tasks) == 0 {
		return []DVEPageRecord{}
	}
	records := make([]DVEPageRecord, 0, len(tasks))
	for _, t := range tasks {
		records = append(records, DVEPageRecord{
			ID: t.ID, Type: t.Type, Status: t.Status,
			ValidatorNodeID: t.AssignedNodeID, CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	return records
}

func (h *DVEProxyHandler) handleInitiatePublicValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}
	var req struct {
		SignedTx   string `json:"signed_tx"`
		Payload    string `json:"payload"`
		WalletAddr string `json:"wallet_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.SignedTx == "" || req.WalletAddr == "" {
		http.Error(w, "signed_tx and wallet_address are required", http.StatusBadRequest)
		return
	}
	if _, err := h.uriRegistry.GetByDVEID(dveID); err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	var sessionID, proof, timestamp string
	status := "pending"

	if h.validationCore != nil {
		task, err := h.validationCore.CreateValidationTask(&validation.CreateTaskRequest{
			Type: "public_validation", RequestedBy: req.WalletAddr,
			Data: map[string]interface{}{"dve_id": dveID, "payload": req.Payload, "signed_tx": req.SignedTx, "public": true},
		})
		if err == nil {
			sessionID = task.ID
			h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", dveID, task.ID, timeNow().Unix())))
			proof = fmt.Sprintf("PROOF_V1:%s:%s", dveID, hex.EncodeToString(h[:]))
			timestamp = timeNow().Format(time.RFC3339)
			status = "success"
		}
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("dve_validation_%s_%d", dveID, timeNow().Unix())
		h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", dveID, timeNow().Unix())))
		proof = fmt.Sprintf("PROOF_V1:%s:%s", dveID, hex.EncodeToString(h[:]))
		timestamp = timeNow().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID, "proof": proof, "timestamp": timestamp,
		"status": status, "dve_id": dveID,
	})
}

func (h *DVEProxyHandler) handleVerifyProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dveID := extractURLParam(r, "dveId")
	resultID := extractURLParam(r, "resultId")
	if dveID == "" || resultID == "" {
		http.Error(w, "dveId and resultId are required", http.StatusBadRequest)
		return
	}

	var valid bool
	var nodeID, timestamp string
	var score float64

	if h.validationCore != nil {
		if tasks, err := h.validationCore.GetValidationTasks(&validation.TaskFilter{}); err == nil {
			for _, t := range tasks {
				if t.ID == resultID || t.ID == dveID {
					nodeID = t.AssignedNodeID
					timestamp = t.CreatedAt.Format(time.RFC3339)
					valid = true
					break
				}
			}
		}
	}
	if !valid {
		valid = strings.HasPrefix(resultID, "PROOF_V1:")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid, "node_id": nodeID, "dve_id": dveID,
		"timestamp": timestamp, "score": score,
	})
}

func (h *DVEProxyHandler) handleResolveURI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ URI string `json:"uri"` }
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
		DVEID string `json:"dve_id"`; FullURI string `json:"full_uri"`; WalletAddr string `json:"wallet_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	dve := &services.DVEURI{DVEID: req.DVEID, FullURI: req.FullURI, WalletAddr: req.WalletAddr}
	if err := h.uriRegistry.Register(dve); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register DVE: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "dve_id": req.DVEID})
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

func extractURLParam(r *http.Request, name string) string {
	if v, ok := mux.Vars(r)[name]; ok {
		return v
	}
	return r.PathValue(name)
}

func truncateWallet(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}

func formatTime(ts int64) string {
	if ts == 0 {
		return "\u2014"
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02")
}

var _ = strings.TrimSpace
