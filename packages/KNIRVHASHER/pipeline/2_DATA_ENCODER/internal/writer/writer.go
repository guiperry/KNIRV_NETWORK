package writer

import (
	"context"
	"fmt"

	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

type NRVWriter struct {
	collection knirvbase.Collection
	counter    uint64
}

func NewNRVWriter(collection knirvbase.Collection) *NRVWriter {
	return &NRVWriter{collection: collection}
}

func (w *NRVWriter) WriteBracket(bracket *nrv.Bracket) error {
	if bracket == nil {
		return fmt.Errorf("cannot write nil bracket")
	}
	w.counter++
	proj := make([]byte, len(bracket.Projections))
	copy(proj, bracket.Projections[:])
	_, err := w.collection.Insert(context.Background(), map[string]interface{}{
		"id":            fmt.Sprintf("bracket_%d", w.counter),
		"Projections":   proj,
		"Syntactic":     bracket.Syntactic,
		"DepHead":       bracket.DepHead,
		"IntentFlags":   bracket.IntentFlags,
		"DomainSig":     bracket.DomainSig,
		"Memory":        bracket.Memory[:],
		"GoldenSeed":    bracket.GoldenSeed,
	})
	if err != nil {
		return fmt.Errorf("insert bracket: %w", err)
	}
	return nil
}
