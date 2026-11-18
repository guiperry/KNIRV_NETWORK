package locator

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"
)

// testBracketEncoder is a deterministic stand-in for KNIRVBASE's
// EncodeTestCaseToNRV. It depends only on the node ID, the position and the
// input, so the same suite always produces the same 8 brackets.
type testBracketEncoder struct{}

func (testBracketEncoder) EncodeTestBracket(errorNodeID string, position int, tc ErrorTestCase) ([BracketSize]byte, error) {
	var b [BracketSize]byte
	in := sha256.Sum256([]byte(tc.Input))
	copy(b[0:32], in[:])
	binary.BigEndian.PutUint32(b[32:36], uint32(position))
	node := sha256.Sum256([]byte(errorNodeID))
	copy(b[36:68], node[:])
	return b, nil
}

// testSuiteTests returns the 8 fixed tests used by the golden checks.
func testSuiteTests() []ErrorTestCase {
	tests := make([]ErrorTestCase, 0, ErrorTestSuiteSize)
	for i := 1; i <= ErrorTestSuiteSize; i++ {
		tests = append(tests, ErrorTestCase{
			ID: fmt.Sprintf("g%d", i), Input: fmt.Sprintf("golden input %d", i),
			Expected: fmt.Sprintf("golden answer %d", i), Assertion: AssertExact, AuthorID: "dev",
		})
	}
	return tests
}

// goldenSuiteHash is H(b0||...||b7) for testSuiteTests() on node "golden-node",
// with testBracketEncoder. It was computed once by the store and is pinned here.
// TestSuiteHashGolden also recomputes it independently from the encoder.
const goldenSuiteHash = "c5b20f4d9e47865e4bc3111e76cea3e7eb70d626175bfda51c3baefa8f00714d"

