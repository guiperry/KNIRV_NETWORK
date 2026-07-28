package main

import (
	"context"
	"testing"

	evo_grpo "github.com/lab/hasher/data-trainer/internal/evo_grpo"
	"github.com/lab/hasher/data-trainer/internal/gates"
	knowledge_base "github.com/lab/hasher/data-trainer/internal/knowledge_base"
	"github.com/lab/hasher/data-trainer/internal/store"
)

// fakeCollection is a minimal in-memory knirvbase.Collection for tests that
// don't need a real distributed backend.
type fakeCollection struct {
	docs map[string]map[string]interface{}
}

func newFakeCollection() *fakeCollection {
	return &fakeCollection{docs: make(map[string]map[string]interface{})}
}

func (f *fakeCollection) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	id, _ := doc["id"].(string)
	f.docs[id] = doc
	return doc, nil
}

func (f *fakeCollection) Update(ctx context.Context, id string, update map[string]interface{}) (int, error) {
	if _, ok := f.docs[id]; !ok {
		return 0, nil
	}
	for k, v := range update {
		f.docs[id][k] = v
	}
	return 1, nil
}

func (f *fakeCollection) Delete(ctx context.Context, id string) (int, error) {
	if _, ok := f.docs[id]; !ok {
		return 0, nil
	}
	delete(f.docs, id)
	return 1, nil
}

func (f *fakeCollection) Find(ctx context.Context, id string) (map[string]interface{}, error) {
	return f.docs[id], nil
}

func (f *fakeCollection) FindAll(ctx context.Context) ([]map[string]interface{}, error) {
	all := make([]map[string]interface{}, 0, len(f.docs))
	for _, d := range f.docs {
		all = append(all, d)
	}
	return all, nil
}

func (f *fakeCollection) AttachToNetwork(networkID string) error { return nil }
func (f *fakeCollection) DetachFromNetwork() error               { return nil }
func (f *fakeCollection) ForceSync() error                       { return nil }

var _ store.Collection = (*fakeCollection)(nil)

func TestUserSecurityGates_Train(t *testing.T) {
	gateTrainer := gates.NewUserSecurityGates(newFakeCollection())
	model, err := gateTrainer.Train("test_user", [][]byte{[]byte("test_data")})
	if err != nil {
		t.Errorf("Train failed: %v", err)
	}
	if model == nil {
		t.Error("Model is nil")
	}
}

func TestEvoGRPO_Optimize(t *testing.T) {
	// Placeholder test
	evoTrainer := &evo_grpo.EvoGRPO{}
	baseModel := &gates.TrainedModel{UserID: "test_user"}
	optimized, err := evoTrainer.Optimize(baseModel)
	if err != nil {
		t.Errorf("Optimize failed: %v", err)
	}
	if optimized == nil {
		t.Error("Optimized model is nil")
	}
}

func TestNRVKnowledgeBase_ReIndex(t *testing.T) {
	kb := knowledge_base.NewNRVKnowledgeBase(newFakeCollection())
	model := &evo_grpo.OptimizedModel{UserID: "test_user"}
	err := kb.ReIndex("test_user", model)
	if err != nil {
		t.Errorf("ReIndex failed: %v", err)
	}
}
