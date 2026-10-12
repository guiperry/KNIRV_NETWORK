package knirvbase

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	coll "github.com/knirvcorp/knirvbase/internal/collection"
	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	db "github.com/knirvcorp/knirvbase/internal/database"
	netpkg "github.com/knirvcorp/knirvbase/internal/network"
	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	stor "github.com/knirvcorp/knirvbase/internal/storage"
	typ "github.com/knirvcorp/knirvbase/internal/types"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// Options configures the KNIRVBASE database.
type Options struct {
	DataDir                   string
	DistributedEnabled        bool     // Deprecated: use Consensus.Enabled
	DistributedNetworkID      string   // Deprecated: use Consensus.NetworkID
	DistributedBootstrapPeers []string // Deprecated: use Consensus.BootstrapPeers
	Consensus                 p2pconsensus.ConsensusConfig
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
	dopts.Distributed.Enabled = opts.Consensus.Enabled || opts.DistributedEnabled
	dopts.Distributed.NetworkID = opts.Consensus.NetworkID
	if opts.Consensus.NetworkID == "" {
		dopts.Distributed.NetworkID = opts.DistributedNetworkID
	}
	dopts.Distributed.BootstrapPeers = opts.Consensus.BootstrapPeers
	if len(opts.Consensus.BootstrapPeers) == 0 {
		dopts.Distributed.BootstrapPeers = opts.DistributedBootstrapPeers
	}
	dopts.Consensus = opts.Consensus

	inner, err := db.NewDistributedDatabase(ctx, dopts, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create distributed database: %w", err)
	}
	return &DB{db: inner, store: store}, nil
}

