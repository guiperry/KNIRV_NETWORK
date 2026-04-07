package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Find home directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	// Default wallet directory
	walletDir := filepath.Join(home, ".knirv", "wallets")

	// Legacy defaults for backward compatibility
	v.SetDefault("node_url", "http://localhost:8545")
	v.SetDefault("wallet_directory", walletDir)
	v.SetDefault("log_level", "info")
	v.SetDefault("default_fee", 1000000000000000) // 0.001 ETH in wei

	// Enhanced KNIRV configuration defaults
	// Network configuration
	v.SetDefault("knirv.network.environment", "development")
	v.SetDefault("knirv.network.discovery.enabled", true)
	v.SetDefault("knirv.network.discovery.interval", "30s")
	v.SetDefault("knirv.network.discovery.timeout", "10s")

	// KNIRVORACLE service defaults (now refers to the web gateway/API)
	v.SetDefault("knirv.services.knirvoracle.url", "http://localhost:9999")
	v.SetDefault("knirv.services.knirvoracle.enabled", true)
	v.SetDefault("knirv.services.knirvoracle.timeout", "30s")
	v.SetDefault("knirv.services.knirvoracle.retries", 3)
	v.SetDefault("knirv.services.knirvoracle.endpoints.api", "/api/v1")
	v.SetDefault("knirv.services.knirvoracle.endpoints.websocket", "/ws")
	v.SetDefault("knirv.services.knirvoracle.endpoints.economics", "/economics")

	// KNIRVORACLED (blockchain daemon) service defaults
	v.SetDefault("knirv.services.knirvoracled.rpc_url", "http://localhost:26657")
	v.SetDefault("knirv.services.knirvoracled.enabled", true)
	v.SetDefault("knirv.services.knirvoracled.timeout", "30s")
	v.SetDefault("knirv.services.knirvoracled.retries", 3)
	v.SetDefault("knirv.services.knirvoracled.endpoints.status", "/status")

	// KNIRVGATEWAY service defaults (renamed from knirvoracle for clarity)
	v.SetDefault("knirv.services.knirvgateway.url", "https://gateway.knirv.network")
	v.SetDefault("knirv.services.knirvgateway.enabled", true)
	v.SetDefault("knirv.services.knirvgateway.timeout", "30s")
	v.SetDefault("knirv.services.knirvgateway.retries", 3)
	v.SetDefault("knirv.services.knirvgateway.endpoints.economics", "/economics")
	v.SetDefault("knirv.services.knirvgateway.endpoints.health", "/health")
	v.SetDefault("knirv.services.knirvgateway.endpoints.poaud", "/poaud")

	// KNIRVSERVER service defaults
	v.SetDefault("knirv.services.knirvserver.url", "http://localhost:8080")
	v.SetDefault("knirv.services.knirvserver.enabled", true)
	v.SetDefault("knirv.services.knirvserver.timeout", "30s")
	v.SetDefault("knirv.services.knirvserver.retries", 3)
	v.SetDefault("knirv.services.knirvserver.endpoints.agentic", "/agentic")
	v.SetDefault("knirv.services.knirvserver.endpoints.inference", "/inference")
	v.SetDefault("knirv.services.knirvserver.endpoints.plugins", "/plugins")

	// KNIRVGRAPH service defaults
	v.SetDefault("knirv.services.knirvgraph.url", "http://localhost:7080")
	v.SetDefault("knirv.services.knirvgraph.enabled", true)
	v.SetDefault("knirv.services.knirvgraph.timeout", "30s")
	v.SetDefault("knirv.services.knirvgraph.retries", 3)
	v.SetDefault("knirv.services.knirvgraph.endpoints.nrv", "/nrv")
	v.SetDefault("knirv.services.knirvgraph.endpoints.graph", "/graph")
	v.SetDefault("knirv.services.knirvgraph.endpoints.transactions", "/transactions")

	// Wallet configuration defaults
	v.SetDefault("knirv.wallet.directory", walletDir)
	v.SetDefault("knirv.wallet.xion.enabled", true)
	v.SetDefault("knirv.wallet.xion.chain_id", "knirv-mainnet-1")
	v.SetDefault("knirv.wallet.xion.meta_account", true)
	v.SetDefault("knirv.wallet.xion.gasless", true)
	v.SetDefault("knirv.wallet.nrn.enabled", true)
	v.SetDefault("knirv.wallet.nrn.faucet_url", "http://localhost:9999/faucet")
	v.SetDefault("knirv.wallet.nrn.auto_refill", true)
	v.SetDefault("knirv.wallet.nrn.min_balance", "1000")

	// Real-time communication defaults
	v.SetDefault("knirv.realtime.websocket.enabled", true)
	v.SetDefault("knirv.realtime.websocket.reconnect_interval", "5s")
	v.SetDefault("knirv.realtime.websocket.max_retries", 3)
	v.SetDefault("knirv.realtime.sse.enabled", true)
	v.SetDefault("knirv.realtime.sse.timeout", "30s")
	v.SetDefault("knirv.realtime.sse.buffer_size", 1024)

	// File server defaults
	v.SetDefault("file_server.enabled", false)
	v.SetDefault("file_server.port", 8080)
	v.SetDefault("file_server.base_url", "http://localhost:8080")

	// AI defaults
	v.SetDefault("ai.provider", "openai")
	v.SetDefault("ai.base_url", "")
	v.SetDefault("ai.default_model", "gpt-4")
	v.SetDefault("ai.max_tokens", 4000)
	v.SetDefault("ai.temperature", 0.7)

	// UI defaults
	v.SetDefault("ui.enable_tui", true)
	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.color_mode", "256")
	v.SetDefault("ui.show_icons", true)
	v.SetDefault("ui.animation_speed", 100)
	v.SetDefault("ui.compact_mode", false)
}
