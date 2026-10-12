package app

import "time"

// SourceType identifies the origin of a training record.
type SourceType string

const (
	SourceTypeClientData   SourceType = "client_data"
	SourceTypeArxiv        SourceType = "arxiv"
	SourceTypeDVETelemetry SourceType = "dve_telemetry"
	SourceTypeUnknown      SourceType = "unknown"
)

// SourceRecord wraps a DocumentRecord with provenance metadata.
type SourceRecord struct {
	Source    SourceType     `json:"source"`
	SourceID  string         `json:"source_id"`
	FetchedAt time.Time      `json:"fetched_at"`
	Record    DocumentRecord `json:"record"`
	Weight    float32        `json:"weight"`
}

// NewSourceRecord constructs a SourceRecord from a DocumentRecord.
func NewSourceRecord(src SourceType, id string, doc DocumentRecord) *SourceRecord {
	return &SourceRecord{
		Source:    src,
		SourceID:  id,
		FetchedAt: time.Now(),
		Record:    doc,
		Weight:    1.0,
	}
}
