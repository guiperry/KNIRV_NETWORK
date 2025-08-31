package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-backend/internal/models"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/tidwall/buntdb"
)

// DVEP2PManager implements P2P networking for DVE nodes aligned with KNIRV-ORACLE
type DVEP2PManager struct {
	host   host.Host
	dht    *dht.IpfsDHT
	pubsub *pubsub.PubSub
	db     *buntdb.DB
	ctx    context.Context
	cancel context.CancelFunc

	// DVE-specific topics
	validationTopic *pubsub.Topic
	resultTopic     *pubsub.Topic
	nodeTopic       *pubsub.Topic

	// Subscriptions
	validationSub *pubsub.Subscription
	resultSub     *pubsub.Subscription
	nodeSub       *pubsub.Subscription

	nodeRole string // "dve-validator", "dve-observer", "dve-coordinator", "dve-manager"
	chainID  string

	// Message handlers
	messageHandlers map[string]MessageHandler
	mu              sync.RWMutex
}

// MessageHandler defines the interface for handling P2P messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *models.P2PMessage) error
}

// DVE Protocol Constants (aligned with KNIRV-ORACLE)
const (
	DVEValidationTopic      = "dve-validation"
	DVEResultTopic          = "dve-results"
	DVENodeTopic            = "dve-nodes"
	DVEValidationProtocolID = "/knirv/dve-validation/1.0.0"
	DVEResultProtocolID     = "/knirv/dve-results/1.0.0"
	DVENodeSyncProtocolID   = "/knirv/dve-sync/1.0.0"
)

// DVE Message Types
const (
	MessageTypeValidationRequest = "validation_request"
	MessageTypeValidationResult  = "validation_result"
	MessageTypeNodeAnnouncement  = "node_announcement"
	MessageTypeNodeHeartbeat     = "node_heartbeat"
	MessageTypeTaskAssignment    = "task_assignment"
	MessageTypeConsensusVote     = "consensus_vote"
)

// NewDVEP2PManager creates a new DVE P2P manager
func NewDVEP2PManager(chainID, nodeRole string, db *buntdb.DB) (*DVEP2PManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create libp2p host (aligned with KNIRV-ORACLE configuration)
	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/4001"),
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Create DHT for node discovery
	dhtInstance, err := dht.New(ctx, host)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Create GossipSub for message distribution
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	manager := &DVEP2PManager{
		host:            host,
		dht:             dhtInstance,
		pubsub:          ps,
		db:              db,
		ctx:             ctx,
		cancel:          cancel,
		nodeRole:        nodeRole,
		chainID:         chainID,
		messageHandlers: make(map[string]MessageHandler),
	}

	// Initialize topics and subscriptions
	if err := manager.setupTopics(); err != nil {
		cancel()
		return nil, err
	}

	return manager, nil
}

// setupTopics initializes P2P topics and subscriptions
func (dpm *DVEP2PManager) setupTopics() error {
	var err error

	// Create topic names with chain ID prefix (aligned with KNIRV-ORACLE)
	validationTopicName := fmt.Sprintf("%s.%s", dpm.chainID, DVEValidationTopic)
	resultTopicName := fmt.Sprintf("%s.%s", dpm.chainID, DVEResultTopic)
	nodeTopicName := fmt.Sprintf("%s.%s", dpm.chainID, DVENodeTopic)

	// Join topics
	dpm.validationTopic, err = dpm.pubsub.Join(validationTopicName)
	if err != nil {
		return fmt.Errorf("failed to join validation topic: %w", err)
	}

	dpm.resultTopic, err = dpm.pubsub.Join(resultTopicName)
	if err != nil {
		return fmt.Errorf("failed to join result topic: %w", err)
	}

	dpm.nodeTopic, err = dpm.pubsub.Join(nodeTopicName)
	if err != nil {
		return fmt.Errorf("failed to join node topic: %w", err)
	}

	// Subscribe to topics
	dpm.validationSub, err = dpm.validationTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to validation topic: %w", err)
	}

	dpm.resultSub, err = dpm.resultTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to result topic: %w", err)
	}

	dpm.nodeSub, err = dpm.nodeTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to node topic: %w", err)
	}

	log.Printf("[DVE][%s] P2P topics initialized: %s, %s, %s",
		dpm.nodeRole, validationTopicName, resultTopicName, nodeTopicName)

	return nil
}

// Start starts the P2P manager
func (dpm *DVEP2PManager) Start() {
	log.Printf("[DVE][%s] Starting P2P manager...", dpm.nodeRole)

	// Start message handlers
	go dpm.handleValidationRequests()
	go dpm.handleValidationResults()
	go dpm.handleNodeAnnouncements()

	// Start node discovery
	go dpm.discoverNodes()

	// Announce this node to the network
	go dpm.announceNode()

	// Start periodic heartbeat
	go dpm.sendHeartbeat()

	log.Printf("[DVE][%s] P2P manager started successfully", dpm.nodeRole)
}

