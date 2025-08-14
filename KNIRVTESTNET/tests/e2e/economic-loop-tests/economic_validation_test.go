package economicloop

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EconomicLoopTestSuite manages economic loop validation testing
type EconomicLoopTestSuite struct {
	BaseURL       string
	Context       context.Context
	InitialState  EconomicState
	TestAccounts  []TestAccount
	TestSkills    []TestSkill
	Transactions  []Transaction
}

// EconomicState represents the economic state of the testnet
type EconomicState struct {
	TotalNRNSupply    float64 `json:"total_nrn_supply"`
	CirculatingSupply float64 `json:"circulating_supply"`
	TotalStaked       float64 `json:"total_staked"`
	TotalBurned       float64 `json:"total_burned"`
	NetworkFees       float64 `json:"network_fees"`
	RewardPool        float64 `json:"reward_pool"`
	Timestamp         time.Time `json:"timestamp"`
}

// TestAccount represents a test account for economic testing
type TestAccount struct {
	ID          string  `json:"id"`
	Address     string  `json:"address"`
	NRNBalance  float64 `json:"nrn_balance"`
	StakedNRN   float64 `json:"staked_nrn"`
	Role        string  `json:"role"` // "user", "validator", "developer", "agent"
	Reputation  float64 `json:"reputation"`
}

