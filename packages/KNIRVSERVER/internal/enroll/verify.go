package enroll

import (
	"context"
	"errors"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

var (
	errAdmissionUnconfigured   = errors.New("validator admission is not configured")
	errDeferredAuthUnsupported = errors.New("verifier does not support deferred nonce consumption")
	errEnrollmentRejected      = errors.New("enrollment signature, approval, expiry, or operator check failed")
	errReplayed                = errors.New("enrollment nonce already consumed (possible replay)")
	errWrongChain              = errors.New("enrollment chain does not match this node's chain")
	errUnknownRegistration     = errors.New("enrollment registration is not approved")
	errValidatorIDMismatch     = errors.New("enrollment public key does not match the registered validator ID")
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

func (v SDKVerifier) Verify(ctx context.Context, e signing.BootnodeEnrollment, nonce, expectedRootPublicKey, expectedOperator string, store signing.NonceStore, now time.Time) (bool, error) {
	valid, _, err := v.VerifyCoded(ctx, e, nonce, expectedRootPublicKey, expectedOperator, store, now)
	return valid, err
}

// CodedVerifier is a Verifier that also reports the core's stable refusal code
// (signing.EnrollCode*) for a rejected enrollment. The code is empty when the
// rejection carries none, such as a nonce the host store already consumed.
type CodedVerifier interface {
	VerifyCoded(ctx context.Context, e signing.BootnodeEnrollment, nonce, expectedRootPublicKey, expectedOperator string, store signing.NonceStore, now time.Time) (valid bool, code string, err error)
}

// VerifyCoded is Verify that also returns the core refusal code.
func (SDKVerifier) VerifyCoded(_ context.Context, e signing.BootnodeEnrollment, _nonce, expectedRootPublicKey, expectedOperator string, store signing.NonceStore, now time.Time) (bool, string, error) {
	return signing.VerifyBootnodeEnrollmentDetail(e, expectedRootPublicKey, expectedOperator, store, now)
}

// Authenticator verifies signatures/approval/expiry/operator but does NOT consume
// the nonce, returning the authenticated nonce to the host. The admission path
// uses this so that any rejection before the durable consume (signature, chain,
// registration, key binding, issuance) leaves the approval redeemable. The
// consume happens immediately before the validator-set write, so an admission
// failure AFTER that point (backend 5xx, publish conflict) DOES burn the nonce;
// the root must issue a fresh challenge and approval to retry.
type Authenticator interface {
	Authenticate(ctx context.Context, e signing.BootnodeEnrollment, expectedRootPublicKey, expectedOperator string, now time.Time) (nonce string, valid bool, err error)
}

func (v SDKVerifier) Authenticate(ctx context.Context, e signing.BootnodeEnrollment, expectedRootPublicKey, expectedOperator string, now time.Time) (string, bool, error) {
	nonce, valid, _, err := v.AuthenticateCoded(ctx, e, expectedRootPublicKey, expectedOperator, now)
	return nonce, valid, err
}

// CodedAuthenticator is an Authenticator that also reports the core refusal code.
type CodedAuthenticator interface {
	AuthenticateCoded(ctx context.Context, e signing.BootnodeEnrollment, expectedRootPublicKey, expectedOperator string, now time.Time) (nonce string, valid bool, code string, err error)
}

// AuthenticateCoded is Authenticate that also returns the core refusal code.
func (SDKVerifier) AuthenticateCoded(_ context.Context, e signing.BootnodeEnrollment, expectedRootPublicKey, expectedOperator string, now time.Time) (string, bool, string, error) {
	capture := &captureNonceStore{}
	valid, code, err := signing.VerifyBootnodeEnrollmentDetail(e, expectedRootPublicKey, expectedOperator, capture, now)
	if err != nil {
		return "", false, "", err
	}
	if !valid {
		return "", false, code, nil
	}
	if capture.nonce == "" {
		return "", false, "", errors.New("core verified enrollment without returning a nonce")
	}
	return capture.nonce, true, "", nil
}

// rejectionReason is the counter a refused enrollment increments. A refusal that
// carries a core code is counted as core_<code>, so each core check has its own
// counter. A refusal with no code (a replayed nonce, or a verifier that reports
// none) stays under verification_rejected.
func rejectionReason(code string) string {
	if code == "" {
		return "verification_rejected"
	}
	return "core_" + code
}

type captureNonceStore struct {
	nonce string
}

func (s *captureNonceStore) Consume(nonce string) (bool, error) {
	s.nonce = nonce
	return true, nil
}
