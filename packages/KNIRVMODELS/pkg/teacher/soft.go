package teacher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// Episode is the part of a SupervisionEpisode v1 the teacher reads. Field names match
// the head trainer's converter (pipeline/5_HEAD_TRAINER/src/episodes.rs).
type Episode struct {
	ID   string `json:"episode_id"`
	Alt  string `json:"event_id"`
	Task struct {
		Request string `json:"request"`
		Intent  string `json:"intent"`
		Risk    string `json:"risk"`
	} `json:"task"`
	State struct {
		Phase      string         `json:"phase"`
		Budget     map[string]int `json:"budget"`
		AgentChain []string       `json:"agent_chain"`
	} `json:"state"`
	StateBefore *struct {
		Phase      string         `json:"phase"`
		Budget     map[string]int `json:"budget"`
		AgentChain []string       `json:"agent_chain"`
	} `json:"state_before"`
	Evidence []struct {
		Class string `json:"class"`
	} `json:"evidence"`
	Trace struct {
		Agent string `json:"agent"`
	} `json:"trace"`
	Decision *struct {
		NextPhase string `json:"next_phase"`
		Abstain   bool   `json:"abstain"`
		Action    string `json:"recommended_action"`
	} `json:"decision"`
	Outcome *struct {
		Verified      bool `json:"verified"`
		HumanOverride bool `json:"human_override"`
	} `json:"outcome"`
}

func (e *Episode) id() string {
	if e.ID != "" {
		return e.ID
	}
	return e.Alt
}

// Verified reports the same condition the converter uses for a row to become a label.
func (e *Episode) Verified() bool {
	return e.Decision != nil && e.Outcome != nil && e.Outcome.Verified && !e.Outcome.HumanOverride &&
		!e.Decision.Abstain && e.Decision.Action != "" && e.Decision.Action != "abstain"
}

var redactions = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36}`),
	regexp.MustCompile(`(?i)(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*[^\s]{8,}`),
}

func redact(s string) string {
	for _, r := range redactions {
		s = r.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

// SoftHeads are the heads the teacher may label: choice heads whose options are the
// values seen in the data. `action` is never taught and `risk` is a score head.
var SoftHeads = []string{"intent", "next_phase", "agent"}

// DefaultSoftHeads are the heads labelled when none are named: the ones the head
// trainer will take. `intent` (and `risk`) are fields of the head's own prompt, so the
// trainer drops teacher distributions for them (finding H4); labelling them would only
// spend the teacher's slow single lane on data that is thrown away.
var DefaultSoftHeads = []string{"next_phase", "agent"}

var agents = map[string]bool{"claude": true, "codex": true, "opencode": true, "hermes": true}

// label is the episode's own (verified) value for a head, "" when it has none.
func (e *Episode) label(head string) string {
	switch head {
	case "intent":
		return e.Task.Intent
	case "next_phase":
		if e.Decision != nil {
			return e.Decision.NextPhase
		}
	case "agent":
		if agents[e.Trace.Agent] {
			return e.Trace.Agent
		}
	}
	return ""
}

// scenario describes the episode for one head WITHOUT the field being labelled, so
// the teacher judges from the request and context rather than copying the answer
// (the leak the head trainer's own prompt has for intent and risk).
func (e *Episode) scenario(head string) string {
	st := e.State
	if e.StateBefore != nil {
		st = struct {
			Phase      string         `json:"phase"`
			Budget     map[string]int `json:"budget"`
			AgentChain []string       `json:"agent_chain"`
		}(*e.StateBefore)
	}
	var classes []string
	for _, ev := range e.Evidence {
		classes = append(classes, ev.Class)
	}
	sort.Strings(classes)
	var b strings.Builder
	fmt.Fprintf(&b, "Current phase: %s\n", st.Phase)
	if head != "intent" && e.Task.Intent != "" {
		fmt.Fprintf(&b, "Intent: %s\n", e.Task.Intent)
	}
	if head != "agent" && len(st.AgentChain) > 0 {
		fmt.Fprintf(&b, "Agents so far: %s\n", strings.Join(st.AgentChain, " > "))
	}
	fmt.Fprintf(&b, "Evidence collected: %s\n", strings.Join(classes, ", "))
	fmt.Fprintf(&b, "Request: %s", redact(e.Task.Request))
	return b.String()
}

// SoftLine is one line of the candidate teacher file. The head trainer reads
// episode_id and labels; the rest is for people and audits.
type SoftLine struct {
	EpisodeID  string                        `json:"episode_id"`
	Labels     map[string]map[string]float64 `json:"labels"`
	Provenance Provenance                    `json:"provenance"`
	Agree      map[string]bool               `json:"order_agreement"`
}

// SoftStats counts the outcome of a run.
type SoftStats struct {
	Read, Used, Skipped, Failed int
	Heads                       map[string]int // head -> options
}

// SoftLabels reads episodes (JSONL), takes the verified ones, and writes teacher
// distributions over each head's options to w. Options are the values that occur in
// the verified episodes; a head with fewer than two is skipped.
func SoftLabels(ctx context.Context, cl *Client, cache *Cache, episodes io.Reader, heads []string, samples int, w io.Writer) (SoftStats, error) {
	st := SoftStats{Heads: map[string]int{}}
	var eps []*Episode
	sc := bufio.NewScanner(episodes)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		st.Read++
		var e Episode
		if json.Unmarshal([]byte(line), &e) != nil || e.id() == "" || !e.Verified() {
			st.Skipped++
			continue
		}
		eps = append(eps, &e)
	}
	if err := sc.Err(); err != nil {
		return st, err
	}
	options := map[string][]string{}
	for _, h := range heads {
		set := map[string]bool{}
		for _, e := range eps {
			if v := e.label(h); v != "" {
				set[v] = true
			}
		}
		if len(set) < 2 {
			continue
		}
		for v := range set {
			options[h] = append(options[h], v)
		}
		sort.Strings(options[h])
		st.Heads[h] = len(options[h])
	}
	if len(options) == 0 {
		return st, fmt.Errorf("no head has two or more options in %d verified episodes", len(eps))
	}
	enc := json.NewEncoder(w)
	for _, e := range eps {
		line := SoftLine{EpisodeID: e.id(), Labels: map[string]map[string]float64{}, Agree: map[string]bool{}}
		for _, h := range heads {
			opts, ok := options[h]
			if !ok {
				continue
			}
			var os []Option
			for _, o := range opts {
				os = append(os, Option{Key: o, Desc: o})
			}
			d, err := Vote(ctx, cl, cache, VoteInput{
				System:  "You are an expert reviewer of software-engineering supervision decisions. Choose the single best option.",
				Context: fmt.Sprintf("Decide the %q of this task.\n\n%s", strings.ReplaceAll(h, "_", " "), e.scenario(h)),
				Options: os, Samples: samples,
			})
			if err != nil {
				if ctx.Err() != nil {
					return st, ctx.Err()
				}
				continue
			}
			line.Labels[h], line.Agree[h], line.Provenance = d.Probs, d.Agree, d.Prov
		}
		if len(line.Labels) == 0 {
			st.Failed++
			continue
		}
		if err := enc.Encode(line); err != nil {
			return st, err
		}
		st.Used++
	}
	return st, nil
}
