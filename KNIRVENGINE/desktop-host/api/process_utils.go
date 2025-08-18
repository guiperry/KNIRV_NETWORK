package api

// terminateProcess sends a termination signal to a process.
// If force is true, it will use a stronger signal (SIGKILL on Unix, TerminateProcess on Windows).
// This function is implemented differently for each platform.
func terminateProcess(pid int, force bool) error {
	return platformTerminateProcess(pid, force)
}

// isProcessAlive checks if a process with the given PID is still running.
// This function is implemented differently for each platform.
func isProcessAlive(pid int) bool {
	return platformIsProcessAlive(pid)
}
