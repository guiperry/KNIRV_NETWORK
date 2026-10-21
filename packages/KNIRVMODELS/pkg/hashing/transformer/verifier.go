package transformer

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"os"
	"regexp"
	"strings"
	"time"
)

type BidirectionalVerifier struct {
	wazeroGate *WazeroGate
	hashNet    *HashNetworkWrapper
	confidence float32
}

func NewBidirectionalVerifier() *BidirectionalVerifier {
	return &BidirectionalVerifier{
		wazeroGate: NewWazeroGate(),
		hashNet:    NewHashNetworkWrapper(),
		confidence: 0.0,
	}
}

func (bv *BidirectionalVerifier) Verify(source, wasmType string) (bool, float32, string) {
	forwardPassed, forwardMsg := bv.ForwardVerify(source, wasmType)
	backwardPassed, backwardMsg := bv.BackwardVerify(source, wasmType)

	bv.confidence = calculateConfidence(forwardPassed, backwardPassed, forwardMsg, backwardMsg)

	if forwardPassed && backwardPassed {
		return true, bv.confidence, fmt.Sprintf("Forward: %s | Backward: %s", forwardMsg, backwardMsg)
	}

	return false, bv.confidence, fmt.Sprintf("Forward: %s | Backward: %s", forwardMsg, backwardMsg)
}

// VerifyBinary is the verification that can actually pass: the forward pass loads
// the compiled .wasm in wazero, instantiates it and calls the export the module
// type requires; the backward pass re-derives the contract from the source it was
// built from. (Verify, which feeds Go source to wazero as if it were WASM bytes,
// fails on every input: "invalid magic number".)
func (bv *BidirectionalVerifier) VerifyBinary(source, wasmType, wasmPath string) (bool, float32, string) {
	fwd, fwdMsg := forwardBinary(wasmPath, wasmType)
	bwd, bwdMsg := bv.verifyBackward(source, wasmType)
	conf := calculateConfidence(fwd, bwd, fwdMsg, bwdMsg)
	return fwd && bwd, conf, fmt.Sprintf("Forward: %s | Backward: %s", fwdMsg, bwdMsg)
}

func forwardBinary(wasmPath, wasmType string) (bool, string) {
	exp := expectedExport(wasmType)
	if exp == "" {
		return false, "unknown module type"
	}
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		return false, "wazero compile failed: " + err.Error()
	}
	// A fresh runtime per call: nothing is shared between verifications.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)
	// TinyGo modules import WASI; give them the minimal host with no files, args or network.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return false, "wazero WASI host failed: " + err.Error()
	}
	compiled, err := rt.CompileModule(ctx, data)
	if err != nil {
		return false, "wazero compile failed: " + err.Error()
	}
	// Do not run main: it exits and closes the module before the export can be called.
	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName("verify").WithStartFunctions())
	if err != nil {
		return false, "wazero instantiate failed: " + err.Error()
	}
	fn := mod.ExportedFunction(exp)
	if fn == nil {
		return false, "no valid exports found: " + exp + " is not exported"
	}
	if _, err := fn.Call(ctx); err != nil {
		return false, "wazero call " + exp + " failed: " + err.Error()
	}
	return true, "forward pass OK"
}

// ForwardVerify runs the forward pass verification (Phase 7).
func (bv *BidirectionalVerifier) ForwardVerify(source, wasmType string) (bool, string) {
	return bv.verifyForward(source, wasmType)
}

// BackwardVerify runs the backward pass verification (Phase 7).
func (bv *BidirectionalVerifier) BackwardVerify(source, wasmType string) (bool, string) {
	return bv.verifyBackward(source, wasmType)
}

func (bv *BidirectionalVerifier) verifyForward(source, wasmType string) (bool, string) {
	valid, errMsg := bv.wazeroGate.Validate(source)
	if !valid {
		return false, fmt.Sprintf("wazero compile failed: %s", errMsg)
	}

	if !hasValidExports(source, wasmType) {
		return false, "no valid exports found"
	}

	return true, "forward pass OK"
}

// verifyBackward re-derives the contract from the source instead of trusting
// the forward pass: the source must hash to a stable non-empty digest, and the
// export the forward pass requires must be bound to a real func of that name.
// It never passes by default; it used to pass whenever the hash network was
// unavailable (S1).
func (bv *BidirectionalVerifier) verifyBackward(source, wasmType string) (bool, string) {
	return backwardCheck(bv.hashNet, source, wasmType)
}

func backwardCheck(hn *HashNetworkWrapper, source, wasmType string) (bool, string) {
	if hn == nil {
		hn = NewHashNetworkWrapper()
	}
	if hn.ComputeHash(source) == "" {
		return false, "backward: empty source"
	}
	if !matchesBackwardInvariant(source, wasmType) {
		return false, "backward invariant violation: export not bound to a func"
	}
	return true, "backward pass OK"
}

