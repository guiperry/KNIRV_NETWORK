package authority

import (
	"bytes"
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// sessionInfo is the HKDF info string fixed by AUTHORITY_SPEC.md section 4.1.
const sessionInfo = "KNIRV-ROOT-SESSION-V1"

// secp256k1 group order n. A session scalar must lie in [1, n-1].
var curveOrder, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

// maxScalarAttempts bounds the rejection-sampling loop. The chance that one HKDF
// output falls outside [1, n-1] is about 2^-128, so more than a couple of attempts
// means the derivation is broken, not unlucky.
const maxScalarAttempts = 8

// Session is one ceremony's result: a private key that exists only in this
// process's memory, and the certificate the long-term key signed for it.
type Session struct {
	Signer knirvsigning.RemoteSigner
	Cert   sdk.SessionCert
	Epoch  uint64
	// Kind is the entropy kind written into the certificate.
	Kind string
}

// CeremonyParams are the inputs of one session ceremony.
type CeremonyParams struct {
	LongTerm knirvsigning.RemoteSigner
	Entropy  EntropySource
	Epoch    uint64
	// Scope is the certificate scope: "*" for a root, the node's own address for a boot.
	Scope    string
	Now      time.Time
	TTL      time.Duration
	Monotime int64 // monotonic nanoseconds since process start, mixed into the key material

	// rejectFirst forces that many rejected scalar attempts; tests only.
	rejectFirst int
}

// RunCeremony performs AUTHORITY_SPEC.md section 4.1 steps 2 to 5:
//
//  2. challenge = crypto/rand(32)
//  3. entropy from the source (the OS CSPRNG; ASIC bytes later)
//  4. session_scalar = HKDF-SHA256(ikm = entropy || nonce || device_bits ||
//     monotonic_ns, salt = witness_hash, info = "KNIRV-ROOT-SESSION-V1"),
//     rejection-sampled into [1, n-1]
//  5. the long-term key signs a SessionCert for the new public key
//
// Step 1 (unlocking the long-term key) happened when the signer was built. The
// private scalar is never written anywhere; the returned Session holds it inside
// the signer, in memory. The caller (SessionManager) records only public facts.
//
// PROVISIONAL (not decided by the owner): the spec's salt is the ASIC "witness
// hash". With no ASIC there is none, so the salt here is SHA-256 of the challenge.
// It is a fresh per-ceremony value, which is all the HKDF salt needs to be. The
// ASIC path (Task 7.1) supplies the real witness hash.
func RunCeremony(ctx context.Context, p CeremonyParams) (*Session, error) {
	if p.LongTerm == nil || p.Entropy == nil {
		return nil, errors.New("authority: a ceremony needs a long-term signer and an entropy source")
	}
	if p.Epoch == 0 || p.TTL <= 0 || p.Scope == "" {
		return nil, errors.New("authority: a ceremony needs an epoch, a TTL and a scope")
	}

	var challenge [32]byte
	if _, err := rand.Read(challenge[:]); err != nil {
		return nil, fmt.Errorf("draw ceremony challenge: %w", err)
	}
	drawn, kind, err := p.Entropy.Draw(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("entropy: %w", err)
	}
	if kind != EntropyCSPRNG && kind != EntropyCSPRNGASIC {
		return nil, fmt.Errorf("entropy source reported unknown kind %q", kind)
	}
	if len(drawn) < 32 {
		return nil, fmt.Errorf("entropy source returned %d bytes, need at least 32", len(drawn))
	}

	// ikm = rand || nonce || device_bits || monotonic_ns. nonce and device_bits are
	// the hardware's contribution and are carried inside drawn by an ASIC source;
	// a software source has none.
	var mono [8]byte
	binary.BigEndian.PutUint64(mono[:], uint64(p.Monotime))
	ikm := append(append([]byte(nil), drawn...), mono[:]...)
	salt := sha256.Sum256(challenge[:])

	scalar, err := deriveScalar(ikm, salt[:], p.rejectFirst)
	zero(ikm)
	zero(drawn)
	if err != nil {
		return nil, err
	}
	signer, err := knirvsigning.NewLocalSigner(scalar)
	zero(scalar) // the signer holds its own copy; ours must not linger
	if err != nil {
		return nil, fmt.Errorf("build session signer: %w", err)
	}

	cert := sdk.SessionCert{
		SessionPub: signer.PublicKey(),
		Operator:   p.LongTerm.Address(),
		Epoch:      p.Epoch,
		NotBefore:  p.Now.Unix() - 60, // tolerate a verifier clock slightly behind ours
		NotAfter:   p.Now.Add(p.TTL).Unix(),
		Scope:      p.Scope,
		Entropy:    kind,
	}
	if bytes.Equal(cert.SessionPub, p.LongTerm.PublicKey()) {
		return nil, errors.New("authority: derived session key equals the long-term key")
	}
	signed, err := sdk.SignSessionCert(ctx, cert, p.LongTerm)
	if err != nil {
		return nil, fmt.Errorf("long-term key could not certify the session key: %w", err)
	}
	return &Session{Signer: signer, Cert: signed, Epoch: p.Epoch, Kind: kind}, nil
}

// deriveScalar runs HKDF-SHA256 and rejection-samples the output into [1, n-1].
// The first attempt uses the spec's exact info string; a rejected attempt retries
// with a counter byte appended, so the common path is the specified one.
func deriveScalar(ikm, salt []byte, rejectFirst int) ([]byte, error) {
	for attempt := 0; attempt < maxScalarAttempts; attempt++ {
		info := sessionInfo
		if attempt > 0 {
			info += string([]byte{byte(attempt)})
		}
		out, err := hkdf.Key(sha256.New, ikm, salt, info, 32)
		if err != nil {
			return nil, fmt.Errorf("hkdf: %w", err)
		}
		v := new(big.Int).SetBytes(out)
		if attempt < rejectFirst {
			zero(out)
			continue
		}
		if v.Sign() > 0 && v.Cmp(curveOrder) < 0 {
			return out, nil
		}
		zero(out)
	}
	return nil, errors.New("authority: no valid session scalar after repeated attempts")
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
