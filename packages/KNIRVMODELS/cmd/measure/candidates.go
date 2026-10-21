package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Candidate generation turns a Wikipedia JSONL dump into unlabelled claim x evidence
// pairs for the owner to label. It never writes gold.jsonl and never assigns a label:
// the "kind" of each pair (how it was built) goes to a sidecar file so the mix can be
// audited, but is kept out of the sheet the owner reads, because "this was altered"
// would hand them the answer.

const (
	kindVerbatim  = "verbatim"  // sentence + the passage it came from (easy; calibrates the labeler and the judge)
	kindAltered   = "altered"   // sentence with one number changed + the original passage
	kindUnrelated = "unrelated" // sentence + a passage from a different article
	kindNeighbour = "neighbour" // sentence + another passage of the same article (the hard, honest case)
)

var candidateKinds = []string{kindVerbatim, kindAltered, kindUnrelated, kindNeighbour}

type CandidateKind struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

func candidatesPath(dir string) string     { return filepath.Join(dir, "candidates.jsonl") }
func candidateKindsPath(dir string) string { return filepath.Join(dir, "candidates.kinds.jsonl") }
func sheetPath(dir string) string          { return filepath.Join(dir, "candidates.csv") }

type wikiArticle struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// cmdCandidates samples articles, builds the four kinds of pair, and writes
// candidates.jsonl (for `suggest`), candidates.kinds.jsonl and candidates.csv (the
// sheet to label).
func cmdCandidates(args []string) error {
	fs := flag.NewFlagSet("candidates", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	wiki := fs.String("wiki", "", "Wikipedia JSONL file, or a directory of them (id,url,title,text per line)")
	n := fs.Int("n", 300, "how many new candidates to produce")
	seed := fs.Int64("seed", 1, "sampling seed (same seed + same input = same candidates)")
	maxLines := fs.Int("max-articles", 200000, "read at most this many articles per run")
	fs.Parse(args)
	if *wiki == "" {
		return fmt.Errorf("candidates: -wiki is required")
	}
	files, err := wikiFiles(*wiki)
	if err != nil {
		return err
	}
	rng := rand.New(rand.NewSource(*seed))

	// Reservoir-sample articles so a 20 GB dump is read once, front to back, and only
	// a few thousand articles stay in memory.
	want := *n * 3
	if want < 50 {
		want = 50
	}
	var pool []wikiArticle
	seen := 0
	for _, f := range files {
		if err := eachArticle(f, func(a wikiArticle) bool {
			if seen >= *maxLines {
				return false
			}
			if len(strings.Fields(a.Text)) < 120 {
				return true // stubs have no second passage to pair with
			}
			seen++
			if len(pool) < want {
				pool = append(pool, a)
			} else if j := rng.Intn(seen); j < want {
				pool[j] = a
			}
			return true
		}); err != nil {
			return err
		}
		if seen >= *maxLines {
			break
		}
	}
	if len(pool) < 2 {
		return fmt.Errorf("candidates: only %d usable article(s) in %s", len(pool), *wiki)
	}

	// Anything already labelled, suggested or produced is not produced again.
	have := map[string]bool{}
	if gold, err := loadGold(*dir); err == nil {
		for _, g := range gold {
			have[g.ID] = true
		}
	}
	if prev, err := readJSONL[GoldCase](candidatesPath(*dir)); err == nil {
		for _, c := range prev {
			have[goldID(c.Claim, c.Evidence)] = true
		}
	}

	type made struct {
		c    GoldCase
		kind string
	}
	var out []made
	counts := map[string]int{}
	perKind := (*n + len(candidateKinds) - 1) / len(candidateKinds)
	for _, i := range rng.Perm(len(pool)) {
		if len(out) >= *n {
			break
		}
		a := pool[i]
		paras := passages(a.Text)
		if len(paras) < 2 {
			continue
		}
		claim, pi, ok := pickClaim(paras, rng)
		if !ok {
			continue
		}
		kind := nextKind(counts, perKind, rng)
		var claimText, evidence string
		switch kind {
		case kindVerbatim:
			claimText, evidence = claim, paras[pi]
		case kindAltered:
			alt, ok := alterNumber(claim, rng)
			if !ok {
				continue
			}
			claimText, evidence = alt, paras[pi]
		case kindNeighbour:
			j := pi + 1
			if j >= len(paras) {
				j = pi - 1
			}
			claimText, evidence = claim, paras[j]
		case kindUnrelated:
			other := pool[(i+1+rng.Intn(len(pool)-1))%len(pool)]
			op := passages(other.Text)
			if len(op) == 0 || other.ID == a.ID {
				continue
			}
			claimText, evidence = claim, op[rng.Intn(len(op))]
		}
		id := goldID(claimText, evidence)
		if have[id] {
			continue
		}
		have[id] = true
		counts[kind]++
		src := "wikipedia:" + a.Title
		if a.URL != "" {
			src += " " + a.URL
		}
		// Group by claim article so every case about one article lands in one split.
		out = append(out, made{GoldCase{Claim: claimText, Evidence: evidence, Group: "wiki-" + a.ID, DataClass: "public", Source: src}, kind})
	}
	if len(out) == 0 {
		return fmt.Errorf("candidates: nothing produced (every candidate already exists or the input is too small)")
	}
	// Shuffle so neighbouring rows are not the same kind or the same article.
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return err
	}
	for _, m := range out {
		if err := appendJSONL(candidatesPath(*dir), m.c); err != nil {
			return err
		}
		if err := appendJSONL(candidateKindsPath(*dir), CandidateKind{ID: goldID(m.c.Claim, m.c.Evidence), Kind: m.kind}); err != nil {
			return err
		}
	}
	cases := make([]GoldCase, len(out))
	for i, m := range out {
		cases[i] = m.c
	}
	if err := writeSheet(sheetPath(*dir), cases); err != nil {
		return err
	}

	r := newResult("candidates")
	r.Data["produced"] = len(out)
	r.Data["by_kind"] = counts
	r.Data["articles_sampled"] = len(pool)
	r.Data["candidates"] = candidatesPath(*dir)
	r.Data["sheet"] = sheetPath(*dir)
	r.note("candidates are unlabelled: fill the `label` column of %s (SUPPORTED | REFUTED | NEI), then `measure gold import-sheet`", sheetPath(*dir))
	r.note("the sheet hides how each pair was built; kinds are in %s for balance checks only", candidateKindsPath(*dir))
	fmt.Printf("produced %d candidates %v\n  sheet: %s\n  next:  fill the label column, then: measure gold import-sheet -dir %s -file %s -labeler <you>\n",
		len(out), counts, sheetPath(*dir), *dir, sheetPath(*dir))
	return r.write(*dir)
}

