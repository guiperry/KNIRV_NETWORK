/**
 * Example usage of KNIRV Gateway SDK in TypeScript/JavaScript
 */

import { KNIRVGatewayClient } from '../client';
import {
  SkillInvocationRequest,
  LLMRegistrationRequest,
  ValidationRewardRequest,
  NetworkFeesRequest,
} from '../types';

// Setup environment variables for development
process.env.ECONOMICS_SERVICE_URL = 'http://localhost:8090';
process.env.GATEWAY_SERVICE_URL = 'http://localhost:8000';
process.env.KNIRVCHAIN_URL = 'http://localhost:8080';
process.env.KNIRVNEXUS_URL = 'http://localhost:8081';
process.env.KNIRVROOT_URL = 'http://localhost:8082';
process.env.KNIRVGRAPH_URL = 'http://localhost:8083';

async function main() {
  try {
    // Example 1: Basic client setup
    await basicExample();

    // Example 2: Economics service operations
    await economicsExample();

    // Example 3: Health monitoring
    await healthExample();

    // Example 4: Integration status
    await integrationExample();

    // Example 5: Advanced usage
    await advancedExample();
  } catch (error) {
    console.error('Example execution failed:', error);
  }
}

async function basicExample() {
  console.log('=== Basic Client Setup ===');

  // Create a client with default options
  const client = new KNIRVGatewayClient();

  // Or create with custom options
  const customClient = new KNIRVGatewayClient({
    environment: 'development',
    debug: true,
    timeout: 60000,
  });

  // Create economics-specific client
  const economicsClient = KNIRVGatewayClient.createEconomicsClient({
    apiKey: 'your-api-key',
    debug: true,
  });

  console.log('Default client created:', !!client);
  console.log('Custom client created:', !!customClient);
  console.log('Economics client created:', !!economicsClient);
}

async function economicsExample() {
  console.log('\n=== Economics Service Operations ===');

  // Create economics client
  const client = KNIRVGatewayClient.createEconomicsClient({
    environment: 'development',
    debug: true,
  });

  try {
    // Example 1: Process skill invocation
    const skillRequest: SkillInvocationRequest = {
      user_id: 'user123',
      skill_id: 'skill456',
      amount: '100000', // 0.1 NRN
    };

    const skillResp = await client.economics.invokeSkill(skillRequest);
    console.log('Skill invocation successful:', skillResp);

    // Example 2: Register LLM
    const llmRequest: LLMRegistrationRequest = {
      user_id: 'user123',
      llm_id: 'llm789',
      registration_fee: '1000000', // 1 NRN
    };

    const llmResp = await client.economics.registerLLM(llmRequest);
    console.log('LLM registration successful:', llmResp);

    // Example 3: Process validation reward
    const validationRequest: ValidationRewardRequest = {
      validator_id: 'validator123',
      target_id: 'target456',
      validation_result: true,
    };

    const validationResp = await client.economics.processValidationReward(validationRequest);
    console.log('Validation reward successful:', validationResp);

    // Example 4: Calculate network fees
    const feesRequest: NetworkFeesRequest = {
      gas_used: 21000,
      priority: 'medium',
    };

    const feesResp = await client.economics.calculateNetworkFees(feesRequest);
    console.log('Fee calculation successful:', feesResp);

    // Example 5: Get economic metrics
    const metrics = await client.economics.getMetrics();
    console.log(`Economic metrics: Total Supply: ${metrics.total_supply}, Total Burned: ${metrics.total_burned}`);

    // Example 6: Get transaction history
    const transactions = await client.economics.listTransactions({ limit: 10 });
    console.log(`Retrieved ${transactions.items.length} transactions`);

    // Example 7: Get burn history
    const burnEvents = await client.economics.getBurnHistory(5);
    console.log(`Retrieved ${burnEvents.items.length} burn events`);

    // Example 8: Get economic rules
    const rules = await client.economics.getEconomicRules();
    console.log(`Skill invocation cost: ${rules.skill_invocation_cost}, LLM registration fee: ${rules.llm_registration_fee}`);

  } catch (error) {
    console.error('Economics operations failed:', error);
  }
}

async function healthExample() {
  console.log('\n=== Health Monitoring ===');

  const client = new KNIRVGatewayClient({
    environment: 'development',
  });

  try {
    // Check economics service health
    const economicsHealth = await client.health.checkEconomicsHealth();
    console.log('Economics service health:', economicsHealth);

    // Check gateway health
    const gatewayHealth = await client.health.checkGatewayHealth();
    console.log('Gateway service health:', gatewayHealth);

    // Check all services
    const allHealth = await client.health.checkAllServices();
    console.log('All services health:', allHealth);

    // Get system health summary
    const systemHealth = await client.health.getSystemHealth();
    console.log('System health summary:', systemHealth);

    // Wait for a service to become healthy (with timeout)
    console.log('Waiting for economics service...');
    const isReady = await client.health.waitForService('economics', {
      timeout: 10000,
      interval: 1000,
    });
    console.log('Economics service ready:', isReady);

  } catch (error) {
    console.error('Health monitoring failed:', error);
  }
}

