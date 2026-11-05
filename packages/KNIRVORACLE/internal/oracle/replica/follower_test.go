package replica

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

type fakeSource struct {
	head    replication.SignedHead
	leaves  []string
	records []StateRecord
}

func (s fakeSource) Head(context.Context) (replication.SignedHead, error) { return s.head, nil }
func (s fakeSource) HeadsAfter(_ context.Context, after uint64) ([]replication.SignedHead, error) {
	if s.head.Head.Seq > after {
		return []replication.SignedHead{s.head}, nil
	}
	return nil, nil
}
func (s fakeSource) Leaves(_ context.Context, from uint64, limit int) ([]string, error) {
	end := int(from) + limit
	if end > len(s.leaves) {
		end = len(s.leaves)
	}
	if int(from) >= len(s.leaves) {
		return nil, nil
	}
	return append([]string(nil), s.leaves[from:end]...), nil
}
func (s fakeSource) State(_ context.Context, from uint64, limit int) ([]StateRecord, error) {
	end := int(from) + limit
	if end > len(s.records) {
		end = len(s.records)
	}
	if int(from) >= len(s.records) {
		return nil, nil
	}
	return append([]StateRecord(nil), s.records[from:end]...), nil
}
func (s fakeSource) Registry(context.Context) (RegistrySnapshot, error) {
	return RegistrySnapshot{Hash: registryHash(nil)}, nil
}
func (s fakeSource) Aux(context.Context) (AuxSnapshot, error) {
	aux := AuxSnapshot{}
	aux.Hash = auxHash(aux)
	return aux, nil
}
func (s fakeSource) Checkpoints(context.Context, string, uint64, int) ([]*types.CheckpointRecord, error) {
	return nil, nil
}

func TestFollowerMirrorsVerifiedHeadAndRetainsItAcrossRestart(t *testing.T) {
	source, anchor := signedSource(t)
	path := filepath.Join(t.TempDir(), "standby", "replica_state.json")
	follower, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(follower)
	if err := follower.Sync(context.Background(), source); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := follower.Status(); got.HeadSeq != 1 || got.LagSeq != 0 || got.Diverged {
		t.Fatalf("status = %+v", got)
	}
	if got := follower.Snapshot(); len(got.Records) != 2 || len(got.Leaves) != 1 {
		t.Fatalf("snapshot = %+v", got)
	}
	source.head.Head.Seq = 2
	source.head.Head.PrevHeadHash = "head-1"
	if err := follower.Sync(context.Background(), source); err != nil {
		t.Fatalf("sync successor: %v", err)
	}
	reopened, err := Open(path, anchor)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.Status().HeadSeq; got != 2 {
		t.Fatalf("persisted sequence = %d, want 2", got)
	}
	if heads := reopened.Snapshot().Heads; len(heads) != 2 || heads[0].Head.Seq != 1 || heads[1].Head.Seq != 2 {
		t.Fatalf("persisted verified head history = %+v", heads)
	}
}

