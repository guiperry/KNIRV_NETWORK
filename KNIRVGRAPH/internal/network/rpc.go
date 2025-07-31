package network

import (
	"blockchain-app/internal/nrv"
	"blockchain-app/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type RPCServer struct {
	graphchain GraphChainInterface
	nrvSystem  *nrv.NRVSystem
	logger     *zap.Logger
	server     *http.Server
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

func NewRPCServer(gc GraphChainInterface, logger *zap.Logger, port int) *RPCServer {
	return NewRPCServerWithNRV(gc, nil, logger, port)
}

func NewRPCServerWithNRV(gc GraphChainInterface, nrvSys *nrv.NRVSystem, logger *zap.Logger, port int) *RPCServer {
	router := mux.NewRouter()

	rpc := &RPCServer{
		graphchain: gc,
		nrvSystem:  nrvSys,
		logger:     logger,
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
	router.HandleFunc("/node/{nodeID}", rpc.getNode).Methods("GET")
	router.HandleFunc("/edge/{edgeID}", rpc.getEdge).Methods("GET")
	router.HandleFunc("/graph/heads", rpc.getHeads).Methods("GET")
	router.HandleFunc("/graph/neighbors/{nodeID}", rpc.getNeighbors).Methods("GET")
	router.HandleFunc("/graph/path/{from}/{to}", rpc.getPath).Methods("GET")
	router.HandleFunc("/graph/traverse", rpc.traverseGraph).Methods("POST")
	router.HandleFunc("/height", rpc.getHeight).Methods("GET")
	router.HandleFunc("/account/{address}", rpc.getAccount).Methods("GET")
	router.HandleFunc("/transaction", rpc.submitGraphTransaction).Methods("POST")
	router.HandleFunc("/node", rpc.createNode).Methods("POST")
	router.HandleFunc("/edge", rpc.createEdge).Methods("POST")

	// Register NRV routes
	if rpc.nrvSystem != nil {
		router.HandleFunc("/nrv/vectors", rpc.getAllVectors).Methods("GET")
		router.HandleFunc("/nrv/vectors", rpc.createVector).Methods("POST")
		router.HandleFunc("/nrv/vectors/resolve/{targetHash}", rpc.resolveTarget).Methods("GET")
		router.HandleFunc("/nrv/errors", rpc.getAllErrors).Methods("GET")
		router.HandleFunc("/nrv/errors", rpc.createError).Methods("POST")
		router.HandleFunc("/nrv/skills", rpc.getAllSkills).Methods("GET")
		router.HandleFunc("/nrv/skills", rpc.createSkill).Methods("POST")
		router.HandleFunc("/nrv/skills/for-error/{errorType}", rpc.getSkillsForError).Methods("GET")
	}

	rpc.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return rpc
}

func (rpc *RPCServer) Start(ctx context.Context) error {
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
	json.NewEncoder(w).Encode(node)
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
	json.NewEncoder(w).Encode(edge)
}

func (rpc *RPCServer) getHeads(w http.ResponseWriter, r *http.Request) {
	heads := rpc.graphchain.GetHeads()

	response := map[string][]string{"heads": heads}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(response)
}

func (rpc *RPCServer) createEdge(w http.ResponseWriter, r *http.Request) {
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
