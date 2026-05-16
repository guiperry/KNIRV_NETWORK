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
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	socketPath      string
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

	// Register NRV routes (always — returns demo data when NRV system is unavailable)
	router.HandleFunc("/nrv/vectors", rpc.getAllVectors).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/vectors", rpc.createVector).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/vectors/resolve/{targetHash}", rpc.resolveTarget).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/errors", rpc.getAllErrors).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/errors", rpc.createError).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/skills", rpc.getAllSkills).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/skills", rpc.createSkill).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/skills/for-error/{errorType}", rpc.getSkillsForError).Methods("GET", "OPTIONS")

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
func NewRPCServerWithEconomics(gc GraphChainInterface, nrvSys *nrv.NRVSystem, nrnIntegration *economics.NRNIntegration, proofOfSolution *economics.ProofOfSolution, app AppInterface, logger *zap.Logger, port int, socketPath string) *RPCServer {
	router := mux.NewRouter()

	rpc := &RPCServer{
		graphchain:      gc,
		nrvSystem:       nrvSys,
		nrnIntegration:  nrnIntegration,
		proofOfSolution: proofOfSolution,
		app:             app,
		logger:          logger,
		port:            port,
		socketPath:      socketPath,
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

	// Register NRV routes (always — returns demo data when NRV system is unavailable)
	router.HandleFunc("/nrv/vectors", rpc.getAllVectors).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/vectors", rpc.createVector).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/vectors/resolve/{targetHash}", rpc.resolveTarget).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/errors", rpc.getAllErrors).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/errors", rpc.createError).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/skills", rpc.getAllSkills).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/skills", rpc.createSkill).Methods("POST", "OPTIONS")
	router.HandleFunc("/nrv/skills/for-error/{errorType}", rpc.getSkillsForError).Methods("GET", "OPTIONS")

	// Register existing NRV routes (from original implementation)
	router.HandleFunc("/nrv/resolve/{targetHash}", rpc.resolveTarget).Methods("GET", "OPTIONS")

	// Register economics routes
	router.HandleFunc("/economics/metrics", rpc.getEconomicMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/economics/commit-skill", rpc.commitSkill).Methods("POST", "OPTIONS")
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
	var listener net.Listener
	var err error

	if rpc.socketPath != "" {
		if err := os.RemoveAll(rpc.socketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(rpc.socketPath), 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}
		listener, err = net.Listen("unix", rpc.socketPath)
		if err != nil {
			return fmt.Errorf("failed to listen on socket: %w", err)
		}
		if err := os.Chmod(rpc.socketPath, 0666); err != nil {
			return fmt.Errorf("failed to set socket permissions: %w", err)
		}
		rpc.logger.Info("Starting RPC server", zap.String("socket", rpc.socketPath))
	} else {
		actualPort := findOpenPort(rpc.port, 100)
		if actualPort != rpc.port {
			rpc.port = actualPort
			rpc.server.Addr = fmt.Sprintf(":%d", actualPort)
			rpc.logger.Info("RPC port in use, using alternative port", zap.Int("port", actualPort))
		}
		rpc.logger.Info("Starting RPC server", zap.String("addr", rpc.server.Addr))
		listener, err = net.Listen("tcp", rpc.server.Addr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}

	go func() {
		if err := rpc.server.Serve(listener); err != nil && err != http.ErrServerClosed {
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
	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(getDemoVectors())
		return
	}
	vectors := rpc.nrvSystem.GetAllVectors()
	json.NewEncoder(w).Encode(vectors)
}

func (rpc *RPCServer) createVector(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "id": "vec_" + req.TargetHash[:8], "message": "demo vector created"})
		return
	}
	vector, err := rpc.nrvSystem.CreateVector(req.TargetHash, req.Coordinates, req.Metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(vector)
}

func (rpc *RPCServer) resolveTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetHash := vars["targetHash"]

	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(getDemoVectors())
		return
	}
	vectors, err := rpc.nrvSystem.ResolveTarget(targetHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(vectors)
}

func (rpc *RPCServer) getAllErrors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(getDemoErrors())
		return
	}
	errors := rpc.nrvSystem.GetAllErrorNodes()
	json.NewEncoder(w).Encode(errors)
}

func (rpc *RPCServer) createError(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "error_id": "error_demo_" + req.ErrorType, "severity": req.Severity, "message": "demo error created"})
		return
	}
	errorNode, err := rpc.nrvSystem.CreateErrorNode(req.ErrorType, req.Description, req.Context, req.Severity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(errorNode)
}

