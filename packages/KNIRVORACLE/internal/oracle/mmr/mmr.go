// Package mmr implements an append-only Merkle Mountain Range (MMR) used by the
// Oracle to commit every foreign-chain checkpoint, finality record, and rejection
// tombstone into a single auditable log.
//
// Hash convention (domain separated to prevent second-preimage attacks):
//
//	leaf node:  SHA256(0x00 || data)
//	parent node: SHA256(0x01 || left || right)
//
// The bagged root folds all current peak roots left→right. An empty MMR roots to
// SHA256("knirv-mmr-empty").
//
// Internal layout: the forest is a list of perfect binary trees ("peaks"). Each
// peak of height H (2^H leaves) is stored as a complete binary tree in a flat,
// subtree-contiguous array (root, then all left-subtree nodes, then all
// right-subtree nodes). Appending a leaf starts a height-0 peak and repeatedly
// merges equal-height trailing peaks. This makes inclusion-proof generation a
// direct subtree walk plus the other peak roots, without fragile global position
// arithmetic.
//
// This file is the canonical owner. Identical copies (same hashing, same root,
// same proof shape, same golden vectors) are vendored into KNIRVCHAIN and the
// backend verifier package and a Rust verify-only copy, so light clients and the
// browser verifier can independently verify inclusion proofs without trusting the
// server (see merkle-math.md §3.1).
package mmr

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

// LeafKind tags every leaf so the log is self-describing.
type LeafKind byte

const (
	LeafCheckpoint LeafKind = 0x01 // phase-1 provisional checkpoint
	LeafFinality   LeafKind = 0x02 // phase-2 finality record
	LeafRejection  LeafKind = 0x03 // window-miss / proof-fail tombstone
)

const (
	leafDomainTag   byte = 0x00
	parentDomainTag byte = 0x01

	emptyRootTag = "knirv-mmr-empty"
)

// ErrIndexOutOfRange is returned when a leaf index is not yet present in the MMR.
var ErrIndexOutOfRange = errors.New("mmr: leaf index out of range")

// Hash is a 32-byte SHA-256 digest.
type Hash [32]byte

// EmptyRoot is the canonical root of an empty MMR.
func EmptyRoot() Hash {
	return sha256.Sum256([]byte(emptyRootTag))
}

// LeafHash computes the domain-separated leaf hash of raw data.
func LeafHash(data []byte) Hash {
	return sha256.Sum256(append([]byte{leafDomainTag}, data...))
}

// ParentHash computes the domain-separated parent hash of two child nodes.
func ParentHash(left, right Hash) Hash {
	buf := make([]byte, 1+2*sha256.Size)
	buf[0] = parentDomainTag
	copy(buf[1:1+sha256.Size], left[:])
	copy(buf[1+sha256.Size:], right[:])
	return sha256.Sum256(buf)
}

// peak is a single perfect binary tree in the forest. nodes has length
// 2^(height+1)-1 and is stored root-first with each subtree contiguous.
type peak struct {
	height uint32
	nodes  []Hash
}

// leafIndex is the next leaf index that will be assigned and treeSize is the
// total number of leaves across all peaks.
type MMR struct {
	peaks    []peak
	nextLeaf uint64
}

// New returns an empty MMR.
func New() *MMR {
	return &MMR{peaks: make([]peak, 0), nextLeaf: 0}
}

// Size returns the number of leaf nodes in the MMR.
func (m *MMR) Size() uint64 {
	return m.nextLeaf
}

// Add appends a new leaf from raw data and returns its leaf index (0-based) and
// the resulting bagged root.
func (m *MMR) Add(data []byte) (uint64, Hash, error) {
	idx, root := m.AddRaw(LeafHash(data))
	return idx, root, nil
}

