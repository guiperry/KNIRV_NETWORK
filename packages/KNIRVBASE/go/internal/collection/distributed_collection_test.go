package collection

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/network"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	typ "github.com/knirvcorp/knirvbase/go/internal/types"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	nrv "github.com/knirvcorp/knirvbase/go/pkg/nrv"
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
func (m *mockStorage) Put(ctx context.Context, key string, value []byte) error            { return nil }
func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, error)                { return nil, nil }
func (m *mockStorage) DeleteKey(ctx context.Context, key string) error                    { return nil }
func (m *mockStorage) StoreObject(ctx context.Context, key string, obj interface{}) error { return nil }
func (m *mockStorage) GetObject(ctx context.Context, key string, dest interface{}) error  { return nil }
func (m *mockStorage) ProjectToMarkdown(ctx context.Context, key string, targetPath string) error {
	return nil
}
func (m *mockStorage) Close() error { return nil }

type mockNetwork struct{}

func (m *mockNetwork) Initialize() error                                                  { return nil }
func (m *mockNetwork) CreateNetwork(cfg typ.NetworkConfig) (string, error)                { return "net1", nil }
func (m *mockNetwork) JoinNetwork(networkID string, bootstrapPeers []string) error        { return nil }
func (m *mockNetwork) LeaveNetwork(networkID string) error                                { return nil }
func (m *mockNetwork) AddCollectionToNetwork(networkID, collectionName string) error      { return nil }
func (m *mockNetwork) RemoveCollectionFromNetwork(networkID, collectionName string) error { return nil }
func (m *mockNetwork) GetNetworkCollections(networkID string) []string                    { return nil }
func (m *mockNetwork) BroadcastMessage(networkID string, msg typ.ProtocolMessage) error   { return nil }
func (m *mockNetwork) SendToPeer(peerID, networkID string, msg typ.ProtocolMessage) error { return nil }
func (m *mockNetwork) OnMessage(mt typ.MessageType, handler network.MessageHandler)       {}
func (m *mockNetwork) GetNetworkStats(networkID string) *typ.NetworkStats                 { return nil }
func (m *mockNetwork) GetNetworks() []*typ.NetworkConfig                                  { return nil }
func (m *mockNetwork) GetPeerID() string                                                  { return "peer1" }
func (m *mockNetwork) Shutdown() error                                                    { return nil }

func TestNewLocalCollection(t *testing.T) {
	store := &mockStorage{}
	coll := NewLocalCollection("test", store)
	if coll.name != "test" {
		t.Errorf("Expected name 'test', got %s", coll.name)
	}
}

func TestLocalCollectionInsert(t *testing.T) {
	store := &mockStorage{}
	coll := NewLocalCollection("test", store)
	doc := map[string]interface{}{"id": "1", "data": "test"}
	result, err := coll.Insert(context.Background(), doc)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}
	if result["id"] != "1" {
		t.Errorf("Expected id '1', got %v", result["id"])
	}
}

func TestLocalCollectionFind(t *testing.T) {
	store := &mockStorage{}
	coll := NewLocalCollection("test", store)
	_, err := coll.Find(context.Background(), "1")
	if err != nil {
		t.Errorf("Find failed: %v", err)
	}
}

func TestNewDistributedCollection(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("test", net, store)
	if coll.Name != "test" {
		t.Errorf("Expected name 'test', got %s", coll.Name)
	}
}

func TestDistributedCollectionInsert(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("test", net, store)
	doc := map[string]interface{}{"id": "1", "entryType": typ.EntryTypeMemory, "payload": map[string]interface{}{}}
	result, err := coll.Insert(context.Background(), doc)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}
	if result["id"] != "1" {
		t.Errorf("Expected id '1', got %v", result["id"])
	}
}

func TestDistributedCollectionFind(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("test", net, store)
	_, err := coll.Find(context.Background(), "1")
	if err != nil {
		t.Errorf("Find failed: %v", err)
	}
}

func TestCloneMap(t *testing.T) {
	original := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{"c": 2},
		"d": []interface{}{3, 4},
	}
	cloned := cloneMap(original)
	if cloned["a"] != 1 {
		t.Errorf("Primitive not cloned correctly")
	}
	if cloned["b"].(map[string]interface{})["c"] != 2 {
		t.Errorf("Nested map not cloned correctly")
	}
	if len(cloned["d"].([]interface{})) != 2 {
		t.Errorf("Slice not cloned correctly")
	}
	// Modify original to ensure clone is independent
	original["a"] = 999
	if cloned["a"] == 999 {
		t.Errorf("Clone is not independent")
	}
}

func TestCloneSlice(t *testing.T) {
	original := []interface{}{1, map[string]interface{}{"a": 2}, []interface{}{3}}
	cloned := cloneSlice(original)
	if cloned[0] != 1 {
		t.Errorf("Primitive not cloned correctly")
	}
	if cloned[1].(map[string]interface{})["a"] != 2 {
		t.Errorf("Nested map not cloned correctly")
	}
	if len(cloned[2].([]interface{})) != 1 {
		t.Errorf("Nested slice not cloned correctly")
	}
}

func TestDistributedCollectionStreamFramesWithNRVStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "streamframes_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create NRVStorage
	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}
	nrvStorage := stor.NewNRVStorage(tmpDir, keyPair)

	// Create network mock
	net := &mockNetwork{}

	// Create distributed collection with NRVStorage
	coll := NewDistributedCollection("test", net, nrvStorage)

	// Insert a test frame using the underlying storage directly
	ctx := context.Background()
	doc := map[string]interface{}{
		"id": "stream-test-frame",
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			"thermo": map[string]float64{
				"temp_celsius": 50.5,
				"voltage_v":    12.0,
				"freq_mhz":     500,
				"fan_rpm":      3000,
			},
			"proof": []byte("test proof"),
		},
		"verified":  true,
		"ergo_rank": 0.85,
	}

	if err := nrvStorage.Insert(ctx, "test", doc); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Test StreamFrames method
	frameChan, err := coll.StreamFrames(ctx, nrv.ModalityVector)
	if err != nil {
		t.Fatalf("StreamFrames failed: %v", err)
	}

	count := 0
	for range frameChan {
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 frame from stream, got %d", count)
	}
}

func TestDistributedCollectionStreamFramesWithNonNRVStorage(t *testing.T) {
	// Create regular file storage (not NRVStorage)
	tmpDir, err := os.MkdirTemp("", "streamframes_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fileStorage := stor.NewFileStorage(tmpDir)

	// Create network mock
	net := &mockNetwork{}

	// Create distributed collection with regular file storage
	coll := NewDistributedCollection("test", net, fileStorage)

	// Test StreamFrames method - should return error
	_, err = coll.StreamFrames(context.Background(), nrv.ModalityVector)
	if err == nil {
		t.Error("expected error when storage is not NRVStorage")
	}

	if err == nil || !strings.Contains(err.Error(), "storage backend does not support NRV streaming") {
		t.Errorf("expected error about NRV streaming not supported, got: %v", err)
	}
}
