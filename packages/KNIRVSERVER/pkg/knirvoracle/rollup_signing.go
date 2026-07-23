package knirvoracle

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

// AuthorSig is a single author's signature over a rollup submission digest,
// wire-compatible with KNIRVORACLE's types.AuthorSig.
type AuthorSig struct {
	Address   string `json:"address"`
	PubKeyHex string `json:"pubkey_hex"`
	Signature []byte `json:"signature"`
}

// RegisteredAuthor is one entry in a chain's registered author set.
type RegisteredAuthor struct {
	Address string `json:"address"`
	PubKey  []byte `json:"pubkey"`
	Weight  uint64 `json:"weight"`
}

// chainRegistration is the payload POSTed to /oracle/v3/registry/register.
// Field set mirrors KNIRVORACLE's types.ChainRegistration exactly (including
// fields this client never sets) — the server recomputes the registration
// digest from the deserialized struct, so registrationBody below must
// construct the identical zero-valued body the server will see.
type chainRegistration struct {
	ChainID              string             `json:"chain_id"`
	Authors              []RegisteredAuthor `json:"authors"`
	QuorumNumer          uint64             `json:"quorum_numer"`
	QuorumDenom          uint64             `json:"quorum_denom"`
	LastHeight           uint64             `json:"last_height"`
	LastCheckHash        [32]byte           `json:"last_check_hash"`
	Bond                 uint64             `json:"bond"`
	BondOwner            string             `json:"bond_owner,omitempty"`
	BondRemaining        uint64             `json:"bond_remaining,omitempty"`
	BondSlashed          uint64             `json:"bond_slashed,omitempty"`
	ProofWindow          uint64             `json:"proof_window"`
	VerificationKey      []byte             `json:"verification_key,omitempty"`
	PreferredProofSystem string             `json:"preferred_proof_system,omitempty"`
	RotationSigs         []AuthorSig        `json:"rotation_sigs,omitempty"`
}

func oracleAddress(pub *ecdsa.PublicKey) string {
	unc := crypto.FromECDSAPub(pub)
	sum := crypto.Keccak256(unc[1:])
	return "0x" + hex.EncodeToString(sum[12:])
}

func signMessage(key *ecdsa.PrivateKey, msg string) ([]byte, error) {
	hash := crypto.Keccak256([]byte(msg))
	full, err := crypto.Sign(hash, key) // 65 bytes: r||s||v
	if err != nil {
		return nil, err
	}
	return full[:64], nil
}

// Digest returns the canonical 32-byte hash over the rollup submission body
// that Proposer signs. Must stay byte-identical to KNIRVORACLE's
// types.RollupRecord.Digest.
func (rec *RollupRecord) Digest() [32]byte {
	parts := [][]byte{
		[]byte(rec.ID), {0x00},
		[]byte(rec.BatchRoot), {0x00},
		[]byte(rec.ChainID), {0x00},
		u64Bytes(uint64(rec.StartHeight)),
		u64Bytes(uint64(rec.EndHeight)),
		u64Bytes(uint64(rec.BlockCount)),
		u64Bytes(uint64(rec.TxCount)),
		[]byte(rec.Proposer), {0x00},
	}
	var data []byte
	for _, p := range parts {
		data = append(data, p...)
	}
	var out [32]byte
	copy(out[:], crypto.Keccak256(data))
	return out
}

func u64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	for i := range 8 {
		b[i] = byte(v >> (56 - i*8))
	}
	return b
}

func registrationBody(c *chainRegistration) ([]byte, error) {
	type body struct {
		ChainID              string             `json:"chain_id"`
		Authors              []RegisteredAuthor `json:"authors"`
		QuorumNumer          uint64             `json:"quorum_numer"`
		QuorumDenom          uint64             `json:"quorum_denom"`
		LastHeight           uint64             `json:"last_height"`
		LastCheckHash        [32]byte           `json:"last_check_hash"`
		Bond                 uint64             `json:"bond"`
		BondOwner            string             `json:"bond_owner,omitempty"`
		BondRemaining        uint64             `json:"bond_remaining,omitempty"`
		BondSlashed          uint64             `json:"bond_slashed,omitempty"`
		ProofWindow          uint64             `json:"proof_window"`
		VerificationKey      []byte             `json:"verification_key,omitempty"`
		PreferredProofSystem string             `json:"preferred_proof_system,omitempty"`
	}
	return json.Marshal(body{
		ChainID: c.ChainID, Authors: c.Authors, QuorumNumer: c.QuorumNumer,
		QuorumDenom: c.QuorumDenom, LastHeight: c.LastHeight, LastCheckHash: c.LastCheckHash,
		Bond: c.Bond, BondOwner: c.BondOwner, BondRemaining: c.BondRemaining,
		BondSlashed: c.BondSlashed, ProofWindow: c.ProofWindow,
		VerificationKey: c.VerificationKey, PreferredProofSystem: c.PreferredProofSystem,
	})
}

