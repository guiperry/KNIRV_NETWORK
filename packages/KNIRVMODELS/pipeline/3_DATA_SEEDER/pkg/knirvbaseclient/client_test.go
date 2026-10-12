package knirvbaseclient

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNRVUsesPublishedArtifact(t *testing.T) {
	dataDir := t.TempDir()
	framesDir := filepath.Join(dataDir, "frames")
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifact := "training_frames-20260925T120000.000000000Z-1-1.nrv"
	if err := os.WriteFile(filepath.Join(framesDir, artifact), []byte("NRV2"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(framesDir, "training_frames.nrv.latest.json"), []byte(`{"artifact":"`+artifact+`"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveNRV(dataDir); got != filepath.Join(framesDir, artifact) {
		t.Fatalf("ResolveNRV() = %q, want %q", got, filepath.Join(framesDir, artifact))
	}
}

func TestSubmitNRVOnceRecordsImmutableArtifactReceipt(t *testing.T) {
	framesDir := t.TempDir()
	artifactDir := filepath.Join(framesDir, "batches", "batch-1")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta, err := json.Marshal(map[string]interface{}{"version": "3.0", "frames": []interface{}{}})
	if err != nil {
		t.Fatal(err)
	}
	var raw [80]byte
	binary.LittleEndian.PutUint16(raw[39:41], 0x2000)
	data := append([]byte("NRV3"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(meta)))
	data = append(data, meta...)
	data = append(data, raw[:]...)
	path := filepath.Join(artifactDir, "training_frames.nrv")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := New(strings.TrimPrefix(server.URL, "http://"))
	count, already, err := client.SubmitNRVOnce(path)
	if err != nil || already || count != 1 {
		t.Fatalf("first submit = count=%d already=%v err=%v", count, already, err)
	}
	count, already, err = client.SubmitNRVOnce(path)
	if err != nil || !already || count != 0 {
		t.Fatalf("second submit = count=%d already=%v err=%v", count, already, err)
	}
	if requests != 1 {
		t.Fatalf("append requests = %d, want 1", requests)
	}
}

func TestMigrateV2BracketUsesCanonicalLayout(t *testing.T) {
	var legacy [80]byte
	legacy[36], legacy[37], legacy[38], legacy[39] = 3, 2, 1, 0xfe
	legacy[40] = 0x7f
	binary.LittleEndian.PutUint16(legacy[41:43], 0x2001)
	binary.LittleEndian.PutUint32(legacy[43:47], 0x11223344)
	binary.LittleEndian.PutUint32(legacy[61:65], 0x55667788)
	got := migrateV2Bracket(legacy)
	if got[36] != 0x63 || got[37] != 0xfe || got[38] != 0x7f {
		t.Fatalf("scalar conversion mismatch: %x", got[36:39])
	}
	if binary.LittleEndian.Uint16(got[39:41]) != 0x2001 || binary.LittleEndian.Uint32(got[41:45]) != 0x11223344 || binary.LittleEndian.Uint32(got[59:63]) != 0x55667788 {
		t.Fatalf("canonical offsets not populated")
	}
}
