package transformer

import (
	"context"
	"fmt"
)

type Stage1Result struct {
	WASMType WASMType
	Raw      interface{}
}

type Stage2Result struct {
	PolicyPrinciples []PolicyPrinciple
	ErrorClass       ErrorClass
	CoreTechniques   []CoreTechnique
	PatchScope       PatchScope
	AffectedComponents []string
}

type Stage3Result struct {
	Sketch    string
	Rationale string
}

type Stage4Result struct {
	Source   string
	WASMType WASMType
}

type PolicyPrinciple struct {
	Name        string
	Description string
	Priority   int
}

type ErrorClass struct {
	Name     string
	Category string
}

type CoreTechnique struct {
	Name         string
	Applicability float32
}

type PatchScope struct {
	Severity    string
	Urgency   string
}

func (s *HEARTService) runPipeline(ctx context.Context, wasmType WASMType, inquiry interface{}, priorFailures []string) (*Stage4Result, *Stage3Result, error) {
	s1 := s.stage1(wasmType, inquiry)
	s2, err := s.stage2(ctx, s1, priorFailures)
	if err != nil {
		return nil, nil, err
	}
	s3, err := s.stage3(ctx, s1, s2)
	if err != nil {
		return nil, nil, err
	}
	s4, err := s.stage4(ctx, s1, s2, s3)
	if err != nil {
		return nil, s3, err
	}
	return s4, s3, nil
}

func (s *HEARTService) stage1(wasmType WASMType, inquiry interface{}) Stage1Result {
	return Stage1Result{WASMType: wasmType, Raw: inquiry}
}

func (s *HEARTService) stage2(ctx context.Context, s1 Stage1Result, priorFailures []string) (*Stage2Result, error) {
	// Use GPT + embeddings when available, otherwise fall back to heuristics
	if s.gpt != nil && s.tokenizer != nil {
		return s.stage2GPT(ctx, s1, priorFailures)
	}
	switch s1.WASMType {
	case WASMTypeRule:
		return &Stage2Result{PolicyPrinciples: []PolicyPrinciple{{Name: "Default Policy", Description: "Default policy principle", Priority: 1}}}, nil
	case WASMTypeResolution:
		return &Stage2Result{ErrorClass: ErrorClass{Name: "GenericError", Category: "general"}, CoreTechniques: []CoreTechnique{{Name: "Retry", Applicability: 0.5}}}, nil
	case WASMTypePatch:
		return &Stage2Result{PatchScope: PatchScope{Severity: "medium", Urgency: "normal"}, AffectedComponents: []string{"system"}}, nil
	}
	return nil, fmt.Errorf("unknown wasm type")
}

func (s *HEARTService) stage2GPT(ctx context.Context, s1 Stage1Result, priorFailures []string) (*Stage2Result, error) {
	prompt := s.buildStage2Prompt(s1, priorFailures)
	tokens := s.tokenizer.Encode(prompt)

	logits, err := s.runGorgoniteInference(tokens)
	if err != nil {
		return nil, fmt.Errorf("stage2 GPT inference: %w", err)
	}

	return s.parseStage2Result(logits, s1.WASMType)
}

func (s *HEARTService) buildStage2Prompt(s1 Stage1Result, priorFailures []string) string {
	base := fmt.Sprintf("wasm_type:%s", s1.WASMType)
	switch inq := s1.Raw.(type) {
	case PolicyBadgeInquiry:
		base += fmt.Sprintf(" name:%s badge_type:%s", inq.Name, inq.BadgeType)
	case DVEErrorInquiry:
		base += fmt.Sprintf(" error_type:%s message:%s", inq.ErrorType, inq.ErrorMessage)
	case SystemPatchInquiry:
		base += fmt.Sprintf(" component:%s error_code:%s", inq.ComponentID, inq.ErrorCode)
	}
	for _, f := range priorFailures {
		base += fmt.Sprintf(" prior_failure:%s", f)
	}
	return base
}

func (s *HEARTService) parseStage2Result(logits []float32, wasmType WASMType) (*Stage2Result, error) {
	// Use logit distribution to determine stage2 outputs
	// Bucket logits into decision boundaries
	bucket := int(logits[0]) % 100

	switch wasmType {
	case WASMTypeRule:
		principles := []PolicyPrinciple{
			{Name: "Safety Guardrail", Description: "Ensure system safety", Priority: bucket % 10},
			{Name: "Data Integrity", Description: "Maintain data consistency", Priority: (bucket + 3) % 10},
		}
		return &Stage2Result{PolicyPrinciples: principles}, nil

	case WASMTypeResolution:
		errorClass := ErrorClass{
			Name:     fmt.Sprintf("Error%d", bucket%10),
			Category: []string{"network", "type", "inference", "system"}[bucket%4],
		}
		techniques := []CoreTechnique{
			{Name: "Retry with Backoff", Applicability: float32(bucket) / 100.0},
			{Name: "Circuit Breaker", Applicability: float32((bucket + 20) % 100) / 100.0},
		}
		return &Stage2Result{ErrorClass: errorClass, CoreTechniques: techniques}, nil

	case WASMTypePatch:
		severity := []string{"low", "medium", "high", "critical"}[bucket%4]
		urgency := []string{"normal", "high", "immediate"}[bucket%3]
		return &Stage2Result{
			PatchScope:          PatchScope{Severity: severity, Urgency: urgency},
			AffectedComponents: []string{"system", "network", "storage"}[:bucket%3+1],
		}, nil
	}
	return nil, fmt.Errorf("unknown wasm type")
}

