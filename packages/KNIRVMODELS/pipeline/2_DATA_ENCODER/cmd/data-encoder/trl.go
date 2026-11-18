package main

import (
	"fmt"
	"log"
	"strings"

	"data-encoder/pkg/schema"
	supervisionpipeline "knirvmodels/pkg/hashing/supervision/pipeline"
)

// trlText is one record's supervised text: prompt is the Alpaca instruction and
// input, completion is the Alpaca output.
type trlText struct {
	prompt     string
	completion string
}

// trlSide carries each encoded record's SFT text beside the frames. The
// TrainingFrame schema has no text fields, so the text cannot ride on the frame.
// Instead every record gets a unique serial at intake, and every frame the record
// appends is tagged with that serial. The text is keyed by serial, never by
// (SourceFile, ChunkID): two records can share those, and a shared key would let
// one record's text overwrite another's.
type trlSide struct {
	next  int64             // last serial handed out
	texts map[int64]trlText // serial -> text; absent when the record has no Output
	owner []int64           // owner[i] is the record serial of frames[i]
	// dropped counts records whose text was refused because a secret-shaped
	// string survived redaction. Their frames are kept, with no TRLSample.
	dropped int
}

func newTRLSide() *trlSide {
	return &trlSide{texts: make(map[int64]trlText)}
}

// beginRecord assigns the record a fresh serial and, when it has an Alpaca
// output, stores its text under that serial. A record without an output gets a
// serial but no text, so it can never inherit another record's text. Chosen and
// rejected belong to KNIRVINFERENCER and are never written here.
//
// The text is redacted with the same patterns NormalizeBronze applies before any
// .nrv frame is written (plan §7.7.1). See buildTRLText for the fail-closed rule.
func (s *trlSide) beginRecord(record *schema.MinedRecord) int64 {
	s.next++
	serial := s.next
	t, ok, reason := buildTRLText(record, supervisionpipeline.RedactText)
	if !ok {
		if reason != "" {
			s.dropped++
			log.Printf("⚠️  TRL text for record serial %d dropped: %s (secret values not logged)", serial, reason)
		}
		return serial
	}
	s.texts[serial] = t
	return serial
}

// buildTRLText builds one record's SFT text. Redaction runs on the joined prompt
// (Instruction, a blank line, Input) and on the completion, so a secret that
// spans the join is redacted too. The redacted text is then rescanned, and if a
// secret-shaped string still remains the record yields no text (fail closed, the
// same rule NormalizeBronze applies). The rescan covers the prompt and the
// completion alone and also the prompt joined to the completion, because a
// secret can span the prompt | completion boundary. ok is false when there is no
// completion (reason is ""), or when the text is dropped (reason says why).
func buildTRLText(record *schema.MinedRecord, redact func(string) string) (t trlText, ok bool, reason string) {
	completion := redact(record.Output)
	if strings.TrimSpace(completion) == "" {
		return trlText{}, false, ""
	}
	raw := record.Instruction
	if in := record.Input; strings.TrimSpace(in) != "" {
		if strings.TrimSpace(raw) != "" {
			raw += "\n\n"
		}
		raw += in
	}
	prompt := redact(raw)
	if supervisionpipeline.ContainsSecret(prompt) || supervisionpipeline.ContainsSecret(completion) || secretAcrossBoundary(prompt, completion) {
		return trlText{}, false, "redaction left a secret-shaped string"
	}
	return trlText{prompt: prompt, completion: completion}, true, ""
}

// secretAcrossBoundary reports whether a secret-shaped string forms where the
// prompt meets the completion. Each field is clean alone, so the per-field rescan
// cannot see it. The .nrv keeps the two fields apart, but a consumer may join
// them with any separator, so the common separators are all checked.
func secretAcrossBoundary(prompt, completion string) bool {
	for _, sep := range []string{"", " ", "\n\n"} {
		if supervisionpipeline.ContainsSecret(prompt + sep + completion) {
			return true
		}
	}
	return false
}

// tagFrames gives every frame not yet tagged the record serial. Frames are only
// ever appended, so the untagged tail is exactly the frames this record added.
func (s *trlSide) tagFrames(frames []schema.TrainingFrame, serial int64) {
	for len(s.owner) < len(frames) {
		s.owner = append(s.owner, serial)
	}
}

// recordOwners returns the record serial of each frame. A nil side means the
// frames carry no text, so every frame reports serial 0 and no text is attached.
// A side whose tags do not line up with the frames is an error: the text could
// be attached to the wrong record.
func (s *trlSide) recordOwners(frames []schema.TrainingFrame) ([]int64, error) {
	owners := make([]int64, len(frames))
	if s == nil {
		return owners, nil
	}
	if len(s.owner) != len(frames) {
		return nil, fmt.Errorf("trl side map tags %d frames, have %d; refusing to attach text to the wrong record", len(s.owner), len(frames))
	}
	copy(owners, s.owner)
	return owners, nil
}

// logSummary writes the run's drop count to the log. It logs the count only,
// never the text or any secret-shaped string. It is nil-safe: a run without TRL
// state has nothing to report.
func (s *trlSide) logSummary() {
	if s == nil {
		return
	}
	log.Printf("🔒 TRL text: %d record(s) dropped by the secret rescan (text not logged)", s.dropped)
}

// text returns the SFT text for a record serial.
func (s *trlSide) text(serial int64) (trlText, bool) {
	if s == nil {
		return trlText{}, false
	}
	t, ok := s.texts[serial]
	return t, ok
}
