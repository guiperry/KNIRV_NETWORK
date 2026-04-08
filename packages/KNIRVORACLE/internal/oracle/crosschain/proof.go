package crosschain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
)

// ValidateProof validates a cross-chain transfer proof
func ValidateProof(proof *TransferProof, transfer *CrossChainTransfer) error {
	if proof == nil {
		return fmt.Errorf("proof is nil")
	}

	// Validate Merkle proof
	if err := validateMerkleProof(proof, transfer); err != nil {
		return fmt.Errorf("merkle proof validation failed: %w", err)
	}

	// Validate validator signatures
	if err := validateValidatorSignatures(proof, transfer); err != nil {
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
func validateValidatorSignatures(proof *TransferProof, transfer *CrossChainTransfer) error {
	if len(proof.ValidatorSigs) == 0 {
		return fmt.Errorf("no validator signatures provided")
	}

	// Calculate required voting power (2/3+)
	totalVotingPower := uint64(0)
	for _, sig := range proof.ValidatorSigs {
		totalVotingPower += sig.VotingPower
	}

	requiredVotingPower := (totalVotingPower * 2) / 3

	// Verify signatures
	validVotingPower := uint64(0)
	transferHash := computeTransferHash(transfer)

	for i, sig := range proof.ValidatorSigs {
		// Decode signature
		sigBytes, err := hex.DecodeString(sig.Signature)
		if err != nil {
			return fmt.Errorf("invalid signature at index %d: %w", i, err)
		}

		// Decode validator address
		validatorAddr, err := hex.DecodeString(sig.ValidatorAddress)
		if err != nil {
			return fmt.Errorf("invalid validator address at index %d: %w", i, err)
		}

		// Verify signature (simplified - in production would verify against validator public key)
		if !crypto.VerifySignature(validatorAddr, transferHash, sigBytes) {
			// For now, we'll be lenient and just log the error
			// In production, this should fail validation
			continue
		}

		validVotingPower += sig.VotingPower
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

	// In production, would verify block hash against light client
	// For now, just basic validation
	if len(proof.BlockHash) != 64 && len(proof.BlockHash) != 66 { // 32 bytes hex, with or without 0x
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
	return crypto.Keccak256([]byte(data))
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
