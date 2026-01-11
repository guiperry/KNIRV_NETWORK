package p2p

import (
	"backend_server/internal/objects"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
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

	// Peer reputation constants
	MaxReputationScore     = 100.0
	MinReputationScore     = 0.0
	DefaultReputationScore = 50.0
	ReputationDecayRate    = 0.95 // Daily decay factor
	BadBehaviorPenalty     = 10.0
	GoodBehaviorReward     = 1.0

	// Message encryption
	MessageEncryptionProtocol = "/knirv/p2p-encryption/1.0.0"
)

// Bootstrap nodes for KNIRV network
var DefaultBootstrapPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

// STUN servers for NAT traversal
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun3.l.google.com:19302",
	"stun4.l.google.com:19302",
}

// TURN servers for NAT traversal (fallback)
var DefaultTURNServers = []string{
	"turn:turn.example.com:3478", // Placeholder - should be configured
}

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

	// P2P Security Integration
	p2pSecurityService *P2PService // eBPF/XDP firewall integration
}

// MessageHandler defines the interface for handling P2P messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *objects.P2PMessage) error
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

// NewDVEP2PManager creates a new DVE P2P manager with integrated security
func NewDVEP2PManager(chainID, nodeRole string, db *buntdb.DB, dhtEnabled bool, securityService *P2PService) (*DVEP2PManager, error) {
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

	// Create DHT for node discovery (conditionally based on dhtEnabled)
	var dhtInstance *dht.IpfsDHT
	if dhtEnabled {
		dhtInstance, err = dht.New(ctx, host)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create DHT: %w", err)
		}
	}

	// Create GossipSub for message distribution
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	serviceID := fmt.Sprintf("knirvnexus-%s-%s", chainID, nodeRole)

	manager := &DVEP2PManager{
		host:               host,
		dht:                dhtInstance,
		pubsub:             ps,
		db:                 db,
		ctx:                ctx,
		cancel:             cancel,
		nodeRole:           nodeRole,
		chainID:            chainID,
		serviceID:          serviceID,
		messageHandlers:    make(map[string]MessageHandler),
		networkPaused:      false,
		p2pSecurityService: securityService, // Integrate eBPF/XDP firewall
	}

	// Register network event handlers for security integration
	host.Network().Notify(manager.createNetworkNotifiee())

	// Initialize topics and subscriptions
	if err := manager.setupTopics(); err != nil {
		cancel()
		return nil, err
	}

	return manager, nil
}

// createNetworkNotifiee creates a network.Notifiee for handling peer connections/disconnections
func (dpm *DVEP2PManager) createNetworkNotifiee() *network.NotifyBundle {
	return &network.NotifyBundle{
		ConnectedF:    dpm.handlePeerConnected,
		DisconnectedF: dpm.handlePeerDisconnected,
	}
}

// handlePeerConnected is called when a peer connects (security integration)
func (dpm *DVEP2PManager) handlePeerConnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	remoteAddr := conn.RemoteMultiaddr()

	// Extract IP address from multiaddr for eBPF whitelist
	ipAddr := extractIPFromMultiaddr(remoteAddr)
	if ipAddr != "" && dpm.p2pSecurityService != nil {
		log.Printf("[DVE][%s] Peer connected: %s, IP: %s. Adding to eBPF whitelist.", dpm.nodeRole, peerID.String(), ipAddr)
		// Call the security service to add the peer's IP to the whitelist
		if err := dpm.p2pSecurityService.OnPeerConnected(peerID.String(), ipAddr); err != nil {
			log.Printf("[DVE][%s] Error adding peer %s (%s) to eBPF whitelist: %v", dpm.nodeRole, peerID.String(), ipAddr, err)
		}
	} else {
		log.Printf("[DVE][%s] Could not extract IP from multiaddr for peer %s: %s", dpm.nodeRole, peerID.String(), remoteAddr.String())
	}
}

// handlePeerDisconnected is called when a peer disconnects (security integration)
func (dpm *DVEP2PManager) handlePeerDisconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	remoteAddr := conn.RemoteMultiaddr()

	ipAddr := extractIPFromMultiaddr(remoteAddr)
	if ipAddr != "" && dpm.p2pSecurityService != nil {
		log.Printf("[DVE][%s] Peer disconnected: %s, IP: %s. Removing from eBPF whitelist.", dpm.nodeRole, peerID.String(), ipAddr)
		// Call the security service to remove the peer's IP from the whitelist
		if err := dpm.p2pSecurityService.OnPeerDisconnected(peerID.String(), ipAddr); err != nil {
			log.Printf("[DVE][%s] Error removing peer %s (%s) from eBPF whitelist: %v", dpm.nodeRole, peerID.String(), ipAddr, err)
		}
	} else {
		log.Printf("[DVE][%s] Could not extract IP from multiaddr for peer %s: %s", dpm.nodeRole, peerID.String(), remoteAddr.String())
	}
}

