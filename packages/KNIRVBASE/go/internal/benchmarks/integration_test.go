package benchmarks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/internal/network"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	"github.com/knirvcorp/knirvbase/go/pkg/knirvbase"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
	"github.com/stretchr/testify/require"
)

func TestEndToEnd_ASICPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	keyPair, err := pqc.GeneratePQCKeyPair("asic-pipeline", "test")
	require.NoError(t, err)

	db, err := knirvbase.NewNRV(context.Background(), knirvbase.Options{DataDir: tmpDir}, keyPair)
	require.NoError(t, err)

	dataset := db.Dataset("asic_pipeline")
	thermo := nrv.ThermoAtmosphere{AvgTempC: 71.5, PeakVoltV: 1.23, ClockMHz: 525}
	batches := []int{334, 333, 333}

	appendBatch := func(start, count int) {
		for i := 0; i < count; i++ {
			err := dataset.AppendBracket(context.Background(), integrationTestBracket(start+i), thermo)
			require.NoError(t, err)
		}
	}

	appendBatch(0, batches[0])
	time.Sleep(1100 * time.Millisecond)
	appendBatch(batches[0], batches[1])
	time.Sleep(1100 * time.Millisecond)
	appendBatch(batches[0]+batches[1], batches[2])
	time.Sleep(1100 * time.Millisecond)

	require.NoError(t, db.Shutdown())

	reader, err := stor.NewNRVReader(filepath.Join(tmpDir, "asic_pipeline.nrv"))
	require.NoError(t, err)
	defer reader.Close()

	registry := reader.GetRegistry()
	require.Len(t, registry.Frames, 3)
	require.Equal(t, 1000, registry.GlobalMetrics.TotalBracketCount)
	require.Equal(t, 3, registry.GlobalMetrics.ValidFrameCount)
	require.NotEmpty(t, registry.PQCManifest.FrameSignatures)
	require.Equal(t, "Dilithium-3", registry.PQCManifest.Algorithm)

	totalBrackets := 0
	for _, frame := range registry.Frames {
		totalBrackets += frame.Brackets.Count
		require.Equal(t, "VALID", frame.Z3.Status)
		require.NotEmpty(t, registry.PQCManifest.FrameSignatures[frame.ID])

		valid, err := reader.VerifyFrame(frame.ID, keyPair)
		require.NoError(t, err)
		require.True(t, valid)
	}
	require.Equal(t, 1000, totalBrackets)
}

type mockServer struct {
	data [][]byte
	ctx  context.Context
}

func (m *mockServer) Send(b *network.BracketBatch) error {
	m.data = append(m.data, b.Data)
	return nil
}

func (m *mockServer) Context() context.Context {
	return m.ctx
}

func TestEndToEnd_FlightGoldStream(t *testing.T) {
	tmpDir := t.TempDir()
	collection := "flight_data"
	path := filepath.Join(tmpDir, collection+".nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	validBuf := make([]byte, 160)
	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
		{ID: "b2", Type: nrv.DeltaTypeI, Offset: 80},
	}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}

	writer, err := stor.NewNRVWriter(path, keyPair)
	require.NoError(t, err)
	_ = writer.AppendFrame("valid-frame", validBuf, metas, thermo, nrv.LinguisticMapping{})
	writer.Close()

	store := stor.NewNRVStorage(tmpDir, keyPair)
	defer store.Close()

	flightServer := network.NewFlightServer(store)

	server := &mockServer{data: make([][]byte, 0), ctx: context.Background()}

	err = flightServer.StreamBrackets("gold."+collection, server)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(server.data), 1, "Gold stream should return data")
}

func TestEndToEnd_FlightResearchStream(t *testing.T) {
	tmpDir := t.TempDir()
	collection := "flight_data"
	path := filepath.Join(tmpDir, collection+".nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	validBuf := make([]byte, 80)
	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
	}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}

	writer, err := stor.NewNRVWriter(path, keyPair)
	require.NoError(t, err)
	_ = writer.AppendFrame("valid-frame", validBuf, metas, thermo, nrv.LinguisticMapping{})
	_ = writer.AppendFrame("invalid-frame", validBuf, metas, thermo, nrv.LinguisticMapping{})
	writer.Close()

	store := stor.NewNRVStorage(tmpDir, keyPair)
	defer store.Close()

	flightServer := network.NewFlightServer(store)

	server := &mockServer{data: make([][]byte, 0), ctx: context.Background()}

	err = flightServer.StreamBrackets("research."+collection, server)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(server.data), 1, "Research stream should return data")
}

