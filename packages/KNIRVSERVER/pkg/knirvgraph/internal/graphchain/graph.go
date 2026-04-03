package graphchain

import (
	"KNIRVGRAPH/internal/operationlog"
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"sync"
)

type GraphChain struct {
	mu         sync.RWMutex
	storage    storage.GraphStorage
	opLog      *operationlog.OperationLog
	nodes      map[string]*types.GraphNode
	edges      map[string]*types.Edge
	heads      []string // Current head nodes
	genesis    string   // Genesis node ID
	config     *GraphConfig
}

type GraphConfig struct {
	MaxNodesPerLevel    int     `json:"max_nodes_per_level"`
	MaxEdgesPerNode     int     `json:"max_edges_per_node"`
	AllowCycles         bool    `json:"allow_cycles"`
	ConsensusThreshold  float64 `json:"consensus_threshold"`
	TraversalDepthLimit int     `json:"traversal_depth_limit"`
	MaxHeads            int     `json:"max_heads"`
	WeightDecayFactor   float64 `json:"weight_decay_factor"`
}

func NewGraphChain(storage storage.GraphStorage) *GraphChain {
	opLog := operationlog.NewOperationLog(storage)
	gc := &GraphChain{
		storage: storage,
		opLog:   opLog,
		nodes:   make(map[string]*types.GraphNode),
		edges:   make(map[string]*types.Edge),
		heads:   []string{},
		config:  DefaultGraphConfig(),
	}

	// Load operation log state
	if err := gc.opLog.LoadState(); err != nil {
		// Log error but don't fail initialization
		fmt.Printf("Warning: failed to load operation log state: %v\n", err)
	}

	return gc
}

func DefaultGraphConfig() *GraphConfig {
	return &GraphConfig{
		MaxNodesPerLevel:    1000,
		MaxEdgesPerNode:     100,
		AllowCycles:         false,
		ConsensusThreshold:  0.67,
		TraversalDepthLimit: 50,
		MaxHeads:            10,
		WeightDecayFactor:   0.95,
	}
}

func (gc *GraphChain) AddNode(node *types.GraphNode) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if err := gc.validateGraphNode(node); err != nil {
		return fmt.Errorf("invalid graph node: %w", err)
	}

	if err := gc.executeNodeAddition(node); err != nil {
		return fmt.Errorf("failed to execute node addition: %w", err)
	}

	nodeData, err := node.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize node: %w", err)
	}

	if err := gc.storage.PutNode(node.ID, nodeData); err != nil {
		return fmt.Errorf("failed to store node: %w", err)
	}

	// Update in-memory cache
	gc.nodes[node.ID] = node

	// Update parent-child relationships
	if err := gc.updateRelationships(node); err != nil {
		return fmt.Errorf("failed to update relationships: %w", err)
	}

	// Update heads if necessary
	if err := gc.updateHeads(node); err != nil {
		return fmt.Errorf("failed to update heads: %w", err)
	}

	return nil
}

func (gc *GraphChain) AddEdge(edge *types.Edge) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if err := gc.validateEdge(edge); err != nil {
		return fmt.Errorf("invalid edge: %w", err)
	}

	edgeData, err := edge.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize edge: %w", err)
	}

	if err := gc.storage.PutEdge(edge.ID, edgeData); err != nil {
		return fmt.Errorf("failed to store edge: %w", err)
	}

	// Update in-memory cache
	gc.edges[edge.ID] = edge

	// Update node relationships
	if err := gc.updateNodeEdgeRelationships(edge); err != nil {
		return fmt.Errorf("failed to update node-edge relationships: %w", err)
	}

	return nil
}

func (gc *GraphChain) validateGraphNode(node *types.GraphNode) error {
	// Check for empty ID
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// Check if node already exists
	if _, exists := gc.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	// Validate parent nodes exist (using unsafe version since we already hold the lock)
	for _, parentID := range node.Parents {
		if _, err := gc.getNodeUnsafe(parentID); err != nil {
			return fmt.Errorf("parent node %s does not exist", parentID)
		}
	}

	// Check max edges per node
	if len(node.Parents) > gc.config.MaxEdgesPerNode {
		return fmt.Errorf("too many parent edges: %d > %d", len(node.Parents), gc.config.MaxEdgesPerNode)
	}

	// Check for cycles if not allowed
	if !gc.config.AllowCycles {
		if err := gc.checkForCycles(node); err != nil {
			return fmt.Errorf("cycle detected: %w", err)
		}
	}

	// Validate transactions in node
	for _, tx := range node.Data.Transactions {
		if !tx.Verify() {
			return fmt.Errorf("invalid transaction signature in node")
		}
	}

	return nil
}

