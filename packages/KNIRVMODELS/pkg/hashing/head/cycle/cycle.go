// Package cycle is stage 5 of the HASHER pipeline: it turns collected
// supervisor Bronze events into a candidate HEAD and offers it to the promotion
// gate.
//
//	Bronze JSONL --(Silver normalise + dedup)--> Gold episodes
//	  --(head-trainer convert)--> head-train/v1 --(head-trainer)--> <out>/head/ (gated)
//
// A cycle never changes the served model by itself: head-trainer publishes only
// when the candidate beats the current model on the same held-out set, and HEART
// hot-swaps whatever `model_latest` points at. Running a cycle is therefore
// always safe; at worst it ends "refused".
package cycle

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"knirvmodels/pkg/hashing/supervision/pipeline"
)

// Outcomes recorded in State.Outcome.
const (
	Skipped  = "skipped"  // not enough new data, or too soon
	Promoted = "promoted" // trainer exit 0
	Refused  = "refused"  // trainer exit 3: trained, gate said no
	Failed   = "failed"
)

// Config is one cycle's inputs. Zero values take the defaults noted.
type Config struct {
	BronzeDir  string // *.jsonl Bronze events (required)
	WorkDir    string // scratch + state (required)
	OutDir     string // trainer --out; the model lands in OutDir/head (required)
	TrainerBin string // path to head-trainer (required)

	Preset     string        // default "small"
	Epochs     int           // default 20
	LR         float64       // default 3e-4
	Seed       uint64        // default 1
	PolicyHash string        // recorded in the model config
	SoftLabels string        // optional teacher distributions (--soft); used only if the file exists
	MinNew     int           // new Gold episodes needed since the last cycle (default 200)
	MinGap     time.Duration // minimum time between cycles (default 6h)
	Force      bool          // ignore MinNew/MinGap

	Log func(format string, args ...any)
}

// State is persisted to WorkDir/state.json after every cycle.
type State struct {
	At       time.Time `json:"at"`
	Outcome  string    `json:"outcome"`
	Reason   string    `json:"reason,omitempty"`
	Episodes int       `json:"episodes"`
	Trained  int       `json:"trained_on"` // episodes seen by the last cycle that actually trained
	Preset   string    `json:"preset,omitempty"`
	Metrics  string    `json:"metrics,omitempty"` // path to the trainer's metrics.json
	Output   string    `json:"output_tail,omitempty"`
}

func (c *Config) defaults() {
	if c.Preset == "" {
		c.Preset = "small"
	}
	if c.Epochs == 0 {
		c.Epochs = 20
	}
	if c.LR == 0 {
		c.LR = 3e-4
	}
	if c.Seed == 0 {
		c.Seed = 1
	}
	if c.MinNew == 0 {
		c.MinNew = 200
	}
	if c.MinGap == 0 {
		c.MinGap = 6 * time.Hour
	}
	if c.Log == nil {
		c.Log = func(string, ...any) {}
	}
}

// LoadState returns the last recorded state; the zero State if none.
func LoadState(workDir string) State {
	var s State
	if b, err := os.ReadFile(filepath.Join(workDir, "state.json")); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func saveState(workDir string, s State) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(workDir, "state.json"), b, 0o640)
	}
}

