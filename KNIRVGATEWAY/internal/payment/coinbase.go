package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CoinbaseClient handles Coinbase Commerce payment processing
type CoinbaseClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewCoinbaseClient creates a new Coinbase client
func NewCoinbaseClient(apiKey string) *CoinbaseClient {
	return &CoinbaseClient{
		apiKey:     apiKey,
		baseURL:    "https://api.commerce.coinbase.com",
		httpClient: &http.Client{},
	}
}

// CreateCharge creates a Coinbase Commerce charge
func (cc *CoinbaseClient) CreateCharge(req *PaymentRequest, networkConfig *NetworkConfig) (*CoinbaseCharge, error) {
	// Calculate token amount (assuming 1 USD = 10 tokens)
	tokenAmount := int(req.Amount * 10)

	chargeData := map[string]interface{}{
		"name":         fmt.Sprintf("%s Tokens", networkConfig.Token),
		"description":  fmt.Sprintf("Purchase %d %s tokens on %s", tokenAmount, networkConfig.Token, networkConfig.Name),
		"pricing_type": "fixed_price",
		"local_price": map[string]interface{}{
			"amount":   fmt.Sprintf("%.2f", req.Amount),
			"currency": "USD",
		},
		"metadata": map[string]interface{}{
			"wallet_address": req.Address,
			"network":        req.Network,
			"token":          networkConfig.Token,
			"token_amount":   tokenAmount,
		},
		"redirect_url": fmt.Sprintf("http://localhost:8080/success.html?network=%s", url.QueryEscape(req.Network)),
		"cancel_url":   fmt.Sprintf("http://localhost:8080/cancel.html?network=%s", url.QueryEscape(req.Network)),
	}

	jsonData, err := json.Marshal(chargeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal charge data: %w", err)
	}

	httpReq, err := http.NewRequest("POST", cc.baseURL+"/charges", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-CC-Api-Key", cc.apiKey)
	httpReq.Header.Set("X-CC-Version", "2018-03-22")

	resp, err := cc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("coinbase API error: %s", resp.Status)
	}

	var charge CoinbaseCharge
	if err := json.NewDecoder(resp.Body).Decode(&charge); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &charge, nil
}