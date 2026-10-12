package nrv

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

const (
	NRV3Magic    = "NRV3"
	NRV2Magic    = "NRV2"
	NRV3MagicLen = 4
)

type NRV3LinguisticMetadata struct {
	Tokens int    `json:"tokens,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type NRV3ThermoMetadata struct {
	Temperature float64 `json:"temperature,omitempty"`
	Entropy     float64 `json:"entropy,omitempty"`
}

type NRV3Z3Metadata struct {
	Satisfiable     bool `json:"satisfiable,omitempty"`
	ConstraintCount int  `json:"constraint_count,omitempty"`
}

type NRV3BracketIndexEntry struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	XOR   []byte `json:"xor,omitempty"`
}

type NRV3FrameEntry struct {
	ID            string                  `json:"id"`
	TimestampUnix int64                   `json:"timestamp_unix"`
	Count         int                     `json:"count"`
	Offset        int64                   `json:"offset"`
	Linguistic    NRV3LinguisticMetadata  `json:"linguistic"`
	Thermo        NRV3ThermoMetadata      `json:"thermo"`
	Z3            NRV3Z3Metadata          `json:"z3"`
	BracketIndex  []NRV3BracketIndexEntry `json:"bracket_index"`
}

type NRV3Registry struct {
	Version string           `json:"version"`
	Frames  []NRV3FrameEntry `json:"frames"`
}

type NRV3Reader struct {
	Registry NRV3Registry
	Brackets []byte
}

func OpenNRV3(path string) (*NRV3Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseNRV3(data)
}

func ParseNRV3(data []byte) (*NRV3Reader, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("nrv3: file too short (%d bytes)", len(data))
	}
	magic := string(data[:4])
	if magic != NRV3Magic && magic != NRV2Magic {
		return nil, fmt.Errorf("nrv3: invalid magic %q, want %q or %q", magic, NRV3Magic, NRV2Magic)
	}
	if magic == NRV2Magic {
		return nil, fmt.Errorf("nrv3: legacy NRV2 layout is not safe to decode as canonical; migrate it through the seeder")
	}
	regLen := binary.LittleEndian.Uint32(data[4:8])
	if int(regLen)+8 > len(data) {
		return nil, fmt.Errorf("nrv3: registry length %d exceeds file size", regLen)
	}
	var reg NRV3Registry
	if err := json.Unmarshal(data[8:8+regLen], &reg); err != nil {
		return nil, fmt.Errorf("nrv3: parse registry: %w", err)
	}
	bracketData := data[8+int(regLen):]
	if len(bracketData)%BracketSize != 0 {
		return nil, fmt.Errorf("nrv3: bracket section length %d not a multiple of %d", len(bracketData), BracketSize)
	}
	return &NRV3Reader{Registry: reg, Brackets: bracketData}, nil
}

// RawBrackets returns the frame's full brackets (Count*BracketSize bytes), or
// nil if the registry points outside the bracket section.
func (r *NRV3Reader) RawBrackets(f NRV3FrameEntry) []byte {
	off, n := f.Offset, int64(f.Count)*BracketSize
	if off < 0 || f.Count < 0 || off+n > int64(len(r.Brackets)) {
		return nil
	}
	return r.Brackets[off : off+n]
}

func (r *NRV3Reader) BracketsForFrame(f NRV3FrameEntry) []*Bracket {
	off := f.Offset
	if off < 0 {
		off = 0
	}
	if f.Count < 0 || int(off)+f.Count*BracketSize > len(r.Brackets) {
		return nil
	}
	out := make([]*Bracket, f.Count)
	for i := 0; i < f.Count; i++ {
		var raw [BracketSize]byte
		copy(raw[:], r.Brackets[off+int64(i)*BracketSize:off+int64(i+1)*BracketSize])
		b := DecodeBracket(raw)
		out[i] = &b
	}
	return out
}

// ReadNRV3At parses an NRV3 container that starts at offset in r.
func ReadNRV3At(r io.ReaderAt, offset int64) (*NRV3Reader, error) {
	header := make([]byte, 8)
	if _, err := r.ReadAt(header, offset); err != nil {
		return nil, err
	}
	if magic := string(header[:4]); magic != NRV3Magic {
		return nil, fmt.Errorf("nrv3: invalid magic at offset %d: %q", offset, magic)
	}
	regLen := int64(binary.LittleEndian.Uint32(header[4:8]))
	regBuf := make([]byte, regLen)
	if _, err := r.ReadAt(regBuf, offset+8); err != nil {
		return nil, err
	}
	var reg NRV3Registry
	if err := json.Unmarshal(regBuf, &reg); err != nil {
		return nil, fmt.Errorf("nrv3: parse registry: %w", err)
	}
	// Read to the end of the source. (A value-receiver reader that never
	// advanced its offset used to make this loop forever.)
	remaining, err := io.ReadAll(io.NewSectionReader(r, offset+8+regLen, math.MaxInt64-(offset+8+regLen)))
	if err != nil {
		return nil, err
	}
	if len(remaining)%BracketSize != 0 {
		return nil, fmt.Errorf("nrv3: bracket section length %d not a multiple of %d", len(remaining), BracketSize)
	}
	return &NRV3Reader{Registry: reg, Brackets: remaining}, nil
}

type NRV3Writer struct {
	registry NRV3Registry
	brackets []byte
}

func NewNRV3Writer() *NRV3Writer {
	return &NRV3Writer{
		registry: NRV3Registry{Version: "3.0"},
	}
}

func (w *NRV3Writer) AddFrame(f NRV3FrameEntry, bracketBuf []byte) {
	w.registry.Frames = append(w.registry.Frames, f)
	w.brackets = append(w.brackets, bracketBuf...)
}

func (w *NRV3Writer) Bytes() ([]byte, error) {
	regBytes, err := json.Marshal(w.registry)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, NRV3MagicLen+4+len(regBytes)+len(w.brackets))
	out = append(out, []byte(NRV3Magic)...)
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(regBytes)))
	out = append(out, lenBuf[:]...)
	out = append(out, regBytes...)
	out = append(out, w.brackets...)
	return out, nil
}

func WriteNRV3(path string, w *NRV3Writer) error {
	b, err := w.Bytes()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
