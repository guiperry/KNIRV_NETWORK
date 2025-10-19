package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"KNIRVCHAIN_GO_Verifyer/config"
)

// registerContentRequest represents the incoming request
type registerContentRequest struct {
	URI      string `json:"uri"`
	Metadata string `json:"metadata"`
}

// registerContentResponse mirrors the structure of your custom blockchain's RPC response.
type registerContentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Helper Functions

// loadConfig loads the configuration from environment variables.
// makeRPCRequest makes an RPC call to the blockchain
func makeRPCRequest(endpoint string, body interface{}) ([]byte, error) {
	// TODO: Implement actual RPC request logic
	return json.Marshal(registerContentResponse{
		Success: true,
		Message: "Mock response",
	})
}

// hashEncryptionKey hashes the encryption key using SHA-256.
func hashMetadataKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}


// Handlers

// registerContentHandler handles the /register-content endpoint.
func registerContentHandler(w http.ResponseWriter, r *http.Request) {
	var req registerContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the encryption key
	metadataHash := hashMetadataKey(req.Metadata)

	// Construct the request body for your Go blockchain's register-content endpoint
	blockchainRequestBody := map[string]interface{}{
		"uri":        req.URI,
		"metadata":        metadataHash,
		"signer_address":    config.AppConfig.ServiceAddress, // Use the service's address from config
	}

	// Make the RPC response from your Go blockchain's register-content end-pointon the root chain
	responseBody, err := makeRPCRequest(config.AppConfig.RPCEndpoint, blockchainRequestBody)

	if err != nil {
		log.Printf("Error making RPC request: %v", err)
		http.Error(w, "Failed to register content on the blockchain", http.StatusInternalServerError)
		return
	}

	var rpcResponse registerContentResponse
	if err := json.Unmarshal(responseBody, &rpcResponse); err != nil {
		log.Printf("Error unmarshalling RPC response: %v, body: %s", err, string(responseBody))
		http.Error(w, "Failed to process blockchain response", http.StatusInternalServerError)
		return
	}

	if !rpcResponse.Success {
		log.Printf("Blockchain registration failed: %s", rpcResponse.Message)
		http.Error(w, "Blockchain registration failed: "+rpcResponse.Message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func registerContent() {
	// Load configuration
	config.LoadConfig()

	// Initialize router
	router := mux.NewRouter()

	// Define endpoints
	router.HandleFunc("/register-content", registerContentHandler).Methods("POST")

	// Start the server
	port := config.AppConfig.Port
	fmt.Printf("Server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}