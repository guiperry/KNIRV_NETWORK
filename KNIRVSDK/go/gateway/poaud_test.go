package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway/option"
)

func TestPoAuDService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poaud/proofs":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"proofs": []map[string]interface{}{
						{
							"id":         "proof-1",
							"skill_id":   "skill-1",
							"user_id":    "user-1",
							"timestamp":  time.Now().Unix(),
							"verified":   true,
							"confidence": 0.95,
						},
						{
							"id":         "proof-2",
							"skill_id":   "skill-2",
							"user_id":    "user-2",
							"timestamp":  time.Now().Unix(),
							"verified":   false,
							"confidence": 0.75,
						},
					},
				})
			} else if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "proof-3",
					"created": true,
					"status":  "pending_verification",
				})
			}
		case "/poaud/proofs/proof-1":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":         "proof-1",
					"skill_id":   "skill-1",
					"user_id":    "user-1",
					"timestamp":  time.Now().Unix(),
					"verified":   true,
					"confidence": 0.95,
					"evidence": map[string]interface{}{
						"execution_time": 1500,
						"success_rate":   0.98,
						"output_quality": 0.92,
					},
				})
			} else if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"updated": true,
				})
			}
		case "/poaud/verify":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"verification_id": "verify-1",
				"status":          "verified",
				"confidence":      0.93,
				"timestamp":       time.Now().Unix(),
			})
		case "/poaud/challenges":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"challenges": []map[string]interface{}{
						{
							"id":          "challenge-1",
							"skill_id":    "skill-1",
							"difficulty":  "medium",
							"reward":      100,
							"deadline":    time.Now().Add(24 * time.Hour).Unix(),
							"description": "Repair network connectivity issue",
						},
					},
				})
			} else if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "challenge-2",
					"created": true,
				})
			}
		case "/poaud/challenges/challenge-1/submit":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"submission_id": "submission-1",
				"status":        "submitted",
				"timestamp":     time.Now().Unix(),
			})
		case "/poaud/reputation/user-1":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user_id":          "user-1",
				"reputation_score": 850,
				"skill_ratings": map[string]interface{}{
					"skill-1": 0.95,
					"skill-2": 0.88,
				},
				"total_proofs":    25,
				"verified_proofs": 23,
				"success_rate":    0.92,
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

	t.Run("List proofs", func(t *testing.T) {
		proofs, err := client.PoAuD.ListProofs(ctx)
		if err != nil {
			t.Errorf("Failed to list proofs: %v", err)
		}

		if len(proofs) != 2 {
			t.Errorf("Expected 2 proofs, got %d", len(proofs))
		}

		if proofs[0].ID != "proof-1" {
			t.Errorf("Expected first proof ID 'proof-1', got %s", proofs[0].ID)
		}

		if !proofs[0].Verified {
			t.Error("Expected first proof to be verified")
		}

		if proofs[1].Verified {
			t.Error("Expected second proof to not be verified")
		}
	})

	t.Run("Get proof by ID", func(t *testing.T) {
		proof, err := client.PoAuD.GetProof(ctx, "proof-1")
		if err != nil {
			t.Errorf("Failed to get proof: %v", err)
		}

		if proof.ID != "proof-1" {
			t.Errorf("Expected proof ID 'proof-1', got %s", proof.ID)
		}

		if proof.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", proof.Confidence)
		}

		if proof.Evidence == nil {
			t.Error("Expected evidence to be present")
		}
	})

	t.Run("Create proof", func(t *testing.T) {
		proofReq := &ProofCreateRequest{
			SkillID: "skill-1",
			UserID:  "user-1",
			Evidence: map[string]interface{}{
				"execution_time": 1200,
				"success_rate":   0.96,
			},
		}

		result, err := client.PoAuD.CreateProof(ctx, proofReq)
		if err != nil {
			t.Errorf("Failed to create proof: %v", err)
		}

		if result.ID != "proof-3" {
			t.Errorf("Expected created proof ID 'proof-3', got %s", result.ID)
		}

		if result.Status != "pending_verification" {
			t.Errorf("Expected status 'pending_verification', got %s", result.Status)
		}
	})

	t.Run("Update proof", func(t *testing.T) {
		updateReq := &ProofUpdateRequest{
			Verified:   true,
			Confidence: 0.97,
		}

		err := client.PoAuD.UpdateProof(ctx, "proof-1", updateReq)
		if err != nil {
			t.Errorf("Failed to update proof: %v", err)
		}
	})

	t.Run("Verify proof", func(t *testing.T) {
		verifyReq := &VerificationRequest{
			ProofID: "proof-1",
			Evidence: map[string]interface{}{
				"validator_id": "validator-1",
				"notes":        "Proof verified successfully",
			},
		}

		result, err := client.PoAuD.VerifyProof(ctx, verifyReq)
		if err != nil {
			t.Errorf("Failed to verify proof: %v", err)
		}

		if result.Status != "verified" {
			t.Errorf("Expected status 'verified', got %s", result.Status)
		}

		if result.Confidence != 0.93 {
			t.Errorf("Expected confidence 0.93, got %f", result.Confidence)
		}
	})

	t.Run("List challenges", func(t *testing.T) {
		challenges, err := client.PoAuD.ListChallenges(ctx)
		if err != nil {
			t.Errorf("Failed to list challenges: %v", err)
		}

		if len(challenges) != 1 {
			t.Errorf("Expected 1 challenge, got %d", len(challenges))
		}

		if challenges[0].ID != "challenge-1" {
			t.Errorf("Expected challenge ID 'challenge-1', got %s", challenges[0].ID)
		}

		if challenges[0].Difficulty != "medium" {
			t.Errorf("Expected difficulty 'medium', got %s", challenges[0].Difficulty)
		}
	})

	t.Run("Create challenge", func(t *testing.T) {
		challengeReq := &ChallengeCreateRequest{
			SkillID:     "skill-2",
			Difficulty:  "hard",
			Reward:      200,
			Description: "Complex data analysis task",
			Deadline:    time.Now().Add(48 * time.Hour),
		}

		result, err := client.PoAuD.CreateChallenge(ctx, challengeReq)
		if err != nil {
			t.Errorf("Failed to create challenge: %v", err)
		}

		if result.ID != "challenge-2" {
			t.Errorf("Expected created challenge ID 'challenge-2', got %s", result.ID)
		}
	})

	t.Run("Submit challenge solution", func(t *testing.T) {
		submissionReq := &ChallengeSubmissionRequest{
			ChallengeID: "challenge-1",
			UserID:      "user-1",
			Solution: map[string]interface{}{
				"approach": "automated repair",
				"steps":    []string{"diagnose", "repair", "verify"},
				"result":   "success",
			},
		}

		result, err := client.PoAuD.SubmitChallenge(ctx, submissionReq)
		if err != nil {
			t.Errorf("Failed to submit challenge: %v", err)
		}

		if result.Status != "submitted" {
			t.Errorf("Expected status 'submitted', got %s", result.Status)
		}
	})

	t.Run("Get user reputation", func(t *testing.T) {
		reputation, err := client.PoAuD.GetUserReputation(ctx, "user-1")
		if err != nil {
			t.Errorf("Failed to get user reputation: %v", err)
		}

		if reputation.UserID != "user-1" {
			t.Errorf("Expected user ID 'user-1', got %s", reputation.UserID)
		}

		if reputation.ReputationScore != 850 {
			t.Errorf("Expected reputation score 850, got %d", reputation.ReputationScore)
		}

		if reputation.TotalProofs != 25 {
			t.Errorf("Expected total proofs 25, got %d", reputation.TotalProofs)
		}

		if reputation.VerifiedProofs != 23 {
			t.Errorf("Expected verified proofs 23, got %d", reputation.VerifiedProofs)
		}

		if len(reputation.SkillRatings) != 2 {
			t.Errorf("Expected 2 skill ratings, got %d", len(reputation.SkillRatings))
		}
	})
}

