package asic

import (
	"fmt"

	"knirvhasher/pkg/hashing/jitter"
)

// DirectASICHASHMethod implements jitter.HashMethod by always calling the
// remote ComputeHash RPC via ASICClient.ComputeHashRemote - never the
// client's own local-software ComputeHash/ComputeDoubleHash. Correctness
// does not depend on the connected hasher-server actually running in
// -direct mode: if it isn't, the server just returns a plain
// sha256.Sum256(data) instead of a real ASIC round trip (Device.ComputeHash's
// existing default behavior), so this degrades to "slow, correct, network
// round trip per pass" rather than failing - the speed and hardware-backed
// property both come specifically from pairing this with a -direct server.
type DirectASICHASHMethod struct {
	client *ASICClient
}

// ComputeHash computes SHA-256 via the remote ComputeHash RPC. In direct
// mode this is a real hardware round trip; otherwise it degrades to the
// server's local-software sha256.
func (a *DirectASICHASHMethod) ComputeHash(data []byte) ([32]byte, error) {
	if a.client == nil {
		return [32]byte{}, fmt.Errorf("ASIC client not available")
	}
	return a.client.ComputeHashRemote(data)
}

// ComputeDoubleHash issues exactly one remote call - see this file's
// package-level note on why two round trips would be wrong, not just slow.
// One ComputeHashRemote call on an 80-byte header already gets back a real
// sha256d from a direct-mode server (Device.computeHashDirectASIC always
// double-hashes). Two round trips would be both wrong (double-double-hashing
// a value that's already been through the ASIC's own internal double-SHA256)
// and needlessly slow.
func (a *DirectASICHASHMethod) ComputeDoubleHash(data []byte) ([32]byte, error) {
	if a.client == nil {
		return [32]byte{}, fmt.Errorf("ASIC client not available")
	}
	return a.client.ComputeHashRemote(data)
}

func (a *DirectASICHASHMethod) ComputeDoubleHashBatch(data [][]byte) ([][32]byte, []uint32, error) {
	return a.client.ComputeBatchWitness(data)
}

var _ jitter.HashMethod = (*DirectASICHASHMethod)(nil)
