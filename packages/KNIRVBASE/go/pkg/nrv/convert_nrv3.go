package nrv

import (
	"fmt"
	"strconv"
)

// MaxBracketsPerFrame bounds the brackets in one converted KNIRVBASE frame. A
// hasher "frame" can hold hundreds of thousands of brackets (a real batch file
// has a single 344k-bracket frame), but KNIRVBASE keeps a per-bracket index in
// its fixed 5 MiB registry, so the converter splits the stream into frames of
// at most this size.
const MaxBracketsPerFrame = 1024

// ConvertedFrame is one KNIRVBASE frame produced from (part of) an NRV3 frame.
// Brackets is ready for NRVWriter.AppendFrame together with Index/Thermo/Ling.
type ConvertedFrame struct {
	ID       string
	Brackets []byte
	Index    []BracketMeta
	Thermo   ThermoAtmosphere
	Ling     LinguisticMapping
}

// ConvertFrame re-encodes one NRV3 frame for KNIRVBASE.
//
// NRV3 stores FULL 80-byte brackets; its per-bracket "xor" blob is a chain
// delta against the previous bracket, kept only for the seeder. KNIRVBASE
// instead delta-encodes projections against an anchor I-bracket (P-bracket
// projections = projections XOR anchor projections). Copying NRV3 bytes and
// labelling them P would therefore corrupt every projection on read, so the
// converter decodes the full brackets and applies KNIRVBASE's own I/P policy
// (IFrameInterval, DriftThreshold — the same one the FrameTicker uses) per
// output frame. Reading the result back yields the original brackets exactly.
//
// raw must hold src.Count*BracketSize bytes. A frame larger than
// MaxBracketsPerFrame is split into frames "<id>#1", "<id>#2", ...; each is
// self-contained (its first bracket is an I-bracket), so anchors never cross
// frames. NRV3 z3 metadata is not carried over: the hasher never fills it.
func ConvertFrame(src NRV3FrameEntry, raw []byte) ([]ConvertedFrame, error) {
	if src.Count != len(src.BracketIndex) {
		return nil, fmt.Errorf("nrv3: frame %s count mismatch: count=%d bracket_index=%d",
			src.ID, src.Count, len(src.BracketIndex))
	}
	if len(raw) != src.Count*BracketSize {
		return nil, fmt.Errorf("nrv3: frame %s has %d bytes of brackets, want %d",
			src.ID, len(raw), src.Count*BracketSize)
	}
	for _, e := range src.BracketIndex {
		if t := DeltaType(e.Type); t != DeltaTypeI && t != DeltaTypeP {
			return nil, fmt.Errorf("nrv3: frame %s bracket %d has invalid delta type %q", src.ID, e.Index, e.Type)
		}
	}

	ling := LinguisticMapping{Token: src.Linguistic.Domain, Unit: strconv.Itoa(src.Linguistic.Tokens)}
	thermo := ThermoAtmosphere{AvgTempC: float32(src.Thermo.Temperature)}

	var out []ConvertedFrame
	for start := 0; start < src.Count || (start == 0 && src.Count == 0); start += MaxBracketsPerFrame {
		end := start + MaxBracketsPerFrame
		if end > src.Count {
			end = src.Count
		}
		id := src.ID
		if src.Count > MaxBracketsPerFrame {
			id = fmt.Sprintf("%s#%d", src.ID, start/MaxBracketsPerFrame+1)
		}
		cf := ConvertedFrame{
			ID:       id,
			Brackets: make([]byte, 0, (end-start)*BracketSize),
			Ling:     ling,
			Thermo:   thermo,
		}

		var anchor [32]byte // projections of the current I-bracket
		var anchorID string
		for i := start; i < end; i++ {
			local := i - start
			var enc [BracketSize]byte
			copy(enc[:], raw[i*BracketSize:(i+1)*BracketSize])
			b := DecodeBracket(enc)

			meta := BracketMeta{
				ID:     fmt.Sprintf("%s-%d", src.ID, src.BracketIndex[i].Index),
				Offset: local * BracketSize,
			}
			isI := local == 0 || i%IFrameInterval == 0
			if !isI {
				meta.DriftScore = EuclideanDrift(b.Projections, anchor)
				isI = meta.DriftScore > DriftThreshold
			}
			if isI {
				meta.Type = DeltaTypeI
				meta.DriftScore = 0
				anchor, anchorID = b.Projections, meta.ID
			} else {
				meta.Type = DeltaTypeP
				a := anchorID
				meta.AnchorID = &a
				b.Projections = XORProjections(b.Projections, anchor)
			}
			e := EncodeBracket(&b)
			cf.Brackets = append(cf.Brackets, e[:]...)
			cf.Index = append(cf.Index, meta)
		}
		out = append(out, cf)
		if src.Count == 0 {
			break
		}
	}
	return out, nil
}
