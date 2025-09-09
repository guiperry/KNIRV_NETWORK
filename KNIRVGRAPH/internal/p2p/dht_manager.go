package p2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

const (
	// DHT configuration
	DiscoveryNamespace     = "knirvgraph"
	NetworkControlTopic    = "network-control"
	GraphAnnouncementTopic = "graph-announcements"

	// Network pause timeout
	NetworkPauseTimeout = 30 * time.Minute
)

// DHTManager handles the Distributed Hash Table functionality for KNIRVGRAPH
type DHTManager struct {
	host           host.Host
	kadDHT         *dht.IpfsDHT
	pubsub         *pubsub.PubSub
	ctx            context.Context
	cancel         context.CancelFunc
	bootstrapPeers []peer.AddrInfo
	mutex          sync.RWMutex

	// Network control
	networkControlTopic *pubsub.Topic
	networkControlSub   *pubsub.Subscription
	networkPaused       bool
	pausedUntil         time.Time
	pauseMutex          sync.RWMutex

	// Graph announcements
	graphTopic *pubsub.Topic
	graphSub   *pubsub.Subscription

	// Service identification
	serviceID string
	chainID   string
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

// GraphAnnouncementMessage represents graph-related announcements
type GraphAnnouncementMessage struct {
	Type      string      `json:"type"`   // "skill", "capability", "property"
	Action    string      `json:"action"` // "minted", "updated", "removed"
	ServiceID string      `json:"service_id"`
	ChainID   string      `json:"chain_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// SkillAnnouncementData represents skill announcement data
type SkillAnnouncementData struct {
	SkillID     string            `json:"skill_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Metadata    map[string]string `json:"metadata"`
}

// CapabilityAnnouncementData represents capability announcement data
type CapabilityAnnouncementData struct {
	CapabilityID string            `json:"capability_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Schema       interface{}       `json:"schema"`
	Metadata     map[string]string `json:"metadata"`
}

// PropertyAnnouncementData represents property announcement data
type PropertyAnnouncementData struct {
	PropertyID string            `json:"property_id"`
	Name       string            `json:"name"`
	Value      interface{}       `json:"value"`
	Type       string            `json:"type"`
	Metadata   map[string]string `json:"metadata"`
}

// NewDHTManager creates a new DHT manager for KNIRVGRAPH
func NewDHTManager(serviceID, chainID string, bootstrapPeers []string) (*DHTManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate a new key pair for this node
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9001",
			"/ip6/::/tcp/9001",
		),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Log the host's addresses
	log.Println("KNIRVGRAPH DHT node started with addresses:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().String())
	}

	// Create a DHT instance in server mode
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Create a PubSub instance
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Parse bootstrap peers
	var bootstrapPeerInfos []peer.AddrInfo
	for _, peerAddr := range bootstrapPeers {
		if peerAddr == "" {
			continue
		}
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Printf("Invalid bootstrap peer address %s: %v", peerAddr, err)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("Failed to parse bootstrap peer %s: %v", peerAddr, err)
			continue
		}
		bootstrapPeerInfos = append(bootstrapPeerInfos, *peerInfo)
	}

	return &DHTManager{
		host:           h,
		kadDHT:         kadDHT,
		pubsub:         ps,
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: bootstrapPeerInfos,
		serviceID:      serviceID,
		chainID:        chainID,
		networkPaused:  false,
	}, nil
}

// Start initializes the DHT and starts all services
func (dm *DHTManager) Start() error {
	// Bootstrap the DHT
	if err := dm.kadDHT.Bootstrap(dm.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	for _, peerInfo := range dm.bootstrapPeers {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, peerInfo); err != nil {
				log.Printf("Failed to connect to bootstrap peer %s: %v", peerInfo.ID, err)
			} else {
				log.Printf("Connected to bootstrap peer: %s", peerInfo.ID)
			}
		}(peerInfo)
	}
	wg.Wait()

	// Join network control topic
	networkTopic, err := dm.pubsub.Join(NetworkControlTopic)
	if err != nil {
		return fmt.Errorf("failed to join network control topic: %w", err)
	}
	dm.networkControlTopic = networkTopic

	// Subscribe to network control messages
	networkSub, err := networkTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to network control topic: %w", err)
	}
	dm.networkControlSub = networkSub

	// Join graph announcement topic
	graphTopic, err := dm.pubsub.Join(GraphAnnouncementTopic)
	if err != nil {
		return fmt.Errorf("failed to join graph announcement topic: %w", err)
	}
	dm.graphTopic = graphTopic

	// Subscribe to graph announcements
	graphSub, err := graphTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to graph announcement topic: %w", err)
	}
	dm.graphSub = graphSub

	// Start message handlers
	go dm.handleNetworkControl()
	go dm.handleGraphAnnouncements()

	// Announce this service to the DHT
	go dm.announceService()

	log.Printf("KNIRVGRAPH DHT manager started successfully for service %s, chain %s", dm.serviceID, dm.chainID)
	return nil
}

// Stop shuts down the DHT manager
func (dm *DHTManager) Stop() {
	log.Println("Stopping KNIRVGRAPH DHT manager...")
	dm.cancel()
	if err := dm.host.Close(); err != nil {
		log.Printf("Error closing libp2p host: %v", err)
	}
	log.Println("KNIRVGRAPH DHT manager stopped")
}

// IsNetworkPaused returns whether the network is currently paused
func (dm *DHTManager) IsNetworkPaused() bool {
	dm.pauseMutex.RLock()
	defer dm.pauseMutex.RUnlock()

	if dm.networkPaused && time.Now().After(dm.pausedUntil) {
		// Pause has expired
		dm.pauseMutex.RUnlock()
		dm.pauseMutex.Lock()
		dm.networkPaused = false
		dm.pauseMutex.Unlock()
		dm.pauseMutex.RLock()
		log.Printf("Network pause expired for KNIRVGRAPH service %s", dm.serviceID)
	}

	return dm.networkPaused
}

// handleNetworkControl processes network control messages (pause/resume)
func (dm *DHTManager) handleNetworkControl() {
	for {
		select {
		case <-dm.ctx.Done():
			return
		default:
			msg, err := dm.networkControlSub.Next(dm.ctx)
			if err != nil {
				if dm.ctx.Err() == nil {
					log.Printf("Error receiving network control message: %v", err)
				}
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dm.host.ID() {
				continue
			}

			// Decode the network control message
			var networkMsg NetworkControlMessage
			if err := json.Unmarshal(msg.Data, &networkMsg); err != nil {
				log.Printf("Error decoding network control message: %v", err)
				continue
			}

			log.Printf("KNIRVGRAPH received network control message: %s", networkMsg.Type)

			// Handle different network control message types
			switch networkMsg.Type {
			case "NetworkPause":
				dm.handleNetworkPause(networkMsg.Payload, msg.ReceivedFrom.String())
			case "NetworkResume":
				dm.handleNetworkResume(networkMsg.Payload, msg.ReceivedFrom.String())
			default:
				log.Printf("Unknown network control message type: %s", networkMsg.Type)
			}
		}
	}
}

// handleNetworkPause processes network pause messages
func (dm *DHTManager) handleNetworkPause(payload interface{}, senderPeerID string) {
	// Parse the pause payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling pause payload: %v", err)
		return
	}

	var pausePayload NetworkPausePayload
	if err := json.Unmarshal(payloadBytes, &pausePayload); err != nil {
		log.Printf("Error unmarshaling pause payload: %v", err)
		return
	}

	initiatorPeerID := pausePayload.InitiatorPeerID
	reason := pausePayload.Reason

	// Set network paused state
	dm.pauseMutex.Lock()
	dm.networkPaused = true
	dm.pausedUntil = time.Now().Add(NetworkPauseTimeout)
	dm.pauseMutex.Unlock()

	log.Printf("KNIRVGRAPH Network PAUSED by %s until %s - Reason: %s",
		initiatorPeerID,
		dm.pausedUntil.Format("2006-01-02 15:04:05 UTC"),
		reason)

	log.Printf("KNIRVGRAPH operations paused - rejecting new announcements")
}

// handleNetworkResume processes network resume messages
func (dm *DHTManager) handleNetworkResume(payload interface{}, senderPeerID string) {
	dm.pauseMutex.Lock()
	dm.networkPaused = false
	dm.pauseMutex.Unlock()

	log.Printf("KNIRVGRAPH Network RESUMED by %s", senderPeerID)
}

// handleGraphAnnouncements processes graph-related announcements from other services
func (dm *DHTManager) handleGraphAnnouncements() {
	for {
		select {
		case <-dm.ctx.Done():
			return
		default:
			msg, err := dm.graphSub.Next(dm.ctx)
			if err != nil {
				if dm.ctx.Err() == nil {
					log.Printf("Error receiving graph announcement: %v", err)
				}
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == dm.host.ID() {
				continue
			}

			// Decode the graph announcement message
			var graphMsg GraphAnnouncementMessage
			if err := json.Unmarshal(msg.Data, &graphMsg); err != nil {
				log.Printf("Error decoding graph announcement: %v", err)
				continue
			}

			log.Printf("KNIRVGRAPH received %s %s announcement from %s",
				graphMsg.Action, graphMsg.Type, graphMsg.ServiceID)

			// Process the announcement based on type
			switch graphMsg.Type {
			case "skill":
				dm.processSkillAnnouncement(graphMsg)
			case "capability":
				dm.processCapabilityAnnouncement(graphMsg)
			case "property":
				dm.processPropertyAnnouncement(graphMsg)
			default:
				log.Printf("Unknown graph announcement type: %s", graphMsg.Type)
			}
		}
	}
}

// processSkillAnnouncement processes skill announcements from other services
func (dm *DHTManager) processSkillAnnouncement(msg GraphAnnouncementMessage) {
	log.Printf("Processing skill %s from %s", msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVGRAPH skill processing logic
}

// processCapabilityAnnouncement processes capability announcements from other services
func (dm *DHTManager) processCapabilityAnnouncement(msg GraphAnnouncementMessage) {
	log.Printf("Processing capability %s from %s", msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVGRAPH capability processing logic
}

// processPropertyAnnouncement processes property announcements from other services
func (dm *DHTManager) processPropertyAnnouncement(msg GraphAnnouncementMessage) {
	log.Printf("Processing property %s from %s", msg.Action, msg.ServiceID)
	// TODO: Integrate with KNIRVGRAPH property processing logic
}

// announceService announces this KNIRVGRAPH service to the DHT
func (dm *DHTManager) announceService() {
	// Create a CID from the service ID
	cid, err := dm.createCIDFromServiceID(dm.serviceID)
	if err != nil {
		log.Printf("Failed to create CID for service announcement: %v", err)
		return
	}

	// Announce that this node provides the KNIRVGRAPH service
	if err := dm.kadDHT.Provide(dm.ctx, cid, true); err != nil {
		log.Printf("Failed to announce KNIRVGRAPH service: %v", err)
		return
	}

	log.Printf("Announced KNIRVGRAPH service %s to DHT", dm.serviceID)
}

// AnnounceSkill announces a new skill minted on the Graph
func (dm *DHTManager) AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error {
	if dm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce skill")
	}

	skillData := SkillAnnouncementData{
		SkillID:     skillID,
		Name:        name,
		Description: description,
		Category:    category,
		Metadata:    metadata,
	}

	announcement := GraphAnnouncementMessage{
		Type:      "skill",
		Action:    "minted",
		ServiceID: dm.serviceID,
		ChainID:   dm.chainID,
		Data:      skillData,
		Timestamp: time.Now().Unix(),
	}

	return dm.publishGraphAnnouncement(announcement)
}

// AnnounceCapability announces a new capability minted on the Graph
func (dm *DHTManager) AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
	if dm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce capability")
	}

	capabilityData := CapabilityAnnouncementData{
		CapabilityID: capabilityID,
		Name:         name,
		Description:  description,
		Schema:       schema,
		Metadata:     metadata,
	}

	announcement := GraphAnnouncementMessage{
		Type:      "capability",
		Action:    "minted",
		ServiceID: dm.serviceID,
		ChainID:   dm.chainID,
		Data:      capabilityData,
		Timestamp: time.Now().Unix(),
	}

	return dm.publishGraphAnnouncement(announcement)
}

// AnnounceProperty announces a new property minted on the Graph
func (dm *DHTManager) AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
	if dm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot announce property")
	}

	propertyData := PropertyAnnouncementData{
		PropertyID: propertyID,
		Name:       name,
		Value:      value,
		Type:       propertyType,
		Metadata:   metadata,
	}

	announcement := GraphAnnouncementMessage{
		Type:      "property",
		Action:    "minted",
		ServiceID: dm.serviceID,
		ChainID:   dm.chainID,
		Data:      propertyData,
		Timestamp: time.Now().Unix(),
	}

	return dm.publishGraphAnnouncement(announcement)
}

// publishGraphAnnouncement publishes a graph announcement to the DHT
func (dm *DHTManager) publishGraphAnnouncement(announcement GraphAnnouncementMessage) error {
	msgBytes, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal graph announcement: %w", err)
	}

	if err := dm.graphTopic.Publish(dm.ctx, msgBytes); err != nil {
		return fmt.Errorf("failed to publish graph announcement: %w", err)
	}

	log.Printf("Published %s %s announcement for %s",
		announcement.Action, announcement.Type, announcement.ServiceID)
	return nil
}

// createCIDFromServiceID creates a CID from a service ID
func (dm *DHTManager) createCIDFromServiceID(serviceID string) (cid.Cid, error) {
	// Create a multihash from the service ID
	hash, err := multihash.Sum([]byte(serviceID), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}

	// Create a CID from the multihash
	return cid.NewCidV1(cid.Raw, hash), nil
}

// FindGraphServices finds other KNIRVGRAPH services on the network
func (dm *DHTManager) FindGraphServices() ([]peer.AddrInfo, error) {
	// Create a CID for KNIRVGRAPH services
	cid, err := dm.createCIDFromServiceID("knirvgraph")
	if err != nil {
		return nil, fmt.Errorf("failed to create CID for service discovery: %w", err)
	}

	// Find providers for KNIRVGRAPH services
	ctx, cancel := context.WithTimeout(dm.ctx, 30*time.Second)
	defer cancel()

	providerChan := dm.kadDHT.FindProvidersAsync(ctx, cid, 20)

	var peerInfos []peer.AddrInfo
	for provider := range providerChan {
		peerInfos = append(peerInfos, provider)
	}

	return peerInfos, nil
}
