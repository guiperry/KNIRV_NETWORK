// governance_machine.p - P State Machine for Governance System
// Models on-chain governance including proposals, voting, and execution.
// Based on: KNIRVGATEWAY/internal/oracle/governance/system.go

machine GovernanceMachine {
    // State variables
    var proposals: map[string, Proposal];
    var votes: map[string, map[Address, Vote]];  // proposalID -> voter -> vote
    var validators: map[Address, Validator];
    var params: NetworkParameters;
    var proposalCounter: int;
    var currentTime: int;  // Simulated time for testing

    // Pending "Second Me" registrations
    var pendingRegistrations: map[Address, map[string, any]];

    // Start state
    start state Init {
        entry {
            proposals = default(map[string, Proposal]);
            votes = default(map[string, map[Address, Vote]]);
            validators = default(map[Address, Validator]);
            pendingRegistrations = default(map[Address, map[string, any]]);
            proposalCounter = 0;
            currentTime = 0;

            // Initialize default network parameters
            params = (
                votingPeriod = 100,  // 100 time units
                proposalDeposit = (value = 100000000, isNegative = false),  // 100 NRN
                quorumThreshold = 33,   // 33%
                passThreshold = 50,     // 50%
                vetoThreshold = 33,     // 33%
                minValidatorStake = (value = 1000000000, isNegative = false),  // 1000 NRN
                maxValidators = 100
            );

            announce eComponentStarted, (componentName = "GovernanceMachine");
        }

        on eOracleStart do {
            goto Active;
        }
    }

    state Active {
        // Handle proposal creation
        on eCreateProposal do (payload: (
            proposalType: ProposalType,
            title: string,
            description: string,
            proposer: Address,
            deposit: BigInt,
            content: map[string, any]
        )) {
            // Validate proposal
            if (payload.title == "" || payload.description == "") {
                announce eProposalCreationFailed, (
                    error = (code = 201, message = "Title and description required")
                );
                return;
            }

            // Check deposit
            if (payload.deposit.value < params.proposalDeposit.value) {
                announce eProposalCreationFailed, (error = InsufficientDeposit());
                return;
            }

            // Generate proposal ID
            proposalCounter = proposalCounter + 1;
            var proposalID: string;
            proposalID = format("prop_{0}", proposalCounter);

            // Create proposal
            var proposal: Proposal;
            proposal = (
                id = proposalID,
                proposalType = payload.proposalType,
                title = payload.title,
                description = payload.description,
                proposer = payload.proposer,
                status = (statusName = "Pending"),
                submitTime = currentTime,
                votingStart = currentTime + 10,  // 10 time units delay
                votingEnd = currentTime + 10 + params.votingPeriod,
                deposit = payload.deposit
            );

            proposals[proposalID] = proposal;
            votes[proposalID] = default(map[Address, Vote]);

            announce eProposalCreated, (proposal = proposal);
        }

        // Handle vote casting
        on eCastVote do (payload: (
            proposalID: string,
            voter: Address,
            option: VoteOption,
            votingPower: BigInt
        )) {
            // Check proposal exists
            if (!(payload.proposalID in proposals)) {
                announce eVoteFailed, (
                    proposalID = payload.proposalID,
                    voter = payload.voter,
                    error = ProposalNotFound()
                );
                return;
            }

            var proposal: Proposal;
            proposal = proposals[payload.proposalID];

            // Check voting is active
            if (proposal.status.statusName != "Active" &&
                (proposal.status.statusName != "Pending" || currentTime < proposal.votingStart)) {
                announce eVoteFailed, (
                    proposalID = payload.proposalID,
                    voter = payload.voter,
                    error = VotingNotActive()
                );
                // Notify governance invariant monitor
                announce eGovernanceInvariantViolation, (
                    proposalID = payload.proposalID,
                    violation = "Vote attempted outside voting period"
                );
                return;
            }

            // Check if voting has ended
            if (currentTime > proposal.votingEnd) {
                announce eVoteFailed, (
                    proposalID = payload.proposalID,
                    voter = payload.voter,
                    error = (code = 202, message = "Voting period ended")
                );
                return;
            }

            // Check if already voted
            if (payload.proposalID in votes) {
                var proposalVotes: map[Address, Vote];
                proposalVotes = votes[payload.proposalID];
                if (payload.voter in proposalVotes) {
                    announce eVoteFailed, (
                        proposalID = payload.proposalID,
                        voter = payload.voter,
                        error = AlreadyVoted()
                    );
                    return;
                }
            }

            // Record vote
            var vote: Vote;
            vote = (
                proposalID = payload.proposalID,
                voter = payload.voter,
                option = payload.option,
                votingPower = payload.votingPower,
                timestamp = currentTime
            );

            var proposalVotesMap: map[Address, Vote];
            if (payload.proposalID in votes) {
                proposalVotesMap = votes[payload.proposalID];
            } else {
                proposalVotesMap = default(map[Address, Vote]);
            }
            proposalVotesMap[payload.voter] = vote;
            votes[payload.proposalID] = proposalVotesMap;

            // Update proposal status if needed
            if (proposal.status.statusName == "Pending" && currentTime >= proposal.votingStart) {
                proposal.status = (statusName = "Active");
                proposals[payload.proposalID] = proposal;
            }

            announce eVoteCast, (vote = vote);
        }

        // Handle proposal status update
        on eUpdateProposalStatus do (payload: (proposalID: string)) {
            if (!(payload.proposalID in proposals)) {
                return;
            }

            var proposal: Proposal;
            proposal = proposals[payload.proposalID];

            // Transition Pending -> Active
            if (proposal.status.statusName == "Pending" && currentTime >= proposal.votingStart) {
                proposal.status = (statusName = "Active");
                proposals[payload.proposalID] = proposal;
            }

            // Check if voting ended
            if (proposal.status.statusName == "Active" && currentTime > proposal.votingEnd) {
                // Tally votes
                var tallyResult: TallyResult;
                tallyResult = TallyVotes(payload.proposalID, votes);

                // Evaluate outcome
                var passed: bool;
                passed = EvaluateProposal(tallyResult, params);

                if (passed) {
                    proposal.status = (statusName = "Passed");
                    proposals[payload.proposalID] = proposal;
                    announce eProposalPassed, (proposalID = payload.proposalID, tallyResult = tallyResult);
                } else {
                    proposal.status = (statusName = "Rejected");
                    proposals[payload.proposalID] = proposal;
                    announce eProposalRejected, (proposalID = payload.proposalID, tallyResult = tallyResult);
                }
            }
        }

        // Handle proposal query
        on eGetProposal do (payload: (proposalID: string)) {
            if (!(payload.proposalID in proposals)) {
                announce eProposalCreationFailed, (error = ProposalNotFound());
                return;
            }

            announce eProposalResponse, (proposal = proposals[payload.proposalID]);
        }

        // Handle list proposals
        on eListProposals do {
            var proposalList: seq[Proposal];
            proposalList = default(seq[Proposal]);

            foreach (id in keys(proposals)) {
                proposalList += (sizeof(proposalList), proposals[id]);
            }

            announce eProposalsListResponse, (proposals = proposalList);
        }

        // Handle network params query
        on eGetNetworkParams do {
            announce eNetworkParamsResponse, (params = params);
        }

        // Handle network params update (through governance)
        on eUpdateNetworkParams do (payload: (params: NetworkParameters)) {
            // Validate parameters
            if (payload.params.quorumThreshold < 0 || payload.params.quorumThreshold > 100) {
                return;
            }
            if (payload.params.passThreshold < 0 || payload.params.passThreshold > 100) {
                return;
            }

            params = payload.params;
            announce eNetworkParamsUpdated, (params = params);
        }

        // Handle proposal cancellation
        on eCancelProposal do (payload: (proposalID: string, requester: Address)) {
            if (!(payload.proposalID in proposals)) {
                return;
            }

            var proposal: Proposal;
            proposal = proposals[payload.proposalID];

            // Can only cancel pending proposals
            if (proposal.status.statusName != "Pending") {
                return;
            }

            // Only proposer can cancel
            if (proposal.proposer != payload.requester) {
                return;
            }

            proposal.status = (statusName = "Cancelled");
            proposals[payload.proposalID] = proposal;

            announce eProposalCancelled, (proposalID = payload.proposalID);
        }

        // ===================================================================
        // "Second Me" Registration Workflow (Section 5.2 of p_implementation.md)
        // ===================================================================

        // Handle registration request
        on eRegisterRequest do (payload: (user: Address, initialData: map[string, any])) {
            // Store pending registration
            pendingRegistrations[payload.user] = payload.initialData;

            // Request fee verification from Economics machine
            announce eVerifyFeePayment, (
                user = payload.user,
                expectedAmount = params.proposalDeposit  // Use proposal deposit as registration fee
            );
        }

        // Handle fee payment verification response
        on eFeePaymentVerified do (payload: (user: Address, amount: BigInt, verified: bool)) {
            if (!payload.verified) {
                // Fee not paid or insufficient
                announce eRegistrationFail, (
                    user = payload.user,
                    reason = "Insufficient fee payment"
                );
                // Clean up pending registration
                if (payload.user in pendingRegistrations) {
                    pendingRegistrations -= payload.user;
                }
                return;
            }

            // Fee verified, complete registration
            if (payload.user in pendingRegistrations) {
                var secondMeID: string;
                proposalCounter = proposalCounter + 1;
                secondMeID = format("second_me_{0}", proposalCounter);

                announce eRegistrationConfirm, (
                    user = payload.user,
                    secondMeID = secondMeID
                );

                // Clean up pending registration
                pendingRegistrations -= payload.user;
            }
        }

        // Handle shutdown
        on eOracleStop do {
            goto Stopped;
        }
    }

    state Stopped {
        entry {
            announce eComponentStopped, (componentName = "GovernanceMachine");
        }

        on eOracleStart do {
            goto Active;
        }

        // Still respond to queries
        on eGetProposal do (payload: (proposalID: string)) {
            if (payload.proposalID in proposals) {
                announce eProposalResponse, (proposal = proposals[payload.proposalID]);
            }
        }

        on eGetNetworkParams do {
            announce eNetworkParamsResponse, (params = params);
        }

        ignore eCreateProposal, eCastVote, eUpdateProposalStatus, eCancelProposal;
        ignore eRegisterRequest, eFeePaymentVerified;
    }
}

