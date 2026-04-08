package nrv

import "github.com/knirvcorp/knirvbase/go/internal/clock"

type ModalityIndex struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

type FrameEntry struct {
	ID            string            `json:"id"`
	TimestampUnix int64             `json:"timestamp_unix"`
	Tombstone     *int64            `json:"tombstone"`
	Linguistic    LinguisticMapping `json:"linguistic"`
	Thermo        ThermoAtmosphere  `json:"thermo"`
	Z3            Z3Result          `json:"z3"`
	Brackets      BracketBinaryMap  `json:"brackets"`
	BracketIndex  []BracketMeta     `json:"bracket_index"`
}

type GlobalMetrics struct {
	AvgTempCMean      float32 `json:"avg_temp_c_mean"`
	AvgTempCMax       float32 `json:"avg_temp_c_max"`
	PeakVoltVMean     float32 `json:"peak_volt_v_mean"`
	ClockMHzMean      float32 `json:"clock_mhz_mean"`
	TotalBracketCount int     `json:"total_bracket_count"`
	ValidFrameCount   int     `json:"valid_frame_count"`
	InvalidFrameCount int     `json:"invalid_frame_count"`
	CompactedAt       *string `json:"compacted_at"`
}

type PQCManifest struct {
	KeyID           string            `json:"key_id"`
	Algorithm       string            `json:"algorithm"`
	FileSignature   string            `json:"file_signature"`
	FrameSignatures map[string]string `json:"frame_signatures"`
}

type Registry struct {
	Version        int               `json:"version"`
	DatasetID      string            `json:"dataset_id"`
	DatasetVersion clock.VectorClock `json:"dataset_version"`
	Chunk0Length   int               `json:"chunk0_length"`
	FrameCount     int               `json:"frame_count"`
	TombstoneCount int               `json:"tombstone_count"`
	GlobalMetrics  GlobalMetrics     `json:"global_metrics"`
	Frames         []FrameEntry      `json:"frames"`
	PQCManifest    PQCManifest       `json:"pqc_manifest"`
}

type ThermoData struct {
	TempCelsius float32
	VoltageV    float32
	FreqMHz     float32
	FanRPM      float32
}
