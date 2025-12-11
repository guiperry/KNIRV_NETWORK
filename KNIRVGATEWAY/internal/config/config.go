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

	// WebGUI
	WebGUIEnabled bool
	WebGUIPort    int

	// Session configuration
	SessionSecret string

	// Browser auto-open configuration
	AutoOpenBrowser bool

	// KNIRV-ORACLE configuration
	KnirvOracleURL string
}

func Load() (*Config, error) {
	// Try to load .env files (don't error if they don't exist)
	_ = godotenv.Load(".env.production")
	_ = godotenv.Load(".env.testnet")
	_ = godotenv.Load(".env")

	cfg := &Config{
		GatewayMode:               getEnv("GATEWAY_MODE", "persistent"),
		Port:                      getEnvInt("PORT", 8080),
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
	return fmt.Sprintf("knirv-gateway-secret-%d", os.Getpid())
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
