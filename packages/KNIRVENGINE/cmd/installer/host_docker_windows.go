//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func ensureDocker(ctx context.Context, run commandRunner) error {
	if !commandExists("docker.exe") && !commandExists("docker") {
		if !commandExists("winget.exe") && !commandExists("winget") {
			return fmt.Errorf("Docker Desktop is not installed and winget is unavailable; install Docker Desktop, then rerun the installer")
		}
		// Start-Process -Verb RunAs triggers Windows' standard UAC consent dialog.
		install := `Start-Process -FilePath winget -Verb RunAs -Wait -ArgumentList 'install -e --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements'`
		if err := run(ctx, "powershell.exe", "-NoProfile", "-Command", install); err != nil {
			return fmt.Errorf("install Docker Desktop (UAC dialog): %w", err)
		}
	}
	start := `Start-Process -FilePath "$Env:ProgramFiles\Docker\Docker\Docker Desktop.exe"`
	if err := run(ctx, "powershell.exe", "-NoProfile", "-Command", start); err != nil {
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
