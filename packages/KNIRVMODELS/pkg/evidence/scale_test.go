package evidence

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

// Scale probe: go test -run Scale -v -args -scale=20000. Skipped by default; the numbers
// it prints go in measurements.md.
func TestScaleProbe(t *testing.T) {
	if os.Getenv("EVIDENCE_SCALE") == "" {
		t.Skip("set EVIDENCE_SCALE=<docs> to run")
	}
	var n int
	fmt.Sscan(os.Getenv("EVIDENCE_SCALE"), &n)
	rng := rand.New(rand.NewSource(1))
	vocab := make([]string, 30000)
	for i := range vocab {
		vocab[i] = fmt.Sprintf("%sx%d", string(rune('a'+rng.Intn(26))), i)
	}
	mk := func(i int) Doc {
		var b strings.Builder
		fmt.Fprintf(&b, "Entity%d Quorra%d was recorded in %d with value %d. ", i, i%977, 1000+i%1000, i*7)
		for w := 0; w < 300; w++ {
			b.WriteString(vocab[int(rng.ExpFloat64()*3000)%len(vocab)])
			b.WriteByte(' ')
		}
		return Doc{ID: fmt.Sprint(i), Text: b.String()}
	}
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	start := time.Now()
	const batch = 5000
	for i := 0; i < n; i += batch {
		var docs []Doc
		for j := i; j < i+batch && j < n; j++ {
			docs = append(docs, mk(j))
		}
		if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: docs}); err != nil {
			t.Fatal(err)
		}
	}
	el := time.Since(start)
	st := ix.Status()
	t.Logf("ingested %d docs / %d spans in %s = %.0f spans/s over %d segments", n, st.Spans, el, float64(st.Spans)/el.Seconds(), st.Segments)
	hits, lat := 0, time.Duration(0)
	const q = 200
	for i := 0; i < q; i++ {
		d := rng.Intn(n)
		t0 := time.Now()
		r, _ := ix.Search(Claim{Text: fmt.Sprintf("Entity%d Quorra%d was recorded in %d with value %d.", d, d%977, 1000+d%1000, d*7)}, Options{K: 5})
		lat += time.Since(t0)
		for _, h := range r.Hits {
			if strings.Contains(h.Text, fmt.Sprintf("Entity%d Quorra", d)) {
				hits++
				break
			}
		}
	}
	t.Logf("recall@5 %d/%d, mean query %s", hits, q, lat/q)
}
