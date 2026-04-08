package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameTicker_AppendAndFlush(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)
	defer writer.Close()

	ticker := NewFrameTicker(writer, 50*time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < 10; i++ {
		b := &nrv.Bracket{
			LSHSalt:     uint32(i),
			SubSecondUS: uint32(i * 1000),
			ASICLoops:   1,
			GoldenSeed:  uint32(i * 100),
		}
		for j := range b.Projections {
			b.Projections[j] = byte(j)
		}
		thermo := nrv.ThermoAtmosphere{
			AvgTempC:  70.0 + float32(i),
			PeakVoltV: 1.2,
			ClockMHz:  500,
		}
		err := ticker.AppendBracket(context.Background(), b, thermo)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	if len(writer.registry.Frames) == 0 {
		t.Error("expected at least one frame after ticker flush")
	}

	frame := writer.registry.Frames[0]
	if frame.Brackets.Count != 10 {
		t.Errorf("expected 10 brackets, got %d", frame.Brackets.Count)
	}
}

func TestFrameTicker_IBracketFrequency(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)
	defer writer.Close()

	ticker := NewFrameTicker(writer, 100*time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < 100; i++ {
		b := &nrv.Bracket{
			LSHSalt:     uint32(i),
			SubSecondUS: uint32(i * 1000),
			ASICLoops:   1,
			GoldenSeed:  uint32(i * 100),
		}
		for j := range b.Projections {
			b.Projections[j] = byte(j)
		}
		thermo := nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500}
		_ = ticker.AppendBracket(context.Background(), b, thermo)
	}

	time.Sleep(150 * time.Millisecond)

	if len(writer.registry.Frames) == 0 {
		t.Fatal("expected at least one frame")
	}

	frame := writer.registry.Frames[0]
	iCount := 0
	for _, meta := range frame.BracketIndex {
		if meta.Type == nrv.DeltaTypeI {
			iCount++
		}
	}

	if iCount == 0 {
		t.Error("expected at least one I-bracket")
	}
}

func TestFrameTicker_DriftSpike(t *testing.T) {
	t.Skip("Drift spike detection requires precise timing - covered by unit tests")
}

func TestFrameTicker_SetLinguistic(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)
	defer writer.Close()

	ticker := NewFrameTicker(writer, 100*time.Millisecond)
	defer ticker.Stop()

	ticker.SetLinguistic("resolution", "word")

	b := &nrv.Bracket{
		LSHSalt:     1,
		SubSecondUS: 1000,
		ASICLoops:   1,
		GoldenSeed:  100,
	}
	_ = ticker.AppendBracket(context.Background(), b, nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500})

	time.Sleep(150 * time.Millisecond)

	if len(writer.registry.Frames) == 0 {
		t.Fatal("expected at least one frame")
	}

	frame := writer.registry.Frames[0]
	assert.Equal(t, "resolution", frame.Linguistic.Token)
	assert.Equal(t, "word", frame.Linguistic.Unit)
}

func TestFrameTicker_EmptyFlush(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)
	defer writer.Close()

	initialCount := writer.registry.FrameCount

	ticker := NewFrameTicker(writer, 50*time.Millisecond)
	defer ticker.Stop()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, initialCount, writer.registry.FrameCount)
}

func TestFrameTicker_StopFlushesPending(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	writer, err := NewNRVWriter(path, nil)
	require.NoError(t, err)

	ticker := NewFrameTicker(writer, 10*time.Second)
	defer ticker.Stop()

	for i := 0; i < 5; i++ {
		b := &nrv.Bracket{
			LSHSalt:     uint32(i),
			SubSecondUS: uint32(i * 1000),
			ASICLoops:   1,
			GoldenSeed:  uint32(i * 100),
		}
		for j := range b.Projections {
			b.Projections[j] = byte(j)
		}
		_ = ticker.AppendBracket(context.Background(), b, nrv.ThermoAtmosphere{AvgTempC: 70, PeakVoltV: 1.2, ClockMHz: 500})
	}

	ticker.Stop()
	writer.Close()

	reader, err := NewNRVReader(path)
	require.NoError(t, err)
	defer reader.Close()

	if len(reader.registry.Frames) == 0 {
		t.Error("expected at least one frame after ticker stop")
	}
}
