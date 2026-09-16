package loader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"knirvhasher/pkg/hashing/schema"
)

// StreamStats reports frame counts observed while streaming a JSON array.
type StreamStats struct {
	Read     int
	Accepted int
	Skipped  int
}

// StreamFrames decodes one frame at a time, so training does not retain a
// potentially million-record JSON archive in memory. visit receives only
// valid frames and may stop the stream by returning an error.
func StreamFrames(path string, visit func(index int, frame schema.TrainingFrame) error) (StreamStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return StreamStats{}, fmt.Errorf("open frames file: %w", err)
	}
	defer file.Close()

	dec := json.NewDecoder(bufio.NewReader(file))
	first, err := dec.Token()
	if err != nil {
		return StreamStats{}, fmt.Errorf("read frames JSON: %w", err)
	}
	if delim, ok := first.(json.Delim); !ok || delim != '[' {
		return StreamStats{}, fmt.Errorf("decode frames JSON: expected array")
	}

	var stats StreamStats
	for dec.More() {
		var frame schema.TrainingFrame
		if err := dec.Decode(&frame); err != nil {
			return stats, fmt.Errorf("decode frame %d: %w", stats.Read, err)
		}
		stats.Read++
		if len(frame.TokenSequence) == 0 {
			stats.Skipped++
			continue
		}
		if err := visit(stats.Accepted, frame); err != nil {
			return stats, err
		}
		stats.Accepted++
	}
	if _, err := dec.Token(); err != nil {
		return stats, fmt.Errorf("finish frames JSON: %w", err)
	}
	return stats, nil
}

// CountFrames performs a low-memory validation/counting pass. It is used
// before streaming training so progress can report a useful total.
func CountFrames(path string) (StreamStats, error) {
	return StreamFrames(path, func(_ int, _ schema.TrainingFrame) error { return nil })
}

// LoadFrames reads a JSON file containing an array of TrainingFrame records
// and returns them as a slice. Frames with empty TokenSequence are filtered
// out defensively.
func LoadFrames(path string) ([]schema.TrainingFrame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read frames file: %w", err)
	}

	var raw []schema.TrainingFrame
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode frames JSON: %w", err)
	}

	filtered := make([]schema.TrainingFrame, 0, len(raw))
	for i, f := range raw {
		if len(f.TokenSequence) == 0 {
			log.Printf("frame %d: skipping empty TokenSequence (source=%s)", i, f.SourceFile)
			continue
		}
		filtered = append(filtered, f)
	}

	log.Printf("loaded %d frames from %s (%d skipped)", len(filtered), path, len(raw)-len(filtered))
	return filtered, nil
}
