package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
	"backend_server/internal/ebpf"
	"backend_server/internal/objects"
	"backend_server/internal/services/p2p"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// DVEManager manages DVE nodes and their operations.
// It also owns the DVECreationService (merged from the dvecreation package)
// so that node registration, session management, and DVE lifecycle are
// accessible through a single service boundary.
type DVEManager struct {
	db                   *database.BuntDBManager
	p2pManager           *p2p.DVEP2PManager
	ebpfManager          ebpf.ManagerInterface
	config               *config.Config
	nodeTracker          *NodeTracker
	loadBalancer         *LoadBalancer
	instanceRegistry     *InstanceRegistry
	lsmIntegration       *ebpf.LSMIntegration
	chainDiscovery       *ChainNativeNodeDiscovery
	ctx                  context.Context
	cancel               context.CancelFunc
	mu                   sync.RWMutex
	enableChainDiscovery bool
}

// NodeTracker tracks the status and health of DVE nodes
type NodeTracker struct {
	nodes map[string]*objects.DVENode
	mu    sync.RWMutex
}

// LoadBalancer handles task assignment and load balancing
type LoadBalancer struct {
	algorithm string // "round_robin", "reputation_based", "resource_based"
	mu        sync.RWMutex
}

// NewDVEManager creates a new DVE Manager instance
func NewDVEManager(db *database.BuntDBManager, p2pManager *p2p.DVEP2PManager, ebpfMgr ebpf.ManagerInterface, cfg *config.Config) (*DVEManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize instance registry for remote DVE discovery
	instanceRegistry, err := NewInstanceRegistry(db, &cfg.DVE.Discovery)
	if err != nil {
		log.Printf("Warning: Failed to initialize instance registry: %v", err)
		// Continue without remote discovery - instance registry will be nil
	}

	// Initialize chain-native node discovery (replaces seedDemoDVENodesIfEmpty)
	var chainDiscovery *ChainNativeNodeDiscovery
	if cfg.DVE.EnableChainDiscovery {
		reputationEngine := NewDVEDefaultReputationEngine(db)
		chainDiscovery = NewChainNativeNodeDiscovery(db, nil, reputationEngine)
		if err := chainDiscovery.loadCacheFromDatabase(); err != nil {
			log.Printf("Warning: Failed to load chain discovery cache: %v", err)
		}
		log.Println("Chain-native node discovery initialized (production mode)")
	}

	manager := &DVEManager{
		db:          db,
		p2pManager:  p2pManager,
		ebpfManager: ebpfMgr,
		config:      cfg,
		nodeTracker: &NodeTracker{
			nodes: make(map[string]*objects.DVENode),
		},
		loadBalancer: &LoadBalancer{
			algorithm: "reputation_based",
		},
		instanceRegistry:     instanceRegistry,
		lsmIntegration:       ebpf.NewLSMIntegration("/var/run/knirv"),
		chainDiscovery:       chainDiscovery,
		enableChainDiscovery: cfg.DVE.EnableChainDiscovery,
		ctx:                  ctx,
		cancel:               cancel,
	}

	// Note: API routes are registered with the unified server

	// Register P2P message handlers
	if p2pManager != nil {
		p2pManager.RegisterMessageHandler(p2p.MessageTypeNodeAnnouncement, manager)
		p2pManager.RegisterMessageHandler(p2p.MessageTypeNodeHeartbeat, manager)
	}

	return manager, nil
}

// SetChainRegistry sets the chain node registry for chain-native discovery
func (dm *DVEManager) SetChainRegistry(registry ChainNodeRegistry) {
	if dm.chainDiscovery != nil {
		dm.chainDiscovery.chainRegistry = registry
		log.Println("Chain registry connected to DVE Manager")
	}
}

