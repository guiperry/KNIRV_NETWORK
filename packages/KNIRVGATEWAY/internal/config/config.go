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
	GatewayMode string
	Port        int
	SocketPath  string
	ChainID     string
	PublicHost  string

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

	// KNIRV-ORACLE configuration
	KnirvOracleURL string

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
}

func Load() (*Config, error) {
	// Try to load .env files (don't error if they don't exist)
	_ = godotenv.Load(".env.production")
	_ = godotenv.Load(".env.testnet")
	_ = godotenv.Load(".env")

	cfg := &Config{
		GatewayMode:               getEnv("GATEWAY_MODE", "persistent"),
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
		KnirvOracleURL:            getEnv("KNIRV_ORACLE_URL", "http://localhost:1317"),
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
