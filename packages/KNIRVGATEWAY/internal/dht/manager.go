package dht

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
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
	DiscoveryNamespace    = "knirvgateway"
	NetworkControlTopic   = "network-control"
	GraphAnnouncementTopic = "graph-announcements"
	NetworkPauseTimeout   = 30 * time.Minute
	DefaultAnnounceInterval = 5 * time.Minute
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

	// Service identification
	serviceID string
	chainID   string

	// Resource cache for broadcast system
	resourceCache ResourceCacheInterface

	// Announcement worker
	announceInterval time.Duration
	workerCtx        context.Context
	workerCancel     context.CancelFunc
}

// Config holds DHT manager configuration.
type Config struct {
	ServiceID        string
	ChainID          string
	BootstrapPeers   []string
	EnableAutoRelay   bool
	Port             int
	AnnounceInterval time.Duration
}

// NewDHTManager creates a new DHT manager.
func NewDHTManager(cfg *Config) (*DHTManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	p2pPort := findOpenPort(9001, 100)
	if cfg.Port > 0 {
		p2pPort = findOpenPort(cfg.Port, 100)
	}

	log.Printf("DHT P2P port: %d", p2pPort)

	opts := getLibp2pOptions(priv, cfg.EnableAutoRelay, p2pPort)

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	log.Println("KNIRVGATEWAY DHT node started with addresses:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().String())
	}

	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

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

	resourceCache := NewResourceCache()

	return &DHTManager{
		host:           h,
		kadDHT:         kadDHT,
		pubsub:         ps,
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: bootstrapPeerInfos,
		serviceID:      cfg.ServiceID,
		chainID:        cfg.ChainID,
		networkPaused:  false,
		resourceCache:  resourceCache,
		announceInterval: announceInterval,
	}, nil
}

// Start initializes the DHT and starts all services.
func (dm *DHTManager) Start() error {
	if err := dm.kadDHT.Bootstrap(dm.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

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

	go dm.handleNetworkControl()
	go dm.handleGraphAnnouncements()

	go dm.announceService()

	// Start resource announcement worker
	dm.workerCtx, dm.workerCancel = context.WithCancel(dm.ctx)
	go dm.startAnnouncementWorker(dm.workerCtx, dm.announceInterval)

	log.Printf("KNIRVGATEWAY DHT manager started successfully for service %s, chain %s", dm.serviceID, dm.chainID)
	return nil
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
				// Use fresh context with timeout for each announcement
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
