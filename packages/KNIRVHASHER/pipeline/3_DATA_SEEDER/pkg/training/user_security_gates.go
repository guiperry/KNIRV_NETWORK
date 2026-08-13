package training

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	trainer "github.com/lab/hasher/data-seeder/pkg/trainer"
)

type UserSecurityGates struct {
	OrgID          string
	UserID         string
	Constraints    []SecurityConstraint
	Patterns       []BehaviorPattern
	MaxGenerations int
	PopulationSize int
	MutationRate   float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SecurityConstraint struct {
	RuleID   string
	Text     string
	Type     string
	Severity string
	Tags     []string
	Weight   float64
}

type BehaviorPattern struct {
	TaskType   string
	Success    bool
	Metrics    map[string]float64
	ActionHash uint64
}

type LogicalRule struct {
	RuleType   string
	Premises   []string
	Conclusion string
	Source     string
	Confidence float64
}

type Seed struct {
	Data    []float64
	Fitness float64
	Hash    []byte
}

type TrainedGates struct {
	Seeds      []Seed
	Rules      []LogicalRule
	OrgID      string
	UserID     string
	Fitness    float64
	TrainedAt  time.Time
	TrainingID string
	Trigger    string
}

type SecurityFitness struct {
	Alignment        float64
	Stability        float64
	Format           float64
	ConstraintScore  float64
	ViolationPenalty float64
}

func (f *SecurityFitness) Total() float64 {
	return f.Alignment*0.25 +
		f.Stability*0.20 +
		f.Format*0.15 +
		f.ConstraintScore*0.25 +
		f.ViolationPenalty*0.15
}

type UserTrainer struct {
	baseNetwork *trainer.HashNetwork
	constraints []SecurityConstraint
	violations  []BehaviorPattern
}

func NewUserTrainer(network *trainer.HashNetwork) *UserTrainer {
	return &UserTrainer{
		baseNetwork: network,
	}
}

func (tg *UserTrainer) TrainUserGates(ctx context.Context, gates *UserSecurityGates) (*TrainedGates, error) {
	if gates.MaxGenerations == 0 {
		gates.MaxGenerations = 100
	}
	if gates.PopulationSize == 0 {
		gates.PopulationSize = 256
	}
	if gates.MutationRate == 0 {
		gates.MutationRate = 0.1
	}

	tg.constraints = gates.Constraints
	tg.violations = gates.Patterns

	population := tg.initPopulation(gates.PopulationSize)
	constraintTokens := tg.extractConstraintTokens(gates.Constraints)
	violationPatterns := tg.extractViolationPatterns(gates.Patterns)

	var bestFitness float64
	bestSeeds := make([]Seed, 0)

	for gen := 0; gen < gates.MaxGenerations; gen++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		results := tg.evaluateWithSecurity(population, constraintTokens, violationPatterns)
		population = tg.selectAndMutate(results, gates.MutationRate)

		if results[0].Fitness > bestFitness {
			bestFitness = results[0].Fitness
			bestSeeds = make([]Seed, len(population[0].Seeds))
			copy(bestSeeds, population[0].Seeds)
		}

		if tg.checkConvergence(results, gen, gates.MaxGenerations) {
			break
		}
	}

	rules := tg.buildLogicalRules(gates.Constraints, gates.Patterns)

	return &TrainedGates{
		Seeds:     bestSeeds,
		Rules:     rules,
		OrgID:     gates.OrgID,
		UserID:    gates.UserID,
		Fitness:   bestFitness,
		TrainedAt: time.Now(),
	}, nil
}

type Candidate struct {
	Seeds   []Seed
	Fitness float64
}

func (tg *UserTrainer) initPopulation(size int) []Candidate {
	population := make([]Candidate, size)
	for i := range population {
		seeds := make([]Seed, 8)
		for j := range seeds {
			data := make([]float64, 32)
			for k := range data {
				data[k] = rand.Float64()
			}
			seeds[j] = Seed{Data: data, Fitness: 0}
		}
		population[i] = Candidate{Seeds: seeds}
	}
	return population
}

func (tg *UserTrainer) extractConstraintTokens(constraints []SecurityConstraint) []uint32 {
	tokens := make([]uint32, 0)
	for _, c := range constraints {
		switch c.Type {
		case "deny", "block":
			tokens = append(tokens, 0x24000000)
		case "flag", "warn":
			tokens = append(tokens, 0x24010000)
		case "allow":
			tokens = append(tokens, 0x10000000)
		}
	}
	return tokens
}

func (tg *UserTrainer) extractViolationPatterns(patterns []BehaviorPattern) []uint64 {
	hashes := make([]uint64, 0)
	for _, p := range patterns {
		if !p.Success {
			hashes = append(hashes, p.ActionHash)
		}
	}
	return hashes
}

