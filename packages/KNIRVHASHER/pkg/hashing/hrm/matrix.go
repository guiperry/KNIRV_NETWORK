package hrm

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
)

// matrixNameRE limits what a client can ask for to the d×d attention
// projections a uLoRA connector is derived from.
var matrixNameRE = regexp.MustCompile(`^[lh]\.[0-9]{1,2}\.attn\.[qkvo]\.weight$`)

// Matrix returns one 2-D f32 tensor of a kept model version as float64 rows
// (the shape KNIRVULORA's connector derivation takes). Only attention
// projections are served; the rest of the weights are not exposed this way.
func (s *Service) Matrix(version, name string) ([][]float64, error) {
	if !matrixNameRE.MatchString(name) {
		return nil, errors.New("tensor not available")
	}
	path, err := s.WeightsPath(version)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readF32Matrix(f, name)
}

func readF32Matrix(f io.ReaderAt, name string) ([][]float64, error) {
	var lenBuf [8]byte
	if _, err := f.ReadAt(lenBuf[:], 0); err != nil {
		return nil, fmt.Errorf("safetensors header: %w", err)
	}
	n := binary.LittleEndian.Uint64(lenBuf[:])
	if n == 0 || n > 64<<20 {
		return nil, errors.New("safetensors header size is implausible")
	}
	hdr := make([]byte, n)
	if _, err := f.ReadAt(hdr, 8); err != nil {
		return nil, fmt.Errorf("safetensors header: %w", err)
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(hdr, &entries); err != nil {
		return nil, fmt.Errorf("safetensors header: %w", err)
	}
	raw, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("tensor %q not in model", name)
	}
	var t struct {
		DType   string   `json:"dtype"`
		Shape   []int    `json:"shape"`
		Offsets []uint64 `json:"data_offsets"`
	}
	if err := json.Unmarshal(raw, &t); err != nil || len(t.Shape) != 2 || len(t.Offsets) != 2 || t.DType != "F32" {
		return nil, fmt.Errorf("tensor %q is not a 2-D F32 matrix", name)
	}
	rows, cols := t.Shape[0], t.Shape[1]
	size := t.Offsets[1] - t.Offsets[0]
	if rows <= 0 || cols <= 0 || uint64(rows)*uint64(cols)*4 != size {
		return nil, fmt.Errorf("tensor %q: shape %v does not match its data size", name, t.Shape)
	}
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, int64(8+n+t.Offsets[0])); err != nil {
		return nil, fmt.Errorf("tensor %q: %w", name, err)
	}
	out := make([][]float64, rows)
	for i := range out {
		out[i] = make([]float64, cols)
		for j := range out[i] {
			v := math.Float32frombits(binary.LittleEndian.Uint32(buf[(i*cols+j)*4:]))
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, fmt.Errorf("tensor %q contains a non-finite value", name)
			}
			out[i][j] = float64(v)
		}
	}
	return out, nil
}
