package transformer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"unicode/utf8"
)

// Field limits. The model chooses values; code bounds them, renders the module
// from a fixed template (no model-written Go reaches the compiler), verifies it
// and only then compiles it.
const (
	maxHintLen      = 256
	maxComponentLen = 128
	maxPatchDataLen = 1024
)

type ruleFields struct {
	PolicyID   uint32  `json:"policy_id"`
	Severity   int     `json:"severity"`
	Action     int     `json:"action"`
	Confidence float64 `json:"confidence"`
}

type resolutionFields struct {
	ErrorCode uint32 `json:"error_code"`
	Hint      string `json:"hint"`
	Severity  int    `json:"severity"`
	Retryable bool   `json:"retryable"`
}

type patchFields struct {
	TargetComponent string `json:"target_component"`
	PatchData       string `json:"patch_data"`
	ApplyPriority   int    `json:"apply_priority"`
}

func strictDecode(raw json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data after the field object")
	}
	return nil
}

func byteRange(name string, v int) error {
	if v < 0 || v > 255 {
		return fmt.Errorf("%s %d is outside 0..255", name, v)
	}
	return nil
}

func boundedText(name, s string, max int) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s is not valid UTF-8", name)
	}
	if len(s) > max {
		return fmt.Errorf("%s is %d bytes, limit %d", name, len(s), max)
	}
	return nil
}

// RenderFields turns the model's field JSON into module source from the fixed
// templates. Unknown fields, out-of-range values and oversize text are errors,
// so a malformed choice is refused rather than clamped into something the model
// did not say.
func RenderFields(wasmType WASMType, raw json.RawMessage) (string, error) {
	switch wasmType {
	case WASMTypeRule:
		var f ruleFields
		if err := strictDecode(raw, &f); err != nil {
			return "", fmt.Errorf("rule fields: %w", err)
		}
		if err := byteRange("severity", f.Severity); err != nil {
			return "", err
		}
		if err := byteRange("action", f.Action); err != nil {
			return "", err
		}
		if f.Confidence < 0 || f.Confidence > 1 {
			return "", fmt.Errorf("confidence %v is outside 0..1", f.Confidence)
		}
		return NewRuleTemplate().WithPolicyID(f.PolicyID).WithSeverity(uint8(f.Severity)).
			WithAction(uint8(f.Action)).WithConfidence(float32(f.Confidence)).Build(), nil
	case WASMTypeResolution:
		var f resolutionFields
		if err := strictDecode(raw, &f); err != nil {
			return "", fmt.Errorf("resolution fields: %w", err)
		}
		if err := byteRange("severity", f.Severity); err != nil {
			return "", err
		}
		if err := boundedText("hint", f.Hint, maxHintLen); err != nil {
			return "", err
		}
		return NewResolutionTemplate().WithErrorCode(f.ErrorCode).WithHint(f.Hint).
			WithSeverity(uint8(f.Severity)).WithRetryable(f.Retryable).Build(), nil
	case WASMTypePatch:
		var f patchFields
		if err := strictDecode(raw, &f); err != nil {
			return "", fmt.Errorf("patch fields: %w", err)
		}
		if err := byteRange("apply_priority", f.ApplyPriority); err != nil {
			return "", err
		}
		if err := boundedText("target_component", f.TargetComponent, maxComponentLen); err != nil {
			return "", err
		}
		if err := boundedText("patch_data", f.PatchData, maxPatchDataLen); err != nil {
			return "", err
		}
		return NewPatchTemplate().WithTargetComponent(f.TargetComponent).WithPatchData(f.PatchData).
			WithChecksum(crc32String(f.PatchData)).WithApplyPriority(uint8(f.ApplyPriority)).Build(), nil
	}
	return "", fmt.Errorf("unknown module type %q", wasmType)
}

// WASMFinisher renders, verifies, compiles and audits a module from field
// choices. It is HEART's authoritative back end for every generator (Qwen or
// the GPT): the generator proposes fields, this decides.
type WASMFinisher struct {
	Compiler *WASMCompiler
	Verifier *BidirectionalVerifier
	Auditor  *Auditor
}

// NewWASMFinisher uses the same compiler, verifier and audit settings as HEART.
func NewWASMFinisher(cfg *HEARTConfig) *WASMFinisher {
	return &WASMFinisher{
		Compiler: NewWASMCompiler(cfg.TinyGoPath, cfg.WASMOutDir),
		Verifier: NewBidirectionalVerifier(),
		Auditor:  NewAuditor(cfg.AuditLogDir),
	}
}

// Check renders the fields and runs the cheap backward check on the source. It
// compiles nothing.
func (f *WASMFinisher) Check(wasmType WASMType, raw json.RawMessage) (source string, ok bool, msg string, err error) {
	source, err = RenderFields(wasmType, raw)
	if err != nil {
		return "", false, "", err
	}
	ok, msg = f.Verifier.BackwardVerify(source, string(wasmType))
	return source, ok, msg, nil
}

// Finish builds the decision: render, backward-check the source, compile, then
// verify the binary in wazero. A rejected choice stops before compilation, and a
// binary that fails the forward pass is deleted.
func (f *WASMFinisher) Finish(ctx context.Context, wasmType WASMType, raw json.RawMessage, rationale string) (WASMDecision, error) {
	d := WASMDecision{WASMType: wasmType, Rationale: rationale, TurnCount: 1}
	source, ok, msg, err := f.Check(wasmType, raw)
	if err != nil {
		return d, err
	}
	if !ok {
		d.BackwardVerifierMsg = msg
		return d, nil
	}
	res, err := f.Compiler.Compile(ctx, source, wasmType)
	if err != nil {
		log.Printf("WASM compile error: %v", err)
		d.BackwardVerifierMsg = "compile failed: " + firstLine(err.Error())
		return d, nil
	}
	passed, conf, vmsg := f.Verifier.VerifyBinary(source, string(wasmType), res.WASMPath)
	d.VerifierConfidence = conf
	if !passed {
		d.BackwardVerifierMsg = vmsg
		_ = os.Remove(res.WASMPath)
		return d, nil
	}
	d.BidirectionalVerified, d.WazeroExecPassed = true, true
	d.ForwardVerifierMsg = vmsg
	d.WASMPath, d.WASMHash = res.WASMPath, res.WASMHash
	if f.Auditor != nil {
		_ = f.Auditor.WriteAuditLog(&AuditRecord{InquiryHash: inquiryHashStr(source), WASMSha256: res.WASMHash,
			WASMType: wasmType, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	return d, nil
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
