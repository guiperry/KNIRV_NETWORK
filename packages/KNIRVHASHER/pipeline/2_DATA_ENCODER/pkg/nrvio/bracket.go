// Package nrvio implements the public KNIRVBASE NRV wire layout locally.
// The encoder is a separate Go module, so this byte layout is a deliberate
// compatibility contract with KNIRVBASE/go/pkg/nrv.
package nrvio

import (
	"encoding/binary"
	"fmt"
)

const BracketSize = 80

// Bracket's canonical offsets are projections [0:32], timestamp [32:36],
// syntactic [36], dependency head [37], intent [38], domain [39:41], golden
// seed [41:45], memory [45:59], LSH salt [59:63], reserved [63:80].
type Bracket struct {
	Projections [32]byte
	SubSecondUS uint32
	Syntactic   uint8
	DepHead     int8
	IntentFlags uint8
	DomainSig   uint16
	GoldenSeed  uint32
	Memory      [14]byte
	LSHSalt     uint32
	Reserved    [17]byte
}

// PackSyntactic matches KNIRVBASE: POS in four low bits, tense in the next
// two, and plurality in the high two.
func PackSyntactic(posTag, tense, plurality uint8) uint8 {
	return (posTag & 0x0f) | ((tense & 0x03) << 4) | ((plurality & 0x03) << 6)
}

func EncodeBracket(b Bracket) [BracketSize]byte {
	var out [BracketSize]byte
	copy(out[0:32], b.Projections[:])
	binary.LittleEndian.PutUint32(out[32:36], b.SubSecondUS)
	out[36] = b.Syntactic
	out[37] = byte(b.DepHead)
	out[38] = b.IntentFlags
	binary.LittleEndian.PutUint16(out[39:41], b.DomainSig)
	binary.LittleEndian.PutUint32(out[41:45], b.GoldenSeed)
	copy(out[45:59], b.Memory[:])
	binary.LittleEndian.PutUint32(out[59:63], b.LSHSalt)
	copy(out[63:80], b.Reserved[:])
	return out
}

func DecodeBracket(data [BracketSize]byte) Bracket {
	var b Bracket
	copy(b.Projections[:], data[0:32])
	b.SubSecondUS = binary.LittleEndian.Uint32(data[32:36])
	b.Syntactic = data[36]
	b.DepHead = int8(data[37])
	b.IntentFlags = data[38]
	b.DomainSig = binary.LittleEndian.Uint16(data[39:41])
	b.GoldenSeed = binary.LittleEndian.Uint32(data[41:45])
	copy(b.Memory[:], data[45:59])
	b.LSHSalt = binary.LittleEndian.Uint32(data[59:63])
	copy(b.Reserved[:], data[63:80])
	return b
}

func Validate(data []byte) error {
	if len(data) != BracketSize {
		return fmt.Errorf("bracket is %d bytes, want %d", len(data), BracketSize)
	}
	return nil
}
