package crosschain

import (
	"fmt"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// BridgeConfig represents the configuration for a cross-chain bridge
type BridgeConfig struct {
	ChainID        types.ChainID `json:"chain_id"`
	ChainName      string        `json:"chain_name"`
	ChannelID      string        `json:"channel_id"`
	PortID         string        `json:"port_id"`
	ConnectionID   string        `json:"connection_id"`
	ClientID       string        `json:"client_id"`
	TrustLevel     float64       `json:"trust_level"`      // 0.0 to 1.0
	FeeBasisPoints uint64        `json:"fee_basis_points"` // Fee in basis points (1/10000)
	MinTransfer    uint64        `json:"min_transfer"`     // Minimum transfer amount
	MaxTransfer    uint64        `json:"max_transfer"`     // Maximum transfer amount
	Enabled        bool          `json:"enabled"`
}

// BridgeManager manages bridge configurations for all chains
type BridgeManager struct {
	bridges map[types.ChainID]*BridgeConfig
	mu      sync.RWMutex
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager() *BridgeManager {
	return &BridgeManager{
		bridges: make(map[types.ChainID]*BridgeConfig),
	}
}

// RegisterBridge registers a new bridge configuration
func (bm *BridgeManager) RegisterBridge(config *BridgeConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.bridges[config.ChainID]; exists {
		return fmt.Errorf("bridge for chain %s already exists", config.ChainID.String())
	}

	// Validate config
	if err := validateBridgeConfig(config); err != nil {
		return fmt.Errorf("invalid bridge config: %w", err)
	}

	bm.bridges[config.ChainID] = config
	return nil
}

// GetBridge retrieves a bridge configuration for a chain
func (bm *BridgeManager) GetBridge(chainID types.ChainID) (*BridgeConfig, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return nil, fmt.Errorf("bridge not found for chain %s", chainID.String())
	}

	return bridge, nil
}

// UpdateBridge updates a bridge configuration
func (bm *BridgeManager) UpdateBridge(config *BridgeConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.bridges[config.ChainID]; !exists {
		return fmt.Errorf("bridge not found for chain %s", config.ChainID.String())
	}

	// Validate config
	if err := validateBridgeConfig(config); err != nil {
		return fmt.Errorf("invalid bridge config: %w", err)
	}

	bm.bridges[config.ChainID] = config
	return nil
}

// EnableBridge enables a bridge
func (bm *BridgeManager) EnableBridge(chainID types.ChainID) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return fmt.Errorf("bridge not found for chain %s", chainID.String())
	}

	bridge.Enabled = true
	return nil
}

// DisableBridge disables a bridge
func (bm *BridgeManager) DisableBridge(chainID types.ChainID) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return fmt.Errorf("bridge not found for chain %s", chainID.String())
	}

	bridge.Enabled = false
	return nil
}

// ListBridges returns all bridge configurations
func (bm *BridgeManager) ListBridges() []*BridgeConfig {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bridges := make([]*BridgeConfig, 0, len(bm.bridges))
	for _, bridge := range bm.bridges {
		bridges = append(bridges, bridge)
	}

	return bridges
}

// CalculateFee calculates the transfer fee for a given amount
func (bm *BridgeManager) CalculateFee(chainID types.ChainID, amount uint64) (uint64, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return 0, fmt.Errorf("bridge not found for chain %s", chainID.String())
	}

	// Fee = amount * (feeBasisPoints / 10000)
	fee := (amount * bridge.FeeBasisPoints) / 10000
	return fee, nil
}

// ValidateTransferAmount validates if a transfer amount is within limits
func (bm *BridgeManager) ValidateTransferAmount(chainID types.ChainID, amount uint64) error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return fmt.Errorf("bridge not found for chain %s", chainID.String())
	}

	if !bridge.Enabled {
		return fmt.Errorf("bridge for chain %s is disabled", chainID.String())
	}

	if amount < bridge.MinTransfer {
		return fmt.Errorf("amount %d below minimum transfer of %d", amount, bridge.MinTransfer)
	}

	if amount > bridge.MaxTransfer {
		return fmt.Errorf("amount %d exceeds maximum transfer of %d", amount, bridge.MaxTransfer)
	}

	return nil
}

