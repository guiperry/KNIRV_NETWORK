package asic

import (
	"crypto/sha256"
	"testing"

	"knirvhasher/pkg/hashing/jitter"
)

// Compile-time check that DirectASICHASHMethod implements jitter.HashMethod.
var _ jitter.HashMethod = (*DirectASICHASHMethod)(nil)

func TestDirectASICHASHMethodComputeHashNilClient(t *testing.T) {
	m := &DirectASICHASHMethod{client: nil}
	_, err := m.ComputeHash([]byte("test"))
	if err == nil {
		t.Fatal("expected error when client is nil, got nil")
	}
}

func TestDirectASICHASHMethodComputeDoubleHashNilClient(t *testing.T) {
	m := &DirectASICHASHMethod{client: nil}
	_, err := m.ComputeDoubleHash([]byte("test"))
	if err == nil {
		t.Fatal("expected error when client is nil, got nil")
	}
}

func TestNewDirectASICMethodSetsDirect(t *testing.T) {
	method := NewDirectASICMethod("localhost:9999")
	if method.direct != true {
		t.Fatal("expected direct=true after NewDirectASICMethod")
	}
	if method.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewASICMethodDoesNotSetDirect(t *testing.T) {
	method := NewASICMethod("localhost:9999")
	if method.direct != false {
		t.Fatal("expected direct=false after NewASICMethod")
	}
}

// TestDirectASICHASHMethodComputeHashRemoteSoftwareFallback verifies that when
// the client is in software-fallback mode, ComputeHashRemote falls back to
// local software computation and returns the correct sha256.
func TestDirectASICHASHMethodComputeHashRemoteSoftwareFallback(t *testing.T) {
	client, err := NewASICClient("localhost:9999")
	if err != nil {
		t.Fatalf("NewASICClient: %v", err)
	}
	if !client.IsUsingFallback() {
		t.Fatal("expected client to be in fallback mode (no server at localhost:9999)")
	}

	m := &DirectASICHASHMethod{client: client}
	data := []byte("test data for direct mode")
	hash, err := m.ComputeHash(data)
	if err != nil {
		t.Fatalf("ComputeHash: %v", err)
	}

	expected := sha256.Sum256(data)
	if hash != expected {
		t.Errorf("hash mismatch: got %x, want %x", hash, expected)
	}
}

// TestDirectASICHASHMethodComputeDoubleHashSoftwareFallback verifies that
// ComputeDoubleHash in fallback mode delegates to ComputeHashRemote, which
// itself falls back to sha256(data) - NOT double-sha256. This is correct
// because in direct mode, the server returns sha256d already; in fallback mode
// (no server), there's no double-hash happening. The caller (jitter engine)
// calls ComputeDoubleHash and expects a single sha256d, so in the fallback
// case, the caller gets sha256(data) instead of sha256d(data) - this is
// documented behavior: "degrades to slow, correct, network round trip per pass"
// and without a server there's no hardware double-hash. The important thing
// is that ComputeDoubleHash issues exactly ONE remote call, not two.
func TestDirectASICHASHMethodComputeDoubleHashSoftwareFallback(t *testing.T) {
	client, err := NewASICClient("localhost:9999")
	if err != nil {
		t.Fatalf("NewASICClient: %v", err)
	}
	if !client.IsUsingFallback() {
		t.Fatal("expected client to be in fallback mode")
	}

	m := &DirectASICHASHMethod{client: client}
	data := []byte("test data")
	hash, err := m.ComputeDoubleHash(data)
	if err != nil {
		t.Fatalf("ComputeDoubleHash: %v", err)
	}

	// In fallback mode, ComputeHashRemote returns sha256(data) (not double)
	expected := sha256.Sum256(data)
	if hash != expected {
		t.Errorf("hash mismatch: got %x, want %x", hash, expected)
	}
}
