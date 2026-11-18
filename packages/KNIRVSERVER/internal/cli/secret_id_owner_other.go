//go:build !unix

package cli

import "os"

// openSecretIDFile is a plain open off unix (no O_NOFOLLOW or O_NONBLOCK). readSecretIDFile
// still refuses non-regular files and symlinks through Lstat and the handle check.
func openSecretIDFile(path string) (*os.File, error) { return os.Open(path) }

// checkSecretIDOwner is a no-op off unix: there is no portable uid to compare against.
// The private-mode check in readSecretIDFile still applies on every platform.
func checkSecretIDOwner(fi os.FileInfo) error { return nil }