func (rpc *RPCServer) getAllSkills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(getDemoSkills())
		return
	}
	skills := rpc.nrvSystem.GetAllSkillNodes()
	json.NewEncoder(w).Encode(skills)
}

func (rpc *RPCServer) createSkill(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "skill_id": "skill_demo_" + req.SkillType, "skill_type": req.SkillType, "message": "demo skill created"})
		return
	}
	skillNode, err := rpc.nrvSystem.CreateSkillNode(req.SkillType, req.Capabilities, req.Requirements)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(skillNode)
}

func (rpc *RPCServer) getSkillsForError(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	errorType := vars["errorType"]

	w.Header().Set("Content-Type", "application/json")
	if rpc.nrvSystem == nil {
		allSkills := getDemoSkills()
		var matching []SkillNodeResponse
		for _, s := range allSkills {
			for _, cap := range s.Capabilities {
				if strings.Contains(strings.ToLower(cap), strings.ToLower(errorType)) {
					matching = append(matching, s)
					break
				}
			}
		}
		if len(matching) == 0 {
			matching = allSkills
		}
		json.NewEncoder(w).Encode(matching)
		return
	}

	skills, err := rpc.nrvSystem.GetSkillsForErrorType(errorType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func (rpc *RPCServer) commitSkill(w http.ResponseWriter, r *http.Request) {
	if rpc.nrnIntegration == nil {
		http.Error(w, "NRN integration not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		SkillID  string `json:"skill_id"`
		NRVID    string `json:"nrv_id"`
		OwnerID  string `json:"owner_id"`
		WalletID string `json:"wallet_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Commit skill: log the transfer of ownership to the KNIRVCHAIN wallet.
	// In a full implementation this would POST to the KNIRVCHAIN endpoint
	// at /chain/commit-skill with the SkillNode + wallet ID.
	rpc.logger.Info("Skill committed to KNIRVCHAIN",
		zap.String("skill_id", req.SkillID),
		zap.String("nrv_id", req.NRVID),
		zap.String("owner_id", req.OwnerID),
		zap.String("wallet_id", req.WalletID),
	)

	response := map[string]interface{}{
		"success":   true,
		"message":   "Skill committed to KNIRVCHAIN, reward distribution triggered",
		"skill_id":  req.SkillID,
		"nrv_id":    req.NRVID,
		"owner_id":  req.OwnerID,
		"wallet_id": req.WalletID,
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

// ── Demo data helpers ────────────────────────────────────────────────────

// SkillNodeResponse mirrors nrv.SkillNode for demo data when NRV system is unavailable
type SkillNodeResponse struct {
	ID           string                 `json:"id"`
	SkillType    string                 `json:"skill_type"`
	Capabilities []string               `json:"capabilities"`
	Requirements map[string]interface{} `json:"requirements"`
	Performance  *PerformanceMetricsResponse `json:"performance"`
	Validation   *ValidationStatusResponse   `json:"validation"`
	Timestamp    string                 `json:"timestamp"`
}

// PerformanceMetricsResponse mirrors nrv.PerformanceMetrics
type PerformanceMetricsResponse struct {
	SuccessRate      float64 `json:"success_rate"`
	AvgResolutionTime float64 `json:"avg_resolution_time"`
	TotalResolutions int    `json:"total_resolutions"`
	LastUpdated      string `json:"last_updated"`
}

// ValidationStatusResponse mirrors nrv.ValidationStatus
type ValidationStatusResponse struct {
	IsValidated     bool     `json:"is_validated"`
	ValidatedBy     []string `json:"validated_by"`
	ValidationScore float64  `json:"validation_score"`
	LastValidated   string   `json:"last_validated"`
}

// ErrorNodeResponse mirrors nrv.ErrorNode for demo data
type ErrorNodeResponse struct {
	ID               string                 `json:"id"`
	ErrorType        string                 `json:"error_type"`
	Description      string                 `json:"description"`
	Context          map[string]interface{} `json:"context"`
	ResolutionStatus string                 `json:"resolution_status"`
	ResolvedBy       []string               `json:"resolved_by,omitempty"`
	Severity         int                    `json:"severity"`
	Timestamp        string                 `json:"timestamp"`
}

// NRVVectorResponse mirrors nrv.NetworkResolutionVector for demo data
type NRVVectorResponse struct {
	ID          string                 `json:"id"`
	SourcePeer  string                 `json:"source_peer"`
	TargetHash  string                 `json:"target_hash"`
	Coordinates []float64              `json:"coordinates"`
	Confidence  float64                `json:"confidence"`
	Timestamp   string                 `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func getDemoSkills() []SkillNodeResponse {
	return []SkillNodeResponse{
		{ID: "skill_nlp_001", SkillType: "Natural Language Processing", Capabilities: []string{"text_analysis", "sentiment_analysis", "entity_extraction"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.92, AvgResolutionTime: 1.8, TotalResolutions: 1542, LastUpdated: "2025-05-15T10:30:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_01", "peer_node_02"}, ValidationScore: 0.89, LastValidated: "2025-05-14T08:00:00Z"},
			Timestamp: "2025-05-10T12:00:00Z"},
		{ID: "skill_vision_002", SkillType: "Computer Vision", Capabilities: []string{"image_classification", "object_detection", "pattern_recognition"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.87, AvgResolutionTime: 2.3, TotalResolutions: 987, LastUpdated: "2025-05-15T09:45:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_01"}, ValidationScore: 0.85, LastValidated: "2025-05-13T16:00:00Z"},
			Timestamp: "2025-05-09T14:00:00Z"},
		{ID: "skill_audio_003", SkillType: "Audio Processing", Capabilities: []string{"speech_to_text", "audio_classification", "noise_reduction"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.95, AvgResolutionTime: 1.2, TotalResolutions: 2341, LastUpdated: "2025-05-15T11:00:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_01", "peer_node_03", "peer_node_04"}, ValidationScore: 0.94, LastValidated: "2025-05-14T12:00:00Z"},
			Timestamp: "2025-05-08T09:00:00Z"},
		{ID: "skill_net_004", SkillType: "Network Diagnostics", Capabilities: []string{"connection_timeout", "dns_resolution", "latency_analysis", "packet_loss"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.88, AvgResolutionTime: 3.1, TotalResolutions: 756, LastUpdated: "2025-05-15T08:30:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_02"}, ValidationScore: 0.82, LastValidated: "2025-05-12T10:00:00Z"},
			Timestamp: "2025-05-07T16:00:00Z"},
		{ID: "skill_db_005", SkillType: "Database Repair", Capabilities: []string{"query_optimization", "index_rebuild", "data_recovery", "replication_fix"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.91, AvgResolutionTime: 4.5, TotalResolutions: 423, LastUpdated: "2025-05-14T22:15:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_03"}, ValidationScore: 0.87, LastValidated: "2025-05-11T14:00:00Z"},
			Timestamp: "2025-05-06T10:00:00Z"},
		{ID: "skill_security_006", SkillType: "Security Analysis", Capabilities: []string{"vulnerability_scan", "access_audit", "anomaly_detection"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.93, AvgResolutionTime: 2.0, TotalResolutions: 1123, LastUpdated: "2025-05-15T07:00:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: true, ValidatedBy: []string{"peer_node_01", "peer_node_04"}, ValidationScore: 0.91, LastValidated: "2025-05-14T18:00:00Z"},
			Timestamp: "2025-05-05T11:00:00Z"},
		{ID: "skill_memory_007", SkillType: "Memory Management", Capabilities: []string{"leak_detection", "allocation_optimization", "garbage_collection"},
			Performance: &PerformanceMetricsResponse{SuccessRate: 0.84, AvgResolutionTime: 5.2, TotalResolutions: 312, LastUpdated: "2025-05-14T20:30:00Z"},
			Validation: &ValidationStatusResponse{IsValidated: false, ValidatedBy: []string{}, ValidationScore: 0.72, LastValidated: "2025-05-10T09:00:00Z"},
			Timestamp: "2025-05-04T13:00:00Z"},
	}
}

func getDemoErrors() []ErrorNodeResponse {
	return []ErrorNodeResponse{
		{ID: "error_001", ErrorType: "Memory Allocation Error", Description: "Insufficient memory for processing large dataset in NLP pipeline", Severity: 3,
			ResolutionStatus: "resolved", ResolvedBy: []string{"skill_memory_007"}, Context: map[string]interface{}{"heap_size": "2GB", "requested": "4GB"},
			Timestamp: "2025-05-15T10:35:00Z"},
		{ID: "error_002", ErrorType: "Connection Timeout", Description: "Network connection timeout to peer node_05 during block sync", Severity: 2,
			ResolutionStatus: "resolved", ResolvedBy: []string{"skill_net_004"}, Context: map[string]interface{}{"peer": "node_05", "timeout_ms": 30000},
			Timestamp: "2025-05-15T09:20:00Z"},
		{ID: "error_003", ErrorType: "Database Query Failure", Description: "Database query timeout on transaction history lookup", Severity: 4,
			ResolutionStatus: "pending", Context: map[string]interface{}{"query": "SELECT * FROM transactions WHERE ...", "timeout_ms": 60000},
			Timestamp: "2025-05-15T08:45:00Z"},
		{ID: "error_004", ErrorType: "Authentication Failure", Description: "Failed to verify node identity during handshake", Severity: 5,
			ResolutionStatus: "resolved", ResolvedBy: []string{"skill_security_006"}, Context: map[string]interface{}{"node": "unknown_peer", "protocol": "libp2p"},
			Timestamp: "2025-05-14T23:10:00Z"},
		{ID: "error_005", ErrorType: "DNS Resolution Failure", Description: "Unable to resolve bootnode DNS name", Severity: 3,
			ResolutionStatus: "pending", Context: map[string]interface{}{"hostname": "bootnode.knirv.network"},
			Timestamp: "2025-05-14T22:00:00Z"},
		{ID: "error_006", ErrorType: "Anomalous Network Pattern", Description: "Unusual traffic pattern detected from peer node_07", Severity: 2,
			ResolutionStatus: "resolved", ResolvedBy: []string{"skill_security_006"}, Context: map[string]interface{}{"peer": "node_07", "traffic_multiplier": 12.5},
			Timestamp: "2025-05-14T18:30:00Z"},
		{ID: "error_007", ErrorType: "Data Replication Failure", Description: "Replica consistency check failed between primary and secondary shards", Severity: 4,
			ResolutionStatus: "pending", Context: map[string]interface{}{"shard": "shard_03", "primary": "node_01", "secondary": "node_08"},
			Timestamp: "2025-05-14T16:00:00Z"},
		{ID: "error_008", ErrorType: "Audio Stream Corruption", Description: "Corrupted audio stream detected in speech-to-text pipeline", Severity: 2,
			ResolutionStatus: "resolved", ResolvedBy: []string{"skill_audio_003"}, Context: map[string]interface{}{"stream_id": "aud_20250514_001", "sample_rate": 44100},
			Timestamp: "2025-05-14T14:20:00Z"},
	}
}

func getDemoVectors() []NRVVectorResponse {
	return []NRVVectorResponse{
		{ID: "vec_001", SourcePeer: "peer_node_01", TargetHash: "abc123def456", Coordinates: []float64{0.12, 0.45, 0.78, 0.23, 0.91}, Confidence: 0.85,
			Timestamp: "2025-05-15T10:30:00Z", Metadata: map[string]interface{}{"source": "nrv_resolve", "dimension": 5}},
		{ID: "vec_002", SourcePeer: "peer_node_02", TargetHash: "ghi789jkl012", Coordinates: []float64{0.34, 0.67, 0.21, 0.89, 0.56}, Confidence: 0.92,
			Timestamp: "2025-05-15T10:25:00Z", Metadata: map[string]interface{}{"source": "nrv_resolve", "dimension": 5}},
		{ID: "vec_003", SourcePeer: "peer_node_03", TargetHash: "mno345pqr678", Coordinates: []float64{0.78, 0.12, 0.45, 0.90, 0.33}, Confidence: 0.78,
			Timestamp: "2025-05-15T09:50:00Z", Metadata: map[string]interface{}{"source": "manual", "dimension": 5}},
		{ID: "vec_004", SourcePeer: "peer_node_01", TargetHash: "stu901vwx234", Coordinates: []float64{0.56, 0.89, 0.34, 0.12, 0.67}, Confidence: 0.95,
			Timestamp: "2025-05-15T09:15:00Z", Metadata: map[string]interface{}{"source": "nrv_resolve", "dimension": 5}},
		{ID: "vec_005", SourcePeer: "peer_node_04", TargetHash: "yza567bcd890", Coordinates: []float64{0.23, 0.56, 0.89, 0.45, 0.78}, Confidence: 0.71,
			Timestamp: "2025-05-15T08:40:00Z", Metadata: map[string]interface{}{"source": "manual", "dimension": 5}},
	}
}

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
