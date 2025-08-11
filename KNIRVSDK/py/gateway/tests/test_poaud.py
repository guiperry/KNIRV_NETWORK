"""
Comprehensive tests for the KNIRV Gateway PoAuD (Proof of Autonomous Delivery) service
"""

import pytest
import responses
import json
from datetime import datetime, timedelta
from unittest.mock import Mock, patch
from knirv_gateway_sdk import KNIRVGateway
from knirv_gateway_sdk.exceptions import KNIRVAPIError, KNIRVValidationError


@pytest.fixture
def client():
    """Create a test client."""
    return KNIRVGateway(
        base_url="https://test.knirv.com",
        api_key="test-key"
    )


@pytest.fixture
def mock_proof_data():
    """Mock proof data for testing."""
    return {
        "id": "proof-1",
        "skill_id": "skill-1",
        "user_id": "user-1",
        "timestamp": int(datetime.now().timestamp()),
        "verified": True,
        "confidence": 0.95,
        "evidence": {
            "execution_time": 1500,
            "success_rate": 0.98,
            "output_quality": 0.92
        },
        "validator_id": "validator-1",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }


@pytest.fixture
def mock_challenge_data():
    """Mock challenge data for testing."""
    return {
        "id": "challenge-1",
        "skill_id": "skill-1",
        "difficulty": "medium",
        "reward": 100,
        "deadline": int((datetime.now() + timedelta(days=1)).timestamp()),
        "description": "Repair network connectivity issue",
        "requirements": {
            "min_reputation": 500,
            "required_skills": ["network", "troubleshooting"]
        },
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z"
    }


