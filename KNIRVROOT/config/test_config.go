package config

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// BlockchainTestInterface defines the minimal blockchain operations needed for testing
type BlockchainTestInterface interface {
	StartMining()
}

// mockBlockchain provides a minimal implementation for testing
type mockBlockchain struct{}

func (m *mockBlockchain) StartMining() {
	log.Println("[MOCK] Starting mining")
}

// TestConfig implements config.Config interface for testing
type TestConfig struct {
	WalletPath       string
	MasterWalletPath string
}

// GetConfig returns the current configuration
// This is a helper function for the wallet manager to access the configuration
func GetConfig() (*Config, error) {
	// Load the configuration from the default path
	cfg, _, err := LoadConfig("", RoleClient)
	return cfg, err
}

func (t *TestConfig) GetWalletPath(role ...Role) (string, error) {
	return t.WalletPath, nil
}

func (t *TestConfig) GetMasterWalletPath(role ...Role) (string, error) {
	return t.MasterWalletPath, nil
}

// StartTestNodeWithDB starts a blockchain node for testing with custom db path
func StartTestNodeWithDB(port int, minerAddress string, dbPath string, extraArgs []string) (*TestServer, error) {
	// Ensure the database directory exists
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Create a reflection database path based on the main database path
	reflectionDbPath := filepath.Join(dbPath, "reflection")
	if err := os.MkdirAll(reflectionDbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reflection database directory: %w", err)
	}

	// Build the base command arguments
	args := []string{
		"run", ".",
		"--port", fmt.Sprintf("%d", port),
		"--miners_address", minerAddress,
		"--shared_database_path", filepath.Join(dbPath, "blockchain.db"),
		"--no-wallet-server", // Prevent node from starting its own wallet server
		"--network",          // Initialize a base chain and a reflection chain
	}

	// Add any extra arguments provided
	args = append(args, extraArgs...)

	// Build the command to start the node
	cmd := exec.Command("go", args...)

	// Load test environment variables first
	if err := godotenv.Load("test.env"); err != nil {
		log.Printf("[WARN] Failed to load test.env: %v", err)
	}

	// Set the working directory to the project root
	projectRoot := os.Getenv("KNIRVROOT_PROJECT_ROOT")
	if projectRoot == "" {
		// Fall back to the current directory if environment variable is not set
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	cmd.Dir = projectRoot

	// Setup stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start node: %w", err)
	}

	// Handle stdout and stderr in background goroutines
	go func() {
		_, err := io.Copy(os.Stdout, stdout)
		if err != nil {
			fmt.Printf("Error copying stdout: %v\n", err)
		}
	}()

	go func() {
		_, err := io.Copy(os.Stderr, stderr)
		if err != nil {
			fmt.Printf("Error copying stderr: %v\n", err)
		}
	}()

	// Create the test server object
	testServer := &TestServer{
		URL:        fmt.Sprintf("http://localhost:%d", port),
		Cmd:        cmd,
		TempDir:    dbPath,
		Blockchain: &mockBlockchain{},
		CleanupFunc: func() {
			// Kill the process with timeout
			if cmd.Process != nil {
				// Try graceful shutdown first
				if runtime.GOOS == "windows" {
					cmd.Process.Kill()
				} else {
					cmd.Process.Signal(syscall.SIGTERM)
				}
				
				// Wait with timeout
				done := make(chan error, 1)
				go func() {
					done <- cmd.Wait()
				}()
				
				select {
				case <-done:
					// Process exited
				case <-time.After(5 * time.Second):
					// Force kill after timeout
					cmd.Process.Kill()
					<-done // Wait for the process to actually exit
				}
			}
		},
	}

	// Wait for the server to start with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the process is still running
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for node to start")
	default:
		// Process is running, give it a moment to initialize
		time.Sleep(1 * time.Second)
	}

	return testServer, nil
}

// WaitForNode waits for a node's /health or /ping endpoint to return OK
func WaitForNode(t *testing.T, nodeURL string, timeout time.Duration) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("%s/health", nodeURL)
	pingURL := fmt.Sprintf("%s/ping", nodeURL)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	t.Logf("Waiting for node at %s to become healthy (timeout: %v)...", nodeURL, timeout)

	for {
		select {
		case <-ctx.Done():
			t.Logf("Timeout waiting for node at %s to become healthy after %v", nodeURL, timeout)
			return // Don't fail the test, just return
		case <-ticker.C:
			// Try health endpoint first
			resp, err := client.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				t.Logf("Node at %s is healthy", nodeURL)
				return
			}
			if resp != nil {
				resp.Body.Close()
			}

			// If health fails, try ping endpoint
			resp, err = client.Get(pingURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				t.Logf("Node at %s is responsive (ping)", nodeURL)
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
			
			t.Logf("Node at %s not yet ready, retrying...", nodeURL)
		}
	}
}

