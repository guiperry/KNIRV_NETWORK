package did

import (
	"testing"
	"time"
)

func TestMemoryStorePutAndGet(t *testing.T) {
	s := NewMemoryStore()
	doc := &DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      "did:knirv:test123",
		Created: time.Now(),
		Updated: time.Now(),
	}

	if err := s.Put(doc); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := s.Get("did:knirv:test123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != "did:knirv:test123" {
		t.Errorf("expected did:knirv:test123, got %s", got.ID)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get("did:knirv:nonexistent")
	if err != ErrDIDNotFound {
		t.Errorf("expected ErrDIDNotFound, got %v", err)
	}
}

func TestMemoryStoreDeactivate(t *testing.T) {
	s := NewMemoryStore()
	doc := &DIDDocument{ID: "did:knirv:deactivate-me"}
	if err := s.Put(doc); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := s.Deactivate("did:knirv:deactivate-me"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	_, err := s.Get("did:knirv:deactivate-me")
	if err != ErrDIDDeactivated {
		t.Errorf("expected ErrDIDDeactivated, got %v", err)
	}
}

func TestMemoryStoreDoubleDeactivate(t *testing.T) {
	s := NewMemoryStore()
	doc := &DIDDocument{ID: "did:knirv:double"}
	if err := s.Put(doc); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := s.Deactivate("did:knirv:double"); err != nil {
		t.Fatalf("first Deactivate failed: %v", err)
	}
	if err := s.Deactivate("did:knirv:double"); err != nil {
		t.Fatalf("second Deactivate should succeed: %v", err)
	}
	_, err := s.Get("did:knirv:double")
	if err != ErrDIDDeactivated {
		t.Errorf("expected ErrDIDDeactivated after second deactivate, got %v", err)
	}
}
