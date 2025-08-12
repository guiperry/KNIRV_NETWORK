"""
Comprehensive tests for the KNIRV Gateway Economics service
"""

import pytest
import responses
import json
from unittest.mock import Mock, patch
from knirv_gateway_sdk import KNIRVGateway, KNIRVAPIError, KNIRVValidationError


@pytest.fixture
def client():
    """Create a test client."""
    return KNIRVGateway(
        base_url="https://test.knirv.com",
        api_key="test-key"
    )


@pytest.fixture
def mock_skill_data():
    """Mock skill data for testing."""
    return {
        "id": "skill-1",
        "name": "Network Repair",
        "description": "Repairs network connectivity issues",
        "cost": 100,
        "success_rate": 0.95,
        "usage_count": 1250,
        "total_earned": 125000,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }


@pytest.fixture
def mock_skills_list():
    """Mock skills list for testing."""
    return {
        "skills": [
            {
                "id": "skill-1",
                "name": "Network Repair",
                "description": "Repairs network connectivity issues",
                "cost": 100,
                "success_rate": 0.95
            },
            {
                "id": "skill-2",
                "name": "Data Analysis",
                "description": "Analyzes data patterns",
                "cost": 150,
                "success_rate": 0.88
            }
        ],
        "total": 2,
        "page": 1,
        "per_page": 10
    }


