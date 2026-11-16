//go:build !unix

package cli

import "os"

// checkSecretIDOwner is a no-op off unix: there is no portable uid to compare against.
// The private-mode check in readSecretIDFile still applies on every platform.
func checkSecretIDOwner(fi os.FileInfo) error { return nil }
