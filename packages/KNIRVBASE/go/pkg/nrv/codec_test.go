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
