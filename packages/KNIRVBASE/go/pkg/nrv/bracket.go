package nrv

type DeltaType string

const (
	DeltaTypeI DeltaType = "I"
	DeltaTypeP DeltaType = "P"
)

const BracketSize = 80

type Bracket struct {
	ID          string
	Projections [32]byte // LSH Projections (Slots 0-3: Semantic Compass)
	SubSecondUS uint32   // Sub-second timestamp in microseconds
	POSTag      uint8    // Slot 4: Part-of-Speech tag
	Tense       uint8    // Slot 4: Tense
	Plurality   uint8    // Slot 4: Plurality
	DepHead     uint8    // Slot 5: Dependency Head
	IntentFlags uint8    // Slot 9: Intent flags (Question/Command/Code)
	DomainSig   uint16   // Slot 10: Domain signature (Math/Code/Prose)
	GoldenSeed  uint32   // Solved nonce from ASIC pass
	Memory      [14]byte // Slots 6-8: Recursive XOR summary of last 10 headers (temporal memory)
	LSHSalt     uint32   // Slot 11: Temporal Lock (LSH forest seed)
	Meta        *BracketMeta
}

type BracketMeta struct {
	ID         string    `json:"id"`
	Type       DeltaType `json:"type"`
	AnchorID   *string   `json:"anchor_id"`
	Offset     int       `json:"offset"`
	DriftScore float64   `json:"drift_score"`
}

type LinguisticMapping struct {
	Token string `json:"token"`
	Unit  string `json:"unit"`
}

type ThermoAtmosphere struct {
	AvgTempC  float32 `json:"avg_temp_c"`
	PeakVoltV float32 `json:"peak_volt_v"`
	ClockMHz  float32 `json:"clock_mhz"`
}

type Z3Result struct {
	Status    string  `json:"status"`
	Relevance float64 `json:"relevance"`
}

type BracketBinaryMap struct {
	Count  int   `json:"count"`
	Offset int64 `json:"offset"`
	Length int   `json:"length"`
}

type SyntacticProfile struct {
	POSTag    uint8
	Tense     uint8
	Plurality uint8
	DepHead   int16
}

type IntentDomain struct {
	IntentFlags uint8
	DomainSig   uint16
}
