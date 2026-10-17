package teacher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// SFTRecord is one synthetic instruction/response pair for training an adapter.
type SFTRecord struct {
	Schema     string     `json:"schema"` // "ulora-sft/v1"
	ID         string     `json:"id"`
	Skill      string     `json:"skill"`
	Prompt     string     `json:"prompt"`
	Response   string     `json:"response"`
	Source     string     `json:"source"` // "seed" or "expanded"
	Synthetic  bool       `json:"synthetic"`
	Provenance Provenance `json:"provenance"`
}

// SFTSpec describes the data to make.
type SFTSpec struct {
	Skill  string   // what the adapter is for
	System string   // the role/instructions the responses are written under
	Seeds  []string // example user prompts
	// ExpandPer asks the teacher for this many new prompts per seed (0: seeds only).
	ExpandPer int
	// MaxTokens bounds one response; a response that hits it is dropped as truncated.
	MaxTokens int
	// Exclude lists texts that must not appear in training data (the gold set and the
	// exams): a prompt that contains one, or is contained in one, is dropped.
	Exclude []string
	// MinChars drops responses shorter than this (default 20).
	MinChars int
}

var listPrefixRE = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)])\s*`)

// ExpandSeeds asks the teacher for variations of a seed prompt.
func ExpandSeeds(ctx context.Context, cl *Client, cache *Cache, skill, seed string, n int) ([]string, error) {
	req := Request{
		System: "You write realistic user requests for testing an AI assistant. Output only the requests, one per line, numbered, no commentary.",
		User: fmt.Sprintf("Skill under test: %s\nExample request: %s\n\nWrite %d different requests a user might send for this skill. "+
			"Vary wording, length and detail; do not repeat the example.", skill, seed, n),
		MaxTokens:   64 * n,
		Temperature: 0.8,
	}
	rep, _, err := cache.Sample(ctx, cl, req, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(rep.Text, "\n") {
		line = strings.TrimSpace(listPrefixRE.ReplaceAllString(line, ""))
		if len(line) >= 8 && !strings.EqualFold(line, seed) {
			out = append(out, line)
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// SFTStats counts what happened to the candidate prompts.
type SFTStats struct {
	Written, Cached, Duplicates, Excluded, Truncated, TooShort, Failed int
}

// GenerateSFT writes records to w as JSONL and returns what it did. Repeated prompts,
// prompts that overlap the Exclude texts, truncated and very short responses are
// dropped. Failures are counted and skipped, so one bad request does not lose a long
// job; the first error is returned with the stats if nothing could be generated.
func GenerateSFT(ctx context.Context, cl *Client, cache *Cache, spec SFTSpec, w io.Writer) (SFTStats, error) {
	var st SFTStats
	if spec.MaxTokens <= 0 {
		spec.MaxTokens = 512
	}
	if spec.MinChars <= 0 {
		spec.MinChars = 20
	}
	excl := make([]string, 0, len(spec.Exclude))
	for _, e := range spec.Exclude {
		if e = normalize(e); len(e) >= 20 {
			excl = append(excl, e)
		}
	}
	seen := map[string]bool{}
	type cand struct{ prompt, source string }
	var cands []cand
	var firstErr error
	for _, seed := range spec.Seeds {
		cands = append(cands, cand{seed, "seed"})
		if spec.ExpandPer > 0 {
			more, err := ExpandSeeds(ctx, cl, cache, spec.Skill, seed, spec.ExpandPer)
			if err != nil {
				if ctx.Err() != nil {
					return st, ctx.Err()
				}
				st.Failed++
				firstErr = firstNonNil(firstErr, err)
				continue
			}
			for _, m := range more {
				cands = append(cands, cand{m, "expanded"})
			}
		}
	}
	enc := json.NewEncoder(w)
	for _, c := range cands {
		key := normalize(c.prompt)
		if seen[key] {
			st.Duplicates++
			continue
		}
		seen[key] = true
		if overlaps(key, excl) {
			st.Excluded++
			continue
		}
		req := Request{System: spec.System, User: c.prompt, MaxTokens: spec.MaxTokens, Temperature: 0.7}
		rep, cached, err := cache.Sample(ctx, cl, req, 0)
		if err != nil {
			if ctx.Err() != nil {
				return st, ctx.Err()
			}
			st.Failed++
			firstErr = firstNonNil(firstErr, err)
			continue
		}
		switch {
		case rep.CompletionTokens >= spec.MaxTokens:
			st.Truncated++
			continue
		case len(rep.Text) < spec.MinChars:
			st.TooShort++
			continue
		}
		if cached {
			st.Cached++
		}
		sum := sha256.Sum256([]byte(spec.Skill + "\x00" + key))
		if err := enc.Encode(SFTRecord{
			Schema: "ulora-sft/v1", ID: hex.EncodeToString(sum[:8]), Skill: spec.Skill,
			Prompt: c.prompt, Response: rep.Text, Source: c.source, Synthetic: true, Provenance: rep.Prov,
		}); err != nil {
			return st, err
		}
		st.Written++
	}
	if st.Written == 0 && firstErr != nil {
		return st, firstErr
	}
	return st, nil
}

func overlaps(prompt string, excl []string) bool {
	for _, e := range excl {
		if strings.Contains(prompt, e) || (len(prompt) >= 20 && strings.Contains(e, prompt)) {
			return true
		}
	}
	return false
}

func firstNonNil(a, b error) error {
	if a != nil {
		return a
	}
	return b
}
