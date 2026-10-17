package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// expectedProbes lists every probe the plan needs, in plan order, with what a
// "blocked" result is waiting on. The report shows a row even when a probe never ran.
var expectedProbes = []struct{ name, item, needs string }{
	{"host", "host snapshot", "run `measure host`"},
	{"llama", "(a) llama latency / cache / grammar / tool envelope", "a running llama (`measure llama -llama-sock ...`)"},
	{"llama-sweep", "(a) ctx_size / threads / parallel", "a readable GGUF + llama-server binary (`measure llama-sweep -yes`)"},
	{"heartbench", "(a) HEART-on-Qwen quality gate", "verified decision cases in heart_cases.jsonl + a running llama"},
	{"gold", "(b) factuality gold set", "the owner's labels via `measure gold add` (D4)"},
	{"retrieval", "(c) lexical retrieval recall", "gold set"},
	{"judge", "(d) Qwen judge accuracy / false-support", "gold set + a running llama"},
	{"embed", "(e) embedder bake-off", "gold set + Ollama (+ Cloudflare)"},
	{"cloudflare", "(e) Cloudflare endpoint and cap", "run `measure cloudflare`"},
	{"lora", "(f) LoRA hot-swap capability", "raw llama-server address; -new-adapter for runtime loading"},
	{"ollama", "(g) authoritative Ollama instance", "run `measure ollama`"},
}

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	fs.Parse(args)

	results := map[string]*Result{}
	// gold is derived from the label file, not stored by a probe.
	if cs, err := loadGold(*dir); err == nil {
		g := goldStats(cs)
		results["gold"] = g
		g.write(*dir)
	}
	ents, _ := os.ReadDir(filepath.Join(*dir, "results"))
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(*dir, "results", e.Name()))
		if err != nil {
			continue
		}
		var r Result
		if json.Unmarshal(b, &r) == nil && results[r.Probe] == nil {
			results[r.Probe] = &r
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Baseline measurements\n\nGenerated %s from `%s/results`. Thresholds marked placeholder are the plan's initial values; refit them from these numbers.\n\n",
		time.Now().UTC().Format(time.RFC3339), *dir)
	sb.WriteString("| item | status | checks | detail |\n|---|---|---|---|\n")
	var pending []string
	failing := 0
	for _, p := range expectedProbes {
		r := results[p.name]
		if r == nil {
			fmt.Fprintf(&sb, "| %s | not run | | needs: %s |\n", p.item, p.needs)
			pending = append(pending, p.item+": "+p.needs)
			continue
		}
		pass, total := 0, len(r.Checks)
		for _, c := range r.Checks {
			if c.Pass {
				pass++
			}
		}
		detail := r.Reason
		if r.Status == statusBlocked {
			pending = append(pending, p.item+": "+r.Reason)
		}
		if pass < total {
			failing += total - pass
		}
		fmt.Fprintf(&sb, "| %s | %s | %d/%d | %s |\n", p.item, r.Status, pass, total, strings.ReplaceAll(detail, "|", "/"))
	}

	sb.WriteString("\n## Checks\n\n")
	names := make([]string, 0, len(results))
	for n := range results {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		r := results[n]
		if len(r.Checks) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "**%s**\n\n", n)
		for _, c := range r.Checks {
			mark := "pass"
			if !c.Pass {
				mark = "FAIL"
			}
			fmt.Fprintf(&sb, "- [%s] %s %s\n", mark, c.Name, c.Info)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Go / no-go\n\n")
	switch {
	case len(pending) > 0:
		fmt.Fprintf(&sb, "**Undecided**: %d item(s) have no measurement yet.\n\n", len(pending))
		for _, p := range pending {
			fmt.Fprintf(&sb, "- %s\n", p)
		}
	case failing > 0:
		fmt.Fprintf(&sb, "**No-go** on %d failing check(s) above.\n", failing)
	default:
		sb.WriteString("**Go**: every item measured and every check passes.\n")
	}
	for _, n := range names {
		if notes := results[n].Notes; len(notes) > 0 {
			fmt.Fprintf(&sb, "\n### %s notes\n\n", n)
			for _, x := range notes {
				fmt.Fprintf(&sb, "- %s\n", x)
			}
		}
	}

	out := filepath.Join(*dir, "REPORT.md")
	if err := os.WriteFile(out, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", out)
	return nil
}
