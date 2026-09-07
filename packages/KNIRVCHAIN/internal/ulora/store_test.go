package ulora

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

func TestStoreUsesActualBundleBytes(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("actual .ulora bundle bytes")
	hash, err := s.Put(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(data)
	if hash != hex.EncodeToString(want[:]) {
		t.Fatalf("hash = %s, want actual bytes digest", hash)
	}
	f, err := s.Open(hash)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := io.ReadAll(f)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("stored bytes = %q, err = %v", got, err)
	}
}
