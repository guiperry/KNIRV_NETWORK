package oracle

import (
	"math/big"
	"testing"
	"time"
)

// clearBlockTimeEnv ensures LoadConfigFromEnv's other required env vars don't
// leak between test cases and that each case starts from a clean slate for
// the two env vars this file is actually about.
func clearBlockTimeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ORACLE_BLOCK_TIME", "")
	t.Setenv("KNIRV_NETWORK_MODE", "")
}

// TestLoadConfigFromEnvDefaultsHeartbeatByNetworkMode is a regression test for
// the "mining" empty blocks every 5s regardless of network mode: with no
// explicit ORACLE_BLOCK_TIME, the heartbeat cadence must now come from
// network mode — a slow keep-alive on testnet, fully disabled (event-only) on
// production — never the old unconditional 5-second default.
func TestLoadConfigFromEnvDefaultsHeartbeatByNetworkMode(t *testing.T) {
	tests := []struct {
		name        string
		networkMode string
		want        time.Duration
	}{
		{"unset defaults to testnet heartbeat", "", 30 * time.Minute},
		{"explicit testnet", "testnet", 30 * time.Minute},
		{"development is not production", "development", 30 * time.Minute},
		{"production disables the heartbeat", "production", 0},
		{"prod alias disables the heartbeat", "prod", 0},
		{"mainnet alias disables the heartbeat", "mainnet", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearBlockTimeEnv(t)
			if tt.networkMode != "" {
				t.Setenv("KNIRV_NETWORK_MODE", tt.networkMode)
			}
			cfg, err := LoadConfigFromEnv()
			if err != nil {
				t.Fatalf("LoadConfigFromEnv: %v", err)
			}
			if cfg.BlockTime != tt.want {
				t.Fatalf("BlockTime = %v, want %v", cfg.BlockTime, tt.want)
			}
		})
	}
}

// TestLoadConfigFromEnvExplicitBlockTimeAlwaysWins verifies an operator's
// explicit ORACLE_BLOCK_TIME overrides the network-mode default in either
// direction (including forcing a heartbeat back on in production).
func TestLoadConfigFromEnvExplicitBlockTimeAlwaysWins(t *testing.T) {
	clearBlockTimeEnv(t)
	t.Setenv("KNIRV_NETWORK_MODE", "production")
	t.Setenv("ORACLE_BLOCK_TIME", "120")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if want := 120 * time.Second; cfg.BlockTime != want {
		t.Fatalf("BlockTime = %v, want %v (explicit override must win over the production default)", cfg.BlockTime, want)
	}
}

// TestValidateConfigAllowsZeroBlockTimeRejectsNegative verifies 0 (heartbeat
// disabled) is accepted — it is the production default — while a negative
// value, which time.NewTicker would panic on, is rejected.
func TestValidateConfigAllowsZeroBlockTimeRejectsNegative(t *testing.T) {
	base := func() *OracleConfig {
		cfg := DefaultOracleConfig()
		cfg.InitialSupply = big.NewInt(1000)
		cfg.MaxSupply = big.NewInt(2000)
		cfg.DataDir = t.TempDir()
		return cfg
	}

	zero := base()
	zero.BlockTime = 0
	if err := ValidateConfig(zero); err != nil {
		t.Fatalf("BlockTime=0 should be valid (disabled heartbeat): %v", err)
	}

	negative := base()
	negative.BlockTime = -time.Second
	if err := ValidateConfig(negative); err == nil {
		t.Fatal("BlockTime<0 should be rejected")
	}
}
