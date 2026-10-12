package hrm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatrixReadsGoldenAttentionProjection(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("testdata/golden.safetensors")
	if err != nil {
		t.Fatal(err)
	}
	const v = "v1-deadbeef"
	if err := os.WriteFile(filepath.Join(dir, "model-"+v+".safetensors"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewService(dir)
	m, err := s.Matrix(v, "h.0.attn.q.weight")
	if err != nil || len(m) == 0 || len(m) != len(m[0]) {
		t.Fatalf("%d x ? %v", len(m), err)
	}
	for _, bad := range []string{"tok.weight", "l.0.mlp.up.weight", "../x", "h.0.attn.q.weight/../../a", "h.99.attn.q.weight"} {
		if _, err := s.Matrix(v, bad); err == nil {
			t.Errorf("served %q", bad)
		}
	}
	if _, err := s.Matrix("../../etc", "h.0.attn.q.weight"); err == nil {
		t.Error("bad version accepted")
	}
}
