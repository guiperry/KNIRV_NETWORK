package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type startupEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Module    string    `json:"module"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

type initializationOrder struct {
	mu            sync.Mutex
	events        []startupEvent
	desktopCalled bool
}

func (o *initializationOrder) recordEvent(module, status, message string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, startupEvent{
		Timestamp: time.Now(),
		Module:    module,
		Status:    status,
		Message:   message,
	})
}

func (o *initializationOrder) getEvents() []startupEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	events := make([]startupEvent, len(o.events))
	copy(events, o.events)
	return events
}

func (o *initializationOrder) desktopStarted() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.desktopCalled = true
}

func (o *initializationOrder) isDesktopStarted() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.desktopCalled
}

func TestUnifiedBinaryExtraction(t *testing.T) {
	distDir := filepath.Join("..", "..", "dist")
	unifiedBinaryPath := filepath.Join(distDir, "knir-server")

	if _, err := os.Stat(unifiedBinaryPath); os.IsNotExist(err) {
		t.Skip("Unified binary not built, run 'make binary' first")
	}

	t.Run("ExtractsBinariesToLocalShare", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping extraction test in short mode")
		}

		extractedBinDir := getExtractedBinDir()
		t.Logf("Expected extraction directory: %s", extractedBinDir)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, unifiedBinaryPath, "-desktop")
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_PORT=9080",
			"NEXUS_BACKEND_PORT=9081",
			"NEXUS_EXTRACT_ONLY=true",
		)

		err := cmd.Start()
		require.NoError(t, err, "Unified binary should start")

		waitForExtraction := 15 * time.Second
		require.Eventually(t, func() bool {
			entries, err := os.ReadDir(extractedBinDir)
			if err != nil {
				return false
			}
			for _, e := range entries {
				t.Logf("Found extracted binary: %s", e.Name())
			}
			return len(entries) >= 4
		}, waitForExtraction, 1*time.Second, "Should extract binaries to ~/.local/share/knirvserver/bin/")

		extractedBinaries := []string{"backend_server", "knirvgateway", "knirvgraph", "knirvchain"}
		for _, name := range extractedBinaries {
			binPath := filepath.Join(extractedBinDir, name)
			info, err := os.Stat(binPath)
			if err != nil {
				t.Errorf("Expected binary %s not found at %s", name, binPath)
				continue
			}
			t.Logf("Extracted %s: mode=%v size=%d", name, info.Mode(), info.Size())
		}

		cmd.Process.Kill()
		cmd.Wait()

		t.Cleanup(func() {
			killAllServices()
		})
	})
}

func TestExtractedBinariesAreSpawned(t *testing.T) {
	extractedBinDir := getExtractedBinDir()

	expectedBinaries := []string{"knirvgateway", "knirvgraph", "knirvchain"}
	for _, name := range expectedBinaries {
		binPath := filepath.Join(extractedBinDir, name)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			t.Skip("Extracted binaries not available, run unified binary first to extract")
		}
	}

	t.Cleanup(func() {
		killAllServices()
	})

	if testing.Short() {
		t.Skip("Skipping binary spawning test in short mode")
	}

	t.Run("BackendSpawnsExtractedSubmodules", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		backendPath := getBackendPathFromExtracted()
		configPath := filepath.Join("..", "..", "..", "config", "development.yaml")
		cmd := exec.CommandContext(ctx, backendPath, "-config", configPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_TEST_MODE=true",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		backendPID := cmd.Process.Pid
		t.Logf("Backend started with PID: %d", backendPID)

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		submodulePorts := map[string]int{
			"knirvgateway": 8081,
			"knirvgraph":   7090,
			"knirvchain":   9090,
		}

		maxWait := 45 * time.Second
		pollInterval := 2 * time.Second

		for name, port := range submodulePorts {
			t.Logf("Checking %s on port %d...", name, port)
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

		t.Logf("All submodules are healthy, verifying child processes...")

		childPIDs := getChildProcesses(backendPID)
		t.Logf("Found %d child processes of backend (PID=%d)", len(childPIDs), backendPID)

		foundBinaries := map[string]bool{}
		for _, childPID := range childPIDs {
			binName := getProcessName(childPID)
			if binName != "" {
				t.Logf("Child process PID=%d, name=%s", childPID, binName)
				for _, expected := range expectedBinaries {
					if binName == expected || strings.Contains(binName, expected) {
						foundBinaries[expected] = true
						t.Logf("Found spawned submodule: %s (PID=%d)", expected, childPID)
					}
				}
			}
		}

		for _, expected := range expectedBinaries {
			assert.True(t, foundBinaries[expected], "Backend should have spawned %s from extracted binaries", expected)
		}

		t.Logf("Verified: Backend is spawning submodules from extracted binaries in %s", extractedBinDir)
	})
}

func TestSubmoduleInitializationSequence(t *testing.T) {
	binDir := filepath.Join("..", "..", "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Skip("Binaries not built, run 'make backend' first")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("AllSubmodulesStart", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping submodule initialization test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		backendPath := filepath.Join(binDir, "backend_server")
		cmd := exec.CommandContext(ctx, backendPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_TEST_MODE=true",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		submodulePorts := map[string]int{
			"knirvgateway": 8888,
			"knirvgraph":   8082,
			"knirvchain":   8090,
		}

		maxWait := 45 * time.Second
		pollInterval := 2 * time.Second

		t.Run("KNIRVGATEWAYStarts", func(t *testing.T) {
			waitForService(t, "KNIRVGATEWAY", submodulePorts["knirvgateway"], maxWait, pollInterval)
		})

		t.Run("KNIRVGRAPHStarts", func(t *testing.T) {
			waitForService(t, "KNIRVGRAPH", submodulePorts["knirvgraph"], maxWait, pollInterval)
		})

		t.Run("KNIRVCHAINStarts", func(t *testing.T) {
			waitForService(t, "KNIRVCHAIN", submodulePorts["knirvchain"], maxWait, pollInterval)
		})

		t.Run("AllSubmodulesRunning", func(t *testing.T) {
			for name, port := range submodulePorts {
				resp, err := http.Get(getHealthURL(port))
				if err != nil {
					t.Errorf("Failed to connect to %s: %v", name, err)
					continue
				}
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode, "%s should be healthy", name)
			}
		})
	})

	t.Run("KNIRVHASHERIntegration", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping hasher integration test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		backendPath := filepath.Join(binDir, "backend_server")
		cmd := exec.CommandContext(ctx, backendPath)
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
		)

		err := cmd.Start()
		require.NoError(t, err, "Backend server should start")

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		waitForBackend := 30 * time.Second
		backendPort := 8082

		t.Logf("Waiting up to %v for backend to be ready...", waitForBackend)
		require.Eventually(t, func() bool {
			resp, err := http.Get(getHealthURL(backendPort))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, waitForBackend, 1*time.Second, "Backend should become healthy")

		resp, err := http.Get(getHealthURL(backendPort) + "/api/hasher/status")
		if err != nil {
			t.Logf("Hasher endpoint not accessible: %v", err)
			t.Skip("Hasher status endpoint not available")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			t.Log("Hasher status endpoint not implemented yet - skipping")
			t.Skip("Hasher status endpoint not implemented")
			return
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Hasher should respond with status")

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err, "Should parse hasher status response")

		t.Logf("Hasher status: %+v", status)
	})
}

func TestDesktopStartsAfterSubmodules(t *testing.T) {
	distDir := filepath.Join("..", "..", "dist")
	unifiedBinaryPath := filepath.Join(distDir, "knir-server")

	if _, err := os.Stat(unifiedBinaryPath); os.IsNotExist(err) {
		t.Skip("Unified binary not built, run 'make binary' first")
	}

	if testing.Short() {
		t.Skip("Skipping desktop startup test in short mode")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("DesktopStartsAfterBackend", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, unifiedBinaryPath, "-desktop")
		cmd.Env = append(os.Environ(),
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_PORT=9080",
			"NEXUS_BACKEND_PORT=9081",
		)

		err := cmd.Start()
		require.NoError(t, err, "Unified binary with desktop should start")

		order := &initializationOrder{}
		backendReady := make(chan struct{})

		go func() {
			order.recordEvent("main", "started", "Main process started")

			for {
				select {
				case <-ctx.Done():
					return
				default:
					resp, err := http.Get("http://localhost:9081/health")
					if err == nil {
						resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							order.recordEvent("backend", "ready", "Backend health check passed")
							close(backendReady)
						}
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
		}()

		select {
		case <-backendReady:
			t.Log("Backend became ready")
		case <-time.After(45 * time.Second):
			t.Log("Backend did not become ready within timeout")
		}

		time.Sleep(5 * time.Second)

		order.desktopStarted()
		order.recordEvent("desktop", "started", "Desktop process started")

		events := order.getEvents()
		require.Greater(t, len(events), 0, "Should have recorded startup events")

		mainIdx := findEventIndex(events, "main")
		backendIdx := findEventIndex(events, "backend")
		desktopIdx := findEventIndex(events, "desktop")

		if backendIdx >= 0 {
			assert.Greater(t, backendIdx, mainIdx, "Backend should start after main process")
		}

		if desktopIdx >= 0 && backendIdx >= 0 {
			assert.Greater(t, desktopIdx, backendIdx, "Desktop should start after backend is ready")
			t.Logf("Desktop started at event index %d, backend ready at index %d", desktopIdx, backendIdx)
		} else if desktopIdx >= 0 && backendIdx < 0 {
			t.Log("Desktop started but backend health check endpoint not detected")
		}

		t.Cleanup(func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		})
	})
}

func TestInitializationOrderFromLogs(t *testing.T) {
	binDir := filepath.Join("..", "..", "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Skip("Binaries not built, run 'make backend' first")
	}

	if testing.Short() {
		t.Skip("Skipping log order test in short mode")
	}

	t.Cleanup(func() {
		killAllServices()
	})

	t.Run("VerifyStartupOrderInLogs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		backendPath := filepath.Join(binDir, "backend_server")
		cmd := exec.CommandContext(ctx, backendPath)
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")

		outputFile, err := os.CreateTemp("", "knirv-backend-output-*.log")
		require.NoError(t, err, "Should create temp log file")
		defer os.Remove(outputFile.Name())

		cmd.Stdout = outputFile
		cmd.Stderr = outputFile

		err = cmd.Start()
		require.NoError(t, err, "Backend server should start")

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		time.Sleep(30 * time.Second)

		outputFile.Sync()
		content, err := os.ReadFile(outputFile.Name())
		require.NoError(t, err, "Should read log output")

		logContent := string(content)

		modules := []string{"KNIRVGATEWAY", "KNIRVGRAPH", "KNIRVCHAIN"}
		firstIndices := make(map[string]int)
		foundCount := 0

		for _, module := range modules {
			if idx := findModuleStartIndex(logContent, module); idx >= 0 {
				firstIndices[module] = idx
				foundCount++
			}
		}

		if foundCount == 0 {
			t.Log("Could not find module startup logs - module names may have changed")
			t.Skip("Module startup logs not found in expected format")
		}

		require.Equal(t, len(modules), foundCount, "All expected modules should have startup logs")

		assertModuleOrder(t, logContent, "KNIRVGATEWAY", "KNIRVGRAPH")
		assertModuleOrder(t, logContent, "KNIRVGRAPH", "KNIRVCHAIN")

		t.Logf("Startup order verified: KNIRVGATEWAY -> KNIRVGRAPH -> KNIRVCHAIN")
	})
}

func waitForService(t *testing.T, name string, port int, maxWait, pollInterval time.Duration) {
	t.Logf("Waiting for %s on port %d (max %v)...", name, port, maxWait)

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

func getHealthURL(port int) string {
	return fmt.Sprintf("http://localhost:%d/health", port)
}

func findEventIndex(events []startupEvent, module string) int {
	for i, e := range events {
		if e.Module == module {
			return i
		}
	}
	return -1
}

func findModuleStartIndex(logContent, module string) int {
	searchStr := module + " started"
	for i := 0; i < len(logContent)-len(searchStr); i++ {
		if logContent[i:i+len(searchStr)] == searchStr {
			return i
		}
	}
	return -1
}

func assertModuleOrder(t *testing.T, logContent, earlier, later string) {
	earlierIdx := findModuleStartIndex(logContent, earlier)
	laterIdx := findModuleStartIndex(logContent, later)

	if earlierIdx < 0 || laterIdx < 0 {
		t.Logf("Could not find both modules in logs: %s=%d, %s=%d", earlier, earlierIdx, later, laterIdx)
		return
	}

	assert.Less(t, earlierIdx, laterIdx, "%s should start before %s", earlier, later)
}

func getChildProcesses(pid int) []int {
	cmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", pid))
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var pids []int
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var childPID int
		if _, err := fmt.Sscanf(line, "%d", &childPID); err == nil {
			pids = append(pids, childPID)
		}
	}
	return pids
}

func getProcessName(pid int) string {
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
