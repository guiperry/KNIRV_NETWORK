// Package safetensors reads the safetensors tensor format.
//
// The format is a little-endian uint64 header length, that many bytes of UTF-8
// JSON describing each tensor (dtype, shape, and a [begin,end) byte range), then
// a data buffer the ranges index into.
//
// This exists because the uLoRA compiler merges bundle weights with
// safetensors.numpy.save_file (internal/compiler/engines/merge_safetensors.py),
// so a bundle's core_weights.safetensors is real binary safetensors — not JSON.
// The binder previously attempted json.Unmarshal on it, which cannot work on a
// binary file, so no compiler-produced bundle could be bound at all.
//
// Implemented in Go rather than by shelling out to Python so the bind path stays
// usable per request without a Python interpreter at inference time.
package safetensors

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// dtypeSizes maps a safetensors dtype string to its element size in bytes.
var dtypeSizes = map[string]int{
	"F64": 8, "F32": 4, "F16": 2, "BF16": 2,
	"I64": 8, "I32": 4, "I16": 2, "I8": 1,
	"U64": 8, "U32": 4, "U16": 2, "U8": 1,
	"BOOL": 1, "F8_E4M3": 1, "F8_E5M2": 1,
}

// Tensor is one tensor's description plus a view of its bytes.
type Tensor struct {
	Dtype string
	Shape []int

	data []byte
}

// Len reports the number of elements.
func (t Tensor) Len() int {
	n := 1
	for _, d := range t.Shape {
		n *= d
	}
	return n
}

// File is a parsed safetensors file.
type File struct {
	Metadata map[string]string
	tensors  map[string]Tensor
	data     []byte
}

// headerEntry is the per-tensor JSON object. "__metadata__" is a string map
// rather than a tensor, so it is handled separately during parsing.
type headerEntry struct {
	Dtype       string `json:"dtype"`
	Shape       []int  `json:"shape"`
	DataOffsets []int  `json:"data_offsets"`
}

// Parse decodes a safetensors buffer.
//
// It validates aggressively: a truncated header, a tensor whose offsets fall
// outside the data buffer, or a shape that disagrees with its byte range are all
// errors. Reading garbage as weights would silently produce a wrong adapter,
// which is worse than failing.
func Parse(raw []byte) (*File, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("safetensors: file is %d bytes, shorter than the 8-byte header length", len(raw))
	}
	headerLen := binary.LittleEndian.Uint64(raw[:8])
	if headerLen == 0 {
		return nil, fmt.Errorf("safetensors: header length is zero")
	}
	headerEnd := 8 + int(headerLen)
	if headerLen > uint64(len(raw)) || headerEnd > len(raw) {
		return nil, fmt.Errorf("safetensors: header length %d exceeds file size %d", headerLen, len(raw))
	}

	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw[8:headerEnd], &entries); err != nil {
		return nil, fmt.Errorf("safetensors: parse header: %w", err)
	}

	file := &File{
		tensors: make(map[string]Tensor, len(entries)),
		data:    raw[headerEnd:],
	}

	if metaRaw, ok := entries["__metadata__"]; ok {
		if err := json.Unmarshal(metaRaw, &file.Metadata); err != nil {
			return nil, fmt.Errorf("safetensors: parse __metadata__: %w", err)
		}
		delete(entries, "__metadata__")
	}

	for name, entryRaw := range entries {
		var entry headerEntry
		if err := json.Unmarshal(entryRaw, &entry); err != nil {
			return nil, fmt.Errorf("safetensors: parse tensor %q: %w", name, err)
		}
		if len(entry.DataOffsets) != 2 {
			return nil, fmt.Errorf("safetensors: tensor %q has %d data_offsets, want 2", name, len(entry.DataOffsets))
		}
		elemSize, ok := dtypeSizes[entry.Dtype]
		if !ok {
			return nil, fmt.Errorf("safetensors: tensor %q has unsupported dtype %q", name, entry.Dtype)
		}

		begin, end := entry.DataOffsets[0], entry.DataOffsets[1]
		if begin < 0 || end < begin || end > len(file.data) {
			return nil, fmt.Errorf("safetensors: tensor %q offsets [%d,%d) outside the %d-byte data buffer",
				name, begin, end, len(file.data))
		}

		tensor := Tensor{Dtype: entry.Dtype, Shape: entry.Shape, data: file.data[begin:end]}

		// A shape whose element count disagrees with the byte range means the
		// header and the payload do not describe the same tensor.
		if expected := tensor.Len() * elemSize; expected != len(tensor.data) {
			return nil, fmt.Errorf("safetensors: tensor %q shape %v with dtype %s implies %d bytes but its range is %d",
				name, entry.Shape, entry.Dtype, expected, len(tensor.data))
		}

		file.tensors[name] = tensor
	}

	return file, nil
}

