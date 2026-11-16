// Package nrvtrainer is the single entry point every KNIRV training stage uses
// to learn from signed .nrv assets. It reads a container once, exposes the
// uniform FramePayload on each frame, refuses to train on frames whose Z3
// assertion did not validate, and dispatches the surviving frames to the
// per-target trainers (HASHER, HEART, HANDS, HEAD) and to the solution-adapter
// compiler (KNIRVULORA).
//
// Two containers exist in the ecosystem and both are read here:
//
//   - NRV3 ("NRV3" magic): registry JSON + a flat bracket section. It has no
//     canonical payload section, so payloads are accepted from an optional
//     "payload" member on the registry frame or from a sidecar file written by the
//     producer.
//   - Canonical (magic 0x4E525621): 12-byte header + padded registry JSON +
//     byte-addressed bracket and payload sections.
//
// The container structs, the frame payload schema and its validation come from
// KNIRVBASE (github.com/knirvcorp/knirvbase/pkg/nrv), the shared .nrv format.
// KNIRVBASE/rust/src/payload.rs stays the schema source of truth.
//
// Signatures are reported, never enforced. Each frame gets a SignatureStatus
// (verified / unsigned / invalid / verify-error) from KNIRVBASE's frame verifier
// when a trusted public key is supplied through a SignatureCheck. Without a key,
// signed frames report verify-error. No status drops a frame from training.
// Asset.Signed and KeyID still report whether a PQCManifest is present. Enforcement
// waits for the owner's public-key distribution (see the F_nrv_signature report).
package nrvtrainer

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// Format identifies the container layout an asset was read from.
type Format string

const (
	FormatNRV3      Format = "nrv3"
	FormatCanonical Format = "nrv"
)

// Z3 status values used by the canonical container.
const (
	Z3Valid   = "VALID"
	Z3Invalid = "INVALID"
	Z3Unknown = "UNKNOWN"
)

// Payload schema, shared with KNIRVBASE. These are aliases, so the trainer and the
// core cannot drift apart.
type (
	FramePayload              = nrv.FramePayload
	TokenWindowPayload        = nrv.TokenWindowPayload
	ErrorTestCasePayload      = nrv.ErrorTestCasePayload
	TRLSamplePayload          = nrv.TRLSamplePayload
	ToolCallPayload           = nrv.ToolCallPayload
	SupervisionEpisodePayload = nrv.SupervisionEpisodePayload
	BoxPayload                = nrv.BoxPayload
	PerceptualPayload         = nrv.PerceptualPayload
)

// Asset is a fully decoded .nrv file. Frames are in registry order.
type Asset struct {
	Path      string
	Format    Format
	Dataset   string
	Frames    []Frame
	Signed    bool
	KeyID     string
	Algorithm string
	// VerificationSkipped is true when PQC material is present but no trusted key
	// was configured, so no frame signature was checked.
	VerificationSkipped bool
	// SignatureCounts is the per-status frame count over every live frame.
	SignatureCounts map[SignatureStatus]int
}

// Frame is one .nrv frame plus the payload it carried, if any.
type Frame struct {
	ID            string
	TimestampUnix int64
	Z3Status      string
	BracketCount  int
	RawBrackets   []byte
	Payload       FramePayload
	HasPayload    bool
	// Signature is this frame's signature status. It is reported only; it never
	// decides whether the frame trains.
	Signature SignatureStatus
	// SignatureDetail carries the reason for an invalid or verify-error status.
	SignatureDetail string
}

// Valid reports whether the frame's assertion validated. Only VALID frames may
// enter a training corpus; treating UNKNOWN as valid would let unverified
// assertions silently become weights.
func (f Frame) Valid() bool {
	return strings.EqualFold(strings.TrimSpace(f.Z3Status), Z3Valid)
}

// LoadAsset reads path with no trusted public key. Signed frames report
// verify-error. Use LoadAssetWithCheck to verify them.
func LoadAsset(path string) (*Asset, error) {
	return LoadAssetWithCheck(path, nil)
}

