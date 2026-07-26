package p2pconsensus

import (
	"bytes"
	"testing"
)

func TestServiceIDAndTopicBindToSecret(t *testing.T) {
	open1 := ServiceID("net", "")
	open2 := ServiceID("net", "")
	if open1 != open2 {
		t.Fatal("open network IDs should be deterministic")
	}

	s1 := ServiceID("net", "secretA")
	s2 := ServiceID("net", "secretB")
	if s1 == s2 {
		t.Fatal("different secrets must produce different service IDs")
	}
	if s1 == open1 {
		t.Fatal("secret-bound ID must differ from open ID")
	}

	// Topic must also be secret-bound.
	if Topic("net", "secretA") == Topic("net", "secretB") {
		t.Fatal("different secrets must produce different topics")
	}
	if Topic("net", "") == Topic("net", "secretA") {
		t.Fatal("open topic must differ from secret-bound topic")
	}
}

func TestSignAndVerifyMessage(t *testing.T) {
	payload := []byte(`{"collection":"c","document_id":"d"}`)

	// With a secret, signing is deterministic and verifiable.
	sig := SignMessage("net", "topsecret", payload)
	if sig == "" {
		t.Fatal("expected non-empty signature with secret")
	}
	if !VerifyMessage("net", "topsecret", sig, payload) {
		t.Fatal("valid signature should verify")
	}
	// Tampered payload must fail.
	if VerifyMessage("net", "topsecret", sig, []byte("tampered")) {
		t.Fatal("tampered payload must not verify")
	}
	// Wrong secret must fail.
	if VerifyMessage("net", "wrong", sig, payload) {
		t.Fatal("wrong secret must not verify")
	}
}

func TestVerifyMessageOpenNetwork(t *testing.T) {
	payload := []byte("hello")
	// No secret configured: any/empty signature is accepted.
	if !VerifyMessage("net", "", "", payload) {
		t.Fatal("open network should accept unsigned payloads")
	}
	if !VerifyMessage("net", "", "anything", payload) {
		t.Fatal("open network should accept arbitrary signature")
	}
	// Once a secret is set, an empty signature must be rejected.
	if VerifyMessage("net", "secret", "", payload) {
		t.Fatal("secret-bound network must reject empty signature")
	}
}

func TestSignMessageEmptySecret(t *testing.T) {
	if sig := SignMessage("net", "", []byte("x")); sig != "" {
		t.Fatalf("open network should produce no signature, got %q", sig)
	}
}

func TestHMACCommutativity(t *testing.T) {
	payload := []byte("consistent")
	a := SignMessage("n", "s", payload)
	b := SignMessage("n", "s", payload)
	if !bytes.Equal([]byte(a), []byte(b)) {
		t.Fatal("signatures must be deterministic for identical inputs")
	}
}
