// network_invariants.p - Network-Wide Invariant Monitors
// Monitors cross-component invariants across the entire KNIRV Network
// Based on: KNIRV Network Architecture

// =====================================================================
// NETWORK INTEGRITY MONITOR
// Ensures overall network health and consistency
// =====================================================================

spec NetworkIntegrityMonitor observes
    eNetworkStart, eNetworkShutdown, eNetworkReady,
    eComponentStart, eComponentReady, eComponentShutdown,
    eEmergencyHalt, eEmergencyResume,
    eNetworkPartitionDetected, eNetworkHealed,
    eMonitorViolation, eNetworkHealthCheck, eNetworkHealthReport
{
    var componentsStarted: set[string];
    var componentsReady: set[string];
    var networkRunning: bool;
    var networkPartitioned: bool;
    var violationCount: int;

    start state Initializing {
        entry {
            componentsStarted = default(set[string]);
            componentsReady = default(set[string]);
            networkRunning = false;
            networkPartitioned = false;
            violationCount = 0;
        }

        on eNetworkStart do {
            goto Running;
        }
    }

    state Running {
        entry {
            networkRunning = true;
        }

        on eComponentStart do (payload: (componentName: string)) {
            componentsStarted = componentsStarted + (payload.componentName);
        }

        on eComponentReady do (payload: (componentName: string)) {
            componentsReady = componentsReady + (payload.componentName);

            // Assert component was started before becoming ready
            assert payload.componentName in componentsStarted,
                format("Component {0} became ready without being started", payload.componentName);
        }

        on eComponentShutdown do (payload: (componentName: string)) {
            componentsReady = componentsReady - (payload.componentName);
            componentsStarted = componentsStarted - (payload.componentName);
        }

        on eNetworkPartitionDetected do {
            networkPartitioned = true;

            // Network partition is a serious issue
            announce eMonitorViolation, ("NetworkIntegrity", "Network partition detected", default(map[string, any]));
        }

        on eNetworkHealed do {
            networkPartitioned = false;

            announce eMonitorAssertion, ("NetworkIntegrity", "Network healed from partition", true);
        }

        on eMonitorViolation do {
            violationCount = violationCount + 1;
        }

        on eEmergencyHalt do {
            goto Halted;
        }

        on eNetworkShutdown do {
            goto Shutdown;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Running;
        }
    }

    state Shutdown {
        entry {
            networkRunning = false;

            // Assert clean shutdown
            assert sizeof(componentsReady) == 0,
                "Components still ready during shutdown";
        }
    }
}

// =====================================================================
// CROSS-CHAIN CONSISTENCY MONITOR
// Ensures consistency across all KNIRV chains
// =====================================================================

spec CrossChainConsistencyMonitor observes
    eSendCrossChainMessage, eCrossChainMessageSent,
    eCrossChainMessageReceived, eCrossChainMessageFailed,
    eIBCSendPacket, eIBCPacketSent, eIBCPacketReceived,
    eIBCPacketAcknowledged, eIBCPacketTimeout
{
    var pendingMessages: map[UUID, CrossChainMessage];
    var deliveredMessages: set[UUID];
    var failedMessages: set[UUID];
    var totalMessagesSent: int;
    var totalMessagesDelivered: int;
    var totalMessagesFailed: int;

    start state Monitoring {
        entry {
            pendingMessages = default(map[UUID, CrossChainMessage]);
            deliveredMessages = default(set[UUID]);
            failedMessages = default(set[UUID]);
            totalMessagesSent = 0;
            totalMessagesDelivered = 0;
            totalMessagesFailed = 0;
        }

        on eSendCrossChainMessage do (payload: (message: CrossChainMessage)) {
            pendingMessages[payload.message.id] = payload.message;
        }

        on eCrossChainMessageSent do (payload: (messageID: UUID)) {
            totalMessagesSent = totalMessagesSent + 1;
        }

        on eCrossChainMessageReceived do (payload: (message: CrossChainMessage)) {
            if (payload.message.id in pendingMessages) {
                pendingMessages -= payload.message.id;
                deliveredMessages = deliveredMessages + (payload.message.id);
                totalMessagesDelivered = totalMessagesDelivered + 1;
            }
        }

        on eCrossChainMessageFailed do (payload: (messageID: UUID, reason: string)) {
            if (payload.messageID in pendingMessages) {
                pendingMessages -= payload.messageID;
                failedMessages = failedMessages + (payload.messageID);
                totalMessagesFailed = totalMessagesFailed + 1;

                announce eMonitorViolation, ("CrossChainConsistency", format("Cross-chain message failed: {0}", payload.reason), default(map[string, any]));
            }
        }

        on eIBCPacketTimeout do (payload: (sequence: int)) {
            // IBC timeout is a cross-chain consistency issue
            announce eMonitorViolation, ("CrossChainConsistency", format("IBC packet timeout: sequence {0}", payload.sequence), default(map[string, any]));
        }
    }
}

