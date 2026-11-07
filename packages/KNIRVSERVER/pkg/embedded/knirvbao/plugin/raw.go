package main

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// rawAllowedNames is the allowlist of key purposes that may be signed through the
// raw (registry-compatible compact) endpoint. Only failover-identity is exposed;
// the OpenBao ACL additionally grants the path only to the backend/failover
// AppRole, so other roles cannot reach it even if this list were widened.
var rawAllowedNames = map[string]bool{"failover-identity": true}

// rawRateLimit caps raw signing requests per minute. The failover control plane
// is low-volume; this bounds the blast radius of a compromised token.
const rawRateLimit = 60

type rawSignRequest struct {
	// Digest is the hex-encoded 32-byte sha256 digest to sign. The caller (the
	// registry client) supplies the digest — typically sha256 of the canonical
	// "|"-joined message — so the plugin never canonicalizes and the signed
	// bytes are byte-identical to the legacy in-process signer.
	Digest string `json:"digest"`
}

type rawSignResponse struct {
	// Signature is the hex 64-byte r||s secp256k1 compact signature (the leading
	// recovery byte is stripped), matching the legacy dcrd
	// SignCompact(priv, digest, true)[1:] form the registry Worker verifies.
	Signature string `json:"signature"`
}

// raw signs a precomputed 32-byte digest with the named identity and returns a
// 64-byte r||s compact secp256k1 signature (recovery byte stripped), byte-for-byte
// identical to the registry's legacy failover signing path. The plugin enforces the
// purpose allowlist and the rate limit; the OpenBao ACL policy is the outer gate.
func (s *signerService) raw(ctx context.Context, storage logical.Storage, name string, req rawSignRequest, now time.Time) (rawSignResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !rawAllowedNames[name] {
		return rawSignResponse{}, errors.New("raw signing not permitted for this key")
	}
	digest, err := hex.DecodeString(strings.TrimSpace(req.Digest))
	if err != nil || len(digest) != 32 {
		return rawSignResponse{}, errors.New("invalid digest")
	}
	if err := consumeRate(ctx, storage, name, rawRateLimit, now); err != nil {
		return rawSignResponse{}, err
	}
	meta, err := readMetadata(ctx, storage, name)
	if err != nil {
		return rawSignResponse{}, err
	}
	key, err := readVersion(ctx, storage, name, meta.Version)
	if err != nil {
		return rawSignResponse{}, err
	}
	if key.Type != meta.Type {
		return rawSignResponse{}, errors.New("stored key type mismatch")
	}
	priv := secp256k1.PrivKeyFromBytes(key.PrivateKey)
	if priv == nil || priv.Key.IsZero() {
		return rawSignResponse{}, errors.New("invalid stored key")
	}
	sig := ecdsa.SignCompact(priv, digest, true)
	return rawSignResponse{Signature: hex.EncodeToString(sig[1:])}, nil
}
