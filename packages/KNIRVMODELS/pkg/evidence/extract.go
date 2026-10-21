package evidence

import (
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// WeightedKey is a key and the family it came from; the family sets its weight in
// scoring (an exact subject/property hit outweighs a shared rare word).
type WeightedKey struct {
	Key  Key
	Kind Kind
}

// Features are the things in a passage the key families are built from.
type Features struct {
	Entities []string // canonical named entities
	Numbers  []string // canonical numbers, ISO dates and years
	Terms    []string // distinct content words (lower case)
	Words    []string // all words, in order, lower case (for shingles)
}

var stopWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`about above after again against all also among and any are because been before being below between both but can could did does doing down during each few for from further had has have having here how into its itself just more most much must not now off once only other our out over own same should some such than that the their them then there these they this those through too under until upon very was were what when where which while who whom why will with would you your yours were been being than then thus whose whether although since however therefore within without across around behind beyond near several many`) {
		m[w] = true
	}
	return m
}()

// sentence starters that are capitalized only because they start a sentence.
var starters = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`The A An In On At It He She They We You I This That These Those There Here When While After Before During Since Although However But And Or If As For To Of With By From`) {
		m[w] = true
	}
	return m
}()

var connectors = map[string]bool{"of": true, "de": true, "van": true, "von": true, "the": true, "and": true, "la": true, "del": true, "al": true, "bin": true}

type token struct {
	text  string
	start bool // first token of a sentence
}

func tokenize(s string) []token {
	var out []token
	var cur []rune
	sentenceStart := true
	flush := func() {
		if len(cur) > 0 {
			out = append(out, token{text: string(cur), start: sentenceStart})
			sentenceStart = false
			cur = cur[:0]
		}
	}
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' && len(cur) > 0 || r == '-' && len(cur) > 0:
			cur = append(cur, r)
		default:
			flush()
			if r == '.' || r == '!' || r == '?' || r == '\n' || r == ':' {
				sentenceStart = true
			}
		}
	}
	flush()
	for i := range out {
		out[i].text = strings.Trim(out[i].text, "'-")
	}
	return out
}

func capitalized(w string) bool {
	for _, r := range w {
		return unicode.IsUpper(r)
	}
	return false
}

// ExtractFeatures reads entities, numbers, terms and words from text.
func ExtractFeatures(text string) Features {
	toks := tokenize(text)
	var f Features
	seenT := map[string]bool{}
	for _, t := range toks {
		w := strings.ToLower(t.text)
		f.Words = append(f.Words, w)
		if len(w) >= 4 && !stopWords[w] && !allDigits(w) && !seenT[w] {
			seenT[w] = true
			f.Terms = append(f.Terms, w)
		}
	}
	// Entities: runs of capitalized tokens, allowing one connector word inside.
	seenE := map[string]bool{}
	for i := 0; i < len(toks); {
		if !capitalized(toks[i].text) || allDigits(toks[i].text) {
			i++
			continue
		}
		j := i + 1
		for j < len(toks) {
			if capitalized(toks[j].text) {
				j++
				continue
			}
			if connectors[strings.ToLower(toks[j].text)] && j+1 < len(toks) && capitalized(toks[j+1].text) {
				j += 2
				continue
			}
			break
		}
		run := toks[i:j]
		// A lone capitalized sentence starter ("The", "In") is not an entity.
		if len(run) == 1 && (run[0].start && starters[run[0].text] || starters[run[0].text]) {
			i = j
			continue
		}
		if len(run) > 1 && run[0].start && starters[run[0].text] {
			run = run[1:] // "The Eiffel Tower" -> "Eiffel Tower"
		}
		var parts []string
		for _, t := range run {
			parts = append(parts, t.text)
		}
		c := Canon(strings.Join(parts, " "))
		if c != "" && len(c) >= 2 && !seenE[c] {
			seenE[c] = true
			f.Entities = append(f.Entities, c)
		}
		i = j
	}
	seenN := map[string]bool{}
	add := func(v string) {
		if !seenN[v] {
			seenN[v] = true
			f.Numbers = append(f.Numbers, v)
		}
	}
	for _, d := range Dates(text) {
		add(d)
	}
	for _, n := range Numbers(text) {
		add(n)
	}
	return f
}

func allDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// minHashBands returns band keys over the 3-word shingles of words: 16 hash
// functions in 8 bands of 2. Two passages that share most shingles share bands.
func minHashBands(words []string) []Key {
	if len(words) < 3 {
		return nil
	}
	const funcs, rows = 16, 2
	mins := make([]uint64, funcs)
	for i := range mins {
		mins[i] = ^uint64(0)
	}
	for i := 0; i+3 <= len(words); i++ {
		sh := words[i] + " " + words[i+1] + " " + words[i+2]
		for f := 0; f < funcs; f++ {
			h := fnv.New64a()
			h.Write([]byte{byte(f)})
			h.Write([]byte(sh))
			if v := h.Sum64(); v < mins[f] {
				mins[f] = v
			}
		}
	}
	var out []Key
	for b := 0; b < funcs/rows; b++ {
		out = append(out, HashKey(KindMinHash, strconv.Itoa(b), strconv.FormatUint(mins[b*rows], 36), strconv.FormatUint(mins[b*rows+1], 36)))
	}
	return out
}

