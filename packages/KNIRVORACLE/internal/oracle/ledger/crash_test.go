package ledger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Crash injection for the append path. The fault hook fails the commit at one
// named stage, which is what a crash or a failed fsync looks like to the caller.
// Every test then reopens the file, as a restart would, and checks what replay
// sees. No process is spawned.

var errInjected = errors.New("injected crash")

func crashAt(stage string) func(string) error {
	return func(s string) error {
		if s == stage {
			return errInjected
		}
		return nil
	}
}

func openWithGenesis(t *testing.T, path string) *Log {
	t.Helper()
	log, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := log.Append(KindGenesis, Genesis{ChainID: "c", Name: "n", Symbol: "s", Owner: "owner", InitialSupply: "100", MaxSupply: "1000"}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	return log
}

// A crash before any byte is written loses the record and nothing else.
func TestCrashBeforeWriteLosesOnlyTheUnackedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	log := openWithGenesis(t, path)
	log.fault = crashAt("before_write")
	if _, err := log.Append(KindTokenMint, TokenMint{To: "alice", Amount: "5"}); !errors.Is(err, errInjected) {
		t.Fatalf("append err = %v, want injected", err)
	}
	if log.Len() != 1 {
		t.Fatalf("in-memory len = %d, want 1", log.Len())
	}
	log.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if reopened.Len() != 1 {
		t.Fatalf("replayed len = %d, want 1 (nothing was written)", reopened.Len())
	}
	if _, err := reopened.Append(KindTokenMint, TokenMint{To: "alice", Amount: "5"}); err != nil {
		t.Fatalf("append after recovery: %v", err)
	}
}

// A crash after the bytes reach the file but before fsync returns: the caller is
// told the append failed, yet the record is on disk. Replay is the authority, so
// the record must replay as committed, with a valid chain.
func TestCrashAfterWriteReplaysTheDurableRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	log := openWithGenesis(t, path)
	log.fault = crashAt("after_write")
	if _, err := log.Append(KindTokenMint, TokenMint{To: "alice", Amount: "5"}); !errors.Is(err, errInjected) {
		t.Fatalf("append err = %v, want injected", err)
	}
	log.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if reopened.Len() != 2 {
		t.Fatalf("replayed len = %d, want 2 (the written record is durable)", reopened.Len())
	}
	state, err := Replay(reopened.Records())
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if state.Balances["alice"] != "5" {
		t.Fatalf("alice = %q, want 5", state.Balances["alice"])
	}
}

// Regression: a failed write or fsync that left bytes on disk must poison the
// live log. Before the fix, the next append reused the sequence number already on
// disk and the file no longer replayed ("broken chain").
func TestFailedCommitPoisonsLiveLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	log := openWithGenesis(t, path)
	log.fault = crashAt("after_sync")
	if _, err := log.Append(KindTokenMint, TokenMint{To: "alice", Amount: "5"}); !errors.Is(err, errInjected) {
		t.Fatalf("append err = %v, want injected", err)
	}
	log.fault = nil
	if _, err := log.Append(KindTokenMint, TokenMint{To: "bob", Amount: "1"}); !errors.Is(err, errLogBroken) {
		t.Fatalf("append after failed commit err = %v, want errLogBroken", err)
	}
	if _, err := log.AppendBatch([]BatchEntry{{KindWalletRegister, WalletRegister{Address: "x"}}}); !errors.Is(err, errLogBroken) {
		t.Fatalf("batch after failed commit err = %v, want errLogBroken", err)
	}
	log.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen after poisoned log: %v", err)
	}
	defer reopened.Close()
	if reopened.Len() != 2 {
		t.Fatalf("replayed len = %d, want 2", reopened.Len())
	}
}

// A crash during a batch write: the batch is applied whole or not at all, at
// every injected stage.
func TestCrashDuringBatchIsAllOrNothing(t *testing.T) {
	for _, stage := range []string{"before_write", "after_write", "after_sync"} {
		t.Run(stage, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state_log.json")
			log := openWithGenesis(t, path)
			log.fault = crashAt(stage)
			_, err := log.AppendBatch([]BatchEntry{
				{KindWalletRegister, WalletRegister{Address: "b1"}},
				{KindWalletRegister, WalletRegister{Address: "b2"}},
			})
			if !errors.Is(err, errInjected) {
				t.Fatalf("batch err = %v, want injected", err)
			}
			log.Close()

			reopened, err := Open(path)
			if err != nil {
				t.Fatalf("reopen: %v", err)
			}
			defer reopened.Close()
			state, err := Replay(reopened.Records())
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			if state.Registered["b1"] != state.Registered["b2"] {
				t.Fatalf("batch applied partially: b1=%v b2=%v", state.Registered["b1"], state.Registered["b2"])
			}
			// Before the write nothing is on disk (genesis only). Once the write
			// has happened the complete frame is on disk, so it replays whole,
			// whether or not fsync returned.
			wantLen := 3
			if stage == "before_write" {
				wantLen = 1
			}
			if reopened.Len() != wantLen {
				t.Fatalf("replayed len = %d, want %d", reopened.Len(), wantLen)
			}
		})
	}
}

// Truncating the file at any byte offset (a torn write of the last line) must
// replay exactly the complete records before the cut, with the same state root
// that a fresh replay of that prefix produces.
func TestTruncationAtEveryOffsetReplaysAPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state_log.json")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	appendAll(t, log)
	all := log.Records()
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Byte offset just past the end of each complete line (record i is complete
	// once its newline is written).
	var ends []int
	for i, b := range raw {
		if b == '\n' {
			ends = append(ends, i+1)
		}
	}
	if len(ends) != len(all) {
		t.Fatalf("lines = %d, records = %d", len(ends), len(all))
	}

	cutPath := filepath.Join(dir, "cut.json")
	for cut := 0; cut <= len(raw); cut++ {
		if err := os.WriteFile(cutPath, raw[:cut], 0600); err != nil {
			t.Fatal(err)
		}
		tl, err := Open(cutPath)
		if err != nil {
			t.Fatalf("cut %d: open: %v", cut, err)
		}
		want := 0
		for _, e := range ends {
			// A record is complete once its JSON bytes are present; a cut that
			// removes only the trailing newline still replays that record.
			if e-1 <= cut {
				want++
			}
		}
		if tl.Len() != want {
			t.Fatalf("cut %d: replayed %d records, want %d", cut, tl.Len(), want)
		}
		got, err := Replay(tl.Records())
		if err != nil {
			t.Fatalf("cut %d: replay: %v", cut, err)
		}
		wantState, err := Replay(all[:want])
		if err != nil {
			t.Fatalf("cut %d: reference replay: %v", cut, err)
		}
		gr, _ := StateRoot(got)
		wr, _ := StateRoot(wantState)
		if gr != wr {
			t.Fatal(fmt.Sprintf("cut %d: prefix state root differs", cut))
		}
		tl.Close()
	}
}
