package dht

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/installer"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"go.uber.org/zap"
)

const (
	DiscoveryNamespace      = "knirvgateway"
	NetworkControlTopic     = "network_control"
	GraphAnnouncementTopic  = "graph-announcements"
	NetworkPauseTimeout     = 30 * time.Minute
	DefaultAnnounceInterval = 5 * time.Minute

	ChainSyncProtocolID = "/knirv/chain-sync/1.0.0"
	BootnodeRegistryURL = "https://registry.knirv.com"
)

// GraphAnnouncementMessage represents graph-related announcements.
type GraphAnnouncementMessage struct {
	Type      string      `json:"type"`
	Action    string      `json:"action"`
	ServiceID string      `json:"service_id"`
	ChainID   string      `json:"chain_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// SkillAnnouncementData represents skill announcement data.
type SkillAnnouncementData struct {
	SkillID     string            `json:"skill_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Metadata    map[string]string `json:"metadata"`
}

// BootnodeInfo from registry.
type BootnodeInfo struct {
	DevID    string `json:"devId,omitempty"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	LastSeen int64  `json:"lastSeen"`
	PeerID   string `json:"peerID,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Config holds DHT manager configuration.
type Config struct {
	ServiceID        string
	ChainID          string
	BootstrapPeers   []string
	EnableAutoRelay  bool
	Port             int
	AnnounceInterval time.Duration
	PublicHost       string

	// Chain P2P proxy config
	NodeRole              string // "Client", "Root", "Bootnode"
	ChainP2PPort          int
	ChainClientOnly       bool
	ChainIsBootnode       bool
	ChainBootnodeRegistry string // defaults to BootnodeRegistryURL
	ChainCallbackSocket   string // unix socket for KNIRVCHAIN callbacks
	BaseNetworkSecret     string // optional PSK for KNIRVBASE callbacks
}

// DHTManager manages DHT operations for KNIRVGATEWAY.
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

	// Chain pubsub topics (KNIRVCHAIN proxy)
	chainBlockTopic *pubsub.Topic
	chainBlockSub   *pubsub.Subscription
	chainTxTopic    *pubsub.Topic
	chainTxSub      *pubsub.Subscription

	// Service identification
	serviceID  string
	chainID    string
	nodeRole   string
	publicHost string

	// Chain P2P proxy state
	chainClientOnly     bool
	chainIsBootnode     bool
	chainCallbackSocket string
	chainCallbackClient *http.Client
	chainP2PPort        int
	bootnodeRegistry    string

	// KNIRVBASE proxy state
	baseCallbackSocket string
	baseCallbackClient *http.Client
	baseOpsTopic       *pubsub.Topic
	baseOpsSub         *pubsub.Subscription
	baseNetworkID      string
	baseNetworkSecret  string

	// Resource cache for broadcast system
	resourceCache ResourceCacheInterface

	// Announcement worker
	announceInterval time.Duration
	workerCtx        context.Context
	workerCancel     context.CancelFunc
}

