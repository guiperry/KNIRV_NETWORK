package nrvio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
)

// sharedFramePayloadCases is the cross-language conformance fixture in KNIRVBASE.
const sharedFramePayloadCases = "../../../../../KNIRVBASE/testdata/framepayload/cases.json"

type framePayloadCase struct {
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Expect    string          `json:"expect"`
	Canonical json.RawMessage `json:"canonical"`
}

func loadFramePayloadCases(t *testing.T) []framePayloadCase {
	t.Helper()
	raw, err := os.ReadFile(sharedFramePayloadCases)
	if err != nil {
		t.Fatalf("read shared fixture: %v", err)
	}
	var doc struct {
		Cases []framePayloadCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode shared fixture: %v", err)
	}
	if len(doc.Cases) == 0 {
		t.Fatal("shared fixture has no cases")
	}
	return doc.Cases
}

// fixtureBrackets returns n distinct brackets so a round trip also proves the
// bracket section is not shifted by the payload.
func fixtureBrackets(n int) []Bracket {
	out := make([]Bracket, n)
	for i := range out {
		out[i] = Bracket{SubSecondUS: uint32(i + 1), Syntactic: PackSyntactic(uint8(i%16), 1, 0), DomainSig: 0x2000, GoldenSeed: uint32(100 + i)}
		out[i].Projections[0] = byte(i)
	}
	return out
}

func jsonValue(t *testing.T, b []byte) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return v
}

// TestFramePayloadFixtureRoundTrip drives every shared case through writer -> file
// -> reader. An expect=ok case must come back as the identical canonical payload
// and the identical brackets; an expect=error case must be refused at decode or
// at write.
func TestFramePayloadFixtureRoundTrip(t *testing.T) {
	for _, c := range loadFramePayloadCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			p, decodeErr := knirvnrv.DecodeFramePayload(c.Input)
			switch c.Expect {
			case "error":
				if decodeErr == nil {
					// Decodes, so the writer must refuse it instead.
					n := 0
					if p.TokenWindow != nil {
						n = int(p.TokenWindow.Brackets)
					}
					w, err := NewWriter(filepath.Join(t.TempDir(), "x.nrv"))
					if err != nil {
						t.Fatal(err)
					}
					if err := w.AppendAssertion("bad", fixtureBrackets(n), p); err == nil {
						t.Fatal("writer accepted a payload the fixture says must be rejected")
					}
				}
				return
			case "ok":
			default:
				t.Fatalf("unknown expect %q", c.Expect)
			}
			if decodeErr != nil {
				t.Fatalf("decode: %v", decodeErr)
			}
			n := 1
			if p.TokenWindow != nil {
				n = int(p.TokenWindow.Brackets)
			}
			want := fixtureBrackets(n)
			path := filepath.Join(t.TempDir(), "assertions.nrv")
			w, err := NewWriter(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.AppendAssertion("case", want, p); err != nil {
				t.Fatalf("write: %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}

			r, err := Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			got, err := r.Assertions()
			if err != nil {
				t.Fatalf("assertions: %v", err)
			}
			if len(got) != 1 || got[0].Payload == nil {
				t.Fatalf("got %d assertions, payload present=%v", len(got), len(got) == 1 && got[0].Payload != nil)
			}
			back, err := json.Marshal(*got[0].Payload)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(jsonValue(t, back), jsonValue(t, c.Canonical)) {
				t.Fatalf("payload round trip differs from canonical:\n got  %s\n want %s", back, c.Canonical)
			}
			if !reflect.DeepEqual(got[0].Brackets, want) {
				t.Fatalf("brackets changed across the file")
			}
		})
	}
}

func TestAssertionPayloadBuildsTokenWindow(t *testing.T) {
	p, err := AssertionPayload([]int32{5, 9}, 12, 3)
	if err != nil {
		t.Fatal(err)
	}
	tw := p.TokenWindow
	if tw == nil || !reflect.DeepEqual(tw.Tokens, []uint32{5, 9}) || tw.Start != 12 || tw.Brackets != 3 {
		t.Fatalf("token window = %#v", tw)
	}
	if _, err := AssertionPayload([]int32{-1}, 0, 1); err == nil {
		t.Fatal("negative token id accepted")
	}
	if _, err := AssertionPayload(nil, -1, 1); err == nil {
		t.Fatal("negative window start accepted")
	}
}

