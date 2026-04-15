package cleaner

import (
	"crypto/sha256"
	"encoding/hex"

	"knirvhasher/pipeline/0_DATA_CONNECTOR/internal/normalizer"
)

type DataCleaner struct {
	deduplicate bool
	seen        map[string]bool
}

func NewDataCleaner(deduplicate bool) *DataCleaner {
	return &DataCleaner{
		deduplicate: deduplicate,
		seen:        make(map[string]bool),
	}
}

func (c *DataCleaner) Clean(records []*normalizer.SecurityRecord) []*normalizer.SecurityRecord {
	if !c.deduplicate {
		return records
	}

	cleaned := make([]*normalizer.SecurityRecord, 0, len(records))
	for _, r := range records {
		hash := c.hashRecord(r)
		if !c.seen[hash] {
			c.seen[hash] = true
			cleaned = append(cleaned, r)
		}
	}
	return cleaned
}

func (c *DataCleaner) hashRecord(r *normalizer.SecurityRecord) string {
	h := sha256.New()
	h.Write([]byte(r.Content))
	h.Write([]byte(r.FileName))
	for _, tag := range r.SecurityTags {
		h.Write([]byte(tag))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (c *DataCleaner) Reset() {
	c.seen = make(map[string]bool)
}

func (c *DataCleaner) Count() int {
	return len(c.seen)
}
