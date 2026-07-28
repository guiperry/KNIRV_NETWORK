package nrvio

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var magic = [4]byte{'N', 'R', 'V', '2'}

type FrameEntry struct {
	ID            string                `json:"id"`
	TimestampUnix int64                 `json:"timestamp_unix"`
	Count         int                   `json:"count"`
	Offset        int64                 `json:"offset"`
	Linguistic    LinguisticMetadata    `json:"linguistic"`
	Thermo        ThermodynamicMetadata `json:"thermo"`
	Z3            Z3Metadata            `json:"z3"`
	BracketIndex  []BracketIndexEntry   `json:"bracket_index"`
}

type LinguisticMetadata struct {
	Tokens int    `json:"tokens"`
	Domain string `json:"domain,omitempty"`
}
type ThermodynamicMetadata struct {
	Temperature float64 `json:"temperature,omitempty"`
	Entropy     float64 `json:"entropy,omitempty"`
}
type Z3Metadata struct {
	Satisfiable     bool `json:"satisfiable,omitempty"`
	ConstraintCount int  `json:"constraint_count,omitempty"`
}
type BracketIndexEntry struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	XOR   []byte `json:"xor,omitempty"`
}
type Registry struct {
	Version string       `json:"version"`
	Frames  []FrameEntry `json:"frames"`
}

type Writer struct {
	file     *os.File
	registry Registry
	brackets []Bracket
	frames   []FrameEntry
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Writer{file: f, registry: Registry{Version: "2.0"}}, nil
}
func (w *Writer) Append(b Bracket) { w.brackets = append(w.brackets, b) }
func (w *Writer) Close() error {
	if w.file == nil {
		return nil
	}
	defer func() { w.file = nil }()
	if len(w.frames) == 0 {
		w.frames = append(w.frames, FrameEntry{ID: "frame-0", Count: len(w.brackets), Offset: 0})
	}
	w.registry.Frames = append(w.registry.Frames, w.frames...)
	meta, _ := json.Marshal(w.registry)
	if _, err := w.file.Write(magic[:]); err != nil {
		return err
	}
	if err := binary.Write(w.file, binary.LittleEndian, uint32(len(meta))); err != nil {
		return err
	}
	if _, err := w.file.Write(meta); err != nil {
		return err
	}
	for _, b := range w.brackets {
		raw := EncodeBracket(b)
		if _, err := w.file.Write(raw[:]); err != nil {
			return err
		}
	}
	return w.file.Close()
}

type Reader struct {
	registry Registry
	data     []byte
	pos      int
}

func Open(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	header := make([]byte, 8)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	if string(header[:4]) != "NRV2" {
		return nil, fmt.Errorf("invalid nrv magic")
	}
	n := binary.LittleEndian.Uint32(header[4:])
	meta := make([]byte, n)
	if _, err := io.ReadFull(f, meta); err != nil {
		return nil, err
	}
	var reg Registry
	if err := json.Unmarshal(meta, &reg); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}
	if len(data)%BracketSize != 0 {
		return nil, fmt.Errorf("invalid bracket section length %d", len(data))
	}
	return &Reader{registry: reg, data: data}, nil
}
func (r *Reader) Registry() Registry { return r.registry }
func (r *Reader) Next() (Bracket, bool) {
	if r.pos >= len(r.data) {
		return Bracket{}, false
	}
	var raw [BracketSize]byte
	copy(raw[:], r.data[r.pos:r.pos+BracketSize])
	r.pos += BracketSize
	return DecodeBracket(raw), true
}