// =====================================================================
// Helper Functions
// =====================================================================

// TallyVotes counts all votes for a proposal
fun TallyVotes(proposalID: string, allVotes: map[string, map[Address, Vote]]): TallyResult {
    var result: TallyResult;
    result = (
        yesVotes = (value = 0, isNegative = false),
        noVotes = (value = 0, isNegative = false),
        abstainVotes = (value = 0, isNegative = false),
        vetoVotes = (value = 0, isNegative = false),
        totalVoted = (value = 0, isNegative = false)
    );

    if (!(proposalID in allVotes)) {
        return result;
    }

    var proposalVotes: map[Address, Vote];
    proposalVotes = allVotes[proposalID];

    foreach (voter in keys(proposalVotes)) {
        var vote: Vote;
        vote = proposalVotes[voter];

        // option: 0=Abstain, 1=Yes, 2=No, 3=NoWithVeto
        if (vote.option.option == 0) {
            result.abstainVotes = AddBigInt(result.abstainVotes, vote.votingPower);
        } else if (vote.option.option == 1) {
            result.yesVotes = AddBigInt(result.yesVotes, vote.votingPower);
        } else if (vote.option.option == 2) {
            result.noVotes = AddBigInt(result.noVotes, vote.votingPower);
        } else if (vote.option.option == 3) {
            result.vetoVotes = AddBigInt(result.vetoVotes, vote.votingPower);
        }

        result.totalVoted = AddBigInt(result.totalVoted, vote.votingPower);
    }

    return result;
}

// EvaluateProposal determines if a proposal passed based on tally and params
fun EvaluateProposal(tally: TallyResult, params: NetworkParameters): bool {
    // Check quorum (simplified: using total voted as proxy for participation)
    // In real implementation, would compare to total stake

    // Check veto threshold
    if (tally.totalVoted.value > 0) {
        var vetoPercent: int;
        vetoPercent = (tally.vetoVotes.value * 100) / tally.totalVoted.value;
        if (vetoPercent >= params.vetoThreshold) {
            return false;
        }
    }

    // Check pass threshold (Yes votes as percentage of non-abstain votes)
    var nonAbstainVotes: int;
    nonAbstainVotes = tally.yesVotes.value + tally.noVotes.value + tally.vetoVotes.value;

    if (nonAbstainVotes > 0) {
        var passPercent: int;
        passPercent = (tally.yesVotes.value * 100) / nonAbstainVotes;
        if (passPercent >= params.passThreshold) {
            return true;
        }
    }

    return false;
}
