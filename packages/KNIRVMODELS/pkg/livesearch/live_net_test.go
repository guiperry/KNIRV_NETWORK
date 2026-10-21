package livesearch

import (
	"context"
	"os"
	"testing"
	"time"

	"knirvmodels/pkg/evidence"
)

// Real network: LIVE_NET=1 go test -run LiveNet -v. A few requests to Wikipedia and arXiv.
func TestLiveNetSmoke(t *testing.T) {
	if os.Getenv("LIVE_NET") == "" {
		t.Skip("set LIVE_NET=1 to contact Wikipedia and arXiv")
	}
	dir := t.TempDir()
	ix, _ := evidence.Open(dir, "")
	defer ix.Close()
	r, err := NewProduction(ix, dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for _, c := range []evidence.Claim{
		{Text: "The Eiffel Tower is 330 metres tall."},
		{Text: "The transformer paper Attention Is All You Need introduced self-attention for sequence models."},
	} {
		t0 := time.Now()
		res, err := r.Retrieve(ctx, c, Options{K: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%q: %s, fetched=%d degraded=%v reasons=%v queries=%v", c.Text, time.Since(t0).Round(time.Millisecond), res.Fetched, res.Degraded, res.Reasons, res.Queries)
		for _, h := range res.Hits {
			t.Logf("   %.1f [%s rev %s] %s | %.90q", h.Score, h.Source, h.Revision, h.Title, h.Text)
		}
		t1 := time.Now()
		again, _ := r.Retrieve(ctx, c, Options{K: 3})
		t.Logf("   repeat: %s, flash_used=%v", time.Since(t1).Round(time.Millisecond), again.FlashUsed)
	}
}
