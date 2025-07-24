//go:build windows

package agentify

import "syscall"

// setProcGroup is a no-op for Windows systems since Setpgid is not available
func setProcGroup(attr *syscall.SysProcAttr) {
	// Windows doesn't support Setpgid, so this is a no-op
}
