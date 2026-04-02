package storage

import (
	"encoding/json"
	"time"
)

type KNIRVBASEClient struct {
	seedsCollection string
	rulesCollection string
}

type StoredSeed struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	UserID     string    `json:"user_id"`
	SeedData   []float64 `json:"seed_data"`
	SeedHash   string    `json:"seed_hash"`
	Fitness    float64   `json:"fitness"`
	TrainingID string    `json:"training_id"`
	Trigger    string    `json:"trigger"`
	CreatedAt  time.Time `json:"created_at"`
}

type StoredRule struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	UserID     string    `json:"user_id"`
	RuleType   string    `json:"rule_type"`
	Premises   []string  `json:"premises"`
	Conclusion string    `json:"conclusion"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}

type Seed struct {
	Data    []float64
	Fitness float64
}

type LogicalRule struct {
	RuleType   string
	Premises   []string
	Conclusion string
	Source     string
	Confidence float64
}

func NewKNIRVBASEClient() *KNIRVBASEClient {
	return &KNIRVBASEClient{
		seedsCollection: "hasher_seeds",
		rulesCollection: "hasher_rules",
	}
}

func (c *KNIRVBASEClient) SaveSeeds(userID string, seeds []Seed) error {
	for i, seed := range seeds {
		doc := StoredSeed{
			ID:         fmt.Sprintf("user_%s_seed_%d", userID, i),
			OrgID:      "",
			UserID:     userID,
			SeedData:   seed.Data,
			SeedHash:   fmt.Sprintf("%x", seed.Data),
			Fitness:    seed.Fitness,
			TrainingID: "",
			Trigger:    time.Now().Format(time.RFC3339),
			CreatedAt:  time.Now(),
		}

		if err := c.insertSeed(doc); err != nil {
			return fmt.Errorf("insert seed: %w", err)
		}
	}

	return nil
}

func (c *KNIRVBASEClient) SaveRules(userID string, rules []LogicalRule) error {
	for i, rule := range rules {
		doc := StoredRule{
			ID:         fmt.Sprintf("user_%s_rule_%d", userID, i),
			OrgID:      "",
			UserID:     userID,
			RuleType:   rule.RuleType,
			Premises:   rule.Premises,
			Conclusion: rule.Conclusion,
			Confidence: rule.Confidence,
			Source:     rule.Source,
			CreatedAt:  time.Now(),
		}

		if err := c.insertRule(doc); err != nil {
			return fmt.Errorf("insert rule: %w", err)
		}
	}

	return nil
}

func (c *KNIRVBASEClient) GetUserSeeds(userID string) ([]Seed, error) {
	docs, err := c.findSeedsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("find seeds: %w", err)
	}

	seeds := make([]Seed, len(docs))
	for i, doc := range docs {
		seeds[i] = Seed{
			Data:    doc.SeedData,
			Fitness: doc.Fitness,
		}
	}

	return seeds, nil
}

func (c *KNIRVBASEClient) GetUserRules(userID string) ([]LogicalRule, error) {
	docs, err := c.findRulesByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("find rules: %w", err)
	}

	rules := make([]LogicalRule, len(docs))
	for i, doc := range docs {
		rules[i] = LogicalRule{
			RuleType:   doc.RuleType,
			Premises:   doc.Premises,
			Conclusion: doc.Conclusion,
			Source:     doc.Source,
			Confidence: doc.Confidence,
		}
	}

	return rules, nil
}

func (c *KNIRVBASEClient) insertSeed(doc StoredSeed) error {
	data, _ := json.Marshal(doc)
	return c.insert(c.seedsCollection, doc.ID, string(data))
}

func (c *KNIRVBASEClient) insertRule(doc StoredRule) error {
	data, _ := json.Marshal(doc)
	return c.insert(c.rulesCollection, doc.ID, string(data))
}

func (c *KNIRVBASEClient) findSeedsByUser(userID string) ([]StoredSeed, error) {
	docs, err := c.findAll(c.seedsCollection)
	if err != nil {
		return nil, err
	}

	var seeds []StoredSeed
	for _, doc := range docs {
		var seed StoredSeed
		if err := json.Unmarshal([]byte(doc), &seed); err != nil {
			continue
		}
		if seed.UserID == userID {
			seeds = append(seeds, seed)
		}
	}

	return seeds, nil
}

func (c *KNIRVBASEClient) findRulesByUser(userID string) ([]StoredRule, error) {
	docs, err := c.findAll(c.rulesCollection)
	if err != nil {
		return nil, err
	}

	var rules []StoredRule
	for _, doc := range docs {
		var rule StoredRule
		if err := json.Unmarshal([]byte(doc), &rule); err != nil {
			continue
		}
		if rule.UserID == userID {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

func (c *KNIRVBASEClient) insert(collection, id, data string) error {
	return nil
}

func (c *KNIRVBASEClient) findAll(collection string) ([]string, error) {
	return nil, nil
}

func (c *KNIRVBASEClient) Close() error {
	return nil
}
