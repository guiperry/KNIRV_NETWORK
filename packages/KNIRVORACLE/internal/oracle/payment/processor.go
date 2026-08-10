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
	Enabled             bool
	StripeSecretKey     string
	StripeWebhookSecret string
	// StripeThinWebhookSecret is the signing secret for a second Stripe Event
	// Destination delivering "thin" payloads (see
	// https://docs.stripe.com/event-destinations#thin-events). Stripe mints a
	// distinct signing secret per destination, and — since account-level thin
	// events for classic v1 resources (refund.*, invoice.*,
	// checkout.session.*, payment_intent.*) are a private preview — selecting
	// a mix of event types can force Stripe's dashboard to split one event
	// selection into two destinations, one per payload style, even though
	// both can point at this same webhook URL. Left empty, only
	// StripeWebhookSecret is checked (unchanged, single-destination
	// behavior).
	StripeThinWebhookSecret string
	CoinbaseAPIKey          string
	CoinbaseWebhookSecret   string
	TokenDecimals           int
	USDPerToken             float64
	ETHPerToken             float64

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
	if v := os.Getenv("STRIPE_THIN_WEBHOOK_SECRET"); v != "" {
		cfg.StripeThinWebhookSecret = v
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

// alreadyProcessed and markProcessed share the idempotency map used across
// every webhook event type this processor handles (checkout, refund,
// invoice) — each caller picks its own prefixed key so unrelated event types
// never collide.
func (p *Processor) alreadyProcessed(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.processedPayments[key]
	return ok
}

func (p *Processor) markProcessed(key, value string) {
	p.mu.Lock()
	p.processedPayments[key] = value
	p.mu.Unlock()
}

// doStripeGet performs an authenticated GET against a fully-qualified Stripe
// API URL, consistent with this file's hand-rolled (no stripe-go)
// integration style.
func (p *Processor) doStripeGet(fullURL string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build stripe request: %w", err)
	}
	req.SetBasicAuth(p.config.StripeSecretKey, "")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe GET %s failed: %v", fullURL, out["error"])
	}
	return out, nil
}

// fetchStripeObject performs an authenticated GET against api.stripe.com/v1/
// for a bare "resource/id" path. Used as a one-hop fallback when a webhook
// object doesn't carry the plan/session metadata inline (see
// resolvePlanSession).
func (p *Processor) fetchStripeObject(path string) (map[string]interface{}, error) {
	return p.doStripeGet("https://api.stripe.com/v1/" + path)
}

// fetchStripeURL performs an authenticated GET against a full API path as
// delivered in a thin event's related_object.url (e.g. "/v1/refunds/re_123"
// or "/v2/core/accounts/acct_123") — distinct from fetchStripeObject, which
// is handed a path relative to the v1 API root rather than the full,
// version-prefixed path Stripe supplies inline.
func (p *Processor) fetchStripeURL(urlPath string) (map[string]interface{}, error) {
	return p.doStripeGet("https://api.stripe.com" + urlPath)
}

// metadataPlanSession reads the plan/sessionId pair CreateCheckoutSession
// stamps onto Stripe objects at checkout time (see form.Set("metadata[...]")
// and, for subscriptions, form.Set("subscription_data[metadata][...]")).
func metadataPlanSession(object map[string]interface{}) (plan, sessionID string) {
	metadata, _ := object["metadata"].(map[string]interface{})
	plan, _ = metadata["plan"].(string)
	sessionID, _ = metadata["sessionId"].(string)
	return
}

