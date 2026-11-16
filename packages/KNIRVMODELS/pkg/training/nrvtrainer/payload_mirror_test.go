package nrvtrainer

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// sharedPayloadCases is the cross-language fixture also read by KNIRVBASE Rust, Go and TS.
const sharedPayloadCases = "../../../../KNIRVBASE/testdata/framepayload/cases.json"

// TestSharedFixtureDecodesThroughKNIRVBASE decodes the shared cross-language fixture
// through KNIRVBASE, the same code path the trainer uses for every frame payload.
func TestSharedFixtureDecodesThroughKNIRVBASE(t *testing.T) {
	data, err := os.ReadFile(sharedPayloadCases)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []struct {
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
			Expect    string          `json:"expect"`
			Canonical json.RawMessage `json:"canonical"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Cases {
		p, err := nrv.DecodeFramePayload(c.Input)
		switch c.Expect {
		case "error":
			if err == nil {
				t.Errorf("%s: expected error", c.Name)
			}
		case "ok":
			if err != nil {
				t.Errorf("%s: %v", c.Name, err)
				continue
			}
			enc, _ := json.Marshal(p)
			var got, want any
			_ = json.Unmarshal(enc, &got)
			_ = json.Unmarshal(c.Canonical, &want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: encoded %s != canonical %s", c.Name, enc, c.Canonical)
			}
		}
	}
}

// TestCheckTokenRangeUsesU32: u32 token ids are range-checked against the vocab size without sign aliasing.
func TestCheckTokenRangeUsesU32(t *testing.T) {
	a := &Asset{Frames: []Frame{{ID: "f1", Z3Status: Z3Valid, Payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{0, 4294967295}}}}}}
	if err := a.CheckTokenRange(4294967295 + 0); err == nil {
		t.Fatal("token 4294967295 must be outside a vocab of 4294967295")
	}
	if err := a.CheckTokenRange(1 << 31); err == nil {
		t.Fatal("token 4294967295 must be outside a vocab of 2^31")
	}
	if err := a.CheckTokenRange(4294967296); err != nil {
		t.Fatalf("both tokens are inside a 2^32 vocab: %v", err)
	}
}
