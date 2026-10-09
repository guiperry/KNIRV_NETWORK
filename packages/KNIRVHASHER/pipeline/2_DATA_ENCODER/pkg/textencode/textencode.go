// Package textencode turns one text record into NRV brackets the way the
// data-encoder CLI does (tiktoken cl100k → sliding windows → embeddings →
// 12-slot frame → bracket), for callers that encode on demand, such as
// backend_server's arena dataset endpoints.
//
// Differences from the CLI pipeline: there is no spaCy metadata (POS, tense
// and dependency slots are zero) and no variance file (signal indices are the
// CLI's spread fallback). Brackets from this package are therefore marked with
// the embedder's model id by the caller so training can tell them apart.
package textencode

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"data-encoder/pkg/embeddings"
	"data-encoder/pkg/nrvio"
	"data-encoder/pkg/sliding"
	"data-encoder/pkg/slotpack"
	"data-encoder/pkg/tokenizer"
)

const (
	// DefaultWindowSize and DefaultStride match the CLI defaults.
	DefaultWindowSize = 128
	DefaultStride     = 1
	// DefaultMaxTokens bounds one record.
	DefaultMaxTokens = 4096
)

// Record is one text unit to encode.
type Record struct {
	// Instruction and Input drive the intent and domain slots, as the CLI's
	// heading/instruction and input fields do.
	Instruction string
	Input       string
	// Content is the text that is tokenized and windowed.
	Content string
}

// Encoder encodes records. It is safe for concurrent use.
type Encoder struct {
	Embedder   embeddings.EmbeddingService
	WindowSize int
	Stride     int
	MaxTokens  int

	once    sync.Once
	tk      *tokenizer.Service
	tkErr   error
	indices []int
}

// New returns an encoder using the deterministic hash-ngram embedder, the
// data-encoder's default backend (no network calls).
func New() *Encoder {
	return &Encoder{Embedder: embeddings.NewDeterministicService()}
}

// ErrTooLong is returned when a record exceeds MaxTokens.
var ErrTooLong = errors.New("record is too long to encode")

func (e *Encoder) init() {
	e.once.Do(func() {
		e.tk, e.tkErr = tokenizer.New()
		e.indices = make([]int, 24)
		for i := range e.indices {
			e.indices[i] = i * 32
		}
	})
}

func orDefault(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// Encode returns one bracket per sliding window of rec.Content.
func (e *Encoder) Encode(rec Record) ([]nrvio.Bracket, error) {
	e.init()
	if e.tkErr != nil {
		return nil, fmt.Errorf("tokenizer unavailable: %w", e.tkErr)
	}
	if e.Embedder == nil {
		return nil, errors.New("no embedder configured")
	}
	tokens := e.tk.Encode(rec.Content)
	if len(tokens) > orDefault(e.MaxTokens, DefaultMaxTokens) {
		return nil, fmt.Errorf("%w: %d tokens (max %d)", ErrTooLong, len(tokens), orDefault(e.MaxTokens, DefaultMaxTokens))
	}
	if len(tokens) < 2 {
		return nil, errors.New("record needs at least two tokens")
	}
	windows := sliding.NewGenerator(orDefault(e.WindowSize, DefaultWindowSize), orDefault(e.Stride, DefaultStride)).GenerateWindows(tokens)
	if len(windows) == 0 {
		return nil, errors.New("record produced no windows")
	}

	instruction := strings.TrimSpace(rec.Instruction)
	batch := orDefault(e.Embedder.GetBatchSize(), embeddings.DefaultBatchSize)
	memory := [3]uint32{0x5F3759DF, 0x12345678, 0x87654321} // the CLI's initial seeds
	out := make([]nrvio.Bracket, 0, len(windows))
	for start := 0; start < len(windows); start += batch {
		end := start + batch
		if end > len(windows) {
			end = len(windows)
		}
		texts := make([]string, end-start)
		for i, w := range windows[start:end] {
			texts[i] = e.tk.Decode(w.ContextTokens)
		}
		vectors, err := e.Embedder.GetBatchEmbeddings(texts)
		if err != nil {
			return nil, fmt.Errorf("embed windows: %w", err)
		}
		if len(vectors) != len(texts) {
			return nil, fmt.Errorf("embedder returned %d vectors for %d windows", len(vectors), len(texts))
		}
		for i, w := range windows[start:end] {
			if len(vectors[i]) <= e.indices[3] {
				return nil, fmt.Errorf("embedding has %d dimensions; need more than %d", len(vectors[i]), e.indices[3])
			}
			slots := slotpack.PackFrame(e.indices, nil, vectors[i], 0, 0, 0, memory[:], uint16(w.EndPos), instruction, rec.Input)
			memory[0] ^= slots[0]
			memory[1] ^= slots[1]
			memory[2] ^= slots[2]
			out = append(out, slotpack.Bracket(slots, int32(w.TargetToken), int32(w.StartPos)))
		}
	}
	return out, nil
}