// TestSkill represents a skill for economic testing
type TestSkill struct {
	ID           string  `json:"id"`
	Creator      string  `json:"creator"`
	Price        float64 `json:"price"`
	UsageCount   int     `json:"usage_count"`
	TotalRevenue float64 `json:"total_revenue"`
	Rating       float64 `json:"rating"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    float64   `json:"amount"`
	Fee       float64   `json:"fee"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewEconomicLoopTestSuite creates a new economic test suite
func NewEconomicLoopTestSuite() *EconomicLoopTestSuite {
	return &EconomicLoopTestSuite{
		BaseURL: "http://localhost:1317", // KNIRV-ROOT endpoint
		Context: context.Background(),
		TestAccounts: []TestAccount{
			{ID: "validator_001", Role: "validator", NRNBalance: 10000, StakedNRN: 5000},
			{ID: "developer_001", Role: "developer", NRNBalance: 1000, StakedNRN: 0},
			{ID: "user_001", Role: "user", NRNBalance: 500, StakedNRN: 0},
			{ID: "agent_001", Role: "agent", NRNBalance: 100, StakedNRN: 0},
		},
		TestSkills: []TestSkill{
			{ID: "skill_001", Creator: "developer_001", Price: 50, UsageCount: 0, TotalRevenue: 0},
			{ID: "skill_002", Creator: "developer_001", Price: 100, UsageCount: 0, TotalRevenue: 0},
		},
	}
}

// TestNRNTokenFlow tests the complete NRN token flow cycle
func TestNRNTokenFlow(t *testing.T) {
	suite := NewEconomicLoopTestSuite()
	
	// Capture initial state
	initialState, err := suite.captureEconomicState()
	require.NoError(t, err)
	suite.InitialState = *initialState

	t.Run("Token Minting", func(t *testing.T) {
		// Test NRN minting through connectivity proofs
		mintData := map[string]interface{}{
			"proof_type":   "connectivity",
			"validator":    "validator_001",
			"amount":       100.0,
			"proof_data":   "mock_connectivity_proof",
		}

		response, err := suite.makeBlockchainCall("POST", "/mint", mintData)
		require.NoError(t, err)

		var mintResponse struct {
			Success       bool    `json:"success"`
			TransactionID string  `json:"transaction_id"`
			AmountMinted  float64 `json:"amount_minted"`
			NewSupply     float64 `json:"new_supply"`
		}

		err = json.Unmarshal(response, &mintResponse)
		require.NoError(t, err)
		assert.True(t, mintResponse.Success)
		assert.Equal(t, 100.0, mintResponse.AmountMinted)
		assert.Greater(t, mintResponse.NewSupply, initialState.TotalNRNSupply)
	})

	t.Run("Token Distribution", func(t *testing.T) {
		// Test token distribution to various participants
		distributions := []struct {
			recipient string
			amount    float64
			reason    string
		}{
			{"developer_001", 50.0, "skill_creation_reward"},
			{"validator_001", 30.0, "validation_reward"},
			{"user_001", 20.0, "participation_reward"},
		}

		for _, dist := range distributions {
			distributionData := map[string]interface{}{
				"recipient": dist.recipient,
				"amount":    dist.amount,
				"reason":    dist.reason,
			}

			response, err := suite.makeBlockchainCall("POST", "/distribute", distributionData)
			require.NoError(t, err)

			var distResponse struct {
				Success       bool   `json:"success"`
				TransactionID string `json:"transaction_id"`
				Recipient     string `json:"recipient"`
				Amount        float64 `json:"amount"`
			}

			err = json.Unmarshal(response, &distResponse)
			require.NoError(t, err)
			assert.True(t, distResponse.Success)
			assert.Equal(t, dist.recipient, distResponse.Recipient)
			assert.Equal(t, dist.amount, distResponse.Amount)
		}
	})

	t.Run("Token Usage", func(t *testing.T) {
		// Test token usage for skill execution
		usageData := map[string]interface{}{
			"user":     "user_001",
			"skill_id": "skill_001",
			"amount":   50.0,
		}

		response, err := suite.makeBlockchainCall("POST", "/spend", usageData)
		require.NoError(t, err)

		var usageResponse struct {
			Success       bool   `json:"success"`
			TransactionID string `json:"transaction_id"`
			AmountSpent   float64 `json:"amount_spent"`
			Recipient     string `json:"recipient"`
		}

		err = json.Unmarshal(response, &usageResponse)
		require.NoError(t, err)
		assert.True(t, usageResponse.Success)
		assert.Equal(t, 50.0, usageResponse.AmountSpent)
	})

	t.Run("Token Burning", func(t *testing.T) {
		// Test token burning mechanism
		burnData := map[string]interface{}{
			"amount": 25.0,
			"reason": "network_fee_burn",
		}

		response, err := suite.makeBlockchainCall("POST", "/burn", burnData)
		require.NoError(t, err)

		var burnResponse struct {
			Success       bool    `json:"success"`
			TransactionID string  `json:"transaction_id"`
			AmountBurned  float64 `json:"amount_burned"`
			NewSupply     float64 `json:"new_supply"`
		}

		err = json.Unmarshal(response, &burnResponse)
		require.NoError(t, err)
		assert.True(t, burnResponse.Success)
		assert.Equal(t, 25.0, burnResponse.AmountBurned)
	})

	t.Run("Economic State Validation", func(t *testing.T) {
		// Validate final economic state
		finalState, err := suite.captureEconomicState()
		require.NoError(t, err)

		// Verify supply changes
		expectedSupplyIncrease := 100.0 - 25.0 // minted - burned
		actualSupplyChange := finalState.TotalNRNSupply - initialState.TotalNRNSupply
		assert.InDelta(t, expectedSupplyIncrease, actualSupplyChange, 1.0)

		// Verify circulation
		assert.Greater(t, finalState.CirculatingSupply, initialState.CirculatingSupply)

		// Verify burn tracking
		assert.Greater(t, finalState.TotalBurned, initialState.TotalBurned)
	})
}

// TestRewardMechanisms tests various reward distribution mechanisms
func TestRewardMechanisms(t *testing.T) {
	suite := NewEconomicLoopTestSuite()

	t.Run("Skill Creation Rewards", func(t *testing.T) {
		// Test rewards for creating skills
		skillData := map[string]interface{}{
			"creator":     "developer_001",
			"skill_name":  "Advanced Analytics",
			"complexity":  "high",
			"category":    "data_science",
		}

		response, err := suite.makeBlockchainCall("POST", "/skills/create_reward", skillData)
		require.NoError(t, err)

		var rewardResponse struct {
			Success     bool    `json:"success"`
			Creator     string  `json:"creator"`
			RewardAmount float64 `json:"reward_amount"`
			RewardType  string  `json:"reward_type"`
		}

		err = json.Unmarshal(response, &rewardResponse)
		require.NoError(t, err)
		assert.True(t, rewardResponse.Success)
		assert.Equal(t, "developer_001", rewardResponse.Creator)
		assert.Greater(t, rewardResponse.RewardAmount, 0.0)
		assert.Equal(t, "skill_creation", rewardResponse.RewardType)
	})

	t.Run("Validation Rewards", func(t *testing.T) {
		// Test rewards for skill validation
		validationData := map[string]interface{}{
			"validator": "validator_001",
			"skill_id":  "skill_001",
			"result":    "approved",
			"quality_score": 0.95,
		}

		response, err := suite.makeBlockchainCall("POST", "/skills/validation_reward", validationData)
		require.NoError(t, err)

		var validationResponse struct {
			Success      bool    `json:"success"`
			Validator    string  `json:"validator"`
			RewardAmount float64 `json:"reward_amount"`
			QualityBonus float64 `json:"quality_bonus"`
		}

		err = json.Unmarshal(response, &validationResponse)
		require.NoError(t, err)
		assert.True(t, validationResponse.Success)
		assert.Equal(t, "validator_001", validationResponse.Validator)
		assert.Greater(t, validationResponse.RewardAmount, 0.0)
	})

	t.Run("Usage-Based Rewards", func(t *testing.T) {
		// Test rewards based on skill usage
		usageData := map[string]interface{}{
			"skill_id":     "skill_001",
			"usage_count":  100,
			"total_revenue": 5000.0,
			"period":       "monthly",
		}

		response, err := suite.makeBlockchainCall("POST", "/skills/usage_reward", usageData)
		require.NoError(t, err)

		var usageResponse struct {
			Success       bool    `json:"success"`
			SkillID       string  `json:"skill_id"`
			CreatorReward float64 `json:"creator_reward"`
			NetworkFee    float64 `json:"network_fee"`
		}

		err = json.Unmarshal(response, &usageResponse)
		require.NoError(t, err)
		assert.True(t, usageResponse.Success)
		assert.Equal(t, "skill_001", usageResponse.SkillID)
		assert.Greater(t, usageResponse.CreatorReward, 0.0)
		assert.Greater(t, usageResponse.NetworkFee, 0.0)
	})

	t.Run("Collaborative Rewards", func(t *testing.T) {
		// Test rewards for agent collaboration
		collaborationData := map[string]interface{}{
			"session_id":    "collab_001",
			"participants":  []string{"agent_001", "agent_002", "agent_003"},
			"task_complexity": "high",
			"success_rate":  0.92,
			"duration":      "2h30m",
		}

		response, err := suite.makeBlockchainCall("POST", "/collaboration/reward", collaborationData)
		require.NoError(t, err)

		var collabResponse struct {
			Success     bool                       `json:"success"`
			SessionID   string                     `json:"session_id"`
			TotalReward float64                    `json:"total_reward"`
			Distribution map[string]float64        `json:"distribution"`
		}

		err = json.Unmarshal(response, &collabResponse)
		require.NoError(t, err)
		assert.True(t, collabResponse.Success)
		assert.Equal(t, "collab_001", collabResponse.SessionID)
		assert.Greater(t, collabResponse.TotalReward, 0.0)
		assert.Len(t, collabResponse.Distribution, 3)
	})
}

// TestCrossChainOperations tests cross-chain economic operations
func TestCrossChainOperations(t *testing.T) {
	suite := NewEconomicLoopTestSuite()

	t.Run("XION Bridge Operations", func(t *testing.T) {
		// Test bridging NRN to XION chain
		bridgeData := map[string]interface{}{
			"from_chain":    "knirv-testnet",
			"to_chain":      "xion-testnet",
			"amount":        100.0,
			"recipient":     "xion1test...",
			"bridge_type":   "nrn_transfer",
		}

		response, err := suite.makeBlockchainCall("POST", "/bridge/transfer", bridgeData)
		require.NoError(t, err)

		var bridgeResponse struct {
			Success       bool   `json:"success"`
			TransactionID string `json:"transaction_id"`
			BridgeID      string `json:"bridge_id"`
			Status        string `json:"status"`
		}

		err = json.Unmarshal(response, &bridgeResponse)
		require.NoError(t, err)
		assert.True(t, bridgeResponse.Success)
		assert.NotEmpty(t, bridgeResponse.BridgeID)
		assert.Equal(t, "pending", bridgeResponse.Status)
	})

	t.Run("Multi-Chain Skill Execution", func(t *testing.T) {
		// Test executing skills across multiple chains
		multiChainData := map[string]interface{}{
			"skill_id":      "skill_001",
			"execution_chain": "knirvchain",
			"payment_chain": "xion",
			"amount":        50.0,
			"user":          "user_001",
		}

		response, err := suite.makeBlockchainCall("POST", "/skills/multichain_execute", multiChainData)
		require.NoError(t, err)

		var multiChainResponse struct {
			Success         bool   `json:"success"`
			ExecutionTxID   string `json:"execution_tx_id"`
			PaymentTxID     string `json:"payment_tx_id"`
			CrossChainProof string `json:"cross_chain_proof"`
		}

		err = json.Unmarshal(response, &multiChainResponse)
		require.NoError(t, err)
		assert.True(t, multiChainResponse.Success)
		assert.NotEmpty(t, multiChainResponse.ExecutionTxID)
		assert.NotEmpty(t, multiChainResponse.PaymentTxID)
	})

	t.Run("Token Exchange Rates", func(t *testing.T) {
		// Test token exchange rate mechanisms
		response, err := suite.makeBlockchainCall("GET", "/exchange/rates", nil)
		require.NoError(t, err)

		var ratesResponse struct {
			Success bool `json:"success"`
			Rates   map[string]struct {
				Price     float64 `json:"price"`
				Volume24h float64 `json:"volume_24h"`
				Change24h float64 `json:"change_24h"`
			} `json:"rates"`
		}

		err = json.Unmarshal(response, &ratesResponse)
		require.NoError(t, err)
		assert.True(t, ratesResponse.Success)
		assert.Contains(t, ratesResponse.Rates, "NRN/USDC")
		assert.Contains(t, ratesResponse.Rates, "NRN/XION")
	})
}

// TestEconomicIncentives tests economic incentive mechanisms
func TestEconomicIncentives(t *testing.T) {
	suite := NewEconomicLoopTestSuite()

	t.Run("Staking Incentives", func(t *testing.T) {
		// Test staking rewards calculation
		stakingData := map[string]interface{}{
			"staker":        "user_001",
			"amount":        1000.0,
			"duration":      "30d",
			"validator":     "validator_001",
		}

		response, err := suite.makeBlockchainCall("POST", "/staking/stake", stakingData)
		require.NoError(t, err)

		var stakingResponse struct {
			Success        bool    `json:"success"`
			StakeID        string  `json:"stake_id"`
			ExpectedReward float64 `json:"expected_reward"`
			APY            float64 `json:"apy"`
		}

		err = json.Unmarshal(response, &stakingResponse)
		require.NoError(t, err)
		assert.True(t, stakingResponse.Success)
		assert.Greater(t, stakingResponse.ExpectedReward, 0.0)
		assert.Greater(t, stakingResponse.APY, 0.0)
	})

	t.Run("Liquidity Provision Incentives", func(t *testing.T) {
		// Test liquidity provision rewards
		liquidityData := map[string]interface{}{
			"provider":    "user_001",
			"token_a":     "NRN",
			"token_b":     "USDC",
			"amount_a":    500.0,
			"amount_b":    500.0,
		}

		response, err := suite.makeBlockchainCall("POST", "/liquidity/provide", liquidityData)
		require.NoError(t, err)

		var liquidityResponse struct {
			Success      bool    `json:"success"`
			PoolID       string  `json:"pool_id"`
			LPTokens     float64 `json:"lp_tokens"`
			ExpectedFees float64 `json:"expected_fees"`
		}

		err = json.Unmarshal(response, &liquidityResponse)
		require.NoError(t, err)
		assert.True(t, liquidityResponse.Success)
		assert.Greater(t, liquidityResponse.LPTokens, 0.0)
	})

	t.Run("Governance Participation Rewards", func(t *testing.T) {
		// Test governance participation incentives
		governanceData := map[string]interface{}{
			"participant": "user_001",
			"proposal_id": "prop_001",
			"vote":        "yes",
			"stake_weight": 1000.0,
		}

		response, err := suite.makeBlockchainCall("POST", "/governance/vote", governanceData)
		require.NoError(t, err)

		var govResponse struct {
			Success           bool    `json:"success"`
			VoteID            string  `json:"vote_id"`
			ParticipationReward float64 `json:"participation_reward"`
		}

		err = json.Unmarshal(response, &govResponse)
		require.NoError(t, err)
		assert.True(t, govResponse.Success)
		assert.Greater(t, govResponse.ParticipationReward, 0.0)
	})
}

// Helper methods

// captureEconomicState captures the current economic state
func (suite *EconomicLoopTestSuite) captureEconomicState() (*EconomicState, error) {
	response, err := suite.makeBlockchainCall("GET", "/economic/state", nil)
	if err != nil {
		return nil, err
	}

	var state EconomicState
	err = json.Unmarshal(response, &state)
	if err != nil {
		return nil, err
	}

	state.Timestamp = time.Now()
	return &state, nil
}

// makeBlockchainCall makes a call to the blockchain API
func (suite *EconomicLoopTestSuite) makeBlockchainCall(method, endpoint string, data interface{}) ([]byte, error) {
	// Implementation would make actual HTTP request to blockchain
	// For now, return mock response based on endpoint
	
	mockResponses := map[string]interface{}{
		"/mint": map[string]interface{}{
			"success": true,
			"transaction_id": "tx_mint_001",
			"amount_minted": 100.0,
			"new_supply": 1000100.0,
		},
		"/distribute": map[string]interface{}{
			"success": true,
			"transaction_id": "tx_dist_001",
			"recipient": "developer_001",
			"amount": 50.0,
		},
		"/economic/state": EconomicState{
			TotalNRNSupply:    1000000.0,
			CirculatingSupply: 800000.0,
			TotalStaked:       150000.0,
			TotalBurned:       50000.0,
			NetworkFees:       5000.0,
			RewardPool:        25000.0,
			Timestamp:         time.Now(),
		},
	}

	if mockResp, exists := mockResponses[endpoint]; exists {
		return json.Marshal(mockResp)
	}

	// Default success response
	return json.Marshal(map[string]interface{}{"success": true})
}

// validateEconomicConsistency validates economic state consistency
func (suite *EconomicLoopTestSuite) validateEconomicConsistency(state *EconomicState) error {
	// Check that circulating + staked + burned <= total supply
	accountedSupply := state.CirculatingSupply + state.TotalStaked + state.TotalBurned
	if accountedSupply > state.TotalNRNSupply {
		return fmt.Errorf("accounted supply (%.2f) exceeds total supply (%.2f)", 
			accountedSupply, state.TotalNRNSupply)
	}

	// Check that values are non-negative
	if state.TotalNRNSupply < 0 || state.CirculatingSupply < 0 || 
	   state.TotalStaked < 0 || state.TotalBurned < 0 {
		return fmt.Errorf("negative values detected in economic state")
	}

	return nil
}

// calculateExpectedReward calculates expected reward based on parameters
func (suite *EconomicLoopTestSuite) calculateExpectedReward(amount, rate float64, duration time.Duration) float64 {
	// Simple compound interest calculation
	years := duration.Hours() / (24 * 365)
	return amount * math.Pow(1+rate, years) - amount
}
