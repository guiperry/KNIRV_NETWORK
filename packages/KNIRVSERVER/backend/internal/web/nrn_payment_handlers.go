package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/tidwall/buntdb"
)

type NRNPaymentHandlers struct {
	transactionChainClient interface {
		GetAccountBalance(address string) (int64, error)
		VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
		GetBlockHeight() (uint64, error)
		TransferNRN(senderID, recipientID string, amount int64) (*objects.NRNPayment, error)
	}
	db *buntdb.DB
}

func NewNRNPaymentHandlers(client interface {
	GetAccountBalance(address string) (int64, error)
	VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
	GetBlockHeight() (uint64, error)
	TransferNRN(senderID, recipientID string, amount int64) (*objects.NRNPayment, error)
}, db *buntdb.DB) *NRNPaymentHandlers {
	return &NRNPaymentHandlers{
		transactionChainClient: client,
		db:                        db,
	}
}

type NRNResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type TransferNRNRequest struct {
	SenderID     string  `json:"sender_id"`
	RecipientID  string  `json:"recipient_id"`
	Amount       int64   `json:"amount"`
	IdempotencyKey string  `json:"idempotency_key,omitempty"`
}

type TransferNRNResponse struct {
	TxHash   string  `json:"tx_hash"`
	Status   string  `json:"status"`
	Amount   int64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *NRNPaymentHandlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		h.sendError(w, "Address is required", http.StatusBadRequest)
		return
	}

	balance, err := h.transactionChainClient.GetAccountBalance(address)
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

	payment, err := h.transactionChainClient.VerifyPaymentTransaction(req.TxHash, req.ExpectedAmount, req.ExpectedRecipient)
	if err != nil {
		h.sendError(w, "Payment verification failed: "+err.Error(), http.StatusPaymentRequired)
		return
	}

	h.sendJSON(w, payment, "Payment verified successfully", http.StatusOK)
}

func (h *NRNPaymentHandlers) GetBlockHeight(w http.ResponseWriter, r *http.Request) {
	height, err := h.transactionChainClient.GetBlockHeight()
	if err != nil {
		h.sendError(w, "Failed to get block height: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, map[string]uint64{"height": height}, "Block height retrieved successfully", http.StatusOK)
}

// TransferNRN handles NRN token transfers with idempotency support
func (h *NRNPaymentHandlers) TransferNRN(w http.ResponseWriter, r *http.Request) {
	var req TransferNRNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.SenderID == "" || req.RecipientID == "" {
		h.sendError(w, "Sender and recipient IDs are required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		h.sendError(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	// Generate idempotency key if not provided
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		// Create a deterministic key based on sender, recipient, amount, and timestamp
		keyData := fmt.Sprintf("%s:%s:%d:%d", req.SenderID, req.RecipientID, req.Amount, time.Now().Unix())
		hash := sha256.Sum256([]byte(keyData))
		idempotencyKey = hex.EncodeToString(hash[:])
	}

	// Check if we've already processed this idempotency key
	existingTx, err := h.getTransactionByIdempotencyKey(idempotencyKey)
	if err != nil && err != buntdb.ErrNotFound {
		h.sendError(w, fmt.Sprintf("Error checking idempotency: %v", err), http.StatusInternalServerError)
		return
	}
	if existingTx != nil {
		// Return existing transaction instead of creating a duplicate
		response := TransferNRNResponse{
			TxHash:   existingTx.TxHash,
			Status:   existingTx.Status,
			Amount:   existingTx.Amount,
			CreatedAt: existingTx.CreatedAt,
		}
		h.sendJSON(w, response, "Transaction already processed (idempotent)", http.StatusOK)
		return
	}

	// Process the transfer
	tx, err := h.transactionChainClient.TransferNRN(req.SenderID, req.RecipientID, req.Amount)
	if err != nil {
		h.sendError(w, "Transfer failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store the transaction with idempotency key for future reference
	if err := h.saveTransactionWithIdempotencyKey(tx, idempotencyKey); err != nil {
		// Log but don't fail - transfer already happened
		fmt.Printf("Failed to save idempotency key: %v\n", err)
	}

	response := TransferNRNResponse{
		TxHash:   tx.TxHash,
		Status:   tx.Status,
		Amount:   tx.Amount,
		CreatedAt: tx.CreatedAt,
	}
	h.sendJSON(w, response, "Transfer completed successfully", http.StatusOK)
}

// getTransactionByIdempotencyKey retrieves a transaction by its idempotency key
func (h *NRNPaymentHandlers) getTransactionByIdempotencyKey(key string) (*objects.NRNPayment, error) {
	var result *objects.NRNPayment
	err := h.db.View(func(btx *buntdb.Tx) error {
		val, err := btx.Get("idempotency_key:" + key)
		if err != nil {
			return err
		}
		if val == "" {
			return buntdb.ErrNotFound
		}
		var storedTx objects.NRNPayment
		if err := json.Unmarshal([]byte(val), &storedTx); err != nil {
			return err
		}
		result = &storedTx
		return nil
	})
	return result, err
}

// saveTransactionWithIdempotencyKey stores a transaction with an idempotency key
func (h *NRNPaymentHandlers) saveTransactionWithIdempotencyKey(tx *objects.NRNPayment, key string) error {
	return h.db.Update(func(btx *buntdb.Tx) error {
		// Store the transaction
		txData, err := json.Marshal(tx)
		if err != nil {
			return err
		}

		// Store with expiration (e.g., 24 hours) to prevent infinite growth
		expiration := time.Now().Add(24 * time.Hour)
		_, _, err = btx.Set(
			"idempotency_key:"+key,
			string(txData),
			&buntdb.SetOptions{Expires: true, TTL: expiration.Sub(time.Now())},
		)
		return err
	})
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
		protectedRouter.HandleFunc("/transfer", h.TransferNRN).Methods("POST", "OPTIONS")
	} else {
		apiRouter.HandleFunc("/balance", h.GetBalance).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/verify-payment", h.VerifyPayment).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/block-height", h.GetBlockHeight).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/transfer", h.TransferNRN).Methods("POST", "OPTIONS")
	}
}
