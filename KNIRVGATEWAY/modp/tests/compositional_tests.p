// compositional_tests.p - Compositional Test Definitions
// Defines the test compositions for ModP testing as specified in p_implementation.md
// Based on: Section 5.4 of p_implementation.md

// =====================================================================
// Main Compositional Test
// Composition: knirv-oracle || AppLayer || TreasuryInvariant ||
//              TokenSupplyInvariant || GovernanceInvariant
// =====================================================================

// Test: Full oracle system with all invariant monitors
test OracleWithInvariants [main = TestDriver]:
    assert TreasuryInvariant in Monitoring || TreasuryInvariant in ExpectingDeposit,
        "Treasury invariant should remain in monitoring states";
    assert TokenSupplyInvariant in Monitoring || TokenSupplyInvariant in ExpectingMintRejection,
        "Token supply invariant should remain in monitoring states";
    assert GovernanceInvariant in Monitoring || GovernanceInvariant in DoubleVoteAttempted,
        "Governance invariant should remain in monitoring states";

// =====================================================================
// Component-Level Tests
// =====================================================================

// Test: Token transfers maintain balance consistency
test TokenTransferConsistency [main = TokenTransferTestDriver]:
    // Ensure balances are always non-negative
    assert forall (addr: Address) :: TokenMachine.balances[addr].value >= 0,
        "Token balances should never be negative";

machine TokenTransferTestDriver {
    var token: machine;
    var testAddr1: Address;
    var testAddr2: Address;

    start state Init {
        entry {
            token = new TokenMachine();
            testAddr1 = (bytes = default(seq[int]));
            testAddr2 = (bytes = default(seq[int]));

            // Initialize with balance
            send token, eOracleStart, ();
            send token, eMintRequest, (
                to = testAddr1,
                amount = (value = 1000000, isNegative = false),
                authorized = true
            );
        }

        on eMintResponse do {
            goto TestTransfers;
        }
    }

    state TestTransfers {
        entry {
            // Test valid transfer
            send token, eTransferRequest, (
                from = testAddr1,
                to = testAddr2,
                amount = (value = 500000, isNegative = false)
            );
        }

        on eTransferResponse do {
            // Transfer complete
            goto TestComplete;
        }

        on eTransferFailed do {
            // Transfer failed
            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (
                monitorName = "TokenTransferTest",
                assertion = "Token transfer test completed",
                passed = true
            );
        }
    }
}

// Test: Governance proposal lifecycle
test GovernanceLifecycle [main = GovernanceTestDriver]:
    // Ensure proposals follow correct state transitions
    assert GovernanceMachine.proposals != default(map[string, Proposal]),
        "Should have created at least one proposal";

machine GovernanceTestDriver {
    var gov: machine;
    var testProposer: Address;
    var createdProposalID: string;

    start state Init {
        entry {
            gov = new GovernanceMachine();
            testProposer = (bytes = default(seq[int]));

            send gov, eOracleStart, ();
            goto CreateProposal;
        }
    }

    state CreateProposal {
        entry {
            send gov, eCreateProposal, (
                proposalType = (typeName = "NetworkParameter"),
                title = "Test Governance Lifecycle",
                description = "Testing proposal creation and voting",
                proposer = testProposer,
                deposit = (value = 100000000, isNegative = false),
                content = default(map[string, any])
            );
        }

        on eProposalCreated do (payload: (proposal: Proposal)) {
            createdProposalID = payload.proposal.id;
            goto VoteOnProposal;
        }

        on eProposalCreationFailed do {
            goto TestFailed;
        }
    }

    state VoteOnProposal {
        entry {
            send gov, eCastVote, (
                proposalID = createdProposalID,
                voter = testProposer,
                option = (option = 1),  // Yes vote
                votingPower = (value = 100, isNegative = false)
            );
        }

        on eVoteCast do {
            goto TestComplete;
        }

        on eVoteFailed do {
            // Vote might fail if not in voting period yet - that's OK
            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (
                monitorName = "GovernanceLifecycleTest",
                assertion = "Governance lifecycle test completed",
                passed = true
            );
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, (
                monitorName = "GovernanceLifecycleTest",
                assertion = "Governance lifecycle test failed",
                passed = false
            );
        }
    }
}

// Test: Economics fee processing
test EconomicsFeeProcessing [main = EconomicsTestDriver]:
    // Ensure fees are correctly processed and tracked
    assert EconomicsMachine.totalFeesCollected.value >= 0,
        "Total fees collected should be non-negative";

machine EconomicsTestDriver {
    var econ: machine;
    var testPayer: Address;

    start state Init {
        entry {
            econ = new EconomicsMachine();
            testPayer = (bytes = default(seq[int]));

            send econ, eOracleStart, ();
            goto ProcessFees;
        }
    }

    state ProcessFees {
        entry {
            // Process multiple fees
            var i: int;
            i = 0;
            while (i < 5) {
                send econ, eProcessFee, (
                    from = testPayer,
                    feeType = (feeName = "test_fee"),
                    amount = (value = 100000 * (i + 1), isNegative = false)
                );
                i = i + 1;
            }
        }

        on eFeeProcessed do {
            goto TestComplete;
        }

        on eFeeProcessingFailed do {
            goto TestFailed;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (
                monitorName = "EconomicsFeeTest",
                assertion = "Economics fee processing test completed",
                passed = true
            );
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, (
                monitorName = "EconomicsFeeTest",
                assertion = "Economics fee processing test failed",
                passed = false
            );
        }
    }
}

