package nrv

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func sampleTestCase() ErrorTestCase {
	return ErrorTestCase{
		Input:       "assert parse('1+1') == 2",
		DomainSig:   0x5A7C,
		IntentFlags: 0x06,
		Position:    3,
		SubSecondUS: 421337,
	}
}

func TestEncodeTestCaseDeterministic(t *testing.T) {
	tc := sampleTestCase()
	a, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	b, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	ea, eb := EncodeBracket(&a), EncodeBracket(&b)
	if !bytes.Equal(ea[:], eb[:]) {
		t.Fatal("same test case encoded to different bytes")
	}
}

func TestEncodeTestCaseInputChangesProjection(t *testing.T) {
	tc := sampleTestCase()
	a, _ := EncodeTestCaseToNRV(tc)
	tc.Input += "x"
	b, _ := EncodeTestCaseToNRV(tc)
	if a.Projections == b.Projections {
		t.Fatal("different inputs produced the same projection")
	}
}

func TestEncodeTestCaseProjectionIsFullInputDigest(t *testing.T) {
	// Long input must not be truncated: the projection digests all of it.
	long := strings.Repeat("A", 1<<20)
	tc := ErrorTestCase{Input: long, Position: 0}
	b, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if b.Projections != sha256.Sum256([]byte(long)) {
		t.Fatal("projection does not digest the full input")
	}
	// Changing only the final byte must change the projection.
	tail := []byte(long)
	tail[len(tail)-1] = 'B'
	c, _ := EncodeTestCaseToNRV(ErrorTestCase{Input: string(tail), Position: 0})
	if c.Projections == b.Projections {
		t.Fatal("tail byte change did not change projection (truncation)")
	}
}

func TestEncodeTestCaseFieldMapping(t *testing.T) {
	tc := sampleTestCase()
	b, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if b.SubSecondUS != tc.SubSecondUS {
		t.Errorf("SubSecondUS = %d, want %d", b.SubSecondUS, tc.SubSecondUS)
	}
	if b.IntentFlags != tc.IntentFlags {
		t.Errorf("IntentFlags = 0x%02X, want 0x%02X", b.IntentFlags, tc.IntentFlags)
	}
	if b.DomainSig != tc.DomainSig {
		t.Errorf("DomainSig = 0x%04X, want 0x%04X", b.DomainSig, tc.DomainSig)
	}
	if want := uint32(tc.Position) << 16; b.LSHSalt != want {
		t.Errorf("LSHSalt = 0x%08X, want 0x%08X", b.LSHSalt, want)
	}
}

func TestEncodeTestCaseLeavesUnusedFieldsZero(t *testing.T) {
	b, err := EncodeTestCaseToNRV(sampleTestCase())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if b.Reserved != [17]byte{} {
		t.Errorf("Reserved must stay zero, got %x", b.Reserved)
	}
	if b.GoldenSeed != 0 {
		t.Errorf("GoldenSeed must be unset for test-case brackets, got 0x%08X", b.GoldenSeed)
	}
	if b.Syntactic != 0 || b.DepHead != 0 || b.Memory != [14]byte{} || b.ID != "" {
		t.Errorf("unexpected non-zero unused field: %+v", b)
	}
}

func TestEncodeTestCaseWireLayout(t *testing.T) {
	tc := sampleTestCase()
	b, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	buf := EncodeBracket(&b)
	if len(buf) != BracketSize {
		t.Fatalf("bracket size %d, want %d", len(buf), BracketSize)
	}
	if !bytes.Equal(buf[0:32], b.Projections[:]) {
		t.Error("projection bytes not at 0x00-0x1F")
	}
	if got := binary.LittleEndian.Uint32(buf[32:36]); got != tc.SubSecondUS {
		t.Errorf("SubSecondUS at 0x20 = %d", got)
	}
	if buf[36] != 0 || buf[37] != 0 {
		t.Errorf("Syntactic/DepHead bytes not zero: %02X %02X", buf[36], buf[37])
	}
	if buf[38] != tc.IntentFlags {
		t.Errorf("IntentFlags at 0x26 = 0x%02X", buf[38])
	}
	if got := binary.LittleEndian.Uint16(buf[39:41]); got != tc.DomainSig {
		t.Errorf("DomainSig at 0x27 = 0x%04X", got)
	}
	if got := binary.LittleEndian.Uint32(buf[41:45]); got != 0 {
		t.Errorf("GoldenSeed at 0x29 = 0x%08X, want 0", got)
	}
	if !bytes.Equal(buf[45:59], make([]byte, 14)) {
		t.Errorf("Memory at 0x2D not zero: %x", buf[45:59])
	}
	if got := binary.LittleEndian.Uint32(buf[59:63]); got != uint32(tc.Position)<<16 {
		t.Errorf("LSHSalt at 0x3B = 0x%08X", got)
	}
	if !bytes.Equal(buf[63:80], make([]byte, 17)) {
		t.Errorf("Reserved at 0x3F not zero: %x", buf[63:80])
	}
}

func TestEncodeTestCaseBracketRoundTrip(t *testing.T) {
	tc := ErrorTestCase{Input: "round trip", DomainSig: 0xFFFF, IntentFlags: 0xFF, Position: 7, SubSecondUS: 0xFFFFFFFF}
	b, err := EncodeTestCaseToNRV(tc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	buf := EncodeBracket(&b)
	got := DecodeBracket(buf)
	if got.Projections != b.Projections ||
		got.SubSecondUS != b.SubSecondUS ||
		got.IntentFlags != b.IntentFlags ||
		got.DomainSig != b.DomainSig ||
		got.LSHSalt != b.LSHSalt ||
		got.GoldenSeed != b.GoldenSeed ||
		got.Reserved != b.Reserved {
		t.Fatalf("decode(encode) mismatch:\n got  %+v\n want %+v", got, b)
	}
	if again := EncodeBracket(&got); again != buf {
		t.Fatal("re-encoding decoded bracket changed bytes")
	}
}

func TestEncodeTestCasePositionRange(t *testing.T) {
	for p := uint8(0); p < ErrorTestSuiteBracketSlots; p++ {
		b, err := EncodeTestCaseToNRV(ErrorTestCase{Input: "x", Position: p})
		if err != nil {
			t.Fatalf("position %d rejected: %v", p, err)
		}
		if b.LSHSalt != uint32(p)<<16 {
			t.Errorf("position %d: LSHSalt 0x%08X", p, b.LSHSalt)
		}
	}
	for _, p := range []uint8{8, 9, 255} {
		_, err := EncodeTestCaseToNRV(ErrorTestCase{Input: "x", Position: p})
		if !errors.Is(err, ErrTestCasePosition) {
			t.Errorf("position %d: err = %v, want ErrTestCasePosition", p, err)
		}
	}
}

func TestEncodeTestCaseRejectsEmptyInput(t *testing.T) {
	_, err := EncodeTestCaseToNRV(ErrorTestCase{Input: "", Position: 0})
	if !errors.Is(err, ErrTestCaseEmptyInput) {
		t.Fatalf("err = %v, want ErrTestCaseEmptyInput", err)
	}
}

func TestEncodeTestCaseSuiteDistinctPositions(t *testing.T) {
	seen := map[uint32]bool{}
	for p := uint8(0); p < ErrorTestSuiteBracketSlots; p++ {
		b, err := EncodeTestCaseToNRV(ErrorTestCase{Input: "same input", Position: p})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if seen[b.LSHSalt] {
			t.Fatalf("duplicate LSHSalt at position %d", p)
		}
		seen[b.LSHSalt] = true
	}
}