// Start starts the DVE Manager service
func (dm *DVEManager) Start(ctx context.Context) error {
	log.Println("Starting DVE Manager service...")

	// Initialize LSM integration for real-time telemetry
	if dm.lsmIntegration != nil {
		if err := dm.lsmIntegration.Initialize(ctx); err != nil {
			log.Printf("Warning: Failed to initialize LSM integration: %v", err)
		} else {
			log.Println("DVE Manager: LSM integration initialized")
		}
	}

	// Note: API routes are registered with the unified server, no separate server needed

	// Start instance registry for remote DVE discovery
	if dm.instanceRegistry != nil {
		if err := dm.instanceRegistry.Start(); err != nil {
			log.Printf("Warning: Failed to start instance registry: %v", err)
		}
	}

	// Start periodic tasks
	go dm.monitorNodes()
	go dm.cleanupExpiredNodes()
	go dm.updateMetrics()

	// Load existing nodes from database
	if err := dm.loadNodesFromDB(); err != nil {
		log.Printf("Warning: Failed to load nodes from database: %v", err)
	}

	// Seed demo DVE nodes if database is empty
	if err := dm.seedDemoDVENodesIfEmpty(); err != nil {
		log.Printf("Warning: Failed to seed demo DVE nodes: %v", err)
	}

	log.Println("DVE Manager service started successfully")
	return nil
}

// Stop stops the DVE Manager service
func (dm *DVEManager) Stop(ctx context.Context) error {
	log.Println("Stopping DVE Manager service...")

	// Stop instance registry for remote DVE discovery
	if dm.instanceRegistry != nil {
		dm.instanceRegistry.Stop()
	}

	// Note: API routes are handled by the unified server

	dm.cancel()
	log.Println("DVE Manager service stopped")
	return nil
}

