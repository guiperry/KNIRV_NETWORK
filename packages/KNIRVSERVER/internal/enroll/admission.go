package enroll

import "context"

// ValidatorAdmission is the fully verified, key-bound enrollment record passed to
// the validator-set writer. It is only built after the boot-node signature, root
// countersignature, expiry, chain, registration and key/address binding have all
// been checked.
type ValidatorAdmission struct {
	ChainID        string
	RegistrationID string
	ValidatorID    string
	PublicKey      string // compressed secp256k1 public key, lower-case hex
}

// ValidatorAdmitter appends an admitted validator to the failover validator set.
// The production implementation lives in backend_server, which owns the root/
// failover signing identity and the registry client; the enrollment server never
// handles that key.
type ValidatorAdmitter interface {
	Admit(ctx context.Context, admission ValidatorAdmission) error
}

// Stats is a snapshot of enrollment outcomes for observability.
type Stats struct {
	Accepted   uint64            `json:"accepted"`
	Rejections map[string]uint64 `json:"rejections"`
}
