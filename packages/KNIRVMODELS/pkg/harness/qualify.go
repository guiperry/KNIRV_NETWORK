package harness

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrHostTooWeak means this machine does not meet the minimum requirements for
// Colibri, so llama.cpp stays the model backend and Colibri is neither downloaded nor
// started (or, when only the measured speed is too low, stopped and not retried).
var ErrHostTooWeak = errors.New("colibri skipped: this host does not meet the minimum requirements")

// DefaultMinTokensPerSec is the slowest generation speed at which Colibri is worth
// running: its jobs are batch work (synthetic data, gold-set suggestions, retrieval
// judging), and below about one token a second even those take days.
const DefaultMinTokensPerSec = 1.0

// cpuInfo is what the host gate needs to know about the CPU.
type cpuInfo struct {
	Model string
	AVX2  bool
	// Known is false when the CPU's features could not be read, in which case the
	// AVX2 requirement is not enforced (the measured-speed check still applies).
	Known bool
}

// readCPUInfo reads /proc/cpuinfo (or the file at path).
func readCPUInfo(path string) cpuInfo {
	var ci cpuInfo
	f, err := os.Open(path)
	if err != nil {
		return ci
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "model name":
			if ci.Model == "" {
				ci.Model = strings.TrimSpace(v)
			}
		case "flags":
			ci.Known = true
			for _, fl := range strings.Fields(v) {
				if fl == "avx2" {
					ci.AVX2 = true
				}
			}
			return ci
		}
	}
	return ci
}

// hostRequirement checks the static requirement: an x86-64 CPU must have AVX2.
// Colibri's engine is built for the host CPU and its expert streaming assumes it; on
// older CPUs (no AVX) it generates a token every few seconds at best.
func (c *Colibri) hostRequirement() error {
	if c.cfg.Force || runtime.GOARCH != "amd64" {
		return nil
	}
	ci := readCPUInfo(c.cfg.CPUInfoPath)
	if !ci.Known || ci.AVX2 {
		return nil
	}
	return permanentError{fmt.Errorf("%w: CPU %q has no AVX2; llama.cpp serves instead (KNIRV_COLIBRI=force overrides the check)",
		ErrHostTooWeak, ci.Model)}
}

// qualification is the recorded result of the speed check.
type qualification struct {
	Fingerprint  string  `json:"fingerprint"`
	OK           bool    `json:"ok"`
	TokensPerSec float64 `json:"tokens_per_sec"`
	Min          float64 `json:"min_tokens_per_sec"`
	Note         string  `json:"note,omitempty"`
	MeasuredAt   string  `json:"measured_at"`
}

func (c *Colibri) qualifyPath() string { return filepath.Join(c.stateDir(), "knirv-qualify.json") }

// fingerprint identifies the machine, release and model a measurement belongs to; a
// different CPU, Colibri version or model measures again.
func (c *Colibri) fingerprint(model string) string {
	return strings.Join([]string{readCPUInfo(c.cfg.CPUInfoPath).Model, runtime.GOARCH, c.cfg.Version, model}, "|")
}

func (c *Colibri) minTPS() float64 {
	switch {
	case c.cfg.Force || c.cfg.MinTokensPerSec < 0:
		return 0
	case c.cfg.MinTokensPerSec == 0:
		return DefaultMinTokensPerSec
	}
	return c.cfg.MinTokensPerSec
}

func (c *Colibri) loadQualification(model string) (qualification, bool) {
	b, err := os.ReadFile(c.qualifyPath())
	if err != nil {
		return qualification{}, false
	}
	var q qualification
	if json.Unmarshal(b, &q) != nil || q.Fingerprint != c.fingerprint(model) || q.Min != c.minTPS() {
		return qualification{}, false
	}
	return q, true
}

func (c *Colibri) saveQualification(q qualification) {
	if err := os.MkdirAll(c.stateDir(), 0o755); err != nil {
		return
	}
	b, _ := json.Marshal(q)
	tmp := c.qualifyPath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, c.qualifyPath())
	}
}

func (q qualification) failure() error {
	return permanentError{fmt.Errorf("%w: measured %.2f tokens/s, below the %.2f tokens/s minimum; llama.cpp serves instead (KNIRV_COLIBRI=force overrides, KNIRV_COLIBRI_MIN_TPS changes the minimum)",
		ErrHostTooWeak, q.TokensPerSec, q.Min)}
}

