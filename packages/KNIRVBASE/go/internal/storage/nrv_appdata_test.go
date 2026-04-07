package storage

// nrv_appdata_test.go — tests that mirror production usage of .nrv files from
// the application data directory, plus cross-language binary-format spec tests.
//
// Production layout (Linux/macOS):
//   $HOME/.local/share/knirvbase/datasets/<collection>.nrv
//
// In tests we substitute a t.TempDir() root while keeping the same sub-path
// structure so the code path through NewNRVStorage, getOrCreateWriter, and
// getNRVPath is exercised identically to production.

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

// appDataDir returns a temp path that mirrors the production app data layout:
//   <tmp>/knirvbase/datasets/
func appDataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "knirvbase", "datasets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("appDataDir: %v", err)
	}
	return dir
}

func newTestKeyPair(t *testing.T) *pqc.PQCKeyPair {
	t.Helper()
	kp, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair: %v", err)
	}
	return kp
}

// newProductionFrame builds a Frame that is representative of the data written
// by KNIRVHASHER during normal operation.
func newProductionFrame(id string, index int) *nrv.Frame {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(index + i)
	}
	return &nrv.Frame{
		ID:     id,
		Vector: [12]float32{float32(index), float32(index + 1), float32(index + 2), float32(index + 3), float32(index + 4), float32(index + 5), float32(index + 6), float32(index + 7), float32(index + 8), float32(index + 9), float32(index + 10), float32(index + 11)},
		Seed:   seed,
		Thermo: nrv.ThermoData{
			TempCelsius: 50.0 + float32(index)*0.5,
			VoltageV:    12.0,
			FreqMHz:     500.0,
			FanRPM:      3000.0,
		},
		Proof: []byte("production proof data"),
	}
}

// ── App data directory lifecycle ─────────────────────────────────────────────

// TestNRVAppDataDirectoryLifecycle is the primary production-scenario test.
// It creates NRVStorage at an OS-appropriate data path, inserts frames, closes,
// reopens and verifies data persists exactly as expected.
func TestNRVAppDataDirectoryLifecycle(t *testing.T) {
	dataDir := appDataDir(t)
	kp := newTestKeyPair(t)
	ctx := context.Background()

	// ── Phase 1: write ────────────────────────────────────────────────────
	storage := NewNRVStorage(dataDir, kp)

	collection := "training_set_v1"
	const numFrames = 5

	for i := 0; i < numFrames; i++ {
		frame := newProductionFrame("node-"+string(rune('0'+i)), i)
		doc := map[string]interface{}{
			"id":        frame.ID,
			"verified":  i%2 == 0,
			"ergo_rank": 0.6 + float64(i)*0.05,
			"payload": map[string]interface{}{
				"vector": frame.Vector[:],
				"seed":   frame.Seed[:],
				"thermo": map[string]float64{
					"temp_celsius": float64(frame.Thermo.TempCelsius),
					"voltage_v":    float64(frame.Thermo.VoltageV),
					"freq_mhz":     float64(frame.Thermo.FreqMHz),
					"fan_rpm":      float64(frame.Thermo.FanRPM),
				},
				"proof": string(frame.Proof),
			},
		}
		if err := storage.Insert(ctx, collection, doc); err != nil {
			t.Fatalf("Insert frame %d: %v", i, err)
		}
	}

	// Verify registry in-memory before close.
	expectedPath := filepath.Join(dataDir, collection+".nrv")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf(".nrv file not created at expected path: %s", expectedPath)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// ── Phase 2: reopen and verify ────────────────────────────────────────
	storage2 := NewNRVStorage(dataDir, kp)
	defer storage2.Close()

	allDocs, err := storage2.FindAll(ctx, collection)
	if err != nil {
		t.Fatalf("FindAll after reopen: %v", err)
	}
	if len(allDocs) != numFrames {
		t.Errorf("expected %d documents, got %d", numFrames, len(allDocs))
	}

	// Verify individual document round-trips.
	doc, err := storage2.Find(ctx, collection, "node-2")
	if err != nil {
		t.Fatalf("Find after reopen: %v", err)
	}
	if doc == nil {
		t.Fatal("node-2 must be findable after reopen")
	}
	if doc["verified"] != true {
		t.Errorf("node-2 verified: expected true, got %v", doc["verified"])
	}
	expectedErgoRank := 0.6 + 2*0.05
	if doc["ergo_rank"] != expectedErgoRank {
		t.Errorf("node-2 ergo_rank: expected %v, got %v", expectedErgoRank, doc["ergo_rank"])
	}
}

