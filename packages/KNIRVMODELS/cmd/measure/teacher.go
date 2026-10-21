package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"knirvmodels/pkg/teacher"
)

// The strong model (Colibri) helps with the gold set and with judging retrieval, and
// does it as a teacher, not as an authority:
//
//   - `suggest` writes suggested labels to suggestions.jsonl. gold.jsonl is the
//     owner's (D4) and is only ever written by `gold add|import|accept`, where a person
//     names the label and is accountable for it.
//   - `retrieval-judge` grades retrieved passages to find relevant ones the gold set
//     has no label for. The grades are machine judgments, reported separately, and
//     never fed into the gold-based recall or into any promotion gate.
//
// Both need the teacher backend: Colibri while it is enabled, otherwise llama.cpp
// (slow on a weak host, but the same procedure). They wait for it and never split a
// run between the two; every record names its source.

const defaultInferenceSocket = "/var/lib/knirvserver/sockets/llama.sock"

type teacherFlags struct {
	socket  *string
	cache   *string
	wait    *time.Duration
	samples *int
}

func addTeacherFlags(fs *flag.FlagSet, dir *string) teacherFlags {
	return teacherFlags{
		socket:  fs.String("socket", defaultInferenceSocket, "inference socket (the harness router)"),
		cache:   fs.String("cache", "", "teacher answer cache (default <dir>/teacher.cache.jsonl)"),
		wait:    fs.Duration("wait", 30*time.Minute, "how long to wait for the teacher backend to be ready"),
		samples: fs.Int("samples", 4, "votes per question (half in reversed option order)"),
	}
}

func (f teacherFlags) open(dir string) (*teacher.Client, *teacher.Cache, error) {
	cl := teacher.NewUnix(*f.socket, 0)
	cl.Patience = *f.wait
	cl.Logf = func(format string, a ...any) { fmt.Fprintf(os.Stderr, "measure: "+format+"\n", a...) }
	path := *f.cache
	if path == "" {
		path = filepath.Join(dir, "teacher.cache.jsonl")
	}
	cache, err := teacher.OpenCache(path)
	return cl, cache, err
}

// teacherReady returns a blocked Result when Colibri is not serving, so the probe says
// so instead of waiting out the whole patience on a host that will never have it.
func teacherReady(ctx context.Context, cl *teacher.Client, probe string) *Result {
	st, err := cl.Status(ctx)
	if err != nil {
		return newResult(probe).blocked("the inference endpoint is not reachable: %v", err)
	}
	if st["teacher"] != nil {
		return nil // Colibri, or llama.cpp when Colibri is not enabled; each datum records which
	}
	reason := "the teacher backend is not ready (Colibri is enabled but not serving)"
	if n, _ := st["primary_note"].(string); n != "" {
		reason += ": " + n
	} else if q, ok := st["colibri_qualification"].(map[string]any); ok {
		if r, _ := q["reason"].(string); r != "" {
			reason += ": " + r
		}
	}
	return newResult(probe).blocked("%s", reason)
}

// Suggestion is a machine-suggested label awaiting the owner. It is not a GoldCase.
type Suggestion struct {
	ID          string             `json:"id"`
	Claim       string             `json:"claim"`
	Evidence    string             `json:"evidence"`
	Group       string             `json:"group,omitempty"`
	DataClass   string             `json:"data_class,omitempty"`
	Source      string             `json:"source,omitempty"`
	Suggested   string             `json:"suggested"`
	Margin      float64            `json:"margin"`
	Probs       map[string]float64 `json:"probs"`
	OrderAgrees bool               `json:"order_agrees"`
	// Confident is true when both option orders agree and the margin is at least 0.12
	// (the plan's judge margin). An unconfident suggestion is NEI by that rule.
	Confident   bool               `json:"confident"`
	Provenance  teacher.Provenance `json:"provenance"`
	SuggestedAt string             `json:"suggested_at"`
}

func suggestionsPath(dir string) string { return filepath.Join(dir, "suggestions.jsonl") }

// judgeMargin is the plan's minimum margin between the top two verdicts (0.12).
const judgeMargin = 0.12

