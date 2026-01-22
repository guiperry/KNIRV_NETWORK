package walletserver

import (
	"KNIRVROUTER/internal/constants"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"KNIRVROUTER/internal/types"
	"KNIRVROUTER/internal/wallet"
)

type WalletServer struct {
	port           uint64
	blockchainNode string
	Server         *http.Server
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

func NewWalletServer(port uint64, blockchainNode string) *WalletServer {
	return &WalletServer{
		port:           port,
		blockchainNode: blockchainNode,
	}
}

func (ws *WalletServer) Start() func() {
	ws.Server = &http.Server{Addr: fmt.Sprintf(":%d", ws.port)}

	http.HandleFunc("/transactions", ws.handlePostTransaction)
	http.HandleFunc("/send_signed_txn", ws.handleSendSignedTransaction)
	http.HandleFunc("/generate_wallet", ws.handleGenerateWallet)

	go func() {
		if err := ws.Server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Failed to start wallet server: %v", err)
		}
	}()
	return func() {
		if err := ws.Server.Shutdown(nil); err != nil {
			panic(err) // failure/timeout shutting down the server gracefully
		}
	}
}

func (ws *WalletServer) handlePostTransaction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ws.handlePostTransactionPost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ws *WalletServer) handleSendSignedTransaction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ws.handleSendSignedTransactionPost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}

func (ws *WalletServer) handlePostTransactionPost(w http.ResponseWriter, r *http.Request) {
	var transactionRequest TransactionRequest
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&transactionRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Create the new transaction
	txn := types.NewTransaction(transactionRequest.FromAddress, transactionRequest.ToAddress, transactionRequest.Value, []byte{}, constants.ORIGIN_PUBLIC)

	// Send the new transaction to the blockchain
	blockchainAddress := fmt.Sprintf("%s/transactions", ws.blockchainNode)

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

func (ws *WalletServer) handleSendSignedTransactionPost(w http.ResponseWriter, r *http.Request) {
	var transactionRequest TransactionRequest
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&transactionRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create the new transaction
	txn := types.NewTransaction(transactionRequest.FromAddress, transactionRequest.ToAddress, transactionRequest.Value, []byte{}, constants.ORIGIN_PUBLIC)
	txn.Signature = transactionRequest.Signature // set the signature
	txn.PublicKey = transactionRequest.PublicKey // set the public key

	// Send the new transaction to the blockchain
	blockchainAddress := fmt.Sprintf("%s/send_txn", ws.blockchainNode)

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

// WalletResponse represents the response for a generated wallet
type WalletResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func (ws *WalletServer) handleGenerateWallet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Generate a new wallet
		newWallet, err := wallet.NewWallet()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate wallet: %v", err), http.StatusInternalServerError)
			return
		}

		// Create response
		response := WalletResponse{
			Address:    newWallet.GetAddress(),
			PrivateKey: newWallet.GetPrivateKeyHex(),
			PublicKey:  newWallet.GetPublicKeyHex(),
		}

		// Return wallet information
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
