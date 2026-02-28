package icme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"backend_server/internal/database"
)

type FactualityAdapter struct {
	validationEndpoint string
	httpClient         *http.Client
	intentRegistry     *IntentRegistry
	db                 *database.BuntDBManager
	logger             *zap.Logger
}

type FactualityRequest struct {
	Prompt            string             `json:"prompt"`
	Response          string             `json:"response"`
	AgentID           string             `json:"agent_id"`
	DVEID             string             `json:"dve_id"`
	OntologyDomains   []string           `json:"ontology_domains"`
	ObjectiveName     string             `json:"objective_name"`
	PreferenceWeights map[string]float64 `json:"preference_weights"`
}

type FactualityResponse struct {
	IsAccurate   bool               `json:"is_accurate"`
	Confidence   float64            `json:"confidence"`
	Citations    []int              `json:"citations"`
	Refused      bool               `json:"refused"`
	Explanation  string             `json:"explanation"`
	DomainScores map[string]float64 `json:"domain_scores"`
}

func NewFactualityAdapter(
	validationEndpoint string,
	intentRegistry *IntentRegistry,
	db *database.BuntDBManager,
	logger *zap.Logger,
) *FactualityAdapter {
	return &FactualityAdapter{
		validationEndpoint: validationEndpoint,
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		intentRegistry:     intentRegistry,
		db:                 db,
		logger:             logger,
	}
}

func (f *FactualityAdapter) ValidateFactuality(ctx context.Context, signal *IntentionalSignal) (*FactualityResponse, error) {
	obj := f.intentRegistry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)

	ontologyDomains := f.getOntologyDomains(obj)
	preferenceWeights := f.getPreferenceWeights(signal.AgentID, signal.DVEID)

	req := FactualityRequest{
		Prompt:            signal.Content,
		Response:          signal.Content,
		AgentID:           signal.AgentID,
		DVEID:             signal.DVEID,
		OntologyDomains:   ontologyDomains,
		ObjectiveName:     signal.ObjectiveName,
		PreferenceWeights: preferenceWeights,
	}

	jsonData, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", f.validationEndpoint+"/api/validation/factuality", strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return f.defaultResponse(), fmt.Errorf("factuality validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return f.defaultResponse(), fmt.Errorf("factuality validation returned %d", resp.StatusCode)
	}

	var factResp FactualityResponse
	if err := json.NewDecoder(resp.Body).Decode(&factResp); err != nil {
		return f.defaultResponse(), fmt.Errorf("decode factuality response: %w", err)
	}

	f.logger.Debug("factuality validated",
		zap.String("agent_id", signal.AgentID),
		zap.Float64("confidence", factResp.Confidence),
		zap.Bool("is_accurate", factResp.IsAccurate),
	)

	return &factResp, nil
}

func (f *FactualityAdapter) defaultResponse() *FactualityResponse {
	return &FactualityResponse{
		IsAccurate:   true,
		Confidence:   0.75,
		DomainScores: map[string]float64{"general": 0.75},
	}
}

func (f *FactualityAdapter) getOntologyDomains(obj *IntentObjective) []string {
	if obj == nil {
		return []string{"general"}
	}

	domains := make([]string, 0, len(obj.DataSources))
	for _, ds := range obj.DataSources {
		domain := extractDomain(ds)
		if domain != "" {
			domains = append(domains, domain)
		}
	}

	if len(domains) == 0 {
		domains = []string{"general"}
	}
	return domains
}

func (f *FactualityAdapter) getPreferenceWeights(agentID, dveID string) map[string]float64 {
	weights := make(map[string]float64)

	prefKey := fmt.Sprintf("icme:preferences:%s:%s", dveID, agentID)
	var prefData []byte
	if err := f.db.GetJSON(prefKey, &prefData); err == nil {
		json.Unmarshal(prefData, &weights)
	}

	if len(weights) == 0 {
		weights = map[string]float64{
			"accuracy":  0.4,
			"relevance": 0.3,
			"coherence": 0.2,
			"safety":    0.1,
		}
	}
	return weights
}

func extractDomain(dataSource string) string {
	domainKeywords := map[string][]string{
		"technical":  {"api", "code", "sdk", "implementation"},
		"scientific": {"research", "study", "experiment", "data"},
		"historical": {"history", "event", "timeline", "past"},
		"financial":  {"finance", "money", "market", "investment"},
		"medical":    {"health", "medical", "clinical", "patient"},
	}

	lower := strings.ToLower(dataSource)
	for domain, keywords := range domainKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return domain
			}
		}
	}
	return "general"
}

func (f *FactualityAdapter) ComputeAlignmentScore(ctx context.Context, signal *IntentionalSignal) (float64, error) {
	factResp, err := f.ValidateFactuality(ctx, signal)
	if err != nil {
		return 0.5, err
	}

	obj := f.intentRegistry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)
	if obj == nil {
		return factResp.Confidence, nil
	}

	baseScore := factResp.Confidence

	domainPenalty := 0.0
	for domain, score := range factResp.DomainScores {
		expected, ok := obj.TradeOffs[domain]
		if ok {
			domainPenalty += (expected - score) * 0.1
		}
	}

	preferenceWeights := f.getPreferenceWeights(signal.AgentID, signal.DVEID)
	preferenceBoost := 0.0
	for pref, weight := range preferenceWeights {
		if strings.Contains(factResp.Explanation, pref) {
			preferenceBoost += weight * 0.05
		}
	}

	alignScore := baseScore + preferenceBoost + domainPenalty

	if alignScore < 0 {
		alignScore = 0
	}
	if alignScore > 1 {
		alignScore = 1
	}

	return alignScore, nil
}
