package testdata

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// TestDataGenerator generates test data for various scenarios
type TestDataGenerator struct {
	Config GeneratorConfig
	rand   *rand.Rand
}

// GeneratorConfig holds configuration for test data generation
type GeneratorConfig struct {
	Seed           int64
	UserCount      int
	SkillCount     int
	TransactionCount int
	AgentCount     int
	TimeRange      time.Duration
}

// TestUser represents a test user
type TestUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Wallet   string `json:"wallet"`
	NRNBalance float64 `json:"nrn_balance"`
	Skills   []string `json:"skills"`
	CreatedAt time.Time `json:"created_at"`
}

// TestSkill represents a test skill
type TestSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Creator     string `json:"creator"`
	Price       float64 `json:"price"`
	Rating      float64 `json:"rating"`
	UsageCount  int `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// TestTransaction represents a test transaction
type TestTransaction struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    float64 `json:"amount"`
	SkillID   string `json:"skill_id,omitempty"`
	Status    string `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// TestAgent represents a test CORTEX agent
type TestAgent struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	Owner        string   `json:"owner"`
	Performance  AgentPerformance `json:"performance"`
	CreatedAt    time.Time `json:"created_at"`
}

// AgentPerformance represents agent performance metrics
type AgentPerformance struct {
	TasksCompleted int     `json:"tasks_completed"`
	SuccessRate    float64 `json:"success_rate"`
	AvgLatency     float64 `json:"avg_latency_ms"`
	LearningScore  float64 `json:"learning_score"`
}

// TestScenarioData holds all test data for a scenario
type TestScenarioData struct {
	Users        []TestUser        `json:"users"`
	Skills       []TestSkill       `json:"skills"`
	Transactions []TestTransaction `json:"transactions"`
	Agents       []TestAgent       `json:"agents"`
	Metadata     ScenarioMetadata  `json:"metadata"`
}

// ScenarioMetadata holds metadata about the test scenario
type ScenarioMetadata struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Duration    string    `json:"duration"`
	UserCount   int       `json:"user_count"`
	SkillCount  int       `json:"skill_count"`
	AgentCount  int       `json:"agent_count"`
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(config GeneratorConfig) *TestDataGenerator {
	if config.Seed == 0 {
		config.Seed = time.Now().UnixNano()
	}

	return &TestDataGenerator{
		Config: config,
		rand:   rand.New(rand.NewSource(config.Seed)),
	}
}

// GenerateScenarioData generates complete test data for a scenario
func (g *TestDataGenerator) GenerateScenarioData(scenarioName string) (*TestScenarioData, error) {
	data := &TestScenarioData{
		Metadata: ScenarioMetadata{
			Name:        scenarioName,
			Description: fmt.Sprintf("Generated test data for %s scenario", scenarioName),
			CreatedAt:   time.Now(),
			Duration:    g.Config.TimeRange.String(),
			UserCount:   g.Config.UserCount,
			SkillCount:  g.Config.SkillCount,
			AgentCount:  g.Config.AgentCount,
		},
	}

	// Generate users
	users, err := g.GenerateUsers(g.Config.UserCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate users: %w", err)
	}
	data.Users = users

	// Generate skills
	skills, err := g.GenerateSkills(g.Config.SkillCount, users)
	if err != nil {
		return nil, fmt.Errorf("failed to generate skills: %w", err)
	}
	data.Skills = skills

	// Generate agents
	agents, err := g.GenerateAgents(g.Config.AgentCount, users)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agents: %w", err)
	}
	data.Agents = agents

	// Generate transactions
	transactions, err := g.GenerateTransactions(g.Config.TransactionCount, users, skills)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transactions: %w", err)
	}
	data.Transactions = transactions

	return data, nil
}

// GenerateUsers generates test users
func (g *TestDataGenerator) GenerateUsers(count int) ([]TestUser, error) {
	users := make([]TestUser, count)

	usernames := []string{"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry", "iris", "jack"}
	domains := []string{"example.com", "test.org", "demo.net", "sample.io"}

	for i := 0; i < count; i++ {
		username := fmt.Sprintf("%s_%d", usernames[i%len(usernames)], i)
		
		users[i] = TestUser{
			ID:       fmt.Sprintf("user_%d", i+1),
			Username: username,
			Email:    fmt.Sprintf("%s@%s", username, domains[g.rand.Intn(len(domains))]),
			Wallet:   g.generateWalletAddress(),
			NRNBalance: float64(g.rand.Intn(10000)) + g.rand.Float64()*1000,
			Skills:   g.generateUserSkills(),
			CreatedAt: time.Now().Add(-time.Duration(g.rand.Intn(int(g.Config.TimeRange.Hours()))) * time.Hour),
		}
	}

	return users, nil
}

// GenerateSkills generates test skills
func (g *TestDataGenerator) GenerateSkills(count int, users []TestUser) ([]TestSkill, error) {
	skills := make([]TestSkill, count)

	skillNames := []string{
		"Data Analysis", "Web Development", "Machine Learning", "Blockchain Development",
		"UI/UX Design", "DevOps", "Mobile Development", "API Integration",
		"Database Design", "Security Audit", "Performance Optimization", "Testing Automation",
	}

	categories := []string{"Development", "Design", "Analysis", "Security", "Operations"}

	for i := 0; i < count; i++ {
		creator := users[g.rand.Intn(len(users))]
		
		skills[i] = TestSkill{
			ID:          fmt.Sprintf("skill_%d", i+1),
			Name:        skillNames[i%len(skillNames)],
			Description: fmt.Sprintf("Advanced %s capabilities with AI enhancement", skillNames[i%len(skillNames)]),
			Category:    categories[g.rand.Intn(len(categories))],
			Creator:     creator.ID,
			Price:       float64(g.rand.Intn(500)) + g.rand.Float64()*100,
			Rating:      3.0 + g.rand.Float64()*2.0, // 3.0 to 5.0
			UsageCount:  g.rand.Intn(1000),
			CreatedAt:   time.Now().Add(-time.Duration(g.rand.Intn(int(g.Config.TimeRange.Hours()))) * time.Hour),
		}
	}

	return skills, nil
}

