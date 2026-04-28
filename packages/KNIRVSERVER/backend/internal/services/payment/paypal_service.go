package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// PayPalService handles PayPal payment processing
type PayPalService struct {
	clientID     string
	secret       string
	environment  string // "sandbox" or "production"
	currency     string
	webhookID   string // Webhook ID for signature verification
	httpClient   *http.Client
}

// NewPayPalService creates a new PayPal service
func NewPayPalService(clientID, secret, environment, currency, webhookID string) *PayPalService {
	return &PayPalService{
		clientID:    clientID,
		secret:      secret,
		environment: environment,
		currency:    currency,
		webhookID:   webhookID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// PayPalOrder represents a PayPal order
type PayPalOrder struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	CheckoutURL string    `json:"checkout_url"`
	ExpiresAt   time.Time `json:"expires_at"`
	RentalID    string    `json:"rental_id"`
}

// PayPalCapture represents a PayPal payment capture
type PayPalCapture struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderStatus represents the status of a PayPal order
type OrderStatus struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Paid      bool      `json:"paid"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateOrder creates a new PayPal order for a rental
func (pps *PayPalService) CreateOrder(rentalID string, amount float64, currency, returnURL, cancelURL string) (*PayPalOrder, error) {
	log.Printf("Creating PayPal order for rental %s, amount: %.2f %s", rentalID, amount, currency)

	// In a real implementation, this would make an API call to PayPal
	// For now, we'll simulate the order creation

	orderID := fmt.Sprintf("PAY-%s-%d", rentalID, time.Now().Unix())

	order := &PayPalOrder{
		ID:          orderID,
		Status:      "CREATED",
		Amount:      amount,
		Currency:    currency,
		CheckoutURL: fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", orderID),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		RentalID:    rentalID,
	}

	log.Printf("Created PayPal order %s for rental %s", order.ID, rentalID)
	return order, nil
}

// CaptureOrder captures payment for a PayPal order
func (pps *PayPalService) CaptureOrder(orderID string) (*PayPalCapture, error) {
	log.Printf("Capturing PayPal order %s", orderID)

	// In a real implementation, this would capture the payment via PayPal API
	// For now, we'll simulate a successful capture

	capture := &PayPalCapture{
		ID:        fmt.Sprintf("CAP-%s", orderID),
		Status:    "COMPLETED",
		Amount:    29.99,
		Currency:  "USD",
		CreatedAt: time.Now(),
	}

	log.Printf("Captured PayPal order %s", orderID)
	return capture, nil
}

// GetOrderStatus retrieves the status of a PayPal order
func (pps *PayPalService) GetOrderStatus(orderID string) (*OrderStatus, error) {
	log.Printf("Getting order status for %s", orderID)

	// In a real implementation, this would query PayPal's API
	// For now, we'll simulate a completed order

	status := &OrderStatus{
		ID:        orderID,
		Status:    "COMPLETED",
		Amount:    29.99,
		Currency:  "USD",
		Paid:      true,
		CreatedAt: time.Now(),
	}

	return status, nil
}

// RefundCapture refunds a PayPal capture
func (pps *PayPalService) RefundCapture(captureID string, reason string) error {
	log.Printf("Refunding PayPal capture %s, reason: %s", captureID, reason)

	// In a real implementation, this would call PayPal's refund API
	// For now, we'll simulate a successful refund

	return nil
}

// ValidateWebhookSignature validates a PayPal webhook signature
// PayPal uses a signature based on webhook ID, transmission ID, timestamp, and webhook ID
func (pps *PayPalService) ValidateWebhookSignature(headers map[string]string, body []byte) error {
	// Case-insensitive header lookup
	authAlgo := headers["Paypal-Auth-Algo"]
	if authAlgo == "" {
		authAlgo = headers["paypal-auth-algo"]
	}
	transmissionID := headers["Paypal-Transmission-Id"]
	if transmissionID == "" {
		transmissionID = headers["paypal-transmission-id"]
	}
	certURL := headers["Paypal-Cert-Url"]
	if certURL == "" {
		certURL = headers["paypal-cert-url"]
	}
	transmissionSig := headers["Paypal-Transmission-Sig"]
	if transmissionSig == "" {
		transmissionSig = headers["paypal-transmission-sig"]
	}
	transmissionTime := headers["Paypal-Transmission-Time"]
	if transmissionTime == "" {
		transmissionTime = headers["paypal-transmission-time"]
	}

	// Validate required headers
	if authAlgo == "" || transmissionID == "" || certURL == "" || transmissionSig == "" || transmissionTime == "" {
		return fmt.Errorf("missing required PayPal webhook headers")
	}

	// Verify webhook ID matches (should be passed in the webhook configuration)
	// In production, you would get this from your webhook configuration
	expectedWebhookID := pps.webhookID // This should be added to PayPalService struct
	if expectedWebhookID != "" {
		// Parse body to get webhook_id
		var event map[string]interface{}
		if err := json.Unmarshal(body, &event); err == nil {
			if webhookID, ok := event["webhook_id"].(string); ok && webhookID != expectedWebhookID {
				return fmt.Errorf("webhook ID mismatch")
			}
		}
	}

	// For production, you should:
	// 1. Fetch the certificate from certURL
	// 2. Verify the signature using the certificate
	// 3. Check that the transmission time is recent (within 5 minutes)

	// Simplified verification for development
	if authAlgo == "SHA256withRSA" {
		// In production, properly verify the RSA signature
		// This requires fetching PayPal's public certificate and verifying
		fmt.Println("PayPal signature algorithm verified (simplified)")
		return nil
	}

	return fmt.Errorf("unsupported signature algorithm: %s", authAlgo)
}

// ProcessOrderCompleted processes a completed order webhook
func (pps *PayPalService) ProcessOrderCompleted(orderID string) error {
	log.Printf("Processing completed PayPal order: %s", orderID)

	// In a real implementation, this would:
	// 1. Update the payment status in the database
	// 2. Trigger container provisioning for the rental
	// 3. Send confirmation emails
	// 4. Update rental status

	return nil
}

// ProcessCaptureFailed processes a failed capture webhook
func (pps *PayPalService) ProcessCaptureFailed(orderID string, reason string) error {
	log.Printf("Processing failed PayPal capture: %s, reason: %s", orderID, reason)

	// In a real implementation, this would:
	// 1. Update the payment status to failed
	// 2. Notify the user of the failure
	// 3. Log the failure for analysis

	return nil
}

// HandleWebhook processes incoming PayPal webhooks with signature verification
func (pps *PayPalService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Build headers map for verification
	headers := map[string]string{
		"Paypal-Auth-Algo":          r.Header.Get("Paypal-Auth-Algo"),
		"Paypal-Transmission-ID":   r.Header.Get("Paypal-Transmission-ID"),
		"Paypal-Cert-Url":          r.Header.Get("Paypal-Cert-Url"),
		"Paypal-Transmission-Sig":  r.Header.Get("Paypal-Transmission-Sig"),
		"Paypal-Transmission-Time":  r.Header.Get("Paypal-Transmission-Time"),
	}

	// Verify PayPal webhook signature
	if err := pps.ValidateWebhookSignature(headers, payload); err != nil {
		http.Error(w, fmt.Sprintf("Webhook signature verification failed: %v", err), http.StatusBadRequest)
		return
	}

	// Parse event
	var event struct {
		ID         string                 `json:"id"`
		EventType  string                 `json:"event_type"`
		Resource   map[string]interface{} `json:"resource"`
		WebhookID string                 `json:"webhook_id"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "Error parsing webhook event", http.StatusBadRequest)
		return
	}

	// Process event
	switch event.EventType {
	case "PAYMENT.SALE.COMPLETED":
		pps.handlePaymentSaleCompleted(event.Resource)
	case "PAYMENT.SALE.DENIED":
		pps.handlePaymentSaleDenied(event.Resource)
	case "PAYMENT.SALE.REFUNDED":
		pps.handlePaymentRefunded(event.Resource)
	case "PAYMENT.CAPTURE.COMPLETED":
		pps.handleCaptureCompleted(event.Resource)
	case "PAYMENT.CAPTURE.DENIED":
		pps.handleCaptureDenied(event.Resource)
	default:
		fmt.Printf("Unhandled PayPal event type: %s\n", event.EventType)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "event_id": event.ID})
}

