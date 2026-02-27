package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/blockchain"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type NRNPaymentHandlers struct {
	blockchainClient *blockchain.NRNClient
}

func NewNRNPaymentHandlers(client *blockchain.NRNClient) *NRNPaymentHandlers {
	return &NRNPaymentHandlers{
		blockchainClient: client,
	}
}

type NRNResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func (h *NRNPaymentHandlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		h.sendError(w, "Address is required", http.StatusBadRequest)
		return
	}

	balance, err := h.blockchainClient.GetAccountBalance(address)
	if err != nil {
		h.sendError(w, "Failed to get balance: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, map[string]int64{"balance": balance}, "Balance retrieved successfully", http.StatusOK)
}

func (h *NRNPaymentHandlers) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxHash            string `json:"tx_hash"`
		ExpectedAmount    int64  `json:"expected_amount"`
		ExpectedRecipient string `json:"expected_recipient"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	payment, err := h.blockchainClient.VerifyPaymentTransaction(req.TxHash, req.ExpectedAmount, req.ExpectedRecipient)
	if err != nil {
		h.sendError(w, "Payment verification failed: "+err.Error(), http.StatusPaymentRequired)
		return
	}

	h.sendJSON(w, payment, "Payment verified successfully", http.StatusOK)
}

func (h *NRNPaymentHandlers) GetBlockHeight(w http.ResponseWriter, r *http.Request) {
	height, err := h.blockchainClient.GetBlockHeight()
	if err != nil {
		h.sendError(w, "Failed to get block height: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, map[string]uint64{"height": height}, "Block height retrieved successfully", http.StatusOK)
}

func (h *NRNPaymentHandlers) sendError(w http.ResponseWriter, message string, code int) {
	response := NRNResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (h *NRNPaymentHandlers) sendJSON(w http.ResponseWriter, data interface{}, message string, code int) {
	response := NRNResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (h *NRNPaymentHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	apiRouter := r.PathPrefix("/api/nrn").Subrouter()

	if authMiddleware != nil {
		protectedRouter := apiRouter.PathPrefix("").Subrouter()
		protectedRouter.Use(authMiddleware.RequireAuth)
		protectedRouter.HandleFunc("/balance", h.GetBalance).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/verify-payment", h.VerifyPayment).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/block-height", h.GetBlockHeight).Methods("GET", "OPTIONS")
	} else {
		apiRouter.HandleFunc("/balance", h.GetBalance).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/verify-payment", h.VerifyPayment).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/block-height", h.GetBlockHeight).Methods("GET", "OPTIONS")
	}
}
