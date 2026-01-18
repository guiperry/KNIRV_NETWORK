// governance_machine.p - Governance State Machine for KNIRVORACLE
// Models on-chain governance including proposals, voting, and Second Me registration
// Based on: KNIRVORACLE governance module implementation

machine GovernanceMachine {
    // State variables
    var proposals: map[string, Proposal];
    var votes: map[string, map[Address, Vote]];
    var proposalCounter: int;
    var pendingRegistrations: map[Address, map[string, any]];

    // Configuration
    var minDeposit: BigInt;
    var votingPeriod: int;  // in blocks
    var quorumThreshold: int;  // percentage * 100
    var passThreshold: int;  // percentage * 100
    var vetoThreshold: int;  // percentage * 100

    // Token machine reference for voting power
    var tokenMachine: machine;

    // Temp variables for event handlers
    var tempProposalID: string;
    var tempProposal: Proposal;
    var tempVote: Vote;
    var tempProposalVotes: map[Address, Vote];
    var tempRegistrationFee: BigInt;
    var tempSecondMeID: string;
    var tempStatus: ProposalStatus;
    var tempBigInt: BigInt;
    var tempTimestamp: Timestamp;
    var tempVoteFailed: VoteFailedData;
    var tempVoteCast: VoteCastData;
    var tempExecFailed: ProposalExecutionFailedData;
    var tempFeeVerify: FeeVerificationData;
    var tempRegFail: RegistrationFailData;
    var tempRegConfirm: RegistrationConfirmData;

    start state Init {
        entry {
            proposals = default(map[string, Proposal]);
            votes = default(map[string, map[Address, Vote]]);
            proposalCounter = 0;
            pendingRegistrations = default(map[Address, map[string, any]]);

            // Configuration defaults
            minDeposit.value = 100000000;  // 100 NRN
            minDeposit.isNegative = false;
            votingPeriod = 100000;  // blocks
            quorumThreshold = 3333;  // 33.33%
            passThreshold = 5000;  // 50%
            vetoThreshold = 3333;  // 33.33%
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // Proposal creation
        on eCreateProposal do (payload: (
            proposalType: ProposalType,
            title: string,
            description: string,
            proposer: Address,
            deposit: BigInt,
            content: map[string, any]
        )) {
            // Validate deposit
            if (payload.deposit.value < minDeposit.value) {
                send this, eProposalCreationFailed, "Insufficient deposit";
                return;
            }

            // Create proposal
            proposalCounter = proposalCounter + 1;
            tempProposalID = format("prop_{0}", proposalCounter);

            // Initialize temp values
            tempStatus.status = 0;  // Pending
            tempBigInt.value = 0;
            tempBigInt.isNegative = false;

            tempProposal.id = tempProposalID;
            tempProposal.proposalType = payload.proposalType;
            tempProposal.title = payload.title;
            tempProposal.description = payload.description;
            tempProposal.proposer = payload.proposer;
            tempProposal.deposit = payload.deposit;
            tempProposal.status = tempStatus;
            tempTimestamp.milliseconds = 0;
            tempProposal.votingStart = tempTimestamp;
            tempTimestamp.milliseconds = votingPeriod;
            tempProposal.votingEnd = tempTimestamp;
            tempProposal.yesVotes = tempBigInt;
            tempProposal.noVotes = tempBigInt;
            tempProposal.abstainVotes = tempBigInt;
            tempProposal.vetoVotes = tempBigInt;
            tempProposal.contentPayload = payload.content;

            proposals[tempProposalID] = tempProposal;
            votes[tempProposalID] = default(map[Address, Vote]);

            // Move to active status
            tempStatus.status = 1;
            tempProposal.status = tempStatus;
            proposals[tempProposalID] = tempProposal;

            announce eProposalCreated, tempProposal;
            send this, eProposalCreated, tempProposal;
        }

        // Vote casting
        on eCastVote do (payload: (proposalID: string, voter: Address, option: VoteOption, votingPower: BigInt)) {
            // Check proposal exists
            if (!(payload.proposalID in proposals)) {
                tempVoteFailed.proposalID = payload.proposalID;
                tempVoteFailed.reason = "Proposal not found";
                send this, eVoteFailed, tempVoteFailed;
                return;
            }

            tempProposal = proposals[payload.proposalID];

            // Check proposal is in voting period
            if (tempProposal.status.status != 1) {
                tempVoteFailed.proposalID = payload.proposalID;
                tempVoteFailed.reason = "Proposal not in voting period";
                send this, eVoteFailed, tempVoteFailed;
                return;
            }

            // Check for double voting
            tempProposalVotes = votes[payload.proposalID];

            if (payload.voter in tempProposalVotes) {
                tempVoteFailed.proposalID = payload.proposalID;
                tempVoteFailed.reason = "Already voted";
                send this, eVoteFailed, tempVoteFailed;
                return;
            }

            // Record vote
            tempTimestamp.milliseconds = 0;
            tempVote.proposalID = payload.proposalID;
            tempVote.voter = payload.voter;
            tempVote.option = payload.option;
            tempVote.votingPower = payload.votingPower;
            tempVote.timestamp = tempTimestamp;

            tempProposalVotes[payload.voter] = tempVote;
            votes[payload.proposalID] = tempProposalVotes;

            // Update vote tallies
            if (payload.option.option == 0) {
                tempProposal.abstainVotes.value = tempProposal.abstainVotes.value + payload.votingPower.value;
            } else if (payload.option.option == 1) {
                tempProposal.yesVotes.value = tempProposal.yesVotes.value + payload.votingPower.value;
            } else if (payload.option.option == 2) {
                tempProposal.noVotes.value = tempProposal.noVotes.value + payload.votingPower.value;
            } else if (payload.option.option == 3) {
                tempProposal.vetoVotes.value = tempProposal.vetoVotes.value + payload.votingPower.value;
            }

            proposals[payload.proposalID] = tempProposal;

            tempVoteCast.proposalID = payload.proposalID;
            tempVoteCast.voter = payload.voter;
            announce eVoteCast, tempVoteCast;
            send this, eVoteCast, tempVoteCast;
        }

        // Proposal execution
        on eExecuteProposal do (payload: (proposalID: string)) {
            if (!(payload.proposalID in proposals)) {
                tempExecFailed.proposalID = payload.proposalID;
                tempExecFailed.reason = "Proposal not found";
                send this, eProposalExecutionFailed, tempExecFailed;
                return;
            }

            tempProposal = proposals[payload.proposalID];

            // Only passed proposals can be executed
            if (tempProposal.status.status != 2) {
                tempExecFailed.proposalID = payload.proposalID;
                tempExecFailed.reason = "Proposal not passed";
                send this, eProposalExecutionFailed, tempExecFailed;
                return;
            }

            // Execute based on type
            tempStatus.status = 4;  // Executed
            tempProposal.status = tempStatus;
            proposals[payload.proposalID] = tempProposal;

            announce eProposalExecuted, payload.proposalID;
            send this, eProposalExecuted, payload.proposalID;
        }

        // Second Me Registration - Step 1: Receive registration request
        on eRegisterRequest do (payload: (user: Address, metadata: map[string, any])) {
            // Store pending registration
            pendingRegistrations[payload.user] = payload.metadata;

            // Request fee verification
            tempRegistrationFee.value = 50000000;  // 50 NRN
            tempRegistrationFee.isNegative = false;

            tempFeeVerify.user = payload.user;
            tempFeeVerify.amount = tempRegistrationFee;
            send this, eVerifyFeePayment, tempFeeVerify;
        }

        // Second Me Registration - Step 4: Fee payment verified
        on eFeePaymentVerified do (payload: (user: Address, amount: BigInt, verified: bool)) {
            if (!payload.verified) {
                pendingRegistrations -= payload.user;
                tempRegFail.user = payload.user;
                tempRegFail.reason = "Fee payment not verified";
                send this, eRegistrationFail, tempRegFail;
                return;
            }

            // Complete registration
            tempSecondMeID = format("sm_{0}", payload.user);

            pendingRegistrations -= payload.user;

            tempRegConfirm.user = payload.user;
            tempRegConfirm.registrationID = tempSecondMeID;
            send this, eRegistrationConfirm, tempRegConfirm;
        }

        // Proposal status change (can be called by time-based trigger)
        on eProposalStatusChanged do (payload: (proposalID: string, oldStatus: ProposalStatus, newStatus: ProposalStatus)) {
            if (payload.proposalID in proposals) {
                tempProposal = proposals[payload.proposalID];
                tempProposal.status = payload.newStatus;
                proposals[payload.proposalID] = tempProposal;
            }
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
            goto Active;
        }
    }

    // Helper to check if proposal exists
    fun HasProposal(proposalID: string): bool {
        return proposalID in proposals;
    }

    // Helper to get proposal status
    fun GetProposalStatus(proposalID: string): ProposalStatus {
        if (proposalID in proposals) {
            return proposals[proposalID].status;
        }
        tempStatus.status = -1;
        return tempStatus;
    }
}
