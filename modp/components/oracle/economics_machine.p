// economics_machine.p - Economics Engine State Machine for KNIRVORACLE
// Models token economics including fees, staking, and rewards
// Based on: KNIRVORACLE economics module implementation

machine EconomicsMachine {
    // State variables
    var treasuryBalance: BigInt;
    var totalFeesCollected: BigInt;
    var stakes: map[Address, seq[StakeInfo]];
    var totalStaked: BigInt;
    var currentEpoch: int;

    // Fee configuration
    var baseFeeRate: int;  // basis points (100 = 1%)
    var feeRecipients: map[string, int];  // fee type -> treasury percentage

    // Staking configuration
    var minStakeAmount: BigInt;
    var unbondingPeriod: int;  // blocks
    var rewardRate: int;  // basis points per epoch

    // References
    var tokenMachine: machine;
    var governanceMachine: machine;

    // Temp variables
    var tempTreasuryShare: int;
    var tempToTreasury: BigInt;
    var tempNewStake: StakeInfo;
    var tempStakerStakes: seq[StakeInfo];
    var tempToUnstake: BigInt;
    var tempDistributions: seq[RewardDistribution];
    var tempTotalStakerReward: BigInt;
    var tempStake: StakeInfo;
    var tempReward: BigInt;
    var tempDist: RewardDistribution;
    var tempIdx: int;
    var tempDeposit: TreasuryDepositData;
    var tempFeeProcessed: FeeProcessedData;
    var tempStakeData: StakeData;
    var tempUnstakeData: UnstakeData;
    var tempTimestamp: Timestamp;
    var staker: Address;

    start state Init {
        entry {
            treasuryBalance.value = 0;
            treasuryBalance.isNegative = false;
            totalFeesCollected.value = 0;
            totalFeesCollected.isNegative = false;
            stakes = default(map[Address, seq[StakeInfo]]);
            totalStaked.value = 0;
            totalStaked.isNegative = false;
            currentEpoch = 0;

            // Fee configuration
            baseFeeRate = 100;  // 1%
            feeRecipients = default(map[string, int]);
            feeRecipients["transfer"] = 8000;  // 80% to treasury
            feeRecipients["governance"] = 10000;  // 100% to treasury
            feeRecipients["registration"] = 9000;  // 90% to treasury

            // Staking configuration
            minStakeAmount.value = 1000000;  // 1 NRN
            minStakeAmount.isNegative = false;
            unbondingPeriod = 14400;  // ~1 day at 6s blocks
            rewardRate = 50;  // 0.5% per epoch
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // Fee processing
        on eProcessFee do (payload: (from: Address, feeType: FeeType, amount: BigInt)) {
            if (payload.amount.value <= 0) {
                send this, eFeeProcessingFailed, "Invalid fee amount";
                return;
            }

            // Calculate treasury share
            if (payload.feeType.feeName in feeRecipients) {
                tempTreasuryShare = feeRecipients[payload.feeType.feeName];
            } else {
                tempTreasuryShare = 10000;  // Default 100% to treasury
            }

            tempToTreasury.value = (payload.amount.value * tempTreasuryShare) / 10000;
            tempToTreasury.isNegative = false;

            // Update balances
            treasuryBalance.value = treasuryBalance.value + tempToTreasury.value;
            totalFeesCollected.value = totalFeesCollected.value + payload.amount.value;

            // Announce for treasury monitor
            tempDeposit.amount = tempToTreasury;
            tempDeposit.source = payload.feeType.feeName;
            announce eTreasuryDeposit, tempDeposit;

            tempFeeProcessed.user = payload.from;
            tempFeeProcessed.feePaid = payload.amount;
            tempFeeProcessed.treasuryAmount = treasuryBalance;
            announce eFeeProcessed, tempFeeProcessed;
            send this, eFeeProcessed, tempFeeProcessed;
        }

        // Fee payment for registrations (used by governance)
        on eFeePayment do (payload: (user: Address, amount: BigInt)) {
            // Process the fee
            treasuryBalance.value = treasuryBalance.value + payload.amount.value;
            totalFeesCollected.value = totalFeesCollected.value + payload.amount.value;

            tempDeposit.amount = payload.amount;
            tempDeposit.source = "registration";
            announce eTreasuryDeposit, tempDeposit;

            // Notify governance that payment is verified
            send this, eFeePaymentVerified, (user = payload.user, amount = payload.amount, verified = true);
        }

        // Staking
        on eStake do (payload: (staker: Address, amount: BigInt, validator: Address)) {
            // Validate stake amount
            if (payload.amount.value < minStakeAmount.value) {
                send this, eStakeFailed, "Below minimum stake amount";
                return;
            }

            // Create stake info
            tempTimestamp.milliseconds = 0;
            tempNewStake.staker = payload.staker;
            tempNewStake.validator = payload.validator;
            tempNewStake.amount = payload.amount;
            tempNewStake.startTime = tempTimestamp;
            tempNewStake.lockPeriod = unbondingPeriod;
            tempNewStake.rewards.value = 0;
            tempNewStake.rewards.isNegative = false;

            // Add to stakes
            if (payload.staker in stakes) {
                tempStakerStakes = stakes[payload.staker];
            } else {
                tempStakerStakes = default(seq[StakeInfo]);
            }

            tempStakerStakes += (sizeof(tempStakerStakes), tempNewStake);
            stakes[payload.staker] = tempStakerStakes;

            // Update total staked
            totalStaked.value = totalStaked.value + payload.amount.value;

            send this, eStaked, tempNewStake;
        }

        // Unstaking
        on eUnstake do (payload: (staker: Address, amount: BigInt)) {
            if (!(payload.staker in stakes)) {
                send this, eUnstakeFailed, "No stakes found";
                return;
            }

            tempStakerStakes = stakes[payload.staker];

            if (sizeof(tempStakerStakes) == 0) {
                send this, eUnstakeFailed, "No stakes found";
                return;
            }

            // For simplicity, unstake from first stake
            tempToUnstake = payload.amount;

            // Update total staked
            totalStaked.value = totalStaked.value - tempToUnstake.value;

            tempUnstakeData.staker = payload.staker;
            tempUnstakeData.amount = tempToUnstake;
            send this, eUnstaked, tempUnstakeData;
        }

        // Reward distribution
        on eDistributeRewards do (payload: (epoch: int)) {
            currentEpoch = payload.epoch;

            tempDistributions = default(seq[RewardDistribution]);

            // Calculate rewards for each staker
            foreach (staker in keys(stakes)) {
                tempStakerStakes = stakes[staker];

                tempTotalStakerReward.value = 0;
                tempTotalStakerReward.isNegative = false;

                tempIdx = 0;
                while (tempIdx < sizeof(tempStakerStakes)) {
                    tempStake = tempStakerStakes[tempIdx];

                    // Calculate reward for this stake
                    tempReward.value = (tempStake.amount.value * rewardRate) / 10000;
                    tempReward.isNegative = false;

                    tempTotalStakerReward.value = tempTotalStakerReward.value + tempReward.value;
                    tempIdx = tempIdx + 1;
                }

                if (tempTotalStakerReward.value > 0) {
                    tempTimestamp.milliseconds = 0;
                    tempDist.recipient = staker;
                    tempDist.amount = tempTotalStakerReward;
                    tempDist.rewardType = "staking";
                    tempDist.epoch = payload.epoch;
                    tempDist.timestamp = tempTimestamp;
                    tempDistributions += (sizeof(tempDistributions), tempDist);
                }
            }

            send this, eRewardsDistributed, tempDistributions;
        }

        // Treasury queries
        on eTreasuryBalanceQuery do {
            send this, eTreasuryBalanceResponse, treasuryBalance;
        }

        // Treasury withdrawal (governance-approved only)
        on eTreasuryWithdrawal do (payload: (amount: BigInt, destination: Address, proposalID: string)) {
            if (payload.amount.value > treasuryBalance.value) {
                // Cannot withdraw more than treasury balance
                return;
            }

            treasuryBalance.value = treasuryBalance.value - payload.amount.value;

            // Announce for monitor
            announce eTreasuryWithdrawal, payload;
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

    // Helper functions
    fun GetTreasuryBalance(): BigInt {
        return treasuryBalance;
    }

    fun GetTotalStaked(): BigInt {
        return totalStaked;
    }

    fun GetTotalFeesCollected(): BigInt {
        return totalFeesCollected;
    }
}
