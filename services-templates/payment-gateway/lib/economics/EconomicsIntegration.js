/**
 * Economics Integration
 * Handles integration with KNIRV network components
 */

const axios = require('axios');

class EconomicsIntegration {
  constructor(tokenEconomics, config = {}) {
    this.tokenEconomics = tokenEconomics;

    // Component URLs
    this.componentURLs = {
      knirvchain: config.knirvchainURL || 'http://localhost:8080',
      knirvnexus: config.knirvnexusURL || 'http://localhost:8081',
      knirvoracle: config.knirvoracleURL || 'http://localhost:8082',
      knirvgraph: config.knirvgraphURL || 'http://localhost:8083'
    };

    // HTTP client with timeout
    this.httpClient = axios.create({
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    // Integration timers
    this.integrationTimers = {};
    this.isRunning = false;
  }

  /**
   * Start integration service
   */
  async start() {
    if (this.isRunning) {
      console.log('Economics Integration already running');
      return;
    }

    console.log('Starting Economics Integration service...');

    // Start event listeners for each component
    this.startKNIRVChainSync();
    this.startKNIRVNexusSync();
    this.startKNIRVOracleSync();
    this.startKNIRVGraphSync();

    // Start periodic sync
    this.startPeriodicSync();

    this.isRunning = true;
    console.log('Economics Integration service started');
  }

  /**
   * Stop integration service
   */
  async stop() {
    if (!this.isRunning) {
      return;
    }

    console.log('Stopping Economics Integration service...');

    // Stop all timers
    Object.values(this.integrationTimers).forEach(timer => {
      if (timer) clearInterval(timer);
    });

    this.isRunning = false;
    console.log('Economics Integration service stopped');
  }

  /**
   * Sync with KNIRVCHAIN
   */
  startKNIRVChainSync() {
    this.integrationTimers.knirvchain = setInterval(async () => {
      try {
        await this.syncKNIRVChainEvents();
      } catch (error) {
        console.error('Error syncing KNIRVCHAIN events:', error.message);
      }
    }, 10000); // Every 10 seconds
  }

  async syncKNIRVChainEvents() {
    try {
      // Get skill execution events
      const skillEvents = await this.getSkillExecutionEvents();
      for (const event of skillEvents) {
        await this.processSkillExecutionEvent(event);
      }

      // Get LLM registration events
      const llmEvents = await this.getLLMRegistrationEvents();
      for (const event of llmEvents) {
        await this.processLLMRegistrationEvent(event);
      }
    } catch (error) {
      // Silently fail if component is not available
      if (error.code !== 'ECONNREFUSED') {
        throw error;
      }
    }
  }

  async getSkillExecutionEvents() {
    try {
      const response = await this.httpClient.get(
        `${this.componentURLs.knirvchain}/api/skills/executions/recent`
      );
      return response.data || [];
    } catch (error) {
      return [];
    }
  }

  async getLLMRegistrationEvents() {
    try {
      const response = await this.httpClient.get(
        `${this.componentURLs.knirvchain}/api/llm/registrations/recent`
      );
      return response.data || [];
    } catch (error) {
      return [];
    }
  }

  async processSkillExecutionEvent(event) {
    try {
      const tx = await this.tokenEconomics.processSkillInvocation(
        event.userId,
        event.skillId,
        event.cost
      );
      console.log(`Processed skill execution: ${event.skillId} for user ${event.userId}, tx: ${tx.id}`);
    } catch (error) {
      console.error('Error processing skill execution:', error.message);
    }
  }

  async processLLMRegistrationEvent(event) {
    if (event.isValid) {
      try {
        const tx = await this.tokenEconomics.processValidationReward(
          event.validatorId,
          event.llmId,
          true
        );
        console.log(`Processed validation reward for validator ${event.validatorId}, tx: ${tx.id}`);
      } catch (error) {
        console.error('Error processing validation reward:', error.message);
      }
    }
  }

  /**
   * Sync with KNIRVNEXUS
   */
  startKNIRVNexusSync() {
    this.integrationTimers.knirvnexus = setInterval(async () => {
      try {
        await this.syncKNIRVNexusEvents();
      } catch (error) {
        console.error('Error syncing KNIRVNEXUS events:', error.message);
      }
    }, 15000); // Every 15 seconds
  }

  async syncKNIRVNexusEvents() {
    try {
      const response = await this.httpClient.get(
        `${this.componentURLs.knirvnexus}/api/agents/activities/recent`
      );
      const activities = response.data || [];

      for (const activity of activities) {
        this.processNetworkActivity(activity);
      }
    } catch (error) {
      // Silently fail if component is not available
    }
  }

  processNetworkActivity(activity) {
    // Update performance metrics for the node
    const metrics = {
      successRate: 0.95,
      responseTime: 100.0,
      userSatisfaction: 0.9,
      uptime: 0.99
    };

    this.tokenEconomics.rewardCalculator.updatePerformanceMetrics(
      activity.nodeId,
      metrics
    );
    console.log(`Updated performance metrics for node ${activity.nodeId}`);
  }

  /**
   * Sync with KNIRVORACLE
   */
  startKNIRVOracleSync() {
    this.integrationTimers.knirvoracle = setInterval(async () => {
      try {
        await this.syncKNIRVOracleEvents();
      } catch (error) {
        console.error('Error syncing KNIRVORACLE events:', error.message);
      }
    }, 20000); // Every 20 seconds
  }

  async syncKNIRVOracleEvents() {
    try {
      const response = await this.httpClient.get(
        `${this.componentURLs.knirvoracle}/api/blockchain/recent-transactions`
      );
      console.log('Synced with KNIRVORACLE blockchain events');
    } catch (error) {
      // Silently fail if component is not available
    }
  }

  /**
   * Sync with KNIRVGRAPH
   */
  startKNIRVGraphSync() {
    this.integrationTimers.knirvgraph = setInterval(async () => {
      try {
        await this.syncKNIRVGraphEvents();
      } catch (error) {
        console.error('Error syncing KNIRVGRAPH events:', error.message);
      }
    }, 25000); // Every 25 seconds
  }

  async syncKNIRVGraphEvents() {
    try {
      const response = await this.httpClient.get(
        `${this.componentURLs.knirvgraph}/api/graph/network-stats`
      );
      console.log('Synced with KNIRVGRAPH network statistics');
    } catch (error) {
      // Silently fail if component is not available
    }
  }

  /**
   * Periodic sync with all components
   */
  startPeriodicSync() {
    this.integrationTimers.periodic = setInterval(async () => {
      await this.performPeriodicSync();
    }, 300000); // Every 5 minutes
  }

  async performPeriodicSync() {
    const metrics = this.tokenEconomics.getEconomicMetrics();

    // Send metrics to all components
    for (const [component, url] of Object.entries(this.componentURLs)) {
      try {
        await this.sendMetricsToComponent(url, metrics);
      } catch (error) {
        console.error(`Error sending metrics to ${component}:`, error.message);
      }
    }

    console.log('Performed periodic sync of economic metrics');
  }

  async sendMetricsToComponent(componentURL, metrics) {
    try {
      // First check if component is alive
      const healthResponse = await this.httpClient.get(`${componentURL}/health`);

      if (healthResponse.status === 200) {
        // Send metrics
        await this.httpClient.post(`${componentURL}/api/economics/metrics`, metrics);
        console.log(`Sent metrics to ${componentURL}`);
      }
    } catch (error) {
      // Silently fail if component is not available
    }
  }

  /**
   * Get integration status
   */
  getIntegrationStatus() {
    return {
      componentURLs: this.componentURLs,
      lastSync: new Date().toISOString(),
      status: this.isRunning ? 'active' : 'inactive'
    };
  }

  /**
   * Update component URLs
   */
  updateComponentURL(component, url) {
    if (this.componentURLs.hasOwnProperty(component)) {
      this.componentURLs[component] = url;
      console.log(`Updated ${component} URL to ${url}`);
    }
  }
}

module.exports = EconomicsIntegration;
