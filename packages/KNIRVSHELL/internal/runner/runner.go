package runner

import (
	"context"
	"errors"
	"io"
	"time"
)

// AgentType discriminates the execution backend.
type AgentType string

const (
	AgentTypeLocal  AgentType = "local"
	AgentTypeDocker AgentType = "docker"
	AgentTypeSSH    AgentType = "ssh"
)

// AgentProcess is the live handle to a running agent.
// It is created by a Runner and consumed by the Watchtower.
type AgentProcess struct {
	ID        string // Unique run ID (UUID v4)
	AgentName string // Matches registry entry
	Type      AgentType
	StartedAt time.Time

	Stdout io.Reader // Combined log stream for Watchtower consumption
	Stderr io.Reader // Separate stderr for error-priority scanning

	// StopFn sends SIGTERM (or Docker stop / SSH session close).
	// Implementations must make this idempotent.
	StopFn func(ctx context.Context) error

	// WaitFn blocks until the agent exits and returns the exit code.
	WaitFn func() (exitCode int, err error)
}

// Runner abstracts the execution environment.
// Implementations: LocalRunner, DockerRunner, SSHRunner.
type Runner interface {
	// Start launches the agent and returns a live AgentProcess.
	// The caller (Watchtower) owns the lifecycle from this point.
	Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error)
}

// AgentConfig is the per-agent launch descriptor, populated from registry.db
// or agents.yaml at runtime.
type AgentConfig struct {
	Name string    `yaml:"name"`
	Type AgentType `yaml:"type"`
	// Local
	Path string   `yaml:"path,omitempty"`
	Args []string `yaml:"args,omitempty"`
	Env  []string `yaml:"env,omitempty"`
	// Docker
	Image  string            `yaml:"image,omitempty"`
	Mounts map[string]string `yaml:"mounts,omitempty"` // host:container
	// SSH
	Host      string `yaml:"host,omitempty"`
	Port      int    `yaml:"port,omitempty"`
	User      string `yaml:"user,omitempty"`
	KeyPath   string `yaml:"key_path,omitempty"`
	RemoteCmd string `yaml:"remote_cmd,omitempty"`
}

// ForType returns the appropriate Runner implementation for the given AgentType.
func ForType(typ AgentType) (Runner, error) {
	switch typ {
	case AgentTypeLocal:
		return &LocalRunner{}, nil
	case AgentTypeDocker:
		return NewDockerRunner()
	case AgentTypeSSH:
		return &SSHRunner{}, nil
	default:
		return nil, errors.New("unknown agent type: " + string(typ))
	}
}
