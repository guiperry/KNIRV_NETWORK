package api

// Named BaseModelSpec templates for the models KNIRVMODELS serves. A caller that
// compiles or transfers a bundle to one of these models asks for the template by
// name instead of hand-typing dimensions.
//
// Every value below is copied from KNIRVMODELS source or configuration; the
// comment beside a value names the file:line it came from. Where a value is not
// in the repository, the field is left unset and marked TODO rather than guessed.
// BaseModelSpec.Validate rejects unset architecture dimensions (hidden size,
// intermediate size, head counts, head dim, layer count) and empty family,
// attention mechanism and activation function, so a template with those TODOs
// cannot be used for binding until they are filled in. Validate does not check
// ParamCount, so a ParamCount TODO is not caught by it.
//
// Paths below are relative to KNIRV_NETWORK/packages/KNIRVMODELS, except a cite
// that names another package (for example KNIRVSERVER/config/...), which is
// relative to KNIRV_NETWORK/packages/.

const (
	BaseModelTemplateHeartDecision = "heart-decision"
	BaseModelTemplateHeartQwen     = "heart-qwen"
	BaseModelTemplateHandsGPT      = "hands-gpt"
	BaseModelTemplateHeadVLM       = "head-vlm"
)

// BaseModelTemplate is one named base model: its architecture (Spec) and the
// semantic adapter roles a ULoRA bundle may target on it (TargetModules).
type BaseModelTemplate struct {
	Name          string
	Spec          BaseModelSpec
	TargetModules []string
}

// attentionRoles are the four attention projections in semantic role form, the
// same names CanonicalCore.TargetModules uses elsewhere in this package.
var attentionRoles = []string{
	"self_attn.q_proj",
	"self_attn.k_proj",
	"self_attn.v_proj",
	"self_attn.o_proj",
}

