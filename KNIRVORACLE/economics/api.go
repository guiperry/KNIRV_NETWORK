package economics

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type EconomicsAPI struct {
	tokenEconomics *TokenEconomics
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SkillInvocationRequest struct {
	UserID  string `json:"user_id"`
	SkillID string `json:"skill_id"`
	Amount  string `json:"amount"`
}

type LLMRegistrationRequest struct {
	UserID          string `json:"user_id"`
	LLMID           string `json:"llm_id"`
	RegistrationFee string `json:"registration_fee"`
}

type ValidationRewardRequest struct {
	ValidatorID      string `json:"validator_id"`
	TargetID         string `json:"target_id"`
	ValidationResult bool   `json:"validation_result"`
}

type NetworkFeesRequest struct {
	GasUsed  uint64 `json:"gas_used"`
	Priority string `json:"priority"`
}

// Treasury-related types
type NRVMetadata struct {
	NRVID             string                 `json:"nrv_id"`
	ProofID           string                 `json:"proof_id"`
	NodeID            string                 `json:"node_id"`
	RouteData         map[string]interface{} `json:"route_data"`
	ConnectivityScore float64                `json:"connectivity_score"`
	NRNValue          string                 `json:"nrn_value"`
	Timestamp         string                 `json:"timestamp"`
	Signature         string                 `json:"signature"`
	TreasuryAddress   string                 `json:"treasury_address"`
	Status            string                 `json:"status"`
}

type TreasuryTransferRequest struct {
	TransferID   string       `json:"transfer_id"`
	NRVMetadata  *NRVMetadata `json:"nrv_metadata"`
	TargetWallet string       `json:"target_wallet"`
	Timestamp    string       `json:"timestamp"`
}

type TreasuryStatus struct {
	TotalNRVReceived int64  `json:"total_nrv_received"`
	TotalNRNMinted   string `json:"total_nrn_minted"`
	RecentInflow     int64  `json:"recent_inflow_nrv"`
	LastTransfer     string `json:"last_transfer"`
	HealthStatus     string `json:"health_status"`
}

func NewEconomicsAPI(tokenEconomics *TokenEconomics) *EconomicsAPI {
	return &EconomicsAPI{
		tokenEconomics: tokenEconomics,
	}
}

func (api *EconomicsAPI) RegisterRoutes(router *mux.Router) {
	// Economic operations
	router.HandleFunc("/economics/skill/invoke", api.handleSkillInvocation).Methods("POST")
	router.HandleFunc("/economics/llm/register", api.handleLLMRegistration).Methods("POST")
	router.HandleFunc("/economics/validation/reward", api.handleValidationReward).Methods("POST")
	router.HandleFunc("/economics/fees/calculate", api.handleCalculateNetworkFees).Methods("POST")

	// Metrics and data retrieval
	router.HandleFunc("/economics/metrics", api.handleGetMetrics).Methods("GET")
	router.HandleFunc("/economics/transaction/{id}", api.handleGetTransaction).Methods("GET")
	router.HandleFunc("/economics/transactions", api.handleGetTransactions).Methods("GET")
	router.HandleFunc("/economics/burn/history", api.handleGetBurnHistory).Methods("GET")
	router.HandleFunc("/economics/burn/total", api.handleGetTotalBurned).Methods("GET")

	// Economic rules and configuration
	router.HandleFunc("/economics/rules", api.handleGetEconomicRules).Methods("GET")
	router.HandleFunc("/economics/rules", api.handleUpdateEconomicRules).Methods("PUT")

	// Service-specific metrics
	router.HandleFunc("/economics/service/{service}/metrics", api.handleGetServiceMetrics).Methods("GET")

	// Treasury operations
	router.HandleFunc("/api/treasury/receive-nrv", api.handleReceiveNRV).Methods("POST")
	router.HandleFunc("/api/treasury/nrv-inflow", api.handleGetNRVInflow).Methods("GET")
	router.HandleFunc("/api/treasury/status", api.handleGetTreasuryStatus).Methods("GET")

	// Health check
	router.HandleFunc("/economics/health", api.handleHealthCheck).Methods("GET")
}

func (api *EconomicsAPI) handleSkillInvocation(w http.ResponseWriter, r *http.Request) {
	var req SkillInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		api.sendError(w, http.StatusBadRequest, "Invalid amount format")
		return
	}

	// Process skill invocation
	tx, err := api.tokenEconomics.ProcessSkillInvocation(req.UserID, req.SkillID, amount)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.sendSuccess(w, map[string]interface{}{
		"transaction_id": tx.ID,
		"status":         tx.Status,
		"amount":         tx.Amount.String(),
		"timestamp":      tx.Timestamp,
	})
}

