package jitter

import (
	"bytes"
	"testing"
)

func TestBatchLoopMatchesSequentialSoftwareReference(t *testing.T) {
	cfg := DefaultJitterConfig()
	cfg.EnableFlashSearch = false
	headers := make([][]byte, 3)
	for i := range headers {
		headers[i] = make([]byte, 80)
		headers[i][0] = byte(i + 1)
		headers[i][79] = byte(10 + i)
	}
	sequential := NewJitterEngine(cfg)
	want := make([]*GoldenNonceResult, len(headers))
	for i := range headers {
		var err error
		want[i], err = sequential.Execute21PassLoop(headers[i], 0x12345678)
		if err != nil {
			t.Fatal(err)
		}
	}
	batched := NewJitterEngine(cfg)
	got, err := batched.Execute21PassLoopBatch(headers, 0x12345678)
	if err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if !bytes.Equal(got[i].FinalHash[:], want[i].FinalHash[:]) || got[i].Stability != want[i].Stability || got[i].Alignment != want[i].Alignment {
			t.Fatalf("batch result %d differs from sequential reference", i)
		}
	}
}
