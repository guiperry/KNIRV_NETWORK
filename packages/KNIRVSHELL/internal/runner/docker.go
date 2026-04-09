package runner

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
)

type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRunner{cli: cli}, nil
}

func (r *DockerRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	out, err := r.cli.ImagePull(pullCtx, cfg.Image, image.PullOptions{})
	if err != nil {
		return nil, err
	}
	io.Copy(io.Discard, out)
	out.Close()

	var binds []string
	for host, cont := range cfg.Mounts {
		binds = append(binds, host+":"+cont)
	}

	resp, err := r.cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.Image,
			Env:   cfg.Env,
		},
		&container.HostConfig{Binds: binds},
		nil, nil, "",
	)
	if err != nil {
		return nil, err
	}

	if err := r.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	attach, err := r.cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		return nil, err
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		stdcopy.StdCopy(stdoutW, stderrW, attach.Reader)
	}()

	containerID := resp.ID
	attachClose := attach.Close

	return &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeDocker,
		StartedAt: time.Now(),
		Stdout:    stdoutR,
		Stderr:    stderrR,
		StopFn: func(ctx context.Context) error {
			timeout := 10
			err := r.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
			attachClose()
			wg.Wait()
			return err
		},
		WaitFn: func() (int, error) {
			waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer waitCancel()
			statusCh, errCh := r.cli.ContainerWait(waitCtx, containerID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				return -1, err
			case status := <-statusCh:
				return int(status.StatusCode), nil
			}
		},
	}, nil
}