func wikiFiles(p string) ([]string, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return []string{p}, nil
	}
	fs, err := filepath.Glob(filepath.Join(p, "*.jsonl"))
	if err != nil || len(fs) == 0 {
		return nil, fmt.Errorf("candidates: no *.jsonl in %s", p)
	}
	sort.Strings(fs)
	return fs, nil
}

func eachArticle(path string, fn func(wikiArticle) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 128<<20)
	for sc.Scan() {
		var a wikiArticle
		if json.Unmarshal(sc.Bytes(), &a) != nil || a.Text == "" {
			continue
		}
		if !fn(a) {
			return nil
		}
	}
	return sc.Err()
}

// passages splits an article into paragraph-sized evidence of 30-250 words; very
// short lines (headings) are dropped and long paragraphs are cut at a sentence end.
func passages(text string) []string {
	var out []string
	for _, p := range strings.Split(text, "\n\n") {
		// A heading sits on its own short line at the top of a paragraph; drop it so it
		// does not glue itself onto the first sentence.
		if lines := strings.Split(strings.TrimSpace(p), "\n"); len(lines) > 1 {
			if h := strings.TrimSpace(lines[0]); len(strings.Fields(h)) <= 6 && !strings.ContainsAny(h, ".!?") {
				p = strings.Join(lines[1:], " ")
			}
		}
		p = strings.Join(strings.Fields(p), " ")
		w := len(strings.Fields(p))
		if w < 30 || !looksLikeProse(p, w) {
			continue
		}
		if w > 250 {
			words := strings.Fields(p)
			p = strings.Join(words[:250], " ")
			if i := strings.LastIndexAny(p, ".!?"); i > len(p)/2 {
				p = p[:i+1]
			}
		}
		out = append(out, p)
	}
	return out
}

