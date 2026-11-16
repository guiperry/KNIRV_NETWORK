package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"

	"google.golang.org/protobuf/encoding/protojson"
)

// fixtureDir returns an existing, empty store directory. Tests run with TMPDIR
// pointed at scratch, so nothing here touches a real database.
func fixtureDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "graph.db")
	s, err := storage.NewBluntDBStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func put(t *testing.T, dir string, kv map[string][]byte) {
	t.Helper()
	s, err := storage.NewBluntDBStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for k, v := range kv {
		if err := s.Put([]byte(k), v); err != nil {
			t.Fatal(err)
		}
	}
}

func row(t *testing.T, id, ref string, legacy bool) []byte {
	t.Helper()
	raw, err := protojson.Marshal(&locator.ErrorNode{Id: id, ErrorType: "leak", Severity: 2, AnnouncementRef: ref})
	if err != nil {
		t.Fatal(err)
	}
	if !legacy {
		return raw
	}
	var doc map[string]interface{}
	_ = json.Unmarshal(raw, &doc)
	delete(doc, "announcementRef")
	doc["nrvSource"] = ref
	out, _ := json.Marshal(doc)
	return out
}

func has(t *testing.T, dir, key string) bool {
	t.Helper()
	s, err := storage.NewBluntDBStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ok, err := s.Has([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func runCmd(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRefusesWithoutDBFlag(t *testing.T) {
	code, _, errOut := runCmd()
	if code != 2 || !strings.Contains(errOut, "--db is required") {
		t.Fatalf("want usage refusal, got %d %q", code, errOut)
	}
}

func TestRefusesNonexistentDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	code, _, errOut := runCmd("--db", missing)
	if code != 2 || !strings.Contains(errOut, "refusing to create") {
		t.Fatalf("want refusal, got %d %q", code, errOut)
	}
}

func TestMigratesFixtureAndPrintsPath(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_nodes:old1": row(t, "old1", "h1", true),
		"nrv:error_nodes:bad":  []byte("{garbage"),
		"graph:error_nodes:n2": row(t, "n2", "h2", false),
	})

	// The corrupt row is skipped, so the run exits 4 and writes no marker.
	code, out, errOut := runCmd("--db", dir)
	if code != 4 {
		t.Fatalf("exit %d, want 4 for a run with skipped rows; stderr %q", code, errOut)
	}
	if !strings.Contains(out, "opening bluntdb store: "+dir) {
		t.Fatalf("must print the opened path, got %q", out)
	}
	if !strings.Contains(out, "scanned=3 rewritten=1 already-new=1 skipped-unreadable=1") {
		t.Fatalf("counts wrong: %q", out)
	}
	if !strings.Contains(out, "skipped: nrv:error_nodes:bad") {
		t.Fatalf("skipped key not reported: %q", out)
	}
	if strings.Contains(out, "marker ") || !strings.Contains(errOut, "NOT written") {
		t.Fatalf("marker must not be reported as written; out %q err %q", out, errOut)
	}
	if !has(t, dir, "graph:error_nodes:old1") || has(t, dir, "nrv:error_nodes:old1") {
		t.Fatal("old1 must be moved to the new prefix")
	}
	if !has(t, dir, "nrv:error_nodes:bad") {
		t.Fatal("corrupt row must stay in place")
	}
	if has(t, dir, "migration:announcement_ref") {
		t.Fatal("marker must not be written while rows are skipped")
	}
}

// A run with skipped rows stays unmarked and repeatable: the same exit and no
// marker on each attempt, and the clean rows are not moved twice.
func TestSkippedRunStaysUnmarkedAndRepeatable(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_nodes:bad": []byte("{garbage"),
		"nrv:error_nodes:a":   row(t, "a", "ha", true),
	})
	for attempt := 1; attempt <= 2; attempt++ {
		code, out, _ := runCmd("--db", dir)
		if code != 4 {
			t.Fatalf("attempt %d: exit %d, want 4", attempt, code)
		}
		if !strings.Contains(out, "skipped: nrv:error_nodes:bad") {
			t.Fatalf("attempt %d: skipped key not reported: %q", attempt, out)
		}
		if has(t, dir, "migration:announcement_ref") {
			t.Fatalf("attempt %d: marker written with a skipped row", attempt)
		}
	}
	if !has(t, dir, "graph:error_nodes:a") || has(t, dir, "nrv:error_nodes:a") {
		t.Fatal("clean row must stay moved across repeats")
	}
	if !has(t, dir, "nrv:error_nodes:bad") {
		t.Fatal("corrupt row must stay in place")
	}
}

