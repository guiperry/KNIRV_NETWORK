package config

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Gateway configuration
	Port       int
	SocketPath string
	ChainID    string
	PublicHost string

	// DHT/P2P configuration
	DisableDHT       bool
	DHTPort          int
	BootstrapPeers   []string
	InternalAPIKey   string
	RenderGatewayAPI string

	// Node.js Services configuration
	NodeJSServicesEnabled   bool
	NodeJSServicesAutoStart bool

	// Payment Gateway
	PaymentGatewayEnabled bool
	PaymentGatewayPort    int

	// Tunnel Registry
	TunnelRegistryEnabled     bool
	TunnelRegistryHTTPPort    int
	TunnelRegistryControlPort int
	TunnelRegistryRelayPort   int
	TunnelRegistrySTUNPort    int

	// Operator Registry
	OperatorRegistryEnabled bool
	OperatorRegistryPort    int

	// Explorer UI
	WebGUIEnabled bool
	WebGUIPort    int

	// Session configuration
	SessionSecret string

	// Browser auto-open configuration
	AutoOpenBrowser bool

	// KNIRV-ORACLE configuration — all oracle traffic flows through the gateway
	// via a Unix socket.  The independent TCP port (historically 1317) is removed;
	// the oracle is only reachable through the gateway's reverse-proxy layer.
	OracleSocketPath string // Unix socket path for KNIRVORACLE (required)

	// TURN Server configuration
	TurnServerEnabled      bool
	TurnServerUDPPort      int
	TurnServerTCPPort      int
	TurnServerAPIPort      int
	TurnServerAuthSecret   string
	TurnServerRealm        string
	TurnServerMinerAddress string

	// Chain P2P proxy configuration (KNIRVGATEWAY manages P2P on behalf of KNIRVCHAIN)
	ChainNodeRole         string // e.g. "Client", "Root", "Bootnode"
	ChainP2PPort          int    // libp2p listening port
	ChainClientOnly       bool   // skip self-announce / mDNS
	ChainIsBootnode       bool
	ChainBootnodeRegistry string // e.g. "https://registry.knirv.com"
	ChainCallbackSocket   string // unix socket path for KNIRVCHAIN P2P callbacks

	// Socket paths for internal service reverse proxies
	BackendSocketPath string // /var/lib/knirvserver/sockets/backend.sock
	ServerBaseURL     string // wrapper-owned native APIs, normally http://127.0.0.1:8090
	ChainSocketPath   string // /var/lib/knirvserver/sockets/chain.sock
	GraphSocketPath   string // /var/lib/knirvserver/sockets/graph.sock

	// InternalAuthToken is the shared service-to-service token KNIRVCHAIN
	// expects on X-KNIRV-Internal-Token for its internal-only mint endpoints
	// (chain_refactor.md §3.2/§4 Phase 2). The gateway attaches it on the
	// caller's behalf for the event-bundle mint proxy so CLI clients never
	// need to hold this secret themselves.
	InternalAuthToken string

	// Shell daemon socket path
	ShellSocketPath string // /var/lib/knirvserver/sockets/shell.sock

	// Agent per-DVE socket directory and concurrency limit
	AgentSocketDir     string // /var/lib/knirvserver/sockets/ (agent-{dveId}.sock lives here)
	AgentMaxConcurrent int    // max concurrent agent processes (default 32)

	// KNIRVHASHER socket path (headless CLI mode)
	HasherSocketPath string // /var/run/knirvhasher.sock

	// KNIRVARENA socket path (in-process static bundle server)
	ArenaSocketPath string // /var/lib/knirvserver/sockets/arena.sock

	// KNIRVBASE callback support
	BaseCallbackSocket string `envconfig:"BASE_CALLBACK_SOCKET"`
	BaseNetworkID      string `envconfig:"BASE_NETWORK_ID"`

	// Cloudflare tunnel node identity
	// NetworkMode is one of production, testnet, development, or enterprise.
	// Derived from KNIRV_NETWORK_MODE env var; falls back to KNIRV_TESTNET detection.
	NetworkMode string // KNIRV_NETWORK_MODE / KNIRV_TESTNET
	// PublicURL is the authoritative external HTTPS origin selected by the
	// KNIRVSERVER wrapper and shared with cloudflared.
	PublicURL string // KNIRV_PUBLIC_URL
	// EnterpriseMode and UserIDTag select the Enterprise subscription hostname.
	EnterpriseMode bool   // KNIRV_ENTERPRISE
	UserIDTag      string // KNIRV_USER_ID_TAG
	// NodeRegistrationID is this bootnode's registration ID (from D1 or env).
	// It remains part of node/DHT identity; public DNS now uses deployment class.
	NodeRegistrationID string // KNIRV_NODE_REGISTRATION_ID
	// CloudflareD1DatabaseID enables auto-lookup of NodeRegistrationID from D1
	// when KNIRV_NODE_REGISTRATION_ID is not set explicitly.
	CloudflareD1DatabaseID string // CLOUDFLARE_D1_DATABASE_ID
}

