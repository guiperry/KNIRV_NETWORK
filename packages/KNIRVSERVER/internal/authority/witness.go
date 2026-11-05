package authority

import "context"

// Witnesser finds the proof-of-work witness for an 80-byte authority header
// (AUTHORITY_SPEC.md section 2.2). header carries every field except the final
// nonce, which the witnesser chooses; targetBits must equal the header's own
// nbits. It returns the nonce and the resulting sha256d witness.
//
// The software witnesser is the only implementation in 2b.1. An ASIC witnesser
// (Task 7.1) implements the same interface and reports Kind "asic"; verification
// is identical for both (section 2.2).
type Witnesser interface {
	Mine(ctx context.Context, header [80]byte, targetBits uint32) (nonce uint32, hash [32]byte, err error)
	// Kind is the record's witness_kind: "sw" or "asic".
	Kind() string
}
