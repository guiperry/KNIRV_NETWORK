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
	"knirvmodels/pkg/hashing/hrm/cycle"
)

// hrmCycleMu serialises HRM cycles: a cycle trains for minutes and the pipeline
// loop would otherwise start one every batch.
var hrmCycleMu sync.Mutex

// hrmCycleConfig derives stage-5 paths from the app data dir. Env overrides:
//
//	KNIRV_HRM_AUTOTRAIN=0        disable the automatic trigger (manual still works)
//	KNIRV_HRM_BRONZE_DIR         Bronze JSONL directory
//	KNIRV_HRM_PRESET/EPOCHS      model size and epochs
//	KNIRV_HRM_MIN_NEW/MIN_GAP    new-episode threshold and minimum hours between cycles
//	KNIRV_HRM_POLICY_HASH        policy bundle the model is trained under
func hrmCycleConfig(force bool) (cycle.Config, error) {
	app, err := embedded.GetAppDataDir()
	if err != nil {
		return cycle.Config{}, err
	}
	trainer, err := embedded.GetBinaryPath("hrm-trainer")
	if err != nil {
		return cycle.Config{}, fmt.Errorf("hrm-trainer not available: %w", err)
	}
	data := filepath.Join(app, "knirvmodels", "data")
	work := filepath.Join(app, "knirvmodels", "hrm-work")
	cfg := cycle.Config{
		BronzeDir:  envOr("KNIRV_HRM_BRONZE_DIR", filepath.Join(app, "knirvmodels", "supervision", "bronze")),
		WorkDir:    work,
		OutDir:     data, // HEART serves <data>/hrm
		TrainerBin: trainer,
		Preset:     os.Getenv("KNIRV_HRM_PRESET"),
		PolicyHash: os.Getenv("KNIRV_HRM_POLICY_HASH"),
		SoftLabels: filepath.Join(work, "teacher.jsonl"),
		Force:      force,
		Log:        func(f string, a ...any) { fmt.Printf(f+"\n", a...) },
	}
	if n, err := strconv.Atoi(os.Getenv("KNIRV_HRM_EPOCHS")); err == nil {
		cfg.Epochs = n
	}
	if n, err := strconv.Atoi(os.Getenv("KNIRV_HRM_MIN_NEW")); err == nil {
		cfg.MinNew = n
	}
	if h, err := strconv.Atoi(os.Getenv("KNIRV_HRM_MIN_GAP_HOURS")); err == nil {
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

// RunHRMCycle runs stage 5 once. force skips the new-data and spacing checks.
func (c *Controller) RunHRMCycle(ctx context.Context, force bool) (cycle.State, error) {
	if !hrmCycleMu.TryLock() {
		return cycle.State{Outcome: cycle.Skipped, Reason: "a cycle is already running"}, nil
	}
	defer hrmCycleMu.Unlock()
	cfg, err := hrmCycleConfig(force)
	if err != nil {
		return cycle.State{Outcome: cycle.Failed, Reason: err.Error()}, err
	}
	st, err := cycle.Run(ctx, cfg)
	if st.Outcome != cycle.Skipped || force {
		fmt.Printf("[hrm] cycle %s (episodes=%d) %s\n", st.Outcome, st.Episodes, st.Reason)
	}
	return st, err
}

// HRMStatus is the last recorded cycle.
func (c *Controller) HRMStatus() cycle.State {
	cfg, err := hrmCycleConfig(false)
	if err != nil {
		return cycle.State{Outcome: cycle.Failed, Reason: err.Error()}
	}
	return cycle.LoadState(cfg.WorkDir)
}

// maybeAutoHRMCycle is called after each complete pipeline batch. It is cheap
// when there is nothing to do and must never fail or delay the pipeline.
func (c *Controller) maybeAutoHRMCycle() {
	if os.Getenv("KNIRV_HRM_AUTOTRAIN") == "0" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[hrm] cycle panic recovered: %v\n", r)
			}
		}()
		_, _ = c.RunHRMCycle(context.Background(), false)
	}()
}

// HRMDir is the directory the daemon serves the model from: exactly where stage 5
// publishes (<app data>/knirvmodels/data/hrm). Deliberately not separately
// overridable, so the writer and the reader cannot drift apart.
func HRMDir() (string, error) {
	app, err := embedded.GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(app, "knirvmodels", "data", "hrm"), nil
}

// RunHRMScheduler checks for a cycle every `every` until ctx ends. Uploaded Bronze
// arrives independently of data-pipeline batches (which only run when an operator
// starts training), so stage 5 must not depend on them. The cycle itself decides
// whether there is enough new data; a check that finds none is silent.
func (c *Controller) RunHRMScheduler(ctx context.Context, every time.Duration) {
	if os.Getenv("KNIRV_HRM_AUTOTRAIN") == "0" || every <= 0 {
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
					fmt.Printf("[hrm] scheduler panic recovered: %v\n", r)
				}
			}()
			_, _ = c.RunHRMCycle(ctx, false)
		}()
		t.Reset(every)
	}
}
