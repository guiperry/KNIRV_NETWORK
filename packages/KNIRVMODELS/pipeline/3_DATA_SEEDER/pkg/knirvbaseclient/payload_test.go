package knirvbaseclient

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	knirvnrv "github.com/knirvcorp/knirvbase/pkg/nrv"
)

// sharedFramePayloadCases is the cross-language conformance fixture in KNIRVBASE.
const sharedFramePayloadCases = "../../../../../KNIRVBASE/testdata/framepayload/cases.json"

type testFrame struct {
	id      string
	count   int
	payload json.RawMessage
}

// buildNRV3 lays out an NRV3 file the way the encoder writes it: magic, a JSON
// registry carrying each frame's payload, then the 80-byte brackets back to back.
func buildNRV3(t *testing.T, frames []testFrame) []byte {
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
	for _, f := range frames {
		reg.Frames = append(reg.Frames, entry{ID: f.id, Count: f.count, Offset: int64(len(section)), Payload: f.payload})
		section = append(section, make([]byte, f.count*80)...)
	}
	meta, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	data := append([]byte("NRV3"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(meta)))
	data = append(data, meta...)
	return append(data, section...)
}

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "training_frames.nrv")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSharedFixtureReadBySeeder reads every shared case from a file. An ok case
// must decode to the canonical payload; an error case must make the seeder refuse
// the file.
func TestSharedFixtureReadBySeeder(t *testing.T) {
	raw, err := os.ReadFile(sharedFramePayloadCases)
	if err != nil {
		t.Fatalf("read shared fixture: %v", err)
	}
	var doc struct {
		Cases []struct {
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
			Expect    string          `json:"expect"`
			Canonical json.RawMessage `json:"canonical"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			count := 1
			if p, err := knirvnrv.DecodeFramePayload(c.Input); err == nil && p.TokenWindow != nil {
				count = int(p.TokenWindow.Brackets)
			}
			data := buildNRV3(t, []testFrame{{id: "case", count: count, payload: c.Input}})
			payloads, err := NRVAssertionPayloads(data)
			switch c.Expect {
			case "error":
				if err == nil {
					t.Fatalf("seeder accepted a payload the fixture says is invalid")
				}
			case "ok":
				if err != nil {
					t.Fatalf("seeder refused a valid payload: %v", err)
				}
				back, err := json.Marshal(*payloads[0])
				if err != nil {
					t.Fatal(err)
				}
				var got, want interface{}
				if err := json.Unmarshal(back, &got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(c.Canonical, &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("seeder payload differs from canonical:\n got  %s\n want %s", back, c.Canonical)
				}
			default:
				t.Fatalf("unknown expect %q", c.Expect)
			}
		})
	}
}

func TestSubmitNRVRefusesInvalidPayloadBeforeAppend(t *testing.T) {
	// Tier 9 is outside the tool tier set 0..=3.
	bad := json.RawMessage(`{"supervision_episode":{"tool_trace":[{"tool":"x","tier":9}]}}`)
	path := writeTemp(t, buildNRV3(t, []testFrame{{id: "ok", count: 1}, {id: "bad", count: 1, payload: bad}}))
	var appends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appends.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := New(strings.TrimPrefix(server.URL, "http://"))
	if _, err := client.SubmitNRV(path); err == nil {
		t.Fatal("SubmitNRV accepted a file with an invalid payload")
	}
	if n := appends.Load(); n != 0 {
		t.Fatalf("%d brackets were appended from a refused file", n)
	}
}

func TestSubmitNRVAppendsEveryBracketOfValidAssertions(t *testing.T) {
	good, err := knirvnrv.EncodeFramePayload(knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Tokens: []uint32{4, 5}, Start: 2, Brackets: 2}})
	if err != nil {
		t.Fatal(err)
	}
	path := writeTemp(t, buildNRV3(t, []testFrame{{id: "a", count: 2, payload: good}, {id: "b", count: 1}}))
	var appends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appends.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	n, err := New(strings.TrimPrefix(server.URL, "http://")).SubmitNRV(path)
	if err != nil || n != 3 || appends.Load() != 3 {
		t.Fatalf("SubmitNRV = %d, %v (appends %d), want 3 brackets", n, err, appends.Load())
	}
}

func TestNRVAssertionPayloadsRejectsTokenWindowBracketMismatch(t *testing.T) {
	p, err := knirvnrv.EncodeFramePayload(knirvnrv.FramePayload{TokenWindow: &knirvnrv.TokenWindowPayload{Brackets: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NRVAssertionPayloads(buildNRV3(t, []testFrame{{id: "m", count: 1, payload: p}})); err == nil {
		t.Fatal("bracket-count mismatch accepted")
	}
}

// rawNRV3 builds an NRV3 file from an explicit registry and bracket section, so a
// test can lay out frames that overlap, gap or leave bytes uncovered.
func rawNRV3(t *testing.T, frames []map[string]interface{}, section int) []byte {
	t.Helper()
	meta, err := json.Marshal(map[string]interface{}{"version": "3.0", "frames": frames})
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	data := append([]byte("NRV3"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(meta)))
	data = append(data, meta...)
	return append(data, make([]byte, section)...)
}

// TestNRVAssertionPayloadsRefusesBadTiling checks that frames must cover the
// bracket section exactly: overlapping, gapped and trailing-byte layouts are refused.
func TestNRVAssertionPayloadsRefusesBadTiling(t *testing.T) {
	frame := func(id string, count int, offset int) map[string]interface{} {
		return map[string]interface{}{"id": id, "count": count, "offset": offset}
	}
	cases := map[string][]byte{
		"overlap":      rawNRV3(t, []map[string]interface{}{frame("a", 2, 0), frame("b", 1, 80)}, 3*80),
		"gap":          rawNRV3(t, []map[string]interface{}{frame("a", 1, 0), frame("b", 1, 160)}, 3*80),
		"trailing":     rawNRV3(t, []map[string]interface{}{frame("a", 1, 0)}, 2*80),
		"out of order": rawNRV3(t, []map[string]interface{}{frame("b", 1, 80), frame("a", 1, 0)}, 2*80),
	}
	for name, data := range cases {
		if _, err := NRVAssertionPayloads(data); err == nil {
			t.Fatalf("%s: tiling violation accepted", name)
		}
	}
	tiled := rawNRV3(t, []map[string]interface{}{frame("a", 1, 0), frame("b", 2, 80)}, 3*80)
	if _, err := NRVAssertionPayloads(tiled); err != nil {
		t.Fatalf("tiled file refused: %v", err)
	}
}
