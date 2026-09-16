package semanticmemory

import (
	"path/filepath"
	"testing"
)

func TestModelUpdatesAndQueriesPrototype(t *testing.T) {
	m, err := New(3, 4, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Update([]float32{1, 0, 0}, 11); err != nil {
		t.Fatal(err)
	}
	if err := m.Update([]float32{0.8, 0.2, 0}, 11); err != nil {
		t.Fatal(err)
	}
	if err := m.Update([]float32{0, 1, 0}, 22); err != nil {
		t.Fatal(err)
	}

	target, score, ok := m.Query([]float32{1, 0, 0})
	if !ok || target != 11 || score < 0.9 {
		t.Fatalf("Query = target %d, score %f, ok %v; want target 11 with high confidence", target, score, ok)
	}
	if m.FramesSeen != 3 || m.FramesIndexed != 3 {
		t.Fatalf("frame counts = %d/%d, want 3/3", m.FramesSeen, m.FramesIndexed)
	}
}

func TestModelBoundedAndRoundTrips(t *testing.T) {
	m, err := New(2, 2, "test")
	if err != nil {
		t.Fatal(err)
	}
	for i := int32(1); i <= 3; i++ {
		if err := m.Update([]float32{float32(i), 1}, i); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.Prototypes) != 2 || m.Evictions != 1 {
		t.Fatalf("prototypes=%d evictions=%d, want 2/1", len(m.Prototypes), m.Evictions)
	}

	path := filepath.Join(t.TempDir(), "semantic_memory.json")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Dimensions != 2 || len(loaded.Prototypes) != 2 {
		t.Fatalf("loaded model mismatch: %#v", loaded)
	}
}
