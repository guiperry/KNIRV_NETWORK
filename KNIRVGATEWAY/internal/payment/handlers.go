package payment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for the payment service
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new payment handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers payment routes with the router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Network status
	router.HandleFunc("/api/network-status", h.handleNetworkStatus).Methods("GET")

	// Faucet endpoints
	router.HandleFunc("/api/faucet/request", h.handleFaucetRequest).Methods("POST")
	router.HandleFunc("/api/faucet/status/{address}", h.handleFaucetStatus).Methods("GET")

	// Payment endpoints
	router.HandleFunc("/api/create-checkout-session", h.handleCreateStripeCheckout).Methods("POST")
	router.HandleFunc("/api/create-coinbase-charge", h.handleCreateCoinbaseCharge).Methods("POST")

	// Economics API endpoints
	router.HandleFunc("/api/economics/skill/invoke", h.handleSkillInvocation).Methods("POST")
	router.HandleFunc("/api/economics/llm/register", h.handleLLMRegistration).Methods("POST")
	router.HandleFunc("/api/economics/validation/reward", h.handleValidationReward).Methods("POST")
	router.HandleFunc("/api/economics/fees/calculate", h.handleCalculateFees).Methods("POST")

	// Economics data endpoints
	router.HandleFunc("/api/economics/metrics", h.handleGetMetrics).Methods("GET")
	router.HandleFunc("/api/economics/transaction/{id}", h.handleGetTransaction).Methods("GET")
	router.HandleFunc("/api/economics/transactions", h.handleGetTransactions).Methods("GET")
	router.HandleFunc("/api/economics/burn/history", h.handleGetBurnHistory).Methods("GET")
	router.HandleFunc("/api/economics/burn/total", h.handleGetTotalBurned).Methods("GET")
	router.HandleFunc("/api/economics/rules", h.handleGetEconomicRules).Methods("GET")
	router.HandleFunc("/api/economics/service/{service}/metrics", h.handleGetServiceMetrics).Methods("GET")

	// Economics integration
	router.HandleFunc("/api/economics/integration/status", h.handleIntegrationStatus).Methods("GET")

	// Health and info
	router.HandleFunc("/api/economics/health", h.handleHealthCheck).Methods("GET")
	router.HandleFunc("/api/economics/info", h.handleSystemInfo).Methods("GET")
}

// handleNetworkStatus handles network status requests
func (h *Handler) handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	network := r.URL.Query().Get("network")
	if network == "" {
		network = "mainnet"
	}

	status := h.service.GetNetworkStatus(network)
	h.sendJSON(w, status)
}

// handleFaucetRequest handles faucet requests
func (h *Handler) handleFaucetRequest(w http.ResponseWriter, r *http.Request) {
	var req FaucetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response := h.service.RequestFaucet(&req)
	h.sendJSON(w, response)
}

// handleFaucetStatus handles faucet status requests
func (h *Handler) handleFaucetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	network := r.URL.Query().Get("network")
	if network == "" {
		network = "public-testnet"
	}

	status := h.service.CheckFaucetStatus(address, network)
	h.sendJSON(w, status)
}