func (tg *UserTrainer) evaluateWithSecurity(population []Candidate, constraintTokens []uint32, violationPatterns []uint64) []Candidate {
	for i := range population {
		fitness := tg.calculateSecurityFitness(population[i].Seeds, constraintTokens, violationPatterns)
		population[i].Fitness = fitness
	}
	return population
}

func (tg *UserTrainer) calculateSecurityFitness(seeds []Seed, constraintTokens []uint32, violationPatterns []uint64) float64 {
	var totalFitness float64

	for _, seed := range seeds {
		input := seed.Data
		hash := trainer.SHA256Hash(input)

		constraintScore := 0.0
		for _, ct := range constraintTokens {
			if (uint32(hash[0])<<24|uint32(hash[1])<<16)&ct != 0 {
				constraintScore += 0.5
			}
		}

		violationPenalty := 0.0
		hash64 := uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
			uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])
		for _, vp := range violationPatterns {
			if hash64 == vp {
				violationPenalty -= 1.0
			}
		}

		alignment := 1.0 - math.Abs(float64(hash[0])/255.0-0.5)
		stability := 1.0
		format := 1.0

		fitness := SecurityFitness{
			Alignment:        alignment,
			Stability:        stability,
			Format:           format,
			ConstraintScore:  constraintScore,
			ViolationPenalty: violationPenalty,
		}

		totalFitness += fitness.Total()
	}

	return totalFitness / float64(len(seeds))
}

func (tg *UserTrainer) selectAndMutate(population []Candidate, mutationRate float64) []Candidate {
	sortByFitness(population)

	eliteCount := len(population) / 10
	newPopulation := make([]Candidate, len(population))

	for i := 0; i < eliteCount; i++ {
		newPopulation[i] = population[i]
	}

	for i := eliteCount; i < len(population); i++ {
		parent1 := population[rand.Intn(eliteCount)].Seeds
		parent2 := population[rand.Intn(eliteCount)].Seeds

		childSeeds := make([]Seed, len(parent1))
		for j := range childSeeds {
			if rand.Float64() < 0.5 {
				childSeeds[j] = parent1[j]
			} else {
				childSeeds[j] = parent2[j]
			}

			if rand.Float64() < mutationRate {
				childSeeds[j] = tg.mutateSeed(childSeeds[j])
			}
		}
		newPopulation[i] = Candidate{Seeds: childSeeds}
	}

	return newPopulation
}

func (tg *UserTrainer) mutateSeed(seed Seed) Seed {
	mutated := seed
	idx := rand.Intn(len(mutated.Data))
	mutated.Data[idx] += (rand.Float64() - 0.5) * 0.1
	mutated.Data[idx] = math.Max(0, math.Min(1, mutated.Data[idx]))
	return mutated
}

func (tg *UserTrainer) checkConvergence(results []Candidate, gen, maxGen int) bool {
	if gen < 10 {
		return false
	}

	var totalDiff float64
	for i := 1; i < len(results); i++ {
		totalDiff += math.Abs(results[i].Fitness - results[i-1].Fitness)
	}

	avgDiff := totalDiff / float64(len(results)-1)
	return avgDiff < 0.0001 || gen >= maxGen-1
}

func (tg *UserTrainer) buildLogicalRules(constraints []SecurityConstraint, patterns []BehaviorPattern) []LogicalRule {
	rules := make([]LogicalRule, 0)

	for _, c := range constraints {
		rule := LogicalRule{
			RuleType:   "constraint",
			Premises:   []string{c.Text},
			Conclusion: c.Type,
			Source:     "guardrail",
			Confidence: 0.9,
		}

		if c.Severity != "" {
			rule.Premises = append(rule.Premises, fmt.Sprintf("severity=%s", c.Severity))
		}

		rules = append(rules, rule)
	}

	for _, p := range patterns {
		if !p.Success {
			rule := LogicalRule{
				RuleType:   "disjoint",
				Premises:   []string{fmt.Sprintf("task=%s", p.TaskType)},
				Conclusion: "deny",
				Source:     "pattern",
				Confidence: 0.7,
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

func sortByFitness(population []Candidate) {
	for i := 0; i < len(population); i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].Fitness > population[i].Fitness {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
}

func (tg *TrainedGates) BestSeedID() string {
	if len(tg.Seeds) == 0 {
		return ""
	}
	best := tg.Seeds[0]
	for i, s := range tg.Seeds[1:] {
		if s.Fitness > best.Fitness {
			best = s
			_ = i
		}
	}
	return fmt.Sprintf("user_%s_seed_%d", tg.UserID, 0)
}

func (tg *TrainedGates) GetRulesByType(ruleType string) []LogicalRule {
	rules := make([]LogicalRule, 0)
	for _, r := range tg.Rules {
		if r.RuleType == ruleType {
			rules = append(rules, r)
		}
	}
	return rules
}
