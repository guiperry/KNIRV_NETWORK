package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway/option"
)

func TestEconomicsSkillsService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/economics/skills":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"skills": []map[string]interface{}{
						{
							"id":           "skill-1",
							"name":         "Network Repair",
							"description":  "Repairs network connectivity issues",
							"cost":         100,
							"success_rate": 0.95,
						},
						{
							"id":           "skill-2",
							"name":         "Data Analysis",
							"description":  "Analyzes data patterns",
							"cost":         150,
							"success_rate": 0.88,
						},
					},
				})
			} else if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "skill-3",
					"created": true,
				})
			}
		case "/economics/skills/skill-1":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":           "skill-1",
					"name":         "Network Repair",
					"description":  "Repairs network connectivity issues",
					"cost":         100,
					"success_rate": 0.95,
					"usage_count":  1250,
					"total_earned": 125000,
				})
			} else if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"updated": true,
				})
			} else if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("List skills", func(t *testing.T) {
		skills, err := client.Economics.Skills.List(ctx)
		if err != nil {
			t.Errorf("Failed to list skills: %v", err)
		}

		if len(skills) != 2 {
			t.Errorf("Expected 2 skills, got %d", len(skills))
		}

		if skills[0].Name != "Network Repair" {
			t.Errorf("Expected first skill name 'Network Repair', got %s", skills[0].Name)
		}
	})

	t.Run("Get skill by ID", func(t *testing.T) {
		skill, err := client.Economics.Skills.Get(ctx, "skill-1")
		if err != nil {
			t.Errorf("Failed to get skill: %v", err)
		}

		if skill.ID != "skill-1" {
			t.Errorf("Expected skill ID 'skill-1', got %s", skill.ID)
		}

		if skill.UsageCount != 1250 {
			t.Errorf("Expected usage count 1250, got %d", skill.UsageCount)
		}
	})

	t.Run("Create skill", func(t *testing.T) {
		newSkill := &SkillCreateRequest{
			Name:        "Test Skill",
			Description: "A test skill",
			Cost:        200,
		}

		result, err := client.Economics.Skills.Create(ctx, newSkill)
		if err != nil {
			t.Errorf("Failed to create skill: %v", err)
		}

		if result.ID != "skill-3" {
			t.Errorf("Expected created skill ID 'skill-3', got %s", result.ID)
		}
	})

	t.Run("Update skill", func(t *testing.T) {
		updateReq := &SkillUpdateRequest{
			Cost: 120,
		}

		updatedSkill, err := client.Economics.Skills.Update(ctx, "skill-1", updateReq)
		if err != nil {
			t.Errorf("Failed to update skill: %v", err)
		}

		if updatedSkill.Cost != 120 {
			t.Errorf("Expected updated cost 120, got %d", updatedSkill.Cost)
		}
	})

	t.Run("Delete skill", func(t *testing.T) {
		err := client.Economics.Skills.Delete(ctx, "skill-1")
		if err != nil {
			t.Errorf("Failed to delete skill: %v", err)
		}
	})
}

func TestEconomicsLLMService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/economics/llm/models":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"id":             "model-1",
						"name":           "GPT-4",
						"cost_per_token": 0.00003,
						"max_tokens":     8192,
					},
					{
						"id":             "model-2",
						"name":           "Claude-3",
						"cost_per_token": 0.000015,
						"max_tokens":     4096,
					},
				},
			})
		case "/economics/llm/usage":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_tokens": 1500000,
				"total_cost":   45.50,
				"requests":     2500,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("List LLM models", func(t *testing.T) {
		models, err := client.Economics.LLM.ListModels(ctx)
		if err != nil {
			t.Errorf("Failed to list LLM models: %v", err)
		}

		if len(models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(models))
		}

		if models[0].Name != "GPT-4" {
			t.Errorf("Expected first model name 'GPT-4', got %s", models[0].Name)
		}
	})

	t.Run("Get LLM usage", func(t *testing.T) {
		usage, err := client.Economics.LLM.GetUsage(ctx)
		if err != nil {
			t.Errorf("Failed to get LLM usage: %v", err)
		}

		if usage.TotalTokens != 1500000 {
			t.Errorf("Expected total tokens 1500000, got %d", usage.TotalTokens)
		}

		if usage.TotalCost != 45.50 {
			t.Errorf("Expected total cost 45.50, got %f", usage.TotalCost)
		}
	})
}