// LoadAssetWithCheck reads path like LoadAsset and classifies every live frame's
// signature with check. A nil check means no trusted key.
func LoadAssetWithCheck(path string, check *SignatureCheck) (*Asset, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return nil, fmt.Errorf("nrv asset path must be absolute: %q", path)
	}
	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("stat nrv asset: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("nrv asset is not a regular file: %s", clean)
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read nrv asset: %w", err)
	}
	if len(data) < 8 {
		return nil, fmt.Errorf("nrv asset too short (%d bytes)", len(data))
	}

	sidecar, err := loadSidecar(clean)
	if err != nil {
		return nil, err
	}

	if string(data[:4]) == nrv3Magic {
		asset, err := loadNRV3(clean, data, sidecar)
		if err != nil {
			return nil, err
		}
		asset.tallySignatures()
		return asset, nil
	}
	var asset *Asset
	if binary.LittleEndian.Uint32(data[:4]) == nrv.Magic {
		asset, err = loadCanonical(clean, data, sidecar, check)
	} else {
		return nil, fmt.Errorf("nrv asset %s has unknown magic %q", clean, data[:4])
	}
	if err != nil {
		return nil, err
	}
	asset.tallySignatures()
	return asset, nil
}

// TokenWindows returns one token slice per payload-bearing frame, split into
// windows of contextLen tokens (stride defaults to contextLen). Frames whose
// TokenWindow payload is absent or too short are skipped, never padded.
func (a *Asset) TokenWindows(contextLen, stride int) [][]int {
	if contextLen < 2 {
		contextLen = 2
	}
	if stride <= 0 {
		stride = contextLen
	}
	var out [][]int
	for _, f := range a.Frames {
		if !f.Valid() || f.Payload.TokenWindow == nil {
			continue
		}
		tokens := f.Payload.TokenWindow.Tokens
		if len(tokens) < contextLen {
			continue
		}
		for start := 0; start+contextLen <= len(tokens); start += stride {
			window := make([]int, contextLen)
			for i := range window {
				window[i] = int(tokens[start+i])
			}
			out = append(out, window)
		}
	}
	return out
}

// CheckTokenRange rejects any token ID outside [0, vocabSize) in a Z3-valid
// token_window payload. It mirrors the JSONL path, which fails on an
// out-of-vocabulary token instead of training on an aliased one. Invalid frames
// are not checked: they never reach training.
func (a *Asset) CheckTokenRange(vocabSize int) error {
	if vocabSize <= 0 {
		return fmt.Errorf("token range check needs a positive vocabulary size, got %d", vocabSize)
	}
	for _, f := range a.Frames {
		if !f.Valid() || f.Payload.TokenWindow == nil {
			continue
		}
		for i, tok := range f.Payload.TokenWindow.Tokens {
			if uint64(tok) >= uint64(vocabSize) {
				return fmt.Errorf("frame %s token[%d] = %d outside vocabulary [0,%d)", f.ID, i, tok, vocabSize)
			}
		}
	}
	return nil
}

// PayloadCounts reports how many frames carried each payload kind, for logs and
// job status. Frames without a payload are not counted.
func (a *Asset) PayloadCounts() map[string]int {
	counts := map[string]int{}
	for _, f := range a.Frames {
		switch {
		case f.Payload.TokenWindow != nil:
			counts["token_window"]++
		case f.Payload.ErrorTestCase != nil:
			counts["error_test_case"]++
		case f.Payload.TRLSample != nil:
			counts["trl_sample"]++
		case f.Payload.SupervisionEpisode != nil:
			counts["supervision_episode"]++
		case f.Payload.Perceptual != nil:
			counts["perceptual"]++
		}
	}
	return counts
}

// sidecarPayloads is the decoded optional payload sidecar. A producer that
// cannot yet write the canonical payload section emits this next to the asset.
type sidecarPayloads struct {
	byID    map[string]json.RawMessage
	byIndex []json.RawMessage
}

func (s *sidecarPayloads) lookup(id string, index int) json.RawMessage {
	if s == nil {
		return nil
	}
	if raw, ok := s.byID[id]; ok {
		return raw
	}
	if index >= 0 && index < len(s.byIndex) {
		return s.byIndex[index]
	}
	return nil
}

