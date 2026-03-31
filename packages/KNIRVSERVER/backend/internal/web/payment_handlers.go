package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/services/payment"
	"backend_server/internal/services/websocket"
)

type PaymentHandlers struct {
	stripeService    *payment.StripeService
	paypalService    *payment.PayPalService
	eventBroadcaster *websocket.EventBroadcaster
}

func NewPaymentHandlers(
	stripeService *payment.StripeService,
	paypalService *payment.PayPalService,
	eventBroadcaster *websocket.EventBroadcaster,
) *PaymentHandlers {
	return &PaymentHandlers{
		stripeService:    stripeService,
		paypalService:    paypalService,
		eventBroadcaster: eventBroadcaster,
	}
}

func (h *PaymentHandlers) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if h.stripeService == nil {
		http.Error(w, "Stripe service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RentalID   string `json:"rental_id"`
		Amount     int64  `json:"amount"`
		Currency   string `json:"currency"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.stripeService.CreateCheckoutSession(req.RentalID, req.Amount, req.Currency, req.SuccessURL, req.CancelURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (h *PaymentHandlers) GetStripeChargeStatus(w http.ResponseWriter, r *http.Request) {
	if h.stripeService == nil {
		http.Error(w, "Stripe service not configured", http.StatusServiceUnavailable)
		return
	}

	chargeID := r.URL.Query().Get("charge_id")
	if chargeID == "" {
		http.Error(w, "charge_id is required", http.StatusBadRequest)
		return
	}

	status, err := h.stripeService.GetChargeStatus(chargeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *PaymentHandlers) RefundStripeCharge(w http.ResponseWriter, r *http.Request) {
	if h.stripeService == nil {
		http.Error(w, "Stripe service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ChargeID string `json:"charge_id"`
		Reason   string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.stripeService.RefundCharge(req.ChargeID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
}

func (h *PaymentHandlers) CreatePayPalOrder(w http.ResponseWriter, r *http.Request) {
	if h.paypalService == nil {
		http.Error(w, "PayPal service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RentalID  string  `json:"rental_id"`
		Amount    float64 `json:"amount"`
		Currency  string  `json:"currency"`
		ReturnURL string  `json:"return_url"`
		CancelURL string  `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.paypalService.CreateOrder(req.RentalID, req.Amount, req.Currency, req.ReturnURL, req.CancelURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *PaymentHandlers) CapturePayPalOrder(w http.ResponseWriter, r *http.Request) {
	if h.paypalService == nil {
		http.Error(w, "PayPal service not configured", http.StatusServiceUnavailable)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capture)
}

func (h *PaymentHandlers) GetPayPalOrderStatus(w http.ResponseWriter, r *http.Request) {
	if h.paypalService == nil {
		http.Error(w, "PayPal service not configured", http.StatusServiceUnavailable)
		return
	}

	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	status, err := h.paypalService.GetOrderStatus(orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *PaymentHandlers) RefundPayPalCapture(w http.ResponseWriter, r *http.Request) {
	if h.paypalService == nil {
		http.Error(w, "PayPal service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		CaptureID string `json:"capture_id"`
		Reason    string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.paypalService.RefundCapture(req.CaptureID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
}
