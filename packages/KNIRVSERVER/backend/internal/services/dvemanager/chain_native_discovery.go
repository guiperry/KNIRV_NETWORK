package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"

	"github.com/tidwall/buntdb"
)

type ChainNodeRegistry interface {
	GetRegisteredNodes() ([]*ChainNodeInfo, error)
	GetNodeReputation(nodeID string) (int, error)
	CommitNodeRegistration(node *objects.DVENode, stakeAmount int64) (string, error)
	IsNodeRegistered(nodeID string) (bool, error)
}

type ChainNodeInfo struct {
	NodeID          string    `json:"node_id"`
	OwnerAddress    string    `json:"owner_address"`
	StakeAmount     int64     `json:"stake_amount"`
	TEEType         string    `json:"tee_type"`
	ReputationScore int       `json:"reputation_score"`
	Status          string    `json:"status"`
	RegisteredAt    time.Time `json:"registered_at"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	Capabilities    []string  `json:"capabilities"`
	Location        string    `json:"location"`
}

type ReputationEngine interface {
	CalculateReputation(nodeID string, performanceData *NodePerformanceData) (int, error)
	GetNodeScore(nodeID string) (int, error)
	UpdateNodeScore(nodeID string, score int) error
}

type NodePerformanceData struct {
	TasksCompleted   int       `json:"tasks_completed"`
	TasksFailed      int       `json:"tasks_failed"`
	AvgResponseTime  float64   `json:"avg_response_time"`
	UptimePercentage float64   `json:"uptime_percentage"`
	LastUpdated      time.Time `json:"last_updated"`
}

type ChainNativeNodeDiscovery struct {
	db                 *database.BuntDBManager
	chainRegistry      ChainNodeRegistry
	reputationEngine   ReputationEngine
	mu                 sync.RWMutex
	running            bool
	ctx                context.Context
	cancel             context.CancelFunc
	discoveryInterval  time.Duration
	reputationInterval time.Duration
	cache              map[string]*ChainNodeInfo
	cacheMu            sync.RWMutex
}

func NewChainNativeNodeDiscovery(db *database.BuntDBManager, registry ChainNodeRegistry, reputationEngine ReputationEngine) *ChainNativeNodeDiscovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &ChainNativeNodeDiscovery{
		db:                 db,
		chainRegistry:      registry,
		reputationEngine:   reputationEngine,
		ctx:                ctx,
		cancel:             cancel,
		discoveryInterval:  30 * time.Second,
		reputationInterval: 5 * time.Minute,
		cache:              make(map[string]*ChainNodeInfo),
	}
}

func (c *ChainNativeNodeDiscovery) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("chain native node discovery is already running")
	}
	c.running = true
	c.mu.Unlock()

	log.Println("Starting Chain-Native Node Discovery service...")

	if err := c.performInitialSync(); err != nil {
		log.Printf("Warning: Initial chain sync failed: %v", err)
	}

	go c.discoveryLoop()
	go c.reputationUpdateLoop()

	log.Println("Chain-Native Node Discovery service started successfully")
	return nil
}

func (c *ChainNativeNodeDiscovery) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	log.Println("Stopping Chain-Native Node Discovery service...")
	c.running = false
	c.cancel()

	if err := c.saveCacheToDatabase(); err != nil {
		log.Printf("Warning: Failed to save node cache: %v", err)
	}

	log.Println("Chain-Native Node Discovery service stopped")
	return nil
}

func (c *ChainNativeNodeDiscovery) performInitialSync() error {
	if c.chainRegistry == nil {
		return fmt.Errorf("chain registry not configured")
	}

	nodes, err := c.chainRegistry.GetRegisteredNodes()
	if err != nil {
		return fmt.Errorf("failed to fetch nodes from chain: %w", err)
	}

	c.cacheMu.Lock()
	for _, node := range nodes {
		c.cache[node.NodeID] = node
	}
	c.cacheMu.Unlock()

	log.Printf("Synced %d nodes from KNIRVCHAIN registry", len(nodes))
	return nil
}

func (c *ChainNativeNodeDiscovery) discoveryLoop() {
	ticker := time.NewTicker(c.discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.syncFromChain()
		}
	}
}

func (c *ChainNativeNodeDiscovery) syncFromChain() {
	if c.chainRegistry == nil {
		return
	}

	nodes, err := c.chainRegistry.GetRegisteredNodes()
	if err != nil {
		log.Printf("Warning: Failed to sync nodes from chain: %v", err)
		return
	}

	c.cacheMu.Lock()
	updatedCount := 0
	addedCount := 0

	for _, node := range nodes {
		if _, exists := c.cache[node.NodeID]; exists {
			updatedCount++
		} else {
			addedCount++
		}
		c.cache[node.NodeID] = node
	}

	c.cacheMu.Unlock()

	if addedCount > 0 || updatedCount > 0 {
		log.Printf("Chain sync: %d new nodes, %d updated", addedCount, updatedCount)
	}
}

func (c *ChainNativeNodeDiscovery) reputationUpdateLoop() {
	ticker := time.NewTicker(c.reputationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateNodeReputations()
		}
	}
}

func (c *ChainNativeNodeDiscovery) updateNodeReputations() {
	if c.reputationEngine == nil {
		return
	}

	c.cacheMu.RLock()
	nodeIDs := make([]string, 0, len(c.cache))
	for nodeID := range c.cache {
		nodeIDs = append(nodeIDs, nodeID)
	}
	c.cacheMu.RUnlock()

	for _, nodeID := range nodeIDs {
		performanceData, err := c.collectNodePerformance(nodeID)
		if err != nil {
			continue
		}

		newScore, err := c.reputationEngine.CalculateReputation(nodeID, performanceData)
		if err != nil {
			log.Printf("Warning: Failed to calculate reputation for node %s: %v", nodeID, err)
			continue
		}

		if err := c.reputationEngine.UpdateNodeScore(nodeID, newScore); err != nil {
			log.Printf("Warning: Failed to update reputation for node %s: %v", nodeID, err)
			continue
		}

		c.cacheMu.Lock()
		if node, exists := c.cache[nodeID]; exists {
			node.ReputationScore = newScore
		}
		c.cacheMu.Unlock()
	}
}

func (c *ChainNativeNodeDiscovery) collectNodePerformance(nodeID string) (*NodePerformanceData, error) {
	var tasksCompleted, tasksFailed int
	var totalResponseTime float64
	var responseCount int

	err := c.db.GetObjectsByPrefix("validation:tasks:", func(key string, value []byte) bool {
		var task objects.ValidationTask
		if err := json.Unmarshal(value, &task); err != nil {
			return true
		}

		if task.AssignedNodeID != nodeID {
			return true
		}

		if task.Status == "completed" {
			tasksCompleted++
			if task.CompletedAt != nil && task.StartedAt != nil && !task.StartedAt.IsZero() {
				totalResponseTime += task.CompletedAt.Sub(*task.StartedAt).Seconds()
				responseCount++
			}
		} else if task.Status == "failed" {
			tasksFailed++
		}

		return true
	})

	if err != nil {
		return nil, err
	}

	avgResponseTime := 0.0
	if responseCount > 0 {
		avgResponseTime = totalResponseTime / float64(responseCount)
	}

	var uptimePercentage float64 = 100.0
	node, err := c.getCachedNode(nodeID)
	if err == nil && node != nil {
		timeSinceHeartbeat := time.Since(node.LastHeartbeat)
		if timeSinceHeartbeat > 5*time.Minute {
			uptimePercentage = 0.0
		} else if timeSinceHeartbeat > 2*time.Minute {
			uptimePercentage = 50.0
		}
	}

	return &NodePerformanceData{
		TasksCompleted:   tasksCompleted,
		TasksFailed:      tasksFailed,
		AvgResponseTime:  avgResponseTime,
		UptimePercentage: uptimePercentage,
		LastUpdated:      time.Now(),
	}, nil
}

func (c *ChainNativeNodeDiscovery) getCachedNode(nodeID string) (*ChainNodeInfo, error) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	node, exists := c.cache[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found in cache")
	}
	return node, nil
}

func (c *ChainNativeNodeDiscovery) GetDiscoveredNodes() []*ChainNodeInfo {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	nodes := make([]*ChainNodeInfo, 0, len(c.cache))
	for _, node := range c.cache {
		nodes = append(nodes, node)
	}
	return nodes
}

func (c *ChainNativeNodeDiscovery) GetActiveNodes() []*ChainNodeInfo {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	nodes := make([]*ChainNodeInfo, 0)
	now := time.Now()
	for _, node := range c.cache {
		if node.Status == "active" && now.Sub(node.LastHeartbeat) < 5*time.Minute {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (c *ChainNativeNodeDiscovery) RegisterNodeFromChain(node *objects.DVENode, stakeAmount int64) (string, error) {
	if c.chainRegistry == nil {
		return "", fmt.Errorf("chain registry not configured")
	}

	txHash, err := c.chainRegistry.CommitNodeRegistration(node, stakeAmount)
	if err != nil {
		return "", fmt.Errorf("failed to register node on chain: %w", err)
	}

	chainNode := &ChainNodeInfo{
		NodeID:          node.ID,
		OwnerAddress:    node.PublicKey,
		StakeAmount:     stakeAmount,
		TEEType:         node.TEEType,
		ReputationScore: node.ReputationScore,
		Status:          "active",
		RegisteredAt:    time.Now(),
		LastHeartbeat:   time.Now(),
		Capabilities:    node.Capabilities,
		Location:        node.Location,
	}

	c.cacheMu.Lock()
	c.cache[node.ID] = chainNode
	c.cacheMu.Unlock()

	return txHash, nil
}

func (c *ChainNativeNodeDiscovery) IsNodeRegistered(nodeID string) bool {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	_, exists := c.cache[nodeID]
	return exists
}

func (c *ChainNativeNodeDiscovery) GetNodeByID(nodeID string) (*ChainNodeInfo, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	node, exists := c.cache[nodeID]
	return node, exists
}

func (c *ChainNativeNodeDiscovery) GetNodesByTEEType(teeType string) []*ChainNodeInfo {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	nodes := make([]*ChainNodeInfo, 0)
	for _, node := range c.cache {
		if node.TEEType == teeType && node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (c *ChainNativeNodeDiscovery) GetNodesByLocation(location string) []*ChainNodeInfo {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	nodes := make([]*ChainNodeInfo, 0)
	for _, node := range c.cache {
		if node.Location == location && node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (c *ChainNativeNodeDiscovery) saveCacheToDatabase() error {
	return c.db.Transaction(func(tx *buntdb.Tx) error {
		c.cacheMu.RLock()
		defer c.cacheMu.RUnlock()

		for nodeID, node := range c.cache {
			data, err := json.Marshal(node)
			if err != nil {
				continue
			}
			tx.Set(fmt.Sprintf("chain:node:%s", nodeID), string(data), nil)
		}
		return nil
	})
}

func (c *ChainNativeNodeDiscovery) loadCacheFromDatabase() error {
	return c.db.GetObjectsByPrefix("chain:node:", func(key string, value []byte) bool {
		var node ChainNodeInfo
		if err := json.Unmarshal(value, &node); err != nil {
			return true
		}

		c.cacheMu.Lock()
		c.cache[node.NodeID] = &node
		c.cacheMu.Unlock()

		return true
	})
}

func (c *ChainNativeNodeDiscovery) UpdateNodeHeartbeat(nodeID string) error {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if node, exists := c.cache[nodeID]; exists {
		node.LastHeartbeat = time.Now()
		return nil
	}
	return fmt.Errorf("node not found: %s", nodeID)
}

type DVEDefaultReputationEngine struct {
	db *database.BuntDBManager
}

func NewDVEDefaultReputationEngine(db *database.BuntDBManager) *DVEDefaultReputationEngine {
	return &DVEDefaultReputationEngine{db: db}
}

func (r *DVEDefaultReputationEngine) CalculateReputation(nodeID string, data *NodePerformanceData) (int, error) {
	if data == nil {
		return 100, nil
	}

	baseScore := 100.0

	totalTasks := data.TasksCompleted + data.TasksFailed
	if totalTasks > 0 {
		successRate := float64(data.TasksCompleted) / float64(totalTasks)
		baseScore = 50.0 + (successRate * 50.0)
	}

	if data.AvgResponseTime > 10.0 {
		baseScore -= min(30.0, data.AvgResponseTime-10.0)
	} else if data.AvgResponseTime < 1.0 {
		baseScore += 10.0
	}

	baseScore = (baseScore * 0.4) + (data.UptimePercentage * 0.6)

	return int(max(0, min(100, baseScore))), nil
}

func (r *DVEDefaultReputationEngine) GetNodeScore(nodeID string) (int, error) {
	var score int
	err := r.db.ViewTransaction(func(tx *buntdb.Tx) error {
		value, err := tx.Get(fmt.Sprintf("node:reputation:%s", nodeID))
		if err != nil {
			return err
		}
		fmt.Sscanf(value, "%d", &score)
		return nil
	})
	if err != nil {
		return 100, nil
	}
	return score, nil
}

func (r *DVEDefaultReputationEngine) UpdateNodeScore(nodeID string, score int) error {
	return r.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(fmt.Sprintf("node:reputation:%s", nodeID), fmt.Sprintf("%d", score), nil)
		return err
	})
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
