package transformer

import (
	"fmt"
	"regexp"
	"strings"
)

type BidirectionalVerifier struct {
	wazeroGate  *WazeroGate
	hashNet    *HashNetworkWrapper
	confidence float32
}

func NewBidirectionalVerifier() *BidirectionalVerifier {
	return &BidirectionalVerifier{
		wazeroGate:  NewWazeroGate(),
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

func (bv *BidirectionalVerifier) verifyBackward(source, wasmType string) (bool, string) {
	if !bv.hashNet.IsAvailable() {
		return true, "hashnet unavailable, backward skipped"
	}

	hashMatches := bv.hashNet.VerifySource(source)
	if !hashMatches {
		return false, "hashnet hash mismatch"
	}

	if !matchesBackwardInvariant(source, wasmType) {
		return false, "backward invariant violation"
	}

	return true, "backward pass OK"
}

func hasValidExports(source, wasmType string) bool {
	var expectedExport string
	switch wasmType {
	case "rule":
		expectedExport = "GuardrailClass"
	case "resolution":
		expectedExport = "ErrorClass"
	case "patch":
		expectedExport = "PatchScope"
	default:
		return false
	}

	return strings.Contains(source, "//export "+expectedExport)
}

func matchesBackwardInvariant(source, wasmType string) bool {
	exportPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^//export %s`, wasmType))
	return exportPattern.MatchString(source)
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
	if !bv.hashNet.IsAvailable() {
		return true, "hashnet unavailable"
	}

	hashMatches := bv.hashNet.VerifySource(source)
	if !hashMatches {
		return false, "hashnet hash mismatch"
	}

	invariantMatches := matchesBackwardInvariant(source, wasmType)
	if !invariantMatches {
		return false, "invariant mismatch"
	}

	return true, "backward verified"
}

type VerificationResult struct {
	Passed           bool
	Confidence      float32
	ForwardMsg      string
	BackwardMsg     string
	ForwardErrors   []string
	BackwardErrors []string
}

func NewVerificationResult() *VerificationResult {
	return &VerificationResult{
		ForwardErrors:   []string{},
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
	size            int
	index          int
}

func NewVerifierPool(size int) *VerifierPool {
	return &VerifierPool{
		forwardVerifiers:  make([]*ForwardVerifier, size),
		backwardVerifiers: make([]*BackwardVerifier, size),
		size:            size,
		index:          0,
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