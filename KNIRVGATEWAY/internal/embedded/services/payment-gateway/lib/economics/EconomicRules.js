/**
 * Economic Rules Configuration
 * Defines all economic parameters for the KNIRV network
 */

class EconomicRules {
  constructor(customRules = {}) {
    // Skill and LLM costs
    this.skillInvocationCost = customRules.skillInvocationCost || BigInt(100000); // 0.1 NRN
    this.llmRegistrationFee = customRules.llmRegistrationFee || BigInt(1000000); // 1 NRN
    this.validationReward = customRules.validationReward || BigInt(50000); // 0.05 NRN

    // Burn rates for different operations
    this.burnRates = customRules.burnRates || {
      skill_invocation: BigInt(100000),
      llm_registration: BigInt(500000),
      validation: BigInt(25000)
    };

    // Minting rules
    this.mintingRules = customRules.mintingRules || {
      maxSupply: BigInt(1000000000000000), // 1B NRN
      inflationRate: 0.05, // 5% annual
      validatorRewards: BigInt(10000000), // 10 NRN per block
      developerRewards: BigInt(5000000), // 5 NRN per block
      communityRewards: BigInt(2000000) // 2 NRN per block
    };

    // Staking requirements
    this.stakingRequirements = customRules.stakingRequirements || {
      minValidatorStake: BigInt(100000000000), // 100K NRN
      minDeveloperStake: BigInt(10000000000), // 10K NRN
      slashingPenalty: 0.05, // 5%
      unbondingPeriod: 21 * 24 * 60 * 60 * 1000 // 21 days in milliseconds
    };

    // Governance thresholds
    this.governanceThresholds = customRules.governanceThresholds || {
      proposalDeposit: BigInt(1000000000), // 1K NRN
      votingThreshold: 0.5, // 50%
      quorumThreshold: 0.33, // 33%
      votingPeriod: 7 * 24 * 60 * 60 * 1000 // 7 days in milliseconds
    };
  }

  // Convert BigInt to string for JSON serialization
  toJSON() {
    return {
      skillInvocationCost: this.skillInvocationCost.toString(),
      llmRegistrationFee: this.llmRegistrationFee.toString(),
      validationReward: this.validationReward.toString(),
      burnRates: Object.fromEntries(
        Object.entries(this.burnRates).map(([k, v]) => [k, v.toString()])
      ),
      mintingRules: {
        ...this.mintingRules,
        maxSupply: this.mintingRules.maxSupply.toString(),
        validatorRewards: this.mintingRules.validatorRewards.toString(),
        developerRewards: this.mintingRules.developerRewards.toString(),
        communityRewards: this.mintingRules.communityRewards.toString()
      },
      stakingRequirements: {
        ...this.stakingRequirements,
        minValidatorStake: this.stakingRequirements.minValidatorStake.toString(),
        minDeveloperStake: this.stakingRequirements.minDeveloperStake.toString()
      },
      governanceThresholds: {
        ...this.governanceThresholds,
        proposalDeposit: this.governanceThresholds.proposalDeposit.toString()
      }
    };
  }

  // Create from JSON
  static fromJSON(json) {
    return new EconomicRules({
      skillInvocationCost: BigInt(json.skillInvocationCost),
      llmRegistrationFee: BigInt(json.llmRegistrationFee),
      validationReward: BigInt(json.validationReward),
      burnRates: Object.fromEntries(
        Object.entries(json.burnRates).map(([k, v]) => [k, BigInt(v)])
      ),
      mintingRules: {
        ...json.mintingRules,
        maxSupply: BigInt(json.mintingRules.maxSupply),
        validatorRewards: BigInt(json.mintingRules.validatorRewards),
        developerRewards: BigInt(json.mintingRules.developerRewards),
        communityRewards: BigInt(json.mintingRules.communityRewards)
      },
      stakingRequirements: {
        ...json.stakingRequirements,
        minValidatorStake: BigInt(json.stakingRequirements.minValidatorStake),
        minDeveloperStake: BigInt(json.stakingRequirements.minDeveloperStake)
      },
      governanceThresholds: {
        ...json.governanceThresholds,
        proposalDeposit: BigInt(json.governanceThresholds.proposalDeposit)
      }
    });
  }
}

module.exports = EconomicRules;
