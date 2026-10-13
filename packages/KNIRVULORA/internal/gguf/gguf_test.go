package gguf

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadRoundTripAndMatrixOrientation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.gguf")
	a := [][]float32{{1, 2, 3}, {4, 5, 6}} // 2 x 3
	if err := Write(path, []Meta{{"general.architecture", "llama"}, {"x.f", float32(1.5)}, {"x.u", uint32(7)}, {"x.b", true}},
		[]WriteTensor{{"w", a}, {"v", [][]float32{{9}}}}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if arch, _ := f.String("general.architecture"); arch != "llama" {
		t.Fatalf("arch %q", arch)
	}
	if v, _ := f.Uint("x.u"); v != 7 {
		t.Fatal("uint")
	}
	if v, _ := f.Float("x.f"); v != 1.5 {
		t.Fatal("float")
	}
	tw, _ := f.Tensor("w")
	// GGUF order is [in, out]: a 2x3 (rows x cols) matrix has ne = [3, 2].
	if len(tw.Dims) != 2 || tw.Dims[0] != 3 || tw.Dims[1] != 2 {
		t.Fatalf("dims %v", tw.Dims)
	}
	m, err := f.Matrix("w")
	if err != nil || len(m) != 2 || m[1][2] != 6 || m[0][1] != 2 {
		t.Fatalf("matrix %v %v", m, err)
	}
	v, _ := f.Matrix("v")
	if v[0][0] != 9 {
		t.Fatal("second tensor misaligned: the data section offsets are wrong")
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("temp file left behind")
	}
}

func TestOpenRejectsGarbage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad")
	os.WriteFile(p, []byte("NOPE0000000000000000"), 0o644)
	if _, err := Open(p); err == nil {
		t.Fatal("garbage accepted")
	}
	os.WriteFile(p, []byte("GGUF"), 0o644)
	if _, err := Open(p); err == nil {
		t.Fatal("truncated header accepted")
	}
}

func h16(f float32) []byte {
	// exact for the small values used here
	if f == 0 {
		return []byte{0, 0}
	}
	b := math.Float32bits(f)
	sign := uint16(b>>31) << 15
	exp := int((b>>23)&0xff) - 127 + 15
	frac := uint16((b >> 13) & 0x3ff)
	var out [2]byte
	binary.LittleEndian.PutUint16(out[:], sign|uint16(exp)<<10|frac)
	return out[:]
}

func TestHalfFloat(t *testing.T) {
	for _, v := range []float32{0, 1, -2, 0.5, 3.25, -0.125, 100} {
		if got := f16(h16(v)); got != v {
			t.Fatalf("f16(%v)=%v", v, got)
		}
	}
	if sub := f16([]byte{1, 0}); sub <= 0 || sub > 1e-6 { // smallest subnormal ~5.96e-8
		t.Fatalf("subnormal %v", sub)
	}
}

func TestDequantQ8_0(t *testing.T) {
	blk := append(h16(0.5), make([]byte, 32)...)
	for j := 0; j < 32; j++ {
		blk[2+j] = byte(int8(j - 16))
	}
	out, err := dequantize(Q8_0, blk, 32)
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 32; j++ {
		if want := float32(j-16) * 0.5; out[j] != want {
			t.Fatalf("q8_0[%d]=%v want %v", j, out[j], want)
		}
	}
}

func TestDequantQ4_0LowThenHighNibbles(t *testing.T) {
	blk := append(h16(2), make([]byte, 16)...)
	for j := 0; j < 16; j++ {
		blk[2+j] = byte(j) | byte((15-j)<<4) // low nibble j, high nibble 15-j
	}
	out, _ := dequantize(Q4_0, blk, 32)
	for j := 0; j < 16; j++ {
		if out[j] != float32(j-8)*2 || out[j+16] != float32(15-j-8)*2 {
			t.Fatalf("q4_0 j=%d got %v/%v", j, out[j], out[j+16])
		}
	}
}

func TestDequantQ5_0UsesTheHighBit(t *testing.T) {
	blk := make([]byte, 22)
	copy(blk, h16(1))
	binary.LittleEndian.PutUint32(blk[2:], 1|1<<16) // high bit for element 0 and element 16
	blk[6] = 0x00                                   // low nibbles 0 -> x0 = 16, x1 = 16 (with high bit)
	out, _ := dequantize(Q5_0, blk, 32)
	if out[0] != 0 || out[16] != 0 { // (16-16)*1
		t.Fatalf("q5_0 %v %v", out[0], out[16])
	}
	if out[1] != -16 { // no high bit: (0-16)
		t.Fatalf("q5_0 out[1]=%v", out[1])
	}
}

func TestKQuantsAreRefusedByName(t *testing.T) {
	_, err := byteSize(TensorType(11), 256)
	if err == nil || !strings.Contains(err.Error(), "Q3_K") {
		t.Fatalf("want a named refusal, got %v", err)
	}
}