func loadSidecar(assetPath string) (*sidecarPayloads, error) {
	candidates := []string{assetPath + ".payload.json"}
	if strings.HasSuffix(assetPath, ".nrv") {
		candidates = append(candidates, strings.TrimSuffix(assetPath, ".nrv")+".payload.json")
	}
	var data []byte
	for _, candidate := range candidates {
		b, err := os.ReadFile(candidate)
		if err == nil {
			data = b
			break
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read payload sidecar %s: %w", candidate, err)
		}
	}
	if data == nil {
		return nil, nil
	}

	// Object of frame ID -> payload, optionally wrapped in {"frames": ...}.
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err == nil {
		out := &sidecarPayloads{byID: map[string]json.RawMessage{}}
		if frames, ok := object["frames"]; ok {
			var framed map[string]json.RawMessage
			if err := json.Unmarshal(frames, &framed); err == nil {
				out.byID = framed
				return out, nil
			}
			var listed []json.RawMessage
			if err := json.Unmarshal(frames, &listed); err != nil {
				return nil, fmt.Errorf("payload sidecar frames must be an object or array: %w", err)
			}
			out.byIndex = listed
			return out, nil
		}
		for id, raw := range object {
			out.byID[id] = raw
		}
		return out, nil
	}

	// Bare array aligned with registry frame order.
	var listed []json.RawMessage
	if err := json.Unmarshal(data, &listed); err != nil {
		return nil, fmt.Errorf("payload sidecar is neither an object nor an array: %w", err)
	}
	return &sidecarPayloads{byIndex: listed}, nil
}

// nrv3Magic is the NRV3 container magic ("NRV3" as bytes). KNIRVBASE exposes the
// same constant as nrv.NRV3Magic.
const nrv3Magic = nrv.NRV3Magic

// Layout constants, aliased from KNIRVBASE so tests and helpers keep short names.
const (
	canonicalMagic  = nrv.Magic
	headerSize      = nrv.HeaderSize
	registryPadding = nrv.RegistryPadding
	bracketSize     = nrv.BracketSize
)

// decodePayload decodes one frame's payload bytes through KNIRVBASE. Absent or
// JSON null means the frame carries no payload, which is not an error.
func decodePayload(raw []byte, frameID string) (FramePayload, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return FramePayload{}, false, nil
	}
	payload, err := nrv.DecodeFramePayload(raw)
	if err != nil {
		return FramePayload{}, false, fmt.Errorf("decode payload for frame %s: %w", frameID, err)
	}
	return payload, true, nil
}

// nrv3PayloadMembers reads the optional per-frame "payload" member that the NRV3
// registry may carry. KNIRVBASE's NRV3FrameEntry does not model it, so it is read
// from the same registry bytes here.
type nrv3PayloadMembers struct {
	Frames []struct {
		Payload json.RawMessage `json:"payload,omitempty"`
	} `json:"frames"`
}

func loadNRV3(path string, data []byte, sidecar *sidecarPayloads) (*Asset, error) {
	reader, err := nrv.ParseNRV3(data)
	if err != nil {
		return nil, err
	}
	regLen := int(binary.LittleEndian.Uint32(data[4:8]))
	var members nrv3PayloadMembers
	if err := json.Unmarshal(data[8:8+regLen], &members); err != nil {
		return nil, fmt.Errorf("nrv3: parse payload members: %w", err)
	}

	asset := &Asset{Path: path, Format: FormatNRV3, Frames: make([]Frame, 0, len(reader.Registry.Frames))}
	for i, entry := range reader.Registry.Frames {
		status := Z3Unknown
		if entry.Z3.ConstraintCount > 0 {
			if entry.Z3.Satisfiable {
				status = Z3Valid
			} else {
				status = Z3Invalid
			}
		}
		frame := Frame{
			ID:            entry.ID,
			TimestampUnix: entry.TimestampUnix,
			Z3Status:      status,
			BracketCount:  entry.Count,
			// NRV3 has no frame-signature section.
			Signature: SignatureUnsigned,
		}
		brackets := reader.RawBrackets(entry)
		if brackets == nil {
			return nil, fmt.Errorf("nrv3: frame %s bracket range outside section", entry.ID)
		}
		frame.RawBrackets = brackets

		var raw json.RawMessage
		if i < len(members.Frames) {
			raw = members.Frames[i].Payload
		}
		if len(raw) == 0 || string(raw) == "null" {
			raw = sidecar.lookup(entry.ID, i)
		}
		payload, hasPayload, err := decodePayload(raw, entry.ID)
		if err != nil {
			return nil, err
		}
		frame.Payload = payload
		frame.HasPayload = hasPayload
		asset.Frames = append(asset.Frames, frame)
	}
	return asset, nil
}

