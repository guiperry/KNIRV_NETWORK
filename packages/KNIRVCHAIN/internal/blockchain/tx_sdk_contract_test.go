package blockchain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// sdkSignedTransfer signs an action with the canonical SDK signer and maps the result to
// the chain's Transaction, the way the Arena's chain writes do.
func sdkSignedTransaction(t *testing.T, txType string, payload []byte, fee uint64) *Transaction {
	t.Helper()
	seed := sha256.Sum256([]byte("knirv-contract-test-key"))
	probe, err := knirvsigning.SignTransaction(seed[:], knirvsigning.SignRequest{
		Action:  knirvsigning.Action{SchemaVersion: knirvsigning.ActionSchemaVersion, Action: txType, Sender: "probe", Payload: payload, TimestampUnix: 1700000000},
		ChainID: "knirv-1",
	})
	if err != nil {
		t.Fatalf("probe sign: %v", err)
	}
	sender := probe.Address
	signed, err := knirvsigning.SignTransaction(seed[:], knirvsigning.SignRequest{
		Action: knirvsigning.Action{
			SchemaVersion: knirvsigning.ActionSchemaVersion, Action: txType, Sender: sender,
			Payload: payload, TimestampUnix: 1700000000,
		},
		ChainID:       "knirv-1",
		AccountNumber: AccountNumberFor(sender),
		Sequence:      0,
		Fee:           knirvsigning.Fee{Denom: "NRN", Amount: "2"},
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return &Transaction{
		From:            sender,
		Data:            payload,
		Timestamp:       1700000000,
		TransactionHash: hex.EncodeToString(signed.Hash),
		PublicKey:       base64.StdEncoding.EncodeToString(signed.PublicKey),
		Signature:       signed.Signatures[0],
		BodyBytes:       base64.StdEncoding.EncodeToString(signed.BodyBytes),
		AuthInfoBytes:   base64.StdEncoding.EncodeToString(signed.AuthInfoBytes),
		ChainID:         "knirv-1",
		AccountNumber:   AccountNumberFor(sender),
		Sequence:        0,
		Fee:             2,
		FeeDenom:        "NRN",
		Type:            txType,
	}
}

// An SDK-signed write verifies on the chain: the mapping the Arena builds is accepted.
func TestSDKSignedCapabilityWriteVerifiesOnTheChain(t *testing.T) {
	t.Setenv(txSequenceActivationEnv, "0")
	tx := sdkSignedTransaction(t, TransactionTypeMCPRegisterCapability, []byte(`{"capabilityID":"skill-1"}`), 2)
	ok, err := tx.VerifySignature()
	if err != nil || !ok {
		t.Fatalf("SDK-signed transaction must verify on the chain: ok=%v err=%v", ok, err)
	}
}

// Changing what was signed invalidates it: the fee is covered by the signature.
func TestSDKSignedWriteRejectsAChangedFee(t *testing.T) {
	t.Setenv(txSequenceActivationEnv, "0")
	tx := sdkSignedTransaction(t, TransactionTypeMCPRegisterCapability, []byte(`{"capabilityID":"skill-1"}`), 2)
	tx.Fee = 3
	if ok, _ := tx.VerifySignature(); ok {
		t.Fatal("a transaction whose fee differs from the signed fee must not verify")
	}
}

// The sequence check reads the transaction's sequence, so the signed sequence must bind it;
// otherwise a relayer could re-sequence a signed transaction past the check.
func TestSDKSignedWriteRejectsAChangedSequence(t *testing.T) {
	t.Setenv(txSequenceActivationEnv, "0")
	tx := sdkSignedTransaction(t, TransactionTypeMCPRegisterCapability, []byte(`{"capabilityID":"skill-1"}`), 2)
	tx.Sequence = 7
	if ok, _ := tx.VerifySignature(); ok {
		t.Fatal("a transaction whose sequence differs from the signed sequence must not verify")
	}
}

// The signed fields are bound by default: a transaction whose fee differs from the signed
// fee does not verify.
func TestSDKSignedWriteBindingIsAlwaysOn(t *testing.T) {
	t.Setenv(txSequenceActivationEnv, "")
	tx := sdkSignedTransaction(t, TransactionTypeMCPRegisterCapability, []byte(`{"capabilityID":"skill-1"}`), 2)
	tx.Fee = 3
	if ok, _ := tx.VerifySignature(); ok {
		t.Fatal("the signed fee must bind by default")
	}
}
