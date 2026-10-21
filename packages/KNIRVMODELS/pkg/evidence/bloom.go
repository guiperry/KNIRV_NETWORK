package evidence

import (
	"encoding/binary"
	"errors"
	"math"
)

// Bloom is a Bloom filter over Keys. Keys are already uniform hashes, so the probe
// positions come straight from their bytes (double hashing); nothing is rehashed.
type Bloom struct {
	bits []uint64
	m    uint64
	k    uint8
}

// NewBloom sizes a filter for n keys at roughly the given false-positive rate.
func NewBloom(n int, fp float64) *Bloom {
	if n < 1 {
		n = 1
	}
	if fp <= 0 || fp >= 1 {
		fp = 0.01
	}
	m := uint64(math.Ceil(-float64(n) * math.Log(fp) / (math.Ln2 * math.Ln2)))
	if m < 64 {
		m = 64
	}
	m = (m + 63) / 64 * 64
	k := uint8(math.Round(float64(m) / float64(n) * math.Ln2))
	if k < 1 {
		k = 1
	}
	if k > 16 {
		k = 16
	}
	return &Bloom{bits: make([]uint64, m/64), m: m, k: k}
}

func (b *Bloom) probe(key Key, i uint8) uint64 {
	h1 := binary.LittleEndian.Uint64(key[0:8])
	h2 := binary.LittleEndian.Uint64(key[8:16]) | 1
	return (h1 + uint64(i)*h2) % b.m
}

func (b *Bloom) Add(key Key) {
	for i := uint8(0); i < b.k; i++ {
		p := b.probe(key, i)
		b.bits[p/64] |= 1 << (p % 64)
	}
}

// MayContain is false only if the key was definitely never added.
func (b *Bloom) MayContain(key Key) bool {
	for i := uint8(0); i < b.k; i++ {
		p := b.probe(key, i)
		if b.bits[p/64]&(1<<(p%64)) == 0 {
			return false
		}
	}
	return true
}

// Marshal is: m (8 bytes) | k (1 byte) | bits.
func (b *Bloom) Marshal() []byte {
	out := make([]byte, 9+len(b.bits)*8)
	binary.LittleEndian.PutUint64(out[0:8], b.m)
	out[8] = b.k
	for i, w := range b.bits {
		binary.LittleEndian.PutUint64(out[9+i*8:], w)
	}
	return out
}

func UnmarshalBloom(p []byte) (*Bloom, error) {
	if len(p) < 9 {
		return nil, errors.New("bloom: short file")
	}
	m := binary.LittleEndian.Uint64(p[0:8])
	k := p[8]
	if m == 0 || m%64 != 0 || k == 0 || uint64(len(p)-9) != m/8 {
		return nil, errors.New("bloom: corrupt file")
	}
	b := &Bloom{bits: make([]uint64, m/64), m: m, k: k}
	for i := range b.bits {
		b.bits[i] = binary.LittleEndian.Uint64(p[9+i*8:])
	}
	return b, nil
}
