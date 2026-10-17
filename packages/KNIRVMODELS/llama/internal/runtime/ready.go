package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	// DefaultHealthTimeout bounds how long llama-server may take to load its model.
	// Loading a multi-GB GGUF from a cold page cache on a busy CPU host takes minutes,
	// so this is deliberately generous; a process that has actually died is detected
	// immediately and does not wait for it. KNIRV_LLAMA_HEALTH_TIMEOUT overrides it.
	DefaultHealthTimeout = 15 * time.Minute
	defaultPollInterval  = 200 * time.Millisecond
	defaultProgressEvery = 15 * time.Second
)

// ErrExited reports that llama-server ended before it became healthy.
var ErrExited = errors.New("llama-server exited during startup (bad model, adapter or out of memory?)")

// HealthTimeoutFromEnv returns KNIRV_LLAMA_HEALTH_TIMEOUT when it parses as a positive
// duration, else DefaultHealthTimeout.
func HealthTimeoutFromEnv() time.Duration {
	if v := os.Getenv("KNIRV_LLAMA_HEALTH_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return DefaultHealthTimeout
}

// HealthState probes /health once. code is 0 when nothing answered. llama-server
// answers 503 while it loads the model, which is progress, not failure.
func HealthState(ctx context.Context, address string) (code int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/health", nil)
	if err != nil {
		return 0, err
	}
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// WaitOptions tunes WaitHealthy. Zero values use the defaults above.
type WaitOptions struct {
	Timeout       time.Duration
	Interval      time.Duration
	ProgressEvery time.Duration
	Logf          func(format string, args ...any)
}

// WaitHealthy blocks until /health is 2xx. It returns early with ErrExited the moment
// exited closes (a dead child never becomes healthy), with ctx.Err() if cancelled, and
// with a timeout error naming the last observed state. exited may be nil.
func WaitHealthy(ctx context.Context, address string, exited <-chan struct{}, o WaitOptions) error {
	if o.Timeout <= 0 {
		o.Timeout = DefaultHealthTimeout
	}
	if o.Interval <= 0 {
		o.Interval = defaultPollInterval
	}
	if o.ProgressEvery <= 0 {
		o.ProgressEvery = defaultProgressEvery
	}
	logf := o.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	start := time.Now()
	deadline := time.NewTimer(o.Timeout)
	defer deadline.Stop()
	poll := time.NewTicker(o.Interval)
	defer poll.Stop()
	progress := time.NewTicker(o.ProgressEvery)
	defer progress.Stop()
	state := "not listening yet"
	for {
		code, err := HealthState(ctx, address)
		switch {
		case err == nil && code >= 200 && code < 300:
			logf("llama-server healthy after %s", time.Since(start).Round(time.Second))
			return nil
		case err == nil:
			state = fmt.Sprintf("loading model (HTTP %d)", code)
		default:
			state = "not listening yet"
		}
		select {
		case <-exited:
			return fmt.Errorf("%w after %s (%s)", ErrExited, time.Since(start).Round(time.Millisecond), state)
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("llama-server did not become healthy within %s (last state: %s)", o.Timeout, state)
		case <-progress.C:
			logf("llama-server %s, %s elapsed", state, time.Since(start).Round(time.Second))
		case <-poll.C:
		}
	}
}
