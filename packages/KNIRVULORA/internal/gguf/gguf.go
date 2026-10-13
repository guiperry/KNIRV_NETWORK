// Package gguf reads GGUF model files and writes GGUF LoRA adapters.
//
// Reading covers the header, metadata and tensor table, plus tensor data in the
// dense and simple block-quantised formats (F32, F16, BF16, Q4_0, Q4_1, Q5_0,
// Q5_1, Q8_0). K-quants (Q2_K..Q6_K) are not dequantised and are refused by name:
// a connector must be derived from real weights, and a wrong dequantisation would
// produce a plausible but meaningless basis.
//
// The format is the one documented in llama.cpp's gguf spec (version 3).
package gguf

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const magic = "GGUF"

// Metadata value types.
const (
	tUint8   = 0
	tInt8    = 1
	tUint16  = 2
	tInt16   = 3
	tUint32  = 4
	tInt32   = 5
	tFloat32 = 6
	tBool    = 7
	tString  = 8
	tArray   = 9
	tUint64  = 10
	tInt64   = 11
	tFloat64 = 12
)

// GGML tensor types (the ones this package knows by name).
type TensorType uint32

const (
	F32  TensorType = 0
	F16  TensorType = 1
	Q4_0 TensorType = 2
	Q4_1 TensorType = 3
	Q5_0 TensorType = 6
	Q5_1 TensorType = 7
	Q8_0 TensorType = 8
	BF16 TensorType = 30
)

var typeNames = map[TensorType]string{
	0: "F32", 1: "F16", 2: "Q4_0", 3: "Q4_1", 6: "Q5_0", 7: "Q5_1", 8: "Q8_0",
	9: "Q8_1", 10: "Q2_K", 11: "Q3_K", 12: "Q4_K", 13: "Q5_K", 14: "Q6_K", 15: "Q8_K", 30: "BF16",
}

func (t TensorType) String() string {
	if n, ok := typeNames[t]; ok {
		return n
	}
	return fmt.Sprintf("type-%d", uint32(t))
}

// Tensor is one entry of the tensor table. Dims are in GGUF order: Dims[0] is the
// fastest-varying (the input width of a weight matrix).
type Tensor struct {
	Name   string
	Dims   []uint64
	Type   TensorType
	Offset uint64 // relative to the start of the data section
}

// Elements is the number of scalar elements.
func (t Tensor) Elements() uint64 {
	n := uint64(1)
	for _, d := range t.Dims {
		n *= d
	}
	return n
}

// File is a parsed GGUF header. It keeps the path so tensor data is read on demand.
type File struct {
	Path     string
	Version  uint32
	KV       map[string]any
	Tensors  []Tensor
	dataBase int64
}

