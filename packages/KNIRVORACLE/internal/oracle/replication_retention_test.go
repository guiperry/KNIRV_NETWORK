package oracle

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	signing "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/replica"
	"go.uber.org/zap"
)

func TestReplicationUsesProvidedSigner(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(key.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}
	signer, err := signing.NewLocalSigner(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg := newStartupTestConfig(t, t.TempDir())
	cfg.OwnerPrivateKey = key.PrivateKeyHex()
	cfg.ReplicationSigner = signer
	o, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer o.cancel()
	head, ok := o.replication.CurrentHead()
	if !ok || head.Head.OwnerAddress != signer.Address() {
		t.Fatalf("wrong replication signer: %+v", head.Head)
	}
}

func TestReplicationRetentionPreservesSequenceAndFailureRollback(t *testing.T) {
	o, err := NewOracle(newStartupTestConfig(t, t.TempDir()), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer o.cancel()
	s := o.replication
	initial, ok := s.CurrentHead()
	if !ok {
		t.Fatal("initial head missing")
	}
	s.heads = make([]replication.SignedHead, 1024)
	for i := range s.heads {
		s.heads[i] = initial
		s.heads[i].Head.Seq = uint64(i + 1)
	}
	next, err := s.ProduceHead()
	if err != nil {
		t.Fatal(err)
	}
	if next.Head.Seq != 1025 || len(s.heads) != 1024 || s.heads[0].Head.Seq != 2 {
		t.Fatalf("seq=%d len=%d first=%d", next.Head.Seq, len(s.heads), s.heads[0].Head.Seq)
	}
	reloaded := &ReplicationSource{path: s.path}
	if err := reloaded.load(); err != nil {
		t.Fatal(err)
	}
	if len(reloaded.heads) != 1024 || reloaded.heads[1023].Head.Seq != 1025 {
		t.Fatal("retained head chain did not survive reload")
	}
	previousPath := s.path
	s.path = filepath.Join(t.TempDir(), "missing", "heads.json")
	if _, err := s.ProduceHead(); err == nil {
		t.Fatal("expected persistence failure")
	}
	if len(s.heads) != 1024 || s.heads[0].Head.Seq != 2 || s.heads[1023].Head.Seq != 1025 {
		t.Fatal("failed persistence changed the retained head chain")
	}
	s.path = previousPath
	last, err := s.ProduceHead()
	if err != nil || last.Head.Seq != 1026 {
		t.Fatalf("next head=%d err=%v", last.Head.Seq, err)
	}
}

func TestStandbyReServesVerifiedSnapshotForReplication(t *testing.T) {
	active, err := NewOracle(newStartupTestConfig(t, t.TempDir()), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer active.cancel()
	head, ok := active.replication.CurrentHead()
	if !ok {
		t.Fatal("active head missing")
	}
	records := active.ledger.Records()
	state, err := ledger.Replay(records)
	if err != nil {
		t.Fatal(err)
	}
	var aux replica.AuxSnapshot
	data, err := json.Marshal(active.replication.AuxSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		t.Fatal(err)
	}
	snapshot := replica.Snapshot{Head: head, Leaves: active.replication.Leaves(0, 1024), Records: records, State: state, Registry: replica.RegistrySnapshot{Chains: active.RegistrySnapshot(), Hash: active.ReplicationRegistryHash()}, Aux: aux}
	data, err = json.Marshal(map[string]any{"snapshot": snapshot})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mirror.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	follower, err := replica.Open(path, head.Head.OwnerAddress)
	if err != nil {
		t.Fatal(err)
	}
	standby := &Oracle{config: &OracleConfig{Mode: ModeStandby}, follower: follower}
	source := &ReplicationSource{oracle: standby}
	got, ok := source.CurrentHead()
	if !ok || got.Head.Seq != head.Head.Seq || got.Signed.Address != head.Signed.Address {
		t.Fatalf("standby head=%+v", got)
	}
	if heads := source.HeadsSince(0, 1); len(heads) != 1 || heads[0].Head.Seq != head.Head.Seq {
		t.Fatalf("standby head page=%+v", heads)
	}
	if got := source.StateRecords(0, 1); len(got) != 1 || got[0]["hash"] != records[0].Hash {
		t.Fatalf("standby state=%+v", got)
	}
	if standby.ReplicationRegistryHash() != head.Head.RegistryHash || source.AuxSnapshot()["hash"] != head.Head.AuxHash {
		t.Fatal("standby snapshot hashes do not match the signed head")
	}
}

func TestPeriodicReplicationPublishesLedgerChanges(t *testing.T) {
	o, err := NewOracle(newStartupTestConfig(t, t.TempDir()), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	initial, ok := o.replication.CurrentHead()
	if !ok {
		o.cancel()
		t.Fatal("initial head missing")
	}
	s := &ReplicationSource{oracle: o, logger: zap.NewNop(), signer: o.replication.signer, path: filepath.Join(t.TempDir(), "heads.json"), heads: []replication.SignedHead{initial}, epoch: 1, interval: 10 * time.Millisecond}
	done := make(chan struct{})
	go func() { defer close(done); s.run() }()
	defer func() { o.cancel(); <-done }()
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	stake := o.GetGovernanceSystem().GetNetworkParameters().MinValidatorStake
	_, err = o.GetGovernanceSystem().RegisterValidator(&governance.ValidatorRegistration{Address: key.Address, PubKey: []byte{2, 4}, StakedAmount: new(big.Int).Add(stake, big.NewInt(1)), CommissionRate: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(10 * time.Second)
	for {
		if next, ok := s.CurrentHead(); ok && next.Head.Seq > initial.Head.Seq {
			if next.Head.StateLogSize <= initial.Head.StateLogSize || next.Head.MMRSize <= initial.Head.MMRSize {
				t.Fatalf("head did not capture ledger and audit change: %+v", next.Head)
			}
			return
		}
		select {
		case <-deadline:
			t.Fatal("ledger mutation did not produce a periodic head")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
