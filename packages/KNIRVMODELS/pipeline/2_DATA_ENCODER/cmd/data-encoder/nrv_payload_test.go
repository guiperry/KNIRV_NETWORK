package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"data-encoder/pkg/mapper"
	"data-encoder/pkg/nrvio"
	"data-encoder/pkg/schema"
	"data-encoder/pkg/tokenizer"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
	supervisionpipeline "knirvmodels/pkg/hashing/supervision/pipeline"
)

// TestWriteNRVCarriesAssertionPayloadPerFrame checks that every encoder frame
// becomes one self-contained assertion: one I-bracket plus a FramePayload whose
// token window is the asserted span at the frame's window start.
func TestWriteNRVCarriesAssertionPayloadPerFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "assertions.nrv")
	// The span {3, 8, 11} ends at WindowEnd 6, so its first token is at position 4.
	frames := []schema.TrainingFrame{
		{TargetTokenID: 11, WindowStart: 4, WindowEnd: 6, AssertionSpan: []int32{3, 8, 11}},
		{TargetTokenID: 12, WindowStart: 9, AssertionSpan: nil},
	}
	if err := writeNRV(path, frames, nil); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	if len(got) != len(frames) {
		t.Fatalf("got %d assertions, want %d", len(got), len(frames))
	}
	for i, a := range got {
		if len(a.Brackets) != 1 || a.Payload == nil || a.Payload.TokenWindow == nil {
			t.Fatalf("assertion %d: brackets=%d payload=%v", i, len(a.Brackets), a.Payload)
		}
		if a.Brackets[0].GoldenSeed != uint32(frames[i].TargetTokenID) {
			t.Fatalf("assertion %d: golden seed %d, want target %d", i, a.Brackets[0].GoldenSeed, frames[i].TargetTokenID)
		}
		tw := a.Payload.TokenWindow
		if want := uint32(tokenWindowStart(frames[i])); tw.Start != want || tw.Brackets != 1 {
			t.Fatalf("assertion %d: start=%d want %d, brackets=%d", i, tw.Start, want, tw.Brackets)
		}
	}
	if want := []uint32{3, 8, 11}; !reflect.DeepEqual(got[0].Payload.TokenWindow.Tokens, want) {
		t.Fatalf("span tokens = %v, want %v", got[0].Payload.TokenWindow.Tokens, want)
	}
	if got[1].Payload.TokenWindow.Tokens != nil {
		t.Fatalf("empty span should carry no tokens, got %v", got[1].Payload.TokenWindow.Tokens)
	}
}

func TestWriteNRVRefusesNegativeWindowStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.nrv")
	if err := writeNRV(path, []schema.TrainingFrame{{WindowStart: -1}}, nil); err == nil {
		t.Fatal("negative window start written")
	}
}

// assertionRun builds one window per target in [start, end] of a record: each
// window's asserted span runs from start to its target, as assertionSpanForWindow
// produces it.
func assertionRun(source string, chunk int32, start, end int32) []schema.TrainingFrame {
	var out []schema.TrainingFrame
	for e := start; e <= end; e++ {
		span := make([]int32, e-start+1)
		for k := range span {
			span[k] = 100 + start + int32(k)
		}
		ws := e - 2
		if ws < 0 {
			ws = 0
		}
		out = append(out, schema.TrainingFrame{
			SourceFile:    source,
			ChunkID:       chunk,
			WindowStart:   ws,
			WindowEnd:     e,
			TargetTokenID: e,
			AssertionSpan: span,
		})
	}
	return out
}