// Open parses the header, metadata and tensor table of a GGUF file.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	cr := &countingReader{r: r}

	var hdr [4]byte
	if _, err := io.ReadFull(cr, hdr[:]); err != nil {
		return nil, fmt.Errorf("gguf: %s: %w", path, err)
	}
	if string(hdr[:]) != magic {
		return nil, fmt.Errorf("gguf: %s is not a GGUF file", path)
	}
	out := &File{Path: path, KV: map[string]any{}}
	if err := binary.Read(cr, binary.LittleEndian, &out.Version); err != nil {
		return nil, err
	}
	if out.Version < 2 || out.Version > 3 {
		return nil, fmt.Errorf("gguf: %s: unsupported version %d", path, out.Version)
	}
	var nTensors, nKV uint64
	if err := binary.Read(cr, binary.LittleEndian, &nTensors); err != nil {
		return nil, err
	}
	if err := binary.Read(cr, binary.LittleEndian, &nKV); err != nil {
		return nil, err
	}
	if nKV > 1<<20 || nTensors > 1<<24 {
		return nil, fmt.Errorf("gguf: %s: implausible counts (kv=%d tensors=%d)", path, nKV, nTensors)
	}
	for i := uint64(0); i < nKV; i++ {
		k, err := readString(cr)
		if err != nil {
			return nil, fmt.Errorf("gguf: key %d: %w", i, err)
		}
		var vt uint32
		if err := binary.Read(cr, binary.LittleEndian, &vt); err != nil {
			return nil, err
		}
		v, err := readValue(cr, vt)
		if err != nil {
			return nil, fmt.Errorf("gguf: value of %q: %w", k, err)
		}
		out.KV[k] = v
	}
	for i := uint64(0); i < nTensors; i++ {
		name, err := readString(cr)
		if err != nil {
			return nil, err
		}
		var nd uint32
		if err := binary.Read(cr, binary.LittleEndian, &nd); err != nil {
			return nil, err
		}
		if nd == 0 || nd > 8 {
			return nil, fmt.Errorf("gguf: tensor %q has %d dims", name, nd)
		}
		t := Tensor{Name: name, Dims: make([]uint64, nd)}
		for d := range t.Dims {
			if err := binary.Read(cr, binary.LittleEndian, &t.Dims[d]); err != nil {
				return nil, err
			}
		}
		var ty uint32
		if err := binary.Read(cr, binary.LittleEndian, &ty); err != nil {
			return nil, err
		}
		t.Type = TensorType(ty)
		if err := binary.Read(cr, binary.LittleEndian, &t.Offset); err != nil {
			return nil, err
		}
		out.Tensors = append(out.Tensors, t)
	}
	align := int64(32)
	if a, ok := out.Uint("general.alignment"); ok && a > 0 {
		align = int64(a)
	}
	out.dataBase = (cr.n + align - 1) / align * align
	return out, nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func readString(r io.Reader) (string, error) {
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	if n > 1<<24 {
		return "", fmt.Errorf("string of %d bytes", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func readValue(r io.Reader, vt uint32) (any, error) {
	switch vt {
	case tUint8:
		var v uint8
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tInt8:
		var v int8
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tUint16:
		var v uint16
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tInt16:
		var v int16
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tUint32:
		var v uint32
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tInt32:
		var v int32
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tFloat32:
		var v float32
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tBool:
		var v uint8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v != 0, err
	case tString:
		return readString(r)
	case tUint64:
		var v uint64
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tInt64:
		var v int64
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tFloat64:
		var v float64
		return v, binary.Read(r, binary.LittleEndian, &v)
	case tArray:
		var et uint32
		var n uint64
		if err := binary.Read(r, binary.LittleEndian, &et); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
			return nil, err
		}
		// Token tables run to hundreds of thousands of entries; they are skipped,
		// not retained, because nothing here needs them.
		for i := uint64(0); i < n; i++ {
			if _, err := readValue(r, et); err != nil {
				return nil, err
			}
		}
		return fmt.Sprintf("array[%d]", n), nil
	}
	return nil, fmt.Errorf("unknown metadata type %d", vt)
}

// String returns a string metadata value.
func (f *File) String(key string) (string, bool) {
	s, ok := f.KV[key].(string)
	return s, ok
}

// Uint returns an integer metadata value of any width.
func (f *File) Uint(key string) (uint64, bool) {
	switch v := f.KV[key].(type) {
	case uint8:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	case int8:
		return uint64(v), v >= 0
	case int16:
		return uint64(v), v >= 0
	case int32:
		return uint64(v), v >= 0
	case int64:
		return uint64(v), v >= 0
	}
	return 0, false
}

// Float returns a floating-point metadata value.
func (f *File) Float(key string) (float64, bool) {
	switch v := f.KV[key].(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// Tensor finds a tensor by name.
func (f *File) Tensor(name string) (Tensor, bool) {
	for _, t := range f.Tensors {
		if t.Name == name {
			return t, true
		}
	}
	return Tensor{}, false
}

// SHA256 is the content hash of the whole file: the identity of a base model. The
// adapter writer records it and the llama side verifies it, so an adapter can only
// be loaded onto the exact weights it was bound against.
func SHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, bufio.NewReaderSize(f, 4<<20)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Matrix reads a 2-D tensor as rows × cols float32 where rows = Dims[1] (the
// output width) and cols = Dims[0] (the input width), matching how a weight is
// applied (y = W·x).
func (f *File) Matrix(name string) ([][]float32, error) {
	t, ok := f.Tensor(name)
	if !ok {
		return nil, fmt.Errorf("gguf: %s has no tensor %q", f.Path, name)
	}
	if len(t.Dims) != 2 {
		return nil, fmt.Errorf("gguf: tensor %q has %d dims, want 2", name, len(t.Dims))
	}
	in, out := int(t.Dims[0]), int(t.Dims[1])
	flat, err := f.ReadFloat32(t)
	if err != nil {
		return nil, err
	}
	m := make([][]float32, out)
	for i := range m {
		m[i] = flat[i*in : (i+1)*in : (i+1)*in]
	}
	return m, nil
}

// ReadFloat32 reads and dequantises a tensor's elements.
func (f *File) ReadFloat32(t Tensor) ([]float32, error) {
	n := int(t.Elements())
	size, err := byteSize(t.Type, n)
	if err != nil {
		return nil, fmt.Errorf("gguf: tensor %q: %w", t.Name, err)
	}
	fh, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	raw := make([]byte, size)
	if _, err := fh.ReadAt(raw, f.dataBase+int64(t.Offset)); err != nil {
		return nil, fmt.Errorf("gguf: read tensor %q: %w", t.Name, err)
	}
	return dequantize(t.Type, raw, n)
}
