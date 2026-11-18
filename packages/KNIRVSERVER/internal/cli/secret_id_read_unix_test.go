//go:build unix

package cli

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A FIFO at the secret_id path must be refused without blocking. Opening a FIFO with no
// writer blocks, so the test runs the read in a goroutine and fails if it does not return.
func TestReadSecretIDFileRefusesFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	type result struct {
		err error
		got string
	}
	done := make(chan result, 1)
	go func() {
		got, err := readSecretIDFile(path)
		done <- result{err: err, got: got}
	}()
	select {
	case r := <-done:
		if r.err == nil {
			t.Fatal("FIFO accepted as a secret-id file")
		}
		if r.got != "" {
			t.Fatal("secret value returned from a FIFO")
		}
		if !strings.Contains(r.err.Error(), "regular file") {
			t.Fatalf("unexpected error: %v", r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("readSecretIDFile blocked on a FIFO")
	}
}
