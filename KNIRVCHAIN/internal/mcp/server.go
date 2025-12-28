package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/knirvchain/internal/blockchain"
	"github.com/knirvchain/internal/bridge"
	"github.com/knirvchain/internal/embedding"
	"github.com/knirvchain/internal/storage"
	"github.com/knirvchain/internal/wallet"
	"github.com/knirvchain/pkg/glb"
)

type MCPServer struct {
	chain    *blockchain.ChainNode
	wallet   *wallet.NRNWallet
	embedder embedding.Embedder
	encoder  *glb.Encoder
	router   *mux.Router
	bridge   *bridge.KNIRVGraphBridge // Optional bridge to KNIRVGRAPH
	logger   *log.Logger
}

// NewMCPServer creates a new MCP server with injected dependencies
func NewMCPServer(nodeID string, chain *blockchain.ChainNode, wallet *wallet.NRNWallet, embedder embedding.Embedder) (*MCPServer, error) {
	return NewMCPServerWithBridge(nodeID, chain, wallet, embedder, nil)
}

// NewMCPServerWithBridge creates a new MCP server with optional KNIRVGRAPH bridge
func NewMCPServerWithBridge(nodeID string, chain *blockchain.ChainNode, wallet *wallet.NRNWallet, embedder embedding.Embedder, graphBridge *bridge.KNIRVGraphBridge) (*MCPServer, error) {
	server := &MCPServer{
		chain:    chain,
		wallet:   wallet,
		embedder: embedder,
		encoder:  glb.NewEncoder(),
		router:   mux.NewRouter(),
		bridge:   graphBridge,
		logger:   log.New(log.Writer(), "[MCP] ", log.LstdFlags),
	}

	server.registerRoutes()
	return server, nil
}

// NewMCPServerWithDefaults creates a new MCP server with default initialization (for backward compatibility)
func NewMCPServerWithDefaults(nodeURL, walletKey string, stor storage.Storage) (*MCPServer, error) {
	chain, err := blockchain.NewChainNode(nodeURL, stor)
	if err != nil {
		return nil, err
	}

	wallet, err := wallet.NewNRNWallet(walletKey)
	if err != nil {
		return nil, err
	}

	// Create embedder with storage for vocabulary persistence
	embedder, err := embedding.NewTFIDFEmbedder(stor, 768)
	if err != nil {
		return nil, err
	}

	return NewMCPServer(nodeURL, chain, wallet, embedder)
}

func (s *MCPServer) registerRoutes() {
	s.router.HandleFunc("/tools/store_memory", s.handleStoreMemory).Methods("POST")
	s.router.HandleFunc("/tools/retrieve_memory", s.handleRetrieveMemory).Methods("POST")
	s.router.HandleFunc("/tools/query_balance", s.handleQueryBalance).Methods("GET")
	s.router.HandleFunc("/tools/estimate_cost", s.handleEstimateCost).Methods("POST")
}

type StoreMemoryRequest struct {
	Content    string                    `json:"content"`
	MemoryType blockchain.MemoryCategory `json:"memory_type"`
	Tags       []string                  `json:"tags,omitempty"`
}

type StoreMemoryResponse struct {
	BlockID string `json:"block_id"`
	Cost    uint64 `json:"cost"`
	TxHash  string `json:"tx_hash"`
	Status  string `json:"status"`
}

