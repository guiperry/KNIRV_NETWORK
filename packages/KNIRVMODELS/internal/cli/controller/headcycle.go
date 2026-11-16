package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"knirvmodels/internal/cli/embedded"
	"knirvmodels/pkg/heart/decision/cycle"
)

// headCycleMu serialises HEAD cycles: a cycle trains for minutes and the pipeline
// loop would otherwise start one every batch.
var headCycleMu sync.Mutex

// trainerName is the stage-5 trainer binary the Makefile embeds
// (build-heart-decision-trainer). legacyTrainerName is the pre-rename name; it is
// accepted only so an embed built before the rename still runs. The fallback is
// removed one release after this rename.
const (
	trainerName       = "heart-decision-trainer"
	legacyTrainerName = "head-trainer"
)

// trainerBinaryPath resolves the trainer through lookup, preferring the current name
// and falling back to the legacy one. The error reports the current name only.
func trainerBinaryPath(lookup func(string) (string, error)) (string, error) {
	p, err := lookup(trainerName)
	if err == nil {
		return p, nil
	}
	if lp, lerr := lookup(legacyTrainerName); lerr == nil {
		return lp, nil
	}
	return "", err
}

// Env prefixes for the stage-5 decision model. The current name is
// KNIRV_HEART_DECISION_<suffix>; the legacy KNIRV_HEAD_<suffix> is still read as a
// fallback. Legacy names are removed one release after the rename. If both are set,
// the new name wins.
const (
	heartDecisionEnvPrefix = "KNIRV_HEART_DECISION_"
	legacyHeadEnvPrefix    = "KNIRV_HEAD_"
)

var (
	legacyEnvLogMu  sync.Mutex
	legacyEnvLogged = map[string]bool{}
)

// HeartDecisionEnv returns the value of KNIRV_HEART_DECISION_<suffix>. An empty value
// counts as unset. If that is unset, it falls back to KNIRV_HEAD_<suffix> and logs a
// deprecation notice once per legacy name. It returns "" when neither is set.
func HeartDecisionEnv(suffix string) string {
	if v := os.Getenv(heartDecisionEnvPrefix + suffix); v != "" {
		return v
	}
	legacy := legacyHeadEnvPrefix + suffix
	v := os.Getenv(legacy)
	if v == "" {
		return ""
	}
	legacyEnvLogMu.Lock()
	first := !legacyEnvLogged[legacy]
	legacyEnvLogged[legacy] = true
	legacyEnvLogMu.Unlock()
	if first {
		// Legacy names are removed one release after the rename.
		log.Printf("DEPRECATED: env %s is read as a fallback; rename it to %s (the legacy name is removed in a later release)", legacy, heartDecisionEnvPrefix+suffix)
	}
	return v
}

// headCycleConfig derives stage-5 paths from the app data dir. Env overrides, each
// read through HeartDecisionEnv (KNIRV_HEART_DECISION_<suffix>, legacy KNIRV_HEAD_<suffix>):
//
//	AUTOTRAIN=0        disable the automatic trigger (manual still works)
//	BRONZE_DIR         Bronze JSONL directory
//	PRESET/EPOCHS      model size and epochs
//	MIN_NEW/MIN_GAP_HOURS  new-episode threshold and minimum hours between cycles
//	POLICY_HASH        policy bundle the model is trained under
func headCycleConfig(force bool) (cycle.Config, error) {
	app, err := embedded.GetAppDataDir()
	if err != nil {
		return cycle.Config{}, err
	}
	trainer, err := trainerBinaryPath(embedded.GetBinaryPath)
	if err != nil {
		return cycle.Config{}, fmt.Errorf("heart-decision-trainer not available: %w", err)
	}
	data := filepath.Join(app, "knirvmodels", "data")
	work := filepath.Join(app, "knirvmodels", "head-work")
	bronze := HeartDecisionEnv("BRONZE_DIR")
	if bronze == "" {
		bronze = filepath.Join(app, "knirvmodels", "supervision", "bronze")
	}
	cfg := cycle.Config{
		BronzeDir:  bronze,
		WorkDir:    work,
		OutDir:     data, // HEART serves <data>/head
		TrainerBin: trainer,
		Preset:     HeartDecisionEnv("PRESET"),
		PolicyHash: HeartDecisionEnv("POLICY_HASH"),
		SoftLabels: filepath.Join(work, "teacher.jsonl"),
		Force:      force,
		Log:        func(f string, a ...any) { fmt.Printf(f+"\n", a...) },
	}
	if n, err := strconv.Atoi(HeartDecisionEnv("EPOCHS")); err == nil {
		cfg.Epochs = n
	}
	if n, err := strconv.Atoi(HeartDecisionEnv("MIN_NEW")); err == nil {
		cfg.MinNew = n
	}
	if h, err := strconv.Atoi(HeartDecisionEnv("MIN_GAP_HOURS")); err == nil {
		cfg.MinGap = time.Duration(h) * time.Hour
	}
	return cfg, nil
}

// RunHEADCycle runs stage 5 once. force skips the new-data and spacing checks.
func (c *Controller) RunHEADCycle(ctx context.Context, force bool) (cycle.State, error) {
	if !headCycleMu.TryLock() {
		return cycle.State{Outcome: cycle.Skipped, Reason: "a cycle is already running"}, nil
	}
	defer headCycleMu.Unlock()
	cfg, err := headCycleConfig(force)
	if err != nil {
		return cycle.State{Outcome: cycle.Failed, Reason: err.Error()}, err
	}
	st, err := cycle.Run(ctx, cfg)
	if st.Outcome != cycle.Skipped || force {
		fmt.Printf("[head] cycle %s (episodes=%d) %s\n", st.Outcome, st.Episodes, st.Reason)
	}
	return st, err
}

// HEADStatus is the last recorded cycle.
func (c *Controller) HEADStatus() cycle.State {
	cfg, err := headCycleConfig(false)
	if err != nil {
		return cycle.State{Outcome: cycle.Failed, Reason: err.Error()}
	}
	return cycle.LoadState(cfg.WorkDir)
}

// maybeAutoHEADCycle is called after each complete pipeline batch. It is cheap
// when there is nothing to do and must never fail or delay the pipeline.
func (c *Controller) maybeAutoHEADCycle() {
	if HeartDecisionEnv("AUTOTRAIN") == "0" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[head] cycle panic recovered: %v\n", r)
			}
		}()
		_, _ = c.RunHEADCycle(context.Background(), false)
	}()
}

// HEADDir is the directory the daemon serves the model from: exactly where stage 5
// publishes (<app data>/knirvmodels/data/head). Deliberately not separately
// overridable, so the writer and the reader cannot drift apart.
func HEADDir() (string, error) {
	app, err := embedded.GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(app, "knirvmodels", "data", "head"), nil
}

// RunHEADScheduler checks for a cycle every `every` until ctx ends. Uploaded Bronze
// arrives independently of data-pipeline batches (which only run when an operator
// starts training), so stage 5 must not depend on them. The cycle itself decides
// whether there is enough new data; a check that finds none is silent.
func (c *Controller) RunHEADScheduler(ctx context.Context, every time.Duration) {
	if HeartDecisionEnv("AUTOTRAIN") == "0" || every <= 0 {
		return
	}
	t := time.NewTimer(time.Minute) // first look shortly after start
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[head] scheduler panic recovered: %v\n", r)
				}
			}()
			_, _ = c.RunHEADCycle(ctx, false)
		}()
		t.Reset(every)
	}
}
