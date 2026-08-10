package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// noopDisburser satisfies the Disburser interface without minting anything —
// none of the billing-lifecycle event tests below exercise the fiat->NRN
// disbursement path.
type noopDisburser struct{}

func (noopDisburser) FundAddress(types.Address, *big.Int, string) (*token.MintReceipt, error) {
	return nil, fmt.Errorf("not implemented in test")
}

// newTestProcessor builds a Processor wired to a local httptest server
// standing in for onboarding.knirv.com's /api/onboarding endpoint, so tests
// can assert on the callback payload without hitting the network.
func newTestProcessor(t *testing.T, onCallback func(payload map[string]interface{})) *Processor {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode callback payload: %v", err)
		}
		if onCallback != nil {
			onCallback(payload)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	return NewProcessor(Config{
		Enabled:               true,
		OnboardingCallbackURL: server.URL,
	}, noopDisburser{}, nil)
}

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

func TestMetadataPlanSession(t *testing.T) {
	object := map[string]interface{}{
		"metadata": map[string]interface{}{
			"plan":      "professional",
			"sessionId": "sess_123",
		},
	}
	plan, sessionID := metadataPlanSession(object)
	if plan != "professional" || sessionID != "sess_123" {
		t.Fatalf("expected (professional, sess_123), got (%s, %s)", plan, sessionID)
	}
}

func TestMetadataPlanSession_Missing(t *testing.T) {
	plan, sessionID := metadataPlanSession(map[string]interface{}{})
	if plan != "" || sessionID != "" {
		t.Fatalf("expected empty plan/session when metadata is absent, got (%s, %s)", plan, sessionID)
	}
}

// resolvePlanSession's mode=payment case: Stripe copies Checkout Session
// metadata directly onto the resulting PaymentIntent/Charge/Refund object, so
// this must resolve with zero Stripe API calls.
func TestResolvePlanSession_DirectMetadata(t *testing.T) {
	p := NewProcessor(Config{}, noopDisburser{}, nil)
	object := map[string]interface{}{
		"id": "re_123",
		"metadata": map[string]interface{}{
			"plan":      "investor",
			"sessionId": "sess_abc",
		},
	}
	plan, sessionID := p.resolvePlanSession(object)
	if plan != "investor" || sessionID != "sess_abc" {
		t.Fatalf("expected (investor, sess_abc), got (%s, %s)", plan, sessionID)
	}
}

// resolvePlanSession's mode=subscription case: metadata lives at
// subscription_details.metadata on the invoice, propagated there by
// CreateCheckoutSession's subscription_data[metadata] fields.
func TestResolvePlanSession_SubscriptionDetails(t *testing.T) {
	p := NewProcessor(Config{}, noopDisburser{}, nil)
	object := map[string]interface{}{
		"id": "in_123",
		"subscription_details": map[string]interface{}{
			"metadata": map[string]interface{}{
				"plan":      "enterprise",
				"sessionId": "sess_xyz",
			},
		},
	}
	plan, sessionID := p.resolvePlanSession(object)
	if plan != "enterprise" || sessionID != "sess_xyz" {
		t.Fatalf("expected (enterprise, sess_xyz), got (%s, %s)", plan, sessionID)
	}
}

// With no inline metadata and no subscription/invoice/payment_intent/charge
// reference to chase, resolution must fail closed (empty), not panic or fall
// back to some default plan.
func TestResolvePlanSession_Unresolvable(t *testing.T) {
	p := NewProcessor(Config{}, noopDisburser{}, nil)
	plan, sessionID := p.resolvePlanSession(map[string]interface{}{"id": "re_999"})
	if plan != "" || sessionID != "" {
		t.Fatalf("expected unresolvable object to yield empty plan/session, got (%s, %s)", plan, sessionID)
	}
}

