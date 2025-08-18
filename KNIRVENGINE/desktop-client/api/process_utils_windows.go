//go:build windows
// +build windows

package api

import (
	"syscall"
	"unsafe"
)

// Windows API constants
const (
	PROCESS_TERMINATE         = 0x0001
	PROCESS_QUERY_INFORMATION = 0x0400
	STILL_ACTIVE              = 259
)

// Windows API functions
var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = kernel32.NewProc("OpenProcess")
	procTerminateProcess   = kernel32.NewProc("TerminateProcess")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
)

// platformTerminateProcess implements process termination for Windows systems
func platformTerminateProcess(pid int, force bool) error {
	// On Windows, TerminateProcess is always forceful (no SIGTERM equivalent)
	// The force parameter is unused but kept for interface consistency
	handle, err := openProcess(PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer closeHandle(handle)

	// Exit code 1 is a general error code
	return windowsTerminateProcess(handle, 1)
}

// platformIsProcessAlive checks if a process is alive on Windows systems
func platformIsProcessAlive(pid int) bool {
	handle, err := openProcess(PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer closeHandle(handle)

	var exitCode uint32
	if err := getExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}

	return exitCode == STILL_ACTIVE
}

// Helper functions for Windows API calls

func openProcess(desiredAccess uint32, inheritHandle bool, processId uint32) (handle syscall.Handle, err error) {
	inherit := 0
	if inheritHandle {
		inherit = 1
	}

	r1, _, e1 := procOpenProcess.Call(
		uintptr(desiredAccess),
		uintptr(inherit),
		uintptr(processId))

	if r1 == 0 {
		return 0, e1
	}
	return syscall.Handle(r1), nil
}

func windowsTerminateProcess(handle syscall.Handle, exitCode uint32) error {
	r1, _, e1 := procTerminateProcess.Call(
		uintptr(handle),
		uintptr(exitCode))

	if r1 == 0 {
		return e1
	}
	return nil
}

func closeHandle(handle syscall.Handle) error {
	r1, _, e1 := procCloseHandle.Call(uintptr(handle))
	if r1 == 0 {
		return e1
	}
	return nil
}

func getExitCodeProcess(handle syscall.Handle, exitCode *uint32) error {
	r1, _, e1 := procGetExitCodeProcess.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(exitCode)))

	if r1 == 0 {
		return e1
	}
	return nil
}
