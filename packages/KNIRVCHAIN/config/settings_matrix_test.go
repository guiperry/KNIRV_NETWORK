package config

import "testing"

func TestRoleDefaultsDisableChainWalletServer(t *testing.T) {
	roles := []Role{Root, RoleBootnode, RolePeer, RoleClient}

	for _, role := range roles {
		cfg := DefaultConfig()
		ApplyRoleDefaults(cfg, role)

		if !cfg.NoWalletServer {
			t.Fatalf("role %s should always disable the KNIRVCHAIN wallet server", role)
		}
		if cfg.WalletPort != 0 {
			t.Fatalf("role %s should not advertise a KNIRVCHAIN wallet port, got %d", role, cfg.WalletPort)
		}
	}
}
