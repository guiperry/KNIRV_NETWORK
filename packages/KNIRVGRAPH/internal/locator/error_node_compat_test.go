package locator

import (
	"strings"
	"testing"
)

// memErrorKV is an in-memory ErrorNodeKV for exercising the persisted-row reader.
type memErrorKV struct{ rows map[string][]byte }

func (m *memErrorKV) Put(key, value []byte) error {
	m.rows[string(key)] = value
	return nil
}

func (m *memErrorKV) Get(key []byte) ([]byte, error) {
	return m.rows[string(key)], nil
}

func (m *memErrorKV) ScanPrefix(prefix []byte) (map[string][]byte, error) {
	out := map[string][]byte{}
	for k, v := range m.rows {
		if strings.HasPrefix(k, string(prefix)) {
			out[k] = v
		}
	}
	return out, nil
}

// Rows persisted under the old nrvSource key must still load, with the value
// read into AnnouncementRef.
func TestPersistedRowWithOldNrvSourceKeyLoadsAsAnnouncementRef(t *testing.T) {
	kv := &memErrorKV{rows: map[string][]byte{
		errorNodeKeyPrefix + "e-old":   []byte(`{"id":"e-old","nrvSource":"src-1","errorType":"leak","severity":3}`),
		errorNodeKeyPrefix + "e-snake": []byte(`{"id":"e-snake","nrv_source":"src-2"}`),
		errorNodeKeyPrefix + "e-new":   []byte(`{"id":"e-new","announcement_ref":"ref-3"}`),
		errorNodeKeyPrefix + "e-both":  []byte(`{"id":"e-both","nrvSource":"old","announcementRef":"new"}`),
	}}
	sys := NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(kv); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"e-old": "src-1", "e-snake": "src-2", "e-new": "ref-3", "e-both": "new"}
	nodes := sys.GetAllErrorNodes()
	if len(nodes) != len(want) {
		t.Fatalf("loaded %d nodes, want %d", len(nodes), len(want))
	}
	for _, n := range nodes {
		if got := n.GetAnnouncementRef(); got != want[n.Id] {
			t.Errorf("%s: AnnouncementRef = %q, want %q", n.Id, got, want[n.Id])
		}
	}
	if raw := kv.rows[errorNodeKeyPrefix+"e-old"]; !strings.Contains(string(raw), "nrvSource") {
		// The alias reads old rows; it never rewrites them (the U3 migration does).
		t.Errorf("reading must not rewrite the stored row")
	}
}

func TestUnreadablePersistedRowIsSkipped(t *testing.T) {
	kv := &memErrorKV{rows: map[string][]byte{
		errorNodeKeyPrefix + "bad":     []byte(`not json`),
		errorNodeKeyPrefix + "unknown": []byte(`{"id":"u","someFutureField":1}`),
	}}
	sys := NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(kv); err != nil {
		t.Fatal(err)
	}
	if n := len(sys.GetAllErrorNodes()); n != 0 {
		t.Fatalf("loaded %d nodes from unreadable rows, want 0 (strict unmarshal kept)", n)
	}
}
