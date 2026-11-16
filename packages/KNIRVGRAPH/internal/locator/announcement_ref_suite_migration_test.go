package locator_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/locator"
)

// One run moves both prefixes, reports suite counts apart from node counts, and writes one marker.
func TestSuiteMigrationMovesBothPrefixesWithSeparateCounts(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:old1"] = legacyRow(t, "old1", "h1")
	kv.m["nrv:error_tests:s1"] = suiteJSON(t, "s1", "t1")
	dupe := suiteJSON(t, "s2", "t1")
	kv.m["graph:error_tests:s2"] = dupe
	kv.m["nrv:error_tests:s2"] = dupe

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rewritten != 1 || rep.Scanned != 1 {
		t.Fatalf("node counts must not include suites: scanned=%d rewritten=%d", rep.Scanned, rep.Rewritten)
	}
	if rep.SuiteScanned != 3 || rep.SuiteRewritten != 1 || rep.SuiteDuplicatesRemoved != 1 || rep.SuiteAlreadyNew != 1 {
		t.Fatalf("suite counts wrong: %+v", rep)
	}
	if string(kv.m["graph:error_tests:s1"]) != string(suiteJSON(t, "s1", "t1")) {
		t.Fatal("suite row must be copied byte-for-byte to the new prefix")
	}
	if _, ok := kv.m["nrv:error_tests:s1"]; ok {
		t.Fatal("legacy suite row must be deleted after the move")
	}
	if _, ok := kv.m["nrv:error_tests:s2"]; ok {
		t.Fatal("identical legacy duplicate must be removed")
	}
	if _, ok := kv.m["migration:announcement_ref"]; !ok {
		t.Fatal("marker must be written")
	}
	marker := string(kv.m["migration:announcement_ref"])
	for _, want := range []string{`"suite_rewritten":1`, `"suite_duplicates_removed":1`, `"rewritten":1`} {
		if !strings.Contains(marker, want) {
			t.Fatalf("marker missing %s: %s", want, marker)
		}
	}
}

// A legacy suite that differs from the current row is never overwritten.
func TestSuiteMigrationConflictLeavesBothRows(t *testing.T) {
	kv := newMemKV()
	cur := suiteJSON(t, "s1", "new-1")
	leg := suiteJSON(t, "s1", "old-1")
	kv.m["graph:error_tests:s1"] = cur
	kv.m["nrv:error_tests:s1"] = leg

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedConflict != 1 || rep.SuiteRewritten != 0 {
		t.Fatalf("want one suite conflict, got %+v", rep)
	}
	if string(kv.m["graph:error_tests:s1"]) != string(cur) || string(kv.m["nrv:error_tests:s1"]) != string(leg) {
		t.Fatal("conflicting suite rows must both be left byte-for-byte")
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0] != "nrv:error_tests:s1" {
		t.Fatalf("skipped key not reported: %v", rep.Skipped)
	}
}

// Unknown fields fail the strict decode; the row is counted and left in place.
func TestSuiteMigrationStrictDecodeSkipsUnknownField(t *testing.T) {
	kv := newMemKV()
	bad := []byte(`{"error_node_id":"s1","tests":[],"status":"collecting","version":1,"updated_at":"2026-11-01T00:00:00Z","surprise":true}`)
	kv.m["nrv:error_tests:s1"] = bad

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedUnreadable != 1 || rep.SuiteRewritten != 0 {
		t.Fatalf("want one unreadable suite, got %+v", rep)
	}
	if string(kv.m["nrv:error_tests:s1"]) != string(bad) {
		t.Fatal("unreadable suite must stay in place, unchanged")
	}
}

// A row whose error_node_id differs from its key is unreadable, not moved.
func TestSuiteMigrationIDMismatchIsUnreadable(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_tests:s1"] = suiteJSON(t, "someone-else", "t1")

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedUnreadable != 1 || rep.SuiteRewritten != 0 {
		t.Fatalf("id mismatch must be skipped, got %+v", rep)
	}
	if _, ok := kv.m["graph:error_tests:s1"]; ok {
		t.Fatal("mismatched suite must not be written under the new key")
	}
}

