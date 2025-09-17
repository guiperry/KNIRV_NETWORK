package main

import (
	"os"
	"testing"
	"time"
)

// TestPrimaryBadgeTypes tests the three primary badge types: Skills, Capabilities, and Properties
func TestPrimaryBadgeTypes(t *testing.T) {
	// Setup test environment with cleanup
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	t.Cleanup(func() {
		chromemManager.Close()
		os.RemoveAll(tempDir)
	})

	// Create an agent for testing
	metadata := map[string]interface{}{"version": "1.0", "test": true}
	agent, err := agentManager.CreateAgent("Primary Badge Test Agent", "Agent for testing primary badge types", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test 1: Create and attach a Skill badge
	t.Run("SkillBadge", func(t *testing.T) {
		skillParameters := map[string]interface{}{
			"execution_cost_nrn": "100",
			"complexity_level":   "intermediate",
			"input_types":        []string{"text", "json"},
			"output_types":       []string{"text", "analysis"},
		}
		skillRequirements := []string{"python>=3.8", "tensorflow>=2.0"}

		skillBadge, err := agentManager.RegisterSkillAsBadge(
			"Data Analysis Skill",
			"Advanced data analysis and visualization capabilities",
			"https://example.com/skill-icon.png",
			wallet.GetAddress(),
			"data_analysis",
			skillParameters,
			skillRequirements,
		)
		if err != nil {
			t.Fatalf("Failed to create skill badge: %v", err)
		}

		// Verify skill badge properties
		if skillBadge.BadgeType != "skill" {
			t.Errorf("Expected badge type 'skill', got '%s'", skillBadge.BadgeType)
		}

		if skillBadge.Metadata["skill_type"] != "data_analysis" {
			t.Errorf("Expected skill_type 'data_analysis', got '%v'", skillBadge.Metadata["skill_type"])
		}

		if skillBadge.Metadata["execution_env"] != "knirvchain" {
			t.Errorf("Expected execution_env 'knirvchain', got '%v'", skillBadge.Metadata["execution_env"])
		}

		// Attach skill badge to agent
		skillAttachment, err := agentManager.AttachBadgeToAgent(agent.ID, skillBadge.ID, map[string]interface{}{
			"proficiency_level": "expert",
			"certification_date": time.Now().Format("2006-01-02"),
		})
		if err != nil {
			t.Fatalf("Failed to attach skill badge: %v", err)
		}

		if skillAttachment.BadgeId != skillBadge.ID {
			t.Errorf("Expected badge ID '%s', got '%s'", skillBadge.ID, skillAttachment.BadgeId)
		}
	})

	// Test 2: Create and attach a Capability badge
	t.Run("CapabilityBadge", func(t *testing.T) {
		capabilitySchema := map[string]interface{}{
			"gas_fee_nrn": "50",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
					"limit": map[string]interface{}{"type": "integer"},
				},
			},
			"output_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"results": map[string]interface{}{"type": "array"},
					"count":   map[string]interface{}{"type": "integer"},
				},
			},
		}
		locationHints := []string{"us-east-1", "eu-west-1"}

		capabilityBadge, err := agentManager.RegisterCapabilityAsBadge(
			"Search API Capability",
			"Provides search functionality with structured queries",
			"https://example.com/capability-icon.png",
			wallet.GetAddress(),
			"search_api",
			capabilitySchema,
			locationHints,
		)
		if err != nil {
			t.Fatalf("Failed to create capability badge: %v", err)
		}

		// Verify capability badge properties
		if capabilityBadge.BadgeType != "capability" {
			t.Errorf("Expected badge type 'capability', got '%s'", capabilityBadge.BadgeType)
		}

		if capabilityBadge.Metadata["capability_type"] != "search_api" {
			t.Errorf("Expected capability_type 'search_api', got '%v'", capabilityBadge.Metadata["capability_type"])
		}

		if capabilityBadge.Metadata["gas_fee_nrn"] != "50" {
			t.Errorf("Expected gas_fee_nrn '50', got '%v'", capabilityBadge.Metadata["gas_fee_nrn"])
		}

		// Attach capability badge to agent
		capabilityAttachment, err := agentManager.AttachBadgeToAgent(agent.ID, capabilityBadge.ID, map[string]interface{}{
			"endpoint_url": "https://api.example.com/search",
			"rate_limit":   "1000/hour",
		})
		if err != nil {
			t.Fatalf("Failed to attach capability badge: %v", err)
		}

		if capabilityAttachment.BadgeId != capabilityBadge.ID {
			t.Errorf("Expected badge ID '%s', got '%s'", capabilityBadge.ID, capabilityAttachment.BadgeId)
		}
	})

	// Test 3: Create and attach a Property badge
	t.Run("PropertyBadge", func(t *testing.T) {
		propertyConstraints := map[string]interface{}{
			"validation_rules": []string{"must_be_positive", "max_value_100"},
			"category":         "performance",
			"unit":             "percentage",
		}

		propertyBadge, err := agentManager.RegisterPropertyAsBadge(
			"Reliability Score",
			"Agent's reliability score based on historical performance",
			"https://example.com/property-icon.png",
			wallet.GetAddress(),
			"reliability_score",
			95.5, // Property value
			propertyConstraints,
		)
		if err != nil {
			t.Fatalf("Failed to create property badge: %v", err)
		}

		// Verify property badge properties
		if propertyBadge.BadgeType != "property" {
			t.Errorf("Expected badge type 'property', got '%s'", propertyBadge.BadgeType)
		}

		if propertyBadge.Metadata["property_type"] != "reliability_score" {
			t.Errorf("Expected property_type 'reliability_score', got '%v'", propertyBadge.Metadata["property_type"])
		}

		if propertyBadge.Metadata["value"] != 95.5 {
			t.Errorf("Expected value '95.5', got '%v'", propertyBadge.Metadata["value"])
		}

		if propertyBadge.Metadata["immutable"] != true {
			t.Errorf("Expected immutable 'true', got '%v'", propertyBadge.Metadata["immutable"])
		}

		// Attach property badge to agent
		propertyAttachment, err := agentManager.AttachBadgeToAgent(agent.ID, propertyBadge.ID, map[string]interface{}{
			"measurement_date": time.Now().Format("2006-01-02"),
			"verified_by":      "automated_system",
		})
		if err != nil {
			t.Fatalf("Failed to attach property badge: %v", err)
		}

		if propertyAttachment.BadgeId != propertyBadge.ID {
			t.Errorf("Expected badge ID '%s', got '%s'", propertyBadge.ID, propertyAttachment.BadgeId)
		}
	})

	// Test 4: Retrieve badges by type
	t.Run("RetrieveBadgesByType", func(t *testing.T) {
		// Test skill badges retrieval
		skillBadges, err := agentManager.GetSkillBadges()
		if err != nil {
			t.Fatalf("Failed to get skill badges: %v", err)
		}
		if len(skillBadges) == 0 {
			t.Error("Expected at least one skill badge")
		}

		// Test capability badges retrieval
		capabilityBadges, err := agentManager.GetCapabilityBadges()
		if err != nil {
			t.Fatalf("Failed to get capability badges: %v", err)
		}
		if len(capabilityBadges) == 0 {
			t.Error("Expected at least one capability badge")
		}

		// Test property badges retrieval
		propertyBadges, err := agentManager.GetPropertyBadges()
		if err != nil {
			t.Fatalf("Failed to get property badges: %v", err)
		}
		if len(propertyBadges) == 0 {
			t.Error("Expected at least one property badge")
		}
	})

	// Test 5: Retrieve agent-specific badges by type
	t.Run("RetrieveAgentBadgesByType", func(t *testing.T) {
		// Test agent skills retrieval
		agentSkills, err := agentManager.GetAgentSkills(agent.ID)
		if err != nil {
			t.Fatalf("Failed to get agent skills: %v", err)
		}
		if len(agentSkills) == 0 {
			t.Error("Expected at least one skill for the agent")
		}

		// Test agent capabilities retrieval (using existing method)
		agentCapabilities, err := agentManager.GetAgentCapabilities(agent.ID)
		if err != nil {
			t.Fatalf("Failed to get agent capabilities: %v", err)
		}
		if len(agentCapabilities) == 0 {
			t.Error("Expected at least one capability for the agent")
		}

		// Test agent properties retrieval
		agentProperties, err := agentManager.GetAgentProperties(agent.ID)
		if err != nil {
			t.Fatalf("Failed to get agent properties: %v", err)
		}
		if len(agentProperties) == 0 {
			t.Error("Expected at least one property for the agent")
		}
	})
}