async function integrationExample() {
  console.log('\n=== Integration Status ===');

  const client = new KNIRVGatewayClient({
    environment: 'development',
    serviceURLs: {
      knirvchain: 'http://localhost:8080',
      knirvnexus: 'http://localhost:8081',
      knirvroot: 'http://localhost:8082',
      knirvgraph: 'http://localhost:8083',
    },
  });

  try {
    // Get integration status
    const status = await client.integration.getStatus();
    console.log('Integration status:', status);

    // Test connectivity to all components
    const connectivity = await client.integration.testConnectivity();
    console.log('Component connectivity:', connectivity);

    // Get integration metrics
    const metrics = await client.integration.getMetrics();
    console.log('Integration metrics:', metrics);

    // Get component details
    const chainDetails = await client.integration.getComponentDetails('knirvchain');
    console.log('KNIRVCHAIN details:', chainDetails);

    // Trigger manual sync
    const syncResult = await client.integration.triggerSync();
    console.log('Manual sync result:', syncResult);

  } catch (error) {
    console.error('Integration operations failed:', error);
  }
}

async function advancedExample() {
  console.log('\n=== Advanced Usage ===');

  const client = new KNIRVGatewayClient({
    environment: 'development',
    verbose: true,
  });

  try {
    // Custom workflow: Check balance, process skill, verify transaction
    const userId = 'advanced_user';
    const skillId = 'advanced_skill';
    const amount = '500000'; // 0.5 NRN

    // 1. Get current metrics to check system state
    const metrics = await client.economics.getMetrics();
    console.log(`Current system state - Total Supply: ${metrics.total_supply}, Network Utilization: ${metrics.network_utilization}`);

    // 2. Calculate fees for the operation
    const feesResp = await client.economics.calculateNetworkFees({
      gas_used: 50000,
      priority: 'high',
    });
    console.log(`Estimated fees: ${feesResp.total_fee}`);

    // 3. Check user economics summary
    const userSummary = await client.economics.getUserEconomicsSummary(userId);
    console.log('User economics summary:', userSummary);

    // 4. Check if user has sufficient balance
    const hasBalance = await client.economics.checkSkillInvocationBalance(userId, skillId, amount);
    console.log('User has sufficient balance:', hasBalance);

    if (hasBalance) {
      // 5. Process the skill invocation
      const skillResp = await client.economics.invokeSkill({
        user_id: userId,
        skill_id: skillId,
        amount: amount,
      });
      console.log(`Skill invocation completed: Transaction ID: ${skillResp.transaction_id}`);

      // 6. Verify the transaction
      const transaction = await client.economics.getTransaction(skillResp.transaction_id);
      console.log(`Transaction verified: Status: ${transaction.status}, Amount: ${transaction.amount}`);

      // 7. Get updated metrics
      const updatedMetrics = await client.economics.getMetrics();
      console.log(`Updated system state - Total Burned: ${updatedMetrics.total_burned}`);
    } else {
      console.log('Insufficient balance for skill invocation');
    }

    // 8. Get network statistics
    const networkStats = await client.economics.getNetworkStatistics();
    console.log('Network statistics:', networkStats);

  } catch (error) {
    console.error('Advanced example failed:', error);
  }
}

// Error handling example
async function errorHandlingExample() {
  console.log('\n=== Error Handling ===');

  const client = new KNIRVGatewayClient({
    environment: 'development',
  });

  try {
    // Attempt to get a non-existent transaction
    await client.economics.getTransaction('non-existent-tx-id');
  } catch (error) {
    if (error.name === 'EconomicsServiceError') {
      console.log('Caught economics service error:', error.message);
    } else {
      console.log('Caught general error:', error.message);
    }
  }

  try {
    // Attempt to invoke skill with invalid data
    await client.economics.invokeSkill({
      user_id: '',
      skill_id: '',
      amount: 'invalid-amount',
    });
  } catch (error) {
    console.log('Caught validation error:', error.message);
  }
}

// Real-time monitoring example
async function monitoringExample() {
  console.log('\n=== Real-time Monitoring ===');

  const client = new KNIRVGatewayClient({
    environment: 'development',
  });

  // Monitor system health every 30 seconds
  const healthMonitor = setInterval(async () => {
    try {
      const systemHealth = await client.health.getSystemHealth();
      console.log(`[${new Date().toISOString()}] System Status: ${systemHealth.status}`);
      
      if (systemHealth.status === 'unhealthy') {
        console.log('⚠️  System is unhealthy, checking individual services...');
        const detailedHealth = await client.health.getDetailedHealth();
        console.log('Detailed health:', detailedHealth);
      }
    } catch (error) {
      console.error('Health monitoring error:', error.message);
    }
  }, 30000);

  // Stop monitoring after 5 minutes
  setTimeout(() => {
    clearInterval(healthMonitor);
    console.log('Health monitoring stopped');
  }, 300000);
}

// Run the examples
if (require.main === module) {
  main().catch(console.error);
}