func DefaultOptions() Options {
	return Options{
		DataDir:              "./data",
		DistributedEnabled:   false,
		DistributedNetworkID: "knirvbase-default",
		Consensus:            p2pconsensus.DefaultConsensusConfig(),
	}
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

// NetworkStats returns the current distributed-network transport counters.
// A nil result means the requested network has not been joined/created.
func (d *DB) NetworkStats(networkID string) *typ.NetworkStats {
	if d.db == nil {
		return nil
	}
	return d.db.NetworkStats(networkID)
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

func (d *DB) Shutdown() error {
	return d.db.Shutdown()
}

// Put stores a raw value by key (Key-Value API).
func (d *DB) Put(ctx context.Context, key string, value []byte) error {
	return d.db.Put(ctx, key, value)
}

// Get retrieves a raw value by key.
func (d *DB) Get(ctx context.Context, key string) ([]byte, error) {
	return d.db.Get(ctx, key)
}

// DeleteKey removes a key from the Key-Value store.
func (d *DB) DeleteKey(ctx context.Context, key string) error {
	return d.db.DeleteKey(ctx, key)
}

// StoreObject stores a JSON-serialisable object by key.
func (d *DB) StoreObject(ctx context.Context, key string, obj interface{}) error {
	return d.db.StoreObject(ctx, key, obj)
}

// GetObject retrieves a JSON object by key into dest.
func (d *DB) GetObject(ctx context.Context, key string, dest interface{}) error {
	return d.db.GetObject(ctx, key, dest)
}

// ProjectToMarkdown projects a KV entry into a Markdown file.
func (d *DB) ProjectToMarkdown(ctx context.Context, key string, targetPath string) error {
	return d.db.ProjectToMarkdown(ctx, key, targetPath)
}

// PutDHT stores a value in the distributed hash table.
func (d *DB) PutDHT(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return d.db.PutDHT(ctx, key, value, ttl)
}

// GetDHT retrieves values from the distributed hash table.
func (d *DB) GetDHT(ctx context.Context, key string) ([]interface{}, error) {
	return d.db.GetDHT(ctx, key)
}

// CreateIndex creates an index on a collection.
func (d *DB) CreateIndex(ctx context.Context, collection, name string, indexType stor.IndexType, fields []string, unique bool, partialExpr string, options map[string]interface{}) error {
	return d.db.CreateIndex(ctx, collection, name, indexType, fields, unique, partialExpr, options)
}

// DropIndex removes an index.
func (d *DB) DropIndex(ctx context.Context, collection, name string) error {
	return d.db.DropIndex(ctx, collection, name)
}

// GetIndex retrieves an index.
func (d *DB) GetIndex(ctx context.Context, collection, name string) *stor.Index {
	return d.db.GetIndex(ctx, collection, name)
}

// GetIndexesForCollection lists all indexes for a collection.
func (d *DB) GetIndexesForCollection(ctx context.Context, collection string) []*stor.Index {
	return d.db.GetIndexesForCollection(ctx, collection)
}

// QueryIndex queries an index.
func (d *DB) QueryIndex(ctx context.Context, collection, indexName string, query map[string]interface{}) ([]string, error) {
	return d.db.QueryIndex(ctx, collection, indexName, query)
}

// SetMasterKey sets the master PQC key for encryption at rest.
func (d *DB) SetMasterKey(keyPair *pqc.PQCKeyPair) {
	d.db.SetMasterKey(keyPair)
}

// GetNetworkManager returns the underlying network manager.
func (d *DB) GetNetworkManager() netpkg.Network {
	return d.db.GetNetworkManager()
}

// GetConsensusManager returns the consensus manager, or nil if not enabled.
func (d *DB) GetConsensusManager() *p2pconsensus.P2PConsensusManager {
	return d.db.GetConsensusManager()
}

// AddCollectionToNetwork attaches a collection to a network.
func (d *DB) AddCollectionToNetwork(networkID, collectionName string) error {
	return d.db.AddCollectionToNetwork(networkID, collectionName)
}

// RemoveCollectionFromNetwork detaches a collection from a network.
func (d *DB) RemoveCollectionFromNetwork(collectionName string) error {
	return d.db.RemoveCollectionFromNetwork(collectionName)
}

// KeyPair is a post-quantum key pair (ML-KEM-768 + ML-DSA-65). Pass one to
// NewNRV to sign every frame the dataset store writes.
type KeyPair = pqc.PQCKeyPair

// KeyInfo is the public description of a signing key: enough for a client to
// verify frame signatures offline.
type KeyInfo struct {
	ID        string `json:"key_id"`
	Algorithm string `json:"algorithm"`  // signature algorithm, e.g. "ML-DSA-65"
	PublicKey string `json:"public_key"` // base64 ML-DSA-65 public key
}

// DescribeKey returns the public parts of kp.
func DescribeKey(kp *KeyPair) KeyInfo {
	return KeyInfo{
		ID:        kp.ID,
		Algorithm: pqc.AlgorithmSig,
		PublicKey: base64.StdEncoding.EncodeToString(kp.DilithiumPublicKeyBytes),
	}
}

// LoadOrCreateKeyPair loads the key pair stored at path, or generates and
// stores a new one (file 0600, directory 0700) if the file does not exist. An
// existing file in an unsupported format is an error, never silently replaced.
// Keep the file outside the data directory it protects.
func LoadOrCreateKeyPair(path, name string) (*KeyPair, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		kp, lerr := pqc.LoadPQCKeyPair(raw)
		if lerr != nil {
			return nil, fmt.Errorf("key file %s: %w", path, lerr)
		}
		return kp, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	kp, err := pqc.GeneratePQCKeyPair(name, "signature")
	if err != nil {
		return nil, err
	}
	data, err := kp.MarshalWithPrivateKeys()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// O_EXCL: if two processes race, one wins and the other re-reads the winner.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return LoadOrCreateKeyPair(path, name)
		}
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return kp, f.Close()
}

// FrameSignature returns the base64 signature recorded for a frame ("" if the
// store was opened without a key).
func (ds *NRVDataset) FrameSignature(frameID string) (string, error) {
	return ds.storage.FrameSignature(ds.name, frameID)
}

// VerifyFrame checks a frame's signature against kp's public key.
func (ds *NRVDataset) VerifyFrame(frameID string, kp *KeyPair) (bool, error) {
	return ds.storage.VerifyFrame(ds.name, frameID, kp)
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

// Flush writes the dataset's pending brackets as a frame immediately.
func (ds *NRVDataset) Flush() error {
	return ds.storage.Flush(ds.name)
}

// Frames lists the dataset's live frames (metadata only).
func (ds *NRVDataset) Frames() ([]nrv.FrameEntry, error) {
	return ds.storage.Frames(ds.name)
}

func (ds *NRVDataset) SetLinguistic(token, unit string) error {
	return ds.storage.SetLinguistic(ds.name, token, unit)
}

func NewNRV(ctx context.Context, opts Options, keyPair *pqc.PQCKeyPair) (*DB, error) {
	store := stor.NewNRVStorage(opts.DataDir, keyPair)
	dopts := db.DistributedDbOptions{}
	dopts.Distributed.Enabled = opts.Consensus.Enabled || opts.DistributedEnabled
	dopts.Distributed.NetworkID = opts.Consensus.NetworkID
	if dopts.Distributed.NetworkID == "" {
		dopts.Distributed.NetworkID = opts.DistributedNetworkID
	}
	dopts.Distributed.BootstrapPeers = opts.Consensus.BootstrapPeers
	if len(dopts.Distributed.BootstrapPeers) == 0 {
		dopts.Distributed.BootstrapPeers = opts.DistributedBootstrapPeers
	}
	dopts.Consensus = opts.Consensus
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