// TestWriteNRVGroupsAssertionWindowsIntoOneFrame checks that the windows of one
// assertion share a single frame holding all of its brackets, that a new assertion
// starts a new frame, and that a window with no span stands alone.
func TestWriteNRVGroupsAssertionWindowsIntoOneFrame(t *testing.T) {
	var frames []schema.TrainingFrame
	frames = append(frames, assertionRun("a.txt", 1, 10, 12)...) // assertion 1: 3 windows
	frames = append(frames, assertionRun("a.txt", 1, 20, 21)...) // assertion 2: 2 windows
	frames = append(frames, schema.TrainingFrame{SourceFile: "a.txt", ChunkID: 1, WindowStart: 3, WindowEnd: 5, TargetTokenID: 5})
	path := filepath.Join(t.TempDir(), "runs.nrv")
	if err := writeNRV(path, frames, nil); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	wantCounts := []int{3, 2, 1}
	if len(got) != len(wantCounts) {
		t.Fatalf("got %d frames, want %d", len(got), len(wantCounts))
	}
	for i, a := range got {
		if len(a.Brackets) != wantCounts[i] || a.Payload == nil || a.Payload.TokenWindow == nil {
			t.Fatalf("frame %d: brackets=%d payload=%v, want %d brackets with a token window", i, len(a.Brackets), a.Payload, wantCounts[i])
		}
		if int(a.Payload.TokenWindow.Brackets) != wantCounts[i] {
			t.Fatalf("frame %d: token_window.brackets=%d, want %d", i, a.Payload.TokenWindow.Brackets, wantCounts[i])
		}
	}
	// The first run's token window is its full span (up to target 12). Its first
	// token is the span start, position 10, not the window's context start (8).
	tw := got[0].Payload.TokenWindow
	if want := []uint32{110, 111, 112}; !reflect.DeepEqual(tw.Tokens, want) {
		t.Fatalf("run 1 tokens = %v, want %v", tw.Tokens, want)
	}
	if tw.Start != 10 {
		t.Fatalf("run 1 start = %d, want 10", tw.Start)
	}
	for k, b := range got[0].Brackets {
		if b.GoldenSeed != uint32(10+k) {
			t.Fatalf("run 1 bracket %d golden seed %d, want %d", k, b.GoldenSeed, 10+k)
		}
	}
}

// TestWriteNRVSplitsRunsAtBracketCap checks that a run longer than the per-frame
// cap is split, and that no frame exceeds the cap.
func TestWriteNRVSplitsRunsAtBracketCap(t *testing.T) {
	n := nrvio.MaxBracketsPerAssertion + 5
	frames := assertionRun("big.txt", 2, 0, int32(n-1))
	path := filepath.Join(t.TempDir(), "cap.nrv")
	if err := writeNRV(path, frames, nil); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	if len(got) != 2 || len(got[0].Brackets) != nrvio.MaxBracketsPerAssertion || len(got[1].Brackets) != 5 {
		t.Fatalf("frames = %d, sizes want [%d 5]", len(got), nrvio.MaxBracketsPerAssertion)
	}
}

// TestAssertionCapMatchesKNIRVBASE pins the local cap to KNIRVBASE's per-frame cap,
// so a frame the encoder writes is never refused by the canonical payload validator.
func TestAssertionCapMatchesKNIRVBASE(t *testing.T) {
	if nrvio.MaxBracketsPerAssertion != knirvnrv.MaxBracketsPerFrame {
		t.Fatalf("MaxBracketsPerAssertion=%d, KNIRVBASE MaxBracketsPerFrame=%d", nrvio.MaxBracketsPerAssertion, knirvnrv.MaxBracketsPerFrame)
	}
}

// fakeEmbeddings stands in for the embedding endpoint: every context gets the same
// fixed vector, so the real window generator can run offline.
type fakeEmbeddings struct{}

func (fakeEmbeddings) GetEmbedding(string) ([]float32, error) { return generateTestEmbedding(0.8), nil }
func (fakeEmbeddings) GetBatchEmbeddings(texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = generateTestEmbedding(0.8)
	}
	return out, nil
}
func (fakeEmbeddings) GetBatchSize() int              { return 1000 }
func (fakeEmbeddings) ValidateEndpoint() error        { return nil }
func (fakeEmbeddings) SetTimeout(timeout interface{}) {}

