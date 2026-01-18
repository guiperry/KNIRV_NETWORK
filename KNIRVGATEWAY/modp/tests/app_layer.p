// app_layer.p - Application Layer Test Machine for Compositional Testing
// Simulates user behavior and generates test scenarios for the oracle system.
// Based on: Section 5.4 of p_implementation.md

// =====================================================================
// AppLayer Test Machine
// Simulates user behavior including:
// - Valid operations (transfers, proposals, votes)
// - Malicious/invalid behavior (insufficient funds, double voting, etc.)
// =====================================================================

machine AppLayerTestMachine {
    // Test configuration
    var testMode: string;  // "normal", "malicious", "stress"
    var testUsers: seq[Address];
    var testIteration: int;
    var maxIterations: int;
    var currentTime: int;  // Simulated time

    // Test counters
    var transfersAttempted: int;
    var transfersSucceeded: int;
    var transfersFailed: int;
    var proposalsCreated: int;
    var votesAttempted: int;
    var registrationsAttempted: int;

    // Reference to oracle machines
    var tokenMachine: machine;
    var governanceMachine: machine;
    var economicsMachine: machine;

    start state Init {
        entry {
            testMode = "normal";
            testUsers = default(seq[Address]);
            testIteration = 0;
            maxIterations = 100;
            currentTime = 0;
            transfersAttempted = 0;
            transfersSucceeded = 0;
            transfersFailed = 0;
            proposalsCreated = 0;
            votesAttempted = 0;
            registrationsAttempted = 0;

            // Create test users
            var i: int;
            i = 0;
            while (i < 10) {
                var userAddr: Address;
                userAddr = (bytes = default(seq[int]));
                testUsers += (i, userAddr);
                i = i + 1;
            }
        }

        on eOracleReady do {
            goto Running;
        }
    }

    state Running {
        entry {
            // Start generating test scenarios
            if (testMode == "normal") {
                goto NormalTesting;
            } else if (testMode == "malicious") {
                goto MaliciousTesting;
            } else {
                goto StressTesting;
            }
        }
    }

    // ===================================================================
    // Normal Testing - Valid Operations
    // ===================================================================
    state NormalTesting {
        entry {
            // Generate normal test scenarios
            GenerateNormalTransfer();
        }

        on eTransferResponse do (payload: (success: bool, receipt: TransferReceipt)) {
            if (payload.success) {
                transfersSucceeded = transfersSucceeded + 1;
            }
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eTransferFailed do {
            transfersFailed = transfersFailed + 1;
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eProposalCreated do (payload: (proposal: Proposal)) {
            proposalsCreated = proposalsCreated + 1;
            // After creating proposal, vote on it
            GenerateVote(payload.proposal.id);
        }

        on eVoteCast do {
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eVoteFailed do {
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eRegistrationConfirm do {
            registrationsAttempted = registrationsAttempted + 1;
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eRegistrationFail do {
            testIteration = testIteration + 1;
            AdvanceTest();
        }
    }

    // ===================================================================
    // Malicious Testing - Invalid/Attack Operations
    // ===================================================================
    state MaliciousTesting {
        entry {
            // Generate malicious test scenarios
            GenerateMaliciousScenario();
        }

        // Track responses to malicious operations
        on eTransferFailed do {
            // Expected - malicious transfer should fail
            transfersFailed = transfersFailed + 1;
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eTransferResponse do (payload: (success: bool, receipt: TransferReceipt)) {
            // Unexpected if we tried an invalid transfer
            if (payload.success) {
                // This might be a security issue
                announce eMonitorViolation, (
                    monitorName = "AppLayerTest",
                    violation = "Invalid transfer succeeded",
                    details = default(map[string, any])
                );
            }
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eMintFailed do {
            // Expected - unauthorized mint should fail
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                // Unauthorized mint succeeded - security issue!
                announce eMonitorViolation, (
                    monitorName = "AppLayerTest",
                    violation = "Unauthorized mint succeeded",
                    details = default(map[string, any])
                );
            }
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eVoteFailed do {
            // Expected for double votes, expired proposals, etc.
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eVoteCast do {
            // Might indicate double vote was allowed
            testIteration = testIteration + 1;
            AdvanceTest();
        }

        on eProposalCreationFailed do {
            // Expected for invalid proposals
            testIteration = testIteration + 1;
            AdvanceTest();
        }
    }

    // ===================================================================
    // Stress Testing - High Volume Operations
    // ===================================================================
    state StressTesting {
        entry {
            // Generate many concurrent operations
            GenerateStressScenario();
        }

        on eTransferResponse do {
            transfersSucceeded = transfersSucceeded + 1;
            testIteration = testIteration + 1;
            if (testIteration < maxIterations) {
                GenerateStressScenario();
            } else {
                goto TestComplete;
            }
        }

        on eTransferFailed do {
            transfersFailed = transfersFailed + 1;
            testIteration = testIteration + 1;
            if (testIteration < maxIterations) {
                GenerateStressScenario();
            } else {
                goto TestComplete;
            }
        }
    }

    state TestComplete {
        entry {
            // Report test results
            announce eMonitorAssertion, (
                monitorName = "AppLayerTest",
                assertion = format("Test completed: {0} iterations, {1} transfers, {2} succeeded",
                    testIteration, transfersAttempted, transfersSucceeded),
                passed = true
            );
        }
    }

    // ===================================================================
    // Helper Functions
    // ===================================================================

    fun AdvanceTest() {
        currentTime = currentTime + 1;

        if (testIteration >= maxIterations) {
            goto TestComplete;
        }

        // Randomly select next action
        var action: int;
        action = testIteration % 5;

        if (action == 0) {
            GenerateNormalTransfer();
        } else if (action == 1) {
            GenerateProposal();
        } else if (action == 2) {
            GenerateRegistration();
        } else if (action == 3) {
            GenerateStakeOperation();
        } else {
            GenerateFeePayment();
        }
    }

    // Generate a normal transfer between two users
    fun GenerateNormalTransfer() {
        if (sizeof(testUsers) < 2) return;

        var fromIndex: int;
        var toIndex: int;
        fromIndex = testIteration % sizeof(testUsers);
        toIndex = (testIteration + 1) % sizeof(testUsers);

        var from: Address;
        var to: Address;
        from = testUsers[fromIndex];
        to = testUsers[toIndex];

        var amount: BigInt;
        amount = (value = 1000000, isNegative = false);  // 1 NRN

        transfersAttempted = transfersAttempted + 1;
        send tokenMachine, eTransferRequest, (from = from, to = to, amount = amount);
    }

    // Generate malicious test scenarios
    fun GenerateMaliciousScenario() {
        var scenario: int;
        scenario = testIteration % 4;

        if (scenario == 0) {
            // Attempt transfer with insufficient balance
            var from: Address;
            var to: Address;
            from = testUsers[0];
            to = testUsers[1];
            var hugeAmount: BigInt;
            hugeAmount = (value = 999999999999, isNegative = false);

            send tokenMachine, eTransferRequest, (from = from, to = to, amount = hugeAmount);
        } else if (scenario == 1) {
            // Attempt unauthorized minting
            var to: Address;
            to = testUsers[0];
            var amount: BigInt;
            amount = (value = 1000000000, isNegative = false);

            send tokenMachine, eMintRequest, (to = to, amount = amount, authorized = false);
        } else if (scenario == 2) {
            // Attempt to create proposal with insufficient deposit
            var proposer: Address;
            proposer = testUsers[0];
            var deposit: BigInt;
            deposit = (value = 1, isNegative = false);  // Way too small

            send governanceMachine, eCreateProposal, (
                proposalType = (typeName = "NetworkParameter"),
                title = "Test Proposal",
                description = "Testing insufficient deposit",
                proposer = proposer,
                deposit = deposit,
                content = default(map[string, any])
            );
        } else {
            // Attempt double vote (if we have an active proposal)
            var voter: Address;
            voter = testUsers[0];
            var votingPower: BigInt;
            votingPower = (value = 100, isNegative = false);

            send governanceMachine, eCastVote, (
                proposalID = "prop_1",  // Assuming we voted on this already
                voter = voter,
                option = (option = 1),  // Yes
                votingPower = votingPower
            );
        }
    }

    // Generate a governance proposal
    fun GenerateProposal() {
        var proposerIndex: int;
        proposerIndex = testIteration % sizeof(testUsers);

        var proposer: Address;
        proposer = testUsers[proposerIndex];

        var deposit: BigInt;
        deposit = (value = 100000000, isNegative = false);  // 100 NRN

        send governanceMachine, eCreateProposal, (
            proposalType = (typeName = "NetworkParameter"),
            title = format("Test Proposal {0}", proposalsCreated),
            description = "Automated test proposal",
            proposer = proposer,
            deposit = deposit,
            content = default(map[string, any])
        );
    }

    // Generate a vote on a proposal
    fun GenerateVote(proposalID: string) {
        var voterIndex: int;
        voterIndex = testIteration % sizeof(testUsers);

        var voter: Address;
        voter = testUsers[voterIndex];

        var votingPower: BigInt;
        votingPower = (value = 100, isNegative = false);

        var option: VoteOption;
        option = (option = (testIteration % 4));  // Rotate through vote options

        votesAttempted = votesAttempted + 1;
        send governanceMachine, eCastVote, (
            proposalID = proposalID,
            voter = voter,
            option = option,
            votingPower = votingPower
        );
    }

    // Generate a "Second Me" registration
    fun GenerateRegistration() {
        var userIndex: int;
        userIndex = testIteration % sizeof(testUsers);

        var user: Address;
        user = testUsers[userIndex];

        registrationsAttempted = registrationsAttempted + 1;
        send governanceMachine, eRegisterRequest, (
            user = user,
            initialData = default(map[string, any])
        );
    }

    // Generate a staking operation
    fun GenerateStakeOperation() {
        var stakerIndex: int;
        stakerIndex = testIteration % sizeof(testUsers);

        var staker: Address;
        staker = testUsers[stakerIndex];

        var amount: BigInt;
        amount = (value = 10000000, isNegative = false);  // 10 NRN

        send economicsMachine, eStake, (
            staker = staker,
            amount = amount,
            validator = testUsers[0]  // Stake with first user as validator
        );
    }

    // Generate a fee payment
    fun GenerateFeePayment() {
        var payerIndex: int;
        payerIndex = testIteration % sizeof(testUsers);

        var payer: Address;
        payer = testUsers[payerIndex];

        var amount: BigInt;
        amount = (value = 100000, isNegative = false);  // 0.1 NRN fee

        send economicsMachine, eProcessFee, (
            from = payer,
            feeType = (feeName = "service_fee"),
            amount = amount
        );
    }

    // Generate stress test scenarios
    fun GenerateStressScenario() {
        // Rapid-fire transfers
        GenerateNormalTransfer();
    }
}

// =====================================================================
// Test Driver - Orchestrates Test Execution
// =====================================================================

machine TestDriver {
    var appLayer: machine;
    var tokenMachine: machine;
    var governanceMachine: machine;
    var economicsMachine: machine;
    var consensusMachine: machine;
    var ibcMachine: machine;
    var p2pInterface: machine;

    // Monitors
    var treasuryInvariant: machine;
    var tokenSupplyInvariant: machine;
    var governanceInvariant: machine;

    start state Init {
        entry {
            // Create all machines
            tokenMachine = new TokenMachine();
            governanceMachine = new GovernanceMachine();
            economicsMachine = new EconomicsMachine();
            consensusMachine = new ConsensusMachine();
            ibcMachine = new IBCMachine();
            p2pInterface = new P2PInterface();

            // Create monitors
            treasuryInvariant = new TreasuryInvariant();
            tokenSupplyInvariant = new TokenSupplyInvariant();
            governanceInvariant = new GovernanceInvariant();

            // Create app layer test machine
            appLayer = new AppLayerTestMachine();

            // Start the oracle
            send tokenMachine, eOracleStart, ();
            send governanceMachine, eOracleStart, ();
            send economicsMachine, eOracleStart, ();
            send consensusMachine, eOracleStart, ();
            send ibcMachine, eOracleStart, ();
            send p2pInterface, eOracleStart, ();

            // Notify app layer that oracle is ready
            send appLayer, eOracleReady, ();
        }
    }
}
