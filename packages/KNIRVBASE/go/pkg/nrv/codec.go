package nrv

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type Header struct {
	Magic       uint32
	Version     uint32
	TotalLength uint32
}

func EncodeHeader(w io.Writer, h Header) error {
	buf := make([]byte, HeaderSize)
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(buf[4:8], h.Version)
	binary.LittleEndian.PutUint32(buf[8:12], h.TotalLength)
	_, err := w.Write(buf)
	return err
}

func DecodeHeader(r io.Reader) (Header, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Header{}, fmt.Errorf("nrv: read header: %w", err)
	}
	h := Header{
		Magic:       binary.LittleEndian.Uint32(buf[0:4]),
		Version:     binary.LittleEndian.Uint32(buf[4:8]),
		TotalLength: binary.LittleEndian.Uint32(buf[8:12]),
	}
	if h.Magic != Magic {
		return Header{}, fmt.Errorf("nrv: invalid magic bytes: got 0x%X, want 0x%X", h.Magic, Magic)
	}
	return h, nil
}

type ModalityMap = map[ModalityType]ModalityIndex

func EncodeBracket(b *Bracket) [BracketSize]byte {
	var buf [BracketSize]byte
	// 0x00-0x1F: LSH Projections (32 bytes, Slots 0-3: Semantic Compass)
	copy(buf[0:32], b.Projections[:])
	// 0x20-0x23: SubSecondUS (4 bytes)
	binary.LittleEndian.PutUint32(buf[32:36], b.SubSecondUS)
	// 0x24: POSTag (Slot 4)
	buf[36] = b.POSTag
	// 0x25: Tense (Slot 4)
	buf[37] = b.Tense
	// 0x26: Plurality (Slot 4)
	buf[38] = b.Plurality
	// 0x27: DepHead (Slot 5)
	buf[39] = b.DepHead
	// 0x28: IntentFlags (Slot 9)
	buf[40] = b.IntentFlags
	// 0x29-0x2A: DomainSig (Slot 10)
	binary.LittleEndian.PutUint16(buf[41:43], b.DomainSig)
	// 0x2B-0x2E: GoldenSeed (4 bytes)
	binary.LittleEndian.PutUint32(buf[43:47], b.GoldenSeed)
	// 0x2F-0x3C: Memory Zone (14 bytes, Slots 6-8: Recursive XOR summary)
	copy(buf[47:61], b.Memory[:])
	// 0x3D-0x40: LSHSalt (Slot 11: Temporal Lock)
	binary.LittleEndian.PutUint32(buf[61:65], b.LSHSalt)
	// 0x41-0x4F: Reserved (15 bytes)
	return buf
}

func DecodeBracket(buf [BracketSize]byte) Bracket {
	var b Bracket
	// 0x00-0x1F: LSH Projections (32 bytes)
	copy(b.Projections[:], buf[0:32])
	// 0x20-0x23: SubSecondUS
	b.SubSecondUS = binary.LittleEndian.Uint32(buf[32:36])
	// 0x24: POSTag
	b.POSTag = buf[36]
	// 0x25: Tense
	b.Tense = buf[37]
	// 0x26: Plurality
	b.Plurality = buf[38]
	// 0x27: DepHead
	b.DepHead = buf[39]
	// 0x28: IntentFlags
	b.IntentFlags = buf[40]
	// 0x29-0x2A: DomainSig
	b.DomainSig = binary.LittleEndian.Uint16(buf[41:43])
	// 0x2B-0x2E: GoldenSeed
	b.GoldenSeed = binary.LittleEndian.Uint32(buf[43:47])
	// 0x2F-0x3C: Memory Zone
	copy(b.Memory[:], buf[47:61])
	// 0x3D-0x40: LSHSalt
	b.LSHSalt = binary.LittleEndian.Uint32(buf[61:65])
	return b
}

func XORProjections(current, anchor [32]byte) [32]byte {
	var diff [32]byte
	for i := range diff {
		diff[i] = current[i] ^ anchor[i]
	}
	return diff
}

func ApplyProjectionDelta(delta, anchor [32]byte) [32]byte {
	return XORProjections(delta, anchor)
}

func EncodeFrame(f *Frame) ([]byte, ModalityMap) {
	proofAligned := Align8(len(f.Proof))
	total := 48 + 32 + 16 + proofAligned
	buf := make([]byte, total)

	for i, v := range f.Vector {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}

	copy(buf[48:], f.Seed[:])

	binary.LittleEndian.PutUint32(buf[80:], math.Float32bits(f.Thermo.TempCelsius))
	binary.LittleEndian.PutUint32(buf[84:], math.Float32bits(f.Thermo.VoltageV))
	binary.LittleEndian.PutUint32(buf[88:], math.Float32bits(f.Thermo.FreqMHz))
	binary.LittleEndian.PutUint32(buf[92:], math.Float32bits(f.Thermo.FanRPM))

	copy(buf[96:], f.Proof)

	modalities := ModalityMap{
		ModalityVector: ModalityIndex{Offset: 0, Length: 48},
		ModalitySeed:   ModalityIndex{Offset: 48, Length: 32},
		ModalityThermo: ModalityIndex{Offset: 80, Length: 16},
		ModalityProof:  ModalityIndex{Offset: 96, Length: len(f.Proof)},
	}
	return buf, modalities
}

type Frame struct {
	ID     string
	Vector [12]float32
	Seed   [32]byte
	Thermo ThermoData
	Proof  []byte
}