// TestProcessedRecordTokenWindowIndexesItsTokens runs the real generator
// (processSingleRecordWithSlidingWindow, real tokenizer and sliding windows) and
// checks every assertion written from it: the token window's first token is the
// token at its Start position, its last token is the frame's target, and the
// TRLSample carries the record's prompt and completion.
func TestProcessedRecordTokenWindowIndexesItsTokens(t *testing.T) {
	tk, err := tokenizer.New()
	if err != nil {
		t.Fatalf("tokenizer: %v", err)
	}
	record := &schema.MinedRecord{
		FileName:    "gen.txt",
		ChunkID:     7,
		Content:     "The quick brown fox jumps over the lazy dog. A second sentence follows, with more words after it here.",
		Instruction: "Describe the fox",
		Input:       "quick and brown",
		Output:      "A fox jumps over a dog.",
	}
	config := &Config{WindowSize: 4, WindowStride: 1, BatchSize: 1000}
	tp := mapper.NewTensorPacker(nil)
	var frames []schema.TrainingFrame
	var count int64
	if err := processSingleRecordWithSlidingWindow(record, &count, tp, tk, fakeEmbeddings{}, &frames, config, nil); err != nil {
		t.Fatalf("process record: %v", err)
	}
	if len(frames) == 0 {
		t.Fatal("no frames generated")
	}
	stream := tk.Encode(record.Content)
	path := filepath.Join(t.TempDir(), "gen.nrv")
	if err := writeNRV(path, frames, config.trl); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	sawMultiWindow := false
	for i, a := range got {
		tw := a.Payload.TokenWindow
		if len(tw.Tokens) == 0 {
			t.Fatalf("assertion %d: empty token window", i)
		}
		if len(a.Brackets) > 1 {
			sawMultiWindow = true
		}
		if int(tw.Start) >= len(stream) || stream[tw.Start] != int(tw.Tokens[0]) {
			t.Fatalf("assertion %d: Tokens[0]=%d is not the token at Start=%d", i, tw.Tokens[0], tw.Start)
		}
		last := int(tw.Start) + len(tw.Tokens) - 1
		if last >= len(stream) || stream[last] != int(tw.Tokens[len(tw.Tokens)-1]) {
			t.Fatalf("assertion %d: token window does not match the stream at its end", i)
		}
		ts := a.Payload.TRLSample
		if ts == nil || ts.Prompt != "Describe the fox\n\nquick and brown" || ts.Completion != "A fox jumps over a dog." {
			t.Fatalf("assertion %d: TRLSample = %+v", i, ts)
		}
		if ts.Chosen != "" || ts.Rejected != "" {
			t.Fatalf("assertion %d: encoder wrote chosen/rejected", i)
		}
	}
	if !sawMultiWindow {
		t.Fatal("no multi-bracket assertion was produced; the test did not exercise run grouping")
	}
}

// spanWindows builds one window per target in [start+len-1 .. end] for a span that
// starts at token start. Each window's span is start..target, as the generator
// produces it.
func spanWindows(source string, chunk int32, start int32, ends ...int32) []schema.TrainingFrame {
	var out []schema.TrainingFrame
	for _, e := range ends {
		span := make([]int32, e-start+1)
		for k := range span {
			span[k] = 100 + start + int32(k)
		}
		out = append(out, schema.TrainingFrame{
			SourceFile: source, ChunkID: chunk, WindowStart: start, WindowEnd: e,
			TargetTokenID: e, AssertionSpan: span,
		})
	}
	return out
}

// TestWriteNRVNeverMergesAcrossRecords kills the mutant that drops the SourceFile
// or ChunkID check: adjacent records that share a span start must stay separate.
func TestWriteNRVNeverMergesAcrossRecords(t *testing.T) {
	var frames []schema.TrainingFrame
	frames = append(frames, spanWindows("a.txt", 1, 10, 11, 12)...)
	frames = append(frames, spanWindows("b.txt", 1, 10, 13, 14)...) // other file, same start
	frames = append(frames, spanWindows("b.txt", 2, 10, 15)...)     // other chunk, same start
	path := filepath.Join(t.TempDir(), "records.nrv")
	if err := writeNRV(path, frames, nil); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	want := []int{2, 2, 1}
	if len(got) != len(want) {
		t.Fatalf("got %d frames, want %d", len(got), len(want))
	}
	for i, a := range got {
		if len(a.Brackets) != want[i] {
			t.Fatalf("frame %d: %d brackets, want %d", i, len(a.Brackets), want[i])
		}
	}
}