func (api *EconomicsAPI) handleLLMRegistration(w http.ResponseWriter, r *http.Request) {
	var req LLMRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse registration fee
	fee, ok := new(big.Int).SetString(req.RegistrationFee, 10)
	if !ok {
		api.sendError(w, http.StatusBadRequest, "Invalid registration fee format")
		return
	}

	// Process LLM registration
	tx, err := api.tokenEconomics.ProcessLLMRegistration(req.UserID, req.LLMID, fee)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.sendSuccess(w, map[string]interface{}{
		"transaction_id": tx.ID,
		"status":         tx.Status,
		"fee":            tx.Amount.String(),
		"timestamp":      tx.Timestamp,
	})
}

func (api *EconomicsAPI) handleValidationReward(w http.ResponseWriter, r *http.Request) {
	var req ValidationRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Process validation reward
	tx, err := api.tokenEconomics.ProcessValidationReward(req.ValidatorID, req.TargetID, req.ValidationResult)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.sendSuccess(w, map[string]interface{}{
		"transaction_id": tx.ID,
		"status":         tx.Status,
		"reward":         tx.Amount.String(),
		"timestamp":      tx.Timestamp,
	})
}

func (api *EconomicsAPI) handleCalculateNetworkFees(w http.ResponseWriter, r *http.Request) {
	var req NetworkFeesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Calculate network fees
	totalFee := api.tokenEconomics.CalculateNetworkFees(req.GasUsed, req.Priority)

	api.sendSuccess(w, map[string]interface{}{
		"gas_used":  req.GasUsed,
		"priority":  req.Priority,
		"total_fee": totalFee.String(),
		"gas_price": new(big.Int).Div(totalFee, big.NewInt(int64(req.GasUsed))).String(),
	})
}

func (api *EconomicsAPI) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := api.tokenEconomics.GetEconomicMetrics()
	api.sendSuccess(w, metrics)
}

func (api *EconomicsAPI) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txID := vars["id"]

	// Get transaction from pool
	api.tokenEconomics.transactionPool.mutex.RLock()
	tx, exists := api.tokenEconomics.transactionPool.confirmedTxs[txID]
	if !exists {
		tx, exists = api.tokenEconomics.transactionPool.pendingTxs[txID]
	}
	api.tokenEconomics.transactionPool.mutex.RUnlock()

	if !exists {
		api.sendError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	api.sendSuccess(w, tx)
}

func (api *EconomicsAPI) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	statusFilter := r.URL.Query().Get("status")

	limit := 100 // default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get transactions from pool
	api.tokenEconomics.transactionPool.mutex.RLock()
	var transactions []*EconomicTransaction

	// Add confirmed transactions
	for _, tx := range api.tokenEconomics.transactionPool.confirmedTxs {
		if statusFilter == "" || tx.Status == statusFilter {
			transactions = append(transactions, tx)
			if len(transactions) >= limit {
				break
			}
		}
	}

	// Add pending transactions if we haven't reached the limit
	if len(transactions) < limit {
		for _, tx := range api.tokenEconomics.transactionPool.pendingTxs {
			if statusFilter == "" || tx.Status == statusFilter {
				transactions = append(transactions, tx)
				if len(transactions) >= limit {
					break
				}
			}
		}
	}
	api.tokenEconomics.transactionPool.mutex.RUnlock()

	api.sendSuccess(w, map[string]interface{}{
		"transactions":  transactions,
		"count":         len(transactions),
		"limit":         limit,
		"status_filter": statusFilter,
	})
}

func (api *EconomicsAPI) handleGetBurnHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	api.tokenEconomics.burnTracker.mutex.RLock()
	burnHistory := api.tokenEconomics.burnTracker.burnHistory

	// Get the most recent burn events up to the limit
	start := 0
	if len(burnHistory) > limit {
		start = len(burnHistory) - limit
	}

	recentBurns := make([]*BurnEvent, len(burnHistory[start:]))
	copy(recentBurns, burnHistory[start:])
	api.tokenEconomics.burnTracker.mutex.RUnlock()

	api.sendSuccess(w, map[string]interface{}{
		"burn_events": recentBurns,
		"count":       len(recentBurns),
		"limit":       limit,
	})
}

func (api *EconomicsAPI) handleGetTotalBurned(w http.ResponseWriter, r *http.Request) {
	totalBurned := api.tokenEconomics.burnTracker.GetTotalBurned()

	api.sendSuccess(w, map[string]interface{}{
		"total_burned": totalBurned.String(),
		"timestamp":    api.tokenEconomics.metrics.LastUpdated,
	})
}