// Stop stops the P2P manager
func (dpm *DVEP2PManager) Stop() {
	log.Printf("[DVE][%s] Stopping P2P manager...", dpm.nodeRole)

	dpm.cancel()

	if dpm.host != nil {
		dpm.host.Close()
	}

	log.Printf("[DVE][%s] P2P manager stopped", dpm.nodeRole)
}

// RegisterMessageHandler registers a message handler for a specific message type
func (dpm *DVEP2PManager) RegisterMessageHandler(messageType string, handler MessageHandler) {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()
	dpm.messageHandlers[messageType] = handler
}

// BroadcastValidationRequest broadcasts a validation request to the network
func (dpm *DVEP2PManager) BroadcastValidationRequest(req *models.ValidationTask) error {
	message := &models.P2PMessage{
		ID:        req.ID,
		Type:      MessageTypeValidationRequest,
		From:      dpm.host.ID().String(),
		Topic:     DVEValidationTopic,
		Payload:   map[string]interface{}{"task": req},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	err = dpm.validationTopic.Publish(dpm.ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish validation request: %w", err)
	}

	log.Printf("[DVE][%s] Validation request %s broadcast to network", dpm.nodeRole, req.ID)
	return nil
}

// BroadcastValidationResult broadcasts a validation result to the network
func (dpm *DVEP2PManager) BroadcastValidationResult(result *models.ValidationResult) error {
	message := &models.P2PMessage{
		ID:        result.ID,
		Type:      MessageTypeValidationResult,
		From:      dpm.host.ID().String(),
		Topic:     DVEResultTopic,
		Payload:   map[string]interface{}{"result": result},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal validation result: %w", err)
	}

	err = dpm.resultTopic.Publish(dpm.ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish validation result: %w", err)
	}

	log.Printf("[DVE][%s] Validation result %s broadcast to network", dpm.nodeRole, result.ID)
	return nil
}

// AnnounceNode announces this node to the network
func (dpm *DVEP2PManager) AnnounceNode(node *models.DVENode) error {
	announcement := map[string]interface{}{
		"node_id":          node.ID,
		"role":             dpm.nodeRole,
		"capabilities":     node.Capabilities,
		"tee_type":         node.TEEType,
		"stake_amount":     node.StakeAmount,
		"reputation_score": node.ReputationScore,
		"location":         node.Location,
		"timestamp":        time.Now(),
	}

	message := &models.P2PMessage{
		ID:        node.ID,
		Type:      MessageTypeNodeAnnouncement,
		From:      dpm.host.ID().String(),
		Topic:     DVENodeTopic,
		Payload:   announcement,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal node announcement: %w", err)
	}

	err = dpm.nodeTopic.Publish(dpm.ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish node announcement: %w", err)
	}

	log.Printf("[DVE][%s] Node %s announced to network", dpm.nodeRole, node.ID)
	return nil
}

// handleValidationRequests handles incoming validation requests
func (dpm *DVEP2PManager) handleValidationRequests() {
	for {
		select {
		case <-dpm.ctx.Done():
			return
		default:
			msg, err := dpm.validationSub.Next(dpm.ctx)
			if err != nil {
				if dpm.ctx.Err() != nil {
					return
				}
				log.Printf("[DVE][%s] Error receiving validation message: %v", dpm.nodeRole, err)
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dpm.host.ID() {
				continue
			}

			var p2pMsg models.P2PMessage
			if err := json.Unmarshal(msg.Data, &p2pMsg); err != nil {
				log.Printf("[DVE][%s] Error unmarshaling validation message: %v", dpm.nodeRole, err)
				continue
			}

			// Handle message based on type
			dpm.handleMessage(&p2pMsg)
		}
	}
}

// handleValidationResults handles incoming validation results
func (dpm *DVEP2PManager) handleValidationResults() {
	for {
		select {
		case <-dpm.ctx.Done():
			return
		default:
			msg, err := dpm.resultSub.Next(dpm.ctx)
			if err != nil {
				if dpm.ctx.Err() != nil {
					return
				}
				log.Printf("[DVE][%s] Error receiving result message: %v", dpm.nodeRole, err)
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dpm.host.ID() {
				continue
			}

			var p2pMsg models.P2PMessage
			if err := json.Unmarshal(msg.Data, &p2pMsg); err != nil {
				log.Printf("[DVE][%s] Error unmarshaling result message: %v", dpm.nodeRole, err)
				continue
			}

			// Handle message based on type
			dpm.handleMessage(&p2pMsg)
		}
	}
}

// handleNodeAnnouncements handles incoming node announcements
func (dpm *DVEP2PManager) handleNodeAnnouncements() {
	for {
		select {
		case <-dpm.ctx.Done():
			return
		default:
			msg, err := dpm.nodeSub.Next(dpm.ctx)
			if err != nil {
				if dpm.ctx.Err() != nil {
					return
				}
				log.Printf("[DVE][%s] Error receiving node message: %v", dpm.nodeRole, err)
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dpm.host.ID() {
				continue
			}

			var p2pMsg models.P2PMessage
			if err := json.Unmarshal(msg.Data, &p2pMsg); err != nil {
				log.Printf("[DVE][%s] Error unmarshaling node message: %v", dpm.nodeRole, err)
				continue
			}

			// Handle message based on type
			dpm.handleMessage(&p2pMsg)
		}
	}
}

// handleMessage routes messages to appropriate handlers
func (dpm *DVEP2PManager) handleMessage(msg *models.P2PMessage) {
	dpm.mu.RLock()
	handler, exists := dpm.messageHandlers[msg.Type]
	dpm.mu.RUnlock()

	if exists {
		if err := handler.HandleMessage(dpm.ctx, msg); err != nil {
			log.Printf("[DVE][%s] Error handling message %s: %v", dpm.nodeRole, msg.Type, err)
		}
	} else {
		log.Printf("[DVE][%s] No handler for message type: %s", dpm.nodeRole, msg.Type)
	}
}

// discoverNodes discovers other DVE nodes in the network
func (dpm *DVEP2PManager) discoverNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dpm.ctx.Done():
			return
		case <-ticker.C:
			// Use DHT to find other DVE nodes
			dpm.findDVENodes()
		}
	}
}

// findDVENodes finds other DVE nodes using DHT
func (dpm *DVEP2PManager) findDVENodes() {
	// Implementation would use DHT to find providers of DVE resources
	// This is aligned with KNIRV-ORACLE's discovery mechanism
	log.Printf("[DVE][%s] Discovering DVE nodes...", dpm.nodeRole)

	// TODO: Implement DHT-based node discovery similar to KNIRV-ORACLE
	// This would involve:
	// 1. Creating a CID for DVE node resources
	// 2. Querying DHT for providers
	// 3. Connecting to discovered peers
}

// announceNode periodically announces this node to the network
func (dpm *DVEP2PManager) announceNode() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-dpm.ctx.Done():
			return
		case <-ticker.C:
			// Get current node info from database and announce
			dpm.announceCurrentNode()
		}
	}
}

