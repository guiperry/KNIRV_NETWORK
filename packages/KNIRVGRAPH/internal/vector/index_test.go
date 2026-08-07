package vector

import (
	"testing"
)

func TestNewVectorIndex(t *testing.T) {
	idx := NewVectorIndex(4)
	if idx == nil {
		t.Fatal("Expected non-nil index")
	}
	if idx.Len() != 0 {
		t.Errorf("Expected empty index, got %d", idx.Len())
	}
}

func TestVectorIndexAddAndSearch(t *testing.T) {
	idx := NewVectorIndex(3)
	idx.Add("a", []float32{1, 0, 0})
	idx.Add("b", []float32{0, 1, 0})
	idx.Add("c", []float32{0, 0, 1})

	if idx.Len() != 3 {
		t.Errorf("Expected 3 nodes, got %d", idx.Len())
	}

	ids, scores, err := idx.Search([]float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(ids))
	}
	if ids[0] != "a" {
		t.Errorf("Expected first result 'a', got %s", ids[0])
	}
	if scores[0] <= 0 {
		t.Error("Expected positive similarity score for exact match")
	}
}

func TestVectorIndexDimensionMismatch(t *testing.T) {
	idx := NewVectorIndex(3)
	err := idx.Add("bad", []float32{1, 2})
	if err == nil {
		t.Error("Expected dimension mismatch error")
	}
}

func TestVectorIndexDelete(t *testing.T) {
	idx := NewVectorIndex(2)
	idx.Add("x", []float32{1, 0})
	idx.Delete("x")
	if idx.Len() != 0 {
		t.Errorf("Expected 0 nodes after delete, got %d", idx.Len())
	}
}

func TestCosineSimilarityFloat32(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	score := cosineSimilarityFloat32(a, b)
	if score != 0 {
		t.Errorf("Expected 0 for orthogonal vectors, got %f", score)
	}

	c := []float32{1, 0}
	d := []float32{1, 0}
	score = cosineSimilarityFloat32(c, d)
	if score != 1 {
		t.Errorf("Expected 1 for identical vectors, got %f", score)
	}
}
