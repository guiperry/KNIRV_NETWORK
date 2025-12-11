/**
 * Token Economics
 * Main orchestrator for KNIRV token economics system
 */

const EconomicRules = require('./EconomicRules');
const TransactionPool = require('./TransactionPool');
const RewardCalculator = require('./RewardCalculator');
const BurnTracker = require('./BurnTracker');
const EconomicMetrics = require('./EconomicMetrics');
const crypto = require('crypto');

class TokenEconomics {
  constructor(config = {}) {
    this.nrnContract = config.nrnContract || '';
    this.xionRPC = config.xionRPC || '';

    // Initialize components
    this.economicRules = new EconomicRules(config.economicRules);
    this.transactionPool = new TransactionPool(config.transactionPool);
    this.rewardCalculator = new RewardCalculator();
    this.burnTracker = new BurnTracker();
    this.metrics = new EconomicMetrics();

    // Background process timers
    this.processors = {
      transaction: null,
      metrics: null,
      reward: null,
      burn: null
    };

    this.isRunning = false;
  }

  /**
   * Start the economics system
   */
  async start() {
    if (this.isRunning) {
      console.log('Token Economics system already running');
      return;
    }

    console.log('Starting Token Economics system...');

    // Start background processors
    this.startTransactionProcessor();
    this.startMetricsUpdater();
    this.startRewardDistributor();
    this.startBurnProcessor();

    // Start transaction pool cleanup
    this.transactionPool.startCleanup();

    this.isRunning = true;
    console.log('Token Economics system started');
  }

  /**
   * Stop the economics system
   */
  async stop() {
    if (!this.isRunning) {
      return;
    }

    console.log('Stopping Token Economics system...');

    // Stop all processors
    Object.values(this.processors).forEach(timer => {
      if (timer) clearInterval(timer);
    });

    // Stop transaction pool cleanup
    this.transactionPool.stopCleanup();

    this.isRunning = false;
    console.log('Token Economics system stopped');
  }

  /**
   * Process skill invocation
   */
  async processSkillInvocation(userId, skillId, amount) {
    const amountBigInt = BigInt(amount);
    const requiredAmount = this.economicRules.skillInvocationCost;

    if (amountBigInt < requiredAmount) {
      throw new Error(
        `Insufficient amount: required ${requiredAmount}, provided ${amountBigInt}`
      );
    }

    // Create transaction
    const tx = {
      id: `skill_${skillId}_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`,
      type: 'skill_invocation',
      from: userId,
      to: 'skill_registry',
      amount: amountBigInt,
      purpose: 'skill_invocation',
      metadata: { skillId },
      status: 'pending',
      timestamp: new Date()
    };

    // Add to transaction pool
    this.transactionPool.addTransaction(tx);

    // Record burn event
    this.burnTracker.addBurnEvent({
      txId: tx.id,
      user: userId,
      amount: amountBigInt,
      purpose: 'skill_invocation',
      skillId,
      timestamp: new Date()
    });

    // Update metrics
    this.metrics.updateServiceMetrics('knirvchain', amountBigInt, 'spent');

    return tx;
  }

  /**
   * Process LLM registration
   */
  async processLLMRegistration(userId, llmId, registrationFee) {
    const feeBigInt = BigInt(registrationFee);
    const requiredFee = this.economicRules.llmRegistrationFee;

    if (feeBigInt < requiredFee) {
      throw new Error(
        `Insufficient registration fee: required ${requiredFee}, provided ${feeBigInt}`
      );
    }

    // Create transaction
    const tx = {
      id: `llm_reg_${llmId}_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`,
      type: 'llm_registration',
      from: userId,
      to: 'llm_registry',
      amount: feeBigInt,
      purpose: 'llm_registration',
      metadata: { llmId },
      status: 'pending',
      timestamp: new Date()
    };

    this.transactionPool.addTransaction(tx);

    // Update metrics
    this.metrics.updateServiceMetrics('knirvchain', feeBigInt, 'earned');

    return tx;
  }

  /**
   * Process validation reward
   */
  async processValidationReward(validatorId, targetId, validationResult) {
    if (!validationResult) {
      throw new Error('Validation failed, no reward');
    }

    // Calculate reward based on performance
    const baseReward = this.economicRules.validationReward;
    const finalReward = this.rewardCalculator.calculateReward(
      validatorId,
      'validation',
      baseReward
    );

    // Create transaction
    const tx = {
      id: `validation_${targetId}_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`,
      type: 'validation_reward',
      from: 'reward_pool',
      to: validatorId,
      amount: finalReward,
      purpose: 'validation_reward',
      metadata: { targetId, validationResult },
      status: 'pending',
      timestamp: new Date()
    };

    this.transactionPool.addTransaction(tx);

    // Update metrics
    this.metrics.updateServiceMetrics('knirvnexus', finalReward, 'earned');

    return tx;
  }

