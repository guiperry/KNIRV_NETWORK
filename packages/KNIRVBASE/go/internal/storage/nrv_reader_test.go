package storage

import (
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
	"github.com/stretchr/testify/require"
)

func setupTestReader(t *testing.T) (*NRVReader, *NRVWriter, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	frameID := "reader-test-frame"
	buf := make([]byte, 80)
	metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
	_ = writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	writer.Close()

	reader, err := NewNRVReader(path)
	require.NoError(t, err)

	return reader, writer, path
}

func TestNewNRVReader(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	require.NotNil(t, reader.registry)
	require.Len(t, reader.registry.Frames, 1)
}

func TestNRVReaderGetFrame(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	entry, brackets, err := reader.GetFrame("reader-test-frame")
	require.NoError(t, err)
	require.NotNil(t, entry)
	require.Equal(t, "reader-test-frame", entry.ID)
	require.NotEmpty(t, brackets)
}

func TestNRVReaderGetFrameNotFound(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	entry, brackets, err := reader.GetFrame("nonexistent-frame")
	require.Error(t, err)
	require.Nil(t, entry)
	require.Nil(t, brackets)
}

func TestNRVReaderStreamBrackets_GoldOnly(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)

	frameID1 := "frame-valid"
	buf1 := make([]byte, 80*2)
	metas1 := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
		{ID: "b2", Type: nrv.DeltaTypeI, Offset: 80},
	}
	_ = writer.AppendFrame(frameID1, buf1, metas1, nrv.ThermoAtmosphere{}, nrv.LinguisticMapping{})

	frameID2 := "frame-invalid"
	bufInvalid := make([]byte, 80*1)
	metas2 := []nrv.BracketMeta{
		{ID: "b3", Type: nrv.DeltaTypeI, Offset: 0},
	}
	_ = writer.AppendFrame(frameID2, bufInvalid, metas2, nrv.ThermoAtmosphere{}, nrv.LinguisticMapping{})
	writer.registry.Frames[1].Z3 = nrv.Z3Result{Status: "INVALID", Relevance: 0.1}
	writer.registry.GlobalMetrics.InvalidFrameCount = 1
	writer.saveRegistry()
	writer.Close()

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	goldCh := reader.StreamBrackets(true)
	goldCount := 0
	for range goldCh {
		goldCount++
	}
	require.Equal(t, 2, goldCount, "Gold stream should have 2 brackets from VALID frame")

	researchCh := reader.StreamBrackets(false)
	researchCount := 0
	for range researchCh {
		researchCount++
	}
	require.Equal(t, 3, researchCount, "Research stream should have all 3 brackets")
}

func TestNRVReaderVerifyFrame(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	frameID := "verify-frame"
	buf := make([]byte, 80)
	metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
	_ = writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	writer.Close()

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	valid, err := reader.VerifyFrame(frameID, keyPair)
	require.NoError(t, err)
	require.True(t, valid, "expected frame signature to be valid")
}

func TestNRVReaderClose(t *testing.T) {
	reader, _, _ := setupTestReader(t)

	err := reader.Close()
	require.NoError(t, err)
}

func TestNRVReader_DecodePBracket(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)

	anchorID := "anchor-1"
	proj1 := [32]byte{}
	for i := 0; i < 32; i++ {
		proj1[i] = byte(i)
	}
	anchorBracket := &nrv.Bracket{
		ID:          anchorID,
		Projections: proj1,
		SubSecondUS: 1000,
		DepHead:     1,
		GoldenSeed:  42,
	}
	encoded1 := nrv.EncodeBracket(anchorBracket)

	targetProj := [32]byte{}
	for i := 0; i < 32; i++ {
		targetProj[i] = byte(255 - i)
	}
	pBracket := &nrv.Bracket{
		ID:          "p-bracket",
		Projections: nrv.XORProjections(targetProj, proj1),
		SubSecondUS: 1001,
		DepHead:     2,
		GoldenSeed:  43,
	}
	encodedP := nrv.EncodeBracket(pBracket)

	frameID := "frame-p-bracket"
	buf := make([]byte, 80*2)
	copy(buf[0:80], encoded1[:])
	copy(buf[80:160], encodedP[:])

	anchorPtr := anchorID
	driftScore := 0.15
	metas := []nrv.BracketMeta{
		{ID: anchorID, Type: nrv.DeltaTypeI, Offset: 0},
		{ID: "p-bracket", Type: nrv.DeltaTypeP, AnchorID: &anchorPtr, Offset: 80, DriftScore: driftScore},
	}
	_ = writer.AppendFrame(frameID, buf, metas, nrv.ThermoAtmosphere{}, nrv.LinguisticMapping{})
	writer.Close()

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	entry, brackets, err := reader.GetFrame(frameID)
	require.NoError(t, err)
	require.NotNil(t, entry)
	require.Len(t, brackets, 2)

	require.Equal(t, anchorID, brackets[0].ID)
	require.NotNil(t, brackets[0].Meta)
	require.Equal(t, nrv.DeltaTypeI, brackets[0].Meta.Type)

	require.Equal(t, "p-bracket", brackets[1].ID)
	require.NotNil(t, brackets[1].Meta)
	require.Equal(t, nrv.DeltaTypeP, brackets[1].Meta.Type)
	require.Equal(t, driftScore, brackets[1].Meta.DriftScore)
	require.Equal(t, targetProj, brackets[1].Projections)
}