class TestEconomicsSkillsService:
    """Test the Economics Skills service."""

    @responses.activate
    def test_list_skills(self, client, mock_skills_list):
        """Test listing skills."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills",
            json=mock_skills_list,
            status=200
        )

        skills = client.economics.skills.list()

        assert len(skills["skills"]) == 2
        assert skills["skills"][0]["name"] == "Network Repair"
        assert skills["skills"][1]["name"] == "Data Analysis"
        assert skills["total"] == 2

    @responses.activate
    def test_list_skills_with_pagination(self, client):
        """Test listing skills with pagination."""
        mock_response = {
            "skills": [{"id": f"skill-{i}", "name": f"Skill {i}"} for i in range(5)],
            "total": 25,
            "page": 2,
            "per_page": 5
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills",
            json=mock_response,
            status=200
        )

        skills = client.economics.skills.list(page=2, per_page=5)
        
        assert len(skills["skills"]) == 5
        assert skills["page"] == 2
        assert skills["per_page"] == 5
        assert skills["total"] == 25

    @responses.activate
    def test_get_skill(self, client, mock_skill_data):
        """Test getting a specific skill."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills/skill-1",
            json=mock_skill_data,
            status=200
        )

        skill = client.economics.skills.get("skill-1")
        
        assert skill["id"] == "skill-1"
        assert skill["name"] == "Network Repair"
        assert skill["cost"] == 100
        assert skill["success_rate"] == 0.95

    @responses.activate
    def test_get_skill_not_found(self, client):
        """Test getting a non-existent skill."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills/nonexistent",
            json={"error": "Skill not found", "code": "SKILL_NOT_FOUND"},
            status=404
        )

        with pytest.raises(KNIRVAPIError) as exc_info:
            client.economics.skills.get("nonexistent")
        
        assert exc_info.value.status_code == 404
        assert "Skill not found" in str(exc_info.value)

    @responses.activate
    def test_create_skill(self, client):
        """Test creating a new skill."""
        skill_data = {
            "name": "Test Skill",
            "description": "A test skill",
            "cost": 200,
            "category": "testing"
        }
        
        mock_response = {
            "id": "skill-3",
            "created": True,
            **skill_data
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/economics/skills",
            json=mock_response,
            status=201
        )

        result = client.economics.skills.create(skill_data)
        
        assert result["id"] == "skill-3"
        assert result["created"] is True
        assert result["name"] == "Test Skill"

    @responses.activate
    def test_create_skill_validation_error(self, client):
        """Test creating a skill with validation errors."""
        responses.add(
            responses.POST,
            "https://test.knirv.com/economics/skills",
            json={
                "error": "Validation failed",
                "code": "VALIDATION_ERROR",
                "details": {"name": ["This field is required"]}
            },
            status=400
        )

        with pytest.raises(KNIRVValidationError) as exc_info:
            client.economics.skills.create({})
        
        assert exc_info.value.status_code == 400
        assert "Validation failed" in str(exc_info.value)

    @responses.activate
    def test_update_skill(self, client):
        """Test updating a skill."""
        update_data = {"cost": 120, "description": "Updated description"}
        
        responses.add(
            responses.PUT,
            "https://test.knirv.com/economics/skills/skill-1",
            json={"updated": True},
            status=200
        )

        result = client.economics.skills.update("skill-1", update_data)
        assert result["updated"] is True

    @responses.activate
    def test_delete_skill(self, client):
        """Test deleting a skill."""
        responses.add(
            responses.DELETE,
            "https://test.knirv.com/economics/skills/skill-1",
            status=204
        )

        # Should not raise an exception
        client.economics.skills.delete("skill-1")

    @responses.activate
    def test_search_skills(self, client):
        """Test searching skills."""
        mock_response = {
            "skills": [
                {
                    "id": "skill-1",
                    "name": "Network Repair",
                    "description": "Repairs network connectivity",
                    "cost": 100
                }
            ],
            "total": 1
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills/search",
            json=mock_response,
            status=200
        )

        results = client.economics.skills.search(query="network", category="repair")
        
        assert len(results["skills"]) == 1
        assert results["skills"][0]["name"] == "Network Repair"


class TestEconomicsLLMService:
    """Test the Economics LLM service."""

    @responses.activate
    def test_list_models(self, client):
        """Test listing LLM models."""
        mock_response = {
            "models": [
                {
                    "id": "model-1",
                    "name": "GPT-4",
                    "cost_per_token": 0.00003,
                    "max_tokens": 8192,
                    "provider": "openai"
                },
                {
                    "id": "model-2",
                    "name": "Claude-3",
                    "cost_per_token": 0.000015,
                    "max_tokens": 4096,
                    "provider": "anthropic"
                }
            ]
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/llm/models",
            json=mock_response,
            status=200
        )

        models = client.economics.llm.list_models()
        
        assert len(models["models"]) == 2
        assert models["models"][0]["name"] == "GPT-4"
        assert models["models"][1]["name"] == "Claude-3"

    @responses.activate
    def test_get_usage(self, client):
        """Test getting LLM usage statistics."""
        mock_response = {
            "total_tokens": 1500000,
            "total_cost": 45.50,
            "requests": 2500,
            "period": "2024-01",
            "breakdown": {
                "gpt-4": {"tokens": 800000, "cost": 24.00},
                "claude-3": {"tokens": 700000, "cost": 21.50}
            }
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/llm/usage",
            json=mock_response,
            status=200
        )

        usage = client.economics.llm.get_usage()
        
        assert usage["total_tokens"] == 1500000
        assert usage["total_cost"] == 45.50
        assert usage["requests"] == 2500
        assert "breakdown" in usage

    @responses.activate
    def test_estimate_cost(self, client):
        """Test estimating LLM cost."""
        mock_response = {
            "estimated_cost": 0.15,
            "token_count": 5000,
            "model": "gpt-4"
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/economics/llm/estimate",
            json=mock_response,
            status=200
        )

        estimate = client.economics.llm.estimate_cost(
            text="This is a test prompt for cost estimation",
            model="gpt-4"
        )
        
        assert estimate["estimated_cost"] == 0.15
        assert estimate["token_count"] == 5000
        assert estimate["model"] == "gpt-4"


class TestEconomicsValidationService:
    """Test the Economics Validation service."""

    @responses.activate
    def test_validate_skill(self, client):
        """Test validating a skill."""
        mock_response = {
            "valid": True,
            "confidence": 0.95,
            "errors": [],
            "warnings": ["Cost is higher than average for this category"]
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/economics/validation/validate",
            json=mock_response,
            status=200
        )

        validation_data = {
            "skill_id": "skill-1",
            "data": {
                "cost": 100,
                "name": "Test Skill",
                "category": "testing"
            }
        }

        result = client.economics.validation.validate(validation_data)
        
        assert result["valid"] is True
        assert result["confidence"] == 0.95
        assert len(result["errors"]) == 0
        assert len(result["warnings"]) == 1

    @responses.activate
    def test_validate_skill_with_errors(self, client):
        """Test validating a skill with errors."""
        mock_response = {
            "valid": False,
            "confidence": 0.2,
            "errors": [
                "Cost cannot be negative",
                "Name is required"
            ],
            "warnings": []
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/economics/validation/validate",
            json=mock_response,
            status=200
        )

        validation_data = {
            "skill_id": "skill-1",
            "data": {
                "cost": -50,
                "name": ""
            }
        }

        result = client.economics.validation.validate(validation_data)
        
        assert result["valid"] is False
        assert result["confidence"] == 0.2
        assert len(result["errors"]) == 2
        assert "Cost cannot be negative" in result["errors"]

    @responses.activate
    def test_list_validation_rules(self, client):
        """Test listing validation rules."""
        mock_response = {
            "rules": [
                {
                    "id": "rule-1",
                    "name": "Cost Validation",
                    "description": "Validates skill cost ranges",
                    "active": True,
                    "category": "cost"
                },
                {
                    "id": "rule-2",
                    "name": "Name Validation",
                    "description": "Validates skill names",
                    "active": True,
                    "category": "naming"
                }
            ]
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/validation/rules",
            json=mock_response,
            status=200
        )

        rules = client.economics.validation.list_rules()
        
        assert len(rules["rules"]) == 2
        assert rules["rules"][0]["name"] == "Cost Validation"
        assert rules["rules"][1]["name"] == "Name Validation"


class TestEconomicsErrorHandling:
    """Test error handling in Economics service."""

    @responses.activate
    def test_network_error(self, client):
        """Test handling network errors."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills",
            body=ConnectionError("Network error")
        )

        with pytest.raises(KNIRVAPIError):
            client.economics.skills.list()

    @responses.activate
    def test_server_error(self, client):
        """Test handling server errors."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/economics/skills",
            json={"error": "Internal server error"},
            status=500
        )

        with pytest.raises(KNIRVAPIError) as exc_info:
            client.economics.skills.list()
        
        assert exc_info.value.status_code == 500

    @responses.activate
    def test_timeout_error(self, client):
        """Test handling timeout errors."""
        with patch('requests.Session.request') as mock_request:
            mock_request.side_effect = TimeoutError("Request timed out")
            
            with pytest.raises(KNIRVAPIError):
                client.economics.skills.list()

    def test_invalid_parameters(self, client):
        """Test handling invalid parameters."""
        with pytest.raises(ValueError):
            client.economics.skills.get("")  # Empty skill ID
        
        with pytest.raises(ValueError):
            client.economics.skills.list(page=0)  # Invalid page number
        
        with pytest.raises(ValueError):
            client.economics.skills.list(per_page=0)  # Invalid per_page
