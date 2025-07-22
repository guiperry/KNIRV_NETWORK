package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	parser "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/parser"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// ResourceData represents data fetched from a KNIRV resource
type ResourceData struct {
	data []byte
}

// Bytes returns the raw byte data
func (r *ResourceData) Bytes() []byte {
	return r.data
}

// String returns the data as a string
func (r *ResourceData) String() string {
	return string(r.data)
}

// ClientConfig holds configuration for the KNIRV client
type ClientConfig struct {
	BootstrapPeers []string
	P2PPort        int
	LogEnabled     bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() ClientConfig {
	return ClientConfig{
		BootstrapPeers: []string{
			"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
			"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
			"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		},
		P2PPort:    0, // Use random port
		LogEnabled: false,
	}
}

// ProviderFinder defines the interface for finding resource providers
type ProviderFinder interface {
	FindResourceProviders(id, resourceType string) ([]peer.AddrInfo, error)
}

// Client is the main KNIRV client
type Client struct {
	host           host.Host
	dht            *dual.DHT
	bootstrapped   bool
	ctx            context.Context
	cancel         context.CancelFunc
	config         ClientConfig
	closeOnce      sync.Once
	closed         bool
	closeMu        sync.Mutex
	providerFinder ProviderFinder
}

// ErrClientClosed is returned when operations are attempted on a closed client
var ErrClientClosed = errors.New("client is closed")

// ErrResourceNotFound is returned when a resource cannot be found
var ErrResourceNotFound = errors.New("resource not found")

// ErrConnectionFailed is returned when connection to a peer fails
var ErrConnectionFailed = errors.New("failed to connect to peer")

// ErrFetchFailed is returned when fetching a resource fails
var ErrFetchFailed = errors.New("failed to fetch resource")

// New creates a new KNIRV client
func New(config ClientConfig, pf ProviderFinder) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate a new key pair for this host
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate node key: %w", err)
	}

	// Create a connection manager
	connManager, err := connmgr.NewConnManager(
		100, // Low watermark
		400, // High watermark
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Setup listening addresses
	listenAddrs := []multiaddr.Multiaddr{}

	// Add IPv4 listening address
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.P2PPort))
	if err == nil {
		listenAddrs = append(listenAddrs, addr)
	}

	// Add IPv6 listening address
	addr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", config.P2PPort))
	if err == nil {
		listenAddrs = append(listenAddrs, addr)
	}

	// Setup libp2p options
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connManager),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
		libp2p.EnableAutoRelay(
			autorelay.WithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				peerChan := make(chan peer.AddrInfo, len(config.BootstrapPeers))
				for _, addr := range config.BootstrapPeers {
					pi, err := peer.AddrInfoFromString(addr)
					if err != nil {
						if config.LogEnabled {
							log.Printf("Warning: Failed to parse bootstrap peer for AutoRelay: %s, %v", addr, err)
						}
						continue
					}
					select {
					case peerChan <- *pi:
					case <-ctx.Done():
						close(peerChan)
						return nil
					}
				}
				close(peerChan)
				return peerChan
			}),
			autorelay.WithMinInterval(time.Minute),
		),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Setup DHT for peer discovery
	idht, err := dual.New(ctx, h,
		dual.DHTOption(dht.Mode(dht.ModeClient)),
		dual.DHTOption(dht.BootstrapPeers(convertToAddrInfo(config.BootstrapPeers)...)),
	)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	if config.LogEnabled {
		log.Printf("KNIRV Client started with PeerID: %s", h.ID().String())
		for _, addr := range h.Addrs() {
			log.Printf("Listening on: %s/p2p/%s", addr.String(), h.ID().String())
		}
	}

	client := &Client{
		host:   h,
		dht:    idht,
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
	if pf != nil {
		client.providerFinder = pf
	} else {
		client.providerFinder = client
	}

	// Register protocol handlers
	client.registerProtocolHandlers()

	return client, nil
}

// registerProtocolHandlers sets up handlers for KNIRV protocols
func (c *Client) registerProtocolHandlers() {
	// Register handler for the resource fetch protocol
	c.host.SetStreamHandler(protocol.ID("/knirv/resource-fetch/1.0.0"), func(s network.Stream) {
		// This is a client, so we don't expect incoming resource fetch requests
		// But we need to handle them gracefully
		defer s.Close()
		_, _ = s.Write([]byte("ERROR: This is a client node and does not serve resources"))
	})
}

// convertToAddrInfo converts string multiaddrs to AddrInfo
func convertToAddrInfo(addrs []string) []peer.AddrInfo {
	var pis []peer.AddrInfo
	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}

		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}

		pis = append(pis, *ai)
	}
	return pis
}