// announceCurrentNode announces the current node status
func (dpm *DVEP2PManager) announceCurrentNode() {
	// TODO: Get current node info from database and announce
	log.Printf("[DVE][%s] Announcing node to network...", dpm.nodeRole)
}

// sendHeartbeat sends periodic heartbeat messages
func (dpm *DVEP2PManager) sendHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dpm.ctx.Done():
			return
		case <-ticker.C:
			dpm.broadcastHeartbeat()
		}
	}
}

// broadcastHeartbeat broadcasts a heartbeat message
func (dpm *DVEP2PManager) broadcastHeartbeat() {
	heartbeat := map[string]interface{}{
		"node_id":   dpm.host.ID().String(),
		"role":      dpm.nodeRole,
		"timestamp": time.Now(),
		"status":    "online",
	}

	message := &models.P2PMessage{
		ID:        fmt.Sprintf("heartbeat-%d", time.Now().Unix()),
		Type:      MessageTypeNodeHeartbeat,
		From:      dpm.host.ID().String(),
		Topic:     DVENodeTopic,
		Payload:   heartbeat,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[DVE][%s] Error marshaling heartbeat: %v", dpm.nodeRole, err)
		return
	}

	if err := dpm.nodeTopic.Publish(dpm.ctx, data); err != nil {
		log.Printf("[DVE][%s] Error publishing heartbeat: %v", dpm.nodeRole, err)
	}
}

// GetConnectedPeers returns information about connected peers
func (dpm *DVEP2PManager) GetConnectedPeers() []models.PeerInfo {
	peers := dpm.host.Network().Peers()
	peerInfos := make([]models.PeerInfo, 0, len(peers))

	for _, peerID := range peers {
		conns := dpm.host.Network().ConnsToPeer(peerID)
		if len(conns) > 0 {
			peerInfo := models.PeerInfo{
				ID:       peerID.String(),
				Address:  conns[0].RemoteMultiaddr().String(),
				Status:   "connected",
				LastSeen: time.Now(),
			}
			peerInfos = append(peerInfos, peerInfo)
		}
	}

	return peerInfos
}

// GetNetworkTopology returns the current network topology
func (dpm *DVEP2PManager) GetNetworkTopology() *models.NetworkTopology {
	peers := dpm.GetConnectedPeers()

	topology := &models.NetworkTopology{
		ID:             fmt.Sprintf("topology-%d", time.Now().Unix()),
		TotalPeers:     len(peers),
		ConnectedPeers: len(peers),
		Peers:          peers,
		Connections:    []models.ConnectionInfo{}, // TODO: Implement connection details
		Timestamp:      time.Now(),
	}

	return topology
}
