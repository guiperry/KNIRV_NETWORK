// consensus_machine.p - P State Machine for Consensus Engine
// Models the consensus mechanism of the KNIRVORACLE blockchain.
// Based on: KNIRVGATEWAY/internal/oracle/consensus/engine.go

machine ConsensusMachine {
    // State variables
    var chainID: string;
    var currentHeight: int;
    var currentRound: int;
    var currentStep: RoundStep;
    var validators: seq[ConsensusValidator];
    var lastValidators: seq[ConsensusValidator];
    var proposerIndex: int;
    var isRunning: bool;
    var blockTime: int;  // in seconds

    // Start state
    start state Init {
        entry {
            chainID = "";
            currentHeight = 0;
            currentRound = 0;
            currentStep = (stepName = "NewHeight");
            validators = default(seq[ConsensusValidator]);
            lastValidators = default(seq[ConsensusValidator]);
            proposerIndex = 0;
            isRunning = false;
            blockTime = 5;  // default 5 seconds
        }

        on eInitChain do (payload: (chainID: string, validators: seq[ConsensusValidator])) {
            chainID = payload.chainID;
            validators = payload.validators;
            lastValidators = payload.validators;
            currentHeight = 1;
            currentRound = 0;
            proposerIndex = 0;

            announce eValidatorsUpdated, (validators = validators);
            goto Running;
        }
    }

    state Running {
        entry {
            isRunning = true;
            announce eComponentStarted, (componentName = "ConsensusMachine");
        }

        // Handle block proposal request
        on eProposeBlock do (payload: (txs: seq[seq[int]])) {
            if (sizeof(validators) == 0) {
                announce eBlockProposalFailed, (error = (code = 100, message = "No validators available"));
                return;
            }

            // Get current proposer
            var proposer: ConsensusValidator;
            proposer = validators[proposerIndex];

            // Create block
            var block: Block;
            block = (
                height = currentHeight,
                chainID = chainID,
                time = 0,  // would be current timestamp
                proposer = proposer.address,
                txCount = sizeof(payload.txs),
                appHash = default(seq[int])
            );

            // Update round state
            currentStep = (stepName = "Propose");

            announce eBlockProposed, (block = block);
        }

        // Handle block commit
        on eCommitBlock do (payload: (block: Block)) {
            // Validate block height
            if (payload.block.height != currentHeight) {
                announce eBlockCommitFailed, (
                    block = payload.block,
                    error = (code = 101, message = "Invalid block height")
                );
                return;
            }

            // Begin block processing
            currentStep = (stepName = "Commit");

            // Commit the block (simplified)
            var appHash: seq[int];
            appHash = default(seq[int]);  // Would be actual state hash

            // Advance to next height
            currentHeight = currentHeight + 1;
            currentRound = 0;
            currentStep = (stepName = "NewHeight");

            // Rotate proposer
            proposerIndex = (proposerIndex + 1) % sizeof(validators);

            announce eBlockCommitted, (block = payload.block, appHash = appHash);
        }

        // Handle validator set update
        on eUpdateValidators do (payload: (validators: seq[ConsensusValidator])) {
            lastValidators = validators;
            validators = payload.validators;

            // Reset proposer index if needed
            if (proposerIndex >= sizeof(validators)) {
                proposerIndex = 0;
            }

            announce eValidatorsUpdated, (validators = validators);
        }

        // Handle state query
        on eGetConsensusState do {
            var state: RoundState;
            state = (
                height = currentHeight,
                round = currentRound,
                step = currentStep,
                validatorCount = sizeof(validators)
            );

            announce eConsensusStateResponse, (state = state);
        }

        // Handle stop request
        on eOracleStop do {
            goto Stopped;
        }
    }

    state Stopped {
        entry {
            isRunning = false;
            announce eComponentStopped, (componentName = "ConsensusMachine");
        }

        // Can restart
        on eOracleStart do {
            goto Running;
        }

        ignore eProposeBlock, eCommitBlock, eUpdateValidators, eGetConsensusState;
    }
}

// =====================================================================
// Validator Set Management Functions
// =====================================================================

// GetTotalVotingPower calculates total voting power of all validators
fun GetTotalVotingPower(validators: seq[ConsensusValidator]): int {
    var total: int;
    var i: int;

    total = 0;
    i = 0;
    while (i < sizeof(validators)) {
        total = total + validators[i].votingPower;
        i = i + 1;
    }

    return total;
}

// GetProposer returns the current proposer based on priority
fun GetProposer(validators: seq[ConsensusValidator], index: int): ConsensusValidator {
    if (sizeof(validators) == 0) {
        return default(ConsensusValidator);
    }
    return validators[index % sizeof(validators)];
}

// IncrementProposerPriority updates proposer priorities for the next round
fun IncrementProposerPriority(validators: seq[ConsensusValidator], times: int): seq[ConsensusValidator] {
    // Simplified: in real implementation, this would update priority based on voting power
    return validators;
}
