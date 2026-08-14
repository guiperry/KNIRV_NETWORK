package storage

import (
	"encoding/json"
	"testing"

	"github.com/lab/hasher/data-seeder/pkg/training"
)

func TestConvertJSONRecordAcceptsEncoderFrameWithoutTokenSequence(t *testing.T) {
	raw := []byte(`{
		"source_file": "hf://stallone/goat/example",
		"chunk_id": 70,
		"window_start": 0,
		"window_end": 1,
		"context_length": 1,
		"asic_slot_0": -2147483648,
		"asic_slot_1": 10,
		"asic_slot_2": 11,
		"asic_slot_3": 12,
		"asic_slot_4": 13,
		"asic_slot_5": 14,
		"asic_slot_6": 15,
		"asic_slot_7": 16,
		"asic_slot_8": 17,
		"asic_slot_9": 18,
		"asic_slot_10": 19,
		"asic_slot_11": 1,
		"target_token_id": 25
	}`)

	var frame JSONTrainingRecord
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("failed to unmarshal encoder frame: %v", err)
	}

	record := NewDataIngestor(t.TempDir()).convertJSONRecord(&frame)
	if record == nil {
		t.Fatal("expected encoder frame to convert to a training record")
	}

	if len(record.TokenSequence) != 1 || record.TokenSequence[0] != frame.TargetTokenID {
		t.Fatalf("expected token sequence to be synthesized from target token, got %#v", record.TokenSequence)
	}
	if record.TargetToken != frame.TargetTokenID {
		t.Fatalf("expected target token %d, got %d", frame.TargetTokenID, record.TargetToken)
	}
}

func TestConvertJSONRecordPreservesTokenSequence(t *testing.T) {
	frame := &JSONTrainingRecord{
		SourceFile:    "source",
		ChunkID:       1,
		TargetTokenID: 25,
		TokenSequence: []int32{4, 5, 6},
		AsicSlots0:    1,
		AsicSlots1:    2,
		AsicSlots2:    3,
		AsicSlots3:    4,
		AsicSlots4:    5,
		AsicSlots5:    6,
		AsicSlots6:    7,
		AsicSlots7:    8,
		AsicSlots8:    9,
		AsicSlots9:    10,
		AsicSlots10:   11,
		AsicSlots11:   12,
	}

	record := NewDataIngestor(t.TempDir()).convertJSONRecord(frame)
	if record == nil {
		t.Fatal("expected JSON frame to convert to a training record")
	}

	if got, want := record.TokenSequence, frame.TokenSequence; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("expected token sequence %#v, got %#v", want, got)
	}
	if got, want := record.ContextHash, training.ComputeContextHash(frame.TokenSequence, 5); got != want {
		t.Fatalf("context hash = %d, want computed hash %d", got, want)
	}
}

func TestConvertJSONRecordPreservesAssertionSpan(t *testing.T) {
	frame := &JSONTrainingRecord{
		SchemaVersion: 2,
		TargetTokenID: 30,
		TokenSequence: []int32{4, 5, 6},
		AssertionSpan: []int32{7, 8, 9},
		AsicSlots0:    1, AsicSlots1: 2, AsicSlots2: 3, AsicSlots3: 4,
		AsicSlots4: 5, AsicSlots5: 6, AsicSlots6: 7, AsicSlots7: 8,
		AsicSlots8: 9, AsicSlots9: 10, AsicSlots10: 11, AsicSlots11: 12,
	}
	record := NewDataIngestor(t.TempDir()).convertJSONRecord(frame)
	if record == nil {
		t.Fatal("expected JSON frame to convert to a training record")
	}
	if len(record.AssertionSpan) != 3 || record.AssertionSpan[2] != 9 {
		t.Fatalf("assertion span = %#v, want [7 8 9]", record.AssertionSpan)
	}
}
