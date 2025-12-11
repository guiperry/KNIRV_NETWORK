/**
 * KNIRV Economics Engine
 * Integrated Token Economics System
 */

const TokenEconomics = require('./TokenEconomics');
const EconomicsIntegration = require('./EconomicsIntegration');
const EconomicsAPI = require('./EconomicsAPI');
const EconomicRules = require('./EconomicRules');
const TransactionPool = require('./TransactionPool');
const RewardCalculator = require('./RewardCalculator');
const BurnTracker = require('./BurnTracker');
const EconomicMetrics = require('./EconomicMetrics');

/**
 * Initialize the complete economics system
 */
async function initializeEconomicsSystem(config = {}) {
  // Create token economics instance
  const tokenEconomics = new TokenEconomics({
    nrnContract: config.nrnContract || process.env.NRN_CONTRACT || '',
    xionRPC: config.xionRPC || process.env.XION_RPC || 'https://rpc.xion-testnet-1.burnt.com:443',
    economicRules: config.economicRules,
    transactionPool: config.transactionPool
  });

  // Create integration service
  const integration = new EconomicsIntegration(tokenEconomics, {
    knirvchainURL: config.knirvchainURL || process.env.KNIRVCHAIN_URL || 'http://localhost:8080',
    knirvnexusURL: config.knirvnexusURL || process.env.KNIRVNEXUS_URL || 'http://localhost:8081',
    knirvoracleURL: config.knirvoracleURL || process.env.KNIRVORACLE_URL || 'http://localhost:8082',
    knirvgraphURL: config.knirvgraphURL || process.env.KNIRVGRAPH_URL || 'http://localhost:8083'
  });

  // Create API handler
  const api = new EconomicsAPI(tokenEconomics, integration);

  // Start the economics system
  await tokenEconomics.start();

  // Start integration service
  await integration.start();

  return {
    tokenEconomics,
    integration,
    api,
    router: api.getRouter()
  };
}

module.exports = {
  TokenEconomics,
  EconomicsIntegration,
  EconomicsAPI,
  EconomicRules,
  TransactionPool,
  RewardCalculator,
  BurnTracker,
  EconomicMetrics,
  initializeEconomicsSystem
};
