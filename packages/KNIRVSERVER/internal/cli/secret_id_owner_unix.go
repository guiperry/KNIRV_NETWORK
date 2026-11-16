//go:build unix

package cli

import (
	"fmt"
	"os"
	"syscall"
)

// checkSecretIDOwner refuses a secret-id file that the current process user does not
// own (Q56). A secret_id is a bearer credential: a file planted or re-owned by another
// account must never be read as ours. fi must come from an open handle (f.Stat), not a
// path lookup, so the owner checked is the owner of the file actually read.
func checkSecretIDOwner(fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("secret-id file owner could not be determined")
	}
	if uint64(st.Uid) != uint64(os.Getuid()) {
		return fmt.Errorf("secret-id file must be owned by the current user")
	}
	return nil
}
