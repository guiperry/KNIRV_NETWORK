package failover

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/database"
)

// NodeRole represents the role of a node in the network
type NodeRole string

const (
	RoleRoot     NodeRole = "root"
	RoleBootnode NodeRole = "bootnode"
	RolePeer     NodeRole = "peer"
)

// NodeStatus represents the health status of a node
type NodeStatus string

const (
	StatusHealthy   NodeStatus = "healthy"
	StatusUnhealthy NodeStatus = "unhealthy"
	StatusUnknown   NodeStatus = "unknown"
)

// NodeInfo represents information about a network node
type NodeInfo struct {
	ID       string     `json:"id"`
	Role     NodeRole   `json:"role"`
	Status   NodeStatus `json:"status"`
	LastSeen time.Time  `json:"last_seen"`
	Endpoint string     `json:"endpoint"`
	Priority int        `json:"priority"` // Higher priority = more likely to be selected as failover
}

// FailoverConfig holds configuration for the failover service
type FailoverConfig struct {
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	FailoverThreshold   int           `json:"failover_threshold"` // Number of failed checks before failover
	EnableAutoFailover  bool          `json:"enable_auto_failover"`
	RootNodes           []string      `json:"root_nodes"` // List of root node endpoints to monitor
}

// FailoverService manages failover between root nodes and bootnodes
type FailoverService struct {
	config     *FailoverConfig
	db         *database.BuntDBManager
	nodes      map[string]*NodeInfo
	currentRoot string
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewFailoverService creates a new failover service instance
func NewFailoverService(config *FailoverConfig, db *database.BuntDBManager) *FailoverService {
	ctx, cancel := context.WithCancel(context.Background())

	return &FailoverService{
		config:     config,
		db:         db,
		nodes:      make(map[string]*NodeInfo),
		currentRoot: "",
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the failover monitoring service
func (fs *FailoverService) Start() error {
	log.Println("Starting Failover Service...")

	// Initialize nodes from configuration
	fs.initializeNodes()

	// Start health monitoring
	fs.wg.Add(1)
	go fs.healthMonitor()

	// Start failover coordinator if auto-failover is enabled
	if fs.config.EnableAutoFailover {
		fs.wg.Add(1)
		go fs.failoverCoordinator()
	}

	log.Println("Failover Service started successfully")
	return nil
}

// Stop stops the failover service
func (fs *FailoverService) Stop() error {
	log.Println("Stopping Failover Service...")

	if fs.cancel != nil {
		fs.cancel()
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		fs.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Failover Service stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Failover Service stop timeout - forcing exit")
	}

	return nil
}

// initializeNodes sets up the initial node registry
func (fs *FailoverService) initializeNodes() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Add root nodes from configuration
	for i, endpoint := range fs.config.RootNodes {
		nodeID := fmt.Sprintf("root-%d", i+1)
		fs.nodes[nodeID] = &NodeInfo{
			ID:       nodeID,
			Role:     RoleRoot,
			Status:   StatusUnknown,
			LastSeen: time.Now(),
			Endpoint: endpoint,
			Priority: len(fs.config.RootNodes) - i, // Higher index = lower priority
		}
	}

	// Set initial root (highest priority healthy node)
	fs.selectInitialRoot()
}

// selectInitialRoot chooses the initial root node
func (fs *FailoverService) selectInitialRoot() {
	var bestNode *NodeInfo
	bestPriority := -1

	for _, node := range fs.nodes {
		if node.Role == RoleRoot && node.Status == StatusHealthy && node.Priority > bestPriority {
			bestNode = node
			bestPriority = node.Priority
		}
	}

	if bestNode != nil {
		fs.currentRoot = bestNode.ID
		log.Printf("Selected initial root node: %s (%s)", bestNode.ID, bestNode.Endpoint)
	} else {
		log.Println("Warning: No healthy root nodes available for initial selection")
	}
}

// healthMonitor continuously checks the health of all nodes
func (fs *FailoverService) healthMonitor() {
	defer fs.wg.Done()

	ticker := time.NewTicker(fs.config.HealthCheckInterval)
	defer ticker.Stop()

	log.Printf("Health monitor started - checking every %v", fs.config.HealthCheckInterval)

	for {
		select {
		case <-fs.ctx.Done():
			log.Println("Health monitor stopping...")
			return
		case <-ticker.C:
			fs.checkAllNodesHealth()
		}
	}
}

// checkAllNodesHealth performs health checks on all registered nodes
func (fs *FailoverService) checkAllNodesHealth() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, node := range fs.nodes {
		go fs.checkNodeHealth(node)
	}
}

// checkNodeHealth performs a health check on a specific node
func (fs *FailoverService) checkNodeHealth(node *NodeInfo) {
	client := &http.Client{
		Timeout: fs.config.HealthCheckTimeout,
	}

	resp, err := client.Get(fmt.Sprintf("%s/health", node.Endpoint))
	if err != nil {
		fs.updateNodeStatus(node.ID, StatusUnhealthy)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fs.updateNodeStatus(node.ID, StatusHealthy)
	} else {
		fs.updateNodeStatus(node.ID, StatusUnhealthy)
	}
}

// updateNodeStatus updates the status of a node
func (fs *FailoverService) updateNodeStatus(nodeID string, status NodeStatus) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if node, exists := fs.nodes[nodeID]; exists {
		oldStatus := node.Status
		node.Status = status
		node.LastSeen = time.Now()

		if oldStatus != status {
			log.Printf("Node %s status changed: %s -> %s", nodeID, oldStatus, status)
		}
	}
}

