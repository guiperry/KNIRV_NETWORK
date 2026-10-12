package transformer

import (
	"fmt"
	"strings"
)

type TinyGoTemplate struct {
	Name        string
	Source     string
	Exports    []string
	ImportPath string
}

func NewTinyGoTemplate(name string) *TinyGoTemplate {
	return &TinyGoTemplate{
		Name:        name,
		Source:     "",
		Exports:    []string{},
		ImportPath: "",
	}
}

func (t *TinyGoTemplate) WithExport(fn string) *TinyGoTemplate {
	t.Exports = append(t.Exports, fn)
	return t
}

func (t *TinyGoTemplate) WithImport(path string) *TinyGoTemplate {
	t.ImportPath = path
	return t
}

func (t *TinyGoTemplate) Build() string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	if t.ImportPath != "" {
		b.WriteString(fmt.Sprintf("import \"%s\"\n\n", t.ImportPath))
	}
	b.WriteString(t.Source)
	b.WriteString("\nfunc main() {}\n")
	return b.String()
}

type RuleTemplate struct {
	GuardrailFunc   string
	PolicyID    uint32
	Severity    uint8
	Action     uint8
	Confidence float32
}

func NewRuleTemplate() *RuleTemplate {
	return &RuleTemplate{
		GuardrailFunc: "GuardrailClass",
		PolicyID:    0,
		Severity:    0,
		Action:     0,
		Confidence: 0.0,
	}
}

func (rt *RuleTemplate) WithPolicyID(id uint32) *RuleTemplate {
	rt.PolicyID = id
	return rt
}

func (rt *RuleTemplate) WithSeverity(sev uint8) *RuleTemplate {
	rt.Severity = sev
	return rt
}

func (rt *RuleTemplate) WithAction(action uint8) *RuleTemplate {
	rt.Action = action
	return rt
}

func (rt *RuleTemplate) WithConfidence(conf float32) *RuleTemplate {
	rt.Confidence = conf
	return rt
}

func (rt *RuleTemplate) Build() string {
	return fmt.Sprintf(`package main

//export %s
func %s() uint32 {
	return %d
}

//export PolicyID
func PolicyID() uint32 {
	return %d
}

//export Severity
func Severity() uint8 {
	return %d
}

//export Action
func Action() uint8 {
	return %d
}

//export Confidence
func Confidence() float32 {
	return %.4f
}

func main() {}
`,
		rt.GuardrailFunc, rt.GuardrailFunc, rt.PolicyID,
		rt.PolicyID,
		rt.Severity,
		rt.Action,
		rt.Confidence,
	)
}

type ResolutionTemplate struct {
	ErrorClassFunc string
	ErrorCode  uint32
	Hint      string
	Severity   uint8
	Retryable bool
}

func NewResolutionTemplate() *ResolutionTemplate {
	return &ResolutionTemplate{
		ErrorClassFunc: "ErrorClass",
		ErrorCode:     0,
		Hint:         "",
		Severity:     0,
		Retryable:   false,
	}
}

func (rt *ResolutionTemplate) WithErrorCode(code uint32) *ResolutionTemplate {
	rt.ErrorCode = code
	return rt
}

func (rt *ResolutionTemplate) WithHint(hint string) *ResolutionTemplate {
	rt.Hint = hint
	return rt
}

func (rt *ResolutionTemplate) WithSeverity(sev uint8) *ResolutionTemplate {
	rt.Severity = sev
	return rt
}

func (rt *ResolutionTemplate) WithRetryable(retry bool) *ResolutionTemplate {
	rt.Retryable = retry
	return rt
}

func (rt *ResolutionTemplate) Build() string {
	retryVal := 0
	if rt.Retryable {
		retryVal = 1
	}
	return fmt.Sprintf(`package main

//export %s
func %s() uint32 {
	return %d
}

//export ErrorCode
func ErrorCode() uint32 {
	return %d
}

//export Hint
func Hint() uint32 {
	return uint32(uintptr(unsafe.Pointer("%s")))
}

//export ErrorSeverity
func ErrorSeverity() uint8 {
	return %d
}

//export Retryable
func Retryable() uint8 {
	return %d
}

func main() {}
`,
		rt.ErrorClassFunc, rt.ErrorClassFunc, rt.ErrorCode,
		rt.ErrorCode,
		rt.Hint,
		rt.Severity,
		retryVal,
	)
}

type PatchTemplate struct {
	PatchFunc       string
	TargetComponent string
	PatchData      string
	Checksum      uint32
	ApplyPriority uint8
}

