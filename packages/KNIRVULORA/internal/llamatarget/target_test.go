package llamatarget

import (
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/gguf"
)

func base(t *testing.T, arch string, hidden, heads, kv, headDim int) string {
	t.Helper()
	meta := []gguf.Meta{
		{Key: "general.architecture", Value: arch},
		{Key: arch + ".embedding_length", Value: uint32(hidden)}, {Key: arch + ".block_count", Value: uint32(4)},
		{Key: arch + ".attention.head_count", Value: uint32(heads)}, {Key: arch + ".feed_forward_length", Value: uint32(32)},
	}
	if kv > 0 {
		meta = append(meta, gguf.Meta{Key: arch + ".attention.head_count_kv", Value: uint32(kv)})
	}
	if headDim > 0 {
		meta = append(meta, gguf.Meta{Key: arch + ".attention.key_length", Value: uint32(headDim)})
	}
	p := filepath.Join(t.TempDir(), "m.gguf")
	if err := gguf.Write(p, meta, []gguf.WriteTensor{{Name: "x", Rows: [][]float32{{1}}}}); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAttentionVariantFromKVHeads(t *testing.T) {
	for name, c := range map[string]struct {
		kv   int
		want string
	}{"mha (kv absent)": {0, "mha"}, "mha": {8, "mha"}, "gqa": {2, "gqa"}, "mqa": {1, "mqa"}} {
		tg, err := FromGGUF(base(t, "llama", 64, 8, c.kv, 0), strings.Repeat("a", 64))
		if err != nil || tg.Spec.AttentionMechanism != c.want {
			t.Errorf("%s: %+v %v", name, tg, err)
		}
	}
}

func TestWidthsAndFamilyCarryTheFileHash(t *testing.T) {
	tg, err := FromGGUF(base(t, "llama", 64, 8, 2, 0), strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	if tg.QDim != 64 || tg.KVDim != 16 || tg.Spec.HeadDim != 8 || tg.Spec.ActivationFunc != "silu" {
		t.Fatalf("%+v", tg)
	}
	if !strings.HasSuffix(tg.Spec.Family, "abababababab") {
		t.Fatalf("family %q does not carry the file hash", tg.Spec.Family)
	}
	g, _ := FromGGUF(base(t, "gemma2", 64, 8, 8, 0), strings.Repeat("a", 64))
	if g.Spec.ActivationFunc != "gelu" {
		t.Fatalf("gemma activation %q", g.Spec.ActivationFunc)
	}
}

func TestQOSupportedKVRefusedWithTheShapes(t *testing.T) {
	tg, _ := FromGGUF(base(t, "llama", 64, 8, 2, 0), strings.Repeat("a", 64))
	q, err := tg.Module("self_attn.q_proj")
	if err != nil || q.In != 64 || q.Out != 64 || q.WeightName(3) != "blk.3.attn_q.weight" {
		t.Fatalf("%+v %v", q, err)
	}
	if o, err := tg.Module("self_attn.o_proj"); err != nil || o.WeightName(0) != "blk.0.attn_output.weight" {
		t.Fatalf("%+v %v", o, err)
	}
	for _, m := range []string{"self_attn.k_proj", "self_attn.v_proj"} {
		_, err := tg.Module(m)
		if err == nil || !strings.Contains(err.Error(), "kv(16)") || !strings.Contains(err.Error(), "P5b") {
			t.Errorf("%s: %v", m, err)
		}
	}
	if _, err := tg.Module("mlp.up_proj"); err == nil {
		t.Error("mlp accepted")
	}
}

func TestNonSquareQIsRefused(t *testing.T) {
	// 8 heads x head_dim 16 = 128 != hidden 64 (Gemma-style)
	tg, err := FromGGUF(base(t, "llama", 64, 8, 8, 16), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tg.Module("self_attn.q_proj"); err == nil || !strings.Contains(err.Error(), "128 != 64") {
		t.Fatalf("%v", err)
	}
}

func TestRefusals(t *testing.T) {
	if _, err := FromGGUF(base(t, "mamba", 64, 8, 8, 0), strings.Repeat("a", 64)); err == nil || !strings.Contains(err.Error(), "mamba") {
		t.Fatalf("unknown arch: %v", err)
	}
	if _, err := FromGGUF(base(t, "llama", 64, 8, 3, 0), strings.Repeat("a", 64)); err == nil {
		t.Fatal("heads not a multiple of kv heads accepted")
	}
}

func TestAttentionCompatible(t *testing.T) {
	qo := []string{"self_attn.q_proj", "self_attn.o_proj"}
	if !AttentionCompatible("gqa", "mha", qo) || !AttentionCompatible("mha", "mha", nil) {
		t.Fatal("q/o must be indifferent to the attention variant")
	}
	if AttentionCompatible("gqa", "mha", []string{"self_attn.q_proj", "self_attn.k_proj"}) {
		t.Fatal("K must not cross variants")
	}
	if AttentionCompatible("gqa", "linear", qo) {
		t.Fatal("unknown variant accepted")
	}
}
