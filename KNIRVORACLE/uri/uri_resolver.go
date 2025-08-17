package uri

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// URIResolver handles resolution of agent:// URIs
type URIResolver struct {
	// Cache of resolved URIs
	cache map[string]*ResolvedURI
}

// ResolvedURI contains the connection details for a agent:// URI
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
		cache: make(map[string]*ResolvedURI),
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

// ParseURI parses a agent:// URI into its components
func (r *URIResolver) ParseURI(uri string) (authority, identifier, resourceType, subPath string, err error) {
	if !strings.HasPrefix(uri, "agent://") {
		return "", "", "", "", fmt.Errorf("invalid URI scheme, must start with agent://")
	}

	// Remove the scheme
	uriWithoutScheme := strings.TrimPrefix(uri, "agent://")

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

// ResolveURI resolves a agent:// URI to connection details
func (r *URIResolver) ResolveURI(uri string) (*ResolvedURI, error) {
	// Check cache first
	if resolved, ok := r.cache[uri]; ok {
		return resolved, nil
	}

	// Parse the URI to get the authority (hostname)
	authority, _, _, _, err := r.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	// Construct the URL to the resolver API
	resolverURL := fmt.Sprintf("http://%s:3003/api/uri/resolve?uri=%s", authority, url.QueryEscape(uri))

	// Make the HTTP request
	resp, err := http.Get(resolverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to URI resolver: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read resolver response: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resolver returned error: %s", string(body))
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
	r.cache[uri] = &resolved

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
	// In a real implementation, this would use libp2p to establish a direct connection
	// using the multiaddress in resolved.Multiaddress
	// For now, we'll just return a placeholder
	return fmt.Sprintf("Direct P2P connection to %s via %s",
		resolved.TargetPeerID, resolved.Multiaddress), nil
}

// connectRelayedP2PCircuit establishes a connection via a libp2p circuit relay
func (r *URIResolver) connectRelayedP2PCircuit(resolved *ResolvedURI) (interface{}, error) {
	// In a real implementation, this would use libp2p's circuit relay functionality
	// using the multiaddress in resolved.Multiaddress
	// For now, we'll just return a placeholder
	return fmt.Sprintf("Relayed P2P circuit connection to %s via %s",
		resolved.TargetPeerID, resolved.Multiaddress), nil
}

// connectTunneled establishes a connection via the custom tunnel server
func (r *URIResolver) connectTunneled(resolved *ResolvedURI) (interface{}, error) {
	// In a real implementation, this would establish a TCP connection to the tunnel server
	// and send the target dev ID as specified in the relay protocol info
	// For now, we'll just return a placeholder
	return fmt.Sprintf("Tunneled connection to %s via %s:%d",
		resolved.TargetPeerID, resolved.TunnelServerHost, resolved.TunnelServerPort), nil
}
