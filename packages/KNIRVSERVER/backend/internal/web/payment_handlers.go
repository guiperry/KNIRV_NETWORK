package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend_server/internal/fintech"
	"backend_server/internal/services/websocket"
)

// PaymentHandlers handles payment-related HTTP requests
type PaymentHandlers struct {
	stripeService     *fintech.StripeService
	paypalService     *fintech.PayPalService
	blockchainService *fintech.BlockchainService
	eventBroadcaster  *websocket.EventBroadcaster
}

// NewPaymentHandlers creates a new PaymentHandlers instance
func NewPaymentHandlers(
	stripeService *fintech.StripeService,
	paypalService *fintech.PayPalService,
	blockchainService *fintech.BlockchainService,
	eventBroadcaster *websocket.EventBroadcaster,
) *PaymentHandlers {
	return &PaymentHandlers{
		stripeService:     stripeService,
		paypalService:     paypalService,
		blockchainService: blockchainService,
		eventBroadcaster:  eventBroadcaster,
	}
}

// CreateStripeCheckoutSession creates a Stripe checkout session
func (h *PaymentHandlers) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if h.stripeService == nil {
		http.Error(w, "Stripe service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Amount        int64             `json:"amount"`
		Currency      string            `json:"currency"`
		ProductName   string            `json:"product_name"`
		SuccessURL    string            `json:"success_url"`
		CancelURL     string            `json:"cancel_url"`
		CustomerEmail string            `json:"customer_email,omitempty"`
		Metadata      map[string]string `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.stripeService.CreateCheckoutSession(fintech.CheckoutSessionRequest{
		Amount:        req.Amount,
		Currency:      req.Currency,
		ProductName:   req.ProductName,
		SuccessURL:    req.SuccessURL,
		CancelURL:     req.CancelURL,
		CustomerEmail: req.CustomerEmail,
		Metadata:      req.Metadata,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast payment event
	if h.eventBroadcaster != nil {
		h.eventBroadcaster.Broadcast("payment:stripe:session_created", map[string]interface{}{
			"session_id": session.ID,
			"amount":     req.Amount,
			"currency":   req.Currency,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// CreateStripePaymentIntent creates a Stripe payment intent
func (h *PaymentHandlers) CreateStripePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount      int64             `json:"amount"`
		Currency    string            `json:"currency"`
		CustomerID  string            `json:"customer_id,omitempty"`
		Description string            `json:"description,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	intent, err := h.stripeService.CreatePaymentIntent(fintech.PaymentIntentRequest{
		Amount:      req.Amount,
		Currency:    req.Currency,
		CustomerID:  req.CustomerID,
		Description: req.Description,
		Metadata:    req.Metadata,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast payment event
	h.eventBroadcaster.Broadcast("payment:stripe:intent_created", map[string]interface{}{
		"intent_id": intent.ID,
		"amount":    req.Amount,
		"currency":  req.Currency,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(intent)
}

// GetStripePaymentIntent retrieves a Stripe payment intent
func (h *PaymentHandlers) GetStripePaymentIntent(w http.ResponseWriter, r *http.Request) {
	intentID := r.URL.Query().Get("intent_id")
	if intentID == "" {
		http.Error(w, "intent_id is required", http.StatusBadRequest)
		return
	}

	intent, err := h.stripeService.GetPaymentIntent(intentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(intent)
}

// RefundStripePayment refunds a Stripe payment
func (h *PaymentHandlers) RefundStripePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentIntentID string `json:"payment_intent_id"`
		Amount          int64  `json:"amount,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	refund, err := h.stripeService.CreateRefund(req.PaymentIntentID, req.Amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast refund event
	h.eventBroadcaster.Broadcast("payment:stripe:refunded", map[string]interface{}{
		"refund_id":         refund.ID,
		"payment_intent_id": req.PaymentIntentID,
		"amount":            refund.Amount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refund)
}

// CreatePayPalOrder creates a PayPal order
func (h *PaymentHandlers) CreatePayPalOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount        int64  `json:"amount"`
		Currency      string `json:"currency"`
		Description   string `json:"description"`
		ReturnURL     string `json:"return_url"`
		CancelURL     string `json:"cancel_url"`
		CustomerEmail string `json:"customer_email,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.paypalService.CreateOrder(fintech.PayPalOrderRequest{
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
		ReturnURL:     req.ReturnURL,
		CancelURL:     req.CancelURL,
		CustomerEmail: req.CustomerEmail,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast payment event
	h.eventBroadcaster.Broadcast("payment:paypal:order_created", map[string]interface{}{
		"order_id": order.ID,
		"amount":   req.Amount,
		"currency": req.Currency,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// GetPayPalOrder retrieves a PayPal order
func (h *PaymentHandlers) GetPayPalOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	order, err := h.paypalService.GetOrder(orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// CapturePayPalOrder captures a PayPal order payment
func (h *PaymentHandlers) CapturePayPalOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	capture, err := h.paypalService.CaptureOrder(req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast payment event
	h.eventBroadcaster.Broadcast("payment:paypal:captured", map[string]interface{}{
		"capture_id": capture.ID,
		"order_id":   req.OrderID,
		"status":     capture.Status,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capture)
}

// RefundPayPalCapture refunds a PayPal capture
func (h *PaymentHandlers) RefundPayPalCapture(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CaptureID string `json:"capture_id"`
		Amount    int64  `json:"amount,omitempty"`
		Currency  string `json:"currency,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	refund, err := h.paypalService.RefundCapture(req.CaptureID, req.Amount, req.Currency)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast refund event
	h.eventBroadcaster.Broadcast("payment:paypal:refunded", map[string]interface{}{
		"refund_id":  refund.ID,
		"capture_id": req.CaptureID,
		"amount":     refund.Amount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refund)
}

// GetBlockchainWalletBalance retrieves blockchain wallet balance
func (h *PaymentHandlers) GetBlockchainWalletBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	token := r.URL.Query().Get("token")

	if address == "" || token == "" {
		http.Error(w, "address and token are required", http.StatusBadRequest)
		return
	}

	wallet, err := h.blockchainService.GetWalletBalance(address, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}

// CreateBlockchainPayment creates a blockchain payment
func (h *PaymentHandlers) CreateBlockchainPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
		Amount      int64  `json:"amount"`
		Token       string `json:"token"`
		Memo        string `json:"memo,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	payment, err := h.blockchainService.CreatePayment(fintech.BlockchainPaymentRequest{
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Amount:      req.Amount,
		Token:       req.Token,
		Memo:        req.Memo,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast payment event
	h.eventBroadcaster.Broadcast("payment:blockchain:created", map[string]interface{}{
		"tx_hash": payment.TxHash,
		"from":    req.FromAddress,
		"to":      req.ToAddress,
		"amount":  req.Amount,
		"token":   req.Token,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

// GetBlockchainTransaction retrieves a blockchain transaction
func (h *PaymentHandlers) GetBlockchainTransaction(w http.ResponseWriter, r *http.Request) {
	txHash := r.URL.Query().Get("tx_hash")
	if txHash == "" {
		http.Error(w, "tx_hash is required", http.StatusBadRequest)
		return
	}

	tx, err := h.blockchainService.GetTransaction(txHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
}

// VerifyBlockchainTransaction verifies a blockchain transaction
func (h *PaymentHandlers) VerifyBlockchainTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxHash          string `json:"tx_hash"`
		ExpectedAmount  int64  `json:"expected_amount"`
		ExpectedRecipient string `json:"expected_recipient"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	verified, err := h.blockchainService.VerifyTransaction(req.TxHash, req.ExpectedAmount, req.ExpectedRecipient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast verification event
	h.eventBroadcaster.Broadcast("payment:blockchain:verified", map[string]interface{}{
		"tx_hash":  req.TxHash,
		"verified": verified,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"verified": verified,
		"tx_hash":  req.TxHash,
	})
}

// EstimateBlockchainGas estimates gas for a blockchain transaction
func (h *PaymentHandlers) EstimateBlockchainGas(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
		Amount      int64  `json:"amount"`
		Token       string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	gasUsed, gasPrice, err := h.blockchainService.EstimateGas(fintech.BlockchainPaymentRequest{
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Amount:      req.Amount,
		Token:       req.Token,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gas_used":  gasUsed,
		"gas_price": gasPrice,
		"total_fee": gasUsed * gasPrice,
	})
}

// GetBlockchainTransactionHistory retrieves transaction history for a wallet
func (h *PaymentHandlers) GetBlockchainTransactionHistory(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	if address == "" {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	transactions, err := h.blockchainService.GetTransactionHistory(address, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": transactions,
		"limit":        limit,
		"offset":       offset,
	})
}
