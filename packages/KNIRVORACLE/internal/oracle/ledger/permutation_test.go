package ledger

import (
	"encoding/json"
	"testing"
)

// mkRecord builds a record the way the log would, without touching disk, so a
// test can choose the order records reach Apply.
func mkRecord(t *testing.T, seq uint64, kind string, payload any) Record {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode %s: %v", kind, err)
	}
	return Record{Seq: seq, Kind: kind, Payload: body, Hash: hashRecord(seq, kind, body, "")}
}

// Records that touch disjoint keys commute: applying them in any order must give
// the same state, so Apply has no dependence on arrival order, map iteration or
// time. Every permutation of four such records is checked.
func TestApplyIsOrderIndependentForDisjointRecords(t *testing.T) {
	genesis := mkRecord(t, 1, KindGenesis, Genesis{ChainID: "c", Name: "n", Symbol: "s", Owner: "owner", InitialSupply: "1000", MaxSupply: "10000"})
	independent := []Record{
		mkRecord(t, 2, KindTokenMint, TokenMint{To: "alice", Amount: "50"}),
		mkRecord(t, 3, KindWalletRegister, WalletRegister{Address: "carol"}),
		mkRecord(t, 4, KindValidatorUpsert, ValidatorUpsert{Address: "v1", Stake: "100", Active: true}),
		mkRecord(t, 5, KindParamsSet, ParamsSet{Params: map[string]any{"min_stake": "10"}}),
	}

	base := func(order []Record) string {
		s := NewState()
		if err := Apply(s, genesis); err != nil {
			t.Fatalf("genesis: %v", err)
		}
		for _, r := range order {
			if err := Apply(s, r); err != nil {
				t.Fatalf("apply %s: %v", r.Kind, err)
			}
		}
		root, err := StateRoot(s)
		if err != nil {
			t.Fatalf("root: %v", err)
		}
		return root
	}

	want := base(independent)
	count := 0
	var permute func(k int, recs []Record)
	permute = func(k int, recs []Record) {
		if k == len(recs) {
			count++
			if got := base(recs); got != want {
				t.Fatalf("permutation %d changed the state root: %s != %s", count, got, want)
			}
			return
		}
		for i := k; i < len(recs); i++ {
			recs[k], recs[i] = recs[i], recs[k]
			permute(k+1, recs)
			recs[k], recs[i] = recs[i], recs[k]
		}
	}
	cp := append([]Record(nil), independent...)
	permute(0, cp)
	if count != 24 {
		t.Fatalf("checked %d permutations, want 24", count)
	}
}

// Applying the same records repeatedly, each time into a fresh state, must give
// one root. Go randomises map iteration per run, so a map-order dependence in
// Apply shows up as a differing root across these iterations.
func TestApplyIsStableAcrossRepeatedReplays(t *testing.T) {
	recs := []Record{
		mkRecord(t, 1, KindGenesis, Genesis{ChainID: "c", Name: "n", Symbol: "s", Owner: "owner", InitialSupply: "1000", MaxSupply: "10000"}),
		mkRecord(t, 2, KindTokenMint, TokenMint{To: "a", Amount: "1"}),
		mkRecord(t, 3, KindTokenMint, TokenMint{To: "b", Amount: "2"}),
		mkRecord(t, 4, KindTokenMint, TokenMint{To: "c", Amount: "3"}),
		mkRecord(t, 5, KindWalletRegister, WalletRegister{Address: "a"}),
		mkRecord(t, 6, KindWalletRegister, WalletRegister{Address: "b"}),
	}
	first := ""
	for i := 0; i < 200; i++ {
		s, err := Replay(recs)
		if err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
		root, err := StateRoot(s)
		if err != nil {
			t.Fatalf("root %d: %v", i, err)
		}
		if i == 0 {
			first = root
			continue
		}
		if root != first {
			t.Fatalf("replay %d root %s != first %s", i, root, first)
		}
	}
}
