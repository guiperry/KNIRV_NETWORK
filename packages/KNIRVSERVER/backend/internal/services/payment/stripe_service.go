package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StripeService handles Stripe payment processing
type StripeService struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	currency       string
	apiVersion     string
	httpClient     *http.Client
}

// StripeSession represents a Stripe checkout session
type StripeSession struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	RentalID    string    `json:"rental_id"`
}

// ChargeStatus represents the status of a Stripe charge
type ChargeStatus struct {
	ID            string    `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Paid          bool      `json:"paid"`
	FailureCode   string    `json:"failure_code,omitempty"`
	FailureReason string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewStripeService creates a new Stripe service
func NewStripeService(secretKey, publishableKey, webhookSecret, currency, apiVersion string) *StripeService {
	return &StripeService{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		webhookSecret:  webhookSecret,
		currency:       currency,
		apiVersion:     apiVersion,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateCheckoutSession creates a new Stripe checkout session for a rental
func (ss *StripeService) CreateCheckoutSession(rentalID string, amount int64, currency, successURL, cancelURL string) (*StripeSession, error) {
	log.Printf("Creating Stripe checkout session for rental %s, amount: %d %s", rentalID, amount, currency)

	// In a real implementation, this would make an API call to Stripe
	// For now, we'll simulate the session creation

	sessionID := fmt.Sprintf("cs_test_%s_%d", rentalID, time.Now().Unix())

	session := &StripeSession{
		ID:        sessionID,
		URL:       fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID),
		Amount:    amount,
		Currency:  currency,
		Status:    "open",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RentalID:  rentalID,
	}

	log.Printf("Created Stripe session %s for rental %s", session.ID, rentalID)
	return session, nil
}

// GetChargeStatus retrieves the status of a Stripe charge
func (ss *StripeService) GetChargeStatus(chargeID string) (*ChargeStatus, error) {
	log.Printf("Getting charge status for %s", chargeID)

	// In a real implementation, this would query Stripe's API
	// For now, we'll simulate a successful charge

	status := &ChargeStatus{
		ID:        chargeID,
		Amount:    2999, // $29.99
		Currency:  "usd",
		Status:    "succeeded",
		Paid:      true,
		CreatedAt: time.Now(),
	}

	return status, nil
}

// RefundCharge refunds a Stripe charge
func (ss *StripeService) RefundCharge(chargeID string, reason string) error {
	log.Printf("Refunding charge %s, reason: %s", chargeID, reason)

	// In a real implementation, this would call Stripe's refund API
	// For now, we'll simulate a successful refund

	return nil
}

// ValidateWebhookSignature validates a Stripe webhook signature
// Uses HMAC-SHA256 to verify the signature matches the expected value
func (ss *StripeService) ValidateWebhookSignature(payload []byte, signature string) error {
	if ss.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// Parse the Stripe-Signature header
	// Format: t=timestamp,v1=signature,v0=signature
	parts := strings.Split(signature, ",")
	var timestamp string
	var signatures []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			signatures = append(signatures, strings.TrimPrefix(part, "v1="))
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return fmt.Errorf("invalid signature format")
	}

	// Check timestamp to prevent replay attacks (within 5 minutes)
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
			return fmt.Errorf("webhook timestamp too old")
		}
	}

	// Compute expected signature
	// Stripe uses: HMAC_SHA256(webhook_secret, timestamp + "." + payload)
	mac := hmac.New(sha256.New, []byte(ss.webhookSecret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expectedSignature)) {
			return nil
		}
	}

	return fmt.Errorf("signature mismatch")
}

// ProcessChargeSucceeded processes a successful charge webhook
func (ss *StripeService) ProcessChargeSucceeded(chargeID string) error {
	log.Printf("Processing successful charge: %s", chargeID)

	// In a real implementation, this would:
	// 1. Update the payment status in the database
	// 2. Trigger container provisioning for the rental
	// 3. Send confirmation emails
	// 4. Update rental status

	return nil
}

// ProcessChargeFailed processes a failed charge webhook
func (ss *StripeService) ProcessChargeFailed(chargeID string, reason string) error {
	log.Printf("Processing failed charge: %s, reason: %s", chargeID, reason)

	// In a real implementation, this would:
	// 1. Update the payment status to failed
	// 2. Notify the user of the failure
	// 3. Log the failure for analysis

	return nil
}

// HandleWebhook processes incoming Stripe webhooks with signature verification
func (ss *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	stripeSignature := r.Header.Get("Stripe-Signature")
	if stripeSignature == "" {
		http.Error(w, "Missing Stripe-Signature header", http.StatusBadRequest)
		return
	}

	// Validate webhook signature
	if err := ss.ValidateWebhookSignature(payload, stripeSignature); err != nil {
		http.Error(w, fmt.Sprintf("Webhook signature verification failed: %v", err), http.StatusBadRequest)
		return
	}

	// Parse the event
	var event struct {
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "Error parsing webhook event", http.StatusBadRequest)
		return
	}

	// Process the event based on type
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent map[string]interface{}
		if err := json.Unmarshal(event.Data, &paymentIntent); err == nil {
			ss.handlePaymentSucceeded(paymentIntent)
		}

	case "payment_intent.payment_failed":
		var paymentIntent map[string]interface{}
		if err := json.Unmarshal(event.Data, &paymentIntent); err == nil {
			ss.handlePaymentFailed(paymentIntent)
		}

	case "charge.succeeded":
		var charge map[string]interface{}
		if err := json.Unmarshal(event.Data, &charge); err == nil {
			ss.handleChargeSucceeded(charge)
		}

	case "charge.failed":
		var charge map[string]interface{}
		if err := json.Unmarshal(event.Data, &charge); err == nil {
			ss.handleChargeFailed(charge)
		}

	default:
		fmt.Printf("Unhandled webhook event type: %s\n", event.Type)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "event_id": event.ID})
}

// handlePaymentSucceeded processes a successful payment intent
func (ss *StripeService) handlePaymentSucceeded(paymentIntent map[string]interface{}) {
	fmt.Printf("Payment succeeded: %v\n", paymentIntent["id"])

	// Extract rental ID from metadata
	if metadata, ok := paymentIntent["metadata"].(map[string]interface{}); ok {
		if rentalID, ok := metadata["rental_id"].(string); ok {
			fmt.Printf("Updating rental %s to active\n", rentalID)
			// Update rental status in database
			// Provision DVE resources
			// Send confirmation email
		}
	}
}

// handlePaymentFailed processes a failed payment intent
func (ss *StripeService) handlePaymentFailed(paymentIntent map[string]interface{}) {
	fmt.Printf("Payment failed: %v\n", paymentIntent["id"])

	if metadata, ok := paymentIntent["metadata"].(map[string]interface{}); ok {
		if rentalID, ok := metadata["rental_id"].(string); ok {
			fmt.Printf("Marking rental %s as failed\n", rentalID)
			// Update rental status to failed
			// Notify user
		}
	}
}

// handleChargeSucceeded processes a successful charge
func (ss *StripeService) handleChargeSucceeded(charge map[string]interface{}) {
	fmt.Printf("Charge succeeded: %v\n", charge["id"])
}

// handleChargeFailed processes a failed charge
func (ss *StripeService) handleChargeFailed(charge map[string]interface{}) {
	fmt.Printf("Charge failed: %v\n", charge["id"])
}

// CalculateAmount calculates the amount in cents for a rental
func (ss *StripeService) CalculateAmount(durationHours int64, hourlyRate float64) int64 {
	amount := float64(durationHours) * hourlyRate
	// Convert to cents and round to nearest integer
	return int64(amount * 100)
}

// FormatAmount formats an amount in cents for display
func (ss *StripeService) FormatAmount(amount int64) string {
	dollars := amount / 100
	cents := amount % 100
	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

// ValidateAmount validates that an amount is reasonable
func (ss *StripeService) ValidateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if amount > 10000000 { // $100,000 max
		return fmt.Errorf("amount exceeds maximum allowed")
	}
	return nil
}

// GetSupportedCurrencies returns the list of supported currencies
func (ss *StripeService) GetSupportedCurrencies() []string {
	return []string{"usd", "eur", "gbp", "cad", "aud"}
}

// IsCurrencySupported checks if a currency is supported
func (ss *StripeService) IsCurrencySupported(currency string) bool {
	supported := ss.GetSupportedCurrencies()
	for _, c := range supported {
		if c == currency {
			return true
		}
	}
	return false
}

// GetPaymentMethods returns available payment methods
func (ss *StripeService) GetPaymentMethods() []string {
	return []string{"card", "alipay", "wechat_pay"}
}

// CreatePaymentIntent creates a payment intent (alternative to checkout sessions)
func (ss *StripeService) CreatePaymentIntent(amount int64, currency string, metadata map[string]string) (string, error) {
	log.Printf("Creating payment intent for %d %s", amount, currency)

	// In a real implementation, this would create a Stripe payment intent
	// For now, we'll return a mock ID

	intentID := fmt.Sprintf("pi_%s_%d", currency, time.Now().Unix())
	return intentID, nil
}

// ConfirmPaymentIntent confirms a payment intent
func (ss *StripeService) ConfirmPaymentIntent(intentID, paymentMethodID string) error {
	log.Printf("Confirming payment intent %s with method %s", intentID, paymentMethodID)

	// In a real implementation, this would confirm the payment intent with Stripe
	return nil
}

// GetPaymentIntent retrieves a payment intent
func (ss *StripeService) GetPaymentIntent(intentID string) (map[string]interface{}, error) {
	log.Printf("Getting payment intent %s", intentID)

	// In a real implementation, this would retrieve the payment intent from Stripe
	// For now, return mock data

	intent := map[string]interface{}{
		"id":       intentID,
		"amount":   2999,
		"currency": "usd",
		"status":   "succeeded",
	}

	return intent, nil
}
