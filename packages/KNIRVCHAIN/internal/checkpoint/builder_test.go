package checkpoint

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"testing"

	"KNIRVCHAIN/internal/database"
	"github.com/ethereum/go-ethereum/crypto"
)

// fixtureChain is an in-memory CheckpointSource for unit tests.
type fixtureChain struct {
	chainID string
	tip     uint64
	roots   map[uint64][32]byte
	authors map[string]bool
}

func (f *fixtureChain) GetChainID() string { return f.chainID }
func (f *fixtureChain) TipHeight() uint64  { return f.tip }
func (f *fixtureChain) AccumRootAt(h uint64) ([32]byte, error) {
	if h > f.tip {
		return [32]byte{}, errHeight
	}
	return f.roots[h], nil
}
func (f *fixtureChain) NetworkAuthors() map[string]bool { return f.authors }

var errHeight = heightError("height beyond tip")

type heightError string

func (e heightError) Error() string { return string(e) }

func genKey(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return key, OracleAddress(&key.PublicKey)
}

// buildFixture builds a chain of n blocks with deterministic AccumRoots.
func buildFixture(n int) *fixtureChain {
	f := &fixtureChain{
		chainID: "test-chain",
		tip:     uint64(n - 1),
		roots:   make(map[uint64][32]byte),
		authors: make(map[string]bool),
	}
	var acc [32]byte
	for i := 0; i < n; i++ {
		acc = sha256sum(append(acc[:], byte(i)))
		f.roots[uint64(i)] = acc
	}
	return f
}

func sha256sum(b []byte) [32]byte {
	return sha256.Sum256(b)
}

func newTestBuilder(t *testing.T, f *fixtureChain, interval, finality uint64) (*Builder, *database.LevelDB, map[string]*ecdsa.PrivateKey) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.NewLevelDB(dir)
	if err != nil {
		t.Fatalf("new leveldb: %v", err)
	}
	// Three authors.
	keys := make(map[string]*ecdsa.PrivateKey)
	for i := 0; i < 3; i++ {
		k, addr := genKey(t)
		keys[addr] = k
		f.authors[addr] = true
	}
	b := NewBuilder(f, db, interval, finality, "builder-addr")
	return b, db, keys
}

func TestCheckpointDigestStable(t *testing.T) {
	cp := &Checkpoint{SchemaVersion: SchemaVersion, ChainID: "c", StartHeight: 1, EndHeight: 32}
	d1 := cp.Digest()
	d2 := cp.Digest()
	if d1 != d2 {
		t.Fatal("digest not stable")
	}
	cp.EndHeight = 33
	if cp.Digest() == d1 {
		t.Fatal("digest changed input but hash identical")
	}
}

func TestBuilderNeverCheckpointsAboveTipMinus32(t *testing.T) {
	f := buildFixture(40) // tip 39
	b, _, _ := newTestBuilder(t, f, 32, 32)

	// At tip 39, candidate end = 32 (lastEnd 0 + interval 32). 32 <= 39-32=7? No.
	// So even at tip 39 (only 7 below tip), the checkpoint covering up to 32 is
	// 7 blocks below tip -> still >= finality(32)? 39-32=7 < 32 -> must NOT fire.
	if _, err := b.OnNewBlock(39); err != nil {
		t.Fatalf("OnNewBlock: %v", err)
	}
	// No checkpoint produced yet; tip must be >= end+finality.
	// Push tip to 63 so end=32 sits exactly 31 below tip (still < 32). Not yet.
	// Push to 64: end 32 -> 64-32=32 == finality -> fires.
	f.tip = 64
	cp, err := b.OnNewBlock(64)
	if err != nil {
		t.Fatalf("OnNewBlock: %v", err)
	}
	if cp == nil {
		t.Fatal("expected a checkpoint at tip 64")
	}
	if cp.EndHeight != 32 {
		t.Fatalf("end height = %d, want 32", cp.EndHeight)
	}
	if cp.StartHeight != 1 {
		t.Fatalf("start height = %d, want 1", cp.StartHeight)
	}
	// Reorg deeper than finality alarm: if tip drops below end+finality we never
	// produce a shallower checkpoint (invariant #3). Simulate tip regression.
	f.tip = 50 // 50-32 = 18 < 32 -> would-be shallow
	cp2, err := b.OnNewBlock(50)
	if err != nil {
		t.Fatalf("OnNewBlock reorg: %v", err)
	}
	if cp2 != nil {
		t.Fatalf("reorg produced a shallow checkpoint (end %d, tip %d) — invariant #3 violated", cp2.EndHeight, 50)
	}
}

