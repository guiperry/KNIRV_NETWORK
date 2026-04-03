package normalizer

type SecurityRecord struct {
	FileName     string   `json:"file_name" arrow:"file_name"`
	ChunkID      int32    `json:"chunk_id" arrow:"chunk_id"`
	Content      string   `json:"content" arrow:"content"`
	Tokens       []string `json:"tokens" arrow:"tokens"`
	POSTags      []int    `json:"pos_tags" arrow:"pos_tags"`
	DepHashes    []uint32 `json:"dep_hashes" arrow:"dep_hashes"`
	SecurityTags []string `json:"security_tags" arrow:"security_tags"`
	Slot4        int      `json:"slot4" arrow:"slot4"`
	Slot10       int      `json:"slot10" arrow:"slot10"`
	Weight       float64  `json:"weight" arrow:"weight"`
}

func (r *SecurityRecord) GetEmbedding() []float32 {
	embedding := make([]float32, 128)
	for i, h := range r.DepHashes {
		if i >= 128 {
			break
		}
		embedding[i] = float32(h) / float32(0xFFFFFFFF)
	}
	for i := len(r.DepHashes); i < 128; i++ {
		embedding[i] = 0.0
	}
	return embedding
}
