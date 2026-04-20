// KNIRV-SERVER Admin Gateway Configuration
// This portal now serves as an administrative gateway for KNIRV-SERVER management

export const adminGatewayConfig = {
  // Gateway identification
  name: "KNIRV-SERVER Admin Gateway",
  version: "1.0.0",
  description: "Administrative interface for KNIRV-SERVER unified deployment management",
  
  // Service endpoints - Remote API configuration
  services: {
    // Primary NEXUS API (remote production endpoint)
    nexus: {
      name: "KNIRV-SERVER Remote API",
      url: "https://nexus-api.knirv.com",
      apiUrl: "https://nexus-api.knirv.com/api",
      healthEndpoint: "/health",
      description: "Remote NEXUS API endpoint for production portal"
    },
    
    // Other KNIRV services for reference
    oracle: {
      name: "KNIRV Oracle",
      url: process.env.KNIRVORACLE_URL || "http://localhost:1317",
      healthEndpoint: "/health"
    },
    
    chain: {
      name: "KNIRV Chain",
      url: process.env.KNIRVCHAIN_URL || "http://localhost:8090",
      healthEndpoint: "/health"
    },
    
    graph: {
      name: "KNIRV Graph",
      url: process.env.KNIRVGRAPH_URL || "http://localhost:8082",
      healthEndpoint: "/height"
    },
    
    router: {
      name: "KNIRV Router",
      url: process.env.KNIRVROUTER_URL || "http://localhost:8086",
      healthEndpoint: "/status"
    },
    
    gateway: {
      name: "KNIRV Gateway",
      url: process.env.KNIRVGATEWAY_URL || "http://localhost:8888",
      healthEndpoint: "/health"
    }
  },
  
  // Admin features
  features: {
    systemMonitoring: true,
    serviceManagement: true,
    userManagement: true,
    configurationManagement: true,
    logViewing: true,
    metricsViewing: true,
    testnetControls: true
  },
  
  // Authentication configuration
  auth: {
    required: true,
    roles: ["admin", "validator", "observer"],
    defaultRole: "observer",
    sessionTimeout: 3600000 // 1 hour
  },
  
  // UI configuration
  ui: {
    theme: "knirv-admin",
    showServiceStatus: true,
    showMetrics: true,
    showLogs: true,
    refreshInterval: 30000 // 30 seconds
  },
  
  // Testnet specific configuration
  testnet: {
    enabled: process.env.TESTNET_MODE === "true",
    features: {
      mockValidation: true,
      simulatedTEE: true,
      reducedSecurity: true,
      debugMode: true
    }
  }
};

export default adminGatewayConfig;