// knownTooSlow is checked before a recorded-too-slow Colibri is started again.
func (c *Colibri) knownTooSlow(model string) error {
	if q, ok := c.loadQualification(model); ok && !q.OK {
		return q.failure()
	}
	return nil
}

// Qualify measures Colibri's generation speed on this host once the model is loaded
// and records the result. A speed below the minimum stops Colibri and returns a
// permanent error, so it is not started again until the machine, release or model
// changes. A recorded pass is not measured again.
func (c *Colibri) Qualify(ctx context.Context) error {
	min := c.minTPS()
	if min == 0 {
		return nil
	}
	model := c.ChosenModel()
	if q, ok := c.loadQualification(model); ok {
		if !q.OK {
			c.Stop(context.Background())
			return q.failure()
		}
		return nil
	}
	c.logf("measuring colibri's generation speed on this host (minimum %.2f tokens/s)", min)
	tps, err := c.measureSpeed(ctx)
	q := qualification{Fingerprint: c.fingerprint(model), TokensPerSec: tps, Min: min, MeasuredAt: time.Now().UTC().Format(time.RFC3339)}
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// A request that does not finish in time is the measurement: it is too slow.
		q.Note = err.Error()
	}
	q.OK = err == nil && tps >= min
	c.saveQualification(q)
	if !q.OK {
		c.logf("colibri is too slow on this host (%.2f tokens/s%s); stopping it", tps, noteSuffix(q.Note))
		c.Stop(context.Background())
		return q.failure()
	}
	c.logf("colibri generates %.2f tokens/s on this host: qualified", tps)
	return nil
}

func noteSuffix(n string) string {
	if n == "" {
		return ""
	}
	return ", " + n
}

// benchTokens is the length of the timed request.
const benchTokens = 16

// measureSpeed warms the model up (the first request reads the expert cache from
// disk) and then times a short generation: completion tokens over wall time,
// including the prompt, so it is the speed a real request sees.
func (c *Colibri) measureSpeed(ctx context.Context) (float64, error) {
	warm := c.cfg.WarmupTimeout
	if warm <= 0 {
		warm = 10 * time.Minute
	}
	if _, _, err := c.completion(ctx, 4, warm); err != nil {
		return 0, fmt.Errorf("warm-up request: %w", err)
	}
	timed := 2 * time.Duration(float64(benchTokens)/c.minTPS()*float64(time.Second))
	if timed < 2*time.Minute {
		timed = 2 * time.Minute
	}
	n, took, err := c.completion(ctx, benchTokens, timed)
	if err != nil {
		return 0, err
	}
	if took <= 0 || n == 0 {
		return 0, errors.New("the benchmark request produced no tokens")
	}
	return float64(n) / took.Seconds(), nil
}

// completion sends one short chat request straight to Colibri and returns the number
// of completion tokens and how long it took.
func (c *Colibri) completion(ctx context.Context, maxTokens int, timeout time.Duration) (int, time.Duration, error) {
	base, model := c.BaseURL(), c.ModelID()
	if base == "" {
		return 0, 0, errors.New("colibri is not running")
	}
	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": "Count from one to twenty in words."}},
		"max_tokens":  maxTokens,
		"temperature": 0,
	})
	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	took := time.Since(start)
	if resp.StatusCode != http.StatusOK {
		return 0, took, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var doc struct {
		Usage struct {
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, took, fmt.Errorf("parse response: %w", err)
	}
	return doc.Usage.Completion, took, nil
}

// Qualification describes the recorded speed check, for /harness/status.
func (c *Colibri) Qualification() map[string]any {
	doc := map[string]any{"min_tokens_per_sec": c.minTPS(), "forced": c.cfg.Force}
	if err := c.hostRequirement(); err != nil {
		doc["qualified"], doc["reason"] = false, err.Error()
		return doc
	}
	if q, ok := c.loadQualification(c.ChosenModel()); ok {
		doc["qualified"], doc["tokens_per_sec"] = q.OK, q.TokensPerSec
		if !q.OK {
			doc["reason"] = q.failure().Error()
		}
	}
	return doc
}

// parseMinTPS reads KNIRV_COLIBRI_MIN_TPS: a number, where 0 turns the speed check off.
func parseMinTPS(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	if f == 0 {
		return -1, true // explicit 0: never measure
	}
	return f, true
}
