package network

import (
	"KNIRVGRAPH/internal/economics"
	"KNIRVGRAPH/internal/nrv"
	"KNIRVGRAPH/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type RPCServer struct {
	graphchain      GraphChainInterface
	nrvSystem       *nrv.NRVSystem
	nrnIntegration  *economics.NRNIntegration
	proofOfSolution *economics.ProofOfSolution
	app             AppInterface
	logger          *zap.Logger
	server          *http.Server
	port            int
}

type GraphChainInterface interface {
	GetCurrentHeight() uint64
	GetState() *types.State
	GetNode(nodeID string) (*types.GraphNode, error)
	GetEdge(edgeID string) (*types.Edge, error)
	GetHeads() []string
	GetNeighbors(nodeID string) ([]string, error)
	FindPath(fromID, toID string, maxDepth int) ([]string, error)
	AddNode(node *types.GraphNode) error
	AddEdge(edge *types.Edge) error
}

type AppInterface interface {
	IsNetworkPaused() bool
}

func NewRPCServer(gc GraphChainInterface, logger *zap.Logger, port int) *RPCServer {
	return NewRPCServerWithNRV(gc, nil, logger, port)
}

func NewRPCServerWithNRV(gc GraphChainInterface, nrvSys *nrv.NRVSystem, logger *zap.Logger, port int) *RPCServer {
	router := mux.NewRouter()

	rpc := &RPCServer{
		graphchain: gc,
		nrvSystem:  nrvSys,
		logger:     logger,
		port:       port,
	}

	// Enable CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Register graph routes
	router.HandleFunc("/node/{nodeID}", rpc.getNode).Methods("GET", "OPTIONS")
	router.HandleFunc("/edge/{edgeID}", rpc.getEdge).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/heads", rpc.getHeads).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/neighbors/{nodeID}", rpc.getNeighbors).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/path/{from}/{to}", rpc.getPath).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/traverse", rpc.traverseGraph).Methods("POST", "OPTIONS")
	router.HandleFunc("/height", rpc.getHeight).Methods("GET", "OPTIONS")
	router.HandleFunc("/account/{address}", rpc.getAccount).Methods("GET", "OPTIONS")
	router.HandleFunc("/transaction", rpc.submitGraphTransaction).Methods("POST", "OPTIONS")
	router.HandleFunc("/node", rpc.createNode).Methods("POST", "OPTIONS")
	router.HandleFunc("/edge", rpc.createEdge).Methods("POST", "OPTIONS")

	// Register NRV routes
	if rpc.nrvSystem != nil {
		router.HandleFunc("/nrv/vectors", rpc.getAllVectors).Methods("GET", "OPTIONS")
		router.HandleFunc("/nrv/vectors", rpc.createVector).Methods("POST", "OPTIONS")
		router.HandleFunc("/nrv/vectors/resolve/{targetHash}", rpc.resolveTarget).Methods("GET", "OPTIONS")
		router.HandleFunc("/nrv/errors", rpc.getAllErrors).Methods("GET", "OPTIONS")
		router.HandleFunc("/nrv/errors", rpc.createError).Methods("POST", "OPTIONS")
		router.HandleFunc("/nrv/skills", rpc.getAllSkills).Methods("GET", "OPTIONS")
		router.HandleFunc("/nrv/skills", rpc.createSkill).Methods("POST", "OPTIONS")
		router.HandleFunc("/nrv/skills/for-error/{errorType}", rpc.getSkillsForError).Methods("GET", "OPTIONS")
	}

	rpc.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second, // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return rpc
}

