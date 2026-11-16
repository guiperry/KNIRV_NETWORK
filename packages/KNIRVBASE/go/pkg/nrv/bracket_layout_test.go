package nrv

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBracketWireLayoutMatchesSpecification(t *testing.T) {
	if BracketSize != 80 {
		t.Fatalf("BracketSize should be 80, got %d", BracketSize)
	}

	b := Bracket{
		Projections: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		SubSecondUS: 0x12345678,
		Syntactic:   PackSyntacticByte(0x0A, 0x02, 0x01),
		DepHead:     5,
		IntentFlags: 0x03,
		DomainSig:   0x1234,
		GoldenSeed:  0xDEADBEEF,
		Memory:      [14]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		LSHSalt:     0xAABBCCDD,
		Reserved:    [17]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55},
	}

	encoded := EncodeBracket(&b)

	tests := []struct {
		name     string
		offset   int
		size     int
		expected []byte
	}{
		{"Projections (Slots 0-3)", 0, 32, b.Projections[:]},
		{"SubSecondUS", 32, 4, []byte{0x78, 0x56, 0x34, 0x12}},
		{"Syntactic (Slot 4, bit-packed)", 36, 1, []byte{PackSyntacticByte(0x0A, 0x02, 0x01)}},
		{"DepHead (Slot 5)", 37, 1, []byte{0x05}},
		{"IntentFlags (Slot 9)", 38, 1, []byte{0x03}},
		{"DomainSig (Slot 10)", 39, 2, []byte{0x34, 0x12}},
		{"GoldenSeed (Nonce Target)", 41, 4, []byte{0xEF, 0xBE, 0xAD, 0xDE}},
		{"Memory (Slots 6-8)", 45, 14, b.Memory[:]},
		{"LSHSalt (Slot 11)", 59, 4, []byte{0xDD, 0xCC, 0xBB, 0xAA}},
		{"Reserved", 63, 17, b.Reserved[:]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoded[tt.offset : tt.offset+tt.size]
			if len(got) != len(tt.expected) {
				t.Fatalf("size mismatch: got %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("byte %d: got 0x%02X, want 0x%02X", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestBracketDecodeReversesEncode(t *testing.T) {
	b := Bracket{
		ID:          "test-bracket-id",
		Projections: [32]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SubSecondUS: 987654321,
		Syntactic:   PackSyntacticByte(5, 2, 1),
		DepHead:     -3,
		IntentFlags: 0x07,
		DomainSig:   0x5678,
		GoldenSeed:  0x12345678,
		Memory:      [14]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		LSHSalt:     0xABCD1234,
		Reserved:    [17]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		FrameID:     "frame-123",
		FrameUnix:   1700000000,
	}

	encoded := EncodeBracket(&b)
	decoded := DecodeBracket(encoded)

	if decoded.Projections != b.Projections {
		t.Errorf("Projections mismatch")
	}
	if decoded.SubSecondUS != b.SubSecondUS {
		t.Errorf("SubSecondUS: got %d, want %d", decoded.SubSecondUS, b.SubSecondUS)
	}
	if decoded.GetPOSTag() != b.GetPOSTag() {
		t.Errorf("POSTag: got %d, want %d", decoded.GetPOSTag(), b.GetPOSTag())
	}
	if decoded.GetTense() != b.GetTense() {
		t.Errorf("Tense: got %d, want %d", decoded.GetTense(), b.GetTense())
	}
	if decoded.GetPlurality() != b.GetPlurality() {
		t.Errorf("Plurality: got %d, want %d", decoded.GetPlurality(), b.GetPlurality())
	}
	if decoded.DepHead != b.DepHead {
		t.Errorf("DepHead: got %d, want %d", decoded.DepHead, b.DepHead)
	}
	if decoded.IntentFlags != b.IntentFlags {
		t.Errorf("IntentFlags: got 0x%02X, want 0x%02X", decoded.IntentFlags, b.IntentFlags)
	}
	if decoded.DomainSig != b.DomainSig {
		t.Errorf("DomainSig: got 0x%04X, want 0x%04X", decoded.DomainSig, b.DomainSig)
	}
	if decoded.GoldenSeed != b.GoldenSeed {
		t.Errorf("GoldenSeed: got 0x%08X, want 0x%08X", decoded.GoldenSeed, b.GoldenSeed)
	}
	if decoded.Memory != b.Memory {
		t.Errorf("Memory mismatch")
	}
	if decoded.LSHSalt != b.LSHSalt {
		t.Errorf("LSHSalt: got 0x%08X, want 0x%08X", decoded.LSHSalt, b.LSHSalt)
	}
	if decoded.Reserved != b.Reserved {
		t.Errorf("Reserved mismatch")
	}
}

func TestLSHSaltFormat(t *testing.T) {
	posIndex := uint16(42)
	temporalSalt := uint16(0xABCD)
	expected := uint32(posIndex)<<16 | uint32(temporalSalt)

	b := Bracket{
		Projections: [32]byte{},
		LSHSalt:     expected,
	}

	encoded := EncodeBracket(&b)
	gotLSH := binary.LittleEndian.Uint32(encoded[59:63])

	if gotLSH != expected {
		t.Errorf("LSHSalt at offset 0x3B: got 0x%08X, want 0x%08X", gotLSH, expected)
	}

	decoded := DecodeBracket(encoded)
	if decoded.LSHSalt != expected {
		t.Errorf("LSHSalt decode: got 0x%08X, want 0x%08X", decoded.LSHSalt, expected)
	}

	if decoded.LSHSalt != b.LSHSalt {
		t.Errorf("Round-trip failed: got 0x%08X, want 0x%08X", decoded.LSHSalt, b.LSHSalt)
	}

	extractedPos := uint16(decoded.LSHSalt >> 16)
	extractedSalt := uint16(decoded.LSHSalt & 0xFFFF)

	if extractedPos != posIndex {
		t.Errorf("PosIndex extraction: got %d, want %d", extractedPos, posIndex)
	}
	if extractedSalt != temporalSalt {
		t.Errorf("TemporalSalt extraction: got 0x%04X, want 0x%04X", extractedSalt, temporalSalt)
	}
}

// v2MigrationSlot is one field of the pre-canonical (v2) encoder layout, as
// migrateV2Bracket reads it (KNIRVMODELS 3_DATA_SEEDER, pkg/knirvbaseclient/client.go).
// v2Off/size are the v2 offsets; canonOff is where migrateV2Bracket writes the same
// bytes in the canonical 80-byte layout that EncodeBracket produces. The syntactic
// byte is packed from three v2 bytes, so it has no byte-for-byte canonical offset and
// is left out of the canonical-placement check.
type v2MigrationSlot struct {
	name     string
	v2Off    int
	size     int
	canonOff int
	packed   bool
}

var v2MigrationSlots = []v2MigrationSlot{
	{name: "Projections", v2Off: 0, size: 32, canonOff: 0},
	{name: "SubSecondUS", v2Off: 32, size: 4, canonOff: 32},
	{name: "Syntactic (POS/Tense/Plurality, v2 bytes 36-38)", v2Off: 36, size: 3, canonOff: 36, packed: true},
	{name: "DepHead", v2Off: 39, size: 1, canonOff: 37},
	{name: "IntentFlags", v2Off: 40, size: 1, canonOff: 38},
	{name: "DomainSig", v2Off: 41, size: 2, canonOff: 39},
	{name: "GoldenSeed", v2Off: 43, size: 4, canonOff: 41},
	{name: "Memory", v2Off: 47, size: 14, canonOff: 45},
	{name: "LSHSalt", v2Off: 61, size: 4, canonOff: 59},
	{name: "Reserved", v2Off: 65, size: 15, canonOff: 63},
}

// TestV2MigrationLayoutSlots pins the v2 encoder layout that migrateV2Bracket reads.
// The v2 slots must tile the 80-byte record with no gap or overlap, so a change to any
// single v2 offset or size is caught. Each non-packed slot must land at its canonical
// offset in EncodeBracket's output, which TestBracketWireLayoutMatchesSpecification
// pins to codec.go. The canonical layout itself is unchanged by this test.
func TestV2MigrationLayoutSlots(t *testing.T) {
	next := 0
	for _, s := range v2MigrationSlots {
		if s.v2Off != next {
			t.Fatalf("v2 slot %s starts at %d, want %d (gap or overlap)", s.name, s.v2Off, next)
		}
		next = s.v2Off + s.size
	}
	if next != BracketSize {
		t.Fatalf("v2 slots cover %d bytes, want %d", next, BracketSize)
	}

	b := Bracket{
		Projections: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		SubSecondUS: 0x11223344,
		DepHead:     5,
		IntentFlags: 0x3C,
		DomainSig:   0xBEEF,
		GoldenSeed:  0xCAFEF00D,
		Memory:      [14]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		LSHSalt:     0x0A0B0C0D,
		Reserved:    [17]byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F, 0x90, 0x91},
	}
	encoded := EncodeBracket(&b)

	want := map[string][]byte{
		"Projections": b.Projections[:],
		"SubSecondUS": {0x44, 0x33, 0x22, 0x11},
		"DepHead":     {0x05},
		"IntentFlags": {0x3C},
		"DomainSig":   {0xEF, 0xBE},
		"GoldenSeed":  {0x0D, 0xF0, 0xFE, 0xCA},
		"Memory":      b.Memory[:],
		"LSHSalt":     {0x0D, 0x0C, 0x0B, 0x0A},
		// migrateV2Bracket copies the first 15 Reserved bytes (v2 65..79) to canonical 63..77.
		"Reserved": b.Reserved[:15],
	}
	for _, s := range v2MigrationSlots {
		if s.packed {
			continue
		}
		t.Run(s.name, func(t *testing.T) {
			expected, ok := want[s.name]
			if !ok {
				t.Fatalf("no expected bytes for slot %s", s.name)
			}
			got := encoded[s.canonOff : s.canonOff+s.size]
			if !bytes.Equal(got, expected) {
				t.Fatalf("canonical offset %d: got % X, want % X", s.canonOff, got, expected)
			}
		})
	}
}

func TestZeroValueBracketEncodeDecode(t *testing.T) {
	var b Bracket
	encoded := EncodeBracket(&b)
	decoded := DecodeBracket(encoded)

	if decoded.Projections != b.Projections {
		t.Error("Projections should be zero")
	}
	if decoded.SubSecondUS != 0 {
		t.Error("SubSecondUS should be zero")
	}
	if decoded.GetPOSTag() != 0 {
		t.Error("POSTag should be zero")
	}
	if decoded.GetTense() != 0 {
		t.Error("Tense should be zero")
	}
	if decoded.GetPlurality() != 0 {
		t.Error("Plurality should be zero")
	}
	if decoded.DepHead != 0 {
		t.Error("DepHead should be zero")
	}
	if decoded.IntentFlags != 0 {
		t.Error("IntentFlags should be zero")
	}
	if decoded.DomainSig != 0 {
		t.Error("DomainSig should be zero")
	}
	if decoded.GoldenSeed != 0 {
		t.Error("GoldenSeed should be zero")
	}
	if decoded.Memory != b.Memory {
		t.Error("Memory should be zero")
	}
	if decoded.LSHSalt != 0 {
		t.Error("LSHSalt should be zero")
	}
}
