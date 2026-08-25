package nrv

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

func TestLoadOrCreateIdentity_PersistsAcrossCalls(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreateIdentity(dir)
	if err != nil {
		t.Fatalf("first LoadOrCreateIdentity: %v", err)
	}
	second, err := LoadOrCreateIdentity(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreateIdentity: %v", err)
	}

	if first.SignerID() != second.SignerID() {
		t.Fatalf("identity not persisted: %q != %q", first.SignerID(), second.SignerID())
	}
}

func TestLoadOrCreateIdentity_DifferentDirsDifferentIdentity(t *testing.T) {
	id1, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	id2, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	if id1.SignerID() == id2.SignerID() {
		t.Fatal("expected distinct identities for distinct data directories")
	}
}

func TestSignerIDIsKnirvAddress(t *testing.T) {
	id, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	raw, err := knirvsigning.DecodeAddress(id.SignerID(), knirvsigning.DefaultAddressPrefix)
	if err != nil {
		t.Fatalf("SignerID is not a valid KNIRV address: %v", err)
	}
	if len(raw) != 20 {
		t.Fatalf("decoded SignerID payload length = %d, want 20", len(raw))
	}
}

func TestSignVerifiesWithSDK(t *testing.T) {
	id, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	const chainID = "knirv-testnet-1"
	root := "sha256:deadbeef"
	wire, err := id.Sign(chainID, root)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(wire)
	if err != nil {
		t.Fatalf("Sign() output is not valid base64: %v", err)
	}
	var signed knirvsigning.SignedMessage
	if err := json.Unmarshal(raw, &signed); err != nil {
		t.Fatalf("Sign() output is not a valid signed message: %v", err)
	}
	if err := knirvsigning.VerifyMessagePayload(signed, nrvSigningDomain, nrvSigningPurpose, chainID, []byte(root), time.Now()); err != nil {
		t.Fatalf("signature does not verify against the identity's own public key: %v", err)
	}
	if signed.Address != id.SignerID() {
		t.Fatalf("signed message address = %q, want %q", signed.Address, id.SignerID())
	}
}

func TestSigningKeyIDFormat(t *testing.T) {
	id, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	const prefix = "secp256k1:"
	if got := id.SigningKeyID(); len(got) <= len(prefix) || got[:len(prefix)] != prefix {
		t.Fatalf("SigningKeyID() = %q, want prefix %q", got, prefix)
	}
}
