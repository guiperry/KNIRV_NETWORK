package runner

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type LocalRunner struct{}

func (r *LocalRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	cmd := exec.CommandContext(ctx, cfg.Path, cfg.Args...)
	cmd.Env = append(cmd.Environ(), cfg.Env...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	proc := &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeLocal,
		StartedAt: time.Now(),
		Stdout:    stdoutPipe,
		Stderr:    stderrPipe,
		StopFn: func(ctx context.Context) error {
			if cmd.Process == nil {
				return nil
			}
			return cmd.Process.Signal(syscall.SIGTERM)
		},
		WaitFn: func() (int, error) {
			err := cmd.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return exitErr.ExitCode(), nil
				}
				return -1, err
			}
			return 0, nil
		},
	}
	return proc, nil
}