var (
	rollupSigner     *ecdsa.PrivateKey
	rollupSignerOnce sync.Once
	rollupSignerErr  error

	registeredRollupChains sync.Map // chainID -> struct{}
)

const rollupSignerFilename = "rollup-submitter-signer.key"

// loadOrCreateRollupSigner returns the secp256k1 identity this KNIRVSERVER
// instance uses to register with and sign rollup submissions to KNIRVORACLE
// on behalf of the embedded Transaction Chain. Persisted once under the app
// data dir so the identity is stable across restarts.
func loadOrCreateRollupSigner() (*ecdsa.PrivateKey, error) {
	rollupSignerOnce.Do(func() {
		dir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR"))
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				rollupSignerErr = fmt.Errorf("resolve key directory: %w", err)
				return
			}
			dir = home
		}
		keyPath := filepath.Join(dir, rollupSignerFilename)

		if data, err := os.ReadFile(keyPath); err == nil {
			key, err := crypto.HexToECDSA(strings.TrimSpace(string(data)))
			if err != nil {
				rollupSignerErr = fmt.Errorf("parse persisted rollup signer at %s: %w", keyPath, err)
				return
			}
			rollupSigner = key
			return
		}

		key, err := crypto.GenerateKey()
		if err != nil {
			rollupSignerErr = fmt.Errorf("generate rollup signer: %w", err)
			return
		}
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			rollupSignerErr = fmt.Errorf("create key directory: %w", err)
			return
		}
		hexKey := fmt.Sprintf("%x", crypto.FromECDSA(key))
		if err := os.WriteFile(keyPath, []byte(hexKey), 0o600); err != nil {
			rollupSignerErr = fmt.Errorf("persist rollup signer: %w", err)
			return
		}
		rollupSigner = key
	})
	return rollupSigner, rollupSignerErr
}

// signRollup attaches Proposer/Signatures to record using this instance's
// rollup signer identity.
func signRollup(record *RollupRecord) error {
	key, err := loadOrCreateRollupSigner()
	if err != nil {
		return err
	}
	pub := key.Public().(*ecdsa.PublicKey)
	record.Proposer = oracleAddress(pub)
	sig, err := signMessage(key, fmt.Sprintf("%x", record.Digest()))
	if err != nil {
		return err
	}
	record.Signatures = []AuthorSig{{
		Address:   record.Proposer,
		PubKeyHex: hex.EncodeToString(crypto.CompressPubkey(pub)),
		Signature: sig,
	}}
	return nil
}

// ensureRollupChainRegistered registers chainID with KNIRVORACLE using this
// instance's rollup signer as the sole author (quorum 1-of-1), tolerating
// "already registered". Cached per-process so repeated submissions for the
// same chain don't re-register every time.
func (c *Client) ensureRollupChainRegistered(chainID string) error {
	if _, ok := registeredRollupChains.Load(chainID); ok {
		return nil
	}
	key, err := loadOrCreateRollupSigner()
	if err != nil {
		return err
	}
	pub := key.Public().(*ecdsa.PublicKey)
	address := oracleAddress(pub)
	reg := &chainRegistration{
		ChainID: chainID,
		Authors: []RegisteredAuthor{
			{Address: address, PubKey: crypto.CompressPubkey(pub), Weight: 1},
		},
		QuorumNumer: 1,
		QuorumDenom: 1,
	}
	body, err := registrationBody(reg)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(body)
	sig, err := signMessage(key, fmt.Sprintf("%x", digest))
	if err != nil {
		return err
	}
	reg.RotationSigs = []AuthorSig{{
		Address:   address,
		PubKeyHex: hex.EncodeToString(crypto.CompressPubkey(pub)),
		Signature: sig,
	}}

	if err := c.post("/oracle/v3/registry/register", reg, nil); err != nil {
		if strings.Contains(err.Error(), "already registered") {
			registeredRollupChains.Store(chainID, struct{}{})
			return nil
		}
		return err
	}
	registeredRollupChains.Store(chainID, struct{}{})
	return nil
}
