package evidence

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// A segment holds three open files, and every query probes every segment's Bloom
// filter, so segment count has to stay bounded while batches keep arriving.
const compactGroup = 16 // how many of the smallest segments one compaction merges

// maxSegments is a variable only so tests can exercise compaction without ingesting
// dozens of batches (each ingest fsyncs its files).
var maxSegments = 48

type cursor struct {
	seg  *segment
	r    *bufio.Reader
	base uint32
	cur  [keyEntrySize]byte
	ok   bool
}

func (c *cursor) next() error {
	if _, err := io.ReadFull(c.r, c.cur[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			c.ok = false
			return nil
		}
		return err
	}
	c.ok = true
	return nil
}

// mergeSegments writes one segment equivalent to srcs, in order: spans are
// concatenated with their ordinals shifted, and the sorted key files are merged.
func mergeSegments(dst string, srcs []*segment, meta segMeta) error {
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}
	mb, _ := json.Marshal(meta)
	if err := writeFileSync(filepath.Join(dst, "meta.json"), mb); err != nil {
		return err
	}
	sp, err := os.Create(filepath.Join(dst, "spans.jsonl"))
	if err != nil {
		return err
	}
	var idx bytes.Buffer
	var base uint64
	var ordBase []uint32
	var total uint32
	var keyUpper int
	for _, s := range srcs {
		ordBase = append(ordBase, total)
		for _, o := range s.offs {
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], base+o)
			idx.Write(b[:])
		}
		n, err := io.Copy(sp, io.NewSectionReader(s.spans, 0, 1<<62))
		if err != nil {
			sp.Close()
			return err
		}
		base += uint64(n)
		total += uint32(s.n)
		keyUpper += s.nkeys
	}
	if err := sp.Sync(); err != nil {
		sp.Close()
		return err
	}
	sp.Close()
	if err := writeFileSync(filepath.Join(dst, "spans.idx"), idx.Bytes()); err != nil {
		return err
	}

	kf, err := os.Create(filepath.Join(dst, "postings.keys"))
	if err != nil {
		return err
	}
	df, err := os.Create(filepath.Join(dst, "postings.dat"))
	if err != nil {
		kf.Close()
		return err
	}
	kw, dw := bufio.NewWriter(kf), bufio.NewWriter(df)
	bloom := NewBloom(keyUpper, 0.01)
	cs := make([]*cursor, len(srcs))
	for i, s := range srcs {
		cs[i] = &cursor{seg: s, r: bufio.NewReaderSize(io.NewSectionReader(s.keys, 0, int64(s.nkeys)*keyEntrySize), 1<<16), base: ordBase[i]}
		if err := cs[i].next(); err != nil {
			return err
		}
	}
	var off uint64
	for {
		var min []byte
		for _, c := range cs {
			if c.ok && (min == nil || bytes.Compare(c.cur[:16], min) < 0) {
				min = append(min[:0], c.cur[:16]...)
			}
		}
		if min == nil {
			break
		}
		var key Key
		copy(key[:], min)
		var count uint32
		for _, c := range cs {
			if !c.ok || !bytes.Equal(c.cur[:16], min) {
				continue
			}
			o := binary.LittleEndian.Uint64(c.cur[16:24])
			n := binary.LittleEndian.Uint32(c.cur[24:28])
			buf := make([]byte, int(n)*4)
			if _, err := c.seg.dat.ReadAt(buf, int64(o)); err != nil {
				return err
			}
			for i := 0; i < int(n); i++ {
				v := binary.LittleEndian.Uint32(buf[i*4:]) + c.base
				var b [4]byte
				binary.LittleEndian.PutUint32(b[:], v)
				dw.Write(b[:])
			}
			count += n
			if err := c.next(); err != nil {
				return err
			}
		}
		var e [keyEntrySize]byte
		copy(e[:16], key[:])
		binary.LittleEndian.PutUint64(e[16:24], off)
		binary.LittleEndian.PutUint32(e[24:28], count)
		kw.Write(e[:])
		bloom.Add(key)
		off += uint64(count) * 4
	}
	for _, w := range []*bufio.Writer{kw, dw} {
		if err := w.Flush(); err != nil {
			return err
		}
	}
	for _, f := range []*os.File{kf, df} {
		if err := f.Sync(); err != nil {
			return err
		}
		f.Close()
	}
	if err := writeFileSync(filepath.Join(dst, "bloom.bin"), bloom.Marshal()); err != nil {
		return err
	}
	return writeFileSync(filepath.Join(dst, "COMPLETE"), []byte("fs1\n"))
}

// Compact merges the group smallest segments into one and removes them. The
// content, and therefore Version, does not change. It holds the write lock, so
// searches wait for it.
func (ix *Index) Compact(group int) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.compactLocked(group)
}

func (ix *Index) compactLocked(group int) error {
	if group < 2 || len(ix.segs) < 2 {
		return nil
	}
	if group > len(ix.segs) {
		group = len(ix.segs)
	}
	order := make([]int, len(ix.segs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return ix.segs[order[a]].n < ix.segs[order[b]].n })
	pick := append([]int(nil), order[:group]...)
	sort.Ints(pick) // keep arrival order inside the merged segment
	var srcs []*segment
	merged := segMeta{Tiers: map[Tier]int{}, BatchID: "merged"}
	for _, i := range pick {
		srcs = append(srcs, ix.segs[i])
		m := ix.metas[i]
		merged.Spans += m.Spans
		for t, n := range m.Tiers {
			merged.Tiers[t] += n
		}
		merged.Roots = append(merged.Roots, m.roots()...)
		if m.Created > merged.Created {
			merged.Created = m.Created
		}
	}
	merged.Root = ""
	name := ix.nextSegName()
	dir := filepath.Join(ix.dir, "segments", name)
	if err := mergeSegments(dir, srcs, merged); err != nil {
		os.RemoveAll(dir)
		return err
	}
	ns, err := openSegment(dir)
	if err != nil {
		os.RemoveAll(dir)
		return err
	}
	drop := map[int]bool{}
	for _, i := range pick {
		drop[i] = true
	}
	var segs []*segment
	var metas []segMeta
	for i := range ix.segs {
		if drop[i] {
			ix.segs[i].close()
			os.RemoveAll(filepath.Join(ix.dir, "segments", ix.segs[i].name))
			continue
		}
		segs, metas = append(segs, ix.segs[i]), append(metas, ix.metas[i])
	}
	ix.segs, ix.metas = append(segs, ns), append(metas, merged)
	return nil
}