// NewDHTManager creates a new DHT manager.
func NewDHTManager(cfg *Config) (*DHTManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	p2pPort := cfg.ChainP2PPort
	if p2pPort <= 0 {
		p2pPort = findOpenPort(cfg.Port, 100)
		if p2pPort == 0 {
			p2pPort = findOpenPort(9001, 100)
		}
	}

	log.Printf("DHT P2P port: %d", p2pPort)

	opts := getLibp2pOptions(priv, cfg.EnableAutoRelay, p2pPort)

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	nodeRole := cfg.NodeRole
	if nodeRole == "" {
		nodeRole = "Client"
	}
	chainID := cfg.ChainID

	// Use KNIRVCHAIN-compatible protocol prefix
	dhtProtocolPrefix := protocol.ID(fmt.Sprintf("/KNIRVCHAIN-dht/%s/%s", chainID, h.ID().String()[:8]))

	kadDHT, err := dht.New(ctx, h,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(dhtProtocolPrefix),
	)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	log.Printf("[%s][%s] DHT initialized with protocol prefix: %s", nodeRole, chainID, dhtProtocolPrefix)

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	var bootstrapPeerInfos []peer.AddrInfo
	for _, peerAddr := range cfg.BootstrapPeers {
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

	announceInterval := cfg.AnnounceInterval
	if announceInterval == 0 {
		announceInterval = DefaultAnnounceInterval
	}

	bootnodeRegistry := cfg.ChainBootnodeRegistry
	if bootnodeRegistry == "" {
		bootnodeRegistry = BootnodeRegistryURL
	}

	log.Printf("[%s][%s] Discovery Manager initialized. PeerID: %s (ClientOnly: %t)",
		nodeRole, chainID, h.ID().String(), cfg.ChainClientOnly)

	for _, addr := range h.Addrs() {
		log.Printf("[%s][%s] Listening on: %s/p2p/%s", nodeRole, chainID, addr.String(), h.ID().String())
	}

	dm := &DHTManager{
		host:                h,
		kadDHT:              kadDHT,
		pubsub:              ps,
		ctx:                 ctx,
		cancel:              cancel,
		bootstrapPeers:      bootstrapPeerInfos,
		serviceID:           cfg.ServiceID,
		chainID:             chainID,
		nodeRole:            nodeRole,
		publicHost:          cfg.PublicHost,
		chainClientOnly:     cfg.ChainClientOnly,
		chainIsBootnode:     cfg.ChainIsBootnode,
		chainCallbackSocket: cfg.ChainCallbackSocket,
		baseNetworkSecret:   cfg.BaseNetworkSecret,
		chainP2PPort:        p2pPort,
		bootnodeRegistry:    bootnodeRegistry,
		networkPaused:       false,
		resourceCache:       NewResourceCache(),
		announceInterval:    announceInterval,
	}

	if cfg.ChainCallbackSocket != "" {
		dm.chainCallbackClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", cfg.ChainCallbackSocket)
				},
			},
		}
	}

	return dm, nil
}

// Start initializes the DHT and starts all services.
func (dm *DHTManager) Start() error {
	if err := dm.kadDHT.Bootstrap(dm.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	if dm.chainIsBootnode {
		if err := dm.RegisterBootnodeWithRegistry(); err != nil {
			log.Printf("[%s][%s] WARNING: failed to register bootnode with registry: %v", dm.nodeRole, dm.chainID, err)
		}
	}

	log.Printf("[%s][%s] Connecting to bootstrap nodes...", dm.nodeRole, dm.chainID)

	// Fetch bootnodes from registry
	registryURL := fmt.Sprintf("%s/nodes", strings.TrimRight(dm.bootnodeRegistry, "/"))
	log.Printf("[%s][%s] Fetching bootnodes from registry: %s", dm.nodeRole, dm.chainID, registryURL)

	bootnodes, err := fetchBootnodesFromRegistry(dm.nodeRole, dm.chainID, registryURL)
	if err != nil {
		log.Printf("[%s][%s] WARNING: Failed to fetch bootnodes from registry: %v", dm.nodeRole, dm.chainID, err)
		log.Printf("[%s][%s] Falling back to configured bootstrap peers", dm.nodeRole, dm.chainID)
	} else {
		for _, info := range bootnodes {
			if info.Type != "" && strings.ToLower(info.Type) != "bootnode" {
				continue
			}
			addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", info.IP, info.Port)
			peerID := info.PeerID
			if peerID == "" {
				peerID = info.DevID
			}
			if peerID != "" {
				addrStr = fmt.Sprintf("%s/p2p/%s", addrStr, peerID)
			}
			ma, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				continue
			}
			ai, err := peer.AddrInfoFromP2pAddr(ma)
			if err != nil {
				continue
			}
			dm.mutex.Lock()
			dm.bootstrapPeers = append(dm.bootstrapPeers, *ai)
			dm.mutex.Unlock()
		}
	}

	var wg sync.WaitGroup
	for _, peerInfo := range dm.bootstrapPeers {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, peerInfo); err != nil {
				log.Printf("[%s][%s] Failed to connect to bootstrap peer %s: %v",
					dm.nodeRole, dm.chainID, peerInfo.ID, err)
			} else {
				log.Printf("[%s][%s] Connected to bootstrap node: %s",
					dm.nodeRole, dm.chainID, peerInfo.ID)
			}
		}(peerInfo)
	}
	wg.Wait()

	networkTopic, err := dm.pubsub.Join(NetworkControlTopic)
	if err != nil {
		return fmt.Errorf("failed to join network control topic: %w", err)
	}
	dm.networkControlTopic = networkTopic

	networkSub, err := networkTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to network control topic: %w", err)
	}
	dm.networkControlSub = networkSub

	graphTopic, err := dm.pubsub.Join(GraphAnnouncementTopic)
	if err != nil {
		return fmt.Errorf("failed to join graph announcement topic: %w", err)
	}
	dm.graphTopic = graphTopic

	graphSub, err := graphTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to graph announcement topic: %w", err)
	}
	dm.graphSub = graphSub

	// Subscribe to KNIRVCHAIN pubsub topics
	if err := dm.setupChainPubSub(); err != nil {
		return fmt.Errorf("failed to setup chain pubsub: %w", err)
	}

	// Register chain sync stream handler
	dm.host.SetStreamHandler(protocol.ID(ChainSyncProtocolID), dm.handleChainSyncStream)
	log.Printf("[%s][%s] Registered chain sync handler for protocol %s",
		dm.nodeRole, dm.chainID, ChainSyncProtocolID)

	go dm.handleNetworkControl()
	go dm.handleGraphAnnouncements()

	go dm.announceService()

	dm.workerCtx, dm.workerCancel = context.WithCancel(dm.ctx)
	go dm.startAnnouncementWorker(dm.workerCtx, dm.announceInterval)

	log.Printf("[%s][%s] KNIRVGATEWAY P2P manager started successfully for service %s",
		dm.nodeRole, dm.chainID, dm.serviceID)
	return nil
}

