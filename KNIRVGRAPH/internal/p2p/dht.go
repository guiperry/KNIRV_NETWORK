package p2p

import "github.com/libp2p/go-libp2p/core/peer"

// DHTManagerInterface defines the interface for DHTManager operations
type DHTManagerInterface interface {
	Start() error
	Stop()
	IsNetworkPaused() bool
	AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error
	AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error
	AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error
	FindGraphServices() ([]peer.AddrInfo, error)
}