// TestWriteNRVRunEndsWhenWindowEndDrops checks that a window whose WindowEnd is
// lower than the previous one's does not join the run, since the run's last
// window is assumed to carry the longest span.
func TestWriteNRVRunEndsWhenWindowEndDrops(t *testing.T) {
	frames := spanWindows("a.txt", 1, 10, 12, 11) // same start, WindowEnd drops 12 -> 11
	path := filepath.Join(t.TempDir(), "drop.nrv")
	if err := writeNRV(path, frames, nil); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	if len(got) != 2 || len(got[0].Brackets) != 1 || len(got[1].Brackets) != 1 {
		t.Fatalf("frames = %d, want two single-bracket frames", len(got))
	}
	if want := []uint32{110, 111, 112}; !reflect.DeepEqual(got[0].Payload.TokenWindow.Tokens, want) {
		t.Fatalf("first frame tokens = %v, want %v", got[0].Payload.TokenWindow.Tokens, want)
	}
	if want := []uint32{110, 111}; !reflect.DeepEqual(got[1].Payload.TokenWindow.Tokens, want) {
		t.Fatalf("second frame tokens = %v, want %v", got[1].Payload.TokenWindow.Tokens, want)
	}
}

// encodeRecordsForTRL runs the real generator over the records in order, sharing
// one run config (and so one TRL side map), and returns the frames and that config.
func encodeRecordsForTRL(t *testing.T, records ...*schema.MinedRecord) ([]schema.TrainingFrame, *Config) {
	t.Helper()
	tk, err := tokenizer.New()
	if err != nil {
		t.Fatalf("tokenizer: %v", err)
	}
	config := &Config{WindowSize: 4, WindowStride: 1, BatchSize: 1000}
	tp := mapper.NewTensorPacker(nil)
	var frames []schema.TrainingFrame
	var count int64
	for _, r := range records {
		if err := processSingleRecordWithSlidingWindow(r, &count, tp, tk, fakeEmbeddings{}, &frames, config, nil); err != nil {
			t.Fatalf("process record %q: %v", r.Content, err)
		}
	}
	return frames, config
}

// readAssertionsForTRL writes frames with the TRL side and reads back the
// assertions.
func readAssertionsForTRL(t *testing.T, frames []schema.TrainingFrame, trl *trlSide) []nrvio.Assertion {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trl.nrv")
	if err := writeNRV(path, frames, trl); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	r, err := nrvio.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Assertions()
	if err != nil {
		t.Fatalf("assertions: %v", err)
	}
	return got
}

// TestTRLSameKeyRecordsKeepOwnText is the validator's probe as a repo test: two
// records share (FileName, ChunkID) but carry different Alpaca outputs. Each
// record's assertions must carry its own completion, and no assertion may carry
// another record's text. The per-record counts come from encoding each record
// alone, so a merge across the two records would also fail the test.
func TestTRLSameKeyRecordsKeepOwnText(t *testing.T) {
	r1 := &schema.MinedRecord{FileName: "alpaca.json", ChunkID: 0,
		Content: "The quick brown fox jumps over the lazy dog.", Instruction: "Q1", Output: "ANSWER-ONE"}
	r2 := &schema.MinedRecord{FileName: "alpaca.json", ChunkID: 0,
		Content: "A completely different sentence about cats and mats here.", Instruction: "Q2", Output: "ANSWER-TWO"}
	f1, c1 := encodeRecordsForTRL(t, r1)
	n1 := len(readAssertionsForTRL(t, f1, c1.trl))
	f2, c2 := encodeRecordsForTRL(t, r2)
	n2 := len(readAssertionsForTRL(t, f2, c2.trl))

	frames, config := encodeRecordsForTRL(t, r1, r2)
	got := readAssertionsForTRL(t, frames, config.trl)
	if len(got) != n1+n2 {
		t.Fatalf("combined run wrote %d assertions, want %d (record 1) + %d (record 2)", len(got), n1, n2)
	}
	for i, a := range got {
		want := "ANSWER-ONE"
		if i >= n1 {
			want = "ANSWER-TWO"
		}
		if a.Payload.TRLSample == nil || a.Payload.TRLSample.Completion != want {
			t.Fatalf("assertion %d: completion %v, want %q", i, a.Payload.TRLSample, want)
		}
	}
}