// After the marker, a second run changes nothing, including suite rows added under the legacy prefix.
func TestSuiteMigrationSecondRunRefused(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_tests:s1"] = suiteJSON(t, "s1", "t1")
	if _, err := locator.MigrateAnnouncementRef(kv, time.Now()); err != nil {
		t.Fatal(err)
	}
	kv.m["nrv:error_tests:s9"] = suiteJSON(t, "s9", "t1")

	_, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != locator.ErrAnnouncementRefMigrationDone {
		t.Fatalf("second run must be refused, got %v", err)
	}
	if _, ok := kv.m["nrv:error_tests:s9"]; !ok {
		t.Fatal("refused run must not touch suite rows")
	}
}

// An interrupted run can leave the new row written and the legacy row still present.
// Repeating the run resolves the identical pair by removing the legacy row.
func TestSuiteMigrationResumesAfterInterruptedRun(t *testing.T) {
	kv := newMemKV()
	row := suiteJSON(t, "s1", "t1")
	kv.m["graph:error_tests:s1"] = row
	kv.m["nrv:error_tests:s1"] = row

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteDuplicatesRemoved != 1 {
		t.Fatalf("resume must remove the identical legacy row, got %+v", rep)
	}
	if _, ok := kv.m["nrv:error_tests:s1"]; ok {
		t.Fatal("legacy row must be gone after resume")
	}
	if string(kv.m["graph:error_tests:s1"]) != string(row) {
		t.Fatal("current row must be kept unchanged")
	}
}

