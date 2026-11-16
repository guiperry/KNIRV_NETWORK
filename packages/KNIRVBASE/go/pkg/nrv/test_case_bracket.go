package nrv

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// ErrorTestSuiteBracketSlots is the number of test positions in a sealed
// ErrorTestSuite (plan D5.1: exactly 8 brackets per suite, positions 0-7).
const ErrorTestSuiteBracketSlots = 8

// ErrorTestCase is the minimal input EncodeTestCaseToNRV needs. It is defined
// here so KNIRVBASE does not import KNIRVGRAPH (where the full test-case type
// lives). Only the fields the bracket encoding uses are present.
type ErrorTestCase struct {
	// Input is the test input text. It is digested into the 32-byte projection
	// block, so it is never truncated; empty input is rejected.
	Input string
	// DomainSig is the ErrorNode domain signature (bracket bytes 0x27-0x28).
	DomainSig uint16
	// IntentFlags is the intent byte (bracket byte 0x26).
	IntentFlags uint8
	// Position is the test's index in its suite, 0-7 (PosIndex half of LSHSalt).
	Position uint8
	// SubSecondUS is the caller-supplied sub-second ticker (bytes 0x20-0x23).
	// Supplied explicitly so the encoding stays deterministic.
	SubSecondUS uint32
}

// ErrTestCaseEmptyInput is returned when ErrorTestCase.Input is empty.
var ErrTestCaseEmptyInput = errors.New("nrv: test case input is empty")

// ErrTestCasePosition is returned when ErrorTestCase.Position is outside 0-7.
var ErrTestCasePosition = errors.New("nrv: test case position must be 0-7")

// EncodeTestCaseToNRV encodes one test case as an 80-byte bracket (plan D5.1).
//
// Field mapping, using only existing Bracket fields:
//   - Projections: SHA-256 of Input (deterministic stand-in for the input embedding)
//   - SubSecondUS: tc.SubSecondUS
//   - IntentFlags, DomainSig: tc.IntentFlags, tc.DomainSig
//   - LSHSalt: uint32(Position) << 16 (PosIndex half; TemporalSalt half is zero)
//
// GoldenSeed is left zero, and Reserved is left zero (plan D5.1: Reserved is
// not used). Syntactic, DepHead, Memory and ID are zero; the caller sets ID.
func EncodeTestCaseToNRV(tc ErrorTestCase) (Bracket, error) {
	if tc.Input == "" {
		return Bracket{}, ErrTestCaseEmptyInput
	}
	if tc.Position >= ErrorTestSuiteBracketSlots {
		return Bracket{}, fmt.Errorf("%w: got %d", ErrTestCasePosition, tc.Position)
	}
	return Bracket{
		Projections: sha256.Sum256([]byte(tc.Input)),
		SubSecondUS: tc.SubSecondUS,
		IntentFlags: tc.IntentFlags,
		DomainSig:   tc.DomainSig,
		LSHSalt:     uint32(tc.Position) << 16,
	}, nil
}
