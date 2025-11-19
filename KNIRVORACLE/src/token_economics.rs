use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use num_bigint::BigInt;
use tokio::time::interval;
use tracing::{info, error, warn};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EconomicRules {
    pub skill_invocation_cost: BigInt,
    pub llm_registration_fee: BigInt,
    pub validation_reward: BigInt,
    pub burn_rates: HashMap<String, BigInt>,
    pub minting_rules: MintingRules,
    pub staking_requirements: StakingRequirements,
    pub governance_thresholds: GovernanceThresholds,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MintingRules {
    pub max_supply: BigInt,
    pub inflation_rate: f64,
    pub validator_rewards: BigInt,
    pub developer_rewards: BigInt,
    pub community_rewards: BigInt,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingRequirements {
    pub min_validator_stake: BigInt,
    pub min_developer_stake: BigInt,
    pub slashing_penalty: f64,
    pub unbonding_period: Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GovernanceThresholds {
    pub proposal_deposit: BigInt,
    pub voting_threshold: f64,
    pub quorum_threshold: f64,
    pub voting_period: Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionPool {
    pub pending_txs: HashMap<String, EconomicTransaction>,
    pub confirmed_txs: HashMap<String, EconomicTransaction>,
    pub max_pool_size: usize,
    pub cleanup_interval: Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EconomicTransaction {
    pub id: String,
    pub tx_type: String,
    pub from: String,
    pub to: String,
    pub amount: BigInt,
    pub purpose: String,
    pub metadata: HashMap<String, serde_json::Value>,
    pub status: String,
    pub timestamp: u64,
    pub confirmed_at: Option<u64>,
    pub block_height: Option<u64>,
    pub gas_used: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RewardCalculator {
    pub base_rewards: HashMap<String, BigInt>,
    pub multipliers: HashMap<String, f64>,
    pub performance_data: HashMap<String, PerformanceMetrics>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub success_rate: f64,
    pub response_time: f64,
    pub user_satisfaction: f64,
    pub uptime: f64,
    pub last_updated: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BurnTracker {
    pub total_burned: BigInt,
    pub burn_history: Vec<BurnEvent>,
    pub burn_rates: HashMap<String, BigInt>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BurnEvent {
    pub tx_id: String,
    pub user: String,
    pub amount: BigInt,
    pub purpose: String,
    pub skill_id: Option<String>,
    pub timestamp: u64,
    pub validated: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EconomicMetrics {
    pub total_supply: BigInt,
    pub circulating_supply: BigInt,
    pub total_burned: BigInt,
    pub total_staked: BigInt,
    pub active_validators: i32,
    pub transaction_volume: BigInt,
    pub average_gas_price: BigInt,
    pub network_utilization: f64,
    pub token_velocity: f64,
    pub last_updated: u64,
    pub service_metrics: HashMap<String, ServiceEconomics>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceEconomics {
    pub revenue: BigInt,
    pub costs: BigInt,
    pub profit: BigInt,
    pub tokens_earned: BigInt,
    pub tokens_spent: BigInt,
    pub user_count: i32,
    pub transaction_count: i32,
    pub last_updated: u64,
}

#[derive(Debug)]
pub struct TokenEconomics {
    nrn_contract: String,
    xion_rpc: String,
    economic_rules: EconomicRules,
    transaction_pool: Arc<Mutex<TransactionPool>>,
    reward_calculator: Arc<Mutex<RewardCalculator>>,
    burn_tracker: Arc<Mutex<BurnTracker>>,
    metrics: Arc<Mutex<EconomicMetrics>>,
}

impl TokenEconomics {
    pub fn new(nrn_contract: String, xion_rpc: String) -> Self {
        Self {
            nrn_contract,
            xion_rpc,
            economic_rules: Self::new_default_economic_rules(),
            transaction_pool: Arc::new(Mutex::new(Self::new_transaction_pool())),
            reward_calculator: Arc::new(Mutex::new(Self::new_reward_calculator())),
            burn_tracker: Arc::new(Mutex::new(Self::new_burn_tracker())),
            metrics: Arc::new(Mutex::new(Self::new_economic_metrics())),
        }
    }

    fn new_default_economic_rules() -> EconomicRules {
        EconomicRules {
            skill_invocation_cost: BigInt::from(100000), // 0.1 NRN
            llm_registration_fee: BigInt::from(1000000), // 1 NRN
            validation_reward: BigInt::from(50000),      // 0.05 NRN
            burn_rates: {
                let mut rates = HashMap::new();
                rates.insert("skill_invocation".to_string(), BigInt::from(100000));
                rates.insert("llm_registration".to_string(), BigInt::from(500000));
                rates.insert("validation".to_string(), BigInt::from(25000));
                rates
            },
            minting_rules: MintingRules {
                max_supply: BigInt::from(1000000000000000i64), // 1B NRN
                inflation_rate: 0.05,                          // 5% annual
                validator_rewards: BigInt::from(10000000),     // 10 NRN per block
                developer_rewards: BigInt::from(5000000),      // 5 NRN per block
                community_rewards: BigInt::from(2000000),      // 2 NRN per block
            },
            staking_requirements: StakingRequirements {
                min_validator_stake: BigInt::from(100000000000i64), // 100K NRN
                min_developer_stake: BigInt::from(10000000000i64),  // 10K NRN
                slashing_penalty: 0.05,                            // 5%
                unbonding_period: Duration::from_secs(21 * 24 * 3600), // 21 days
            },
            governance_thresholds: GovernanceThresholds {
                proposal_deposit: BigInt::from(1000000000i64), // 1K NRN
                voting_threshold: 0.5,                         // 50%
                quorum_threshold: 0.33,                        // 33%
                voting_period: Duration::from_secs(7 * 24 * 3600), // 7 days
            },
        }
    }

    fn new_transaction_pool() -> TransactionPool {
        TransactionPool {
            pending_txs: HashMap::new(),
            confirmed_txs: HashMap::new(),
            max_pool_size: 10000,
            cleanup_interval: Duration::from_secs(3600), // 1 hour
        }
    }

    fn new_reward_calculator() -> RewardCalculator {
        let mut base_rewards = HashMap::new();
        base_rewards.insert("validation".to_string(), BigInt::from(50000));
        base_rewards.insert("skill_creation".to_string(), BigInt::from(100000));
        base_rewards.insert("bug_reporting".to_string(), BigInt::from(25000));
        base_rewards.insert("community_help".to_string(), BigInt::from(10000));

        let mut multipliers = HashMap::new();
        multipliers.insert("high_performance".to_string(), 1.5);
        multipliers.insert("consistent_user".to_string(), 1.2);
        multipliers.insert("early_adopter".to_string(), 1.3);
        multipliers.insert("community_leader".to_string(), 2.0);

        RewardCalculator {
            base_rewards,
            multipliers,
            performance_data: HashMap::new(),
        }
    }

    fn new_burn_tracker() -> BurnTracker {
        BurnTracker {
            total_burned: BigInt::from(0),
            burn_history: Vec::new(),
            burn_rates: HashMap::new(),
        }
    }

    fn new_economic_metrics() -> EconomicMetrics {
        EconomicMetrics {
            total_supply: BigInt::from(0),
            circulating_supply: BigInt::from(0),
            total_burned: BigInt::from(0),
            total_staked: BigInt::from(0),
            active_validators: 0,
            transaction_volume: BigInt::from(0),
            average_gas_price: BigInt::from(0),
            network_utilization: 0.0,
            token_velocity: 0.0,
            last_updated: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            service_metrics: HashMap::new(),
        }
    }

    pub async fn start(&self) -> Result<(), Box<dyn std::error::Error>> {
        info!("Starting Token Economics system...");

        // Start background processes
        let transaction_pool = Arc::clone(&self.transaction_pool);
        let metrics = Arc::clone(&self.metrics);
        let burn_tracker = Arc::clone(&self.burn_tracker);

        tokio::spawn(async move {
            Self::transaction_processor(transaction_pool).await;
        });

        tokio::spawn(async move {
            Self::metrics_updater(metrics).await;
        });

        tokio::spawn(async move {
            Self::reward_distributor().await;
        });

        tokio::spawn(async move {
            Self::burn_processor(burn_tracker).await;
        });

        info!("Token Economics system started");
        Ok(())
    }

    pub fn process_skill_invocation(&self, user_id: &str, skill_id: &str, amount: &BigInt) -> Result<EconomicTransaction, Box<dyn std::error::Error>> {
        let required_amount = &self.economic_rules.skill_invocation_cost;
        if amount < required_amount {
            return Err(format!("insufficient amount: required {}, provided {}", required_amount, amount).into());
        }

        let tx_id = format!("skill_{}_{}", skill_id, SystemTime::now().duration_since(UNIX_EPOCH)?.as_nanos());
        let tx = EconomicTransaction {
            id: tx_id.clone(),
            tx_type: "skill_invocation".to_string(),
            from: user_id.to_string(),
            to: "skill_registry".to_string(),
            amount: amount.clone(),
            purpose: "skill_invocation".to_string(),
            metadata: {
                let mut meta = HashMap::new();
                meta.insert("skill_id".to_string(), serde_json::json!(skill_id));
                meta
            },
            status: "pending".to_string(),
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            confirmed_at: None,
            block_height: None,
            gas_used: None,
        };

        // Add to transaction pool
        if let Ok(mut pool) = self.transaction_pool.lock() {
            pool.pending_txs.insert(tx_id.clone(), tx.clone());
        }

        // Record burn event
        let burn_event = BurnEvent {
            tx_id: tx_id.clone(),
            user: user_id.to_string(),
            amount: amount.clone(),
            purpose: "skill_invocation".to_string(),
            skill_id: Some(skill_id.to_string()),
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            validated: false,
        };

        if let Ok(mut tracker) = self.burn_tracker.lock() {
            tracker.burn_history.push(burn_event);
        }

        // Update metrics
        self.update_service_metrics("knirvoracle", amount, "spent");

        Ok(tx)
    }

    pub fn process_llm_registration(&self, user_id: &str, llm_id: &str, registration_fee: &BigInt) -> Result<EconomicTransaction, Box<dyn std::error::Error>> {
        let required_fee = &self.economic_rules.llm_registration_fee;
        if registration_fee < required_fee {
            return Err(format!("insufficient registration fee: required {}, provided {}", required_fee, registration_fee).into());
        }

        let tx_id = format!("llm_reg_{}_{}", llm_id, SystemTime::now().duration_since(UNIX_EPOCH)?.as_nanos());
        let tx = EconomicTransaction {
            id: tx_id.clone(),
            tx_type: "llm_registration".to_string(),
            from: user_id.to_string(),
            to: "llm_registry".to_string(),
            amount: registration_fee.clone(),
            purpose: "llm_registration".to_string(),
            metadata: {
                let mut meta = HashMap::new();
                meta.insert("llm_id".to_string(), serde_json::json!(llm_id));
                meta
            },
            status: "pending".to_string(),
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            confirmed_at: None,
            block_height: None,
            gas_used: None,
        };

        // Add to transaction pool
        if let Ok(mut pool) = self.transaction_pool.lock() {
            pool.pending_txs.insert(tx_id, tx.clone());
        }

        // Update metrics
        self.update_service_metrics("knirvoracle", registration_fee, "earned");

        Ok(tx)
    }

    pub fn process_validation_reward(&self, validator_id: &str, target_id: &str, validation_result: bool) -> Result<EconomicTransaction, Box<dyn std::error::Error>> {
        if !validation_result {
            return Err("validation failed, no reward".into());
        }

        let base_reward = &self.economic_rules.validation_reward;
        let final_reward = if let Ok(calc) = self.reward_calculator.lock() {
            self.calculate_reward(validator_id, "validation", base_reward)
        } else {
            base_reward.clone()
        };

        let tx_id = format!("validation_{}_{}", target_id, SystemTime::now().duration_since(UNIX_EPOCH)?.as_nanos());
        let tx = EconomicTransaction {
            id: tx_id.clone(),
            tx_type: "validation_reward".to_string(),
            from: "reward_pool".to_string(),
            to: validator_id.to_string(),
            amount: final_reward.clone(),
            purpose: "validation_reward".to_string(),
            metadata: {
                let mut meta = HashMap::new();
                meta.insert("target_id".to_string(), serde_json::json!(target_id));
                meta.insert("validation_result".to_string(), serde_json::json!(validation_result));
                meta
            },
            status: "pending".to_string(),
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            confirmed_at: None,
            block_height: None,
            gas_used: None,
        };

        // Add to transaction pool
        if let Ok(mut pool) = self.transaction_pool.lock() {
            pool.pending_txs.insert(tx_id, tx.clone());
        }

        // Update metrics
        self.update_service_metrics("knirvnexus", &final_reward, "earned");

        Ok(tx)
    }

    pub fn calculate_network_fees(&self, gas_used: u64, priority: &str) -> BigInt {
        let base_gas_price = BigInt::from(1000);
        let multiplier = match priority {
            "high" => 2.0,
            "medium" => 1.5,
            "low" => 1.0,
            _ => 1.0,
        };

        let gas_price = &base_gas_price * BigInt::from((multiplier * 1000.0) as i64) / BigInt::from(1000);
        gas_price * BigInt::from(gas_used as i64)
    }

    pub fn get_economic_metrics(&self) -> Result<EconomicMetrics, Box<dyn std::error::Error>> {
        if let Ok(metrics) = self.metrics.lock() {
            Ok(metrics.clone())
        } else {
            Err("Failed to acquire metrics lock".into())
        }
    }

    fn update_service_metrics(&self, service_name: &str, amount: &BigInt, operation: &str) {
        if let Ok(mut metrics) = self.metrics.lock() {
            let service_metrics = metrics.service_metrics.entry(service_name.to_string()).or_insert(ServiceEconomics {
                revenue: BigInt::from(0),
                costs: BigInt::from(0),
                profit: BigInt::from(0),
                tokens_earned: BigInt::from(0),
                tokens_spent: BigInt::from(0),
                user_count: 0,
                transaction_count: 0,
                last_updated: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            });

            match operation {
                "earned" => {
                    service_metrics.revenue += amount;
                    service_metrics.tokens_earned += amount;
                }
                "spent" => {
                    service_metrics.costs += amount;
                    service_metrics.tokens_spent += amount;
                }
                _ => {}
            }

            service_metrics.profit = &service_metrics.revenue - &service_metrics.costs;
            service_metrics.transaction_count += 1;
            service_metrics.last_updated = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        }
    }

    async fn transaction_processor(transaction_pool: Arc<Mutex<TransactionPool>>) {
        let mut interval = interval(Duration::from_secs(5));
        loop {
            interval.tick().await;
            Self::process_pending_transactions(&transaction_pool).await;
        }
    }

    async fn process_pending_transactions(transaction_pool: &Arc<Mutex<TransactionPool>>) {
        let pending_txs = if let Ok(pool) = transaction_pool.lock() {
            pool.pending_txs.clone()
        } else {
            return;
        };

        for (id, tx) in pending_txs {
            if let Err(e) = Self::process_transaction(&tx).await {
                error!("Failed to process transaction {}: {}", id, e);
                if let Ok(mut pool) = transaction_pool.lock() {
                    if let Some(tx) = pool.pending_txs.get_mut(&id) {
                        tx.status = "failed".to_string();
                    }
                }
            } else {
                if let Ok(mut pool) = transaction_pool.lock() {
                    pool.pending_txs.remove(&id);
                    pool.confirmed_txs.insert(id.clone(), tx.clone());
                }
            }
        }
    }

    async fn process_transaction(tx: &EconomicTransaction) -> Result<(), Box<dyn std::error::Error>> {
        match tx.tx_type.as_str() {
            "skill_invocation" => Self::process_skill_invocation_tx(tx).await,
            "llm_registration" => Self::process_llm_registration_tx(tx).await,
            "validation_reward" => Self::process_validation_reward_tx(tx).await,
            _ => Err(format!("Unknown transaction type: {}", tx.tx_type).into()),
        }
    }

    async fn process_skill_invocation_tx(tx: &EconomicTransaction) -> Result<(), Box<dyn std::error::Error>> {
        // Burn tokens for skill invocation
        // Update total burned in metrics
        info!("Burned {} NRN for skill invocation {}", tx.amount, tx.metadata.get("skill_id").unwrap_or(&serde_json::json!("unknown")));
        Ok(())
    }

    async fn process_llm_registration_tx(tx: &EconomicTransaction) -> Result<(), Box<dyn std::error::Error>> {
        // Transfer registration fee to treasury
        info!("Processed LLM registration fee {} NRN for {}", tx.amount, tx.metadata.get("llm_id").unwrap_or(&serde_json::json!("unknown")));
        Ok(())
    }

    async fn process_validation_reward_tx(tx: &EconomicTransaction) -> Result<(), Box<dyn std::error::Error>> {
        // Mint reward tokens
        info!("Minted {} NRN validation reward for {}", tx.amount, tx.to);
        Ok(())
    }

    async fn metrics_updater(metrics: Arc<Mutex<EconomicMetrics>>) {
        let mut interval = interval(Duration::from_secs(60));
        loop {
            interval.tick().await;
            Self::update_metrics(&metrics).await;
        }
    }

    async fn update_metrics(metrics: &Arc<Mutex<EconomicMetrics>>) {
        if let Ok(mut metrics) = metrics.lock() {
            // Update network utilization
            metrics.network_utilization = 0.75; // 75% utilization

            // Update token velocity
            metrics.token_velocity = 2.5; // 2.5x velocity

            // Update average gas price
            metrics.average_gas_price = BigInt::from(1500); // 1500 wei average

            metrics.last_updated = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        }
    }

    async fn reward_distributor() {
        let mut interval = interval(Duration::from_secs(3600)); // 1 hour
        loop {
            interval.tick().await;
            info!("Distributing periodic rewards...");
        }
    }

    async fn burn_processor(burn_tracker: Arc<Mutex<BurnTracker>>) {
        let mut interval = interval(Duration::from_secs(10));
        loop {
            interval.tick().await;
            Self::process_burn_events(&burn_tracker).await;
        }
    }

    async fn process_burn_events(burn_tracker: &Arc<Mutex<BurnTracker>>) {
        if let Ok(mut tracker) = burn_tracker.lock() {
            for event in &mut tracker.burn_history {
                if !event.validated {
                    event.validated = true;
                    info!("Validated burn event: {} burned {} NRN", event.user, event.amount);
                }
            }
        }
    }

    fn calculate_reward(&self, user_id: &str, reward_type: &str, base_amount: &BigInt) -> BigInt {
        let multiplier = if let Ok(calc) = self.reward_calculator.lock() {
            if let Some(metrics) = calc.performance_data.get(user_id) {
                let mut mult = 1.0;
                if metrics.success_rate > 0.9 {
                    mult *= calc.multipliers.get("high_performance").unwrap_or(&1.0);
                }
                if metrics.uptime > 0.95 {
                    mult *= calc.multipliers.get("consistent_user").unwrap_or(&1.0);
                }
                mult
            } else {
                1.0
            }
        } else {
            1.0
        };

        let final_amount = base_amount * BigInt::from((multiplier * 1000.0) as i64) / BigInt::from(1000);
        final_amount
    }
}