func (api *EconomicsAPI) handleGetEconomicRules(w http.ResponseWriter, r *http.Request) {
	api.tokenEconomics.mutex.RLock()
	rules := api.tokenEconomics.economicRules
	api.tokenEconomics.mutex.RUnlock()

	api.sendSuccess(w, rules)
}

func (api *EconomicsAPI) handleUpdateEconomicRules(w http.ResponseWriter, r *http.Request) {
	var newRules EconomicRules
	if err := json.NewDecoder(r.Body).Decode(&newRules); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	api.tokenEconomics.mutex.Lock()
	api.tokenEconomics.economicRules = &newRules
	api.tokenEconomics.mutex.Unlock()

	log.Println("Economic rules updated")
	api.sendSuccess(w, map[string]interface{}{
		"message": "Economic rules updated successfully",
		"rules":   newRules,
	})
}

func (api *EconomicsAPI) handleGetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["service"]

	api.tokenEconomics.metrics.mutex.RLock()
	serviceMetrics, exists := api.tokenEconomics.metrics.ServiceMetrics[serviceName]
	api.tokenEconomics.metrics.mutex.RUnlock()

	if !exists {
		api.sendError(w, http.StatusNotFound, fmt.Sprintf("Service '%s' not found", serviceName))
		return
	}

	api.sendSuccess(w, serviceMetrics)
}

func (api *EconomicsAPI) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	api.sendSuccess(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": api.tokenEconomics.metrics.LastUpdated,
		"version":   "1.0.0",
	})
}

func (api *EconomicsAPI) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (api *EconomicsAPI) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// Treasury handler functions

func (api *EconomicsAPI) handleReceiveNRV(w http.ResponseWriter, r *http.Request) {
	var req TreasuryTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate NRV metadata
	if req.NRVMetadata == nil {
		api.sendError(w, http.StatusBadRequest, "NRV metadata is required")
		return
	}

	// Process NRV transfer and mint corresponding NRN tokens
	nrnAmount, err := api.processNRVTransfer(&req)
	if err != nil {
		log.Printf("Failed to process NRV transfer: %v", err)
		api.sendError(w, http.StatusInternalServerError, "Failed to process NRV transfer")
		return
	}

	log.Printf("Received NRV %s from KNIRVROUTER, minted %s NRN",
		req.NRVMetadata.NRVID, nrnAmount.String())

	api.sendSuccess(w, map[string]interface{}{
		"transfer_id":      req.TransferID,
		"nrv_id":           req.NRVMetadata.NRVID,
		"nrn_minted":       nrnAmount.String(),
		"status":           "confirmed",
		"treasury_balance": "updated",
	})
}

func (api *EconomicsAPI) handleGetNRVInflow(w http.ResponseWriter, r *http.Request) {
	// Get recent NRV inflow metrics
	metrics := api.tokenEconomics.GetEconomicMetrics()

	// Calculate recent inflow (last 24 hours)
	recentInflow := int64(10) // Placeholder - should be calculated from actual data

	api.sendSuccess(w, map[string]interface{}{
		"recent_inflow_nrv":  recentInflow,
		"total_supply":       metrics.TotalSupply.String(),
		"circulating_supply": metrics.CirculatingSupply.String(),
		"last_updated":       "2025-01-09T12:00:00Z",
	})
}

func (api *EconomicsAPI) handleGetTreasuryStatus(w http.ResponseWriter, r *http.Request) {
	metrics := api.tokenEconomics.GetEconomicMetrics()

	status := TreasuryStatus{
		TotalNRVReceived: 150, // Placeholder - should be tracked
		TotalNRNMinted:   metrics.TotalSupply.String(),
		RecentInflow:     10, // Placeholder - should be calculated
		LastTransfer:     "2025-01-09T12:00:00Z",
		HealthStatus:     "healthy",
	}

	api.sendSuccess(w, status)
}

// processNRVTransfer processes an NRV transfer and mints corresponding NRN tokens
func (api *EconomicsAPI) processNRVTransfer(req *TreasuryTransferRequest) (*big.Int, error) {
	// Parse NRN value from NRV metadata
	nrnValue, ok := new(big.Int).SetString(req.NRVMetadata.NRNValue, 10)
	if !ok {
		return nil, fmt.Errorf("invalid NRN value: %s", req.NRVMetadata.NRNValue)
	}

	// Create treasury reward transaction
	tx, err := api.tokenEconomics.ProcessTreasuryReward(
		req.TargetWallet,
		req.NRVMetadata.NRVID,
		nrnValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process treasury reward: %w", err)
	}

	log.Printf("Processed treasury reward transaction: %s", tx.ID)
	return nrnValue, nil
}
