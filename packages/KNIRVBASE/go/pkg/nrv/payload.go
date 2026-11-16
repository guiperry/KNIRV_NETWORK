package nrv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// PayloadModality is the key under which a frame's FramePayload is indexed in
// FrameEntry.Modalities (Rust and TS use the same key). The ModalityIndex it maps
// to is the byte range of the encoded payload inside the frame.
const PayloadModality = "payload"

// FramePayload is the uniform open payload every .nrv frame may carry. Producers
// fill the fields they have; downstream stages enrich the rest on the same schema.
//
// rust/src/payload.rs is the source of truth. Widths, range checks and the
// absent-means-empty rule mirror it exactly; testdata/framepayload/cases.json pins
// them across Rust, Go and TS.
type FramePayload struct {
	// 1. Token & assertion window (1-to-1 with the frame's 80-byte brackets).
	TokenWindow *TokenWindowPayload `json:"token_window,omitempty"`
	// 2. Deterministic test case from an ErrorNode's test suite.
	ErrorTestCase *ErrorTestCasePayload `json:"error_test_case,omitempty"`
	// 3. SFT / DPO preference sample.
	TRLSample *TRLSamplePayload `json:"trl_sample,omitempty"`
	// 4. Closed-set decision and supervision episode.
	SupervisionEpisode *SupervisionEpisodePayload `json:"supervision_episode,omitempty"`
	// 5. Visual / multimodal grounding.
	Perceptual *PerceptualPayload `json:"perceptual,omitempty"`
}

// TokenWindowPayload. Widths: tokens u32 (vocabulary ids), start u32 (offset in the
// frame's token stream), brackets u16 (<= MaxBracketsPerFrame).
type TokenWindowPayload struct {
	Tokens   []uint32 `json:"tokens,omitempty"`
	Start    uint32   `json:"start"`
	Brackets uint16   `json:"brackets"`
}

type ErrorTestCasePayload struct {
	ID       string `json:"id,omitempty"`
	Input    string `json:"input,omitempty"`
	Expected string `json:"expected,omitempty"`
}

type TRLSamplePayload struct {
	Prompt     string `json:"prompt,omitempty"`
	Completion string `json:"completion,omitempty"`
	Chosen     string `json:"chosen,omitempty"`
	Rejected   string `json:"rejected,omitempty"`
}

// MaxToolTier is the highest tool tier. The tier set is 0..=3: 0 execute, 1 ask,
// 2 deny, 3 escalate (owner-decided; same as rust/src/payload.rs MAX_TOOL_TIER).
const MaxToolTier = 3

// BoxEpsilon is the tolerance for x+w <= 1 and y+h <= 1 on normalised boxes.
const BoxEpsilon = 1e-9

// ToolCallPayload. Tier is u8 on the wire width; the valid set is 0..=MaxToolTier.
type ToolCallPayload struct {
	Tool    string `json:"tool,omitempty"`
	Tier    uint8  `json:"tier"`
	Outcome string `json:"outcome,omitempty"`
}

type SupervisionEpisodePayload struct {
	State     string            `json:"state,omitempty"`
	Choices   []string          `json:"choices,omitempty"`
	Decision  string            `json:"decision,omitempty"`
	ToolTrace []ToolCallPayload `json:"tool_trace,omitempty"`
	Outcome   string            `json:"outcome,omitempty"`
}

// BoxPayload is a bounding box in normalised frame coordinates: each of X, Y, W,
// H lies in [0,1] (top-left origin; W and H are extents), and the box stays inside
// the frame, X+W <= 1 and Y+H <= 1 (within BoxEpsilon).
type BoxPayload struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Label string  `json:"label,omitempty"`
}

type PerceptualPayload struct {
	ImageHashes []string     `json:"image_hashes,omitempty"`
	Boxes       []BoxPayload `json:"boxes,omitempty"`
	OCR         string       `json:"ocr,omitempty"`
	Embedding   []float32    `json:"embedding,omitempty"`
}

// PayloadError reports a decode, validation or range failure with the field path.
type PayloadError struct {
	Field  string
	Detail string
}

func (e *PayloadError) Error() string {
	if e.Field == "" {
		return "nrv payload: " + e.Detail
	}
	return fmt.Sprintf("nrv payload field %s: %s", e.Field, e.Detail)
}

func (t TokenWindowPayload) empty() bool {
	return len(t.Tokens) == 0 && t.Start == 0 && t.Brackets == 0
}

func (e ErrorTestCasePayload) empty() bool {
	return e.ID == "" && e.Input == "" && e.Expected == ""
}

func (t TRLSamplePayload) empty() bool {
	return t.Prompt == "" && t.Completion == "" && t.Chosen == "" && t.Rejected == ""
}