func Load() (*Config, error) {
	// Try to load .env files (don't error if they don't exist)
	_ = godotenv.Load(".env.production")
	_ = godotenv.Load(".env.testnet")
	_ = godotenv.Load(".env")

	cfg := &Config{
		Port:                      getEnvInt("PORT", 8080),
		SocketPath:                getEnv("SOCKET_PATH", ""),
		ChainID:                   getEnv("KNIRV_CHAIN_ID", "testnet"),
		PublicHost:                getEnv("PUBLIC_HOST", "localhost"),
		DisableDHT:                getEnvBool("DISABLE_DHT", false),
		DHTPort:                   getEnvInt("DHT_PORT", 0), // 0 means auto-select
		BootstrapPeers:            getEnvArray("KNIRV_BOOTSTRAP_PEERS", []string{}),
		InternalAPIKey:            getEnv("INTERNAL_API_KEY", ""),
		RenderGatewayAPI:          getEnv("RENDER_GATEWAY_INTERNAL_API", ""),
		NodeJSServicesEnabled:     getEnvBool("NODEJS_SERVICES_ENABLED", true),
		NodeJSServicesAutoStart:   getEnvBool("NODEJS_SERVICES_AUTOSTART", true),
		PaymentGatewayEnabled:     getEnvBool("PAYMENT_GATEWAY_ENABLED", true),
		PaymentGatewayPort:        getEnvInt("PAYMENT_GATEWAY_PORT", 3001),
		TunnelRegistryEnabled:     getEnvBool("TUNNEL_REGISTRY_ENABLED", true),
		TunnelRegistryHTTPPort:    getEnvInt("TUNNEL_REGISTRY_HTTP_PORT", 3002),
		TunnelRegistryControlPort: getEnvInt("TUNNEL_REGISTRY_CONTROL_PORT", 3003),
		TunnelRegistryRelayPort:   getEnvInt("TUNNEL_REGISTRY_PUBLIC_RELAY_PORT", 3004),
		TunnelRegistrySTUNPort:    getEnvInt("TUNNEL_REGISTRY_STUN_PORT", 3005),
		OperatorRegistryEnabled:   getEnvBool("OPERATOR_REGISTRY_ENABLED", false), // Disabled by default
		OperatorRegistryPort:      getEnvInt("OPERATOR_REGISTRY_PORT", 3006),
		WebGUIEnabled:             getEnvBool("WEBGUI_ENABLED", true),
		WebGUIPort:                getEnvInt("WEBGUI_PORT", 3007),
		SessionSecret:             getEnv("SESSION_SECRET", generateSessionSecret()),
		AutoOpenBrowser:           getEnvBool("AUTO_OPEN_BROWSER", true),
		OracleSocketPath:          getEnv("ORACLE_SOCKET_PATH", ""),
		TurnServerEnabled:         getEnvBool("TURN_SERVER_ENABLED", true),
		TurnServerUDPPort:         getEnvInt("TURN_SERVER_UDP_PORT", 3478),
		TurnServerTCPPort:         getEnvInt("TURN_SERVER_TCP_PORT", 3479),
		TurnServerAPIPort:         getEnvInt("TURN_SERVER_API_PORT", 3476),
		TurnServerAuthSecret:      getEnv("TURN_SERVER_AUTH_SECRET", "knirvchain-turn-secret"),
		TurnServerRealm:           getEnv("TURN_SERVER_REALM", "knirvgateway.local"),
		TurnServerMinerAddress:    getEnv("TURN_SERVER_MINER_ADDRESS", "GATEWAY_MINER"),
		ChainNodeRole:             getEnv("CHAIN_NODE_ROLE", "Client"),
		ChainP2PPort:              getEnvInt("CHAIN_P2P_PORT", 4001),
		ChainClientOnly:           getEnvBool("CHAIN_CLIENT_ONLY", true),
		ChainIsBootnode:           getEnvBool("CHAIN_IS_BOOTNODE", false),
		ChainBootnodeRegistry:     getEnv("CHAIN_BOOTNODE_REGISTRY", "https://registry.knirv.com"),
		ChainCallbackSocket:       getEnv("CHAIN_CALLBACK_SOCKET", ""),
		BackendSocketPath:         getEnv("BACKEND_SOCKET_PATH", ""),
		ServerBaseURL:             getEnv("KNIRV_SERVER_BASE_URL", "http://127.0.0.1:8090"),
		ChainSocketPath:           getEnv("CHAIN_SOCKET_PATH", ""),
		InternalAuthToken:         getEnv("KNIRV_INTERNAL_AUTH_TOKEN", ""),
		GraphSocketPath:           getEnv("GRAPH_SOCKET_PATH", ""),
		ShellSocketPath:           getEnv("SHELL_SOCKET_PATH", ""),
		AgentSocketDir:            getEnv("AGENT_SOCKET_DIR", ""),
		AgentMaxConcurrent:        getEnvInt("AGENT_MAX_CONCURRENT", 32),
		HasherSocketPath:          getEnv("HASHER_SOCKET_PATH", "/var/run/knirvhasher.sock"),
		ArenaSocketPath:           getEnv("ARENA_SOCKET_PATH", ""),
		BaseCallbackSocket:        getEnv("BASE_CALLBACK_SOCKET", ""),
		BaseNetworkID:             getEnv("BASE_NETWORK_ID", ""),
		NetworkMode:               resolveNetworkMode(),
		PublicURL:                 getEnv("KNIRV_PUBLIC_URL", ""),
		EnterpriseMode:            getEnvBool("KNIRV_ENTERPRISE", false),
		UserIDTag:                 getEnv("KNIRV_USER_ID_TAG", ""),
		NodeRegistrationID:        getEnv("KNIRV_NODE_REGISTRATION_ID", ""),
		CloudflareD1DatabaseID:    getEnv("CLOUDFLARE_D1_DATABASE_ID", ""),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// resolveNetworkMode returns the explicit deployment class, or a legacy
// production/testnet fallback when KNIRV_NETWORK_MODE is absent.
// Explicit KNIRV_NETWORK_MODE takes priority; otherwise KNIRV_TESTNET=true
// (propagated from KNIRVSERVER's --testnet flag) maps to "testnet".
// Default is "production" so misconfigured nodes do not accidentally create
// testnet tunnels.
func resolveNetworkMode() string {
	if mode := os.Getenv("KNIRV_NETWORK_MODE"); mode != "" {
		return mode
	}
	if getEnvBool("KNIRV_TESTNET", false) {
		return "testnet"
	}
	return "production"
}

func getEnvArray(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

func generateSessionSecret() string {
	// Generate a simple session secret (for development)
	// In production, this should be set via environment variable
	return fmt.Sprintf("knirv-oracle-secret-%d", os.Getpid())
}

// OpenBrowser opens the specified URL in the default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		if os.Getenv("DISPLAY") != "" {
			// Try xdg-open first (most common)
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", url)
			} else if _, err := exec.LookPath("sensible-browser"); err == nil {
				cmd = exec.Command("sensible-browser", url)
			} else if _, err := exec.LookPath("gnome-open"); err == nil {
				cmd = exec.Command("gnome-open", url)
			} else if _, err := exec.LookPath("kde-open"); err == nil {
				cmd = exec.Command("kde-open", url)
			} else {
				return fmt.Errorf("could not find a browser to open URL")
			}
		} else {
			return fmt.Errorf("no display detected for opening browser")
		}
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform for opening browser")
	}

	return cmd.Start()
}
