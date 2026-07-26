package distributed

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	coll "github.com/knirvcorp/knirvbase/internal/collection"
	netpkg "github.com/knirvcorp/knirvbase/internal/network"
	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	stor "github.com/knirvcorp/knirvbase/internal/storage"
	typ "github.com/knirvcorp/knirvbase/internal/types"
)

type mockStorage struct{}

func (m *mockStorage) Insert(ctx context.Context, collection string, doc map[string]interface{}) error {
	return nil
}
func (m *mockStorage) Update(ctx context.Context, collection, id string, update map[string]interface{}) error {
	return nil
}
func (m *mockStorage) Delete(ctx context.Context, collection, id string) error { return nil }
func (m *mockStorage) Find(ctx context.Context, collection, id string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStorage) FindAll(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStorage) Put(ctx context.Context, key string, value []byte) error            { return nil }
func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, error)                { return nil, nil }
func (m *mockStorage) DeleteKey(ctx context.Context, key string) error                    { return nil }
func (m *mockStorage) StoreObject(ctx context.Context, key string, obj interface{}) error { return nil }
func (m *mockStorage) GetObject(ctx context.Context, key string, dest interface{}) error  { return nil }
func (m *mockStorage) ProjectToMarkdown(ctx context.Context, key string, targetPath string) error {
	return nil
}
func (m *mockStorage) CreateIndex(ctx context.Context, collection, name string, indexType stor.IndexType, fields []string, unique bool, partialExpr string, options map[string]interface{}) error {
	return nil
}
func (m *mockStorage) DropIndex(ctx context.Context, collection, name string) error      { return nil }
func (m *mockStorage) GetIndex(ctx context.Context, collection, name string) *stor.Index { return nil }
func (m *mockStorage) GetIndexesForCollection(ctx context.Context, collection string) []*stor.Index {
	return nil
}
func (m *mockStorage) QueryIndex(ctx context.Context, collection, indexName string, query map[string]interface{}) ([]string, error) {
	return nil, nil
}
func (m *mockStorage) Close() error { return nil }

type mockNetwork struct {
	networks  map[string]*typ.NetworkConfig
	peerID    string
	mu        sync.Mutex
	handlers  map[typ.MessageType][]netpkg.MessageHandler
	kvStore   map[string][]netpkg.DHTValue
	peers     map[string]*typ.PeerInfo
	connected bool
}

func newMockNetwork() *mockNetwork {
	return &mockNetwork{
		networks: make(map[string]*typ.NetworkConfig),
		peerID:   "mock-peer-id",
		handlers: make(map[typ.MessageType][]netpkg.MessageHandler),
		kvStore:  make(map[string][]netpkg.DHTValue),
		peers:    make(map[string]*typ.PeerInfo),
	}
}

func (m *mockNetwork) Initialize() error {
	m.connected = true
	return nil
}
func (m *mockNetwork) CreateNetwork(cfg typ.NetworkConfig) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networks[cfg.NetworkID] = &cfg
	return cfg.NetworkID, nil
}
func (m *mockNetwork) JoinNetwork(networkID string, bootstrapPeers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networks[networkID] = &typ.NetworkConfig{NetworkID: networkID, Name: "Mock " + networkID}
	return nil
}
func (m *mockNetwork) LeaveNetwork(networkID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.networks, networkID)
	return nil
}
func (m *mockNetwork) AddCollectionToNetwork(networkID, collectionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if net, ok := m.networks[networkID]; ok {
		net.Collections[collectionName] = true
	}
	return nil
}
func (m *mockNetwork) RemoveCollectionFromNetwork(networkID, collectionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if net, ok := m.networks[networkID]; ok {
		delete(net.Collections, collectionName)
	}
	return nil
}
func (m *mockNetwork) GetNetworkCollections(networkID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if net, ok := m.networks[networkID]; ok {
		var cols []string
		for k := range net.Collections {
			cols = append(cols, k)
		}
		return cols
	}
	return nil
}
func (m *mockNetwork) BroadcastMessage(networkID string, msg typ.ProtocolMessage) error { return nil }
func (m *mockNetwork) SendToPeer(peerID string, networkID string, msg typ.ProtocolMessage) error {
	return nil
}
func (m *mockNetwork) OnMessage(mt typ.MessageType, handler netpkg.MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[mt] = append(m.handlers[mt], handler)
}
func (m *mockNetwork) GetNetworkStats(networkID string) *typ.NetworkStats { return nil }
func (m *mockNetwork) GetNetworks() []*typ.NetworkConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	var nets []*typ.NetworkConfig
	for _, n := range m.networks {
		nets = append(nets, n)
	}
	return nets
}
func (m *mockNetwork) GetPeerID() string { return m.peerID }
func (m *mockNetwork) Shutdown() error {
	m.connected = false
	return nil
}
func (m *mockNetwork) PutDHT(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kvStore[key] = append(m.kvStore[key], netpkg.DHTValue{Key: key, Value: value, TTL: int64(ttl.Seconds()), Timestamp: time.Now().Unix()})
	return nil
}
func (m *mockNetwork) GetDHT(key string) ([]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []interface{}
	for _, v := range m.kvStore[key] {
		results = append(results, v.Value)
	}
	return results, nil
}