// AddRaw appends a pre-hashed leaf digest (output of LeafHash) and returns its
// leaf index and the resulting bagged root.
func (m *MMR) AddRaw(leaf Hash) (uint64, Hash) {
	idx := m.nextLeaf
	m.nextLeaf++

	// Start a new height-0 peak (a single leaf node).
	newPeak := peak{height: 0, nodes: make([]Hash, 1)}
	newPeak.nodes[0] = leaf
	m.peaks = append(m.peaks, newPeak)

	// Merge trailing peaks of equal height into one peak of height+1. The heap
	// layout of the merged tree is [ParentHash(L,R), leftNodes..., rightNodes...],
	// which keeps every peak a complete binary tree and makes the forest shape
	// exactly match the binary decomposition of the leaf count.
	for len(m.peaks) >= 2 {
		a := m.peaks[len(m.peaks)-2]
		b := m.peaks[len(m.peaks)-1]
		if a.height != b.height {
			break
		}
		merged := peak{
			height: a.height + 1,
			nodes:  make([]Hash, 0, (uint64(1)<<(a.height+2))-1),
		}
		merged.nodes = append(merged.nodes, ParentHash(a.root(), b.root()))
		merged.nodes = append(merged.nodes, a.nodes...)
		merged.nodes = append(merged.nodes, b.nodes...)
		m.peaks = m.peaks[:len(m.peaks)-2]
		m.peaks = append(m.peaks, merged)
	}

	return idx, m.BagRoot()
}

// leafCount returns the number of leaves currently in the peak.
func (p *peak) leafCount() uint64 {
	// number of leaves = (nodeCount + 1) / 2
	return (uint64(len(p.nodes)) + 1) / 2
}

// root returns the peak's root (node 0).
func (p *peak) root() Hash {
	return p.nodes[0]
}

// BagRoot folds all current peak roots left→right. An empty MMR returns
// EmptyRoot().
func (m *MMR) BagRoot() Hash {
	if len(m.peaks) == 0 {
		return EmptyRoot()
	}
	root := m.peaks[0].root()
	for i := 1; i < len(m.peaks); i++ {
		root = ParentHash(root, m.peaks[i].root())
	}
	return root
}

// PeakRoots returns the roots of all peaks in order (useful for debugging and
// for the verifier to understand the forest shape).
func (m *MMR) PeakRoots() []Hash {
	out := make([]Hash, len(m.peaks))
	for i, p := range m.peaks {
		out[i] = p.root()
	}
	return out
}

// GetLeaf returns the stored leaf digest at the given leaf index.
func (m *MMR) GetLeaf(leafIndex uint64) (Hash, error) {
	if leafIndex >= m.nextLeaf {
		return Hash{}, ErrIndexOutOfRange
	}
	pk, rel, err := m.locate(leafIndex)
	if err != nil {
		return Hash{}, err
	}
	return m.peaks[pk].nodes[heapIndexOfLeaf(m.peaks[pk].height, rel)], nil
}

// locate returns the peak index and the relative leaf offset within that peak for
// a global leaf index.
func (m *MMR) locate(leafIndex uint64) (int, uint64, error) {
	seen := uint64(0)
	for i, p := range m.peaks {
		c := p.leafCount()
		if leafIndex < seen+c {
			return i, leafIndex - seen, nil
		}
		seen += c
	}
	return 0, 0, ErrIndexOutOfRange
}

// heapIndexOfLeaf returns the position of the leaf at relative offset rel within a
// peak of the given height, in the merged subtree-contiguous layout (root at 0, left
// subtree at relative 1, right subtree at relative 1+size(left)). This matches how
// AddRaw stores merged peaks and how GenerateProof/GetLeaf address nodes.
func heapIndexOfLeaf(height uint32, rel uint64) uint64 {
	if height == 0 {
		return 0
	}
	half := uint64(1) << (height - 1)
	if rel < half {
		return 1 + heapIndexOfLeaf(height-1, rel)
	}
	leftSize := (uint64(1) << height) - 1
	return 1 + leftSize + heapIndexOfLeaf(height-1, rel-half)
}

// Side indicates whether a sibling subtree sits to the left or right of the
// current subtree during folding. H is non-associative, so the side must be
// recorded explicitly.
type Side bool

const (
	// SideLeft means the sibling is on the left; the current value is the right
	// child, so the parent is H(sibling, current).
	SideLeft Side = false
	// SideRight means the sibling is on the right; the current value is the left
	// child, so the parent is H(current, sibling).
	SideRight Side = true
)

// Sibling is one step in an inclusion-proof fold path.
type Sibling struct {
	Hash Hash `json:"hash"`
	Side Side `json:"side"`
}

