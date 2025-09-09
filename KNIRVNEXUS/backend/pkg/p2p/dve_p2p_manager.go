package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-backend/internal/models"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/tidwall/buntdb"
)

const (
	// DHT configuration
	NetworkControlTopic    = "network-control"
	DVEAnnouncementTopic   = "dve-announcements"
	ValidationRequestTopic = "validation-requests"

	// Network pause timeout
	NetworkPauseTimeout = 30 * time.Minute
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

	// Network control
	networkControlTopic *pubsub.Topic
	networkControlSub   *pubsub.Subscription
	networkPaused       bool
	pausedUntil         time.Time
	pauseMutex          sync.RWMutex

	// DVE announcements
	dveAnnouncementTopic *pubsub.Topic
	dveAnnouncementSub   *pubsub.Subscription

	nodeRole  string // "dve-validator", "dve-observer", "dve-coordinator", "dve-manager"
	chainID   string
	serviceID string

	// Message handlers
	messageHandlers map[string]MessageHandler
	mu              sync.RWMutex
}

// MessageHandler defines the interface for handling P2P messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *models.P2PMessage) error
}

// NetworkControlMessage represents network control messages
type NetworkControlMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NetworkPausePayload represents network pause message payload
type NetworkPausePayload struct {
	InitiatorPeerID string `json:"initiator_peer_id"`
	Reason          string `json:"reason"`
	Timestamp       int64  `json:"timestamp"`
}

