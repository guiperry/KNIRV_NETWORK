package p2pconsensus

// ConsensusConfig consolidates P2P consensus settings for KNIRVBASE.
type ConsensusConfig struct {
	Enabled   bool   `json:"enabled"`
	NetworkID string `json:"network_id"`
	// SocketPath is the Unix socket used for gateway callbacks (gateway mode only).
	SocketPath string `json:"socket_path"`
	// NetworkSecret is a pre-shared key (PSK) required to join a network. It is
	// derived (salted, SHA-256) into the DHT service ID, the pubsub topic, and the
	// HMAC used to authenticate gateway-sourced messages. An empty secret means the
	// network is open (legacy behaviour) but a secret SHOULD always be configured
	// for any non-local deployment.
	NetworkSecret  string   `json:"network_secret"`
	Mode           string   `json:"mode"` // "auto" | "gateway" | "standalone" | "disabled"
	GatewayURL     string   `json:"gateway_url"`
	GatewayTimeout string   `json:"gateway_timeout"`
	ListenAddr     string   `json:"listen_addr"`
	Port           int      `json:"port"`
	BootstrapPeers []string `json:"bootstrap_peers"`
}

// DefaultConsensusConfig returns sensible defaults (P2P ON by default).
func DefaultConsensusConfig() ConsensusConfig {
	return ConsensusConfig{
		Enabled:        true,
		NetworkID:      "knirvbase-default",
		Mode:           "auto",
		GatewayURL:     "http://localhost:8080",
		GatewayTimeout: "2s",
		ListenAddr:     "0.0.0.0",
		Port:           4002,
		BootstrapPeers: []string{},
	}
}