// setupChainPubSub subscribes to KNIRVCHAIN block/transaction topics.
func (dm *DHTManager) setupChainPubSub() error {
	log.Printf("[%s][%s] Using default topic validation settings", dm.nodeRole, dm.chainID)

	blockTopicName := fmt.Sprintf("%s.blocks", dm.chainID)
	txTopicName := fmt.Sprintf("%s.transactions", dm.chainID)

	blockTopic, err := dm.pubsub.Join(blockTopicName)
	if err != nil {
		return fmt.Errorf("failed to join block topic: %w", err)
	}
	dm.chainBlockTopic = blockTopic

	blockSub, err := blockTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to block topic: %w", err)
	}
	dm.chainBlockSub = blockSub

	txTopic, err := dm.pubsub.Join(txTopicName)
	if err != nil {
		return fmt.Errorf("failed to join transaction topic: %w", err)
	}
	dm.chainTxTopic = txTopic

	txSub, err := txTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to transaction topic: %w", err)
	}
	dm.chainTxSub = txSub

	log.Printf("[%s][%s] P2P consensus manager subscribed to topics: %s, %s, %s",
		dm.nodeRole, dm.chainID, blockTopicName, txTopicName, NetworkControlTopic)

	go dm.handleChainBlocks()
	go dm.handleChainTransactions()

	return nil
}

// handleChainBlocks forwards received blocks to KNIRVCHAIN callback.
func (dm *DHTManager) handleChainBlocks() {
	for {
		msg, err := dm.chainBlockSub.Next(dm.ctx)
		if err != nil {
			if dm.ctx.Err() != nil {
				return
			}
			log.Printf("[%s][%s] Error receiving block from pubsub: %v", dm.nodeRole, dm.chainID, err)
			continue
		}
		if msg.ReceivedFrom == dm.host.ID() {
			continue
		}
		dm.forwardToChain("/internal/p2p/received-block", msg.Data)
	}
}

// handleChainTransactions forwards received transactions to KNIRVCHAIN callback.
func (dm *DHTManager) handleChainTransactions() {
	for {
		msg, err := dm.chainTxSub.Next(dm.ctx)
		if err != nil {
			if dm.ctx.Err() != nil {
				return
			}
			log.Printf("[%s][%s] Error receiving transaction from pubsub: %v", dm.nodeRole, dm.chainID, err)
			continue
		}
		if msg.ReceivedFrom == dm.host.ID() {
			continue
		}
		dm.forwardToChain("/internal/p2p/received-tx", msg.Data)
	}
}

