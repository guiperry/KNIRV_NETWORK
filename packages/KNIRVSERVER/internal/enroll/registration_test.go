package enroll

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileRegistrationStoreMissingFileIsEmpty(t *testing.T) {
	store := NewFileRegistrationStore(filepath.Join(t.TempDir(), "absent.json"))
	_, ok, err := store.Lookup(context.Background(), "reg-1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if ok {
		t.Fatal("missing file must not yield an approved registration")
	}
}

func TestFileRegistrationStoreSupportsBothShapesAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registrations.json")
	store := NewFileRegistrationStore(path)

	bare := []Registration{{RegistrationID: "reg-1", ValidatorID: "knirv1aaa", ChainID: "chain-a"}}
	writeJSONFile(t, path, bare)
	reg, ok, err := store.Lookup(context.Background(), "reg-1")
	if err != nil || !ok {
		t.Fatalf("lookup bare array: ok=%v err=%v", ok, err)
	}
	if reg.ValidatorID != "knirv1aaa" || reg.ChainID != "chain-a" {
		t.Fatalf("unexpected registration: %+v", reg)
	}

	wrapped := map[string]any{"registrations": []Registration{
		{RegistrationID: "reg-1", ValidatorID: "knirv1aaa", ChainID: "chain-a"},
		{RegistrationID: "reg-2", ValidatorID: "knirv1bbb"},
	}}
	writeJSONFile(t, path, wrapped)
	reg, ok, err = store.Lookup(context.Background(), "reg-2")
	if err != nil || !ok {
		t.Fatalf("lookup wrapped object after reload: ok=%v err=%v", ok, err)
	}
	if reg.ValidatorID != "knirv1bbb" || reg.ChainID != "" {
		t.Fatalf("unexpected registration: %+v", reg)
	}
}

func TestFileRegistrationStoreRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{name: "missing validator id", data: []Registration{{RegistrationID: "reg-1"}}},
		{name: "missing registration id", data: []Registration{{ValidatorID: "knirv1aaa"}}},
		{name: "duplicate registration", data: []Registration{
			{RegistrationID: "reg-1", ValidatorID: "knirv1aaa"},
			{RegistrationID: "reg-1", ValidatorID: "knirv1bbb"},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "registrations.json")
			writeJSONFile(t, path, tc.data)
			if _, _, err := NewFileRegistrationStore(path).Lookup(context.Background(), "reg-1"); err == nil {
				t.Fatal("expected invalid registration file to be rejected")
			}
		})
	}
}

func TestParseEnrollmentClaims(t *testing.T) {
	claims := EnrollmentClaims{RegistrationID: "reg-1", ValidatorID: "knirv1aaa"}
	inner := MarshalClaims(claims)

	// Shape produced by the Rust core: {"operator":...,"claims":[byte,...]}.
	wrapped, err := json.Marshal(map[string]any{"operator": "operator:alice", "claims": inner})
	if err != nil {
		t.Fatalf("marshal wrapper: %v", err)
	}
	got, err := ParseEnrollmentClaims(wrapped)
	if err != nil {
		t.Fatalf("parse wrapped claims: %v", err)
	}
	if got != claims {
		t.Fatalf("got %+v, want %+v", got, claims)
	}

	// Bare claim object (forwards compatibility).
	got, err = ParseEnrollmentClaims(inner)
	if err != nil {
		t.Fatalf("parse bare claims: %v", err)
	}
	if got != claims {
		t.Fatalf("got %+v, want %+v", got, claims)
	}

	if _, err := ParseEnrollmentClaims([]byte(`{"registration_id":""}`)); err == nil {
		t.Fatal("empty registration_id must be rejected")
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
