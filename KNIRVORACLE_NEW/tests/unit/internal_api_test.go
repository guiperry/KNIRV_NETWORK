package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"KNIRVORACLE/pkg/types"

	"github.com/gorilla/mux"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestBlockchainServer is a test struct that mimics the real server
type TestBlockchainServer struct {
	discoveryManager *MockDiscoveryService
	db               Database
	BlockchainPtr    Blockchain
}

// handleInternalDHTFindResource is a test implementation of the DHT find resource endpoint
func (s *TestBlockchainServer) handleInternalDHTFindResource(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	resourceType := r.URL.Query().Get("type")

	if id == "" || resourceType == "" {
		http.Error(w, "Missing required parameters: id and type", http.StatusBadRequest)
		return
	}

	// Call the discovery service to find the resource
	peers, err := s.discoveryManager.FindResource(r.Context(), id, types.DiscoveryResourceType(resourceType))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error finding resource: %v", err), http.StatusNotFound)
		return
	}

	// Return the peers as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

// handleInternalDHTAnnounceResource is a test implementation of the DHT announce resource endpoint
func (s *TestBlockchainServer) handleInternalDHTAnnounceResource(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Multiaddress string `json:"multiaddress"`
	}

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the request
	if request.ID == "" || request.Type == "" {
		http.Error(w, "Missing required parameters: id and type", http.StatusBadRequest)
		return
	}

	// Call the discovery service to announce the resource
	err := s.discoveryManager.AnnounceMintedResource(r.Context(), request.ID, types.DiscoveryResourceType(request.Type))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error announcing resource: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// handleInternalDBGetCapability is a test implementation of the DB get capability endpoint
func (s *TestBlockchainServer) handleInternalDBGetCapability(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
		return
	}

	// Call the database to get the capability
	capability, err := s.db.GetCapabilityByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting capability: %v", err), http.StatusNotFound)
		return
	}

	// Return the capability as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capability)
}

// handleInternalDBIDExists is a test implementation of the DB ID exists endpoint
func (s *TestBlockchainServer) handleInternalDBIDExists(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
		return
	}

	// Check if the ID exists in the blockchain
	existsInBlocks := s.BlockchainPtr.CheckIfIDExistsInBlocks(id)
	existsInPool := s.BlockchainPtr.CheckIfIDExistsInTransactionPool(id)

	// Return the result as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": existsInBlocks || existsInPool})
}

// MockDiscoveryService is a mock implementation of the DiscoveryService interface
type MockDiscoveryService struct {
	mock.Mock
}

func (m *MockDiscoveryService) FindGenericResource(id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error) {
	args := m.Called(id, resourceType)
	return args.Get(0).([]peer.AddrInfo), args.Error(1)
}

func (m *MockDiscoveryService) AnnounceGenericResource(id string, resourceType types.DiscoveryResourceType) error {
	args := m.Called(id, resourceType)
	return args.Error(0)
}

func (m *MockDiscoveryService) FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error) {
	args := m.Called(ctx, id, mcpTypeString)
	return args.Get(0).([]peer.AddrInfo), args.Error(1)
}

func (m *MockDiscoveryService) AnnounceMCPCapability(ctx context.Context, id string, mcpTypeString string) error {
	args := m.Called(ctx, id, mcpTypeString)
	return args.Error(0)
}

func (m *MockDiscoveryService) FindResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error) {
	args := m.Called(ctx, id, resourceType)
	return args.Get(0).([]peer.AddrInfo), args.Error(1)
}

func (m *MockDiscoveryService) AnnounceMintedResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) error {
	args := m.Called(ctx, id, resourceType)
	return args.Error(0)
}

func (m *MockDiscoveryService) Run(interval time.Duration) {
	m.Called(interval)
}

func (m *MockDiscoveryService) GetPeerID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDiscoveryService) GetSelfMultiaddrs() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockDiscoveryService) Close() {
	m.Called()
}

func (m *MockDiscoveryService) IsRecentlyRegistered(nodeID string) bool {
	args := m.Called(nodeID)
	return args.Bool(0)
}

func (m *MockDiscoveryService) ConnectToPeer(nodeInfo peer.AddrInfo, ctx context.Context) error {
	args := m.Called(nodeInfo, ctx)
	return args.Error(0)
}

// Blockchain defines the test interface for blockchain operations
type Blockchain interface {
	CheckIfIDExistsInBlocks(id string) bool
	CheckIfIDExistsInTransactionPool(id string) bool
}

// MockBlockchain is a mock implementation of the Blockchain
type MockBlockchain struct {
	mock.Mock
}