func TestEndToEnd_CompactionPreservesGold(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "compaction.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	frameBuf := make([]byte, 80)
	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
	}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}

	writer, err := stor.NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		frameID := fmt.Sprintf("valid-frame-%d", i)
		require.NoError(t, writer.AppendFrame(frameID, frameBuf, metas, thermo, nrv.LinguisticMapping{}))
	}

	for i := 0; i < 3; i++ {
		frameID := fmt.Sprintf("invalid-frame-%d", i)
		require.NoError(t, writer.AppendFrame(frameID, frameBuf, metas, thermo, nrv.LinguisticMapping{}))
		registry := writer.GetRegistry()
		registry.Frames[len(registry.Frames)-1].Z3 = nrv.Z3Result{Status: "INVALID", Relevance: 0}
	}
	writer.GetRegistry().GlobalMetrics.ValidFrameCount = 5
	writer.GetRegistry().GlobalMetrics.InvalidFrameCount = 3
	require.NoError(t, writer.PersistRegistry())
	require.NoError(t, writer.Close())

	compactor := stor.NewCompactor(path, keyPair)
	reader, err := stor.NewNRVReader(path)
	require.NoError(t, err)
	compactor.MaybeCompact(reader.GetRegistry())
	reader.Close()

	require.Eventually(t, func() bool {
		compactReader, err := stor.NewNRVReader(path)
		if err != nil {
			return false
		}
		defer compactReader.Close()

		registry := compactReader.GetRegistry()
		if len(registry.Frames) != 5 {
			return false
		}
		for _, frame := range registry.Frames {
			if frame.Z3.Status != "VALID" || frame.Tombstone != nil {
				return false
			}
		}
		return true
	}, 3*time.Second, 50*time.Millisecond)
}

func TestEndToEnd_CrashRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "crash.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	require.NoError(t, err)

	writer, err := stor.NewNRVWriter(path, keyPair)
	require.NoError(t, err)

	initialBuf := make([]byte, 80*3)
	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
		{ID: "b2", Type: nrv.DeltaTypeI, Offset: 80},
		{ID: "b3", Type: nrv.DeltaTypeI, Offset: 160},
	}
	thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
	require.NoError(t, writer.AppendFrame("frame1", initialBuf, metas, thermo, nrv.LinguisticMapping{}))
	require.NoError(t, writer.Close())

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	lastGoodLength := fileInfo.Size()

	wal := stor.NewWAL(path + ".wal")
	require.NoError(t, wal.Begin(stor.WALEntry{
		FrameID:        "crash-frame",
		LastGoodLength: lastGoodLength,
		Committed:      false,
	}))

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	require.NoError(t, err)
	require.NoError(t, f.Close())

	recoveredWriter, err := stor.NewNRVWriter(path, keyPair)
	require.NoError(t, err)
	require.NoError(t, recoveredWriter.Close())

	fileInfo, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, lastGoodLength, fileInfo.Size())

	reader, err := stor.NewNRVReader(path)
	require.NoError(t, err)
	require.Len(t, reader.GetRegistry().Frames, 1)
	entry, _, err := reader.GetFrame("frame1")
	require.NoError(t, err)
	require.NotNil(t, entry)
	reader.Close()
}

func integrationTestBracket(i int) *nrv.Bracket {
	var projections [32]byte
	for j := range projections {
		projections[j] = byte((i + j) % 251)
	}
	return &nrv.Bracket{
		ID:          fmt.Sprintf("b-%d", i),
		Projections: projections,
		SubSecondUS: uint32(1000 + i),
		DepHead:     uint8(i % 256),
		GoldenSeed:  uint32(10000 + i),
	}
}