// forwardToChain sends data to KNIRVCHAIN via the callback unix socket.
func (dm *DHTManager) forwardToChain(path string, data []byte) {
	if dm.chainCallbackClient == nil {
		return
	}
	resp, err := dm.chainCallbackClient.Post(
		"http://localhost"+path,
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		log.Printf("[%s][%s] Failed to forward to chain %s: %v", dm.nodeRole, dm.chainID, path, err)
		return
	}
	resp.Body.Close()
}

// handleChainSyncStream handles incoming chain sync requests from remote peers.
func (dm *DHTManager) handleChainSyncStream(s network.Stream) {
	defer s.Close()
	nodeID := s.Conn().RemotePeer()
	log.Printf("[%s][%s] Received chain sync stream from %s", dm.nodeRole, dm.chainID, nodeID)

	if dm.chainCallbackClient == nil {
		log.Printf("[%s][%s] No chain callback configured, cannot handle sync from %s", dm.nodeRole, dm.chainID, nodeID)
		return
	}

	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)

	var request map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&request); err != nil {
		log.Printf("[%s][%s] Error decoding sync request from %s: %v", dm.nodeRole, dm.chainID, nodeID, err)
		return
	}

	reqData, _ := json.Marshal(request)
	resp, err := dm.chainCallbackClient.Post(
		"http://localhost/internal/chain/sync",
		"application/json",
		bytes.NewReader(reqData),
	)
	if err != nil {
		log.Printf("[%s][%s] Failed to proxy sync request to chain: %v", dm.nodeRole, dm.chainID, err)
		return
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s][%s] Failed to read chain sync response: %v", dm.nodeRole, dm.chainID, err)
		return
	}

	if _, err := writer.Write(responseData); err != nil {
		log.Printf("[%s][%s] Failed to write sync response to stream: %v", dm.nodeRole, dm.chainID, err)
		return
	}
	writer.Flush()
}

// PublishBlock broadcasts a block to the chain pubsub topic.
func (dm *DHTManager) PublishBlock(ctx context.Context, data []byte) error {
	if dm.chainBlockTopic == nil {
		return fmt.Errorf("chain block topic not initialized")
	}
	return dm.chainBlockTopic.Publish(ctx, data)
}

// PublishTransaction broadcasts a transaction to the chain pubsub topic.
func (dm *DHTManager) PublishTransaction(ctx context.Context, data []byte) error {
	if dm.chainTxTopic == nil {
		return fmt.Errorf("chain tx topic not initialized")
	}
	return dm.chainTxTopic.Publish(ctx, data)
}

// SetChainCallbackSocket updates the unix socket used for KNIRVCHAIN callbacks.
func (dm *DHTManager) SetChainCallbackSocket(socketPath string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.chainCallbackSocket = socketPath
	dm.chainCallbackClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// SetBaseCallbackSocket updates the Unix socket path for KNIRVBASE callbacks.
func (dm *DHTManager) SetBaseCallbackSocket(socketPath string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.baseCallbackSocket = socketPath
	if socketPath != "" {
		dm.baseCallbackClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
			Timeout: 30 * time.Second,
		}
	} else {
		dm.baseCallbackClient = nil
	}
}

// SetupCRDTPubSub joins a GossipSub topic for CRDT operations scoped to a networkID.
func (dm *DHTManager) SetupCRDTPubSub(networkID string, requestedTopic ...string) error {
	topicName := networkID + ".ops"
	if len(requestedTopic) > 0 && requestedTopic[0] != "" {
		topicName = requestedTopic[0]
	}
	topic, err := dm.pubsub.Join(topicName)
	if err != nil {
		return fmt.Errorf("join topic %s: %w", topicName, err)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("subscribe topic %s: %w", topicName, err)
	}
	dm.mutex.Lock()
	dm.baseOpsTopic = topic
	dm.baseOpsSub = sub
	dm.baseNetworkID = networkID
	dm.mutex.Unlock()

	// Start handling incoming operations
	go dm.handleBaseCRDTOperations(sub, topicName)
	return nil
}

