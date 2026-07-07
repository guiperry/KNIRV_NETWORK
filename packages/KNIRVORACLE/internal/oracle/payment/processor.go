// Package payment implements fiat-triggered NRN disbursement.
//
// This replaces KNIRVCHAIN's internal/wallet/payment_processor.go, which signed
// disbursement transactions with a P-256 identity wallet. Disbursement is an
// oracle-only privilege (KNIRVORACLE is the sole minter/transferrer of NRN), so
// this version disburses through the oracle's own secp256k1-signed token engine
// (internal/oracle/token) instead of holding a separate signing key.
package payment

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

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

// HandleStripeWebhook processes incoming Stripe webhook events and disburses
// NRN to the address supplied in the charge's metadata.
//
// TODO: this does not yet verify the Stripe-Signature header against
// config.StripeWebhookSecret using the Stripe SDK. Do not enable Enabled in
// production until that verification is wired in — see
// docs/Validator_Terms_and_Conditions_DRAFT.md and
// docs/Bootnode_Failover_Implementation_Plan.md Phase 1 for the tracked
// follow-up.
func (p *Processor) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if !p.config.Enabled {
		http.Error(w, "payment processor disabled", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Cannot parse request body", http.StatusBadRequest)
		return
	}

	eventType, _ := event["type"].(string)
	if eventType != "charge.succeeded" {
		fmt.Fprintf(w, "Unhandled event type")
		return
	}

	data, _ := event["data"].(map[string]interface{})
	object, _ := data["object"].(map[string]interface{})
	if object == nil {
		http.Error(w, "Event object missing", http.StatusBadRequest)
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
