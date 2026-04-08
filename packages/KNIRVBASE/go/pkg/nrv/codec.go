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
	binary.LittleEndian.PutUint32(buf[0:4], b.LSHSalt)
	copy(buf[4:68], b.Projections[:])
	binary.LittleEndian.PutUint32(buf[68:72], b.SubSecondUS)
	binary.LittleEndian.PutUint32(buf[72:76], b.ASICLoops)
	binary.LittleEndian.PutUint32(buf[76:80], b.GoldenSeed)
	return buf
}

func DecodeBracket(buf [BracketSize]byte) Bracket {
	var b Bracket
	b.LSHSalt = binary.LittleEndian.Uint32(buf[0:4])
	copy(b.Projections[:], buf[4:68])
	b.SubSecondUS = binary.LittleEndian.Uint32(buf[68:72])
	b.ASICLoops = binary.LittleEndian.Uint32(buf[72:76])
	b.GoldenSeed = binary.LittleEndian.Uint32(buf[76:80])
	return b
}

func XORProjections(current, anchor [64]byte) [64]byte {
	var diff [64]byte
	for i := range diff {
		diff[i] = current[i] ^ anchor[i]
	}
	return diff
}

func ApplyProjectionDelta(delta, anchor [64]byte) [64]byte {
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