class TestPoAuDProofsService:
    """Test the PoAuD Proofs service."""

    @responses.activate
    def test_list_proofs(self, client):
        """Test listing proofs."""
        mock_response = {
            "proofs": [
                {
                    "id": "proof-1",
                    "skill_id": "skill-1",
                    "user_id": "user-1",
                    "verified": True,
                    "confidence": 0.95
                },
                {
                    "id": "proof-2",
                    "skill_id": "skill-2",
                    "user_id": "user-2",
                    "verified": False,
                    "confidence": 0.75
                }
            ],
            "total": 2,
            "page": 1,
            "per_page": 10
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/proofs",
            json=mock_response,
            status=200
        )

        proofs = client.poaud.proofs.list()
        
        assert len(proofs["proofs"]) == 2
        assert proofs["proofs"][0]["verified"] is True
        assert proofs["proofs"][1]["verified"] is False
        assert proofs["total"] == 2

    @responses.activate
    def test_list_proofs_with_filters(self, client):
        """Test listing proofs with filters."""
        mock_response = {
            "proofs": [
                {
                    "id": "proof-1",
                    "skill_id": "skill-1",
                    "user_id": "user-1",
                    "verified": True
                }
            ],
            "total": 1
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/proofs",
            json=mock_response,
            status=200
        )

        proofs = client.poaud.proofs.list(
            skill_id="skill-1",
            verified=True,
            user_id="user-1"
        )
        
        assert len(proofs["proofs"]) == 1
        assert proofs["proofs"][0]["skill_id"] == "skill-1"

    @responses.activate
    def test_get_proof(self, client, mock_proof_data):
        """Test getting a specific proof."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/proofs/proof-1",
            json=mock_proof_data,
            status=200
        )

        proof = client.poaud.proofs.get("proof-1")
        
        assert proof["id"] == "proof-1"
        assert proof["verified"] is True
        assert proof["confidence"] == 0.95
        assert "evidence" in proof

    @responses.activate
    def test_create_proof(self, client):
        """Test creating a new proof."""
        proof_data = {
            "skill_id": "skill-1",
            "user_id": "user-1",
            "evidence": {
                "execution_time": 1200,
                "success_rate": 0.96,
                "logs": ["step1", "step2", "step3"]
            }
        }
        
        mock_response = {
            "id": "proof-3",
            "created": True,
            "status": "pending_verification",
            **proof_data
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/proofs",
            json=mock_response,
            status=201
        )

        result = client.poaud.proofs.create(proof_data)
        
        assert result["id"] == "proof-3"
        assert result["created"] is True
        assert result["status"] == "pending_verification"

    @responses.activate
    def test_update_proof(self, client):
        """Test updating a proof."""
        update_data = {
            "verified": True,
            "confidence": 0.97,
            "validator_notes": "Proof verified successfully"
        }
        
        responses.add(
            responses.PUT,
            "https://test.knirv.com/poaud/proofs/proof-1",
            json={"updated": True},
            status=200
        )

        result = client.poaud.proofs.update("proof-1", update_data)
        assert result["updated"] is True

    @responses.activate
    def test_verify_proof(self, client):
        """Test verifying a proof."""
        verification_data = {
            "proof_id": "proof-1",
            "evidence": {
                "validator_id": "validator-1",
                "notes": "Proof verified successfully",
                "verification_method": "automated"
            }
        }
        
        mock_response = {
            "verification_id": "verify-1",
            "status": "verified",
            "confidence": 0.93,
            "timestamp": int(datetime.now().timestamp())
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/verify",
            json=mock_response,
            status=200
        )

        result = client.poaud.verify_proof(verification_data)
        
        assert result["status"] == "verified"
        assert result["confidence"] == 0.93
        assert "verification_id" in result


class TestPoAuDChallengesService:
    """Test the PoAuD Challenges service."""

    @responses.activate
    def test_list_challenges(self, client):
        """Test listing challenges."""
        mock_response = {
            "challenges": [
                {
                    "id": "challenge-1",
                    "skill_id": "skill-1",
                    "difficulty": "medium",
                    "reward": 100,
                    "status": "active"
                },
                {
                    "id": "challenge-2",
                    "skill_id": "skill-2",
                    "difficulty": "hard",
                    "reward": 200,
                    "status": "completed"
                }
            ],
            "total": 2
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/challenges",
            json=mock_response,
            status=200
        )

        challenges = client.poaud.challenges.list()
        
        assert len(challenges["challenges"]) == 2
        assert challenges["challenges"][0]["difficulty"] == "medium"
        assert challenges["challenges"][1]["difficulty"] == "hard"

    @responses.activate
    def test_get_challenge(self, client, mock_challenge_data):
        """Test getting a specific challenge."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/challenges/challenge-1",
            json=mock_challenge_data,
            status=200
        )

        challenge = client.poaud.challenges.get("challenge-1")
        
        assert challenge["id"] == "challenge-1"
        assert challenge["difficulty"] == "medium"
        assert challenge["reward"] == 100
        assert "requirements" in challenge

    @responses.activate
    def test_create_challenge(self, client):
        """Test creating a new challenge."""
        challenge_data = {
            "skill_id": "skill-2",
            "difficulty": "hard",
            "reward": 200,
            "description": "Complex data analysis task",
            "deadline": int((datetime.now() + timedelta(days=2)).timestamp()),
            "requirements": {
                "min_reputation": 750,
                "required_skills": ["data_analysis", "machine_learning"]
            }
        }
        
        mock_response = {
            "id": "challenge-2",
            "created": True,
            **challenge_data
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/challenges",
            json=mock_response,
            status=201
        )

        result = client.poaud.challenges.create(challenge_data)
        
        assert result["id"] == "challenge-2"
        assert result["created"] is True
        assert result["difficulty"] == "hard"

    @responses.activate
    def test_submit_challenge(self, client):
        """Test submitting a challenge solution."""
        submission_data = {
            "challenge_id": "challenge-1",
            "user_id": "user-1",
            "solution": {
                "approach": "automated repair",
                "steps": ["diagnose", "repair", "verify"],
                "result": "success",
                "execution_time": 1800,
                "logs": ["Connected to system", "Issue identified", "Repair completed"]
            }
        }
        
        mock_response = {
            "submission_id": "submission-1",
            "status": "submitted",
            "timestamp": int(datetime.now().timestamp()),
            "estimated_review_time": 3600
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/challenges/challenge-1/submit",
            json=mock_response,
            status=200
        )

        result = client.poaud.challenges.submit("challenge-1", submission_data)
        
        assert result["status"] == "submitted"
        assert "submission_id" in result
        assert "estimated_review_time" in result

    @responses.activate
    def test_get_challenge_submissions(self, client):
        """Test getting challenge submissions."""
        mock_response = {
            "submissions": [
                {
                    "id": "submission-1",
                    "user_id": "user-1",
                    "status": "submitted",
                    "score": None
                },
                {
                    "id": "submission-2",
                    "user_id": "user-2",
                    "status": "reviewed",
                    "score": 85
                }
            ],
            "total": 2
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/challenges/challenge-1/submissions",
            json=mock_response,
            status=200
        )

        submissions = client.poaud.challenges.get_submissions("challenge-1")
        
        assert len(submissions["submissions"]) == 2
        assert submissions["submissions"][0]["status"] == "submitted"
        assert submissions["submissions"][1]["status"] == "reviewed"


