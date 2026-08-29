package loader

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"knirvhasher/pkg/hashing/schema"
)

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
