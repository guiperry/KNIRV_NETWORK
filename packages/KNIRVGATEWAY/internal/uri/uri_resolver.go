package uri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// URIResolver handles resolution of knirv:// URIs
type URIResolver struct {
	// Cache of resolved URIs
	cache      map[string]*ResolvedURI
	mu         sync.RWMutex
	httpClient *http.Client
}

// ResolvedURI contains the connection details for a knirv:// URI
type ResolvedURI struct {
	OriginalURI        string `json:"originalUri"`
	ResolvedIdentifier string `json:"resolvedIdentifier"`
	ResourceType       string `json:"resourceType"`
	SubPathWithQuery   string `json:"subPathWithQuery"`
	Authority          string `json:"authority"` // Add Authority field
	ConnectionType     string `json:"connectionType"`
	TargetPeerID       string `json:"targetPeerId"`
	Multiaddress       string `json:"multiaddress,omitempty"`
	TunnelServerHost   string `json:"tunnelServerHost,omitempty"`
	TunnelServerPort   int    `json:"tunnelServerPort,omitempty"`
	RelayProtocolInfo  string `json:"relayProtocolInfo,omitempty"`
}

// NewURIResolver creates a new URI resolver
func NewURIResolver() *URIResolver {
	return &URIResolver{
		cache:      make(map[string]*ResolvedURI),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// validateResourceType checks if a resource type is valid
func (r *URIResolver) validateResourceType(resourceType string) error {
	validTypes := map[string]bool{
		"chain":      true,
		"nrn":        true,
		"capability": true,
		"dev":        true,
		"resource":   true,
		"tool":       true,
		"prompt":     true,
		"memory":     true,
	}

	if !validTypes[resourceType] {
		return fmt.Errorf("invalid resource type: %s", resourceType)
	}

	return nil
}

// ParseURI parses a knirv:// URI into its components
func (r *URIResolver) ParseURI(uri string) (authority, identifier, resourceType, subPath string, err error) {
	if !strings.HasPrefix(uri, "knirv://") {
		return "", "", "", "", fmt.Errorf("invalid URI scheme, must start with knirv://")
	}

	// Remove the scheme
	uriWithoutScheme := strings.TrimPrefix(uri, "knirv://")

	// Split by the first slash to get authority and path
	parts := strings.SplitN(uriWithoutScheme, "/", 2)
	if len(parts) < 2 {
		return "", "", "", "", fmt.Errorf("invalid URI format, missing path component")
	}

	authority = parts[0]
	path := parts[1]

	// Split the path by the first slash to get identifier.resourceType and subPath
	pathParts := strings.SplitN(path, "/", 2)
	identifierAndType := pathParts[0]

	// Split identifier.resourceType by the last dot
	lastDotIndex := strings.LastIndex(identifierAndType, ".")
	if lastDotIndex == -1 {
		return "", "", "", "", fmt.Errorf("invalid URI format, missing resource type")
	}

	identifier = identifierAndType[:lastDotIndex]
	resourceType = identifierAndType[lastDotIndex+1:]

	// Validate the resource type
	if err := r.validateResourceType(resourceType); err != nil {
		return authority, identifier, resourceType, "", err
	}

	// Get the subPath if it exists
	subPath = ""
	if len(pathParts) > 1 {
		subPath = "/" + pathParts[1]
	}

	return authority, identifier, resourceType, subPath, nil
}

// ResolveURI resolves a knirv:// URI to connection details
func (r *URIResolver) ResolveURI(uri string) (*ResolvedURI, error) {
	// Check cache first
	r.mu.RLock()
	cached, ok := r.cache[uri]
	r.mu.RUnlock()
	if ok {
		return cached, nil
	}

	// Parse the URI to get the authority (hostname)
	authority, _, _, _, err := r.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	// URI resolution is a gateway service. Do not contact an untrusted URI
	// authority over cleartext or assume an obsolete port; use the canonical
	// mainnet/testnet/local discovery order.
	resolverBases := []string{
		"https://gateway.knirv.network",
		"https://testnet-gateway.knirv.network",
		"http://localhost:8080",
	}
	var body []byte
	var failures []string
	for _, base := range resolverBases {
		resolverURL := base + "/api/uri/resolve?uri=" + url.QueryEscape(uri)
		resp, requestErr := r.httpClient.Get(resolverURL)
		if requestErr != nil {
			failures = append(failures, base+": "+requestErr.Error())
			continue
		}
		candidate, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			failures = append(failures, base+": "+readErr.Error())
			continue
		}
		if resp.StatusCode != http.StatusOK {
			failures = append(failures, fmt.Sprintf("%s: HTTP %d", base, resp.StatusCode))
			continue
		}
		body = candidate
		break
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("no KNIRV URI resolver is available: %s", strings.Join(failures, "; "))
	}

	// Parse the JSON response
	var resolved ResolvedURI
	if err := json.Unmarshal(body, &resolved); err != nil {
		return nil, fmt.Errorf("failed to parse resolver response: %w", err)
	}

	// Set the Authority field if not already set
	if resolved.Authority == "" {
		resolved.Authority = authority
	}

	// Cache the result
	r.mu.Lock()
	r.cache[uri] = &resolved
	r.mu.Unlock()

	return &resolved, nil
}

// ConnectToURI establishes a connection based on the resolved URI
// Returns a connection object that can be used to communicate with the target
func (r *URIResolver) ConnectToURI(uri string) (interface{}, error) {
	resolved, err := r.ResolveURI(uri)
	if err != nil {
		return nil, err
	}

	switch resolved.ConnectionType {
	case "DIRECT_P2P":
		// Connect directly using the multiaddress
		// This would typically use libp2p to establish a direct connection
		return r.connectDirectP2P(resolved)

	case "RELAYED_P2P_CIRCUIT":
		// Connect via a libp2p circuit relay
		// This would use libp2p's circuit relay functionality
		return r.connectRelayedP2PCircuit(resolved)

	case "TUNNELED":
		// Connect via the custom tunnel server
		return r.connectTunneled(resolved)

	default:
		return nil, fmt.Errorf("unsupported connection type: %s", resolved.ConnectionType)
	}
}

// connectDirectP2P establishes a direct P2P connection
func (r *URIResolver) connectDirectP2P(resolved *ResolvedURI) (interface{}, error) {
	return connectLibp2p(resolved, false)
}

// connectRelayedP2PCircuit establishes a connection via a libp2p circuit relay
func (r *URIResolver) connectRelayedP2PCircuit(resolved *ResolvedURI) (interface{}, error) {
	return connectLibp2p(resolved, true)
}

// connectTunneled establishes a connection via the custom tunnel server
func (r *URIResolver) connectTunneled(resolved *ResolvedURI) (interface{}, error) {
	if resolved.TunnelServerHost == "" || resolved.TunnelServerPort <= 0 || resolved.TargetPeerID == "" {
		return nil, fmt.Errorf("tunnel host, port, and target peer are required")
	}
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(resolved.TunnelServerHost, fmt.Sprint(resolved.TunnelServerPort)), 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect tunnel: %w", err)
	}
	handshake, _ := json.Marshal(map[string]string{"target_peer_id": resolved.TargetPeerID, "protocol": resolved.RelayProtocolInfo})
	_ = connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := connection.Write(append(handshake, '\n')); err != nil {
		connection.Close()
		return nil, fmt.Errorf("send tunnel handshake: %w", err)
	}
	_ = connection.SetWriteDeadline(time.Time{})
	return connection, nil
}