// =====================================================================
// TOKEN ECONOMICS INVARIANT MONITOR
// Ensures token supply and economics rules are maintained
// =====================================================================

spec TokenEconomicsMonitor observes
    eMintRequest, eMintResponse, eMintFailed,
    eBurnRequest, eBurnResponse,
    eTransferRequest, eTransferResponse, eTransferFailed,
    eTreasuryDeposit, eTreasuryWithdrawal,
    eStake, eStaked, eUnstake, eUnstaked,
    eRewardsDistributed
{
    var totalMinted: int;
    var totalBurned: int;
    var treasuryBalance: int;
    var totalStaked: int;
    var unauthorizedMintAttempts: int;

    start state Monitoring {
        entry {
            totalMinted = 0;
            totalBurned = 0;
            treasuryBalance = 0;
            totalStaked = 0;
            unauthorizedMintAttempts = 0;
        }

        on eMintRequest do (payload: (recipient: Address, amount: BigInt, authorized: bool)) {
            if (!payload.authorized) {
                unauthorizedMintAttempts = unauthorizedMintAttempts + 1;
            }
        }

        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                totalMinted = totalMinted + payload.receipt.amount.value;

                // Invariant: unauthorized mints should never succeed
                assert payload.receipt.authorized,
                    "Unauthorized mint succeeded - CRITICAL SECURITY VIOLATION";
            }
        }

        on eMintFailed do (reason: string) {
            // Expected for unauthorized attempts
            announce eMonitorAssertion, ("TokenEconomics", format("Mint correctly rejected: {0}", reason), true);
        }

        on eBurnResponse do (payload: (success: bool, amount: BigInt)) {
            if (payload.success) {
                totalBurned = totalBurned + payload.amount.value;

                // Invariant: can't burn more than minted
                assert totalBurned <= totalMinted,
                    "Total burned exceeds total minted - CRITICAL INVARIANT VIOLATION";
            }
        }

        on eTreasuryDeposit do (payload: (amount: BigInt, source: string)) {
            treasuryBalance = treasuryBalance + payload.amount.value;

            // Invariant: treasury should never be negative
            assert treasuryBalance >= 0,
                "Treasury balance is negative - CRITICAL INVARIANT VIOLATION";
        }

        on eTreasuryWithdrawal do (payload: (amount: BigInt, destination: Address, proposalID: string)) {
            treasuryBalance = treasuryBalance - payload.amount.value;

            // Invariant: treasury should never go negative
            assert treasuryBalance >= 0,
                "Treasury withdrawal caused negative balance - CRITICAL INVARIANT VIOLATION";
        }

        on eStaked do (stakeInfo: StakeInfo) {
            totalStaked = totalStaked + stakeInfo.amount.value;
        }

        on eUnstaked do (unstakeData: UnstakeData) {
            totalStaked = totalStaked - unstakeData.amount.value;

            // Invariant: total staked should never be negative
            assert totalStaked >= 0,
                "Total staked is negative - CRITICAL INVARIANT VIOLATION";
        }
    }
}

// =====================================================================
// GOVERNANCE PROCESS MONITOR
// Ensures governance rules are followed
// =====================================================================

