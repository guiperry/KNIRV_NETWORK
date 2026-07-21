package types

import (
	"encoding/json"
	"testing"
)

func TestCheckpointLeafIsKindTagged(t *testing.T) {
	payload, err := json.Marshal((&Checkpoint{SchemaVersion: "knirv.checkpoint.v1", ChainID: "chain", StartHeight: 1, EndHeight: 2}).CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["kind"] != float64(1) {
		t.Fatalf("checkpoint leaf kind = %v, want 1", decoded["kind"])
	}
}