func TestWriteLoRAValidationAndReadBack(t *testing.T) {
	dir := t.TempDir()
	good := LayerLoRA{"blk.0.attn_q.weight", [][]float32{{1, 2, 3, 4}, {5, 6, 7, 8}}, [][]float32{{1, 0}, {0, 1}, {1, 1}, {2, 2}}}
	p := filepath.Join(dir, "a.gguf")
	if err := WriteLoRA(p, AdapterInfo{Architecture: "llama", Alpha: 16, BaseSHA256: "abc", BundleHash: "def", Skill: "s"}, []LayerLoRA{good}); err != nil {
		t.Fatal(err)
	}
	info, f, err := ReadAdapterInfo(p)
	if err != nil || info.Alpha != 16 || info.BaseSHA256 != "abc" || info.Architecture != "llama" || info.BundleHash != "def" {
		t.Fatalf("%+v %v", info, err)
	}
	a, _ := f.Matrix("blk.0.attn_q.weight.lora_a")
	b, _ := f.Matrix("blk.0.attn_q.weight.lora_b")
	if len(a) != 2 || len(a[0]) != 4 || len(b) != 4 || len(b[0]) != 2 || b[3][1] != 2 {
		t.Fatalf("a=%v b=%v", a, b)
	}
	bad := good
	bad.B = [][]float32{{1, 0, 0}} // rank mismatch
	if err := WriteLoRA(filepath.Join(dir, "b.gguf"), AdapterInfo{Architecture: "llama"}, []LayerLoRA{bad}); err == nil {
		t.Fatal("rank mismatch accepted")
	}
	if err := WriteLoRA(filepath.Join(dir, "c.gguf"), AdapterInfo{}, []LayerLoRA{good}); err == nil {
		t.Fatal("missing architecture accepted")
	}
	if _, _, err := ReadAdapterInfo(filepath.Join(dir, "none")); err == nil {
		t.Fatal("missing file accepted")
	}
}

// Cross-checks against llama.cpp's own gguf-py when it is available: the file this
// package writes must parse there, with the shapes llama.cpp expects.
func TestAdapterIsReadableByLlamaCppGGUFPy(t *testing.T) {
	py := os.Getenv("LLAMA_CPP_DIR")
	if py == "" {
		py = "/var/lib/knirvserver/knirvllama/llama.cpp"
	}
	if _, err := os.Stat(filepath.Join(py, "gguf-py")); err != nil {
		t.Skip("llama.cpp gguf-py not available")
	}
	if exec.Command("python3", "-c", "import numpy").Run() != nil {
		t.Skip("numpy not available")
	}
	p := filepath.Join(t.TempDir(), "a.gguf")
	A := [][]float32{{1, 2, 3, 4}, {5, 6, 7, 8}}
	B := [][]float32{{1, 0}, {0, 1}, {1, 1}}
	if err := WriteLoRA(p, AdapterInfo{Architecture: "llama", Alpha: 8}, []LayerLoRA{{"blk.0.attn_q.weight", A, B}}); err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from gguf import GGUFReader
r = GGUFReader(sys.argv[1])
f = {k: v.contents() for k, v in r.fields.items()}
assert f["general.type"] == "adapter" and f["adapter.type"] == "lora", f
assert abs(f["adapter.lora.alpha"] - 8) < 1e-6
t = {x.name: x for x in r.tensors}
a, b = t["blk.0.attn_q.weight.lora_a"], t["blk.0.attn_q.weight.lora_b"]
assert list(a.shape) == [4, 2], a.shape      # ne = [in, rank]
assert list(b.shape) == [2, 3], b.shape      # ne = [rank, out]
assert a.data.reshape(2, 4)[1][3] == 8 and b.data.reshape(3, 2)[2][1] == 1
print("ok")
`
	cmd := exec.Command("python3", "-c", script, p)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(py, "gguf-py"))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("gguf-py rejected the adapter: %v\n%s", err, out)
	}
}

// The dequantisers must agree with llama.cpp's own gguf-py on data gguf-py itself
// quantised: this is the check that the block layouts are right, not just
// self-consistent.
func TestDequantMatchesLlamaCppGGUFPy(t *testing.T) {
	py := os.Getenv("LLAMA_CPP_DIR")
	if py == "" {
		py = "/var/lib/knirvserver/knirvllama/llama.cpp"
	}
	if _, err := os.Stat(filepath.Join(py, "gguf-py")); err != nil {
		t.Skip("llama.cpp gguf-py not available")
	}
	if exec.Command("python3", "-c", "import numpy").Run() != nil {
		t.Skip("numpy not available")
	}
	dir := t.TempDir()
	cmd := exec.Command("python3", filepath.Join("..", "..", "scripts", "gguf-spike", "quants_fixture.py"), dir)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(py, "gguf-py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "expected.json"))
	var want map[string][][]float64
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	f, err := Open(filepath.Join(dir, "quants.gguf"))
	if err != nil {
		t.Fatal(err)
	}
	for name, rows := range want {
		got, err := f.Matrix(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		var worst float64
		for i := range rows {
			for j := range rows[i] {
				if d := math.Abs(float64(got[i][j]) - rows[i][j]); d > worst {
					worst = d
				}
			}
		}
		if worst > 1e-6 {
			tt, _ := f.Tensor(name)
			t.Errorf("%s (%s): max |go - gguf-py| = %g", name, tt.Type, worst)
		}
	}
}
