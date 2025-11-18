package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config"
	"KNIRVORACLE/pkg/utils"
)

// PaymentProcessorImpl handles the payment processing functionality
type PaymentProcessorImpl struct {
	config             config.PaymentProcessorConfig
	disbursementWallet *WalletImpl // The loaded wallet for sending tokens
	httpClient         *http.Client
	processedPayments  map[string]string // Map of payment IDs to transaction hashes
	server             *http.Server
	mutex              sync.RWMutex // For thread-safe access to processedPayments
}

// NewPaymentProcessor creates a new payment processor instance
func NewPaymentProcessor(cfg config.PaymentProcessorConfig, masterWallet *WalletImpl) (*PaymentProcessorImpl, error) {
	log.Println("Initializing payment processor...")

	if masterWallet == nil {
		return nil, fmt.Errorf("master wallet is required for payment processor")
	}

	log.Printf("Using master wallet with address: %s for token disbursement", masterWallet.GetAddress())

	// Initialize payment gateway SDKs here if needed
	// stripe.Key = cfg.StripeSecretKey
	// ...

	return &PaymentProcessorImpl{
		config:             cfg,
		disbursementWallet: masterWallet,
		httpClient:         &http.Client{Timeout: 10 * time.Second},
		processedPayments:  make(map[string]string),
	}, nil
}

// Start starts the payment processor HTTP server
func (pp *PaymentProcessorImpl) Start() error {
	if !pp.config.Enabled {
		log.Println("Payment processor is disabled, not starting server")
		return nil
	}

	// Create a new HTTP server
	mux := http.NewServeMux()

	// Register webhook handlers
	mux.HandleFunc("/webhooks/stripe", pp.handleStripeWebhook)
	// Add handlers for other gateways: /webhooks/coinbase, etc.

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Payment processor is running"))
	})

	// --- Find an available port ---
	actualWebhookPort := utils.FindAvailablePort(uint64(pp.config.WebhookPort))
	if actualWebhookPort != uint64(pp.config.WebhookPort) {
		log.Printf("[PaymentProcessor] Webhook Port %d in use, using %d instead.", pp.config.WebhookPort, actualWebhookPort)
	}

	// Create the server
	pp.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", actualWebhookPort),
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting payment processor webhook server on port %d", pp.config.WebhookPort)
		if err := pp.server.ListenAndServe(); err != nil && err != http.ErrServerClosed { // Use actualWebhookPort in log
			log.Printf("Payment processor server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the payment processor HTTP server
func (pp *PaymentProcessorImpl) Stop() error {
	if pp.server != nil {
		log.Println("Stopping payment processor webhook server...")
		return pp.server.Close()
	}
	return nil
}

// calculateTokenAmount determines how many native tokens to send based on payment.
// amountReceived: The amount of fiat/crypto received.
// currency: The symbol of the currency received (e.g., "usd", "eth").
// Returns the amount in the smallest unit of your token (like Wei for ETH).
func (pp *PaymentProcessorImpl) calculateTokenAmount(amountReceived float64, currency string) (*big.Int, error) {
	var rate float64
	currencyLower := strings.ToLower(currency)

	switch currencyLower {
	case "usd":
		if pp.config.USDPerToken <= 0 {
			return nil, fmt.Errorf("USD exchange rate not configured or invalid")
		}
		rate = 1.0 / pp.config.USDPerToken // Tokens per USD
	case "eth":
		if pp.config.ETHPerToken <= 0 {
			return nil, fmt.Errorf("ETH exchange rate not configured or invalid")
		}
		rate = 1.0 / pp.config.ETHPerToken // Tokens per ETH
	// Add cases for other supported currencies/cryptos
	default:
		return nil, fmt.Errorf("unsupported currency: %s", currency)
	}

	tokenValue := amountReceived * rate

	// Convert to the smallest unit using decimals
	// Example: If tokenValue is 1.5 NRN and decimals is 6, result is 1,500,000
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(pp.config.TokenDecimals)), nil))
	tokenAmountFloat := new(big.Float).Mul(new(big.Float).SetFloat64(tokenValue), multiplier)

	// Convert to big.Int (truncates any fractional part of the smallest unit)
	tokenAmountInt, _ := tokenAmountFloat.Int(nil) // Use accuracy check if needed

	if tokenAmountInt.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("calculated token amount is zero or negative")
	}

	log.Printf("Calculated: %.2f %s -> %s smallest units of %s",
		amountReceived, currency, tokenAmountInt.String(), pp.config.TokenSymbol)
	return tokenAmountInt, nil
}