// HandleMessage implements the P2P MessageHandler interface
func (dm *DVEManager) HandleMessage(ctx context.Context, msg *objects.P2PMessage) error {
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
func (dm *DVEManager) RegisterNode(req *objects.RegisterNodeRequest) (*objects.DVENode, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.registerNodeInternal(req)
}

// registerNodeInternal registers a new DVE node without locking (caller must hold dm.mu)
func (dm *DVEManager) registerNodeInternal(req *objects.RegisterNodeRequest) (*objects.DVENode, error) {
	// Validate TEE type
	validTEETypes := map[string]bool{
		"sgx":              true,
		"sev-snp":          true,
		"tdx":              true,
		"software":         true,
		"browser-extension": true,
	}
	if !validTEETypes[req.TEEType] {
		return nil, fmt.Errorf("invalid TEE type: %s. Valid types: sgx, sev-snp, tdx, software, browser-extension", req.TEEType)
	}

	node := &objects.DVENode{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Status:          "online",
		TEEType:         req.TEEType,
		StakeAmount:     req.StakeAmount,
		ReputationScore: 100, // Initial reputation
		Location:        req.Location,
		IPAddress:       req.IPAddress,
		SSHPort:         req.SSHPort,
		PublicKey:       req.PublicKey,
		Capabilities:    req.Capabilities,
		LastHeartbeat:   time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
	}

	// Set browser-extension specific defaults
	if req.TEEType == "browser-extension" {
		node.IsRemote = true
		node.Connected = true
		node.SSHPort = 0 // No SSH for browser extensions
	}

	// Store in database
	if err := dm.storeNode(node); err != nil {
		return nil, fmt.Errorf("failed to store node: %w", err)
	}

	// Add to tracker
	dm.nodeTracker.AddNode(node)

	// Announce to P2P network
	if dm.p2pManager != nil {
		if err := dm.p2pManager.AnnounceNode(node); err != nil {
			log.Printf("Warning: Failed to announce node to P2P network: %v", err)
		}
	}

	// Register with chain-native discovery if available
	if dm.chainDiscovery != nil {
		if _, err := dm.chainDiscovery.RegisterNodeFromChain(node, req.StakeAmount); err != nil {
			log.Printf("Warning: Failed to register node with chain-native discovery: %v", err)
		}
	}

	log.Printf("DVE node %s (%s) registered successfully", node.ID, node.Name)
	return node, nil
}

// GetNodes returns a list of DVE nodes with optional filtering
func (dm *DVEManager) GetNodes(filter *NodeFilter) ([]*objects.DVENode, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var nodes []*objects.DVENode

	err := dm.db.GetObjectsByPrefix("dve:nodes:", func(key string, value []byte) bool {
		var node objects.DVENode
		if err := json.Unmarshal(value, &node); err != nil {
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

	return nodes, err
}

// AllocateTask allocates a validation task to an optimal DVE node
func (dm *DVEManager) AllocateTask(task *objects.ValidationTask) (*objects.DVENode, error) {
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
func (dm *DVEManager) GetSystemHealth() (*objects.SystemHealth, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	activeNodes := dm.nodeTracker.GetActiveNodes()
	totalNodes := dm.nodeTracker.GetTotalNodes()

	// Calculate pending/completed/failed tasks
	pendingTasks, completedTasks, failedTasks, err := dm.getTaskStats()
	if err != nil {
		return nil, err
	}

	health := &objects.SystemHealth{
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

// RegisterNodeRequest is defined in backend_server/internal/objects

// NodeFilter represents filters for node queries
type NodeFilter struct {
	Status        string   `json:"status,omitempty"`
	TEEType       string   `json:"tee_type,omitempty"`
	Location      string   `json:"location,omitempty"`
	MinStake      int64    `json:"min_stake,omitempty"`
	MinReputation int      `json:"min_reputation,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

// Matches checks if a node matches the filter criteria
func (nf *NodeFilter) Matches(node *objects.DVENode) bool {
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
func (dm *DVEManager) storeNode(node *objects.DVENode) error {
	return dm.db.Transaction(func(tx *buntdb.Tx) error {
		nodeJSON, err := json.Marshal(node)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("dve:nodes:%s", node.ID), string(nodeJSON), nil)
		return err
	})
}

// getNodeFromDB retrieves a node from the database
func (dm *DVEManager) getNodeFromDB(nodeID string) (*objects.DVENode, error) {
	var node objects.DVENode
	err := dm.db.ViewTransaction(func(tx *buntdb.Tx) error {
		value, err := tx.Get(fmt.Sprintf("dve:nodes:%s", nodeID))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &node)
	})
	return &node, err
}

// storeTask stores a task in the database
func (dm *DVEManager) storeTask(task *objects.ValidationTask) error {
	return dm.db.Transaction(func(tx *buntdb.Tx) error {
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
	return dm.db.GetObjectsByPrefix("dve:nodes:", func(key string, value []byte) bool {
		var node objects.DVENode
		if err := json.Unmarshal(value, &node); err != nil {
			log.Printf("Error loading node from DB: %v", err)
			return true
		}
		dm.nodeTracker.AddNode(&node)
		return true
	})
}

// seedDemoDVENodesIfEmpty seeds demo DVE nodes if the database is empty
// In production mode, this uses chain-native discovery instead of demo nodes
func (dm *DVEManager) seedDemoDVENodesIfEmpty() error {
	totalNodes := dm.nodeTracker.GetTotalNodes()
	if totalNodes > 0 {
		log.Printf("DVE nodes already exist (%d nodes), skipping node seeding", totalNodes)
		return nil
	}

	if dm.enableChainDiscovery && dm.chainDiscovery != nil {
		log.Println("Chain-native discovery enabled: syncing nodes from KNIRVCHAIN registry")
		nodes := dm.chainDiscovery.GetDiscoveredNodes()
		if len(nodes) > 0 {
			for _, chainNode := range nodes {
				dveNode := &objects.DVENode{
					ID:              chainNode.NodeID,
					Name:            fmt.Sprintf("DVE Node %s", chainNode.NodeID[:8]),
					Status:          chainNode.Status,
					TEEType:         chainNode.TEEType,
					StakeAmount:     chainNode.StakeAmount,
					ReputationScore: chainNode.ReputationScore,
					Location:        chainNode.Location,
					Capabilities:    chainNode.Capabilities,
					LastHeartbeat:   chainNode.LastHeartbeat,
					CreatedAt:       chainNode.RegisteredAt,
					UpdatedAt:       time.Now(),
				}
				if err := dm.storeNode(dveNode); err != nil {
					log.Printf("Error storing chain-synced node %s: %v", chainNode.NodeID, err)
					continue
				}
				dm.nodeTracker.AddNode(dveNode)
				log.Printf("Synced DVE node from chain: %s (TEE: %s, Reputation: %d)",
					chainNode.NodeID[:12], chainNode.TEEType, chainNode.ReputationScore)
			}
			log.Printf("Chain-native discovery: synced %d nodes from KNIRVCHAIN", len(nodes))
			return nil
		}
		log.Println("Chain-native discovery: no nodes found on KNIRVCHAIN registry")
	}

	if dm.chainDiscovery != nil && dm.chainDiscovery.chainRegistry == nil {
		log.Println("Chain registry not connected - seeding local demo nodes")
	} else if dm.enableChainDiscovery && dm.chainDiscovery != nil {
		// Chain discovery is active and registry is connected but returned no nodes.
		// Seed demo nodes so the dashboard is not empty during initial setup.
		log.Println("Chain registry connected but no nodes discovered yet - seeding demo nodes")
	}

	demoDVENodes := []*objects.RegisterNodeRequest{
		{
			Name:         "Demo DVE Node 1 (SGX - US-East)",
			TEEType:      "sgx",
			StakeAmount:  50000,
			Location:     "us-east-1",
			IPAddress:    "192.168.1.101",
			PublicKey:    "pub_key_demo_1",
			Capabilities: []string{"validation", "attestation", "sgx-compute"},
			Latitude:     40.7128,
			Longitude:    -74.0060,
		},
		{
			Name:         "Demo DVE Node 2 (SEV-SNP - US-West)",
			TEEType:      "sev-snp",
			StakeAmount:  75000,
			Location:     "us-west-2",
			IPAddress:    "192.168.1.102",
			PublicKey:    "pub_key_demo_2",
			Capabilities: []string{"validation", "attestation", "sev-snp-compute"},
			Latitude:     34.0522,
			Longitude:    -118.2437,
		},
		{
			Name:         "Demo DVE Node 3 (TDX - EU-Central)",
			TEEType:      "tdx",
			StakeAmount:  100000,
			Location:     "eu-central-1",
			IPAddress:    "192.168.1.103",
			PublicKey:    "pub_key_demo_3",
			Capabilities: []string{"validation", "attestation", "tdx-compute", "ml-inference"},
			Latitude:     52.5200,
			Longitude:    13.4050,
		},
		{
			Name:         "Demo DVE Node 4 (Software TEE - Asia-Pacific)",
			TEEType:      "software",
			StakeAmount:  25000,
			Location:     "ap-southeast-1",
			IPAddress:    "192.168.1.104",
			PublicKey:    "pub_key_demo_4",
			Capabilities: []string{"validation", "software-tee"},
			Latitude:     1.3521,
			Longitude:    103.8198,
		},
		{
			Name:         "Demo DVE Node 5 (SGX - EU-West)",
			TEEType:      "sgx",
			StakeAmount:  60000,
			Location:     "eu-west-1",
			IPAddress:    "192.168.1.105",
			PublicKey:    "pub_key_demo_5",
			Capabilities: []string{"validation", "attestation", "sgx-compute", "ml-inference"},
			Latitude:     53.3498,
			Longitude:    -6.2603,
		},
		{
			Name:         "Demo Browser DVE Node",
			TEEType:      "browser-extension",
			StakeAmount:  5000,
			Location:     "browser",
			Capabilities: []string{"policy-check", "signature-verify", "reasoning-simple", "skill-lint"},
		},
	}

	for _, req := range demoDVENodes {
		node, err := dm.RegisterNode(req)
		if err != nil {
			log.Printf("Error registering demo node %s: %v", req.Name, err)
			continue
		}
		log.Printf("Successfully registered demo DVE node: %s (ID: %s)", node.Name, node.ID)
	}

	return nil
}

// handleNodeAnnouncement handles node announcement messages
func (dm *DVEManager) handleNodeAnnouncement(msg *objects.P2PMessage) error {
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

	var node objects.DVENode
	if err := json.Unmarshal(nodeJSON, &node); err != nil {
		return err
	}

	// Update node tracker (in-memory)
	dm.nodeTracker.UpdateNode(&node)

	// Persist to database so it's visible to API queries
	if err := dm.storeNode(&node); err != nil {
		log.Printf("Warning: Failed to persist P2P discovered node %s to database: %v", node.ID, err)
	}

	log.Printf("Received node announcement from %s and persisted to database", node.ID)
	return nil
}

// handleNodeHeartbeat handles node heartbeat messages
func (dm *DVEManager) handleNodeHeartbeat(msg *objects.P2PMessage) error {
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
	// Check if context is cancelled before doing any work
	select {
	case <-dm.ctx.Done():
		return
	default:
	}

	health, err := dm.GetSystemHealth()
	if err != nil {
		// Only log error if context is not cancelled
		select {
		case <-dm.ctx.Done():
			return
		default:
			log.Printf("Error collecting system health: %v", err)
		}
		return
	}

	// Store metrics snapshot
	snapshot := &objects.MetricsSnapshot{
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
func (dm *DVEManager) storeMetricsSnapshot(snapshot *objects.MetricsSnapshot) error {
	return dm.db.Transaction(func(tx *buntdb.Tx) error {
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
	err = dm.db.GetObjectsByPrefix("validation:tasks:", func(key string, value []byte) bool {
		var task objects.ValidationTask
		if err := json.Unmarshal(value, &task); err != nil {
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
	dm.mu.RLock()
	ebpfMgr := dm.ebpfManager
	dm.mu.RUnlock()

	if ebpfMgr != nil {
		stats, err := ebpfMgr.GetProcessMetrics()
		if err == nil && len(stats) > 0 {
			var totalCPUTime uint64
			var processCount int
			for _, s := range stats {
				totalCPUTime += s.CPUTimeNs
				processCount++
			}
			if processCount > 0 {
				avgCPUTimeNs := float64(totalCPUTime) / float64(processCount)
				return avgCPUTimeNs / 1_000_000.0
			}
		}
	}

	nodes := dm.nodeTracker.GetAllNodes()
	if len(nodes) == 0 {
		return 0.0
	}

	return 10.0
}

func (dm *DVEManager) calculateNetworkLatency() float64 {
	dm.mu.RLock()
	ebpfMgr := dm.ebpfManager
	dm.mu.RUnlock()

	if ebpfMgr != nil {
		stats, err := ebpfMgr.GetProcessMetrics()
		if err == nil && len(stats) > 0 {
			var totalContextSwitches uint32
			var totalPageFaults uint32
			var processCount int
			for _, s := range stats {
				totalContextSwitches += s.ContextSwitches
				totalPageFaults += s.PageFaults
				processCount++
			}
			if processCount > 0 {
				avgContextSwitches := float64(totalContextSwitches) / float64(processCount)
				avgPageFaults := float64(totalPageFaults) / float64(processCount)
				latencyScore := (avgContextSwitches * 0.3) + (avgPageFaults * 0.7)
				return latencyScore
			}
		}
	}

	if dm.p2pManager != nil {
		peers := dm.p2pManager.GetConnectedPeers()
		if len(peers) > 0 {
			return float64(len(peers)) * 2.0
		}
	}

	return 0.0
}

func (dm *DVEManager) calculateTEEHealthScore() float64 {
	dm.mu.RLock()
	lsm := dm.lsmIntegration
	dm.mu.RUnlock()

	if lsm != nil && lsm.IsEnabled() {
		auditLog := lsm.GetAuditLog()
		if len(auditLog) == 0 {
			return 1.0
		}

		deniedCount := 0
		for _, entry := range auditLog {
			if entry.Decision == "DENY" {
				deniedCount++
			}
		}

		return 1.0 - (float64(deniedCount) / float64(len(auditLog)))
	}
	return 0.98 // High confidence default
}

// GetAllNodes returns all nodes managed by the DVE Manager
func (dm *DVEManager) GetAllNodes() []*objects.DVENode {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.nodeTracker == nil {
		return []*objects.DVENode{}
	}

	dm.nodeTracker.mu.RLock()
	defer dm.nodeTracker.mu.RUnlock()

	nodes := make([]*objects.DVENode, 0, len(dm.nodeTracker.nodes))
	for _, node := range dm.nodeTracker.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNode returns a specific node by ID
func (dm *DVEManager) GetNode(nodeID string) (*objects.DVENode, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// First try to get from in-memory tracker
	if dm.nodeTracker != nil {
		if node, exists := dm.nodeTracker.GetNode(nodeID); exists {
			return node, nil
		}
	}

	// Fall back to database lookup
	node, err := dm.getNodeFromDB(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found")
	}

	// Add to tracker for future lookups (best effort, don't fail if tracker is nil)
	if dm.nodeTracker != nil {
		dm.nodeTracker.AddNode(node)
	}

	return node, nil
}

// UpdateNode updates a DVE node with new information
func (dm *DVEManager) UpdateNode(nodeID string, updates map[string]interface{}) (*objects.DVENode, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Get existing node from database
	node, err := dm.getNodeFromDB(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		node.Name = name
	}
	if status, ok := updates["status"].(string); ok && status != "" {
		node.Status = status
	}
	if location, ok := updates["location"].(string); ok {
		node.Location = location
	}
	if ipAddress, ok := updates["ip_address"].(string); ok {
		node.IPAddress = ipAddress
	}
	if stakeAmount, ok := updates["stake_amount"].(float64); ok {
		node.StakeAmount = int64(stakeAmount)
	}
	if capabilities, ok := updates["capabilities"].([]interface{}); ok {
		node.Capabilities = make([]string, len(capabilities))
		for i, cap := range capabilities {
			if capStr, ok := cap.(string); ok {
				node.Capabilities[i] = capStr
			}
		}
	}

	node.UpdatedAt = time.Now()

	// Store updated node in database
	if err := dm.storeNode(node); err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	// Update in tracker
	dm.nodeTracker.UpdateNode(node)

	log.Printf("DVE node %s updated successfully", nodeID)
	return node, nil
}

// RemoveNode removes a DVE node from the system
func (dm *DVEManager) RemoveNode(nodeID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if node exists
	_, err := dm.getNodeFromDB(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	// Remove from database
	err = dm.db.Transaction(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(fmt.Sprintf("dve:nodes:%s", nodeID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to remove node from database: %w", err)
	}

	// Remove from tracker
	dm.nodeTracker.RemoveNode(nodeID)

	log.Printf("DVE node %s removed successfully", nodeID)
	return nil
}

// RegisterBrowserDVE registers a new browser-extension DVE node with wallet-based identity
func (dm *DVEManager) RegisterBrowserDVE(walletAddress string, capabilities []string, badgeNFTIDs []string, extensionID string, browserVersion string) (*objects.DVENode, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	walletLabel := walletAddress
	if len(walletLabel) > 8 {
		walletLabel = walletLabel[:8]
	}

	req := &objects.RegisterNodeRequest{
		Name:           "Browser DVE - " + walletLabel,
		TEEType:        "browser-extension",
		StakeAmount:    10000,
		Location:       "browser",
		Capabilities:   capabilities,
		WalletAddress:  walletAddress,
		ExtensionID:    extensionID,
		BrowserVersion: browserVersion,
		BadgeNFTIDs:    badgeNFTIDs,
	}

	node, err := dm.registerNodeInternal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register browser DVE: %w", err)
	}

	// Set browser-specific fields
	node.WalletAddress = walletAddress
	node.ExtensionID = extensionID
	node.BrowserVersion = browserVersion
	node.BadgeNFTIDs = badgeNFTIDs
	node.IsRemote = true
	node.Connected = true
	node.SSHPort = 0

	// Update in database
	if err := dm.storeNode(node); err != nil {
		return nil, fmt.Errorf("failed to store browser DVE node: %w", err)
	}
	dm.nodeTracker.UpdateNode(node)

	// Register with chain-native discovery if available
	if dm.chainDiscovery != nil {
		if err := dm.chainDiscovery.RegisterBrowserDVEIdentity(node.ID, walletAddress, capabilities); err != nil {
			log.Printf("Warning: Failed to register browser DVE with chain discovery: %v", err)
		}
	}

	walletShort := walletAddress
	if len(walletShort) > 12 {
		walletShort = walletShort[:12]
	}
	log.Printf("Browser DVE node %s registered (wallet: %s, extension: %s)", node.ID, walletShort, extensionID)
	return node, nil
}

// UpdateBrowserDVEHeartbeat updates the heartbeat for a browser-extension node with its WS connection ID
func (dm *DVEManager) UpdateBrowserDVEHeartbeat(nodeID string, wsConnectionID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	node, err := dm.getNodeFromDB(nodeID)
	if err != nil {
		return fmt.Errorf("browser DVE node not found: %w", err)
	}

	if node.TEEType != "browser-extension" {
		return fmt.Errorf("node %s is not a browser-extension DVE", nodeID)
	}

	node.WSConnectionID = wsConnectionID
	node.LastHeartbeat = time.Now()
	node.UpdatedAt = time.Now()
	node.Status = "online"
	node.Connected = true

	if err := dm.storeNode(node); err != nil {
		return fmt.Errorf("failed to update browser DVE heartbeat: %w", err)
	}
	dm.nodeTracker.UpdateNode(node)

	return nil
}

// GetRemoteInstances returns all managed remote DVE instances
func (dm *DVEManager) GetRemoteInstances() map[string]map[string]interface{} {
	if dm.instanceRegistry == nil {
		return make(map[string]map[string]interface{})
	}
	return dm.instanceRegistry.GetAllInstancesStats()
}

// GetRemoteInstanceNodes returns all nodes from a specific remote instance
func (dm *DVEManager) GetRemoteInstanceNodes(instanceURL string) ([]*objects.DVENode, error) {
	if dm.instanceRegistry == nil {
		return nil, fmt.Errorf("instance registry not initialized")
	}
	return dm.instanceRegistry.GetInstanceNodes(instanceURL)
}

// AddRemoteInstance registers a new remote DVE instance for discovery
func (dm *DVEManager) AddRemoteInstance(instanceURL string) error {
	if dm.instanceRegistry == nil {
		return fmt.Errorf("instance registry not initialized")
	}
	dm.instanceRegistry.AddInstance(instanceURL)
	return nil
}

// RemoveRemoteInstance unregisters a remote DVE instance
func (dm *DVEManager) RemoveRemoteInstance(instanceURL string) error {
	if dm.instanceRegistry == nil {
		return fmt.Errorf("instance registry not initialized")
	}
	dm.instanceRegistry.RemoveInstance(instanceURL)
	return nil
}

// GetHealthyRemoteInstances returns only healthy remote instances
func (dm *DVEManager) GetHealthyRemoteInstances() map[string]map[string]interface{} {
	if dm.instanceRegistry == nil {
		return make(map[string]map[string]interface{})
	}

	stats := dm.instanceRegistry.GetAllInstancesStats()
	healthyStats := make(map[string]map[string]interface{})

	for url, stat := range stats {
		if isHealthy, ok := stat["is_healthy"].(bool); ok && isHealthy {
			healthyStats[url] = stat
		}
	}

	return healthyStats
}

// GetNodeTasks returns all tasks assigned to a specific node
func (dm *DVEManager) GetNodeTasks(nodeID string) ([]*objects.ValidationTask, error) {
	var tasks []*objects.ValidationTask

	err := dm.db.GetObjectsByPrefix("validation:tasks:", func(key string, value []byte) bool {
		var task objects.ValidationTask
		if err := json.Unmarshal(value, &task); err != nil {
			return true
		}
		if task.AssignedNodeID == nodeID {
			tasks = append(tasks, &task)
		}
		return true
	})

	return tasks, err
}

// CreateTask creates a new validation task
func (dm *DVEManager) CreateTask(task *objects.ValidationTask) error {
	return dm.storeTask(task)
}

// GetTask retrieves a specific task by ID
func (dm *DVEManager) GetTask(taskID string) (*objects.ValidationTask, error) {
	var task objects.ValidationTask
	err := dm.db.ViewTransaction(func(tx *buntdb.Tx) error {
		value, err := tx.Get(fmt.Sprintf("validation:tasks:%s", taskID))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &task)
	})
	return &task, err
}

// UpdateTask updates a task with new information
func (dm *DVEManager) UpdateTask(taskID string, updates map[string]interface{}) (*objects.ValidationTask, error) {
	task, err := dm.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	if status, ok := updates["status"].(string); ok {
		task.Status = status
	}
	if priority, ok := updates["priority"].(float64); ok {
		task.Priority = int(priority)
	}
	if parameters, ok := updates["parameters"].(map[string]interface{}); ok {
		task.Parameters = parameters
	}
	task.UpdatedAt = time.Now()

	if err := dm.storeTask(task); err != nil {
		return nil, err
	}

	return task, nil
}

// ListTasks returns all tasks with optional filtering
func (dm *DVEManager) ListTasks(statusFilter, nodeIDFilter string) ([]*objects.ValidationTask, error) {
	var tasks []*objects.ValidationTask

	err := dm.db.GetObjectsByPrefix("validation:tasks:", func(key string, value []byte) bool {
		var task objects.ValidationTask
		if err := json.Unmarshal(value, &task); err != nil {
			return true
		}

		if statusFilter != "" && task.Status != statusFilter {
			return true
		}
		if nodeIDFilter != "" && task.AssignedNodeID != nodeIDFilter {
			return true
		}

		tasks = append(tasks, &task)
		return true
	})

	return tasks, err
}

// GetNodeMetrics returns the current metrics for a specific node
func (dm *DVEManager) GetNodeMetrics(nodeID string) (map[string]interface{}, error) {
	node, err := dm.GetNode(nodeID)
	if err != nil {
		return nil, err
	}

	metrics := map[string]interface{}{
		"node_id":      node.ID,
		"node_name":    node.Name,
		"status":       node.Status,
		"reputation":   node.ReputationScore,
		"last_seen":    node.LastHeartbeat,
		"location":     node.Location,
		"tee_type":     node.TEEType,
		"capabilities": node.Capabilities,
	}

	if dm.ebpfManager != nil {
		stats, err := dm.ebpfManager.GetProcessMetrics()
		if err == nil && len(stats) > 0 {
			var totalCPUTime uint64
			var totalMemory uint64
			var totalNetTx, totalNetRx uint64
			for _, s := range stats {
				totalCPUTime += s.CPUTimeNs
				totalMemory += s.MemoryBytes
				totalNetTx += s.NetTxBytes
				totalNetRx += s.NetRxBytes
			}
			metrics["cpu_time_ns"] = totalCPUTime
			metrics["memory_bytes"] = totalMemory
			metrics["net_tx_bytes"] = totalNetTx
			metrics["net_rx_bytes"] = totalNetRx
		}
	}

	return metrics, nil
}

// GetNodeMetricsHistory returns historical metrics for a specific node
func (dm *DVEManager) GetNodeMetricsHistory(nodeID string, limitStr string) ([]map[string]interface{}, error) {
	limit := 10
	if limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && l > 0 {
			// Use provided limit
		}
	}

	var history []map[string]interface{}
	_ = dm.db.GetObjectsByPrefix("metrics:historical:", func(key string, value []byte) bool {
		var snapshot objects.MetricsSnapshot
		if err := json.Unmarshal(value, &snapshot); err != nil {
			return true
		}

		if data, ok := snapshot.Data["node_id"]; ok && data == nodeID {
			history = append(history, map[string]interface{}{
				"timestamp": snapshot.Timestamp,
				"data":      snapshot.Data,
			})
		}

		return true
	})

	if len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}

// GetConnectedP2PPeers returns the list of connected P2P peers from the P2P manager.
func (dm *DVEManager) GetConnectedP2PPeers() []objects.PeerInfo {
	if dm.p2pManager == nil {
		return []objects.PeerInfo{}
	}
	return dm.p2pManager.GetConnectedPeers()
}
