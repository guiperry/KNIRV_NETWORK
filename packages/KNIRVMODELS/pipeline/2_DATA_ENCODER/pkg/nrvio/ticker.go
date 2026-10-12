package nrvio

import (
	"fmt"
	"sync"
	"time"
)

// FrameTicker assembles one-second frames in the encoder-owned wire format.
// The first bracket is an I bracket; later brackets carry an XOR delta in the
// index while the original bracket remains available to the reader.
type FrameTicker struct {
	writer   *Writer
	period   time.Duration
	mu       sync.Mutex
	current  *FrameEntry
	previous [BracketSize]byte
	closed   bool
}

func NewFrameTicker(w *Writer, period time.Duration) *FrameTicker {
	if period <= 0 {
		period = time.Second
	}
	return &FrameTicker{writer: w, period: period}
}

func (t *FrameTicker) Append(b Bracket) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.writer == nil {
		return fmt.Errorf("frame ticker is closed")
	}
	bucketSize := t.period.Microseconds()
	if bucketSize <= 0 {
		bucketSize = int64(time.Second / time.Microsecond)
	}
	bucket := int64(b.SubSecondUS) / bucketSize
	id := fmt.Sprintf("frame-%d", bucket)
	if t.current == nil || t.current.ID != id {
		if err := t.flushLocked(); err != nil {
			return err
		}
		t.current = &FrameEntry{ID: id, TimestampUnix: time.Now().Unix(), Offset: int64(len(t.writer.brackets)), Linguistic: LinguisticMetadata{Domain: domainForBracket(b)}}
	}
	raw := EncodeBracket(b)
	delta := make([]byte, BracketSize)
	for i := range raw {
		delta[i] = raw[i] ^ t.previous[i]
	}
	typeName := "P"
	if len(t.current.BracketIndex) == 0 {
		typeName = "I"
	}
	t.current.BracketIndex = append(t.current.BracketIndex, BracketIndexEntry{Index: len(t.writer.brackets) - int(t.current.Offset), Type: typeName, XOR: delta})
	t.current.Count++
	t.writer.Append(b)
	t.previous = raw
	return nil
}

func (t *FrameTicker) Flush() error { t.mu.Lock(); defer t.mu.Unlock(); return t.flushLocked() }
func (t *FrameTicker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	return t.flushLocked()
}
func (t *FrameTicker) flushLocked() error {
	if t.current != nil {
		t.writer.frames = append(t.writer.frames, *t.current)
		t.current = nil
	}
	return nil
}

func domainForBracket(b Bracket) string {
	switch b.DomainSig & 0xf000 {
	case 0x4000:
		return "academic"
	case 0x3000:
		return "code"
	case 0x2000:
		return "math"
	default:
		return "prose"
	}
}