// TestTRLSameKeyIdenticalContentStaysSeparate runs two records with the same key
// and identical content. Their frames match on every field the run check used
// before, so only the record serial keeps them apart: each record's windows must
// form the same assertions they form alone.
func TestTRLSameKeyIdenticalContentStaysSeparate(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog."
	r1 := &schema.MinedRecord{FileName: "same.json", ChunkID: 3, Content: content, Output: "FIRST"}
	r2 := &schema.MinedRecord{FileName: "same.json", ChunkID: 3, Content: content, Output: "SECOND"}
	f1, c1 := encodeRecordsForTRL(t, r1)
	n1 := len(readAssertionsForTRL(t, f1, c1.trl))
	frames, config := encodeRecordsForTRL(t, r1, r2)
	got := readAssertionsForTRL(t, frames, config.trl)
	if len(got) != 2*n1 {
		t.Fatalf("two identical-content records wrote %d assertions, want %d (2 x %d)", len(got), 2*n1, n1)
	}
	for i, a := range got {
		want := "FIRST"
		if i >= n1 {
			want = "SECOND"
		}
		if a.Payload.TRLSample == nil || a.Payload.TRLSample.Completion != want {
			t.Fatalf("assertion %d: completion %v, want %q", i, a.Payload.TRLSample, want)
		}
	}
}

// TestTRLEmptyOutputNeverInheritsText: a later record with the same key and an
// empty Output must not pick up the earlier record's completion.
func TestTRLEmptyOutputNeverInheritsText(t *testing.T) {
	r1 := &schema.MinedRecord{FileName: "alpaca.json", ChunkID: 0,
		Content: "The quick brown fox jumps over the lazy dog.", Instruction: "Q1", Output: "ANSWER-ONE"}
	r2 := &schema.MinedRecord{FileName: "alpaca.json", ChunkID: 0,
		Content: "A completely different sentence about cats and mats here.", Instruction: "Q2", Output: ""}
	f1, c1 := encodeRecordsForTRL(t, r1)
	n1 := len(readAssertionsForTRL(t, f1, c1.trl))
	frames, config := encodeRecordsForTRL(t, r1, r2)
	got := readAssertionsForTRL(t, frames, config.trl)
	if len(got) <= n1 {
		t.Fatalf("combined run wrote %d assertions, want more than record 1's %d", len(got), n1)
	}
	for i, a := range got {
		if i < n1 {
			if a.Payload.TRLSample == nil || a.Payload.TRLSample.Completion != "ANSWER-ONE" {
				t.Fatalf("record 1 assertion %d: completion %v, want ANSWER-ONE", i, a.Payload.TRLSample)
			}
			continue
		}
		if a.Payload.TRLSample != nil {
			t.Fatalf("record 2 (empty output) assertion %d inherited TRL %+v", i, a.Payload.TRLSample)
		}
	}
}

// TestWriteNRVRunEndsAtRecordBoundary kills the mutant that drops the record
// serial from the run check. The four windows share file, chunk, span start and a
// non-decreasing WindowEnd; only the record serial marks the boundary after the
// second window.
func TestWriteNRVRunEndsAtRecordBoundary(t *testing.T) {
	frames := spanWindows("a.txt", 1, 10, 11, 12, 13, 14)
	trl := newTRLSide()
	trl.owner = []int64{1, 1, 2, 2}
	got := readAssertionsForTRL(t, frames, trl)
	if len(got) != 2 || len(got[0].Brackets) != 2 || len(got[1].Brackets) != 2 {
		t.Fatalf("got %d assertions, want 2 frames of 2 brackets each", len(got))
	}
}

