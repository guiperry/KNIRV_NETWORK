// Package tokenizer holds the text<->token encoders used by the HANDS GPT and
// any other model that needs a plain string tokenizer. It was extracted from
// pkg/hashing/transformer/gpt.go during the models migration (Phase 1) so the
// inner GPT lives under pkg/hands without dragging the HEART/HASHER transformer
// code along.
package tokenizer

import (
	"encoding/json"
	"math/rand"
	"os"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

// Tokenizer is the minimal string<->token contract the GPT trains and generates with.
type Tokenizer interface {
	Encode(text string) []int
	Decode(tokens []int) string
}

// BPETokenizer is a vocab-file-backed tokenizer (word-level BPE). It is the
// offline fallback when no tiktoken model is available.
type BPETokenizer struct {
	vocab     map[string]int
	invVocab  map[int]string
	vocabSize int
}

// NewBPETokenizer loads a JSON {word: id} vocab. An empty path yields an empty
// vocab (Encode maps every word to <UNK>), which is acceptable for a model that
// is only ever trained with a real tokenizer supplied by the caller.
func NewBPETokenizer(vocabPath string) *BPETokenizer {
	vocab := loadVocab(vocabPath)

	invVocab := make(map[int]string, len(vocab))
	for token, idx := range vocab {
		invVocab[idx] = token
	}

	return &BPETokenizer{
		vocab:     vocab,
		invVocab:  invVocab,
		vocabSize: len(vocab),
	}
}

// VocabSize reports how many tokens the vocab knows.
func (t *BPETokenizer) VocabSize() int { return t.vocabSize }

// Encode maps whitespace-separated words to their vocab ids (unknown -> <UNK>).
func (t *BPETokenizer) Encode(text string) []int {
	tokens := make([]int, 0)

	words := strings.Fields(text)
	for _, word := range words {
		if idx, exists := t.vocab[word]; exists {
			tokens = append(tokens, idx)
		} else {
			tokens = append(tokens, t.vocab["<UNK>"])
		}
	}

	return tokens
}

// Decode maps ids back to words, joining with spaces.
func (t *BPETokenizer) Decode(tokens []int) string {
	words := make([]string, len(tokens))
	for i, tok := range tokens {
		if word, exists := t.invVocab[tok]; exists {
			words[i] = word
		} else {
			words[i] = "<UNK>"
		}
	}
	return strings.Join(words, " ")
}

func loadVocab(path string) map[string]int {
	vocab := make(map[string]int)
	if path == "" {
		return vocab
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return vocab
	}
	_ = json.Unmarshal(data, &vocab)
	return vocab
}

// TiktokenTokenizer wraps a tiktoken encoding (default cl100k_base).
type TiktokenTokenizer struct {
	enc *tiktoken.Tiktoken
}

// NewTiktokenTokenizer creates a tokenizer using the given tiktoken encoding (default: cl100k_base).
func NewTiktokenTokenizer(encoding string) (*TiktokenTokenizer, error) {
	if encoding == "" {
		encoding = "cl100k_base"
	}
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, err
	}
	return &TiktokenTokenizer{enc: enc}, nil
}

// Encode tokenizes text with tiktoken.
func (t *TiktokenTokenizer) Encode(text string) []int {
	return t.enc.Encode(text, nil, nil)
}

// Decode detokenizes tiktoken ids back to text.
func (t *TiktokenTokenizer) Decode(tokens []int) string {
	return t.enc.Decode(tokens)
}

// Compile-time guard that rand is referenced (kept for parity with the original
// sampler which consumes package-level randomness).
var _ = rand.Intn
