package fintech

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PayPalService handles real PayPal payment processing
type PayPalService struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// PayPalOrder represents a PayPal order
type PayPalOrder struct {
	ID     string           `json:"id"`
	Status string           `json:"status"`
	Links  []PayPalLink     `json:"links"`
	Payer  *PayPalPayer     `json:"payer,omitempty"`
}

// PayPalLink represents a PayPal HATEOAS link
type PayPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// PayPalPayer represents a PayPal payer
type PayPalPayer struct {
	Email         string `json:"email_address,omitempty"`
	PayerID       string `json:"payer_id,omitempty"`
	Name          *PayPalName `json:"name,omitempty"`
}

// PayPalName represents a PayPal name
type PayPalName struct {
	GivenName string `json:"given_name,omitempty"`
	Surname   string `json:"surname,omitempty"`
}

// PayPalCapture represents a PayPal payment capture
type PayPalCapture struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount *PayPalAmount `json:"amount,omitempty"`
}

// PayPalAmount represents a PayPal amount
type PayPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// PayPalAccessToken represents a PayPal access token response
type PayPalAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewPayPalService creates a new real PayPal service
func NewPayPalService(clientID, clientSecret string, sandbox bool) *PayPalService {
	baseURL := "https://api-m.paypal.com"
	if sandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}

	return &PayPalService{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getAccessToken obtains or refreshes the PayPal access token
func (p *PayPalService) getAccessToken() error {
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return nil
	}

	url := fmt.Sprintf("%s/v1/oauth2/token", p.baseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", p.clientID, p.clientSecret)))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("paypal API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("paypal auth error (status %d): %s", resp.StatusCode, string(body))
	}

	var token PayPalAccessToken
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	p.accessToken = token.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn-60) * time.Second)

	return nil
}

// CreateOrder creates a real PayPal order
func (p *PayPalService) CreateOrder(req PayPalOrderRequest) (*PayPalOrder, error) {
	if err := p.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s/v2/checkout/orders", p.baseURL)

	orderReq := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]interface{}{
					"currency_code": req.Currency,
					"value":         fmt.Sprintf("%.2f", float64(req.Amount)/100),
				},
				"description": req.Description,
			},
		},
		"application_context": map[string]interface{}{
			"return_url": req.ReturnURL,
			"cancel_url": req.CancelURL,
			"brand_name": "KNIRV Network",
		},
	}

	if req.CustomerEmail != "" {
		orderReq["payer"] = map[string]interface{}{
			"email_address": req.CustomerEmail,
		}
	}

	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Prefer", "return=representation")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paypal order error (status %d): %s", resp.StatusCode, string(body))
	}

	var order PayPalOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return &order, nil
}

// GetOrder retrieves a PayPal order by ID
func (p *PayPalService) GetOrder(orderID string) (*PayPalOrder, error) {
	if err := p.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s/v2/checkout/orders/%s", p.baseURL, orderID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paypal order error (status %d): %s", resp.StatusCode, string(body))
	}

	var order PayPalOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return &order, nil
}

// CaptureOrder captures a PayPal order payment
func (p *PayPalService) CaptureOrder(orderID string) (*PayPalCapture, error) {
	if err := p.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s/v2/checkout/orders/%s/capture", p.baseURL, orderID)

	httpReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paypal capture error (status %d): %s", resp.StatusCode, string(body))
	}

	var capture PayPalCapture
	if err := json.Unmarshal(body, &capture); err != nil {
		return nil, fmt.Errorf("failed to parse capture response: %w", err)
	}

	return &capture, nil
}

// RefundCapture refunds a PayPal capture
func (p *PayPalService) RefundCapture(captureID string, amount int64, currency string) (*PayPalCapture, error) {
	if err := p.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s/v2/payments/captures/%s/refund", p.baseURL, captureID)

	var refundReq map[string]interface{}
	if amount > 0 {
		refundReq = map[string]interface{}{
			"amount": map[string]interface{}{
				"currency_code": currency,
				"value":         fmt.Sprintf("%.2f", float64(amount)/100),
			},
		}
	}

	var reqBody *bytes.Buffer
	if refundReq != nil {
		jsonData, err := json.Marshal(refundReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paypal refund error (status %d): %s", resp.StatusCode, string(body))
	}

	var capture PayPalCapture
	if err := json.Unmarshal(body, &capture); err != nil {
		return nil, fmt.Errorf("failed to parse refund response: %w", err)
	}

	return &capture, nil
}

// PayPalOrderRequest represents a request to create a PayPal order
type PayPalOrderRequest struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description"`
	ReturnURL     string `json:"return_url"`
	CancelURL     string `json:"cancel_url"`
	CustomerEmail string `json:"customer_email,omitempty"`
}
