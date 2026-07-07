package transactionchain

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"strings"
	"syscall"
	"time"
)

//go:embed bundle/*
var embeddedFiles embed.FS

const (
	extractionSubdir = "transaction_chain"
	hashFilename     = ".version_hash"
)

var (
	instance *TransactionChain
	once     sync.Once
)

// TransactionChain manages the embedded Node.js transaction chain service
type TransactionChain struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	cmd        *exec.Cmd
	port       int
	restartMu  sync.Mutex
	running    bool
}

// Get returns the singleton TransactionChain instance
func Get() *TransactionChain {
	once.Do(func() {
		instance = &TransactionChain{}
	})
	return instance
}

// calculateContentHash computes SHA256 hash of all embedded files
func calculateContentHash() (string, error) {
	h := sha256.New()

	err := fs.WalkDir(embeddedFiles, "bundle", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := embeddedFiles.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractFiles extracts embedded files to destination directory, stripping the bundle/ prefix
func extractFiles(destDir string) error {
	return fs.WalkDir(embeddedFiles, "bundle", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the leading "bundle/" prefix so files land at destDir directly
		rel := strings.TrimPrefix(path, "bundle/")
		if rel == "bundle" || rel == "" {
			return nil
		}

		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcFile, err := embeddedFiles.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// ExtractEmbeddedApp extracts embedded transaction chain to system cache directory
func ExtractEmbeddedApp() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	destDir := filepath.Join(cacheDir, "knirvserver", extractionSubdir)
	currentHash, err := calculateContentHash()
	if err != nil {
		return "", fmt.Errorf("failed to calculate content hash: %w", err)
	}

	hashFile := filepath.Join(destDir, hashFilename)
	if existingHash, err := os.ReadFile(hashFile); err == nil {
		if string(existingHash) == currentHash {
			return destDir, nil
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("failed to clean old extraction: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := extractFiles(destDir); err != nil {
		return "", fmt.Errorf("failed to extract files: %w", err)
	}

	if err := os.WriteFile(hashFile, []byte(currentHash), 0644); err != nil {
		return "", fmt.Errorf("failed to write version hash: %w", err)
	}

	return destDir, nil
}

func checkNodeNpmVersions() error {
	nodeCmd := exec.Command("node", "--version")
	if err := nodeCmd.Run(); err != nil {
		return fmt.Errorf("node.js not found: %w", err)
	}

	npmCmd := exec.Command("npm", "--version")
	if err := npmCmd.Run(); err != nil {
		return fmt.Errorf("npm not found: %w", err)
	}

	return nil
}

// Start launches the transaction chain service
func (tc *TransactionChain) Start(ctx context.Context, port int) error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if tc.running {
		return fmt.Errorf("transaction chain already running")
	}

	tc.ctx, tc.cancelFunc = context.WithCancel(ctx)
	tc.port = port

	destDir, err := ExtractEmbeddedApp()
	if err != nil {
		return err
	}

	if err := checkNodeNpmVersions(); err != nil {
		return err
	}

	installCmd := exec.CommandContext(tc.ctx, "npm", "install", "--production")
	installCmd.Dir = destDir
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	tc.cmd = exec.CommandContext(tc.ctx, "node", "main.js", "--port", strconv.Itoa(port))
	tc.cmd.Dir = destDir
	tc.cmd.Stdout = os.Stdout
	tc.cmd.Stderr = os.Stderr
	tc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := tc.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start transaction chain: %w", err)
	}

	tc.running = true

	go tc.monitorProcess(destDir)

	return nil
}

func (tc *TransactionChain) monitorProcess(destDir string) {
	for {
		_ = tc.cmd.Wait()
		tc.restartMu.Lock()

		if !tc.running {
			tc.restartMu.Unlock()
			return
		}

		select {
		case <-tc.ctx.Done():
			tc.running = false
			tc.restartMu.Unlock()
			return
		default:
		}

		time.Sleep(2 * time.Second)

		tc.cmd = exec.CommandContext(tc.ctx, "node", "main.js", "--port", strconv.Itoa(tc.port))
		tc.cmd.Dir = destDir
		tc.cmd.Stdout = os.Stdout
		tc.cmd.Stderr = os.Stderr
		tc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := tc.cmd.Start(); err != nil {
			tc.restartMu.Unlock()
			continue
		}

		tc.restartMu.Unlock()
	}
}

// Stop gracefully stops the transaction chain service
func (tc *TransactionChain) Stop() error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if !tc.running {
		return nil
	}

	tc.running = false
	tc.cancelFunc()

	if tc.cmd != nil && tc.cmd.Process != nil {
		syscall.Kill(-tc.cmd.Process.Pid, syscall.SIGTERM)
	}

	return nil
}

// HealthCheck verifies the transaction chain service is running
func (tc *TransactionChain) HealthCheck() error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if !tc.running {
		return fmt.Errorf("transaction chain not running")
	}

	if tc.cmd == nil || tc.cmd.Process == nil {
		return fmt.Errorf("transaction chain process not initialized")
	}

	select {
	case <-tc.ctx.Done():
		return fmt.Errorf("transaction chain context cancelled")
	default:
	}

	return nil
}

// Port returns the port the transaction chain is running on
func (tc *TransactionChain) Port() int {
	return tc.port
}
