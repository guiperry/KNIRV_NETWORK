package locator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// BracketSize is the byte length of one encoded NRV bracket (the 80-byte
// `.nrv` layout owned by KNIRVBASE). This package never parses a bracket; it
// only hashes and indexes the bytes a BracketEncoder returns.
const BracketSize = 80

// BracketEncoder encodes one test case as its 80-byte NRV bracket.
//
// The production encoder is KNIRVBASE's nrv.EncodeTestCaseToNRV. KNIRVGRAPH does
// not import KNIRVBASE, so the encoder is injected (WithBracketEncoder) rather
// than reimplemented here. position is the test's index in the suite (0-7).
// The encoder must be deterministic: the same test at the same position must
// always produce the same bytes, or suite_hash is not reproducible.
type BracketEncoder interface {
	EncodeTestBracket(errorNodeID string, position int, tc ErrorTestCase) ([BracketSize]byte, error)
}

// ErrNoBracketEncoder is returned when a suite would seal but the store has no
// BracketEncoder. The store fails closed: it never seals with a hash that is
// not derived from the 8 brackets.
var ErrNoBracketEncoder = errors.New("no NRV bracket encoder is configured on this server; the error node's test suite cannot be sealed")

// ErrSuiteTooLarge is returned when a sidecar index is built over more than
// ErrorTestSuiteSize tests.
var ErrSuiteTooLarge = fmt.Errorf("an error node's test suite holds at most %d tests", ErrorTestSuiteSize)

// ErrNRVIndexNotFound is returned when no sidecar index exists for a node.
var ErrNRVIndexNotFound = errors.New("no NRV index is recorded for this error node")

// errorNodeNRVIndexKeyPrefix is where sidecar indexes are written. It is kept
// apart from the suite prefix so the bracket linkage never changes a suite row.
const errorNodeNRVIndexKeyPrefix = "graph:error_nrv_index:"

