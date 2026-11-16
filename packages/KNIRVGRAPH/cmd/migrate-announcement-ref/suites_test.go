package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"
)

// suiteRowTests is suiteRow with several tests in one legacy or current row.
func suiteRowTests(t *testing.T, nodeID string, testIDs ...string) []byte {
	t.Helper()
	s := locator.ErrorTestSuite{ErrorNodeID: nodeID, Status: locator.SuiteCollecting, Version: 1}
	for _, id := range testIDs {
		s.Tests = append(s.Tests, locator.ErrorTestCase{ID: id, Input: "in", Expected: "out", AuthorID: "dev"})
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// contribute adds one test through the real suite store, as the CLI's
// contribute path does. On a legacy-only suite it writes the full test list
// (legacy tests plus the new one) under graph:error_tests:.
func contribute(t *testing.T, dir, nodeID, testID string) {
	t.Helper()
	s, err := storage.NewBluntDBStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	st := locator.NewErrorTestSuiteStore(s)
	if _, err := st.ContributeTest(nodeID, locator.ErrorTestCase{ID: testID, Input: "in", Expected: "out", AuthorID: "dev"}); err != nil {
		t.Fatal(err)
	}
}

func suiteRow(t *testing.T, nodeID, testID string) []byte {
	t.Helper()
	raw, err := json.Marshal(locator.ErrorTestSuite{
		ErrorNodeID: nodeID, Status: locator.SuiteCollecting, Version: 1,
		Tests: []locator.ErrorTestCase{{ID: testID, Input: "in", Expected: "out", AuthorID: "dev"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Test-suite rows move in the same run as error nodes, with their own count line.
func TestMigratesTestSuitesWithSeparateCountLine(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_nodes:old1": row(t, "old1", "h1", true),
		"nrv:error_tests:s1":   suiteRow(t, "s1", "t1"),
		"nrv:error_tests:s2":   suiteRow(t, "s2", "t9"),
		"graph:error_tests:s2": suiteRow(t, "s2", "t9"),
	})

	code, out, errOut := runCmd("--db", dir)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "scanned=1 rewritten=1 ") {
		t.Fatalf("node counts must not include suites: %q", out)
	}
	if !strings.Contains(out, "test-suites: scanned=3 rewritten=1 already-new=1 duplicates-removed=1 skipped-unreadable=0 skipped-conflict=0") {
		t.Fatalf("suite count line wrong: %q", out)
	}
	if !has(t, dir, "graph:error_tests:s1") || has(t, dir, "nrv:error_tests:s1") {
		t.Fatal("s1 suite must move to the new prefix")
	}
	if has(t, dir, "nrv:error_tests:s2") {
		t.Fatal("identical legacy s2 duplicate must be removed")
	}
	if !has(t, dir, "migration:announcement_ref") {
		t.Fatal("marker must be written")
	}
}

// The second run is refused for suites too: nothing moves.
func TestSecondRunRefusedWithSuitesPresent(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{"nrv:error_tests:s1": suiteRow(t, "s1", "t1")})
	if code, _, errOut := runCmd("--db", dir); code != 0 {
		t.Fatalf("first run exit %d: %q", code, errOut)
	}
	put(t, dir, map[string][]byte{"nrv:error_tests:s7": suiteRow(t, "s7", "t1")})

	code, _, errOut := runCmd("--db", dir)
	if code != 3 || !strings.Contains(errOut, "already present") {
		t.Fatalf("second run must be refused with exit 3, got %d %q", code, errOut)
	}
	if !has(t, dir, "nrv:error_tests:s7") {
		t.Fatal("refused run must not move suite rows")
	}
}

// A legacy-only suite that is then contributed to: ContributeTest writes
// graph:error_tests:s1 with the legacy test plus the new one, leaving the legacy
// row a strict subset. The migration treats that row as already migrated, so the
// run exits 0 and writes the marker; a rerun is then refused with exit 3.
func TestContributedSupersetLegacySuiteWritesMarker(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{"nrv:error_tests:s1": suiteRow(t, "s1", "t1")})
	contribute(t, dir, "s1", "t2")

	code, out, errOut := runCmd("--db", dir)
	if code != 0 {
		t.Fatalf("contributed superset must migrate: exit %d, stderr %q, out %q", code, errOut, out)
	}
	if strings.Contains(out, "skipped:") || !strings.Contains(out, "superseded-removed=1") {
		t.Fatalf("superset legacy row must be reported as removed, not skipped: %q", out)
	}
	if has(t, dir, "nrv:error_tests:s1") {
		t.Fatal("superseded legacy row must be removed")
	}
	if !has(t, dir, "graph:error_tests:s1") {
		t.Fatal("current superset row must be kept")
	}
	if !has(t, dir, "migration:announcement_ref") {
		t.Fatal("marker must be written")
	}
	if code, _, _ := runCmd("--db", dir); code != 3 {
		t.Fatalf("rerun after the marker must be refused with exit 3, got %d", code)
	}
}

// A legacy suite with a test the current row lacks is still a divergence: exit 4,
// no marker, and both rows stay in place.
func TestDivergentLegacySuiteStillExits4(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_tests:s1":   suiteRowTests(t, "s1", "t1", "t2"),
		"graph:error_tests:s1": suiteRowTests(t, "s1", "t1"),
	})

	for attempt := 1; attempt <= 2; attempt++ {
		code, out, errOut := runCmd("--db", dir)
		if code != 4 {
			t.Fatalf("attempt %d: divergent suite must exit 4, got %d (stderr %q)", attempt, code, errOut)
		}
		if !strings.Contains(out, "skipped: nrv:error_tests:s1") {
			t.Fatalf("attempt %d: divergent key not reported: %q", attempt, out)
		}
		if has(t, dir, "migration:announcement_ref") {
			t.Fatalf("attempt %d: marker written with a divergent suite", attempt)
		}
	}
	if !has(t, dir, "nrv:error_tests:s1") || !has(t, dir, "graph:error_tests:s1") {
		t.Fatal("divergent rows must both stay in place")
	}
}

