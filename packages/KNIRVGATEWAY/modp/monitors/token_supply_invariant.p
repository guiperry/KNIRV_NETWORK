// token_supply_invariant.p - Token Supply Invariant Specification Machine (Monitor)
// Enforces the network's monetary policy ensuring total supply never exceeds
// max supply and minting only occurs through authorized mechanisms.
// Based on: Section 5.3 of p_implementation.md

// =====================================================================
// TokenSupplyInvariant Monitor
// Purpose: Enforces network monetary policy by tracking token supply
// and ensuring only authorized minting operations occur.
// =====================================================================

spec TokenSupplyInvariant observes
    eMintRequest,
    eMintResponse,
    eMintFailed,
    eBurnResponse,
    eTokenInfoResponse,
    eTokenSupplyInvariantViolation,
    eProposalPassed,
    eBlockCommitted
{
    // Track token supply
    var trackedTotalSupply: BigInt;
    var trackedMaxSupply: BigInt;
    var pendingAuthorizedMints: set[Address];  // Addresses authorized to receive mints
    var violationCount: int;
    var blockRewardMintingAllowed: bool;  // Block rewards are authorized minting

    start state Monitoring {
        entry {
            trackedTotalSupply = (value = 1000000000, isNegative = false);  // Initial supply
            trackedMaxSupply = (value = 10000000000, isNegative = false);   // Max supply
            pendingAuthorizedMints = default(set[Address]);
            violationCount = 0;
            blockRewardMintingAllowed = true;  // Block rewards are always authorized
        }

        // Monitor mint requests
        on eMintRequest do (payload: (to: Address, amount: BigInt, authorized: bool)) {
            if (!payload.authorized) {
                // INVARIANT VIOLATION: Unauthorized minting attempt
                violationCount = violationCount + 1;
                announce eMonitorViolation, (
                    monitorName = "TokenSupplyInvariant",
                    violation = "Unauthorized minting attempted",
                    details = default(map[string, any])
                );

                // Don't go to violation state yet - wait to see if it was rejected
                goto ExpectingMintRejection;
            }

            // Authorized mint - check if it would exceed max supply
            var potentialNewSupply: BigInt;
            potentialNewSupply = AddBigInt(trackedTotalSupply, payload.amount);

            if (potentialNewSupply.value > trackedMaxSupply.value) {
                // This mint would exceed max supply
                announce eMonitorAssertion, (
                    monitorName = "TokenSupplyInvariant",
                    assertion = "Mint would exceed max supply",
                    passed = false
                );
            }
        }

        // Monitor successful mints
        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                // Update tracked supply
                trackedTotalSupply = payload.receipt.newTotalSupply;

                // Verify supply still within bounds
                if (trackedTotalSupply.value > trackedMaxSupply.value) {
                    violationCount = violationCount + 1;
                    announce eMonitorViolation, (
                        monitorName = "TokenSupplyInvariant",
                        violation = "Total supply exceeded max supply after mint",
                        details = default(map[string, any])
                    );
                    goto ViolationDetected;
                }

                announce eMonitorAssertion, (
                    monitorName = "TokenSupplyInvariant",
                    assertion = "Mint kept supply within bounds",
                    passed = true
                );
            }
        }

        // Monitor failed mints (good - unauthorized mints should fail)
        on eMintFailed do (payload: (to: Address, amount: BigInt, error: ErrorCode)) {
            // Mint failed - this is expected for unauthorized or over-supply mints
            announce eMonitorAssertion, (
                monitorName = "TokenSupplyInvariant",
                assertion = "Invalid mint correctly rejected",
                passed = true
            );
        }

        // Monitor burns (reduce supply)
        on eBurnResponse do (payload: (success: bool, receipt: BurnReceipt)) {
            if (payload.success) {
                trackedTotalSupply = payload.receipt.newTotalSupply;

                // Verify supply is non-negative
                if (trackedTotalSupply.isNegative) {
                    violationCount = violationCount + 1;
                    announce eMonitorViolation, (
                        monitorName = "TokenSupplyInvariant",
                        violation = "Burn resulted in negative supply",
                        details = default(map[string, any])
                    );
                    goto ViolationDetected;
                }
            }
        }

        // Sync supply from token info queries
        on eTokenInfoResponse do (payload: (info: TokenInfo)) {
            trackedTotalSupply = payload.info.totalSupply;
            trackedMaxSupply = payload.info.maxSupply;
        }

        // Block commits may include block rewards (authorized minting)
        on eBlockCommitted do (payload: (block: Block, appHash: seq[int])) {
            // Block rewards are an authorized form of minting
            // The actual mint would come through eMintRequest with authorized=true
        }

        // Governance can authorize specific minting operations
        on eProposalPassed do (payload: (proposalID: string, tallyResult: TallyResult)) {
            // A passed proposal might authorize minting
            // In real implementation, we'd parse the proposal content
        }

        // Explicit violation notification
        on eTokenSupplyInvariantViolation do (payload: (
            totalSupply: BigInt,
            maxSupply: BigInt,
            unauthorized: bool
        )) {
            violationCount = violationCount + 1;

            if (payload.unauthorized) {
                announce eMonitorViolation, (
                    monitorName = "TokenSupplyInvariant",
                    violation = "Unauthorized minting operation",
                    details = default(map[string, any])
                );
            } else {
                announce eMonitorViolation, (
                    monitorName = "TokenSupplyInvariant",
                    violation = "Supply exceeded max supply",
                    details = default(map[string, any])
                );
            }

            goto ViolationDetected;
        }
    }

    // State when we detected an unauthorized mint request
    hot state ExpectingMintRejection {
        // If the mint fails, that's correct behavior
        on eMintFailed do (payload: (to: Address, amount: BigInt, error: ErrorCode)) {
            announce eMonitorAssertion, (
                monitorName = "TokenSupplyInvariant",
                assertion = "Unauthorized mint correctly rejected",
                passed = true
            );
            goto Monitoring;
        }

        // If the mint succeeds after unauthorized request, that's a violation
        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                announce eMonitorViolation, (
                    monitorName = "TokenSupplyInvariant",
                    violation = "Unauthorized mint was not rejected",
                    details = default(map[string, any])
                );
                goto ViolationDetected;
            }
            goto Monitoring;
        }

        // Continue monitoring other events
        on eBurnResponse goto Monitoring;
        on eTokenInfoResponse goto Monitoring;
        on eBlockCommitted goto Monitoring;
        on eProposalPassed goto Monitoring;
        on eTokenSupplyInvariantViolation goto ViolationDetected;
    }

    // Error state when a violation is detected
    cold state ViolationDetected {
        entry {
            announce eMonitorAssertion, (
                monitorName = "TokenSupplyInvariant",
                assertion = "Token supply invariant maintained",
                passed = false
            );
        }

        // Can continue monitoring after violation
        on eMintRequest goto Monitoring;
        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                trackedTotalSupply = payload.receipt.newTotalSupply;
            }
            goto Monitoring;
        }
        on eMintFailed goto Monitoring;
        on eBurnResponse do (payload: (success: bool, receipt: BurnReceipt)) {
            if (payload.success) {
                trackedTotalSupply = payload.receipt.newTotalSupply;
            }
            goto Monitoring;
        }
        on eTokenInfoResponse do (payload: (info: TokenInfo)) {
            trackedTotalSupply = payload.info.totalSupply;
            goto Monitoring;
        }

        on eTokenSupplyInvariantViolation do {
            violationCount = violationCount + 1;
        }
    }
}

