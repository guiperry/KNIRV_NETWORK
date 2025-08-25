// KNIRV-NEXUS Admin Gateway Configuration
// This portal now serves as an administrative gateway for KNIRV-NEXUS management

export const adminGatewayConfig = {
  // Gateway identification
  name: "KNIRV-NEXUS Admin Gateway",
  version: "1.0.0",
  description: "Administrative interface for KNIRV-NEXUS unified deployment management",
  
  // Service endpoints - updated for unified NEXUS deployment
  services: {
    // Unified NEXUS service (replaces separate DVE Manager and Validation Core)
    nexus: {
      name: "KNIRV-NEXUS Unified Service",
      url: process.env.KNIRVNEXUS_URL || "http://localhost:8084",
      apiUrl: process.env.KNIRVNEXUS_API_URL || "http://localhost:8084/api",
      healthEndpoint: "/health",
      description: "Unified NEXUS service with embedded frontend and backend"
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