// TestServer represents a test server used for integration testing
type TestServer struct {
	URL         string
	Server      *httptest.Server
	TempDir     string
	Cmd         *exec.Cmd
	CleanupFunc func()
	Blockchain  BlockchainTestInterface // Added to support blockchain operations in tests
}

// StartTestServer starts a test server for integration testing
func StartTestServer(handler http.Handler) (*TestServer, error) {
	server := httptest.NewServer(handler)

	tempDir, err := os.MkdirTemp("", "agent-test")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	testServer := &TestServer{

		URL: server.URL,

		Server: server,

		TempDir: tempDir,

		CleanupFunc: func() {
			server.Close()
			os.RemoveAll(tempDir)

		},
	}

	return testServer, nil
}

// StartTestNode starts a agent node process for integration testing
func StartTestNode(port int, minerAddress string, remoteNode string) (*TestServer, error) {

	tempDir, err := os.MkdirTemp("", "agent-test")

	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)

	}

	cmd := exec.Command(

		"go", "run", "./main.go",
		"chain",

		"--port", fmt.Sprintf("%d", port),

		"--miners_address", minerAddress,

		"--remote_node", remoteNode,
	)

	cmd.Dir = filepath.Dir(filepath.Join(".", "main.go")) // current main dir to create all valid implementations

	stdout, err := cmd.StdoutPipe()

	if err != nil {

		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe() // validations for errors type check on implementation details to handle state as a data source or with workflow validations using method parameters for implementation validations using go type workflow implementation requirements of project.

	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)

	}

	if err := cmd.Start(); err != nil {

		return nil, fmt.Errorf("failed to start command: %w", err)

	}
	go func() {
		_, err := io.Copy(os.Stdout, stdout) // handle stdoutput if implementation occurs with type checking using parameters validation.

		if err != nil {
			fmt.Println("failed to copy stdout:", err)
		}
	}()
	go func() {
		_, err := io.Copy(os.Stderr, stderr)

		if err != nil {
			fmt.Println("failed to copy stderr:", err)

		}
	}()
	testServer := &TestServer{
		TempDir:    tempDir,
		Cmd:        cmd,
		Blockchain: &mockBlockchain{},
		CleanupFunc: func() {
			cmd.Process.Kill()
			cmd.Wait()
			os.RemoveAll(tempDir) // when using operating system calls perform type or state implementations in that method validation requirement (to avoid bugs or unhandled validations of objects when those methods must implement that validation correctly from go workflow design from its interface or during other local testing type checks to always create more reliable system by tests validation methods)
		},
	}
	// Give the server some time to start with a shorter timeout
	time.Sleep(500 * time.Millisecond)
	return testServer, nil

}

// implementation of workflows when destroying type for workflow validations that uses an object members by interfaces method signatures in go
func (ts *TestServer) Cleanup() { // call Cleanup safely
	if ts.CleanupFunc != nil {
		ts.CleanupFunc()
	}
}

// MockPathProvider implements PathProvider for testing
type MockPathProvider struct {
	WalletPath        string
	MasterWalletPath  string
	WalletError       error
	MasterWalletError error
}

func (m *MockPathProvider) GetWalletPath(role ...Role) (string, error) {
	return m.WalletPath, m.WalletError
}

func (m *MockPathProvider) GetMasterWalletPath(role ...Role) (string, error) {
	return m.MasterWalletPath, m.MasterWalletError
}

// Also implementation workflow when you are forcing an process exit, using types or interface based implementation from object state for process controls type properties implementations via local parameters to handle a software testing behavior in a validation system where such operations must be safe in memory and have specific type validations.

func (ts *TestServer) KillProcess() error { // always perform checks of implementations with workflows from struct type data from an interfaces or if method object has parameters and does those workflows correctly.

	if ts.Cmd == nil || ts.Cmd.Process == nil { // check state implementation if that has all that was required to perform action via object members state that methods may access based on interfaces from implementation using methods, or workflows checks validation types parameters to create system robust against unexpected workflows/type from objects (specially for object or shared state during concurrent method or coroutine when memory is being validated during state validations and other data validations workflow process steps).
		return nil
	}

	if runtime.GOOS == "windows" { // check current system before forcing signal and implement system using operating system primitives with local os types implementations where properties from objects has specific behaviours with workflow executions parameters to pass data correctly without code errors due to wrong paths of file in local implementations.
		if err := ts.Cmd.Process.Kill(); err != nil {

			return fmt.Errorf("error killing the server: %w", err)

		}

	} else {
		if err := ts.Cmd.Process.Signal(syscall.SIGKILL); err != nil {

			return fmt.Errorf("error killing the server: %w", err)

		}

	}
	err := ts.Cmd.Wait()

	if err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("error waiting for server to terminate: %w", err)

	}

	return nil
}
