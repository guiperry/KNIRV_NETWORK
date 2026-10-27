
// Auto-generated KNIRV Gateway Links Configuration
// Generated at: 2025-09-22T04:33:43.707Z
// Deployment Mode: public_testnet

window.KNIRV_GATEWAY_CONFIG = {
  "deployment_mode": "public_testnet",
  "oracle_services": {
    "payment_oracle": {
      "base_url": "https://payments.knirv.com/",
      "health_url": "https://payments.knirv.com/health",
      "domain": "payments.knirv.com",
      "port": 3001,
      "path": "/"
    },
    "tunnel_registry": {
      "base_url": "https://tunnel.knirv.com/",
      "health_url": "https://tunnel.knirv.com/status",
      "domain": "tunnel.knirv.com",
      "port": 3004,
      "path": "/"
    },
    "operator_registry": {
      "base_url": "https://operators.knirv.com/",
      "health_url": "https://operators.knirv.com/health",
      "domain": "operators.knirv.com",
      "port": 3007,
      "path": "/"
    },
    "webgui": {
      "base_url": "https://dashboard.knirv.com/",
      "health_url": "https://dashboard.knirv.com/health",
      "domain": "dashboard.knirv.com",
      "port": 3007,
      "path": "/"
    }
  },
  "external_services": {
    "payment_oracle": "https://payments.knirv.com",
    "tunnel_registry": "https://tunnel.knirv.com",
    "operator_registry": "https://operators.knirv.com",
    "webgui": "https://dashboard.knirv.com",
    "knirv_website": "https://knirv.com",
    "testnet_access": "https://testnet.knirv.com"
  },
  "navigation": {
    "main_site": "https://knirv.com",
    "documentation": "../documentation/docsify/",
    "graphchain_explorer": "../graphchain-explorer/",
    "nexus_portal": "../nexus-portal/",
    "support_desk": "../support-desk/src/",
    "nanda_ans": "../nanda_ans/"
  },
  "timestamp": "2025-09-22T04:33:43.707Z"
};

// Helper functions for easy access
window.getGatewayServiceUrl = function(serviceName) {
  const service = window.KNIRV_GATEWAY_CONFIG.oracle_services[serviceName];
  return service ? service.base_url : null;
};

window.getGatewayServiceHealthUrl = function(serviceName) {
  const service = window.KNIRV_GATEWAY_CONFIG.oracle_services[serviceName];
  return service ? service.health_url : null;
};

window.isPrivateTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'private_testnet';
};

window.isPublicTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'public_testnet';
};
