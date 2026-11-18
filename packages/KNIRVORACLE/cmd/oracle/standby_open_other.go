//go:build !unix

package main

import "os"

// openStandbySecretIDFile is a plain open off unix (no O_NOFOLLOW or O_NONBLOCK). The caller
// still refuses non-regular files and symlinks through Lstat and the handle checks.
func openStandbySecretIDFile(path string) (*os.File, error) { return os.Open(path) }
