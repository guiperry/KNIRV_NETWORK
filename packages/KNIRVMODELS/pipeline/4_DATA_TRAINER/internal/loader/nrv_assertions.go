package loader

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
)

// NRVAssertion is one self-contained frame of an encoder NRV3 artifact: how many
// brackets it holds and its decoded KNIRVBASE FramePayload. Payload is nil for a
// frame that carries none.
type NRVAssertion struct {
	ID       string
	Brackets int
	Payload  *knirvnrv.FramePayload
}

type nrvRegistryFrame struct {
	ID      string          `json:"id"`
	Count   int             `json:"count"`
	Offset  int64           `json:"offset"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// LoadNRVAssertions reads an encoder NRV3 file. Every frame's payload is decoded and
// validated by KNIRVBASE and must agree with its frame's bracket count. Any
// violation fails the whole file rather than yielding a partial training set.
func LoadNRVAssertions(path string) ([]NRVAssertion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read nrv artifact: %w", err)
	}
	return ParseNRVAssertions(data)
}

// ParseNRVAssertions is LoadNRVAssertions over bytes already in memory. The
// container (magic, registry, bracket section) comes from KNIRVBASE's canonical
// reader. Frames must tile the bracket section exactly: no overlap, no gap, no
// trailing bytes. The canonical registry has no payload field, so payloads are
// read from the same registry bytes.
func ParseNRVAssertions(data []byte) ([]NRVAssertion, error) {
	canon, err := knirvnrv.ParseNRV3(data)
	if err != nil {
		return nil, fmt.Errorf("nrv artifact: %w", err)
	}
	if err := checkFrameTiling(canon.Registry.Frames, int64(len(canon.Brackets))); err != nil {
		return nil, err
	}
	regLen := binary.LittleEndian.Uint32(data[4:8])
	var reg struct {
		Frames []nrvRegistryFrame `json:"frames"`
	}
	if err := json.Unmarshal(data[8:8+int(regLen)], &reg); err != nil {
		return nil, fmt.Errorf("nrv artifact: parse registry payloads: %w", err)
	}
	if len(reg.Frames) != len(canon.Registry.Frames) {
		return nil, fmt.Errorf("nrv artifact: registry payload view has %d frames, canonical view has %d", len(reg.Frames), len(canon.Registry.Frames))
	}
	out := make([]NRVAssertion, 0, len(reg.Frames))
	for i, f := range reg.Frames {
		if f.ID != canon.Registry.Frames[i].ID || f.Count != canon.Registry.Frames[i].Count {
			return nil, fmt.Errorf("nrv artifact: frame %d disagrees between registry views", i)
		}
		a := NRVAssertion{ID: f.ID, Brackets: f.Count}
		if len(f.Payload) > 0 {
			p, err := knirvnrv.DecodeFramePayload(f.Payload)
			if err != nil {
				return nil, fmt.Errorf("nrv artifact: frame %s payload: %w", f.ID, err)
			}
			if p.TokenWindow != nil && int(p.TokenWindow.Brackets) != f.Count {
				return nil, fmt.Errorf("nrv artifact: frame %s token_window.brackets=%d but frame has %d brackets", f.ID, p.TokenWindow.Brackets, f.Count)
			}
			a.Payload = &p
		}
		out = append(out, a)
	}
	return out, nil
}

// checkFrameTiling requires the frames to cover the bracket section exactly, in
// registry order: each frame starts where the previous one ended.
func checkFrameTiling(frames []knirvnrv.NRV3FrameEntry, section int64) error {
	var cursor int64
	for _, f := range frames {
		if f.Count < 0 {
			return fmt.Errorf("nrv artifact: frame %s has negative count %d", f.ID, f.Count)
		}
		if f.Offset != cursor {
			return fmt.Errorf("nrv artifact: frame %s at offset %d, want %d (overlapping or gapped frames)", f.ID, f.Offset, cursor)
		}
		cursor += int64(f.Count) * knirvnrv.BracketSize
		if cursor > section {
			return fmt.Errorf("nrv artifact: frame %s runs past the bracket section", f.ID)
		}
	}
	if cursor != section {
		return fmt.Errorf("nrv artifact: %d trailing bytes not covered by any frame", section-cursor)
	}
	return nil
}