// failoverCoordinator monitors for failover conditions and initiates failover when needed
func (fs *FailoverService) failoverCoordinator() {
	defer fs.wg.Done()

	ticker := time.NewTicker(fs.config.HealthCheckInterval * 2) // Check less frequently than health monitor
	defer ticker.Stop()

	log.Println("Failover coordinator started")

	failoverCount := make(map[string]int)

	for {
		select {
		case <-fs.ctx.Done():
			log.Println("Failover coordinator stopping...")
			return
		case <-ticker.C:
			fs.checkFailoverConditions(failoverCount)
		}
	}
}

// checkFailoverConditions checks if failover conditions are met
func (fs *FailoverService) checkFailoverConditions(failoverCount map[string]int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Check if current root is unhealthy
	if fs.currentRoot != "" {
		if currentRootNode, exists := fs.nodes[fs.currentRoot]; exists {
			if currentRootNode.Status == StatusUnhealthy {
				failoverCount[fs.currentRoot]++

				if failoverCount[fs.currentRoot] >= fs.config.FailoverThreshold {
					log.Printf("Root node %s has failed %d health checks - initiating failover",
						fs.currentRoot, failoverCount[fs.currentRoot])

					if fs.performFailover() {
						// Reset failover count on successful failover
						failoverCount[fs.currentRoot] = 0
					}
				}
			} else {
				// Reset counter if node becomes healthy again
				failoverCount[fs.currentRoot] = 0
			}
		}
	}
}

// performFailover attempts to failover to a healthy bootnode
func (fs *FailoverService) performFailover() bool {
	// Find the best available bootnode to promote to root
	var bestBootnode *NodeInfo
	bestPriority := -1

	for _, node := range fs.nodes {
		if node.Role == RoleBootnode && node.Status == StatusHealthy && node.Priority > bestPriority {
			bestBootnode = node
			bestPriority = node.Priority
		}
	}

	if bestBootnode == nil {
		log.Println("Failover failed: No healthy bootnodes available")
		return false
	}

	// Perform the failover
	oldRoot := fs.currentRoot
	fs.currentRoot = bestBootnode.ID

	// Update the node's role (in a real implementation, this would involve
	// communicating with the node to change its role)
	bestBootnode.Role = RoleRoot

	log.Printf("Failover completed: %s promoted from bootnode to root (replacing %s)",
		bestBootnode.ID, oldRoot)

	// Notify other components about the failover
	fs.notifyFailover(oldRoot, fs.currentRoot)

	return true
}

// notifyFailover notifies other services about the failover event
func (fs *FailoverService) notifyFailover(oldRoot, newRoot string) {
	// This would typically send notifications to:
	// - Load balancers
	// - DNS services
	// - Other network nodes
	// - Monitoring systems

	log.Printf("Failover notification: Root changed from %s to %s", oldRoot, newRoot)

	// Store failover event in database for audit trail
	if fs.db != nil {
		event := map[string]interface{}{
			"type":      "failover",
			"old_root":  oldRoot,
			"new_root":  newRoot,
			"timestamp": time.Now().Unix(),
		}
		// In a real implementation, you'd store this in the database
		_ = event // Prevent unused variable warning
	}
}

// RegisterBootnode registers a bootnode with the failover service
func (fs *FailoverService) RegisterBootnode(nodeID, endpoint string, priority int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.nodes[nodeID] = &NodeInfo{
		ID:       nodeID,
		Role:     RoleBootnode,
		Status:   StatusUnknown,
		LastSeen: time.Now(),
		Endpoint: endpoint,
		Priority: priority,
	}

	log.Printf("Registered bootnode: %s (%s) with priority %d", nodeID, endpoint, priority)
}

// GetCurrentRoot returns the ID of the current root node
func (fs *FailoverService) GetCurrentRoot() string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.currentRoot
}

// GetNodeStatus returns the status of a specific node
func (fs *FailoverService) GetNodeStatus(nodeID string) (NodeStatus, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if node, exists := fs.nodes[nodeID]; exists {
		return node.Status, true
	}
	return StatusUnknown, false
}

// GetAllNodes returns information about all registered nodes
func (fs *FailoverService) GetAllNodes() map[string]*NodeInfo {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Return a copy to prevent external modification
	nodes := make(map[string]*NodeInfo)
	for id, node := range fs.nodes {
		nodeCopy := *node // Shallow copy
		nodes[id] = &nodeCopy
	}

	return nodes
}

// IsFailoverEnabled returns whether auto-failover is enabled
func (fs *FailoverService) IsFailoverEnabled() bool {
	return fs.config.EnableAutoFailover
}