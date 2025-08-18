//go:build !windows
// +build !windows

package api

import (
	"syscall"
)

// platformTerminateProcess implements process termination for Unix systems
func platformTerminateProcess(pid int, force bool) error {
	var signal syscall.Signal
	if force {
		signal = syscall.SIGKILL
	} else {
		signal = syscall.SIGTERM
	}
	return syscall.Kill(pid, signal)
}

// platformIsProcessAlive checks if a process is alive on Unix systems
func platformIsProcessAlive(pid int) bool {
	// On Unix, sending signal 0 checks if process exists without actually sending a signal
	err := syscall.Kill(pid, 0)
	return err == nil
}
