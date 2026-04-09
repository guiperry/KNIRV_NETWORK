package runner

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type SSHRunner struct{}

func (r *SSHRunner) Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error) {
	key, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: known_hosts in prod
		Timeout:         10 * time.Second,
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port)), sshCfg)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, err
	}

	if err := session.Start(cfg.RemoteCmd); err != nil {
		session.Close()
		client.Close()
		return nil, err
	}

	var (
		lifecycleMu sync.Mutex
		closed      bool
		closeOnce   sync.Once
	)

	closeResources := func() {
		closeOnce.Do(func() {
			_ = session.Close()
			_ = client.Close()
			closed = true
		})
	}

	return &AgentProcess{
		ID:        uuid.NewString(),
		AgentName: cfg.Name,
		Type:      AgentTypeSSH,
		StartedAt: time.Now(),
		Stdout:    stdoutPipe,
		Stderr:    stderrPipe,
		StopFn: func(ctx context.Context) error {
			lifecycleMu.Lock()
			defer lifecycleMu.Unlock()
			if closed {
				return nil
			}
			return session.Signal(ssh.SIGTERM)
		},
		WaitFn: func() (int, error) {
			err := session.Wait()
			lifecycleMu.Lock()
			closeResources()
			lifecycleMu.Unlock()
			if err != nil {
				if exitErr, ok := err.(*ssh.ExitError); ok {
					return exitErr.ExitStatus(), nil
				}
				return -1, err
			}
			return 0, nil
		},
	}, nil
}
