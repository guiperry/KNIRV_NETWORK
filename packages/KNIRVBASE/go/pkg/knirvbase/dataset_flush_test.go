package knirvbase

import (
	"context"
	"testing"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

func collect(t *testing.T, ds *NRVDataset) []*nrv.Bracket {
	t.Helper()
	ch, err := ds.StreamBrackets(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	var out []*nrv.Bracket
	for b := range ch {
		out = append(out, b)
	}
	return out
}

// Brackets appended after an earlier read must be visible once flushed.
func TestDatasetFlushMakesNewBracketsReadable(t *testing.T) {
	db, err := NewNRV(context.Background(), Options{DataDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Shutdown()
	ds := db.Dataset("arena-err-1")

	for i := 0; i < 3; i++ {
		if err := ds.AppendBracket(context.Background(), &nrv.Bracket{GoldenSeed: uint32(i + 1)}, nrv.ThermoAtmosphere{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := ds.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := len(collect(t, ds)); got != 3 {
		t.Fatalf("first read: %d brackets, want 3", got)
	}

	for i := 0; i < 2; i++ {
		if err := ds.AppendBracket(context.Background(), &nrv.Bracket{GoldenSeed: uint32(10 + i)}, nrv.ThermoAtmosphere{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := ds.Flush(); err != nil {
		t.Fatal(err)
	}
	all := collect(t, ds)
	if len(all) != 5 {
		t.Fatalf("second read: %d brackets, want 5 (stale reader?)", len(all))
	}
	frames, err := ds.Frames()
	if err != nil || len(frames) != 2 {
		t.Fatalf("frames = %d, %v; want 2", len(frames), err)
	}
	entry, brackets, err := ds.GetFrame(context.Background(), frames[1].ID)
	if err != nil || entry == nil || len(brackets) != 2 || brackets[0].GoldenSeed != 10 {
		t.Fatalf("GetFrame = %v, %d brackets, %v", entry, len(brackets), err)
	}
}

func TestDatasetStreamStopsOnCancel(t *testing.T) {
	db, err := NewNRV(context.Background(), Options{DataDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Shutdown()
	ds := db.Dataset("arena-err-2")
	for i := 0; i < 600; i++ {
		_ = ds.AppendBracket(context.Background(), &nrv.Bracket{GoldenSeed: uint32(i)}, nrv.ThermoAtmosphere{})
	}
	if err := ds.Flush(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := ds.StreamBrackets(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	<-ch
	cancel()
	n := 0
	for range ch {
		n++
	}
	if n >= 599 {
		t.Fatalf("stream kept going after cancel (%d more brackets)", n)
	}
}
