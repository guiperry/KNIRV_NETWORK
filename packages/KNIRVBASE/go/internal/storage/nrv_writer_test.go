package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
	"github.com/stretchr/testify/require"
)

func setupTestWriter(t *testing.T) (*NRVWriter, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	writer, err := NewNRVWriter(path, keyPair)
	if err != nil {
		t.Fatalf("NewNRVWriter failed: %v", err)
	}

	return writer, path
}

func TestNewNRVWriterCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)
	defer writer.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected .nrv file to be created")
	}

	require.NotNil(t, writer.registry)
	require.Equal(t, 1, writer.registry.Version)
}

func TestAppendFrame_NewSignature(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frameID := "test-frame-1"
	brackets := make([]byte, 80*3)
	for i := range brackets {
		brackets[i] = byte(i)
	}

	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0, DriftScore: 0},
		{ID: "b2", Type: nrv.DeltaTypeP, AnchorID: ptr("b1"), Offset: 80, DriftScore: 0.1},
		{ID: "b3", Type: nrv.DeltaTypeI, Offset: 160, DriftScore: 0},
	}

	thermo := nrv.ThermoAtmosphere{AvgTempC: 71.5, PeakVoltV: 1.3, ClockMHz: 550}
	ling := nrv.LinguisticMapping{Token: "test", Unit: "word"}

	err := writer.AppendFrame(frameID, brackets, metas, thermo, ling)
	require.NoError(t, err)

	require.Len(t, writer.registry.Frames, 1)
	require.Equal(t, frameID, writer.registry.Frames[0].ID)
	require.Equal(t, "VALID", writer.registry.Frames[0].Z3.Status)

	if len(writer.registry.PQCManifest.FrameSignatures) == 0 {
		t.Error("expected signature in PQCManifest")
	}

	require.Equal(t, 1, writer.registry.GlobalMetrics.ValidFrameCount)
	require.Equal(t, 3, writer.registry.GlobalMetrics.TotalBracketCount)
}

func TestAppendMultipleFrames(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frames := []string{"frame-1", "frame-2", "frame-3"}
	for _, id := range frames {
		buf := make([]byte, 80)
		metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		err := writer.AppendFrame(id, buf, metas, thermo, nrv.LinguisticMapping{})
		require.NoError(t, err, "AppendFrame failed for %s", id)
	}

	require.Len(t, writer.registry.Frames, 3)
	require.Equal(t, 3, writer.registry.FrameCount)
	require.Equal(t, 3, writer.registry.GlobalMetrics.ValidFrameCount)
}

func TestAppendFrameUpdatesMetrics(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frameID := "test-frame"
	buf := make([]byte, 80)
	metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 75.0, PeakVoltV: 1.5, ClockMHz: 600}

	err := writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	require.NoError(t, err)

	require.Equal(t, 1, writer.registry.GlobalMetrics.ValidFrameCount)
	require.Equal(t, 1, writer.registry.GlobalMetrics.TotalBracketCount)
}

func TestAppendFrameWithSignature(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frameID := "signed-frame"
	buf := make([]byte, 80)
	metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}

	err := writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	require.NoError(t, err)

	_, ok := writer.registry.PQCManifest.FrameSignatures["signed-frame"]
	require.True(t, ok, "expected frame signature to be stored")
}

func TestNRVWriterClose(t *testing.T) {
	writer, _ := setupTestWriter(t)

	err := writer.Close()
	require.NoError(t, err)
}

func TestNewNRVWriterReopensExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer1, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	frameID := "persisted-frame"
	buf := make([]byte, 80)
	metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}

	err = writer1.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	require.NoError(t, err)
	writer1.Close()

	writer2, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)
	defer writer2.Close()

	require.Len(t, writer2.registry.Frames, 1)
}

func ptr(s string) *string {
	return &s
}
