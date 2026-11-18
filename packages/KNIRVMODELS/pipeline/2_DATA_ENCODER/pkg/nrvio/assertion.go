package nrvio

import (
	"errors"
	"fmt"
	"math"
	"time"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
)

// MaxBracketsPerAssertion is the most brackets one assertion frame may hold. It
// mirrors KNIRVBASE's per-frame cap (pkg/nrv MaxBracketsPerFrame); the payload
// validator refuses a token window that exceeds it.
const MaxBracketsPerAssertion = 1024

// Assertion is one self-contained frame of an NRV file: its brackets plus the
// uniform FramePayload defined by KNIRVBASE (pkg/nrv/payload.go). Payload is nil
// for frames that carry none.
type Assertion struct {
	ID       string
	Brackets []Bracket
	Payload  *knirvnrv.FramePayload
}

// AssertionPayload builds the token-window payload for one encoder frame. span is
// the fact's token ids, windowStart is the frame's offset in the token stream and
// brackets is the frame's bracket count. Encoding and range checks are left to
// knirvbase, which refuses an invalid payload.
func AssertionPayload(span []int32, windowStart int32, brackets int) (knirvnrv.FramePayload, error) {
	if windowStart < 0 {
		return knirvnrv.FramePayload{}, fmt.Errorf("nrvio: window start %d is negative", windowStart)
	}
	if brackets < 0 || brackets > math.MaxUint16 {
		return knirvnrv.FramePayload{}, fmt.Errorf("nrvio: bracket count %d is outside u16", brackets)
	}
	var tokens []uint32
	if len(span) > 0 {
		tokens = make([]uint32, len(span))
	}
	for i, id := range span {
		if id < 0 {
			return knirvnrv.FramePayload{}, fmt.Errorf("nrvio: token id %d at span[%d] is negative", id, i)
		}
		tokens[i] = uint32(id)
	}
	return knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{
		Tokens:   tokens,
		Start:    uint32(windowStart),
		Brackets: uint16(brackets),
	}}, nil
}

// AppendAssertion writes brackets as one self-contained frame that carries p.
// The bracket section stays a plain run of 80-byte brackets; the frame's Offset is
// a byte offset into it, as in the canonical NRV3 reader. p is normalised and
// validated by knirvbase before anything is recorded. When p has a token window,
// its bracket count must equal len(brackets). Every bracket is an I-bracket, so no
// frame depends on another.
func (w *Writer) AppendAssertion(id string, brackets []Bracket, p knirvnrv.FramePayload) error {
	if w.file == nil {
		return errors.New("nrvio: writer is closed")
	}
	n := p.Normalized()
	if n.TokenWindow != nil && int(n.TokenWindow.Brackets) != len(brackets) {
		return fmt.Errorf("nrvio: token_window.brackets=%d but frame %s has %d brackets", n.TokenWindow.Brackets, id, len(brackets))
	}
	raw, err := knirvnrv.EncodeFramePayload(p)
	if err != nil {
		return fmt.Errorf("nrvio: frame %s: %w", id, err)
	}
	tokens := 0
	if n.TokenWindow != nil {
		tokens = len(n.TokenWindow.Tokens)
	}
	index := make([]BracketIndexEntry, len(brackets))
	for i := range index {
		index[i] = BracketIndexEntry{Index: i, Type: "I"}
	}
	domain := domainForBracket(Bracket{})
	if len(brackets) > 0 {
		domain = domainForBracket(brackets[0])
	}
	w.frames = append(w.frames, FrameEntry{
		ID:            id,
		TimestampUnix: time.Now().Unix(),
		Count:         len(brackets),
		Offset:        int64(len(w.brackets)) * BracketSize,
		Linguistic:    LinguisticMetadata{Tokens: tokens, Domain: domain},
		BracketIndex:  index,
		Payload:       raw,
	})
	w.brackets = append(w.brackets, brackets...)
	return nil
}

// Assertions reads every frame of an opened file. Frames must tile the bracket
// section exactly, and any payload must decode and validate and agree with the
// frame's bracket count.
func (r *Reader) Assertions() ([]Assertion, error) {
	out := make([]Assertion, 0, len(r.registry.Frames))
	var cursor int64
	for _, f := range r.registry.Frames {
		if f.Count < 0 || len(f.BracketIndex) != f.Count {
			return nil, fmt.Errorf("nrvio: frame %s has count %d with %d index entries", f.ID, f.Count, len(f.BracketIndex))
		}
		if f.Offset != cursor {
			return nil, fmt.Errorf("nrvio: frame %s offset %d, want %d", f.ID, f.Offset, cursor)
		}
		size := int64(f.Count) * BracketSize
		if cursor+size > int64(len(r.data)) {
			return nil, fmt.Errorf("nrvio: frame %s runs past the bracket section", f.ID)
		}
		brackets := make([]Bracket, f.Count)
		for i := range brackets {
			var raw [BracketSize]byte
			copy(raw[:], r.data[cursor+int64(i)*BracketSize:])
			brackets[i] = DecodeBracket(raw)
		}
		cursor += size
		a := Assertion{ID: f.ID, Brackets: brackets}
		if len(f.Payload) > 0 {
			p, err := knirvnrv.DecodeFramePayload(f.Payload)
			if err != nil {
				return nil, fmt.Errorf("nrvio: frame %s payload: %w", f.ID, err)
			}
			if p.TokenWindow != nil && int(p.TokenWindow.Brackets) != f.Count {
				return nil, fmt.Errorf("nrvio: frame %s token_window.brackets=%d, frame has %d", f.ID, p.TokenWindow.Brackets, f.Count)
			}
			a.Payload = &p
		}
		out = append(out, a)
	}
	if cursor != int64(len(r.data)) {
		return nil, fmt.Errorf("nrvio: %d trailing bytes not covered by any frame", int64(len(r.data))-cursor)
	}
	return out, nil
}
