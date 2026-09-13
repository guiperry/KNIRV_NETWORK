package training

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CandidateScorer is an optional, non-authoritative ranking signal. A scorer
// may be backed by knirvllama, but it never participates in IsWinningSeed.
type CandidateScorer interface {
	Score(seed []byte, record *TrainingRecord) (float64, error)
}

// LlamaCandidateScorer asks the local OpenAI-compatible runtime for a bounded
// consistency score. It is deliberately optional: transport/model errors are
// ignored by EvaluatePopulationBatch and never affect proof acceptance.
type LlamaCandidateScorer struct {
	endpoint, model string
	client          *http.Client
}

func NewLlamaCandidateScorer(endpoint, model string) *LlamaCandidateScorer {
	return &LlamaCandidateScorer{endpoint: strings.TrimRight(endpoint, "/"), model: model, client: &http.Client{Timeout: 10 * time.Second}}
}

func (s *LlamaCandidateScorer) Score(seed []byte, record *TrainingRecord) (float64, error) {
	if s.endpoint == "" {
		return 0, fmt.Errorf("llama endpoint not configured")
	}
	prompt := fmt.Sprintf("Score the internal consistency of this proposed KNIRV assertion from 0 to 1. Return only a decimal number. context=%v assertion=%v candidate=%s", record.TokenSequence, record.Span(), hex.EncodeToString(seed))
	body, err := json.Marshal(map[string]interface{}{"model": s.model, "temperature": 0, "messages": []map[string]string{{"role": "user", "content": prompt}}})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, s.endpoint+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("llama status %d", resp.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, err
	}
	if len(decoded.Choices) == 0 {
		return 0, fmt.Errorf("llama returned no choices")
	}
	score, err := strconv.ParseFloat(strings.TrimSpace(decoded.Choices[0].Message.Content), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid llama score: %w", err)
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}

// SetCandidateScorer installs an advisory scorer. Its values only break ties
// among candidates that have already been deterministically evaluated.
func (eh *EvolutionaryHarness) SetCandidateScorer(scorer CandidateScorer, weight float64) {
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}
	eh.candidateScorer, eh.candidateScoreWeight = scorer, weight
}
