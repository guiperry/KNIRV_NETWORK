package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appendAll(t *testing.T, log *Log) {
	t.Helper()
	steps := []struct {
		kind    string
		payload any
	}{
		{KindGenesis, Genesis{ChainID: "knirvoracle-1", Name: "NRN", Symbol: "NRN", Owner: "owner", InitialSupply: "1000", MaxSupply: "10000"}},
		{KindTokenMint, TokenMint{To: "alice", Amount: "50"}},
		{KindTokenTransfer, TokenTransfer{From: "owner", To: "bob", Amount: "25", Nonce: 0, SetNonce: true}},
		{KindTokenFee, TokenFee{From: "bob", RewardPool: "pool", Burn: "1", Reward: "1"}},
		{KindTokenBurn, TokenBurn{From: "alice", Amount: "5", Reason: "test"}},
		{KindWalletRegister, WalletRegister{Address: "bob"}},
		{KindValidatorUpsert, ValidatorUpsert{Address: "v1", Stake: "100", Active: true}},
		{KindProposalUpsert, ProposalUpsert{ID: "p1", Proposal: map[string]any{"id": "p1", "status": "voting"}}},
		{KindVoteCast, VoteCast{ProposalID: "p1", Voter: "v1", Vote: map[string]any{"option": "yes"}}},
		{KindParamsSet, ParamsSet{Params: map[string]any{"min_stake": "10"}}},
	}
	for _, step := range steps {
		if _, err := log.Append(step.kind, step.payload); err != nil {
			t.Fatalf("append %s: %v", step.kind, err)
		}
	}
}

func TestReplayIsDeterministicAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state_log.json")

	log, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	appendAll(t, log)

	state, err := Replay(log.Records())
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	root, err := StateRoot(state)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	if state.Balances["owner"] != "975" {
		t.Errorf("owner balance = %s, want 975", state.Balances["owner"])
	}
	if state.Balances["bob"] != "23" {
		t.Errorf("bob balance = %s, want 23 (25 received, 2 fee)", state.Balances["bob"])
	}
	if state.TotalSupply != "1044" {
		t.Errorf("total supply = %s, want 1044 (1000+50-1-5)", state.TotalSupply)
	}
	if state.Nonces["owner"] != 1 {
		t.Errorf("owner nonce = %d, want 1", state.Nonces["owner"])
	}
	if !state.Registered["bob"] {
		t.Error("bob should be registered")
	}
	if state.Validators["v1"].Stake != "100" || !state.Validators["v1"].Active {
		t.Errorf("validator state = %+v", state.Validators["v1"])
	}
	if _, ok := state.Proposals["p1"]; !ok {
		t.Error("proposal p1 missing")
	}
	if state.Votes["p1"]["v1"] == nil {
		t.Error("vote missing")
	}
	if !strings.Contains(string(state.Params), "min_stake") {
		t.Errorf("params = %s", state.Params)
	}
	log.Close()

	// Restart: a fresh Open reads the same records and replays to the same root.
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if reopened.Len() != 10 {
		t.Fatalf("reopened log has %d records, want 10", reopened.Len())
	}
	again, err := Replay(reopened.Records())
	if err != nil {
		t.Fatalf("replay after restart: %v", err)
	}
	againRoot, err := StateRoot(again)
	if err != nil {
		t.Fatalf("root after restart: %v", err)
	}
	if againRoot != root {
		t.Fatalf("state root changed across restart: %s != %s", againRoot, root)
	}
}

func TestStateRootIgnoresInsertionOrder(t *testing.T) {
	a := NewState()
	a.Balances["a"] = "1"
	a.Balances["b"] = "2"
	b := NewState()
	b.Balances["b"] = "2"
	b.Balances["a"] = "1"

	ra, err := StateRoot(a)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := StateRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	if ra != rb {
		t.Fatalf("state root depends on map order: %s != %s", ra, rb)
	}
}

func TestLogRejectsTamperedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.Append(KindGenesis, Genesis{ChainID: "c", Name: "n", Symbol: "s", Owner: "o", InitialSupply: "1", MaxSupply: "1"}); err != nil {
		t.Fatal(err)
	}
	log.Close()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(body), `"initial_supply":"1"`, `"initial_supply":"9"`, 1)
	if tampered == string(body) {
		t.Fatal("test did not change the record")
	}
	if err := os.WriteFile(path, []byte(tampered), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("Open accepted a tampered record")
	}
}

func TestApplyFailsClosedOnUnknownKind(t *testing.T) {
	state := NewState()
	err := Apply(state, Record{Seq: 1, Kind: "not_a_kind", Payload: json.RawMessage(`{}`)})
	if err == nil {
		t.Fatal("Apply accepted an unknown record kind")
	}
}

func TestBurnCannotGoNegative(t *testing.T) {
	state := NewState()
	payload, _ := json.Marshal(TokenBurn{From: "nobody", Amount: "1"})
	if err := Apply(state, Record{Seq: 1, Kind: KindTokenBurn, Payload: payload}); err == nil {
		t.Fatal("Apply allowed a burn from an empty balance")
	}
}
