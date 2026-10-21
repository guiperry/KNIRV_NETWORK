package livesearch

import (
	"errors"
	"regexp"
	"strings"

	"knirvmodels/pkg/evidence"
)

// ErrNoQuery means nothing safe and useful could be sent for this claim.
var ErrNoQuery = errors.New("livesearch: the claim has nothing to search for after redaction")

// Outbound text is built from the claim only, never from the user's prompt, and every
// word in it is a word of the claim. Secrets and personal identifiers are removed before
// words are chosen, so they cannot be chosen.
var redactions = []*regexp.Regexp{
	// The labelled form goes first, so the label is removed together with its value.
	regexp.MustCompile(`(?i)(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*\S+`), // key=value secrets
	regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),                   // email
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                              // AWS key id
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36}`),                                     // GitHub token
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{16,}`),                                        // API-style key
	regexp.MustCompile(`https?://\S+`),                                                  // any URL
	regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),                                         // US SSN
	regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`),                                        // card-like digit runs
	regexp.MustCompile(`\+?\d[\d\s().\-]{8,}\d`),                                        // phone numbers
	regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),                                   // IPv4
	regexp.MustCompile(`\b[A-Fa-f0-9]{24,}\b`),                                          // long hex (hashes, ids)
	regexp.MustCompile(`\b[A-Za-z0-9+/_\-]{32,}={0,2}`),                                 // long base64-ish blobs
}

// Redact removes secrets and personal identifiers from s.
func Redact(s string) string {
	for _, r := range redactions {
		s = r.ReplaceAllString(s, " ")
	}
	return s
}

const (
	maxQueryWords = 8
	maxQueryChars = 100
)

// Minimize turns a claim into the search string that leaves the machine: the claim's
// named entities first (what a search engine needs most), then its rarest content words,
// then at most a couple of numbers, at most 8 words and 100 characters in all.
func Minimize(c evidence.Claim) (string, error) {
	text := Redact(c.Text)
	subject := Redact(c.Subject)
	f := evidence.ExtractFeatures(text + " " + subject)
	var words []string
	seen := map[string]bool{}
	add := func(ws ...string) {
		for _, w := range ws {
			w = strings.TrimSpace(w)
			if w == "" || seen[w] || len(words) >= maxQueryWords {
				continue
			}
			seen[w] = true
			words = append(words, w)
		}
	}
	for _, e := range f.Entities {
		add(strings.Fields(e)...)
	}
	nums := 0
	for _, n := range f.Numbers {
		if nums < 2 && len(n) >= 2 {
			add(n)
			nums++
		}
	}
	terms := append([]string(nil), f.Terms...)
	// Longer words first: they are rarer on average, and this needs no corpus statistics.
	for i := 1; i < len(terms); i++ {
		for j := i; j > 0 && len(terms[j]) > len(terms[j-1]); j-- {
			terms[j], terms[j-1] = terms[j-1], terms[j]
		}
	}
	for _, t := range terms {
		if len(words) >= 5 {
			break
		}
		add(t)
	}
	q := strings.Join(words, " ")
	if len(q) > maxQueryChars {
		q = strings.TrimSpace(q[:maxQueryChars])
		if i := strings.LastIndexByte(q, ' '); i > 0 {
			q = q[:i]
		}
	}
	if len(strings.Fields(q)) == 0 {
		return "", ErrNoQuery
	}
	return q, nil
}