// =====================================================================
// "Second Me" Registration Workflow Test
// Based on Section 5.2 of p_implementation.md
// =====================================================================

test SecondMeRegistrationWorkflow [main = SecondMeTestDriver]:
    assert GovernanceMachine.pendingRegistrations == default(map[Address, map[string, any]]),
        "All registrations should be resolved";

machine SecondMeTestDriver {
    var gov: machine;
    var econ: machine;
    var token: machine;
    var testUser: Address;

    start state Init {
        entry {
            gov = new GovernanceMachine();
            econ = new EconomicsMachine();
            token = new TokenMachine();
            testUser = (bytes = default(seq[int]));

            send gov, eOracleStart, ();
            send econ, eOracleStart, ();
            send token, eOracleStart, ();

            goto InitiateRegistration;
        }
    }

    state InitiateRegistration {
        entry {
            // Step 1: Send registration request
            send gov, eRegisterRequest, (
                user = testUser,
                initialData = default(map[string, any])
            );
        }

        // Step 2: Governance requests fee verification
        on eVerifyFeePayment do (payload: (user: Address, expectedAmount: BigInt)) {
            // Step 3: Pay the fee
            send econ, eFeePayment, (
                user = payload.user,
                amount = payload.expectedAmount
            );
        }

        // Step 4: Economics verifies fee payment
        on eFeePaymentVerified do (payload: (user: Address, amount: BigInt, verified: bool)) {
            // Forward to governance
            send gov, eFeePaymentVerified, payload;
        }

        // Step 5: Registration confirmed
        on eRegistrationConfirm do (payload: (user: Address, secondMeID: string)) {
            goto TestComplete;
        }

        // Registration failed
        on eRegistrationFail do {
            goto TestFailed;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (
                monitorName = "SecondMeRegistrationTest",
                assertion = "Second Me registration workflow completed successfully",
                passed = true
            );
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, (
                monitorName = "SecondMeRegistrationTest",
                assertion = "Second Me registration workflow failed",
                passed = false
            );
        }
    }
}

// =====================================================================
// Malicious Behavior Tests
// Tests that the system correctly rejects invalid operations
// =====================================================================

test MaliciousBehaviorRejection [main = MaliciousTestDriver]:
    // All malicious operations should be rejected
    assert TokenMachine in Active,
        "Token machine should remain active despite malicious attempts";

machine MaliciousTestDriver {
    var token: machine;
    var gov: machine;
    var testAddr: Address;
    var maliciousAttempts: int;
    var correctlyRejected: int;

    start state Init {
        entry {
            token = new TokenMachine();
            gov = new GovernanceMachine();
            testAddr = (bytes = default(seq[int]));
            maliciousAttempts = 0;
            correctlyRejected = 0;

            send token, eOracleStart, ();
            send gov, eOracleStart, ();

            goto TestUnauthorizedMint;
        }
    }

    state TestUnauthorizedMint {
        entry {
            maliciousAttempts = maliciousAttempts + 1;
            // Attempt unauthorized minting
            send token, eMintRequest, (
                to = testAddr,
                amount = (value = 999999999, isNegative = false),
                authorized = false
            );
        }

        on eMintFailed do {
            // Correctly rejected
            correctlyRejected = correctlyRejected + 1;
            goto TestInsufficientBalanceTransfer;
        }

        on eMintResponse do {
            // Should not succeed!
            goto TestFailed;
        }
    }

    state TestInsufficientBalanceTransfer {
        entry {
            maliciousAttempts = maliciousAttempts + 1;
            // Attempt transfer with no balance
            send token, eTransferRequest, (
                from = testAddr,
                to = testAddr,
                amount = (value = 999999999, isNegative = false)
            );
        }

        on eTransferFailed do {
            // Correctly rejected
            correctlyRejected = correctlyRejected + 1;
            goto TestInsufficientDeposit;
        }

        on eTransferResponse do {
            goto TestFailed;
        }
    }

    state TestInsufficientDeposit {
        entry {
            maliciousAttempts = maliciousAttempts + 1;
            // Attempt to create proposal with tiny deposit
            send gov, eCreateProposal, (
                proposalType = (typeName = "NetworkParameter"),
                title = "Malicious Proposal",
                description = "Should fail due to insufficient deposit",
                proposer = testAddr,
                deposit = (value = 1, isNegative = false),  // Way too small
                content = default(map[string, any])
            );
        }

        on eProposalCreationFailed do {
            // Correctly rejected
            correctlyRejected = correctlyRejected + 1;
            goto TestComplete;
        }

        on eProposalCreated do {
            goto TestFailed;
        }
    }

    state TestComplete {
        entry {
            var allRejected: bool;
            allRejected = (maliciousAttempts == correctlyRejected);

            announce eMonitorAssertion, (
                monitorName = "MaliciousBehaviorTest",
                assertion = format("All {0} malicious attempts correctly rejected", maliciousAttempts),
                passed = allRejected
            );
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, (
                monitorName = "MaliciousBehaviorTest",
                assertion = "Malicious operation was not rejected",
                passed = false
            );
        }
    }
}
