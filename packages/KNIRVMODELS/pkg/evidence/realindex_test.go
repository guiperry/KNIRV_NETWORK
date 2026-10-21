package evidence

import (
	"os"
	"testing"
	"time"
)

// Probe against a real index: EVIDENCE_REAL=<dir> go test -run RealIndex -v
func TestRealIndexTiming(t *testing.T) {
	dir := os.Getenv("EVIDENCE_REAL")
	if dir == "" {
		t.Skip("set EVIDENCE_REAL=<index dir>")
	}
	t0 := time.Now()
	ix, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	t.Logf("open: %s", time.Since(t0))
	for _, q := range []string{
		"The capital of France is Paris.", "Barack Obama was born in 1961.", "Mont Blanc is the highest mountain in the Alps.",
		"The Eiffel Tower is 330 metres tall.", "Albert Einstein developed the theory of relativity.",
		"Water boils at 100 degrees Celsius.", "The Amazon is the largest rainforest in the world.", "Mount Everest is 8,849 metres high.",
	} {
		for pass := 0; pass < 2; pass++ {
			t1 := time.Now()
			r, _ := ix.Search(Claim{Text: q}, Options{K: 3})
			if pass == 0 {
				t.Logf("search %.50q: cold %s, %d hits", q, time.Since(t1), len(r.Hits))
				for _, h := range r.Hits {
					t.Logf("    %.2f %s", h.Score, h.Title)
				}
			} else {
				t.Logf("    warm %s", time.Since(t1))
			}
		}
	}
}
