package enroll

import (
	"context"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// Verifier validates an approved boot-node enrollment: it enforces the signed
// challenge, the root owner's approval, and replay prevention via a NonceStore.
// The production implementation delegates to signing.VerifyBootnodeEnrollment
// (the D14 server half); tests inject a fake that exercises the same contract.
//
// nonce is the enrollment nonce carried by the request. The production verifier
// derives the nonce from the core and ignores this argument; a fake verifier
// consumes this explicit nonce so the test can drive replay prevention through
// the real signing.NonceStore.
type Verifier interface {
	Verify(ctx context.Context, e signing.BootnodeEnrollment, nonce, expectedRootPublicKey, expectedOperator string, store signing.NonceStore, now time.Time) (bool, error)
}

// SDKVerifier is the production verifier. It calls the already-implemented D14
// server half (signing.VerifyBootnodeEnrollment), which authenticates the full
// approval->challenge->root binding through the Rust core and durably consumes
// the returned nonce via the host-owned NonceStore.
type SDKVerifier struct{}

func (SDKVerifier) Verify(ctx context.Context, e signing.BootnodeEnrollment, _nonce, expectedRootPublicKey, expectedOperator string, store signing.NonceStore, now time.Time) (bool, error) {
	return signing.VerifyBootnodeEnrollment(e, expectedRootPublicKey, expectedOperator, store, now)
}
