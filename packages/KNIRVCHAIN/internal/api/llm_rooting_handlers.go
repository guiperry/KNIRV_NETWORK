package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
	"KNIRVCHAIN/internal/blockchain"
	pb "KNIRVCHAIN/internal/protocol/proto"
)

// LLMRootingAPI provides API endpoints for LLM rooting functionality
type LLMRootingAPI struct {
	blockchain *blockchain.BlockchainStruct
}

// NewLLMRootingAPI creates a new LLM rooting API instance
func NewLLMRootingAPI(bc *blockchain.BlockchainStruct) *LLMRootingAPI {
	return &LLMRootingAPI{
		blockchain: bc,
	}
}

// SetupRoutes sets up the LLM rooting API routes
func (api *LLMRootingAPI) SetupRoutes(router *mux.Router) {
	// LLM Rooting endpoints
	router.HandleFunc("/api/v1/root", api.handleSubmitLLMRooting).Methods("POST")
	router.HandleFunc("/api/v1/chain/latest", api.handleGetLatestBlock).Methods("GET")
	router.HandleFunc("/api/v1/resolve", api.handleResolveCMU).Methods("GET")
	router.HandleFunc("/api/v1/model/{modelHash}", api.handleGetModelHistory).Methods("GET")
}

// handleSubmitLLMRooting handles POST /api/v1/root
func (api *LLMRootingAPI) handleSubmitLLMRooting(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelName   string `json:"model_name"`
		ModelOwner  string `json:"model_owner"`
		APIEndpoint string `json:"api_endpoint"`
		MetadataCID string `json:"metadata_cid"`
		Signature   string `json:"signature"`
		Fee         uint64 `json:"fee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ModelName == "" || req.ModelOwner == "" || req.APIEndpoint == "" || req.MetadataCID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create LLM rooting data
	llmData := blockchain.LLMRootingData{
		ModelName:   req.ModelName,
		ModelOwner:  req.ModelOwner,
		APIEndpoint: req.APIEndpoint,
		MetadataCID: req.MetadataCID,
	}

	// Create transaction
	tx, err := blockchain.NewLLMRootingTransaction(req.ModelOwner, llmData, req.Fee)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Set signature
	tx.PublicKey = req.Signature // This should be the public key, not signature
	tx.Signature = []byte(req.Signature)

	// Add to transaction pool
	if err := api.blockchain.AddTransactionToTransactionPool(tx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add transaction: %v", err), http.StatusBadRequest)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":        "success",
		"transaction_id": tx.TransactionHash,
		"cmu":           llmData.CMU,
		"message":       "LLM rooting transaction submitted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetLatestBlock handles GET /api/v1/chain/latest
func (api *LLMRootingAPI) handleGetLatestBlock(w http.ResponseWriter, r *http.Request) {
	block, err := api.blockchain.GetLastBlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get latest block: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(block)
}

// handleResolveCMU handles GET /api/v1/resolve?cmu=<cmu>
func (api *LLMRootingAPI) handleResolveCMU(w http.ResponseWriter, r *http.Request) {
	cmu := r.URL.Query().Get("cmu")
	if cmu == "" {
		http.Error(w, "Missing CMU parameter", http.StatusBadRequest)
		return
	}

	// Validate CMU format
	if !strings.HasPrefix(cmu, "knirv://") {
		http.Error(w, "Invalid CMU format", http.StatusBadRequest)
		return
	}

	endpoint, err := api.blockchain.ResolveCMU(cmu)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve CMU: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"cmu":      cmu,
		"endpoint": endpoint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetModelHistory handles GET /api/v1/model/{modelHash}
func (api *LLMRootingAPI) handleGetModelHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelHash := vars["modelHash"]

	if modelHash == "" {
		http.Error(w, "Missing model hash", http.StatusBadRequest)
		return
	}

	transactions, err := api.blockchain.GetLLMTransactionsByModelHash(modelHash)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get model history: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert transactions to response format
	var history []map[string]interface{}
	for _, tx := range transactions {
		var protoData pb.LLMRootingDataProto
		if err := proto.Unmarshal(tx.Data, &protoData); err != nil {
			continue // Skip invalid transactions
		}

		history = append(history, map[string]interface{}{
			"transaction_id": tx.TransactionHash,
			"timestamp":      tx.Timestamp,
			"model_name":     protoData.ModelName,
			"model_owner":    protoData.ModelOwner,
			"api_endpoint":   protoData.ApiEndpoint,
			"metadata_cid":   protoData.MetadataCid,
			"cmu":            protoData.Cmu,
			"block_hash":     tx.BlockHash,
		})
	}

	response := map[string]interface{}{
		"model_hash":   modelHash,
		"transactions": history,
		"count":        len(history),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}