// TestNRVMultipleCollectionsInAppDataDir verifies that multiple .nrv files
// can coexist in the same data directory (errors/, capabilities/, ideas/).
func TestNRVMultipleCollectionsInAppDataDir(t *testing.T) {
	dataDir := appDataDir(t)
	kp := newTestKeyPair(t)
	ctx := context.Background()

	storage := NewNRVStorage(dataDir, kp)
	defer storage.Close()

	collections := []string{"errors", "capabilities", "ideas"}
	for _, name := range collections {
		doc := map[string]interface{}{
			"id":        name + "-frame-0",
			"verified":  true,
			"ergo_rank": 0.9,
			"payload": map[string]interface{}{
				"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				"thermo": map[string]float64{"temp_celsius": 50, "voltage_v": 12, "freq_mhz": 500, "fan_rpm": 3000},
				"proof":  "proof",
			},
		}
		if err := storage.Insert(ctx, name, doc); err != nil {
			t.Fatalf("Insert into %s: %v", name, err)
		}
	}

	// Verify each collection has its own .nrv file.
	for _, name := range collections {
		p := filepath.Join(dataDir, name+".nrv")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("%s.nrv must exist in app data dir", name)
		}
	}

	// Verify data is readable per collection.
	for _, name := range collections {
		docs, err := storage.FindAll(ctx, name)
		if err != nil {
			t.Fatalf("FindAll(%s): %v", name, err)
		}
		if len(docs) != 1 {
			t.Errorf("%s: expected 1 doc, got %d", name, len(docs))
		}
	}
}

// TestNRVAppDataDirectoryModalityAccess verifies modality-level reads work
// via the production storage path (NRVStorage.GetModality).
func TestNRVAppDataDirectoryModalityAccess(t *testing.T) {
	dataDir := appDataDir(t)
	kp := newTestKeyPair(t)
	ctx := context.Background()

	storage := NewNRVStorage(dataDir, kp)
	defer storage.Close()

	doc := map[string]interface{}{
		"id":        "modality-frame",
		"verified":  true,
		"ergo_rank": 0.85,
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			"thermo": map[string]float64{"temp_celsius": 50.5, "voltage_v": 12.0, "freq_mhz": 500, "fan_rpm": 3000},
			"proof":  "proof for modality test",
		},
	}
	if err := storage.Insert(ctx, "modality-collection", doc); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	vectorBytes, err := storage.GetModality(ctx, "modality-collection", "modality-frame", nrv.ModalityVector)
	if err != nil {
		t.Fatalf("GetModality(vector): %v", err)
	}
	if len(vectorBytes) != 48 {
		t.Errorf("vector modality: expected 48 bytes, got %d", len(vectorBytes))
	}
	// First float32 LE must decode as 1.0.
	if v := math.Float32frombits(binary.LittleEndian.Uint32(vectorBytes[0:4])); v != 1.0 {
		t.Errorf("vector[0]: expected 1.0, got %v", v)
	}

	seedBytes, err := storage.GetModality(ctx, "modality-collection", "modality-frame", nrv.ModalitySeed)
	if err != nil {
		t.Fatalf("GetModality(seed): %v", err)
	}
	if len(seedBytes) != 32 {
		t.Errorf("seed modality: expected 32 bytes, got %d", len(seedBytes))
	}
	if seedBytes[0] != 1 {
		t.Errorf("seed[0]: expected 1, got %d", seedBytes[0])
	}

	thermoBytes, err := storage.GetModality(ctx, "modality-collection", "modality-frame", nrv.ModalityThermo)
	if err != nil {
		t.Fatalf("GetModality(thermo): %v", err)
	}
	if len(thermoBytes) != 16 {
		t.Errorf("thermo modality: expected 16 bytes, got %d", len(thermoBytes))
	}
	temp := math.Float32frombits(binary.LittleEndian.Uint32(thermoBytes[0:4]))
	if math.Abs(float64(temp)-50.5) > 0.001 {
		t.Errorf("thermo.temp_celsius: expected 50.5, got %v", temp)
	}
}

