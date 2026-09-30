package nrv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ErrorTestSuiteSize is the number of tests that seals an ErrorNode's suite.
//
// The tests are not authored separately from the work on the error: they are
// the tests KNIRVARENA developers and their agents write while resolving it,
// contributed to the node one at a time (ContributeTest). Once the node holds
// ErrorTestSuiteSize tests the suite is sealed, and those same tests serve
// two consumers:
//
//   - the KNIRVARENA developer-agent swarms, whose candidate resolutions for the
//     error are judged against them, and
//   - KNIRVSERVER's badge-credential benchmark, where an agent earns the badge
//     for the skill that resolves this error by passing all of them.
//
// One suite, one grader (GradeOutputs): a badge certifies an agent clears the
// exact bar the swarm's work was held to, not an approximation of it.
const ErrorTestSuiteSize = 8

// SuiteStatus is a suite's lifecycle state.
type SuiteStatus string

const (
	// SuiteCollecting: fewer than ErrorTestSuiteSize tests; not gradable.
	SuiteCollecting SuiteStatus = "collecting"
	// SuiteSealed: exactly ErrorTestSuiteSize tests; immutable except by a
	// versioned full replacement (Replace).
	SuiteSealed SuiteStatus = "sealed"
)

// AssertionKind is how a test's expected value is compared with an output.
// Every kind is deterministic, so a grade is reproducible by anyone holding
// the suite. Unset means exact, matching KNIRVARENA's DVETestCase semantics
// (input → expectedOutput).
type AssertionKind string

const (
	AssertExact       AssertionKind = "exact"        // trimmed output equals expected
	AssertContains    AssertionKind = "contains"     // output contains expected
	AssertNotContains AssertionKind = "not_contains" // output does not contain expected
	AssertRegex       AssertionKind = "regex"        // output matches expected (RE2)
)

func (k AssertionKind) valid() bool {
	switch k {
	case AssertExact, AssertContains, AssertNotContains, AssertRegex:
		return true
	}
	return false
}

// ErrorTestCase is one test on an ErrorNode. Field names follow the
// platform's existing test-case contract (backend objects.TestCase and
// KNIRVARENA's DVETestCase: input/expected). Expected is private: it is never
// returned by the public suite view and never included in a grade report.
type ErrorTestCase struct {
	ID            string        `json:"id"`
	Name          string        `json:"name,omitempty"`
	Description   string        `json:"description,omitempty"`
	Input         string        `json:"input"`
	Expected      string        `json:"expected"`
	Assertion     AssertionKind `json:"assertion,omitempty"`
	CaseSensitive bool          `json:"case_sensitive,omitempty"`
	// AuthorID is the KNIRVARENA developer or agent that contributed the test.
	AuthorID      string    `json:"author_id"`
	ContributedAt time.Time `json:"contributed_at"`
}

func (tc *ErrorTestCase) normalize() {
	if tc.Assertion == "" {
		tc.Assertion = AssertExact
	}
	if tc.Name == "" {
		tc.Name = tc.ID
	}
}

func (tc *ErrorTestCase) validate() error {
	if strings.TrimSpace(tc.ID) == "" {
		return errors.New("test id is required")
	}
	if strings.TrimSpace(tc.Input) == "" {
		return fmt.Errorf("test %q: input is required", tc.ID)
	}
	if strings.TrimSpace(tc.Expected) == "" {
		return fmt.Errorf("test %q: expected is required", tc.ID)
	}
	if strings.TrimSpace(tc.AuthorID) == "" {
		return fmt.Errorf("test %q: author_id is required", tc.ID)
	}
	if !tc.Assertion.valid() {
		return fmt.Errorf("test %q: unsupported assertion %q", tc.ID, tc.Assertion)
	}
	if tc.Assertion == AssertRegex {
		if _, err := compileAssertion(*tc); err != nil {
			return fmt.Errorf("test %q: invalid regex: %w", tc.ID, err)
		}
	}
	return nil
}

