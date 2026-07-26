package distributed

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	coll "github.com/knirvcorp/knirvbase/internal/collection"
	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	netpkg "github.com/knirvcorp/knirvbase/internal/network"
	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	stor "github.com/knirvcorp/knirvbase/internal/storage"
	typ "github.com/knirvcorp/knirvbase/internal/types"
)

type DistributedDbOptions struct {
	Distributed struct {
		Enabled        bool
		NetworkID      string
		BootstrapPeers []string
	}
	Consensus p2pconsensus.ConsensusConfig
}

type DistributedDatabase struct {
	network     netpkg.Network
	consensus   *p2pconsensus.P2PConsensusManager
	storage     stor.Storage
	distributed bool
	collections map[string]*coll.DistributedCollection
	mu          sync.Mutex
}

func NewDistributedDatabase(ctx context.Context, opts DistributedDbOptions, store stor.Storage, mockNet ...netpkg.Network) (*DistributedDatabase, error) {
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
	if opts.Consensus.Enabled {
		// Wire a real EventHandler backed by the DistributedCollections created
		// through this database. Operations and sync requests arriving via the
		// consensus layer are routed to the right collection by name.
		handler := &DistributedEventHandler{db: db}
		socketPath := opts.Consensus.SocketPath
		cm := p2pconsensus.NewP2PConsensusManager(opts.Consensus, socketPath, handler)
		if err := cm.Start(ctx); err != nil {
			return nil, err
		}
		db.consensus = cm
	}
	return db, nil
}

func (db *DistributedDatabase) Collection(name string, store stor.Storage) *coll.DistributedCollection {
	db.mu.Lock()
	defer db.mu.Unlock()

	if c, ok := db.collections[name]; ok {
		return c
	}
	c := coll.NewDistributedCollection(name, db.network, store)
	if db.consensus != nil {
		c.SetConsensusManager(db.consensus)
	}
	db.collections[name] = c
	return c
}

func (db *DistributedDatabase) CreateNetwork(cfg typ.NetworkConfig) (string, error) {
	if db.network == nil {
		return "", errors.New("network manager not initialized")
	}
	return db.network.CreateNetwork(cfg)
}

func (db *DistributedDatabase) JoinNetwork(networkID string, bootstrapPeers []string) error {
	if db.network == nil {
		return errors.New("network manager not initialized")
	}
	return db.network.JoinNetwork(networkID, bootstrapPeers)
}

func (db *DistributedDatabase) LeaveNetwork(networkID string) error {
	if db.network == nil {
		return errors.New("network manager not initialized")
	}
	return db.network.LeaveNetwork(networkID)
}

func (db *DistributedDatabase) AddCollectionToNetwork(networkID string, collectionName string) error {
	c := db.collections[collectionName]
	if c == nil {
		return errors.New("collection not found")
	}
	return c.AttachToNetwork(networkID)
}

func (db *DistributedDatabase) RemoveCollectionFromNetwork(collectionName string) error {
	c := db.collections[collectionName]
	if c == nil {
		return nil
	}
	return c.DetachFromNetwork()
}

func (db *DistributedDatabase) GetNetworkManager() netpkg.Network { return db.network }

func (db *DistributedDatabase) GetConsensusManager() *p2pconsensus.P2PConsensusManager {
	return db.consensus
}

// DistributedEventHandler implements p2pconsensus.EventHandler for a whole
// DistributedDatabase. Inbound operations and sync requests are dispatched to
// the matching DistributedCollection (by name); a discovered peer triggers a
// sync across all attached collections.
type DistributedEventHandler struct {
	db *DistributedDatabase
}

func (h *DistributedEventHandler) collection(name string) *coll.DistributedCollection {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()
	return h.db.collections[name]
}

func (h *DistributedEventHandler) OnOperationReceived(op p2pconsensus.OperationEnvelope) error {
	c := h.collection(op.Collection)
	if c == nil {
		return fmt.Errorf("no collection %q for inbound operation", op.Collection)
	}
	return coll.NewCollectionEventHandler(c).OnOperationReceived(op)
}

