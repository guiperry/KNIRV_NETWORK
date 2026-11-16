package nrv

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// sharedCases is the cross-language fixture read by Rust, Go and TS.
const sharedCases = "../../../testdata/framepayload/cases.json"

func fullPayload() FramePayload {
	return FramePayload{
		TokenWindow:        &TokenWindowPayload{Tokens: []uint32{1, 2, 3}, Start: 0, Brackets: 2},
		ErrorTestCase:      &ErrorTestCasePayload{ID: "t1", Input: "2+2", Expected: "4"},
		TRLSample:          &TRLSamplePayload{Prompt: "p", Chosen: "good", Rejected: "bad"},
		SupervisionEpisode: &SupervisionEpisodePayload{State: "node healthy", Choices: []string{"no-op", "drain"}, Decision: "no-op", ToolTrace: []ToolCallPayload{{Tool: "manifest.get", Tier: 0, Outcome: "execute"}}, Outcome: "ok"},
		Perceptual:         &PerceptualPayload{ImageHashes: []string{"ab"}, Boxes: []BoxPayload{{X: 0.1, Y: 0.2, W: 0.3, H: 0.4, Label: "cat"}}, OCR: "hi", Embedding: []float32{0.5}},
	}
}

func TestFramePayloadRoundTripAllFiveFields(t *testing.T) {
	in := fullPayload()
	raw, err := EncodeFramePayload(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeFramePayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(in)
	b, _ := json.Marshal(out)
	if string(a) != string(b) {
		t.Fatalf("round trip changed payload:\n%s\n%s", a, b)
	}
	if out.TokenWindow == nil || out.ErrorTestCase == nil || out.TRLSample == nil || out.SupervisionEpisode == nil || out.Perceptual == nil {
		t.Fatal("one of the five payload fields was lost")
	}
}

func TestPublicViewStripsExpectedOnly(t *testing.T) {
	in := fullPayload()
	pv := in.PublicView()
	if pv.ErrorTestCase.Expected != "" {
		t.Fatalf("PublicView leaked expected answer %q", pv.ErrorTestCase.Expected)
	}
	if in.ErrorTestCase.Expected != "4" {
		t.Fatal("PublicView must not mutate the receiver")
	}
	if pv.ErrorTestCase.Input != "2+2" || pv.TRLSample == nil {
		t.Fatal("PublicView removed more than the expected answer")
	}
}

func TestEmptyPayloadOmitsAllFields(t *testing.T) {
	raw, _ := EncodeFramePayload(FramePayload{})
	if string(raw) != "{}" {
		t.Fatalf("empty payload encoded as %s", raw)
	}
}

func TestTokenWidthBounds(t *testing.T) {
	ok := FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{0, 4294967295}, Start: 4294967295, Brackets: MaxBracketsPerFrame}}
	if _, err := EncodeFramePayload(ok); err != nil {
		t.Fatalf("max-edge values rejected: %v", err)
	}
	for name, raw := range map[string]string{
		"token_negative":  `{"token_window":{"tokens":[-1],"start":0,"brackets":0}}`,
		"token_max_plus1": `{"token_window":{"tokens":[4294967296],"start":0,"brackets":0}}`,
		"start_negative":  `{"token_window":{"tokens":[1],"start":-1,"brackets":0}}`,
		"start_max_plus1": `{"token_window":{"tokens":[1],"start":4294967296,"brackets":0}}`,
		"brackets_neg":    `{"token_window":{"tokens":[1],"start":0,"brackets":-1}}`,
	} {
		if _, err := DecodeFramePayload([]byte(raw)); err == nil {
			t.Errorf("%s: expected decode error", name)
		}
	}
}

func TestBracketsFrameCap(t *testing.T) {
	over := FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1}, Brackets: MaxBracketsPerFrame + 1}}
	err := func() error { _, e := EncodeFramePayload(over); return e }()
	if pe, ok := err.(*PayloadError); !ok || pe.Field != "token_window.brackets" {
		t.Fatalf("expected named brackets range error, got %v", err)
	}
}

func TestToolTierIs0To3(t *testing.T) {
	for tier := 0; tier <= MaxToolTier; tier++ {
		raw := fmt.Sprintf(`{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":%d,"outcome":"o"}]}}`, tier)
		if _, err := DecodeFramePayload([]byte(raw)); err != nil {
			t.Fatalf("tier %d rejected: %v", tier, err)
		}
	}
	for _, raw := range []string{
		`{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":4,"outcome":"o"}]}}`,
		`{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":255,"outcome":"o"}]}}`,
		`{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":256,"outcome":"o"}]}}`,
		`{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":-1,"outcome":"o"}]}}`,
	} {
		if _, err := DecodeFramePayload([]byte(raw)); err == nil {
			t.Errorf("expected tier error for %s", raw)
		}
	}
}

