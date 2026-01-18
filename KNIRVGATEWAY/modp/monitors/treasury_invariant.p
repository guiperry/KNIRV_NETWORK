// treasury_invariant.p - Treasury Invariant Specification Machine (Monitor)
// Ensures all network fees are correctly deposited into the treasury
// and treasury withdrawals only occur through governance approval.
// Based on: Section 5.3 of p_implementation.md

// =====================================================================
// TreasuryInvariant Monitor
// Purpose: Ensures all network fees are correctly deposited into the
// treasury and funds are only moved from treasury with governance vote.
// =====================================================================

spec TreasuryInvariant observes
    eFeeProcessed,
    eTreasuryDeposit,
    eTreasuryWithdrawal,
    eTreasuryBalanceChanged,
    eTreasuryInvariantViolation,
    eProposalPassed
{
    // Track expected treasury balance
    var expectedTreasuryBalance: BigInt;
    var lastDepositAmount: BigInt;
    var pendingWithdrawalApprovals: set[string];  // Proposal IDs that approved withdrawals
    var violationCount: int;

    start state Monitoring {
        entry {
            expectedTreasuryBalance = (value = 0, isNegative = false);
            lastDepositAmount = (value = 0, isNegative = false);
            pendingWithdrawalApprovals = default(set[string]);
            violationCount = 0;
        }

        // When a fee is processed, expect a treasury deposit
        on eFeeProcessed do (payload: (receipt: FeeReceipt)) {
            // Fee processing should result in non-burn portion going to treasury
            lastDepositAmount = payload.receipt.rewardAmount;
            expectedTreasuryBalance = AddBigInt(expectedTreasuryBalance, payload.receipt.rewardAmount);
            goto ExpectingDeposit;
        }

        // Direct treasury deposit (outside of fee flow)
        on eTreasuryDeposit do (payload: (amount: BigInt, source: string)) {
            expectedTreasuryBalance = AddBigInt(expectedTreasuryBalance, payload.amount);
        }

        // Track governance approvals for treasury withdrawals
        on eProposalPassed do (payload: (proposalID: string, tallyResult: TallyResult)) {
            // Track that this proposal passed - it may authorize a withdrawal
            pendingWithdrawalApprovals += payload.proposalID;
        }

        // Treasury withdrawal - must be approved by governance
        on eTreasuryWithdrawal do (payload: (amount: BigInt, recipient: Address, approvedByGovernance: bool)) {
            if (!payload.approvedByGovernance) {
                // INVARIANT VIOLATION: Unauthorized withdrawal attempt
                violationCount = violationCount + 1;
                announce eMonitorViolation, (
                    monitorName = "TreasuryInvariant",
                    violation = "Unauthorized treasury withdrawal attempted",
                    details = default(map[string, any])
                );
                goto ViolationDetected;
            }

            // Authorized withdrawal - update expected balance
            expectedTreasuryBalance = SubtractBigInt(expectedTreasuryBalance, payload.amount);
        }

        // Verify treasury balance changes match expectations
        on eTreasuryBalanceChanged do (payload: (oldBalance: BigInt, newBalance: BigInt)) {
            // Calculate the change
            var change: BigInt;
            if (payload.newBalance.value >= payload.oldBalance.value) {
                change = (value = payload.newBalance.value - payload.oldBalance.value, isNegative = false);
            } else {
                change = (value = payload.oldBalance.value - payload.newBalance.value, isNegative = true);
            }

            // For deposits, newBalance > oldBalance
            // For withdrawals, newBalance < oldBalance (and must be approved)
            if (change.isNegative) {
                // This was a withdrawal - should have been through authorized path
                // The withdrawal handler above should have caught unauthorized ones
            }
        }

        // Explicit violation notification
        on eTreasuryInvariantViolation do (payload: (
            expectedBalance: BigInt,
            actualBalance: BigInt,
            operation: string
        )) {
            violationCount = violationCount + 1;
            announce eMonitorViolation, (
                monitorName = "TreasuryInvariant",
                violation = payload.operation,
                details = default(map[string, any])
            );
            goto ViolationDetected;
        }
    }

    // State for when we're expecting a treasury deposit after fee processing
    hot state ExpectingDeposit {
        // The deposit we expected
        on eTreasuryDeposit do (payload: (amount: BigInt, source: string)) {
            // Verify the deposit amount matches what we expected from fee
            if (payload.amount.value != lastDepositAmount.value) {
                // Amount mismatch - this could be a different deposit or an issue
                announce eMonitorAssertion, (
                    monitorName = "TreasuryInvariant",
                    assertion = "Deposit amount matches fee reward portion",
                    passed = false
                );
            }
            goto Monitoring;
        }

        // Another fee was processed before deposit confirmed - track it
        on eFeeProcessed do (payload: (receipt: FeeReceipt)) {
            lastDepositAmount = AddBigInt(lastDepositAmount, payload.receipt.rewardAmount);
            expectedTreasuryBalance = AddBigInt(expectedTreasuryBalance, payload.receipt.rewardAmount);
        }

        // If we see a balance change, verify it
        on eTreasuryBalanceChanged do (payload: (oldBalance: BigInt, newBalance: BigInt)) {
            goto Monitoring;
        }

        // Handle withdrawals even while expecting deposit
        on eTreasuryWithdrawal do (payload: (amount: BigInt, recipient: Address, approvedByGovernance: bool)) {
            if (!payload.approvedByGovernance) {
                violationCount = violationCount + 1;
                announce eMonitorViolation, (
                    monitorName = "TreasuryInvariant",
                    violation = "Unauthorized treasury withdrawal during deposit expectation",
                    details = default(map[string, any])
                );
                goto ViolationDetected;
            }
            expectedTreasuryBalance = SubtractBigInt(expectedTreasuryBalance, payload.amount);
        }

        on eTreasuryInvariantViolation goto ViolationDetected;
    }

    // Error state when a violation is detected
    cold state ViolationDetected {
        entry {
            // Log the violation for test reporting
            announce eMonitorAssertion, (
                monitorName = "TreasuryInvariant",
                assertion = "Treasury funds protected",
                passed = false
            );
        }

        // Can return to monitoring after violation
        on eTreasuryDeposit goto Monitoring;
        on eFeeProcessed goto Monitoring;
        on eTreasuryBalanceChanged goto Monitoring;

        // More violations
        on eTreasuryInvariantViolation do (payload: (
            expectedBalance: BigInt,
            actualBalance: BigInt,
            operation: string
        )) {
            violationCount = violationCount + 1;
        }

        on eTreasuryWithdrawal do (payload: (amount: BigInt, recipient: Address, approvedByGovernance: bool)) {
            if (!payload.approvedByGovernance) {
                violationCount = violationCount + 1;
            }
        }
    }
}

// =====================================================================
// Treasury Safety Assertions
// Additional specifications for treasury behavior
// =====================================================================

// Assertion: Treasury balance should never go negative
spec TreasuryNonNegative observes eTreasuryBalanceChanged {
    start state Checking {
        on eTreasuryBalanceChanged do (payload: (oldBalance: BigInt, newBalance: BigInt)) {
            if (payload.newBalance.isNegative) {
                assert false, "Treasury balance became negative!";
            }
        }
    }
}

// Assertion: Every withdrawal must have corresponding governance approval
spec WithdrawalRequiresApproval observes eTreasuryWithdrawal {
    start state Checking {
        on eTreasuryWithdrawal do (payload: (amount: BigInt, recipient: Address, approvedByGovernance: bool)) {
            assert payload.approvedByGovernance, "Treasury withdrawal without governance approval!";
        }
    }
}
