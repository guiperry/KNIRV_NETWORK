package blockchain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/types"
)

const (
	seqSender    = "knirv1sender"
	seqRecipient = "knirv1recipient"
)

func openSequenceDB(t *testing.T) *database.LevelDB {
	t.Helper()
	db, err := database.NewLevelDB(filepath.Join(t.TempDir(), "chain.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Client.Close() })
	return db
}

func setActivation(t *testing.T, value string) {
	t.Helper()
	t.Setenv(txSequenceActivationEnv, value)
}

func transferTx(from string, sequence uint64) *Transaction {
	return &Transaction{
		From:            from,
		To:              seqRecipient,
		Value:           1,
		AccountNumber:   AccountNumberFor(from),
		Sequence:        sequence,
		TransactionHash: "tx-" + from,
	}
}

func TestSequenceEnforcementIsOnByDefaultAndActivationIsOverridable(t *testing.T) {
	setActivation(t, "")
	if !sequenceEnforcedAt(0) {
		t.Fatal("sequencing must be enforced from genesis by default")
	}
	setActivation(t, "5")
	if sequenceEnforcedAt(4) {
		t.Fatal("an overridden activation height must not apply below it")
	}
	if !sequenceEnforcedAt(5) {
		t.Fatal("an overridden activation height must apply at it")
	}
	setActivation(t, "not-a-number")
	if !sequenceEnforcedAt(10) {
		t.Fatal("an invalid override must fall back to enforcement from genesis")
	}
}

func TestAccountNumberIsStableAndPerAddress(t *testing.T) {
	if AccountNumberFor(seqSender) != AccountNumberFor(seqSender) {
		t.Fatal("account number must be deterministic")
	}
	if AccountNumberFor(seqSender) == AccountNumberFor(seqRecipient) {
		t.Fatal("distinct addresses should not share an account number")
	}
}

func TestApplySequenceRequiresTheNextSequenceAndAdvances(t *testing.T) {
	setActivation(t, "0")
	db := openSequenceDB(t)
	effects := newBlockStateEffects()

	if err := effects.applySequence(db, 1, transferTx(seqSender, 0)); err != nil {
		t.Fatalf("first transaction at sequence 0 must apply: %v", err)
	}
	if err := effects.applySequence(db, 1, transferTx(seqSender, 0)); err == nil {
		t.Fatal("replaying sequence 0 in the same block must fail")
	}
	if err := effects.applySequence(db, 1, transferTx(seqSender, 1)); err != nil {
		t.Fatalf("sequence 1 must apply after 0: %v", err)
	}
	if effects.sequences[seqSender] != 2 {
		t.Fatalf("next sequence = %d, want 2", effects.sequences[seqSender])
	}
}

func TestApplySequenceRejectsWrongAccountNumber(t *testing.T) {
	setActivation(t, "0")
	effects := newBlockStateEffects()
	tx := transferTx(seqSender, 0)
	tx.AccountNumber = AccountNumberFor(seqSender) + 1
	if err := effects.applySequence(openSequenceDB(t), 1, tx); err == nil {
		t.Fatal("a transaction signed with another account's number must be rejected")
	}
}

func TestApplySequenceIsSkippedBeforeAnOverriddenActivation(t *testing.T) {
	setActivation(t, "100")
	effects := newBlockStateEffects()
	if err := effects.applySequence(openSequenceDB(t), 1, transferTx(seqSender, 42)); err != nil {
		t.Fatalf("before an overridden activation sequences are not checked: %v", err)
	}
}

func TestCapabilityFromRegistrationBuildsTypedTool(t *testing.T) {
	registration := types.MCPRegisterCapabilityData{
		CapabilityID:   "skill-1",
		Name:           "Skill One",
		Owner:          seqSender,
		Version:        "1",
		Description:    "a tool",
		CapabilityType: "TOOL",
		GasFeeNRN:      3,
		CapabilityDescriptor: map[string]interface{}{
			"inputSchemaJSON":  `{"type":"object"}`,
			"outputSchemaJSON": `{"type":"string"}`,
		},
	}
	got, err := capabilityFromRegistration(registration, 99)
	if err != nil {
		t.Fatalf("capabilityFromRegistration: %v", err)
	}
	tool, ok := got.(types.ToolDescriptor)
	if !ok {
		t.Fatalf("got %T, want types.ToolDescriptor", got)
	}
	if tool.ID != "skill-1" || tool.Name != "Skill One" || tool.Owner != seqSender || tool.Timestamp != 99 {
		t.Fatalf("base fields not carried over: %+v", tool.BaseDescriptor)
	}
	if tool.InputSchemaJSON != `{"type":"object"}` || tool.CapabilityType != types.CapabilityTypeTool {
		t.Fatalf("tool fields not carried over: %+v", tool)
	}
}

func TestCapabilityFromRegistrationRejectsUnknownType(t *testing.T) {
	_, err := capabilityFromRegistration(types.MCPRegisterCapabilityData{CapabilityID: "x", CapabilityType: "LLM"}, 1)
	if err == nil {
		t.Fatal("an unknown capability type must be rejected")
	}
}

func TestRegisteredCapabilityRejectsDuplicatesInTheBlockAndTheLedger(t *testing.T) {
	db := openSequenceDB(t)
	effects := newBlockStateEffects()
	data, err := json.Marshal(types.MCPRegisterCapabilityData{CapabilityID: "dup", Name: "n", Owner: seqSender, CapabilityType: "TOOL"})
	if err != nil {
		t.Fatal(err)
	}
	tx := &Transaction{Type: TransactionTypeMCPRegisterCapability, Data: data, TransactionHash: "reg-1"}

	if err := effects.addRegisteredCapability(db, tx); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if err := effects.addRegisteredCapability(db, tx); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second registration in the same block must fail, got %v", err)
	}
	if len(effects.capabilities) != 1 {
		t.Fatalf("queued %d capabilities, want 1", len(effects.capabilities))
	}
}

func TestIntakeSequenceCountsPendingTransactionsFromTheSender(t *testing.T) {
	setActivation(t, "0")
	bcs := &BlockchainServer{
		db: openSequenceDB(t),
		BlockchainPtr: &BlockchainStruct{
			ChainID:         "knirv-test",
			TransactionPool: []*Transaction{transferTx(seqSender, 0)},
		},
	}
	if err := bcs.db.SaveAccountSequence(seqSender, 4); err != nil {
		t.Fatal(err)
	}

	// Committed sequence 4 plus one pending transaction: the next valid sequence is 5.
	if status, err := bcs.checkIntakeSequence(transferTx(seqSender, 4)); err == nil || status != http.StatusConflict {
		t.Fatalf("a stale sequence must be rejected with 409, got status %d err %v", status, err)
	}
	if _, err := bcs.checkIntakeSequence(transferTx(seqSender, 5)); err != nil {
		t.Fatalf("the next sequence must be accepted: %v", err)
	}
}

func TestAccountInfoReportsTheNextSequence(t *testing.T) {
	setActivation(t, "0")
	bcs := &BlockchainServer{
		db: openSequenceDB(t),
		BlockchainPtr: &BlockchainStruct{
			ChainID:         "knirv-test",
			TransactionPool: []*Transaction{transferTx(seqSender, 2)},
		},
	}
	if err := bcs.db.SaveAccountSequence(seqSender, 2); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	bcs.handleAccountInfo(rec, httptest.NewRequest(http.MethodGet, "/accounts/info?address="+seqSender, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var info map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info["sequence"].(float64) != 2 || info["pending"].(float64) != 1 || info["next_sequence"].(float64) != 3 {
		t.Fatalf("unexpected account info: %v", info)
	}
	if uint64(info["account_number"].(float64)) != AccountNumberFor(seqSender) {
		t.Fatalf("account_number = %v, want %d", info["account_number"], AccountNumberFor(seqSender))
	}
}

func TestAccountInfoRequiresAnAddress(t *testing.T) {
	rec := httptest.NewRecorder()
	(&BlockchainServer{}).handleAccountInfo(rec, httptest.NewRequest(http.MethodGet, "/accounts/info", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestFundsAfterPendingCountsQueuedCosts(t *testing.T) {
	pending := []*Transaction{{From: seqSender, Fee: 3, Value: 2}} // costs 5
	ok := &Transaction{From: seqSender, Fee: 3, Value: 0}
	if err := fundsAfterPending(10, pending, ok); err != nil {
		t.Fatalf("balance 10 covers 5 pending plus 3: %v", err)
	}
	short := &Transaction{From: seqSender, Fee: 6, Value: 0}
	if err := fundsAfterPending(10, pending, short); err == nil {
		t.Fatal("balance 10 cannot cover 5 pending plus 6")
	}
	valued := &Transaction{From: seqSender, Fee: 4, Value: 1}
	if err := fundsAfterPending(10, pending, valued); err != nil {
		t.Fatalf("fee plus value of 5 fits the remaining 5: %v", err)
	}
	if err := fundsAfterPending(0, nil, &Transaction{From: seqSender, Fee: 1}); err == nil {
		t.Fatal("an unfunded sender must be rejected")
	}
}

func TestRegistrationContentRejectsMissingFields(t *testing.T) {
	missing, err := json.Marshal(types.MCPRegisterCapabilityData{CapabilityID: "x", CapabilityType: "TOOL"})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRegistrationContent(&Transaction{Data: missing}); err == nil {
		t.Fatal("a registration without name and owner must be rejected")
	}
	complete, err := json.Marshal(types.MCPRegisterCapabilityData{CapabilityID: "x", Name: "n", Owner: seqSender, CapabilityType: "TOOL"})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRegistrationContent(&Transaction{Data: complete}); err != nil {
		t.Fatalf("a complete tool registration must pass: %v", err)
	}
}
