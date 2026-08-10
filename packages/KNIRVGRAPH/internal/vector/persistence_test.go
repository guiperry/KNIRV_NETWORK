package vector

import (
	"KNIRVGRAPH/internal/storage"
	"testing"
)

func TestPersistentIndexMetricsAndRecovery(t *testing.T) {
	for _, metric := range []Metric{MetricCosine, MetricDot, MetricEuclidean} {
		store, err := storage.NewMemoryStorage()
		if err != nil {
			t.Fatal(err)
		}
		idx, err := NewPersistentVectorIndex(2, Options{Metric: metric, M: 4, EFConstruction: 16, EFSearch: 16, PersistenceKey: "test"}, store)
		if err != nil {
			t.Fatal(err)
		}
		if err := idx.Add("x", []float32{1, 0}); err != nil {
			t.Fatal(err)
		}
		if err := idx.Add("y", []float32{0, 1}); err != nil {
			t.Fatal(err)
		}
		ids, _, err := idx.Search([]float32{1, 0}, 1)
		if err != nil || len(ids) != 1 || ids[0] != "x" {
			t.Fatalf("metric %s: ids=%v err=%v", metric, ids, err)
		}
		reopened, err := NewPersistentVectorIndex(2, Options{PersistenceKey: "test"}, store)
		if err != nil {
			t.Fatal(err)
		}
		if reopened.Len() != 2 {
			t.Fatalf("recovered %d vectors", reopened.Len())
		}
		if err := reopened.DeleteWithError("x"); err != nil {
			t.Fatal(err)
		}
	}
}