type ownedP2PStream struct {
	network.Stream
	host host.Host
}

func (stream *ownedP2PStream) Close() error {
	streamErr := stream.Stream.Close()
	hostErr := stream.host.Close()
	if streamErr != nil {
		return streamErr
	}
	return hostErr
}

func connectLibp2p(resolved *ResolvedURI, requireRelay bool) (io.ReadWriteCloser, error) {
	if resolved.Multiaddress == "" || resolved.TargetPeerID == "" {
		return nil, fmt.Errorf("peer ID and multiaddress are required")
	}
	address, err := multiaddr.NewMultiaddr(resolved.Multiaddress)
	if err != nil {
		return nil, fmt.Errorf("parse multiaddress: %w", err)
	}
	if requireRelay && !strings.Contains(address.String(), "/p2p-circuit") {
		return nil, fmt.Errorf("relayed connection requires a /p2p-circuit multiaddress")
	}
	peerID, err := peer.Decode(resolved.TargetPeerID)
	if err != nil {
		return nil, fmt.Errorf("parse target peer ID: %w", err)
	}
	h, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return nil, fmt.Errorf("create libp2p client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := h.Connect(ctx, peer.AddrInfo{ID: peerID, Addrs: []multiaddr.Multiaddr{address}}); err != nil {
		h.Close()
		return nil, fmt.Errorf("connect peer: %w", err)
	}
	protocolID := strings.TrimSpace(resolved.RelayProtocolInfo)
	if protocolID == "" {
		protocolID = "/knirv/uri/1.0.0"
	}
	stream, err := h.NewStream(ctx, peerID, protocol.ID(protocolID))
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("open peer stream: %w", err)
	}
	return &ownedP2PStream{Stream: stream, host: h}, nil
}
