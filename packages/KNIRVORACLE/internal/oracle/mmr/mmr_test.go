package mmr

import (
	"encoding/binary"
	"testing"
)

// goldenLeaves are fixed inputs. Their leaf hashes, indices, root, and proofs are
// asserted against hardcoded expected values so that any drift in the hashing or
// bagging algorithm (and therefore in the vendored copies) fails loudly.
var goldenLeaves = [][]byte{
	[]byte("checkpoint:knirvchain-1:0-63"),
	[]byte("checkpoint:knirvchain-1:64-127"),
	[]byte("finality:knirvchain-1:leaf-0"),
	[]byte("rejection:knirvchain-1:leaf-1"),
	[]byte("checkpoint:knirvchain-2:0-63"),
	[]byte("checkpoint:knirvchain-2:64-127"),
	[]byte("finality:knirvchain-2:leaf-0"),
}

// expectedGoldenRoot is the SHA-256-based bagged root of the golden leaves,
// computed by this implementation at test-authoring time and frozen here.
// Regenerate only when the spec intentionally changes.
const expectedGoldenRoot = "91e49d93eb68d6e486401d1d4fa7b1dd46760d768e082c788b55d3f8a6ce45f7"

func TestLeafParentDomainSeparation(t *testing.T) {
	leaf := LeafHash([]byte("x"))
	parent := ParentHash(leaf, leaf)
	// A leaf hashed then wrapped as a parent must not collide with a leaf hash,
	// because the domain tags (0x00 vs 0x01) differ in the preimage.
	if leaf == parent {
		t.Fatal("domain separation failed: leaf == parent")
	}
	if LeafHash([]byte("x")) != LeafHash([]byte("x")) {
		t.Fatal("LeafHash not deterministic")
	}
	if LeafHash([]byte("x")) == LeafHash([]byte("y")) {
		t.Fatal("LeafHash collided on distinct inputs")
	}
}

func TestEmptyRootIsStable(t *testing.T) {
	if EmptyRoot() != EmptyRoot() {
		t.Fatal("EmptyRoot not stable")
	}
	m := New()
	if m.BagRoot() != EmptyRoot() {
		t.Fatalf("empty MMR root mismatch: got %s want %s", m.BagRoot().Hex(), EmptyRoot().Hex())
	}
	if m.Size() != 0 {
		t.Fatalf("empty MMR size = %d, want 0", m.Size())
	}
}

func TestAppendAndCount(t *testing.T) {
	m := New()
	for i, l := range goldenLeaves {
		idx, root, err := m.Add(l)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if idx != uint64(i) {
			t.Fatalf("leaf index = %d, want %d", idx, i)
		}
		if root == EmptyRoot() && i > 0 {
			t.Fatalf("non-empty MMR returned empty root at i=%d", i)
		}
	}
	if m.Size() != uint64(len(goldenLeaves)) {
		t.Fatalf("size = %d, want %d", m.Size(), len(goldenLeaves))
	}
}

func TestGoldenRootMatchesFrozenValue(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		if _, _, err := m.Add(l); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	got := m.BagRoot().Hex()
	if got != expectedGoldenRoot {
		t.Fatalf("golden root drift:\n got  %s\n want %s", got, expectedGoldenRoot)
	}
}

func TestInclusionProofRoundTripEveryLeaf(t *testing.T) {
	m := New()
	hashes := make([]Hash, 0, len(goldenLeaves))
	for _, l := range goldenLeaves {
		if _, _, err := m.Add(l); err != nil {
			t.Fatalf("Add: %v", err)
		}
		hashes = append(hashes, LeafHash(l))
	}
	root := m.BagRoot()
	treeSize := m.Size()
	for i := uint64(0); i < treeSize; i++ {
		proof, err := m.GenerateProof(i)
		if err != nil {
			t.Fatalf("generate proof for leaf %d: %v", i, err)
		}
		if proof.TreeSize != treeSize {
			t.Fatalf("leaf %d: TreeSize = %d, want %d", i, proof.TreeSize, treeSize)
		}
		if !VerifyProof(root, hashes[i], proof, treeSize) {
			t.Fatalf("verify proof failed for leaf %d", i)
		}
	}
}

