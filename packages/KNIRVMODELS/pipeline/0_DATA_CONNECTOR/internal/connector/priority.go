package connector

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Tier string

const (
	TierOntology    Tier = "ontology"
	TierHuggingFace Tier = "huggingface"
	TierArxiv       Tier = "arxiv"
)

type TierState struct {
	Tier Tier `json:"tier"`
}

type Source interface {
	Fetch() ([]RawRecord, bool, error)
}

// PriorityRunner implements the per-run state machine. Ontology is selected
// when available; HF is consumed until exhausted; arXiv is then consumed.
// State is persisted only after a tier has been exhausted.
type PriorityRunner struct {
	StatePath   string
	Override    Tier
	Ontology    func() ([]RawRecord, bool, error)
	HuggingFace func() ([]RawRecord, bool, error)
	Arxiv       func() ([]RawRecord, bool, error)
}

func (r *PriorityRunner) Run() ([]RawRecord, Tier, error) {
	tier := r.Override
	if tier == "" {
		tier = r.loadState()
	}
	if tier == "" {
		tier = TierOntology
	}
	for _, candidate := range tierOrder(tier) {
		fetch := map[Tier]func() ([]RawRecord, bool, error){TierOntology: r.Ontology, TierHuggingFace: r.HuggingFace, TierArxiv: r.Arxiv}[candidate]
		if fetch == nil {
			continue
		}
		records, exhausted, err := fetch()
		if err != nil {
			if candidate == TierOntology {
				continue
			}
			return nil, candidate, err
		}
		if len(records) > 0 || candidate == TierOntology {
			if exhausted {
				r.saveState(nextTier(candidate))
			} else {
				r.saveState(candidate)
			}
			return records, candidate, nil
		}
		if exhausted {
			r.saveState(nextTier(candidate))
			tier = nextTier(candidate)
			continue
		}
	}
	return nil, TierArxiv, nil
}

func tierOrder(start Tier) []Tier {
	switch start {
	case TierHuggingFace:
		return []Tier{TierHuggingFace, TierArxiv}
	case TierArxiv:
		return []Tier{TierArxiv}
	default:
		return []Tier{TierOntology, TierHuggingFace, TierArxiv}
	}
}

func nextTier(t Tier) Tier {
	switch t {
	case TierOntology:
		return TierHuggingFace
	case TierHuggingFace:
		return TierArxiv
	default:
		return TierArxiv
	}
}
func (r *PriorityRunner) loadState() Tier {
	if r.StatePath == "" {
		return ""
	}
	b, err := os.ReadFile(r.StatePath)
	if err != nil {
		return ""
	}
	var s TierState
	if json.Unmarshal(b, &s) != nil {
		return ""
	}
	return s.Tier
}
func (r *PriorityRunner) saveState(t Tier) {
	if r.StatePath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(r.StatePath), 0755)
	b, _ := json.Marshal(TierState{Tier: t})
	_ = os.WriteFile(r.StatePath, b, 0644)
}

var ErrExhausted = errors.New("source exhausted")
