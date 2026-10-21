package evidence

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SpanRec is the metadata stored for one indexed passage. The passage text itself
// lives in the snapshot store, so an index never holds a second copy.
type SpanRec struct {
	ID        string `json:"id"`       // sha256:<hex> of (snapshot, start, end)
	Snapshot  string `json:"snapshot"` // sha256:<hex>
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Tier      Tier   `json:"tier"`
	Tenant    string `json:"tenant,omitempty"` // user tier only
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	License   string `json:"license,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
	Fact      bool   `json:"fact,omitempty"`     // a structured (subject, property, value) fact
	Source    string `json:"source,omitempty"`   // wikipedia, arxiv, ...
	Revision  string `json:"revision,omitempty"` // the pinned page revision / paper version this text is
}

const keyEntrySize = 32 // 16 key | 8 offset | 4 count | 4 pad

// writeSegment writes an immutable segment. keys[i] are the keys of spans[i].
// Layout (all in dir): spans.jsonl, spans.idx (uint64 line offsets), postings.keys
// (sorted fixed-size entries), postings.dat (uint32 ordinals), bloom.bin.
func writeSegment(dir string, spans []SpanRec, keys [][]WeightedKey) error {
	if len(spans) != len(keys) {
		return fmt.Errorf("evidence: %d spans but %d key sets", len(spans), len(keys))
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	var spansBuf, idx bytes.Buffer
	post := map[Key][]uint32{}
	for i, sp := range spans {
		var off [8]byte
		binary.LittleEndian.PutUint64(off[:], uint64(spansBuf.Len()))
		idx.Write(off[:])
		line, err := json.Marshal(sp)
		if err != nil {
			return err
		}
		spansBuf.Write(line)
		spansBuf.WriteByte('\n')
		for _, k := range keys[i] {
			post[k.Key] = append(post[k.Key], uint32(i))
		}
	}
	sorted := make([]Key, 0, len(post))
	for k := range post {
		sorted = append(sorted, k)
	}
	sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i][:], sorted[j][:]) < 0 })
	bloom := NewBloom(len(sorted), 0.01)
	var keyBuf, dat bytes.Buffer
	for _, k := range sorted {
		ords := post[k]
		bloom.Add(k)
		var e [keyEntrySize]byte
		copy(e[:16], k[:])
		binary.LittleEndian.PutUint64(e[16:24], uint64(dat.Len()))
		binary.LittleEndian.PutUint32(e[24:28], uint32(len(ords)))
		keyBuf.Write(e[:])
		for _, o := range ords {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], o)
			dat.Write(b[:])
		}
	}
	for name, data := range map[string][]byte{
		"spans.jsonl": spansBuf.Bytes(), "spans.idx": idx.Bytes(),
		"postings.keys": keyBuf.Bytes(), "postings.dat": dat.Bytes(), "bloom.bin": bloom.Marshal(),
	} {
		if err := writeFileSync(filepath.Join(dir, name), data); err != nil {
			return err
		}
	}
	// The marker is written last: a segment without it is a half-written build and is ignored.
	return writeFileSync(filepath.Join(dir, "COMPLETE"), []byte("fs1\n"))
}

func writeFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// segment is an opened, read-only segment.
type segment struct {
	name  string
	spans *os.File
	offs  []uint64
	keys  *os.File
	nkeys int
	dat   *os.File
	bloom *Bloom
	n     int
}

func openSegment(dir string) (*segment, error) {
	if _, err := os.Stat(filepath.Join(dir, "COMPLETE")); err != nil {
		return nil, fmt.Errorf("segment %s is incomplete", dir)
	}
	s := &segment{name: filepath.Base(dir)}
	idx, err := os.ReadFile(filepath.Join(dir, "spans.idx"))
	if err != nil {
		return nil, err
	}
	s.n = len(idx) / 8
	s.offs = make([]uint64, s.n)
	for i := range s.offs {
		s.offs[i] = binary.LittleEndian.Uint64(idx[i*8:])
	}
	bb, err := os.ReadFile(filepath.Join(dir, "bloom.bin"))
	if err != nil {
		return nil, err
	}
	if s.bloom, err = UnmarshalBloom(bb); err != nil {
		return nil, err
	}
	for name, dst := range map[string]**os.File{"spans.jsonl": &s.spans, "postings.keys": &s.keys, "postings.dat": &s.dat} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			s.close()
			return nil, err
		}
		*dst = f
	}
	st, err := s.keys.Stat()
	if err != nil {
		s.close()
		return nil, err
	}
	s.nkeys = int(st.Size() / keyEntrySize)
	return s, nil
}

func (s *segment) close() {
	for _, f := range []*os.File{s.spans, s.keys, s.dat} {
		if f != nil {
			f.Close()
		}
	}
}

// lookup returns the span ordinals indexed under key. The Bloom filter answers
// almost every miss without touching the disk.
func (s *segment) lookup(key Key) ([]uint32, error) {
	if !s.bloom.MayContain(key) {
		return nil, nil
	}
	lo, hi := 0, s.nkeys
	var e [keyEntrySize]byte
	for lo < hi {
		mid := (lo + hi) / 2
		if _, err := s.keys.ReadAt(e[:], int64(mid)*keyEntrySize); err != nil {
			return nil, err
		}
		switch c := bytes.Compare(e[:16], key[:]); {
		case c == 0:
			off := binary.LittleEndian.Uint64(e[16:24])
			cnt := binary.LittleEndian.Uint32(e[24:28])
			buf := make([]byte, int(cnt)*4)
			if _, err := s.dat.ReadAt(buf, int64(off)); err != nil {
				return nil, err
			}
			out := make([]uint32, cnt)
			for i := range out {
				out[i] = binary.LittleEndian.Uint32(buf[i*4:])
			}
			return out, nil
		case c < 0:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return nil, nil
}

// count is the posting count of key without reading the postings.
func (s *segment) count(key Key) (int, error) {
	if !s.bloom.MayContain(key) {
		return 0, nil
	}
	lo, hi := 0, s.nkeys
	var e [keyEntrySize]byte
	for lo < hi {
		mid := (lo + hi) / 2
		if _, err := s.keys.ReadAt(e[:], int64(mid)*keyEntrySize); err != nil {
			return 0, err
		}
		switch c := bytes.Compare(e[:16], key[:]); {
		case c == 0:
			return int(binary.LittleEndian.Uint32(e[24:28])), nil
		case c < 0:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return 0, nil
}

func (s *segment) span(ord uint32) (SpanRec, error) {
	var sp SpanRec
	if int(ord) >= s.n {
		return sp, fmt.Errorf("span %d out of range", ord)
	}
	end := int64(-1)
	if int(ord)+1 < s.n {
		end = int64(s.offs[ord+1])
	} else if st, err := s.spans.Stat(); err == nil {
		end = st.Size()
	}
	buf := make([]byte, end-int64(s.offs[ord]))
	if _, err := s.spans.ReadAt(buf, int64(s.offs[ord])); err != nil {
		return sp, err
	}
	return sp, json.Unmarshal(bytes.TrimSpace(buf), &sp)
}

// readLines is a helper for the line-oriented registries.
func readLines(path string, f func([]byte) error) error {
	fh, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		if err := f(sc.Bytes()); err != nil {
			return err
		}
	}
	return sc.Err()
}
