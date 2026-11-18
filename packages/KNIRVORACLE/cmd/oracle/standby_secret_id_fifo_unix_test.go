//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A FIFO swapped in between the Lstat check and the open must not block the standby reader.
// The seam replaces the checked regular file with a FIFO at the same path, then calls the
// real opener. With O_NONBLOCK the open returns at once and the IsRegular check refuses it.
func TestReadStandbySecretIDRefusesFIFOSwappedAfterCheckWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := standbyOpenFile
	t.Cleanup(func() { standbyOpenFile = orig })
	standbyOpenFile = func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := syscall.Mkfifo(name, 0o600); err != nil {
			return nil, err
		}
		return openStandbySecretIDFile(name)
	}

	type result struct {
		got string
		err error
	}
	done := make(chan result, 1)
	go func() {
		got, err := readStandbySecretID(path)
		done <- result{got, err}
	}()
	select {
	case r := <-done:
		if r.err == nil {
			t.Fatal("FIFO swapped in after the check was accepted")
		}
		if r.got != "" {
			t.Fatal("secret value returned from a FIFO")
		}
		if !strings.Contains(r.err.Error(), "regular file") {
			t.Fatalf("expected the regular-file refusal, got: %v", r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("standby reader blocked on a FIFO swapped in after the check (open did not return)")
	}
}