func expectedExport(wasmType string) string {
	switch wasmType {
	case "rule":
		return "GuardrailClass"
	case "resolution":
		return "ErrorClass"
	case "patch":
		return "PatchScope"
	}
	return ""
}

func hasValidExports(source, wasmType string) bool {
	exp := expectedExport(wasmType)
	return exp != "" && strings.Contains(source, "//export "+exp)
}

// matchesBackwardInvariant requires `//export X` to be immediately followed by
// `func X(` for the export the forward pass demands.
func matchesBackwardInvariant(source, wasmType string) bool {
	exp := expectedExport(wasmType)
	if exp == "" {
		return false
	}
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^//export %s\s*\n\s*func %s\(`, exp, exp))
	return re.MatchString(source)
}

func calculateConfidence(forwardPassed, backwardPassed bool, forwardMsg, backwardMsg string) float32 {
	confidence := float32(0.0)

	if forwardPassed {
		confidence += 0.5
	}
	if backwardPassed {
		confidence += 0.5
	}

	if strings.Contains(forwardMsg, "OK") && strings.Contains(backwardMsg, "OK") {
		confidence = 1.0
	}

	return confidence
}

type ForwardVerifier struct {
	wazeroGate *WazeroGate
}

func NewForwardVerifier() *ForwardVerifier {
	return &ForwardVerifier{
		wazeroGate: NewWazeroGate(),
	}
}

func (fv *ForwardVerifier) Verify(source string) (bool, string, error) {
	valid, errMsg := fv.wazeroGate.Validate(source)
	if !valid {
		return false, errMsg, nil
	}

	results, err := fv.wazeroGate.Run()
	if err != nil {
		return false, err.Error(), err
	}

	if len(results) > 0 {
		return true, fmt.Sprintf("verified with %d exports", len(results)), nil
	}

	return false, "no exports", nil
}

func (fv *ForwardVerifier) Close() error {
	return fv.wazeroGate.Close()
}

type BackwardVerifier struct {
	hashNet *HashNetworkWrapper
}

func NewBackwardVerifier() *BackwardVerifier {
	return &BackwardVerifier{
		hashNet: NewHashNetworkWrapper(),
	}
}

func (bv *BackwardVerifier) Verify(source string, wasmType string) (bool, string) {
	return backwardCheck(bv.hashNet, source, wasmType)
}

type VerificationResult struct {
	Passed         bool
	Confidence     float32
	ForwardMsg     string
	BackwardMsg    string
	ForwardErrors  []string
	BackwardErrors []string
}

func NewVerificationResult() *VerificationResult {
	return &VerificationResult{
		ForwardErrors:  []string{},
		BackwardErrors: []string{},
	}
}

func (vr *VerificationResult) Merge(other *VerificationResult) {
	if other.Passed {
		vr.Passed = true
	}
	vr.Confidence = (vr.Confidence + other.Confidence) / 2

	if other.ForwardErrors != nil {
		vr.ForwardErrors = append(vr.ForwardErrors, other.ForwardErrors...)
	}
	if other.BackwardErrors != nil {
		vr.BackwardErrors = append(vr.BackwardErrors, other.BackwardErrors...)
	}
}

type VerifierPool struct {
	forwardVerifiers  []*ForwardVerifier
	backwardVerifiers []*BackwardVerifier
	size              int
	index             int
}

func NewVerifierPool(size int) *VerifierPool {
	return &VerifierPool{
		forwardVerifiers:  make([]*ForwardVerifier, size),
		backwardVerifiers: make([]*BackwardVerifier, size),
		size:              size,
		index:             0,
	}
}

func (vp *VerifierPool) AcquireForward() *ForwardVerifier {
	if vp.forwardVerifiers[vp.index] == nil {
		vp.forwardVerifiers[vp.index] = NewForwardVerifier()
	}

	fv := vp.forwardVerifiers[vp.index]
	vp.index = (vp.index + 1) % vp.size

	return fv
}

func (vp *VerifierPool) AcquireBackward() *BackwardVerifier {
	if vp.backwardVerifiers[vp.index] == nil {
		vp.backwardVerifiers[vp.index] = NewBackwardVerifier()
	}

	bv := vp.backwardVerifiers[vp.index]
	vp.index = (vp.index + 1) % vp.size

	return bv
}

func (vp *VerifierPool) CloseAll() {
	for _, fv := range vp.forwardVerifiers {
		if fv != nil {
			_ = fv.Close()
		}
	}
}
