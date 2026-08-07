package processing

import (
	"KNIRVGRAPH/internal/types"
	"regexp"
	"strings"
	"time"
)

var (
	sentenceEnders = regexp.MustCompile(`[.!?]\s+`)
	wordSplitter   = regexp.MustCompile(`\s+`)
)

type Chunker struct {
	config types.ChunkingConfig
}

func NewChunker(config types.ChunkingConfig) *Chunker {
	if config.LengthFunc == nil {
		config.LengthFunc = func(s string) int {
			return len([]rune(s))
		}
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	if config.Overlap < 0 {
		config.Overlap = 0
	}
	if len(config.Separators) == 0 {
		config.Separators = []string{"\n\n", "\n", " ", ""}
	}
	return &Chunker{config: config}
}

func (c *Chunker) Chunk(documentID, text string) ([]types.Chunk, error) {
	switch c.config.Strategy {
	case types.ChunkStrategyToken:
		return c.chunkByToken(documentID, text)
	case types.ChunkStrategySemantic:
		return c.chunkBySemantic(documentID, text)
	default:
		return c.chunkByRecursive(documentID, text)
	}
}

func (c *Chunker) chunkByRecursive(documentID, text string) ([]types.Chunk, error) {
	if c.config.LengthFunc(text) <= c.config.ChunkSize {
		return []types.Chunk{{
			ID:         generateChunkID(documentID, 0),
			DocumentID: documentID,
			Text:       strings.TrimSpace(text),
			Index:      0,
			Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
			CreatedAt:  time.Now(),
		}}, nil
	}

	for _, sep := range c.config.Separators {
		if sep == "" {
			// Hard split by character count
			return c.chunkBySize(documentID, text), nil
		}
		parts := strings.Split(text, sep)
		if len(parts) <= 1 {
			continue
		}
		var chunks []types.Chunk
		var current strings.Builder
		idx := 0
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			candidate := strings.TrimSpace(current.String() + sep + part)
			if c.config.LengthFunc(candidate) > c.config.ChunkSize && current.Len() > 0 {
				chunks = append(chunks, types.Chunk{
					ID:         generateChunkID(documentID, idx),
					DocumentID: documentID,
					Text:       strings.TrimSpace(current.String()),
					Index:      idx,
					Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
					CreatedAt:  time.Now(),
				})
				idx++
				current.Reset()
				if c.config.Overlap > 0 {
					overlapText := getLastN(current.String(), c.config.Overlap)
					current.WriteString(overlapText)
					if sep != "" {
						current.WriteString(sep)
					}
				}
			}
			if current.Len() > 0 {
				current.WriteString(sep)
			}
			current.WriteString(part)
		}
		if current.Len() > 0 {
			chunks = append(chunks, types.Chunk{
				ID:         generateChunkID(documentID, idx),
				DocumentID: documentID,
				Text:       strings.TrimSpace(current.String()),
				Index:      idx,
				Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
				CreatedAt:  time.Now(),
			})
		}
		if len(chunks) > 0 {
			for i := range chunks {
				chunks[i].StartOffset = computeStartOffset(text, chunks[i].Text)
				chunks[i].EndOffset = chunks[i].StartOffset + len([]rune(chunks[i].Text))
			}
			return chunks, nil
		}
	}
	return c.chunkBySize(documentID, text), nil
}

func (c *Chunker) chunkByToken(documentID, text string) ([]types.Chunk, error) {
	tokens := wordSplitter.Split(strings.TrimSpace(text), -1)
	if len(tokens) == 0 {
		return []types.Chunk{}, nil
	}
	tokenSize := c.config.ChunkSize
	if tokenSize > len(tokens) {
		tokenSize = len(tokens)
	}
	overlap := c.config.Overlap
	var chunks []types.Chunk
	idx := 0
	for start := 0; start < len(tokens); start += tokenSize - overlap {
		if start > 0 && start+tokenSize > len(tokens) {
			break
		}
		end := start + tokenSize
		if end > len(tokens) {
			end = len(tokens)
		}
		chunkText := strings.Join(tokens[start:end], " ")
		chunks = append(chunks, types.Chunk{
			ID:         generateChunkID(documentID, idx),
			DocumentID: documentID,
			Text:       chunkText,
			Index:      idx,
			Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy), "tokens": end - start},
			CreatedAt:  time.Now(),
		})
		idx++
		if end == len(tokens) {
			break
		}
	}
	return chunks, nil
}

func (c *Chunker) chunkBySemantic(documentID, text string) ([]types.Chunk, error) {
	sentences := sentenceEnders.Split(text, -1)
	var chunks []types.Chunk
	var current strings.Builder
	idx := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		candidate := strings.TrimSpace(current.String() + ". " + s)
		if c.config.LengthFunc(candidate) > c.config.ChunkSize && current.Len() > 0 {
			chunks = append(chunks, types.Chunk{
				ID:         generateChunkID(documentID, idx),
				DocumentID: documentID,
				Text:       strings.TrimSpace(current.String()),
				Index:      idx,
				Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
				CreatedAt:  time.Now(),
			})
			idx++
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(". ")
		}
		current.WriteString(s)
	}
	if current.Len() > 0 {
		chunks = append(chunks, types.Chunk{
			ID:         generateChunkID(documentID, idx),
			DocumentID: documentID,
			Text:       strings.TrimSpace(current.String()),
			Index:      idx,
			Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
			CreatedAt:  time.Now(),
		})
	}
	if len(chunks) == 0 {
		return []types.Chunk{{
			ID:         generateChunkID(documentID, 0),
			DocumentID: documentID,
			Text:       strings.TrimSpace(text),
			Index:      0,
			Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
			CreatedAt:  time.Now(),
		}}, nil
	}
	return chunks, nil
}

func (c *Chunker) chunkBySize(documentID, text string) []types.Chunk {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return []types.Chunk{}
	}
	size := c.config.ChunkSize
	overlap := c.config.Overlap
	var chunks []types.Chunk
	idx := 0
	for start := 0; start < len(runes); start += size - overlap {
		if start > 0 && start >= len(runes) {
			break
		}
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunkText := string(runes[start:end])
		chunks = append(chunks, types.Chunk{
			ID:         generateChunkID(documentID, idx),
			DocumentID: documentID,
			Text:       chunkText,
			Index:      idx,
			StartOffset: start,
			EndOffset:   end,
			Metadata:   map[string]interface{}{"strategy": string(c.config.Strategy)},
			CreatedAt:  time.Now(),
		})
		idx++
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func generateChunkID(documentID string, index int) string {
	return documentID + "_chunk_" + itoa(index)
}

func getLastN(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}

func computeStartOffset(fullText, chunkText string) int {
	idx := strings.Index(fullText, chunkText)
	if idx >= 0 {
		return idx
	}
	return 0
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
