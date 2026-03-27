package fintech

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StripeService handles real Stripe payment processing
type StripeService struct {
	apiKey     string
	httpClient *http.Client
}

// StripeCheckoutSession represents a Stripe checkout session
type StripeCheckoutSession struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	PaymentStatus string                `json:"payment_status"`
	AmountTotal  int64                  `json:"amount_total"`
	Currency     string                 `json:"currency"`
	Metadata     map[string]string      `json:"metadata"`
	Created      int64                  `json:"created"`
	ExpiresAt    int64                  `json:"expires_at"`
}

// StripePaymentIntent represents a Stripe payment intent
type StripePaymentIntent struct {
	ID            string `json:"id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	ClientSecret  string `json:"client_secret"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

// StripeRefund represents a Stripe refund
type StripeRefund struct {
	ID            string `json:"id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	PaymentIntent string `json:"payment_intent"`
}

// NewStripeService creates a new real Stripe service
func NewStripeService(apiKey string) *StripeService {
	return &StripeService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateCheckoutSession creates a real Stripe checkout session
func (s *StripeService) CreateCheckoutSession(req CheckoutSessionRequest) (*StripeCheckoutSession, error) {
	url := "https://api.stripe.com/v1/checkout/sessions"

	params := fmt.Sprintf(
		"payment_method_types[]=card&mode=payment&success_url=%s&cancel_url=%s&line_items[0][price_data][currency]=%s&line_items[0][price_data][product_data][name]=%s&line_items[0][price_data][unit_amount]=%d&line_items[0][quantity]=1",
		req.SuccessURL,
		req.CancelURL,
		req.Currency,
		req.ProductName,
		req.Amount,
	)

	if req.CustomerEmail != "" {
		params += fmt.Sprintf("&customer_email=%s", req.CustomerEmail)
	}

	for k, v := range req.Metadata {
		params += fmt.Sprintf("&metadata[%s]=%s", k, v)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBufferString(params))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(s.apiKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("stripe API error (status %d): %s", resp.StatusCode, string(body))
	}

	var session StripeCheckoutSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &session, nil
}

// CreatePaymentIntent creates a real Stripe payment intent
func (s *StripeService) CreatePaymentIntent(req PaymentIntentRequest) (*StripePaymentIntent, error) {
	url := "https://api.stripe.com/v1/payment_intents"

	params := fmt.Sprintf(
		"amount=%d&currency=%s&payment_method_types[]=card",
		req.Amount,
		req.Currency,
	)

	if req.CustomerID != "" {
		params += fmt.Sprintf("&customer=%s", req.CustomerID)
	}

	if req.Description != "" {
		params += fmt.Sprintf("&description=%s", req.Description)
	}

	for k, v := range req.Metadata {
		params += fmt.Sprintf("&metadata[%s]=%s", k, v)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBufferString(params))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(s.apiKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("stripe API error (status %d): %s", resp.StatusCode, string(body))
	}

	var intent StripePaymentIntent
	if err := json.Unmarshal(body, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

// GetPaymentIntent retrieves a payment intent from Stripe
func (s *StripeService) GetPaymentIntent(paymentIntentID string) (*StripePaymentIntent, error) {
	url := fmt.Sprintf("https://api.stripe.com/v1/payment_intents/%s", paymentIntentID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(s.apiKey, "")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe API error (status %d): %s", resp.StatusCode, string(body))
	}

	var intent StripePaymentIntent
	if err := json.Unmarshal(body, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

// CreateRefund creates a refund for a payment intent
func (s *StripeService) CreateRefund(paymentIntentID string, amount int64) (*StripeRefund, error) {
	url := "https://api.stripe.com/v1/refunds"

	params := fmt.Sprintf("payment_intent=%s", paymentIntentID)
	if amount > 0 {
		params += fmt.Sprintf("&amount=%d", amount)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBufferString(params))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(s.apiKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("stripe API error (status %d): %s", resp.StatusCode, string(body))
	}

	var refund StripeRefund
	if err := json.Unmarshal(body, &refund); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &refund, nil
}

// CheckoutSessionRequest represents a request to create a checkout session
type CheckoutSessionRequest struct {
	Amount        int64             `json:"amount"`
	Currency      string            `json:"currency"`
	ProductName   string            `json:"product_name"`
	SuccessURL    string            `json:"success_url"`
	CancelURL     string            `json:"cancel_url"`
	CustomerEmail string            `json:"customer_email,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// PaymentIntentRequest represents a request to create a payment intent
type PaymentIntentRequest struct {
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	CustomerID  string            `json:"customer_id,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
