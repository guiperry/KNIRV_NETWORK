package locator_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"
)

// suiteJSON is a stored test suite in the shape the store writes.
func suiteJSON(t *testing.T, nodeID string, testIDs ...string) []byte {
	t.Helper()
	s := locator.ErrorTestSuite{ErrorNodeID: nodeID, Status: locator.SuiteCollecting, Version: 1,
		UpdatedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	for _, id := range testIDs {
		s.Tests = append(s.Tests, locator.ErrorTestCase{ID: id, Input: "in-" + id, Expected: "out-" + id, AuthorID: "dev"})
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func openSuiteStore(t *testing.T) *storage.BluntDBStorage {
	t.Helper()
	s, err := storage.NewBluntDBStorage(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// A suite that exists only under the legacy prefix still loads.
func TestSuiteLegacyRowLoadsByFallback(t *testing.T) {
	store := openSuiteStore(t)
	if err := store.Put([]byte("nrv:error_tests:n1"), suiteJSON(t, "n1", "t1", "t2")); err != nil {
		t.Fatal(err)
	}
	got, err := locator.NewErrorTestSuiteStore(store).Get("n1")
	if err != nil {
		t.Fatalf("legacy suite must load through the fallback: %v", err)
	}
	if len(got.Tests) != 2 || got.Tests[0].ID != "t1" {
		t.Fatalf("wrong suite loaded: %+v", got)
	}
	if _, err := locator.NewErrorTestSuiteStore(store).Get("absent"); err != locator.ErrTestSuiteNotFound {
		t.Fatalf("missing suite must be ErrTestSuiteNotFound, got %v", err)
	}
}

// A write over a legacy-only suite goes to the new prefix and leaves the legacy row untouched.
func TestSuiteContributeWritesNewPrefix(t *testing.T) {
	store := openSuiteStore(t)
	legacy := suiteJSON(t, "n1", "t1")
	if err := store.Put([]byte("nrv:error_tests:n1"), legacy); err != nil {
		t.Fatal(err)
	}
	st := locator.NewErrorTestSuiteStore(store)
	if _, err := st.ContributeTest("n1", locator.ErrorTestCase{ID: "t2", Input: "a", Expected: "b", AuthorID: "dev"}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.Has([]byte("graph:error_tests:n1")); !ok {
		t.Fatal("contributed suite must be written under graph:error_tests:")
	}
	if raw, _ := store.Get([]byte("nrv:error_tests:n1")); string(raw) != string(legacy) {
		t.Fatal("nothing may write under the legacy prefix; legacy row must be unchanged")
	}
	got, err := st.Get("n1")
	if err != nil || len(got.Tests) != 2 {
		t.Fatalf("read after write must see both tests, got %v %+v", err, got)
	}
}

// When both prefixes hold a suite for the same id, the current prefix wins.
func TestSuiteCurrentPrefixWinsOverLegacyDuplicate(t *testing.T) {
	store := openSuiteStore(t)
	if err := store.Put([]byte("nrv:error_tests:n1"), suiteJSON(t, "n1", "old-1")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put([]byte("graph:error_tests:n1"), suiteJSON(t, "n1", "new-1", "new-2")); err != nil {
		t.Fatal(err)
	}
	got, err := locator.NewErrorTestSuiteStore(store).Get("n1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tests) != 2 || got.Tests[0].ID != "new-1" {
		t.Fatalf("current-prefix suite must win, got %+v", got.Tests)
	}
}
