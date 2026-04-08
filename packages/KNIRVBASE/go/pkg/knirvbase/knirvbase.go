package knirvbase

import (
	"context"
	"fmt"

	coll "github.com/knirvcorp/knirvbase/go/internal/collection"
	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	db "github.com/knirvcorp/knirvbase/go/internal/database"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	typ "github.com/knirvcorp/knirvbase/go/internal/types"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type Options struct {
	DataDir                   string
	DistributedEnabled        bool
	DistributedNetworkID      string
	DistributedBootstrapPeers []string
}

type DB struct {
	db    *db.DistributedDatabase
	store stor.Storage
}

func New(ctx context.Context, opts Options) (*DB, error) {
	if opts.DataDir == "" {
		return nil, fmt.Errorf("DataDir cannot be empty")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	store := stor.NewFileStorage(opts.DataDir)
	dopts := db.DistributedDbOptions{}
	dopts.Distributed.Enabled = opts.DistributedEnabled
	dopts.Distributed.NetworkID = opts.DistributedNetworkID
	dopts.Distributed.BootstrapPeers = opts.DistributedBootstrapPeers

	inner, err := db.NewDistributedDatabase(ctx, dopts, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create distributed database: %w", err)
	}
	return &DB{db: inner, store: store}, nil
}

func (d *DB) CreateNetwork(cfg typ.NetworkConfig) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	return d.db.CreateNetwork(cfg)
}

func (d *DB) JoinNetwork(networkID string, bootstrapPeers []string) error {
	return d.db.JoinNetwork(networkID, bootstrapPeers)
}

func (d *DB) LeaveNetwork(networkID string) error {
	return d.db.LeaveNetwork(networkID)
}

func (d *DB) Collection(name string) Collection {
	if d.db == nil {
		panic("database not initialized")
	}
	if name == "" {
		panic("collection name cannot be empty")
	}
	c := d.db.Collection(name, d.store)
	return &collectionAdapter{c: c}
}

func (d *DB) Raw() *db.DistributedDatabase { return d.db }

func (d *DB) RawCollection(name string) *coll.DistributedCollection {
	return d.db.Collection(name, d.store)
}

func (d *DB) Shutdown() error {
	return d.db.Shutdown()
}

type NRVDataset struct {
	name    string
	storage *stor.NRVStorage
	inner   *coll.DistributedCollection
}

func (d *DB) Dataset(name string) *NRVDataset {
	nrvStore, ok := d.store.(*stor.NRVStorage)
	if !ok {
		panic("DB was not created with NRVStorage — use NewNRV() constructor")
	}
	return &NRVDataset{
		name:    name,
		storage: nrvStore,
		inner:   d.db.Collection(name, d.store),
	}
}

func (ds *NRVDataset) AppendBracket(ctx context.Context, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
	return ds.storage.AppendBracketDirect(ds.name, b, thermo)
}

func (ds *NRVDataset) StreamBrackets(ctx context.Context, goldOnly bool) (<-chan *nrv.Bracket, error) {
	return ds.storage.StreamBrackets(ctx, ds.name, goldOnly)
}

func (ds *NRVDataset) GetFrame(ctx context.Context, frameID string) (*nrv.FrameEntry, []*nrv.Bracket, error) {
	return ds.storage.GetFrame(ctx, ds.name, frameID)
}

func (ds *NRVDataset) SetLinguistic(token, unit string) error {
	return ds.storage.SetLinguistic(ds.name, token, unit)
}

func NewNRV(ctx context.Context, opts Options, keyPair *pqc.PQCKeyPair) (*DB, error) {
	store := stor.NewNRVStorage(opts.DataDir, keyPair)
	dopts := db.DistributedDbOptions{}
	dopts.Distributed.Enabled = opts.DistributedEnabled
	dopts.Distributed.NetworkID = opts.DistributedNetworkID
	dopts.Distributed.BootstrapPeers = opts.DistributedBootstrapPeers
	inner, err := db.NewDistributedDatabase(ctx, dopts, store)
	if err != nil {
		return nil, err
	}
	return &DB{db: inner, store: store}, nil
}

type Collection interface {
	Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error)
	Update(ctx context.Context, id string, update map[string]interface{}) (int, error)
	Delete(ctx context.Context, id string) (int, error)
	Find(ctx context.Context, id string) (map[string]interface{}, error)
	FindAll(ctx context.Context) ([]map[string]interface{}, error)
	AttachToNetwork(networkID string) error
	DetachFromNetwork() error
	ForceSync() error
}

type collectionAdapter struct{ c *coll.DistributedCollection }

func (a *collectionAdapter) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}
	if id, ok := doc["id"].(string); !ok || id == "" {
		return nil, fmt.Errorf("document must contain a non-empty 'id' field")
	}
	return a.c.Insert(ctx, doc)
}
func (a *collectionAdapter) Update(ctx context.Context, id string, update map[string]interface{}) (int, error) {
	return a.c.Update(ctx, id, update)
}
func (a *collectionAdapter) Delete(ctx context.Context, id string) (int, error) {
	return a.c.Delete(ctx, id)
}
func (a *collectionAdapter) Find(ctx context.Context, id string) (map[string]interface{}, error) {
	return a.c.Find(ctx, id)
}
func (a *collectionAdapter) FindAll(ctx context.Context) ([]map[string]interface{}, error) {
	return a.c.FindAll(ctx)
}
func (a *collectionAdapter) AttachToNetwork(networkID string) error {
	return a.c.AttachToNetwork(networkID)
}
func (a *collectionAdapter) DetachFromNetwork() error { return a.c.DetachFromNetwork() }
func (a *collectionAdapter) ForceSync() error         { return a.c.ForceSync() }