func TestEconomicsValidationService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/economics/validation/validate":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":      true,
				"confidence": 0.95,
				"errors":     []string{},
			})
		case "/economics/validation/rules":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"rules": []map[string]interface{}{
					{
						"id":          "rule-1",
						"name":        "Cost Validation",
						"description": "Validates skill cost ranges",
						"active":      true,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Validate skill", func(t *testing.T) {
		validateReq := &ValidationRequest{
			SkillID: "skill-1",
			Data: map[string]interface{}{
				"cost": 100,
				"name": "Test Skill",
			},
		}

		result, err := client.Economics.Validation.Validate(ctx, validateReq)
		if err != nil {
			t.Errorf("Failed to validate: %v", err)
		}

		if !result.Valid {
			t.Error("Expected validation to pass")
		}

		if result.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
		}
	})

	t.Run("List validation rules", func(t *testing.T) {
		rules, err := client.Economics.Validation.ListRules(ctx)
		if err != nil {
			t.Errorf("Failed to list validation rules: %v", err)
		}

		if len(rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(rules))
		}

		if rules[0].Name != "Cost Validation" {
			t.Errorf("Expected rule name 'Cost Validation', got %s", rules[0].Name)
		}
	})
}

func TestEconomicsFeesService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/economics/fees/calculate":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"base_fee":     10,
				"skill_fee":    100,
				"platform_fee": 5,
				"total_fee":    115,
			})
		case "/economics/fees/structure":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"base_fee_percentage":     0.05,
				"platform_fee_percentage": 0.02,
				"minimum_fee":             1,
				"maximum_fee":             1000,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Calculate fees", func(t *testing.T) {
		feeReq := &FeeCalculationRequest{
			SkillID: "skill-1",
			Amount:  100,
		}

		fees, err := client.Economics.Fees.Calculate(ctx, feeReq)
		if err != nil {
			t.Errorf("Failed to calculate fees: %v", err)
		}

		if fees.TotalFee != 15 { // 10 + 5 = 15 from our mock
			t.Errorf("Expected total fee 15, got %f", fees.TotalFee)
		}

		if fees.BaseFee != 10 {
			t.Errorf("Expected base fee 10, got %f", fees.BaseFee)
		}
	})

	t.Run("Get fee structure", func(t *testing.T) {
		structure, err := client.Economics.Fees.GetStructure(ctx)
		if err != nil {
			t.Errorf("Failed to get fee structure: %v", err)
		}

		if structure.BaseFeePercentage != 0.05 {
			t.Errorf("Expected base fee percentage 0.05, got %f", structure.BaseFeePercentage)
		}

		if structure.MinimumFee != 1 {
			t.Errorf("Expected minimum fee 1, got %f", structure.MinimumFee)
		}
	})
}

func TestEconomicsMetricsService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/economics/metrics/overview":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_revenue": 50000,
				"total_skills":  25,
				"active_users":  150,
				"success_rate":  0.92,
				"average_cost":  125,
			})
		case "/economics/metrics/skills":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"skill_metrics": []map[string]interface{}{
					{
						"skill_id":     "skill-1",
						"usage_count":  500,
						"revenue":      25000,
						"success_rate": 0.95,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Get metrics overview", func(t *testing.T) {
		overview, err := client.Economics.Metrics.GetOverview(ctx)
		if err != nil {
			t.Errorf("Failed to get metrics overview: %v", err)
		}

		if overview.TotalRevenue != 50000 {
			t.Errorf("Expected total revenue 50000, got %d", overview.TotalRevenue)
		}

		if overview.ActiveUsers != 150 {
			t.Errorf("Expected active users 150, got %d", overview.ActiveUsers)
		}
	})

	t.Run("Get skill metrics", func(t *testing.T) {
		metrics, err := client.Economics.Metrics.GetSkillMetrics(ctx)
		if err != nil {
			t.Errorf("Failed to get skill metrics: %v", err)
		}

		if len(metrics) != 1 {
			t.Errorf("Expected 1 skill metric, got %d", len(metrics))
		}

		if metrics[0].SkillID != "skill-1" {
			t.Errorf("Expected skill ID 'skill-1', got %s", metrics[0].SkillID)
		}
	})
}