// resolvePlanSession traces a billing-lifecycle event's Stripe object back to
// the plan/session metadata set at Checkout time. Where that metadata lands
// depends on checkout mode and object type:
//   - mode=payment (investor): Stripe copies Checkout Session metadata onto
//     the resulting PaymentIntent/Charge directly.
//   - mode=subscription (professional/enterprise): CreateCheckoutSession sets
//     subscription_data[metadata], so it lands on the Subscription; modern
//     Stripe API versions also embed a snapshot of it at each invoice's
//     subscription_details.metadata.
//
// When neither is present inline, this falls back to one authenticated GET
// against whichever of subscription/invoice/payment_intent/charge the object
// references, to bound worst-case latency/cost on a webhook handler to a
// single extra round trip.
func (p *Processor) resolvePlanSession(object map[string]interface{}) (plan, sessionID string) {
	if plan, sessionID = metadataPlanSession(object); plan != "" && sessionID != "" {
		return
	}
	if sd, ok := object["subscription_details"].(map[string]interface{}); ok {
		if plan, sessionID = metadataPlanSession(sd); plan != "" && sessionID != "" {
			return
		}
	}

	for _, ref := range []struct{ field, path string }{
		{"subscription", "subscriptions/"},
		{"invoice", "invoices/"},
		{"payment_intent", "payment_intents/"},
		{"charge", "charges/"},
	} {
		id, _ := object[ref.field].(string)
		if id == "" {
			continue
		}
		fetched, err := p.fetchStripeObject(ref.path + id)
		if err != nil {
			p.logger.Warn("failed to resolve plan/session via Stripe API fallback",
				zap.String("field", ref.field), zap.String("id", id), zap.Error(err))
			continue
		}
		if plan, sessionID = metadataPlanSession(fetched); plan != "" && sessionID != "" {
			return
		}
		if sd, ok := fetched["subscription_details"].(map[string]interface{}); ok {
			if plan, sessionID = metadataPlanSession(sd); plan != "" && sessionID != "" {
				return
			}
		}
	}
	return "", ""
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
	if price.mode == "subscription" {
		// Checkout Session metadata is NOT copied onto the Subscription Stripe
		// creates for mode=subscription (unlike mode=payment, where it's
		// copied onto the PaymentIntent automatically) — without this,
		// renewal-cycle events (invoice.paid, invoice.payment_action_required,
		// etc.) would have no way to trace back to a plan/session via
		// resolvePlanSession.
		form.Set("subscription_data[metadata][plan]", req.Plan)
		form.Set("subscription_data[metadata][sessionId]", req.SessionID)
	}
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

// handleCheckoutSessionCompleted processes a checkout session that has
// completed and actually collected payment — called for both
// "checkout.session.completed" and "checkout.session.async_payment_succeeded"
// (as opposed to the one-time "charge.succeeded" NRN-disbursement path
// handled below): it confirms payment with onboarding.knirv.com rather than
// minting tokens, since a plan purchase has no wallet address to fund.
//
// Delayed payment methods (e.g. bank debits) report
// "checkout.session.completed" with payment_status="unpaid" — funds aren't
// captured yet at that point, and confirming here would grant paid status
// before money actually moved. Only payment_status="paid" is treated as a
// real confirmation; an unpaid completed session just waits for the
// following async_payment_succeeded/async_payment_failed event, which this
// same function handles once payment_status has flipped to "paid".
func (p *Processor) handleCheckoutSessionCompleted(w http.ResponseWriter, object map[string]interface{}) {
	checkoutID, _ := object["id"].(string)
	if checkoutID == "" {
		http.Error(w, "Checkout session ID missing", http.StatusBadRequest)
		return
	}

	if paymentStatus, _ := object["payment_status"].(string); paymentStatus != "paid" {
		fmt.Fprintf(w, "Ignored: checkout session payment_status is %q, awaiting async confirmation", paymentStatus)
		return
	}

	metadata, _ := object["metadata"].(map[string]interface{})
	plan, _ := metadata["plan"].(string)
	onboardingSessionID, _ := metadata["sessionId"].(string)

	if plan == "" || onboardingSessionID == "" {
		fmt.Fprintf(w, "Ignored: checkout session has no tracked plan/sessionId metadata")
		return
	}

	dedupeKey := "checkout:" + checkoutID
	if p.alreadyProcessed(dedupeKey) {
		fmt.Fprintf(w, "Duplicate event, already processed")
		return
	}

	if err := p.notifyOnboardingPaid(onboardingSessionID, plan); err != nil {
		p.logger.Error("failed to confirm plan payment with onboarding",
			zap.String("onboarding_session_id", onboardingSessionID), zap.String("plan", plan), zap.Error(err))
		http.Error(w, "Failed to confirm onboarding payment", http.StatusBadGateway)
		return
	}

	p.markProcessed(dedupeKey, "onboarding-notified")

	p.logger.Info("confirmed plan payment with onboarding",
		zap.String("onboarding_session_id", onboardingSessionID), zap.String("plan", plan))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
}

// handleRefundEvent processes refund.created and refund.failed. A refund
// that has actually succeeded revokes the plan's paid status on
// onboarding.knirv.com; a failed refund *attempt* changes nothing about the
// customer's access (the original charge still stands) and is only logged.
func (p *Processor) handleRefundEvent(w http.ResponseWriter, object map[string]interface{}, eventType string) {
	refundID, _ := object["id"].(string)

	if eventType == "refund.failed" {
		p.logger.Warn("stripe refund attempt failed", zap.String("refund_id", refundID))
		fmt.Fprintf(w, "Logged: refund attempt failed")
		return
	}

	// refund.created: card refunds typically report status=succeeded
	// immediately; other statuses (pending/requires_action, e.g. for bank
	// debits) aren't final yet, so there's nothing to revoke until a later
	// delivery reports succeeded.
	status, _ := object["status"].(string)
	if status != "succeeded" {
		fmt.Fprintf(w, "Ignored: refund not yet succeeded (status=%s)", status)
		return
	}

	dedupeKey := "refund-paid:" + refundID
	if p.alreadyProcessed(dedupeKey) {
		fmt.Fprintf(w, "Duplicate event, already processed")
		return
	}

	plan, sessionID := p.resolvePlanSession(object)
	if plan == "" || sessionID == "" {
		p.logger.Warn("refund succeeded but could not resolve plan/session", zap.String("refund_id", refundID))
		fmt.Fprintf(w, "Ignored: could not resolve plan/session for refund")
		return
	}

	if err := p.notifyOnboardingStatus(sessionID, plan, "payment-refunded", plan+"-refunded"); err != nil {
		p.logger.Error("failed to confirm refund with onboarding",
			zap.String("onboarding_session_id", sessionID), zap.String("plan", plan), zap.Error(err))
		http.Error(w, "Failed to confirm onboarding refund", http.StatusBadGateway)
		return
	}

	p.markProcessed(dedupeKey, "onboarding-notified")
	p.logger.Info("confirmed refund with onboarding",
		zap.String("onboarding_session_id", sessionID), zap.String("plan", plan))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
}

// handleInvoicePaidEvent processes invoice.paid and invoice.payment_succeeded
// — Stripe fires both for the same successful invoice, deduped below by
// invoice ID. A cleared invoice means a subscription renewal went through, so
// the plan should read (or stay) "paid", same as the initial checkout.
func (p *Processor) handleInvoicePaidEvent(w http.ResponseWriter, object map[string]interface{}) {
	invoiceID, _ := object["id"].(string)
	dedupeKey := "invoice-paid:" + invoiceID
	if p.alreadyProcessed(dedupeKey) {
		fmt.Fprintf(w, "Duplicate event, already processed")
		return
	}

	plan, sessionID := p.resolvePlanSession(object)
	if plan == "" || sessionID == "" {
		// Not every invoice belongs to a plan-checkout subscription (the
		// fiat->NRN disbursement path below never creates one) — that's the
		// normal case here, not an error.
		fmt.Fprintf(w, "Ignored: invoice has no tracked plan/session metadata")
		return
	}

	if err := p.notifyOnboardingPaid(sessionID, plan); err != nil {
		p.logger.Error("failed to confirm invoice payment with onboarding",
			zap.String("onboarding_session_id", sessionID), zap.String("plan", plan), zap.Error(err))
		http.Error(w, "Failed to confirm onboarding payment", http.StatusBadGateway)
		return
	}

	p.markProcessed(dedupeKey, "onboarding-notified")
	p.logger.Info("confirmed renewal payment with onboarding",
		zap.String("onboarding_session_id", sessionID), zap.String("plan", plan))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
}

// handlePaymentActionRequiredEvent processes invoice.payment_action_required
// and payment_intent.requires_action: the customer needs to complete
// additional authentication (e.g. 3D Secure) before a renewal/charge clears.
// This flags the account rather than revoking access — Stripe keeps
// retrying/prompting on its own schedule, and onboarding.ts (resolveStatus)
// deliberately will not downgrade an already-paid account off this signal
// alone, only surface it for accounts that aren't paid yet.
func (p *Processor) handlePaymentActionRequiredEvent(w http.ResponseWriter, object map[string]interface{}, eventType string) {
	objectID, _ := object["id"].(string)
	dedupeKey := "action-required:" + eventType + ":" + objectID
	if p.alreadyProcessed(dedupeKey) {
		fmt.Fprintf(w, "Duplicate event, already processed")
		return
	}

	plan, sessionID := p.resolvePlanSession(object)
	if plan == "" || sessionID == "" {
		fmt.Fprintf(w, "Ignored: could not resolve plan/session for %s", eventType)
		return
	}

	if err := p.notifyOnboardingStatus(sessionID, plan, "payment-action-required", plan+"-action-required"); err != nil {
		p.logger.Error("failed to flag action-required payment with onboarding",
			zap.String("onboarding_session_id", sessionID), zap.String("plan", plan), zap.Error(err))
		http.Error(w, "Failed to notify onboarding", http.StatusBadGateway)
		return
	}

	p.markProcessed(dedupeKey, "onboarding-notified")
	p.logger.Info("flagged payment action required with onboarding",
		zap.String("onboarding_session_id", sessionID), zap.String("plan", plan), zap.String("event_type", eventType))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "action_required"})
}

