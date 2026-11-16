package locator_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/locator"

	"google.golang.org/protobuf/encoding/protojson"
)

// memKV is an in-memory MigrationKV for the migration tests.
type memKV struct {
	m map[string][]byte
}

func newMemKV() *memKV { return &memKV{m: map[string][]byte{}} }

func (k *memKV) Get(key []byte) ([]byte, error) {
	v, ok := k.m[string(key)]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}
func (k *memKV) Put(key, value []byte) error {
	k.m[string(key)] = append([]byte(nil), value...)
	return nil
}
func (k *memKV) Delete(key []byte) error { delete(k.m, string(key)); return nil }
func (k *memKV) Has(key []byte) (bool, error) {
	_, ok := k.m[string(key)]
	return ok, nil
}
func (k *memKV) ScanPrefix(prefix []byte) (map[string][]byte, error) {
	out := map[string][]byte{}
	for key, v := range k.m {
		if strings.HasPrefix(key, string(prefix)) {
			out[key] = v
		}
	}
	return out, nil
}

// errorNodeJSON returns a valid stored ErrorNode row for id, as protojson writes it.
func errorNodeJSON(t *testing.T, id, ref string) []byte {
	t.Helper()
	node := &locator.ErrorNode{Id: id, ErrorType: "leak", Description: "fd leak", Severity: 3, AnnouncementRef: ref}
	raw, err := protojson.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// legacyRow is a stored row in the pre-rename shape: the field is nrvSource.
func legacyRow(t *testing.T, id, value string) []byte {
	t.Helper()
	var doc map[string]interface{}
	if err := json.Unmarshal(errorNodeJSON(t, id, ""), &doc); err != nil {
		t.Fatal(err)
	}
	delete(doc, "announcementRef")
	doc["nrvSource"] = value
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

var when = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

func TestMigrationMovesLegacyRowAndRenamesField(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:e1"] = legacyRow(t, "e1", "hash-1")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rewritten != 1 || rep.OldFieldRows != 1 || rep.Scanned != 1 || rep.SkippedUnreadable != 0 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_nodes:e1"]; ok {
		t.Fatal("legacy key must be gone after the move")
	}
	raw, ok := kv.m["graph:error_nodes:e1"]
	if !ok {
		t.Fatal("row must be under graph:error_nodes:")
	}
	if strings.Contains(string(raw), "nrvSource") || strings.Contains(string(raw), "nrv_source") {
		t.Fatalf("legacy field name must not remain: %s", raw)
	}
	node := &locator.ErrorNode{}
	if err := protojson.Unmarshal(raw, node); err != nil {
		t.Fatalf("migrated row must pass the strict loader: %v", err)
	}
	if node.AnnouncementRef != "hash-1" || node.Id != "e1" {
		t.Fatalf("value not carried over: %+v", node)
	}
	if _, ok := kv.m["migration:announcement_ref"]; !ok {
		t.Fatal("marker must be written")
	}
}

func TestMigrationAcceptsSnakeLegacyField(t *testing.T) {
	kv := newMemKV()
	var doc map[string]interface{}
	_ = json.Unmarshal(errorNodeJSON(t, "e2", ""), &doc)
	delete(doc, "announcementRef")
	doc["nrv_source"] = "hash-2"
	raw, _ := json.Marshal(doc)
	kv.m["nrv:error_nodes:e2"] = raw

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OldFieldRows != 1 || rep.Rewritten != 1 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	node := &locator.ErrorNode{}
	if err := protojson.Unmarshal(kv.m["graph:error_nodes:e2"], node); err != nil || node.AnnouncementRef != "hash-2" {
		t.Fatalf("snake legacy field not renamed: %v %+v", err, node)
	}
}

func TestMigrationSecondRunRefused(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:e1"] = legacyRow(t, "e1", "h")
	if _, err := locator.MigrateAnnouncementRef(kv, when); err != nil {
		t.Fatal(err)
	}
	before := len(kv.m)
	snapshot := map[string][]byte{}
	for k, v := range kv.m {
		snapshot[k] = v
	}

	_, err := locator.MigrateAnnouncementRef(kv, when)
	if !errors.Is(err, locator.ErrAnnouncementRefMigrationDone) {
		t.Fatalf("second run must be refused, got %v", err)
	}
	if len(kv.m) != before {
		t.Fatal("refused run must not change the store")
	}
	for k, v := range snapshot {
		if string(kv.m[k]) != string(v) {
			t.Fatalf("refused run changed %s", k)
		}
	}
}

func TestMigrationMixedOldAndNewRows(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:old"] = legacyRow(t, "old", "h-old")
	kv.m["graph:error_nodes:new"] = errorNodeJSON(t, "new", "h-new")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Scanned != 2 || rep.Rewritten != 1 || rep.AlreadyNew != 1 || rep.OldFieldRows != 1 || rep.SkippedUnreadable != 0 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	for _, key := range []string{"graph:error_nodes:old", "graph:error_nodes:new"} {
		if _, ok := kv.m[key]; !ok {
			t.Fatalf("%s missing after migration", key)
		}
	}
	if len(kv.m) != 3 {
		t.Fatalf("expected 2 rows and the marker, got %d keys", len(kv.m))
	}
}

func TestMigrationEmptyStore(t *testing.T) {
	kv := newMemKV()
	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Scanned != 0 || rep.Rewritten != 0 || rep.AlreadyNew != 0 || rep.SkippedUnreadable != 0 || len(rep.Skipped) != 0 {
		t.Fatalf("empty store must report zeros, got %+v", rep)
	}
	if _, err := locator.MigrateAnnouncementRef(kv, when); !errors.Is(err, locator.ErrAnnouncementRefMigrationDone) {
		t.Fatalf("empty store second run must be refused, got %v", err)
	}
}

func TestMigrationSkipsCorruptRowAndCountsIt(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:bad"] = []byte("{not json")
	kv.m["nrv:error_nodes:good"] = legacyRow(t, "good", "h")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SkippedUnreadable != 1 || rep.Rewritten != 1 || rep.Scanned != 2 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if got := string(kv.m["nrv:error_nodes:bad"]); got != "{not json" {
		t.Fatal("corrupt row must be left exactly as it was")
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0] != "nrv:error_nodes:bad" {
		t.Fatalf("skipped keys wrong: %v", rep.Skipped)
	}
}

func TestMigrationStrictDecodeSkipsUnknownField(t *testing.T) {
	kv := newMemKV()
	var doc map[string]interface{}
	_ = json.Unmarshal(errorNodeJSON(t, "u", ""), &doc)
	doc["bogusField"] = 1
	raw, _ := json.Marshal(doc)
	kv.m["nrv:error_nodes:u"] = raw

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SkippedUnreadable != 1 || rep.Rewritten != 0 {
		t.Fatalf("unknown field must fail the strict decode and be skipped: %+v", rep)
	}
}

func TestMigrationRemovesIdenticalDuplicateAndKeepsConflict(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:dup"] = legacyRow(t, "dup", "same")
	kv.m["graph:error_nodes:dup"] = errorNodeJSON(t, "dup", "same")
	kv.m["nrv:error_nodes:clash"] = legacyRow(t, "clash", "one")
	kv.m["graph:error_nodes:clash"] = errorNodeJSON(t, "clash", "two")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.DuplicatesRemoved != 1 || rep.SkippedConflict != 1 || rep.Rewritten != 0 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_nodes:dup"]; ok {
		t.Fatal("identical legacy duplicate must be removed")
	}
	if _, ok := kv.m["nrv:error_nodes:clash"]; !ok {
		t.Fatal("conflicting legacy row must be left in place")
	}
	node := &locator.ErrorNode{}
	_ = protojson.Unmarshal(kv.m["graph:error_nodes:clash"], node)
	if node.AnnouncementRef != "two" {
		t.Fatal("conflicting current row must not be overwritten")
	}
}

func TestMigrationResumesAfterInterruptedRun(t *testing.T) {
	// Simulate a run that wrote the new key and stopped before deleting the old one.
	kv := newMemKV()
	kv.m["nrv:error_nodes:r"] = legacyRow(t, "r", "h")
	kv.m["graph:error_nodes:r"] = errorNodeJSON(t, "r", "h")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if rep.DuplicatesRemoved != 1 {
		t.Fatalf("resume must clear the legacy copy: %+v", rep)
	}
	if _, ok := kv.m["nrv:error_nodes:r"]; ok {
		t.Fatal("legacy copy must be gone")
	}
}

// A run that skips a row writes no marker and reports MarkerWritten false, so
// the legacy fallback stays in force and the run can be repeated.
func TestMarkerNotWrittenWhenRowsSkipped(t *testing.T) {
	kv := newMemKV()
	kv.m["nrv:error_nodes:bad"] = []byte("{not json")
	kv.m["nrv:error_nodes:good"] = legacyRow(t, "good", "h")

	rep, err := locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatalf("a run with skipped rows returns normally, got %v", err)
	}
	if rep.MarkerWritten {
		t.Fatal("MarkerWritten must be false when rows were skipped")
	}
	if _, ok := kv.m[locator.AnnouncementRefMigrationMarker]; ok {
		t.Fatal("marker must not be written when rows were skipped")
	}
	if _, err := locator.MigrateAnnouncementRef(kv, when); err != nil {
		t.Fatalf("repeat with the skipped row still present: %v", err)
	}
	if _, ok := kv.m[locator.AnnouncementRefMigrationMarker]; ok {
		t.Fatal("marker must still be absent on repeat")
	}

	// Once the bad row is gone, a run is clean and the marker is written.
	delete(kv.m, "nrv:error_nodes:bad")
	rep, err = locator.MigrateAnnouncementRef(kv, when)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.MarkerWritten {
		t.Fatalf("clean run must write the marker: %+v", rep)
	}
	if _, ok := kv.m[locator.AnnouncementRefMigrationMarker]; !ok {
		t.Fatal("marker missing after a clean run")
	}
}
