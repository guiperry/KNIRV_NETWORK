package knirvsearch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// processChild is the production Child. It launches a child process only when a
// Binary is provided (it never touches a system service manager) and otherwise
// probes an already-running URL.
type processChild struct {
	cfg        Config
	cmd        *exec.Cmd
	httpClient *http.Client
}

// defaultChildFactory is the production ChildFactory used by New unless the
// caller overrides Manager.ChildFactory (the test seam).
func defaultChildFactory(ctx context.Context, cfg Config) (Child, error) {
	return &processChild{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

// Start launches the child process when a Binary is configured. When only a URL
// is given (no binary) the backend is assumed to already be running and Start is
// a no-op success.
func (c *processChild) Start(ctx context.Context) error {
	if c.cfg.Binary == "" {
		if c.cfg.URL == "" {
			return fmt.Errorf("knirvsearch: no binary and no URL; cannot start")
		}
		return nil
	}

	// Launch ONLY a child process the deployer explicitly provided. We never
	// exec a system service (systemd/docker/etc.) here.
	cmd := exec.CommandContext(ctx, c.cfg.Binary, c.cfg.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("knirvsearch: start child %q: %w", c.cfg.Binary, err)
	}
	c.cmd = cmd
	return nil
}

// Healthy reports backend health. With a launched binary it first confirms the
// process is alive, then (when a URL is known) probes it over HTTP. With only a
// URL it probes the URL.
func (c *processChild) Healthy(ctx context.Context) (bool, error) {
	if c.cmd != nil && c.cmd.Process != nil {
		// Signal 0 is the portable "is the process alive?" check.
		if err := c.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			return false, fmt.Errorf("child process not running: %w", err)
		}
	}

	if c.cfg.URL == "" {
		// Binary with no URL: process-alive is the health signal.
		if c.cmd != nil {
			return true, nil
		}
		return false, fmt.Errorf("no URL to probe and no running child")
	}

	target := c.cfg.URL
	if c.cfg.Health != "" {
		target = joinURL(c.cfg.URL, c.cfg.Health)
	}
	// A non-default probe method on a plain GET-ish endpoint is unusual; the
	// simple reachability check covers the common SearXNG /healthz case.
	return probeURL(ctx, c.httpClient, target)
}

// Stop terminates the launched child, if any.
func (c *processChild) Stop() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	if err := c.cmd.Process.Kill(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("knirvsearch: kill child: %w", err)
	}
	_ = c.cmd.Wait()
	return nil
}

// joinURL joins a base URL and a path, tolerating missing slashes. It also
// accepts unix:// socket targets by leaving them untouched.
func joinURL(base, p string) string {
	if strings.HasPrefix(base, "unix://") {
		return base
	}
	if p == "" {
		return base
	}
	if strings.HasSuffix(base, "/") || strings.HasPrefix(p, "/") {
		return base + p
	}
	return base + "/" + p
}

// probeURL is a small helper used by callers that only want to check reachability
// of an already-running backend (e.g. for diagnostics).
func probeURL(ctx context.Context, client *http.Client, target string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500, nil
}