func (s SupervisionEpisodePayload) empty() bool {
	return s.State == "" && len(s.Choices) == 0 && s.Decision == "" && len(s.ToolTrace) == 0 && s.Outcome == ""
}

func (p PerceptualPayload) empty() bool {
	return len(p.ImageHashes) == 0 && len(p.Boxes) == 0 && p.OCR == "" && len(p.Embedding) == 0
}

// PublicView returns the payload as a test-taker may see it: the expected answer
// of an ErrorTestCase is removed. The receiver is not modified.
func (p FramePayload) PublicView() FramePayload {
	if p.ErrorTestCase != nil {
		c := *p.ErrorTestCase
		c.Expected = ""
		p.ErrorTestCase = &c
	}
	return p
}

// Normalized returns a copy in which every empty optional sub-payload is nil, so an
// empty object and an absent one encode and compare the same.
func (p FramePayload) Normalized() FramePayload {
	if p.TokenWindow != nil && p.TokenWindow.empty() {
		p.TokenWindow = nil
	}
	if p.ErrorTestCase != nil && p.ErrorTestCase.empty() {
		p.ErrorTestCase = nil
	}
	if p.TRLSample != nil && p.TRLSample.empty() {
		p.TRLSample = nil
	}
	if p.SupervisionEpisode != nil && p.SupervisionEpisode.empty() {
		p.SupervisionEpisode = nil
	}
	if p.Perceptual != nil && p.Perceptual.empty() {
		p.Perceptual = nil
	}
	return p
}

// Validate checks the range rules that the integer widths do not already enforce.
// It is run on the normalized payload.
func (p FramePayload) Validate() error {
	if tw := p.TokenWindow; tw != nil && tw.Brackets > MaxBracketsPerFrame {
		return &PayloadError{Field: "token_window.brackets",
			Detail: fmt.Sprintf("%d exceeds the per-frame cap of %d", tw.Brackets, MaxBracketsPerFrame)}
	}
	if se := p.SupervisionEpisode; se != nil {
		for i, t := range se.ToolTrace {
			if t.Tier > MaxToolTier {
				return &PayloadError{Field: fmt.Sprintf("supervision_episode.tool_trace[%d].tier", i),
					Detail: fmt.Sprintf("%d is outside the tool tier set 0..=%d", t.Tier, MaxToolTier)}
			}
		}
	}
	if pp := p.Perceptual; pp != nil {
		for i, b := range pp.Boxes {
			for _, c := range []struct {
				name string
				v    float64
			}{{"x", b.X}, {"y", b.Y}, {"w", b.W}, {"h", b.H}} {
				if math.IsNaN(c.v) || math.IsInf(c.v, 0) {
					return &PayloadError{Field: fmt.Sprintf("perceptual.boxes[%d].%s", i, c.name),
						Detail: fmt.Sprintf("%v is not finite", c.v)}
				}
				if c.v < 0 || c.v > 1 {
					return &PayloadError{Field: fmt.Sprintf("perceptual.boxes[%d].%s", i, c.name),
						Detail: fmt.Sprintf("%v is outside the normalised range 0..=1", c.v)}
				}
			}
			if b.X+b.W > 1+BoxEpsilon {
				return &PayloadError{Field: fmt.Sprintf("perceptual.boxes[%d].x", i),
					Detail: fmt.Sprintf("x + w = %v extends past the frame (1)", b.X+b.W)}
			}
			if b.Y+b.H > 1+BoxEpsilon {
				return &PayloadError{Field: fmt.Sprintf("perceptual.boxes[%d].y", i),
					Detail: fmt.Sprintf("y + h = %v extends past the frame (1)", b.Y+b.H)}
			}
		}
		for i, e := range pp.Embedding {
			if math.IsNaN(float64(e)) || math.IsInf(float64(e), 0) {
				return &PayloadError{Field: fmt.Sprintf("perceptual.embedding[%d]", i), Detail: fmt.Sprintf("%v is not finite", e)}
			}
		}
	}
	return nil
}

// EncodeFramePayload normalizes, validates and serialises a payload. Invalid payloads
// are refused rather than written.
func EncodeFramePayload(p FramePayload) ([]byte, error) {
	n := p.Normalized()
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n)
}

// DecodeFramePayload parses payload bytes read from a frame, then normalizes and
// validates them. The bytes must be a JSON object, as in the Rust core.
func DecodeFramePayload(b []byte) (FramePayload, error) {
	var p FramePayload
	trimmed := bytes.TrimLeft(b, " \t\r\n")
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return FramePayload{}, errors.New("nrv payload: payload must be a JSON object")
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return FramePayload{}, err
	}
	n := p.Normalized()
	if err := n.Validate(); err != nil {
		return FramePayload{}, err
	}
	return n, nil
}