// TestWriteNRVRefusesUntaggedFrames: a side map whose tags do not line up with
// the frames must be refused, not used to attach text to the wrong record.
func TestWriteNRVRefusesUntaggedFrames(t *testing.T) {
	frames := spanWindows("a.txt", 1, 10, 11, 12)
	trl := newTRLSide()
	trl.owner = []int64{1}
	path := filepath.Join(t.TempDir(), "untagged.nrv")
	if err := writeNRV(path, frames, trl); err == nil {
		t.Fatal("writeNRV accepted a side map that tags 1 of 2 frames")
	}
}

// TestTRLTextIsRedactedBeforeNRVWrite checks plan §7.7.1: a secret-shaped string
// in an Alpaca record never reaches the .nrv bytes. The secrets are obvious fakes
// that the redactor's patterns match. The redaction marker must appear instead,
// so the text was redacted, not dropped.
func TestTRLTextIsRedactedBeforeNRVWrite(t *testing.T) {
	secrets := []string{"fakefakefake123", "fakefakefakefake", "AKIA0000000000000000"}
	record := &schema.MinedRecord{
		FileName:    "alpaca.json",
		ChunkID:     0,
		Content:     "The quick brown fox jumps over the lazy dog. A second sentence follows here.",
		Instruction: "Log in with password=" + secrets[0],
		Input:       "api_key: " + secrets[1],
		Output:      "The key id is " + secrets[2] + ".",
	}
	frames, config := encodeRecordsForTRL(t, record)
	path := filepath.Join(t.TempDir(), "redact.nrv")
	if err := writeNRV(path, frames, config.trl); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range secrets {
		if bytes.Contains(raw, []byte(sec)) {
			t.Fatalf(".nrv bytes contain secret-shaped %q", sec)
		}
	}
	if !bytes.Contains(raw, []byte("[REDACTED]")) {
		t.Fatal(".nrv bytes carry no redaction marker: the TRL text was dropped, not redacted")
	}
	got := readAssertionsForTRL(t, frames, config.trl)
	if len(got) == 0 || got[0].Payload.TRLSample == nil {
		t.Fatal("no TRLSample on the record's assertions")
	}
	if !bytes.Contains([]byte(got[0].Payload.TRLSample.Prompt), []byte("[REDACTED]")) {
		t.Fatalf("prompt not redacted: %q", got[0].Payload.TRLSample.Prompt)
	}
}

// TestTRLSecretSpanningJoinIsRedacted is the join-spanning probe as a repo test.
// The instruction ends with "password=" and the input is the secret, so each
// field alone is harmless and only the joined prompt holds a secret-shaped
// string. The record must keep its TRLSample with the marker, and the secret
// must not reach the .nrv bytes.
func TestTRLSecretSpanningJoinIsRedacted(t *testing.T) {
	const sec = "fakefakefake123"
	record := &schema.MinedRecord{
		FileName:    "alpaca.json",
		ChunkID:     0,
		Content:     "The quick brown fox jumps over the lazy dog. A second sentence follows here.",
		Instruction: "Use this login, password=",
		Input:       sec,
		Output:      "Done.",
	}
	frames, config := encodeRecordsForTRL(t, record)
	path := filepath.Join(t.TempDir(), "join.nrv")
	if err := writeNRV(path, frames, config.trl); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(sec)) {
		t.Fatalf(".nrv bytes contain secret-shaped %q across the instruction/input join", sec)
	}
	got := readAssertionsForTRL(t, frames, config.trl)
	if len(got) == 0 || got[0].Payload.TRLSample == nil {
		t.Fatal("record lost its TRLSample: the join should be redacted, not dropped")
	}
	if !bytes.Contains([]byte(got[0].Payload.TRLSample.Prompt), []byte("[REDACTED]")) {
		t.Fatalf("joined prompt not redacted: %q", got[0].Payload.TRLSample.Prompt)
	}
}

