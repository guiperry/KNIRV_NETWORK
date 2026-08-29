//go:build darwin

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func ensureDocker(ctx context.Context, run commandRunner) error {
	if !commandExists("docker") {
		if !commandExists("brew") {
			return fmt.Errorf("Docker Desktop is not installed and Homebrew is unavailable; install Docker Desktop, then rerun the installer")
		}
		// osascript's administrator-privileges clause displays Apple's native
		// authentication dialog instead of trying to emulate sudo in a GUI app.
		install := `do shell script "PATH=/opt/homebrew/bin:/usr/local/bin:$PATH brew install --cask docker" with administrator privileges`
		if err := run(ctx, "osascript", "-e", install); err != nil {
			return fmt.Errorf("install Docker Desktop (administrator dialog): %w", err)
		}
	}
	if err := run(ctx, "open", "-a", "Docker"); err != nil {
		return fmt.Errorf("launch Docker Desktop: %w", err)
	}
	return waitForDocker(ctx, run)
}

func waitForDocker(ctx context.Context, run commandRunner) error {
	deadline := time.NewTimer(2 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		if err := run(ctx, "docker", "info"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("Docker Desktop did not become ready within two minutes")
		case <-tick.C:
		}
	}
}

func commandExists(name string) bool { _, err := exec.LookPath(name); return err == nil }

func runHostCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}
