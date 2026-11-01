package oracle

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// ReplicationSource is the active side: it produces signed heads
// when state advances and serves the ranges each head commits to.
type ReplicationSource struct {
	oracle *Oracle
	logger *zap.Logger
	signer knirvsigning.RemoteSigner
	path   string

	mu       sync.Mutex
	heads    []replication.SignedHead
	epoch    uint64
	nonces   map[string]int64
	lastPull map[string]int64
	interval time.Duration
}

// Replication returns the source (nil on a node that never produced one).
func (o *Oracle) Replication() *ReplicationSource { return o.replication }

// initReplicationSource builds the signer and loads the persisted head chain.
// It runs for active and standby nodes: a standby re-serves the root-signed
// head it verified, so it needs the same storage.
func (o *Oracle) initReplicationSource() error {
	source := &ReplicationSource{
		oracle:   o,
		logger:   o.logger,
		path:     filepath.Join(o.config.DataDir, "replication_heads.json"),
		epoch:    1,
		nonces:   map[string]int64{},
		lastPull: map[string]int64{},
		interval: 30 * time.Second,
	}

	if o.config.OwnerPrivateKey != "" {
		key, err := hex.DecodeString(strip0x(o.config.OwnerPrivateKey))
		if err != nil {
			return fmt.Errorf("replication signer: %w", err)
		}
		signer, err := knirvsigning.NewLocalSigner(key)
		if err != nil {
			return fmt.Errorf("replication signer: %w", err)
		}
		source.signer = signer
	}

	if err := source.load(); err != nil {
		return err
	}
	o.replication = source
	return nil
}

func (s *ReplicationSource) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("replication: read heads: %w", err)
	}
	var heads []replication.SignedHead
	if err := json.Unmarshal(data, &heads); err != nil {
		return fmt.Errorf("replication: decode heads: %w", err)
	}
	s.heads = heads
	if len(heads) > 0 {
		s.epoch = heads[len(heads)-1].Head.Epoch
	}
	return nil
}

