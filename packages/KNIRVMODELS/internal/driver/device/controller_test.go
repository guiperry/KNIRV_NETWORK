package device

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func TestBuildTxTaskFromHeaderPreservesKNIRVTarget(t *testing.T) {
	header := make([]byte, 80)
	const target = uint32(0x1f0fffff)
	binary.LittleEndian.PutUint32(header[72:76], target)
	packet := BuildTxTaskFromHeader(header, 7)
	if len(packet) != 52 || packet[5] != 7 {
		t.Fatalf("invalid task packet: len=%d workID=%d", len(packet), packet[5])
	}
	if got := binary.LittleEndian.Uint32(packet[46:50]); got != target {
		t.Fatalf("nBits = %#x, want %#x", got, target)
	}
}

func TestComputeHashFromNonceReconstructsDoubleSHA256Header(t *testing.T) {
	header := make([]byte, 80)
	for i := range header {
		header[i] = byte(i)
	}
	const nonce = uint32(0xdecafbad)
	d := &Device{}
	got := d.computeHashFromNonce(header, nonce)
	binary.LittleEndian.PutUint32(header[76:80], nonce)
	first := sha256.Sum256(header)
	want := sha256.Sum256(first[:])
	if got != want {
		t.Fatalf("got %x, want %x", got, want)
	}
}
