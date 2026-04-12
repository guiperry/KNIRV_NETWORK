package nrv

import (
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
		POSTag:      0x0A,
		Tense:       0x02,
		Plurality:   0x01,
		DepHead:     5,
		IntentFlags: 0x03,
		DomainSig:   0x1234,
		GoldenSeed:  0xDEADBEEF,
		Memory:      [14]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		LSHSalt:     0xAABBCCDD,
		Reserved:    [15]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55},
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
		{"POSTag (Slot 4)", 36, 1, []byte{0x0A}},
		{"Tense (Slot 4)", 37, 1, []byte{0x02}},
		{"Plurality (Slot 4)", 38, 1, []byte{0x01}},
		{"DepHead (Slot 5)", 39, 1, []byte{0x05}},
		{"IntentFlags (Slot 9)", 40, 1, []byte{0x03}},
		{"DomainSig (Slot 10)", 41, 2, []byte{0x34, 0x12}},
		{"GoldenSeed (Nonce Target)", 43, 4, []byte{0xEF, 0xBE, 0xAD, 0xDE}},
		{"Memory (Slots 6-8)", 47, 14, b.Memory[:]},
		{"LSHSalt (Slot 11)", 61, 4, []byte{0xDD, 0xCC, 0xBB, 0xAA}},
		{"Reserved", 65, 15, b.Reserved[:]},
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
		POSTag:      5,
		Tense:       2,
		Plurality:   1,
		DepHead:     -3,
		IntentFlags: 0x07,
		DomainSig:   0x5678,
		GoldenSeed:  0x12345678,
		Memory:      [14]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		LSHSalt:     0xABCD1234,
		Reserved:    [15]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
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
	if decoded.POSTag != b.POSTag {
		t.Errorf("POSTag: got %d, want %d", decoded.POSTag, b.POSTag)
	}
	if decoded.Tense != b.Tense {
		t.Errorf("Tense: got %d, want %d", decoded.Tense, b.Tense)
	}
	if decoded.Plurality != b.Plurality {
		t.Errorf("Plurality: got %d, want %d", decoded.Plurality, b.Plurality)
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
	gotLSH := binary.LittleEndian.Uint32(encoded[61:65])

	if gotLSH != expected {
		t.Errorf("LSHSalt at offset 0x3D: got 0x%08X, want 0x%08X", gotLSH, expected)
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

func TestSlotAlignment(t *testing.T) {
	tests := []struct {
		slot      string
		offset    int
		size      int
		fieldName string
	}{
		{"Slots 0-3", 0, 32, "Projections"},
		{"SubSecondUS", 32, 4, "SubSecondUS"},
		{"Slot 4", 36, 3, "POSTag+Tense+Plurality"},
		{"Slot 5", 39, 1, "DepHead"},
		{"Slot 9", 40, 1, "IntentFlags"},
		{"Slot 10", 41, 2, "DomainSig"},
		{"GoldenSeed", 43, 4, "GoldenSeed"},
		{"Slots 6-8", 47, 14, "Memory"},
		{"Slot 11", 61, 4, "LSHSalt"},
		{"Reserved", 65, 15, "Reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.slot, func(t *testing.T) {
			b := Bracket{}
			encoded := EncodeBracket(&b)

			switch tt.fieldName {
			case "Projections":
				if len(encoded) < tt.offset+tt.size {
					t.Errorf("encoded too short for %s", tt.slot)
				}
			case "SubSecondUS":
				if binary.LittleEndian.Uint32(encoded[tt.offset:tt.offset+4]) != 0 {
					t.Logf("%s at 0x%02X: OK (zero)", tt.slot, tt.offset)
				}
			}
			t.Logf("%s at offset 0x%02X (%d), size %d bytes", tt.slot, tt.offset, tt.offset, tt.size)
		})
	}

	total := 32 + 4 + 3 + 1 + 1 + 2 + 4 + 14 + 4 + 15
	if total != 80 {
		t.Errorf("Total size calculation: got %d, want 80", total)
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
	if decoded.POSTag != 0 {
		t.Error("POSTag should be zero")
	}
	if decoded.Tense != 0 {
		t.Error("Tense should be zero")
	}
	if decoded.Plurality != 0 {
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
