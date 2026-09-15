package types

import (
	"encoding/json"
	"strings"
	"testing"

	"KNIRVCHAIN/internal/protocol/proto"
)

// The storage form and the canonical wire form are deliberately different types
// (see ErrorNodeRecord's doc comment). These tests pin the property that made
// that necessary: the persisted JSON shape must not change, because
// internal/graph/node_store.go reads previously stored rows straight back into
// the record. Adopting the wire type as the storage type would have changed
// "Open" to 1 and the context object to a protobuf Struct wrapper, silently
// breaking existing rows.
func TestErrorNodeRecordPersistedJSONShapeIsStable(t *testing.T) {
	record := &ErrorNodeRecord{
		ID:             "err-1",
		ErrorType:      "runtime",
		ErrorSignature: "sig-1",
		ModelOrigin:    "llama-3-8b",
		Context:        map[string]interface{}{"host": "example.com"},
		FailureCount:   2,
		CreatedAt:      1700000000,
		Status:         NodeStatusOpen,
	}

	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	encoded := string(raw)

	// Status stays a human-readable string, not an enum ordinal.
	if !strings.Contains(encoded, `"status":"Open"`) {
		t.Fatalf("status must persist as the string %q; got %s", "Open", encoded)
	}
	// Context stays a plain JSON object, not {"fields":{...}}.
	if !strings.Contains(encoded, `"context":{"host":"example.com"}`) {
		t.Fatalf("context must persist as a plain JSON object; got %s", encoded)
	}

	// And it reads back into the record unchanged.
	var decoded ErrorNodeRecord
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal stored record: %v", err)
	}
	if decoded.Status != NodeStatusOpen {
		t.Fatalf("status round trip = %q, want %q", decoded.Status, NodeStatusOpen)
	}
	if decoded.Context["host"] != "example.com" {
		t.Fatalf("context round trip = %v", decoded.Context)
	}
	if decoded.ID != "err-1" || decoded.ModelOrigin != "llama-3-8b" || decoded.FailureCount != 2 {
		t.Fatalf("record fields did not round trip: %+v", decoded)
	}
}

func TestErrorNodeRecordCanonicalRoundTrip(t *testing.T) {
	record := &ErrorNodeRecord{
		ID:             "err-1",
		ErrorType:      "runtime",
		ErrorSignature: "sig-1",
		ModelOrigin:    "llama-3-8b",
		Context:        map[string]interface{}{"host": "example.com", "attempts": float64(3)},
		FailureCount:   7,
		CreatedAt:      1700000000,
		Status:         NodeStatusResolved,
	}

	canonical, err := record.ToCanonical()
	if err != nil {
		t.Fatalf("ToCanonical: %v", err)
	}
	if canonical.GetId() != record.ID || canonical.GetModelOrigin() != record.ModelOrigin {
		t.Fatalf("canonical lost identity/provenance: %+v", canonical)
	}
	if canonical.GetStatus() != string(NodeStatusResolved) {
		t.Fatalf("canonical status = %q", canonical.GetStatus())
	}
	if canonical.GetContext().AsMap()["host"] != "example.com" {
		t.Fatalf("canonical context = %v", canonical.GetContext().AsMap())
	}

	back, err := ErrorNodeRecordFromCanonical(canonical)
	if err != nil {
		t.Fatalf("ErrorNodeRecordFromCanonical: %v", err)
	}
	if back.ID != record.ID ||
		back.ErrorType != record.ErrorType ||
		back.ErrorSignature != record.ErrorSignature ||
		back.ModelOrigin != record.ModelOrigin ||
		back.FailureCount != record.FailureCount ||
		back.CreatedAt != record.CreatedAt ||
		back.Status != record.Status {
		t.Fatalf("round trip lost fields:\n got  %+v\n want %+v", back, record)
	}
	if back.Context["host"] != "example.com" {
		t.Fatalf("round trip context = %v", back.Context)
	}
}

// An unrecognised status on the wire must be refused, not stored, so a bad value
// cannot corrupt the node store.
func TestErrorNodeRecordFromCanonicalRejectsUnknownStatus(t *testing.T) {
	_, err := ErrorNodeRecordFromCanonical(&proto.ErrorNode{Id: "err-1", Status: "Bogus"})
	if err == nil {
		t.Fatal("unknown status must be rejected")
	}
}

// A canonical node with no status defaults to Open rather than failing, since
// that is the state every newly created node starts in.
func TestErrorNodeRecordFromCanonicalDefaultsEmptyStatus(t *testing.T) {
	record, err := ErrorNodeRecordFromCanonical(&proto.ErrorNode{Id: "err-1"})
	if err != nil {
		t.Fatalf("empty status should default, got error: %v", err)
	}
	if record.Status != NodeStatusOpen {
		t.Fatalf("status = %q, want %q", record.Status, NodeStatusOpen)
	}
}
