package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/models"
	"github.com/knirv/nexus-backend/pkg/p2p"
	"github.com/tidwall/buntdb"
)

// DVEManager manages DVE nodes and their operations
type DVEManager struct {
	db           *buntdb.DB
	p2pManager   *p2p.DVEP2PManager
	config       *config.Config
	nodeTracker  *NodeTracker
	loadBalancer *LoadBalancer
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// NodeTracker tracks the status and health of DVE nodes
type NodeTracker struct {
	nodes map[string]*models.DVENode
	mu    sync.RWMutex
}

// LoadBalancer handles task assignment and load balancing
type LoadBalancer struct {
	algorithm string // "round_robin", "reputation_based", "resource_based"
	mu        sync.RWMutex
}

// NewDVEManager creates a new DVE Manager instance
func NewDVEManager(db *buntdb.DB, p2pManager *p2p.DVEP2PManager, cfg *config.Config) (*DVEManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &DVEManager{
		db:         db,
		p2pManager: p2pManager,
		config:     cfg,
		nodeTracker: &NodeTracker{
			nodes: make(map[string]*models.DVENode),
		},
		loadBalancer: &LoadBalancer{
			algorithm: "reputation_based",
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Register P2P message handlers
	p2pManager.RegisterMessageHandler(p2p.MessageTypeNodeAnnouncement, manager)
	p2pManager.RegisterMessageHandler(p2p.MessageTypeNodeHeartbeat, manager)

	return manager, nil
}

// Start starts the DVE Manager service
func (dm *DVEManager) Start(ctx context.Context) error {
	log.Println("Starting DVE Manager service...")

	// Start periodic tasks
	go dm.monitorNodes()
	go dm.cleanupExpiredNodes()
	go dm.updateMetrics()

	// Load existing nodes from database
	if err := dm.loadNodesFromDB(); err != nil {
		log.Printf("Warning: Failed to load nodes from database: %v", err)
	}

	log.Println("DVE Manager service started successfully")
	return nil
}

// Stop stops the DVE Manager service
func (dm *DVEManager) Stop(ctx context.Context) error {
	log.Println("Stopping DVE Manager service...")
	dm.cancel()
	log.Println("DVE Manager service stopped")
	return nil
}

// HandleMessage implements the P2P MessageHandler interface
func (dm *DVEManager) HandleMessage(ctx context.Context, msg *models.P2PMessage) error {
	switch msg.Type {
	case p2p.MessageTypeNodeAnnouncement:
		return dm.handleNodeAnnouncement(msg)
	case p2p.MessageTypeNodeHeartbeat:
		return dm.handleNodeHeartbeat(msg)
	default:
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
}

// RegisterNode registers a new DVE node
func (dm *DVEManager) RegisterNode(req *RegisterNodeRequest) (*models.DVENode, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	node := &models.DVENode{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Status:          "online",
		TEEType:         req.TEEType,
		StakeAmount:     req.StakeAmount,
		ReputationScore: 100, // Default starting reputation
		Location:        req.Location,
		IPAddress:       req.IPAddress,
		PublicKey:       req.PublicKey,
		Capabilities:    req.Capabilities,
		LastHeartbeat:   time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
	}

	// Store in database
	if err := dm.storeNode(node); err != nil {
		return nil, fmt.Errorf("failed to store node: %w", err)
	}

	// Add to tracker
	dm.nodeTracker.AddNode(node)

	// Announce to P2P network
	if err := dm.p2pManager.AnnounceNode(node); err != nil {
		log.Printf("Warning: Failed to announce node to P2P network: %v", err)
	}

	log.Printf("DVE node %s (%s) registered successfully", node.ID, node.Name)
	return node, nil
}

// GetNodes returns a list of DVE nodes with optional filtering
func (dm *DVEManager) GetNodes(filter *NodeFilter) ([]*models.DVENode, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var nodes []*models.DVENode

	err := dm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("dve:nodes:*", func(key, value string) bool {
			var node models.DVENode
			if err := json.Unmarshal([]byte(value), &node); err != nil {
				log.Printf("Error unmarshaling node: %v", err)
				return true
			}

			// Apply filters
			if filter != nil && !filter.Matches(&node) {
				return true
			}

			nodes = append(nodes, &node)
			return true
		})
	})

	return nodes, err
}

// AllocateTask allocates a validation task to an optimal DVE node
func (dm *DVEManager) AllocateTask(task *models.ValidationTask) (*models.DVENode, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Find optimal node based on load balancing algorithm
	node, err := dm.loadBalancer.SelectNode(task, dm.nodeTracker.GetActiveNodes())
	if err != nil {
		return nil, fmt.Errorf("failed to select node: %w", err)
	}

	// Update task assignment
	task.AssignedNodeID = node.ID
	task.Status = "assigned"
	task.UpdatedAt = time.Now()

	// Store updated task
	if err := dm.storeTask(task); err != nil {
		return nil, fmt.Errorf("failed to store task: %w", err)
	}

	log.Printf("Task %s allocated to node %s", task.ID, node.ID)
	return node, nil
}

// UpdateNodeStatus updates the status of a DVE node
func (dm *DVEManager) UpdateNodeStatus(nodeID, status string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	node, err := dm.getNodeFromDB(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	node.Status = status
	node.UpdatedAt = time.Now()

	if err := dm.storeNode(node); err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	dm.nodeTracker.UpdateNode(node)
	log.Printf("Node %s status updated to %s", nodeID, status)
	return nil
}

// GetSystemHealth returns overall system health metrics
func (dm *DVEManager) GetSystemHealth() (*models.SystemHealth, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	activeNodes := dm.nodeTracker.GetActiveNodes()
	totalNodes := dm.nodeTracker.GetTotalNodes()

	// Calculate pending/completed/failed tasks
	pendingTasks, completedTasks, failedTasks, err := dm.getTaskStats()
	if err != nil {
		return nil, err
	}

	health := &models.SystemHealth{
		ID:                  uuid.New().String(),
		OverallStatus:       dm.calculateOverallStatus(len(activeNodes), totalNodes),
		ActiveNodes:         len(activeNodes),
		TotalNodes:          totalNodes,
		PendingTasks:        pendingTasks,
		CompletedTasks:      completedTasks,
		FailedTasks:         failedTasks,
		AverageResponseTime: dm.calculateAverageResponseTime(),
		NetworkLatency:      dm.calculateNetworkLatency(),
		TEEHealthScore:      dm.calculateTEEHealthScore(),
		Timestamp:           time.Now(),
	}

	return health, nil
}

// RegisterNodeRequest represents a node registration request
type RegisterNodeRequest struct {
	Name         string   `json:"name"`
	TEEType      string   `json:"tee_type"`
	StakeAmount  int64    `json:"stake_amount"`
	Location     string   `json:"location"`
	IPAddress    string   `json:"ip_address"`
	PublicKey    string   `json:"public_key"`
	Capabilities []string `json:"capabilities"`
	Latitude     float64  `json:"latitude,omitempty"`
	Longitude    float64  `json:"longitude,omitempty"`
}

// NodeFilter represents filters for node queries
type NodeFilter struct {
	Status        string   `json:"status,omitempty"`
	TEEType       string   `json:"tee_type,omitempty"`
	Location      string   `json:"location,omitempty"`
	MinStake      int64    `json:"min_stake,omitempty"`
	MinReputation int      `json:"min_reputation,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
}

// Matches checks if a node matches the filter criteria
func (nf *NodeFilter) Matches(node *models.DVENode) bool {
	if nf.Status != "" && node.Status != nf.Status {
		return false
	}
	if nf.TEEType != "" && node.TEEType != nf.TEEType {
		return false
	}
	if nf.Location != "" && node.Location != nf.Location {
		return false
	}
	if nf.MinStake > 0 && node.StakeAmount < nf.MinStake {
		return false
	}
	if nf.MinReputation > 0 && node.ReputationScore < nf.MinReputation {
		return false
	}
	// Check capabilities
	if len(nf.Capabilities) > 0 {
		nodeCapMap := make(map[string]bool)
		for _, cap := range node.Capabilities {
			nodeCapMap[cap] = true
		}
		for _, reqCap := range nf.Capabilities {
			if !nodeCapMap[reqCap] {
				return false
			}
		}
	}
	return true
}

// Helper methods for DVE Manager

// storeNode stores a node in the database
func (dm *DVEManager) storeNode(node *models.DVENode) error {
	return dm.db.Update(func(tx *buntdb.Tx) error {
		nodeJSON, err := json.Marshal(node)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("dve:nodes:%s", node.ID), string(nodeJSON), nil)
		return err
	})
}

// getNodeFromDB retrieves a node from the database
func (dm *DVEManager) getNodeFromDB(nodeID string) (*models.DVENode, error) {
	var node models.DVENode
	err := dm.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(fmt.Sprintf("dve:nodes:%s", nodeID))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &node)
	})
	return &node, err
}

// storeTask stores a task in the database
func (dm *DVEManager) storeTask(task *models.ValidationTask) error {
	return dm.db.Update(func(tx *buntdb.Tx) error {
		taskJSON, err := json.Marshal(task)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("validation:tasks:%s", task.ID), string(taskJSON), nil)
		return err
	})
}

// loadNodesFromDB loads existing nodes from database
func (dm *DVEManager) loadNodesFromDB() error {
	return dm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("dve:nodes:*", func(key, value string) bool {
			var node models.DVENode
			if err := json.Unmarshal([]byte(value), &node); err != nil {
				log.Printf("Error loading node from DB: %v", err)
				return true
			}
			dm.nodeTracker.AddNode(&node)
			return true
		})
	})
}

// handleNodeAnnouncement handles node announcement messages
func (dm *DVEManager) handleNodeAnnouncement(msg *models.P2PMessage) error {
	// Extract node information from message payload
	nodeData, ok := msg.Payload["node"]
	if !ok {
		return fmt.Errorf("missing node data in announcement")
	}

	// Convert to DVENode struct
	nodeJSON, err := json.Marshal(nodeData)
	if err != nil {
		return err
	}

	var node models.DVENode
	if err := json.Unmarshal(nodeJSON, &node); err != nil {
		return err
	}

	// Update node tracker
	dm.nodeTracker.UpdateNode(&node)

	log.Printf("Received node announcement from %s", node.ID)
	return nil
}

// handleNodeHeartbeat handles node heartbeat messages
func (dm *DVEManager) handleNodeHeartbeat(msg *models.P2PMessage) error {
	nodeID, ok := msg.Payload["node_id"].(string)
	if !ok {
		return fmt.Errorf("missing node_id in heartbeat")
	}

	// Update last heartbeat time
	dm.nodeTracker.UpdateHeartbeat(nodeID)
	return nil
}

// monitorNodes monitors node health and status
func (dm *DVEManager) monitorNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.checkNodeHealth()
		}
	}
}

// checkNodeHealth checks the health of all nodes
func (dm *DVEManager) checkNodeHealth() {
	nodes := dm.nodeTracker.GetAllNodes()
	for _, node := range nodes {
		// Check if node hasn't sent heartbeat in too long
		if time.Since(node.LastHeartbeat) > 2*time.Minute {
			if node.Status != "offline" {
				node.Status = "offline"
				node.UpdatedAt = time.Now()
				dm.storeNode(node)
				dm.nodeTracker.UpdateNode(node)
				log.Printf("Node %s marked as offline due to missing heartbeat", node.ID)
			}
		}
	}
}

// cleanupExpiredNodes removes expired nodes
func (dm *DVEManager) cleanupExpiredNodes() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			// Remove nodes that have been offline for too long
			dm.removeExpiredNodes()
		}
	}
}

// removeExpiredNodes removes nodes that have been offline for extended periods
func (dm *DVEManager) removeExpiredNodes() {
	nodes := dm.nodeTracker.GetAllNodes()
	for _, node := range nodes {
		if node.Status == "offline" && time.Since(node.UpdatedAt) > 24*time.Hour {
			dm.nodeTracker.RemoveNode(node.ID)
			log.Printf("Removed expired node %s", node.ID)
		}
	}
}

// updateMetrics updates system metrics
func (dm *DVEManager) updateMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.collectAndStoreMetrics()
		}
	}
}

// collectAndStoreMetrics collects and stores system metrics
func (dm *DVEManager) collectAndStoreMetrics() {
	health, err := dm.GetSystemHealth()
	if err != nil {
		log.Printf("Error collecting system health: %v", err)
		return
	}

	// Store metrics snapshot
	snapshot := &models.MetricsSnapshot{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      "system_health",
		Data: map[string]interface{}{
			"active_nodes":          health.ActiveNodes,
			"total_nodes":           health.TotalNodes,
			"pending_tasks":         health.PendingTasks,
			"completed_tasks":       health.CompletedTasks,
			"failed_tasks":          health.FailedTasks,
			"average_response_time": health.AverageResponseTime,
			"network_latency":       health.NetworkLatency,
			"tee_health_score":      health.TEEHealthScore,
		},
	}

	if err := dm.storeMetricsSnapshot(snapshot); err != nil {
		log.Printf("Error storing metrics snapshot: %v", err)
	}
}

// storeMetricsSnapshot stores a metrics snapshot
func (dm *DVEManager) storeMetricsSnapshot(snapshot *models.MetricsSnapshot) error {
	return dm.db.Update(func(tx *buntdb.Tx) error {
		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("metrics:historical:%d:%s", snapshot.Timestamp.Unix(), snapshot.Type)
		_, _, err = tx.Set(key, string(snapshotJSON), nil)
		return err
	})
}

// Helper calculation methods
func (dm *DVEManager) getTaskStats() (pending, completed, failed int, err error) {
	err = dm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("validation:tasks:*", func(key, value string) bool {
			var task models.ValidationTask
			if err := json.Unmarshal([]byte(value), &task); err != nil {
				return true
			}
			switch task.Status {
			case "pending", "assigned", "running":
				pending++
			case "completed":
				completed++
			case "failed":
				failed++
			}
			return true
		})
	})
	return
}

func (dm *DVEManager) calculateOverallStatus(activeNodes, totalNodes int) string {
	if totalNodes == 0 {
		return "critical"
	}
	ratio := float64(activeNodes) / float64(totalNodes)
	if ratio >= 0.8 {
		return "healthy"
	} else if ratio >= 0.5 {
		return "degraded"
	}
	return "critical"
}

func (dm *DVEManager) calculateAverageResponseTime() float64 {
	// TODO: Implement actual response time calculation
	return 150.0 // Placeholder
}

func (dm *DVEManager) calculateNetworkLatency() float64 {
	// TODO: Implement actual network latency calculation
	return 25.0 // Placeholder
}

func (dm *DVEManager) calculateTEEHealthScore() float64 {
	// TODO: Implement actual TEE health score calculation
	return 0.95 // Placeholder
}
