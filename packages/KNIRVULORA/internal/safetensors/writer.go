package safetensors

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// Write encodes F32 tensors into the safetensors format.
//
// Scope note: this currently exists to produce faithful fixtures for tests —
// in particular so the bind route can be verified against the format the
// compiler actually writes, rather than a hand-rolled stand-in. The compiler
// still merges weights through Python (merge_safetensors.py); if that path is
// ever moved into Go, this is the piece to build on, which is why it lives in
// the format package rather than a test file.
//
// Tensor names must be unique. An empty map is an error: a weights file with no
// tensors is a bug upstream, not a valid empty bundle.
func Write(tensors map[string][][]float32, metadata map[string]string) ([]byte, error) {
	if len(tensors) == 0 {
		return nil, fmt.Errorf("safetensors: refusing to write a file with no tensors")
	}

	// Sorted so the output is byte-stable for identical input, which matters for
	// content-addressed bundles.
	names := make([]string, 0, len(tensors))
	for name := range tensors {
		names = append(names, name)
	}
	sort.Strings(names)

	header := make(map[string]any, len(tensors)+1)
	if len(metadata) > 0 {
		header["__metadata__"] = metadata
	}

	var payload []byte
	for _, name := range names {
		matrix := tensors[name]
		rows := len(matrix)
		cols := 0
		if rows > 0 {
			cols = len(matrix[0])
		}

		start := len(payload)
		for r := 0; r < rows; r++ {
			if len(matrix[r]) != cols {
				return nil, fmt.Errorf("safetensors: tensor %q row %d has %d columns, want %d", name, r, len(matrix[r]), cols)
			}
			for c := 0; c < cols; c++ {
				var buf [4]byte
				binary.LittleEndian.PutUint32(buf[:], math.Float32bits(matrix[r][c]))
				payload = append(payload, buf[:]...)
			}
		}

		header[name] = headerEntry{
			Dtype:       "F32",
			Shape:       []int{rows, cols},
			DataOffsets: []int{start, len(payload)},
		}
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("safetensors: marshal header: %w", err)
	}

	out := make([]byte, 8, 8+len(headerJSON)+len(payload))
	binary.LittleEndian.PutUint64(out[:8], uint64(len(headerJSON)))
	out = append(out, headerJSON...)
	out = append(out, payload...)
	return out, nil
}
