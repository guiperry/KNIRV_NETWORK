// Package runtime manages the child llama-server process.
package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"
)

type Server struct{ Command *exec.Cmd }

func Healthy(ctx context.Context, address string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/health", nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func Start(path, model, address string) (*Server, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid llama address %q: %w", address, err)
	}
	cmd := exec.Command(path, "-m", model, "--host", "127.0.0.1", "--port", port)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	return &Server{Command: cmd}, nil
}

func (s *Server) Stop() {
	if s == nil || s.Command == nil || s.Command.Process == nil {
		return
	}
	_ = s.Command.Process.Kill()
	_ = s.Command.Wait()
}
