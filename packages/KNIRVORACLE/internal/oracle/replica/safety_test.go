package replica

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	signing "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
)

func TestRealTamperedHeadSignatureIsRejected(t *testing.T) {
	source, _ := signedSource(t)
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
	source.head.Head.OwnerAddress = signer.Address()
	source.head, err = replication.SignHead(signer, source.head.Head)
	if err != nil {
		t.Fatal(err)
	}
	source.head.Signed.Signature[0] ^= 1
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), signer.Address())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(context.Background(), source); err == nil {
		t.Fatal("tampered signature accepted")
	}
	if f.Status().HeadSeq != 0 {
		t.Fatal("tampered head advanced the mirror")
	}
}

func TestSignedRootConvergesAndForkPreservesSnapshot(t *testing.T) {
	source, _ := signedSource(t)
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
	source.head.Head.OwnerAddress = signer.Address()
	source.head, err = replication.SignHead(signer, source.head.Head)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "replica.json")
	f, err := Open(path, signer.Address())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SyncSources(context.Background(), []Source{source}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err = Open(path, signer.Address())
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Status(); got.HeadSeq != 1 {
		t.Fatalf("restart status=%+v", got)
	}
	fork := source.head.Head
	fork.MMRRoot = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	source.head, err = replication.SignHead(signer, fork)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SyncSources(context.Background(), []Source{source}); !errors.Is(err, replication.ErrDiverged) {
		t.Fatalf("fork: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("signed fork changed the durable snapshot")
	}
}

type unavailableSource struct{ Source }

type missingHeads struct{ fakeSource }

func (missingHeads) HeadsAfter(context.Context, uint64) ([]replication.SignedHead, error) {
	return nil, nil
}

func TestMissingHeadPageCannotMakeMirrorReady(t *testing.T) {
	source, anchor := signedSource(t)
	source.head.Head.Seq = 3
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.SyncSources(context.Background(), []Source{missingHeads{source}}); !errors.Is(err, ErrGap) {
		t.Fatalf("missing page: %v", err)
	}
	if status := f.Status(); !status.Paused || status.HeadSeq != 0 || status.LagSeq != 3 {
		t.Fatalf("status: %+v", status)
	}
}

type countingUnavailable struct {
	Source
	calls int
}

func (s *countingUnavailable) Head(context.Context) (replication.SignedHead, error) {
	s.calls++
	return replication.SignedHead{}, errors.New("offline")
}

func TestSourceRetriesBackOffIndependently(t *testing.T) {
	source, anchor := signedSource(t)
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	now := time.Unix(100, 0)
	f.now = func() time.Time { return now }
	failed := &countingUnavailable{}
	if err := f.SyncSources(context.Background(), []Source{failed, source}); err != nil {
		t.Fatal(err)
	}
	if err := f.SyncSources(context.Background(), []Source{failed, source}); err != nil {
		t.Fatal(err)
	}
	if failed.calls != 1 {
		t.Fatalf("calls before backoff expiry = %d", failed.calls)
	}
	if err := f.SyncSources(context.Background(), []Source{source, failed}); err != nil {
		t.Fatalf("reordered fallback: %v", err)
	}
	now = now.Add(time.Second)
	if err := f.SyncSources(context.Background(), []Source{failed, source}); err != nil {
		t.Fatal(err)
	}
	if failed.calls != 2 {
		t.Fatalf("calls after backoff expiry = %d", failed.calls)
	}
}

func TestGapReportsLagWithoutAdvancing(t *testing.T) {
	source, anchor := signedSource(t)
	source.head.Head.Seq = 3
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.Sync(context.Background(), source); !errors.Is(err, ErrGap) {
		t.Fatalf("gap: %v", err)
	}
	if st := f.Status(); st.HeadSeq != 0 || st.LagSeq != 3 {
		t.Fatalf("gap status=%+v", st)
	}
}

func (unavailableSource) Head(context.Context) (replication.SignedHead, error) {
	return replication.SignedHead{}, errors.New("offline")
}

func TestSourceFallbackAndAllSourcesDown(t *testing.T) {
	source, anchor := signedSource(t)
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.SyncSources(context.Background(), []Source{unavailableSource{}, source}); err != nil {
		t.Fatal(err)
	}
	if st := f.Status(); st.Paused || st.HeadSeq != 1 {
		t.Fatalf("fallback status=%+v", st)
	}
	if err := f.SyncSources(context.Background(), []Source{unavailableSource{}}); err == nil {
		t.Fatal("expected outage")
	}
	if st := f.Status(); !st.Paused || st.HeadSeq != 1 {
		t.Fatalf("outage status=%+v", st)
	}
}

func TestStaleSourceDoesNotMakeMirrorReady(t *testing.T) {
	source, anchor := signedSource(t)
	f, err := Open(filepath.Join(t.TempDir(), "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.SyncSources(context.Background(), []Source{source}); err != nil {
		t.Fatal(err)
	}
	old := source
	source.head.Head.Seq = 2
	source.head.Head.PrevHeadHash = "head-1"
	if err := f.SyncSources(context.Background(), []Source{source}); err != nil {
		t.Fatal(err)
	}
	if err := f.SyncSources(context.Background(), []Source{old}); !errors.Is(err, ErrGap) {
		t.Fatalf("stale source: %v", err)
	}
	if got := f.Status(); !got.Paused || got.HeadSeq != 2 {
		t.Fatalf("stale status: %+v", got)
	}
}

func TestForkStatusSurvivesRestartWithoutChangingSnapshot(t *testing.T) {
	source, anchor := signedSource(t)
	path := filepath.Join(t.TempDir(), "replica.json")
	f, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.Sync(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source.head.Head.MMRRoot = "different"
	if err := f.Sync(context.Background(), source); !errors.Is(err, replication.ErrDiverged) {
		t.Fatalf("fork: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("fork changed the durable snapshot")
	}
	reopened, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Status().Diverged {
		t.Fatalf("restart status=%+v", reopened.Status())
	}
}

func TestClearDivergenceRequiresRetainedHeadAcknowledgement(t *testing.T) {
	source, anchor := signedSource(t)
	path := filepath.Join(t.TempDir(), "replica.json")
	f, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	if err := f.Sync(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source.head.Head.MMRRoot = "different"
	if err := f.Sync(context.Background(), source); !errors.Is(err, replication.ErrDiverged) {
		t.Fatalf("fork: %v", err)
	}
	if err := f.ClearDivergence("not-the-retained-head"); err == nil {
		t.Fatal("wrong acknowledgement cleared divergence")
	}
	headHash, err := f.Snapshot().Head.Head.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.ClearDivergence(headHash); err != nil {
		t.Fatalf("clear divergence: %v", err)
	}
	if got := f.Status(); got.Diverged || !got.Paused {
		t.Fatalf("clear status=%+v", got)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("clear changed the durable snapshot")
	}
	reopened, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Status(); got.Diverged || !got.Paused {
		t.Fatalf("reopened status=%+v", got)
	}
}

func TestBadSignatureAndFailedWriteNeverApply(t *testing.T) {
	source, anchor := signedSource(t)
	dir := t.TempDir()
	f, err := Open(filepath.Join(dir, "replica.json"), anchor)
	if err != nil {
		t.Fatal(err)
	}
	f.verifyHead = func(replication.SignedHead, string, time.Time) (replication.Head, error) {
		return replication.Head{}, replication.ErrBadSignature
	}
	if err := f.Sync(context.Background(), source); !errors.Is(err, replication.ErrBadSignature) {
		t.Fatalf("bad signature: %v", err)
	}
	if f.Status().HeadSeq != 0 {
		t.Fatal("bad signature advanced head")
	}
	useTestVerification(f)
	called := false
	f.SetOnApplied(func(Snapshot) error { called = true; return nil })
	f.path = filepath.Join(dir, "missing", "replica.json")
	if err := f.Sync(context.Background(), source); err == nil {
		t.Fatal("expected failed durable write")
	}
	if called || f.Status().HeadSeq != 0 {
		t.Fatal("failed write exposed an uncommitted snapshot")
	}
}

func TestFailedHydrationKeepsPausedRecoverySnapshot(t *testing.T) {
	source, anchor := signedSource(t)
	path := filepath.Join(t.TempDir(), "replica.json")
	f, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	useTestVerification(f)
	f.SetOnApplied(func(Snapshot) error { return errors.New("partial write") })
	if err := f.Sync(context.Background(), source); err == nil {
		t.Fatal("expected hydration failure")
	}
	if !f.Status().Paused || f.Status().HeadSeq != 0 {
		t.Fatalf("live status: %+v", f.Status())
	}
	reopened, err := Open(path, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Status(); got.HeadSeq != 1 || !got.Paused {
		t.Fatalf("recovery status: %+v", got)
	}
}
