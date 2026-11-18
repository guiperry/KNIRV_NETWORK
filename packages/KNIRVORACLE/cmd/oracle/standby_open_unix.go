//go:build unix

package main

import (
	"os"
	"syscall"
)

// openStandbySecretIDFile opens the secret-id path without following a final symlink and
// without blocking on a FIFO or device swapped in after the Lstat check. The caller re-checks
// the opened handle (IsRegular, SameFile).
func openStandbySecretIDFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}
