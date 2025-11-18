package wallet

import (
	"KNIRVCHAIN/pkg/types"
	"KNIRVCHAIN/pkg/utils"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MCP Transaction type constants
const (
	TransactionTypeStandard              = "standard_txn"            // For standard NRN transfers
	TransactionTypeMCPRegisterCapability = "register_capability_txn" // For ResourceDescriptor, ToolDescriptor, PromptDescriptor, MemoryServiceDescriptor
	TransactionTypeMCPInvokeCapability   = "invoke_capability_txn"   // For logging usage of any capability, including plugin execution, sampling, memory ops
	TransactionTypeMCPUpdateCapability   = "update_capability_txn"   // For updating existing capabilities

	// Transaction status constants
	TXN_VERIFICATION_SUCCESS = "txn_verification_success"
	TXN_VERIFICATION_FAILURE = "txn_verification_failure"
)

type WalletServerImpl struct {
	port           uint64
	blockchainNode string
	Server         *http.Server
	portChan       chan uint64 // Re-add channel signaling
}

// New struct to represent incoming transaction requests
type TransactionRequest struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Value       uint64 `json:"value"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Signature   []byte `json:"signature"`
}

// MCPCapabilityRegistrationRequest represents a request to register a new MCP capability
type MCPCapabilityRegistrationRequest struct {
	PrivateKey     string                 `json:"private_key"`
	PublicKey      string                 `json:"public_key"`
	CapabilityType types.CapabilityType   `json:"capability_type"`
	Descriptor     map[string]interface{} `json:"descriptor"` // Generic descriptor that can hold any capability type
	Fee            uint64                 `json:"fee"`        // NRN token fee for registration
}

// MCPCapabilityInvocationRequest represents a request to invoke an MCP capability
type MCPCapabilityInvocationRequest struct {
	PrivateKey      string                 `json:"private_key"`
	PublicKey       string                 `json:"public_key"`
	CapabilityID    string                 `json:"capability_id"`
	InteractionType types.InteractionType  `json:"interaction_type"`
	InputData       map[string]interface{} `json:"input_data"`
	Fee             uint64                 `json:"fee"` // NRN token fee for invocation
}

func NewWalletServer(port uint64, blockchainNode string) *WalletServerImpl {
	return &WalletServerImpl{
		port:           port,
		blockchainNode: blockchainNode,
		portChan:       make(chan uint64, 1), // Re-add channel initialization
	}
}

// GetPortChan returns the port channel for testing purposes
func (ws *WalletServerImpl) GetPortChan() chan uint64 {
	return ws.portChan
}

// handleGetBalance handles GET /balance/{address} requests
func (ws *WalletServerImpl) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract address from URL path
	address := strings.TrimPrefix(r.URL.Path, "/balance/")
	if address == "" {
		http.Error(w, "Address parameter required", http.StatusBadRequest)
		return
	}

	// Query blockchain node for balance
	balanceURL := fmt.Sprintf("%s/balance/%s", ws.blockchainNode, address)
	resp, err := http.Get(balanceURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query balance: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, string(body), resp.StatusCode)
		return
	}

	// Forward the balance response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}

func (ws *WalletServerImpl) Start() func() {
	// 1. Create a dedicated ServeMux for this server instance
	mux := http.NewServeMux()

	// 2. Register handlers on the dedicated mux
	mux.HandleFunc("/transactions", ws.handlePostTransaction)
	mux.HandleFunc("/send_signed_txn", ws.handleSendSignedTransaction)
	mux.HandleFunc("/generate_wallet", ws.handleGenerateWallet)
	// Add a ping handler for health checks in tests
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "pong")
	})

	// Add balance endpoint
	mux.HandleFunc("/balance/", ws.handleGetBalance)

	// Add MCP capability endpoints
	mux.HandleFunc("/wallet/mcp/create_register_capability", ws.handleRegisterMCPCapability)
	mux.HandleFunc("/wallet/mcp/create_invoke_capability", ws.handleInvokeMCPCapability)

	// --- Find Available Port ---
	var listener net.Listener
	var err error
	actualPort := ws.port
	maxPort := ws.port + 10 // Try up to 10 ports

	for p := ws.port; p <= maxPort; p++ {
		addr := ":" + strconv.FormatUint(p, 10)
		// Try to create a TCP listener
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			actualPort = p // Found an available port
			log.Printf("Wallet server will listen on port %d", actualPort)
			break
		}
		log.Printf("Port %d busy, trying next...", p)
	}

	if listener == nil {
		// If no port was found after trying
		log.Fatalf("Wallet server failed to find an available port between %d and %d", ws.port, maxPort)
		// Note: log.Fatalf will exit the program. Consider returning an error if used outside main/tests.
		return func() {} // Return empty func on failure
	}
	// --- Port Found ---

	// 3. Configure the server with the found address and the dedicated mux
	ws.Server = &http.Server{
		Addr:    listener.Addr().String(), // Use the address from the listener
		Handler: mux,                      // Use the dedicated mux
	}

	// 4. Start the server in a goroutine using the listener we found
	go func() {
		log.Printf("Starting wallet server on %s...", ws.Server.Addr)
		// Use Serve with the listener we already opened
		if err := ws.Server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// Log fatal if Serve fails unexpectedly
			log.Printf("Wallet server Serve error: %v", err)
		}
		log.Println("Wallet server shut down.") // Log when shutdown completes
	}()

	// 5. Signal the actual port being used
	go func() {
		// Send the port in a goroutine to avoid potential deadlock
		log.Printf("Wallet server signaling port %d to main process", actualPort)
		select {
		case ws.portChan <- actualPort: // Send the port number back
			log.Printf("Wallet server successfully signaled port %d", actualPort)
		case <-time.After(1 * time.Second):
			log.Printf("Warning: Timeout sending port signal from wallet server")
		}
	}()

	// 6. Return the shutdown function
	// The test will use WaitForNode to confirm readiness externally
	return func() {
		log.Println("Shutting down wallet server...")
		// Give it a longer deadline to shutdown gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ws.Server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down wallet server: %v", err)
		} else {
			log.Println("Wallet server stopped gracefully.")
		}
		close(ws.portChan) // Close the channel on shutdown
	}
}

// --- Rest of the handler functions remain the same ---
// handlePostTransaction, handleSendSignedTransaction, handleGenerateWallet etc.

// Make sure the method checks are inside the specific POST handlers now
func (ws *WalletServerImpl) handlePostTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // Keep method check
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ws.handlePostTransactionPost(w, r) // Call the actual logic
}

func (ws *WalletServerImpl) handleSendSignedTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // Keep method check
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ws.handleSendSignedTransactionPost(w, r) // Call the actual logic
}

func (ws *WalletServerImpl) handleGenerateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // Keep method check
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// ... (rest of generate wallet logic) ...
	ws.handleGenerateWalletGet(w) // Assuming logic is moved here
}

// Extracted logic for GET /generate_wallet
func (ws *WalletServerImpl) handleGenerateWalletGet(w http.ResponseWriter) {
	// Generate a new wallet
	newWallet, err := NewWallet()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate wallet: %v", err), http.StatusInternalServerError)
		return
	}

	// Save wallet securely to file
	walletPath := filepath.Join("wallets", fmt.Sprintf("%s.wallet", newWallet.GetAddress()))
	if err := ws.saveWalletToFile(newWallet, walletPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save wallet: %v", err), http.StatusInternalServerError)
		return
	}

	// Create response (don't return private key)
	response := WalletResponse{
		Address:    newWallet.GetAddress(),
		PrivateKey: "", // Don't return private key
		PublicKey:  newWallet.GetPublicKeyHex(),
	}

	// Return wallet information
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Generated and saved new wallet at %s", walletPath)
}

// saveWalletToFile saves a wallet securely to disk
func (ws *WalletServerImpl) saveWalletToFile(wallet *WalletImpl, path string) error {
	// Encrypt private key before saving
	encryptedKey, err := utils.Encrypt([]byte(wallet.GetPrivateKeyHex()), utils.GenerateEncryptionKey())
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	walletData := struct {
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
	}{
		PrivateKey: string(encryptedKey),
		PublicKey:  wallet.GetPublicKeyHex(),
	}

	data, err := json.Marshal(walletData)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create wallet directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

// handlePostTransactionPost and handleSendSignedTransactionPost remain largely the same
func (ws *WalletServerImpl) handlePostTransactionPost(w http.ResponseWriter, r *http.Request) {
	var transactionRequest TransactionRequest
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() // Ensure body is closed
	err := json.NewDecoder(r.Body).Decode(&transactionRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Create the new transaction (UNSIGNED here)
	// NOTE: This endpoint seems redundant if you have /send_signed_txn.
	// It creates a transaction but doesn't sign it.
	// Consider if this endpoint is still needed.
	txn := &Transaction{
		From:      transactionRequest.FromAddress,
		To:        transactionRequest.ToAddress,
		Amount:    big.NewInt(int64(transactionRequest.Value)),
		Data:      []byte{},
		Timestamp: time.Now(),
	}

	// Send the new transaction to the blockchain
	blockchainAddress := fmt.Sprintf("%s/transactions", ws.blockchainNode) // Is this the correct blockchain endpoint?

	data, err := json.Marshal(txn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction: %v", err), http.StatusInternalServerError)
		return
	}
	resp, err := http.Post(blockchainAddress, "application/json", bytes.NewBuffer(data))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to post transaction to blockchain: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Error posting transaction %s", string(body)), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)

}

func (ws *WalletServerImpl) handleSendSignedTransactionPost(w http.ResponseWriter, r *http.Request) {
	var transactionRequest TransactionRequest
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() // Ensure body is closed
	err := json.NewDecoder(r.Body).Decode(&transactionRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// --- SIGNING SHOULD HAPPEN HERE ---
	// The request includes private/public keys. The wallet server's job
	// is typically to sign the transaction.
	wallet := NewWalletFromPrivateKeyHex(transactionRequest.PrivateKey)
	// Ensure public key matches
	if wallet.GetPublicKeyHex() != transactionRequest.PublicKey {
		http.Error(w, "Private key does not match public key", http.StatusBadRequest)
		return
	}

	// Create the transaction *without* signature/pubkey initially
	unsignedTxn := &Transaction{
		From:      transactionRequest.FromAddress,
		To:        transactionRequest.ToAddress,
		Amount:    big.NewInt(int64(transactionRequest.Value)),
		Data:      []byte{},
		Timestamp: time.Now(),
	}

	// Sign the transaction
	signedTxn, err := wallet.GetSignedTxn(*unsignedTxn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send the SIGNED transaction to the blockchain node ---
	// The blockchain node expects a fully formed transaction via its /transaction endpoint
	blockchainAddress := fmt.Sprintf("%s/transaction", ws.blockchainNode) // Use the correct endpoint on the blockchain node

	// Need to wrap the signed transaction correctly for the blockchain node's endpoint
	// Assuming the blockchain node's /transaction endpoint expects {"transaction": {...}}
	payload := map[string]interface{}{
		"transaction": signedTxn,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal signed transaction payload: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(blockchainAddress, "application/json", bytes.NewBuffer(data))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to post transaction to blockchain node: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Error posting transaction to blockchain node: %s", string(body)), resp.StatusCode)
		return
	}

	log.Printf("Successfully forwarded signed transaction %s to blockchain node", signedTxn.Hash)

	// Respond to the original client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(signedTxn) // Respond with the signed transaction
}

// WalletResponse represents the response for a generated wallet
type WalletResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// MCPTransactionResponse represents the response for MCP transactions
type MCPTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	FeePaid       uint64 `json:"fee_paid"`
}

// handleRegisterMCPCapability handles POST /wallet/mcp/create_register_capability
func (ws *WalletServerImpl) handleRegisterMCPCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPCapabilityRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate wallet keys
	wallet := NewWalletFromPrivateKeyHex(req.PrivateKey)
	if wallet.GetPublicKeyHex() != req.PublicKey {
		http.Error(w, "Private key does not match public key", http.StatusBadRequest)
		return
	}

	// Create MCP registration transaction
	data, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": req.Descriptor,
		"type":                 "register_capability_txn",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	txn := &Transaction{
		From:      wallet.GetAddress(),
		To:        "", // Not used for MCP
		Amount:    big.NewInt(0),
		Data:      data,
		Fee:       big.NewInt(int64(req.Fee)),
		Timestamp: time.Now(),
	}

	// Sign and send transaction
	signedTxn, err := wallet.GetSignedTxn(*txn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward to blockchain node
	if _, err := ws.sendTransactionToBlockchain(signedTxn); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(MCPTransactionResponse{
		TransactionID: signedTxn.Hash,
		Status:        "pending",
		FeePaid:       req.Fee,
	})
}

// handleInvokeMCPCapability handles POST /wallet/mcp/create_invoke_capability
func (ws *WalletServerImpl) handleInvokeMCPCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPCapabilityInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate wallet keys
	wallet := NewWalletFromPrivateKeyHex(req.PrivateKey)
	if wallet.GetPublicKeyHex() != req.PublicKey {
		http.Error(w, "Private key does not match public key", http.StatusBadRequest)
		return
	}

	// Create context record
	contextRecord := types.ContextRecord{
		ID:              GenerateUUID(),
		CapabilityID:    req.CapabilityID,
		InteractionType: req.InteractionType,
		Initiator:       wallet.GetAddress(),
		Timestamp:       time.Now().Unix(),
		Details:         req.InputData,
	}

	// Create MCP invocation transaction
	data, err := json.Marshal(map[string]interface{}{
		"contextRecord": contextRecord,
		"type":          "invoke_capability_txn",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	txn := &Transaction{
		From:      wallet.GetAddress(),
		To:        "", // Not used for MCP
		Amount:    big.NewInt(0),
		Data:      data,
		Fee:       big.NewInt(int64(req.Fee)),
		Timestamp: time.Now(),
	}

	// Sign and send transaction
	signedTxn, err := wallet.GetSignedTxn(*txn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward to blockchain node
	if _, err := ws.sendTransactionToBlockchain(signedTxn); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(MCPTransactionResponse{
		TransactionID: signedTxn.Hash,
		Status:        "pending",
		FeePaid:       req.Fee,
	})

	// Forward to blockchain node
	if _, err := ws.sendTransactionToBlockchain(signedTxn); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(MCPTransactionResponse{
		TransactionID: signedTxn.Hash,
		Status:        "pending",
		FeePaid:       req.Fee,
	})
}

// sendTransactionToBlockchain helper function to send transaction to blockchain node
func (ws *WalletServerImpl) sendTransactionToBlockchain(txn *Transaction) (*http.Response, error) {
	payload := map[string]interface{}{
		"transaction": txn,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	blockchainAddress := fmt.Sprintf("%s/transaction", ws.blockchainNode)
	resp, err := http.Post(blockchainAddress, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to post transaction: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("blockchain node rejected transaction: %s", string(body))
	}

	return resp, nil
}

// GenerateUUID generates a random UUID for context records
func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
