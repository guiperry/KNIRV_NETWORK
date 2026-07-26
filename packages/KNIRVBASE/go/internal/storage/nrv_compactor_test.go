package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
	"github.com/stretchr/testify/require"
)

func setupTestCompactor(t *testing.T) (*Compactor, *NRVWriter, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer, err := NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	compactor := NewCompactor(path, keyPair)

	return compactor, writer, path
}

func TestCompactorMaybeCompactBelowThreshold(t *testing.T) {
	compactor, writer, _ := setupTestCompactor(t)
	defer writer.Close()

	for i := 0; i < 10; i++ {
		frameID := "frame-below-threshold"
		buf := make([]byte, 80)
		metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		_ = writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	}

	compactor.MaybeCompact(writer.registry)

	time.Sleep(100 * time.Millisecond)

	require.False(t, compactor.running, "expected compaction not to start below threshold")
}

func TestCompactorMaybeCompactAboveThreshold(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	totalFrames := 10
	deleteCount := 3

	for i := 0; i < totalFrames; i++ {
		frameID := "frame-above-threshold"
		buf := make([]byte, 80)
		metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		_ = writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	}

	for i := 0; i < deleteCount; i++ {
		nowNano := time.Now().UnixNano()
		writer.registry.Frames[i].Tombstone = &nowNano
		writer.registry.TombstoneCount++
	}

	err := writer.saveRegistry()
	require.NoError(t, err)

	ratio := float64(writer.registry.TombstoneCount) / float64(writer.registry.FrameCount)
	require.GreaterOrEqual(t, ratio, compactionThreshold, "test setup error: ratio is below threshold")

	compactor.MaybeCompact(writer.registry)

	time.Sleep(500 * time.Millisecond)

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	liveCount := 0
	for _, entry := range reader.registry.Frames {
		if entry.Tombstone == nil {
			liveCount++
		}
	}

	expectedLive := totalFrames - deleteCount
	require.Equal(t, expectedLive, liveCount)
}

func TestCompactorCompactPreservesData(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	frameIDs := []string{"compact-frame-1", "compact-frame-2", "compact-frame-3"}
	for _, id := range frameIDs {
		buf := make([]byte, 80)
		metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		_ = writer.AppendFrame(id, buf, metas, thermo, nrv.LinguisticMapping{})
	}

	nowNano := time.Now().UnixNano()
	writer.registry.Frames[0].Tombstone = &nowNano
	writer.registry.TombstoneCount++

	err := writer.saveRegistry()
	require.NoError(t, err)

	err = compactor.compact()
	require.NoError(t, err)

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	require.Len(t, reader.registry.Frames, 2)

	for _, entry := range reader.registry.Frames {
		require.Nil(t, entry.Tombstone, "expected no tombstoned frames after compaction")
	}
}

func TestCompactorCompactUpdatesMetrics(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	for i := 0; i < 5; i++ {
		frameID := "metrics-frame"
		buf := make([]byte, 80)
		metas := []nrv.BracketMeta{{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0}}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		_ = writer.AppendFrame(frameID, buf, metas, thermo, nrv.LinguisticMapping{})
	}

	nowNano := time.Now().UnixNano()
	writer.registry.Frames[0].Tombstone = &nowNano
	writer.registry.TombstoneCount++

	err := writer.saveRegistry()
	require.NoError(t, err)

	err = compactor.compact()
	require.NoError(t, err)

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	require.Equal(t, 0, reader.registry.TombstoneCount)
	require.NotNil(t, reader.registry.GlobalMetrics.CompactedAt)
}

func TestCompactorStartStop(t *testing.T) {
	compactor, _, _ := setupTestCompactor(t)

	compactor.Start()

	time.Sleep(50 * time.Millisecond)

	compactor.Stop()

	require.False(t, compactor.running)
}
