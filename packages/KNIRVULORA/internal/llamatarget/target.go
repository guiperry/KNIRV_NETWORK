// Package llamatarget turns a llama.cpp base model (a GGUF file) into what ULoRA
// binds against: a BaseModelSpec, the weight-name mapping for each semantic
// module, and the per-module input/output widths.
//
// Why widths matter: HEAD has square d×d projections, so one hidden-sized P_in and
// P_out serves every module. A llama model with grouped-query attention does not:
// K and V project hidden → kv_heads·head_dim, which is narrower than hidden. Q and
// O are hidden-sized only when heads·head_dim == hidden. Support is therefore
// declared per module, and an unsupported one is refused with the reason instead of
// being bound with the wrong shape.
package llamatarget

import (
	"fmt"
	"strings"

	"ulora/internal/api"
	"ulora/internal/gguf"
)

// Target is a llama base model as ULoRA sees it.
type Target struct {
	Path   string
	SHA256 string
	Arch   string
	Spec   api.BaseModelSpec
	// QDim is heads·head_dim (the width of Q's output and O's input).
	QDim int
	// KVDim is kv_heads·head_dim (the width of K's and V's output).
	KVDim int
}

// Module describes one semantic module on this target.
type Module struct {
	Semantic string // "self_attn.q_proj"
	GGUFName string // "attn_q" -> blk.N.attn_q.weight
	In, Out  int
}

// activations by architecture. An unknown architecture is refused rather than
// guessed: the binder's topology check compares this string.
var activations = map[string]string{
	"llama": "silu", "mistral": "silu", "qwen2": "silu", "qwen3": "silu", "phi3": "silu",
	"gemma": "gelu", "gemma2": "gelu", "gemma3": "gelu",
}

// FromGGUF reads the base model's metadata. sha is its content hash (pass "" to
// compute it; callers with a cache pass the cached value).
func FromGGUF(path, sha string) (*Target, error) {
	f, err := gguf.Open(path)
	if err != nil {
		return nil, err
	}
	arch, ok := f.String(gguf.KeyArchitecture)
	if !ok || arch == "" {
		return nil, fmt.Errorf("llamatarget: %s has no general.architecture", path)
	}
	act, ok := activations[arch]
	if !ok {
		return nil, fmt.Errorf("llamatarget: architecture %q is not supported yet (known: llama, mistral, qwen2, qwen3, phi3, gemma*)", arch)
	}
	get := func(k string) (int, error) {
		v, ok := f.Uint(arch + "." + k)
		if !ok || v == 0 {
			return 0, fmt.Errorf("llamatarget: %s lacks %s.%s", path, arch, k)
		}
		return int(v), nil
	}
	hidden, err := get("embedding_length")
	if err != nil {
		return nil, err
	}
	layers, err := get("block_count")
	if err != nil {
		return nil, err
	}
	heads, err := get("attention.head_count")
	if err != nil {
		return nil, err
	}
	ffn, err := get("feed_forward_length")
	if err != nil {
		return nil, err
	}
	kv := heads
	if v, ok := f.Uint(arch + ".attention.head_count_kv"); ok && v > 0 {
		kv = int(v)
	}
	if heads%kv != 0 {
		return nil, fmt.Errorf("llamatarget: %d heads is not a multiple of %d kv heads", heads, kv)
	}
	headDim := hidden / heads
	if v, ok := f.Uint(arch + ".attention.key_length"); ok && v > 0 {
		headDim = int(v)
	}
	mech := "mha"
	switch {
	case kv == 1 && heads > 1:
		mech = "mqa"
	case kv < heads:
		mech = "gqa"
	}
	if sha == "" {
		if sha, err = gguf.SHA256(path); err != nil {
			return nil, err
		}
	}
	size, _ := f.String("general.size_label")
	if size == "" {
		size = fmt.Sprintf("%dL-%dd", layers, hidden)
	}
	return &Target{
		Path: path, SHA256: sha, Arch: arch, QDim: heads * headDim, KVDim: kv * headDim,
		Spec: api.BaseModelSpec{
			// The family carries the file hash: a connector is derived from THIS
			// model's weights, so two finetunes of one architecture must not share a
			// cache entry.
			Family: "gguf-" + arch + "-" + sha[:12], ParamCount: strings.ToLower(size),
			HiddenSize: hidden, IntermediateSize: ffn, NumAttentionHeads: heads, NumKeyValueHeads: kv,
			HeadDim: headDim, NumLayers: layers, AttentionMechanism: mech, ActivationFunc: act,
		},
	}, nil
}

// semantic -> (gguf tensor stem, which widths)
var moduleNames = map[string]string{
	"self_attn.q_proj": "attn_q", "self_attn.k_proj": "attn_k",
	"self_attn.v_proj": "attn_v", "self_attn.o_proj": "attn_output",
}

// Module resolves a semantic module on this target, or says why it cannot be bound.
//
// Supported now: Q and O when heads·head_dim equals the hidden size. K and V, and
// the MLP projections, need a connector with a different output width than the
// hidden-sized pair the cache holds today; they are refused.
func (t *Target) Module(semantic string) (Module, error) {
	stem, ok := moduleNames[semantic]
	if !ok {
		return Module{}, fmt.Errorf("module %q is not an attention projection ULoRA can bind to a llama target yet (supported: self_attn.q_proj, self_attn.o_proj)", semantic)
	}
	h := t.Spec.HiddenSize
	switch semantic {
	case "self_attn.k_proj", "self_attn.v_proj":
		return Module{}, fmt.Errorf("module %q projects hidden(%d) -> kv(%d) on this %s model; the connector pair is hidden-sized, so binding it would produce wrongly-shaped factors. K/V need a per-module connector (planned, P5b)", semantic, h, t.KVDim, t.Spec.AttentionMechanism)
	case "self_attn.q_proj", "self_attn.o_proj":
		if t.QDim != h {
			return Module{}, fmt.Errorf("module %q needs heads*head_dim == hidden, but this model has %d != %d (planned, P5b)", semantic, t.QDim, h)
		}
	}
	return Module{Semantic: semantic, GGUFName: stem, In: h, Out: h}, nil
}

// WeightName is the GGUF tensor a module targets in one layer.
func (m Module) WeightName(layer int) string {
	return fmt.Sprintf("blk.%d.%s.weight", layer, m.GGUFName)
}

// ConnectorSource is the weight matrix a connector is derived from: the middle
// layer's Q projection, hidden × hidden, the llama counterpart of the matrix the
// HEAD serves for the same purpose.
func (t *Target) ConnectorSource() string {
	return Module{GGUFName: "attn_q"}.WeightName(t.Spec.NumLayers / 2)
}

// AttentionCompatible reports whether a source model's attention variant can feed a
// target's for the given modules. MHA, GQA and MQA differ only in K/V sharing, so a
// Q/O-only core is indifferent to them.
func AttentionCompatible(source, target string, modules []string) bool {
	if source == target {
		return true
	}
	for _, m := range modules {
		if m != "self_attn.q_proj" && m != "self_attn.o_proj" {
			return false
		}
	}
	variant := func(s string) bool { return s == "mha" || s == "gqa" || s == "mqa" }
	return variant(source) && variant(target)
}
