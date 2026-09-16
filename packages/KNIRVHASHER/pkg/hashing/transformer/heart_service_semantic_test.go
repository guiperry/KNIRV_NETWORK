package transformer

import (
	"testing"

	"github.com/guiperry/text-embedder/pkg/embed"
	"knirvhasher/pkg/hashing/semanticmemory"
)

type semanticTestTokenizer struct{}

func (semanticTestTokenizer) Encode(string) []int { return nil }
func (semanticTestTokenizer) Decode([]int) string { return "a context the semantic memory has seen" }

func TestGorgoniteInferenceUsesSemanticMemory(t *testing.T) {
	memory, err := semanticmemory.New(embed.Dims, 4, embed.ModelID)
	if err != nil {
		t.Fatalf("create semantic memory: %v", err)
	}
	const target = int32(7)
	if err := memory.Update(embed.Embed("a context the semantic memory has seen"), target); err != nil {
		t.Fatalf("update semantic memory: %v", err)
	}

	hs := &HEARTService{
		semanticMemory: memory,
		tokenizer:      semanticTestTokenizer{},
		config:         &HEARTConfig{Gorgonite: GorgoniteConfig{VocabSize: 16}},
	}
	logits, err := hs.runGorgoniteInference([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("semantic inference: %v", err)
	}
	if len(logits) != 16 {
		t.Fatalf("logits length = %d, want 16", len(logits))
	}
	if logits[target] <= 1 {
		t.Fatalf("target logit = %v, want learned semantic prediction", logits[target])
	}
}
