// Command teacher runs batch jobs on the teacher backend through the
// inference socket: synthetic training data for adapters, soft labels for the head,
// and a status check. Gold-set suggestions and retrieval judging are `measure`
// commands, next to the gold set they work on.
//
// The teacher is Colibri while it is enabled and llama.cpp when it is not (hosts that
// fail Colibri's requirements: see GET /harness/status, field "teacher"). A job waits
// for that one backend, never moves between them mid-run, and records which model
// produced each line.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"knirvmodels/pkg/teacher"
)

const defaultSocket = "/var/lib/knirvserver/sockets/llama.sock"

func usage() {
	fmt.Fprint(os.Stderr, `teacher: batch jobs on the teacher backend (Colibri, or llama.cpp when Colibri is not enabled)

  teacher status                      is a teacher backend serving, and how fast
  teacher synth        -seeds FILE -skill NAME -out FILE   synthetic instruction/response pairs for an adapter
  teacher soft-labels  -episodes FILE -out FILE            teacher distributions for the head (candidate file)

Common flags: -socket PATH (default `+defaultSocket+`), -cache FILE, -wait DURATION.
Jobs are resumable: answers are cached by model and prompt, so rerunning a job is cheap.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var err error
	switch os.Args[1] {
	case "status":
		err = cmdStatus(ctx, os.Args[2:])
	case "synth":
		err = cmdSynth(ctx, os.Args[2:])
	case "soft-labels":
		err = cmdSoft(ctx, os.Args[2:])
	case "-h", "-help", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "teacher:", err)
		os.Exit(1)
	}
}

type common struct {
	socket, cache *string
	wait          *time.Duration
}

func addCommon(fs *flag.FlagSet, name string) common {
	return common{
		socket: fs.String("socket", defaultSocket, "inference socket"),
		cache:  fs.String("cache", name+".cache.jsonl", "answer cache (resumes interrupted jobs)"),
		wait:   fs.Duration("wait", 30*time.Minute, "how long to wait for the teacher backend to be ready"),
	}
}

func (c common) client() (*teacher.Client, *teacher.Cache, error) {
	cl := teacher.NewUnix(*c.socket, 0)
	cl.Patience = *c.wait
	cl.Logf = func(f string, a ...any) { fmt.Fprintf(os.Stderr, "teacher: "+f+"\n", a...) }
	cache, err := teacher.OpenCache(*c.cache)
	return cl, cache, err
}

func cmdStatus(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	socket := fs.String("socket", defaultSocket, "inference socket")
	fs.Parse(args)
	st, err := teacher.NewUnix(*socket, 30*time.Second).Status(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(b))
	if st["teacher"] == nil {
		return fmt.Errorf("no teacher backend is ready: Colibri is enabled but not serving yet, or llama.cpp has not started (see colibri_qualification / primary_note above)")
	}
	fmt.Fprintf(os.Stderr, "teacher backend: %v\n", st["teacher"])
	return nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			out = append(out, l)
		}
	}
	return out, sc.Err()
}

// excludeTexts loads the texts that must never become training data: every string
// field of every JSONL line (the gold set's claims and evidence, exam items).
func excludeTexts(paths string) ([]string, error) {
	var out []string
	for _, p := range strings.Split(paths, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		lines, err := readLines(p)
		if err != nil {
			return nil, err
		}
		for _, l := range lines {
			var m map[string]any
			if json.Unmarshal([]byte(l), &m) != nil {
				out = append(out, l)
				continue
			}
			for _, v := range m {
				if s, ok := v.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out, nil
}

func cmdSynth(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("synth", flag.ExitOnError)
	cm := addCommon(fs, "synth")
	seeds := fs.String("seeds", "", "file of example prompts, one per line (or JSONL with a \"prompt\" field)")
	skill := fs.String("skill", "", "what the adapter is for")
	system := fs.String("system", "", "instructions the responses are written under (default: a helpful, concise assistant for the skill)")
	out := fs.String("out", "", "output JSONL (ulora-sft/v1)")
	expand := fs.Int("expand", 0, "new prompts the teacher writes per seed")
	maxTokens := fs.Int("max-tokens", 512, "longest response; longer ones are dropped as truncated")
	exclude := fs.String("exclude", "", "comma-separated JSONL files (gold set, exams) whose text must not appear in the data")
	fs.Parse(args)
	if *seeds == "" || *skill == "" || *out == "" {
		return fmt.Errorf("need -seeds, -skill and -out")
	}
	lines, err := readLines(*seeds)
	if err != nil {
		return err
	}
	var prompts []string
	for _, l := range lines {
		var m struct{ Prompt string }
		if strings.HasPrefix(l, "{") && json.Unmarshal([]byte(l), &m) == nil && m.Prompt != "" {
			l = m.Prompt
		}
		prompts = append(prompts, l)
	}
	excl, err := excludeTexts(*exclude)
	if err != nil {
		return fmt.Errorf("-exclude: %w", err)
	}
	if *system == "" {
		*system = "You are a helpful, accurate assistant specialised in: " + *skill + ". Answer concisely."
	}
	cl, cache, err := cm.client()
	if err != nil {
		return err
	}
	defer cache.Close()
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := teacher.GenerateSFT(ctx, cl, cache, teacher.SFTSpec{
		Skill: *skill, System: *system, Seeds: prompts, ExpandPer: *expand, MaxTokens: *maxTokens, Exclude: excl,
	}, f)
	fmt.Fprintf(os.Stderr, "teacher: wrote %d records to %s (cached %d, duplicates %d, excluded as gold/exam overlap %d, truncated %d, too short %d, failed %d)\n",
		st.Written, *out, st.Cached, st.Duplicates, st.Excluded, st.Truncated, st.TooShort, st.Failed)
	return err
}

func cmdSoft(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("soft-labels", flag.ExitOnError)
	cm := addCommon(fs, "soft")
	episodes := fs.String("episodes", "", "SupervisionEpisode v1 JSONL (the head trainer's gold.jsonl)")
	out := fs.String("out", "", "teacher file; head-work/teacher.jsonl is picked up by the head cycle and applied to the training split only")
	heads := fs.String("heads", strings.Join(teacher.DefaultSoftHeads, ","), "heads to label")
	samples := fs.Int("samples", 4, "votes per head")
	fs.Parse(args)
	if *episodes == "" || *out == "" {
		return fmt.Errorf("need -episodes and -out")
	}
	in, err := os.Open(*episodes)
	if err != nil {
		return err
	}
	defer in.Close()
	cl, cache, err := cm.client()
	if err != nil {
		return err
	}
	defer cache.Close()
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := teacher.SoftLabels(ctx, cl, cache, in, strings.Split(*heads, ","), *samples, f)
	fmt.Fprintf(os.Stderr, "teacher: %d episodes read, %d labelled, %d skipped (unverified/malformed), %d failed; options per head: %v\n",
		st.Read, st.Used, st.Skipped, st.Failed, st.Heads)
	return err
}