// Proof is an inclusion proof for a single leaf. Path is the ordered list of
// sibling subtree roots to fold with the leaf's subtree, bottom-up: first the
// within-peak siblings from the leaf to its peak root, then the bagging-tree
// siblings from the peak to the MMR root. LeafIndex and TreeSize let a verifier
// confirm the folding without any Oracle state.
type Proof struct {
	LeafIndex uint64    `json:"leaf_index"`
	TreeSize  uint64    `json:"tree_size"`
	Path      []Sibling `json:"path"`
}

// GenerateProof returns an inclusion proof for the leaf at leafIndex.
func (m *MMR) GenerateProof(leafIndex uint64) (Proof, error) {
	if leafIndex >= m.nextLeaf {
		return Proof{}, ErrIndexOutOfRange
	}
	treeSize := m.nextLeaf
	pk, rel, err := m.locate(leafIndex)
	if err != nil {
		return Proof{}, err
	}

	p := &m.peaks[pk]
	path := make([]Sibling, 0)

	// Within-peak path: walk from the leaf up to the peak root. The merged
	// storage layout is subtree-contiguous (root at 0, left subtree at 1..,
	// right subtree at 1+size(left)..), NOT a standard 2i+1/2i+2 heap. A node's
	// children are therefore at base+1 (left) and base+1+size(subtree) (right).
	// We descend to the leaf first, then record siblings bottom-up so the fold
	// matches VerifyProof's order.
	var walk func(base uint64, h uint32, r uint64)
	walk = func(base uint64, h uint32, r uint64) {
		if h == 0 {
			return
		}
		half := uint64(1) << (h - 1)
		leftRoot := p.nodes[base+1]
		rightBase := base + 1 + ((uint64(1) << h) - 1)
		rightRoot := p.nodes[rightBase]
		if r < half {
			walk(base+1, h-1, r)
			path = append(path, Sibling{Hash: rightRoot, Side: SideRight})
		} else {
			walk(rightBase, h-1, r-half)
			path = append(path, Sibling{Hash: leftRoot, Side: SideLeft})
		}
	}
	walk(0, p.height, rel)

	// Bagging-tree path: walk from the leaf's peak up to the MMR root, collecting
	// the sibling subtree at each internal node of the left-associative bagging
	// tree (BagRoot). This mirrors exactly how BagRoot folds peaks.
	path = append(path, m.baggingPath(pk)...)

	return Proof{LeafIndex: leafIndex, TreeSize: treeSize, Path: path}, nil
}

// baggingPath returns the sibling subtrees on the path from peak pk up to the
// bagged root, in leaf-to-root order. It mirrors exactly how BagRoot folds peaks:
// a left-associative fold result = H(H(...H(H(P0,P1),P2)...,P_{k-1}),Pk).
//
// For a leaf in peak pk the fold path is:
//  1. if pk > 0, the bag of peaks P0..P_{pk-1} is the LEFT sibling of Pk
//     (Pk is the right child), giving root = H(bag(P0..P_{pk-1}), Pk).
//  2. for each later peak Pj (j = pk+1..k), Pj is the RIGHT sibling, so
//     root = H(current, Pj).
func (m *MMR) baggingPath(pk int) []Sibling {
	k := len(m.peaks) - 1
	if k < 0 {
		return nil
	}
	// acc[i] = left-fold of peaks P0..Pi = bag(P0..Pi).
	acc := make([]Hash, len(m.peaks))
	acc[0] = m.peaks[0].root()
	for i := 1; i <= k; i++ {
		acc[i] = ParentHash(acc[i-1], m.peaks[i].root())
	}

	out := make([]Sibling, 0)
	if pk > 0 {
		// bag(P0..P_{pk-1}) = acc[pk-1], on the left.
		out = append(out, Sibling{Hash: acc[pk-1], Side: SideLeft})
	}
	for j := pk + 1; j <= k; j++ {
		out = append(out, Sibling{Hash: m.peaks[j].root(), Side: SideRight})
	}
	return out
}