  /**
   * Calculate network fees
   */
  calculateNetworkFees(gasUsed, priority = 'medium') {
    const baseGasPrice = BigInt(1000);
    let multiplier = 1000; // Base multiplier (1.0x in scaled form)

    switch (priority) {
      case 'high':
        multiplier = 2000; // 2.0x
        break;
      case 'medium':
        multiplier = 1500; // 1.5x
        break;
      case 'low':
        multiplier = 1000; // 1.0x
        break;
    }

    const gasPrice = (baseGasPrice * BigInt(multiplier)) / BigInt(1000);
    const totalFee = gasPrice * BigInt(gasUsed);

    return totalFee;
  }

  /**
   * Get economic metrics
   */
  getEconomicMetrics() {
    return this.metrics.getAllMetrics();
  }

  /**
   * Get economic rules
   */
  getEconomicRules() {
    return this.economicRules.toJSON();
  }

  /**
   * Update economic rules (admin only)
   */
  updateEconomicRules(newRules) {
    this.economicRules = EconomicRules.fromJSON(newRules);
    console.log('Economic rules updated');
  }

  /**
   * Background: Process pending transactions
   */
  startTransactionProcessor() {
    this.processors.transaction = setInterval(() => {
      const pendingTxs = this.transactionPool.getPendingTransactions();

      for (const tx of pendingTxs) {
        try {
          this.processTransaction(tx);
          this.transactionPool.confirmTransaction(tx.id);
        } catch (error) {
          console.error(`Failed to process transaction ${tx.id}:`, error.message);
          this.transactionPool.failTransaction(tx.id, error.message);
        }
      }
    }, 5000); // Every 5 seconds
  }

  /**
   * Process a single transaction
   */
  processTransaction(tx) {
    switch (tx.type) {
      case 'skill_invocation':
        return this.processSkillInvocationTx(tx);
      case 'llm_registration':
        return this.processLLMRegistrationTx(tx);
      case 'validation_reward':
        return this.processValidationRewardTx(tx);
      default:
        throw new Error(`Unknown transaction type: ${tx.type}`);
    }
  }

  /**
   * Process skill invocation transaction
   */
  processSkillInvocationTx(tx) {
    // Burn tokens
    this.burnTracker.processBurn(tx.id, tx.amount);

    // Update metrics
    this.metrics.updateBurned(tx.amount);
    this.metrics.updateTransactionVolume(tx.amount);

    console.log(`Burned ${tx.amount} NRN for skill invocation ${tx.metadata.skillId}`);
  }

  /**
   * Process LLM registration transaction
   */
  processLLMRegistrationTx(tx) {
    // Transfer registration fee to treasury
    console.log(`Processed LLM registration fee ${tx.amount} NRN for ${tx.metadata.llmId}`);
  }

  /**
   * Process validation reward transaction
   */
  processValidationRewardTx(tx) {
    // Mint reward tokens
    this.metrics.updateTotalSupply(tx.amount, 'add');
    this.metrics.updateTransactionVolume(tx.amount);

    console.log(`Minted ${tx.amount} NRN validation reward for ${tx.to}`);
  }

  /**
   * Background: Update metrics
   */
  startMetricsUpdater() {
    this.processors.metrics = setInterval(() => {
      this.metrics.updateCalculatedMetrics();
    }, 60000); // Every 1 minute
  }

  /**
   * Background: Distribute rewards
   */
  startRewardDistributor() {
    this.processors.reward = setInterval(() => {
      // Distribute periodic rewards
      console.log('Distributing periodic rewards...');
      // This would distribute validator, developer, and community rewards
    }, 3600000); // Every 1 hour
  }

  /**
   * Background: Process burn events
   */
  startBurnProcessor() {
    this.processors.burn = setInterval(() => {
      const unvalidatedBurns = this.burnTracker.getBurnHistory({ validated: false });

      for (const event of unvalidatedBurns) {
        // Validate burn event
        event.validated = true;
        console.log(`Validated burn event: ${event.user} burned ${event.amount} NRN`);
      }
    }, 10000); // Every 10 seconds
  }

  /**
   * Export system state
   */
  toJSON() {
    return {
      nrnContract: this.nrnContract,
      xionRPC: this.xionRPC,
      economicRules: this.economicRules.toJSON(),
      metrics: this.metrics.toJSON(),
      burnTracker: this.burnTracker.toJSON(),
      rewardCalculator: this.rewardCalculator.toJSON(),
      isRunning: this.isRunning,
      timestamp: new Date().toISOString()
    };
  }
}

module.exports = TokenEconomics;