func TestVerifyProofRejectsTamperedLeaf(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	root := m.BagRoot()
	treeSize := m.Size()
	proof, err := m.GenerateProof(0)
	if err != nil {
		t.Fatal(err)
	}
	tampered := LeafHash([]byte("evil"))
	if VerifyProof(root, tampered, proof, treeSize) {
		t.Fatal("VerifyProof accepted a tampered leaf")
	}
}

func TestVerifyProofRejectsWrongTreeSize(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	root := m.BagRoot()
	proof, _ := m.GenerateProof(2)
	if VerifyProof(root, LeafHash(goldenLeaves[2]), proof, m.Size()+1) {
		t.Fatal("VerifyProof accepted mismatched tree size")
	}
}

func TestVerifyProofRejectsRelabelledIndex(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	proof, err := m.GenerateProof(0)
	if err != nil {
		t.Fatal(err)
	}
	proof.LeafIndex = 1
	if VerifyProof(m.BagRoot(), LeafHash(goldenLeaves[0]), proof, m.Size()) {
		t.Fatal("VerifyProof accepted a proof relabelled with another leaf index")
	}
}

func TestVerifyProofRejectsWrongSide(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	proof, err := m.GenerateProof(0)
	if err != nil {
		t.Fatal(err)
	}
	proof.Path[0].Side = SideLeft
	if VerifyProof(m.BagRoot(), LeafHash(goldenLeaves[0]), proof, m.Size()) {
		t.Fatal("VerifyProof accepted an invalid sibling side")
	}
}

func TestProofIndexOutOfRange(t *testing.T) {
	m := New()
	_, _, _ = m.Add([]byte("a"))
	if _, err := m.GenerateProof(5); err != ErrIndexOutOfRange {
		t.Fatalf("expected ErrIndexOutOfRange, got %v", err)
	}
}

func TestGetLeaf(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	for i, l := range goldenLeaves {
		got, err := m.GetLeaf(uint64(i))
		if err != nil {
			t.Fatal(err)
		}
		if want := LeafHash(l); got != want {
			t.Fatalf("leaf %d mismatch", i)
		}
	}
	if _, err := m.GetLeaf(999); err != ErrIndexOutOfRange {
		t.Fatalf("expected ErrIndexOutOfRange, got %v", err)
	}
}

// TestPeakShapeAndRoot verifies the forest shape for sizes 1..32: bagged root
// equals a manual pairwise fold of peak roots, and the peak count matches
// popcount(size).
func TestPeakShapeAndRoot(t *testing.T) {
	for size := uint64(1); size <= 32; size++ {
		m := New()
		for i := uint64(0); i < size; i++ {
			_, _, _ = m.Add([]byte{byte(i), byte(i >> 8)})
		}
		peaks := m.PeakRoots()
		if uint64(len(peaks)) != popcount(size) {
			t.Fatalf("size %d: peak count = %d, want %d", size, len(peaks), popcount(size))
		}
		manual := peaks[0]
		for i := 1; i < len(peaks); i++ {
			manual = ParentHash(manual, peaks[i])
		}
		if m.BagRoot() != manual {
			t.Fatalf("size %d: bagged root != manual fold", size)
		}
	}
}

func TestFromLeavesEqualsAdd(t *testing.T) {
	raw := make([]Hash, 0, len(goldenLeaves))
	for _, l := range goldenLeaves {
		raw = append(raw, LeafHash(l))
	}
	a := New()
	for _, l := range goldenLeaves {
		_, _, _ = a.Add(l)
	}
	b := FromLeaves(raw)
	if a.BagRoot() != b.BagRoot() {
		t.Fatal("FromLeaves root != Add root")
	}
	if a.Size() != b.Size() {
		t.Fatal("FromLeaves size != Add size")
	}
}

func TestBinaryRoundTrip(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	root := m.BagRoot()
	data := m.MarshalBinary()
	m2, err := UnmarshalBinary(data)
	if err != nil {
		t.Fatal(err)
	}
	if m2.BagRoot() != root {
		t.Fatal("binary round trip changed root")
	}
	if m2.Size() != m.Size() {
		t.Fatal("binary round trip changed size")
	}
	p, err := m2.GenerateProof(0)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyProof(root, LeafHash(goldenLeaves[0]), p, m2.Size()) {
		t.Fatal("verify failed after binary reload")
	}
}

func TestBinaryRoundTripEmpty(t *testing.T) {
	m := New()
	data := m.MarshalBinary()
	m2, err := UnmarshalBinary(data)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Size() != 0 || m2.BagRoot() != EmptyRoot() {
		t.Fatal("empty binary round trip failed")
	}
}