func TestBuilderYoungChainDoesNotUnderflowFinality(t *testing.T) {
	f := buildFixture(2)
	b, _, _ := newTestBuilder(t, f, 1, 32)
	cp, err := b.OnNewBlock(1)
	if err != nil {
		t.Fatal(err)
	}
	if cp != nil {
		t.Fatalf("young chain produced checkpoint at end %d", cp.EndHeight)
	}
}

func TestBuilderIntervalAndContiguity(t *testing.T) {
	f := buildFixture(200)
	b, _, keys := newTestBuilder(t, f, 32, 32)
	b.SetProgress(0, [32]byte{})

	var produced int
	for tip := uint64(0); tip <= 199; tip++ {
		cp, err := b.OnNewBlock(tip)
		if err != nil {
			t.Fatalf("OnNewBlock %d: %v", tip, err)
		}
		if cp == nil {
			continue
		}
		if err := b.Finalize(cp, keys); err != nil {
			t.Fatalf("finalize tip %d: %v", tip, err)
		}
		// Re-feed the same tip must not produce another checkpoint.
		if again, _ := b.OnNewBlock(tip); again != nil {
			t.Fatalf("duplicate checkpoint at tip %d", tip)
		}
		produced++
	}
	if produced == 0 {
		t.Fatal("no checkpoints produced")
	}
	// Tip 199, interval 32, finality 32 -> end heights: 32,64,96,128,160 (192 is
	// 199-192=7 < 32, withheld). So 5 checkpoints.
	if produced != 5 {
		t.Fatalf("produced %d checkpoints, want 5", produced)
	}
	stored, err := b.LoadStored("test-chain")
	if err != nil {
		t.Fatalf("load stored: %v", err)
	}
	if len(stored) != 5 {
		t.Fatalf("stored %d, want 5", len(stored))
	}
	// Contiguity + prev-hash chain.
	var prev [32]byte
	for i, cp := range stored {
		if cp.PrevCheckHash != prev {
			t.Fatalf("cp %d prev hash mismatch", i)
		}
		if !cp.VerifyQuorum(f.authors, 2, 3) {
			t.Fatalf("cp %d quorum failed", i)
		}
		prev = cp.Digest()
	}
}

func TestBuilderReorgResume(t *testing.T) {
	// Fixture uses HandleChainReorganization pattern: tip drops, then grows back.
	f := buildFixture(120)
	b, _, keys := newTestBuilder(t, f, 32, 32)
	b.SetProgress(0, [32]byte{})

	// Grow to tip 96 -> checkpoints at 32, 64 (96-64=32 ok), 96 withheld (96-96=0).
	for tip := uint64(0); tip <= 96; tip++ {
		cp, _ := b.OnNewBlock(tip)
		if cp != nil {
			if err := b.Finalize(cp, keys); err != nil {
				t.Fatalf("finalize: %v", err)
			}
		}
	}
	lastEnd, _ := b.Progress()
	if lastEnd != 64 {
		t.Fatalf("lastEnd = %d, want 64", lastEnd)
	}
	// Reorg: tip falls to 70. The 64-checkpoint already landed; next candidate is
	// 96 which is now shallow (70-96 < 0). No new checkpoint should appear, and
	// progress must not regress or skip.
	f.tip = 70
	cp, err := b.OnNewBlock(70)
	if err != nil {
		t.Fatalf("reorg OnNewBlock: %v", err)
	}
	if cp != nil {
		t.Fatalf("reorg produced checkpoint at tip 70 (end %d)", cp.EndHeight)
	}
	// Resume: tip climbs back to 128. end 96 now 128-96=32 -> fires; 128 withheld.
	f.tip = 128
	var resumed int
	for tip := uint64(71); tip <= 128; tip++ {
		cp, _ := b.OnNewBlock(tip)
		if cp != nil {
			if err := b.Finalize(cp, keys); err != nil {
				t.Fatalf("resume finalize: %v", err)
			}
			resumed++
		}
	}
	if resumed != 1 {
		t.Fatalf("resumed %d checkpoints, want 1 (end 96)", resumed)
	}
}