// =====================================================================
// Additional Supply Invariant Specifications
// =====================================================================

// Assertion: Total supply should never exceed max supply
spec SupplyBoundInvariant observes eMintResponse, eTokenInfoResponse {
    var currentSupply: BigInt;
    var maxSupply: BigInt;

    start state Tracking {
        entry {
            currentSupply = (value = 1000000000, isNegative = false);
            maxSupply = (value = 10000000000, isNegative = false);
        }

        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                currentSupply = payload.receipt.newTotalSupply;
                assert currentSupply.value <= maxSupply.value,
                    "Total supply exceeded max supply!";
            }
        }

        on eTokenInfoResponse do (payload: (info: TokenInfo)) {
            currentSupply = payload.info.totalSupply;
            maxSupply = payload.info.maxSupply;
            assert currentSupply.value <= maxSupply.value,
                "Total supply exceeded max supply!";
        }
    }
}

// Assertion: Supply should always be non-negative
spec NonNegativeSupply observes eBurnResponse, eTokenInfoResponse {
    start state Checking {
        on eBurnResponse do (payload: (success: bool, receipt: BurnReceipt)) {
            if (payload.success) {
                assert !payload.receipt.newTotalSupply.isNegative,
                    "Supply became negative after burn!";
            }
        }

        on eTokenInfoResponse do (payload: (info: TokenInfo)) {
            assert !payload.info.totalSupply.isNegative,
                "Token supply is negative!";
        }
    }
}