func TestPoAuDValidation(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("Validate proof creation request", func(t *testing.T) {
		// Test with invalid request (missing required fields)
		invalidReq := &ProofCreateRequest{}

		err := client.PoAuD.validateProofCreateRequest(invalidReq)
		if err == nil {
			t.Error("Expected validation error for invalid request")
		}

		// Test with valid request
		validReq := &ProofCreateRequest{
			SkillID: "skill-1",
			UserID:  "user-1",
			Evidence: map[string]interface{}{
				"execution_time": 1200,
			},
		}

		err = client.PoAuD.validateProofCreateRequest(validReq)
		if err != nil {
			t.Errorf("Expected no validation error for valid request: %v", err)
		}
	})

	t.Run("Validate challenge creation request", func(t *testing.T) {
		// Test with invalid request
		invalidReq := &ChallengeCreateRequest{
			Reward: -100, // Invalid negative reward
		}

		err := client.PoAuD.validateChallengeCreateRequest(invalidReq)
		if err == nil {
			t.Error("Expected validation error for negative reward")
		}

		// Test with valid request
		validReq := &ChallengeCreateRequest{
			SkillID:     "skill-1",
			Difficulty:  "medium",
			Reward:      100,
			Description: "Test challenge",
			Deadline:    time.Now().Add(24 * time.Hour),
		}

		err = client.PoAuD.validateChallengeCreateRequest(validReq)
		if err != nil {
			t.Errorf("Expected no validation error for valid request: %v", err)
		}
	})
}

func TestPoAuDErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/poaud/proofs/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Proof not found",
				"code":  "PROOF_NOT_FOUND",
			})
		case "/poaud/proofs":
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Invalid proof data",
					"code":  "INVALID_PROOF_DATA",
				})
			}
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client, err := NewClient(option.WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Handle not found error", func(t *testing.T) {
		_, err := client.PoAuD.GetProof(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent proof")
		}
	})

	t.Run("Handle bad request error", func(t *testing.T) {
		invalidReq := &ProofCreateRequest{} // Invalid empty request

		_, err := client.PoAuD.CreateProof(ctx, invalidReq)
		if err == nil {
			t.Error("Expected error for invalid proof creation")
		}
	})
}
