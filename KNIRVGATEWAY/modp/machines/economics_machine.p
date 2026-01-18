// economics_machine.p - P State Machine for Economics Engine
// Models token economics including fees, rewards, staking, and burns.
// Based on: KNIRVGATEWAY/internal/oracle/economics/engine.go

machine EconomicsMachine {
    // State variables
    var totalFeesCollected: BigInt;
    var totalRewardsDistributed: BigInt;
    var rewardPoolBalance: BigInt;
    var totalStaked: BigInt;
    var totalBurned: BigInt;

    // Fee configuration
    var burnPercentage: int;  // Percentage of fees to burn (0-100)

    // Staking state
    var stakers: map[Address, StakeInfo];
    var stakerList: seq[Address];

    // Treasury
    var treasuryBalance: BigInt;
    var treasuryAddress: Address;

    // Reference to token machine
    var tokenMachine: machine;

    // Start state
    start state Init {
        entry {
            totalFeesCollected = (value = 0, isNegative = false);
            totalRewardsDistributed = (value = 0, isNegative = false);
            rewardPoolBalance = (value = 0, isNegative = false);
            totalStaked = (value = 0, isNegative = false);
            totalBurned = (value = 0, isNegative = false);
            treasuryBalance = (value = 0, isNegative = false);
            treasuryAddress = default(Address);
            burnPercentage = 10;  // 10% of fees are burned
            stakers = default(map[Address, StakeInfo]);
            stakerList = default(seq[Address]);

            announce eComponentStarted, (componentName = "EconomicsMachine");
        }

        on eOracleStart do {
            goto Active;
        }
    }

    state Active {
        // Handle fee processing
        on eProcessFee do (payload: (from: Address, feeType: FeeType, amount: BigInt)) {
            // Validate amount
            if (payload.amount.isNegative || payload.amount.value <= 0) {
                announce eFeeProcessingFailed, (
                    from = payload.from,
                    error = InvalidAmount()
                );
                return;
            }

            // Calculate burn and reward portions
            var burnAmount: BigInt;
            var rewardAmount: BigInt;

            burnAmount = (value = (payload.amount.value * burnPercentage) / 100, isNegative = false);
            rewardAmount = (value = payload.amount.value - burnAmount.value, isNegative = false);

            // Update totals
            totalFeesCollected = AddBigInt(totalFeesCollected, payload.amount);
            totalBurned = AddBigInt(totalBurned, burnAmount);
            rewardPoolBalance = AddBigInt(rewardPoolBalance, rewardAmount);

            // Update treasury balance
            var oldTreasuryBalance: BigInt;
            oldTreasuryBalance = treasuryBalance;
            treasuryBalance = AddBigInt(treasuryBalance, rewardAmount);

            // Notify treasury deposit for monitor
            announce eTreasuryDeposit, (
                amount = rewardAmount,
                source = payload.feeType.feeName
            );
            announce eTreasuryBalanceChanged, (
                oldBalance = oldTreasuryBalance,
                newBalance = treasuryBalance
            );

            // Generate receipt
            var receipt: FeeReceipt;
            receipt = (
                txHash = format("fee_{0}", totalFeesCollected.value),
                from = payload.from,
                feeType = payload.feeType,
                amount = payload.amount,
                burnAmount = burnAmount,
                rewardAmount = rewardAmount,
                timestamp = 0
            );

            announce eFeeProcessed, (receipt = receipt);
        }

        // Handle staking
        on eStake do (payload: (staker: Address, amount: BigInt, validator: Address)) {
            if (payload.amount.isNegative || payload.amount.value <= 0) {
                announce eStakingFailed, (
                    staker = payload.staker,
                    error = InvalidAmount()
                );
                return;
            }

            // Update or create stake info
            var stakeInfo: StakeInfo;
            if (payload.staker in stakers) {
                stakeInfo = stakers[payload.staker];
                stakeInfo.stakedAmount = AddBigInt(stakeInfo.stakedAmount, payload.amount);
            } else {
                stakeInfo = (
                    staker = payload.staker,
                    stakedAmount = payload.amount,
                    rewards = (value = 0, isNegative = false),
                    stakedAt = 0
                );
                stakerList += (sizeof(stakerList), payload.staker);
            }

            stakers[payload.staker] = stakeInfo;
            totalStaked = AddBigInt(totalStaked, payload.amount);

            announce eStaked, (
                staker = payload.staker,
                amount = payload.amount,
                totalStaked = totalStaked
            );
        }

        // Handle unstaking
        on eUnstake do (payload: (staker: Address, amount: BigInt)) {
            if (!(payload.staker in stakers)) {
                announce eStakingFailed, (
                    staker = payload.staker,
                    error = (code = 301, message = "Not a staker")
                );
                return;
            }

            var stakeInfo: StakeInfo;
            stakeInfo = stakers[payload.staker];

            if (stakeInfo.stakedAmount.value < payload.amount.value) {
                announce eStakingFailed, (
                    staker = payload.staker,
                    error = InsufficientBalance()
                );
                return;
            }

            // Update stake
            stakeInfo.stakedAmount = SubtractBigInt(stakeInfo.stakedAmount, payload.amount);
            stakers[payload.staker] = stakeInfo;
            totalStaked = SubtractBigInt(totalStaked, payload.amount);

            announce eUnstaked, (
                staker = payload.staker,
                amount = payload.amount,
                remainingStaked = stakeInfo.stakedAmount
            );
        }

        // Handle reward distribution
        on eDistributeRewards do (payload: (stakers: seq[StakeInfo], totalAmount: BigInt)) {
            if (rewardPoolBalance.value < payload.totalAmount.value) {
                // Not enough in reward pool
                return;
            }

            if (sizeof(payload.stakers) == 0) {
                return;
            }

            // Calculate per-staker reward (proportional to stake)
            var i: int;
            var recipientCount: int;
            i = 0;
            recipientCount = 0;

            while (i < sizeof(payload.stakers)) {
                var stakerInfo: StakeInfo;
                stakerInfo = payload.stakers[i];

                if (stakerInfo.staker in stakers) {
                    // Calculate proportional reward (simplified)
                    var stakerReward: BigInt;
                    stakerReward = (
                        value = (payload.totalAmount.value * stakerInfo.stakedAmount.value) / totalStaked.value,
                        isNegative = false
                    );

                    // Update staker rewards
                    var currentInfo: StakeInfo;
                    currentInfo = stakers[stakerInfo.staker];
                    currentInfo.rewards = AddBigInt(currentInfo.rewards, stakerReward);
                    stakers[stakerInfo.staker] = currentInfo;

                    recipientCount = recipientCount + 1;
                }
                i = i + 1;
            }

            // Update totals
            rewardPoolBalance = SubtractBigInt(rewardPoolBalance, payload.totalAmount);
            totalRewardsDistributed = AddBigInt(totalRewardsDistributed, payload.totalAmount);

            announce eRewardsDistributed, (
                totalAmount = payload.totalAmount,
                recipientCount = recipientCount
            );
        }

        // Handle reward claiming
        on eClaimRewards do (payload: (staker: Address)) {
            if (!(payload.staker in stakers)) {
                return;
            }

            var stakeInfo: StakeInfo;
            stakeInfo = stakers[payload.staker];

            if (stakeInfo.rewards.value == 0) {
                return;
            }

            var claimedAmount: BigInt;
            claimedAmount = stakeInfo.rewards;

            // Reset rewards
            stakeInfo.rewards = (value = 0, isNegative = false);
            stakers[payload.staker] = stakeInfo;

            announce eRewardsClaimed, (
                staker = payload.staker,
                amount = claimedAmount
            );
        }

        // Handle economic metrics query
        on eGetEconomicMetrics do {
            var metrics: EconomicMetrics;
            metrics = (
                totalSupply = (value = 0, isNegative = false),  // Would be from TokenMachine
                maxSupply = (value = 0, isNegative = false),
                circulatingSupply = (value = 0, isNegative = false),
                totalStaked = totalStaked,
                totalBurned = totalBurned,
                totalFeesCollected = totalFeesCollected,
                totalRewardsDistributed = totalRewardsDistributed,
                rewardPoolBalance = rewardPoolBalance,
                stakingAPY = CalculateStakingAPY(totalStaked, totalRewardsDistributed),
                burnRate = CalculateBurnRate(totalBurned, totalFeesCollected)
            );

            announce eEconomicMetricsResponse, (metrics = metrics);
        }

        // Handle stake info query
        on eGetStakeInfo do (payload: (staker: Address)) {
            if (!(payload.staker in stakers)) {
                var emptyInfo: StakeInfo;
                emptyInfo = (
                    staker = payload.staker,
                    stakedAmount = (value = 0, isNegative = false),
                    rewards = (value = 0, isNegative = false),
                    stakedAt = 0
                );
                announce eStakeInfoResponse, (stakeInfo = emptyInfo);
                return;
            }

            announce eStakeInfoResponse, (stakeInfo = stakers[payload.staker]);
        }

        // Handle fee payment verification (for "Second Me" registration)
        on eFeePayment do (payload: (user: Address, amount: BigInt)) {
            // Verify the fee payment was successful
            // In a real system, this would check the token transfer
            var verified: bool;
            verified = payload.amount.value >= 100000000;  // Minimum fee

            announce eFeePaymentVerified, (
                user = payload.user,
                amount = payload.amount,
                verified = verified
            );
        }

        // Handle treasury withdrawal (only through governance)
        on eTreasuryWithdrawal do (payload: (amount: BigInt, recipient: Address, approvedByGovernance: bool)) {
            if (!payload.approvedByGovernance) {
                // Report violation to monitor
                announce eTreasuryInvariantViolation, (
                    expectedBalance = treasuryBalance,
                    actualBalance = SubtractBigInt(treasuryBalance, payload.amount),
                    operation = "Unauthorized treasury withdrawal"
                );
                return;
            }

            if (treasuryBalance.value < payload.amount.value) {
                return;
            }

            var oldBalance: BigInt;
            oldBalance = treasuryBalance;
            treasuryBalance = SubtractBigInt(treasuryBalance, payload.amount);

            announce eTreasuryBalanceChanged, (
                oldBalance = oldBalance,
                newBalance = treasuryBalance
            );
        }

        // Handle shutdown
        on eOracleStop do {
            goto Stopped;
        }
    }

    state Stopped {
        entry {
            announce eComponentStopped, (componentName = "EconomicsMachine");
        }

        on eOracleStart do {
            goto Active;
        }

        // Still respond to queries
        on eGetEconomicMetrics do {
            var metrics: EconomicMetrics;
            metrics = (
                totalSupply = (value = 0, isNegative = false),
                maxSupply = (value = 0, isNegative = false),
                circulatingSupply = (value = 0, isNegative = false),
                totalStaked = totalStaked,
                totalBurned = totalBurned,
                totalFeesCollected = totalFeesCollected,
                totalRewardsDistributed = totalRewardsDistributed,
                rewardPoolBalance = rewardPoolBalance,
                stakingAPY = 0,
                burnRate = 0
            );
            announce eEconomicMetricsResponse, (metrics = metrics);
        }

        on eGetStakeInfo do (payload: (staker: Address)) {
            if (payload.staker in stakers) {
                announce eStakeInfoResponse, (stakeInfo = stakers[payload.staker]);
            }
        }

        ignore eProcessFee, eStake, eUnstake, eDistributeRewards, eClaimRewards;
        ignore eFeePayment, eTreasuryWithdrawal;
    }
}

// =====================================================================
// Helper Functions
// =====================================================================

// CalculateStakingAPY calculates annual percentage yield for staking
fun CalculateStakingAPY(totalStaked: BigInt, totalRewards: BigInt): int {
    if (totalStaked.value == 0) {
        return 0;
    }
    // Returns APY as percentage * 100 (e.g., 500 = 5%)
    return (totalRewards.value * 10000) / totalStaked.value;
}

// CalculateBurnRate calculates the burn rate as percentage
fun CalculateBurnRate(totalBurned: BigInt, totalFees: BigInt): int {
    if (totalFees.value == 0) {
        return 0;
    }
    // Returns rate as percentage * 100
    return (totalBurned.value * 10000) / totalFees.value;
}
