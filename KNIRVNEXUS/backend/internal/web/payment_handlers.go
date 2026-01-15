package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend_server/internal/services/payment"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// PaymentHandlers handles payment API requests
type PaymentHandlers struct {
	stripeService *payment.StripeService
	paypalService *payment.PayPalService
}

// NewPaymentHandlers creates new payment handlers
func NewPaymentHandlers(stripeService *payment.StripeService, paypalService *payment.PayPalService) *PaymentHandlers {
	return &PaymentHandlers{
		stripeService: stripeService,
		paypalService: paypalService,
	}
}

// PaymentResponse represents a standard API response for payment operations
type PaymentResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// CreateStripeSession handles POST /api/payments/stripe/create-session
func (ph *PaymentHandlers) CreateStripeSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentalID     string `json:"rental_id"`
		Amount       int64  `json:"amount"`
		Currency     string `json:"currency"`
		SuccessURL   string `json:"success_url"`
		CancelURL    string `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required fields
	if req.RentalID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "rental_id is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set defaults
	if req.Currency == "" {
		req.Currency = "usd"
	}
	if req.SuccessURL == "" {
		req.SuccessURL = "https://app.example.com/rentals/success"
	}
	if req.CancelURL == "" {
		req.CancelURL = "https://app.example.com/rentals/cancelled"
	}

	// Validate currency
	if !ph.stripeService.IsCurrencySupported(req.Currency) {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Unsupported currency: %s", req.Currency),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate amount
	if err := ph.stripeService.ValidateAmount(req.Amount); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create Stripe session
	session, err := ph.stripeService.CreateCheckoutSession(req.RentalID, req.Amount, req.Currency, req.SuccessURL, req.CancelURL)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create Stripe session: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Data:      session,
		Message:   "Stripe checkout session created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStripeSession handles GET /api/payments/stripe/session/{session_id}
func (ph *PaymentHandlers) GetStripeSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session_id"]

	if sessionID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// In a real implementation, this would retrieve the session from Stripe
	// For now, we'll return mock data
	session := map[string]interface{}{
		"id":        sessionID,
		"url":       fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID),
		"status":    "open",
		"amount":    2999,
		"currency":  "usd",
		"expires_at": time.Now().Add(24 * time.Hour),
	}

	response := PaymentResponse{
		Success:   true,
		Data:      session,
		Message:   "Stripe session retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleStripeWebhook handles POST /api/payments/stripe/webhook
func (ph *PaymentHandlers) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	ph.stripeService.HandleWebhook(w, r)
}

// GetStripeChargeStatus handles GET /api/payments/stripe/charge/{charge_id}/status
func (ph *PaymentHandlers) GetStripeChargeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chargeID := vars["charge_id"]

	if chargeID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Charge ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	status, err := ph.stripeService.GetChargeStatus(chargeID)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to get charge status: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Data:      status,
		Message:   "Charge status retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RefundStripeCharge handles POST /api/payments/stripe/refund
func (ph *PaymentHandlers) RefundStripeCharge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChargeID string `json:"charge_id"`
		Reason   string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.ChargeID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "charge_id is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := ph.stripeService.RefundCharge(req.ChargeID, req.Reason)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to refund charge: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Message:   "Charge refunded successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreatePayPalOrder handles POST /api/payments/paypal/create-order
func (ph *PaymentHandlers) CreatePayPalOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentalID  string  `json:"rental_id"`
		Amount    float64 `json:"amount"`
		Currency  string  `json:"currency"`
		ReturnURL string  `json:"return_url"`
		CancelURL string  `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required fields
	if req.RentalID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "rental_id is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set defaults
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.ReturnURL == "" {
		req.ReturnURL = "https://app.example.com/rentals/success"
	}
	if req.CancelURL == "" {
		req.CancelURL = "https://app.example.com/rentals/cancelled"
	}

	// Validate currency
	if !ph.paypalService.IsCurrencySupported(req.Currency) {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Unsupported currency: %s", req.Currency),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate amount
	if err := ph.paypalService.ValidateAmount(req.Amount); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create PayPal order
	order, err := ph.paypalService.CreateOrder(req.RentalID, req.Amount, req.Currency, req.ReturnURL, req.CancelURL)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to create PayPal order: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Data:      order,
		Message:   "PayPal order created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPayPalOrder handles GET /api/payments/paypal/order/{order_id}
func (ph *PaymentHandlers) GetPayPalOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["order_id"]

	if orderID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Order ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// In a real implementation, this would retrieve the order from PayPal
	// For now, we'll return mock data
	order := map[string]interface{}{
		"id":           orderID,
		"status":       "CREATED",
		"checkout_url": fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", orderID),
		"amount":       29.99,
		"currency":     "USD",
		"expires_at":   time.Now().Add(24 * time.Hour),
	}

	response := PaymentResponse{
		Success:   true,
		Data:      order,
		Message:   "PayPal order retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CapturePayPalOrder handles POST /api/payments/paypal/capture
func (ph *PaymentHandlers) CapturePayPalOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.OrderID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "order_id is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	capture, err := ph.paypalService.CaptureOrder(req.OrderID)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to capture PayPal order: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Data:      capture,
		Message:   "PayPal order captured successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandlePayPalWebhook handles POST /api/payments/paypal/webhook
func (ph *PaymentHandlers) HandlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	ph.paypalService.HandleWebhook(w, r)
}

// RefundPayPalCapture handles POST /api/payments/paypal/refund
func (ph *PaymentHandlers) RefundPayPalCapture(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CaptureID string `json:"capture_id"`
		Reason    string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.CaptureID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "capture_id is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := ph.paypalService.RefundCapture(req.CaptureID, req.Reason)
	if err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to refund PayPal capture: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success:   true,
		Message:   "PayPal capture refunded successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPaymentHistory handles GET /api/payments/history
func (ph *PaymentHandlers) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token
	userID := middleware.GetUserIDFromRequest(r)
	if userID == "" {
		// Fallback to query parameter for development/testing
		userID = r.URL.Query().Get("user_id")
	}
	if userID == "" {
		userID = "test-user-default"
	}

	// In a real implementation, this would query the database for payment history
	// For now, we'll return mock data
	history := []map[string]interface{}{
		{
			"id":         "pay_123",
			"provider":   "stripe",
			"amount":     2999,
			"currency":   "usd",
			"status":     "completed",
			"created_at": time.Now().Add(-24 * time.Hour),
		},
		{
			"id":         "pay_456",
			"provider":   "paypal",
			"amount":     49.99,
			"currency":   "usd",
			"status":     "completed",
			"created_at": time.Now().Add(-48 * time.Hour),
		},
	}

	response := PaymentResponse{
		Success:   true,
		Data:      history,
		Message:   "Payment history retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPaymentDetails handles GET /api/payments/{payment_id}
func (ph *PaymentHandlers) GetPaymentDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["payment_id"]

	if paymentID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Payment ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// In a real implementation, this would query the database for payment details
	// For now, we'll return mock data
	payment := map[string]interface{}{
		"id":         paymentID,
		"provider":   "stripe",
		"amount":     2999,
		"currency":   "usd",
		"status":     "completed",
		"rental_id":  "rental-123",
		"created_at": time.Now().Add(-24 * time.Hour),
	}

	response := PaymentResponse{
		Success:   true,
		Data:      payment,
		Message:   "Payment details retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPaymentReceipt handles GET /api/payments/{payment_id}/receipt
func (ph *PaymentHandlers) GetPaymentReceipt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["payment_id"]

	if paymentID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Payment ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// In a real implementation, this would generate or retrieve a receipt
	// For now, we'll return mock receipt data
	receipt := map[string]interface{}{
		"payment_id": paymentID,
		"receipt_url": fmt.Sprintf("https://receipts.example.com/%s", paymentID),
		"amount":     29.99,
		"currency":   "usd",
		"date":       time.Now().Add(-24 * time.Hour),
	}

	response := PaymentResponse{
		Success:   true,
		Data:      receipt,
		Message:   "Payment receipt retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RequestPaymentRefund handles POST /api/payments/{payment_id}/refund-request
func (ph *PaymentHandlers) RequestPaymentRefund(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["payment_id"]

	if paymentID == "" {
		response := PaymentResponse{
			Success:   false,
			Error:     "Payment ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := PaymentResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// In a real implementation, this would:
	// 1. Validate the refund request
	// 2. Check refund eligibility
	// 3. Process the refund through the payment provider
	// 4. Update the database

	log.Printf("Processing refund request for payment %s, reason: %s", paymentID, req.Reason)

	response := PaymentResponse{
		Success:   true,
		Message:   "Refund request submitted successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the payment routes with the router
func (ph *PaymentHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for payment endpoints
	paymentRouter := r.PathPrefix("/api/payments").Subrouter()

	// Stripe routes
	paymentRouter.HandleFunc("/stripe/create-session", ph.CreateStripeSession).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/session/{session_id}", ph.GetStripeSession).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/webhook", ph.HandleStripeWebhook).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/charge/{charge_id}/status", ph.GetStripeChargeStatus).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/refund", ph.RefundStripeCharge).Methods("POST", "OPTIONS")

	// PayPal routes
	paymentRouter.HandleFunc("/paypal/create-order", ph.CreatePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/order/{order_id}", ph.GetPayPalOrder).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/capture", ph.CapturePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/webhook", ph.HandlePayPalWebhook).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/refund", ph.RefundPayPalCapture).Methods("POST", "OPTIONS")

	// General payment routes (require authentication)
	if authMiddleware != nil {
		protectedPaymentRouter := paymentRouter.PathPrefix("").Subrouter()
		protectedPaymentRouter.Use(authMiddleware.RequireAuth)
		protectedPaymentRouter.HandleFunc("/history", ph.GetPaymentHistory).Methods("GET", "OPTIONS")
		protectedPaymentRouter.HandleFunc("/{payment_id}", ph.GetPaymentDetails).Methods("GET", "OPTIONS")
		protectedPaymentRouter.HandleFunc("/{payment_id}/receipt", ph.GetPaymentReceipt).Methods("GET", "OPTIONS")
		protectedPaymentRouter.HandleFunc("/{payment_id}/refund-request", ph.RequestPaymentRefund).Methods("POST", "OPTIONS")
	}

	// Handle OPTIONS requests for CORS
	paymentRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
	})
}
