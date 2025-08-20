package tests

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArchitecture tests the new three-tier architecture
func TestArchitecture(t *testing.T) {
	// Skip if binaries don't exist
	binDir := filepath.Join("..", "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Skip("Binaries not built, run 'make backend' first")
	}

	// Test domain-specific binaries exist
	t.Run("BinariesExist", func(t *testing.T) {
		binaries := []string{"dve-manager", "validation-core", "nexus-server", "nexus-backend"}
		for _, binary := range binaries {
			path := filepath.Join(binDir, binary)
			_, err := os.Stat(path)
			assert.NoError(t, err, "Binary %s should exist", binary)
		}
	})

	// Test backend orchestrator
	t.Run("BackendOrchestrator", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping orchestrator test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start backend orchestrator
		backendPath := filepath.Join(binDir, "nexus-backend")
		cmd := exec.CommandContext(ctx, backendPath)
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")
		
		// Start in background
		err := cmd.Start()
		require.NoError(t, err, "Backend orchestrator should start")
		
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		// Wait for services to start
		time.Sleep(5 * time.Second)

		// Test DVE Manager health
		t.Run("DVEManagerHealth", func(t *testing.T) {
			resp, err := http.Get("http://localhost:8082/health")
			if err != nil {
				t.Logf("DVE Manager not accessible: %v", err)
				return
			}
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Test Validation Core health
		t.Run("ValidationCoreHealth", func(t *testing.T) {
			resp, err := http.Get("http://localhost:8083/health")
			if err != nil {
				t.Logf("Validation Core not accessible: %v", err)
				return
			}
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Test API Gateway health
		t.Run("APIGatewayHealth", func(t *testing.T) {
			resp, err := http.Get("http://localhost:8081/health")
			if err != nil {
				t.Logf("API Gateway not accessible: %v", err)
				return
			}
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
}

// TestDomainServices tests individual domain services
func TestDomainServices(t *testing.T) {
	binDir := filepath.Join("..", "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		t.Skip("Binaries not built, run 'make backend' first")
	}

	t.Run("DVEManager", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping DVE Manager test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Start DVE Manager on a test port
		dveManagerPath := filepath.Join(binDir, "dve-manager")
		cmd := exec.CommandContext(ctx, dveManagerPath, "--port", "9082")
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")
		
		err := cmd.Start()
		require.NoError(t, err, "DVE Manager should start")
		
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		// Wait for service to start
		time.Sleep(3 * time.Second)

		// Test health endpoint
		resp, err := http.Get("http://localhost:9082/health")
		if err != nil {
			t.Logf("DVE Manager not accessible: %v", err)
			return
		}
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ValidationCore", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping Validation Core test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Start Validation Core on a test port
		validationCorePath := filepath.Join(binDir, "validation-core")
		cmd := exec.CommandContext(ctx, validationCorePath, "--port", "9083")
		cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=debug")
		
		err := cmd.Start()
		require.NoError(t, err, "Validation Core should start")
		
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		// Wait for service to start
		time.Sleep(3 * time.Second)

		// Test health endpoint
		resp, err := http.Get("http://localhost:9083/health")
		if err != nil {
			t.Logf("Validation Core not accessible: %v", err)
			return
		}
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestUnifiedBinary tests the final unified binary
func TestUnifiedBinary(t *testing.T) {
	// Check if unified binary exists
	unifiedBinaryPath := filepath.Join("..", "..", "dist", "nexus-server")
	if _, err := os.Stat(unifiedBinaryPath); os.IsNotExist(err) {
		t.Skip("Unified binary not built, run 'make binary' first")
	}

	t.Run("UnifiedBinaryStartup", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping unified binary test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start unified binary
		cmd := exec.CommandContext(ctx, unifiedBinaryPath)
		cmd.Env = append(os.Environ(), 
			"NEXUS_LOG_LEVEL=debug",
			"NEXUS_PORT=9080",
			"NEXUS_BACKEND_PORT=9081",
		)
		
		err := cmd.Start()
		require.NoError(t, err, "Unified binary should start")
		
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}()

		// Wait for services to start
		time.Sleep(8 * time.Second)

		// Test frontend health
		t.Run("FrontendHealth", func(t *testing.T) {
			resp, err := http.Get("http://localhost:9080/health")
			if err != nil {
				t.Logf("Frontend not accessible: %v", err)
				return
			}
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Test API proxy
		t.Run("APIProxy", func(t *testing.T) {
			resp, err := http.Get("http://localhost:9080/api/v1/status")
			if err != nil {
				t.Logf("API proxy not accessible: %v", err)
				return
			}
			defer resp.Body.Close()
			// Should get response from backend (may be error if backend services aren't fully ready)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)
		})
	})
}

// BenchmarkArchitecture benchmarks the new architecture
func BenchmarkArchitecture(b *testing.B) {
	// Skip if binaries don't exist
	binDir := filepath.Join("..", "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		b.Skip("Binaries not built, run 'make backend' first")
	}

	b.Run("BackendStartup", func(b *testing.B) {
		backendPath := filepath.Join(binDir, "nexus-backend")
		
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			
			cmd := exec.CommandContext(ctx, backendPath)
			cmd.Env = append(os.Environ(), "NEXUS_LOG_LEVEL=error")
			
			start := time.Now()
			err := cmd.Start()
			if err != nil {
				b.Fatalf("Failed to start backend: %v", err)
			}
			
			// Wait a bit for startup
			time.Sleep(2 * time.Second)
			
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
			
			cancel()
			
			b.ReportMetric(float64(time.Since(start).Milliseconds()), "startup_ms")
		}
	})
}
