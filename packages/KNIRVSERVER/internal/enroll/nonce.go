package enroll

import (
	"path/filepath"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// NewNonceStore returns a durable NonceStore backed by a file under dataDir. It
// records consumed boot-node enrollment nonces so a verified enrollment cannot
// be replayed after its approval expires. It never holds secrets: only the
// opaque nonce strings are persisted.
func NewNonceStore(dataDir string) signing.NonceStore {
	return signing.NewFileNonceStore(filepath.Join(dataDir, "bootnode_enrollment_nonces.json"))
}