func (gc *GraphChain) validateEdge(edge *types.Edge) error {
	// Check if edge already exists
	if _, exists := gc.edges[edge.ID]; exists {
		return fmt.Errorf("edge %s already exists", edge.ID)
	}

	// Validate from and to nodes exist (using unsafe version since we already hold the lock)
	if _, err := gc.getNodeUnsafe(edge.From); err != nil {
		return fmt.Errorf("from node %s does not exist", edge.From)
	}

	if _, err := gc.getNodeUnsafe(edge.To); err != nil {
		return fmt.Errorf("to node %s does not exist", edge.To)
	}

	// Validate weight
	if edge.Weight < 0 {
		return fmt.Errorf("edge weight cannot be negative")
	}

	return nil
}

// getNodeUnsafe retrieves a node without acquiring locks (assumes caller holds appropriate lock)
func (gc *GraphChain) getNodeUnsafe(nodeID string) (*types.GraphNode, error) {
	// Check in-memory cache first
	if node, exists := gc.nodes[nodeID]; exists {
		return node, nil
	}

	// Load from storage
	data, err := gc.storage.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	var node types.GraphNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	// Cache the node
	gc.nodes[nodeID] = &node

	return &node, nil
}

func (gc *GraphChain) checkForCycles(newNode *types.GraphNode) error {
	// Simple cycle detection using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node, err := gc.getNodeUnsafe(nodeID)
		if err != nil {
			return false
		}

		for _, childID := range node.Children {
			if !visited[childID] {
				if dfs(childID) {
					return true
				}
			} else if recStack[childID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	// Check if adding this node would create a cycle
	for _, parentID := range newNode.Parents {
		if dfs(parentID) {
			return fmt.Errorf("adding node would create a cycle")
		}
	}

	return nil
}

func (gc *GraphChain) executeNodeAddition(node *types.GraphNode) error {
	// Execute transactions in the node using the operation log
	for _, tx := range node.Data.Transactions {
		if err := gc.executeTransaction(&tx, node.ID); err != nil {
			return fmt.Errorf("failed to execute transaction %s: %w", tx.ID, err)
		}
	}

	// Apply state changes
	for _, stateChange := range node.Data.StateChanges {
		if err := gc.applyStateChange(&stateChange, node.ID); err != nil {
			return fmt.Errorf("failed to apply state change: %w", err)
		}
	}

	return nil
}

func (gc *GraphChain) executeTransaction(tx *types.Transaction, nodeID string) error {
	switch {
	case tx.To != "":
		// Transfer transaction - create audited operation
		amount := new(big.Int).SetUint64(tx.Amount)
		op := types.NewAuditedOperation(
			types.TransferOp,
			tx.From,
			tx.To,
			amount,
			nodeID,
			"",
			map[string]interface{}{
				"fee": tx.Fee,
				"data": string(tx.Data),
			},
		)
		return gc.opLog.ExecuteAndAudit(op)
	default:
		// Other transaction types - create generic operation
		op := types.NewAuditedOperation(
			types.NodeAddOp,
			"",
			"",
			nil,
			nodeID,
			"",
			map[string]interface{}{
				"transaction_id": tx.ID,
				"fee": tx.Fee,
				"data": string(tx.Data),
			},
		)
		return gc.opLog.ExecuteAndAudit(op)
	}
}

func (gc *GraphChain) applyStateChange(change *types.StateChange, nodeID string) error {
	// Create state change operation
	op := types.NewAuditedOperation(
		types.StateChangeOp,
		"",
		"",
		nil,
		nodeID,
		"",
		map[string]interface{}{
			"change_type": change.Type,
			"key": change.Key,
			"old_value": change.OldValue,
			"new_value": change.NewValue,
		},
	)
	return gc.opLog.ExecuteAndAudit(op)
}

func (gc *GraphChain) updateRelationships(node *types.GraphNode) error {
	// Update parent nodes to include this node as a child (using unsafe version since we already hold the lock)
	for _, parentID := range node.Parents {
		parent, err := gc.getNodeUnsafe(parentID)
		if err != nil {
			continue
		}

		parent.AddChild(node.ID)
		parentData, _ := parent.Serialize()
		gc.storage.PutNode(parent.ID, parentData)
		gc.nodes[parent.ID] = parent
	}

	// Store parent and child relationships separately for efficient queries
	if err := gc.storage.PutParents(node.ID, node.Parents); err != nil {
		return err
	}

	if err := gc.storage.PutChildren(node.ID, node.Children); err != nil {
		return err
	}

	return nil
}

func (gc *GraphChain) updateNodeEdgeRelationships(edge *types.Edge) error {
	// Update from node (using unsafe version since we already hold the lock)
	fromNode, err := gc.getNodeUnsafe(edge.From)
	if err != nil {
		return err
	}

	fromNode.AddChild(edge.To)
	fromNodeData, _ := fromNode.Serialize()
	gc.storage.PutNode(fromNode.ID, fromNodeData)
	gc.nodes[fromNode.ID] = fromNode

	// Update to node (using unsafe version since we already hold the lock)
	toNode, err := gc.getNodeUnsafe(edge.To)
	if err != nil {
		return err
	}

	toNode.AddParent(edge.From)
	toNodeData, _ := toNode.Serialize()
	gc.storage.PutNode(toNode.ID, toNodeData)
	gc.nodes[toNode.ID] = toNode

	return nil
}

func (gc *GraphChain) updateHeads(node *types.GraphNode) error {
	// If node has no children, it's a potential head
	if len(node.Children) == 0 {
		gc.heads = append(gc.heads, node.ID)
	}

	// Remove parents from heads if they now have children
	newHeads := []string{}
	for _, headID := range gc.heads {
		if headID == node.ID {
			newHeads = append(newHeads, headID)
			continue
		}

		isParent := false
		for _, parentID := range node.Parents {
			if parentID == headID {
				isParent = true
				break
			}
		}

		if !isParent {
			newHeads = append(newHeads, headID)
		}
	}

	gc.heads = newHeads

	// Limit number of heads
	if len(gc.heads) > gc.config.MaxHeads {
		// Sort by weight and keep top heads (using unsafe version since we already hold the lock)
		sort.Slice(gc.heads, func(i, j int) bool {
			nodeI, _ := gc.getNodeUnsafe(gc.heads[i])
			nodeJ, _ := gc.getNodeUnsafe(gc.heads[j])
			return nodeI.Weight > nodeJ.Weight
		})
		gc.heads = gc.heads[:gc.config.MaxHeads]
	}

	return gc.storage.UpdateHeads(gc.heads)
}

func (gc *GraphChain) GetNode(nodeID string) (*types.GraphNode, error) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	return gc.getNodeUnsafe(nodeID)
}

func (gc *GraphChain) GetEdge(edgeID string) (*types.Edge, error) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	// Check in-memory cache first
	if edge, exists := gc.edges[edgeID]; exists {
		return edge, nil
	}

	// Load from storage
	data, err := gc.storage.GetEdge(edgeID)
	if err != nil {
		return nil, fmt.Errorf("edge not found: %w", err)
	}

	var edge types.Edge
	if err := json.Unmarshal(data, &edge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edge: %w", err)
	}

	// Cache the edge
	gc.edges[edgeID] = &edge

	return &edge, nil
}

