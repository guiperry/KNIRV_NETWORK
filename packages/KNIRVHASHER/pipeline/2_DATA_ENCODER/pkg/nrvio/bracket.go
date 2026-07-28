// Package nrvio is the encoder-owned Phase 2 wire format implementation.
// Keep this in sync with KNIRVBASE's public .nrv format when that format
// evolves; the encoder must not import KNIRVBASE/internal storage packages.
package nrvio

import (
	"encoding/binary"
	"fmt"
)

const BracketSize = 80

type Bracket struct {
	Projections                       [32]byte
	SubSecondUS                       uint32
	POSTag, Tense, Plurality, DepHead uint8
	IntentFlags                       uint8
	DomainSig                         uint16
	GoldenSeed                        uint32
	Memory                            [14]byte
	LSHSalt                           uint32
	Reserved                          [15]byte
}

func EncodeBracket(b Bracket) [BracketSize]byte {
	var out [BracketSize]byte
	copy(out[0:32], b.Projections[:])
	binary.LittleEndian.PutUint32(out[32:36], b.SubSecondUS)
	out[36], out[37], out[38], out[39] = b.POSTag, b.Tense, b.Plurality, b.DepHead
	out[40] = b.IntentFlags
	binary.LittleEndian.PutUint16(out[41:43], b.DomainSig)
	binary.LittleEndian.PutUint32(out[43:47], b.GoldenSeed)
	copy(out[47:61], b.Memory[:])
	binary.LittleEndian.PutUint32(out[61:65], b.LSHSalt)
	copy(out[65:80], b.Reserved[:])
	return out
}

func DecodeBracket(data [BracketSize]byte) Bracket {
	return Bracket{Projections: func() [32]byte { var x [32]byte; copy(x[:], data[:32]); return x }(), SubSecondUS: binary.LittleEndian.Uint32(data[32:36]), POSTag: data[36], Tense: data[37], Plurality: data[38], DepHead: data[39], IntentFlags: data[40], DomainSig: binary.LittleEndian.Uint16(data[41:43]), GoldenSeed: binary.LittleEndian.Uint32(data[43:47]), Memory: func() [14]byte { var x [14]byte; copy(x[:], data[47:61]); return x }(), LSHSalt: binary.LittleEndian.Uint32(data[61:65]), Reserved: func() [15]byte { var x [15]byte; copy(x[:], data[65:]); return x }()}
}

func Validate(data []byte) error {
	if len(data) != BracketSize {
		return fmt.Errorf("bracket is %d bytes, want %d", len(data), BracketSize)
	}
	return nil
}