func TestHandleRefundEvent_FailedIsLoggedOnly(t *testing.T) {
	called := false
	p := newTestProcessor(t, func(map[string]interface{}) { called = true })

	rec := httptest.NewRecorder()
	p.handleRefundEvent(rec, map[string]interface{}{"id": "re_1", "status": "failed"}, "refund.failed")

	if called {
		t.Fatal("refund.failed must never notify onboarding")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleRefundEvent_PendingIsIgnored(t *testing.T) {
	called := false
	p := newTestProcessor(t, func(map[string]interface{}) { called = true })

	rec := httptest.NewRecorder()
	p.handleRefundEvent(rec, map[string]interface{}{
		"id":     "re_2",
		"status": "pending",
		"metadata": map[string]interface{}{
			"plan": "professional", "sessionId": "sess_1",
		},
	}, "refund.created")

	if called {
		t.Fatal("a non-succeeded refund.created must not notify onboarding yet")
	}
}

func TestHandleRefundEvent_SucceededNotifiesAndDedupes(t *testing.T) {
	var payloads []map[string]interface{}
	p := newTestProcessor(t, func(payload map[string]interface{}) {
		payloads = append(payloads, payload)
	})

	object := map[string]interface{}{
		"id":     "re_3",
		"status": "succeeded",
		"metadata": map[string]interface{}{
			"plan": "professional", "sessionId": "sess_2",
		},
	}

	rec1 := httptest.NewRecorder()
	p.handleRefundEvent(rec1, object, "refund.created")
	if len(payloads) != 1 {
		t.Fatalf("expected exactly one onboarding notification, got %d", len(payloads))
	}
	if payloads[0]["eventType"] != "payment-refunded" || payloads[0]["plan"] != "professional" || payloads[0]["sessionId"] != "sess_2" {
		t.Fatalf("unexpected callback payload: %+v", payloads[0])
	}

	// Stripe may redeliver the same event; the dedupe key must suppress it.
	rec2 := httptest.NewRecorder()
	p.handleRefundEvent(rec2, object, "refund.created")
	if len(payloads) != 1 {
		t.Fatalf("expected redelivered refund.created to be deduped, got %d total notifications", len(payloads))
	}
}

func TestHandleInvoicePaidEvent_DedupesAcrossBothEventNames(t *testing.T) {
	var payloads []map[string]interface{}
	p := newTestProcessor(t, func(payload map[string]interface{}) {
		payloads = append(payloads, payload)
	})

	object := map[string]interface{}{
		"id": "in_1",
		"subscription_details": map[string]interface{}{
			"metadata": map[string]interface{}{
				"plan": "enterprise", "sessionId": "sess_3",
			},
		},
	}

	rec1 := httptest.NewRecorder()
	p.handleInvoicePaidEvent(rec1, object)
	if len(payloads) != 1 || payloads[0]["eventType"] != "payment-confirmed" {
		t.Fatalf("expected one payment-confirmed notification, got %+v", payloads)
	}

	// Stripe fires invoice.paid AND invoice.payment_succeeded for the same
	// invoice — both route through this same handler and must dedupe by
	// invoice ID, not by event name.
	rec2 := httptest.NewRecorder()
	p.handleInvoicePaidEvent(rec2, object)
	if len(payloads) != 1 {
		t.Fatalf("expected the second delivery for the same invoice ID to be deduped, got %d total notifications", len(payloads))
	}
}

func TestHandleInvoicePaidEvent_NoMetadataIsIgnoredNotErrored(t *testing.T) {
	called := false
	p := newTestProcessor(t, func(map[string]interface{}) { called = true })

	rec := httptest.NewRecorder()
	p.handleInvoicePaidEvent(rec, map[string]interface{}{"id": "in_2"})

	if called {
		t.Fatal("an invoice with no plan/session metadata (e.g. the fiat->NRN path) must not notify onboarding")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (ignored, not an error), got %d", rec.Code)
	}
}

func TestHandlePaymentActionRequiredEvent_NotifiesAndDedupes(t *testing.T) {
	var payloads []map[string]interface{}
	p := newTestProcessor(t, func(payload map[string]interface{}) {
		payloads = append(payloads, payload)
	})

	object := map[string]interface{}{
		"id": "pi_1",
		"metadata": map[string]interface{}{
			"plan": "professional", "sessionId": "sess_4",
		},
	}

	rec1 := httptest.NewRecorder()
	p.handlePaymentActionRequiredEvent(rec1, object, "payment_intent.requires_action")
	if len(payloads) != 1 || payloads[0]["eventType"] != "payment-action-required" {
		t.Fatalf("expected one payment-action-required notification, got %+v", payloads)
	}

	rec2 := httptest.NewRecorder()
	p.handlePaymentActionRequiredEvent(rec2, object, "payment_intent.requires_action")
	if len(payloads) != 1 {
		t.Fatalf("expected redelivery to be deduped, got %d total notifications", len(payloads))
	}
}
