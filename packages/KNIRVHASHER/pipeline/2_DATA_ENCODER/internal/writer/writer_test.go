package writer

import (
	"context"
	"fmt"
	"testing"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

type mockCollection struct {
	inserted []map[string]interface{}
}

func (m *mockCollection) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	m.inserted = append(m.inserted, doc)
	return doc, nil
}
func (m *mockCollection) Update(_ context.Context, _ string, _ map[string]interface{}) (int, error) {
	return 0, nil
}
func (m *mockCollection) Delete(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockCollection) Find(_ context.Context, _ string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockCollection) FindAll(_ context.Context) ([]map[string]interface{}, error) {
	return m.inserted, nil
}
func (m *mockCollection) AttachToNetwork(_ string) error { return nil }
func (m *mockCollection) DetachFromNetwork() error      { return nil }
func (m *mockCollection) ForceSync() error              { return nil }

func TestNRVWriterWriteBracket(t *testing.T) {
	mc := &mockCollection{}
	w := NewNRVWriter(mc)

	bracket := &nrv.Bracket{
		Projections: [32]byte{1, 2, 3},
		Syntactic:   0x42,
		DepHead:     5,
		IntentFlags: 0x01,
		DomainSig:   0x2000,
		Memory:      [14]byte{10, 20, 30},
		GoldenSeed:  12345,
	}

	err := w.WriteBracket(bracket)
	if err != nil {
		t.Fatalf("WriteBracket failed: %v", err)
	}
	if len(mc.inserted) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(mc.inserted))
	}
}

func TestNRVWriterWriteBracket_Nil(t *testing.T) {
	mc := &mockCollection{}
	w := NewNRVWriter(mc)

	err := w.WriteBracket(nil)
	if err == nil {
		t.Fatal("expected error for nil bracket, got nil")
	}
}

func TestNewNRVWriter_Interface(t *testing.T) {
	mc := &mockCollection{}
	w := NewNRVWriter(mc)
	if w == nil {
		t.Fatal("NewNRVWriter returned nil")
	}
	if w.collection != mc {
		t.Fatal("NewNRVWriter did not store collection")
	}
}