// ── Cross-language binary format specification ───────────────────────────────
//
// These tests pin the exact byte layout of the NRV binary frame format so any
// deviation breaks immediately.  The same assertions exist in the Rust and
// TypeScript test suites; if all three pass, the three implementations are
// wire-compatible.

// TestNRVBinaryFrameEncodingSpec verifies the raw bytes produced by EncodeFrame
// match the spec shared by the Go, Rust, and TypeScript implementations.
func TestNRVBinaryFrameEncodingSpec(t *testing.T) {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(i + 1)
	}

	frame := &nrv.Frame{
		ID:     "spec-frame",
		Vector: [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Seed:   seed,
		Thermo: nrv.ThermoData{TempCelsius: 50.5, VoltageV: 12.0, FreqMHz: 500.0, FanRPM: 3000.0},
		Proof:  []byte("ok"),
	}

	bytes, mods := nrv.EncodeFrame(frame)

	// Total = 48 (vector) + 32 (seed) + 16 (thermo) + align8(2) (proof "ok") = 104
	if len(bytes) != 104 {
		t.Errorf("frame binary length: expected 104, got %d", len(bytes))
	}

	// Modality offsets must match the spec.
	checkModality := func(name string, wantOffset, wantLen int) {
		t.Helper()
		m, ok := mods[nrv.ModalityType(name)]
		if !ok {
			t.Errorf("modality %q missing from map", name)
			return
		}
		if m.Offset != wantOffset {
			t.Errorf("modality %q offset: expected %d, got %d", name, wantOffset, m.Offset)
		}
		if m.Length != wantLen {
			t.Errorf("modality %q length: expected %d, got %d", name, wantLen, m.Length)
		}
	}
	checkModality("vector", 0, 48)
	checkModality("seed", 48, 32)
	checkModality("thermo", 80, 16)
	checkModality("proof", 96, 2)

	// vector[0] = 1.0f32 LE
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[0:4])); v != 1.0 {
		t.Errorf("vector[0]: expected 1.0, got %v", v)
	}
	// vector[11] = 12.0f32 LE (offset 44)
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[44:48])); v != 12.0 {
		t.Errorf("vector[11]: expected 12.0, got %v", v)
	}

	// seed bytes 1..32
	for i, b := range bytes[48:80] {
		if b != byte(i+1) {
			t.Errorf("seed[%d]: expected %d, got %d", i, i+1, b)
		}
	}

	// thermo at offsets 80..96
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[80:84])); math.Abs(float64(v)-50.5) > 0.001 {
		t.Errorf("thermo.temp_celsius: expected 50.5, got %v", v)
	}
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[84:88])); v != 12.0 {
		t.Errorf("thermo.voltage_v: expected 12.0, got %v", v)
	}
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[88:92])); v != 500.0 {
		t.Errorf("thermo.freq_mhz: expected 500.0, got %v", v)
	}
	if v := math.Float32frombits(binary.LittleEndian.Uint32(bytes[92:96])); v != 3000.0 {
		t.Errorf("thermo.fan_rpm: expected 3000.0, got %v", v)
	}

	// proof "ok" at offset 96
	if bytes[96] != 'o' || bytes[97] != 'k' {
		t.Errorf("proof bytes: expected 'ok', got %v", bytes[96:98])
	}
	// 6 bytes of alignment padding must be zero
	for i := 98; i < 104; i++ {
		if bytes[i] != 0 {
			t.Errorf("alignment padding byte[%d]: expected 0, got %d", i, bytes[i])
		}
	}
}

// TestNRVHeaderBytesSpec pins the 12-byte header format.
func TestNRVHeaderBytesSpec(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.nrv")
	kp := newTestKeyPair(t)

	w, err := NewNRVWriter(path, kp)
	if err != nil {
		t.Fatalf("NewNRVWriter: %v", err)
	}
	w.Close()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Magic 0x4E525621 in little-endian = [0x21, 0x56, 0x52, 0x4E]
	want := []byte{0x21, 0x56, 0x52, 0x4E}
	if string(b[0:4]) != string(want) {
		t.Errorf("magic bytes: expected %v, got %v", want, b[0:4])
	}
	if v := binary.LittleEndian.Uint32(b[4:8]); v != 1 {
		t.Errorf("version: expected 1, got %d", v)
	}
}