// sealedSuiteRow builds a sealed legacy or current row with the given seal metadata.
func sealedSuiteRow(t *testing.T, nodeID, hash string, version int, testIDs ...string) []byte {
	t.Helper()
	sealedAt := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	s := locator.ErrorTestSuite{ErrorNodeID: nodeID, Status: locator.SuiteSealed, Version: version,
		SuiteHash: hash, SealedAt: &sealedAt, UpdatedAt: sealedAt}
	for _, id := range testIDs {
		s.Tests = append(s.Tests, locator.ErrorTestCase{ID: id, Input: "in-" + id, Expected: "out-" + id, AuthorID: "dev"})
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

var suiteEightIDs = []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8"}

// P3: both rows sealed with the same 8 tests but different suite_hash. The legacy
// hash must not be dropped: exit 4, no marker, both rows stay.
func TestSealedLegacyDifferentHashExits4(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_tests:s1":   sealedSuiteRow(t, "s1", "LEGACYHASH", 1, suiteEightIDs...),
		"graph:error_tests:s1": sealedSuiteRow(t, "s1", "CURRENTHASH", 2, suiteEightIDs...),
	})
	code, out, errOut := runCmd("--db", dir)
	if code != 4 {
		t.Fatalf("sealed legacy with a different suite_hash must exit 4, got %d (stderr %q, out %q)", code, errOut, out)
	}
	if has(t, dir, "migration:announcement_ref") {
		t.Fatal("marker must not be written")
	}
	if !has(t, dir, "nrv:error_tests:s1") || !has(t, dir, "graph:error_tests:s1") {
		t.Fatal("both rows must stay in place")
	}
}

// P4: same 8 tests and same suite_hash, but the legacy version differs. Exit 4, both rows stay.
func TestSealedLegacyDifferentVersionExits4(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_tests:s1":   sealedSuiteRow(t, "s1", "SAMEHASH", 1, suiteEightIDs...),
		"graph:error_tests:s1": sealedSuiteRow(t, "s1", "SAMEHASH", 2, suiteEightIDs...),
	})
	code, out, errOut := runCmd("--db", dir)
	if code != 4 {
		t.Fatalf("sealed legacy with a different version must exit 4, got %d (stderr %q, out %q)", code, errOut, out)
	}
	if has(t, dir, "migration:announcement_ref") || !has(t, dir, "nrv:error_tests:s1") {
		t.Fatal("marker written or legacy seal metadata dropped")
	}
}
