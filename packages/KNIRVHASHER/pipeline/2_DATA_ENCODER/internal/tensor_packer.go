package internal

import (
	"fmt"
	"time"

	"github.com/knirvcorp/knirvbase/pkg/nrv"

	"data-encoder/internal/writer"
)

// DomainMath is the Slot 10 domain signature for Math Mode.
const DomainMath uint16 = 0x2000

// MathSymbol enumerates the three Symbolic Categories emitted by the NLP
// Refraction Layer (nlp_bridge.go §3.3) for Math Mode frames. These replace
// the previous fine-grained operator enum. Specific arithmetic operations
// (add, mul, etc.) are captured by the 16-dim LSH projection in Slots 0-3;
// Slot 4 carries only the structural role of the token.
type MathSymbol uint8

const (
	// MathSymbolOperand: a numeric literal or measured quantity.
	// SpaCy: dep=nummod, pos=NUM
	MathSymbolOperand MathSymbol = 0x01

	// MathSymbolVariable: an algebraic placeholder or named quantity.
	// SpaCy: dep=nsubj (in math context), pos=SYM
	MathSymbolVariable MathSymbol = 0x02

	// MathSymbolOperator: an arithmetic or relational operator.
	// SpaCy: dep=ROOT (where token value is an operator), pos=VERB
	// The specific operator (add/mul/eq/lt/…) is encoded in Slots 0-3.
	MathSymbolOperator MathSymbol = 0x04
)

// PackSlot4 builds the 32-bit Slot 4 register.
//
//	operatorByte — POS tag (non-math) or MathOperator (math mode); fills 0x000000FF.
//	jitterPayload — 24-bit value from FlashSearch; fills 0xFFFFFF00.
//
// The jitter payload is always injected regardless of domain. In Math Mode the
// operator byte must be a MathOperator constant; any value > 0xFF is truncated.
func PackSlot4(operatorByte uint8, jitterPayload uint32) uint32 {
	return uint32(operatorByte) | ((jitterPayload & 0x00FFFFFF) << 8)
}

// UnpackSlot4 extracts the operator byte and jitter payload from a packed register.
func UnpackSlot4(reg uint32) (operatorByte uint8, jitterPayload uint32) {
	operatorByte = uint8(reg & 0x000000FF)
	jitterPayload = (reg >> 8) & 0x00FFFFFF
	return
}

// TensorPacker orchestrates all 12 slots into the NeuralFrame binary.
type TensorPacker struct {
	flashSearch *FlashSearchHelper
	nrvKB       *NRVKnowledgeBase // 16-dim re-indexed NRV KB
}

// NewTensorPacker creates a new TensorPacker instance.
func NewTensorPacker() *TensorPacker {
	return &TensorPacker{
		flashSearch: &FlashSearchHelper{},
		nrvKB:       &NRVKnowledgeBase{},
	}
}

// Orchestrate builds the full 12-slot uint32 array for one bracket.
// It injects the Temporal Salt (Slot 11) and Flash Search jitter (Slot 4 upper bits).
func (tp *TensorPacker) Orchestrate(base *SlotVector, pos uint16, domainSig uint16) []uint32 {
	slots := base.Copy()

	// Slot 11: Temporal Salt — (PosIndex << 16) | TemporalSalt
	salt := uint16(time.Now().UnixNano() & 0xFFFF)
	slots[11] = uint32(pos) | (uint32(salt) << 16)

	// Flash Search: use first 4 bytes of current hash state as lookup key.
	jitter := tp.flashSearch.Lookup(slots[:4], tp.nrvKB)

	// Slot 4: pack operator (0xFF zone) + jitter payload (0xFFFFFF zone).
	var operatorByte uint8
	if domainSig == DomainMath {
		operatorByte = uint8(slots[4] & 0xFF) // preserve Math operator set by encoder
	} else {
		operatorByte = uint8(slots[4] & 0xFF) // POS/Grammar from nlp_bridge
	}
	slots[4] = PackSlot4(operatorByte, jitter)

	return slots
}

// SaveTrainingFrames writes frames as .nrv Tier-3 Brackets into the
// encoder_output KNIRVBASE collection via the embedded NRVWriter.
func (tp *TensorPacker) SaveTrainingFrames(frames []*NeuralFrame, w *writer.NRVWriter) error {
	for _, frame := range frames {
		projBytes := SlotsToProjections(frame.Slots[:4])
		var proj [32]byte
		copy(proj[:], projBytes)

		memBytes := Slots6to8(frame.Slots[6:9])
		var mem [14]byte
		copy(mem[:], memBytes)

		bracket := &nrv.Bracket{
			Projections: proj,
			Syntactic:   uint8(frame.Slots[4] & 0xFF),
			DepHead:     int8(frame.Slots[5]),
			IntentFlags: uint8(frame.Slots[9]),
			DomainSig:   uint16(frame.Slots[10]),
			Memory:      mem,
			GoldenSeed:  frame.Slots[11],
		}
		if err := w.WriteBracket(bracket); err != nil {
			return fmt.Errorf("saveTrainingFrames: frame %d: %w", frame.FrameID, err)
		}
	}
	return nil
}