func (h *DistributedEventHandler) OnSyncRequestReceived(req p2pconsensus.SyncRequest) (*p2pconsensus.SyncResponse, error) {
	c := h.collection(req.Collection)
	if c == nil {
		return nil, fmt.Errorf("no collection %q for inbound sync request", req.Collection)
	}
	return coll.NewCollectionEventHandler(c).OnSyncRequestReceived(req)
}

func (h *DistributedEventHandler) OnPeerDiscovered(peer p2pconsensus.PeerInfo) error {
	h.db.mu.Lock()
	cols := make([]*coll.DistributedCollection, 0, len(h.db.collections))
	for _, c := range h.db.collections {
		cols = append(cols, c)
	}
	h.db.mu.Unlock()
	var firstErr error
	for _, c := range cols {
		if err := coll.NewCollectionEventHandler(c).OnPeerDiscovered(peer); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SetMasterKey sets the master PQC key for encryption at rest
func (db *DistributedDatabase) SetMasterKey(keyPair *pqc.PQCKeyPair) {
	if fs, ok := db.storage.(interface{ SetMasterKey(*pqc.PQCKeyPair) }); ok {
		fs.SetMasterKey(keyPair)
	}
}

// Key-Value and JSON operations
func (db *DistributedDatabase) Put(ctx context.Context, key string, value []byte) error {
	return db.storage.Put(ctx, key, value)
}

func (db *DistributedDatabase) Get(ctx context.Context, key string) ([]byte, error) {
	return db.storage.Get(ctx, key)
}

func (db *DistributedDatabase) DeleteKey(ctx context.Context, key string) error {
	return db.storage.DeleteKey(ctx, key)
}

func (db *DistributedDatabase) StoreObject(ctx context.Context, key string, obj interface{}) error {
	return db.storage.StoreObject(ctx, key, obj)
}

func (db *DistributedDatabase) GetObject(ctx context.Context, key string, dest interface{}) error {
	return db.storage.GetObject(ctx, key, dest)
}

func (db *DistributedDatabase) ProjectToMarkdown(ctx context.Context, key string, targetPath string) error {
	return db.storage.ProjectToMarkdown(ctx, key, targetPath)
}

// Global DHT operations
func (db *DistributedDatabase) PutDHT(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if nm, ok := db.network.(*netpkg.NetworkManager); ok {
		return nm.PutDHT(key, value, ttl)
	}
	return errors.New("DHT not supported by current network manager")
}

func (db *DistributedDatabase) GetDHT(ctx context.Context, key string) ([]interface{}, error) {
	if nm, ok := db.network.(*netpkg.NetworkManager); ok {
		return nm.GetDHT(key)
	}
	return nil, errors.New("DHT not supported by current network manager")
}

// Index management methods
func (db *DistributedDatabase) CreateIndex(ctx context.Context, collection, name string, indexType stor.IndexType, fields []string, unique bool, partialExpr string, options map[string]interface{}) error {
	return db.storage.CreateIndex(ctx, collection, name, indexType, fields, unique, partialExpr, options)
}

func (db *DistributedDatabase) DropIndex(ctx context.Context, collection, name string) error {
	return db.storage.DropIndex(ctx, collection, name)
}

func (db *DistributedDatabase) GetIndex(ctx context.Context, collection, name string) *stor.Index {
	return db.storage.GetIndex(ctx, collection, name)
}

func (db *DistributedDatabase) GetIndexesForCollection(ctx context.Context, collection string) []*stor.Index {
	return db.storage.GetIndexesForCollection(ctx, collection)
}

func (db *DistributedDatabase) QueryIndex(ctx context.Context, collection, indexName string, query map[string]interface{}) ([]string, error) {
	return db.storage.QueryIndex(ctx, collection, indexName, query)
}

func (db *DistributedDatabase) Shutdown() error {
	if db.consensus != nil {
		_ = db.consensus.Stop()
	}
	if db.network == nil {
		return nil
	}
	return db.network.Shutdown()
}