func cmdSuggest(args []string) error {
	fs := flag.NewFlagSet("suggest", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	tf := addTeacherFlags(fs, dir)
	cand := fs.String("candidates", "", "JSONL of {claim, evidence[, group, data_class, source]} to suggest labels for")
	limit := fs.Int("limit", 0, "suggest for at most this many new candidates (0 = all)")
	fs.Parse(args)
	if *cand == "" {
		return fmt.Errorf("suggest: -candidates is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	in, err := readJSONL[GoldCase](*cand)
	if err != nil {
		return err
	}
	cl, cache, err := tf.open(*dir)
	if err != nil {
		return err
	}
	defer cache.Close()
	if blocked := teacherReady(ctx, cl, "suggest"); blocked != nil {
		return blocked.write(*dir)
	}
	gold, err := loadGold(*dir)
	if err != nil {
		return err
	}
	done := map[string]bool{}
	for _, g := range gold {
		done[g.ID] = true
	}
	if prev, err := readJSONL[Suggestion](suggestionsPath(*dir)); err == nil {
		for _, s := range prev {
			done[s.ID] = true
		}
	}
	r := newResult("suggest")
	n, skipped := 0, 0
	for _, c := range in {
		id := goldID(c.Claim, c.Evidence)
		if c.Claim == "" || c.Evidence == "" || done[id] {
			skipped++
			continue
		}
		if *limit > 0 && n >= *limit {
			break
		}
		d, err := teacher.SuggestFactuality(ctx, cl, cache, c.Claim, c.Evidence, *tf.samples)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			fmt.Fprintf(os.Stderr, "measure: %s: %v\n", id, err)
			continue
		}
		top, margin := d.Top()
		s := Suggestion{
			ID: id, Claim: c.Claim, Evidence: c.Evidence, Group: c.Group, DataClass: c.DataClass, Source: c.Source,
			Suggested: top, Margin: margin, Probs: d.Probs, OrderAgrees: d.Agree,
			Confident: d.Agree && margin >= judgeMargin, Provenance: d.Prov,
			SuggestedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if !s.Confident {
			s.Suggested = LabelNEI // disagreement or a thin margin is NEI (plan section 5.4)
		}
		if err := appendJSONL(suggestionsPath(*dir), s); err != nil {
			return err
		}
		done[id] = true
		n++
	}
	r.Data["suggested"], r.Data["skipped"] = n, skipped
	r.Data["file"] = suggestionsPath(*dir)
	r.note("suggestions are not labels: review them and record yours with `measure gold accept` (or `gold add`); gold.jsonl is untouched")
	return r.write(*dir)
}

// cmdGoldAccept records the OWNER's label for a suggested case. The label is the
// person's: it must be given explicitly, or `-agree` must say they agree with the
// suggestion. Either way the case is marked as teacher-suggested so agreement can be
// audited.
func cmdGoldAccept(fs *flag.FlagSet, dir *string, rest []string) error {
	var id, label, labeler, group, dataClass string
	var agree bool
	fs.StringVar(&id, "id", "", "suggestion id")
	fs.StringVar(&label, "label", "", "your label: SUPPORTED | REFUTED | NEI")
	fs.BoolVar(&agree, "agree", false, "you checked the evidence and agree with the suggested label")
	fs.StringVar(&labeler, "labeler", os.Getenv("USER"), "who labelled it (accountable for the label)")
	fs.StringVar(&group, "group", "", "split group (default: the suggestion's, else hash of the evidence)")
	fs.StringVar(&dataClass, "data-class", "", "public | private (default: the suggestion's, else private)")
	fs.Parse(rest)
	sugg, err := readJSONL[Suggestion](suggestionsPath(*dir))
	if err != nil {
		return err
	}
	var s *Suggestion
	for i := range sugg {
		if sugg[i].ID == id {
			s = &sugg[i]
		}
	}
	if s == nil {
		return fmt.Errorf("no suggestion %q in %s", id, suggestionsPath(*dir))
	}
	if label != "" && agree {
		return fmt.Errorf("give your own -label, or -agree, not both")
	}
	if agree {
		label = s.Suggested
	}
	if label == "" {
		return fmt.Errorf("a person's label is required: -label SUPPORTED|REFUTED|NEI, or -agree")
	}
	c := GoldCase{
		Claim: s.Claim, Evidence: s.Evidence, Label: label, Labeler: labeler,
		Group: firstNonEmpty(group, s.Group), DataClass: firstNonEmpty(dataClass, s.DataClass),
		Source: "teacher-suggested:" + s.Provenance.Catalog + ":" + suggestedNote(s, label),
	}
	if err := c.normalize(); err != nil {
		return err
	}
	cs, err := loadGold(*dir)
	if err != nil {
		return err
	}
	for _, e := range cs {
		if e.ID == c.ID {
			return fmt.Errorf("duplicate case %s", c.ID)
		}
	}
	if err := appendJSONL(goldPath(*dir), c); err != nil {
		return err
	}
	fmt.Printf("added %s (%s, split %s; labeler %s, teacher suggested %s)\n", c.ID, c.Label, splitOf(c.Group), c.Labeler, s.Suggested)
	return nil
}

func suggestedNote(s *Suggestion, label string) string {
	if s.Suggested == label {
		return "agreed"
	}
	return "overridden-from-" + s.Suggested
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Judgment is one machine-graded (claim, passage) pair.
type Judgment struct {
	Claim      string             `json:"claim"`
	Passage    string             `json:"passage"`
	Rank       int                `json:"rank"`
	IsGold     bool               `json:"is_gold_passage"`
	Grade      string             `json:"grade"`
	Margin     float64            `json:"margin"`
	Probs      map[string]float64 `json:"probs"`
	Provenance teacher.Provenance `json:"provenance"`
}

func cmdRetrievalJudge(args []string) error {
	fs := flag.NewFlagSet("retrieval-judge", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	tf := addTeacherFlags(fs, dir)
	split := fs.String("split", "all", "dev | test | all")
	k := fs.Int("k", 5, "judge the top k retrieved passages of each query")
	limit := fs.Int("limit", 0, "judge at most this many queries (0 = all)")
	distract := fs.String("distractors", "", "JSONL of {\"text\":...} extra passages to make the pool realistic")
	fs.Parse(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cs, blocked := goldReady(*dir, "retrieval-judge", *split)
	if blocked != nil {
		return blocked.write(*dir)
	}
	r := newResult("retrieval-judge")
	var extra []string
	if *distract != "" {
		d, err := readJSONL[struct{ Text string }](*distract)
		if err != nil {
			return r.failed("distractors: %v", err).write(*dir)
		}
		for _, x := range d {
			extra = append(extra, x.Text)
		}
	}
	pool, qs, gold := retrievalPool(cs, extra)
	if len(qs) == 0 {
		return r.blocked("no SUPPORTED/REFUTED cases to retrieve for").write(*dir)
	}
	cl, cache, err := tf.open(*dir)
	if err != nil {
		return err
	}
	defer cache.Close()
	if b := teacherReady(ctx, cl, "retrieval-judge"); b != nil {
		return b.write(*dir)
	}
	if *limit > 0 && len(qs) > *limit {
		qs, gold = qs[:*limit], gold[:*limit]
	}
	idx := newBM25(pool)

	out := filepath.Join(*dir, "retrieval_judgments.jsonl")
	os.Remove(out) // the file describes this run; the cache makes rerunning cheap
	var (
		queries, goldRelevant, goldJudged int
		anyRelevantAtK, unlabeledRelevant int
		relevantAtK                       float64
		examples                          []map[string]any
		failed                            int
	)
	for qi, q := range qs {
		if ctx.Err() != nil {
			break
		}
		ranked := idx.rank(q)
		if len(ranked) > *k {
			ranked = ranked[:*k]
		}
		queries++
		rel, anyRel := 0, false
		for rank, pi := range ranked {
			d, err := teacher.JudgeRelevance(ctx, cl, cache, q, pool[pi], *tf.samples)
			if err != nil {
				failed++
				continue
			}
			grade, margin := d.Top()
			j := Judgment{Claim: q, Passage: pool[pi], Rank: rank + 1, IsGold: pi == gold[qi], Grade: grade, Margin: margin, Probs: d.Probs, Provenance: d.Prov}
			if err := appendJSONL(out, j); err != nil {
				return err
			}
			if j.IsGold {
				goldJudged++
				if grade == teacher.Relevant {
					goldRelevant++
				}
			}
			if grade == teacher.Relevant {
				rel++
				anyRel = true
				if !j.IsGold {
					unlabeledRelevant++
					if len(examples) < 20 {
						examples = append(examples, map[string]any{"claim": q, "passage": pool[pi], "rank": rank + 1})
					}
				}
			}
		}
		relevantAtK += float64(rel) / float64(len(ranked))
		if anyRel {
			anyRelevantAtK++
		}
	}
	if queries == 0 {
		return r.failed("no query was judged").write(*dir)
	}
	r.Data["judge"] = "colibri (machine-judged; supplements the gold-based recall, never replaces it)"
	r.Data["queries"], r.Data["k"], r.Data["failed_judgments"] = queries, *k, failed
	r.Data["judged_precision_at_k"] = relevantAtK / float64(queries)
	r.Data["queries_with_a_relevant_passage_in_top_k"] = float64(anyRelevantAtK) / float64(queries)
	r.Data["relevant_passages_without_a_gold_label"] = unlabeledRelevant
	r.Data["examples_for_owner_review"] = examples
	r.Data["judgments_file"] = out
	if goldJudged > 0 {
		agree := float64(goldRelevant) / float64(goldJudged)
		r.Data["judge_agrees_gold_passage_is_relevant"] = agree
		r.check("teacher calls the owner's gold passage relevant >= 0.80 (trust check)", agree >= 0.80, fmt.Sprintf("%.2f over %d", agree, goldJudged))
	}
	r.note("relevant passages without a gold label mean the gold-based recall@k understates retrieval; the owner decides whether to label them")
	return r.write(*dir)
}
