package nrv

type DeltaType string

const (
	DeltaTypeI DeltaType = "I"
	DeltaTypeP DeltaType = "P"
)

const BracketSize = 80

type Bracket struct {
	ID          string
	Projections [32]byte // 0x00-0x1F: slots 0-3 — semantic compass / LSH projections
	SubSecondUS uint32   // 0x20-0x23: metadata — SubSecondUS ticker
	POSTag      uint8    // Slot 4 (unpacked); wire 0x24 bit-packs with Tense, Plurality (PackSyntacticByte)
	Tense       uint8    // Slot 4
	Plurality   uint8    // Slot 4
	DepHead     int8     // Slot 5, 0x27 — structural logic: DepHead
	IntentFlags uint8    // Slot 9, 0x28 — identity: IntentFlags
	DomainSig   uint16   // Slot 10, 0x29-0x2A — mode: DomainSig (LE)
	GoldenSeed  uint32   // 0x2B-0x2E: nonce target — the solved "Weight"
	Memory      [14]byte // Slots 6-8, 0x2F-0x3C — recursive context: Memory
	LSHSalt     uint32   // Slot 11, 0x3D-0x40 — `(PosIndex << 16) | TemporalSalt` — Contextual Anchor (warm uniqueness)
	Reserved    [15]byte // 0x41-0x4F: metadata — Z3 validation trace / reserved
	Meta        *BracketMeta
	FrameID     string
	FrameUnix   int64
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
	DepHead   int8
}

type IntentDomain struct {
	IntentFlags uint8
	DomainSig   uint16
}
