// Package payment implements fiat-triggered NRN disbursement.
//
// This replaces KNIRVCHAIN's internal/wallet/payment_processor.go, which signed
// disbursement transactions with a P-256 identity wallet. Disbursement is an
// oracle-only privilege (KNIRVORACLE is the sole minter/transferrer of NRN), so
// this version disburses through the oracle's own secp256k1-signed token engine
// (internal/oracle/token) instead of holding a separate signing key.
package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// Config holds configuration for the payment processor.
type Config struct {
	Enabled               bool
	StripeSecretKey       string
	StripeWebhookSecret   string
	CoinbaseAPIKey        string
	CoinbaseWebhookSecret string
	TokenDecimals         int
	USDPerToken           float64
	ETHPerToken           float64

	// Plan checkout (KNIRV.COM pricing tiers, sold through
	// onboarding.knirv.com). Separate from the fiat->NRN disbursement path
	// above: a plan checkout confirms a purchase with the onboarding site
	// rather than minting tokens. Each is a pre-created Stripe Price ID;
	// falls back to inline price_data (see planPricing) when empty.
	StripeProfessionalPriceID string
	StripeEnterprisePriceID   string
	StripeInvestorPriceID     string
	DefaultSuccessURL         string
	DefaultCancelURL          string
	OnboardingCallbackURL     string // e.g. https://onboarding.knirv.com/api/onboarding
}

// LoadConfigFromEnv overlays payment processor settings from environment
// variables onto cfg, mirroring the env vars the KNIRVCHAIN payment processor
// used to read (STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET, etc).
func LoadConfigFromEnv(cfg *Config) {
	if v := os.Getenv("STRIPE_SECRET_KEY"); v != "" {
		cfg.StripeSecretKey = v
	}
	if v := os.Getenv("STRIPE_WEBHOOK_SECRET"); v != "" {
		cfg.StripeWebhookSecret = v
	}
	if v := os.Getenv("COINBASE_API_KEY"); v != "" {
		cfg.CoinbaseAPIKey = v
	}
	if v := os.Getenv("COINBASE_WEBHOOK_SECRET"); v != "" {
		cfg.CoinbaseWebhookSecret = v
	}
	if v := os.Getenv("ORACLE_PAYMENT_USD_PER_TOKEN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.USDPerToken = f
		}
	}
	if v := os.Getenv("ORACLE_PAYMENT_ETH_PER_TOKEN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.ETHPerToken = f
		}
	}
	if v := os.Getenv("ORACLE_PAYMENT_TOKEN_DECIMALS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TokenDecimals = n
		}
	}
	if v := os.Getenv("ORACLE_STRIPE_PROFESSIONAL_PRICE_ID"); v != "" {
		cfg.StripeProfessionalPriceID = v
	}
	if v := os.Getenv("ORACLE_STRIPE_ENTERPRISE_PRICE_ID"); v != "" {
		cfg.StripeEnterprisePriceID = v
	}
	if v := os.Getenv("ORACLE_STRIPE_INVESTOR_PRICE_ID"); v != "" {
		cfg.StripeInvestorPriceID = v
	}
	if v := os.Getenv("ORACLE_PAYMENT_SUCCESS_URL"); v != "" {
		cfg.DefaultSuccessURL = v
	}
	if v := os.Getenv("ORACLE_PAYMENT_CANCEL_URL"); v != "" {
		cfg.DefaultCancelURL = v
	}
	if v := os.Getenv("ORACLE_PAYMENT_ONBOARDING_CALLBACK_URL"); v != "" {
		cfg.OnboardingCallbackURL = v
	}
}

// Disburser is the narrow interface the payment processor needs from the
// oracle: mint-to-address, which is the only NRN issuance path. The oracle
// itself satisfies this interface (see Oracle.FundAddress).
type Disburser interface {
	FundAddress(addr types.Address, amount *big.Int, reason string) (*token.MintReceipt, error)
}

// Processor handles fiat webhook events and disburses NRN through the oracle's
// own mint path. It never holds a private key of its own.
type Processor struct {
	config            Config
	disburser         Disburser
	logger            *zap.Logger
	mu                sync.RWMutex
	processedPayments map[string]string // charge ID -> transaction hash, for idempotency
}