// VerifyBaseMessage authenticates a KNIRVBASE request when a gateway PSK is configured.
func (dm *DHTManager) VerifyBaseMessage(networkID, signature string, payload []byte) bool {
	dm.mutex.RLock()
	secret := dm.baseNetworkSecret
	configuredID := dm.baseNetworkID
	dm.mutex.RUnlock()
	if configuredID != "" && networkID != configuredID {
		return false
	}
	if secret == "" {
		return true
	}
	h := sha256.New()
	_, _ = h.Write([]byte("knirvbase-network-v1"))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(networkID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(secret))
	mac := hmac.New(sha256.New, h.Sum(nil))
	_, _ = mac.Write(payload)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(signature))
}

// PublishOperation publishes a CRDT operation to the GossipSub topic.
func (dm *DHTManager) PublishOperation(ctx context.Context, data []byte) error {
	dm.mutex.RLock()
	topic := dm.baseOpsTopic
	dm.mutex.RUnlock()
	if topic == nil {
		return fmt.Errorf("CRDT pubsub topic not set up")
	}
	return topic.Publish(ctx, data)
}

func (dm *DHTManager) PublishSyncRequest(ctx context.Context, networkID string, data []byte) error {
	dm.mutex.RLock()
	topic := dm.baseOpsTopic
	dm.mutex.RUnlock()
	if topic == nil {
		return fmt.Errorf("CRDT pubsub topic not set up")
	}
	wire := struct {
		Type      string          `json:"type"`
		NetworkID string          `json:"network_id"`
		Payload   json.RawMessage `json:"payload"`
	}{Type: "sync_request", NetworkID: networkID, Payload: data}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return err
	}
	return topic.Publish(ctx, encoded)
}

// handleBaseCRDTOperations receives CRDT ops from GossipSub and forwards to KNIRVBASE.
func (dm *DHTManager) handleBaseCRDTOperations(sub *pubsub.Subscription, topicName string) {
	for {
		msg, err := sub.Next(dm.ctx)
		if err != nil {
			select {
			case <-dm.ctx.Done():
				return
			default:
				continue
			}
		}
		// Don't process our own messages
		if msg.GetFrom() == dm.host.ID() {
			continue
		}
		var wire struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		path := "/internal/p2p/received-op"
		data := msg.Data
		if json.Unmarshal(msg.Data, &wire) == nil && wire.Type == "sync_request" {
			path = "/internal/p2p/received-sync-request"
			data = wire.Payload
			if response, err := dm.forwardToBaseResponse(path, data); err == nil {
				var syncResponse struct {
					Operations []json.RawMessage `json:"operations"`
				}
				if json.Unmarshal(response, &syncResponse) == nil {
					for _, operation := range syncResponse.Operations {
						_ = dm.PublishOperation(dm.ctx, operation)
					}
				}
			}
			continue
		}
		dm.forwardToBase(path, data)
	}
}

// HandleCRDTOperations forwards an incoming CRDT operation to KNIRVBASE.
// This is the exported entry point called by KNIRVCHAIN/HTTP handlers.
func (dm *DHTManager) HandleCRDTOperations(ctx context.Context, data []byte) error {
	path := "/internal/p2p/received-op"
	dm.forwardToBase(path, data)
	return nil
}

// forwardToBase forwards data to KNIRVBASE via Unix socket callback.
func (dm *DHTManager) forwardToBase(path string, data []byte) {
	_, _ = dm.forwardToBaseResponse(path, data)
}