class TestPoAuDReputationService:
    """Test the PoAuD Reputation service."""

    @responses.activate
    def test_get_user_reputation(self, client):
        """Test getting user reputation."""
        mock_response = {
            "user_id": "user-1",
            "reputation_score": 850,
            "skill_ratings": {
                "skill-1": 0.95,
                "skill-2": 0.88,
                "skill-3": 0.92
            },
            "total_proofs": 25,
            "verified_proofs": 23,
            "success_rate": 0.92,
            "challenges_completed": 15,
            "challenges_won": 12,
            "rank": "expert",
            "badges": ["network_specialist", "quick_resolver"]
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/reputation/user-1",
            json=mock_response,
            status=200
        )

        reputation = client.poaud.reputation.get_user_reputation("user-1")
        
        assert reputation["user_id"] == "user-1"
        assert reputation["reputation_score"] == 850
        assert len(reputation["skill_ratings"]) == 3
        assert reputation["success_rate"] == 0.92
        assert "badges" in reputation

    @responses.activate
    def test_get_leaderboard(self, client):
        """Test getting reputation leaderboard."""
        mock_response = {
            "leaderboard": [
                {
                    "user_id": "user-1",
                    "reputation_score": 950,
                    "rank": 1
                },
                {
                    "user_id": "user-2",
                    "reputation_score": 850,
                    "rank": 2
                },
                {
                    "user_id": "user-3",
                    "reputation_score": 750,
                    "rank": 3
                }
            ],
            "total_users": 100,
            "page": 1,
            "per_page": 10
        }
        
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/reputation/leaderboard",
            json=mock_response,
            status=200
        )

        leaderboard = client.poaud.reputation.get_leaderboard()
        
        assert len(leaderboard["leaderboard"]) == 3
        assert leaderboard["leaderboard"][0]["rank"] == 1
        assert leaderboard["total_users"] == 100

    @responses.activate
    def test_update_reputation(self, client):
        """Test updating user reputation."""
        update_data = {
            "user_id": "user-1",
            "skill_id": "skill-1",
            "performance_score": 0.95,
            "proof_id": "proof-1"
        }
        
        mock_response = {
            "updated": True,
            "new_reputation_score": 875,
            "change": 25
        }
        
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/reputation/update",
            json=mock_response,
            status=200
        )

        result = client.poaud.reputation.update_reputation(update_data)
        
        assert result["updated"] is True
        assert result["new_reputation_score"] == 875
        assert result["change"] == 25


class TestPoAuDValidation:
    """Test PoAuD validation and error handling."""

    def test_proof_validation(self, client):
        """Test proof data validation."""
        # Test missing required fields
        with pytest.raises(ValueError):
            client.poaud.proofs.create({})  # Missing skill_id and user_id
        
        with pytest.raises(ValueError):
            client.poaud.proofs.create({"skill_id": "skill-1"})  # Missing user_id

    def test_challenge_validation(self, client):
        """Test challenge data validation."""
        # Test invalid reward
        with pytest.raises(ValueError):
            client.poaud.challenges.create({
                "skill_id": "skill-1",
                "difficulty": "medium",
                "reward": -100  # Negative reward
            })
        
        # Test invalid difficulty
        with pytest.raises(ValueError):
            client.poaud.challenges.create({
                "skill_id": "skill-1",
                "difficulty": "invalid",  # Invalid difficulty level
                "reward": 100
            })

    @responses.activate
    def test_api_error_handling(self, client):
        """Test API error handling."""
        responses.add(
            responses.GET,
            "https://test.knirv.com/poaud/proofs/nonexistent",
            json={"error": "Proof not found", "code": "PROOF_NOT_FOUND"},
            status=404
        )

        with pytest.raises(KNIRVAPIError) as exc_info:
            client.poaud.proofs.get("nonexistent")
        
        assert exc_info.value.status_code == 404
        assert "Proof not found" in str(exc_info.value)

    @responses.activate
    def test_validation_error_handling(self, client):
        """Test validation error handling."""
        responses.add(
            responses.POST,
            "https://test.knirv.com/poaud/proofs",
            json={
                "error": "Validation failed",
                "code": "VALIDATION_ERROR",
                "details": {
                    "skill_id": ["This field is required"],
                    "evidence": ["Evidence must contain execution data"]
                }
            },
            status=400
        )

        with pytest.raises(KNIRVValidationError) as exc_info:
            client.poaud.proofs.create({})
        
        assert exc_info.value.status_code == 400
        assert "Validation failed" in str(exc_info.value)