func NewPatchTemplate() *PatchTemplate {
	return &PatchTemplate{
		PatchFunc:       "PatchScope",
		TargetComponent: "",
		PatchData:       "",
		Checksum:       0,
		ApplyPriority:  0,
	}
}

func (pt *PatchTemplate) WithTargetComponent(comp string) *PatchTemplate {
	pt.TargetComponent = comp
	return pt
}

func (pt *PatchTemplate) WithPatchData(data string) *PatchTemplate {
	pt.PatchData = data
	return pt
}

func (pt *PatchTemplate) WithChecksum(checksum uint32) *PatchTemplate {
	pt.Checksum = checksum
	return pt
}

func (pt *PatchTemplate) WithApplyPriority(priority uint8) *PatchTemplate {
	pt.ApplyPriority = priority
	return pt
}

func (pt *PatchTemplate) Build() string {
	return fmt.Sprintf(`package main

//export %s
func %s() uint32 {
	return %d
}

//export TargetComponent
func TargetComponent() uint32 {
	return uint32(uintptr(unsafe.Pointer("%s")))
}

//export PatchData
func PatchData() uint32 {
	return uint32(uintptr(unsafe.Pointer("%s")))
}

//export Checksum
func Checksum() uint32 {
	return %d
}

//export ApplyPriority
func ApplyPriority() uint8 {
	return %d
}

func main() {}
`,
		pt.PatchFunc, pt.PatchFunc, pt.ApplyPriority,
		pt.TargetComponent,
		pt.PatchData,
		pt.Checksum,
		pt.ApplyPriority,
	)
}

func GenerateTinyGoSourceForInquiry(inquiry interface{}, wasmType WASMType) string {
	switch wasmType {
	case WASMTypeRule:
		if p, ok := inquiry.(PolicyBadgeInquiry); ok {
			rt := NewRuleTemplate().
				WithPolicyID(uint32(hashStringToID(p.Name))).
				WithSeverity(uint8(len(p.ValuesSignals))).
				WithConfidence(0.5)
			return rt.Build()
		}
		return NewRuleTemplate().Build()

	case WASMTypeResolution:
		if d, ok := inquiry.(DVEErrorInquiry); ok {
			rt := NewResolutionTemplate().
				WithErrorCode(uint32(hashStringToID(d.ErrorType))).
				WithHint(d.ErrorMessage).
				WithSeverity(uint8(len(d.ErrorType))).
				WithRetryable(d.ErrorType == "retryable")
			return rt.Build()
		}
		return NewResolutionTemplate().Build()

	case WASMTypePatch:
		if p, ok := inquiry.(SystemPatchInquiry); ok {
			pt := NewPatchTemplate().
				WithTargetComponent(p.ComponentID).
				WithPatchData(p.SystemState).
				WithChecksum(crc32String(p.ErrorCode)).
				WithApplyPriority(1)
			return pt.Build()
		}
		return NewPatchTemplate().Build()
	}
	return ""
}

func hashStringToID(s string) uint32 {
	h := uint32(5381)
	for _, c := range []byte(s) {
		h = h*33 + uint32(c)
	}
	return h
}

func crc32String(s string) uint32 {
	h := uint32(0x811c9dc5)
	for _, c := range []byte(s) {
		h ^= uint32(c)
		h *= 0x01000193
	}
	return h
}

type BatchTemplate struct {
	Templates []string
	Count    int
}

func NewBatchTemplate() *BatchTemplate {
	return &BatchTemplate{
		Templates: []string{},
		Count:    0,
	}
}

func (bt *BatchTemplate) AddTemplate(tmpl string) *BatchTemplate {
	bt.Templates = append(bt.Templates, tmpl)
	bt.Count++
	return bt
}

func (bt *BatchTemplate) BuildAll() []string {
	return bt.Templates
}

func (bt *BatchTemplate) Merge() string {
	if len(bt.Templates) == 0 {
		return "package main\n\nfunc main() {}"
	}
	var merged strings.Builder
	merged.WriteString("package main\n\n")

	var exports []string
	var imports []string

	for _, tmpl := range bt.Templates {
		lines := strings.Split(tmpl, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "import") && !strings.Contains(merged.String(), line) {
				imports = append(imports, line)
			}
			if strings.HasPrefix(line, "//export") {
				exports = append(exports, line)
			}
		}
	}

	for _, imp := range imports {
		merged.WriteString(imp + "\n")
	}
	merged.WriteString("\n")

	for _, exp := range exports {
		merged.WriteString(exp + "\n")
	}

	merged.WriteString("\nfunc main() {}\n")
	return merged.String()
}