func independentSuiteHash(t *testing.T, nodeID string, tests []ErrorTestCase) string {
	t.Helper()
	h := sha256.New()
	for i, tc := range tests {
		b, err := testBracketEncoder{}.EncodeTestBracket(nodeID, i, tc)
		if err != nil {
			t.Fatal(err)
		}
		h.Write(b[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestSuiteHashGolden(t *testing.T) {
	st := newStore()
	s, err := st.Replace("golden-node", testSuiteTests())
	if err != nil {
		t.Fatal(err)
	}
	want := independentSuiteHash(t, "golden-node", testSuiteTests())
	if s.SuiteHash != want {
		t.Fatalf("suite_hash must equal H(8 brackets in order): got %s want %s", s.SuiteHash, want)
	}
	if s.SuiteHash != goldenSuiteHash {
		t.Fatalf("golden suite_hash changed: got %s want %s", s.SuiteHash, goldenSuiteHash)
	}
	if len(s.NRVIDs) != ErrorTestSuiteSize {
		t.Fatalf("a sealed suite carries 8 nrv ids, got %d", len(s.NRVIDs))
	}
	// Stable across a second store with the same inputs.
	s2, err := newStore().Replace("golden-node", testSuiteTests())
	if err != nil {
		t.Fatal(err)
	}
	if s2.SuiteHash != s.SuiteHash {
		t.Fatal("suite_hash must be stable for the same inputs")
	}
}

func TestSuiteHashIsOrderSensitive(t *testing.T) {
	st := newStore()
	tests := testSuiteTests()
	a, err := st.Replace("order-node", tests)
	if err != nil {
		t.Fatal(err)
	}
	rev := make([]ErrorTestCase, len(tests))
	for i := range tests {
		rev[i] = tests[len(tests)-1-i]
	}
	b, err := newStore().Replace("order-node", rev)
	if err != nil {
		t.Fatal(err)
	}
	if a.SuiteHash == b.SuiteHash {
		t.Fatal("reversing the 8 tests must change suite_hash")
	}
}

func TestSuiteCollectingBelowEight(t *testing.T) {
	st := newStore()
	for i := 1; i < ErrorTestSuiteSize; i++ {
		if _, err := st.ContributeTest("c1", arenaTest(i)); err != nil {
			t.Fatal(err)
		}
		idx, err := st.indexes.Get("c1")
		if err != nil {
			t.Fatalf("after %d tests the sidecar must exist: %v", i, err)
		}
		if idx.Status != SuiteCollecting || idx.SuiteHash != "" || idx.Count() != i {
			t.Fatalf("after %d tests want collecting with %d brackets, got %+v", i, i, idx)
		}
	}
	got, err := st.Get("c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != SuiteCollecting || got.SuiteHash != "" || len(got.NRVIDs) != 0 {
		t.Fatalf("0-7 tests must stay collecting with no hash or sealed ids: %+v", got)
	}
}

func TestNinthContributionRefused(t *testing.T) {
	st := newStore()
	if _, err := st.Replace("n9", testSuiteTests()); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ContributeTest("n9", arenaTest(9)); !errors.Is(err, ErrSuiteSealed) {
		t.Fatalf("a 9th test must be refused: %v", err)
	}
	nine := make([]ErrorTestCase, 0, ErrorTestSuiteSize+1)
	nine = append(nine, testSuiteTests()...)
	nine = append(nine, arenaTest(9))
	if _, err := buildNRVIndex(testBracketEncoder{}, &ErrorTestSuite{ErrorNodeID: "n9", Tests: nine}); !errors.Is(err, ErrSuiteTooLarge) {
		t.Fatalf("an index over 9 tests must be refused: %v", err)
	}
}

func TestSealFailsClosedWithoutEncoder(t *testing.T) {
	st := NewErrorTestSuiteStore(&memKV{m: map[string][]byte{}})
	for i := 1; i < ErrorTestSuiteSize; i++ {
		if _, err := st.ContributeTest("nc", arenaTest(i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.ContributeTest("nc", arenaTest(ErrorTestSuiteSize)); !errors.Is(err, ErrNoBracketEncoder) {
		t.Fatalf("sealing without an encoder must fail closed, got %v", err)
	}
	got, err := st.Get("nc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != SuiteCollecting || len(got.Tests) != ErrorTestSuiteSize-1 || got.SuiteHash != "" {
		t.Fatalf("a refused seal must leave the suite collecting and unchanged: %+v", got)
	}
}

func TestSidecarRoundTrip(t *testing.T) {
	kv := &memKV{m: map[string][]byte{}}
	st := NewErrorTestSuiteStore(kv).WithBracketEncoder(testBracketEncoder{})
	for i := 1; i <= 3; i++ {
		if _, err := st.ContributeTest("rt", arenaTest(i)); err != nil {
			t.Fatal(err)
		}
	}
	// A fresh store over the same KV reads the same sidecar.
	idx, err := NewErrorNodeNRVIndexStore(kv).Get("rt")
	if err != nil {
		t.Fatal(err)
	}
	if idx.Status != SuiteCollecting || idx.Count() != 3 || idx.NRVIDs[0] == "" || idx.NRVIDs[3] != "" {
		t.Fatalf("collecting sidecar wrong after round-trip: %+v", idx)
	}
	for i := 4; i <= ErrorTestSuiteSize; i++ {
		if _, err := st.ContributeTest("rt", arenaTest(i)); err != nil {
			t.Fatal(err)
		}
	}
	suite, err := NewErrorTestSuiteStore(kv).Get("rt")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := NewErrorNodeNRVIndexStore(kv).Get("rt")
	if err != nil {
		t.Fatal(err)
	}
	if sealed.Status != SuiteSealed || sealed.SuiteHash != suite.SuiteHash || sealed.Count() != ErrorTestSuiteSize {
		t.Fatalf("sealed sidecar must match the suite: %+v vs hash %s", sealed, suite.SuiteHash)
	}
	for i := range suite.NRVIDs {
		if sealed.NRVIDs[i] != suite.NRVIDs[i] {
			t.Fatalf("sidecar and suite disagree on nrv id %d", i)
		}
	}
	if _, err := NewErrorNodeNRVIndexStore(kv).Get("absent"); !errors.Is(err, ErrNRVIndexNotFound) {
		t.Fatalf("missing index must be ErrNRVIndexNotFound, got %v", err)
	}
}

func TestSidecarValidateRejectsBadShapes(t *testing.T) {
	s := NewErrorNodeNRVIndexStore(&memKV{m: map[string][]byte{}})
	gap := &ErrorNodeNRVIndex{ErrorNodeID: "x", Status: SuiteCollecting, UpdatedAt: time.Now()}
	gap.NRVIDs[1], gap.TestDigests[1] = "id", "d"
	if err := s.Put(gap); err == nil {
		t.Fatal("a gap in positions must be refused")
	}
	unsealed := &ErrorNodeNRVIndex{ErrorNodeID: "x", Status: SuiteSealed}
	if err := s.Put(unsealed); err == nil {
		t.Fatal("a sealed index without 8 brackets and a hash must be refused")
	}
	hashed := &ErrorNodeNRVIndex{ErrorNodeID: "x", Status: SuiteCollecting, SuiteHash: "h"}
	if err := s.Put(hashed); err == nil {
		t.Fatal("a collecting index must not carry a suite_hash")
	}
}

func TestBuildIndexRefusesHashMismatch(t *testing.T) {
	st := newStore()
	s, err := st.Replace("mm", testSuiteTests())
	if err != nil {
		t.Fatal(err)
	}
	s.SuiteHash = "tampered"
	if _, err := buildNRVIndex(testBracketEncoder{}, s); err == nil {
		t.Fatal("a suite whose hash does not match its brackets must not be indexed")
	}
}