// disburseTokens creates, signs, and broadcasts the KNIRVORACLE transaction.
func (pp *PaymentProcessorImpl) disburseTokens(recipientAddress string, amount *big.Int, paymentID string) (string, error) {
	// 1. Validate Recipient Address
	if !isValidKNIRVORACLEAddress(recipientAddress) {
		return "", fmt.Errorf("invalid recipient address: %s", recipientAddress)
	}

	// Convert big.Int to uint64 for KNIRVORACLE transaction
	// Note: This assumes the amount fits in uint64, which should be checked
	// 2. Construct the Transaction
	// Create a data field that includes the payment ID for tracking
	data := []byte(fmt.Sprintf("Payment: %s", paymentID))

	// Create the transaction using wallet.Transaction type
	tx := &Transaction{
		From:      pp.disbursementWallet.GetAddress(),
		To:        recipientAddress,
		Amount:    amount,
		Data:      data,
		Timestamp: time.Now(),
	}
	log.Printf("Constructed transaction: From %s To %s Amount %s", tx.From, tx.To, tx.Amount.String())

	// 3. Sign the Transaction
	err := pp.disbursementWallet.SignTransaction(tx)
	if err != nil {
		log.Printf("Error signing transaction for payment %s: %v", paymentID, err)
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// 4. Broadcast the Transaction
	txHash, err := pp.broadcastTransaction(tx)
	if err != nil {
		log.Printf("Error broadcasting transaction for payment %s: %v", paymentID, err)
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	log.Printf("Successfully broadcasted transaction for payment %s. TxHash: %s", paymentID, txHash)
	return txHash, nil
}

// broadcastTransaction sends the signed transaction to the KNIRVORACLE node.
func (pp *PaymentProcessorImpl) broadcastTransaction(tx *Transaction) (string, error) {
	// Marshal the transaction to JSON
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Determine the correct API endpoint on your node
	broadcastURL := pp.config.NodeRPC + "/transactions/new"

	req, err := http.NewRequest("POST", broadcastURL, bytes.NewBuffer(txJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create broadcast request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pp.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction to node %s: %w", broadcastURL, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("node returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response to get the transaction hash
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("Warning: Failed to parse node response body: %v. Body: %s", err, string(bodyBytes))
		// Decide if you want to return an error or just the raw body/success indication
		return fmt.Sprintf("Node response: %s", string(bodyBytes)), nil // Or return error
	}

	txHash, ok := result["transaction_hash"].(string) // Example key
	if !ok {
		log.Printf("Warning: Transaction hash not found in node response. Body: %s", string(bodyBytes))
		return fmt.Sprintf("Node response: %s", string(bodyBytes)), nil // Or return error
	}

	return txHash, nil
}

// handleStripeWebhook processes incoming events from Stripe.
func (pp *PaymentProcessorImpl) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Stripe webhook request")

	// 1. *** SECURITY: Verify the webhook signature ***
	// Use stripe.Webhook.ConstructEvent using r.Body and the 'Stripe-Signature' header
	// and your pp.config.StripeWebhookSecret. If verification fails, return 400 Bad Request.
	// This is CRITICAL to prevent fake requests.
	// payload, err := io.ReadAll(r.Body)
	// event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), pp.config.StripeWebhookSecret)
	// if err != nil {
	// 	 log.Printf("Webhook signature verification failed: %v", err)
	// 	 http.Error(w, "Webhook signature verification failed", http.StatusBadRequest)
	// 	 return
	// }

	// For demonstration without SDK verification (NOT FOR PRODUCTION):
	var event map[string]interface{} // Use proper Stripe event structs with SDK
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Cannot parse request body: %v", err)
		http.Error(w, "Cannot parse request body", http.StatusBadRequest)
		return
	}
	eventType, ok := event["type"].(string)
	if !ok {
		log.Printf("Event type missing in webhook payload")
		http.Error(w, "Event type missing", http.StatusBadRequest)
		return
	}
	log.Printf("Received Stripe event type: %s", eventType)

	// 2. Handle the relevant event type (e.g., 'charge.succeeded' or 'checkout.session.completed')
	if eventType == "charge.succeeded" { // Adjust based on your Stripe integration
		data, ok := event["data"].(map[string]interface{})
		if !ok {
			log.Printf("Event data missing in webhook payload")
			http.Error(w, "Event data missing", http.StatusBadRequest)
			return
		}

		object, ok := data["object"].(map[string]interface{})
		if !ok {
			log.Printf("Event object missing in webhook payload")
			http.Error(w, "Event object missing", http.StatusBadRequest)
			return
		}

		chargeID, ok := object["id"].(string)
		if !ok {
			log.Printf("Charge ID missing in webhook payload")
			http.Error(w, "Charge ID missing", http.StatusBadRequest)
			return
		}

		// Check if we've already processed this payment
		pp.mutex.RLock()
		txHash, processed := pp.processedPayments[chargeID]
		pp.mutex.RUnlock()

		if processed {
			log.Printf("Duplicate event received for charge %s. Already processed with tx: %s", chargeID, txHash)
			fmt.Fprintf(w, "Duplicate event")
			return
		}

		amountReceivedFloat, currency, err := extractPaymentDetails(object)
		if err != nil {
			log.Printf("Error extracting payment details: %v", err)
			http.Error(w, "Error extracting payment details", http.StatusBadRequest)
			return
		}

		// 3. *** IMPORTANT: Get the recipient's KNIRVORACLE address ***
		// You MUST collect this from the user during checkout and pass it to Stripe,
		// often via 'metadata' in the PaymentIntent or Checkout Session.
		metadata, okMeta := object["metadata"].(map[string]interface{})
		recipientAddress, okAddr := "", false
		if okMeta {
			addr, ok := metadata["KNIRVORACLE_address"].(string)
			if ok && addr != "" {
				recipientAddress = addr
				okAddr = true
			}
		}

		if !okAddr {
			log.Printf("ERROR: KNIRVORACLE_address missing in metadata for charge %s", chargeID)
			// Respond to Stripe but handle internally (e.g., notify admin, maybe refund)
			fmt.Fprintf(w, "Error: Missing recipient address metadata")
			return
		}

		// 5. Calculate token amount
		tokenAmount, err := pp.calculateTokenAmount(amountReceivedFloat, currency)
		if err != nil {
			log.Printf("Error calculating token amount for charge %s: %v", chargeID, err)
			// Respond to Stripe but handle internally
			fmt.Fprintf(w, "Error: Cannot calculate token amount")
			return
		}

		// 6. Disburse tokens
		txHash, err = pp.disburseTokens(recipientAddress, tokenAmount, chargeID)
		if err != nil {
			log.Printf("CRITICAL ERROR: Failed to disburse tokens for charge %s: %v", chargeID, err)
			// Respond to Stripe but handle internally (ALERTING/RETRY/REFUND NEEDED)
			fmt.Fprintf(w, "Error: Token disbursement failed")
			return
		}

		// 7. Mark payment as processed (Idempotency)
		pp.mutex.Lock()
		pp.processedPayments[chargeID] = txHash
		pp.mutex.Unlock()

		log.Printf("Successfully processed Stripe charge %s. Disbursed %s tokens to %s. Tx: %s",
			chargeID, tokenAmount.String(), recipientAddress, txHash)

		// 8. Acknowledge the webhook successfully
		fmt.Fprintf(w, "Success")

	} else {
		// Handle other event types or ignore
		log.Printf("Ignoring unhandled Stripe event type: %s", eventType)
		fmt.Fprintf(w, "Unhandled event type")
	}
}

// Helper function to extract payment details from Stripe object
func extractPaymentDetails(object map[string]interface{}) (float64, string, error) {
	amountReceived, ok := object["amount"].(float64)
	if !ok {
		// Try as json.Number
		if amountStr, ok := object["amount"].(json.Number); ok {
			var err error
			amountReceived, err = amountStr.Float64()
			if err != nil {
				return 0, "", fmt.Errorf("invalid amount format: %v", err)
			}
		} else {
			return 0, "", fmt.Errorf("amount missing or invalid format")
		}
	}

	// Stripe amounts are in cents, convert to dollars
	amountReceivedFloat := amountReceived / 100.0

	currency, ok := object["currency"].(string)
	if !ok {
		return 0, "", fmt.Errorf("currency missing")
	}

	return amountReceivedFloat, currency, nil
}

// isValidKNIRVORACLEAddress checks if an address is a valid KNIRVORACLE address
func isValidKNIRVORACLEAddress(addr string) bool {
	// Implement your specific address validation logic here.
	// Check length, prefix, checksum, etc.
	if !strings.HasPrefix(addr, utils.ADDRESS_PREFIX) {
		return false
	}
	return len(addr) > 10 // Basic validation
}

// LoadPaymentProcessorConfig loads the payment processor configuration from environment variables
func LoadPaymentProcessorConfig(cfg *config.PaymentProcessorConfig) {
	// Override config with environment variables if provided
	if nodeRPC := os.Getenv("KNIRVORACLE_NODE_RPC"); nodeRPC != "" {
		cfg.NodeRPC = nodeRPC
	}

	if stripeSecret := os.Getenv("STRIPE_WEBHOOK_SECRET"); stripeSecret != "" {
		cfg.StripeWebhookSecret = stripeSecret
	}

	if stripeKey := os.Getenv("STRIPE_SECRET_KEY"); stripeKey != "" {
		cfg.StripeSecretKey = stripeKey
	}

	if coinbaseKey := os.Getenv("COINBASE_API_KEY"); coinbaseKey != "" {
		cfg.CoinbaseAPIKey = coinbaseKey
	}

	if coinbaseSecret := os.Getenv("COINBASE_WEBHOOK_SECRET"); coinbaseSecret != "" {
		cfg.CoinbaseWebhookSecret = coinbaseSecret
	}
}
