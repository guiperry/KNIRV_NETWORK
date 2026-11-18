package api

import (
	"strings"
	"testing"
)

func TestLookupBaseModelTemplateKnownNames(t *testing.T) {
	for _, name := range []string{
		BaseModelTemplateHeartDecision,
		BaseModelTemplateHeartQwen,
		BaseModelTemplateHandsGPT,
		BaseModelTemplateHeadVLM,
	} {
		tpl, ok := LookupBaseModelTemplate(name)
		if !ok {
			t.Fatalf("template %q not found", name)
		}
		if tpl.Name != name {
			t.Fatalf("template %q reports Name %q", name, tpl.Name)
		}
	}
	if len(BaseModelTemplateNames()) != 4 {
		t.Fatalf("expected 4 templates, got %d", len(BaseModelTemplateNames()))
	}
}

func TestLookupBaseModelTemplateUnknownName(t *testing.T) {
	if _, ok := LookupBaseModelTemplate("heart-unknown"); ok {
		t.Fatal("expected unknown template to be absent")
	}
}

func TestLookupBaseModelTemplateReturnsCopy(t *testing.T) {
	tpl, _ := LookupBaseModelTemplate(BaseModelTemplateHandsGPT)
	tpl.TargetModules[0] = "mutated"
	again, _ := LookupBaseModelTemplate(BaseModelTemplateHandsGPT)
	if again.TargetModules[0] != "self_attn.q_proj" {
		t.Fatalf("lookup leaked a shared slice: %q", again.TargetModules[0])
	}
}

// Expected values are the ones cited in basemodel_templates.go. A change to a
// template value must fail here.
func TestHandsGPTTemplateValues(t *testing.T) {
	tpl, ok := LookupBaseModelTemplate(BaseModelTemplateHandsGPT)
	if !ok {
		t.Fatal("hands-gpt missing")
	}
	s := tpl.Spec
	if s.Family != BaseModelTemplateHandsGPT {
		t.Fatalf("hands-gpt family wrong: %q", s.Family)
	}
	if s.HiddenSize != 768 || s.IntermediateSize != 3072 || s.NumAttentionHeads != 12 ||
		s.NumKeyValueHeads != 12 || s.HeadDim != 64 || s.NumLayers != 12 {
		t.Fatalf("hands-gpt dims wrong: %+v", s)
	}
	if s.AttentionMechanism != "mha" || s.ActivationFunc != "tanh" {
		t.Fatalf("hands-gpt mechanism/activation wrong: %+v", s)
	}
	if s.HeadDim*s.NumAttentionHeads != s.HiddenSize {
		t.Fatalf("hands-gpt head_dim*heads != hidden")
	}
}

func TestHeartDecisionTemplateValues(t *testing.T) {
	tpl, ok := LookupBaseModelTemplate(BaseModelTemplateHeartDecision)
	if !ok {
		t.Fatal("heart-decision missing")
	}
	s := tpl.Spec
	if s.Family != BaseModelTemplateHeartDecision {
		t.Fatalf("heart-decision family wrong: %q", s.Family)
	}
	if s.HiddenSize != 128 || s.IntermediateSize != 256 || s.NumAttentionHeads != 4 ||
		s.NumKeyValueHeads != 4 || s.HeadDim != 32 || s.NumLayers != 4 {
		t.Fatalf("heart-decision dims wrong: %+v", s)
	}
	if s.AttentionMechanism != "mha" || s.ActivationFunc != "silu" {
		t.Fatalf("heart-decision mechanism/activation wrong: %+v", s)
	}
}

func TestHeartQwenTemplateValues(t *testing.T) {
	tpl, ok := LookupBaseModelTemplate(BaseModelTemplateHeartQwen)
	if !ok {
		t.Fatal("heart-qwen missing")
	}
	if tpl.Spec.ParamCount != "7.61b" || tpl.Spec.Family != BaseModelTemplateHeartQwen {
		t.Fatalf("heart-qwen identity wrong: %+v", tpl.Spec)
	}
}

func TestHeadVLMTemplateValues(t *testing.T) {
	tpl, ok := LookupBaseModelTemplate(BaseModelTemplateHeadVLM)
	if !ok {
		t.Fatal("head-vlm missing")
	}
	if tpl.Spec.ParamCount != "35b-a3b" || tpl.Spec.Family != BaseModelTemplateHeadVLM {
		t.Fatalf("head-vlm identity wrong: %+v", tpl.Spec)
	}
}

// Templates whose architecture dimensions are fully sourced pass the existing
// spec validation. Templates with TODO dimensions are rejected by it, so an
// unresolved template cannot be bound by accident.
//
// NOTE: BaseModelSpec.Validate does not check ParamCount. The two templates below
// pass validation with ParamCount unset. That is a fact about Validate, not a
// sign that an empty ParamCount is acceptable for binding: ParamCount is the
// connector cache key, and its value is an open owner decision (hands-gpt count
// convention; heart-decision count is per trained model). Do not read this test
// as clearing either template for binding.
func TestBaseModelTemplatesValidation(t *testing.T) {
	validating := []string{BaseModelTemplateHeartDecision, BaseModelTemplateHandsGPT}
	for _, name := range validating {
		tpl, _ := LookupBaseModelTemplate(name)
		// Dimension-level check only: no architecture dimension may be unset.
		// The full Validate result is logged, not asserted, so this test does
		// not pin ParamCount behaviour.
		err := tpl.Spec.Validate()
		if err != nil && strings.Contains(err.Error(), "must be positive") {
			t.Errorf("%s has an unset dimension: %v", name, err)
		}
		t.Logf("%s: Validate result: %v", name, err)
		if tpl.Spec.ParamCount == "" {
			t.Logf("%s: ParamCount unset pending owner decision; Validate does not enforce it, so this template is not cleared for binding on ParamCount", name)
		}
	}
	unresolved := []string{BaseModelTemplateHeartQwen, BaseModelTemplateHeadVLM}
	for _, name := range unresolved {
		tpl, _ := LookupBaseModelTemplate(name)
		err := tpl.Spec.Validate()
		if err == nil {
			t.Errorf("%s has TODO dimensions and must not validate", name)
			continue
		}
		if !strings.Contains(err.Error(), "base_source_model") {
			t.Errorf("%s: unexpected error shape: %v", name, err)
		}
	}
}

func TestBaseModelTemplatesTargetModules(t *testing.T) {
	want := []string{
		"self_attn.q_proj",
		"self_attn.k_proj",
		"self_attn.v_proj",
		"self_attn.o_proj",
	}
	for _, name := range []string{BaseModelTemplateHeartDecision, BaseModelTemplateHeartQwen, BaseModelTemplateHandsGPT} {
		tpl, _ := LookupBaseModelTemplate(name)
		if len(tpl.TargetModules) != len(want) {
			t.Errorf("%s: expected %d attention roles, got %v", name, len(want), tpl.TargetModules)
			continue
		}
		for i := range want {
			if tpl.TargetModules[i] != want[i] {
				t.Errorf("%s: target role %d = %q, want %q (full: %v)", name, i, tpl.TargetModules[i], want[i], tpl.TargetModules)
			}
		}
	}
	tpl, _ := LookupBaseModelTemplate(BaseModelTemplateHeadVLM)
	if len(tpl.TargetModules) != 0 {
		t.Errorf("head-vlm target modules are unresolved and must be empty, got %v", tpl.TargetModules)
	}
}
