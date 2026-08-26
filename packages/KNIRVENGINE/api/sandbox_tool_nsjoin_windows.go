//go:build windows

package api

// geteuidImpl on Windows always returns 0 (Windows uses a different
// privilege model; tool namespace-join is Linux-only).
func geteuidImpl() int {
	return 0
}
