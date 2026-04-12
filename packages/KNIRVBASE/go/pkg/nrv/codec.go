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

// Bracket wire layout — 80-byte ASIC header (GoldenSeed and LSHSalt are little-endian):
//
//	[ Offset ] [ Bytes ]            [ Slot Mapping ]
//	0x00-0x1F: [ 32 Bytes ] ------> Slots 0-3 (Semantic Compass / LSH Projections)
//	0x20-0x23: [ 04 Bytes ] ------> (Metadata: SubSecondUS Ticker)
//	0x24-0x24: [ 01 Byte  ] ------> Slot 4 (Bit-Packed: POSTag, Tense, Plurality)
//	0x25-0x26: [ 02 Bytes ] ------> (Metadata: Reserved for future)
//	0x27-0x27: [ 01 Byte  ] ------> Slot 5 (Structural Logic: DepHead)
//	0x28-0x28: [ 01 Byte  ] ------> Slot 9 (Identity: IntentFlags)
//	0x29-0x2A: [ 02 Bytes ] ------> Slot 10 (Mode: DomainSig)
//	0x2B-0x2E: [ 04 Bytes ] ------> (Nonce Target: GoldenSeed)
//	0x2F-0x3C: [ 14 Bytes ] ------> Slots 6-8 (Recursive Context: Memory)
//	0x3D-0x40: [ 04 Bytes ] ------> Slot 11 (LSH Salt: `(PosIndex << 16) | TemporalSalt` — Contextual Anchor)
//	0x41-0x4F: [ 15 Bytes ] ------> (Metadata: Z3 Validation Trace / Reserved)
func EncodeBracket(b *Bracket) [BracketSize]byte {
	var buf [BracketSize]byte
	// 0x00-0x1F: Projections (Slots 0-3)
	copy(buf[0:32], b.Projections[:])
	// 0x20-0x23: SubSecondUS
	binary.LittleEndian.PutUint32(buf[32:36], b.SubSecondUS)
	// 0x24-0x26: POSTag, Tense, Plurality (Slot 4 - unpacked for clarity)
	buf[36] = b.POSTag
	buf[37] = b.Tense
	buf[38] = b.Plurality
	// 0x27: DepHead (Slot 5)
	buf[39] = byte(b.DepHead)
	// 0x28: IntentFlags (Slot 9)
	buf[40] = b.IntentFlags
	// 0x29-0x2A: DomainSig (Slot 10)
	binary.LittleEndian.PutUint16(buf[41:43], b.DomainSig)
	// 0x2B-0x2E: GoldenSeed (Nonce Target)
	binary.LittleEndian.PutUint32(buf[43:47], b.GoldenSeed)
	// 0x2F-0x3C: Memory (Slots 6-8)
	copy(buf[47:61], b.Memory[:])
	// 0x3D-0x40: LSHSalt (Slot 11: Temporal Salt)
	binary.LittleEndian.PutUint32(buf[61:65], b.LSHSalt)
	// 0x41-0x4F: Reserved
	copy(buf[65:80], b.Reserved[:])
	return buf
}

func DecodeBracket(buf [BracketSize]byte) Bracket {
	var b Bracket
	// 0x00-0x1F: Projections
	copy(b.Projections[:], buf[0:32])
	// 0x20-0x23: SubSecondUS
	b.SubSecondUS = binary.LittleEndian.Uint32(buf[32:36])
	// 0x24-0x26: POSTag, Tense, Plurality
	b.POSTag = buf[36]
	b.Tense = buf[37]
	b.Plurality = buf[38]
	// 0x27: DepHead
	b.DepHead = int8(buf[39])
	// 0x28: IntentFlags
	b.IntentFlags = buf[40]
	// 0x29-0x2A: DomainSig
	b.DomainSig = binary.LittleEndian.Uint16(buf[41:43])
	// 0x2B-0x2E: GoldenSeed
	b.GoldenSeed = binary.LittleEndian.Uint32(buf[43:47])
	// 0x2F-0x3C: Memory
	copy(b.Memory[:], buf[47:61])
	// 0x3D-0x40: LSHSalt
	b.LSHSalt = binary.LittleEndian.Uint32(buf[61:65])
	// 0x41-0x4F: Reserved
	copy(b.Reserved[:], buf[65:80])
	return b
}

// PackSyntacticByte packs Slot 4 per NRV_Master_Specification §3.2 (4-bit POS, 2-bit tense, 2-bit plurality).
func PackSyntacticByte(post, tense, plurality uint8) uint8 {
	return (post & 0x0F) | ((tense & 3) << 4) | ((plurality & 3) << 6)
}

// UnpackSyntacticByte expands the single Slot-4 bitmask byte.
func UnpackSyntacticByte(b uint8) (post, tense, plurality uint8) {
	post = b & 0x0F
	tense = (b >> 4) & 3
	plurality = (b >> 6) & 3
	return post, tense, plurality
}

// EncodeSyntactic packs syntactic fields into a uint32 (wire-adjacent layout: POSTag, Tense, Plurality, DepHead).
func EncodeSyntactic(sp SyntacticProfile) uint32 {
	var val uint32
	val |= uint32(sp.POSTag)
	val |= uint32(sp.Tense) << 8
	val |= uint32(sp.Plurality) << 16
	val |= uint32(uint8(sp.DepHead)) << 24
	return val
}

// DecodeSyntactic unpacks EncodeSyntactic.
func DecodeSyntactic(val uint32) SyntacticProfile {
	return SyntacticProfile{
		POSTag:    uint8(val),
		Tense:     uint8(val >> 8),
		Plurality: uint8(val >> 16),
		DepHead:   int8(val >> 24),
	}
}

// EncodeIntentDomain packs intent (low nibble) and domain signature.
func EncodeIntentDomain(id IntentDomain) uint32 {
	return uint32(id.IntentFlags&0xFF) | (uint32(id.DomainSig) << 16)
}

// DecodeIntentDomain unpacks EncodeIntentDomain.
func DecodeIntentDomain(val uint32) IntentDomain {
	return IntentDomain{
		IntentFlags: uint8(val & 0xFF),
		DomainSig:   uint16(val >> 16),
	}
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