spec GovernanceProcessMonitor observes
    eCreateProposal, eProposalCreated, eProposalCreationFailed,
    eCastVote, eVoteCast, eVoteFailed,
    eProposalStatusChanged, eExecuteProposal, eProposalExecuted,
    eRegisterRequest, eRegistrationConfirm, eRegistrationFail
{
    var activeProposals: set[string];
    var proposalVoters: map[string, set[Address]];
    var proposalStatuses: map[string, int];
    var doubleVoteAttempts: int;
    var prematureExecutions: int;
    var tmpVoters: set[Address];
    var tmpValidTransition: bool;

    start state Monitoring {
        entry {
            activeProposals = default(set[string]);
            proposalVoters = default(map[string, set[Address]]);
            proposalStatuses = default(map[string, int]);
            doubleVoteAttempts = 0;
            prematureExecutions = 0;
        }

        on eProposalCreated do (proposal: Proposal) {
            activeProposals = activeProposals + (proposal.id);
            proposalVoters[proposal.id] = default(set[Address]);
            proposalStatuses[proposal.id] = proposal.status.status;
        }

        on eCastVote do (payload: (proposalID: string, voter: Address, option: VoteOption, votingPower: BigInt)) {
            // Check for double voting attempt
            if (payload.proposalID in proposalVoters) {
                tmpVoters = proposalVoters[payload.proposalID];

                if (payload.voter in tmpVoters) {
                    doubleVoteAttempts = doubleVoteAttempts + 1;
                }
            }
        }

        on eVoteCast do (payload: (proposalID: string, voter: Address)) {
            // Record successful vote
            if (payload.proposalID in proposalVoters) {
                tmpVoters = proposalVoters[payload.proposalID];
                tmpVoters = tmpVoters + (payload.voter);
                proposalVoters[payload.proposalID] = tmpVoters;
            }
        }

        on eVoteFailed do (payload: (proposalID: string, reason: string)) {
            // Double vote attempts should fail
            if (reason == "Already voted") {
                announce eMonitorAssertion, ("GovernanceProcess", "Double vote correctly rejected", true);
            }
        }

        on eProposalStatusChanged do (payload: (proposalID: string, oldStatus: ProposalStatus, newStatus: ProposalStatus)) {
            proposalStatuses[payload.proposalID] = payload.newStatus.status;

            // Invariant: status transitions must be valid
            // Valid transitions: 0->1 (Pending->Active), 1->2/3/5 (Active->Passed/Rejected/Vetoed), 2->4 (Passed->Executed)
            tmpValidTransition = false;

            if (payload.oldStatus.status == 0 && payload.newStatus.status == 1) {
                tmpValidTransition = true;
            } else if (payload.oldStatus.status == 1 && (payload.newStatus.status == 2 || payload.newStatus.status == 3 || payload.newStatus.status == 5)) {
                tmpValidTransition = true;
            } else if (payload.oldStatus.status == 2 && payload.newStatus.status == 4) {
                tmpValidTransition = true;
            }

            assert tmpValidTransition,
                format("Invalid proposal status transition: {0} -> {1}", payload.oldStatus.status, payload.newStatus.status);
        }

        on eExecuteProposal do (payload: (proposalID: string)) {
            // Check proposal is in passed status before execution
            if (payload.proposalID in proposalStatuses) {
                if (proposalStatuses[payload.proposalID] != 2) {
                    prematureExecutions = prematureExecutions + 1;

                    announce eMonitorViolation, ("GovernanceProcess", "Attempted to execute non-passed proposal", default(map[string, any]));
                }
            }
        }

        on eProposalExecuted do (proposalID: string) {
            activeProposals = activeProposals - (proposalID);
            proposalStatuses[proposalID] = 4;  // Executed
        }
    }
}

// =====================================================================
// VALIDATION INTEGRITY MONITOR
// Ensures validation tasks are processed correctly
// =====================================================================

