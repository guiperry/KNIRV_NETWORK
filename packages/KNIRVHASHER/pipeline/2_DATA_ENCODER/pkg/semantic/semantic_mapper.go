package semantic

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"math/rand"
	"strings"

	"data-encoder/pkg/nrvio"
)

type RecordMetadata struct {
	DatasetID                         string
	ChunkID                           int32
	Text                              string
	POSTag, Tense, Plurality, DepHead uint8
	SubSecondUS                       uint32
}
type SemanticMapper struct{ matrix [16][768]float32 }

func NewSemanticMapper(seed int64) *SemanticMapper {
	m := &SemanticMapper{}
	r := rand.New(rand.NewSource(seed))
	for i := range m.matrix {
		for j := range m.matrix[i] {
			m.matrix[i][j] = float32(r.NormFloat64())
		}
	}
	return m
}

func (m *SemanticMapper) MapToBracket(embedding []float32, meta RecordMetadata) nrvio.Bracket {
	b := nrvio.Bracket{SubSecondUS: meta.SubSecondUS, POSTag: meta.POSTag, Tense: meta.Tense, Plurality: meta.Plurality, DepHead: meta.DepHead, LSHSalt: deriveSalt(meta.DatasetID, meta.ChunkID), IntentFlags: detectIntent(meta.Text), DomainSig: classifyDomain(meta.Text)}
	var values [16]float32
	for i := range values {
		for j := 0; j < len(embedding) && j < 768; j++ {
			values[i] += embedding[j] * m.matrix[i][j]
		}
	}
	var norm float32
	for _, v := range values {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range values {
			values[i] /= norm
		}
	}
	for i, v := range values {
		binary.LittleEndian.PutUint16(b.Projections[i*2:], float32ToHalf(v))
	}
	return b
}

func deriveSalt(dataset string, chunk int32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(dataset))
	var x [4]byte
	binary.LittleEndian.PutUint32(x[:], uint32(chunk))
	h.Write(x[:])
	return h.Sum32()
}
func detectIntent(text string) uint8 {
	l := strings.ToLower(strings.TrimSpace(text))
	var f uint8
	if strings.Contains(l, "?") || strings.HasPrefix(l, "what ") {
		f |= 1
	}
	if strings.HasPrefix(l, "write ") || strings.HasPrefix(l, "create ") {
		f |= 2
	}
	if strings.Contains(l, "func ") || strings.Contains(l, "def ") {
		f |= 4
	}
	return f
}
func classifyDomain(text string) uint16 {
	l := strings.ToLower(text)
	if strings.ContainsAny(l, "+-*/=") || strings.Contains(l, "equation") || strings.Contains(l, "calculate") {
		return 0x2000
	}
	if strings.Contains(l, "func ") || strings.Contains(l, "def ") || strings.Contains(l, "import ") {
		return 0x3000
	}
	return 0x1000
}

func float32ToHalf(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := uint16((bits >> 16) & 0x8000)
	exp := int((bits>>23)&0xff) - 127 + 15
	mant := uint16((bits >> 13) & 0x3ff)
	if exp <= 0 {
		return sign
	}
	if exp >= 31 {
		return sign | 0x7c00
	}
	return sign | uint16(exp<<10) | mant
}
