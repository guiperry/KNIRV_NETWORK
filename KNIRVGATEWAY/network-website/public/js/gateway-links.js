/**
 * KNIRV Gateway Links Management
 * 
 * Provides dynamic link management for the network-website based on deployment environment.
 * Automatically detects public testnet vs private testnet and adjusts links accordingly.
 */

class KNIRVGatewayLinks {
  constructor() {
    this.config = null;
    this.isLoaded = false;
    this.loadPromise = this.loadConfiguration();
  }

  /**
   * Load configuration from the generated config file
   */
  async loadConfiguration() {
    try {
      // Try to load from generated config first
      if (window.KNIRV_GATEWAY_CONFIG) {
        this.config = window.KNIRV_GATEWAY_CONFIG;
        this.isLoaded = true;
        console.log('[Gateway Links] Configuration loaded from window.KNIRV_GATEWAY_CONFIG');
        return this.config;
      }

      // Fallback: load from JSON config file
      const response = await fetch('/config/portal-config.json');
      if (!response.ok) {
        throw new Error(`Failed to load configuration: ${response.status}`);
      }
      
      const portalConfig = await response.json();
      
      // Transform portal config to gateway config format
      this.config = {
        deployment_mode: this.detectDeploymentMode(),
        gateway_services: this.transformGatewayServices(portalConfig.gateway_services),
        external_services: portalConfig.external_services || {},
        navigation: portalConfig.navigation || {},
        timestamp: new Date().toISOString()
      };
      
      this.isLoaded = true;
      console.log(`[Gateway Links] Configuration loaded from portal-config.json (${this.config.deployment_mode})`);
      return this.config;
    } catch (error) {
      console.error('[Gateway Links] Failed to load configuration:', error);
      
      // Ultimate fallback: use hardcoded localhost configuration
      this.config = this.getFallbackConfiguration();
      this.isLoaded = true;
      console.warn('[Gateway Links] Using fallback localhost configuration');
      return this.config;
    }
  }

  /**
   * Detect deployment mode based on current hostname
   */
  detectDeploymentMode() {
    const hostname = window.location.hostname;
    
    if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname.startsWith('192.168.')) {
      return 'private_testnet';
    }
    