// Run executes one cycle. The returned error is non-nil only for Failed.
func Run(ctx context.Context, cfg Config) (State, error) {
	cfg.defaults()
	for name, v := range map[string]string{"BronzeDir": cfg.BronzeDir, "WorkDir": cfg.WorkDir, "OutDir": cfg.OutDir, "TrainerBin": cfg.TrainerBin} {
		if v == "" {
			return State{}, fmt.Errorf("cycle: %s is required", name)
		}
	}
	if err := os.MkdirAll(cfg.WorkDir, 0o750); err != nil {
		return State{}, err
	}
	prev := LoadState(cfg.WorkDir)
	st := State{At: time.Now().UTC(), Preset: cfg.Preset, Trained: prev.Trained}
	finish := func(outcome, reason string, err error) (State, error) {
		st.Outcome, st.Reason = outcome, reason
		// A skip must not erase the record of the last real cycle.
		if outcome != Skipped {
			saveState(cfg.WorkDir, st)
		}
		return st, err
	}

	if !cfg.Force && !prev.At.IsZero() && time.Since(prev.At) < cfg.MinGap {
		return finish(Skipped, fmt.Sprintf("last cycle %s ago (< %s)", time.Since(prev.At).Round(time.Second), cfg.MinGap), nil)
	}

	gold := filepath.Join(cfg.WorkDir, "gold.jsonl")
	n, err := materialize(cfg.BronzeDir, gold)
	if err != nil {
		if errors.Is(err, errNoBronze) {
			st.Episodes = 0
			return finish(Skipped, err.Error(), nil)
		}
		return finish(Failed, "gold: "+err.Error(), err)
	}
	st.Episodes = n
	if !cfg.Force && n-prev.Trained < cfg.MinNew {
		return finish(Skipped, fmt.Sprintf("%d new episodes (< %d)", n-prev.Trained, cfg.MinNew), nil)
	}

	train := filepath.Join(cfg.WorkDir, "train.jsonl")
	conv := []string{"convert", "--episodes", gold, "--out", train}
	if cfg.SoftLabels != "" {
		if _, err := os.Stat(cfg.SoftLabels); err == nil {
			conv = append(conv, "--soft", cfg.SoftLabels)
			cfg.Log("[head] using teacher labels %s", cfg.SoftLabels)
		}
	}
	if out, code, err := runTrainer(ctx, cfg.TrainerBin, conv); err != nil || code != 0 {
		return finish(Failed, "convert: "+tail(out), fmt.Errorf("head-trainer convert: exit %d: %v", code, err))
	}

	args := []string{"--data", train, "--out", cfg.OutDir, "--preset", cfg.Preset,
		"--epochs", fmt.Sprint(cfg.Epochs), "--lr", fmt.Sprint(cfg.LR), "--seed", fmt.Sprint(cfg.Seed)}
	if cfg.PolicyHash != "" {
		args = append(args, "--policy-bundle-hash", cfg.PolicyHash)
	}
	out, code, err := runTrainer(ctx, cfg.TrainerBin, args)
	st.Output = tail(out)
	st.Metrics = filepath.Join(cfg.OutDir, "head", "metrics.json")
	switch {
	case err != nil && code < 0:
		return finish(Failed, "trainer did not run", err)
	case code == 0:
		st.Trained = n
		return finish(Promoted, "", nil)
	case code == 3:
		st.Trained = n // the data was consumed; do not retrain on it until more arrives
		return finish(Refused, "candidate did not beat the current model", nil)
	default:
		return finish(Failed, fmt.Sprintf("trainer exit %d", code), fmt.Errorf("head-trainer: exit %d", code))
	}
}

var errNoBronze = errors.New("no Bronze events yet")

// materialize normalises every Bronze file under dir and writes Gold episodes.
// It returns the episode count.
func materialize(dir, goldPath string) (int, error) {
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	sort.Strings(files)
	if len(files) == 0 {
		return 0, errNoBronze
	}
	cfg := pipeline.DefaultSilverConfig()
	var events []pipeline.SilverEvent
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return 0, err
		}
		s := bufio.NewScanner(fh)
		s.Buffer(make([]byte, 64<<10), 64<<20)
		for s.Scan() {
			if ev, ok := pipeline.NormalizeBronze(s.Bytes(), cfg); ok {
				events = append(events, ev)
			}
		}
		err = s.Err()
		fh.Close()
		if err != nil {
			return 0, err
		}
	}
	clean, _ := pipeline.DedupSilver(events)
	if len(clean) == 0 {
		return 0, errNoBronze
	}
	g := &pipeline.SplitGenerator{Splits: []pipeline.SplitSpec{{Name: "train"}}}
	eps, _, err := pipeline.Materialize(clean, g, "deterministic-v1", "v1", "supervision-v1", 768)
	if err != nil {
		return 0, err
	}
	if err := pipeline.WriteJSONL(goldPath, eps); err != nil {
		return 0, err
	}
	return len(eps), nil
}

// runTrainer returns combined output and the exit code (-1 if it could not start).
func runTrainer(ctx context.Context, bin string, args []string) (string, int, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return buf.String(), ee.ExitCode(), nil
	}
	if err != nil {
		return buf.String(), -1, err
	}
	return buf.String(), 0, nil
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		s = s[len(s)-2000:]
	}
	return s
}