// notifyOnboardingStatus tells onboarding.knirv.com's session-sync endpoint
// about a billing-lifecycle status change for the given session (see
// KNIRV_CORP/websites/ONBOARDING.KNIRV.COM/functions/api/onboarding.ts'
// resolveStatus, which switches on eventType: "payment-confirmed" ->  paid,
// "payment-refunded" -> refunded, "payment-action-required" -> action
// required unless already paid).
func (p *Processor) notifyOnboardingStatus(sessionID, plan, eventType, stage string) error {
	if p.config.OnboardingCallbackURL == "" {
		return fmt.Errorf("onboarding callback URL not configured")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"sessionId": sessionID,
		"plan":      plan,
		"stage":     stage,
		"eventType": eventType,
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

// notifyOnboardingPaid is the "payment-confirmed" case of
// notifyOnboardingStatus, kept as a named wrapper since it's the original/
// most common call (initial checkout and every renewal).
func (p *Processor) notifyOnboardingPaid(sessionID, plan string) error {
	return p.notifyOnboardingStatus(sessionID, plan, "payment-confirmed", plan+"-paid")
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

// verifyWebhookSignature checks the raw request body's Stripe-Signature
// against every configured secret, succeeding on the first match. This is
// what makes it safe to point two Stripe Event Destinations — a snapshot one
// and a thin one, each with its own Stripe-issued signing secret — at this
// same URL (see the StripeThinWebhookSecret doc comment on Config). The
// per-secret check itself (including replay-window tolerance and rotation
// support for multiple v1= values in one header) is unchanged, see
// verifyStripeSignature.
func (p *Processor) verifyWebhookSignature(body []byte, sigHeader string) error {
	secrets := []string{p.config.StripeWebhookSecret, p.config.StripeThinWebhookSecret}
	var lastErr error
	configured := false
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		configured = true
		if err := verifyStripeSignature(body, sigHeader, secret, stripeSignatureTolerance); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if !configured {
		return fmt.Errorf("stripe webhook secret not configured")
	}
	return lastErr
}

// normalizeStripeEvent resolves an incoming webhook body to (eventType,
// object) regardless of which Stripe Event Destination payload style
// delivered it, so every handler below only ever deals with one shape:
//
//   - Snapshot events (`"object": "event"`) embed the full resource inline at
//     data.object — used as-is.
//   - Thin events (`"object": "v2.core.event"`) carry only a pointer at
//     related_object.url; this fetches that URL from the Stripe API to
//     obtain the same shape a snapshot event would have embedded directly.
//     See https://docs.stripe.com/event-destinations#thin-events — thin
//     payloads for classic v1 resources (refund.*, invoice.*,
//     checkout.session.*, payment_intent.*) are what forces a second,
//     thin-style Event Destination alongside the snapshot one.
func (p *Processor) normalizeStripeEvent(raw map[string]interface{}) (eventType string, object map[string]interface{}, err error) {
	eventType, _ = raw["type"].(string)
	if eventType == "" {
		return "", nil, fmt.Errorf("event missing type")
	}

	if objectKind, _ := raw["object"].(string); objectKind == "v2.core.event" {
		relatedObject, _ := raw["related_object"].(map[string]interface{})
		objURL, _ := relatedObject["url"].(string)
		if objURL == "" {
			return "", nil, fmt.Errorf("thin event missing related_object.url")
		}
		fetched, ferr := p.fetchStripeURL(objURL)
		if ferr != nil {
			return "", nil, fmt.Errorf("failed to fetch thin event's related object: %w", ferr)
		}
		return eventType, fetched, nil
	}

	data, _ := raw["data"].(map[string]interface{})
	object, _ = data["object"].(map[string]interface{})
	if object == nil {
		return "", nil, fmt.Errorf("event object missing")
	}
	return eventType, object, nil
}

// HandleStripeWebhook processes incoming Stripe webhook events, after
// verifying the Stripe-Signature header against every configured secret (see
// verifyWebhookSignature) — without this, anyone who discovered this public
// URL could forge a "checkout.session.completed" or "charge.succeeded" event
// and mark a plan paid or trigger an NRN mint for free.
//
// This endpoint accepts both Stripe Event Destination payload styles —
// classic snapshot (`{id, object: "event", type, data: {object: {...}}}`)
// and thin (`{id, object: "v2.core.event", type, related_object: {url}}`) —
// normalized to the same (eventType, object) shape by normalizeStripeEvent
// before dispatch. Point both a snapshot-style and a thin-style Stripe Event
// Destination at this same URL; each needs its own secret configured (see
// Config.StripeThinWebhookSecret).
//
// Event types handled:
//   - "checkout.session.completed" / "checkout.session.async_payment_succeeded"
//     confirm a plan purchase once payment_status is actually "paid" (see
//     handleCheckoutSessionCompleted) — "completed" alone doesn't mean funds
//     cleared for delayed payment methods, hence the async event too
//   - "checkout.session.expired" / "checkout.session.async_payment_failed"
//     are logged only — the session never collected payment, so there's
//     nothing to confirm or revoke
//   - "refund.created" / "refund.failed" revoke paid status on a completed
//     refund, or just log a failed refund attempt (see handleRefundEvent)
//   - "invoice.paid" / "invoice.payment_succeeded" confirm a subscription
//     renewal, same as the initial checkout (see handleInvoicePaidEvent)
//   - "invoice.payment_action_required" / "payment_intent.requires_action"
//     flag an account as needing customer action, without revoking an
//     already-paid account (see handlePaymentActionRequiredEvent)
//   - "invoice.sent" is logged only — informational, no plan-checkout
//     subscription in this codebase uses Stripe's send-invoice collection
//     method
//   - "charge.succeeded" disburses NRN to the address supplied in the
//     charge's metadata (the original fiat->NRN path, unchanged below)
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

	if err := p.verifyWebhookSignature(body, r.Header.Get("Stripe-Signature")); err != nil {
		p.logger.Warn("rejected stripe webhook: signature verification failed", zap.Error(err))
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Cannot parse request body", http.StatusBadRequest)
		return
	}

	eventType, object, err := p.normalizeStripeEvent(raw)
	if err != nil {
		p.logger.Warn("failed to normalize stripe event", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch eventType {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		p.handleCheckoutSessionCompleted(w, object)
		return
	case "checkout.session.expired", "checkout.session.async_payment_failed":
		checkoutID, _ := object["id"].(string)
		p.logger.Info("stripe checkout session did not collect payment",
			zap.String("checkout_id", checkoutID), zap.String("event_type", eventType))
		fmt.Fprintf(w, "Logged: %s", eventType)
		return
	case "refund.created", "refund.failed":
		p.handleRefundEvent(w, object, eventType)
		return
	case "invoice.paid", "invoice.payment_succeeded":
		p.handleInvoicePaidEvent(w, object)
		return
	case "invoice.payment_action_required", "payment_intent.requires_action":
		p.handlePaymentActionRequiredEvent(w, object, eventType)
		return
	case "invoice.sent":
		invoiceID, _ := object["id"].(string)
		p.logger.Info("stripe invoice sent", zap.String("invoice_id", invoiceID))
		fmt.Fprintf(w, "Logged: invoice sent")
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