// NewRPCServerWithEconomics creates a new RPC server with economics integration
func NewRPCServerWithEconomics(gc GraphChainInterface, nrvSys *nrv.NRVSystem, nrnIntegration *economics.NRNIntegration, proofOfSolution *economics.ProofOfSolution, app AppInterface, logger *zap.Logger, port int) *RPCServer {
	router := mux.NewRouter()

	rpc := &RPCServer{
		graphchain:      gc,
		nrvSystem:       nrvSys,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		app:             app,
		logger:          logger,
	}

	// Enable CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Register graph routes
	router.HandleFunc("/node/{nodeID}", rpc.getNode).Methods("GET", "OPTIONS")
	router.HandleFunc("/edge/{edgeID}", rpc.getEdge).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/heads", rpc.getHeads).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/neighbors/{nodeID}", rpc.getNeighbors).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/path/{from}/{to}", rpc.getPath).Methods("GET", "OPTIONS")
	router.HandleFunc("/graph/traverse", rpc.traverseGraph).Methods("POST", "OPTIONS")
	router.HandleFunc("/height", rpc.getHeight).Methods("GET", "OPTIONS")
	router.HandleFunc("/account/{address}", rpc.getAccount).Methods("GET", "OPTIONS")
	router.HandleFunc("/transaction", rpc.submitGraphTransaction).Methods("POST", "OPTIONS")
	router.HandleFunc("/node", rpc.createNode).Methods("POST", "OPTIONS")
	router.HandleFunc("/edge", rpc.createEdge).Methods("POST", "OPTIONS")

	// Register existing NRV routes (from original implementation)
	router.HandleFunc("/nrv/resolve/{targetHash}", rpc.resolveTarget).Methods("GET", "OPTIONS")

	// Register economics routes
	router.HandleFunc("/economics/metrics", rpc.getEconomicMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/economics/skill/confirm", rpc.confirmSkill).Methods("POST", "OPTIONS")
	router.HandleFunc("/economics/rewards/distribute", rpc.distributeRewards).Methods("POST", "OPTIONS")
	router.HandleFunc("/economics/proof/solution", rpc.submitSolutionProof).Methods("POST", "OPTIONS")

	// Health check
	router.HandleFunc("/health", rpc.healthCheck).Methods("GET", "OPTIONS")

	rpc.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second, // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return rpc
}

func (rpc *RPCServer) Start(ctx context.Context) error {
	// Find an open port if the configured port is in use
	actualPort := findOpenPort(rpc.port, 100)
	if actualPort != rpc.port {
		rpc.port = actualPort
		rpc.server.Addr = fmt.Sprintf(":%d", actualPort)
		rpc.logger.Info("RPC port in use, using alternative port", zap.Int("port", actualPort))
	}
	rpc.logger.Info("Starting RPC server", zap.String("addr", rpc.server.Addr))

	go func() {
		if err := rpc.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rpc.logger.Error("RPC server error", zap.Error(err))
		}
	}()

	return nil
}

func (rpc *RPCServer) Stop(ctx context.Context) error {
	return rpc.server.Shutdown(ctx)
}

