package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func signStripePayload(secret string, timestamp int64, payload []byte) string {
	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyStripeSignature_Valid(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"type":"checkout.session.completed"}`)
	now := time.Now().Unix()
	sig := signStripePayload(secret, now, payload)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	if err := verifyStripeSignature(payload, header, secret, stripeSignatureTolerance); err != nil {
		t.Fatalf("expected valid signature to pass, got error: %v", err)
	}
}

func TestVerifyStripeSignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	now := time.Now().Unix()
	sig := signStripePayload("whsec_actual_secret", now, payload)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	if err := verifyStripeSignature(payload, header, "whsec_wrong_secret", stripeSignatureTolerance); err == nil {
		t.Fatal("expected signature computed with a different secret to be rejected")
	}
}

func TestVerifyStripeSignature_TamperedPayload(t *testing.T) {
	secret := "whsec_test_secret"
	originalPayload := []byte(`{"type":"checkout.session.completed","metadata":{"plan":"professional"}}`)
	now := time.Now().Unix()
	sig := signStripePayload(secret, now, originalPayload)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	tamperedPayload := []byte(`{"type":"checkout.session.completed","metadata":{"plan":"investor"}}`)
	if err := verifyStripeSignature(tamperedPayload, header, secret, stripeSignatureTolerance); err == nil {
		t.Fatal("expected a payload that doesn't match the signed one to be rejected")
	}
}

func TestVerifyStripeSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"type":"checkout.session.completed"}`)
	old := time.Now().Add(-1 * time.Hour).Unix()
	sig := signStripePayload(secret, old, payload)
	header := fmt.Sprintf("t=%d,v1=%s", old, sig)

	if err := verifyStripeSignature(payload, header, secret, stripeSignatureTolerance); err == nil {
		t.Fatal("expected a signature with a timestamp outside the tolerance window to be rejected (replay protection)")
	}
}

func TestVerifyStripeSignature_MissingHeader(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	if err := verifyStripeSignature(payload, "", "whsec_test_secret", stripeSignatureTolerance); err == nil {
		t.Fatal("expected a missing Stripe-Signature header to be rejected")
	}
}

func TestVerifyStripeSignature_MissingSecret(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	now := time.Now().Unix()
	sig := signStripePayload("whatever", now, payload)
	header := fmt.Sprintf("t=%d,v1=%s", now, sig)

	if err := verifyStripeSignature(payload, header, "", stripeSignatureTolerance); err == nil {
		t.Fatal("expected verification with no configured webhook secret to fail closed, not pass")
	}
}

func TestVerifyStripeSignature_MalformedHeader(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	cases := []string{
		"not-a-valid-header",
		"t=notanumber,v1=abcd",
		"v1=abcd",      // missing timestamp
		"t=1700000000", // missing v1
	}
	for _, header := range cases {
		if err := verifyStripeSignature(payload, header, "whsec_test_secret", stripeSignatureTolerance); err == nil {
			t.Fatalf("expected malformed header %q to be rejected", header)
		}
	}
}

// Stripe sends signatures for both the old and new secret during signing-
// secret rotation, as multiple v1= entries in the same header — a match on
// any one of them must be accepted.
func TestVerifyStripeSignature_RotationMultipleV1(t *testing.T) {
	payload := []byte(`{"type":"checkout.session.completed"}`)
	now := time.Now().Unix()
	oldSecret := "whsec_old"
	newSecret := "whsec_new"
	oldSig := signStripePayload(oldSecret, now, payload)
	newSig := signStripePayload(newSecret, now, payload)
	header := fmt.Sprintf("t=%d,v1=%s,v1=%s", now, oldSig, newSig)

	if err := verifyStripeSignature(payload, header, newSecret, stripeSignatureTolerance); err != nil {
		t.Fatalf("expected verification against the new secret to succeed during rotation, got: %v", err)
	}
	if err := verifyStripeSignature(payload, header, oldSecret, stripeSignatureTolerance); err != nil {
		t.Fatalf("expected verification against the old secret to still succeed during rotation, got: %v", err)
	}
}

// v0 entries are the legacy SHA-1 scheme; verification must ignore them and
// key only off v1, matching Stripe's own guidance.
func TestVerifyStripeSignature_IgnoresV0(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"type":"checkout.session.completed"}`)
	now := time.Now().Unix()
	sig := signStripePayload(secret, now, payload)
	header := fmt.Sprintf("t=%d,v0=deadbeef,v1=%s", now, sig)

	if err := verifyStripeSignature(payload, header, secret, stripeSignatureTolerance); err != nil {
		t.Fatalf("expected a valid v1 signature alongside an unrelated v0 entry to pass, got: %v", err)
	}
}