spec ValidationIntegrityMonitor observes
    eSubmitValidationTask, eValidationTaskQueued, eValidationTaskRejected,
    eStartValidation, eValidationStarted,
    eValidationComplete, eValidationFailed, eValidationTimeout,
    eResourceLimitExceeded
{
    var queuedTasks: set[UUID];
    var runningTasks: set[UUID];
    var completedTasks: set[UUID];
    var failedTasks: set[UUID];
    var resourceViolations: int;

    start state Monitoring {
        entry {
            queuedTasks = default(set[UUID]);
            runningTasks = default(set[UUID]);
            completedTasks = default(set[UUID]);
            failedTasks = default(set[UUID]);
            resourceViolations = 0;
        }

        on eValidationTaskQueued do (payload: (task: ValidationTask)) {
            queuedTasks = queuedTasks + (payload.task.id);
        }

        on eValidationStarted do (payload: (taskID: UUID)) {
            // Invariant: task must be queued before starting
            assert payload.taskID in queuedTasks,
                "Task started without being queued";

            queuedTasks = queuedTasks - (payload.taskID);
            runningTasks = runningTasks + (payload.taskID);
        }

        on eValidationComplete do (payload: (result: ValidationResult)) {
            // Invariant: task must be running before completion
            assert payload.result.taskID in runningTasks,
                "Task completed without being started";

            runningTasks = runningTasks - (payload.result.taskID);
            completedTasks = completedTasks + (payload.result.taskID);
        }

        on eValidationFailed do (payload: (taskID: UUID, reason: string)) {
            if (payload.taskID in runningTasks) {
                runningTasks = runningTasks - (payload.taskID);
            }
            failedTasks = failedTasks + (payload.taskID);
        }

        on eValidationTimeout do (payload: (taskID: UUID)) {
            if (payload.taskID in runningTasks) {
                runningTasks = runningTasks - (payload.taskID);
            }
            failedTasks = failedTasks + (payload.taskID);

            announce eMonitorViolation, ("ValidationIntegrity", format("Validation timeout: {0}", payload.taskID.value), default(map[string, any]));
        }

        on eResourceLimitExceeded do (payload: (sandboxID: UUID, resourceType: string, limit: int, actual: int)) {
            resourceViolations = resourceViolations + 1;

            // This is expected behavior (sandbox correctly enforcing limits)
            announce eMonitorAssertion, ("ValidationIntegrity", format("Resource limit correctly enforced: {0}", payload.resourceType), true);
        }
    }
}

// =====================================================================
// KNOWLEDGE GRAPH CONSISTENCY MONITOR
// Ensures knowledge graph maintains consistency
// =====================================================================

spec KnowledgeGraphConsistencyMonitor observes
    eRecordError, eErrorRecorded, eErrorDuplicate,
    eSubmitSolution, eSolutionSubmitted, eSolutionValidated,
    eLinkErrorToSolution, eErrorSolutionLinked,
    ePatternDetected, ePatternUpdated
{
    var errorCount: int;
    var solutionCount: int;
    var linkedPairs: int;
    var patternCount: int;
    var duplicateErrors: int;

    start state Monitoring {
        entry {
            errorCount = 0;
            solutionCount = 0;
            linkedPairs = 0;
            patternCount = 0;
            duplicateErrors = 0;
        }

        on eErrorRecorded do (payload: (record: ErrorRecord)) {
            errorCount = errorCount + 1;
        }

        on eErrorDuplicate do (payload: (existingID: UUID, occurrences: int)) {
            duplicateErrors = duplicateErrors + 1;

            // Duplicate detection is working correctly
            announce eMonitorAssertion, ("KnowledgeGraphConsistency", format("Duplicate error detected, occurrences: {0}", payload.occurrences), true);
        }

        on eSolutionSubmitted do (payload: (solution: SolutionRecord)) {
            solutionCount = solutionCount + 1;
        }

        on eErrorSolutionLinked do (payload: (errorID: UUID, solutionID: UUID)) {
            linkedPairs = linkedPairs + 1;
        }

        on ePatternDetected do (payload: (pattern: ErrorPattern)) {
            patternCount = patternCount + 1;

            announce eMonitorAssertion, ("KnowledgeGraphConsistency", format("Pattern detected: {0}", payload.pattern.patternName), true);
        }

        on eSolutionValidated do (payload: (solutionID: UUID, newEffectiveness: int)) {
            // Invariant: effectiveness should be 0-100
            assert payload.newEffectiveness >= 0 && payload.newEffectiveness <= 100,
                format("Invalid effectiveness score: {0}", payload.newEffectiveness);
        }
    }
}
