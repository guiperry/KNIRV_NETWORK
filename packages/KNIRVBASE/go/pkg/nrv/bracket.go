package nrv

type DeltaType string

const (
	DeltaTypeI DeltaType = "I"
	DeltaTypeP DeltaType = "P"
)

const BracketSize = 80

type Bracket struct {
	ID          string
	LSHSalt     uint32
	Projections [64]byte
	SubSecondUS uint32
	ASICLoops   uint32
	GoldenSeed  uint32
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
