package checkpoint

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
)

type RegisteredAuthor struct {
	Address string `json:"address"`
	PubKey  []byte `json:"pubkey"`
	Weight  uint64 `json:"weight"`
}

// ChainRegistration mirrors the Oracle wire body. RotationSigs also carries
// initial-enrollment authorization so an attacker cannot squat a chain ID.
type ChainRegistration struct {
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

func NewSignedRegistration(chainID string, signers map[string]*ecdsa.PrivateKey) (*ChainRegistration, error) {
	reg := &ChainRegistration{ChainID: chainID, QuorumNumer: 2, QuorumDenom: 3, ProofWindow: 256}
	addresses := make([]string, 0, len(signers))
	for address := range signers {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	for _, address := range addresses {
		key := signers[address]
		if key == nil {
			continue
		}
		pub := key.Public().(*ecdsa.PublicKey)
		canonicalAddress := OracleAddress(pub)
		reg.Authors = append(reg.Authors, RegisteredAuthor{Address: canonicalAddress, PubKey: crypto.CompressPubkey(pub), Weight: 1})
	}
	if len(reg.Authors) == 0 {
		return nil, fmt.Errorf("at least one checkpoint signer is required")
	}
	body, err := registrationBody(reg)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(body)
	for _, address := range addresses {
		key := signers[address]
		if key == nil {
			continue
		}
		sig, err := SignMessage(key, fmt.Sprintf("%x", digest))
		if err != nil {
			return nil, err
		}
		pub := key.Public().(*ecdsa.PublicKey)
		reg.RotationSigs = append(reg.RotationSigs, AuthorSig{Address: OracleAddress(pub), PubKeyHex: hex.EncodeToString(crypto.CompressPubkey(pub)), Signature: sig})
	}
	return reg, nil
}

func registrationBody(reg *ChainRegistration) ([]byte, error) {
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
	return json.Marshal(body{reg.ChainID, reg.Authors, reg.QuorumNumer, reg.QuorumDenom, reg.LastHeight, reg.LastCheckHash, reg.Bond, reg.BondOwner, reg.BondRemaining, reg.BondSlashed, reg.ProofWindow, reg.VerificationKey, reg.PreferredProofSystem})
}