// After the corrupt row is fixed, a clean re-run writes the marker; a further
// run is then refused.
func TestCleanRerunAfterFixingCorruptRowWritesMarker(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{
		"nrv:error_nodes:bad": []byte("{garbage"),
		"nrv:error_nodes:a":   row(t, "a", "ha", true),
	})
	if code, _, _ := runCmd("--db", dir); code != 4 {
		t.Fatalf("first run must exit 4 with a corrupt row, got %d", code)
	}

	// Fix the corrupt row in place: a valid legacy row under the same key.
	put(t, dir, map[string][]byte{"nrv:error_nodes:bad": row(t, "bad", "hb", true)})

	code, out, errOut := runCmd("--db", dir)
	if code != 0 {
		t.Fatalf("clean re-run exit %d, want 0; stderr %q", code, errOut)
	}
	if strings.Contains(out, "skipped:") {
		t.Fatalf("clean re-run must report no skipped rows: %q", out)
	}
	if !strings.Contains(out, "marker \"migration:announcement_ref\" written") {
		t.Fatalf("marker not reported as written: %q", out)
	}
	if !has(t, dir, "migration:announcement_ref") {
		t.Fatal("marker must be written by the clean re-run")
	}
	if !has(t, dir, "graph:error_nodes:bad") || has(t, dir, "nrv:error_nodes:bad") {
		t.Fatal("fixed row must move to the new prefix")
	}

	if code, _, _ := runCmd("--db", dir); code != 3 {
		t.Fatalf("run after the marker must be refused with exit 3, got %d", code)
	}
}

func TestSecondRunRefused(t *testing.T) {
	dir := fixtureDir(t)
	put(t, dir, map[string][]byte{"nrv:error_nodes:a": row(t, "a", "h", true)})
	if code, _, errOut := runCmd("--db", dir); code != 0 {
		t.Fatalf("first run exit %d: %s", code, errOut)
	}
	code, out, errOut := runCmd("--db", dir)
	if code != 3 || !strings.Contains(errOut, "refused") || strings.Contains(out, "scanned=") {
		t.Fatalf("second run must be refused with exit 3, got %d out=%q err=%q", code, out, errOut)
	}
}

func TestEmptyDBRuns(t *testing.T) {
	dir := fixtureDir(t)
	code, out, _ := runCmd("--db", dir)
	if code != 0 || !strings.Contains(out, "scanned=0 rewritten=0 already-new=0 skipped-unreadable=0") {
		t.Fatalf("empty DB: exit %d out %q", code, out)
	}
}

func TestLevelDBEngineFixture(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "lvl")
	s, err := storage.NewLevelDBStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put([]byte("nrv:error_nodes:l1"), row(t, "l1", "hl", true)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCmd("--db", dir, "--engine", "leveldb")
	if code != 0 || !strings.Contains(out, "rewritten=1") {
		t.Fatalf("leveldb: exit %d out %q err %q", code, out, errOut)
	}
}

func TestUnknownEngineRefused(t *testing.T) {
	dir := fixtureDir(t)
	code, _, errOut := runCmd("--db", dir, "--engine", "sqlite")
	if code != 1 || !strings.Contains(errOut, "unknown --engine") {
		t.Fatalf("got %d %q", code, errOut)
	}
}
