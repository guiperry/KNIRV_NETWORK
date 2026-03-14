//go:build mock

package p2p

import (
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// MockDHTManager is a mock implementation of DHTManagerInterface for testing
type MockDHTManager struct {
	MockStart           func() error
	MockStop            func()
	MockIsNetworkPaused func() bool
	MockAnnounceSkill   func(skillID, name, description, category string, metadata map[string]string) error
	MockAnnounceCapability func(capabilityID, name, description string, schema interface{}, metadata map[string]string) error
	MockAnnounceProperty func(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error
	MockFindGraphServices func() ([]peer.AddrInfo, error)

	// Keep track of calls
	StartCalled bool
	StopCalled  bool
	PauseMutex  sync.RWMutex
}

// NewDHTManager creates a new MockDHTManager for test builds
// This function will be used instead of the production NewDHTManager when the "test" build tag is active.
func NewDHTManager(serviceID, chainID string, bootstrapPeers []string, _ bool) (DHTManagerInterface, error) {
	return &MockDHTManager{
		MockIsNetworkPaused: func() bool { return false }, // Default to not paused
		MockStart: func() error {
			log.Println("MockDHTManager: Start called")
			return nil
		},
		MockStop: func() {
		},
		MockAnnounceSkill: func(skillID, name, description, category string, metadata map[string]string) error {
			log.Printf("MockDHTManager: AnnounceSkill called for %s", skillID)
			return nil
		},
		MockAnnounceCapability: func(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
			log.Printf("MockDHTManager: AnnounceCapability called for %s", capabilityID)
			return nil
		},
		MockAnnounceProperty: func(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
			log.Printf("MockDHTManager: AnnounceProperty called for %s", propertyID)
			return nil
		},
		MockFindGraphServices: func() ([]peer.AddrInfo, error) {
			log.Println("MockDHTManager: FindGraphServices called")
			return nil, nil
		},
	}, nil
}

// Start implements DHTManagerInterface
func (m *MockDHTManager) Start() error {
	m.StartCalled = true
	if m.MockStart != nil {
		return m.MockStart()
	}
	return nil
}

// Stop implements DHTManagerInterface
func (m *MockDHTManager) Stop() {
	m.StopCalled = true
	if m.MockStop != nil {
		m.MockStop()
	}
}

// IsNetworkPaused implements DHTManagerInterface
func (m *MockDHTManager) IsNetworkPaused() bool {
	if m.MockIsNetworkPaused != nil {
		return m.MockIsNetworkPaused()
	}
	return false
}

// AnnounceSkill implements DHTManagerInterface
func (m *MockDHTManager) AnnounceSkill(skillID, name, description, category string, metadata map[string]string) error {
	if m.MockAnnounceSkill != nil {
		return m.MockAnnounceSkill(skillID, name, description, category, metadata)
	}
	return nil
}

// AnnounceCapability implements DHTManagerInterface
func (m *MockDHTManager) AnnounceCapability(capabilityID, name, description string, schema interface{}, metadata map[string]string) error {
	if m.MockAnnounceCapability != nil {
		return m.MockAnnounceCapability(capabilityID, name, description, schema, metadata)
	}
	return nil
}

// AnnounceProperty implements DHTManagerInterface
func (m *MockDHTManager) AnnounceProperty(propertyID, name, propertyType string, value interface{}, metadata map[string]string) error {
	if m.MockAnnounceProperty != nil {
		return m.MockAnnounceProperty(propertyID, name, propertyType, value, metadata)
	}
	return nil
}

// FindGraphServices implements DHTManagerInterface
func (m *MockDHTManager) FindGraphServices() ([]peer.AddrInfo, error) {
	if m.MockFindGraphServices != nil {
		return m.MockFindGraphServices()
	}
	return nil, nil
}
