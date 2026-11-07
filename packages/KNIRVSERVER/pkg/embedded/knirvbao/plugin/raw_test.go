package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// storeRawKey writes a key version directly to storage, mirroring storeVersion but
// skipping the WASM-backed NewLocalSigner key validation that cannot run in this
// environment. svc.raw signs with the decred library directly and never touches
// the WASM signer, so this exercises the real signing path.
func storeRawKey(t *testing.T, storage logical.Storage, name string, key []byte) {
	t.Helper()
	if !validName(name) {
		t.Fatalf("invalid name %q", name)
	}
	entry, err := logical.StorageEntryJSON(keyPath(name, 1), keyVersion{PrivateKey: key, Type: "node-identity"})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	meta, err := logical.StorageEntryJSON("keys/"+name+"/metadata", keyMetadata{Version: 1, Type: "node-identity"})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), meta); err != nil {
		t.Fatal(err)
	}
}

// TestRawSignMatchesLegacyFailoverSignature proves the raw endpoint produces a
// 64-byte r||s compact signature byte-identical to the registry's legacy
// dcrd SignCompact(priv, digest, true)[1:] form, so a Bao-signed failover
// signature verifies against the same public key with no Worker change.
func TestRawSignMatchesLegacyFailoverSignature(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}

	rawKey := make([]byte, 32)
	rawKey[31] = 1
	storeRawKey(t, storage, "failover-identity", rawKey)

	message := strings.Join([]string{"knirv-testnet-1", "knirv1validator", "replica-v1", "standby"}, "|")
	digest := sha256.Sum256([]byte(message))
	wantPriv := secp256k1.PrivKeyFromBytes(rawKey)
	want := hex.EncodeToString(ecdsa.SignCompact(wantPriv, digest[:], true)[1:])

	got, err := svc.raw(ctx, storage, "failover-identity", rawSignRequest{Digest: hex.EncodeToString(digest[:])}, time.Now())
	if err != nil {
		t.Fatalf("raw sign: %v", err)
	}
	if got.Signature != want {
		t.Fatalf("raw signature %s != legacy %s", got.Signature, want)
	}
}

// TestRawSignRejectsNonAllowlistedPurpose confirms only failover-identity may be
// signed through the raw endpoint; other key purposes are refused even if they
// exist and the ACL were misconfigured.
func TestRawSignRejectsNonAllowlistedPurpose(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}

	rawKey := make([]byte, 32)
	rawKey[31] = 1
	storeRawKey(t, storage, "root-owner", rawKey)
	digest := sha256.Sum256([]byte("x"))
	if _, err := svc.raw(ctx, storage, "root-owner", rawSignRequest{Digest: hex.EncodeToString(digest[:])}, time.Now()); err == nil {
		t.Fatal("non-allowlisted purpose was signed")
	}
	if _, err := svc.raw(ctx, storage, "anything", rawSignRequest{Digest: hex.EncodeToString(digest[:])}, time.Now()); err == nil {
		t.Fatal("unknown purpose was signed")
	}
}

// TestRawSignRejectsInvalidDigest confirms malformed or wrong-length digests are
// refused rather than signed.
func TestRawSignRejectsInvalidDigest(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}

	rawKey := make([]byte, 32)
	rawKey[31] = 1
	storeRawKey(t, storage, "failover-identity", rawKey)
	for _, d := range []string{"", "zzzz", "abcd"} {
		if _, err := svc.raw(ctx, storage, "failover-identity", rawSignRequest{Digest: d}, time.Now()); err == nil {
			t.Fatalf("invalid digest %q accepted", d)
		}
	}
}

// TestRawSignRouteIsRegistered confirms the public surface exposes the raw path
// with the allowlisted purpose, so the ACL grant in bootstrapPolicies can reach it.
func TestRawSignRouteIsRegistered(t *testing.T) {
	found := false
	for _, p := range candidatePaths(&signerService{}) {
		if strings.Contains(p.Pattern, "sign/raw/") {
			found = true
		}
	}
	if !found {
		t.Fatal("sign/raw/ route is not registered")
	}
}

// TestRawSignMatchesFrozenKnownAnswerVector proves the raw endpoint reproduces an
// EXTERNALLY frozen compact signature, independent of any in-process dcrd primitive.
// Unlike TestRawSignMatchesLegacyFailoverSignature (which re-derives the expected
// value from dcrd on both sides and would miss a shared misuse), this pins the
// expected 64-byte r||s to the canonical KAT in core/src/secrets.rs raw_sign_kat —
// the same vector the Rust core, the Go binding and the TypeScript binding assert.
// A divergence here means the plugin no longer matches the registry's legacy
// failover signature format.
func TestRawSignMatchesFrozenKnownAnswerVector(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}

	// Frozen KAT: private key 1 → secp256k1 generator point G. Digest is
	// sha256("knirv-raw-sign-kat-v1"). See core/src/secrets.rs raw_sign_kat.
	const katDigest = "27a63df12ab9b1c25967ac3a371c19b65d6f326fcd0748b2b3f657bfd0e64e22"
	const katCompact = "20031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4"
	const katSig64 = "031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4"

	priv := make([]byte, 32)
	priv[31] = 1
	storeRawKey(t, storage, "failover-identity", priv)

	digest, err := hex.DecodeString(katDigest)
	if err != nil {
		t.Fatalf("decode KAT digest: %v", err)
	}
	got, err := svc.raw(ctx, storage, "failover-identity", rawSignRequest{Digest: katDigest}, time.Now())
	if err != nil {
		t.Fatalf("raw sign: %v", err)
	}
	if got.Signature != katSig64 {
		t.Fatalf("raw signature %s != frozen KAT sig64 %s", got.Signature, katSig64)
	}

	// The full compact form (recovery byte + 64 r||s) must also be stable and
	// match the frozen vector, proving the strip-[1:] convention is exact.
	wantPriv := secp256k1.PrivKeyFromBytes(priv)
	compact := hex.EncodeToString(ecdsa.SignCompact(wantPriv, digest, true))
	if compact != katCompact {
		t.Fatalf("compact form %s != frozen KAT compact %s", compact, katCompact)
	}
	if compact[2:] != katSig64 {
		t.Fatalf("compact[1:] %s != sig64 %s", compact[2:], katSig64)
	}
}