// VerifyProof recomputes the bagged root from a leaf hash and its inclusion
// proof, then compares against expectedRoot. It is a pure function with no MMR
// state — suitable for light clients and the browser verifier.
func VerifyProof(expectedRoot, leafHash Hash, proof Proof, expectedTreeSize uint64) bool {
	if proof.TreeSize != expectedTreeSize {
		return false
	}
	if proof.LeafIndex >= proof.TreeSize {
		return false
	}
	expectedSides, ok := proofSides(proof.TreeSize, proof.LeafIndex)
	if !ok || len(expectedSides) != len(proof.Path) {
		return false
	}
	root := leafHash
	for i, s := range proof.Path {
		// TreeSize and LeafIndex imply one unique peak and leaf position. This
		// prevents a valid path from being relabelled as another leaf index.
		if s.Side != expectedSides[i] {
			return false
		}
		if s.Side == SideRight {
			root = ParentHash(root, s.Hash)
		} else {
			root = ParentHash(s.Hash, root)
		}
	}
	return root == expectedRoot
}

// proofSides returns the unique sibling-side sequence implied by treeSize and
// leafIndex: within-peak levels bottom-up, followed by left-fold peak bagging.
// Peaks are the set bits of treeSize ordered from highest to lowest.
func proofSides(treeSize, leafIndex uint64) ([]Side, bool) {
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
			peakHeight = h
			relative = leafIndex
			found = true
			break
		}
		leafIndex -= count
		peakIndex++
	}
	if !found {
		return nil, false
	}

	sides := make([]Side, 0, int(peakHeight)+peakCount)
	for level := uint(0); level < peakHeight; level++ {
		if relative&(uint64(1)<<level) == 0 {
			sides = append(sides, SideRight)
		} else {
			sides = append(sides, SideLeft)
		}
	}
	if peakIndex > 0 {
		sides = append(sides, SideLeft)
	}
	for later := peakIndex + 1; later < peakCount; later++ {
		sides = append(sides, SideRight)
	}
	return sides, true
}

// FromLeaves reconstructs an MMR from a sequence of already-hashed leaf digests
// (the values returned by LeafHash). It is the recovery path used when replaying
// a persisted leaf log.
func FromLeaves(leaves []Hash) *MMR {
	m := New()
	for _, l := range leaves {
		m.AddRaw(l)
	}
	return m
}

// Nodes returns a flat serialization of every peak's node array, concatenated in
// peak order. Used for persistence.
func (m *MMR) Nodes() []Hash {
	total := 0
	for _, p := range m.peaks {
		total += len(p.nodes)
	}
	out := make([]Hash, 0, total)
	for _, p := range m.peaks {
		out = append(out, p.nodes...)
	}
	return out
}

// Leaves returns the leaf digests in append order. Each peak of height h is a
// complete binary tree whose deepest level holds 2^h leaves, which occupy the
// trailing 2^h entries of its node array (layout: [root, left-subtree…,
// right-subtree…]). Concatenating those trailing slices across peaks yields the
// leaf log, which is what must be persisted for append-only recovery.
func (m *MMR) Leaves() []Hash {
	out := make([]Hash, 0, m.nextLeaf)
	for _, p := range m.peaks {
		leafCount := uint64(1) << p.height
		start := uint64(len(p.nodes)) - leafCount
		out = append(out, p.nodes[start:start+leafCount]...)
	}
	return out
}

// FromNodes reconstructs an MMR from a Nodes() dump. Peak boundaries are
// implicit in the complete-tree sizes 1, 3, 7, 15, ...; greedily taking the
// largest complete tree produces the unique descending-height peak sequence.
func FromNodes(nodes []Hash) (*MMR, error) {
	m := New()
	i := 0
	for i < len(nodes) {
		// Determine the size of the next complete peak's node array.
		size := nextPeakSize(uint64(len(nodes) - i))
		if size == 0 {
			return nil, errors.New("mmr: malformed node stream")
		}
		peakNodes := make([]Hash, size)
		copy(peakNodes, nodes[i:i+int(size)])
		height := peakHeightFromNodes(size)
		m.peaks = append(m.peaks, peak{height: height, nodes: peakNodes})
		m.nextLeaf += (size + 1) / 2
		i += int(size)
	}
	return m, nil
}

