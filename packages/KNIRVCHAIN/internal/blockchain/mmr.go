package blockchain

import (
	"crypto/sha256"
	"fmt"
	"math/bits"
)

// MMRLeafKind tags Oracle audit-log leaves.
type MMRLeafKind byte

const (
	MMRLeafCheckpoint MMRLeafKind = 0x01
	MMRLeafFinality   MMRLeafKind = 0x02
	MMRLeafRejection  MMRLeafKind = 0x03

	mmrLeafDomainTag   byte = 0x00
	mmrParentDomainTag byte = 0x01
	mmrEmptyRootTag         = "knirv-mmr-empty"
)

// MMRHash is a SHA-256 digest in an Oracle MMR proof.
type MMRHash [sha256.Size]byte

// MMREmptyRoot returns SHA256("knirv-mmr-empty").
func MMREmptyRoot() MMRHash {
	return sha256.Sum256([]byte(mmrEmptyRootTag))
}

// MMRLeafHash returns SHA256(0x00 || data).
func MMRLeafHash(data []byte) MMRHash {
	preimage := make([]byte, 1, 1+len(data))
	preimage[0] = mmrLeafDomainTag
	preimage = append(preimage, data...)
	return sha256.Sum256(preimage)
}

// MMRParentHash returns SHA256(0x01 || left || right).
func MMRParentHash(left, right MMRHash) MMRHash {
	var preimage [1 + 2*sha256.Size]byte
	preimage[0] = mmrParentDomainTag
	copy(preimage[1:1+sha256.Size], left[:])
	copy(preimage[1+sha256.Size:], right[:])
	return sha256.Sum256(preimage[:])
}

// MMRSide records which side of the current subtree a sibling occupies.
type MMRSide bool

const (
	MMRSideLeft  MMRSide = false
	MMRSideRight MMRSide = true
)

// MMRSibling is one bottom-up proof-fold step.
type MMRSibling struct {
	Hash MMRHash `json:"hash"`
	Side MMRSide `json:"side"`
}

// MMRProof proves one leaf at a particular index and tree size.
type MMRProof struct {
	LeafIndex uint64       `json:"leaf_index"`
	TreeSize  uint64       `json:"tree_size"`
	Path      []MMRSibling `json:"path"`
}

// VerifyMMRProof verifies both the hash fold and the unique sibling-side shape
// implied by LeafIndex and TreeSize. Peak bagging is an exact left fold.
func VerifyMMRProof(expectedRoot, leafHash MMRHash, proof MMRProof, expectedTreeSize uint64) bool {
	if proof.TreeSize != expectedTreeSize || proof.LeafIndex >= proof.TreeSize {
		return false
	}
	expectedSides, ok := mmrProofSides(proof.TreeSize, proof.LeafIndex)
	if !ok || len(expectedSides) != len(proof.Path) {
		return false
	}
	root := leafHash
	for i, sibling := range proof.Path {
		if sibling.Side != expectedSides[i] {
			return false
		}
		if sibling.Side == MMRSideRight {
			root = MMRParentHash(root, sibling.Hash)
		} else {
			root = MMRParentHash(sibling.Hash, root)
		}
	}
	return root == expectedRoot
}

func mmrProofSides(treeSize, leafIndex uint64) ([]MMRSide, bool) {
	if treeSize == 0 || leafIndex >= treeSize {
		return nil, false
	}
	peakIndex := 0
	peakCount := bits.OnesCount64(treeSize)
	var peakHeight uint
	var relative uint64
	found := false
	for height := bits.Len64(treeSize); height > 0; height-- {
		h := uint(height - 1)
		if treeSize&(uint64(1)<<h) == 0 {
			continue
		}
		count := uint64(1) << h
		if leafIndex < count {
			peakHeight, relative, found = h, leafIndex, true
			break
		}
		leafIndex -= count
		peakIndex++
	}
	if !found {
		return nil, false
	}
	sides := make([]MMRSide, 0, int(peakHeight)+peakCount)
	for level := uint(0); level < peakHeight; level++ {
		if relative&(uint64(1)<<level) == 0 {
			sides = append(sides, MMRSideRight)
		} else {
			sides = append(sides, MMRSideLeft)
		}
	}
	if peakIndex > 0 {
		sides = append(sides, MMRSideLeft)
	}
	for later := peakIndex + 1; later < peakCount; later++ {
		sides = append(sides, MMRSideRight)
	}
	return sides, true
}

// MMRHashFromBytes decodes an exact 32-byte MMR digest.
func MMRHashFromBytes(value []byte) (MMRHash, error) {
	if len(value) != sha256.Size {
		return MMRHash{}, fmt.Errorf("mmr: expected %d bytes, got %d", sha256.Size, len(value))
	}
	var hash MMRHash
	copy(hash[:], value)
	return hash, nil
}

// Hex returns the lowercase hexadecimal hash encoding.
func (hash MMRHash) Hex() string {
	const digits = "0123456789abcdef"
	encoded := make([]byte, 2*sha256.Size)
	for i, value := range hash {
		encoded[2*i] = digits[value>>4]
		encoded[2*i+1] = digits[value&0x0f]
	}
	return string(encoded)
}
