package benchmarks

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/knirvcorp/knirvbase/go/internal/network"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
	"github.com/stretchr/testify/require"
)

func BenchmarkAppendBracket(b *testing.B) {
	tmpDir := b.TempDir()
	writer, err := stor.NewNRVWriter(filepath.Join(tmpDir, "append.nrv"), nil)
	require.NoError(b, err)
	defer writer.Close()

	ticker := stor.NewFrameTicker(writer, time.Hour)
	defer ticker.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			err := ticker.AppendBracket(context.Background(), integrationTestBracket(i), nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500})
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkFlush_1000Brackets(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writer, err := stor.NewNRVWriter(filepath.Join(tmpDir, "flush.nrv"), nil)
		require.NoError(b, err)

		ticker := stor.NewFrameTicker(writer, time.Hour)
		for j := 0; j < 1000; j++ {
			err = ticker.AppendBracket(context.Background(), integrationTestBracket(j), nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500})
			require.NoError(b, err)
		}

		b.StartTimer()
		ticker.Stop()
		b.StopTimer()

		require.NoError(b, writer.Close())
	}
}

func BenchmarkStreamBrackets_Gold(b *testing.B) {
	tmpDir := b.TempDir()
	collection := "gold_stream"
	path := filepath.Join(tmpDir, collection+".nrv")

	writer, err := stor.NewNRVWriter(path, nil)
	require.NoError(b, err)

	for frame := 0; frame < 10; frame++ {
		buf := make([]byte, 0, 100*nrv.BracketSize)
		metas := make([]nrv.BracketMeta, 0, 100)
		for j := 0; j < 100; j++ {
			bracket := integrationTestBracket(frame*100 + j)
			encoded := nrv.EncodeBracket(bracket)
			buf = append(buf, encoded[:]...)
			metas = append(metas, nrv.BracketMeta{ID: bracket.ID, Type: nrv.DeltaTypeI, Offset: j * nrv.BracketSize})
		}
		require.NoError(b, writer.AppendFrame(integrationTestBracket(frame).ID, buf, metas, nrv.ThermoAtmosphere{}, nrv.LinguisticMapping{}))
	}
	require.NoError(b, writer.Close())

	store := stor.NewNRVStorage(tmpDir, nil)
	defer store.Close()
	flightServer := network.NewFlightServer(store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := &mockServer{ctx: context.Background()}
		err := flightServer.StreamBrackets("gold."+collection, server)
		if err != nil {
			b.Fatal(err)
		}
		totalBytes := 0
		for _, batch := range server.data {
			totalBytes += len(batch)
		}
		b.SetBytes(int64(totalBytes))
	}
}

func BenchmarkDecodePBracket(b *testing.B) {
	anchor := integrationTestBracket(1).Projections
	target := integrationTestBracket(2).Projections
	delta := nrv.XORProjections(target, anchor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded := nrv.ApplyProjectionDelta(delta, anchor)
		if decoded != target {
			b.Fatal("decoded projections did not match target")
		}
	}
}
