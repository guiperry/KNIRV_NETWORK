package nrvio

import (
	"path/filepath"
	"testing"
	"time"
)

// TestFrameTickerIndexRestartsPerFrame appends two one-second buckets. Offset is a
// byte offset, so BracketIndex.Index must be the bracket's position inside its own
// frame, restarting at 0 for every frame.
func TestFrameTickerIndexRestartsPerFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ticker.nrv")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	tk := NewFrameTicker(w, time.Second)
	us := []uint32{100, 200, 1_100_000, 1_200_000, 1_300_000}
	for _, s := range us {
		if err := tk.Append(Bracket{SubSecondUS: s}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tk.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	frames := r.Registry().Frames
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	wantOffset := []int64{0, 2 * BracketSize}
	wantCount := []int{2, 3}
	for fi, f := range frames {
		if f.Offset != wantOffset[fi] {
			t.Errorf("frame %d offset = %d, want %d", fi, f.Offset, wantOffset[fi])
		}
		if f.Count != wantCount[fi] || len(f.BracketIndex) != wantCount[fi] {
			t.Fatalf("frame %d count = %d with %d index entries, want %d", fi, f.Count, len(f.BracketIndex), wantCount[fi])
		}
		for i, e := range f.BracketIndex {
			if e.Index != i {
				t.Errorf("frame %d bracket %d Index = %d, want %d (position within frame)", fi, i, e.Index, i)
			}
		}
	}
}
