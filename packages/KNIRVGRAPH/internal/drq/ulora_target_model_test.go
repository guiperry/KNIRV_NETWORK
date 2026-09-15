package drq

import (
	"strings"
	"testing"
)

func completeSpec() ULoRATargetModel {
	return ULoRATargetModel{
		Family:             "llama",
		ParamCount:         "8b",
		HiddenSize:         4096,
		IntermediateSize:   11008,
		NumAttentionHeads:  32,
		NumKeyValueHeads:   8,
		HeadDim:            128,
		NumLayers:          32,
		AttentionMechanism: "gqa",
		ActivationFunc:     "silu",
	}
}

func TestTargetModelSpecAcceptsCompleteArchitecture(t *testing.T) {
	if err := completeSpec().validate(); err != nil {
		t.Fatalf("a complete spec must validate, got: %v", err)
	}
}

// attention_mechanism and activation_func are required by ulora's
// BaseModelSpec.Validate, which runs when a bundle is opened. A spec missing them
// compiles into a bundle that can never be bound — the compile succeeds and the
// artifact is unusable — so they must be caught here.
func TestTargetModelSpecRequiresUloraValidatedFields(t *testing.T) {
	for _, field := range []string{"attention_mechanism", "activation_func"} {
		spec := completeSpec()
		switch field {
		case "attention_mechanism":
			spec.AttentionMechanism = ""
		case "activation_func":
			spec.ActivationFunc = ""
		}
		err := spec.validate()
		if err == nil {
			t.Fatalf("a spec with no %s must be rejected", field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("the error should name %s, got: %v", field, err)
		}
	}
}

// Whitespace is not a value.
func TestTargetModelSpecRejectsBlankFields(t *testing.T) {
	spec := completeSpec()
	spec.AttentionMechanism = "   "
	if err := spec.validate(); err == nil {
		t.Fatal("a whitespace-only attention_mechanism must be rejected")
	}
}

func TestTargetModelSpecRejectsNonPositiveDimensions(t *testing.T) {
	cases := map[string]func(*ULoRATargetModel){
		"hidden_size":       func(s *ULoRATargetModel) { s.HiddenSize = 0 },
		"intermediate_size": func(s *ULoRATargetModel) { s.IntermediateSize = -1 },
		"num_layers":        func(s *ULoRATargetModel) { s.NumLayers = 0 },
	}
	for field, mutate := range cases {
		spec := completeSpec()
		mutate(&spec)
		if err := spec.validate(); err == nil {
			t.Fatalf("a spec with a non-positive %s must be rejected", field)
		}
	}
}

// The corpus builder must refuse the spec rather than send it to the compiler.
func TestTargetModelsForCorpusRejectsSpecMissingUloraFields(t *testing.T) {
	dataset := []ULoRADatasetRecord{{Context: "c", CorrectedCompletion: "f", TargetModel: "llama-8b"}}

	spec := completeSpec()
	spec.ActivationFunc = ""

	_, err := targetModelsForCorpus(dataset, map[string]ULoRATargetModel{"llama-8b": spec})
	if err == nil {
		t.Fatal("a spec missing activation_func must be refused before compiling")
	}
	if !strings.Contains(err.Error(), "activation_func") {
		t.Fatalf("the error should name the missing field, got: %v", err)
	}
}

func TestTargetModelsForCorpusAcceptsCompleteSpec(t *testing.T) {
	dataset := []ULoRADatasetRecord{{Context: "c", CorrectedCompletion: "f", TargetModel: "llama-8b"}}

	models, err := targetModelsForCorpus(dataset, map[string]ULoRATargetModel{"llama-8b": completeSpec()})
	if err != nil {
		t.Fatalf("a complete spec must be accepted, got: %v", err)
	}
	if len(models) != 1 || models[0].AttentionMechanism != "gqa" {
		t.Fatalf("models = %+v", models)
	}
}
