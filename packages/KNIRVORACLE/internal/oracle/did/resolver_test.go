package did

import (
	"math/big"
	"testing"
	"time"
)

func TestParseDID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMethod Method
		wantID     string
		wantErr    bool
	}{
		{"knirv method", "did:knirv:abc-123", MethodKNIRV, "abc-123", false},
		{"key method", "did:key:z6MkhaXgBZDvotDkL", MethodKey, "z6MkhaXgBZDvotDkL", false},
		{"missing prefix", "knirv:abc", "", "", true},
		{"empty method", "did::abc", "", "", true},
		{"empty id", "did:knirv:", "", "", true},
		{"unsupported method", "did:eth:abc", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, id, err := parseDID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if method != tt.wantMethod {
				t.Errorf("expected method %s, got %s", tt.wantMethod, method)
			}
			if id != tt.wantID {
				t.Errorf("expected id %s, got %s", tt.wantID, id)
			}
		})
	}
}

func TestBase58Decode(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLen    int
		wantErr    bool
	}{
		{"empty", "", 0, true},
		{"invalid char", "0OIl", 0, true},
		{"valid btc address prefix", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := base58Decode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("expected length %d, got %d", tt.wantLen, len(got))
			}
		})
	}
}

func TestBase58DecodeLeadingZeros(t *testing.T) {
	result, err := base58Decode("11abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) < 3 || result[0] != 0 || result[1] != 0 {
		t.Error("expected leading zero bytes from '1' chars")
	}
}

func TestResolverResolveKNIRV(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	doc := &DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      "did:knirv:node42",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if err := store.Put(doc); err != nil {
		t.Fatalf("store.Put failed: %v", err)
	}

	got, err := r.Resolve("did:knirv:node42")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.ID != "did:knirv:node42" {
		t.Errorf("expected did:knirv:node42, got %s", got.ID)
	}
}

func TestResolverResolveKNIRVNotFound(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)
	_, err := r.Resolve("did:knirv:ghost")
	if err != ErrDIDNotFound {
		t.Errorf("expected ErrDIDNotFound, got %v", err)
	}
}

func TestResolverRegister(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	doc := &DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      "did:knirv:registered-node",
		VerificationMethod: []VerificationMethod{
			{
				ID:         "did:knirv:registered-node#key-1",
				Type:       "Ed25519VerificationKey2020",
				Controller: "did:knirv:registered-node",
			},
		},
	}

	if err := r.Register(doc); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := r.Resolve("did:knirv:registered-node")
	if err != nil {
		t.Fatalf("Resolve after register failed: %v", err)
	}
	if got.ID != "did:knirv:registered-node" {
		t.Errorf("expected registered-node, got %s", got.ID)
	}
	if got.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if got.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}
}

func TestResolverRegisterInvalidID(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	tests := []struct {
		name string
		doc  *DIDDocument
	}{
		{"empty id", &DIDDocument{ID: ""}},
		{"non-knirv prefix", &DIDDocument{ID: "did:key:abc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := r.Register(tt.doc); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestResolverDeactivate(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	doc := &DIDDocument{ID: "did:knirv:to-deactivate"}
	if err := store.Put(doc); err != nil {
		t.Fatalf("store.Put failed: %v", err)
	}

	if err := r.Deactivate("did:knirv:to-deactivate"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	_, err := r.Resolve("did:knirv:to-deactivate")
	if err != ErrDIDDeactivated {
		t.Errorf("expected ErrDIDDeactivated, got %v", err)
	}
}

func TestResolverDeactivateNonKNIRV(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)
	if err := r.Deactivate("did:key:abc"); err == nil {
		t.Fatal("expected error for non-knirv deactivate")
	}
}

func TestResolverResolveKeyMultibase(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	t.Run("unsupported prefix", func(t *testing.T) {
		_, err := r.Resolve("did:key:wabc")
		if err == nil {
			t.Fatal("expected error for unsupported multibase prefix")
		}
	})

	t.Run("too short", func(t *testing.T) {
		_, err := r.Resolve("did:key:z")
		if err == nil {
			t.Fatal("expected error for too short encoded key")
		}
	})
}

func TestResolverUnsupportedMethod(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)
	_, err := r.Resolve("did:eth:0xabc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolverResolveKeyBase58Ed25519(t *testing.T) {
	store := NewMemoryStore()
	r := NewResolver(store)

	multibaseKey := "z" + base58Encode(append([]byte{ed25519Multicodec}, make([]byte, 32)...))
	did := "did:key:" + multibaseKey

	doc, err := r.Resolve(did)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	_ = doc

	if doc.ID != did {
		t.Errorf("expected DID %s, got %s", did, doc.ID)
	}

	if len(doc.VerificationMethod) != 1 {
		t.Fatalf("expected 1 verification method, got %d", len(doc.VerificationMethod))
	}

	vm := doc.VerificationMethod[0]
	if vm.Type != "Ed25519VerificationKey2020" {
		t.Errorf("expected Ed25519VerificationKey2020, got %s", vm.Type)
	}
	if vm.Controller != did {
		t.Errorf("expected controller %s, got %s", did, vm.Controller)
	}
	if vm.PublicKeyMultibase != multibaseKey {
		t.Errorf("expected publicKeyMultibase %s, got %s", multibaseKey, vm.PublicKeyMultibase)
	}

	if len(doc.Authentication) != 1 {
		t.Errorf("expected 1 authentication, got %d", len(doc.Authentication))
	}
	if len(doc.AssertionMethod) != 1 {
		t.Errorf("expected 1 assertionMethod, got %d", len(doc.AssertionMethod))
	}
}

func base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	zeroCount := 0
	for _, b := range input {
		if b == 0 {
			zeroCount++
		} else {
			break
		}
	}
	n := new(big.Int).SetBytes(input)
	radix := big.NewInt(58)
	zero := big.NewInt(0)
	var result []byte
	for n.Cmp(zero) > 0 {
		mod := new(big.Int)
		n.DivMod(n, radix, mod)
		result = append([]byte{base58Alphabet[mod.Int64()]}, result...)
	}
	for i := 0; i < zeroCount; i++ {
		result = append([]byte{'1'}, result...)
	}
	return string(result)
}
