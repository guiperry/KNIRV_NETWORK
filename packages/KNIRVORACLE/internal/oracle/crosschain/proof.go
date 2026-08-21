package crosschain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// ValidateProof validates a cross-chain transfer proof
func ValidateProof(proof *TransferProof, transfer *CrossChainTransfer, validators []*governance.Validator) error {
	if proof == nil {
		return fmt.Errorf("proof is nil")
	}

	// Validate Merkle proof
	if err := validateMerkleProof(proof, transfer); err != nil {
		return fmt.Errorf("merkle proof validation failed: %w", err)
	}

	// Validate validator signatures
	if err := validateValidatorSignatures(proof, transfer, validators); err != nil {
		return fmt.Errorf("validator signature validation failed: %w", err)
	}

	// Validate block hash
	if err := validateBlockHash(proof); err != nil {
		return fmt.Errorf("block hash validation failed: %w", err)
	}

	return nil
}

// validateMerkleProof validates the Merkle proof
func validateMerkleProof(proof *TransferProof, transfer *CrossChainTransfer) error {
	if proof.MerkleRoot == "" {
		return fmt.Errorf("merkle root is empty")
	}

	if len(proof.MerkleProof) == 0 {
		return fmt.Errorf("merkle proof is empty")
	}

	// Compute transfer hash (leaf)
	transferHash := computeTransferHash(transfer)

	// Verify Merkle proof
	currentHash := transferHash
	for i, proofHash := range proof.MerkleProof {
		// Decode proof hash
		proofBytes, err := hex.DecodeString(proofHash)
		if err != nil {
			return fmt.Errorf("invalid proof hash at index %d: %w", i, err)
		}

		// Combine hashes (order matters for Merkle tree)
		combined := append(currentHash, proofBytes...)
		hash := sha256.Sum256(combined)
		currentHash = hash[:]
	}

	// Compare with Merkle root
	computedRoot := hex.EncodeToString(currentHash)
	if computedRoot != proof.MerkleRoot {
		return fmt.Errorf("merkle root mismatch: expected %s, got %s", proof.MerkleRoot, computedRoot)
	}

	return nil
}

// validateValidatorSignatures validates validator signatures
func validateValidatorSignatures(proof *TransferProof, transfer *CrossChainTransfer, validators []*governance.Validator) error {
	if len(proof.ValidatorSigs) == 0 {
		return fmt.Errorf("no validator signatures provided")
	}

	registered := make(map[types.Address]*governance.Validator, len(validators))
	totalVotingPower := uint64(0)
	for _, validator := range validators {
		if validator == nil || !validator.Active || validator.Jailed || validator.VotingPower == nil || !validator.VotingPower.IsUint64() {
			continue
		}
		power := validator.VotingPower.Uint64()
		if math.MaxUint64-totalVotingPower < power {
			return fmt.Errorf("registered validator voting power overflow")
		}
		registered[validator.Address] = validator
		totalVotingPower += power
	}
	if totalVotingPower == 0 {
		return fmt.Errorf("registered validator set is empty")
	}
	if totalVotingPower > math.MaxUint64/2 {
		return fmt.Errorf("registered validator voting power is too large")
	}
	requiredVotingPower := (totalVotingPower*2)/3 + 1

	// Verify signatures
	validVotingPower := uint64(0)
	transferHash := computeTransferHash(transfer)
	seen := make(map[types.Address]struct{})

	for i, sig := range proof.ValidatorSigs {
		validatorAddr, err := types.AddressFromString(sig.ValidatorAddress)
		if err != nil {
			return fmt.Errorf("invalid validator address at index %d: %w", i, err)
		}
		validator, ok := registered[validatorAddr]
		if !ok {
			return fmt.Errorf("validator %s is not in the active registered set", validatorAddr.String())
		}
		if _, duplicate := seen[validatorAddr]; duplicate {
			return fmt.Errorf("duplicate validator signature from %s", validatorAddr.String())
		}
		seen[validatorAddr] = struct{}{}
		var signed knirvsigning.SignedMessage
		if err := json.Unmarshal([]byte(sig.Signature), &signed); err != nil {
			return fmt.Errorf("decode validator signature at index %d: %w", i, err)
		}
		if signed.Address != validatorAddr.String() {
			return fmt.Errorf("validator signature address mismatch at index %d", i)
		}
		if err := knirvsigning.VerifyMessagePayload(
			signed, "knirv.oracle", "crosschain-proof", transfer.SourceChain.String(), transferHash,
			time.Unix(proof.Timestamp, 0),
		); err != nil {
			return fmt.Errorf("invalid validator signature at index %d: %w", i, err)
		}
		validVotingPower += validator.VotingPower.Uint64()
	}

	// Check if we have enough voting power
	if validVotingPower < requiredVotingPower {
		return fmt.Errorf("insufficient voting power: have %d, need %d", validVotingPower, requiredVotingPower)
	}

	return nil
}

// validateBlockHash validates the block hash
func validateBlockHash(proof *TransferProof) error {
	if proof.BlockHash == "" {
		return fmt.Errorf("block hash is empty")
	}

	if proof.BlockHeight == 0 {
		return fmt.Errorf("block height is zero")
	}

	blockHash := proof.BlockHash
	if len(blockHash) == 66 && blockHash[:2] == "0x" {
		blockHash = blockHash[2:]
	}
	decoded, err := hex.DecodeString(blockHash)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("invalid block hash length")
	}

	return nil
}

// computeTransferHash computes the hash of a transfer
func computeTransferHash(transfer *CrossChainTransfer) []byte {
	data := fmt.Sprintf("%s:%s:%s:%s:%s:%d:%s",
		transfer.TransferID,
		transfer.SourceChain.String(),
		transfer.DestChain.String(),
		transfer.Sender,
		transfer.Recipient,
		transfer.Amount,
		transfer.Denom,
	)
	digest := sha256.Sum256([]byte(data))
	return digest[:]
}

// GenerateProof generates a proof for a transfer (used by validators)
func GenerateProof(transfer *CrossChainTransfer, merkleRoot string, merkleProof []string, blockHeight uint64, blockHash string) *TransferProof {
	return &TransferProof{
		MerkleRoot:    merkleRoot,
		MerkleProof:   merkleProof,
		ValidatorSigs: []ValidatorSignature{}, // Would be filled by validators
		BlockHeight:   blockHeight,
		BlockHash:     blockHash,
		Timestamp:     transfer.CreatedAt,
	}
}

// AddValidatorSignature adds a validator signature to a proof
func AddValidatorSignature(proof *TransferProof, validatorAddr, signature string, votingPower uint64) {
	proof.ValidatorSigs = append(proof.ValidatorSigs, ValidatorSignature{
		ValidatorAddress: validatorAddr,
		Signature:        signature,
		VotingPower:      votingPower,
	})
}
