package loader

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
)

// sharedFramePayloadCases is the cross-language conformance fixture in KNIRVBASE.
const sharedFramePayloadCases = "../../../../../KNIRVBASE/testdata/framepayload/cases.json"

type nrvTestFrame struct {
	id      string
	count   int
	payload json.RawMessage
}

// buildNRV3Bytes lays out an NRV3 artifact as the encoder writes it: magic, a JSON
// registry carrying each frame's payload, then the 80-byte brackets back to back.
func buildNRV3Bytes(t *testing.T, frames []nrvTestFrame) []byte {
	t.Helper()
	type entry struct {
		ID      string          `json:"id"`
		Count   int             `json:"count"`
		Offset  int64           `json:"offset"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}
	var reg struct {
		Version string  `json:"version"`
		Frames  []entry `json:"frames"`
	}
	reg.Version = "3.0"
	var section []byte
	for _, f := range frames {
		reg.Frames = append(reg.Frames, entry{ID: f.id, Count: f.count, Offset: int64(len(section)), Payload: f.payload})
		section = append(section, make([]byte, f.count*80)...)
	}
	meta, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	data := append([]byte("NRV3"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(meta)))
	data = append(data, meta...)
	return append(data, section...)
}

// TestLoadNRVAssertionsReadsSharedFixture reads every shared case from a file.
// An ok case must decode to the canonical payload; an error case must fail the load.
func TestLoadNRVAssertionsReadsSharedFixture(t *testing.T) {
	raw, err := os.ReadFile(sharedFramePayloadCases)
	if err != nil {
		t.Fatalf("read shared fixture: %v", err)
	}
	var doc struct {
		Cases []struct {
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
			Expect    string          `json:"expect"`
			Canonical json.RawMessage `json:"canonical"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			count := 1
			if p, err := knirvnrv.DecodeFramePayload(c.Input); err == nil && p.TokenWindow != nil {
				count = int(p.TokenWindow.Brackets)
			}
			path := filepath.Join(t.TempDir(), "frames.nrv")
			if err := os.WriteFile(path, buildNRV3Bytes(t, []nrvTestFrame{{id: "case", count: count, payload: c.Input}}), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := LoadNRVAssertions(path)
			switch c.Expect {
			case "error":
				if err == nil {
					t.Fatal("loader accepted a payload the fixture says is invalid")
				}
			case "ok":
				if err != nil {
					t.Fatalf("loader refused a valid payload: %v", err)
				}
				if len(got) != 1 || got[0].Payload == nil {
					t.Fatalf("got %d frames, payload present=%v", len(got), len(got) == 1 && got[0].Payload != nil)
				}
				back, err := json.Marshal(*got[0].Payload)
				if err != nil {
					t.Fatal(err)
				}
				var gotV, wantV interface{}
				if err := json.Unmarshal(back, &gotV); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(c.Canonical, &wantV); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(gotV, wantV) {
					t.Fatalf("loaded payload differs from canonical:\n got  %s\n want %s", back, c.Canonical)
				}
			default:
				t.Fatalf("unknown expect %q", c.Expect)
			}
		})
	}
}

func TestLoadNRVAssertionsAllowsPayloadlessFrames(t *testing.T) {
	got, err := ParseNRVAssertions(buildNRV3Bytes(t, []nrvTestFrame{{id: "legacy", count: 2}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Brackets != 2 || got[0].Payload != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestLoadNRVAssertionsRejectsBracketCountMismatch(t *testing.T) {
	p, err := knirvnrv.EncodeFramePayload(knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Brackets: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseNRVAssertions(buildNRV3Bytes(t, []nrvTestFrame{{id: "m", count: 1, payload: p}})); err == nil {
		t.Fatal("token_window.brackets=3 accepted for a one-bracket frame")
	}
}

func TestLoadNRVAssertionsRejectsInvalidTier(t *testing.T) {
	bad := json.RawMessage(`{"supervision_episode":{"tool_trace":[{"tool":"x","tier":4}]}}`)
	if _, err := ParseNRVAssertions(buildNRV3Bytes(t, []nrvTestFrame{{id: "t", count: 1, payload: bad}})); err == nil {
		t.Fatal("tier 4 accepted")
	}
}
