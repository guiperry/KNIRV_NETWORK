package loader

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

// rawTrainerNRV3 lays out an NRV3 file from an explicit registry and a bracket
// section of the given size, so a test can make frames overlap, gap or leave bytes
// uncovered.
func rawTrainerNRV3(t *testing.T, frames []map[string]interface{}, section int) []byte {
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

// TestParseNRVAssertionsRefusesBadTiling checks that frames must cover the bracket
// section exactly, in order. Overlapping, gapped, out-of-order and trailing-byte
// layouts fail the whole file.
func TestParseNRVAssertionsRefusesBadTiling(t *testing.T) {
	frame := func(id string, count, offset int) map[string]interface{} {
		return map[string]interface{}{"id": id, "count": count, "offset": offset}
	}
	cases := map[string][]byte{
		"overlap":      rawTrainerNRV3(t, []map[string]interface{}{frame("a", 2, 0), frame("b", 1, 80)}, 3*80),
		"gap":          rawTrainerNRV3(t, []map[string]interface{}{frame("a", 1, 0), frame("b", 1, 160)}, 3*80),
		"trailing":     rawTrainerNRV3(t, []map[string]interface{}{frame("a", 1, 0)}, 2*80),
		"out of order": rawTrainerNRV3(t, []map[string]interface{}{frame("b", 1, 80), frame("a", 1, 0)}, 2*80),
	}
	for name, data := range cases {
		if _, err := ParseNRVAssertions(data); err == nil {
			t.Fatalf("%s: tiling violation accepted", name)
		}
	}
	tiled := rawTrainerNRV3(t, []map[string]interface{}{frame("a", 1, 0), frame("b", 2, 80)}, 3*80)
	got, err := ParseNRVAssertions(tiled)
	if err != nil {
		t.Fatalf("tiled file refused: %v", err)
	}
	if len(got) != 2 || got[0].Brackets != 1 || got[1].Brackets != 2 {
		t.Fatalf("tiled frames = %+v", got)
	}
}
