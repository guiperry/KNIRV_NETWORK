package gguf

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// WriteTensor is an F32 tensor to write. Rows × Cols is stored so that the row is
// the fastest-varying axis' complement: GGUF ne = [Cols, Rows], data row-major,
// which is the layout numpy writes for shape (Rows, Cols).
type WriteTensor struct {
	Name string
	Rows [][]float32
}

// Meta is one metadata entry to write: a string, float32, uint32 or bool.
type Meta struct {
	Key   string
	Value any
}

const writeAlign = 32

// Write writes a GGUF file atomically (temp file + rename), so a half-written
// adapter can never be picked up by the llama side.
func Write(path string, meta []Meta, tensors []WriteTensor) error {
	sort.SliceStable(meta, func(i, j int) bool { return meta[i].Key < meta[j].Key })
	for _, t := range tensors {
		if len(t.Rows) == 0 || len(t.Rows[0]) == 0 {
			return fmt.Errorf("gguf: tensor %q is empty", t.Name)
		}
		for i, r := range t.Rows {
			if len(r) != len(t.Rows[0]) {
				return fmt.Errorf("gguf: tensor %q is ragged at row %d", t.Name, i)
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()
	w := bufio.NewWriterSize(f, 1<<20)
	le := binary.LittleEndian
	put := func(v any) { _ = binary.Write(w, le, v) }
	str := func(s string) { put(uint64(len(s))); w.WriteString(s) }

	w.WriteString(magic)
	put(uint32(3))
	put(uint64(len(tensors)))
	put(uint64(len(meta)))
	for _, m := range meta {
		str(m.Key)
		switch v := m.Value.(type) {
		case string:
			put(uint32(tString))
			str(v)
		case float32:
			put(uint32(tFloat32))
			put(v)
		case uint32:
			put(uint32(tUint32))
			put(v)
		case bool:
			put(uint32(tBool))
			b := uint8(0)
			if v {
				b = 1
			}
			put(b)
		default:
			return fmt.Errorf("gguf: metadata %q has unsupported type %T", m.Key, m.Value)
		}
	}
	var off uint64
	offsets := make([]uint64, len(tensors))
	for i, t := range tensors {
		offsets[i] = off
		n := uint64(len(t.Rows)) * uint64(len(t.Rows[0])) * 4
		off += (n + writeAlign - 1) / writeAlign * writeAlign
		str(t.Name)
		put(uint32(2))
		put(uint64(len(t.Rows[0])))
		put(uint64(len(t.Rows)))
		put(uint32(F32))
		put(offsets[i])
	}
	if err := w.Flush(); err != nil {
		return err
	}
	pos, _ := f.Seek(0, 1)
	pad := (writeAlign - pos%writeAlign) % writeAlign
	w.Write(make([]byte, pad))
	for _, t := range tensors {
		var n int
		for _, row := range t.Rows {
			for _, v := range row {
				var b [4]byte
				le.PutUint32(b[:], math.Float32bits(v))
				w.Write(b[:])
				n += 4
			}
		}
		w.Write(make([]byte, (writeAlign-n%writeAlign)%writeAlign))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return os.Rename(tmp, path)
}
