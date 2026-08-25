// Package nrv packages detected errors and vulnerabilities into self-contained
// .nrv reports and submits them to KNIRVGRAPH via KNIRVGATEWAY's public
// /api/graph/nrv/errors/commit endpoint.
package nrv

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

const identityFileName = "identity.key"

// nrvSigningDomain and nrvSigningPurpose domain-separate .nrv error-commit
// signatures within KNIRVSDK's canonical signed-message scheme (see
// packages/KNIRVSDK/go-package/signing/direct.go's MessageEnvelope) — the
// same scheme KNIRVCHAIN, KNIRVGATEWAY, and KNIRVORACLE use for their own
// signer identities.
const (
	nrvSigningDomain     = "knirv.nrv"
	nrvSigningPurpose    = "error-node-commit"
	nrvSignatureValidity = 5 * time.Minute
)

// Identity is a locally-generated secp256k1 keypair used to sign .nrv error
// commits via KNIRVSDK's signing package (github.com/guiperry/knirv-sdk-go),
// rather than an ad-hoc scheme private to KNIRVCLIENT. KNIRVGRAPH does not
// yet verify signatures against a trust store (see rpc.go's
// createErrorCommit comment), so any stable, self-consistent identity
// satisfies the wire contract; this gives each KNIRVCLIENT installation a
// durable, network-recognizable signer address across scans and projects.
type Identity struct {
	private []byte // 32-byte secp256k1 private key
	pubKey  []byte // 33-byte compressed secp256k1 public key
	address string // bech32 "knirv1..." address derived from pubKey
}

// LoadOrCreateIdentity reads the persisted secp256k1 key from
// <dataDir>/nrv/identity.key, generating and persisting a new one if none
// exists yet.
func LoadOrCreateIdentity(dataDir string) (*Identity, error) {
	nrvDir := filepath.Join(dataDir, "nrv")
	if err := os.MkdirAll(nrvDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create nrv identity directory: %w", err)
	}
	keyPath := filepath.Join(nrvDir, identityFileName)

	if raw, err := os.ReadFile(keyPath); err == nil {
		if id, idErr := identityFromPrivateKey(raw); idErr == nil {
			return id, nil
		}
		// Corrupt/unexpected key file — regenerate rather than fail the caller.
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read nrv identity: %w", err)
	}

	priv := make([]byte, 32)
	for {
		if _, err := rand.Read(priv); err != nil {
			return nil, fmt.Errorf("failed to generate nrv identity: %w", err)
		}
		id, err := identityFromPrivateKey(priv)
		if err != nil {
			continue // astronomically unlikely zero-scalar draw; retry
		}
		if err := os.WriteFile(keyPath, priv, 0600); err != nil {
			return nil, fmt.Errorf("failed to persist nrv identity: %w", err)
		}
		return id, nil
	}
}

func identityFromPrivateKey(raw []byte) (*Identity, error) {
	if len(raw) != 32 {
		return nil, fmt.Errorf("secp256k1 private key must be 32 bytes")
	}
	key := secp256k1.PrivKeyFromBytes(raw)
	if key.Key.IsZero() {
		return nil, fmt.Errorf("secp256k1 private key cannot be zero")
	}
	pub := key.PubKey().SerializeCompressed()
	address, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return nil, err
	}
	return &Identity{private: append([]byte(nil), raw...), pubKey: pub, address: address}, nil
}

// SignerID returns the KNIRV bech32 address derived from this identity's
// public key — the same address format KNIRVCHAIN, KNIRVGATEWAY, and
// KNIRVORACLE use to identify signers.
func (id *Identity) SignerID() string {
	return id.address
}

// SigningKeyID returns a short fingerprint identifying the key used.
func (id *Identity) SigningKeyID() string {
	fingerprint := hex.EncodeToString(id.pubKey)
	if len(fingerprint) > 16 {
		fingerprint = fingerprint[:16]
	}
	return "secp256k1:" + fingerprint
}

// Sign produces a KNIRVSDK-canonical, domain-separated signature over root
// (an ErrorNodeCommit's ErrorRoot digest) for chainID, returned as a
// base64-encoded JSON knirvsigning.SignedMessage — the same signed-message
// envelope format KNIRVGATEWAY's operator registration and KNIRVORACLE
// verify with knirvsigning.VerifyMessage/VerifyMessagePayload.
func (id *Identity) Sign(chainID, root string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate signing nonce: %w", err)
	}
	now := time.Now()
	envelope := knirvsigning.MessageEnvelope{
		Domain:        nrvSigningDomain,
		Purpose:       nrvSigningPurpose,
		ChainID:       chainID,
		Nonce:         hex.EncodeToString(nonce),
		IssuedAtUnix:  now.Unix(),
		ExpiresAtUnix: now.Add(nrvSignatureValidity).Unix(),
		Payload:       []byte(root),
	}
	signed, err := knirvsigning.SignMessage(id.private, envelope)
	if err != nil {
		return "", fmt.Errorf("failed to sign nrv commit: %w", err)
	}
	wire, err := json.Marshal(signed)
	if err != nil {
		return "", fmt.Errorf("failed to encode signed nrv commit: %w", err)
	}
	return base64.StdEncoding.EncodeToString(wire), nil
}