// TestBuildTRLTextFailsClosed: when redaction leaves a secret-shaped string
// behind, the record yields no text, and the reason does not echo the secret.
// The identity redactor stands in for a redactor that misses a secret.
func TestBuildTRLTextFailsClosed(t *testing.T) {
	const sec = "fakefakefake123"
	identity := func(s string) string { return s }
	inPrompt := &schema.MinedRecord{Instruction: "Log in with password=" + sec, Output: "ok"}
	if _, ok, reason := buildTRLText(inPrompt, identity); ok || reason == "" {
		t.Fatalf("secret in prompt survived but record kept text (ok=%v reason=%q)", ok, reason)
	}
	inCompletion := &schema.MinedRecord{Instruction: "Q", Output: "key is api_key: " + sec}
	if _, ok, _ := buildTRLText(inCompletion, identity); ok {
		t.Fatal("secret in completion survived but record kept text")
	}
	clean := &schema.MinedRecord{Instruction: "Q", Output: "ok"}
	if tt, ok, _ := buildTRLText(clean, identity); !ok || tt.completion != "ok" {
		t.Fatalf("clean record lost its text: ok=%v tt=%+v", ok, tt)
	}
}

// TestTRLSecretAcrossPromptCompletionIsDropped covers the prompt | completion
// boundary the joint rescan closes. Each field is clean alone, so the per-field
// rescan passed these records and the secret reached the .nrv bytes. The record
// must lose its text (fail closed), and the secret must not reach the .nrv bytes.
func TestTRLSecretAcrossPromptCompletionIsDropped(t *testing.T) {
	const sec = "fakefakefake123"
	cases := map[string]*schema.MinedRecord{
		"instruction|output": {Instruction: "Use this login, password=", Output: sec},
		"input|output":       {Instruction: "Use this login", Input: "password=", Output: sec},
	}
	for name, record := range cases {
		t.Run(name, func(t *testing.T) {
			record.FileName, record.ChunkID = "alpaca.json", 0
			record.Content = "The quick brown fox jumps over the lazy dog. A second sentence follows here."
			frames, config := encodeRecordsForTRL(t, record)
			path := filepath.Join(t.TempDir(), "boundary.nrv")
			if err := writeNRV(path, frames, config.trl); err != nil {
				t.Fatalf("writeNRV: %v", err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(raw, []byte(sec)) {
				t.Fatalf(".nrv bytes contain secret-shaped %q across the prompt|completion boundary", sec)
			}
			for i, a := range readAssertionsForTRL(t, frames, config.trl) {
				if a.Payload.TRLSample != nil {
					t.Fatalf("assertion %d kept TRL text %+v; the record should be dropped", i, a.Payload.TRLSample)
				}
			}
		})
	}
}

// TestTRLDropLogHoldsNoSecret captures the log written when a record's text is
// dropped and asserts the secret never appears in it. The drop must happen (the
// counter moves and a line is logged), so the absence check is meaningful. Covers
// both the boundary drop and the real-redactor fail-closed branch.
func TestTRLDropLogHoldsNoSecret(t *testing.T) {
	const sec = "fakefakefake123"
	cases := map[string]*schema.MinedRecord{
		"boundary": {Instruction: "Use this login, password=", Output: sec},
		"residue":  {Instruction: "fakefakefake123token:secret=\nword=redis://abcdefghijkpassredis://abcdefghijk", Output: "ok"},
	}
	for name, record := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(prev)

			s := newTRLSide()
			s.beginRecord(record)
			if s.dropped != 1 {
				t.Fatalf("dropped = %d, want 1: the record was not dropped, so the log check proves nothing", s.dropped)
			}
			out := buf.String()
			if !strings.Contains(out, "dropped") {
				t.Fatalf("no drop line logged: %q", out)
			}
			if strings.Contains(out, sec) {
				t.Fatalf("drop log contains the secret: %q", out)
			}
		})
	}
}

