use bevy::prelude::*;
use serde::{Deserialize, Serialize};
use crate::components::*;
use crate::resources::*;

// ============================================================================
// NRN TOKEN ECONOMICS COMPONENTS AND RESOURCES
// ============================================================================

/// Resource for managing NRN token transactions and blockchain integration
#[derive(Resource, Default)]
pub struct NRNTokenManager {
    pub pending_transactions: Vec<PendingTransaction>,
    pub transaction_history: Vec<CompletedTransaction>,
    pub blockchain_connected: bool,
    pub last_sync_time: f64,
    pub gas_price: f64,
    pub network_fees: f64,
}

/// Represents a pending NRN transaction
#[derive(Serialize, Deserialize, Clone)]
pub struct PendingTransaction {
    pub id: String,
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub recipient: Option<String>,
    pub metadata: TransactionMetadata,
    pub created_at: f64,
    pub estimated_completion: f64,
}

/// Represents a completed NRN transaction
#[derive(Serialize, Deserialize, Clone)]
pub struct CompletedTransaction {
    pub id: String,
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub recipient: Option<String>,
    pub metadata: TransactionMetadata,
    pub completed_at: f64,
    pub blockchain_hash: Option<String>,
    pub gas_used: f64,
}

/// Types of NRN transactions in the game
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum TransactionType {
    AgentDeployment,
    SkillInvocation,
    BountyReward,
    AgentUpgrade,
    SkillPurchase,
    NetworkFee,
    Staking,
    Unstaking,
}

/// Metadata associated with transactions
#[derive(Serialize, Deserialize, Clone)]
pub struct TransactionMetadata {
    pub agent_id: Option<String>,
    pub error_node_id: Option<String>,
    pub skill_id: Option<String>,
    pub description: String,
    pub game_context: String,
}

/// Resource for tracking NRN costs and pricing
#[derive(Resource)]
pub struct NRNPricing {
    pub agent_deployment_cost: f64,
    pub agent_upgrade_base_cost: f64,
    pub skill_invocation_base_cost: f64,
    pub network_fee_percentage: f64,
    pub bounty_multiplier: f64,
    pub staking_reward_rate: f64,
    pub dynamic_pricing_enabled: bool,
}

impl Default for NRNPricing {
    fn default() -> Self {
        Self {
            agent_deployment_cost: 10.0,
            agent_upgrade_base_cost: 25.0,
            skill_invocation_base_cost: 5.0,
            network_fee_percentage: 0.02, // 2%
            bounty_multiplier: 1.0,
            staking_reward_rate: 0.05, // 5% APY
            dynamic_pricing_enabled: true,
        }
    }
}

/// Component for entities that can consume NRN tokens
#[derive(Component)]
pub struct NRNConsumer {
    pub consumption_rate: f64,
    pub last_consumption: f64,
    pub total_consumed: f64,
    pub efficiency_bonus: f32,
}

/// Component for entities that can generate NRN rewards
#[derive(Component)]
pub struct NRNRewardSource {
    pub base_reward: f64,
    pub multiplier: f32,
    pub last_reward_time: f64,
    pub total_rewards_given: f64,
}

// ============================================================================
// NRN ECONOMICS SYSTEMS
// ============================================================================

