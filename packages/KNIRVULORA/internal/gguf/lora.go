package gguf

import "fmt"

// LoRA adapter keys and tensor names as llama.cpp's loader expects them
// (llama_adapter_lora_init: general.type=adapter, adapter.type=lora,
// adapter.lora.alpha, and <weight name>.lora_a / .lora_b per target weight).
const (
	KeyGeneralType  = "general.type"
	KeyAdapterType  = "adapter.type"
	KeyLoRAAlpha    = "adapter.lora.alpha"
	KeyArchitecture = "general.architecture"

	// KNIRV provenance, ignored by llama.cpp and checked by the llama wrapper.
	KeyBaseSHA256  = "knirv.base_sha256"
	KeyBundleHash  = "knirv.bundle_content_hash"
	KeySkillName   = "knirv.skill_name"
	KeyModulesNote = "knirv.target_modules"
)

// LayerLoRA is the factor pair for one target weight: A is rank × in, B is out × rank.
type LayerLoRA struct {
	Weight string // e.g. "blk.3.attn_q.weight"
	A, B   [][]float32
}

// AdapterInfo is the metadata of an adapter file.
type AdapterInfo struct {
	Architecture string
	Alpha        float32
	BaseSHA256   string
	BundleHash   string
	Skill        string
	Modules      string
}

// WriteLoRA writes a GGUF LoRA adapter. alpha is llama.cpp's adapter.lora.alpha: it
// scales the update by alpha/rank at load time, so it must be the bundle's own
// scaling factor and the rank must match the A/B shapes.
func WriteLoRA(path string, info AdapterInfo, layers []LayerLoRA) error {
	if info.Architecture == "" {
		return fmt.Errorf("gguf: adapter architecture is required")
	}
	if len(layers) == 0 {
		return fmt.Errorf("gguf: an adapter needs at least one target weight")
	}
	meta := []Meta{
		{KeyGeneralType, "adapter"}, {KeyAdapterType, "lora"},
		{KeyArchitecture, info.Architecture}, {KeyLoRAAlpha, info.Alpha},
	}
	for k, v := range map[string]string{KeyBaseSHA256: info.BaseSHA256, KeyBundleHash: info.BundleHash, KeySkillName: info.Skill, KeyModulesNote: info.Modules} {
		if v != "" {
			meta = append(meta, Meta{k, v})
		}
	}
	var ts []WriteTensor
	rank := 0
	for _, l := range layers {
		if len(l.A) == 0 || len(l.B) == 0 {
			return fmt.Errorf("gguf: %s: empty factors", l.Weight)
		}
		r := len(l.A)
		if len(l.B[0]) != r {
			return fmt.Errorf("gguf: %s: A has rank %d but B has %d columns", l.Weight, r, len(l.B[0]))
		}
		if rank != 0 && r != rank {
			return fmt.Errorf("gguf: %s: rank %d differs from %d (llama.cpp applies one alpha per adapter)", l.Weight, r, rank)
		}
		rank = r
		ts = append(ts, WriteTensor{l.Weight + ".lora_a", l.A}, WriteTensor{l.Weight + ".lora_b", l.B})
	}
	return Write(path, meta, ts)
}

// ReadAdapterInfo reads an adapter's metadata and validates that it is a LoRA.
func ReadAdapterInfo(path string) (AdapterInfo, *File, error) {
	f, err := Open(path)
	if err != nil {
		return AdapterInfo{}, nil, err
	}
	if t, _ := f.String(KeyGeneralType); t != "adapter" {
		return AdapterInfo{}, nil, fmt.Errorf("gguf: %s is not an adapter (general.type=%q)", path, t)
	}
	if t, _ := f.String(KeyAdapterType); t != "lora" {
		return AdapterInfo{}, nil, fmt.Errorf("gguf: %s is not a LoRA adapter (adapter.type=%q)", path, t)
	}
	var info AdapterInfo
	info.Architecture, _ = f.String(KeyArchitecture)
	if a, ok := f.Float(KeyLoRAAlpha); ok {
		info.Alpha = float32(a)
	}
	info.BaseSHA256, _ = f.String(KeyBaseSHA256)
	info.BundleHash, _ = f.String(KeyBundleHash)
	info.Skill, _ = f.String(KeySkillName)
	info.Modules, _ = f.String(KeyModulesNote)
	return info, f, nil
}