func TestBoxSizeAndFiniteness(t *testing.T) {
	cases := []struct {
		name string
		box  BoxPayload
		ok   bool
	}{
		{"zero_size", BoxPayload{W: 0, H: 0}, true},
		{"whole_frame", BoxPayload{X: 0, Y: 0, W: 1, H: 1}, true},
		{"flush_to_far_edge", BoxPayload{X: 0.25, Y: 0.25, W: 0.75, H: 0.75}, true},
		{"origin_at_far_corner", BoxPayload{X: 1, Y: 1}, true},
		{"negative_origin", BoxPayload{X: -2.5}, false},
		{"negative_y", BoxPayload{Y: -0.1}, false},
		{"origin_past_frame", BoxPayload{X: 1.5}, false},
		{"x_plus_w_past_frame", BoxPayload{X: 0.6, W: 0.5}, false},
		{"y_plus_h_past_frame", BoxPayload{Y: 0.6, H: 0.5}, false},
		{"x_plus_w_exact_edge", BoxPayload{X: 0.5, W: 0.5}, true},
		{"y_plus_h_exact_edge", BoxPayload{Y: 0.5, H: 0.5}, true},
		{"x_plus_w_within_epsilon", BoxPayload{X: 0.5, W: 0.5 + BoxEpsilon/2}, true},
		{"x_plus_w_just_over", BoxPayload{X: 0.5, W: 0.500001}, false},
		{"y_plus_h_just_over", BoxPayload{Y: 0.5, H: 0.500001}, false},
		{"w_past_one", BoxPayload{W: 1.5}, false},
		{"negative_w", BoxPayload{W: -0.5}, false},
		{"negative_h", BoxPayload{H: -1}, false},
		{"nan_x", BoxPayload{X: nan()}, false},
		{"inf_h", BoxPayload{H: inf()}, false},
	}
	for _, c := range cases {
		p := FramePayload{Perceptual: &PerceptualPayload{Boxes: []BoxPayload{c.box}}}
		_, err := EncodeFramePayload(p)
		if (err == nil) != c.ok {
			t.Errorf("%s: ok=%v err=%v", c.name, c.ok, err)
		}
	}
}

func TestEmptyMeansAbsent(t *testing.T) {
	p := FramePayload{
		TokenWindow:        &TokenWindowPayload{},
		ErrorTestCase:      &ErrorTestCasePayload{},
		TRLSample:          &TRLSamplePayload{},
		SupervisionEpisode: &SupervisionEpisodePayload{},
		Perceptual:         &PerceptualPayload{},
	}
	raw, err := EncodeFramePayload(p)
	if err != nil || string(raw) != "{}" {
		t.Fatalf("empty sub-payloads should be absent, got %s (%v)", raw, err)
	}
	if _, err := DecodeFramePayload([]byte(`{"perceptual":{"boxes":[]}}`)); err != nil {
		t.Fatal(err)
	}
}

func TestNonObjectTopLevelRejected(t *testing.T) {
	for _, raw := range []string{"null", "[]", "1"} {
		if _, err := DecodeFramePayload([]byte(raw)); err == nil {
			t.Errorf("%s: expected error", raw)
		}
	}
}

// jsonEqual compares parsed JSON so 1 and 1.0 match, as the Rust helper does.
func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}

func TestSharedConformanceFixture(t *testing.T) {
	data, err := os.ReadFile(sharedCases)
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
	if len(doc.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}
	for _, c := range doc.Cases {
		p, err := DecodeFramePayload(c.Input)
		switch c.Expect {
		case "error":
			if err == nil {
				t.Errorf("%s: expected error", c.Name)
			}
		case "ok":
			if err != nil {
				t.Errorf("%s: decode failed: %v", c.Name, err)
				continue
			}
			enc, err := EncodeFramePayload(p)
			if err != nil {
				t.Errorf("%s: encode failed: %v", c.Name, err)
				continue
			}
			if !jsonEqual(t, enc, c.Canonical) {
				t.Errorf("%s: encoded %s != canonical %s", c.Name, enc, c.Canonical)
			}
		default:
			t.Errorf("%s: unknown expect %q", c.Name, c.Expect)
		}
	}
}

func nan() float64 { z := 0.0; return z / z }
func inf() float64 { z := 0.0; return 1 / z }