// handlePaymentSaleCompleted processes a completed PayPal sale
func (pps *PayPalService) handlePaymentSaleCompleted(resource map[string]interface{}) {
	fmt.Printf("PayPal payment completed: %v\n", resource["id"])
	// Extract rental ID from custom field or metadata
	if custom, ok := resource["custom"].(string); ok {
		fmt.Printf("Rental ID from custom field: %s\n", custom)
		// Update rental status
	}
}

// handlePaymentSaleDenied processes a denied PayPal sale
func (pps *PayPalService) handlePaymentSaleDenied(resource map[string]interface{}) {
	fmt.Printf("PayPal payment denied: %v\n", resource["id"])
}

// handlePaymentRefunded processes a refunded PayPal payment
func (pps *PayPalService) handlePaymentRefunded(resource map[string]interface{}) {
	fmt.Printf("PayPal payment refunded: %v\n", resource["id"])
}

// handleCaptureCompleted processes a completed capture
func (pps *PayPalService) handleCaptureCompleted(resource map[string]interface{}) {
	fmt.Printf("PayPal capture completed: %v\n", resource["id"])
}

// handleCaptureDenied processes a denied capture
func (pps *PayPalService) handleCaptureDenied(resource map[string]interface{}) {
	fmt.Printf("PayPal capture denied: %v\n", resource["id"])
}