// Bootstrap connects to bootstrap nodes and joins the DHT network
func (c *Client) Bootstrap() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return ErrClientClosed
	}
	c.closeMu.Unlock()

	if c.bootstrapped {
		return nil
	}

	if c.config.LogEnabled {
		log.Println("Connecting to bootstrap nodes...")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()

	// Connect to bootstrap peers
	wg := sync.WaitGroup{}
	for _, peerInfo := range convertToAddrInfo(c.config.BootstrapPeers) {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := c.host.Connect(ctx, peerInfo); err != nil {
				if c.config.LogEnabled {
					log.Printf("Error connecting to bootstrap node %s: %s", peerInfo.ID.String(), err)
				}
			} else if c.config.LogEnabled {
				log.Printf("Connected to bootstrap node: %s", peerInfo.ID.String())
			}
		}(peerInfo)
	}

	// Wait with a timeout for connections to bootstrap nodes
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if c.config.LogEnabled {
			log.Println("Bootstrap connections completed")
		}
	case <-time.After(30 * time.Second):
		if c.config.LogEnabled {
			log.Println("Bootstrap timed out, continuing with available connections")
		}
	}

	// Bootstrap the DHT
	if err := c.dht.Bootstrap(c.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	c.bootstrapped = true
	if c.config.LogEnabled {
		log.Println("DHT bootstrapped successfully")
	}

	return nil
}

// Close shuts down the client
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closeMu.Lock()
		c.closed = true
		c.closeMu.Unlock()

		if c.cancel != nil {
			c.cancel()
		}

		if c.host != nil {
			err = c.host.Close()
		}
	})
	return err
}

// FetchResource fetches a resource from the KNIRV network
func (c *Client) FetchResource(uriString string) (*ResourceData, error) {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil, ErrClientClosed
	}
	c.closeMu.Unlock()

	// Ensure client is bootstrapped
	if !c.bootstrapped {
		if err := c.Bootstrap(); err != nil {
			return nil, fmt.Errorf("bootstrap failed: %w", err)
		}
	}

	// Parse the URI
	parsedURI, err := parser.Parse(uriString)
	if err != nil {
		return nil, err
	}

	// Find providers for the resource
	providers, err := c.providerFinder.FindResourceProviders(parsedURI.ID, parsedURI.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResourceNotFound, err)
	}

	// Try to connect to providers and fetch the resource
	for _, provider := range providers {
		data, err := c.fetchFromProvider(provider, parsedURI)
		if err != nil {
			if c.config.LogEnabled {
				log.Printf("Failed to fetch from provider %s: %v", provider.ID.String(), err)
			}
			continue
		}
		return &ResourceData{data: data}, nil
	}

	return nil, ErrResourceNotFound
}

// FindResourceProviders finds providers for a resource in the DHT
func (c *Client) FindResourceProviders(id, resourceType string) ([]peer.AddrInfo, error) {
	// Create a CID from the resource identifier
	resourceKey := fmt.Sprintf("%s.%s", id, resourceType)
	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return nil, fmt.Errorf("failed to create multihash: %w", err)
	}

	resourceCID := cid.NewCidV1(cid.Raw, mh)

	// Find providers for this resource
	if c.config.LogEnabled {
		log.Printf("Looking for providers of resource: %s (CID: %s)", resourceKey, resourceCID.String())
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	providers := c.dht.FindProvidersAsync(ctx, resourceCID, 20)

	var results []peer.AddrInfo
	for p := range providers {
		if p.ID == c.host.ID() {
			// Skip ourselves
			continue
		}
		results = append(results, p)
		if c.config.LogEnabled {
			log.Printf("Found provider for %s: %s", resourceKey, p.ID.String())
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no providers found for resource: %s", resourceKey)
	}

	return results, nil
}

// fetchFromProvider fetches a resource from a specific provider
func (c *Client) fetchFromProvider(provider peer.AddrInfo, uri *parser.KnirvURI) ([]byte, error) {
	// Connect to the provider
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if c.host.Network().Connectedness(provider.ID) != network.Connected {
		if c.config.LogEnabled {
			log.Printf("Connecting to provider: %s", provider.ID.String())
		}
		if err := c.host.Connect(ctx, provider); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
		}
	}

	// Open a stream to the provider
	protocolID := protocol.ID("/knirv/resource-fetch/1.0.0")
	stream, err := c.host.NewStream(ctx, provider.ID, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Prepare the request
	request := fmt.Sprintf("GET %s", uri.Path)
	if len(uri.Query) > 0 {
		queryParams := make([]string, 0, len(uri.Query))
		for k, values := range uri.Query {
			for _, v := range values {
				queryParams = append(queryParams, fmt.Sprintf("%s=%s", k, v))
			}
		}
		request += "?" + strings.Join(queryParams, "&")
	}
	request += "\n"

	// Send the request
	if c.config.LogEnabled {
		log.Printf("Sending request to %s: %s", provider.ID.String(), request)
	}
	if _, err := stream.Write([]byte(request)); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read the response
	var responseData []byte
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		responseData = append(responseData, buf[:n]...)
	}

	if len(responseData) == 0 {
		return nil, fmt.Errorf("empty response from provider")
	}

	// Check if the response indicates an error
	if bytes.HasPrefix(responseData, []byte("ERROR:")) {
		return nil, fmt.Errorf("%w: %s", ErrFetchFailed, string(responseData))
	}

	return responseData, nil
}

// GetPeerID returns the peer ID of this client
func (c *Client) GetPeerID() string {
	return c.host.ID().String()
}

// GetMultiaddrs returns the multiaddresses of this client
func (c *Client) GetMultiaddrs() []string {
	var addrs []string
	for _, addr := range c.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), c.host.ID().String()))
	}
	return addrs
}