// nextPeakSize returns the size (number of nodes) of the next peak given the
// remaining node count. After the merge-based append, every peak is a complete
// binary tree of size 2^(h+1)-1, and an MMR forest can be uniquely split by
// greedily taking the largest complete tree that fits the remainder.
func nextPeakSize(remaining uint64) uint64 {
	if remaining == 0 {
		return 0
	}
	h := uint32(0)
	for {
		full := (uint64(1) << (h + 1)) - 1
		if full > remaining {
			// Largest complete tree that fits is the previous one.
			return (uint64(1) << h) - 1
		}
		if full == remaining {
			return full
		}
		h++
	}
}

// peakHeightFromNodes derives the peak height from the number of nodes in a
// complete binary tree: nodes = 2^(h+1)-1.
func peakHeightFromNodes(size uint64) uint32 {
	h := uint32(0)
	for {
		full := (uint64(1) << (h + 1)) - 1
		if full >= size {
			return h
		}
		h++
	}
}

// MarshalBinary serializes the MMR for append-only persistence (mmr_nodes.bin).
// Layout: 4-byte peak count (LE) + per peak: 4-byte height, 4-byte node count,
// nodeCount*32 bytes.
func (m *MMR) MarshalBinary() []byte {
	buf := make([]byte, 0, 4+len(m.peaks)*12)
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(m.peaks)))
	buf = append(buf, header[:]...)
	for _, p := range m.peaks {
		var ph [4]byte
		var pn [4]byte
		binary.LittleEndian.PutUint32(ph[:], p.height)
		binary.LittleEndian.PutUint32(pn[:], uint32(len(p.nodes)))
		buf = append(buf, ph[:]...)
		buf = append(buf, pn[:]...)
		for _, n := range p.nodes {
			buf = append(buf, n[:]...)
		}
	}
	return buf
}

// UnmarshalBinary restores an MMR from MarshalBinary output.
func UnmarshalBinary(data []byte) (*MMR, error) {
	if len(data) < 4 {
		return nil, errors.New("mmr: truncated header")
	}
	count := binary.LittleEndian.Uint32(data[0:4])
	off := 4
	m := New()
	var previousHeight uint32
	for i := uint32(0); i < count; i++ {
		if off+8 > len(data) {
			return nil, errors.New("mmr: truncated peak header")
		}
		height := binary.LittleEndian.Uint32(data[off : off+4])
		ncount := binary.LittleEndian.Uint32(data[off+4 : off+8])
		off += 8
		if height >= 63 {
			return nil, errors.New("mmr: invalid peak height")
		}
		expectedNodes := (uint64(1) << (height + 1)) - 1
		if uint64(ncount) != expectedNodes {
			return nil, errors.New("mmr: invalid peak node count")
		}
		if i > 0 && height >= previousHeight {
			return nil, errors.New("mmr: peaks not in descending height order")
		}
		if uint64(ncount) > uint64((len(data)-off)/sha256.Size) {
			return nil, errors.New("mmr: truncated peak nodes")
		}
		nodes := make([]Hash, ncount)
		for j := uint32(0); j < ncount; j++ {
			copy(nodes[j][:], data[off:off+sha256.Size])
			off += sha256.Size
		}
		m.peaks = append(m.peaks, peak{height: height, nodes: nodes})
		m.nextLeaf += (uint64(ncount) + 1) / 2
		previousHeight = height
	}
	if off != len(data) {
		return nil, errors.New("mmr: trailing data")
	}
	return m, nil
}

// Version marks the persisted format; bump on incompatible changes.
func (m *MMR) Version() uint32 { return 1 }

// Hex returns the lowercase hex encoding of the hash for logging and API display.
func (h Hash) Hex() string {
	const hextable = "0123456789abcdef"
	out := make([]byte, 64)
	for i, b := range h {
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return string(out)
}

// HashFromBytes builds a Hash from a 32-byte slice, returning an error if the
// length is wrong. Used when decoding wire payloads.
func HashFromBytes(b []byte) (Hash, error) {
	if len(b) != sha256.Size {
		return Hash{}, fmt.Errorf("mmr: expected %d bytes, got %d", sha256.Size, len(b))
	}
	var h Hash
	copy(h[:], b)
	return h, nil
}