// IsBridgeEnabled checks if a bridge is enabled
func (bm *BridgeManager) IsBridgeEnabled(chainID types.ChainID) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bridge, exists := bm.bridges[chainID]
	if !exists {
		return false
	}

	return bridge.Enabled
}

// validateBridgeConfig validates a bridge configuration
func validateBridgeConfig(config *BridgeConfig) error {
	if config.ChainName == "" {
		return fmt.Errorf("chain name is required")
	}

	if config.ChannelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	if config.PortID == "" {
		return fmt.Errorf("port ID is required")
	}

	if config.ConnectionID == "" {
		return fmt.Errorf("connection ID is required")
	}

	if config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	if config.TrustLevel < 0 || config.TrustLevel > 1 {
		return fmt.Errorf("trust level must be between 0 and 1")
	}

	if config.FeeBasisPoints > 10000 {
		return fmt.Errorf("fee basis points cannot exceed 10000")
	}

	if config.MinTransfer > config.MaxTransfer {
		return fmt.Errorf("min transfer cannot exceed max transfer")
	}

	return nil
}

// DefaultBridgeConfigs returns default bridge configurations for all KNIRV chains
func DefaultBridgeConfigs() []*BridgeConfig {
	return []*BridgeConfig{
		{
			ChainID:        types.ChainKnirvChain,
			ChainName:      "KNIRV Chain",
			ChannelID:      "channel-0",
			PortID:         "port-0",
			ConnectionID:   "connection-0",
			ClientID:       "07-tendermint-0",
			TrustLevel:     0.66,
			FeeBasisPoints: 10,         // 0.1% fee
			MinTransfer:    100,        // Minimum 100 units
			MaxTransfer:    1000000000, // Maximum 1B units
			Enabled:        true,
		},
		{
			ChainID:        types.ChainKnirvNexus,
			ChainName:      "KNIRV Nexus",
			ChannelID:      "channel-1",
			PortID:         "port-1",
			ConnectionID:   "connection-1",
			ClientID:       "07-tendermint-1",
			TrustLevel:     0.66,
			FeeBasisPoints: 10,
			MinTransfer:    100,
			MaxTransfer:    1000000000,
			Enabled:        true,
		},
		{
			ChainID:        types.ChainKnirvGraph,
			ChainName:      "KNIRV Graph",
			ChannelID:      "channel-2",
			PortID:         "port-2",
			ConnectionID:   "connection-2",
			ClientID:       "07-tendermint-2",
			TrustLevel:     0.66,
			FeeBasisPoints: 10,
			MinTransfer:    100,
			MaxTransfer:    1000000000,
			Enabled:        true,
		},
		{
			ChainID:        types.ChainKnirvRouter,
			ChainName:      "KNIRV Router",
			ChannelID:      "channel-3",
			PortID:         "port-3",
			ConnectionID:   "connection-3",
			ClientID:       "07-tendermint-3",
			TrustLevel:     0.66,
			FeeBasisPoints: 10,
			MinTransfer:    100,
			MaxTransfer:    1000000000,
			Enabled:        true,
		},
		{
			ChainID:        types.ChainXion,
			ChainName:      "XION",
			ChannelID:      "channel-xion",
			PortID:         "port-xion",
			ConnectionID:   "connection-xion",
			ClientID:       "07-tendermint-xion",
			TrustLevel:     0.66,
			FeeBasisPoints: 25,        // 0.25% fee for external chain
			MinTransfer:    1000,      // Higher minimum for external
			MaxTransfer:    100000000, // Lower maximum for external
			Enabled:        true,
		},
	}
}