// ErrorTestSuite is the full, private suite for one ErrorNode.
type ErrorTestSuite struct {
	ErrorNodeID string          `json:"error_node_id"`
	Tests       []ErrorTestCase `json:"tests"`
	Status      SuiteStatus     `json:"status"`
	Version     int             `json:"version"`
	// SuiteHash is set when the suite seals; it identifies exactly which tests
	// a credential was earned against.
	SuiteHash string     `json:"suite_hash,omitempty"`
	SealedAt  *time.Time `json:"sealed_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Sealed reports whether the suite is complete and gradable.
func (s *ErrorTestSuite) Sealed() bool {
	return s.Status == SuiteSealed && len(s.Tests) == ErrorTestSuiteSize
}

// PublicErrorTestCase is what a test-taker may see: enough to attempt the
// test, nothing that reveals how it is graded.
type PublicErrorTestCase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Input       string `json:"input"`
	AuthorID    string `json:"author_id"`
}

// PublicErrorTestSuite is the redacted suite view.
type PublicErrorTestSuite struct {
	ErrorNodeID string                `json:"error_node_id"`
	Tests       []PublicErrorTestCase `json:"tests"`
	Status      SuiteStatus           `json:"status"`
	Required    int                   `json:"required"`
	Version     int                   `json:"version"`
	SuiteHash   string                `json:"suite_hash,omitempty"`
}

// TestGrade is the verdict for one test. Reason explains a failure without
// disclosing the expected value.
type TestGrade struct {
	TestID string `json:"test_id"`
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

// GradeReport is the verdict for a full set of outputs against a suite.
type GradeReport struct {
	ErrorNodeID string      `json:"error_node_id"`
	SuiteHash   string      `json:"suite_hash"`
	Version     int         `json:"version"`
	Results     []TestGrade `json:"results"`
	Passed      int         `json:"passed"`
	Total       int         `json:"total"`
	AllPassed   bool        `json:"all_passed"`
}

// ErrSuiteNotSealed is returned when grading is attempted before the node has
// collected all ErrorTestSuiteSize tests.
var ErrSuiteNotSealed = fmt.Errorf("this error node's test suite is not sealed yet (needs %d contributed tests)", ErrorTestSuiteSize)

// ErrSuiteSealed is returned when a contribution arrives after sealing.
var ErrSuiteSealed = errors.New("this error node's test suite is sealed; replace it with a new version instead")

func (s *ErrorTestSuite) computeHash() string {
	type graded struct {
		ID, Input, Expected string
		Assertion           AssertionKind
		CaseSensitive       bool
	}
	g := make([]graded, 0, len(s.Tests))
	for _, tc := range s.Tests {
		g = append(g, graded{tc.ID, tc.Input, tc.Expected, tc.Assertion, tc.CaseSensitive})
	}
	canonical, _ := json.Marshal(struct {
		ErrorNodeID string   `json:"error_node_id"`
		Tests       []graded `json:"tests"`
	}{s.ErrorNodeID, g})
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// Public returns the redacted view of the suite.
func (s *ErrorTestSuite) Public() *PublicErrorTestSuite {
	out := &PublicErrorTestSuite{
		ErrorNodeID: s.ErrorNodeID,
		Tests:       make([]PublicErrorTestCase, 0, len(s.Tests)),
		Status:      s.Status,
		Required:    ErrorTestSuiteSize,
		Version:     s.Version,
		SuiteHash:   s.SuiteHash,
	}
	for _, tc := range s.Tests {
		out.Tests = append(out.Tests, PublicErrorTestCase{
			ID: tc.ID, Name: tc.Name, Description: tc.Description, Input: tc.Input, AuthorID: tc.AuthorID,
		})
	}
	return out
}

// GradeOutputs grades outputs (test ID → output) against a sealed suite. A
// test with no output fails. Unknown test IDs in outputs are ignored.
func (s *ErrorTestSuite) GradeOutputs(outputs map[string]string) (*GradeReport, error) {
	if !s.Sealed() {
		return nil, ErrSuiteNotSealed
	}
	report := &GradeReport{
		ErrorNodeID: s.ErrorNodeID,
		SuiteHash:   s.SuiteHash,
		Version:     s.Version,
		Results:     make([]TestGrade, 0, len(s.Tests)),
		Total:       len(s.Tests),
	}
	for _, tc := range s.Tests {
		grade := TestGrade{TestID: tc.ID, Name: tc.Name}
		output, ok := outputs[tc.ID]
		if !ok || strings.TrimSpace(output) == "" {
			grade.Reason = "no output was produced for this test"
		} else {
			grade.Passed, grade.Reason = gradeOne(tc, output)
		}
		if grade.Passed {
			report.Passed++
		}
		report.Results = append(report.Results, grade)
	}
	report.AllPassed = report.Passed == ErrorTestSuiteSize
	return report, nil
}

func gradeOne(tc ErrorTestCase, output string) (bool, string) {
	got := strings.TrimSpace(output)
	want := strings.TrimSpace(tc.Expected)
	if !tc.CaseSensitive && tc.Assertion != AssertRegex {
		got, want = strings.ToLower(got), strings.ToLower(want)
	}
	switch tc.Assertion {
	case AssertExact, "":
		if got == want {
			return true, ""
		}
		return false, "output did not exactly match the expected answer"
	case AssertContains:
		if strings.Contains(got, want) {
			return true, ""
		}
		return false, "output did not contain the required content"
	case AssertNotContains:
		if !strings.Contains(got, want) {
			return true, ""
		}
		return false, "output contained content this test forbids"
	case AssertRegex:
		re, err := compileAssertion(tc)
		if err != nil {
			return false, "test assertion is invalid"
		}
		if re.MatchString(got) {
			return true, ""
		}
		return false, "output did not match the required pattern"
	}
	return false, "unsupported assertion"
}

func compileAssertion(tc ErrorTestCase) (*regexp.Regexp, error) {
	pattern := tc.Expected
	if !tc.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// KV is the minimal persistence the suite store needs. KNIRVGRAPH's
// storage.Storage satisfies it; keeping the dependency this narrow avoids
// coupling nrv to the storage package.
type KV interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
}

const errorTestSuiteKeyPrefix = "nrv:error_tests:"

// ErrorTestSuiteStore persists ErrorTestSuites keyed by ErrorNode ID.
type ErrorTestSuiteStore struct {
	mu  sync.Mutex
	kv  KV
	now func() time.Time
}

// NewErrorTestSuiteStore builds a store over kv.
func NewErrorTestSuiteStore(kv KV) *ErrorTestSuiteStore {
	return &ErrorTestSuiteStore{kv: kv, now: time.Now}
}

// ErrTestSuiteNotFound is returned when no test has been contributed yet.
var ErrTestSuiteNotFound = errors.New("no tests have been contributed to this error node")

// ContributeTest adds one KNIRVARENA-authored test to errorNodeID's suite.
// The suite seals automatically when it reaches ErrorTestSuiteSize tests.
func (st *ErrorTestSuiteStore) ContributeTest(errorNodeID string, tc ErrorTestCase) (*ErrorTestSuite, error) {
	if strings.TrimSpace(errorNodeID) == "" {
		return nil, errors.New("error_node_id is required")
	}
	tc.normalize()
	if err := tc.validate(); err != nil {
		return nil, err
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	suite, err := st.getLocked(errorNodeID)
	if errors.Is(err, ErrTestSuiteNotFound) {
		suite = &ErrorTestSuite{ErrorNodeID: errorNodeID, Status: SuiteCollecting, Version: 1}
	} else if err != nil {
		return nil, err
	}
	if suite.Status == SuiteSealed {
		return nil, ErrSuiteSealed
	}
	for _, existing := range suite.Tests {
		if existing.ID == tc.ID {
			return nil, fmt.Errorf("test %q is already on this error node", tc.ID)
		}
	}

	now := st.now().UTC()
	tc.ContributedAt = now
	suite.Tests = append(suite.Tests, tc)
	suite.UpdatedAt = now
	if len(suite.Tests) == ErrorTestSuiteSize {
		suite.seal(now)
	}
	return suite, st.putLocked(suite)
}

// Replace installs a complete ErrorTestSuiteSize-test suite as a new version,
// e.g. when a contributed test turns out to be wrong after sealing. Every
// test must still carry its author.
func (st *ErrorTestSuiteStore) Replace(errorNodeID string, tests []ErrorTestCase) (*ErrorTestSuite, error) {
	if len(tests) != ErrorTestSuiteSize {
		return nil, fmt.Errorf("a replacement suite needs exactly %d tests, got %d", ErrorTestSuiteSize, len(tests))
	}
	seen := make(map[string]bool, len(tests))
	normalized := make([]ErrorTestCase, 0, len(tests))
	now := st.now().UTC()
	for _, tc := range tests {
		tc.normalize()
		if err := tc.validate(); err != nil {
			return nil, err
		}
		if seen[tc.ID] {
			return nil, fmt.Errorf("duplicate test id %q", tc.ID)
		}
		seen[tc.ID] = true
		if tc.ContributedAt.IsZero() {
			tc.ContributedAt = now
		}
		normalized = append(normalized, tc)
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	version := 1
	if prev, err := st.getLocked(errorNodeID); err == nil {
		version = prev.Version + 1
	} else if !errors.Is(err, ErrTestSuiteNotFound) {
		return nil, err
	}
	suite := &ErrorTestSuite{ErrorNodeID: errorNodeID, Tests: normalized, Version: version, UpdatedAt: now}
	suite.seal(now)
	return suite, st.putLocked(suite)
}

func (s *ErrorTestSuite) seal(at time.Time) {
	s.Status = SuiteSealed
	s.SealedAt = &at
	s.SuiteHash = s.computeHash()
}

// Get returns the full, private suite for errorNodeID.
func (st *ErrorTestSuiteStore) Get(errorNodeID string) (*ErrorTestSuite, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.getLocked(errorNodeID)
}

func (st *ErrorTestSuiteStore) getLocked(errorNodeID string) (*ErrorTestSuite, error) {
	raw, err := st.kv.Get([]byte(errorTestSuiteKeyPrefix + errorNodeID))
	if err != nil || len(raw) == 0 {
		// Both KNIRVGRAPH storage backends report a missing key as an error.
		return nil, ErrTestSuiteNotFound
	}
	var suite ErrorTestSuite
	if err := json.Unmarshal(raw, &suite); err != nil {
		return nil, fmt.Errorf("decode test suite for %s: %w", errorNodeID, err)
	}
	return &suite, nil
}

func (st *ErrorTestSuiteStore) putLocked(suite *ErrorTestSuite) error {
	raw, err := json.Marshal(suite)
	if err != nil {
		return fmt.Errorf("marshal test suite: %w", err)
	}
	if err := st.kv.Put([]byte(errorTestSuiteKeyPrefix+suite.ErrorNodeID), raw); err != nil {
		return fmt.Errorf("persist test suite: %w", err)
	}
	return nil
}