// A legacy suite whose tests are all in the current row, unchanged, is migrated:
// the legacy row is removed and nothing is skipped, so the marker is written.
// This is the state ContributeTest leaves behind on a legacy-only suite.
func TestSuiteMigrationSupersetLegacyRowIsNotSkipped(t *testing.T) {
	kv := newMemKV()
	kv.m["graph:error_tests:s1"] = suiteJSON(t, "s1", "t1", "t2")
	kv.m["nrv:error_tests:s1"] = suiteJSON(t, "s1", "t1")

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSupersededRemoved != 1 || rep.SuiteSkippedConflict != 0 || len(rep.Skipped) != 0 {
		t.Fatalf("superset legacy row must be removed, not skipped: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_tests:s1"]; ok {
		t.Fatal("superseded legacy row must be deleted")
	}
	if string(kv.m["graph:error_tests:s1"]) != string(suiteJSON(t, "s1", "t1", "t2")) {
		t.Fatal("current row must be kept unchanged")
	}
	if !rep.MarkerWritten {
		t.Fatal("marker must be written when no row is skipped")
	}
	if _, ok := kv.m["migration:announcement_ref"]; !ok {
		t.Fatal("marker must be in the store")
	}
}

// A legacy test the current row lacks is a real divergence: both rows stay,
// the run is not clean, and no marker is written.
func TestSuiteMigrationDivergentLegacyRowStaysConflict(t *testing.T) {
	kv := newMemKV()
	cur := suiteJSON(t, "s1", "t1")
	leg := suiteJSON(t, "s1", "t1", "t2")
	kv.m["graph:error_tests:s1"] = cur
	kv.m["nrv:error_tests:s1"] = leg

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedConflict != 1 || rep.SuiteSupersededRemoved != 0 || rep.MarkerWritten {
		t.Fatalf("divergent legacy row must be a conflict and block the marker: %+v", rep)
	}
	if string(kv.m["graph:error_tests:s1"]) != string(cur) || string(kv.m["nrv:error_tests:s1"]) != string(leg) {
		t.Fatal("divergent rows must both be left byte-for-byte")
	}
	if _, ok := kv.m["migration:announcement_ref"]; ok {
		t.Fatal("marker must not be written while a divergent row is skipped")
	}
}

// A test id present in both rows with different content is not the same test,
// so the legacy row is not treated as contained.
func TestSuiteMigrationSameIDDifferentContentStaysConflict(t *testing.T) {
	kv := newMemKV()
	cur := locator.ErrorTestSuite{ErrorNodeID: "s1", Status: locator.SuiteCollecting, Version: 1,
		Tests: []locator.ErrorTestCase{{ID: "t1", Input: "in", Expected: "changed", AuthorID: "dev"}, {ID: "t2", Input: "in", Expected: "out", AuthorID: "dev"}}}
	leg := locator.ErrorTestSuite{ErrorNodeID: "s1", Status: locator.SuiteCollecting, Version: 1,
		Tests: []locator.ErrorTestCase{{ID: "t1", Input: "in", Expected: "out", AuthorID: "dev"}}}
	curRaw, _ := json.Marshal(cur)
	legRaw, _ := json.Marshal(leg)
	kv.m["graph:error_tests:s1"] = curRaw
	kv.m["nrv:error_tests:s1"] = legRaw

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedConflict != 1 || rep.SuiteSupersededRemoved != 0 || rep.MarkerWritten {
		t.Fatalf("same id with different content must be a conflict: %+v", rep)
	}
}

// sealedSuiteJSON builds a sealed suite row with the given seal metadata.
func sealedSuiteJSON(t *testing.T, nodeID string, sealedAt time.Time, hash string, version int, testIDs ...string) []byte {
	t.Helper()
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

var sealTime = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

var eightIDs = []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8"}

// A sealed legacy row whose suite_hash differs from the current row is seal
// metadata the current row does not match. Dropping it would lose the hash that
// solution attestations verify against, so both rows stay and the run is a conflict.
func TestSuiteMigrationSealedLegacyDifferentHashStaysConflict(t *testing.T) {
	kv := newMemKV()
	cur := sealedSuiteJSON(t, "s1", sealTime, "CURRENTHASH", 2, eightIDs...)
	leg := sealedSuiteJSON(t, "s1", sealTime, "LEGACYHASH", 1, eightIDs...)
	kv.m["graph:error_tests:s1"] = cur
	kv.m["nrv:error_tests:s1"] = leg

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedConflict != 1 || rep.SuiteSupersededRemoved != 0 || rep.MarkerWritten {
		t.Fatalf("sealed legacy row with a different suite_hash must be a conflict: %+v", rep)
	}
	if string(kv.m["nrv:error_tests:s1"]) != string(leg) || string(kv.m["graph:error_tests:s1"]) != string(cur) {
		t.Fatal("both rows must be left byte-for-byte")
	}
	if _, ok := kv.m["migration:announcement_ref"]; ok {
		t.Fatal("marker must not be written while the sealed legacy row is skipped")
	}
}

// Same tests and same suite_hash, but the legacy row's version differs from the
// current row. Version is seal metadata, so this is also a conflict.
func TestSuiteMigrationSealedLegacyDifferentVersionStaysConflict(t *testing.T) {
	kv := newMemKV()
	cur := sealedSuiteJSON(t, "s1", sealTime, "SAMEHASH", 2, eightIDs...)
	leg := sealedSuiteJSON(t, "s1", sealTime, "SAMEHASH", 1, eightIDs...)
	kv.m["graph:error_tests:s1"] = cur
	kv.m["nrv:error_tests:s1"] = leg

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSkippedConflict != 1 || rep.SuiteSupersededRemoved != 0 || rep.MarkerWritten {
		t.Fatalf("sealed legacy row with a different version must be a conflict: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_tests:s1"]; !ok {
		t.Fatal("legacy row must be kept")
	}
}

// A legacy row whose seal metadata matches the current row exactly, with its tests
// contained in the current row, carries nothing the current row lacks: it is superseded.
func TestSuiteMigrationSealedLegacyMatchingMetadataIsSuperseded(t *testing.T) {
	kv := newMemKV()
	kv.m["graph:error_tests:s1"] = sealedSuiteJSON(t, "s1", sealTime, "SAMEHASH", 1, eightIDs...)
	kv.m["nrv:error_tests:s1"] = sealedSuiteJSON(t, "s1", sealTime, "SAMEHASH", 1, eightIDs[:7]...)

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSupersededRemoved != 1 || rep.SuiteSkippedConflict != 0 || !rep.MarkerWritten {
		t.Fatalf("matching seal metadata must allow supersession: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_tests:s1"]; ok {
		t.Fatal("superseded legacy row must be deleted")
	}
}

// A legacy row with no seal metadata (collecting, no hash, no sealed_at) is superseded
// even when the current row is sealed: there is nothing on the legacy row to lose.
func TestSuiteMigrationCollectingLegacyWithSealedCurrentIsSuperseded(t *testing.T) {
	kv := newMemKV()
	kv.m["graph:error_tests:s1"] = sealedSuiteJSON(t, "s1", sealTime, "CURRENTHASH", 2, eightIDs...)
	kv.m["nrv:error_tests:s1"] = suiteJSON(t, "s1", eightIDs[:7]...)

	rep, err := locator.MigrateAnnouncementRef(kv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.SuiteSupersededRemoved != 1 || rep.SuiteSkippedConflict != 0 || !rep.MarkerWritten {
		t.Fatalf("collecting legacy row without seal metadata must be superseded: %+v", rep)
	}
}
