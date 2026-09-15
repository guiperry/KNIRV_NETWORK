package safetensors

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// A round trip through real encoding: this is the property the binder depends
// on, since the compiler writes genuine safetensors bytes.
func TestRoundTrip(t *testing.T) {
	want := map[string][][]float32{
		"source/lora_A": {{0.1, 0.2}, {0.3, 0.4}},
		"source/lora_B": {{1.5}, {-2.5}},
	}

	raw, err := Write(want, map[string]string{"producer": "test"})
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	file, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := file.Metadata["producer"]; got != "test" {
		t.Fatalf("metadata producer = %q", got)
	}

	names := file.Names()
	if len(names) != 2 || names[0] != "source/lora_A" {
		t.Fatalf("names = %v", names)
	}

	for name, expected := range want {
		tensor, ok := file.Tensor(name)
		if !ok {
			t.Fatalf("tensor %q missing", name)
		}
		matrix, err := tensor.Float32Matrix()
		if err != nil {
			t.Fatalf("matrix %q: %v", name, err)
		}
		if len(matrix) != len(expected) {
			t.Fatalf("%q rows = %d, want %d", name, len(matrix), len(expected))
		}
		for r := range expected {
			for c := range expected[r] {
				if matrix[r][c] != expected[r][c] {
					t.Fatalf("%q[%d][%d] = %v, want %v", name, r, c, matrix[r][c], expected[r][c])
				}
			}
		}
	}
}

// The exact failure that made every real bundle unbindable: binary safetensors
// is not JSON, so a JSON reader must fail on it. This pins the format so the
// binder cannot quietly go back to assuming JSON.
func TestBinaryHeaderIsNotJSON(t *testing.T) {
	raw, err := Write(map[string][][]float32{"a": {{1}}}, nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	var sink map[string]any
	if err := json.Unmarshal(raw, &sink); err == nil {
		t.Fatal("a real safetensors file must not parse as JSON")
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	good, err := Write(map[string][][]float32{"a": {{1, 2}, {3, 4}}}, nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	cases := map[string][]byte{
		"too short":          {1, 2, 3},
		"zero header length": make([]byte, 8),
		"header claims more than the file": func() []byte {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, 9999)
			return append(b, []byte("{}")...)
		}(),
		"header is not JSON": func() []byte {
			b := make([]byte, 8)
			header := []byte("not json at all")
			binary.LittleEndian.PutUint64(b, uint64(len(header)))
			return append(b, header...)
		}(),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(input); err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	// An offsets range pointing past the payload must be refused rather than
	// slicing out of bounds.
	truncated := good[:len(good)-4]
	if _, err := Parse(truncated); err == nil {
		t.Fatal("a tensor range past the data buffer must be refused")
	}
}

// A tensor whose shape disagrees with its byte range is corrupt; reading it
// would fabricate weights.
func TestParseRejectsShapeByteMismatch(t *testing.T) {
	header := map[string]any{
		"w": map[string]any{"dtype": "F32", "shape": []int{3, 3}, "data_offsets": []int{0, 16}},
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(len(headerJSON)))
	raw = append(raw, headerJSON...)
	raw = append(raw, make([]byte, 16)...)

	if _, err := Parse(raw); err == nil {
		t.Fatal("shape/byte-count mismatch must be refused")
	}
}

func TestUnsupportedDtypeIsRejected(t *testing.T) {
	header := map[string]any{
		"w": map[string]any{"dtype": "F8_E4M3", "shape": []int{1, 4}, "data_offsets": []int{0, 4}},
	}
	headerJSON, _ := json.Marshal(header)
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(len(headerJSON)))
	raw = append(raw, headerJSON...)
	raw = append(raw, make([]byte, 4)...)

	file, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tensor, _ := file.Tensor("w")
	if _, err := tensor.Float32Matrix(); err == nil {
		t.Fatal("a float8 dtype must be rejected rather than reinterpreted")
	}
}

// Only 2-D tensors are LoRA A/B matrices.
func TestNonMatrixIsRejected(t *testing.T) {
	header := map[string]any{
		"v": map[string]any{"dtype": "F32", "shape": []int{4}, "data_offsets": []int{0, 16}},
	}
	headerJSON, _ := json.Marshal(header)
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(len(headerJSON)))
	raw = append(raw, headerJSON...)
	raw = append(raw, make([]byte, 16)...)

	file, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tensor, _ := file.Tensor("v")
	if _, err := tensor.Float32Matrix(); err == nil {
		t.Fatal("a 1-D tensor is not a matrix")
	}
}

// F16 and BF16 appear in real adapters, so they must convert, not error.
func TestFloat16AndBFloat16Conversion(t *testing.T) {
	if got := float16ToFloat32(0x3C00); got != 1.0 {
		t.Fatalf("half 0x3C00 = %v, want 1.0", got)
	}
	if got := float16ToFloat32(0x0000); got != 0.0 {
		t.Fatalf("half 0 = %v, want 0", got)
	}
	if got := float16ToFloat32(0x8000); !math.Signbit(float64(got)) {
		t.Fatalf("half 0x8000 must be negative zero, got %v", got)
	}
	if got := float16ToFloat32(0xBC00); got != -1.0 {
		t.Fatalf("half 0xBC00 = %v, want -1.0", got)
	}
	// 0x3C00 is also BF16 for 1.0 when shifted into the high bits.
	bf16 := math.Float32frombits(uint32(0x3F80) << 16)
	if bf16 != 1.0 {
		t.Fatalf("bfloat16 0x3F80 = %v, want 1.0", bf16)
	}
}

func TestWriteRejectsEmptyAndRagged(t *testing.T) {
	if _, err := Write(nil, nil); err == nil {
		t.Fatal("empty tensor map must be refused")
	}
	if _, err := Write(map[string][][]float32{"a": {{1, 2}, {3}}}, nil); err == nil {
		t.Fatal("ragged rows must be refused")
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	tensors := map[string][][]float32{"b": {{2}}, "a": {{1}}}
	first, err := Write(tensors, nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	second, err := Write(tensors, nil)
	if err != nil {
		t.Fatalf("write again: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("identical input must produce identical bytes (content addressing depends on it)")
	}
	if !strings.Contains(string(first[8:]), "\"a\"") {
		t.Fatalf("header should carry tensor names, got %q", string(first[8:]))
	}
}