var sentenceEnd = regexp.MustCompile(`([.!?])\s+([A-Z0-9"'(])`)
var hasNumber = regexp.MustCompile(`\b\d{1,4}(,\d{3})*\b`)

func sentences(p string) []string {
	var out []string
	last := 0
	for _, m := range sentenceEnd.FindAllStringSubmatchIndex(p, -1) {
		out = append(out, strings.TrimSpace(p[last:m[3]]))
		last = m[4]
	}
	if last < len(p) {
		out = append(out, strings.TrimSpace(p[last:]))
	}
	return out
}

// pickClaim returns a checkable sentence (it carries a number, 8-35 words) and the
// index of the passage it came from. Sentences with numbers are what the factuality
// slice most needs to get right, and they can be altered unambiguously.
func pickClaim(paras []string, rng *rand.Rand) (string, int, bool) {
	for _, i := range rng.Perm(len(paras)) {
		ss := sentences(paras[i])
		for _, j := range rng.Perm(len(ss)) {
			s := ss[j]
			w := len(strings.Fields(s))
			if w < 8 || w > 35 || !hasNumber.MatchString(s) || strings.ContainsAny(s, "\n\t") {
				continue
			}
			if !unicode.IsUpper([]rune(s)[0]) || dependsOnContext(s) {
				continue
			}
			return s, i, true
		}
	}
	return "", 0, false
}

// alterNumber changes the first standalone number so the claim now contradicts the
// passage it came from. Years move by 1-9 and other numbers by roughly a quarter to
// double, never to the same value.
func alterNumber(s string, rng *rand.Rand) (string, bool) {
	loc := hasNumber.FindStringIndex(s)
	if loc == nil {
		return "", false
	}
	tok := s[loc[0]:loc[1]]
	n, err := strconv.Atoi(strings.ReplaceAll(tok, ",", ""))
	if err != nil {
		return "", false
	}
	var m int
	if n >= 1000 && n <= 2100 && !strings.Contains(tok, ",") {
		m = n + 1 + rng.Intn(9)
		if rng.Intn(2) == 0 {
			m = n - 1 - rng.Intn(9)
		}
	} else {
		m = n*(2+rng.Intn(3))/2 + 1 + rng.Intn(3)
		if m == n {
			m = n + 1
		}
	}
	rep := strconv.Itoa(m)
	if strings.Contains(tok, ",") {
		rep = commas(m)
	}
	return s[:loc[0]] + rep + s[loc[1]:], true
}

