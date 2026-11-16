//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// standbyEffectiveUID returns the uid the secret-id file must be owned by. It is a
// package variable so tests can simulate an ownership mismatch without chown or sudo.
var standbyEffectiveUID = os.Geteuid

// checkSecretIDOwner requires the secret-id file to be owned by the process's effective
// uid. It fails closed when the platform stat record is not available.
func checkSecretIDOwner(info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return errors.New("standby: secret-id file owner cannot be verified on this platform")
	}
	if int(st.Uid) != standbyEffectiveUID() {
		return fmt.Errorf("standby: secret-id file must be owned by the oracle process uid (%d)", standbyEffectiveUID())
	}
	return nil
}