// Names returns the tensor names in sorted order, so iteration is deterministic.
func (f *File) Names() []string {
	names := make([]string, 0, len(f.tensors))
	for name := range f.tensors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Tensor looks a tensor up by name. Parsing a shape of [0] means absent.
func (f *File) Tensor(name string) (Tensor, bool) {
	t, ok := f.tensors[name]
	return t, ok
}

// Has reports whether a tensor is present.
func (f *File) Has(name string) bool {
	_, ok := f.tensors[name]
	return ok
}

// Float32Matrix returns a 2-D tensor as a row-major matrix.
//
// F32, F16 and BF16 are supported; other dtypes are rejected rather than
// reinterpreted. A non-2-D tensor is an error, because callers treat the result
// as a LoRA A/B matrix.
func (t Tensor) Float32Matrix() ([][]float32, error) {
	if len(t.Shape) != 2 {
		return nil, fmt.Errorf("safetensors: tensor has %d dimensions, want 2 (shape %v)", len(t.Shape), t.Shape)
	}
	rows, cols := t.Shape[0], t.Shape[1]

	matrix := make([][]float32, rows)
	for r := range matrix {
		matrix[r] = make([]float32, cols)
	}

	switch t.Dtype {
	case "F32":
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				off := (r*cols + c) * 4
				matrix[r][c] = math.Float32frombits(binary.LittleEndian.Uint32(t.data[off : off+4]))
			}
		}
	case "F16":
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				off := (r*cols + c) * 2
				matrix[r][c] = float16ToFloat32(binary.LittleEndian.Uint16(t.data[off : off+2]))
			}
		}
	case "BF16":
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				off := (r*cols + c) * 2
				// bfloat16 is the high 16 bits of a float32.
				matrix[r][c] = math.Float32frombits(uint32(binary.LittleEndian.Uint16(t.data[off:off+2])) << 16)
			}
		}
	default:
		return nil, fmt.Errorf("safetensors: dtype %q is not a float type this reader converts", t.Dtype)
	}

	return matrix, nil
}

// float16ToFloat32 converts an IEEE 754 half to a float32.
func float16ToFloat32(h uint16) float32 {
	sign := uint32(h>>15) & 0x1
	exp := uint32(h>>10) & 0x1F
	mant := uint32(h) & 0x3FF

	switch {
	case exp == 0 && mant == 0:
		// Zero (preserving the sign).
		return math.Float32frombits(sign << 31)
	case exp == 0x1F:
		// Inf or NaN.
		return math.Float32frombits(sign<<31 | 0xFF<<23 | mant<<13)
	case exp == 0:
		// Subnormal: renormalise into a float32 exponent.
		for mant&0x400 == 0 {
			mant <<= 1
			exp--
		}
		mant &= 0x3FF
		exp++
		return math.Float32frombits(sign<<31 | (exp+127-15)<<23 | mant<<13)
	default:
		return math.Float32frombits(sign<<31 | (exp+127-15)<<23 | mant<<13)
	}
}