// NewProcessor creates a new payment processor bound to the given disburser
// (normally the *oracle.Oracle instance).
func NewProcessor(cfg Config, disburser Disburser, logger *zap.Logger) *Processor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Processor{
		config:            cfg,
		disburser:         disburser,
		logger:            logger,
		processedPayments: make(map[string]string),
	}
}

// calculateTokenAmount determines how many smallest-unit NRN to disburse for a
// given fiat/crypto amount.
func (p *Processor) calculateTokenAmount(amountReceived float64, currency string) (*big.Int, error) {
	var rate float64
	switch strings.ToLower(currency) {
	case "usd":
		if p.config.USDPerToken <= 0 {
			return nil, fmt.Errorf("USD exchange rate not configured")
		}
		rate = 1.0 / p.config.USDPerToken
	case "eth":
		if p.config.ETHPerToken <= 0 {
			return nil, fmt.Errorf("ETH exchange rate not configured")
		}
		rate = 1.0 / p.config.ETHPerToken
	default:
		return nil, fmt.Errorf("unsupported currency: %s", currency)
	}

	tokenValue := amountReceived * rate
	decimals := p.config.TokenDecimals
	if decimals <= 0 {
		decimals = 18
	}
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	tokenAmountFloat := new(big.Float).Mul(new(big.Float).SetFloat64(tokenValue), multiplier)

	tokenAmountInt, _ := tokenAmountFloat.Int(nil)
	if tokenAmountInt.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("calculated token amount is zero or negative")
	}
	return tokenAmountInt, nil
}

// CheckoutSessionRequest is a public request to start a plan checkout,
// submitted by onboarding.knirv.com through the public KNIRVGATEWAY
// hostnames (gateway.knirv.network / testnet-gateway.knirv.network).
type CheckoutSessionRequest struct {
	Plan          string `json:"plan"`
	SessionID     string `json:"sessionId"`
	CustomerEmail string `json:"customerEmail,omitempty"`
	SuccessURL    string `json:"successUrl,omitempty"`
	CancelURL     string `json:"cancelUrl,omitempty"`
}

// CheckoutSessionResponse carries the Stripe-hosted checkout URL back to the
// browser so it can redirect the operator to complete payment.
type CheckoutSessionResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// planPrice describes one plan's Stripe price: the checkout mode, its
// inline-price fallback in USD cents, and (for subscriptions) the billing
// interval. Prices mirror the KNIRV.COM pricing page
// (websites/KNIRV.COM/src/app/pricing/page.tsx).
type planPrice struct {
	mode           string // "subscription" or "payment" (one-time)
	usdCents       int64
	productName    string
	priceIDFromCfg func(cfg Config) string // pre-created Stripe Price ID, if configured
}

// planPricing is the source of truth for what each plan actually charges.
// "investor" is the $50,000 one-time Bootnode-operator package rebranded for
// the pricing page — its checkout is created here, but confirmation routes
// to the bootnode flow rather than onboarding.ts (see notifyOnboardingPaid).
var planPricing = map[string]planPrice{
	"professional": {
		mode: "subscription", usdCents: 1999, productName: "KNIRV Professional Plan",
		priceIDFromCfg: func(cfg Config) string { return cfg.StripeProfessionalPriceID },
	},
	"enterprise": {
		mode: "subscription", usdCents: 19999, productName: "KNIRV Enterprise Plan",
		priceIDFromCfg: func(cfg Config) string { return cfg.StripeEnterprisePriceID },
	},
	"investor": {
		mode: "payment", usdCents: 5000000, productName: "KNIRV Investor / Bootnode Package",
		priceIDFromCfg: func(cfg Config) string { return cfg.StripeInvestorPriceID },
	},
}

