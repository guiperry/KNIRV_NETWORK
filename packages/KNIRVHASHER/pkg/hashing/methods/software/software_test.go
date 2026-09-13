package software

import (
	"testing"

	"knirvhasher/pkg/hashing/core"
	"knirvhasher/pkg/hashing/jitter"
)

func TestSoftwareMethodHashesExactInputBytes(t *testing.T) {
	m := NewSoftwareMethod()
	if err := m.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { _ = m.Shutdown() })

	inputs := [][]byte{
		[]byte("KNIRV attestation input"),
		make([]byte, 80),
		{0x00, 0xff, 0x10, 0x00, 0x7f},
	}
	got, err := m.ComputeBatch(inputs)
	if err != nil {
		t.Fatalf("ComputeBatch: %v", err)
	}

	canonical := core.NewCanonicalSHA256()
	for i, input := range inputs {
		if want := canonical.ComputeSHA256(input); got[i] != want {
			t.Fatalf("input %d: got %x, want %x", i, got[i], want)
		}
	}
	if !m.GetCapabilities().ProductionReady {
		t.Fatal("software PoW must advertise production readiness")
	}
}

func TestSoftwareMiningUsesKNIRVOwnedPayloadAndDifficulty(t *testing.T) {
	m := NewSoftwareMethodWithDifficulty(8)
	if err := m.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Shutdown() })
	payload := []byte("assertion payload, not a Bitcoin header")
	nonce, err := m.MineHeader(payload, 0, 4096)
	if err != nil {
		t.Fatalf("MineHeader: %v", err)
	}
	canonical := core.NewCanonicalSHA256WithDifficulty(8)
	if !canonical.IsValidProofOfWork(canonical.ComputeProofOfWork(payload, nonce)) {
		t.Fatal("nonce is not a valid KNIRV PoW witness")
	}
}

func TestSoftwareJitterHashMatchesCanonicalDoubleHash(t *testing.T) {
	header := make([]byte, 80)
	for i := range header {
		header[i] = byte(i)
	}

	got, err := (&jitter.SoftwareHashMethod{}).ComputeDoubleHash(header)
	if err != nil {
		t.Fatalf("ComputeDoubleHash: %v", err)
	}
	want := core.NewCanonicalSHA256().ComputeDoubleSHA256(header)
	if got != want {
		t.Fatalf("got %x, want %x", got, want)
	}
}
