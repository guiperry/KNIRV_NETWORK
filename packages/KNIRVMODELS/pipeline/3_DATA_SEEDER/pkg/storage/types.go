package storage

import (
	"encoding/json"
)

type WeightRecord struct {
	TokenID      int32   `json:"token_id"`
	BestSeed     []byte  `json:"best_seed"`
	FitnessScore float64 `json:"fitness_score"`
	Generation   int32   `json:"generation"`
	ContextKey   uint32  `json:"context_key"`
}

// JSONTrainingRecord represents the structure in the JSON file
// Using PascalCase to match the existing training_frames.json format
type JSONTrainingRecord struct {
	SchemaVersion int32
	// Metadata
	SourceFile    string
	ChunkID       int32
	WindowStart   int32
	WindowEnd     int32
	ContextLength int32

	// ASIC input slots (12 x 4 bytes = 48 bytes)
	AsicSlots0  int32
	AsicSlots1  int32
	AsicSlots2  int32
	AsicSlots3  int32
	AsicSlots4  int32
	AsicSlots5  int32
	AsicSlots6  int32
	AsicSlots7  int32
	AsicSlots8  int32
	AsicSlots9  int32
	AsicSlots10 int32
	AsicSlots11 int32

	// Target
	TargetTokenID int32
	TokenSequence []int32
	AssertionSpan []int32

	// Seed (placeholder for Stage 3)
	BestSeed []byte
}

// jsonTrainingRecordWire accepts both the current snake_case encoder output
// and the historical PascalCase representation without decoding every object
// through map[string]interface{}.
type jsonTrainingRecordWire struct {
	SchemaVersion       int32   `json:"schema_version"`
	LegacySchemaVersion int32   `json:"SchemaVersion"`
	SourceFile          string  `json:"source_file"`
	LegacySourceFile    string  `json:"SourceFile"`
	ChunkID             int32   `json:"chunk_id"`
	LegacyChunkID       int32   `json:"ChunkID"`
	WindowStart         int32   `json:"window_start"`
	LegacyWindowStart   int32   `json:"WindowStart"`
	WindowEnd           int32   `json:"window_end"`
	LegacyWindowEnd     int32   `json:"WindowEnd"`
	ContextLength       int32   `json:"context_length"`
	LegacyContextLength int32   `json:"ContextLength"`
	FeatureVector       []int32 `json:"feature_vector"`
	AsicSlots0          int32   `json:"asic_slot_0"`
	AsicSlots1          int32   `json:"asic_slot_1"`
	AsicSlots2          int32   `json:"asic_slot_2"`
	AsicSlots3          int32   `json:"asic_slot_3"`
	AsicSlots4          int32   `json:"asic_slot_4"`
	AsicSlots5          int32   `json:"asic_slot_5"`
	AsicSlots6          int32   `json:"asic_slot_6"`
	AsicSlots7          int32   `json:"asic_slot_7"`
	AsicSlots8          int32   `json:"asic_slot_8"`
	AsicSlots9          int32   `json:"asic_slot_9"`
	AsicSlots10         int32   `json:"asic_slot_10"`
	AsicSlots11         int32   `json:"asic_slot_11"`
	LegacyAsicSlots0    int32   `json:"AsicSlots0"`
	LegacyAsicSlots1    int32   `json:"AsicSlots1"`
	LegacyAsicSlots2    int32   `json:"AsicSlots2"`
	LegacyAsicSlots3    int32   `json:"AsicSlots3"`
	LegacyAsicSlots4    int32   `json:"AsicSlots4"`
	LegacyAsicSlots5    int32   `json:"AsicSlots5"`
	LegacyAsicSlots6    int32   `json:"AsicSlots6"`
	LegacyAsicSlots7    int32   `json:"AsicSlots7"`
	LegacyAsicSlots8    int32   `json:"AsicSlots8"`
	LegacyAsicSlots9    int32   `json:"AsicSlots9"`
	LegacyAsicSlots10   int32   `json:"AsicSlots10"`
	LegacyAsicSlots11   int32   `json:"AsicSlots11"`
	TargetToken         int32   `json:"target_token"`
	TargetTokenID       int32   `json:"target_token_id"`
	LegacyTargetTokenID int32   `json:"TargetTokenID"`
	TokenSequence       []int32 `json:"token_sequence"`
	LegacyTokenSequence []int32 `json:"TokenSequence"`
	AssertionSpan       []int32 `json:"assertion_span"`
	LegacyAssertionSpan []int32 `json:"AssertionSpan"`
	BestSeed            string  `json:"best_seed"`
	LegacyBestSeed      string  `json:"BestSeed"`
}

