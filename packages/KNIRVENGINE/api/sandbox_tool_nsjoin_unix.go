//go:build !windows

package api

import "syscall"

// geteuidImpl returns the effective user ID on Unix systems.
func geteuidImpl() int {
	return syscall.Geteuid()
}
