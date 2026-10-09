package knirvarena

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

const (
	testnetServerURL = "https://testnet-gateway.knirv.com"
	mainnetServerURL = "https://gateway.knirv.com"
)

// RuntimeConfig is served to the KNIRVARENA bundle at /runtime-config.json.
// The bundle is built once and embedded, so the network it talks to is chosen
// here at runtime from the server's deployment class rather than baked in at
// build time.
type RuntimeConfig struct {
	Network   string `json:"network"`
	ServerURL string `json:"serverUrl"`
}

// RuntimeConfigFromEnv follows the KNIRVSERVER launcher: KNIRV_NETWORK_MODE
// is "testnet" unless the server was started with -prod (production). Set
// KNIRV_ARENA_SERVER_URL to point the arena at a different KNIRVSERVER.
func RuntimeConfigFromEnv() RuntimeConfig {
	cfg := RuntimeConfig{Network: "testnet", ServerURL: testnetServerURL}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KNIRV_NETWORK_MODE"))) {
	case "production", "prod", "mainnet":
		cfg = RuntimeConfig{Network: "mainnet", ServerURL: mainnetServerURL}
	}
	if override := strings.TrimSpace(os.Getenv("KNIRV_ARENA_SERVER_URL")); override != "" {
		cfg.ServerURL = strings.TrimRight(override, "/")
	}
	return cfg
}

func handleRuntimeConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(RuntimeConfigFromEnv())
}