func dedup(ks []WeightedKey) []WeightedKey {
	seen := map[Key]bool{}
	out := ks[:0:0]
	for _, k := range ks {
		if !seen[k.Key] {
			seen[k.Key] = true
			out = append(out, k)
		}
	}
	return out
}

// maxTermKeys is how many distinctive terms a chunk is indexed under; with about 10
// entity, 5 number and 8 MinHash keys this keeps a chunk near the plan's 30 keys.
const maxTermKeys = 12

// ChunkKeys are the keys a passage is indexed under. df maps a term to the number
// of chunks in the ingest batch that contain it, so each chunk is indexed under its
// rarest words (a term shared by every chunk finds nothing). A nil df falls back to
// the longest words.
func ChunkKeys(f Features, df map[string]int) []WeightedKey {
	var ks []WeightedKey
	for _, e := range f.Entities {
		ks = append(ks, WeightedKey{HashKey(KindEntity, e), KindEntity})
	}
	for _, n := range f.Numbers {
		ks = append(ks, WeightedKey{HashKey(KindNumber, n), KindNumber})
	}
	type scored struct {
		term string
		df   int
	}
	sc := make([]scored, len(f.Terms))
	for i, t := range f.Terms {
		sc[i] = scored{t, df[t]} // one map lookup per term, not one per comparison
	}
	sort.SliceStable(sc, func(i, j int) bool {
		if sc[i].df != sc[j].df {
			return sc[i].df < sc[j].df
		}
		return len(sc[i].term) > len(sc[j].term)
	})
	if len(sc) > maxTermKeys {
		sc = sc[:maxTermKeys]
	}
	terms := make([]string, len(sc))
	for i, x := range sc {
		terms[i] = x.term
	}
	for _, t := range terms {
		ks = append(ks, WeightedKey{HashKey(KindTerm, t), KindTerm})
	}
	for _, k := range minHashBands(f.Words) {
		ks = append(ks, WeightedKey{k, KindMinHash})
	}
	return dedup(ks)
}

// Claim is what retrieval is asked about. Subject and Property are optional
// structured fields from the claim extractor; Text is always present.
type Claim struct {
	Text     string `json:"text"`
	Subject  string `json:"subject,omitempty"`
	Property string `json:"property,omitempty"`
	Value    string `json:"value,omitempty"`
}

// ClaimKeys are the keys a claim looks up: every one of its entities, numbers and
// content words, its MinHash bands, and the exact subject/property key when given.
func ClaimKeys(c Claim) []WeightedKey {
	var ks []WeightedKey
	if c.Subject != "" && c.Property != "" {
		ks = append(ks, WeightedKey{HashKey(KindSubjectProperty, Canon(c.Subject), Canon(c.Property)), KindSubjectProperty})
	}
	f := ExtractFeatures(c.Text + " " + c.Subject + " " + c.Value)
	for _, e := range f.Entities {
		ks = append(ks, WeightedKey{HashKey(KindEntity, e), KindEntity})
	}
	if c.Subject != "" {
		if e := Canon(c.Subject); e != "" {
			ks = append(ks, WeightedKey{HashKey(KindEntity, e), KindEntity})
		}
	}
	for _, n := range f.Numbers {
		ks = append(ks, WeightedKey{HashKey(KindNumber, n), KindNumber})
	}
	for _, t := range f.Terms {
		ks = append(ks, WeightedKey{HashKey(KindTerm, t), KindTerm})
	}
	for _, k := range minHashBands(f.Words) {
		ks = append(ks, WeightedKey{k, KindMinHash})
	}
	for _, g := range titleGrams(c.Text + " " + c.Subject) {
		ks = append(ks, WeightedKey{HashKey(KindTitle, g), KindTitle})
	}
	return dedup(ks)
}

// FactKey is the exact-match key for a structured fact (subject, property).
func FactKey(subject, property string) Key {
	return HashKey(KindSubjectProperty, Canon(subject), Canon(property))
}

// TitleKey is the key every passage of a document is indexed under for its title.
// A claim that names a thing reaches that thing's article even when the article's
// own text spells the name differently ("Barack Hussein Obama II" for "Barack Obama").
func TitleKey(title string) (Key, bool) {
	c := Canon(title)
	if c == "" {
		return Key{}, false
	}
	return HashKey(KindTitle, c), true
}

// titleGrams are the word n-grams (1 to 5) of a claim, canonicalized, that could be
// the title of an article. N-grams made only of stop words are skipped.
func titleGrams(text string) []string {
	words := strings.Fields(Canon(text))
	seen := map[string]bool{}
	var out []string
	for n := 1; n <= 5; n++ {
		for i := 0; i+n <= len(words); i++ {
			g := words[i : i+n]
			allStop := true
			for _, w := range g {
				if !stopWords[w] && len(w) > 1 {
					allStop = false
				}
			}
			if allStop || stopWords[g[0]] || stopWords[g[len(g)-1]] {
				continue
			}
			s := strings.Join(g, " ")
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}