// extractIPFromMultiaddr extracts an IP address from a multiaddr
func extractIPFromMultiaddr(addr multiaddr.Multiaddr) string {
	// Iterate through protocols to find IP address
	for _, p := range addr.Protocols() {
		if p.Code == multiaddr.P_IP4 || p.Code == multiaddr.P_IP6 {
			v, err := addr.ValueForProtocol(p.Code)
			if err == nil {
				return v
			}
		}
	}
	return ""
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

	// Bootstrap to the network
	if err := dpm.bootstrapToNetwork(); err != nil {
		log.Printf("[DVE][%s] Bootstrap failed: %v", dpm.nodeRole, err)
		// Continue anyway - we might still connect via other means
	}

	// Configure NAT traversal
	go dpm.setupNATTraversal()

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

	// Start reputation management
	go dpm.managePeerReputation()

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

// Close is an alias for Stop for compatibility with io.Closer and other interfaces
func (dpm *DVEP2PManager) Close() error {
	dpm.Stop()
	return nil
}

// RegisterMessageHandler registers a message handler for a specific message type
func (dpm *DVEP2PManager) RegisterMessageHandler(messageType string, handler MessageHandler) {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()
	dpm.messageHandlers[messageType] = handler
}

// BroadcastValidationRequest broadcasts a validation request to the network
func (dpm *DVEP2PManager) BroadcastValidationRequest(req *objects.ValidationTask) error {
	message := &objects.P2PMessage{
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
func (dpm *DVEP2PManager) BroadcastValidationResult(result *objects.ValidationResult) error {
	message := &objects.P2PMessage{
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
func (dpm *DVEP2PManager) AnnounceNode(node *objects.DVENode) error {
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

	message := &objects.P2PMessage{
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

			var p2pMsg objects.P2PMessage
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

			var p2pMsg objects.P2PMessage
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

			var p2pMsg objects.P2PMessage
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
func (dpm *DVEP2PManager) handleMessage(msg *objects.P2PMessage) {
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
	// Skip DHT discovery if DHT is disabled
	if dpm.dht == nil {
		log.Printf("[DVE][%s] DHT disabled, skipping node discovery", dpm.nodeRole)
		return
	}

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
	// Skip DHT announcement if DHT is disabled
	if dpm.dht == nil {
		log.Printf("[DVE][%s] DHT disabled, skipping service announcement", dpm.nodeRole)
		return
	}

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
	// Get current node status and announce to network
	nodeStatus := map[string]interface{}{
		"node_id":       dpm.host.ID().String(),
		"role":          dpm.nodeRole,
		"chain_id":      dpm.chainID,
		"service_id":    dpm.serviceID,
		"status":        "active",
		"capabilities":  []string{"validation", "computation", "storage"},
		"timestamp":     time.Now().Unix(),
		"version":       "1.0.0",
	}

	message := &objects.P2PMessage{
		ID:        fmt.Sprintf("status-%d", time.Now().Unix()),
		Type:      MessageTypeNodeAnnouncement,
		From:      dpm.host.ID().String(),
		Topic:     DVENodeTopic,
		Payload:   nodeStatus,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[DVE][%s] Error marshaling node status: %v", dpm.nodeRole, err)
		return
	}

	if err := dpm.nodeTopic.Publish(dpm.ctx, data); err != nil {
		log.Printf("[DVE][%s] Error publishing node status: %v", dpm.nodeRole, err)
	} else {
		log.Printf("[DVE][%s] Node status announced to network", dpm.nodeRole)
	}
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

	message := &objects.P2PMessage{
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
func (dpm *DVEP2PManager) GetConnectedPeers() []objects.PeerInfo {
	peers := dpm.host.Network().Peers()
	peerInfos := make([]objects.PeerInfo, 0, len(peers))

	for _, peerID := range peers {
		conns := dpm.host.Network().ConnsToPeer(peerID)
		if len(conns) > 0 {
			peerInfo := objects.PeerInfo{
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
func (dpm *DVEP2PManager) GetNetworkTopology() *objects.NetworkTopology {
	peers := dpm.GetConnectedPeers()

	topology := &objects.NetworkTopology{
		ID:             fmt.Sprintf("topology-%d", time.Now().Unix()),
		TotalPeers:     len(peers),
		ConnectedPeers: len(peers),
		Peers:          peers,
		Connections:    []objects.ConnectionInfo{}, // TODO: Implement connection details
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
	// Log the pause request from sender
	log.Printf("[DVE][%s] Received network pause request from peer: %s", dpm.nodeRole, senderPeerID)
	
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
	// Log the resume request from sender with payload details
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DVE][%s] Error marshaling resume payload: %v", dpm.nodeRole, err)
	} else {
		log.Printf("[DVE][%s] Received network resume request from peer: %s, payload: %s", dpm.nodeRole, senderPeerID, string(payloadBytes))
	}
	
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

	// Extract validation request data
	var requestData ValidationRequestData
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if reqID, ok := data["request_id"].(string); ok {
			requestData.RequestID = reqID
		}
		if reqType, ok := data["type"].(string); ok {
			requestData.Type = reqType
		}
		if priority, ok := data["priority"].(float64); ok {
			requestData.Priority = int(priority)
		}
		// Extract requirements and metadata maps
		if reqs, ok := data["requirements"].(map[string]interface{}); ok {
			requestData.Requirements = make(map[string]string)
			for k, v := range reqs {
				if str, ok := v.(string); ok {
					requestData.Requirements[k] = str
				}
			}
		}
		if meta, ok := data["metadata"].(map[string]interface{}); ok {
			requestData.Metadata = make(map[string]string)
			for k, v := range meta {
				if str, ok := v.(string); ok {
					requestData.Metadata[k] = str
				}
			}
		}
	}

	// Route to appropriate message handler for processing
	dpm.mu.RLock()
	handler, exists := dpm.messageHandlers[MessageTypeValidationRequest]
	dpm.mu.RUnlock()

	if exists {
		p2pMsg := &objects.P2PMessage{
			ID:        requestData.RequestID,
			Type:      MessageTypeValidationRequest,
			From:      msg.ServiceID,
			Topic:     DVEValidationTopic,
			Payload:   map[string]interface{}{"request_data": requestData},
			Timestamp: time.Unix(msg.Timestamp, 0),
		}

		if err := handler.HandleMessage(dpm.ctx, p2pMsg); err != nil {
			log.Printf("[DVE][%s] Error handling validation request announcement: %v", dpm.nodeRole, err)
		}
	} else {
		log.Printf("[DVE][%s] No handler for validation request announcements", dpm.nodeRole)
	}
}

// processValidationResultAnnouncement processes validation result announcements
func (dpm *DVEP2PManager) processValidationResultAnnouncement(msg DVEAnnouncementMessage) {
	log.Printf("[DVE][%s] Processing validation result %s from %s", dpm.nodeRole, msg.Action, msg.ServiceID)

	// Extract validation result data
	var resultData ValidationResultData
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if reqID, ok := data["request_id"].(string); ok {
			resultData.RequestID = reqID
		}
		if result, ok := data["result"].(string); ok {
			resultData.Result = result
		}
		if score, ok := data["score"].(float64); ok {
			resultData.Score = score
		}
		// Extract evidence and metadata maps
		if evidence, ok := data["evidence"].(map[string]interface{}); ok {
			resultData.Evidence = make(map[string]string)
			for k, v := range evidence {
				if str, ok := v.(string); ok {
					resultData.Evidence[k] = str
				}
			}
		}
		if meta, ok := data["metadata"].(map[string]interface{}); ok {
			resultData.Metadata = make(map[string]string)
			for k, v := range meta {
				if str, ok := v.(string); ok {
					resultData.Metadata[k] = str
				}
			}
		}
	}

	// Route to appropriate message handler for processing
	dpm.mu.RLock()
	handler, exists := dpm.messageHandlers[MessageTypeValidationResult]
	dpm.mu.RUnlock()

	if exists {
		p2pMsg := &objects.P2PMessage{
			ID:        resultData.RequestID,
			Type:      MessageTypeValidationResult,
			From:      msg.ServiceID,
			Topic:     DVEResultTopic,
			Payload:   map[string]interface{}{"result_data": resultData},
			Timestamp: time.Unix(msg.Timestamp, 0),
		}

		if err := handler.HandleMessage(dpm.ctx, p2pMsg); err != nil {
			log.Printf("[DVE][%s] Error handling validation result announcement: %v", dpm.nodeRole, err)
		}
	} else {
		log.Printf("[DVE][%s] No handler for validation result announcements", dpm.nodeRole)
	}
}

// processNodeStatusAnnouncement processes node status announcements
func (dpm *DVEP2PManager) processNodeStatusAnnouncement(msg DVEAnnouncementMessage) {
	log.Printf("[DVE][%s] Processing node status %s from %s", dpm.nodeRole, msg.Action, msg.ServiceID)

	// Extract node status data
	var statusData NodeStatusData
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if nodeID, ok := data["node_id"].(string); ok {
			statusData.NodeID = nodeID
		}
		if status, ok := data["status"].(string); ok {
			statusData.Status = status
		}
		if load, ok := data["load"].(float64); ok {
			statusData.Load = load
		}
		// Extract capabilities slice
		if caps, ok := data["capabilities"].([]interface{}); ok {
			statusData.Capabilities = make([]string, len(caps))
			for i, cap := range caps {
				if str, ok := cap.(string); ok {
					statusData.Capabilities[i] = str
				}
			}
		}
		// Extract metadata map
		if meta, ok := data["metadata"].(map[string]interface{}); ok {
			statusData.Metadata = make(map[string]string)
			for k, v := range meta {
				if str, ok := v.(string); ok {
					statusData.Metadata[k] = str
				}
			}
		}
	}

	// Update peer reputation based on status
	switch statusData.Status {
	case "active":
		dpm.UpdatePeerReputation(msg.ServiceID, "node_active", true)
	case "inactive":
		dpm.UpdatePeerReputation(msg.ServiceID, "node_inactive", false)
	}

	// Route to appropriate message handler for processing
	dpm.mu.RLock()
	handler, exists := dpm.messageHandlers[MessageTypeNodeAnnouncement]
	dpm.mu.RUnlock()

	if exists {
		p2pMsg := &objects.P2PMessage{
			ID:        statusData.NodeID,
			Type:      MessageTypeNodeAnnouncement,
			From:      msg.ServiceID,
			Topic:     DVENodeTopic,
			Payload:   map[string]interface{}{"status_data": statusData},
			Timestamp: time.Unix(msg.Timestamp, 0),
		}

		if err := handler.HandleMessage(dpm.ctx, p2pMsg); err != nil {
			log.Printf("[DVE][%s] Error handling node status announcement: %v", dpm.nodeRole, err)
		}
	} else {
		log.Printf("[DVE][%s] No handler for node status announcements", dpm.nodeRole)
	}
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

// bootstrapToNetwork connects to bootstrap nodes and initializes DHT
func (dpm *DVEP2PManager) bootstrapToNetwork() error {
	log.Printf("[DVE][%s] Bootstrapping to KNIRV network...", dpm.nodeRole)

	// Parse bootstrap peer addresses
	var bootstrapPeers []peer.AddrInfo
	for _, addrStr := range DefaultBootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("[DVE][%s] Invalid bootstrap address %s: %v", dpm.nodeRole, addrStr, err)
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("[DVE][%s] Failed to parse bootstrap peer %s: %v", dpm.nodeRole, addrStr, err)
			continue
		}

		bootstrapPeers = append(bootstrapPeers, *peerInfo)
	}

	if len(bootstrapPeers) == 0 {
		return fmt.Errorf("no valid bootstrap peers available")
	}

	log.Printf("[DVE][%s] Connecting to %d bootstrap peers...", dpm.nodeRole, len(bootstrapPeers))

	// Connect to bootstrap peers with timeout
	connectCtx, cancel := context.WithTimeout(dpm.ctx, 30*time.Second)
	defer cancel()

	connectedCount := 0
	for _, peerInfo := range bootstrapPeers {
		if peerInfo.ID == dpm.host.ID() {
			continue // Skip ourselves
		}

		if err := dpm.host.Connect(connectCtx, peerInfo); err != nil {
			log.Printf("[DVE][%s] Failed to connect to bootstrap peer %s: %v", dpm.nodeRole, peerInfo.ID, err)
		} else {
			log.Printf("[DVE][%s] Connected to bootstrap peer: %s", dpm.nodeRole, peerInfo.ID)
			connectedCount++
		}
	}

	if connectedCount == 0 {
		log.Printf("[DVE][%s] Warning: Failed to connect to any bootstrap peers", dpm.nodeRole)
	} else {
		log.Printf("[DVE][%s] Successfully connected to %d/%d bootstrap peers", dpm.nodeRole, connectedCount, len(bootstrapPeers))
	}

	// Bootstrap the DHT if enabled
	if dpm.dht != nil {
		if err := dpm.dht.Bootstrap(dpm.ctx); err != nil {
			return fmt.Errorf("failed to bootstrap DHT: %w", err)
		}
		log.Printf("[DVE][%s] DHT bootstrapped successfully", dpm.nodeRole)
	} else {
		log.Printf("[DVE][%s] DHT disabled, skipping bootstrap", dpm.nodeRole)
	}
	return nil
}

// setupNATTraversal configures NAT traversal using STUN/TURN servers
func (dpm *DVEP2PManager) setupNATTraversal() {
	log.Printf("[DVE][%s] Setting up NAT traversal...", dpm.nodeRole)

	// For now, log the STUN/TURN server configuration
	// In a full implementation, this would integrate with libp2p's NAT traversal
	log.Printf("[DVE][%s] STUN servers configured: %v", dpm.nodeRole, DefaultSTUNServers)
	log.Printf("[DVE][%s] TURN servers configured: %v", dpm.nodeRole, DefaultTURNServers)

	// TODO: Implement actual STUN/TURN client integration
	// This would involve:
	// 1. STUN client to discover public IP and NAT type
	// 2. TURN client for relay when direct connection fails
	// 3. UPnP integration for port forwarding
	// 4. Circuit relay configuration for full NAT traversal

	log.Printf("[DVE][%s] NAT traversal setup completed (basic configuration)", dpm.nodeRole)
}

// managePeerReputation manages peer reputation scores and blacklisting
func (dpm *DVEP2PManager) managePeerReputation() {
	ticker := time.NewTicker(1 * time.Hour) // Daily reputation updates
	defer ticker.Stop()

	for {
		select {
		case <-dpm.ctx.Done():
			return
		case <-ticker.C:
			dpm.updatePeerReputations()
			dpm.enforceReputationBlacklist()
		}
	}
}

// updatePeerReputations updates reputation scores for all known peers
func (dpm *DVEP2PManager) updatePeerReputations() {
	log.Printf("[DVE][%s] Updating peer reputations...", dpm.nodeRole)

	// Get all connected peers
	peers := dpm.host.Network().Peers()

	for _, peerID := range peers {
		reputation := dpm.getPeerReputation(peerID.String())

		// Apply reputation decay
		reputation *= ReputationDecayRate

		// Ensure reputation stays within bounds
		if reputation < MinReputationScore {
			reputation = MinReputationScore
		}
		if reputation > MaxReputationScore {
			reputation = MaxReputationScore
		}

		// Store updated reputation
		dpm.setPeerReputation(peerID.String(), reputation)
	}

	log.Printf("[DVE][%s] Updated reputations for %d peers", dpm.nodeRole, len(peers))
}

// enforceReputationBlacklist disconnects from peers with very low reputation
func (dpm *DVEP2PManager) enforceReputationBlacklist() {
	log.Printf("[DVE][%s] Enforcing reputation blacklist...", dpm.nodeRole)

	peers := dpm.host.Network().Peers()
	disconnectedCount := 0

	for _, peerID := range peers {
		reputation := dpm.getPeerReputation(peerID.String())

		// Disconnect from peers with reputation below threshold
		if reputation < MinReputationScore+10.0 {
			log.Printf("[DVE][%s] Disconnecting from low-reputation peer %s (score: %.2f)",
				dpm.nodeRole, peerID, reputation)

			if err := dpm.host.Network().ClosePeer(peerID); err != nil {
				log.Printf("[DVE][%s] Error disconnecting from peer %s: %v", dpm.nodeRole, peerID, err)
			} else {
				disconnectedCount++
			}
		}
	}

	if disconnectedCount > 0 {
		log.Printf("[DVE][%s] Disconnected from %d low-reputation peers", dpm.nodeRole, disconnectedCount)
	}
}

// getPeerReputation retrieves the reputation score for a peer
func (dpm *DVEP2PManager) getPeerReputation(peerID string) float64 {
	// In a real implementation, this would query the database
	// For now, return default reputation
	log.Printf("[DVE][%s] Getting reputation for peer: %s", dpm.nodeRole, peerID)
	return DefaultReputationScore
}

// setPeerReputation stores the reputation score for a peer
func (dpm *DVEP2PManager) setPeerReputation(peerID string, reputation float64) {
	// In a real implementation, this would store in the database
	log.Printf("[DVE][%s] Updated reputation for peer %s: %.2f", dpm.nodeRole, peerID, reputation)
}

// UpdatePeerReputation updates a peer's reputation based on behavior
func (dpm *DVEP2PManager) UpdatePeerReputation(peerID string, behavior string, positive bool) {
	currentRep := dpm.getPeerReputation(peerID)

	var delta float64
	if positive {
		delta = GoodBehaviorReward
	} else {
		delta = -BadBehaviorPenalty
	}

	newRep := currentRep + delta

	// Clamp to valid range
	if newRep < MinReputationScore {
		newRep = MinReputationScore
	}
	if newRep > MaxReputationScore {
		newRep = MaxReputationScore
	}

	dpm.setPeerReputation(peerID, newRep)

	log.Printf("[DVE][%s] Updated reputation for peer %s (%s): %.2f -> %.2f",
		dpm.nodeRole, peerID, behavior, currentRep, newRep)
}

// EncryptMessage encrypts a message for secure P2P communication
func (dpm *DVEP2PManager) EncryptMessage(message []byte, recipientPeerID string) ([]byte, error) {
	// Generate a random symmetric key for this message
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// In a real implementation, this would:
	// 1. Use recipient's public key to encrypt the symmetric key
	// 2. Encrypt the message with the symmetric key
	// 3. Return encrypted key + encrypted message

	// For now, return the message as-is (placeholder)
	log.Printf("[DVE][%s] Message encryption placeholder - would encrypt for peer %s", dpm.nodeRole, recipientPeerID)
	return message, nil
}

// DecryptMessage decrypts a received encrypted message
func (dpm *DVEP2PManager) DecryptMessage(encryptedMessage []byte) ([]byte, error) {
	// In a real implementation, this would:
	// 1. Extract encrypted symmetric key
	// 2. Decrypt symmetric key with our private key
	// 3. Decrypt message with symmetric key

	// For now, return the message as-is (placeholder)
	log.Printf("[DVE][%s] Message decryption placeholder", dpm.nodeRole)
	return encryptedMessage, nil
}

// OptimizeNetworkTopology optimizes the network topology by selecting better peers
func (dpm *DVEP2PManager) OptimizeNetworkTopology() {
	log.Printf("[DVE][%s] Optimizing network topology...", dpm.nodeRole)

	peers := dpm.host.Network().Peers()
	if len(peers) < 3 {
		log.Printf("[DVE][%s] Not enough peers for topology optimization (%d)", dpm.nodeRole, len(peers))
		return
	}

	// Calculate peer scores based on reputation, latency, and capabilities
	peerScores := make(map[string]float64)

	for _, peerID := range peers {
		score := dpm.calculatePeerScore(peerID)
		peerScores[peerID.String()] = score
	}

	// Sort peers by score and keep only the top N
	// In a real implementation, this would disconnect from low-scoring peers
	// and attempt to connect to higher-scoring discovered peers

	log.Printf("[DVE][%s] Topology optimization completed for %d peers", dpm.nodeRole, len(peers))
}

// calculatePeerScore calculates a score for peer selection
func (dpm *DVEP2PManager) calculatePeerScore(peerID peer.ID) float64 {
	reputation := dpm.getPeerReputation(peerID.String())

	// Get connection quality metrics
	conns := dpm.host.Network().ConnsToPeer(peerID)
	connectionScore := 0.0

	if len(conns) > 0 {
		// Higher score for longer-lived connections
		connectionAge := time.Since(conns[0].Stat().Opened)
		connectionScore = math.Min(connectionAge.Hours()/24.0, 1.0) * 20.0
	}

	// Combine scores (reputation has highest weight)
	totalScore := (reputation * 0.7) + (connectionScore * 0.3)

	return totalScore
}

// GetPeerStats returns statistics about peer connections and reputation
func (dpm *DVEP2PManager) GetPeerStats() map[string]interface{} {
	peers := dpm.host.Network().Peers()

	stats := map[string]interface{}{
		"total_peers":       len(peers),
		"connected_peers":   len(peers),
		"bootstrap_peers":   len(DefaultBootstrapPeers),
		"reputation_range":  fmt.Sprintf("%.1f-%.1f", MinReputationScore, MaxReputationScore),
		"average_reputation": DefaultReputationScore,
		"blacklisted_peers":  0, // Would be calculated from database
	}

	return stats
}
