package nrv

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type memKV struct{ m map[string][]byte }

func (k *memKV) Put(key, value []byte) error { k.m[string(key)] = value; return nil }
func (k *memKV) Get(key []byte) ([]byte, error) {
	v, ok := k.m[string(key)]
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

func arenaTest(i int) ErrorTestCase {
	return ErrorTestCase{
		ID: fmt.Sprintf("t%d", i), Input: fmt.Sprintf("input %d", i),
		Expected: fmt.Sprintf("answer-%d", i), Assertion: AssertContains,
		AuthorID: fmt.Sprintf("arena-agent-%d", i%3),
	}
}

func newStore() *ErrorTestSuiteStore { return NewErrorTestSuiteStore(&memKV{m: map[string][]byte{}}) }

func TestContributionsSealAtEight(t *testing.T) {
	st := newStore()
	if _, err := st.Get("e1"); !errors.Is(err, ErrTestSuiteNotFound) {
		t.Fatalf("expected ErrTestSuiteNotFound, got %v", err)
	}
	for i := 1; i < ErrorTestSuiteSize; i++ {
		s, err := st.ContributeTest("e1", arenaTest(i))
		if err != nil {
			t.Fatal(err)
		}
		if s.Sealed() || s.SuiteHash != "" {
			t.Fatalf("suite sealed early at %d tests", i)
		}
		if _, err := s.GradeOutputs(nil); !errors.Is(err, ErrSuiteNotSealed) {
			t.Fatalf("expected an unsealed suite to refuse grading, got %v", err)
		}
	}
	if _, err := st.ContributeTest("e1", arenaTest(1)); err == nil {
		t.Fatal("expected a duplicate test id to be rejected")
	}
	s, err := st.ContributeTest("e1", arenaTest(ErrorTestSuiteSize))
	if err != nil {
		t.Fatal(err)
	}
	if !s.Sealed() || s.SuiteHash == "" || s.SealedAt == nil {
		t.Fatalf("expected the 8th contribution to seal the suite: %+v", s)
	}
	if _, err := st.ContributeTest("e1", arenaTest(9)); !errors.Is(err, ErrSuiteSealed) {
		t.Fatalf("expected contributions after sealing to be refused, got %v", err)
	}
}

func TestContributionValidation(t *testing.T) {
	st := newStore()
	tc := arenaTest(1)
	tc.AuthorID = ""
	if _, err := st.ContributeTest("e1", tc); err == nil {
		t.Fatal("expected a test without an author to be rejected")
	}
	tc = arenaTest(1)
	tc.Assertion = AssertRegex
	tc.Expected = "("
	if _, err := st.ContributeTest("e1", tc); err == nil {
		t.Fatal("expected an invalid regex to be rejected")
	}
	tc = arenaTest(1)
	tc.Assertion = ""
	s, err := st.ContributeTest("e1", tc)
	if err != nil {
		t.Fatal(err)
	}
	if s.Tests[0].Assertion != AssertExact {
		t.Fatalf("expected an unset assertion to default to exact, got %q", s.Tests[0].Assertion)
	}
}

func TestGradingDoesNotLeakExpected(t *testing.T) {
	st := newStore()
	tests := make([]ErrorTestCase, 0, ErrorTestSuiteSize)
	for i := 1; i <= ErrorTestSuiteSize; i++ {
		tests = append(tests, arenaTest(i))
	}
	tests[1].Assertion, tests[1].Expected = AssertExact, "Exact Answer"
	tests[2].Assertion, tests[2].Expected = AssertNotContains, "panic"
	tests[3].Assertion, tests[3].Expected = AssertRegex, `^status: (ok|done)$`
	s, err := st.Replace("e1", tests)
	if err != nil {
		t.Fatal(err)
	}

	outputs := map[string]string{}
	for i, tc := range tests {
		outputs[tc.ID] = fmt.Sprintf("here is ANSWER-%d", i+1)
	}
	outputs["t2"] = "  exact answer "
	outputs["t3"] = "all good"
	outputs["t4"] = "STATUS: done"
	report, err := s.GradeOutputs(outputs)
	if err != nil {
		t.Fatal(err)
	}
	if !report.AllPassed {
		t.Fatalf("expected all tests to pass, got %+v", report)
	}

	outputs["t3"] = "it will panic"
	delete(outputs, "t8")
	report, _ = s.GradeOutputs(outputs)
	if report.AllPassed || report.Passed != ErrorTestSuiteSize-2 {
		t.Fatalf("expected two failures, got %d passed", report.Passed)
	}
	for _, r := range report.Results {
		if strings.Contains(r.Reason, "answer-") || strings.Contains(r.Reason, "panic") {
			t.Fatalf("grade reason leaks the expected value: %q", r.Reason)
		}
	}
	for _, pub := range s.Public().Tests {
		if pub.Input == "" || pub.AuthorID == "" {
			t.Fatalf("public view dropped input/author: %+v", pub)
		}
	}
}

func TestReplaceBumpsVersionAndHash(t *testing.T) {
	st := newStore()
	tests := make([]ErrorTestCase, 0, ErrorTestSuiteSize)
	for i := 1; i <= ErrorTestSuiteSize; i++ {
		tests = append(tests, arenaTest(i))
	}
	first, err := st.Replace("e1", tests)
	if err != nil {
		t.Fatal(err)
	}
	tests[0].Expected = "different"
	second, err := st.Replace("e1", tests)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != first.Version+1 || second.SuiteHash == first.SuiteHash {
		t.Fatalf("expected a new version and hash: %+v vs %+v", first, second)
	}
	if _, err := st.Replace("e1", tests[:7]); err == nil {
		t.Fatal("expected a 7-test replacement to be rejected")
	}
}