func (s *MCPServer) handleStoreMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req StoreMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Estimate and verify cost
	estimatedCost := s.estimateStorageCost(len(req.Content), req.MemoryType)

	hasBalance, err := s.wallet.HasBalance(ctx, estimatedCost)
	if err != nil || !hasBalance {
		http.Error(w, "Insufficient NRN balance", http.StatusPaymentRequired)
		return
	}

	// Generate semantic embedding
	embedding, err := s.generateEmbedding(ctx, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode as GLB
	metadata := glb.Metadata{
		Category:  string(req.MemoryType),
		Tags:      req.Tags,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	glbData, err := s.encoder.Encode(req.Content, metadata, embedding)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create block
	blockID := uuid.New()
	payloadHash := blockchain.SHA256Hash(glbData)

	block := blockchain.Block{
		BlockID:        blockID,
		Timestamp:      time.Now().Unix(),
		PayloadHash:    payloadHash,
		Data:           glbData,
		DataURI:        "",
		Category:       req.MemoryType,
		NRNCost:        estimatedCost,
		SemanticVector: embedding,
	}

	// Deduct tokens
	txHash, err := s.wallet.Spend(ctx, estimatedCost, "STORE:"+blockID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit to chain
	if err := s.chain.CommitBlock(ctx, &block); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Bridge to KNIRVGRAPH if applicable
	if s.shouldBridgeToGraph(req.MemoryType) {
		go s.bridgeToGraph(context.Background(), &block)
	}

	response := StoreMemoryResponse{
		BlockID: blockID.String(),
		Cost:    estimatedCost,
		TxHash:  txHash,
		Status:  "committed",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type RetrieveMemoryRequest struct {
	Query    string                     `json:"query"`
	Limit    int                        `json:"limit"`
	Category *blockchain.MemoryCategory `json:"category,omitempty"`
}

type Memory struct {
	BlockID    string                 `json:"block_id"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Similarity float64                `json:"similarity"`
	Timestamp  int64                  `json:"timestamp"`
}

type RetrieveMemoryResponse struct {
	Memories []Memory `json:"memories"`
	Cost     uint64   `json:"cost"`
}

func (s *MCPServer) handleRetrieveMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RetrieveMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Generate query embedding
	queryEmbedding, err := s.generateEmbedding(ctx, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate retrieval cost
	retrievalCost := s.calculateRetrievalCost(req.Limit)

	hasBalance, err := s.wallet.HasBalance(ctx, retrievalCost)
	if err != nil || !hasBalance {
		http.Error(w, "Insufficient NRN balance", http.StatusPaymentRequired)
		return
	}

	// Perform semantic search
	searchReq := blockchain.SearchRequest{
		Vector:   queryEmbedding,
		Limit:    req.Limit,
		Category: req.Category,
	}

	results, err := s.chain.SemanticSearch(ctx, searchReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Deduct tokens
	if _, err := s.wallet.Spend(ctx, retrievalCost,
		fmt.Sprintf("RETRIEVE:%d blocks", len(results))); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Decode GLB data
	memories := make([]Memory, 0, len(results))
	for _, block := range results {
		decoded, err := s.encoder.Decode(block.Data)
		if err != nil {
			continue // Skip corrupted blocks
		}

		memories = append(memories, Memory{
			BlockID:    block.BlockID.String(),
			Content:    decoded.Content,
			Metadata:   decoded.Metadata,
			Similarity: block.SimilarityScore,
			Timestamp:  block.Timestamp,
		})
	}

	response := RetrieveMemoryResponse{
		Memories: memories,
		Cost:     retrievalCost,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) handleQueryBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	balance, err := s.wallet.Balance(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"balance": balance,
		"unit":    "NRN",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type EstimateCostRequest struct {
	Operation string                 `json:"operation"`
	Params    map[string]interface{} `json:"params"`
}

type EstimateCostResponse struct {
	Operation string `json:"operation"`
	BaseCost  uint64 `json:"base_cost"`
	SizeCost  uint64 `json:"size_cost,omitempty"`
	Premium   uint64 `json:"premium,omitempty"`
	Total     uint64 `json:"total"`
}

func (s *MCPServer) handleEstimateCost(w http.ResponseWriter, r *http.Request) {
	var req EstimateCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var estimate EstimateCostResponse
	estimate.Operation = req.Operation

	switch req.Operation {
	case "store":
		size := int(req.Params["size"].(float64))
		category := blockchain.MemoryCategory(req.Params["category"].(string))

		estimate.BaseCost = 10
		estimate.SizeCost = uint64(size / 1024)
		estimate.Premium = s.getCategoryPremium(category)
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium

	case "retrieve":
		limit := int(req.Params["limit"].(float64))
		estimate.BaseCost = 5
		estimate.SizeCost = uint64(limit * 2)
		estimate.Total = estimate.BaseCost + estimate.SizeCost
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estimate)
}

func (s *MCPServer) getCategoryPremium(category blockchain.MemoryCategory) uint64 {
	premiums := map[blockchain.MemoryCategory]uint64{
		blockchain.CategoryError:   2,
		blockchain.CategoryContext: 5,
		blockchain.CategoryIdea:    8,
		blockchain.CategoryTask:    3,
		blockchain.CategoryGeneral: 5,
	}

	if premium, ok := premiums[category]; ok {
		return premium
	}
	return 5
}

func (s *MCPServer) estimateStorageCost(contentSize int, category blockchain.MemoryCategory) uint64 {
	baseCost := uint64(10)
	sizeKB := contentSize / 1024
	sizeCost := uint64(sizeKB)

	categoryPremium := map[blockchain.MemoryCategory]uint64{
		blockchain.CategoryError:   2,
		blockchain.CategoryContext: 5,
		blockchain.CategoryIdea:    8,
		blockchain.CategoryTask:    3,
		blockchain.CategoryGeneral: 5,
	}

	premium, ok := categoryPremium[category]
	if !ok {
		premium = 5
	}

	return baseCost + sizeCost + premium
}

func (s *MCPServer) calculateRetrievalCost(limit int) uint64 {
	return uint64(5 + (limit * 2))
}

func (s *MCPServer) shouldBridgeToGraph(category blockchain.MemoryCategory) bool {
	return category == blockchain.CategoryIdea ||
		category == blockchain.CategoryError ||
		category == blockchain.CategoryContext
}

func (s *MCPServer) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Use embedder to generate semantic vector
	return s.embedder.Generate(ctx, text)
}

func (s *MCPServer) bridgeToGraph(ctx context.Context, block *blockchain.Block) {
	// Skip if bridge is not configured
	if s.bridge == nil {
		return
	}

	// Send transaction to KNIRVGRAPH in background
	// We don't want to fail the main operation if bridging fails
	if err := s.bridge.SendTransaction(ctx, block); err != nil {
		s.logger.Printf("Failed to bridge block %s to KNIRVGRAPH: %v", block.BlockID, err)
		// Log error but don't propagate - bridging is optional
	} else {
		s.logger.Printf("Successfully bridged block %s to KNIRVGRAPH", block.BlockID)
	}
}

func (s *MCPServer) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
