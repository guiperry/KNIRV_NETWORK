package teacher

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Option is one answer the teacher may pick.
type Option struct {
	Key  string // what the caller records, e.g. "SUPPORTED"
	Desc string // what the teacher reads
}

// VoteInput is a closed-choice question put to the teacher several times.
type VoteInput struct {
	System  string
	Context string // everything before the options
	Options []Option
	// Samples is the number of votes (default 4). Half use the options in the given
	// order and half in reverse, so a position bias cancels instead of being counted.
	Samples     int
	Temperature float64 // default 0.6
}

// Distribution is the vote.
type Distribution struct {
	Probs  map[string]float64
	Counts map[string]int
	Valid  int // votes that named one of the options
	Asked  int
	// Agree is true when the forward and reversed orders picked the same winner.
	Agree bool
	Prov  Provenance
}

// Top is the leading option and its margin over the runner-up (0 when there is none).
func (d Distribution) Top() (key string, margin float64) {
	type kv struct {
		k string
		v float64
	}
	var all []kv
	for k, v := range d.Probs {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].k < all[j].k
	})
	if len(all) == 0 {
		return "", 0
	}
	if len(all) == 1 {
		return all[0].k, all[0].v
	}
	return all[0].k, all[0].v - all[1].v
}

var letterRE = regexp.MustCompile(`^\W*([A-Z])\b`)

func renderOptions(opts []Option) string {
	var b strings.Builder
	for i, o := range opts {
		fmt.Fprintf(&b, "%c: %s\n", 'A'+i, o.Desc)
	}
	return b.String()
}

// Vote asks the question Samples times and returns the distribution of the answers.
// Unparseable answers are dropped; a vote with none valid is an error.
func Vote(ctx context.Context, cl *Client, cache *Cache, in VoteInput) (Distribution, error) {
	if len(in.Options) < 2 || len(in.Options) > 26 {
		return Distribution{}, fmt.Errorf("need 2 to 26 options, got %d", len(in.Options))
	}
	n := in.Samples
	if n <= 0 {
		n = 4
	}
	temp := in.Temperature
	if temp == 0 {
		temp = 0.6
	}
	rev := make([]Option, len(in.Options))
	for i, o := range in.Options {
		rev[len(rev)-1-i] = o
	}
	d := Distribution{Probs: map[string]float64{}, Counts: map[string]int{}, Asked: n}
	byOrder := [2]map[string]int{{}, {}}
	for i := 0; i < n; i++ {
		order := in.Options
		if i%2 == 1 {
			order = rev
		}
		req := Request{
			System: in.System,
			User: in.Context + "\n\nOptions:\n" + renderOptions(order) +
				"Answer with the letter of one option and nothing else.",
			MaxTokens:   8,
			Temperature: temp,
		}
		rep, _, err := cache.Sample(ctx, cl, req, i)
		if err != nil {
			return Distribution{}, err
		}
		d.Prov = rep.Prov
		m := letterRE.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(rep.Text)))
		if m == nil {
			continue
		}
		idx := int(m[1][0] - 'A')
		if idx >= len(order) {
			continue
		}
		key := order[idx].Key
		d.Counts[key]++
		byOrder[i%2][key]++
		d.Valid++
	}
	if d.Valid == 0 {
		return d, fmt.Errorf("none of %d answers named an option", n)
	}
	for k, c := range d.Counts {
		d.Probs[k] = float64(c) / float64(d.Valid)
	}
	d.Prov.Samples, d.Prov.Temp = d.Valid, temp
	d.Agree = winner(byOrder[0]) == winner(byOrder[1])
	return d, nil
}

func winner(m map[string]int) string {
	best, bc := "", 0
	for k, c := range m {
		if c > bc || (c == bc && k < best) {
			best, bc = k, c
		}
	}
	return best
}