func (jtr *JSONTrainingRecord) UnmarshalJSON(data []byte) error {
	var wire jsonTrainingRecordWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	chooseInt := func(current, legacy int32) int32 {
		if current != 0 {
			return current
		}
		return legacy
	}
	chooseString := func(current, legacy string) string {
		if current != "" {
			return current
		}
		return legacy
	}
	chooseSlice := func(current, legacy []int32) []int32 {
		if len(current) > 0 {
			return current
		}
		return legacy
	}
	jtr.SchemaVersion = chooseInt(wire.SchemaVersion, wire.LegacySchemaVersion)
	jtr.SourceFile = chooseString(wire.SourceFile, wire.LegacySourceFile)
	jtr.ChunkID, jtr.WindowStart = chooseInt(wire.ChunkID, wire.LegacyChunkID), chooseInt(wire.WindowStart, wire.LegacyWindowStart)
	jtr.WindowEnd, jtr.ContextLength = chooseInt(wire.WindowEnd, wire.LegacyWindowEnd), chooseInt(wire.ContextLength, wire.LegacyContextLength)
	slots := [12]int32{wire.AsicSlots0, wire.AsicSlots1, wire.AsicSlots2, wire.AsicSlots3, wire.AsicSlots4, wire.AsicSlots5, wire.AsicSlots6, wire.AsicSlots7, wire.AsicSlots8, wire.AsicSlots9, wire.AsicSlots10, wire.AsicSlots11}
	if slots == [12]int32{} {
		slots = [12]int32{wire.LegacyAsicSlots0, wire.LegacyAsicSlots1, wire.LegacyAsicSlots2, wire.LegacyAsicSlots3, wire.LegacyAsicSlots4, wire.LegacyAsicSlots5, wire.LegacyAsicSlots6, wire.LegacyAsicSlots7, wire.LegacyAsicSlots8, wire.LegacyAsicSlots9, wire.LegacyAsicSlots10, wire.LegacyAsicSlots11}
	}
	if len(wire.FeatureVector) >= len(slots) {
		copy(slots[:], wire.FeatureVector)
	}
	jtr.AsicSlots0, jtr.AsicSlots1, jtr.AsicSlots2, jtr.AsicSlots3 = slots[0], slots[1], slots[2], slots[3]
	jtr.AsicSlots4, jtr.AsicSlots5, jtr.AsicSlots6, jtr.AsicSlots7 = slots[4], slots[5], slots[6], slots[7]
	jtr.AsicSlots8, jtr.AsicSlots9, jtr.AsicSlots10, jtr.AsicSlots11 = slots[8], slots[9], slots[10], slots[11]
	jtr.TargetTokenID = chooseInt(wire.TargetToken, chooseInt(wire.TargetTokenID, wire.LegacyTargetTokenID))
	jtr.TokenSequence, jtr.AssertionSpan = chooseSlice(wire.TokenSequence, wire.LegacyTokenSequence), chooseSlice(wire.AssertionSpan, wire.LegacyAssertionSpan)
	jtr.BestSeed = []byte(chooseString(wire.BestSeed, wire.LegacyBestSeed))
	return nil
}

// Helper methods to convert between different formats
func (jtr *JSONTrainingRecord) GetTargetToken() int32 {
	return jtr.TargetTokenID
}

func (jtr *JSONTrainingRecord) SetTargetToken(token int32) {
	jtr.TargetTokenID = token
}

func (jtr *JSONTrainingRecord) GetAsicSlots() [12]int32 {
	return [12]int32{
		jtr.AsicSlots0, jtr.AsicSlots1, jtr.AsicSlots2, jtr.AsicSlots3,
		jtr.AsicSlots4, jtr.AsicSlots5, jtr.AsicSlots6, jtr.AsicSlots7,
		jtr.AsicSlots8, jtr.AsicSlots9, jtr.AsicSlots10, jtr.AsicSlots11,
	}
}

func (jtr *JSONTrainingRecord) SetAsicSlots(slots [12]int32) {
	jtr.AsicSlots0, jtr.AsicSlots1, jtr.AsicSlots2, jtr.AsicSlots3 = slots[0], slots[1], slots[2], slots[3]
	jtr.AsicSlots4, jtr.AsicSlots5, jtr.AsicSlots6, jtr.AsicSlots7 = slots[4], slots[5], slots[6], slots[7]
	jtr.AsicSlots8, jtr.AsicSlots9, jtr.AsicSlots10, jtr.AsicSlots11 = slots[8], slots[9], slots[10], slots[11]
}