    return 'public_testnet';
  }

  /**
   * Transform gateway services configuration
   */
  transformGatewayServices(gatewayServices) {
    if (!gatewayServices) return {};
    
    const mode = this.detectDeploymentMode();
    const services = gatewayServices[mode] || gatewayServices.private_testnet || {};
    
    const transformed = {};
    for (const [serviceName, serviceConfig] of Object.entries(services)) {
      const protocol = serviceConfig.domain === 'localhost' ? 'http' : 'https';
      const port = serviceConfig.domain === 'localhost' ? `:${serviceConfig.port}` : '';
      
      transformed[serviceName] = {
        base_url: `${protocol}://${serviceConfig.domain}${port}${serviceConfig.path}`,
        health_url: `${protocol}://${serviceConfig.domain}${port}${serviceConfig.health_endpoint}`,
        domain: serviceConfig.domain,
        port: serviceConfig.port,
        path: serviceConfig.path
      };
    }
    
    return transformed;
  }

  /**
   * Get fallback configuration for when all else fails
   */
  getFallbackConfiguration() {
    return {
      deployment_mode: 'private_testnet',
      gateway_services: {
        payment_gateway: {
          base_url: 'http://localhost:3001/',
          health_url: 'http://localhost:3001/health',
          domain: 'localhost',
          port: 3001,
          path: '/'
        },
        tunnel_registry: {
          base_url: 'http://localhost:3004/',
          health_url: 'http://localhost:3004/status',
          domain: 'localhost',
          port: 3004,
          path: '/'
        },
        operator_registry: {
          base_url: 'http://localhost:3007/',
          health_url: 'http://localhost:3007/health',
          domain: 'localhost',
          port: 3007,
          path: '/'
        },
        webgui: {
          base_url: 'http://localhost:3007/',
          health_url: 'http://localhost:3007/health',
          domain: 'localhost',
          port: 3007,
          path: '/'
        }
      },
      external_services: {
        knirv_website: 'https://knirv.com',
        testnet_access: 'https://testnet.knirv.com'
      },
      navigation: {},
      timestamp: new Date().toISOString()
    };
  }

  /**
   * Get URL for a gateway service
   */
  async getServiceUrl(serviceName) {
    await this.loadPromise;
    
    const service = this.config.gateway_services[serviceName];
    return service ? service.base_url : null;
  }

  /**
   * Get health check URL for a gateway service
   */
  async getServiceHealthUrl(serviceName) {
    await this.loadPromise;
    
    const service = this.config.gateway_services[serviceName];
    return service ? service.health_url : null;
  }

  /**
   * Get external service URL
   */
  async getExternalServiceUrl(serviceName) {
    await this.loadPromise;
    
    return this.config.external_services[serviceName] || null;
  }

  /**
   * Check if running in private testnet mode
   */
  async isPrivateTestnet() {
    await this.loadPromise;
    return this.config.deployment_mode === 'private_testnet';
  }

  /**
   * Check if running in public testnet mode
   */
  async isPublicTestnet() {
    await this.loadPromise;
    return this.config.deployment_mode === 'public_testnet';
  }

  /**
   * Get all gateway service URLs
   */
  async getAllServiceUrls() {
    await this.loadPromise;
    return this.config.gateway_services;
  }

  /**
   * Update links in the DOM
   */
  async updateDOMLinks() {
    await this.loadPromise;
    
    // Update gateway service links
    for (const [serviceName, serviceConfig] of Object.entries(this.config.gateway_services)) {
      const elements = document.querySelectorAll(`[data-gateway-service="${serviceName}"]`);
      elements.forEach(element => {
        if (element.tagName === 'A') {
          element.href = serviceConfig.base_url;
        } else {
          element.textContent = serviceConfig.base_url;
        }
      });
    }

    // Update external service links
    for (const [serviceName, url] of Object.entries(this.config.external_services)) {
      const elements = document.querySelectorAll(`[data-external-service="${serviceName}"]`);
      elements.forEach(element => {
        if (element.tagName === 'A') {
          element.href = url;
        } else {
          element.textContent = url;
        }
      });
    }

    // Add deployment mode indicator
    const modeElements = document.querySelectorAll('[data-deployment-mode]');
    modeElements.forEach(element => {
      element.textContent = this.config.deployment_mode.replace('_', ' ').toUpperCase();
      element.className = `deployment-mode ${this.config.deployment_mode}`;
    });

    console.log('[Gateway Links] DOM links updated');
  }

  /**
   * Create a service link element
   */
  async createServiceLink(serviceName, text = null, className = '') {
    const url = await this.getServiceUrl(serviceName);
    if (!url) return null;

    const link = document.createElement('a');
    link.href = url;
    link.textContent = text || serviceName.replace('_', ' ').toUpperCase();
    link.className = className;
    link.target = '_blank';
    link.rel = 'noopener noreferrer';

    return link;
  }

  /**
   * Check service health status
   */
  async checkServiceHealth(serviceName) {
    const healthUrl = await this.getServiceHealthUrl(serviceName);
    if (!healthUrl) return { healthy: false, error: 'Service not configured' };

    try {
      const response = await fetch(healthUrl, { 
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache'
      });
      
      return {
        healthy: response.ok,
        status: response.status,
        url: healthUrl
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message,
        url: healthUrl
      };
    }
  }
}

// Create global instance
window.knirvGatewayLinks = new KNIRVGatewayLinks();

// Auto-update DOM links when page loads
document.addEventListener('DOMContentLoaded', () => {
  window.knirvGatewayLinks.updateDOMLinks();
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = KNIRVGatewayLinks;
}