// DVEAnnouncementMessage represents DVE-related announcements
type DVEAnnouncementMessage struct {
	Type      string      `json:"type"`   // "validation_request", "validation_result", "node_status"
	Action    string      `json:"action"` // "available", "processing", "completed", "failed"
	ServiceID string      `json:"service_id"`
	ChainID   string      `json:"chain_id"`
	NodeRole  string      `json:"node_role"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// ValidationRequestData represents validation request announcement data
type ValidationRequestData struct {
	RequestID    string            `json:"request_id"`
	Type         string            `json:"type"`
	Priority     int               `json:"priority"`
	Requirements map[string]string `json:"requirements"`
	Metadata     map[string]string `json:"metadata"`
}

// ValidationResultData represents validation result announcement data
type ValidationResultData struct {
	RequestID string            `json:"request_id"`
	Result    string            `json:"result"`
	Score     float64           `json:"score"`
	Evidence  map[string]string `json:"evidence"`
	Metadata  map[string]string `json:"metadata"`
}

// NodeStatusData represents node status announcement data
type NodeStatusData struct {
	NodeID       string            `json:"node_id"`
	Status       string            `json:"status"`
	Capabilities []string          `json:"capabilities"`
	Load         float64           `json:"load"`
	Metadata     map[string]string `json:"metadata"`
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

	serviceID := fmt.Sprintf("knirvnexus-%s-%s", chainID, nodeRole)

	manager := &DVEP2PManager{
		host:            host,
		dht:             dhtInstance,
		pubsub:          ps,
		db:              db,
		ctx:             ctx,
		cancel:          cancel,
		nodeRole:        nodeRole,
		chainID:         chainID,
		serviceID:       serviceID,
		messageHandlers: make(map[string]MessageHandler),
		networkPaused:   false,
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

	// Join network control topic
	dpm.networkControlTopic, err = dpm.pubsub.Join(NetworkControlTopic)
	if err != nil {
		return fmt.Errorf("failed to join network control topic: %w", err)
	}

	// Subscribe to network control messages
	dpm.networkControlSub, err = dpm.networkControlTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to network control topic: %w", err)
	}

	// Join DVE announcement topic
	dpm.dveAnnouncementTopic, err = dpm.pubsub.Join(DVEAnnouncementTopic)
	if err != nil {
		return fmt.Errorf("failed to join DVE announcement topic: %w", err)
	}

	// Subscribe to DVE announcements
	dpm.dveAnnouncementSub, err = dpm.dveAnnouncementTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to DVE announcement topic: %w", err)
	}

	log.Printf("[DVE][%s] P2P topics initialized: %s, %s, %s, %s, %s",
		dpm.nodeRole, validationTopicName, resultTopicName, nodeTopicName,
		NetworkControlTopic, DVEAnnouncementTopic)

	return nil
}

// Start starts the P2P manager
func (dpm *DVEP2PManager) Start() {
	log.Printf("[DVE][%s] Starting P2P manager...", dpm.nodeRole)

	// Start message handlers
	go dpm.handleValidationRequests()
	go dpm.handleValidationResults()
	go dpm.handleNodeAnnouncements()
	go dpm.handleNetworkControl()
	go dpm.handleDVEAnnouncements()

	// Start node discovery
	go dpm.discoverNodes()

	// Announce this node to the network
	go dpm.announceNode()

	// Announce this service to the DHT
	go dpm.announceServiceToDHT()

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
	log.Printf("[DVE][%s] Discovering DVE nodes...", dpm.nodeRole)

	// Create a CID for DVE node resources
	cid, err := dpm.createCIDFromServiceID("knirvnexus-dve")
	if err != nil {
		log.Printf("[DVE][%s] Failed to create CID for DVE discovery: %v", dpm.nodeRole, err)
		return
	}

	// Find providers for DVE services
	ctx, cancel := context.WithTimeout(dpm.ctx, 30*time.Second)
	defer cancel()

	providerChan := dpm.dht.FindProvidersAsync(ctx, cid, 20)

	var discoveredPeers []peer.AddrInfo
	for provider := range providerChan {
		discoveredPeers = append(discoveredPeers, provider)
	}

	log.Printf("[DVE][%s] Discovered %d DVE nodes", dpm.nodeRole, len(discoveredPeers))

	// Connect to discovered peers
	for _, peerInfo := range discoveredPeers {
		if peerInfo.ID == dpm.host.ID() {
			continue // Skip ourselves
		}

		// Try to connect to the peer
		connectCtx, connectCancel := context.WithTimeout(dpm.ctx, 15*time.Second)
		err := dpm.host.Connect(connectCtx, peerInfo)
		connectCancel()

		if err != nil {
			log.Printf("[DVE][%s] Failed to connect to DVE node %s: %v", dpm.nodeRole, peerInfo.ID, err)
		} else {
			log.Printf("[DVE][%s] Connected to DVE node: %s", dpm.nodeRole, peerInfo.ID)
		}
	}
}

// createCIDFromServiceID creates a CID from a service ID
func (dpm *DVEP2PManager) createCIDFromServiceID(serviceID string) (cid.Cid, error) {
	// Create a multihash from the service ID
	hash, err := multihash.Sum([]byte(serviceID), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}

	// Create a CID from the multihash
	return cid.NewCidV1(cid.Raw, hash), nil
}

// announceServiceToDHT announces this DVE service to the DHT
func (dpm *DVEP2PManager) announceServiceToDHT() {
	// Create a CID from the service ID
	cid, err := dpm.createCIDFromServiceID("knirvnexus-dve")
	if err != nil {
		log.Printf("[DVE][%s] Failed to create CID for service announcement: %v", dpm.nodeRole, err)
		return
	}

	// Announce that this node provides the DVE service
	if err := dpm.dht.Provide(dpm.ctx, cid, true); err != nil {
		log.Printf("[DVE][%s] Failed to announce DVE service: %v", dpm.nodeRole, err)
		return
	}

	log.Printf("[DVE][%s] Announced DVE service to DHT", dpm.nodeRole)
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

// handleNetworkControl processes network control messages (pause/resume)
func (dpm *DVEP2PManager) handleNetworkControl() {
	for {
		select {
		case <-dpm.ctx.Done():
			return
		default:
			msg, err := dpm.networkControlSub.Next(dpm.ctx)
			if err != nil {
				if dpm.ctx.Err() == nil {
					log.Printf("[DVE][%s] Error receiving network control message: %v", dpm.nodeRole, err)
				}
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dpm.host.ID() {
				continue
			}

			// Decode the network control message
			var networkMsg NetworkControlMessage
			if err := json.Unmarshal(msg.Data, &networkMsg); err != nil {
				log.Printf("[DVE][%s] Error decoding network control message: %v", dpm.nodeRole, err)
				continue
			}

			log.Printf("[DVE][%s] Received network control message: %s", dpm.nodeRole, networkMsg.Type)

			// Handle different network control message types
			switch networkMsg.Type {
			case "NetworkPause":
				dpm.handleNetworkPause(networkMsg.Payload, msg.ReceivedFrom.String())
			case "NetworkResume":
				dpm.handleNetworkResume(networkMsg.Payload, msg.ReceivedFrom.String())
			default:
				log.Printf("[DVE][%s] Unknown network control message type: %s", dpm.nodeRole, networkMsg.Type)
			}
		}
	}
}

// handleNetworkPause processes network pause messages
func (dpm *DVEP2PManager) handleNetworkPause(payload interface{}, senderPeerID string) {
	// Parse the pause payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DVE][%s] Error marshaling pause payload: %v", dpm.nodeRole, err)
		return
	}

	var pausePayload NetworkPausePayload
	if err := json.Unmarshal(payloadBytes, &pausePayload); err != nil {
		log.Printf("[DVE][%s] Error unmarshaling pause payload: %v", dpm.nodeRole, err)
		return
	}

	initiatorPeerID := pausePayload.InitiatorPeerID
	reason := pausePayload.Reason

	// Set network paused state
	dpm.pauseMutex.Lock()
	dpm.networkPaused = true
	dpm.pausedUntil = time.Now().Add(NetworkPauseTimeout)
	dpm.pauseMutex.Unlock()

	log.Printf("[DVE][%s] Network PAUSED by %s until %s - Reason: %s",
		dpm.nodeRole,
		initiatorPeerID,
		dpm.pausedUntil.Format("2006-01-02 15:04:05 UTC"),
		reason)

	log.Printf("[DVE][%s] DVE operations paused - rejecting new validation requests", dpm.nodeRole)
}

// handleNetworkResume processes network resume messages
func (dpm *DVEP2PManager) handleNetworkResume(payload interface{}, senderPeerID string) {
	dpm.pauseMutex.Lock()
	dpm.networkPaused = false
	dpm.pauseMutex.Unlock()

	log.Printf("[DVE][%s] Network RESUMED by %s", dpm.nodeRole, senderPeerID)
}

// IsNetworkPaused returns whether the network is currently paused
func (dpm *DVEP2PManager) IsNetworkPaused() bool {
	dpm.pauseMutex.RLock()
	defer dpm.pauseMutex.RUnlock()

	if dpm.networkPaused && time.Now().After(dpm.pausedUntil) {
		// Pause has expired
		dpm.pauseMutex.RUnlock()
		dpm.pauseMutex.Lock()
		dpm.networkPaused = false
		dpm.pauseMutex.Unlock()
		dpm.pauseMutex.RLock()
		log.Printf("[DVE][%s] Network pause expired", dpm.nodeRole)
	}

	return dpm.networkPaused
}

// handleDVEAnnouncements processes DVE-related announcements from other nodes
func (dpm *DVEP2PManager) handleDVEAnnouncements() {
	for {
		select {
		case <-dpm.ctx.Done():
			return
		default:
			msg, err := dpm.dveAnnouncementSub.Next(dpm.ctx)
			if err != nil {
				if dpm.ctx.Err() == nil {
					log.Printf("[DVE][%s] Error receiving DVE announcement: %v", dpm.nodeRole, err)
				}
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dpm.host.ID() {
				continue
			}

			// Decode the DVE announcement message
			var dveMsg DVEAnnouncementMessage
			if err := json.Unmarshal(msg.Data, &dveMsg); err != nil {
				log.Printf("[DVE][%s] Error decoding DVE announcement: %v", dpm.nodeRole, err)
				continue
			}

			log.Printf("[DVE][%s] Received %s %s announcement from %s",
				dpm.nodeRole, dveMsg.Action, dveMsg.Type, dveMsg.ServiceID)

			// Process the announcement based on type
			switch dveMsg.Type {
			case "validation_request":
				dpm.processValidationRequestAnnouncement(dveMsg)
			case "validation_result":
				dpm.processValidationResultAnnouncement(dveMsg)
			case "node_status":
				dpm.processNodeStatusAnnouncement(dveMsg)
			default:
				log.Printf("[DVE][%s] Unknown DVE announcement type: %s", dpm.nodeRole, dveMsg.Type)
			}
		}
	}
}

// processValidationRequestAnnouncement processes validation request announcements
func (dpm *DVEP2PManager) processValidationRequestAnnouncement(msg DVEAnnouncementMessage) {
	log.Printf("[DVE][%s] Processing validation request %s from %s", dpm.nodeRole, msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVNEXUS validation request processing logic
}

// processValidationResultAnnouncement processes validation result announcements
func (dpm *DVEP2PManager) processValidationResultAnnouncement(msg DVEAnnouncementMessage) {
	log.Printf("[DVE][%s] Processing validation result %s from %s", dpm.nodeRole, msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVNEXUS validation result processing logic
}

// processNodeStatusAnnouncement processes node status announcements
func (dpm *DVEP2PManager) processNodeStatusAnnouncement(msg DVEAnnouncementMessage) {
	log.Printf("[DVE][%s] Processing node status %s from %s", dpm.nodeRole, msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVNEXUS node status processing logic
}

// AnnounceValidationRequest announces a new validation request available for processing
func (dpm *DVEP2PManager) AnnounceValidationRequest(requestID, requestType string, priority int, requirements, metadata map[string]string) error {
	if dpm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce validation request")
	}

	requestData := ValidationRequestData{
		RequestID:    requestID,
		Type:         requestType,
		Priority:     priority,
		Requirements: requirements,
		Metadata:     metadata,
	}

	announcement := DVEAnnouncementMessage{
		Type:      "validation_request",
		Action:    "available",
		ServiceID: dpm.serviceID,
		ChainID:   dpm.chainID,
		NodeRole:  dpm.nodeRole,
		Data:      requestData,
		Timestamp: time.Now().Unix(),
	}

	return dpm.publishDVEAnnouncement(announcement)
}

// AnnounceValidationResult announces a validation result
func (dpm *DVEP2PManager) AnnounceValidationResult(requestID, result string, score float64, evidence, metadata map[string]string) error {
	if dpm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce validation result")
	}

	resultData := ValidationResultData{
		RequestID: requestID,
		Result:    result,
		Score:     score,
		Evidence:  evidence,
		Metadata:  metadata,
	}

	announcement := DVEAnnouncementMessage{
		Type:      "validation_result",
		Action:    "completed",
		ServiceID: dpm.serviceID,
		ChainID:   dpm.chainID,
		NodeRole:  dpm.nodeRole,
		Data:      resultData,
		Timestamp: time.Now().Unix(),
	}

	return dpm.publishDVEAnnouncement(announcement)
}

// AnnounceNodeStatus announces the current node status
func (dpm *DVEP2PManager) AnnounceNodeStatus(nodeID, status string, capabilities []string, load float64, metadata map[string]string) error {
	if dpm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce node status")
	}

	statusData := NodeStatusData{
		NodeID:       nodeID,
		Status:       status,
		Capabilities: capabilities,
		Load:         load,
		Metadata:     metadata,
	}

	announcement := DVEAnnouncementMessage{
		Type:      "node_status",
		Action:    "updated",
		ServiceID: dpm.serviceID,
		ChainID:   dpm.chainID,
		NodeRole:  dpm.nodeRole,
		Data:      statusData,
		Timestamp: time.Now().Unix(),
	}

	return dpm.publishDVEAnnouncement(announcement)
}

// publishDVEAnnouncement publishes a DVE announcement to the DHT
func (dpm *DVEP2PManager) publishDVEAnnouncement(announcement DVEAnnouncementMessage) error {
	msgBytes, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal DVE announcement: %w", err)
	}

	if err := dpm.dveAnnouncementTopic.Publish(dpm.ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish DVE announcement: %w", err)
	}

	log.Printf("[DVE][%s] Published %s %s announcement",
		dpm.nodeRole, announcement.Action, announcement.Type)
	return nil
}
