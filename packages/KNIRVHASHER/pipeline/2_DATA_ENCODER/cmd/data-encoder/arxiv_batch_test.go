package main

import (
	"testing"

	"data-encoder/pkg/schema"
)

func TestArxivUsesNonOverlappingWindows(t *testing.T) {
	record := &schema.MinedRecord{FileName: "arxiv:cs.LG/papers/17"}
	config := &Config{WindowSize: 128, WindowStride: 1}
	if got := effectiveWindowStride(record, config); got != 128 {
		t.Fatalf("stride=%d, want 128", got)
	}
}