func TestUnmarshalBinaryRejectsMalformedState(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	valid := m.MarshalBinary()

	tests := map[string][]byte{
		"truncated":    valid[:len(valid)-1],
		"trailing":     append(append([]byte(nil), valid...), 0xff),
		"wrong nodes":  append([]byte(nil), valid...),
		"wrong height": append([]byte(nil), valid...),
	}
	binary.LittleEndian.PutUint32(tests["wrong nodes"][8:12], 1)
	binary.LittleEndian.PutUint32(tests["wrong height"][4:8], 63)

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := UnmarshalBinary(data); err == nil {
				t.Fatal("expected malformed state to be rejected")
			}
		})
	}
}

func TestNodesRoundTrip(t *testing.T) {
	m := New()
	for _, l := range goldenLeaves {
		_, _, _ = m.Add(l)
	}
	root := m.BagRoot()
	m2, err := FromNodes(m.Nodes())
	if err != nil {
		t.Fatal(err)
	}
	if m2.BagRoot() != root {
		t.Fatal("FromNodes changed root")
	}
	if m2.Size() != m.Size() {
		t.Fatal("FromNodes changed size")
	}
}

func TestHashFromBytes(t *testing.T) {
	h := LeafHash([]byte("abc"))
	got, err := HashFromBytes(h[:])
	if err != nil {
		t.Fatal(err)
	}
	if got != h {
		t.Fatal("HashFromBytes mismatch")
	}
	if _, err := HashFromBytes([]byte("tooshort")); err == nil {
		t.Fatal("expected error for short input")
	}
}

func TestAppendOnlyInvariant(t *testing.T) {
	// Adding more leaves never changes previously valid proofs.
	m := New()
	hashes := make([]Hash, 0)
	for _, l := range goldenLeaves[:3] {
		_, _, _ = m.Add(l)
		hashes = append(hashes, LeafHash(l))
	}
	proofs := make([]Proof, 3)
	for i := range proofs {
		proofs[i], _ = m.GenerateProof(uint64(i))
	}
	root3 := m.BagRoot()
	for _, l := range goldenLeaves[3:] {
		_, _, _ = m.Add(l)
	}
	finalRoot := m.BagRoot()
	if finalRoot == root3 {
		t.Fatal("root should change after appending")
	}
	for i := range proofs {
		if !VerifyProof(root3, hashes[i], proofs[i], 3) {
			t.Fatalf("old proof invalid after append (leaf %d)", i)
		}
	}
}

func TestDeterministicRootAcrossSizes(t *testing.T) {
	// The root after N leaves must equal the root of an MMR rebuilt from those N
	// leaves alone, for N up to 40.
	for n := uint64(1); n <= 40; n++ {
		leaves := make([]Hash, n)
		for i := uint64(0); i < n; i++ {
			leaves[i] = LeafHash([]byte{byte(i)})
		}
		a := FromLeaves(leaves)
		b := New()
		for i := uint64(0); i < n; i++ {
			_, _, _ = b.Add([]byte{byte(i)})
		}
		if a.BagRoot() != b.BagRoot() {
			t.Fatalf("n=%d: FromLeaves != incremental root", n)
		}
	}
}

func TestProofAndPersistencePropertiesAcrossForestShapes(t *testing.T) {
	for size := uint64(1); size <= 64; size++ {
		m := New()
		leaves := make([]Hash, size)
		for i := uint64(0); i < size; i++ {
			leaves[i] = LeafHash([]byte{byte(size), byte(i)})
			m.AddRaw(leaves[i])
		}
		for i, leaf := range leaves {
			proof, err := m.GenerateProof(uint64(i))
			if err != nil {
				t.Fatalf("size %d leaf %d: %v", size, i, err)
			}
			if !VerifyProof(m.BagRoot(), leaf, proof, size) {
				t.Fatalf("size %d leaf %d: proof failed", size, i)
			}
		}
		restored, err := FromNodes(m.Nodes())
		if err != nil {
			t.Fatalf("size %d: restore: %v", size, err)
		}
		if restored.Size() != size || restored.BagRoot() != m.BagRoot() {
			t.Fatalf("size %d: node persistence drift", size)
		}
	}
}

func popcount(x uint64) uint64 {
	c := uint64(0)
	for x > 0 {
		c++
		x &= x - 1
	}
	return c
}
