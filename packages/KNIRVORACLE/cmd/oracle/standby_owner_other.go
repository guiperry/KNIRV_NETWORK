//go:build !unix

package main

import "os"

// checkSecretIDOwner is a no-op on non-Unix builds. Windows has no POSIX uid to compare
// against, so the owner check is not applied there. The mode (0600 or stricter) and
// symlink checks in readStandbySecretID still apply. The standby launcher runs on
// Linux and Unix, so this path is not used in production.
func checkSecretIDOwner(info os.FileInfo) error {
	return nil
}
