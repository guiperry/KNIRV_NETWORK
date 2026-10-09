// Package slotpack packs one encoded window into the 12-slot frame and maps
// the frame onto an 80-byte NRV bracket. It has no dependencies beyond the
// standard library and nrvio so that other modules (backend_server's arena
// dataset endpoints) can encode exactly as the data-encoder CLI does.
package slotpack

import (
	"encoding/binary"
	"strings"

	"data-encoder/pkg/nrvio"
)

// PackFrame creates the 12-slot bitmask tensor according to the HASHER-MAPPER
// spec. domain, when non-nil, sets Slot 10; otherwise it is detected from the
// instruction and input keywords.
func PackFrame(
	signalIndices []int,
	domain *uint32,
	embedding []float32,
	pos uint8,
	tense uint8,
	depHash uint32,
	lastHeaders []uint32,
	tokenPos uint16,
	instruction string,
	input string,
) [12]uint32 {
	var slots [12]uint32

	// Zone 1: Identity (Slots 0–3)
	for i := 0; i < 4 && i < len(signalIndices); i++ {
		dimIdx := signalIndices[i]
		slots[i] = QuantizeFloatToUint32(embedding[dimIdx])
	}

	// Slot 4: Grammar (POS Tag ID | Tense/Mood ID)
	slots[4] = uint32(pos) | (uint32(tense) << 8)

	// Slot 5: Syntax (Dependency Link Hash)
	slots[5] = depHash

	// Slots 6–8: Memory (Recursive Summary — last headers XORed)
	for i := 0; i < 3 && i < len(lastHeaders); i++ {
		slots[6+i] = lastHeaders[i]
	}

	// Slot 9: Intent flags
	if IsQuestion(instruction) {
		slots[9] |= 0x1
	}
	if IsCode(input) || IsCode(instruction) {
		slots[9] |= 0x2
	}

	// Slot 10: Domain Signature — schema-driven when available, keyword fallback otherwise
	if domain != nil {
		slots[10] = *domain
	} else {
		slots[10] = DetectDomain(instruction, input)
	}

	// Slot 11: Lock (Token Position Index)
	slots[11] = uint32(tokenPos)

	return slots
}

func QuantizeFloatToUint32(val float32) uint32 {
	if val < -1.0 {
		val = -1.0
	}
	if val > 1.0 {
		val = 1.0
	}
	scaled := (float64(val) + 1.0) / 2.0 * 4294967295.0
	return uint32(scaled + 0.5)
}

func IsQuestion(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "?") ||
		strings.HasPrefix(s, "what") ||
		strings.HasPrefix(s, "how") ||
		strings.HasPrefix(s, "why") ||
		strings.HasPrefix(s, "can you")
}

func IsCode(s string) bool {
	s = strings.ToLower(s)
	for _, ind := range []string{"func ", "var ", "import ", "{", "}", "[]", "const ", "def ", "class "} {
		if strings.Contains(s, ind) {
			return true
		}
	}
	return false
}

// DetectDomain is the keyword-based domain fallback used when no schema is loaded.
func DetectDomain(instr, input string) uint32 {
	instr = strings.ToLower(instr)
	input = strings.ToLower(input)

	for _, ind := range []string{
		"calculate", "math", "equation", "solve", "sum", "multiply", "divide",
		"algebra", "geometry", "theorem", "calculus", "topology", "number theory",
		"+", "-", "*", "/",
	} {
		if strings.Contains(instr, ind) || strings.Contains(input, ind) {
			return 0x2000 // DOMAIN_MATH
		}
	}

	if IsCode(input) || containsAny(instr, "code", "program", "function", "compiler", "software", "source code") {
		return 0x3000 // DOMAIN_CODE
	}

	if containsAny(instr,
		"arxiv", "paper", "research", "neural", "learning", "model",
		"quantum", "physics", "biology", "chemistry", "language",
	) {
		return 0x4000 // DOMAIN_ACADEMIC
	}

	return 0x1000 // DOMAIN_PROSE
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}

// SlotsToProjections converts Slots 0-3 to a 32-byte projection array by
// writing each byte of each slot twice: [a b c d] -> [a a b b c c d d].
func SlotsToProjections(slots []uint32) []byte {
	if len(slots) < 4 {
		padded := make([]uint32, 4)
		copy(padded, slots)
		slots = padded
	}
	result := make([]byte, 32)
	for i := 0; i < 4; i++ {
		val := slots[i]
		for j := 0; j < 4; j++ {
			byteVal := byte((val >> (8 * j)) & 0xFF)
			pos := i*8 + j*2
			result[pos] = byteVal
			result[pos+1] = byteVal
		}
	}
	return result
}

// Bracket maps a packed frame onto the NRV bracket layout, as the encoder's
// .nrv artifacts store it: GoldenSeed carries the target token id, LSHSalt
// Slot 11 and SubSecondUS the window start.
func Bracket(slots [12]uint32, targetTokenID int32, windowStart int32) nrvio.Bracket {
	var b nrvio.Bracket
	copy(b.Projections[:], SlotsToProjections(slots[:4]))
	b.Syntactic = nrvio.PackSyntactic(uint8(slots[4]), 0, 0)
	b.DepHead = int8(slots[5])
	b.IntentFlags = uint8(slots[9])
	b.DomainSig = uint16(slots[10])
	b.GoldenSeed = uint32(targetTokenID)
	b.LSHSalt = slots[11]
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint32(b.Memory[i*4:], slots[6+i])
	}
	b.SubSecondUS = uint32(windowStart)
	return b
}
