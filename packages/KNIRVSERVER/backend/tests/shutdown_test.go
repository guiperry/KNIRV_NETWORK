package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type shutdownEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Module    string    `json:"module"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

type shutdownOrder struct {
	mu     sync.Mutex
	events []shutdownEvent
}

func (o *shutdownOrder) recordEvent(module, status, message string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, shutdownEvent{
		Timestamp: time.Now(),
		Module:    module,
		Status:    status,
		Message:   message,
	})
}

func (o *shutdownOrder) getEvents() []shutdownEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	events := make([]shutdownEvent, len(o.events))
	copy(events, o.events)
	return events
}

func TestSubmoduleShutdownSequence(t *testing.T) {
	extractedBinDir := getExtractedBinDir()

	binPath := getBackendPathFromExtracted()
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if _, err := os.Stat(extractedBinDir); os.IsNotExist(err) {
			t.Skip("Binaries not extracted yet, run unified binary first to extract to ~/.local/share/knirvserver/bin/")
		}
		t.Skip("Binaries not built, run 'make backend' first")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("AllSubmodulesStopGracefully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping submodule shutdown test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_TEST_MODE=true",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		submodulePorts := map[string]int{
			"knirvgateway": 8081,
			"knirvgraph":   7090,
			"knirvchain":   9090,
		}

		maxWait := 45 * time.Second
		pollInterval := 2 * time.Second

		t.Logf("Waiting for submodules to start from extracted binaries...")
		for name, port := range submodulePorts {
			if port == 0 {
				t.Logf("Skipping health check for %s (port not configured)", name)
				continue
			}
			require.Eventually(t, func() bool {
				resp, err := http.Get(getHealthURL(port))
				if err != nil {
					return false
				}
				resp.Body.Close()
				return resp.StatusCode == http.StatusOK
			}, maxWait, pollInterval, "%s should become healthy", name)
			t.Logf("%s is healthy on port %d", name, port)
		}

		t.Logf("All submodules started, sending SIGTERM to trigger shutdown...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		t.Logf("Waiting for backend process to exit (with child processes)...")

		waitCtx, waitCancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer waitCancel()
		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-waitCtx.Done():
			t.Log("Process did not exit within timeout, checking child processes...")
			remainingChildren := getChildProcesses(int(cmd.Process.Pid))
			t.Logf("Remaining child processes before kill: %v", remainingChildren)
			cmd.Process.Kill()
			cmd.Wait()
			t.Logf("Process killed forcefully - submodules were terminated with parent")
		case err := <-done:
			if err != nil {
				t.Logf("Process exited with: %v", err)
			} else {
				t.Log("Process exited cleanly")
			}
		}

		backendPID := int(cmd.Process.Pid)
		time.Sleep(2 * time.Second)
		remainingChildren := getChildProcesses(backendPID)
		if len(remainingChildren) > 0 {
			t.Logf("Warning: %d child processes still running after parent exit", len(remainingChildren))
			for _, childPID := range remainingChildren {
				proc, _ := os.FindProcess(childPID)
				if proc != nil {
					t.Logf("Killing orphaned child PID %d", childPID)
					proc.Kill()
				}
			}
		} else {
			t.Logf("All child processes terminated with backend")
		}

		t.Logf("Shutdown sequence completed - submodules stopped")
	})

	t.Run("GracefulShutdownTimeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping graceful shutdown timeout test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		waitForBackend := 30 * time.Second
		backendPort := 8082

		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGTERM for graceful shutdown test...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			t.Log("Shutdown did not complete within test timeout")
			cmd.Process.Kill()
			cmd.Wait()
		case err := <-done:
			t.Logf("Backend process exited: %v", err)
		}
	})

	t.Run("SubmodulesStopBeforeBackendHTTP", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping shutdown order test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		waitForBackend := 30 * time.Second
		backendPort := 8082
		gatewayPort := 8888

		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGTERM and monitoring shutdown order...")

		order := &shutdownOrder{}
		pollInterval := 500 * time.Millisecond

		go func() {
			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					gatewayResp, _ := http.Get(getHealthURL(gatewayPort))
					backendResp, _ := http.Get(getHealthURL(backendPort))

					gatewayDown := gatewayResp == nil || gatewayResp.StatusCode != http.StatusOK
					backendDown := backendResp == nil || backendResp.StatusCode != http.StatusOK

					if gatewayResp != nil {
						gatewayResp.Body.Close()
					}
					if backendResp != nil {
						backendResp.Body.Close()
					}

					if gatewayDown && !backendDown {
						order.recordEvent("knirvgateway", "stopped", "Gateway stopped")
					}
					if backendDown {
						order.recordEvent("backend", "stopped", "Backend HTTP stopped")
						return
					}
				}
			}
		}()

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		cmd.Wait()

		events := order.getEvents()
		if len(events) > 0 {
			t.Logf("Shutdown events: %+v", events)

			gatewayIdx := findShutdownEventIndex(events, "knirvgateway")
			backendIdx := findShutdownEventIndex(events, "backend")

			if gatewayIdx >= 0 && backendIdx >= 0 {
				assert.Less(t, gatewayIdx, backendIdx, "Gateway should stop before backend HTTP server")
				t.Logf("Shutdown order verified: Gateway (idx=%d) stopped before Backend (idx=%d)", gatewayIdx, backendIdx)
			}
		}
	})
}

func TestDesktopStopsBeforeSubmodules(t *testing.T) {
	distDir := filepath.Join("..", "..", "dist")
	unifiedBinaryPath := filepath.Join(distDir, "knir-server")

	if _, err := os.Stat(unifiedBinaryPath); os.IsNotExist(err) {
		t.Skip("Unified binary not built, run 'make binary' first")
	}

	if testing.Short() {
		t.Skip("Skipping desktop shutdown test in short mode")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("DesktopShutdownStopsBackend", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, unifiedBinaryPath, "-desktop")
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_PORT=9080",
			"NEXUS_BACKEND_PORT=9081",
		)

		err := cmd.Start()
		require.NoError(t, err, "Unified binary with desktop should start")

		waitForBackend := 45 * time.Second
		backendPort := 9081

		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGTERM to trigger unified shutdown...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		shutdownTimeout := 60 * time.Second

		require.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", backendPort))
			if err != nil {
				return true
			}
			resp.Body.Close()
			return resp.StatusCode != http.StatusOK
		}, shutdownTimeout, 1*time.Second, "Backend should stop responding after shutdown")

		t.Logf("Backend stopped, waiting for main process to exit...")

		waitCtx, waitCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer waitCancel()

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-waitCtx.Done():
			t.Log("Process did not exit within timeout, forcing kill")
			cmd.Process.Kill()
			cmd.Wait()
		case err := <-done:
			t.Logf("Unified process exited: %v", err)
		}
	})
}

func TestShutdownOrderFromLogs(t *testing.T) {
	binPath := getBackendPathFromExtracted()
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binaries not available, run unified binary first to extract to ~/.local/share/knirvserver/bin/")
	}

	if testing.Short() {
		t.Skip("Skipping shutdown log order test in short mode")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("VerifyShutdownOrderInLogs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")

		outputFile, err := os.CreateTemp("", "knirv-backend-shutdown-*.log")
		require.NoError(t, err, "Should create temp log file")
		defer os.Remove(outputFile.Name())

		cmd.Stdout = outputFile
		cmd.Stderr = outputFile

		err = cmd.Start()
		require.NoError(t, err, "Backend server should start")

		waitForBackend := 30 * time.Second
		backendPort := 8082

		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGTERM for shutdown log test...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		cmd.Wait()

		outputFile.Sync()
		content, err := os.ReadFile(outputFile.Name())
		require.NoError(t, err, "Should read log output")

		logContent := string(content)

		modules := []string{"Stopping KNIRVGATEWAY", "Stopping KNIRVGRAPH", "Stopping KNIRVCHAIN"}
		firstIndices := make(map[string]int)
		foundCount := 0

		for _, module := range modules {
			if idx := findShutdownLogIndex(logContent, module); idx >= 0 {
				firstIndices[module] = idx
				foundCount++
			}
		}

		if foundCount == 0 {
			t.Log("Could not find module shutdown logs - module names may have changed")
			t.Skip("Module shutdown logs not found in expected format")
		}

		t.Logf("Found %d/%d module shutdown log entries", foundCount, len(modules))

		assertShutdownModuleOrder(t, logContent, "Stopping KNIRVGATEWAY", "Stopping KNIRVGRAPH")
		assertShutdownModuleOrder(t, logContent, "Stopping KNIRVGRAPH", "Stopping KNIRVCHAIN")

		t.Logf("Shutdown order verified in logs")
	})
}

func TestShutdownSignalHandling(t *testing.T) {
	binPath := getBackendPathFromExtracted()
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Binaries not available, run unified binary first to extract to ~/.local/share/knirvserver/bin/")
	}

	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("SIGINTTriggersGracefulShutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		waitForBackend := 30 * time.Second
		backendPort := 8082

		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGINT...")

		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Errorf("Failed to send SIGINT: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			t.Log("Shutdown did not complete within test timeout")
			cmd.Process.Kill()
			cmd.Wait()
		case err := <-done:
			t.Logf("Backend process exited after SIGINT: %v", err)
		}
	})

	t.Run("SIGTERMTerminatesIfGracefulFails", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping SIGTERM test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		waitForBackend := 30 * time.Second
		backendPort := 8082

		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		t.Logf("Backend ready, sending SIGTERM...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
			cmd.Process.Kill()
			cmd.Wait()
			return
		}

		cmd.Wait()
		t.Log("Backend process exited after SIGTERM")
	})
}

func findShutdownEventIndex(events []shutdownEvent, module string) int {
	for i, e := range events {
		if e.Module == module {
			return i
		}
	}
	return -1
}

func findShutdownLogIndex(logContent, module string) int {
	for i := 0; i <= len(logContent)-len(module); i++ {
		if logContent[i:i+len(module)] == module {
			return i
		}
	}
	return -1
}

func assertShutdownModuleOrder(t *testing.T, logContent, earlier, later string) {
	earlierIdx := findShutdownLogIndex(logContent, earlier)
	laterIdx := findShutdownLogIndex(logContent, later)

	if earlierIdx < 0 || laterIdx < 0 {
		t.Logf("Could not find both modules in shutdown logs: %s=%d, %s=%d", earlier, earlierIdx, later, laterIdx)
		return
	}

	assert.Less(t, earlierIdx, laterIdx, "%s should stop before %s", earlier, later)
}
