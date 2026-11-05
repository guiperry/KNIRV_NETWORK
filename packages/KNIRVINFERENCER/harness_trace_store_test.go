package inferencer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mkTrace(i int) Trace {
	return Trace{ID: fmt.Sprintf("t%04d", i), Time: time.Unix(1_750_000_000+int64(i), 0).UTC(), Kind: KindGenerate, Chosen: "heart", Candidates: []Candidate{{Model: "heart"}}}
}

func TestTracesPersistAndReloadNewestFirst(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "traces") // a directory the store creates is private
	s, err := NewTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		s.Append(mkTrace(i))
	}
	got, err := s.Read(0)
	if err != nil || len(got) != 5 || got[0].ID != "t0004" || got[4].ID != "t0000" {
		t.Fatalf("%v %+v", err, got)
	}
	if got, _ := s.Read(2); len(got) != 2 || got[0].ID != "t0004" || got[1].ID != "t0003" {
		t.Fatalf("limit: %+v", got)
	}
	_ = s.Close()
	s2, _ := NewTraceStore(dir)
	s2.Append(mkTrace(5)) // appends continue the same file after a restart
	if got, _ := s2.Read(0); len(got) != 6 || got[0].ID != "t0005" {
		t.Fatalf("%+v", got)
	}
	// Files are private.
	st, _ := os.Stat(filepath.Join(dir, "traces.jsonl"))
	di, _ := os.Stat(dir)
	if st.Mode().Perm() != 0o600 || di.Mode().Perm() != 0o700 {
		t.Fatalf("file %v dir %v", st.Mode().Perm(), di.Mode().Perm())
	}
}

func TestTraceFilesRotateAndOldOnesArePruned(t *testing.T) {
	dir := t.TempDir()
	s, err := NewTraceStore(dir, WithTraceFileLimits(600, 3))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		s.Append(mkTrace(i))
		time.Sleep(time.Millisecond) // rotated names are nanosecond stamps
	}
	rot, _ := s.rotated()
	if len(rot) != 3 {
		t.Fatalf("%d rotated files kept, want 3", len(rot))
	}
	for _, f := range rot {
		if st, _ := os.Stat(f); st.Size() > 700 {
			t.Fatalf("%s is %d bytes", f, st.Size())
		}
	}
	got, _ := s.Read(0)
	if len(got) == 0 || len(got) >= 60 || got[0].ID != "t0059" {
		t.Fatalf("%d traces, newest %v", len(got), got[0].ID)
	}
	// Newest first and strictly contiguous across file boundaries.
	for i := 1; i < len(got); i++ {
		if got[i].Time.After(got[i-1].Time) || got[i-1].Time.Sub(got[i].Time) != time.Second {
			t.Fatalf("order broken at %d: %s then %s", i, got[i-1].ID, got[i].ID)
		}
	}
	// Append-only: an old line is never rewritten.
	if all, _ := os.ReadFile(filepath.Join(dir, "traces.jsonl")); !strings.HasSuffix(string(all), "\n") {
		t.Fatal("active file does not end in a newline")
	}
}

func TestTornAndCorruptLinesAreSkipped(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewTraceStore(dir)
	s.Append(mkTrace(1))
	s.Append(mkTrace(2))
	_ = s.Close()
	f, _ := os.OpenFile(filepath.Join(dir, "traces.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	_, _ = f.WriteString("{not json}\n{\"id\":\"\"}\n{\"id\":\"torn\",\"kind\":\"gen") // a crash mid-write
	_ = f.Close()
	s2, _ := NewTraceStore(dir)
	got, err := s2.Read(0)
	if err != nil || len(got) != 2 || got[0].ID != "t0002" {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestAPersistenceFailureNeverFailsACall(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewTraceStore(dir)
	_ = s.Close() // writes now fail
	m := up("heart")
	h := NewHarness([]HarnessModel{m}, WithTraceStore(s), WithRoutes(map[Kind][]string{KindGenerate: {"heart"}}))
	if _, _, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "hi"}); err != nil {
		t.Fatalf("the call failed because the trace could not be written: %v", err)
	}
	if n, msg := s.Errors(); n != 1 || msg == "" {
		t.Fatalf("%d %q", n, msg)
	}
	if got := h.Traces(5); len(got) != 1 {
		t.Fatalf("the in-memory ring must still hold it: %+v", got)
	}
}

func TestAnOversizedTraceKeepsItsRecordAndDropsFreeText(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewTraceStore(dir)
	big := mkTrace(1)
	big.Notes = []string{strings.Repeat("n", 70<<10)}
	big.Error = strings.Repeat("e", 70<<10)
	s.Append(big)
	got, _ := s.Read(0)
	if len(got) != 1 || got[0].ID != "t0001" || len(got[0].Notes) != 0 || len(got[0].Error) > 300 {
		t.Fatalf("%+v", got)
	}
	if n, _ := s.Errors(); n != 0 {
		t.Fatalf("errors: %d", n)
	}
}

func TestHarnessRefillsItsRingFromTheStoreInOrder(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewTraceStore(dir)
	for i := 0; i < 10; i++ {
		s.Append(mkTrace(i))
	}
	h := NewHarness([]HarnessModel{up("heart")}, WithTraceStore(s), WithTraceCapacity(4), WithRoutes(map[Kind][]string{KindGenerate: {"heart"}}))
	got := h.Traces(0)
	if len(got) != 4 || got[0].ID != "t0009" || got[3].ID != "t0006" {
		t.Fatalf("%+v", got)
	}
	// New traces go on top and are persisted.
	if _, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "hi"}); err != nil || h.Traces(1)[0].ID != tr.ID {
		t.Fatal(err)
	}
	if all, _ := s.Read(0); len(all) != 11 || all[0].Kind != KindGenerate {
		t.Fatalf("%d stored", len(all))
	}
}
