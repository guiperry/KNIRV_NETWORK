package network

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	circuit "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

// RelayConfig contains configuration for the circuit relay
type RelayConfig struct {
	Enabled            bool
	Resources          circuit.Resources
	AdvertiseInterval  time.Duration
	DiscoveryNamespace string
}

// DefaultRelayConfig returns a default relay configuration
func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		Enabled: true,
		Resources: circuit.Resources{
			MaxCircuits:            128,
			MaxReservations:        128,
			ReservationTTL:         time.Hour,
			MaxReservationsPerPeer: 4,
			MaxReservationsPerIP:   8,
			MaxReservationsPerASN:  16,
		},
		AdvertiseInterval:  time.Minute * 10,
		DiscoveryNamespace: "KNIRVCHAIN-relay",
	}
}

// EnableRelayOnHost configures the given libp2p host as a circuit relay
func EnableRelayOnHost(ctx context.Context, h host.Host, dht *dht.IpfsDHT, config RelayConfig) (*circuit.Relay, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("relay is not enabled in configuration")
	}

	// Create a relay with the given resources
	relayService, err := circuit.New(h, circuit.WithResources(config.Resources))
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit relay: %w", err)
	}

	// Create a child context that can be canceled when the host is closed
	relayCtx, relayCancel := context.WithCancel(ctx)

	// Set up a goroutine to cancel the relay context when the host is closed
	go func() {
		<-ctx.Done()
		log.Printf("Main context canceled, stopping relay advertising for host %s", h.ID())
		relayCancel()
	}()

	// Advertise this node as a relay in the DHT with the cancellable context
	go advertiseRelay(relayCtx, h, dht, config.DiscoveryNamespace, config.AdvertiseInterval)

	log.Printf("Circuit relay enabled on host %s", h.ID())
	return relayService, nil
}

// advertiseRelay periodically advertises this node as a relay in the DHT
func advertiseRelay(ctx context.Context, h host.Host, dht *dht.IpfsDHT, ns string, interval time.Duration) {
	// Check if context is already done
	select {
	case <-ctx.Done():
		log.Printf("Context already canceled, not starting relay advertising for host %s", h.ID())
		return
	default:
		// Continue with advertising
	}

	// Create a routing discovery instance
	routingDiscovery := drouting.NewRoutingDiscovery(dht)

	// Advertise this node as a relay
	// Use a derived context for the initial advertisement to allow for quick cancellation if main ctx is already done.
	advCtx, cancelAdv := context.WithTimeout(ctx, 30*time.Second)
	defer cancelAdv() // Ensure this is always called

	// Try to advertise, but don't block if it fails
	advertiseWithTimeout := func(ctx context.Context) {
		done := make(chan struct{})
		go func() {
			routingDiscovery.Advertise(ctx, ns)
			close(done)
		}()

		select {
		case <-done:
			// Advertisement completed
		case <-time.After(5 * time.Second):
			log.Printf("WARNING: Relay advertisement taking too long for host %s, continuing...", h.ID())
		}
	}

	// Initial advertisement
	advertiseWithTimeout(advCtx)
	log.Printf("Host %s started advertising as a relay in namespace %s", h.ID(), ns)

	// Periodically re-advertise
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if context is done before advertising
			select {
			case <-ctx.Done():
				log.Printf("Context canceled, stopping relay advertising for host %s", h.ID())
				return
			default:
				// Continue with advertising
			}

			advCtxPeriodic, cancelAdvPeriodic := context.WithTimeout(ctx, 30*time.Second)
			advertiseWithTimeout(advCtxPeriodic)
			cancelAdvPeriodic()
			log.Printf("Host %s re-advertised as a relay in namespace %s", h.ID(), ns)

		case <-ctx.Done():
			log.Printf("Stopping relay advertising for host %s in namespace %s due to context cancellation.", h.ID(), ns)
			return
		}
	}
}

// ConfigureHostAsRelayClient configures the given libp2p host to use circuit relays
func ConfigureHostAsRelayClient(h host.Host, bootstrapPeers []peer.AddrInfo) error {
	// Connect to bootstrap nodes
	for _, nodeInfo := range bootstrapPeers {
		if nodeInfo.ID == h.ID() {
			continue // Skip self
		}
		if err := h.Connect(context.Background(), nodeInfo); err != nil {
			log.Printf("Failed to connect to bootstrap node %s: %v", nodeInfo.ID, err)
		} else {
			log.Printf("Connected to bootstrap node %s", nodeInfo.ID)
		}
	}

	return nil
}

// FindRelays discovers relay nodes in the DHT
func FindRelays(ctx context.Context, dht *dht.IpfsDHT, ns string) ([]peer.AddrInfo, error) {
	// Create a routing discovery instance
	routingDiscovery := drouting.NewRoutingDiscovery(dht)

	// Find relay nodes
	nodeChan, err := routingDiscovery.FindPeers(ctx, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to find relay nodes: %w", err)
	}

	// Collect nodes
	var relays []peer.AddrInfo
	for p := range nodeChan {
		if p.ID == "" {
			continue // Skip invalid nodes
		}
		relays = append(relays, p)
	}

	return relays, nil
}

// NewHostWithRelayClient creates a new libp2p host configured to use circuit relays
func NewHostWithRelayClient(ctx context.Context, bootstrapPeers []peer.AddrInfo) (host.Host, routing.Routing, error) {
	// Create a new libp2p host
	h, err := libp2p.New(
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Create a DHT for node discovery
	kadDHT, err := dht.New(ctx, h)
	if err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT
	if err := kadDHT.Bootstrap(ctx); err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap nodes
	if err := ConfigureHostAsRelayClient(h, bootstrapPeers); err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to configure host as relay client: %w", err)
	}

	return h, kadDHT, nil
}