func (gc *GraphChain) GetHeads() []string {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	return append([]string{}, gc.heads...)
}

func (gc *GraphChain) GetNeighbors(nodeID string) ([]string, error) {
	node, err := gc.GetNode(nodeID)
	if err != nil {
		return nil, err
	}

	neighbors := make([]string, 0, len(node.Parents)+len(node.Children))
	neighbors = append(neighbors, node.Parents...)
	neighbors = append(neighbors, node.Children...)

	return neighbors, nil
}

func (gc *GraphChain) FindPath(fromID, toID string, maxDepth int) ([]string, error) {
	if maxDepth <= 0 {
		maxDepth = gc.config.TraversalDepthLimit
	}

	visited := make(map[string]bool)
	path := []string{}

	var dfs func(currentID string, depth int) bool
	dfs = func(currentID string, depth int) bool {
		if depth > maxDepth {
			return false
		}

		if currentID == toID {
			path = append(path, currentID)
			return true
		}

		if visited[currentID] {
			return false
		}

		visited[currentID] = true
		path = append(path, currentID)

		node, err := gc.GetNode(currentID)
		if err != nil {
			path = path[:len(path)-1]
			return false
		}

		// Try children first
		for _, childID := range node.Children {
			if dfs(childID, depth+1) {
				return true
			}
		}

		// Then try parents
		for _, parentID := range node.Parents {
			if dfs(parentID, depth+1) {
				return true
			}
		}

		path = path[:len(path)-1]
		return false
	}

	if dfs(fromID, 0) {
		return path, nil
	}

	return nil, fmt.Errorf("no path found from %s to %s", fromID, toID)
}

func (gc *GraphChain) GetState() *types.State {
	// Return a minimal state representation
	// In a full implementation, this would reconstruct state from operation log
	return &types.State{
		Height: gc.opLog.GetCurrentHeight(),
	}
}

func (gc *GraphChain) GetCurrentHeight() uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.opLog.GetCurrentHeight()
}