// TestBuildTRLTextRealRedactorFailsClosed runs the real redactor through the
// fail-closed branch. The input's secret-shaped residue survives redaction, so the
// record must yield no text and a reason, with no secret literal in the reason.
func TestBuildTRLTextRealRedactorFailsClosed(t *testing.T) {
	record := &schema.MinedRecord{
		Instruction: "fakefakefake123token:secret=\nword=redis://abcdefghijkpassredis://abcdefghijk",
		Output:      "ok",
	}
	tt, ok, reason := buildTRLText(record, supervisionpipeline.RedactText)
	if ok || reason == "" {
		t.Fatalf("residue survived redaction but the record kept text (ok=%v reason=%q tt=%+v)", ok, reason, tt)
	}
	if strings.Contains(reason, "fakefakefake123") {
		t.Fatalf("reason echoes the secret: %q", reason)
	}
}

// assertTRLDroppedNoLiteral encodes one record, writes the .nrv, and asserts the
// secret literal is absent from the bytes and no assertion kept TRL text.
func assertTRLDroppedNoLiteral(t *testing.T, record *schema.MinedRecord, sec string) {
	t.Helper()
	record.FileName, record.ChunkID = "alpaca.json", 0
	record.Content = "The quick brown fox jumps over the lazy dog. A second sentence follows here."
	frames, config := encodeRecordsForTRL(t, record)
	path := filepath.Join(t.TempDir(), "separator.nrv")
	if err := writeNRV(path, frames, config.trl); err != nil {
		t.Fatalf("writeNRV: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(sec)) {
		t.Fatalf(".nrv bytes contain secret-shaped %q across the record boundary", sec)
	}
	for i, a := range readAssertionsForTRL(t, frames, config.trl) {
		if a.Payload.TRLSample != nil {
			t.Fatalf("assertion %d kept TRL text %+v; the record should be dropped", i, a.Payload.TRLSample)
		}
	}
}

// TestTRLKeywordSpanningPromptCompletionBoundaryIsDropped covers a keyword split
// across the boundary: "pass" | "word=<secret>" joins to "password=<secret>", and
// only the empty separator produces that match. Kills a mutant that drops "" from
// the joint separator set.
func TestTRLKeywordSpanningPromptCompletionBoundaryIsDropped(t *testing.T) {
	const sec = "fakefakefake123"
	assertTRLDroppedNoLiteral(t, &schema.MinedRecord{Instruction: "Use this pass", Output: "word=" + sec}, sec)
}

// TestTRLPrivateKeyHeaderSplitAcrossBoundaryIsDropped covers a PEM header split
// as "-----BEGIN" | "PRIVATE KEY-----". The header only forms with the " "
// separator. Kills a mutant that drops " " from the joint separator set.
func TestTRLPrivateKeyHeaderSplitAcrossBoundaryIsDropped(t *testing.T) {
	const header = "-----BEGIN PRIVATE KEY-----"
	assertTRLDroppedNoLiteral(t, &schema.MinedRecord{Instruction: "Key:\n-----BEGIN", Output: "PRIVATE KEY-----"}, header)
}

// TestTRLSummaryReportsDropCountOnly reads the drop counter through logSummary.
// It must report the count of dropped records, never their text or the secret,
// and it must be nil-safe.
func TestTRLSummaryReportsDropCountOnly(t *testing.T) {
	const sec = "fakefakefake123"
	s := newTRLSide()
	s.beginRecord(&schema.MinedRecord{Instruction: "Use this login, password=", Output: sec})
	s.beginRecord(&schema.MinedRecord{Instruction: "Plain question", Output: "Plain answer."})

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	s.logSummary()
	out := buf.String()
	if !strings.Contains(out, "1 record(s) dropped") {
		t.Fatalf("summary does not report the drop count of 1: %q", out)
	}
	if strings.Contains(out, sec) || strings.Contains(out, "Plain answer") {
		t.Fatalf("summary leaks record text: %q", out)
	}

	buf.Reset()
	var nilSide *trlSide
	nilSide.logSummary()
	if buf.Len() != 0 {
		t.Fatalf("nil side logged %q; want nothing", buf.String())
	}
}
