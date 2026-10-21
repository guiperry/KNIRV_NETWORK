package evidence

import "crypto/sha256"

// Batch witnessing (plan §5.6): the spans of an ingest batch are hashed into a
// Merkle tree and only the root is mined or recorded. Leaves and inner nodes are
// domain separated so an inner node can never be passed off as a leaf.

func leafHash(d Digest) Digest {
	return sha256.Sum256(append([]byte("fs1:leaf\x00"), d[:]...))
}

func nodeHash(l, r Digest) Digest {
	b := append([]byte("fs1:node\x00"), l[:]...)
	return sha256.Sum256(append(b, r[:]...))
}

// MerkleRoot is the root over ids in order; an odd node is paired with itself.
// The root of nothing is the zero digest.
func MerkleRoot(ids []Digest) Digest {
	if len(ids) == 0 {
		return Digest{}
	}
	level := make([]Digest, len(ids))
	for i, d := range ids {
		level[i] = leafHash(d)
	}
	for len(level) > 1 {
		var next []Digest
		for i := 0; i < len(level); i += 2 {
			j := i + 1
			if j == len(level) {
				j = i
			}
			next = append(next, nodeHash(level[i], level[j]))
		}
		level = next
	}
	return level[0]
}

// ProofStep is one sibling on the path to the root.
type ProofStep struct {
	Sibling Digest
	Left    bool // the sibling is the left child
}

// MerkleProof is the inclusion proof for ids[index].
func MerkleProof(ids []Digest, index int) []ProofStep {
	if index < 0 || index >= len(ids) {
		return nil
	}
	level := make([]Digest, len(ids))
	for i, d := range ids {
		level[i] = leafHash(d)
	}
	var proof []ProofStep
	for len(level) > 1 {
		sib := index ^ 1
		if sib >= len(level) {
			sib = index
		}
		proof = append(proof, ProofStep{Sibling: level[sib], Left: sib < index})
		var next []Digest
		for i := 0; i < len(level); i += 2 {
			j := i + 1
			if j == len(level) {
				j = i
			}
			next = append(next, nodeHash(level[i], level[j]))
		}
		level = next
		index /= 2
	}
	return proof
}

// VerifyProof checks that id is included under root.
func VerifyProof(id Digest, proof []ProofStep, root Digest) bool {
	h := leafHash(id)
	for _, s := range proof {
		if s.Left {
			h = nodeHash(s.Sibling, h)
		} else {
			h = nodeHash(h, s.Sibling)
		}
	}
	return h == root
}