// NewDistributedDatabase creates a new DistributedDatabase instance, optionally with a mock network for testing.
// This function is for internal testing use only.
func newTestDistributedDatabase(ctx context.Context, opts DistributedDbOptions, store stor.Storage, mockNet ...netpkg.Network) (*DistributedDatabase, error) {
	var nm netpkg.Network
	if len(mockNet) > 0 && mockNet[0] != nil {
		nm = mockNet[0]
	} else {
		nm = netpkg.NewNetworkManager(ctx)
	}

	db := &DistributedDatabase{network: nm, storage: store, distributed: opts.Distributed.Enabled, collections: make(map[string]*coll.DistributedCollection)}
	if db.distributed {
		if err := nm.Initialize(); err != nil {
			return nil, err
		}
		if opts.Distributed.NetworkID != "" {
			if len(opts.Distributed.BootstrapPeers) > 0 {
				if err := nm.JoinNetwork(opts.Distributed.NetworkID, opts.Distributed.BootstrapPeers); err != nil {
					return nil, err
				}
			} else {
				_, err := nm.CreateNetwork(typ.NetworkConfig{NetworkID: opts.Distributed.NetworkID, Name: "Network " + opts.Distributed.NetworkID})
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return db, nil
}

func TestNewDistributedDatabase(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	store := &mockStorage{}
	db, err := newTestDistributedDatabase(ctx, opts, store, newMockNetwork())
	if err != nil {
		t.Errorf("newTestDistributedDatabase failed: %v", err)
	}
	if db == nil {
		t.Error("Database is nil")
	}
}

func TestDistributedDatabaseCollection(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	store := &mockStorage{}
	db, _ := newTestDistributedDatabase(ctx, opts, store, newMockNetwork())
	coll := db.Collection("test", store)
	if coll == nil {
		t.Error("Collection is nil")
	}
	if coll.Name != "test" {
		t.Errorf("Expected name 'test', got %s", coll.Name)
	}
}

func TestDistributedDatabaseCreateNetwork(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	store := &mockStorage{}
	db, _ := newTestDistributedDatabase(ctx, opts, store, newMockNetwork())
	cfg := typ.NetworkConfig{NetworkID: "net1", Name: "Test Network"}
	id, err := db.CreateNetwork(cfg)
	if err != nil {
		t.Errorf("CreateNetwork failed: %v", err)
	}
	if id != "net1" {
		t.Errorf("Expected id 'net1', got %s", id)
	}
}

func TestDistributedDatabaseShutdown(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	store := &mockStorage{}
	db, _ := newTestDistributedDatabase(ctx, opts, store, newMockNetwork())
	err := db.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestDistributedDatabaseConsensusInitialization(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	opts.Consensus.Enabled = true
	opts.Consensus.NetworkID = "test-consensus-net"
	opts.Consensus.Mode = "standalone"
	store := &mockStorage{}
	db, err := NewDistributedDatabase(ctx, opts, store, newMockNetwork())
	if err != nil {
		t.Fatalf("NewDistributedDatabase with consensus failed: %v", err)
	}
	defer db.Shutdown()
	if db.consensus == nil {
		t.Fatal("expected consensus manager to be initialized")
	}
	if !db.consensus.Enabled() {
		t.Error("expected consensus to be enabled")
	}
}

func TestConsensusPassedToCollections(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	opts.Consensus.Enabled = true
	opts.Consensus.NetworkID = "test-collection-net"
	opts.Consensus.Mode = "standalone"
	store := &mockStorage{}
	db, err := NewDistributedDatabase(ctx, opts, store, newMockNetwork())
	if err != nil {
		t.Fatalf("NewDistributedDatabase failed: %v", err)
	}
	defer db.Shutdown()
	c := db.Collection("consensus-coll", store)
	if c == nil {
		t.Fatal("expected collection to be created")
	}
}

func TestDistributedEventHandlerRoutesToCollection(t *testing.T) {
	ctx := context.Background()
	opts := DistributedDbOptions{}
	opts.Consensus.Enabled = true
	opts.Consensus.NetworkID = "route-net"
	opts.Consensus.Mode = "standalone"
	store := &mockStorage{}
	db, err := NewDistributedDatabase(ctx, opts, store, newMockNetwork())
	if err != nil {
		t.Fatalf("NewDistributedDatabase failed: %v", err)
	}
	defer db.Shutdown()

	coll := db.Collection("route-coll", store)
	coll.AttachToNetwork("route-net")

	handler := &DistributedEventHandler{db: db}
	op := p2pconsensus.OperationEnvelope{
		Collection:  "route-coll",
		DocumentID:  "doc-route",
		Data:        mustMarshalCRDT(t, typ.CRDTOperation{ID: "r1", Type: typ.OpInsert, Collection: "route-coll", DocumentID: "doc-route", Data: &typ.DistributedDocument{ID: "doc-route", Payload: map[string]interface{}{"id": "doc-route"}}}),
		VectorClock: map[string]int64{"peerR": 1},
		Timestamp:   1,
		PeerID:      "peerR",
	}
	if err := handler.OnOperationReceived(op); err != nil {
		t.Fatalf("OnOperationReceived: %v", err)
	}
	if _, err := coll.Find(ctx, "doc-route"); err != nil {
		t.Fatalf("expected routed operation to be applied: %v", err)
	}

	// Unknown collection should error rather than panic.
	if err := handler.OnOperationReceived(p2pconsensus.OperationEnvelope{Collection: "missing"}); err == nil {
		t.Fatal("expected error for unknown collection")
	}
}

func mustMarshalCRDT(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
