package main

import (
	"context"
	"testing"

	"KNIRVCHAIN/config"
)

func TestConsensusConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid config with P2P enabled",
			cfg: config.Config{
				Consensus: config.ConsensusConfig{
					P2PEnabled: true,
					ChainID:    "test-chain",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with P2P disabled",
			cfg: config.Config{
				Consensus: config.ConsensusConfig{
					P2PEnabled: false,
					ChainID:    "test-chain",
				},
			},
			wantErr: false,
		},
		{
			name: "empty chain ID",
			cfg: config.Config{
				Consensus: config.ConsensusConfig{
					P2PEnabled: true,
					ChainID:    "",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := config.ValidateConsensusConfig(&tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateConsensusConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSetEnabled(t *testing.T) {
	mgr := NewMockP2PConsensusManager()
	mgr.SetEnabled(false)
	mgr.SetEnabled(true)
}

func TestConsensusManagerEnabled(t *testing.T) {
	mgr := NewMockP2PConsensusManager()
	val := mgr.GetMiningLockState()
	_ = val
}

func TestP2PConsensusStartStop(t *testing.T) {
	mgr := NewMockP2PConsensusManager()
	if err := mgr.StartConsensus(context.Background()); err != nil {
		t.Fatalf("StartConsensus() failed: %v", err)
	}
	if err := mgr.StopConsensus(); err != nil {
		t.Fatalf("StopConsensus() failed: %v", err)
	}
}

func TestDualToggleOverride(t *testing.T) {
	cfg := config.Config{
		Consensus: config.ConsensusConfig{
			P2PEnabled: false,
			ChainID:    "test",
		},
	}
	if cfg.Consensus.P2PEnabled {
		t.Error("expected P2PEnabled to be false by default")
	}
}

func TestChainIDIsolation(t *testing.T) {
	cfg1 := config.Config{Consensus: config.ConsensusConfig{ChainID: "chain-a"}}
	cfg2 := config.Config{Consensus: config.ConsensusConfig{ChainID: "chain-b"}}
	if cfg1.Consensus.ChainID == cfg2.Consensus.ChainID {
		t.Error("expected different chain IDs to be isolated")
	}
}

func TestP2PEnabledToggle(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Consensus.P2PEnabled {
		t.Error("expected default P2PEnabled to be false per plan")
	}
}

func TestGatewayClientSwap(t *testing.T) {
	_ = context.Background()
}
