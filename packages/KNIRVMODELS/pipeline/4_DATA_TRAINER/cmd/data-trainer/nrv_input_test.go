package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nrvFile lays out a minimal NRV3 file: one frame of one zero bracket per entry in
// payloads, each carrying the given raw payload.
func nrvFile(t *testing.T, payloads []string) string {
	t.Helper()
	type entry struct {
		ID      string          `json:"id"`
		Count   int             `json:"count"`
		Offset  int64           `json:"offset"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}
	var reg struct {
		Version string  `json:"version"`
		Frames  []entry `json:"frames"`
	}
	reg.Version = "3.0"
	var section []byte
	for i, p := range payloads {
		reg.Frames = append(reg.Frames, entry{ID: "f" + string(rune('0'+i)), Count: 1, Offset: int64(len(section)), Payload: json.RawMessage(p)})
		section = append(section, make([]byte, 80)...)
	}
	meta := []byte(mustJSON(t, reg))
	data := append([]byte("NRV3"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(meta)))
	data = append(data, meta...)
	data = append(data, section...)
	path := filepath.Join(t.TempDir(), "in.nrv")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestRunNRVInputValidatesThenRefuses: a valid artifact is validated and refused
// with a message that says why; the JSON path is not taken.
func TestRunNRVInputValidatesThenRefuses(t *testing.T) {
	path := nrvFile(t, []string{`{"token_window":{"tokens":[1,2],"start":0,"brackets":1}}`})
	err := Run(&Config{InputPath: path})
	if err == nil {
		t.Fatal("trainer accepted .nrv input it cannot yet train from")
	}
	if !strings.Contains(err.Error(), "1 assertions validated") || !strings.Contains(err.Error(), "JSON artifact") {
		t.Fatalf("error = %v", err)
	}
}

// TestRunNRVInputFailsClosedOnBadPayload: one invalid payload fails the load with the
// payload error, not the refusal.
func TestRunNRVInputFailsClosedOnBadPayload(t *testing.T) {
	path := nrvFile(t, []string{
		`{"token_window":{"tokens":[1],"start":0,"brackets":1}}`,
		`{"supervision_episode":{"tool_trace":[{"tool":"x","tier":9}]}}`,
	})
	err := Run(&Config{InputPath: path})
	if err == nil || !strings.HasPrefix(err.Error(), "load nrv input") {
		t.Fatalf("error = %v, want a load failure", err)
	}
}

// TestIsNRVInput keeps the JSON selection unchanged.
func TestIsNRVInput(t *testing.T) {
	for p, want := range map[string]bool{"x.nrv": true, "X.NRV": true, "training_frames.json": false, "latest.json": false} {
		if got := isNRVInput(p); got != want {
			t.Errorf("isNRVInput(%q) = %v, want %v", p, got, want)
		}
	}
}