func loadCanonical(path string, data []byte, sidecar *sidecarPayloads, check *SignatureCheck) (*Asset, error) {
	if len(data) < nrv.HeaderSize+nrv.RegistryPadding {
		return nil, fmt.Errorf("nrv: file too short for header + registry padding (%d bytes)", len(data))
	}
	padded := data[nrv.HeaderSize : nrv.HeaderSize+nrv.RegistryPadding]
	end := 0
	for i := len(padded) - 1; i >= 0; i-- {
		if b := padded[i]; b != 0 && b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			end = i + 1
			break
		}
	}
	if end == 0 {
		return nil, fmt.Errorf("nrv: empty registry")
	}
	var registry nrv.Registry
	if err := json.Unmarshal(padded[:end], &registry); err != nil {
		return nil, fmt.Errorf("nrv: parse registry: %w", err)
	}

	asset := &Asset{
		Path:      path,
		Format:    FormatCanonical,
		Dataset:   registry.DatasetID,
		Frames:    make([]Frame, 0, len(registry.Frames)),
		KeyID:     registry.PQCManifest.KeyID,
		Algorithm: registry.PQCManifest.Algorithm,
		Signed:    registry.PQCManifest.FileSignature != "",
	}
	// Validate the trusted key once for this load. With no usable key nothing is
	// verified, so VerificationSkipped is set for any signed content.
	keyProblem := check.keyProblem()
	asset.VerificationSkipped = keyProblem != "" && (asset.Signed || len(registry.PQCManifest.FrameSignatures) > 0)

	for i, entry := range registry.Frames {
		if entry.Tombstone != nil {
			continue
		}
		frame := Frame{
			ID:            entry.ID,
			TimestampUnix: entry.TimestampUnix,
			Z3Status:      normalizeZ3(entry.Z3.Status),
			BracketCount:  entry.Brackets.Count,
		}
		// Bracket geometry must match the declared count exactly, as the NRV3
		// path enforces: length == count*BracketSize (which also guarantees
		// 80-byte alignment), with non-negative values.
		if entry.Brackets.Count < 0 || entry.Brackets.Length < 0 ||
			int64(entry.Brackets.Length) != int64(entry.Brackets.Count)*nrv.BracketSize {
			return nil, fmt.Errorf("nrv: frame %s bracket length %d does not equal count %d * %d", entry.ID, entry.Brackets.Length, entry.Brackets.Count, nrv.BracketSize)
		}
		if entry.Brackets.Length > 0 {
			start, limit := entry.Brackets.Offset, int64(entry.Brackets.Length)
			if start < 0 || limit < 0 || start+limit > int64(len(data)) {
				return nil, fmt.Errorf("nrv: frame %s bracket range [%d,%d) outside file", entry.ID, start, start+limit)
			}
			frame.RawBrackets = data[start : start+limit]
		}
		var raw []byte
		if entry.Payload != nil && entry.Payload.Length > 0 {
			start, limit := int64(entry.Payload.Offset), int64(entry.Payload.Length)
			if start < 0 || limit < 0 || start+limit > int64(len(data)) {
				return nil, fmt.Errorf("nrv: frame %s payload range [%d,%d) outside file", entry.ID, start, start+limit)
			}
			raw = data[start : start+limit]
		} else {
			raw = sidecar.lookup(entry.ID, i)
		}
		payload, hasPayload, err := decodePayload(raw, entry.ID)
		if err != nil {
			return nil, err
		}
		frame.Payload = payload
		frame.HasPayload = hasPayload
		frame.Signature, frame.SignatureDetail = classifyFrameSignature(check, keyProblem, &registry, data, entry)
		asset.Frames = append(asset.Frames, frame)
	}
	return asset, nil
}

func normalizeZ3(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case Z3Valid:
		return Z3Valid
	case Z3Invalid:
		return Z3Invalid
	default:
		return Z3Unknown
	}
}