// ErrorNodeNRVIndex is the sidecar linking an ErrorNode's test suite to its
// NRV brackets (DVE plan D5.2). It lives beside the suite, never inside the
// bracket bytes.
//
//   - Status collecting: fewer than ErrorTestSuiteSize tests; SuiteHash is empty
//     and NRVIDs/TestDigests are filled for the positions contributed so far.
//   - Status sealed: all ErrorTestSuiteSize positions are filled and SuiteHash
//     is H(the 8 brackets in position order).
//
// TestDigests are content hashes of the full private test case. They are never
// returned by a public view.
type ErrorNodeNRVIndex struct {
	ErrorNodeID string      `json:"error_node_id"`
	SuiteHash   string      `json:"suite_hash,omitempty"`
	NRVIDs      [8]string   `json:"nrv_ids"`
	TestDigests [8]string   `json:"test_digests"`
	Status      SuiteStatus `json:"status"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Count is the number of positions that carry a bracket.
func (x *ErrorNodeNRVIndex) Count() int {
	n := 0
	for _, id := range x.NRVIDs {
		if id != "" {
			n++
		}
	}
	return n
}

// Validate checks the invariants that Put enforces and Get re-checks.
func (x *ErrorNodeNRVIndex) Validate() error {
	if x.ErrorNodeID == "" {
		return errors.New("nrv index: error_node_id is required")
	}
	count := x.Count()
	for i := range x.NRVIDs {
		// Filled positions must form a prefix 0..count-1, with digests in step.
		if (x.NRVIDs[i] != "") != (x.TestDigests[i] != "") {
			return fmt.Errorf("nrv index: position %d has an id without a digest or the reverse", i)
		}
		if (x.NRVIDs[i] != "") != (i < count) {
			return fmt.Errorf("nrv index: position %d breaks the contiguous 0..%d prefix", i, count-1)
		}
	}
	switch x.Status {
	case SuiteCollecting:
		if x.SuiteHash != "" {
			return errors.New("nrv index: a collecting index must not carry a suite_hash")
		}
		if count >= ErrorTestSuiteSize {
			return errors.New("nrv index: a full index must be sealed")
		}
	case SuiteSealed:
		if count != ErrorTestSuiteSize || x.SuiteHash == "" {
			return fmt.Errorf("nrv index: a sealed index needs %d brackets and a suite_hash", ErrorTestSuiteSize)
		}
	default:
		return fmt.Errorf("nrv index: unknown status %q", x.Status)
	}
	return nil
}

// ErrorNodeNRVIndexStore persists sidecar indexes keyed by ErrorNode ID.
type ErrorNodeNRVIndexStore struct {
	kv KV
}

// NewErrorNodeNRVIndexStore builds a sidecar store over kv.
func NewErrorNodeNRVIndexStore(kv KV) *ErrorNodeNRVIndexStore {
	return &ErrorNodeNRVIndexStore{kv: kv}
}

// Put validates and persists idx, replacing any earlier index for its node.
func (s *ErrorNodeNRVIndexStore) Put(idx *ErrorNodeNRVIndex) error {
	if err := idx.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshal nrv index: %w", err)
	}
	if err := s.kv.Put([]byte(errorNodeNRVIndexKeyPrefix+idx.ErrorNodeID), raw); err != nil {
		return fmt.Errorf("persist nrv index: %w", err)
	}
	return nil
}

// Get returns the index for errorNodeID, or ErrNRVIndexNotFound.
func (s *ErrorNodeNRVIndexStore) Get(errorNodeID string) (*ErrorNodeNRVIndex, error) {
	raw, err := s.kv.Get([]byte(errorNodeNRVIndexKeyPrefix + errorNodeID))
	if err != nil || len(raw) == 0 {
		return nil, ErrNRVIndexNotFound
	}
	var idx ErrorNodeNRVIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("decode nrv index for %s: %w", errorNodeID, err)
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}
	return &idx, nil
}

// suiteHashFromBrackets is H(b0 || b1 || ... || b7), taken in position order.
// Changing the order of the brackets changes the hash.
func suiteHashFromBrackets(brackets [][BracketSize]byte) string {
	h := sha256.New()
	for i := range brackets {
		h.Write(brackets[i][:])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// nrvIDFromBracket is the stable identifier of one bracket: the SHA-256 of its
// 80 bytes.
func nrvIDFromBracket(b [BracketSize]byte) string {
	sum := sha256.Sum256(b[:])
	return hex.EncodeToString(sum[:])
}

// testDigestFromCase is the content hash of the full private test case,
// including expected. It is stored only in the sidecar, never in a public view.
func testDigestFromCase(tc ErrorTestCase) (string, error) {
	raw, err := json.Marshal(tc)
	if err != nil {
		return "", fmt.Errorf("digest test %q: %w", tc.ID, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// encodeSuiteBrackets encodes every test in suite by position. It returns the
// brackets and their nrv ids in position order.
func encodeSuiteBrackets(enc BracketEncoder, suite *ErrorTestSuite) ([][BracketSize]byte, []string, error) {
	if len(suite.Tests) > ErrorTestSuiteSize {
		return nil, nil, ErrSuiteTooLarge
	}
	brackets := make([][BracketSize]byte, 0, len(suite.Tests))
	ids := make([]string, 0, len(suite.Tests))
	for i, tc := range suite.Tests {
		b, err := enc.EncodeTestBracket(suite.ErrorNodeID, i, tc)
		if err != nil {
			return nil, nil, fmt.Errorf("encode test %q at position %d: %w", tc.ID, i, err)
		}
		brackets = append(brackets, b)
		ids = append(ids, nrvIDFromBracket(b))
	}
	return brackets, ids, nil
}

// buildNRVIndex derives the sidecar for suite. For a sealed suite it also checks
// that the stored suite_hash is the hash of the brackets the encoder produces
// now; a mismatch means the encoder or the tests changed and the index is refused.
func buildNRVIndex(enc BracketEncoder, suite *ErrorTestSuite) (*ErrorNodeNRVIndex, error) {
	brackets, ids, err := encodeSuiteBrackets(enc, suite)
	if err != nil {
		return nil, err
	}
	idx := &ErrorNodeNRVIndex{
		ErrorNodeID: suite.ErrorNodeID,
		Status:      SuiteCollecting,
		UpdatedAt:   suite.UpdatedAt,
	}
	for i, tc := range suite.Tests {
		digest, err := testDigestFromCase(tc)
		if err != nil {
			return nil, err
		}
		idx.NRVIDs[i] = ids[i]
		idx.TestDigests[i] = digest
	}
	if suite.Sealed() {
		if want := suiteHashFromBrackets(brackets); suite.SuiteHash != want {
			return nil, fmt.Errorf("suite_hash %s does not match its brackets (%s); the encoder or the tests changed", suite.SuiteHash, want)
		}
		idx.Status = SuiteSealed
		idx.SuiteHash = suite.SuiteHash
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}
	return idx, nil
}
