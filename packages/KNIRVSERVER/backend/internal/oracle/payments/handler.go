// Package payments provides oracle-gated payment processing handlers.
// Access to payment routes is restricted to root nodes that have loaded
// the encrypted root key (packages/KNIRVSERVER/bin/root.key).
package payments

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/services/plugins/fintech"

	"github.com/gorilla/mux"
)

// OraclePaymentHandler exposes payment processing endpoints under /oracle/payments/.
// It is only mounted when the oracle key is present.
type OraclePaymentHandler struct {
	stripeService *fintech.StripeService
	paypalService *fintech.PayPalService
	// oracleKeyPresent is a function that returns true when the root key is loaded.
	oracleKeyPresent func() bool
}

// NewOraclePaymentHandler creates a new handler.
// stripeService and paypalService may be nil; a 503 is returned for missing services.
func NewOraclePaymentHandler(
	stripe *fintech.StripeService,
	paypal *fintech.PayPalService,
	oracleKeyPresent func() bool,
) *OraclePaymentHandler {
	return &OraclePaymentHandler{
		stripeService:    stripe,
		paypalService:    paypal,
		oracleKeyPresent: oracleKeyPresent,
	}
}

// RegisterRoutes mounts the oracle payment routes. Call this only when the oracle
// key is confirmed present.
func (h *OraclePaymentHandler) RegisterRoutes(r *mux.Router) {
	sub := r.PathPrefix("/oracle/payments").Subrouter()
	sub.Use(h.requireOracleKey)
	sub.HandleFunc("/stripe/checkout", h.HandleStripeCheckout).Methods("POST")
	sub.HandleFunc("/paypal/pay", h.HandlePayPalPayment).Methods("POST")
}

// requireOracleKey middleware blocks access when the oracle key is absent.
func (h *OraclePaymentHandler) requireOracleKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.oracleKeyPresent != nil && !h.oracleKeyPresent() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "oracle not available on this node",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// HandleStripeCheckout creates a Stripe checkout session.
func (h *OraclePaymentHandler) HandleStripeCheckout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.stripeService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "stripe service not configured"})
		return
	}

	var req fintech.CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	session, err := h.stripeService.CreateCheckoutSession(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// HandlePayPalPayment creates a PayPal order.
func (h *OraclePaymentHandler) HandlePayPalPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.paypalService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "paypal service not configured"})
		return
	}

	var req fintech.PayPalOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	order, err := h.paypalService.CreateOrder(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}