// handleCreateStripeCheckout handles Stripe checkout session creation
func (h *Handler) handleCreateStripeCheckout(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	session, err := h.service.CreateStripeCheckoutSession(&req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.sendJSON(w, session)
}

// handleCreateCoinbaseCharge handles Coinbase charge creation
func (h *Handler) handleCreateCoinbaseCharge(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	charge, err := h.service.CreateCoinbaseCharge(&req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.sendJSON(w, charge)
}

// handleSkillInvocation handles skill invocation requests
func (h *Handler) handleSkillInvocation(w http.ResponseWriter, r *http.Request) {
	var req SkillInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tx, err := h.service.ProcessSkillInvocation(&req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.sendSuccess(w, map[string]interface{}{
		"transactionId": tx.ID,
		"status":        tx.Status,
		"amount":        tx.AmountStr,
		"timestamp":     tx.Timestamp,
	})
}

// handleLLMRegistration handles LLM registration requests
func (h *Handler) handleLLMRegistration(w http.ResponseWriter, r *http.Request) {
	var req LLMRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tx, err := h.service.ProcessLLMRegistration(&req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.sendSuccess(w, map[string]interface{}{
		"transactionId": tx.ID,
		"status":        tx.Status,
		"fee":           tx.AmountStr,
		"timestamp":     tx.Timestamp,
	})
}

// handleValidationReward handles validation reward requests
func (h *Handler) handleValidationReward(w http.ResponseWriter, r *http.Request) {
	var req ValidationRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tx, err := h.service.ProcessValidationReward(&req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.sendSuccess(w, map[string]interface{}{
		"transactionId": tx.ID,
		"status":        tx.Status,
		"reward":        tx.AmountStr,
		"timestamp":     tx.Timestamp,
	})
}

// handleCalculateFees handles fee calculation requests
func (h *Handler) handleCalculateFees(w http.ResponseWriter, r *http.Request) {
	var req FeeCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	response := h.service.CalculateNetworkFees(&req)
	h.sendSuccess(w, response)
}

// handleGetMetrics handles metrics requests
func (h *Handler) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.service.GetEconomicMetrics()
	h.sendSuccess(w, metrics)
}

// handleGetTransaction handles transaction retrieval
func (h *Handler) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	// For now, return a mock response since we don't have direct transaction lookup
	h.sendError(w, http.StatusNotImplemented, "Transaction lookup not implemented")
}

// handleGetTransactions handles transaction list requests
func (h *Handler) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	status := r.URL.Query().Get("status")

	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	transactions := h.service.GetTransactions(limit, status)

	// Convert to response format
	txData := make([]map[string]interface{}, len(transactions))
	for i, tx := range transactions {
		txData[i] = map[string]interface{}{
			"id":        tx.ID,
			"type":      tx.Type,
			"from":      tx.From,
			"to":        tx.To,
			"amount":    tx.AmountStr,
			"purpose":   tx.Purpose,
			"metadata":  tx.Metadata,
			"status":    tx.Status,
			"timestamp": tx.Timestamp,
		}
	}

	h.sendSuccess(w, map[string]interface{}{
		"transactions": txData,
		"count":        len(txData),
		"limit":        limit,
		"statusFilter": status,
	})
}

// handleGetBurnHistory handles burn history requests
func (h *Handler) handleGetBurnHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	user := r.URL.Query().Get("user")
	purpose := r.URL.Query().Get("purpose")

	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	history := h.service.GetBurnHistory(limit, user, purpose)

	// Convert to response format
	historyData := make([]map[string]interface{}, len(history))
	for i, event := range history {
		historyData[i] = map[string]interface{}{
			"txId":      event.TxID,
			"user":      event.User,
			"amount":    event.AmountStr,
			"purpose":   event.Purpose,
			"skillId":   event.SkillID,
			"timestamp": event.Timestamp,
			"validated": event.Validated,
		}
	}

	h.sendSuccess(w, map[string]interface{}{
		"burnEvents": historyData,
		"count":      len(historyData),
		"limit":      limit,
	})
}

// handleGetTotalBurned handles total burned requests
func (h *Handler) handleGetTotalBurned(w http.ResponseWriter, r *http.Request) {
	total := h.service.GetTotalBurned()
	h.sendSuccess(w, map[string]interface{}{
		"totalBurned": total,
		"timestamp":   "current",
	})
}

// handleGetEconomicRules handles economic rules requests
func (h *Handler) handleGetEconomicRules(w http.ResponseWriter, r *http.Request) {
	rules := h.service.GetEconomicRules()
	h.sendSuccess(w, rules)
}

// handleGetServiceMetrics handles service metrics requests
func (h *Handler) handleGetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := vars["service"]

	metrics := h.service.GetServiceMetrics(service)
	h.sendSuccess(w, metrics)
}

// handleIntegrationStatus handles integration status requests
func (h *Handler) handleIntegrationStatus(w http.ResponseWriter, r *http.Request) {
	// Mock integration status for now
	status := map[string]interface{}{
		"status": "operational",
		"services": map[string]interface{}{
			"stripe":      "configured",
			"coinbase":    "configured",
			"economics":   "active",
			"blockchain":  "connected",
		},
		"timestamp": "current",
	}
	h.sendSuccess(w, status)
}

// handleHealthCheck handles health check requests
func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	h.sendSuccess(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": "current",
		"version":   "1.0.0",
		"service":   "KNIRV Payment Gateway & Economics Engine",
	})
}

// handleSystemInfo handles system info requests
func (h *Handler) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	h.sendSuccess(w, map[string]interface{}{
		"service":   "KNIRV Economics Service",
		"version":   "1.0.0",
		"isRunning": true,
		"timestamp": "current",
	})
}

// Helper methods

func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (h *Handler) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}