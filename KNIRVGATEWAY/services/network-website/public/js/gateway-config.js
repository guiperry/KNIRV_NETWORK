
// Auto-generated KNIRV Gateway Links Configuration
// Generated at: 2025-09-22T04:33:43.707Z
// Deployment Mode: public_testnet

window.KNIRV_GATEWAY_CONFIG = {
  "deployment_mode": "public_testnet",
  "gateway_services": {
    "payment_gateway": {
      "base_url": "https://payments.knirv.network/",
      "health_url": "https://payments.knirv.network/health",
      "domain": "payments.knirv.network",
      "port": 3001,
      "path": "/"
    },
    "tunnel_registry": {
      "base_url": "https://tunnel.knirv.network/",
      "health_url": "https://tunnel.knirv.network/status",
      "domain": "tunnel.knirv.network",
      "port": 3004,
      "path": "/"
    },
    "operator_registry": {
      "base_url": "https://operators.knirv.network/",
      "health_url": "https://operators.knirv.network/health",
      "domain": "operators.knirv.network",
      "port": 3007,
      "path": "/"
    },
    "webgui": {
      "base_url": "https://dashboard.knirv.network/",
      "health_url": "https://dashboard.knirv.network/health",
      "domain": "dashboard.knirv.network",
      "port": 3007,
      "path": "/"
    }
  },
  "external_services": {
    "payment_gateway": "https://payments.knirv.network",
    "tunnel_registry": "https://tunnel.knirv.network",
    "operator_registry": "https://operators.knirv.network",
    "webgui": "https://dashboard.knirv.network",
    "knirv_website": "https://knirv.com",
    "testnet_access": "https://testnet.knirv.network"
  },
  "navigation": {
    "main_site": "https://knirv.network",
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
  const service = window.KNIRV_GATEWAY_CONFIG.gateway_services[serviceName];
  return service ? service.base_url : null;
};

window.getGatewayServiceHealthUrl = function(serviceName) {
  const service = window.KNIRV_GATEWAY_CONFIG.gateway_services[serviceName];
  return service ? service.health_url : null;
};

window.isPrivateTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'private_testnet';
};

window.isPublicTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'public_testnet';
};