/// System to handle NRN token consumption for agent actions
pub fn nrn_consumption_system(
    mut agent_query: Query<(&mut AIAgent, &mut NRNConsumer)>,
    mut player_resources: ResMut<PlayerResources>,
    mut nrn_manager: ResMut<NRNTokenManager>,
    nrn_pricing: Res<NRNPricing>,
    time: Res<Time>,
) {
    for (mut agent, mut consumer) in agent_query.iter_mut() {
        match agent.status {
            AgentStatus::Working => {
                // Calculate consumption based on agent efficiency and time
                let base_consumption = nrn_pricing.skill_invocation_base_cost;
                let efficiency_factor = agent.efficiency;
                let time_factor = time.delta_seconds() as f64;
                
                let consumption = base_consumption * efficiency_factor as f64 * time_factor * 0.1;
                
                if player_resources.nrn_balance >= consumption {
                    player_resources.nrn_balance -= consumption;
                    consumer.total_consumed += consumption;
                    consumer.last_consumption = time.elapsed_seconds_f64();
                    
                    // Create transaction record
                    let transaction = PendingTransaction {
                        id: format!("consume_{}_{}", agent.id, fastrand::u32(..)),
                        transaction_type: TransactionType::SkillInvocation,
                        amount: consumption,
                        recipient: None,
                        metadata: TransactionMetadata {
                            agent_id: Some(agent.id.clone()),
                            error_node_id: agent.current_task.clone(),
                            skill_id: None,
                            description: "NRN consumed for agent skill invocation".to_string(),
                            game_context: "agent_working".to_string(),
                        },
                        created_at: time.elapsed_seconds_f64(),
                        estimated_completion: time.elapsed_seconds_f64() + 1.0,
                    };
                    
                    nrn_manager.pending_transactions.push(transaction);
                } else {
                    // Insufficient NRN - reduce agent efficiency or stop work
                    agent.efficiency *= 0.9; // Reduce efficiency by 10%
                    if agent.efficiency < 0.1 {
                        agent.status = AgentStatus::Resting;
                        agent.thought_process.push("Insufficient NRN for continued operation".to_string());
                    }
                }
            },
            _ => {
                // Slowly restore efficiency when not working
                if agent.efficiency < 1.0 {
                    agent.efficiency = (agent.efficiency + 0.1 * time.delta_seconds()).min(1.0);
                }
            }
        }
    }
}

/// System to handle NRN bounty rewards when ErrorNodes are solved
pub fn nrn_bounty_system(
    mut error_node_query: Query<(&mut ErrorNode, &mut NRNRewardSource), Changed<ErrorNode>>,
    mut player_resources: ResMut<PlayerResources>,
    mut nrn_manager: ResMut<NRNTokenManager>,
    nrn_pricing: Res<NRNPricing>,
    agent_query: Query<&AIAgent>,
    time: Res<Time>,
) {
    for (mut error_node, mut reward_source) in error_node_query.iter_mut() {
        // Check if ErrorNode was just completed
        if error_node.progress >= 1.0 && error_node.is_being_solved {
            if let Some(solver_agent_id) = &error_node.solver_agent_id {
                // Find the solving agent to determine bonus
                let mut efficiency_bonus = 1.0;
                for agent in agent_query.iter() {
                    if agent.id == *solver_agent_id {
                        efficiency_bonus = agent.efficiency;
                        break;
                    }
                }
                
                // Calculate final bounty with bonuses
                let base_bounty = error_node.bounty;
                let difficulty_bonus = 1.0 + error_node.difficulty as f64 * 0.5;
                let efficiency_bonus = efficiency_bonus as f64;
                let network_fee = base_bounty * nrn_pricing.network_fee_percentage;
                
                let final_bounty = (base_bounty * difficulty_bonus * efficiency_bonus) - network_fee;
                
                // Award bounty to player
                player_resources.nrn_balance += final_bounty;
                player_resources.total_bounty_earned += final_bounty;
                reward_source.total_rewards_given += final_bounty;
                reward_source.last_reward_time = time.elapsed_seconds_f64();
                
                // Create transaction record
                let transaction = CompletedTransaction {
                    id: format!("bounty_{}_{}", error_node.id, fastrand::u32(..)),
                    transaction_type: TransactionType::BountyReward,
                    amount: final_bounty,
                    recipient: Some("player".to_string()),
                    metadata: TransactionMetadata {
                        agent_id: Some(solver_agent_id.clone()),
                        error_node_id: Some(error_node.id.clone()),
                        skill_id: None,
                        description: format!("Bounty reward for solving ErrorNode {}", error_node.id),
                        game_context: "error_node_solved".to_string(),
                    },
                    completed_at: time.elapsed_seconds_f64(),
                    blockchain_hash: None, // Would be filled by blockchain integration
                    gas_used: network_fee,
                };
                
                nrn_manager.transaction_history.push(transaction);
                
                info!("Awarded {} NRN bounty for solving ErrorNode {}", final_bounty, error_node.id);
            }
        }
    }
}