func TestFollowerDoesNotOverwriteOnFork(t *testing.T) {
	source, anchor := signedSource(t)
	follower, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(follower)
	if err := follower.Sync(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	before := follower.Snapshot()
	forkLeaf := mmr.LeafHash([]byte("fork"))
	source.leaves[0] = hex.EncodeToString(forkLeaf[:])
	if err := follower.Sync(context.Background(), source); err != nil {
		t.Fatalf("old head must be a no-op: %v", err)
	}
	// A new head that cannot extend the retained leaves is divergence, not a
	// recovery opportunity. Its rejected data never replaces the mirror.
	// Sign a valid successor that commits to a different first leaf. The
	// signature is valid, but it cannot extend the local MMR prefix.
	original := source.head.Head
	otherFirst := mmr.LeafHash([]byte("other-first"))
	remoteSecond := mmr.LeafHash([]byte("remote-second"))
	forkRoot := mmr.FromLeaves([]mmr.Hash{otherFirst, remoteSecond}).BagRoot()
	forkHead := original
	forkHead.Seq = 2
	forkHead.PrevHeadHash = "head-1"
	forkHead.MMRSize = 2
	forkHead.MMRRoot = hex.EncodeToString(forkRoot[:])
	forkHead.ProducedAt = time.Now().Unix()
	source.head = replication.SignedHead{Head: forkHead}
	source.leaves = []string{hex.EncodeToString(otherFirst[:]), hex.EncodeToString(remoteSecond[:])}
	if err := follower.Sync(context.Background(), source); err == nil {
		t.Fatal("fork accepted")
	}
	after := follower.Snapshot()
	if after.Head.Head.Seq != before.Head.Head.Seq || after.Leaves[0] != before.Leaves[0] {
		t.Fatalf("fork overwrote local mirror: before=%+v after=%+v", before, after)
	}
}

func TestVerifyCheckpointsRequiresDeclaredMMRLeaf(t *testing.T) {
	leaf := mmr.LeafHash([]byte("checkpoint"))
	record := &types.CheckpointRecord{Checkpoint: types.Checkpoint{ChainID: "chain-a", StartHeight: 1}, MMRPosition: 0, LeafHash: [32]byte(leaf)}
	leaves := []string{hex.EncodeToString(leaf[:])}
	if err := verifyCheckpoints([]*types.CheckpointRecord{record}, leaves); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	bad := *record
	bad.LeafHash = [32]byte(mmr.LeafHash([]byte("other")))
	if err := verifyCheckpoints([]*types.CheckpointRecord{&bad}, leaves); err == nil {
		t.Fatal("checkpoint with a non-committed leaf was accepted")
	}
}

func signedSource(t *testing.T) (fakeSource, string) {
	t.Helper()
	dir := t.TempDir()
	log, err := ledger.Open(filepath.Join(dir, "state_log.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	genesis, err := log.Append(ledger.KindGenesis, ledger.Genesis{ChainID: "oracle-test", Name: "NRN", Symbol: "NRN", Owner: "owner", InitialSupply: "100", MaxSupply: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	mint, err := log.Append(ledger.KindTokenMint, ledger.TokenMint{To: "alice", Amount: "5"})
	if err != nil {
		t.Fatal(err)
	}
	state, err := ledger.Replay([]ledger.Record{genesis, mint})
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := ledger.StateRoot(state)
	if err != nil {
		t.Fatal(err)
	}
	leaf := mmr.LeafHash([]byte("state:" + mint.Hash))
	root := mmr.FromLeaves([]mmr.Hash{leaf}).BagRoot()

	head := replication.Head{Version: replication.Version, NetworkID: "testnet", OracleChainID: "oracle-test", OwnerAddress: "root", Epoch: 1, Seq: 1, MMRSize: 1, MMRRoot: hex.EncodeToString(root[:]), RegistryHash: registryHash(nil), StateLogSize: 2, StateRoot: stateRoot, AuxHash: auxHash(AuxSnapshot{}), ConsensusHeight: 1, ProducedAt: time.Now().Unix()}
	return fakeSource{head: replication.SignedHead{Head: head}, leaves: []string{hex.EncodeToString(leaf[:])}, records: []StateRecord{
		{Seq: genesis.Seq, Kind: genesis.Kind, Payload: genesis.Payload, PrevHash: genesis.PrevHash, Hash: genesis.Hash},
		{Seq: mint.Seq, Kind: mint.Kind, Payload: mint.Payload, PrevHash: mint.PrevHash, Hash: mint.Hash, Leaf: hex.EncodeToString(leaf[:])},
	}}, "root"
}

func useTestVerification(f *Follower) {
	f.verifyHead = func(s replication.SignedHead, _ string, _ time.Time) (replication.Head, error) { return s.Head, nil }
	f.compare = func(local, next replication.Head) (replication.FollowResult, error) {
		if next.Seq <= local.Seq {
			return replication.FollowOld, nil
		}
		if next.Seq > local.Seq+1 {
			return replication.FollowGap, nil
		}
		if next.PrevHeadHash != "head-1" {
			return replication.FollowDiverged, replication.ErrDiverged
		}
		return replication.FollowAccepted, nil
	}
}
