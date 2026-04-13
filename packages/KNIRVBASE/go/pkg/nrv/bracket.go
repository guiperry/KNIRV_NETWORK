package nrv

type DeltaType string

const (
	DeltaTypeI DeltaType = "I"
	DeltaTypeP DeltaType = "P"
)

const BracketSize = 80

type Bracket struct {
	ID          string
	Projections [32]byte // 0x00-0x1F: slots 0-3 — semantic compass / LSH projections (16-dim uint16 encoded)
	SubSecondUS uint32   // 0x20-0x23: metadata — SubSecondUS ticker
	Syntactic   uint8    // Slot 4 (bit-packed): bits 0-3 POSTag, bits 4-5 Tense, bits 6-7 Plurality
	DepHead     int8     // Slot 5, 0x25 — structural logic: DepHead
	IntentFlags uint8    // Slot 9, 0x26 — identity: IntentFlags
	DomainSig   uint16   // Slot 10, 0x27-0x28 — mode: DomainSig (LE)
	GoldenSeed  uint32   // 0x29-0x2C: nonce target — the solved "Weight"
	Memory      [14]byte // Slots 6-8, 0x2D-0x3A — recursive context: Memory
	LSHSalt     uint32   // Slot 11, 0x3B-0x3E — `(PosIndex << 16) | TemporalSalt` — Contextual Anchor (warm uniqueness)
	Reserved    [17]byte // 0x3F-0x4F: metadata — Z3 validation trace / reserved
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
	Syntactic uint8 // Slot 4 (bit-packed): bits 0-3 POSTag, bits 4-5 Tense, bits 6-7 Plurality
	DepHead   int8  // Slot 5: Dependency Head index (1 byte, -128 to 127)
}

type IntentDomain struct {
	IntentFlags uint8
	DomainSig   uint16
}

func (b *Bracket) GetPOSTag() uint8 {
	return b.Syntactic & 0x0F
}

func (b *Bracket) SetPOSTag(v uint8) {
	b.Syntactic = (b.Syntactic & 0xF0) | (v & 0x0F)
}

func (b *Bracket) GetTense() uint8 {
	return (b.Syntactic >> 4) & 0x03
}

func (b *Bracket) SetTense(v uint8) {
	b.Syntactic = (b.Syntactic & 0xCF) | ((v & 0x03) << 4)
}

func (b *Bracket) GetPlurality() uint8 {
	return (b.Syntactic >> 6) & 0x03
}

func (b *Bracket) SetPlurality(v uint8) {
	b.Syntactic = (b.Syntactic & 0x3F) | ((v & 0x03) << 6)
}
