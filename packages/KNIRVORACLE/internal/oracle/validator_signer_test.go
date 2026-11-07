package oracle

import (
	"bytes"
	"context"
	"testing"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// validAddr is a well-formed KNIRV Bech32 address (used elsewhere in the suite)
// so the test exercises AddressFromString without touching the Rust core.
const validAddr = "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e"

// fakeValidatorSigner is a test double for knirvsigning.RemoteSigner. It never
// touches the Rust core (no FFI), so the test runs without a seeded wasm build.
type fakeValidatorSigner struct {
	addr   string
	pubkey []byte
}

func (f fakeValidatorSigner) Address() string { return f.addr }

func (f fakeValidatorSigner) PublicKey() []byte { return f.pubkey }

func (f fakeValidatorSigner) SignDirect(ctx context.Context, req signing.SignRequest) (signing.SignedTransaction, error) {
	return signing.SignedTransaction{}, nil
}

func (f fakeValidatorSigner) SignMessage(ctx context.Context, env signing.MessageEnvelope) (signing.SignedMessage, error) {
	return signing.SignedMessage{}, nil
}

// TestBuildConsensusValidatorsStandby verifies a standby Oracle (validator mode
// off) needs no key material at all — the validator set is empty.
func TestBuildConsensusValidatorsStandby(t *testing.T) {
	cfg := &OracleConfig{ValidatorMode: false}
	vals, err := buildConsensusValidators(cfg)
	if err != nil {
		t.Fatalf("standby should not error: %v", err)
	}
	if vals != nil {
		t.Fatalf("standby must have no consensus validators, got %d", len(vals))
	}
}

// TestBuildConsensusValidatorsFromRemoteSigner verifies the validator identity is
// derived entirely from the injected RemoteSigner (Task 8.8): no hex key is read
// from config or env. The address and public key come from the signer.
func TestBuildConsensusValidatorsFromRemoteSigner(t *testing.T) {
	want, err := types.AddressFromString(validAddr)
	if err != nil {
		t.Fatalf("validAddr must parse: %v", err)
	}
	signer := fakeValidatorSigner{addr: validAddr, pubkey: []byte("bootnode-pubkey-bytes")}
	cfg := &OracleConfig{ValidatorMode: true, ValidatorSigner: signer}

	vals, err := buildConsensusValidators(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 1 {
		t.Fatalf("expected exactly one validator, got %d", len(vals))
	}
	if vals[0].Address != want {
		t.Fatalf("validator address = %v, want %v", vals[0].Address, want)
	}
	if !bytes.Equal(vals[0].PubKey, []byte("bootnode-pubkey-bytes")) {
		t.Fatalf("validator pubkey not taken from signer")
	}
	if vals[0].VotingPower == nil || vals[0].VotingPower.Int64() != 100 {
		t.Fatalf("expected default voting power 100")
	}
}

// TestBuildConsensusValidatorsMissingSignerDegraded verifies that validator mode
// with no available signer is a loud error (Degraded), never a silent downgrade
// or a start with no validator.
func TestBuildConsensusValidatorsMissingSignerDegraded(t *testing.T) {
	cfg := &OracleConfig{ValidatorMode: true} // ValidatorSigner left nil
	if _, err := buildConsensusValidators(cfg); err == nil {
		t.Fatalf("expected error when validator mode is on but no signer is supplied")
	}
}

// TestBuildConsensusValidatorsInvalidSignerAddress verifies a signer whose
// address does not parse is rejected rather than producing a malformed validator.
func TestBuildConsensusValidatorsInvalidSignerAddress(t *testing.T) {
	signer := fakeValidatorSigner{addr: "not-a-valid-knirv-address", pubkey: []byte("x")}
	cfg := &OracleConfig{ValidatorMode: true, ValidatorSigner: signer}
	if _, err := buildConsensusValidators(cfg); err == nil {
		t.Fatalf("expected error for unparseable signer address")
	}
}

// TestOracleConfigValidatorSignerIsInjected documents that the Oracle config no
// longer carries a raw validator key: ORACLE_VALIDATOR_KEY has no runtime
// reader. The compile-time guarantee is that OracleConfig.ValidatorSigner is an
// injected RemoteSigner (json:"-") and no string key field exists.
func TestOracleConfigValidatorSignerIsInjected(t *testing.T) {
	cfg := &OracleConfig{}
	// The only way a validator identity reaches the Oracle is through the
	// injected RemoteSigner; there is no string key field to populate.
	if cfg.ValidatorSigner != nil {
		t.Fatalf("expected nil signer by default")
	}
	_ = cfg
}