// CreateCheckoutSession creates a Stripe Checkout Session for a supported
// plan and returns the hosted checkout URL. This calls the Stripe REST API
// directly (form-encoded, no SDK) to stay consistent with
// HandleStripeWebhook's hand-rolled JSON handling below — the oracle module
// does not depend on stripe-go.
func (p *Processor) CreateCheckoutSession(req *CheckoutSessionRequest) (*CheckoutSessionResponse, error) {
	if !p.config.Enabled {
		return nil, fmt.Errorf("payment processor disabled")
	}
	if p.config.StripeSecretKey == "" {
		return nil, fmt.Errorf("stripe secret key not configured")
	}
	price, ok := planPricing[req.Plan]
	if !ok {
		return nil, fmt.Errorf("unsupported plan: %q", req.Plan)
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}

	successURL := req.SuccessURL
	if successURL == "" {
		successURL = p.config.DefaultSuccessURL
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = p.config.DefaultCancelURL
	}
	if successURL == "" || cancelURL == "" {
		return nil, fmt.Errorf("success/cancel URL not configured")
	}

	form := url.Values{}
	form.Set("mode", price.mode)
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("line_items[0][quantity]", "1")
	if priceID := price.priceIDFromCfg(p.config); priceID != "" {
		form.Set("line_items[0][price]", priceID)
	} else {
		form.Set("line_items[0][price_data][currency]", "usd")
		form.Set("line_items[0][price_data][product_data][name]", price.productName)
		form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(price.usdCents, 10))
		// Stripe rejects a `recurring` field on mode=payment line items, so
		// this is only set for subscription-mode plans.
		if price.mode == "subscription" {
			form.Set("line_items[0][price_data][recurring][interval]", "month")
		}
	}
	form.Set("metadata[plan]", req.Plan)
	form.Set("metadata[sessionId]", req.SessionID)
	if req.CustomerEmail != "" {
		form.Set("customer_email", req.CustomerEmail)
	}

	httpReq, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build stripe request: %w", err)
	}
	httpReq.SetBasicAuth(p.config.StripeSecretKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if resp.StatusCode >= 300 || body.Error != nil {
		msg := "unknown error"
		if body.Error != nil {
			msg = body.Error.Message
		}
		return nil, fmt.Errorf("stripe error: %s", msg)
	}

	return &CheckoutSessionResponse{ID: body.ID, URL: body.URL}, nil
}

