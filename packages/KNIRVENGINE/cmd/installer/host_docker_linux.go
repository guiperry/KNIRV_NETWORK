//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func ensureDocker(ctx context.Context, run commandRunner) error {
	if !commandExists("docker") {
		var installArgs []string
		switch {
		case commandExists("apt-get"):
			if err := run(ctx, "apt-get", "update"); err != nil {
				return fmt.Errorf("update apt metadata: %w", err)
			}
			installArgs = []string{"apt-get", "install", "-y", "docker.io"}
		case commandExists("dnf"):
			installArgs = []string{"dnf", "install", "-y", "docker"}
		case commandExists("yum"):
			installArgs = []string{"yum", "install", "-y", "docker"}
		case commandExists("pacman"):
			installArgs = []string{"pacman", "-Sy", "--noconfirm", "docker"}
		case commandExists("zypper"):
			installArgs = []string{"zypper", "--non-interactive", "install", "docker"}
		case commandExists("apk"):
			installArgs = []string{"apk", "add", "docker"}
		default:
			return fmt.Errorf("Docker is not installed and no supported package manager was found")
		}
		if err := run(ctx, installArgs[0], installArgs[1:]...); err != nil {
			return fmt.Errorf("install Docker: %w", err)
		}
	}
	if err := run(ctx, "docker", "info"); err == nil {
		return nil
	}
	if commandExists("systemctl") {
		if err := run(ctx, "systemctl", "enable", "--now", "docker"); err != nil {
			return fmt.Errorf("start Docker service: %w", err)
		}
	} else if commandExists("service") {
		if err := run(ctx, "service", "docker", "start"); err != nil {
			return fmt.Errorf("start Docker service: %w", err)
		}
	} else {
		return fmt.Errorf("Docker is installed but its daemon is unavailable and no service manager was found")
	}
	if err := run(ctx, "docker", "info"); err != nil {
		return fmt.Errorf("Docker daemon is unavailable after starting service: %w", err)
	}
	return nil
}

func commandExists(name string) bool { _, err := exec.LookPath(name); return err == nil }

func runHostCommand(ctx context.Context, name string, args ...string) error {
	if os.Geteuid() != 0 {
		args, name = append([]string{"--", name}, args...), "sudo"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}
