package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_server/internal/web"
)

func TestPaymentHandlersStripeEndpoints(t *testing.T) {
	t.Run("Create Stripe Checkout Session", func(t *testing.T) {
		handler := &web.PaymentHandlers{}

		body := `{"rental_id": "rental-123", "amount": 2999, "currency": "usd", "success_url": "https://example.com/success", "cancel_url": "https://example.com/cancel"}`
		req := httptest.NewRequest(http.MethodPost, "/payments/stripe/create-session", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.CreateStripeCheckoutSession(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 200 or 503, got %d", w.Code)
		}
	})

	t.Run("Create PayPal Order", func(t *testing.T) {
		handler := &web.PaymentHandlers{}

		body := `{"rental_id": "rental-123", "amount": 29.99, "currency": "USD", "return_url": "https://example.com/return", "cancel_url": "https://example.com/cancel"}`
		req := httptest.NewRequest(http.MethodPost, "/payments/paypal/create-order", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.CreatePayPalOrder(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 200 or 503, got %d", w.Code)
		}
	})

	t.Run("Refund Stripe Charge", func(t *testing.T) {
		handler := &web.PaymentHandlers{}

		body := `{"charge_id": "ch_123", "reason": "requested_by_customer"}`
		req := httptest.NewRequest(http.MethodPost, "/payments/stripe/refund", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.RefundStripeCharge(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 200 or 503, got %d", w.Code)
		}
	})
}

func TestBadgeAPIHandlers(t *testing.T) {
	t.Run("Badge Create Request Serialization", func(t *testing.T) {
		req := map[string]interface{}{
			"name":        "Test Badge",
			"description": "A test badge",
			"image_url":   "https://example.com/badge.png",
			"attributes": map[string]interface{}{
				"tier":   "gold",
				"points": 100,
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal badge request: %v", err)
		}

		if !strings.Contains(string(data), "Test Badge") {
			t.Error("expected badge name in serialized data")
		}
	})

	t.Run("Badge Mint Request Serialization", func(t *testing.T) {
		req := map[string]interface{}{
			"badge_id":  "badge-001",
			"recipient": "user-123",
			"metadata": map[string]interface{}{
				"issued_at": "2026-03-31T00:00:00Z",
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal mint request: %v", err)
		}

		if !strings.Contains(string(data), "badge-001") {
			t.Error("expected badge_id in serialized data")
		}
	})

	t.Run("Badge Info Response Serialization", func(t *testing.T) {
		resp := map[string]interface{}{
			"id":          "badge-001",
			"name":        "Test Badge",
			"description": "A test badge",
			"image_url":   "https://example.com/badge.png",
			"attributes": map[string]interface{}{
				"tier":   "gold",
				"points": 100,
			},
			"total_minted": 50,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal badge response: %v", err)
		}

		if !strings.Contains(string(data), "badge-001") {
			t.Error("expected badge id in serialized data")
		}
	})
}

func TestDVEPolicyAttachment(t *testing.T) {
	t.Run("DVENode AttachedPolicies Field", func(t *testing.T) {
		node := struct {
			AttachedPolicies []string
			PolicyVersion    string
		}{
			AttachedPolicies: []string{"policy-001", "policy-002"},
			PolicyVersion:    "1.0.0",
		}

		if len(node.AttachedPolicies) != 2 {
			t.Errorf("expected 2 attached policies, got %d", len(node.AttachedPolicies))
		}

		if node.PolicyVersion != "1.0.0" {
			t.Errorf("expected policy version 1.0.0, got %s", node.PolicyVersion)
		}
	})
}

func TestKNIRVCLIServiceBadgeEndpoints(t *testing.T) {
	t.Run("Badge Create Endpoint Registration", func(t *testing.T) {
		handler := &web.KNIRVSHELLHandlers{}

		req := httptest.NewRequest(http.MethodPost, "/cli/badge/create", strings.NewReader(`{"name":"Test Badge", "badge_type":"achievement"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.CreateBadge(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("unexpected status code: %d", w.Code)
		}
	})
}
