package types

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/ethereum/go-ethereum/crypto"
)

func (c *Checkpoint) Digest() [32]byte {
	parts := [][]byte{
		[]byte(c.SchemaVersion), {0x00}, []byte(c.ChainID), {0x00},
		u64Bytes(c.StartHeight), u64Bytes(c.EndHeight), c.Root[:], c.PrevCheckHash[:],
		[]byte(c.Proposer), {0x00},
	}
	var data []byte
	for _, part := range parts {
		data = append(data, part...)
	}
	return crypto.Keccak256Hash(data)
}

func u64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (56 - i*8))
	}
	return b
}

func (c *Checkpoint) SignedMessage() string { return fmt.Sprintf("%x", c.Digest()) }

func VerifyAuthorSig(cp *Checkpoint, sig AuthorSig) bool {
	return VerifyAuthorSigFor(cp.Digest(), sig, "chain-checkpoint", cp.ChainID)
}

// VerifyAuthorSigDigest remains for wire callers that predate explicit
// purpose parameters. It still requires a canonical KNIRV envelope and uses
// the purpose/chain encoded into that signed envelope.
func VerifyAuthorSigDigest(digest [32]byte, sig AuthorSig) bool {
	envelope, err := knirvsigning.ParseMessageEnvelope(sig.Envelope)
	if err != nil {
		return false
	}
	return VerifyAuthorSigFor(digest, sig, envelope.Purpose, envelope.ChainID)
}

func VerifyAuthorSigFor(digest [32]byte, sig AuthorSig, purpose, chainID string) bool {
	pub, err := hex.DecodeString(stripHexPrefix(sig.PubKeyHex))
	if err != nil {
		return false
	}
	signed := knirvsigning.SignedMessage{Envelope: sig.Envelope, Signature: sig.Signature, PublicKey: pub, Address: sig.Address}
	return knirvsigning.VerifyMessagePayload(signed, "knirv.chain", purpose, chainID, []byte(fmt.Sprintf("%x", digest)), time.Now()) == nil
}

func OracleAddress(pub *ecdsa.PublicKey) string {
	address, err := knirvsigning.Address(crypto.CompressPubkey(pub), knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	return address
}

func stripHexPrefix(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

func signedDigest(key *ecdsa.PrivateKey, digest [32]byte, purpose, chainID string) (knirvsigning.SignedMessage, error) {
	now := time.Now().Unix()
	return knirvsigning.SignMessage(crypto.FromECDSA(key), knirvsigning.MessageEnvelope{
		Domain: "knirv.chain", Purpose: purpose, ChainID: chainID,
		Nonce: fmt.Sprintf("%x", digest), IssuedAtUnix: now, ExpiresAtUnix: math.MaxInt64,
		Payload: []byte(fmt.Sprintf("%x", digest)),
	})
}

func SignDigest(key *ecdsa.PrivateKey, digest [32]byte, purpose, chainID string) (AuthorSig, error) {
	signed, err := signedDigest(key, digest, purpose, chainID)
	if err != nil {
		return AuthorSig{}, err
	}
	return AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	}, nil
}

func SignCheckpoint(cp *Checkpoint, key *ecdsa.PrivateKey) error {
	signed, err := signedDigest(key, cp.Digest(), "chain-checkpoint", cp.ChainID)
	if err != nil {
		return err
	}
	cp.Signatures = append(cp.Signatures, AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	})
	return nil
}

func SignRegistration(reg *ChainRegistration, key *ecdsa.PrivateKey) error {
	digest, err := ChainRegistrationDigest(reg)
	if err != nil {
		return err
	}
	signed, err := signedDigest(key, digest, "chain-registration", reg.ChainID)
	if err != nil {
		return err
	}
	reg.RotationSigs = append(reg.RotationSigs, AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	})
	return nil
}
