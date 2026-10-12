package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"knirvmodels/internal/cli/embedded"
	"knirvmodels/pkg/head/cycle"
)

// headCycleMu serialises HEAD cycles: a cycle trains for minutes and the pipeline
// loop would otherwise start one every batch.
var headCycleMu sync.Mutex

// headCycleConfig derives stage-5 paths from the app data dir. Env overrides:
//
//	KNIRV_HEAD_AUTOTRAIN=0        disable the automatic trigger (manual still works)
//	KNIRV_HEAD_BRONZE_DIR         Bronze JSONL directory
//	KNIRV_HEAD_PRESET/EPOCHS      model size and epochs
//	KNIRV_HEAD_MIN_NEW/MIN_GAP    new-episode threshold and minimum hours between cycles
//	KNIRV_HEAD_POLICY_HASH        policy bundle the model is trained under
func headCycleConfig(force bool) (cycle.Config, error) {
	app, err := embedded.GetAppDataDir()
	if err != nil {
		return cycle.Config{}, err
	}
	trainer, err := embedded.GetBinaryPath("head-trainer")
	if err != nil {
		return cycle.Config{}, fmt.Errorf("head-trainer not available: %w", err)
	}
	data := filepath.Join(app, "knirvmodels", "data")
	work := filepath.Join(app, "knirvmodels", "head-work")
	cfg := cycle.Config{
		BronzeDir:  envOr("KNIRV_HEAD_BRONZE_DIR", filepath.Join(app, "knirvmodels", "supervision", "bronze")),
		WorkDir:    work,
		OutDir:     data, // HEART serves <data>/head
		TrainerBin: trainer,
		Preset:     os.Getenv("KNIRV_HEAD_PRESET"),
		PolicyHash: os.Getenv("KNIRV_HEAD_POLICY_HASH"),
		SoftLabels: filepath.Join(work, "teacher.jsonl"),
		Force:      force,
		Log:        func(f string, a ...any) { fmt.Printf(f+"\n", a...) },
	}
	if n, err := strconv.Atoi(os.Getenv("KNIRV_HEAD_EPOCHS")); err == nil {
		cfg.Epochs = n
	}
	if n, err := strconv.Atoi(os.Getenv("KNIRV_HEAD_MIN_NEW")); err == nil {
		cfg.MinNew = n
	}
	if h, err := strconv.Atoi(os.Getenv("KNIRV_HEAD_MIN_GAP_HOURS")); err == nil {
		cfg.MinGap = time.Duration(h) * time.Hour
	}
	return cfg, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
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
	if os.Getenv("KNIRV_HEAD_AUTOTRAIN") == "0" {
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
	if os.Getenv("KNIRV_HEAD_AUTOTRAIN") == "0" || every <= 0 {
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
