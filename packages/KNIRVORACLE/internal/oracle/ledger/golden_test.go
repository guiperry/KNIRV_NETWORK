package ledger

import (
	"path/filepath"
	"testing"
)

// Golden values for the fixed log that appendAll writes. The brief asks that
// replay of the same log in two fresh processes give the same state_root. Spawning
// processes is banned for this suite, so this test pins the value a fresh process
// must reproduce: the log's head hash and the state root after replay. Any change
// to Apply, the canonical encoding or the record hash changes these constants and
// must be a deliberate format change. Regenerate them only with that intent.
const (
	goldenStateRoot = "37e0936ef79a8dfa5abcfe90d04a9ea04e91055119f760e2a469c37a4989740a"
	goldenLastHash  = "3dc0a499f4ef07f7bb0235b0b2a7a7b24a0fa88e637a1f91cf346d2a9a4b1bae"
)

func TestReplayMatchesGoldenStateRootAndHead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	writer, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	appendAll(t, writer)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	// A fresh Log and a fresh State, as a new process would build them.
	reader, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if reader.LastHash() != goldenLastHash {
		t.Fatalf("log head = %s, want %s", reader.LastHash(), goldenLastHash)
	}
	state, err := Replay(reader.Records())
	if err != nil {
		t.Fatal(err)
	}
	root, err := StateRoot(state)
	if err != nil {
		t.Fatal(err)
	}
	if root != goldenStateRoot {
		t.Fatalf("state root = %s, want %s", root, goldenStateRoot)
	}
}
