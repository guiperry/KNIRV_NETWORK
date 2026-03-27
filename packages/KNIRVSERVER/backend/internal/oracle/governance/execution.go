package governance

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"backend_server/internal/oracle/types"
	"go.uber.org/zap"
)

var (
	ErrProposalNotPassed       = fmt.Errorf("proposal has not passed")
	ErrProposalAlreadyExecuted = fmt.Errorf("proposal already executed")
	ErrExecutionFailed         = fmt.Errorf("proposal execution failed")
)

// ExecutionCallback is called when a proposal is executed
type ExecutionCallback func(proposal *Proposal) error

// ExecutionHandler manages proposal execution callbacks
type ExecutionHandler struct {
	callbacks map[ProposalType]ExecutionCallback
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler() *ExecutionHandler {
	return &ExecutionHandler{
		callbacks: make(map[ProposalType]ExecutionCallback),
	}
}

// RegisterCallback registers an execution callback for a proposal type
func (eh *ExecutionHandler) RegisterCallback(proposalType ProposalType, callback ExecutionCallback) {
	eh.callbacks[proposalType] = callback
}

// ExecuteProposal executes a passed proposal
func (gs *GovernanceSystem) ExecuteProposal(proposalID string, handler *ExecutionHandler) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	proposal, exists := gs.proposals[proposalID]
	if !exists {
		return ErrProposalNotFound
	}

	// Check proposal status
	if proposal.Status != ProposalStatusPassed {
		return ErrProposalNotPassed
	}

	// Check if already executed
	if proposal.Status == ProposalStatusExecuted {
		return ErrProposalAlreadyExecuted
	}

	// Execute based on proposal type
	var err error
	switch proposal.Type {
	case ProposalTypeNetworkParameter:
		err = gs.executeNetworkParameterProposal(proposal)
	case ProposalTypeValidatorSet:
		err = gs.executeValidatorSetProposal(proposal)
	case ProposalTypeTokenEconomics:
		err = gs.executeTokenEconomicsProposal(proposal)
	case ProposalTypeCrossChainConfig:
		err = gs.executeCrossChainConfigProposal(proposal)
	case ProposalTypeEmergencyAction:
		err = gs.executeEmergencyActionProposal(proposal)
	case ProposalTypeTextProposal:
		// Text proposals don't execute any action
		err = nil
	default:
		// Use custom callback if registered
		if callback, ok := handler.callbacks[proposal.Type]; ok {
			err = callback(proposal)
		} else {
			err = fmt.Errorf("unknown proposal type: %v", proposal.Type)
		}
	}

	now := time.Now()
	if err != nil {
		proposal.Status = ProposalStatusFailed
		proposal.ExecutionTime = &now
		gs.logger.Error("Proposal execution failed",
			zap.String("proposal_id", proposalID),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	proposal.Status = ProposalStatusExecuted
	proposal.ExecutionTime = &now

	gs.logger.Info("Proposal executed",
		zap.String("proposal_id", proposalID),
		zap.String("type", proposal.Type.String()),
	)

	return nil
}

// executeNetworkParameterProposal executes a network parameter update
func (gs *GovernanceSystem) executeNetworkParameterProposal(proposal *Proposal) error {
	// Extract parameters from content
	paramsData, ok := proposal.Content["parameters"]
	if !ok {
		return fmt.Errorf("parameters not found in proposal content")
	}

	// Marshal to JSON and unmarshal to NetworkParameters
	paramsJSON, err := json.Marshal(paramsData)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var newParams NetworkParameters
	if err := json.Unmarshal(paramsJSON, &newParams); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Validate and apply
	if err := gs.validateNetworkParameters(&newParams); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	gs.params = &newParams

	gs.logger.Info("Network parameters updated via governance",
		zap.String("proposal_id", proposal.ID),
	)

	return nil
}

// executeValidatorSetProposal executes a validator set change
func (gs *GovernanceSystem) executeValidatorSetProposal(proposal *Proposal) error {
	action, ok := proposal.Content["action"].(string)
	if !ok {
		return fmt.Errorf("action not found in proposal content")
	}

	validatorData, ok := proposal.Content["validator"]
	if !ok {
		return fmt.Errorf("validator not found in proposal content")
	}

	switch action {
	case "add":
		// Add new validator
		validatorJSON, err := json.Marshal(validatorData)
		if err != nil {
			return fmt.Errorf("failed to marshal validator: %w", err)
		}

		var reg ValidatorRegistration
		if err := json.Unmarshal(validatorJSON, &reg); err != nil {
			return fmt.Errorf("failed to unmarshal validator: %w", err)
		}

		_, err = gs.RegisterValidator(&reg)
		return err

	case "remove":
		// Remove validator
		addressStr, ok := validatorData.(string)
		if !ok {
			return fmt.Errorf("invalid validator address")
		}

		address, err := types.AddressFromString(addressStr)
		if err != nil {
			return fmt.Errorf("invalid address format: %w", err)
		}

		return gs.RemoveValidator(address)

	case "jail":
		// Jail validator
		addressStr, ok := validatorData.(string)
		if !ok {
			return fmt.Errorf("invalid validator address")
		}

		address, err := types.AddressFromString(addressStr)
		if err != nil {
			return fmt.Errorf("invalid address format: %w", err)
		}

		reason, _ := proposal.Content["reason"].(string)
		return gs.JailValidator(address, reason)

	case "unjail":
		// Unjail validator
		addressStr, ok := validatorData.(string)
		if !ok {
			return fmt.Errorf("invalid validator address")
		}

		address, err := types.AddressFromString(addressStr)
		if err != nil {
			return fmt.Errorf("invalid address format: %w", err)
		}

		return gs.UnjailValidator(address)

	default:
		return fmt.Errorf("unknown validator set action: %s", action)
	}
}

// executeTokenEconomicsProposal executes a token economics change
func (gs *GovernanceSystem) executeTokenEconomicsProposal(proposal *Proposal) error {
	// Extract economic parameters
	action, ok := proposal.Content["action"].(string)
	if !ok {
		return fmt.Errorf("action not found in proposal content")
	}

	switch action {
	case "update_fee_rate":
		feeRate, ok := proposal.Content["fee_rate"].(float64)
		if !ok {
			return fmt.Errorf("fee_rate not found or invalid")
		}
		if feeRate < 0 || feeRate > 1 {
			return fmt.Errorf("fee_rate must be between 0 and 1")
		}
		gs.params.BaseFeeRate = feeRate

	case "update_burn_rate":
		burnRate, ok := proposal.Content["burn_rate"].(float64)
		if !ok {
			return fmt.Errorf("burn_rate not found or invalid")
		}
		if burnRate < 0 || burnRate > 1 {
			return fmt.Errorf("burn_rate must be between 0 and 1")
		}
		gs.params.BurnRate = burnRate

	case "update_reward_rate":
		rewardRate, ok := proposal.Content["reward_rate"].(float64)
		if !ok {
			return fmt.Errorf("reward_rate not found or invalid")
		}
		if rewardRate < 0 || rewardRate > 1 {
			return fmt.Errorf("reward_rate must be between 0 and 1")
		}
		gs.params.RewardRate = rewardRate

	default:
		return fmt.Errorf("unknown token economics action: %s", action)
	}

	gs.logger.Info("Token economics updated via governance",
		zap.String("proposal_id", proposal.ID),
		zap.String("action", action),
	)

	return nil
}

// executeCrossChainConfigProposal executes a cross-chain configuration change
func (gs *GovernanceSystem) executeCrossChainConfigProposal(proposal *Proposal) error {
	action, ok := proposal.Content["action"].(string)
	if !ok {
		return fmt.Errorf("action not found in proposal content")
	}

	switch action {
	case "update_timeout":
		timeoutBlocks, ok := proposal.Content["timeout_blocks"].(float64)
		if !ok {
			return fmt.Errorf("timeout_blocks not found or invalid")
		}
		if timeoutBlocks <= 0 {
			return fmt.Errorf("timeout_blocks must be positive")
		}
		gs.params.IBCTimeoutBlocks = uint64(timeoutBlocks)

	case "update_transfer_limits":
		minTransferStr, ok := proposal.Content["min_transfer"].(string)
		if !ok {
			return fmt.Errorf("min_transfer not found or invalid")
		}
		maxTransferStr, ok := proposal.Content["max_transfer"].(string)
		if !ok {
			return fmt.Errorf("max_transfer not found or invalid")
		}

		minTransfer, ok := new(big.Int).SetString(minTransferStr, 10)
		if !ok {
			return fmt.Errorf("invalid min_transfer format")
		}
		maxTransfer, ok := new(big.Int).SetString(maxTransferStr, 10)
		if !ok {
			return fmt.Errorf("invalid max_transfer format")
		}

		if minTransfer.Cmp(maxTransfer) > 0 {
			return fmt.Errorf("min_transfer cannot exceed max_transfer")
		}

		gs.params.MinTransferAmount = minTransfer
		gs.params.MaxTransferAmount = maxTransfer

	default:
		return fmt.Errorf("unknown cross-chain config action: %s", action)
	}

	gs.logger.Info("Cross-chain config updated via governance",
		zap.String("proposal_id", proposal.ID),
		zap.String("action", action),
	)

	return nil
}

// executeEmergencyActionProposal executes an emergency action
func (gs *GovernanceSystem) executeEmergencyActionProposal(proposal *Proposal) error {
	action, ok := proposal.Content["action"].(string)
	if !ok {
		return fmt.Errorf("action not found in proposal content")
	}

	gs.logger.Warn("Emergency action executed",
		zap.String("proposal_id", proposal.ID),
		zap.String("action", action),
	)

	// Emergency actions would be handled by external systems
	// This is just logging for now
	return nil
}
