//go:build !windows

package authority

import "syscall"

// setUmask sets the process umask and returns the previous one. It is used only
// around net.Listen so the socket file is created owner-only with no window in which
// another local user could connect.
func setUmask(mask int) int { return syscall.Umask(mask) }
