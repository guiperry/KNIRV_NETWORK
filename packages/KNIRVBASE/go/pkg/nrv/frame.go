package nrv

import "github.com/knirvcorp/knirvbase/go/internal/clock"

type ModalityIndex struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

type FrameEntry struct {
	ID         string                         `json:"id"`
	Offset     int64                          `json:"offset"`
	Length     int                            `json:"length"`
	Tombstone  *int64                         `json:"tombstone"`
	Verified   bool                           `json:"verified"`
	ERGORank   float64                        `json:"ergo_rank"`
	Modalities map[ModalityType]ModalityIndex `json:"modalities"`
}

type GlobalMetrics struct {
	FeatureMin                   [12]float32 `json:"feature_min"`
	FeatureMax                   [12]float32 `json:"feature_max"`
	FeatureMean                  [12]float32 `json:"feature_mean"`
	FeatureStd                   [12]float32 `json:"feature_std"`
	ThermoCorrelationCoefficient float64     `json:"thermo_correlation_coefficient"`
	ERGORankSum                  float64     `json:"ergo_rank_sum"`
	VerifiedFrameCount           int         `json:"verified_frame_count"`
	CompactedAt                  *string     `json:"compacted_at"`
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

type Frame struct {
	ID     string
	Vector [12]float32
	Seed   [32]byte
	Thermo ThermoData
	Proof  []byte
}

type ThermoData struct {
	TempCelsius float32
	VoltageV    float32
	FreqMHz     float32
	FanRPM      float32
}
