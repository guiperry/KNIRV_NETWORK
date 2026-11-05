//go:build windows

package authority

// setUmask is a no-op on Windows, which has no umask.
func setUmask(mask int) int { return 0 }
