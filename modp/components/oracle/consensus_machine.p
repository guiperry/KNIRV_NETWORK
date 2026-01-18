// consensus_machine.p - Consensus Engine State Machine for KNIRVORACLE
// Models ABCI-style consensus including block proposal and validation
// Based on: KNIRVORACLE consensus module implementation

machine ConsensusMachine {
    // State variables
    var currentHeight: int;
    var currentRound: int;
    var validators: set[Address];
    var proposer: Address;
    var pendingBlock: BlockHeader;
    var committedBlocks: seq[BlockHeader];

    // Configuration
    var blockTime: int;  // milliseconds
    var maxValidators: int;
    var minValidatorStake: BigInt;

    // Round state
    var prepareVotes: map[Address, bool];
    var commitVotes: map[Address, bool];

    // Temp variables
    var tempValidatorList: seq[Address];
    var tempProposerIndex: int;
    var tempPrepareCount: int;
    var tempCommitCount: int;
    var tempThreshold: int;
    var tempHash: Hash;
    var tempTimestamp: Timestamp;
    var tempAddress: Address;
    var tempValidators: set[Address];
    var validator: Address;
    var v: Address;
    var temp_payload: (string, string, bool);

    start state Init {
        entry {
            currentHeight = 0;
            currentRound = 0;
            validators = default(set[Address]);
            proposer.bytes = default(seq[int]);
            pendingBlock = default(BlockHeader);
            committedBlocks = default(seq[BlockHeader]);

            // Configuration
            blockTime = 6000;  // 6 seconds
            maxValidators = 100;
            minValidatorStake.value = 1000000000;  // 10,000 NRN (model value)
            minValidatorStake.isNegative = false;

            prepareVotes = default(map[Address, bool]);
            commitVotes = default(map[Address, bool]);
        }

        on eComponentStart do {
            goto WaitingForValidators;
        }

        on eNetworkStart do {
            goto WaitingForValidators;
        }
    }

    state WaitingForValidators {
        entry {
            // Wait for minimum validators to join
        }

        on eStaked do (stakeInfo: StakeInfo) {
            // Add validator if stake meets minimum
            if (stakeInfo.amount.value >= minValidatorStake.value) {
                if (sizeof(validators) < maxValidators) {
                    // Start consensus when we have at least 1 validator
                    if (sizeof(validators) >= 1) {
                        goto Proposing;
                    }
                }
            }
        }

        on eNetworkShutdown do {
            goto Halted;
        }
    }

    state Proposing {
        entry {
            tempValidatorList = default(seq[Address]);
            foreach (v in validators) {
                tempValidatorList += (sizeof(tempValidatorList), v);
            }

            if (sizeof(tempValidatorList) > 0) {
                tempProposerIndex = currentRound % sizeof(tempValidatorList);
                proposer = tempValidatorList[tempProposerIndex];

                // Create pending block
                tempHash.bytes = default(seq[int]);
                tempTimestamp.milliseconds = 0;

                pendingBlock.height = currentHeight + 1;
                pendingBlock.chainID = "knirv-oracle";
                pendingBlock.previousHash = tempHash;
                pendingBlock.stateRoot = tempHash;
                pendingBlock.timestamp = tempTimestamp;
                pendingBlock.proposer = proposer;

                // Reset votes
                prepareVotes = default(map[Address, bool]);
                commitVotes = default(map[Address, bool]);

                // Broadcast proposal (in real implementation)
                goto Preparing;
            }
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Preparing {
        entry {
            // Validators verify and vote to prepare
            // Proposer automatically prepares their own block
            prepareVotes[proposer] = true;
        }

        // Receive prepare votes
        on eValidationComplete do (payload: (result: ValidationResult)) {
            if (payload.result.passed) {
                // Count as prepare vote
                prepareVotes[payload.result.validator.id] = true;

                // Check if we have 2/3+ prepares
                tempPrepareCount = sizeof(prepareVotes);
                tempThreshold = (sizeof(validators) * 2) / 3;

                if (tempPrepareCount > tempThreshold) {
                    goto Committing;
                }
            }
        }

        // Timeout handling
        on eValidationTimeout do (payload: (taskID: UUID)) {
            currentRound = currentRound + 1;
            goto Proposing;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Committing {
        entry {
            // Proposer automatically commits
            commitVotes[proposer] = true;
        }

        // Receive commit votes
        on eValidationComplete do (payload: (result: ValidationResult)) {
            if (payload.result.passed) {
                // Count as commit vote
                commitVotes[payload.result.validator.id] = true;

                // Check if we have 2/3+ commits
                tempCommitCount = sizeof(commitVotes);
                tempThreshold = (sizeof(validators) * 2) / 3;

                if (tempCommitCount > tempThreshold) {
                    // Block is committed
                    committedBlocks += (sizeof(committedBlocks), pendingBlock);
                    currentHeight = currentHeight + 1;
                    currentRound = 0;

                    // Announce block committed
                    temp_payload = ("ConsensusMonitor", "Block committed", true);
                    announce eMonitorAssertion, temp_payload;

                    goto Proposing;
                }
            }
        }

        // Timeout handling
        on eValidationTimeout do (payload: (taskID: UUID)) {
            currentRound = currentRound + 1;
            goto Proposing;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Proposing;
        }
    }

    // Helper functions
    fun GetCurrentHeight(): int {
        return currentHeight;
    }

    fun GetValidatorCount(): int {
        return sizeof(validators);
    }

    fun IsValidator(addr: Address): bool {
        return addr in validators;
    }
}
