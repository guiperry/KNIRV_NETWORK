package validationchain

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var (
	instance *ValidationChain
	once     sync.Once
)

// ValidationChain manages the embedded Validation Chain binary as a
// subprocess, bound to a Unix domain socket. This is the sole owner of its
// lifecycle — backend_server (in the separate KNIRV_CORP repo) only holds an
// HTTP client dialing the same socket; it does not spawn its own copy and
// nothing exposes a TCP port for it. KNIRVGATEWAY/backend_server proxy
// requests to it over that socket (see api_router.go's chain proxy routes).
type ValidationChain struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	cmd        *exec.Cmd
	socketPath string
	client     *http.Client
	restartMu  sync.Mutex
	running    bool
}

// Get returns the singleton ValidationChain instance.
func Get() *ValidationChain {
	once.Do(func() {
		instance = &ValidationChain{}
	})
	return instance
}

// unixHTTPClient builds an http.Client that dials a Unix domain socket
// regardless of the URL host given to it — callers use the "http://unix"
// base URL by convention (matching pkg/knirvoracle/client.go's pattern).
func unixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// Start extracts the embedded binary (if not already staged) and launches it
// as a subprocess bound to socketPath, waiting for its /health endpoint
// before returning.
func (vc *ValidationChain) Start(ctx context.Context, socketPath string) error {
	vc.restartMu.Lock()
	defer vc.restartMu.Unlock()

	if vc.running {
		return fmt.Errorf("validation chain already running")
	}

	binPath, err := ExtractEmbeddedBinary("")
	if err != nil {
		return fmt.Errorf("failed to stage validation chain binary: %w", err)
	}

	vc.ctx, vc.cancelFunc = context.WithCancel(ctx)
	vc.socketPath = socketPath
	vc.client = unixHTTPClient(socketPath, 5*time.Second)

	if err := os.MkdirAll(dirOf(socketPath), 0o700); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	_ = os.Remove(socketPath) // clear a stale socket file from a previous run

	vc.cmd = exec.CommandContext(vc.ctx, binPath)
	vc.cmd.Env = append(os.Environ(),
		fmt.Sprintf("VALIDATION_CHAIN_SOCKET_PATH=%s", socketPath),
		"CHAIN_ID=validation-testnet",
	)
	vc.cmd.Stdout = os.Stdout
	vc.cmd.Stderr = os.Stderr
	vc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := vc.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start validation chain: %w", err)
	}

	if err := vc.waitForHealth(vc.ctx, 30*time.Second); err != nil {
		_ = syscall.Kill(-vc.cmd.Process.Pid, syscall.SIGKILL)
		return err
	}

	vc.running = true
	go vc.monitorProcess(binPath)

	return nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func (vc *ValidationChain) waitForHealth(ctx context.Context, timeout time.Duration) error {
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for validation chain health check")
		case <-ticker.C:
			resp, err := vc.client.Get("http://unix/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

func (vc *ValidationChain) monitorProcess(binPath string) {
	for {
		_ = vc.cmd.Wait()
		vc.restartMu.Lock()

		if !vc.running {
			vc.restartMu.Unlock()
			return
		}

		select {
		case <-vc.ctx.Done():
			vc.running = false
			vc.restartMu.Unlock()
			return
		default:
		}

		time.Sleep(2 * time.Second)

		_ = os.Remove(vc.socketPath)
		vc.cmd = exec.CommandContext(vc.ctx, binPath)
		vc.cmd.Env = append(os.Environ(),
			fmt.Sprintf("VALIDATION_CHAIN_SOCKET_PATH=%s", vc.socketPath),
			"CHAIN_ID=validation-testnet",
		)
		vc.cmd.Stdout = os.Stdout
		vc.cmd.Stderr = os.Stderr
		vc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := vc.cmd.Start(); err != nil {
			vc.restartMu.Unlock()
			continue
		}

		vc.restartMu.Unlock()
	}
}

// Stop gracefully stops the validation chain subprocess.
func (vc *ValidationChain) Stop() error {
	vc.restartMu.Lock()
	defer vc.restartMu.Unlock()

	if !vc.running {
		return nil
	}

	vc.running = false
	vc.cancelFunc()

	if vc.cmd != nil && vc.cmd.Process != nil {
		syscall.Kill(-vc.cmd.Process.Pid, syscall.SIGTERM)
	}
	_ = os.Remove(vc.socketPath)

	return nil
}

// HealthCheck verifies the validation chain subprocess is running.
func (vc *ValidationChain) HealthCheck() error {
	vc.restartMu.Lock()
	defer vc.restartMu.Unlock()

	if !vc.running {
		return fmt.Errorf("validation chain not running")
	}
	resp, err := vc.client.Get("http://unix/health")
	if err != nil {
		return fmt.Errorf("validation chain health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("validation chain health check returned %d", resp.StatusCode)
	}
	return nil
}

// SocketPath returns the Unix domain socket path the validation chain
// subprocess listens on, valid once Start has succeeded.
func (vc *ValidationChain) SocketPath() string {
	return vc.socketPath
}