// HandleCreateCheckoutSession is the public HTTP handler mounted at
// /oracle/v3/payment/checkout/create. It is reached exclusively through
// KNIRVGATEWAY's public hostnames — the gateway's blanket "/oracle/" proxy
// forwards to this route with no path stripping, and the gateway's own CORS
// middleware (internal/server/server.go) is the authority for cross-origin
// browser calls, stripping any CORS headers this handler might set.
func (p *Processor) HandleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	session, err := p.CreateCheckoutSession(&req)
	if err != nil {
		p.logger.Warn("failed to create plan checkout session",
			zap.String("plan", req.Plan), zap.String("session_id", req.SessionID), zap.Error(err))
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// writeJSONError writes {"error": message} — the onboarding site's browser
// JS (startProCheckout/startEnterpriseCheckout/startBootnodeCheckout in
// index.html) parses error responses as JSON and falls back to a useless
// generic "HTTP <code>" message when that parse fails, which is exactly
// what plain-text http.Error() responses caused here before.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleCheckoutSessionCompleted processes a completed subscription-plan
// checkout (as opposed to the one-time "charge.succeeded" NRN-disbursement
// path handled below): it confirms payment with onboarding.knirv.com rather
// than minting tokens, since a Pro-plan purchase has no wallet address to
// fund.
func (p *Processor) handleCheckoutSessionCompleted(w http.ResponseWriter, object map[string]interface{}) {
	checkoutID, _ := object["id"].(string)
	if checkoutID == "" {
		http.Error(w, "Checkout session ID missing", http.StatusBadRequest)
		return
	}

	metadata, _ := object["metadata"].(map[string]interface{})
	plan, _ := metadata["plan"].(string)
	onboardingSessionID, _ := metadata["sessionId"].(string)

	if plan == "" || onboardingSessionID == "" {
		fmt.Fprintf(w, "Ignored: checkout session has no tracked plan/sessionId metadata")
		return
	}

	p.mu.RLock()
	_, processed := p.processedPayments[checkoutID]
	p.mu.RUnlock()
	if processed {
		fmt.Fprintf(w, "Duplicate event, already processed")
		return
	}

	if err := p.notifyOnboardingPaid(onboardingSessionID, plan); err != nil {
		p.logger.Error("failed to confirm plan payment with onboarding",
			zap.String("onboarding_session_id", onboardingSessionID), zap.String("plan", plan), zap.Error(err))
		http.Error(w, "Failed to confirm onboarding payment", http.StatusBadGateway)
		return
	}

	p.mu.Lock()
	p.processedPayments[checkoutID] = "onboarding-notified"
	p.mu.Unlock()

	p.logger.Info("confirmed plan payment with onboarding",
		zap.String("onboarding_session_id", onboardingSessionID), zap.String("plan", plan))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
}

// notifyOnboardingPaid tells onboarding.knirv.com's session-sync endpoint
// that the given onboarding session has completed payment, so it can flip
// the customer profile from "pending" to "paid" (see
// KNIRV_CORP/websites/ONBOARDING.KNIRV.COM/functions/api/onboarding.ts,
// which gates paid-customer status on eventType == "payment-confirmed").
func (p *Processor) notifyOnboardingPaid(sessionID, plan string) error {
	if p.config.OnboardingCallbackURL == "" {
		return fmt.Errorf("onboarding callback URL not configured")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"sessionId": sessionID,
		"plan":      plan,
		"stage":     plan + "-paid",
		"eventType": "payment-confirmed",
	})
	if err != nil {
		return fmt.Errorf("failed to encode callback payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.config.OnboardingCallbackURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to build callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}
	return nil
}

const (
	// stripeSignatureTolerance bounds how old a webhook's t= timestamp may be
	// before it's rejected as a possible replay, per Stripe's documented
	// scheme (https://stripe.com/docs/webhooks/signatures#replay-attacks).
	stripeSignatureTolerance = 5 * time.Minute
	// maxWebhookBodyBytes caps how much of the request body this public
	// endpoint will buffer before verifying its signature — Stripe's actual
	// payloads are a few KB at most.
	maxWebhookBodyBytes = 1 << 20 // 1 MiB
)

// verifyStripeSignature implements Stripe's documented webhook signature
// scheme by hand (https://stripe.com/docs/webhooks/signatures), consistent
// with the rest of this file's hand-rolled Stripe integration (no stripe-go
// dependency). The Stripe-Signature header looks like
// "t=<unix-seconds>,v1=<hex-hmac>[,v1=<hex-hmac>...]" — multiple v1 entries
// appear during Stripe's own signing-secret rotation, and a match on any one
// of them is accepted; v0 entries are the legacy SHA-1 scheme and are
// ignored, matching Stripe's own guidance to only check v1.
func verifyStripeSignature(payload []byte, sigHeader, secret string, tolerance time.Duration) error {
	if secret == "" {
		return fmt.Errorf("stripe webhook secret not configured")
	}
	if sigHeader == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	var timestamp int64
	var signatures []string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp in Stripe-Signature header: %w", err)
			}
			timestamp = ts
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}
	if timestamp == 0 {
		return fmt.Errorf("missing timestamp in Stripe-Signature header")
	}
	if len(signatures) == 0 {
		return fmt.Errorf("missing v1 signature in Stripe-Signature header")
	}

	if tolerance > 0 {
		age := time.Since(time.Unix(timestamp, 0))
		if age < 0 {
			age = -age
		}
		if age > tolerance {
			return fmt.Errorf("Stripe-Signature timestamp outside tolerance window (possible replay)")
		}
	}

	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := mac.Sum(nil)

	for _, sig := range signatures {
		decoded, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expected) {
			return nil
		}
	}

	return fmt.Errorf("no matching Stripe-Signature v1 value")
}