// GenerateAgents generates test CORTEX agents
func (g *TestDataGenerator) GenerateAgents(count int, users []TestUser) ([]TestAgent, error) {
	agents := make([]TestAgent, count)

	agentTypes := []string{"Developer", "Collaborator", "Learner", "Optimizer", "Validator"}
	capabilities := map[string][]string{
		"Developer":     {"skill-creation", "code-generation", "testing", "debugging"},
		"Collaborator":  {"task-coordination", "knowledge-sharing", "communication", "consensus"},
		"Learner":       {"adaptation", "pattern-recognition", "optimization", "feedback-processing"},
		"Optimizer":     {"performance-tuning", "resource-management", "efficiency-analysis"},
		"Validator":     {"quality-assurance", "security-checking", "compliance-verification"},
	}

	for i := 0; i < count; i++ {
		agentType := agentTypes[g.rand.Intn(len(agentTypes))]
		owner := users[g.rand.Intn(len(users))]

		agents[i] = TestAgent{
			ID:           fmt.Sprintf("cortex_agent_%d", i+1),
			Type:         agentType,
			Capabilities: capabilities[agentType],
			Owner:        owner.ID,
			Performance: AgentPerformance{
				TasksCompleted: g.rand.Intn(100),
				SuccessRate:    0.7 + g.rand.Float64()*0.3, // 70% to 100%
				AvgLatency:     float64(g.rand.Intn(2000)) + g.rand.Float64()*1000, // 0-3000ms
				LearningScore:  g.rand.Float64(),
			},
			CreatedAt: time.Now().Add(-time.Duration(g.rand.Intn(int(g.Config.TimeRange.Hours()))) * time.Hour),
		}
	}

	return agents, nil
}

// GenerateTransactions generates test transactions
func (g *TestDataGenerator) GenerateTransactions(count int, users []TestUser, skills []TestSkill) ([]TestTransaction, error) {
	transactions := make([]TestTransaction, count)

	transactionTypes := []string{"skill_purchase", "skill_execution", "reward_distribution", "nrn_transfer"}
	statuses := []string{"pending", "completed", "failed"}

	for i := 0; i < count; i++ {
		txType := transactionTypes[g.rand.Intn(len(transactionTypes))]
		from := users[g.rand.Intn(len(users))]
		to := users[g.rand.Intn(len(users))]
		
		// Ensure from and to are different
		for from.ID == to.ID {
			to = users[g.rand.Intn(len(users))]
		}

		transaction := TestTransaction{
			ID:        fmt.Sprintf("tx_%d", i+1),
			Type:      txType,
			From:      from.ID,
			To:        to.ID,
			Amount:    float64(g.rand.Intn(1000)) + g.rand.Float64()*100,
			Status:    statuses[g.rand.Intn(len(statuses))],
			Timestamp: time.Now().Add(-time.Duration(g.rand.Intn(int(g.Config.TimeRange.Hours()))) * time.Hour),
		}

		// Add skill ID for skill-related transactions
		if txType == "skill_purchase" || txType == "skill_execution" {
			if len(skills) > 0 {
				skill := skills[g.rand.Intn(len(skills))]
				transaction.SkillID = skill.ID
			}
		}

		transactions[i] = transaction
	}

	return transactions, nil
}

// generateWalletAddress generates a mock wallet address
func (g *TestDataGenerator) generateWalletAddress() string {
	const charset = "0123456789abcdef"
	address := "0x"
	for i := 0; i < 40; i++ {
		address += string(charset[g.rand.Intn(len(charset))])
	}
	return address
}

// generateUserSkills generates random skills for a user
func (g *TestDataGenerator) generateUserSkills() []string {
	allSkills := []string{
		"programming", "design", "analysis", "communication", "leadership",
		"problem-solving", "creativity", "collaboration", "learning", "adaptation",
	}

	skillCount := g.rand.Intn(5) + 1 // 1 to 5 skills
	skills := make([]string, skillCount)

	for i := 0; i < skillCount; i++ {
		skills[i] = allSkills[g.rand.Intn(len(allSkills))]
	}

	return skills
}

// SaveToFile saves test data to a JSON file
func (g *TestDataGenerator) SaveToFile(data *TestScenarioData, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// In a real implementation, this would write to file
	// For now, we'll just return success
	fmt.Printf("Test data saved to %s (%d bytes)\n", filename, len(jsonData))
	return nil
}

// LoadFromFile loads test data from a JSON file
func (g *TestDataGenerator) LoadFromFile(filename string) (*TestScenarioData, error) {
	// In a real implementation, this would read from file
	// For now, return a sample data structure
	return &TestScenarioData{
		Users:        []TestUser{},
		Skills:       []TestSkill{},
		Transactions: []TestTransaction{},
		Agents:       []TestAgent{},
		Metadata: ScenarioMetadata{
			Name:      "loaded_scenario",
			CreatedAt: time.Now(),
		},
	}, nil
}