var baseModelTemplates = map[string]BaseModelTemplate{
	// heart-decision: the HEART/HEAD decision transformer (knirv-heart/v1).
	//
	// No production head_config.json is committed. The dimensions are the trainer's
	// default preset "small" (pipeline/5_HEAD_TRAINER/src/main.rs:33 and :78,
	// preset table pipeline/5_HEAD_TRAINER/src/lib.rs:21 ->
	// (d=128, nh=4, ffn=2, l=2, h=2, seq=160)). The trainer's own doc names
	// "base" (d=512, nh=8, ffn=4, l=3, h=3) as the ~27M target
	// (pipeline/5_HEAD_TRAINER/src/lib.rs:16 doc, row at :22); that is NOT used here
	// because it is a target, not a served config.
	BaseModelTemplateHeartDecision: {
		Name: BaseModelTemplateHeartDecision,
		Spec: BaseModelSpec{
			Family: BaseModelTemplateHeartDecision,
			// TODO(param_count): unset, pending owner decision. This count is per trained
			// model, not per template: it is computed by HeadConfig::param_count
			// (heart/decision/src/config.rs:186-220) from the promoted head_config.json,
			// because the decision heads come from the training data. Nothing in the
			// repo stores a fixed value.
			ParamCount: "",
			// d_model (lib.rs:21 -> 128).
			HiddenSize: 128,
			// f = ffn_mult * d_model (heart/decision/src/config.rs:189) with ffn_mult=2
			// (lib.rs:21) -> 256.
			IntermediateSize: 256,
			// n_heads (lib.rs:21 -> 4).
			NumAttentionHeads: 4,
			// q, k and v are each d x d and split into n_heads (heart/decision/src/model.rs:279-281
			// and :301), so every head has its own K and V: no key/value grouping.
			NumKeyValueHeads: 4,
			// head_dim = d_model / n_heads (heart/decision/src/config.rs:183) -> 32.
			HeadDim: 32,
			// Total blocks: l_layers + h_layers (heart/decision/src/config.rs:196) = 2 + 2.
			NumLayers: 4,
			// Softmax attention over per-head q/k/v (heart/decision/src/model.rs:300-304).
			AttentionMechanism: "mha",
			// SwiGLU MLP: silu(gate(h)) * up(h) (heart/decision/src/model.rs:313).
			ActivationFunc: "silu",
		},
		// Attention projections only. The adapter merges q/k/v/o and reports MLP roles
		// as skipped (heart/decision/src/adapter.rs:10-13, role_target at :89-98).
		TargetModules: attentionRoles,
	},

	// heart-qwen: HEART's shared local Qwen backend (pkg/heartqwen).
	//
	// The configured served model is Qwen2.5-7B-Instruct, Q3_K_M GGUF
	// (KNIRVSERVER/config/testnet.yaml:25-28 and production.yaml:32-35 set
	// llama.model_url/model_name to qwen2.5-7b-instruct-q3_k_m.gguf). "7.61B" is its
	// size (pkg/knirvllama/README.md:84, 7B reference section at :81-93). TinyLlama-1.1B
	// is the documented default when nothing is configured (README.md:75-79), so it
	// is only the code fallback, not the configured model.
	//
	// None of the architecture dimensions are in the repo: no config.json, GGUF
	// header, or dimension constant. They are unset and must be filled from the
	// model's own config before this template can validate.
	BaseModelTemplateHeartQwen: {
		Name: BaseModelTemplateHeartQwen,
		Spec: BaseModelSpec{
			Family: BaseModelTemplateHeartQwen,
			// "7.61B Qwen2.5 Instruct model" (pkg/knirvllama/README.md:84), lowercased to
			// the size-label form the connector key uses.
			ParamCount: "7.61b",
			// TODO(hidden_size): not in repo.
			HiddenSize: 0,
			// TODO(intermediate_size): not in repo.
			IntermediateSize: 0,
			// TODO(num_attention_heads): not in repo.
			NumAttentionHeads: 0,
			// TODO(num_key_value_heads): not in repo.
			NumKeyValueHeads: 0,
			// TODO(head_dim): not in repo.
			HeadDim: 0,
			// TODO(num_layers): not in repo.
			NumLayers: 0,
			// TODO(attention_mechanism): not in repo.
			AttentionMechanism: "",
			// TODO(activation_func): not in repo.
			ActivationFunc: "",
		},
		// Q/K/V/O is the adapter surface ULoRA uses on every attention model
		// (the same four roles as heart-decision). Confirmed for the HEART decision model
		// only; the Qwen binding is not yet implemented.
		TargetModules: attentionRoles,
	},

	// hands-gpt: the HANDS backprop-trained GPT (pkg/hands/gpt).
	//
	// The defaults come from gpt.DefaultConfig (pkg/hands/gpt/gpt.go:55-66), which is
	// the config the trained model and HEART's Gorgonite config both use
	// (pkg/hashing/transformer/config.go:55).
	//
	// Conflict: pkg/hands/assets/defaults.json names a different architecture
	// (embed 256, heads 4, layers 6, ffn 1024, vocab 50257). Its only reader is
	// pkg/hands/embedded.go DefaultConfig(), which nothing in the repo calls, so it
	// is not used here. Flagged for the owner.
	BaseModelTemplateHandsGPT: {
		Name: BaseModelTemplateHandsGPT,
		Spec: BaseModelSpec{
			Family: BaseModelTemplateHandsGPT,
			// TODO(param_count): unset, pending owner decision. pruner.go:333-336 derives a
			// count from the Config fields, but it omits the attention weights; the
			// structural count differs. Owner must pick the convention. Like heart-decision,
			// the count is per trained model (from the checkpoint config), not a fixed value.
			ParamCount: "",
			// EmbedDim (gpt.go:58) = 768.
			HiddenSize: 768,
			// FFNHiddenDim (gpt.go:63) = 3072.
			IntermediateSize: 3072,
			// NumHeads (gpt.go:59) = 12.
			NumAttentionHeads: 12,
			// One key projection per head (gpt.go:108-109, :116): no grouping.
			NumKeyValueHeads: 12,
			// headDim = EmbedDim / NumHeads (gpt.go:97) = 64.
			HeadDim: 64,
			// NumLayers (gpt.go:60) = 12.
			NumLayers: 12,
			// Per-head query/key/value projections (gpt.go:73-75).
			AttentionMechanism: "mha",
			// The FFN uses tanh (gpt.go:373: gorgonia.Tanh). The comment there says GELU
			// would be closer, but the code uses tanh, so tanh is what is recorded.
			ActivationFunc: "tanh",
		},
		// wQuery, wKey, wValue, wOutput (gpt.go:73-76) map to the four attention roles.
		TargetModules: attentionRoles,
	},

	// head-vlm: HEAD's vision-language backend (pkg/vlm, minicpm backend).
	//
	// The served model is MiniCPM-V-4_7-35B-A3B, IQ4_XS quant (pkg/vlm/minicpm.go:31).
	// The name gives the size label (35B total, A3B active). Neither the architecture
	// dimensions nor any config.json are in the repo, so those fields are unset.
	BaseModelTemplateHeadVLM: {
		Name: BaseModelTemplateHeadVLM,
		Spec: BaseModelSpec{
			Family: BaseModelTemplateHeadVLM,
			// "35B-A3B" from the model name (pkg/vlm/minicpm.go:31), lowercased.
			ParamCount: "35b-a3b",
			// TODO(hidden_size): not in repo.
			HiddenSize: 0,
			// TODO(intermediate_size): not in repo.
			IntermediateSize: 0,
			// TODO(num_attention_heads): not in repo.
			NumAttentionHeads: 0,
			// TODO(num_key_value_heads): not in repo.
			NumKeyValueHeads: 0,
			// TODO(head_dim): not in repo.
			HeadDim: 0,
			// TODO(num_layers): not in repo.
			NumLayers: 0,
			// TODO(attention_mechanism): not in repo.
			AttentionMechanism: "",
			// TODO(activation_func): not in repo.
			ActivationFunc: "",
		},
		// TODO(target_modules): the vision-side adapter surface is not defined in the repo.
		TargetModules: nil,
	},
}

// LookupBaseModelTemplate returns the named template. The result is a copy, so a
// caller cannot change the shared table through it.
func LookupBaseModelTemplate(name string) (BaseModelTemplate, bool) {
	t, ok := baseModelTemplates[name]
	if !ok {
		return BaseModelTemplate{}, false
	}
	if t.TargetModules != nil {
		t.TargetModules = append([]string(nil), t.TargetModules...)
	}
	return t, true
}

// BaseModelTemplateNames lists every template name, in no particular order.
func BaseModelTemplateNames() []string {
	names := make([]string, 0, len(baseModelTemplates))
	for name := range baseModelTemplates {
		names = append(names, name)
	}
	return names
}