func TestAppendAssertionRejectsBracketCountMismatch(t *testing.T) {
	w, err := NewWriter(filepath.Join(t.TempDir(), "m.nrv"))
	if err != nil {
		t.Fatal(err)
	}
	p := knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Brackets: 2}}
	if err := w.AppendAssertion("m", fixtureBrackets(1), p); err == nil {
		t.Fatal("token_window.brackets=2 accepted for a one-bracket frame")
	}
}

func TestAssertionsRejectCorruptPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.nrv")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.AppendAssertion("c", fixtureBrackets(1), knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Brackets: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Tamper with the decoded registry in memory: a tier outside 0..=3.
	r.registry.Frames[0].Payload = json.RawMessage(`{"supervision_episode":{"tool_trace":[{"tool":"x","tier":9}]}}`)
	if _, err := r.Assertions(); err == nil {
		t.Fatal("invalid payload accepted by the reader")
	}
}

// TestAppendAssertionRefusesEveryFixtureErrorCase checks the write side on its own.
// Every expect=error case whose input unmarshals into FramePayload with plain
// encoding/json must be refused by AppendAssertion, so the writer cannot produce a
// file that only the reader would later reject.
func TestAppendAssertionRefusesEveryFixtureErrorCase(t *testing.T) {
	refused := 0
	for _, c := range loadFramePayloadCases(t) {
		if c.Expect != "error" {
			continue
		}
		if string(bytes.TrimSpace(c.Input)) == "null" {
			// A Go FramePayload value cannot express JSON null: the zero value is
			// the same value as {} (an accepted empty payload). The writer only
			// sees typed values, so this case is refused at decode, not at write.
			continue
		}
		var plain knirvnrv.FramePayload
		if err := json.Unmarshal(c.Input, &plain); err != nil {
			continue // not a FramePayload object at all; nothing for the writer to accept
		}
		t.Run(c.Name, func(t *testing.T) {
			n := 1
			if plain.TokenWindow != nil {
				n = int(plain.TokenWindow.Brackets)
			}
			w, err := NewWriter(filepath.Join(t.TempDir(), "x.nrv"))
			if err != nil {
				t.Fatal(err)
			}
			if err := w.AppendAssertion("bad", fixtureBrackets(n), plain); err == nil {
				t.Fatal("AppendAssertion accepted a payload the fixture says must be rejected")
			}
			if len(w.frames) != 0 {
				t.Fatal("a refused payload left a frame behind")
			}
		})
		refused++
	}
	if refused == 0 {
		t.Fatal("no fixture error case reached the writer; the test would prove nothing")
	}
}

// TestCanonicalOpenNRV3ReadsAssertionFiles opens a file written by the encoder with
// KNIRVBASE's canonical reader. The registry offsets, counts and bracket bytes must
// agree with what was written. The payload is not visible to OpenNRV3 (its
// NRV3FrameEntry has no payload field), so it is checked through Open above.
func TestCanonicalOpenNRV3ReadsAssertionFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.nrv")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	counts := []int{1, 2, 1}
	var all [][]Bracket
	for i, n := range counts {
		b := fixtureBrackets(n)
		for k := range b {
			b[k].SubSecondUS += uint32(i * 100)
		}
		all = append(all, b)
		p := knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Tokens: []uint32{1, 2}, Brackets: uint16(n)}}
		if err := w.AppendAssertion(fmt.Sprintf("f%d", i), b, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := knirvnrv.OpenNRV3(path)
	if err != nil {
		t.Fatalf("canonical OpenNRV3 refused the file: %v", err)
	}
	if len(r.Registry.Frames) != len(counts) {
		t.Fatalf("frames = %d, want %d", len(r.Registry.Frames), len(counts))
	}
	for i, f := range r.Registry.Frames {
		if f.Count != counts[i] {
			t.Fatalf("frame %d count = %d, want %d", i, f.Count, counts[i])
		}
		brackets := r.BracketsForFrame(f)
		if len(brackets) != counts[i] {
			t.Fatalf("frame %d: canonical reader returned %d brackets, want %d", i, len(brackets), counts[i])
		}
		for k, cb := range brackets {
			want := all[i][k]
			if cb.SubSecondUS != want.SubSecondUS || cb.GoldenSeed != want.GoldenSeed || cb.Projections[0] != want.Projections[0] {
				t.Fatalf("frame %d bracket %d differs between writer and canonical reader", i, k)
			}
		}
	}
	if r.Registry.Frames[1].Offset != int64(counts[0])*BracketSize {
		t.Fatalf("frame 1 offset = %d, want %d", r.Registry.Frames[1].Offset, counts[0]*BracketSize)
	}
}
