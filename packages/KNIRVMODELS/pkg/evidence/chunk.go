package evidence

import "unicode"

// Span is a byte range of a normalized snapshot.
type Span struct{ Start, End int }

// DefaultChunkWords is the window size. The plan says 512-token chunks; English
// averages about 0.75 words per token, so 384 words is about 512 tokens. This is an
// estimate made without the model's tokenizer: the index header records the value
// used so a different window is a different index.
const (
	DefaultChunkWords   = 384
	DefaultChunkOverlap = 48
)

// ChunkWords splits text into windows of at most size words, each starting overlap
// words before the previous one ended, always on word boundaries. Text shorter than
// a window is one span. Offsets are bytes into text.
func ChunkWords(text string, size, overlap int) []Span {
	if size <= 0 {
		size = DefaultChunkWords
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	type word struct{ s, e int }
	var words []word
	start := -1
	for i, r := range text {
		if unicode.IsSpace(r) {
			if start >= 0 {
				words = append(words, word{start, i})
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, word{start, len(text)})
	}
	if len(words) == 0 {
		return nil
	}
	var out []Span
	for i := 0; i < len(words); {
		j := i + size
		if j > len(words) {
			j = len(words)
		}
		out = append(out, Span{words[i].s, words[j-1].e})
		if j == len(words) {
			break
		}
		i = j - overlap
	}
	return out
}