func (m *MockBlockchain) CheckIfIDExistsInBlocks(id string) bool {
	args := m.Called(id)
	return args.Bool(0)
}

func (m *MockBlockchain) CheckIfIDExistsInTransactionPool(id string) bool {
	args := m.Called(id)
	return args.Bool(0)
}

// Database defines the test interface for database operations
type Database interface {
	GetCapabilityByID(id string) (map[string]interface{}, error)
}

// MockDB is a mock implementation of the Database interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetCapabilityByID(id string) (map[string]interface{}, error) {
	args := m.Called(id)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func TestHandleInternalDHTFindResource(t *testing.T) {
	// Create a mock discovery service
	mockDiscovery := new(MockDiscoveryService)

	// Create a test server
	bcs := &TestBlockchainServer{
		discoveryManager: mockDiscovery,
	}

	// Create a router
	router := mux.NewRouter()
	router.HandleFunc("/internal/dht/findResource", bcs.handleInternalDHTFindResource)

	// Test cases
	tests := []struct {
		name           string
		id             string
		resourceType   string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:         "Successful resource lookup",
			id:           "test-resource",
			resourceType: "chain",
			mockSetup: func() {
				// Mock the FindResource method to return a successful result
				// Create a valid peer ID
				validPeerID, _ := peer.Decode("12D3KooWEKxzRUXdYBDzZbZvbzEaWfH4RaYJWZpLgKKQGKKGKKGK")
				validMultiaddr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001/p2p/12D3KooWEKxzRUXdYBDzZbZvbzEaWfH4RaYJWZpLgKKQGKKGKKGK")

				mockDiscovery.On("FindResource", mock.Anything, "test-resource", types.DiscoveryResourceType("chain")).Return(
					[]peer.AddrInfo{
						{
							ID: validPeerID,
							Addrs: []multiaddr.Multiaddr{
								validMultiaddr,
							},
						},
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func() []peer.AddrInfo {
				validPeerID, _ := peer.Decode("12D3KooWEKxzRUXdYBDzZbZvbzEaWfH4RaYJWZpLgKKQGKKGKKGK")
				validMultiaddr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001/p2p/12D3KooWEKxzRUXdYBDzZbZvbzEaWfH4RaYJWZpLgKKQGKKGKKGK")
				return []peer.AddrInfo{
					{
						ID: validPeerID,
						Addrs: []multiaddr.Multiaddr{
							validMultiaddr,
						},
					},
				}
			}(),
		},
		{
			name:         "Resource not found",
			id:           "non-existent",
			resourceType: "chain",
			mockSetup: func() {
				// Mock the FindResource method to return an error
				mockDiscovery.On("FindResource", mock.Anything, "non-existent", types.DiscoveryResourceType("chain")).Return(
					[]peer.AddrInfo{},
					fmt.Errorf("no providers found"),
				)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Missing parameters",
			id:             "",
			resourceType:   "",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockDiscovery.ExpectedCalls = nil

			// Setup mock
			tt.mockSetup()

			// Create a request
			req, err := http.NewRequest("GET", "/internal/dht/findResource?id="+tt.id+"&type="+tt.resourceType, nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If we expect a successful response, check the body
			if tt.expectedStatus == http.StatusOK {
				var response []peer.AddrInfo
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			// Verify that all expected calls were made
			mockDiscovery.AssertExpectations(t)
		})
	}
}

func TestHandleInternalDHTAnnounceResource(t *testing.T) {
	// Create a mock discovery service
	mockDiscovery := new(MockDiscoveryService)

	// Create a test server
	bcs := &TestBlockchainServer{
		discoveryManager: mockDiscovery,
	}

	// Create a router
	router := mux.NewRouter()
	router.HandleFunc("/internal/dht/announceResource", bcs.handleInternalDHTAnnounceResource)

	// Test cases
	tests := []struct {
		name           string
		requestBody    map[string]string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name: "Successful resource announcement",
			requestBody: map[string]string{
				"id":           "test-resource",
				"type":         "chain",
				"multiaddress": "/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerId123",
			},
			mockSetup: func() {
				// Mock the AnnounceMintedResource method to return success
				mockDiscovery.On("AnnounceMintedResource", mock.Anything, "test-resource", types.DiscoveryResourceType("chain")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Failed resource announcement",
			requestBody: map[string]string{
				"id":           "test-resource",
				"type":         "chain",
				"multiaddress": "/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerId123",
			},
			mockSetup: func() {
				// Mock the AnnounceMintedResource method to return an error
				mockDiscovery.On("AnnounceMintedResource", mock.Anything, "test-resource", types.DiscoveryResourceType("chain")).Return(
					fmt.Errorf("failed to announce resource"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Missing parameters",
			requestBody: map[string]string{
				"id": "test-resource",
				// Missing type
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockDiscovery.ExpectedCalls = nil

			// Setup mock
			tt.mockSetup()

			// Create a request
			requestBody, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)
			req, err := http.NewRequest("POST", "/internal/dht/announceResource", bytes.NewBuffer(requestBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify that all expected calls were made
			mockDiscovery.AssertExpectations(t)
		})
	}
}

func TestHandleInternalDBGetCapability(t *testing.T) {
	// Create a mock DB
	mockDB := new(MockDB)

	// Create a test server
	bcs := &TestBlockchainServer{
		db: mockDB,
	}

	// Create a router
	router := mux.NewRouter()
	router.HandleFunc("/internal/db/getCapability", bcs.handleInternalDBGetCapability)

	// Test cases
	tests := []struct {
		name           string
		id             string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "Successful capability lookup",
			id:   "test-capability",
			mockSetup: func() {
				// Mock the GetCapabilityByID method to return a successful result
				mockDB.On("GetCapabilityByID", "test-capability").Return(
					map[string]interface{}{
						"ID":         "test-capability",
						"ProviderID": "QmTestPeerId123",
						"Type":       "test-type",
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":         "test-capability",
				"ProviderID": "QmTestPeerId123",
				"Type":       "test-type",
			},
		},
		{
			name: "Capability not found",
			id:   "non-existent",
			mockSetup: func() {
				// Mock the GetCapabilityByID method to return an error
				mockDB.On("GetCapabilityByID", "non-existent").Return(
					map[string]interface{}{},
					fmt.Errorf("capability not found"),
				)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Missing ID parameter",
			id:             "",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockDB.ExpectedCalls = nil

			// Setup mock
			tt.mockSetup()

			// Create a request
			req, err := http.NewRequest("GET", "/internal/db/getCapability?id="+tt.id, nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If we expect a successful response, check the body
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			// Verify that all expected calls were made
			mockDB.AssertExpectations(t)
		})
	}
}

func TestHandleInternalDBIDExists(t *testing.T) {
	// Create a mock blockchain
	mockBlockchain := new(MockBlockchain)

	// Create a test server
	bcs := &TestBlockchainServer{
		BlockchainPtr: mockBlockchain,
	}

	// Create a router
	router := mux.NewRouter()
	router.HandleFunc("/internal/db/idExists", bcs.handleInternalDBIDExists)

	// Test cases
	tests := []struct {
		name           string
		id             string
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]bool
	}{
		{
			name: "ID exists in blocks",
			id:   "existing-id",
			mockSetup: func() {
				// Mock the CheckIfIDExistsInBlocks method to return true
				mockBlockchain.On("CheckIfIDExistsInBlocks", "existing-id").Return(true)
				// Mock the CheckIfIDExistsInTransactionPool method to return false
				mockBlockchain.On("CheckIfIDExistsInTransactionPool", "existing-id").Return(false)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]bool{"exists": true},
		},
		{
			name: "ID exists in transaction pool",
			id:   "pending-id",
			mockSetup: func() {
				// Mock the CheckIfIDExistsInBlocks method to return false
				mockBlockchain.On("CheckIfIDExistsInBlocks", "pending-id").Return(false)
				// Mock the CheckIfIDExistsInTransactionPool method to return true
				mockBlockchain.On("CheckIfIDExistsInTransactionPool", "pending-id").Return(true)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]bool{"exists": true},
		},
		{
			name: "ID does not exist",
			id:   "non-existent-id",
			mockSetup: func() {
				// Mock the CheckIfIDExistsInBlocks method to return false
				mockBlockchain.On("CheckIfIDExistsInBlocks", "non-existent-id").Return(false)
				// Mock the CheckIfIDExistsInTransactionPool method to return false
				mockBlockchain.On("CheckIfIDExistsInTransactionPool", "non-existent-id").Return(false)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]bool{"exists": false},
		},
		{
			name:           "Missing ID parameter",
			id:             "",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockBlockchain.ExpectedCalls = nil

			// Setup mock
			tt.mockSetup()

			// Create a request
			req, err := http.NewRequest("GET", "/internal/db/idExists?id="+tt.id, nil)
			assert.NoError(t, err)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If we expect a successful response, check the body
			if tt.expectedStatus == http.StatusOK {
				var response map[string]bool
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			// Verify that all expected calls were made
			mockBlockchain.AssertExpectations(t)
		})
	}
}
