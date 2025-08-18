//go:build unix

package agentify

import "syscall"

// setProcGroup sets the process group for Unix systems
func setProcGroup(attr *syscall.SysProcAttr) {
	attr.Setpgid = true
}
