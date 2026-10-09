package mapper

import "data-encoder/pkg/slotpack"

// TensorPacker orchestrates bit-level placement of metadata into the 12-slot format.
// When a SlotSchema is provided via NewSchemaAwarePacker, Slot 10 is taken directly
// from schema.Domain.Slot10Base instead of being auto-detected from content keywords.
type TensorPacker struct {
	signalIndices []int
	schema        *SlotSchema // optional; nil = keyword-based auto-detection
}

// NewTensorPacker creates a packer that auto-detects the domain from content keywords.
func NewTensorPacker(signalIndices []int) *TensorPacker {
	return &TensorPacker{signalIndices: signalIndices}
}

// NewSchemaAwarePacker creates a packer whose Slot 10 domain value is driven by the schema.
// Use this when the domain is known at construction time (e.g. loaded from a YAML schema file).
func NewSchemaAwarePacker(signalIndices []int, schema *SlotSchema) *TensorPacker {
	return &TensorPacker{signalIndices: signalIndices, schema: schema}
}

// PackFrame creates the 12-slot bitmask tensor according to the HASHER-MAPPER spec.
// When the packer was created with NewSchemaAwarePacker, Slot 10 uses the schema's
// domain base; otherwise it falls back to keyword-based detection.
func (p *TensorPacker) PackFrame(
	embedding []float32,
	pos uint8,
	tense uint8,
	depHash uint32,
	lastHeaders []uint32,
	tokenPos uint16,
	instruction string,
	input string,
) [12]uint32 {
	var domain *uint32
	if p.schema != nil {
		base := p.schema.Domain.Slot10Base
		domain = &base
	}
	return slotpack.PackFrame(p.signalIndices, domain, embedding, pos, tense, depHash, lastHeaders, tokenPos, instruction, input)
}

// detectDomain is the legacy keyword-based fallback used when no schema is loaded.
func detectDomain(instr, input string) uint32 { return slotpack.DetectDomain(instr, input) }

func quantizeFloatToUint16(val float32) uint16 {
	if val < -1.0 {
		val = -1.0
	}
	if val > 1.0 {
		val = 1.0
	}
	scaled := (val + 1.0) / 2.0 * 65535.0
	return uint16(scaled + 0.5)
}