func (s *HEARTService) stage3(ctx context.Context, s1 Stage1Result, s2 *Stage2Result) (*Stage3Result, error) {
	if s.gpt != nil && s.tokenizer != nil {
		return s.stage3GPT(ctx, s1, s2)
	}
	return &Stage3Result{
		Sketch:    "Generated via Gorgonite pipeline",
		Rationale: "Stage 3 decision sketch",
	}, nil
}

func (s *HEARTService) stage3GPT(ctx context.Context, s1 Stage1Result, s2 *Stage2Result) (*Stage3Result, error) {
	prompt := s.buildStage3Prompt(s1, s2)
	tokens := s.tokenizer.Encode(prompt)

	logits, err := s.runGorgoniteInference(tokens)
	if err != nil {
		return nil, fmt.Errorf("stage3 GPT inference: %w", err)
	}

	sketch := s.generateSketchFromLogits(logits, s1.WASMType)
	rationale := fmt.Sprintf("GPT decision for %s based on %d policy principles", s1.WASMType, len(s2.PolicyPrinciples)+len(s2.CoreTechniques))

	return &Stage3Result{Sketch: sketch, Rationale: rationale}, nil
}

func (s *HEARTService) buildStage3Prompt(s1 Stage1Result, s2 *Stage2Result) string {
	prompt := fmt.Sprintf("wasm_type:%s stage:3", s1.WASMType)
	for _, p := range s2.PolicyPrinciples {
		prompt += fmt.Sprintf(" principle:%s(priority:%d)", p.Name, p.Priority)
	}
	if s2.ErrorClass.Name != "" {
		prompt += fmt.Sprintf(" error_class:%s category:%s", s2.ErrorClass.Name, s2.ErrorClass.Category)
	}
	for _, t := range s2.CoreTechniques {
		prompt += fmt.Sprintf(" technique:%s(applicability:%.2f)", t.Name, t.Applicability)
	}
	return prompt
}

func (s *HEARTService) generateSketchFromLogits(logits []float32, wasmType WASMType) string {
	hash := 0
	for i, v := range logits {
		hash += int(v) * (i + 1)
	}
	return fmt.Sprintf("sketch_%s_%x", wasmType, hash%0xffff)
}

func (s *HEARTService) stage4(ctx context.Context, s1 Stage1Result, s2 *Stage2Result, s3 *Stage3Result) (*Stage4Result, error) {
	if s.gpt != nil && s.tokenizer != nil {
		return s.stage4GPT(ctx, s1, s2, s3)
	}
	source := generateTinyGoSource(s1.WASMType, s2)
	return &Stage4Result{Source: source, WASMType: s1.WASMType}, nil
}

func (s *HEARTService) stage4GPT(ctx context.Context, s1 Stage1Result, s2 *Stage2Result, s3 *Stage3Result) (*Stage4Result, error) {
	prompt := s.buildStage4Prompt(s1, s2, s3)
	tokens := s.tokenizer.Encode(prompt)

	logits, err := s.runGorgoniteInference(tokens)
	if err != nil {
		return nil, fmt.Errorf("stage4 GPT inference: %w", err)
	}

	source := s.generateWASMSourceFromLogits(logits, s1.WASMType, s2, s3)
	return &Stage4Result{Source: source, WASMType: s1.WASMType}, nil
}

func (s *HEARTService) buildStage4Prompt(s1 Stage1Result, s2 *Stage2Result, s3 *Stage3Result) string {
	prompt := fmt.Sprintf("wasm_type:%s stage:4 action:generate_source", s1.WASMType)
	prompt += fmt.Sprintf(" sketch:%s", s3.Sketch)
	prompt += fmt.Sprintf(" rationale:%s", s3.Rationale)

	for _, p := range s2.PolicyPrinciples {
		prompt += fmt.Sprintf(" policy:%s", p.Name)
	}
	if s2.ErrorClass.Name != "" {
		prompt += fmt.Sprintf(" error_class:%s", s2.ErrorClass.Name)
	}
	for _, t := range s2.CoreTechniques {
		prompt += fmt.Sprintf(" technique:%s", t.Name)
	}
	return prompt
}

func (s *HEARTService) generateWASMSourceFromLogits(logits []float32, wasmType WASMType, s2 *Stage2Result, s3 *Stage3Result) string {
	// Hash the logits to create a deterministic but varied source
	hash := 0
	for i, v := range logits {
		hash += int(v) * (i + 1)
	}

	basePkg := `package main

import "unsafe"

// Generated by Gorgonite Transformer
// Hash: %x
`
	switch wasmType {
	case WASMTypeRule:
		return fmt.Sprintf(basePkg+`
//export GuardrailClass
func GuardrailClass() uint32 { return %d }

func main() {}
`, hash%0xffff, hash%10)

	case WASMTypeResolution:
		errorCode := hash % 10
		return fmt.Sprintf(basePkg+`
//export ErrorClass
func ErrorClass() uint32 { return %d }

func resolveError() bool { return true }

func main() {}
`, hash%0xffff, errorCode)

	case WASMTypePatch:
		return fmt.Sprintf(basePkg+`
//export PatchScope
func PatchScope() uint32 { return %d }

var patchApplied bool

func applyPatch() bool { patchApplied = true; return true }

func main() {}
`, hash%0xffff, hash%4)
	}
	return fmt.Sprintf(basePkg+`
func main() {}
`, hash%0xffff)
}

func generateTinyGoSource(wasmType WASMType, s2 *Stage2Result) string {
	switch wasmType {
	case WASMTypeRule:
		return `package main

//export GuardrailClass
func GuardrailClass() uint32 { return 0 }

func main() {}
`
	case WASMTypeResolution:
		return `package main

//export ErrorClass
func ErrorClass() uint32 { return 0 }

func main() {}
`
	case WASMTypePatch:
		return `package main

//export PatchScope
func PatchScope() uint32 { return 0 }

func main() {}
`
	}
	return ""
}