func (rpc *RPCServer) getNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeID"]

	node, err := rpc.graphchain.GetNode(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(node); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (rpc *RPCServer) getEdge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	edgeID := vars["edgeID"]

	edge, err := rpc.graphchain.GetEdge(edgeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(edge); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (rpc *RPCServer) getHeads(w http.ResponseWriter, r *http.Request) {
	heads := rpc.graphchain.GetHeads()

	response := map[string][]string{"heads": heads}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (rpc *RPCServer) getNeighbors(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeID"]

	neighbors, err := rpc.graphchain.GetNeighbors(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string][]string{"neighbors": neighbors}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) getPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fromID := vars["from"]
	toID := vars["to"]

	maxDepthStr := r.URL.Query().Get("max_depth")
	maxDepth := 50 // default
	if maxDepthStr != "" {
		if depth, err := strconv.Atoi(maxDepthStr); err == nil {
			maxDepth = depth
		}
	}

	path, err := rpc.graphchain.FindPath(fromID, toID, maxDepth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string][]string{"path": path}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) traverseGraph(w http.ResponseWriter, r *http.Request) {
	var query types.GraphQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "invalid query format", http.StatusBadRequest)
		return
	}

	// Execute graph traversal based on query type
	var result interface{}
	var err error

	switch query.Type {
	case types.FindPath:
		result, err = rpc.graphchain.FindPath(query.StartNode, query.EndNode, query.MaxDepth)
	case types.FindNeighbors:
		result, err = rpc.graphchain.GetNeighbors(query.StartNode)
	default:
		http.Error(w, "unsupported query type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"result": result})
}

func (rpc *RPCServer) getHeight(w http.ResponseWriter, r *http.Request) {
	height := rpc.graphchain.GetCurrentHeight()

	response := map[string]uint64{"height": height}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) getAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	account := rpc.graphchain.GetState().GetAccount(address)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (rpc *RPCServer) submitGraphTransaction(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, transactions not accepted", http.StatusServiceUnavailable)
		return
	}

	var tx types.GraphTransaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "invalid graph transaction format", http.StatusBadRequest)
		return
	}

	if !tx.Verify() {
		http.Error(w, "invalid graph transaction signature", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would be sent to the graph mempool
	response := map[string]string{"status": "accepted", "tx_id": tx.ID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) createNode(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, node creation not allowed", http.StatusServiceUnavailable)
		return
	}

	var node types.GraphNode
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "invalid node format", http.StatusBadRequest)
		return
	}

	if err := rpc.graphchain.AddNode(&node); err != nil {
		http.Error(w, fmt.Sprintf("failed to add node: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"status": "created", "node_id": node.ID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) createEdge(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, edge creation not allowed", http.StatusServiceUnavailable)
		return
	}

	var edge types.Edge
	if err := json.NewDecoder(r.Body).Decode(&edge); err != nil {
		http.Error(w, "invalid edge format", http.StatusBadRequest)
		return
	}

	if err := rpc.graphchain.AddEdge(&edge); err != nil {
		http.Error(w, fmt.Sprintf("failed to add edge: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"status": "created", "edge_id": edge.ID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NRV Handler Methods

func (rpc *RPCServer) getAllVectors(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	vectors := rpc.nrvSystem.GetAllVectors()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vectors)
}

func (rpc *RPCServer) createVector(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, vector creation not allowed", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		TargetHash  string                 `json:"target_hash"`
		Coordinates []float64              `json:"coordinates"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid vector format", http.StatusBadRequest)
		return
	}

	vector, err := rpc.nrvSystem.CreateVector(req.TargetHash, req.Coordinates, req.Metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vector)
}

func (rpc *RPCServer) resolveTarget(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	targetHash := vars["targetHash"]

	vectors, err := rpc.nrvSystem.ResolveTarget(targetHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vectors)
}

func (rpc *RPCServer) getAllErrors(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	errors := rpc.nrvSystem.GetAllErrorNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(errors)
}

func (rpc *RPCServer) createError(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, error creation not allowed", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ErrorType   string                 `json:"error_type"`
		Description string                 `json:"description"`
		Context     map[string]interface{} `json:"context"`
		Severity    int                    `json:"severity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid error format", http.StatusBadRequest)
		return
	}

	errorNode, err := rpc.nrvSystem.CreateErrorNode(req.ErrorType, req.Description, req.Context, req.Severity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(errorNode)
}

func (rpc *RPCServer) getAllSkills(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	skills := rpc.nrvSystem.GetAllSkillNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skills)
}

func (rpc *RPCServer) createSkill(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	// Check if network is paused
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		http.Error(w, "network is paused, skill creation not allowed", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		SkillType    string                 `json:"skill_type"`
		Capabilities []string               `json:"capabilities"`
		Requirements map[string]interface{} `json:"requirements"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid skill format", http.StatusBadRequest)
		return
	}

	skillNode, err := rpc.nrvSystem.CreateSkillNode(req.SkillType, req.Capabilities, req.Requirements)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skillNode)
}

func (rpc *RPCServer) getSkillsForError(w http.ResponseWriter, r *http.Request) {
	if rpc.nrvSystem == nil {
		http.Error(w, "NRV system not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	errorType := vars["errorType"]

	skills, err := rpc.nrvSystem.GetSkillsForErrorType(errorType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skills)
}

// Economics-related handlers

func (rpc *RPCServer) getEconomicMetrics(w http.ResponseWriter, r *http.Request) {
	if rpc.nrnIntegration == nil {
		http.Error(w, "NRN integration not available", http.StatusServiceUnavailable)
		return
	}

	metrics := rpc.nrnIntegration.GetEconomicMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (rpc *RPCServer) confirmSkill(w http.ResponseWriter, r *http.Request) {
	if rpc.nrnIntegration == nil {
		http.Error(w, "NRN integration not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		SkillID   string `json:"skill_id"`
		NRVID     string `json:"nrv_id"`
		CreatorID string `json:"creator_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Process skill confirmation for KNIRVCHAIN commitment
	if err := rpc.nrnIntegration.ProcessSkillConfirmation(req.SkillID, req.NRVID, req.CreatorID); err != nil {
		http.Error(w, fmt.Sprintf("skill confirmation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    "Skill confirmed for KNIRVCHAIN commitment",
		"skill_id":   req.SkillID,
		"nrv_id":     req.NRVID,
		"creator_id": req.CreatorID,
		"note":       "Skill will be committed to KNIRVCHAIN for invocation",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) distributeRewards(w http.ResponseWriter, r *http.Request) {
	if rpc.nrnIntegration == nil {
		http.Error(w, "NRN integration not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RecipientID string `json:"recipient_id"`
		Amount      string `json:"amount"`
		Reason      string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		http.Error(w, "invalid amount format", http.StatusBadRequest)
		return
	}

	// Distribute rewards
	if err := rpc.nrnIntegration.DistributeRewards(req.RecipientID, amount, req.Reason); err != nil {
		http.Error(w, fmt.Sprintf("reward distribution failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":      true,
		"message":      "Rewards distributed successfully",
		"recipient_id": req.RecipientID,
		"amount":       req.Amount,
		"reason":       req.Reason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) submitSolutionProof(w http.ResponseWriter, r *http.Request) {
	if rpc.proofOfSolution == nil {
		http.Error(w, "Proof-of-Solution not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ErrorNodeID     string  `json:"error_node_id"`
		SkillNodeID     string  `json:"skill_node_id"`
		SolverID        string  `json:"solver_id"`
		EfficiencyScore float64 `json:"efficiency_score"`
		QualityScore    float64 `json:"quality_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Calculate reward based on scores
	baseReward := big.NewInt(10000000) // 0.01 NRN base
	efficiencyBonus := big.NewInt(int64(req.EfficiencyScore * 100))
	qualityBonus := big.NewInt(int64(req.QualityScore * 100))

	totalReward := new(big.Int).Add(baseReward, efficiencyBonus)
	totalReward.Add(totalReward, qualityBonus)

	// Create resolution event
	event := economics.ResolutionEvent{
		ErrorNodeID:     req.ErrorNodeID,
		SkillNodeID:     req.SkillNodeID,
		SolverID:        req.SolverID,
		EfficiencyScore: req.EfficiencyScore,
		QualityScore:    req.QualityScore,
		RewardEarned:    totalReward,
	}

	// Process successful resolution
	if err := rpc.proofOfSolution.ProcessSuccessfulResolution(event); err != nil {
		http.Error(w, fmt.Sprintf("solution proof processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":       true,
		"message":       "Solution proof processed successfully",
		"error_node_id": req.ErrorNodeID,
		"skill_node_id": req.SkillNodeID,
		"solver_id":     req.SolverID,
		"reward_earned": totalReward.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler method aliases for backward compatibility with tests
func (rpc *RPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	rpc.healthCheck(w, r)
}

func (rpc *RPCServer) handleGetHeight(w http.ResponseWriter, r *http.Request) {
	rpc.getHeight(w, r)
}

func (rpc *RPCServer) handleGetNode(w http.ResponseWriter, r *http.Request) {
	rpc.getNode(w, r)
}

func (rpc *RPCServer) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	rpc.createNode(w, r)
}

func (rpc *RPCServer) handleGetHeads(w http.ResponseWriter, r *http.Request) {
	rpc.getHeads(w, r)
}

func (rpc *RPCServer) handleFindPath(w http.ResponseWriter, r *http.Request) {
	rpc.getPath(w, r)
}

// enableCORS wraps a handler with CORS headers
func (rpc *RPCServer) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rpc *RPCServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"services": map[string]interface{}{
			"graphchain": "running",
			"nrv_system": "running",
		},
	}

	if rpc.nrnIntegration != nil && rpc.nrnIntegration.IsEnabled() {
		status["services"].(map[string]interface{})["nrn_integration"] = "running"
	} else {
		status["services"].(map[string]interface{})["nrn_integration"] = "disabled"
	}

	if rpc.proofOfSolution != nil {
		status["services"].(map[string]interface{})["proof_of_solution"] = "running"
	} else {
		status["services"].(map[string]interface{})["proof_of_solution"] = "disabled"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// findOpenPort searches for an open port starting from preferredPort
func findOpenPort(preferredPort, maxAttempts int) int {
	for port := preferredPort; port < preferredPort+maxAttempts; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	// Fall back to the original port and let it fail naturally
	return preferredPort
}