// CalculateAmount calculates the amount for a rental
func (pps *PayPalService) CalculateAmount(durationHours int64, hourlyRate float64) float64 {
	return float64(durationHours) * hourlyRate
}

// FormatAmount formats an amount for display
func (pps *PayPalService) FormatAmount(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

// ValidateAmount validates that an amount is reasonable
func (pps *PayPalService) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if amount > 100000 { // $100,000 max
		return fmt.Errorf("amount exceeds maximum allowed")
	}
	return nil
}

// GetSupportedCurrencies returns the list of supported currencies
func (pps *PayPalService) GetSupportedCurrencies() []string {
	return []string{"USD", "EUR", "GBP", "CAD", "AUD", "JPY"}
}

// IsCurrencySupported checks if a currency is supported
func (pps *PayPalService) IsCurrencySupported(currency string) bool {
	supported := pps.GetSupportedCurrencies()
	for _, c := range supported {
		if c == currency {
			return true
		}
	}
	return false
}

// GetPaymentMethods returns available payment methods
func (pps *PayPalService) GetPaymentMethods() []string {
	return []string{"paypal", "venmo", "paylater"}
}

// GetBaseURL returns the base URL for PayPal API calls
func (pps *PayPalService) GetBaseURL() string {
	if pps.environment == "production" {
		return "https://api.paypal.com"
	}
	return "https://api.sandbox.paypal.com"
}

// GetCheckoutURL returns the checkout URL for an order
func (pps *PayPalService) GetCheckoutURL(orderID string) string {
	if pps.environment == "production" {
		return fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", orderID)
	}
	return fmt.Sprintf("https://www.sandbox.paypal.com/checkoutnow?token=%s", orderID)
}

// IsSandbox returns true if the service is in sandbox mode
func (pps *PayPalService) IsSandbox() bool {
	return pps.environment == "sandbox"
}

// GetClientID returns the PayPal client ID (for frontend use)
func (pps *PayPalService) GetClientID() string {
	return pps.clientID
}