func (s *ReplicationSource) persistLocked() error {
	// Bounded index: the chain is recoverable from the Oracle's own history,
	// the file is an index, not the authority.
	const keep = 1024
	if len(s.heads) > keep {
		s.heads = s.heads[len(s.heads)-keep:]
	}
	body, err := json.MarshalIndent(s.heads, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// ProduceHead appends and signs one head for the current state. Callers are the
// checkpoint commit path and the periodic ticker; it is safe to call twice for
// an unchanged state (the second head names the first and carries no new data).
func (s *ReplicationSource) ProduceHead() (replication.SignedHead, error) {
	if s.signer == nil {
		return replication.SignedHead{}, fmt.Errorf("replication: no signing identity")
	}

	mmrSize, mmrRoot, height := s.oracle.replicationCheckpointView()
	head := replication.Head{
		Version:         replication.Version,
		NetworkID:       s.oracle.config.NetworkID,
		OracleChainID:   s.oracle.config.ChainID,
		OwnerAddress:    s.signer.Address(),
		Epoch:           s.epoch,
		MMRSize:         mmrSize,
		MMRRoot:         mmrRoot,
		RegistryHash:    s.oracle.registryHash(),
		StateLogSize:    uint64(s.oracle.ledger.Len()),
		StateRoot:       s.oracle.stateRoot,
		AuxHash:         s.oracle.auxHash(),
		ConsensusHeight: height,
		ProducedAt:      time.Now().Unix(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	head.Seq = uint64(len(s.heads)) + 1
	if len(s.heads) > 0 {
		prevHash, err := s.heads[len(s.heads)-1].Head.Hash()
		if err != nil {
			return replication.SignedHead{}, err
		}
		head.PrevHeadHash = prevHash
	}
	signed, err := replication.SignHead(s.signer, head)
	if err != nil {
		return replication.SignedHead{}, err
	}
	s.heads = append(s.heads, signed)
	if err := s.persistLocked(); err != nil {
		s.heads = s.heads[:len(s.heads)-1]
		return replication.SignedHead{}, err
	}
	s.logger.Info("Replication head produced",
		zap.Uint64("seq", head.Seq), zap.Uint64("mmr_size", head.MMRSize),
		zap.Uint64("state_log_size", head.StateLogSize), zap.String("state_root", head.StateRoot))
	return signed, nil
}

// ProduceHeadIfChanged produces a head only when the committed state differs
// from the last head (checkpoint commits and the periodic ticker both use it).
func (s *ReplicationSource) ProduceHeadIfChanged() (replication.SignedHead, bool, error) {
	mmrSize, mmrRoot, height := s.oracle.replicationCheckpointView()
	if head, ok := s.CurrentHead(); ok {
		if head.Head.MMRSize == mmrSize && head.Head.MMRRoot == mmrRoot &&
			head.Head.StateLogSize == uint64(s.oracle.ledger.Len()) &&
			head.Head.StateRoot == s.oracle.stateRoot && head.Head.ConsensusHeight == height {
			return head, false, nil
		}
	}
	head, err := s.ProduceHead()
	return head, true, err
}

// CurrentHead returns the newest signed head.
func (s *ReplicationSource) CurrentHead() (replication.SignedHead, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.heads) == 0 {
		return replication.SignedHead{}, false
	}
	return s.heads[len(s.heads)-1], true
}

// HeadsSince returns heads with Seq > after, oldest first.
func (s *ReplicationSource) HeadsSince(after uint64, limit int) []replication.SignedHead {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > 512 {
		limit = 512
	}
	out := make([]replication.SignedHead, 0, limit)
	for _, h := range s.heads {
		if h.Head.Seq > after && len(out) < limit {
			out = append(out, h)
		}
	}
	return out
}

// Leaves returns hex-encoded audit leaves [from, from+limit).
func (s *ReplicationSource) Leaves(from uint64, limit int) []string {
	s.oracle.checkpoint.mu.RLock()
	defer s.oracle.checkpoint.mu.RUnlock()
	if limit <= 0 || limit > 1024 {
		limit = 1024
	}
	if from >= uint64(len(s.oracle.checkpoint.leafLog)) {
		return nil
	}
	end := from + uint64(limit)
	if end > uint64(len(s.oracle.checkpoint.leafLog)) {
		end = uint64(len(s.oracle.checkpoint.leafLog))
	}
	out := make([]string, 0, end-from)
	for _, leaf := range s.oracle.checkpoint.leafLog[from:end] {
		out = append(out, hex.EncodeToString(leaf[:]))
	}
	return out
}

// StateRecords returns ledger records [from, from+limit) with the leaf they
// commit to.
func (s *ReplicationSource) StateRecords(from uint64, limit int) []map[string]any {
	records := s.oracle.ledger.RecordsRange(from, limit)
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		leaf := mmr.LeafHash([]byte("state:" + record.Hash))
		out = append(out, map[string]any{
			"seq":       record.Seq,
			"kind":      record.Kind,
			"payload":   json.RawMessage(record.Payload),
			"prev_hash": record.PrevHash,
			"hash":      record.Hash,
			"leaf":      hex.EncodeToString(leaf[:]),
		})
	}
	return out
}

// AuthorizePull validates the X-Knirv-Replica-Auth header: a base64-encoded
// signed replication-pull envelope. Nonce replay and a per-signer rate limit
// are enforced here; the signature and validator-set checks are the SDK's.
func (s *ReplicationSource) AuthorizePull(header string, now time.Time) error {
	if header == "" {
		return fmt.Errorf("%w: missing X-Knirv-Replica-Auth", replication.ErrPullAuth)
	}
	raw, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return fmt.Errorf("%w: header is not base64", replication.ErrPullAuth)
	}
	var signed knirvsigning.SignedMessage
	if err := json.Unmarshal(raw, &signed); err != nil {
		return fmt.Errorf("%w: header is not a signed message", replication.ErrPullAuth)
	}

	allowed := map[string]bool{}
	for _, validator := range s.oracle.governanceSystem.ListValidators() {
		allowed[validator.Address.String()] = true
	}
	request, err := replication.VerifyPull(signed, allowed, now, replication.MaxClockSkew)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, seen := s.nonces[request.Nonce]; seen {
		return fmt.Errorf("%w: nonce already used", replication.ErrPullAuth)
	}
	if last, ok := s.lastPull[signed.Address]; ok && now.Unix()-last < 1 {
		return fmt.Errorf("%w: rate limited", replication.ErrPullAuth)
	}
	s.nonces[request.Nonce] = now.Unix()
	s.lastPull[signed.Address] = now.Unix()
	// Keep the replay set bounded: drop entries older than the skew window.
	for nonce, at := range s.nonces {
		if now.Unix()-at > int64(replication.MaxClockSkew/time.Second)*2 {
			delete(s.nonces, nonce)
		}
	}
	return nil
}

// replicationCheckpointView snapshots the audit MMR and highest checkpoint end
// height under one lock.
func (o *Oracle) replicationCheckpointView() (uint64, string, uint64) {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	size := uint64(o.checkpoint.mmr.Size())
	bag := o.checkpoint.mmr.BagRoot()
	root := hex.EncodeToString(bag[:])
	var height uint64
	for _, record := range o.checkpoint.records {
		if record.Checkpoint.EndHeight > height {
			height = record.Checkpoint.EndHeight
		}
	}
	return size, root, height
}

// registryHash hashes the chain registry deterministically.
func (o *Oracle) registryHash() string {
	chains := o.checkpoint.registry.List()
	sort.Slice(chains, func(i, j int) bool { return chains[i].ChainID < chains[j].ChainID })
	body, err := json.Marshal(chains)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// auxHash hashes the skill-ownership registry and rollups (R6).
func (o *Oracle) auxHash() string {
	o.skillOwnershipMu.RLock()
	skills := make(map[string]any, len(o.skillOwnership))
	for id, record := range o.skillOwnership {
		skills[id] = record
	}
	o.skillOwnershipMu.RUnlock()

	o.rollupsMu.RLock()
	rollups := make(map[string]any, len(o.rollups))
	for id, record := range o.rollups {
		rollups[id] = record
	}
	o.rollupsMu.RUnlock()

	body, err := json.Marshal(map[string]any{"skill_ownership": skills, "rollups": rollups})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// AuxSnapshot returns the R6 data and its hash.
func (s *ReplicationSource) AuxSnapshot() map[string]any {
	o := s.oracle
	o.skillOwnershipMu.RLock()
	skills := make(map[string]any, len(o.skillOwnership))
	for id, record := range o.skillOwnership {
		skills[id] = record
	}
	o.skillOwnershipMu.RUnlock()

	o.rollupsMu.RLock()
	rollups := make(map[string]any, len(o.rollups))
	for id, record := range o.rollups {
		rollups[id] = record
	}
	o.rollupsMu.RUnlock()

	return map[string]any{
		"skill_ownership": skills,
		"rollups":         rollups,
		"hash":            o.auxHash(),
	}
}

// RegistrySnapshot returns the chain registrations (R3) in a stable order.
func (o *Oracle) RegistrySnapshot() []*types.ChainRegistration {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	chains := o.checkpoint.registry.List()
	out := make([]*types.ChainRegistration, len(chains))
	copy(out, chains)
	return out
}

// ReplicationRegistryHash exposes the head's registry hash to the endpoint.
func (o *Oracle) ReplicationRegistryHash() string { return o.registryHash() }