func (dm *DHTManager) forwardToBaseResponse(path string, data []byte) ([]byte, error) {
	dm.mutex.RLock()
	client := dm.baseCallbackClient
	socket := dm.baseCallbackSocket
	dm.mutex.RUnlock()
	if client == nil || socket == "" {
		return nil, fmt.Errorf("base callback is not configured")
	}
	url := "http://knirvbase" + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var envelope struct {
		NetworkID string `json:"network_id"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.NetworkID != "" {
		if sig := dm.SignBaseMessage(envelope.NetworkID, data); sig != "" {
			req.Header.Set("X-KNIRV-Signature", sig)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("base callback returned %d", resp.StatusCode)
	}
	return body, nil
}

// SignBaseMessage creates the same signature as the KNIRVBASE proxy client.
func (dm *DHTManager) SignBaseMessage(networkID string, payload []byte) string {
	dm.mutex.RLock()
	secret := dm.baseNetworkSecret
	dm.mutex.RUnlock()
	if secret == "" {
		return ""
	}
	h := sha256.New()
	_, _ = h.Write([]byte("knirvbase-network-v1"))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(networkID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(secret))
	mac := hmac.New(sha256.New, h.Sum(nil))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// PublishToBaseDiscovery lets peers discover KNIRVBASE resources via DHT.
func (dm *DHTManager) PublishToBaseDiscovery(ctx context.Context, networkID string) error {
	serviceID := "knirvbase-" + networkID
	return dm.AnnounceService(ctx, serviceID)
}

// FindBasePeers discovers other KNIRVBASE peers via DHT.
func (dm *DHTManager) FindBasePeers(ctx context.Context, networkID string) ([]peer.AddrInfo, error) {
	serviceID := "knirvbase-" + networkID
	return dm.FindServices(ctx, serviceID)
}

// GetPeerID returns the local peer ID.
func (dm *DHTManager) GetPeerID() string {
	return dm.host.ID().String()
}

// GetSelfMultiaddrs returns this node's multiaddresses.
func (dm *DHTManager) GetSelfMultiaddrs() []string {
	addrs := make([]string, 0, len(dm.host.Addrs()))
	for _, addr := range dm.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), dm.host.ID().String()))
	}
	return addrs
}

// GetConnectedPeers returns connected peer IDs.
func (dm *DHTManager) GetConnectedPeers() []string {
	conns := dm.host.Network().Conns()
	peers := make([]string, 0, len(conns))
	for _, c := range conns {
		peers = append(peers, c.RemotePeer().String())
	}
	return peers
}

// Stop shuts down the DHT manager.
func (dm *DHTManager) Stop() {
	log.Println("Stopping KNIRVGATEWAY DHT manager...")
	if dm.workerCancel != nil {
		dm.workerCancel()
	}
	dm.cancel()
	if err := dm.host.Close(); err != nil {
		log.Printf("Error closing libp2p host: %v", err)
	}
	log.Println("KNIRVGATEWAY DHT manager stopped")
}

// IsNetworkPaused returns whether the network is currently paused.
func (dm *DHTManager) IsNetworkPaused() bool {
	dm.pauseMutex.RLock()
	defer dm.pauseMutex.RUnlock()

	if dm.networkPaused && time.Now().After(dm.pausedUntil) {
		dm.pauseMutex.RUnlock()
		dm.pauseMutex.Lock()
		dm.networkPaused = false
		dm.pauseMutex.Unlock()
		dm.pauseMutex.RLock()
		log.Println("Network pause expired for KNIRVGATEWAY")
	}
	return dm.networkPaused
}

// Provide announces to the DHT that we provide a given CID.
func (dm *DHTManager) Provide(ctx context.Context, cid cid.Cid) error {
	return dm.kadDHT.Provide(ctx, cid, true)
}

// FindProviders finds providers for a given CID.
func (dm *DHTManager) FindProviders(ctx context.Context, cid cid.Cid) ([]peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	providerChan := dm.kadDHT.FindProvidersAsync(ctx, cid, 20)

	var peerInfos []peer.AddrInfo
	for provider := range providerChan {
		peerInfos = append(peerInfos, provider)
	}

	return peerInfos, nil
}

// AnnounceService announces this service to the DHT.
func (dm *DHTManager) AnnounceService(ctx context.Context, serviceID string) error {
	cid, err := createCIDFromServiceID(serviceID)
	if err != nil {
		return fmt.Errorf("failed to create CID for service announcement: %w", err)
	}

	if err := dm.kadDHT.Provide(ctx, cid, true); err != nil {
		return fmt.Errorf("failed to announce service: %w", err)
	}

	log.Printf("Announced service %s to DHT", serviceID)
	return nil
}

// FindServices finds providers for a given service ID.
func (dm *DHTManager) FindServices(ctx context.Context, serviceID string) ([]peer.AddrInfo, error) {
	cid, err := createCIDFromServiceID(serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create CID for service discovery: %w", err)
	}

	return dm.FindProviders(ctx, cid)
}

// PublishAnnouncement publishes a graph announcement to the DHT.
func (dm *DHTManager) PublishAnnouncement(ctx context.Context, topic string, data []byte) error {
	if dm.IsNetworkPaused() {
		return fmt.Errorf("network is paused, cannot publish to topic %s", topic)
	}

	t, err := dm.pubsub.Join(topic)
	if err != nil {
		return fmt.Errorf("failed to join topic %s: %w", topic, err)
	}

	if err := t.Publish(ctx, data); err != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, err)
	}

	return nil
}

// SubscribeAnnouncements subscribes to graph announcements on a topic.
func (dm *DHTManager) SubscribeAnnouncements(ctx context.Context, topic string) (<-chan []byte, error) {
	if dm.IsNetworkPaused() {
		return nil, fmt.Errorf("network is paused, cannot subscribe to topic %s", topic)
	}

	t, err := dm.pubsub.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic %s: %w", topic, err)
	}

	sub, err := t.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	ch := make(chan []byte, 10)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := sub.Next(ctx)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf("Error receiving announcement: %v", err)
					}
					return
				}
				ch <- msg.Data
			}
		}
	}()

	return ch, nil
}

// GetResourceCache returns the resource cache.
func (dm *DHTManager) GetResourceCache() ResourceCacheInterface {
	return dm.resourceCache
}

// AnnounceAllCached announces all cached resources to the DHT.
func (dm *DHTManager) AnnounceAllCached(ctx context.Context) (int, error) {
	resources := dm.resourceCache.GetAllResources()
	announced := 0

	for _, res := range resources {
		cid, err := createCID(res.ID, res.Type)
		if err != nil {
			log.Printf("Failed to create CID for resource %s: %v", res.ID, err)
			continue
		}

		if err := dm.kadDHT.Provide(ctx, cid, true); err != nil {
			log.Printf("Failed to announce cached resource %s: %v", res.ID, err)
			continue
		}
		announced++
	}

	log.Printf("Announced %d/%d cached resources to DHT", announced, len(resources))
	return announced, nil
}

// startAnnouncementWorker periodically announces all cached resources.
func (dm *DHTManager) startAnnouncementWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resources := dm.resourceCache.GetAllResources()
			for _, res := range resources {
				cid, err := createCID(res.ID, res.Type)
				if err != nil {
					log.Printf("Failed to create CID for resource %s: %v", res.ID, err)
					continue
				}
				announceCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := dm.kadDHT.Provide(announceCtx, cid, true); err != nil {
					log.Printf("Failed to announce cached resource %s: %v", res.ID, err)
				}
				cancel()
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleNetworkControl processes network control messages.
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

			if msg.ReceivedFrom == dm.host.ID() {
				continue
			}

			log.Printf("KNIRVGATEWAY received network control message")
		}
	}
}

// handleGraphAnnouncements processes graph-related announcements.
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

			if msg.ReceivedFrom == dm.host.ID() {
				continue
			}

			log.Printf("KNIRVGATEWAY received graph announcement")
		}
	}
}

// announceService announces this KNIRVGATEWAY service to the DHT.
func (dm *DHTManager) announceService() {
	cid, err := createCIDFromServiceID(dm.serviceID)
	if err != nil {
		log.Printf("Failed to create CID for service announcement: %v", err)
		return
	}

	if err := dm.kadDHT.Provide(dm.ctx, cid, true); err != nil {
		log.Printf("Failed to announce KNIRVGATEWAY service: %v", err)
		return
	}

	log.Printf("Announced KNIRVGATEWAY service %s to DHT", dm.serviceID)
}

// createCID creates a CID from a resource ID and type.
func createCID(id, resourceType string) (cid.Cid, error) {
	hash, err := multihash.Sum([]byte(fmt.Sprintf("%s:%s", id, resourceType)), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.Raw, hash), nil
}

// createCIDFromServiceID creates a CID from a service ID.
func createCIDFromServiceID(serviceID string) (cid.Cid, error) {
	hash, err := multihash.Sum([]byte(serviceID), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.Raw, hash), nil
}

type registrationPayload struct {
	ChainID string `json:"chainID"`
	IP      string `json:"ip,omitempty"`
	Port    int    `json:"port"`
	Type    string `json:"type"`
	PeerID  string `json:"nodeID"`
}

func (dm *DHTManager) RegisterBootnodeWithRegistry() error {
	if !dm.chainIsBootnode {
		return nil
	}
	if dm.bootnodeRegistry == "" {
		return fmt.Errorf("bootnode registry is not configured")
	}
	if dm.chainP2PPort <= 0 {
		return fmt.Errorf("chain p2p port is not configured")
	}

	publicHost, err := dm.resolveRegistryPublicHost()
	if err != nil {
		return err
	}

	return RegisterWithRegistry(dm.bootnodeRegistry, dm.chainID, dm.chainP2PPort, publicHost, dm.host.ID().String())
}

func (dm *DHTManager) resolveRegistryPublicHost() (string, error) {
	candidate := strings.TrimSpace(dm.publicHost)
	if candidate != "" && net.ParseIP(candidate) != nil {
		return candidate, nil
	}

	stunClient := installer.NewSTUNClient(nil, zap.NewNop())
	result, err := stunClient.DiscoverExternalAddress()
	if err == nil && result != nil {
		if ip := strings.TrimSpace(result.ExternalIP); ip != "" {
			dm.publicHost = ip
			log.Printf("[%s][%s] Resolved public IP for bootnode registry via STUN: %s", dm.nodeRole, dm.chainID, ip)
			return ip, nil
		}
	}

	if candidate != "" && candidate != "localhost" && candidate != "127.0.0.1" && candidate != "::1" {
		log.Printf("[%s][%s] Falling back to configured public host for bootnode registry: %s", dm.nodeRole, dm.chainID, candidate)
		return candidate, nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to resolve public host for bootnode registry: %w", err)
	}
	return "", fmt.Errorf("failed to resolve public host for bootnode registry")
}

// RegisterWithRegistry publishes this node to the external bootnode registry.
func RegisterWithRegistry(registryURL string, nodeChainID string, nodeP2PPort int, nodeIP string, nodePeerID string) error {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		return fmt.Errorf("registry URL is required")
	}
	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		registryURL = "http://" + registryURL
	}

	payload := registrationPayload{
		ChainID: nodeChainID,
		IP:      nodeIP,
		Port:    nodeP2PPort,
		Type:    "bootnode",
		PeerID:  nodePeerID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	baseURL := strings.TrimRight(registryURL, "/")
	registerURL := baseURL + "/register"

	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVGATEWAY/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read registration response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registry returned non-success status %d: %s", resp.StatusCode, string(body))
	}

	var responseData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &responseData); err == nil {
			if message, ok := responseData["message"].(string); ok && message != "" {
				log.Printf("Bootnode registry response: %s", message)
			}
		}
	}

	log.Printf("Registered bootnode %s with registry %s", nodeChainID, baseURL)
	return nil
}

// findOpenPort searches for an open port starting from preferredPort.
func findOpenPort(preferredPort, maxAttempts int) int {
	for port := preferredPort; port < preferredPort+maxAttempts; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

// fetchBootnodesFromRegistry fetches the bootnode list from the registry.
func fetchBootnodesFromRegistry(role, chainID, registryURL string) (map[string]BootnodeInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", registryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KNIRVGATEWAY/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var bootnodes map[string]BootnodeInfo
	if err := json.Unmarshal(body, &bootnodes); err != nil {
		return nil, err
	}

	return bootnodes, nil
}