func commas(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// nextKind picks the kind furthest below its share, breaking ties at random, so the
// mix stays balanced even when some articles cannot make some kinds.
func nextKind(counts map[string]int, perKind int, rng *rand.Rand) string {
	best, bestN := "", 1<<30
	for _, i := range rng.Perm(len(candidateKinds)) {
		k := candidateKinds[i]
		if counts[k] < perKind && counts[k] < bestN {
			best, bestN = k, counts[k]
		}
	}
	if best == "" {
		return candidateKinds[rng.Intn(len(candidateKinds))]
	}
	return best
}

var sheetHeader = []string{"id", "label", "claim", "evidence", "note"}

func writeSheet(path string, cs []GoldCase) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write(sheetHeader)
	for _, c := range cs {
		w.Write([]string{goldID(c.Claim, c.Evidence), "", c.Claim, c.Evidence, ""})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// cmdGoldImportSheet reads a filled sheet. Rows with a blank label are left for later;
// every labelled row becomes a gold case under the named labeler, taking group, data
// class and source from the candidate it came from.
func cmdGoldImportSheet(fs *flag.FlagSet, dir *string, rest []string) error {
	var file, labeler string
	fs.StringVar(&file, "file", "", "the filled candidates.csv (default <dir>/candidates.csv)")
	fs.StringVar(&labeler, "labeler", os.Getenv("USER"), "who labelled the sheet (accountable for the labels)")
	fs.Parse(rest)
	if file == "" {
		file = sheetPath(*dir)
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	rd := csv.NewReader(f)
	rd.FieldsPerRecord = -1
	rows, err := rd.ReadAll()
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if len(rows) == 0 || len(rows[0]) < 4 || rows[0][0] != "id" || rows[0][1] != "label" {
		return fmt.Errorf("%s: not a candidates sheet (expected header %v)", file, sheetHeader)
	}
	meta := map[string]GoldCase{}
	if cands, err := readJSONL[GoldCase](candidatesPath(*dir)); err == nil {
		for _, c := range cands {
			meta[goldID(c.Claim, c.Evidence)] = c
		}
	}
	gold, err := loadGold(*dir)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, g := range gold {
		have[g.ID] = true
	}
	added, blank, dup := 0, 0, 0
	var fresh []GoldCase
	for i, row := range rows[1:] {
		if len(row) < 4 {
			return fmt.Errorf("row %d: expected at least 4 columns, got %d", i+2, len(row))
		}
		if strings.TrimSpace(row[1]) == "" {
			blank++
			continue
		}
		c := GoldCase{Claim: row[2], Evidence: row[3], Label: row[1], Labeler: labeler}
		if m, ok := meta[goldID(row[2], row[3])]; ok {
			c.Group, c.DataClass, c.Source = m.Group, m.DataClass, m.Source
		}
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			c.Source = strings.TrimSpace(c.Source + " | note: " + row[4])
		}
		if err := c.normalize(); err != nil {
			return fmt.Errorf("row %d (%s): %w", i+2, row[0], err)
		}
		if c.ID != row[0] {
			return fmt.Errorf("row %d: id %s does not match its claim and evidence (edited text?); change the label column only", i+2, row[0])
		}
		if have[c.ID] {
			dup++
			continue
		}
		have[c.ID] = true
		fresh = append(fresh, c)
		added++
	}
	// Validate the whole sheet before writing any of it.
	for _, c := range fresh {
		if err := appendJSONL(goldPath(*dir), c); err != nil {
			return err
		}
	}
	fmt.Printf("imported %d labelled row(s) as %q; %d left blank, %d already in gold; gold now %d\n",
		added, labeler, blank, dup, len(gold)+added)
	return nil
}

func newRng(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// contextWords open sentences that cannot be judged without the sentence before them.
var contextWords = map[string]bool{
	"he": true, "she": true, "it": true, "they": true, "his": true, "her": true, "its": true, "their": true,
	"this": true, "that": true, "these": true, "those": true, "there": true, "then": true, "also": true,
	"however": true, "but": true, "and": true, "so": true, "such": true, "both": true, "another": true,
}

// dependsOnContext reports a sentence that starts with a pronoun or connective, whose
// subject is only named elsewhere. A claim has to stand alone to be labelled.
func dependsOnContext(s string) bool {
	w := strings.ToLower(strings.TrimFunc(strings.Fields(s)[0], func(r rune) bool { return !unicode.IsLetter(r) }))
	return contextWords[w]
}

// looksLikeProse rejects reference lists, category dumps and track listings: real
// paragraphs end sentences about as often as every 40 words and cite no "Retrieved".
func looksLikeProse(p string, words int) bool {
	if strings.Contains(p, "Retrieved ") || strings.Contains(p, " births ") {
		return false
	}
	return strings.Count(p, ". ")+strings.Count(p, "! ")+strings.Count(p, "? ")+1 >= words/40
}
