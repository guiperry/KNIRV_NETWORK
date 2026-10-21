package evidence

import (
	"math"
	"strings"
)

// Rerank weights. Key overlap finds candidates cheaply; it cannot tell a passage that
// is about the claim from one that merely shares common words ("capital", "France",
// "Paris" appear in a football club's article). The rerank reads the candidate text
// and scores what a fact check actually needs: the claim's words, its named things
// and its numbers being present, and the passage being about the claim's subject.
const (
	rerankPool   = 100 // visible candidates whose text is read
	bm25K1       = 1.2 // term-frequency saturation
	bm25B        = 0.75
	wTitleFull   = 6.0 // the passage's article title is in the claim
	wTitlePart   = 1.0 // at least half the title's words are
	wEntityCover = 3.0 // fraction of the claim's named entities found in the passage
	wNumberCover = 3.0 // fraction of the claim's numbers/dates found in the passage
	wLead        = 0.75
	wKeyTie      = 0.02 // the first-stage key score, as a tie-break
)

type scored struct {
	hit  Hit
	norm string // Canon(passage), for entity containment
	toks []string
}

// rerank orders hits by BM25 over the candidate set plus coverage bonuses. IDF is
// taken from the candidate set itself: a word every candidate contains carries no
// information about which one is right.
func rerank(c Claim, hits []Hit) []Hit {
	if len(hits) < 2 {
		return hits
	}
	qf := ExtractFeatures(c.Text + " " + c.Subject + " " + c.Property + " " + c.Value)
	qterms := map[string]bool{}
	for _, t := range qf.Terms {
		qterms[t] = true
	}
	for _, e := range qf.Entities {
		for _, w := range strings.Fields(e) {
			if len(w) >= 3 && !stopWords[w] {
				qterms[w] = true
			}
		}
	}
	items := make([]scored, len(hits))
	df := map[string]int{}
	var totalLen int
	for i, h := range hits {
		toks := tokenize(h.Text)
		ws := make([]string, len(toks))
		seen := map[string]bool{}
		for j, t := range toks {
			ws[j] = strings.ToLower(t.text)
			if qterms[ws[j]] && !seen[ws[j]] {
				seen[ws[j]] = true
				df[ws[j]]++
			}
		}
		items[i] = scored{hit: h, norm: " " + Canon(h.Text) + " ", toks: ws}
		totalLen += len(ws)
	}
	n := float64(len(items))
	avg := float64(totalLen) / n
	if avg < 1 {
		avg = 1
	}
	claimNorm := " " + Canon(c.Text+" "+c.Subject) + " "
	for i := range items {
		it := &items[i]
		tf := map[string]int{}
		for _, w := range it.toks {
			if qterms[w] {
				tf[w]++
			}
		}
		var bm float64
		dl := float64(len(it.toks))
		for w, f := range tf {
			idf := math.Log(1 + (n-float64(df[w])+0.5)/(float64(df[w])+0.5))
			ft := float64(f)
			bm += idf * ft * (bm25K1 + 1) / (ft + bm25K1*(1-bm25B+bm25B*dl/avg))
		}
		var bonus float64
		if t := Canon(it.hit.Title); t != "" {
			switch {
			case strings.Contains(claimNorm, " "+t+" "):
				bonus += wTitleFull
			default:
				tw := strings.Fields(t)
				in := 0
				for _, w := range tw {
					if strings.Contains(claimNorm, " "+w+" ") {
						in++
					}
				}
				if len(tw) >= 2 && float64(in)/float64(len(tw)) >= 0.5 {
					bonus += wTitlePart
				}
			}
			if it.hit.Start == 0 && bonus > 0 {
				bonus += wLead // the lead of an article that is about the subject
			}
		}
		if len(qf.Entities) > 0 {
			in := 0
			for _, e := range qf.Entities {
				if strings.Contains(it.norm, " "+e+" ") {
					in++
				}
			}
			bonus += wEntityCover * float64(in) / float64(len(qf.Entities))
		}
		if len(qf.Numbers) > 0 {
			pn := map[string]bool{}
			for _, x := range ExtractFeatures(it.hit.Text).Numbers {
				pn[x] = true
			}
			in := 0
			for _, x := range qf.Numbers {
				if pn[x] {
					in++
				}
			}
			bonus += wNumberCover * float64(in) / float64(len(qf.Numbers))
		}
		it.hit.Score = math.Round((bm+bonus+wKeyTie*it.hit.Score)*1000) / 1000
	}
	// Stable insertion sort by score, then by the first-stage order (already deterministic).
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].hit.Score > items[j-1].hit.Score; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	out := make([]Hit, len(items))
	for i := range items {
		out[i] = items[i].hit
	}
	return out
}
