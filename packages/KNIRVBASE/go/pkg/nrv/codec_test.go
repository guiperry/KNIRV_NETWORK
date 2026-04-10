package nrv

import (
	"bytes"
	"testing"
)

func TestAlign8(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{15, 16},
		{16, 16},
	}
	for _, tt := range tests {
		if got := Align8(tt.input); got != tt.expected {
			t.Errorf("Align8(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeDecodeHeader(t *testing.T) {
	h := Header{
		Magic:       Magic,
		Version:     Version,
		TotalLength: 1024,
	}

	var buf bytes.Buffer
	if err := EncodeHeader(&buf, h); err != nil {
		t.Fatalf("EncodeHeader failed: %v", err)
	}

	decoded, err := DecodeHeader(&buf)
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decoded.Magic != h.Magic {
		t.Errorf("Magic mismatch: got 0x%X, want 0x%X", decoded.Magic, h.Magic)
	}
	if decoded.Version != h.Version {
		t.Errorf("Version mismatch: got %d, want %d", decoded.Version, h.Version)
	}
	if decoded.TotalLength != h.TotalLength {
		t.Errorf("TotalLength mismatch: got %d, want %d", decoded.TotalLength, h.TotalLength)
	}
}

func TestDecodeHeaderInvalidMagic(t *testing.T) {
	var buf bytes.Buffer
	invalid := Header{Magic: 0xDEADBEEF, Version: 1, TotalLength: 100}
	EncodeHeader(&buf, invalid)

	_, err := DecodeHeader(&buf)
	if err == nil {
		t.Fatal("expected error for invalid magic bytes")
	}
}

func TestEncodeFrameAlignment(t *testing.T) {
	frame := &Frame{
		ID:     "test-frame",
		Vector: [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Seed:   [32]byte{1, 2, 3, 4, 5},
		Thermo: ThermoData{TempCelsius: 50.5, VoltageV: 12.0, FreqMHz: 500, FanRPM: 3000},
		Proof:  []byte("test proof"),
	}

	data, _ := EncodeFrame(frame)
	if len(data)%8 != 0 {
		t.Errorf("frame length %d is not 8-byte aligned", len(data))
	}

	expectedMinLen := 48 + 32 + 16 + 8
	if len(data) < expectedMinLen {
		t.Errorf("frame length %d is less than expected minimum %d", len(data), expectedMinLen)
	}
}

func TestEncodeFrameModalities(t *testing.T) {
	frame := &Frame{
		ID:     "test-frame",
		Vector: [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Seed:   [32]byte{1, 2, 3, 4, 5},
		Thermo: ThermoData{TempCelsius: 50.5, VoltageV: 12.0, FreqMHz: 500, FanRPM: 3000},
		Proof:  []byte("test proof"),
	}

	_, modalities := EncodeFrame(frame)

	if len(modalities) != 4 {
		t.Errorf("expected 4 modalities, got %d", len(modalities))
	}

	vectorIdx := modalities[ModalityVector]
	if vectorIdx.Offset != 0 || vectorIdx.Length != 48 {
		t.Errorf("vector modality has wrong offset/length: %v", vectorIdx)
	}

	seedIdx := modalities[ModalitySeed]
	if seedIdx.Offset != 48 || seedIdx.Length != 32 {
		t.Errorf("seed modality has wrong offset/length: %v", seedIdx)
	}

	thermoIdx := modalities[ModalityThermo]
	if thermoIdx.Offset != 80 || thermoIdx.Length != 16 {
		t.Errorf("thermo modality has wrong offset/length: %v", thermoIdx)
	}

	proofIdx := modalities[ModalityProof]
	if proofIdx.Offset != 96 {
		t.Errorf("proof modality has wrong offset: %d", proofIdx.Offset)
	}
	if proofIdx.Length != len(frame.Proof) {
		t.Errorf("proof modality has wrong length: got %d, want %d", proofIdx.Length, len(frame.Proof))
	}
}

func TestEncodeBracket_RoundTrip(t *testing.T) {
	b := &Bracket{
		ID:          "test-bracket-123",
		SubSecondUS: 123456,
		POSTag:      0x01,
		Tense:       0x02,
		Plurality:   0x01,
		DepHead:     0x05,
		IntentFlags: 0x03,
		DomainSig:   0x1234,
		GoldenSeed:  0xDEADBEEF,
		LSHSalt:     0x12345678,
	}
	for i := range b.Projections {
		b.Projections[i] = byte(i % 256)
	}
	for i := range b.Memory {
		b.Memory[i] = byte((i * 7) % 256)
	}

	encoded := EncodeBracket(b)
	decoded := DecodeBracket(encoded)

	if decoded.SubSecondUS != b.SubSecondUS {
		t.Errorf("SubSecondUS mismatch: got %d, want %d", decoded.SubSecondUS, b.SubSecondUS)
	}
	if decoded.POSTag != b.POSTag {
		t.Errorf("POSTag mismatch: got 0x%X, want 0x%X", decoded.POSTag, b.POSTag)
	}
	if decoded.Tense != b.Tense {
		t.Errorf("Tense mismatch: got 0x%X, want 0x%X", decoded.Tense, b.Tense)
	}
	if decoded.Plurality != b.Plurality {
		t.Errorf("Plurality mismatch: got 0x%X, want 0x%X", decoded.Plurality, b.Plurality)
	}
	if decoded.DepHead != b.DepHead {
		t.Errorf("DepHead mismatch: got 0x%X, want 0x%X", decoded.DepHead, b.DepHead)
	}
	if decoded.IntentFlags != b.IntentFlags {
		t.Errorf("IntentFlags mismatch: got 0x%X, want 0x%X", decoded.IntentFlags, b.IntentFlags)
	}
	if decoded.DomainSig != b.DomainSig {
		t.Errorf("DomainSig mismatch: got 0x%X, want 0x%X", decoded.DomainSig, b.DomainSig)
	}
	if decoded.GoldenSeed != b.GoldenSeed {
		t.Errorf("GoldenSeed mismatch: got 0x%X, want 0x%X", decoded.GoldenSeed, b.GoldenSeed)
	}
	if decoded.LSHSalt != b.LSHSalt {
		t.Errorf("LSHSalt mismatch: got 0x%X, want 0x%X", decoded.LSHSalt, b.LSHSalt)
	}
	if decoded.Projections != b.Projections {
		t.Errorf("Projections mismatch")
	}
	if decoded.Memory != b.Memory {
		t.Errorf("Memory mismatch")
	}
	if len(encoded) != BracketSize {
		t.Errorf("encoded length: got %d, want %d", len(encoded), BracketSize)
	}
}

func TestXORProjections_Inverse(t *testing.T) {
	var a, b [32]byte
	for i := range a {
		a[i] = byte(i * 3)
		b[i] = byte(i * 7)
	}

	diff := XORProjections(a, b)
	recovered := ApplyProjectionDelta(diff, b)

	for i := range a {
		if recovered[i] != a[i] {
			t.Errorf("XOR not invertible at index %d: got 0x%X, want 0x%X", i, recovered[i], a[i])
		}
	}
}

func TestXORProjections_SameInput(t *testing.T) {
	var a [32]byte
	for i := range a {
		a[i] = byte(i)
	}

	diff := XORProjections(a, a)
	var zero [32]byte
	for i := range diff {
		if diff[i] != zero[i] {
			t.Errorf("XOR of same input should be zero at index %d", i)
		}
	}
}

func TestBracketSize_Constant(t *testing.T) {
	if BracketSize != 80 {
		t.Errorf("BracketSize = %d, want 80", BracketSize)
	}
}

func TestDeltaType_Constants(t *testing.T) {
	if DeltaTypeI != "I" {
		t.Errorf("DeltaTypeI = %s, want I", DeltaTypeI)
	}
	if DeltaTypeP != "P" {
		t.Errorf("DeltaTypeP = %s, want P", DeltaTypeP)
	}
}