/// System to handle agent deployment costs
pub fn agent_deployment_cost_system(
    mut agent_query: Query<&mut AIAgent, Changed<AIAgent>>,
    mut player_resources: ResMut<PlayerResources>,
    mut nrn_manager: ResMut<NRNTokenManager>,
    nrn_pricing: Res<NRNPricing>,
    time: Res<Time>,
) {
    for mut agent in agent_query.iter_mut() {
        // Check if agent was just deployed (status changed to Moving)
        if agent.status == AgentStatus::Moving && agent.current_task.is_some() {
            let deployment_cost = nrn_pricing.agent_deployment_cost;
            
            if player_resources.nrn_balance >= deployment_cost {
                player_resources.nrn_balance -= deployment_cost;
                
                // Create transaction record
                let transaction = PendingTransaction {
                    id: format!("deploy_{}_{}", agent.id, fastrand::u32(..)),
                    transaction_type: TransactionType::AgentDeployment,
                    amount: deployment_cost,
                    recipient: None,
                    metadata: TransactionMetadata {
                        agent_id: Some(agent.id.clone()),
                        error_node_id: agent.current_task.clone(),
                        skill_id: None,
                        description: format!("Deployment cost for agent {}", agent.id),
                        game_context: "agent_deployment".to_string(),
                    },
                    created_at: time.elapsed_seconds_f64(),
                    estimated_completion: time.elapsed_seconds_f64() + 2.0,
                };
                
                nrn_manager.pending_transactions.push(transaction);
                
                info!("Charged {} NRN for deploying agent {}", deployment_cost, agent.id);
            } else {
                // Insufficient funds - cancel deployment
                agent.status = AgentStatus::Idle;
                agent.current_task = None;
                agent.thought_process.push("Deployment cancelled: insufficient NRN".to_string());
                
                warn!("Agent deployment cancelled: insufficient NRN balance");
            }
        }
    }
}

/// System to handle dynamic pricing based on network activity
pub fn dynamic_pricing_system(
    mut nrn_pricing: ResMut<NRNPricing>,
    knirv_graph: Res<KnirvGraphState>,
    player_resources: Res<PlayerResources>,
    time: Res<Time>,
) {
    if !nrn_pricing.dynamic_pricing_enabled {
        return;
    }
    
    // Adjust pricing based on network activity
    let error_node_count = knirv_graph.error_nodes.len() as f64;
    let agent_count = knirv_graph.agents.len() as f64;
    
    // Higher demand (more agents, fewer errors) increases prices
    let demand_factor = if error_node_count > 0.0 {
        agent_count / error_node_count
    } else {
        1.0
    };
    
    // Adjust deployment cost based on demand
    let base_deployment_cost = 10.0;
    nrn_pricing.agent_deployment_cost = base_deployment_cost * (1.0 + demand_factor * 0.2);
    
    // Adjust skill invocation cost based on player's NRN balance
    let balance_factor = if player_resources.nrn_balance > 1000.0 {
        1.2 // Higher costs for wealthy players
    } else if player_resources.nrn_balance < 100.0 {
        0.8 // Lower costs for players with low balance
    } else {
        1.0
    };
    
    let base_skill_cost = 5.0;
    nrn_pricing.skill_invocation_base_cost = base_skill_cost * balance_factor;
}

/// System to process pending transactions and simulate blockchain integration
pub fn blockchain_integration_system(
    mut nrn_manager: ResMut<NRNTokenManager>,
    time: Res<Time>,
) {
    let current_time = time.elapsed_seconds_f64();
    
    // Process pending transactions
    let mut completed_transactions = Vec::new();
    nrn_manager.pending_transactions.retain(|transaction| {
        if current_time >= transaction.estimated_completion {
            // Simulate blockchain confirmation
            let completed = CompletedTransaction {
                id: transaction.id.clone(),
                transaction_type: transaction.transaction_type.clone(),
                amount: transaction.amount,
                recipient: transaction.recipient.clone(),
                metadata: transaction.metadata.clone(),
                completed_at: current_time,
                blockchain_hash: Some(format!("0x{:x}", fastrand::u64(..))),
                gas_used: transaction.amount * 0.01, // 1% gas fee
            };
            
            completed_transactions.push(completed);
            false // Remove from pending
        } else {
            true // Keep in pending
        }
    });
    
    // Add completed transactions to history
    nrn_manager.transaction_history.extend(completed_transactions);
    
    // Simulate blockchain connection status
    nrn_manager.blockchain_connected = fastrand::f32() > 0.05; // 95% uptime
    nrn_manager.last_sync_time = current_time;
}