// HandleStripeWebhook processes incoming Stripe webhook events, after
// verifying the Stripe-Signature header against config.StripeWebhookSecret
// (see verifyStripeSignature) — without this, anyone who discovered this
// public URL could forge a "checkout.session.completed" or "charge.succeeded"
// event and mark a plan paid or trigger an NRN mint for free. Two event
// types are handled: "checkout.session.completed" confirms a subscription
// plan purchase with onboarding.knirv.com (see handleCheckoutSessionCompleted
// above), and "charge.succeeded" disburses NRN to the address supplied in
// the charge's metadata (the original fiat->NRN path, unchanged below).
func (p *Processor) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if !p.config.Enabled {
		http.Error(w, "payment processor disabled", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes+1))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > maxWebhookBodyBytes {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := verifyStripeSignature(body, r.Header.Get("Stripe-Signature"), p.config.StripeWebhookSecret, stripeSignatureTolerance); err != nil {
		p.logger.Warn("rejected stripe webhook: signature verification failed", zap.Error(err))
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Cannot parse request body", http.StatusBadRequest)
		return
	}

	eventType, _ := event["type"].(string)
	data, _ := event["data"].(map[string]interface{})
	object, _ := data["object"].(map[string]interface{})
	if object == nil {
		http.Error(w, "Event object missing", http.StatusBadRequest)
		return
	}

	switch eventType {
	case "checkout.session.completed":
		p.handleCheckoutSessionCompleted(w, object)
		return
	case "charge.succeeded":
		// falls through to the disbursement logic below
	default:
		fmt.Fprintf(w, "Unhandled event type")
		return
	}

	chargeID, _ := object["id"].(string)
	if chargeID == "" {
		http.Error(w, "Charge ID missing", http.StatusBadRequest)
		return
	}

	p.mu.RLock()
	txHash, processed := p.processedPayments[chargeID]
	p.mu.RUnlock()
	if processed {
		fmt.Fprintf(w, "Duplicate event, already processed as %s", txHash)
		return
	}

	amountReceived, currency, err := extractPaymentDetails(object)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error extracting payment details: %v", err), http.StatusBadRequest)
		return
	}

	metadata, _ := object["metadata"].(map[string]interface{})
	recipientStr, _ := metadata["_address"].(string)
	if recipientStr == "" {
		p.logger.Error("stripe webhook missing recipient address metadata", zap.String("charge_id", chargeID))
		http.Error(w, "Missing recipient address metadata", http.StatusBadRequest)
		return
	}

	recipient, err := types.AddressFromString(recipientStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid recipient address: %v", err), http.StatusBadRequest)
		return
	}

	tokenAmount, err := p.calculateTokenAmount(amountReceived, currency)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot calculate token amount: %v", err), http.StatusBadRequest)
		return
	}

	receipt, err := p.disburser.FundAddress(recipient, tokenAmount, "stripe:"+chargeID)
	if err != nil {
		p.logger.Error("failed to disburse NRN for stripe charge",
			zap.String("charge_id", chargeID), zap.Error(err))
		http.Error(w, "Token disbursement failed", http.StatusInternalServerError)
		return
	}

	p.mu.Lock()
	p.processedPayments[chargeID] = receipt.TransactionHash
	p.mu.Unlock()

	p.logger.Info("disbursed NRN for stripe charge",
		zap.String("charge_id", chargeID),
		zap.String("recipient", recipient.String()),
		zap.String("amount", tokenAmount.String()),
		zap.String("transaction_hash", receipt.TransactionHash),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipt)
}

func extractPaymentDetails(object map[string]interface{}) (float64, string, error) {
	amountReceived, ok := object["amount"].(float64)
	if !ok {
		if amountStr, ok := object["amount"].(json.Number); ok {
			var err error
			amountReceived, err = amountStr.Float64()
			if err != nil {
				return 0, "", fmt.Errorf("invalid amount format: %w", err)
			}
		} else {
			return 0, "", fmt.Errorf("amount missing or invalid format")
		}
	}

	// Stripe amounts are in cents.
	amountReceivedFloat := amountReceived / 100.0

	currency, ok := object["currency"].(string)
	if !ok {
		return 0, "", fmt.Errorf("currency missing")
	}

	return amountReceivedFloat, currency, nil
}
