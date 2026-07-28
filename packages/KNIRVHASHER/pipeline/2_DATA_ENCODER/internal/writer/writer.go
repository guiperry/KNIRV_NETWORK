package writer

import (
	"context"
	"fmt"

	"data-encoder/pkg/nrvio"
	"data-encoder/pkg/store"
)

type NRVWriter struct {
	collection store.Collection
	counter    uint64
}

func NewNRVWriter(collection store.Collection) *NRVWriter {
	return &NRVWriter{collection: collection}
}

func (w *NRVWriter) WriteBracket(bracket *nrvio.Bracket) error {
	if bracket == nil {
		return fmt.Errorf("cannot write nil bracket")
	}
	w.counter++
	proj := make([]byte, len(bracket.Projections))
	copy(proj, bracket.Projections[:])
	_, err := w.collection.Insert(context.Background(), map[string]interface{}{
		"id":          fmt.Sprintf("bracket_%d", w.counter),
		"Projections": proj,
		"Syntactic":   bracket.POSTag,
		"DepHead":     bracket.DepHead,
		"IntentFlags": bracket.IntentFlags,
		"DomainSig":   bracket.DomainSig,
		"Memory":      bracket.Memory[:],
		"GoldenSeed":  bracket.GoldenSeed,
	})
	if err != nil {
		return fmt.Errorf("insert bracket: %w", err)
	}
	return nil
}
