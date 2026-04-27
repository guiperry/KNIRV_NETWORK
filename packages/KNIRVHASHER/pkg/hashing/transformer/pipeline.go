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

func (s *HEARTService) stage3(ctx context.Context, s1 Stage1Result, s2 *Stage2Result) (*Stage3Result, error) {
	return &Stage3Result{
		Sketch:    "Generated via Gorgonite pipeline",
		Rationale: "Stage 3 decision sketch",
	}, nil
}

func (s *HEARTService) stage4(ctx context.Context, s1 Stage1Result, s2 *Stage2Result, s3 *Stage3Result) (*Stage4Result, error) {
	source := generateTinyGoSource(s1.WASMType, s2)
	return &Stage4Result{Source: source, WASMType: s1.WASMType}, nil
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