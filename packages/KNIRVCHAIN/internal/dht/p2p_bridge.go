package dht

import (
	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/p2p"
)

// DiscoveryService re-exports the p2p.DiscoveryService interface so callers can use dht.DiscoveryService.
type DiscoveryService = p2p.DiscoveryService

// P2PConsensusManager re-exports p2p.P2PConsensusManager so callers can use dht.P2PConsensusManager.
type P2PConsensusManager = p2p.P2PConsensusManager

// NewP2PConsensusManager creates a P2PConsensusManager; thin wrapper around p2p.NewP2PConsensusManager.
func NewP2PConsensusManager(blockchain p2p.Blockchain, discoveryManager p2p.DiscoveryService, role config.Role) (*p2p.P2PConsensusManager, error) {
	return p2p.NewP2PConsensusManager(blockchain, discoveryManager, role)
}
